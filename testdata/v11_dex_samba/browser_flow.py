#!/usr/bin/env python3
"""Real browser-like OIDC/PKCE harness for the isolated V11 Dex/Samba smoke."""

import argparse
import base64
import hashlib
import http.cookiejar
import http.server
import json
import os
import queue
import secrets
import socket
import ssl
import subprocess
import threading
import urllib.error
import urllib.parse
import urllib.request
from html.parser import HTMLParser


class LoginPage(HTMLParser):
    def __init__(self):
        super().__init__()
        self.forms = []
        self.links = []
        self._form = None

    def handle_starttag(self, tag, attrs):
        values = dict(attrs)
        if tag == "form":
            self._form = {
                "action": values.get("action", ""),
                "method": values.get("method", "get").lower(),
                "fields": {},
            }
        elif tag == "input" and self._form is not None:
            name = values.get("name")
            if name:
                self._form["fields"][name] = values.get("value", "")
        elif tag == "a" and values.get("href"):
            self.links.append(values["href"])

    def handle_endtag(self, tag):
        if tag == "form" and self._form is not None:
            self.forms.append(self._form)
            self._form = None


class CallbackHandler(http.server.BaseHTTPRequestHandler):
    callbacks = None

    def do_GET(self):
        parsed = urllib.parse.urlparse(self.path)
        self.callbacks.put(urllib.parse.parse_qs(parsed.query))
        body = b"OIDC callback captured"
        self.send_response(200)
        self.send_header("Content-Type", "text/plain; charset=utf-8")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def log_message(self, _format, *_args):
        return


def read_secret(path):
    with open(path, "r", encoding="utf-8") as handle:
        value = handle.read().strip()
    if not value:
        raise RuntimeError("v11.secret_empty")
    return value


def b64url(value):
    return base64.urlsafe_b64encode(value).rstrip(b"=").decode("ascii")


def open_page(opener, request):
    try:
        with opener.open(request, timeout=10) as response:
            return response.geturl(), response.read(1 << 20).decode("utf-8", "replace")
    except urllib.error.HTTPError as error:
        with error:
            return error.geturl(), error.read(1 << 20).decode("utf-8", "replace")


def authorize(issuer, client_id, redirect_uri, username, password, callbacks, expect_success):
    while True:
        try:
            callbacks.get_nowait()
        except queue.Empty:
            break
    verifier = b64url(os.urandom(48))
    challenge = b64url(hashlib.sha256(verifier.encode("ascii")).digest())
    state = secrets.token_urlsafe(24)
    nonce = secrets.token_urlsafe(24)
    query = urllib.parse.urlencode({
        "client_id": client_id,
        "redirect_uri": redirect_uri,
        "response_type": "code",
        "scope": "openid groups profile email",
        "state": state,
        "nonce": nonce,
        "code_challenge": challenge,
        "code_challenge_method": "S256",
        "connector_id": "ldap",
    })
    opener = urllib.request.build_opener(
        urllib.request.HTTPCookieProcessor(http.cookiejar.CookieJar())
    )
    current_url, body = open_page(opener, issuer + "/auth?" + query)

    submitted = False
    for _ in range(6):
        parser = LoginPage()
        parser.feed(body)
        login_form = next(
            (form for form in parser.forms if "login" in form["fields"] and "password" in form["fields"]),
            None,
        )
        if login_form is not None:
            if submitted:
                break
            submitted = True
            fields = dict(login_form["fields"])
            fields["login"] = username
            fields["password"] = password
            target = urllib.parse.urljoin(current_url, login_form["action"])
            request = urllib.request.Request(
                target,
                data=urllib.parse.urlencode(fields).encode("ascii"),
                headers={"Content-Type": "application/x-www-form-urlencoded"},
                method="POST",
            )
            current_url, body = open_page(opener, request)
        else:
            ldap_link = next((link for link in parser.links if "/auth/ldap" in link), None)
            if ldap_link is None:
                break
            current_url, body = open_page(opener, urllib.parse.urljoin(current_url, ldap_link))

        if urllib.parse.urlparse(current_url).path == urllib.parse.urlparse(redirect_uri).path:
            break

    try:
        callback = callbacks.get(timeout=1)
    except queue.Empty:
        callback = None

    if not expect_success:
        if callback is not None:
            raise RuntimeError("v11.invalid_directory_credentials_accepted")
        return None
    if callback is None:
        raise RuntimeError("v11.oidc_callback_missing")
    if callback.get("state", [""])[0] != state:
        raise RuntimeError("v11.state_mismatch")
    code = callback.get("code", [""])[0]
    if not code:
        raise RuntimeError("v11.authorization_code_missing")

    token_request = urllib.request.Request(
        issuer + "/token",
        data=urllib.parse.urlencode({
            "grant_type": "authorization_code",
            "code": code,
            "client_id": client_id,
            "redirect_uri": redirect_uri,
            "code_verifier": verifier,
        }).encode("ascii"),
        headers={"Content-Type": "application/x-www-form-urlencoded"},
        method="POST",
    )
    with urllib.request.urlopen(token_request, timeout=10) as response:
        tokens = json.load(response)
    id_token = tokens.get("id_token", "")
    if not id_token or tokens.get("token_type") != "bearer":
        raise RuntimeError("v11.id_token_missing")
    pieces = id_token.split(".")
    if len(pieces) != 3:
        raise RuntimeError("v11.id_token_malformed")
    padded = pieces[1] + "=" * (-len(pieces[1]) % 4)
    claims = json.loads(base64.urlsafe_b64decode(padded))
    if claims.get("nonce") != nonce:
        raise RuntimeError("v11.nonce_mismatch")
    return {"token": id_token, "claims": claims}


