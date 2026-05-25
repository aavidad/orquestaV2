#!/usr/bin/env python3
"""
Send one direct email through the existing EWS/OWA configuration.

This script is intentionally separate from the USO digest sender. It reuses the
same EWS credential file format, but recipients, subject and body always come
from the current command or JSON input.
"""

from __future__ import annotations

import argparse
import base64
import html
import json
import mimetypes
import os
from pathlib import Path
import re
import subprocess
import sys
import tempfile
from typing import Any, Iterable
from xml.etree import ElementTree as ET


DEFAULT_ENV_PATH = "/opt/dipgra_web/email_digest.env"
DEFAULT_SECRET_KEY_PATH = "/opt/dipgra_web/email_digest.key"


def parse_env(path: Path) -> dict[str, str]:
    values: dict[str, str] = {}
    if not path.exists():
        return values
    for raw in path.read_text(encoding="utf-8").splitlines():
        line = raw.strip()
        if not line or line.startswith("#") or "=" not in line:
            continue
        key, value = line.split("=", 1)
        key = key.strip()
        value = value.strip()
        if len(value) >= 2 and value[0] == value[-1] and value[0] in {"'", '"'}:
            value = value[1:-1]
        values[key] = value
    return values


def env_bool(values: dict[str, str], key: str, default: bool = False) -> bool:
    raw = values.get(key, "").strip().lower()
    if raw == "":
        return default
    return raw not in {"0", "false", "no", "off"}


def decrypt_openssl_secret(ciphertext: str, key_path: Path) -> str:
    if not key_path.exists():
        raise SystemExit("missing email secret key file")
    proc = subprocess.run(
        [
            "openssl",
            "enc",
            "-aes-256-cbc",
            "-d",
            "-a",
            "-A",
            "-pbkdf2",
            "-pass",
            "file:" + str(key_path),
        ],
        input=ciphertext,
        text=True,
        capture_output=True,
        timeout=15,
        check=False,
    )
    if proc.returncode != 0:
        raise SystemExit("email secret decrypt failed")
    return proc.stdout


def resolve_email_credentials(values: dict[str, str]) -> dict[str, str]:
    resolved = dict(values)
    key_path = Path(values.get("EMAIL_SECRET_KEY_PATH") or DEFAULT_SECRET_KEY_PATH)
    for key in ["EMAIL_EWS_USER", "EMAIL_EWS_PASSWORD"]:
        encrypted = values.get(key + "_ENC", "").strip()
        if encrypted:
            resolved[key] = decrypt_openssl_secret(encrypted, key_path)
    return resolved


def split_addresses(raw: str | Iterable[str] | None) -> list[str]:
    if raw is None:
        return []
    if isinstance(raw, str):
        parts = re.split(r"[,;]", raw)
    else:
        parts = []
        for item in raw:
            parts.extend(re.split(r"[,;]", str(item)))
    return [part.strip() for part in parts if part.strip()]


def unique_addresses(addresses: Iterable[str]) -> list[str]:
    unique: list[str] = []
    seen = set()
    for address in addresses:
        key = address.lower()
        if key in seen:
            continue
        seen.add(key)
        unique.append(address)
    return unique


def validate_addresses(label: str, addresses: list[str]) -> None:
    pattern = re.compile(r"^[^@\s<>]+@[^@\s<>]+\.[^@\s<>]+$")
    invalid = [address for address in addresses if not pattern.match(address)]
    if invalid:
        raise SystemExit(f"invalid {label} address: " + ", ".join(invalid))


def plain_text_to_html(text: str) -> str:
    escaped = html.escape(text, quote=False)
    return "<br>\n".join(escaped.splitlines())


def recipient_xml(tag: str, addresses: list[str]) -> str:
    if not addresses:
        return ""
    entries = "".join(
        "<t:Mailbox><t:EmailAddress>"
        + html.escape(address, quote=False)
        + "</t:EmailAddress></t:Mailbox>"
        for address in addresses
    )
    return f"<t:{tag}>{entries}</t:{tag}>"


def resolve_attachment(path_value: str) -> dict[str, str]:
    path = Path(path_value).expanduser().resolve()
    if not path.exists():
        raise SystemExit("attachment does not exist: " + str(path))
    if not path.is_file():
        raise SystemExit("attachment is not a file: " + str(path))
    content_type = mimetypes.guess_type(path.name)[0] or "application/octet-stream"
    try:
        content = base64.b64encode(path.read_bytes()).decode("ascii")
    except OSError as exc:
        raise SystemExit("attachment read failed: " + str(path)) from exc
    return {
        "name": path.name,
        "path": str(path),
        "content_type": content_type,
        "content": content,
    }