def product_probe(probe, issuer, client_id, group, token):
    completed = subprocess.run(
        [probe, issuer, client_id, group],
        input=token + "\n",
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        timeout=20,
        check=False,
    )
    if completed.returncode != 0:
        raise RuntimeError("v11.product_probe_failed")
    try:
        return json.loads(completed.stdout)
    except json.JSONDecodeError as error:
        raise RuntimeError("v11.product_probe_invalid_output") from error


def assert_tls(host, port, ca_path, wrong_ca_path):
    good = ssl.create_default_context(cafile=ca_path)
    with socket.create_connection((host, port), timeout=5) as raw:
        with good.wrap_socket(raw, server_hostname="samba") as connection:
            if not connection.getpeercert():
                raise RuntimeError("v11.ldaps_peer_certificate_missing")
    bad = ssl.create_default_context(cafile=wrong_ca_path)
    try:
        with socket.create_connection((host, port), timeout=5) as raw:
            with bad.wrap_socket(raw, server_hostname="samba"):
                pass
    except ssl.SSLCertVerificationError:
        return
    raise RuntimeError("v11.ldaps_invalid_ca_accepted")


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--issuer", required=True)
    parser.add_argument("--client-id", required=True)
    parser.add_argument("--required-group", required=True)
    parser.add_argument("--callback-port", required=True, type=int)
    parser.add_argument("--probe", required=True)
    parser.add_argument("--samba-container", required=True)
    parser.add_argument("--ldaps-port", required=True, type=int)
    parser.add_argument("--ca", required=True)
    parser.add_argument("--wrong-ca", required=True)
    parser.add_argument("--alice-password-file", required=True)
    parser.add_argument("--disabled-password-file", required=True)
    args = parser.parse_args()

    callbacks = queue.Queue()
    CallbackHandler.callbacks = callbacks
    server = http.server.ThreadingHTTPServer(("127.0.0.1", args.callback_port), CallbackHandler)
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()
    redirect_uri = f"http://127.0.0.1:{args.callback_port}/callback"
    alice_password = read_secret(args.alice_password_file)
    disabled_password = read_secret(args.disabled_password_file)

    try:
        assert_tls("127.0.0.1", args.ldaps_port, args.ca, args.wrong_ca)
        print("PASS ldaps_valid_chain_and_invalid_ca_rejected")

        first = authorize(
            args.issuer, args.client_id, redirect_uri, "alice", alice_password, callbacks, True
        )
        first_groups = first["claims"].get("groups", [])
        if args.required_group not in first_groups:
            raise RuntimeError("v11.required_group_claim_missing")
        first_probe = product_probe(
            args.probe, args.issuer, args.client_id, args.required_group, first["token"]
        )
        if first_probe.get("status") != "accepted":
            raise RuntimeError("v11.required_group_not_admitted")
        print("PASS valid_login_and_required_group_admission")

        second = authorize(
            args.issuer, args.client_id, redirect_uri, "alice", alice_password, callbacks, True
        )
        second_probe = product_probe(
            args.probe, args.issuer, args.client_id, args.required_group, second["token"]
        )
        if (
            first["claims"].get("sub") != second["claims"].get("sub")
            or first_probe.get("principal_ref") != second_probe.get("principal_ref")
            or first_probe.get("actor_ref") != second_probe.get("actor_ref")
        ):
            raise RuntimeError("v11.subject_or_principal_not_stable")
        print("PASS stable_subject_principal_and_actor")

        authorize(
            args.issuer, args.client_id, redirect_uri, "alice", "definitely-wrong", callbacks, False
        )
        print("PASS wrong_password_rejected")
        authorize(
            args.issuer, args.client_id, redirect_uri, "disabled", disabled_password, callbacks, False
        )
        print("PASS disabled_user_rejected")

        mutation = subprocess.run(
            [
                "docker",
                "exec",
                args.samba_container,
                "samba-tool",
                "group",
                "removemembers",
                args.required_group,
                "alice",
            ],
            stdout=subprocess.DEVNULL,
            stderr=subprocess.DEVNULL,
            timeout=20,
            check=False,
        )
        if mutation.returncode != 0:
            raise RuntimeError("v11.group_removal_failed")

        old_probe = product_probe(
            args.probe, args.issuer, args.client_id, args.required_group, first["token"]
        )
        if old_probe.get("status") != "accepted":
            raise RuntimeError("v11.stateless_issued_token_revoked_early")
        removed = authorize(
            args.issuer, args.client_id, redirect_uri, "alice", alice_password, callbacks, True
        )
        if args.required_group in removed["claims"].get("groups", []):
            raise RuntimeError("v11.removed_group_still_in_new_token")
        removed_probe = product_probe(
            args.probe, args.issuer, args.client_id, args.required_group, removed["token"]
        )
        if removed_probe != {"status": "rejected", "code": "oidc.group_denied"}:
            raise RuntimeError("v11.removed_group_not_rejected")
        print("PASS removed_group_rejects_new_token_old_token_lives_to_expiry")
        print("V11_DEX_SAMBA_AD_SMOKE=PASS")
    finally:
        server.shutdown()
        server.server_close()
        thread.join(timeout=2)


if __name__ == "__main__":
    main()