def attachments_xml(items: list[dict[str, str]]) -> str:
    if not items:
        return ""
    parts = ["<t:Attachments>"]
    for item in items:
        parts.append(
            "<t:FileAttachment>"
            "<t:Name>" + html.escape(item["name"], quote=False) + "</t:Name>"
            "<t:ContentType>" + html.escape(item["content_type"], quote=False) + "</t:ContentType>"
            "<t:Content>" + item["content"] + "</t:Content>"
            "</t:FileAttachment>"
        )
    parts.append("</t:Attachments>")
    return "".join(parts)


def build_ews_envelope(
    subject: str,
    body: str,
    body_is_html: bool,
    to_addrs: list[str],
    cc_addrs: list[str],
    bcc_addrs: list[str],
    attachments: list[dict[str, str]],
) -> bytes:
    body_type = "HTML" if body_is_html else "Text"
    xml = f"""<?xml version="1.0" encoding="utf-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
  xmlns:t="http://schemas.microsoft.com/exchange/services/2006/types"
  xmlns:m="http://schemas.microsoft.com/exchange/services/2006/messages">
  <soap:Header>
    <t:RequestServerVersion Version="Exchange2010_SP2"/>
  </soap:Header>
  <soap:Body>
    <m:CreateItem MessageDisposition="SendAndSaveCopy">
      <m:SavedItemFolderId>
        <t:DistinguishedFolderId Id="sentitems"/>
      </m:SavedItemFolderId>
      <m:Items>
        <t:Message>
          <t:Subject>{html.escape(subject, quote=False)}</t:Subject>
          <t:Body BodyType="{body_type}">{html.escape(body, quote=False)}</t:Body>
          {attachments_xml(attachments)}
          {recipient_xml("ToRecipients", to_addrs)}
          {recipient_xml("CcRecipients", cc_addrs)}
          {recipient_xml("BccRecipients", bcc_addrs)}
        </t:Message>
      </m:Items>
    </m:CreateItem>
  </soap:Body>
</soap:Envelope>
"""
    return xml.encode("utf-8")


def curl_config_escape(value: str) -> str:
    return value.replace("\\", "\\\\").replace('"', '\\"').replace("\n", "")


def require(values: dict[str, str], keys: Iterable[str]) -> None:
    missing = [key for key in keys if not values.get(key, "").strip()]
    if missing:
        raise SystemExit("missing email config: " + ", ".join(missing))


def parse_ews_response(response_xml: str) -> None:
    try:
        root = ET.fromstring(response_xml.encode("utf-8"))
    except ET.ParseError as exc:
        raise SystemExit("ews response parse failed: " + str(exc)) from exc
    ns = {"m": "http://schemas.microsoft.com/exchange/services/2006/messages"}
    response = root.find(".//m:CreateItemResponseMessage", ns)
    if response is None:
        raise SystemExit("ews response error: missing CreateItemResponseMessage")
    if response.attrib.get("ResponseClass") != "Success":
        code = response.findtext("m:ResponseCode", default="", namespaces=ns)
        message = response.findtext("m:MessageText", default="", namespaces=ns)
        detail = " ".join(part for part in [code, message] if part)
        raise SystemExit("ews response error: " + (detail or "unknown"))


def send_ews(
    values: dict[str, str],
    subject: str,
    body: str,
    body_is_html: bool,
    to_addrs: list[str],
    cc_addrs: list[str],
    bcc_addrs: list[str],
    attachments: list[dict[str, str]],
    dry_run: bool,
) -> dict[str, Any]:
    if dry_run:
        return {
            "status": "dry_run",
            "to_count": len(to_addrs),
            "cc_count": len(cc_addrs),
            "bcc_count": len(bcc_addrs),
            "attachment_count": len(attachments),
            "attachments": [item["name"] for item in attachments],
            "subject": subject,
        }

    values = resolve_email_credentials(values)
    require(values, ["EMAIL_EWS_URL", "EMAIL_EWS_USER", "EMAIL_EWS_PASSWORD"])
    envelope_file = None
    try:
        fd, envelope_file = tempfile.mkstemp(prefix="owa-send-mail-", suffix=".xml")
        with os.fdopen(fd, "wb") as fh:
            fh.write(build_ews_envelope(subject, body, body_is_html, to_addrs, cc_addrs, bcc_addrs, attachments))

        auth_user = values["EMAIL_EWS_USER"].replace("\\\\", "\\")
        userpwd = auth_user + ":" + values["EMAIL_EWS_PASSWORD"]
        config = "\n".join(
            [
                f'url = "{curl_config_escape(values["EMAIL_EWS_URL"])}"',
                "request = POST",
                "ntlm",
                f'user = "{curl_config_escape(userpwd)}"',
                'header = "Content-Type: text/xml; charset=utf-8"',
                'header = "SOAPAction: http://schemas.microsoft.com/exchange/services/2006/messages/CreateItem"',
                f'data-binary = "@{curl_config_escape(envelope_file)}"',
                "silent",
                "show-error",
                'write-out = "\\nHTTP_STATUS:%{http_code}\\n"',
            ]
        )
        if not env_bool(values, "EMAIL_EWS_VERIFY_TLS", True):
            config += "\ninsecure\n"
        proc = subprocess.run(
            ["curl", "--config", "-"],
            input=config,
            text=True,
            capture_output=True,
            timeout=60,
            check=False,
        )
    finally:
        if envelope_file:
            Path(envelope_file).unlink(missing_ok=True)

    if proc.returncode != 0:
        raise SystemExit("curl/ews failed: " + (proc.stderr.strip() or f"exit {proc.returncode}"))
    status_match = re.search(r"HTTP_STATUS:(\d+)", proc.stdout)
    status = int(status_match.group(1)) if status_match else 0
    response_xml = proc.stdout.split("\nHTTP_STATUS:", 1)[0]
    if status < 200 or status >= 300:
        raise SystemExit(f"ews http status {status}")
    parse_ews_response(response_xml)
    return {
        "status": "sent",
        "to_count": len(to_addrs),
        "cc_count": len(cc_addrs),
        "bcc_count": len(bcc_addrs),
        "attachment_count": len(attachments),
        "attachments": [item["name"] for item in attachments],
        "subject": subject,
    }


def read_json_payload(path: str | None) -> dict[str, Any]:
    if path:
        raw = Path(path).read_text(encoding="utf-8")
    else:
        raw = sys.stdin.read()
    if not raw.strip():
        raise SystemExit("missing JSON input")
    payload = json.loads(raw)
    if not isinstance(payload, dict):
        raise SystemExit("JSON input must be an object")
    return payload


def body_from_args(args: argparse.Namespace, payload: dict[str, Any]) -> str:
    body = payload.get("body")
    if args.body_file:
        body = Path(args.body_file).read_text(encoding="utf-8")
    if args.body is not None:
        body = args.body
    if body is None and not sys.stdin.isatty() and not args.json_input:
        body = sys.stdin.read()
    if body is None or not str(body).strip():
        raise SystemExit("missing email body")
    return str(body)


def collect_attachments(args: argparse.Namespace, payload: dict[str, Any]) -> list[dict[str, str]]:
    raw = []
    payload_attachments = payload.get("attachments") or payload.get("attach") or []
    if isinstance(payload_attachments, str):
        raw.append(payload_attachments)
    else:
        raw.extend(str(item) for item in payload_attachments)
    raw.extend(args.attach or [])
    return [resolve_attachment(item) for item in raw if str(item).strip()]


def main() -> int:
    parser = argparse.ArgumentParser(description="Send a direct email through EWS/OWA.")
    parser.add_argument("--env", default=DEFAULT_ENV_PATH)
    parser.add_argument("--json-input", nargs="?", const="-", help="Read payload JSON from file or stdin.")
    parser.add_argument("--to", action="append", default=[])
    parser.add_argument("--cc", action="append", default=[])
    parser.add_argument("--bcc", action="append", default=[])
    parser.add_argument("--subject")
    parser.add_argument("--body")
    parser.add_argument("--body-file")
    parser.add_argument("--attach", action="append", default=[], help="File path to attach. Can be repeated.")
    parser.add_argument("--html", action="store_true", help="Treat body as HTML.")
    parser.add_argument("--dry-run", action="store_true", help="Validate inputs without sending.")
    args = parser.parse_args()

    payload: dict[str, Any] = {}
    if args.json_input:
        payload = read_json_payload(None if args.json_input == "-" else args.json_input)

    subject = args.subject or str(payload.get("subject") or "").strip()
    if not subject:
        raise SystemExit("missing email subject")

    body = body_from_args(args, payload)
    body_is_html = bool(args.html or payload.get("html"))
    if not body_is_html:
        body = plain_text_to_html(body)
        body_is_html = True

    to_addrs = unique_addresses(split_addresses(payload.get("to")) + split_addresses(args.to))
    cc_addrs = unique_addresses(split_addresses(payload.get("cc")) + split_addresses(args.cc))
    bcc_addrs = unique_addresses(split_addresses(payload.get("bcc")) + split_addresses(args.bcc))
    if not to_addrs and not cc_addrs and not bcc_addrs:
        raise SystemExit("missing email recipients")
    validate_addresses("to", to_addrs)
    validate_addresses("cc", cc_addrs)
    validate_addresses("bcc", bcc_addrs)
    attachments = collect_attachments(args, payload)

    values = parse_env(Path(args.env))
    result = send_ews(values, subject, body, body_is_html, to_addrs, cc_addrs, bcc_addrs, attachments, args.dry_run)
    print(json.dumps(result, ensure_ascii=False, sort_keys=True))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
