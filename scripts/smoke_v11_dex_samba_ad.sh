#!/usr/bin/env bash
set -euo pipefail

DEX_IMAGE='ghcr.io/dexidp/dex@sha256:8499afd690c437f52301efd2b05b2455da5bd2dfc20332cd697dc9937f808462'
CLIENT_ID='orquesta-v11'
REQUIRED_GROUP='orquesta-users'
stage='preflight'

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
runtime_root=$(mktemp -d "${TMPDIR:-/tmp}/orquesta-v11-dex-samba.XXXXXXXX")
suffix="$$-${RANDOM}${RANDOM}"
network="orquesta-v11-${suffix}"
samba_container="orquesta-v11-samba-${suffix}"
dex_container="orquesta-v11-dex-${suffix}"
samba_image="orquesta-v11-samba:${suffix}"

cleanup() {
    status=$?
    trap - EXIT INT TERM
    if ((status != 0)); then
        for failed_container in "${samba_container}" "${dex_container}"; do
            if docker inspect "${failed_container}" >/dev/null 2>&1; then
                diagnostics=$(docker logs "${failed_container}" 2>&1 | tail -n 80)
                for secret_file in "${runtime_root}"/secrets/*; do
                    [[ -f ${secret_file} ]] || continue
                    secret_value=$(<"${secret_file}")
                    if [[ -n ${secret_value} ]]; then
                        diagnostics=${diagnostics//"${secret_value}"/[REDACTED]}
                    fi
                done
                printf '%s logs (redacted):\n%s\n' "${failed_container}" "${diagnostics}" >&2
            fi
        done
    fi
    docker rm --force "${dex_container}" "${samba_container}" >/dev/null 2>&1 || true
    docker network rm "${network}" >/dev/null 2>&1 || true
    docker image rm --force "${samba_image}" >/dev/null 2>&1 || true
    rm -rf -- "${runtime_root}"
    if docker ps --all --format '{{.Names}}' | grep -Fxq -e "${dex_container}" -e "${samba_container}"; then
        printf 'FAIL cleanup_container_residue\n' >&2
        exit 1
    fi
    if docker network ls --format '{{.Name}}' | grep -Fxq "${network}"; then
        printf 'FAIL cleanup_network_residue\n' >&2
        exit 1
    fi
    if ((status == 0)); then
        printf 'PASS cleanup_no_container_network_or_runtime_residue\n'
    else
        printf 'V11_DEX_SAMBA_AD_SMOKE=FAIL stage=%s\n' "${stage}" >&2
    fi
    exit "${status}"
}
trap cleanup EXIT INT TERM

require_command() {
    command -v "$1" >/dev/null 2>&1 || {
        printf 'missing command: %s\n' "$1" >&2
        return 1
    }
}

free_port() {
    python3 - <<'PY'
import socket
with socket.socket() as listener:
    listener.bind(("127.0.0.1", 0))
    print(listener.getsockname()[1])
PY
}

wait_for_dex() {
    for _ in $(seq 1 90); do
        if [[ $(docker inspect --format '{{.State.Running}}' "${dex_container}" 2>/dev/null) != true ]]; then
            printf 'Dex container exited during startup\n' >&2
            return 1
        fi
        if curl --fail --silent --show-error --connect-timeout 2 --max-time 5 \
            "${issuer}/.well-known/openid-configuration" >/dev/null 2>&1; then
            return 0
        fi
        sleep 1
    done
    printf 'timeout waiting for Dex OIDC discovery\n' >&2
    return 1
}

wait_for_samba() {
    for _ in $(seq 1 90); do
        if [[ $(docker inspect --format '{{.State.Running}}' "${samba_container}" 2>/dev/null) != true ]]; then
            printf 'Samba AD container exited during startup\n' >&2
            return 1
        fi
        if docker exec "${samba_container}" \
            bash -c 'exec 3<>/dev/tcp/127.0.0.1/636' >/dev/null 2>&1; then
            return 0
        fi
        sleep 1
    done
    printf 'timeout waiting for Samba AD LDAPS\n' >&2
    return 1
}

for command in docker openssl curl python3 go grep; do
    require_command "${command}"
done
docker info >/dev/null
available_kb=$(df -Pk "${runtime_root}" | awk 'NR == 2 {print $4}')
if ((available_kb < 3 * 1024 * 1024)); then
    printf 'need at least 3 GiB free in runtime filesystem\n' >&2
    exit 1
fi

stage='runtime_material'
umask 077
install -d -m 0700 \
    "${runtime_root}/certificates" \
    "${runtime_root}/dex-data" \
    "${runtime_root}/secrets"

printf 'V11-Admin!%s\n' "$(openssl rand -hex 16)" > "${runtime_root}/secrets/administrator_password"
printf 'V11-Alice!%s\n' "$(openssl rand -hex 16)" > "${runtime_root}/secrets/alice_password"
printf 'V11-Disabled!%s\n' "$(openssl rand -hex 16)" > "${runtime_root}/secrets/disabled_password"
printf 'V11-Dex!%s\n' "$(openssl rand -hex 16)" > "${runtime_root}/secrets/dex_password"

openssl req -x509 -newkey rsa:3072 -nodes -sha256 -days 2 \
    -subj '/CN=Orquesta V11 test CA' \
    -addext 'basicConstraints=critical,CA:TRUE' \
    -addext 'keyUsage=critical,keyCertSign,cRLSign' \
    -addext 'subjectKeyIdentifier=hash' \
    -keyout "${runtime_root}/certificates/ca.key" \
    -out "${runtime_root}/certificates/ca.crt" >/dev/null 2>&1
openssl req -new -newkey rsa:3072 -nodes -sha256 \
    -subj '/CN=samba' -addext 'subjectAltName=DNS:samba' \
    -addext 'basicConstraints=critical,CA:FALSE' \
    -addext 'keyUsage=critical,digitalSignature,keyEncipherment' \
    -addext 'extendedKeyUsage=serverAuth' \
    -keyout "${runtime_root}/certificates/samba.key" \
    -out "${runtime_root}/certificates/samba.csr" >/dev/null 2>&1
openssl x509 -req -sha256 -days 2 \
    -in "${runtime_root}/certificates/samba.csr" \
    -CA "${runtime_root}/certificates/ca.crt" \
    -CAkey "${runtime_root}/certificates/ca.key" \
    -CAcreateserial -copy_extensions copy \
    -out "${runtime_root}/certificates/samba.crt" >/dev/null 2>&1
openssl req -x509 -newkey rsa:2048 -nodes -sha256 -days 2 \
    -subj '/CN=Wrong V11 test CA' \
    -addext 'basicConstraints=critical,CA:TRUE' \
    -addext 'keyUsage=critical,keyCertSign,cRLSign' \
    -keyout "${runtime_root}/certificates/wrong-ca.key" \
    -out "${runtime_root}/certificates/wrong-ca.crt" >/dev/null 2>&1

stage='build_samba'
docker build \
    --quiet \
    --file "${repo_root}/testdata/v11_dex_samba/Dockerfile.samba" \
    --tag "${samba_image}" \
    "${repo_root}/testdata/v11_dex_samba" >/dev/null

stage='start_samba'
ldaps_port=$(free_port)
docker network create "${network}" >/dev/null
docker run --detach \
    --name "${samba_container}" \
    --hostname samba \
    --network "${network}" \
    --cap-add SYS_ADMIN \
    --publish "127.0.0.1:${ldaps_port}:636" \
    --mount "type=bind,src=${runtime_root}/certificates,dst=/run/v11-certificates,readonly" \
    --mount "type=bind,src=${runtime_root}/secrets,dst=/run/v11-secrets,readonly" \
    "${samba_image}" >/dev/null
wait_for_samba

stage='configure_dex'
dex_port=$(free_port)
callback_port=$(free_port)
issuer="http://127.0.0.1:${dex_port}/dex"
redirect_uri="http://127.0.0.1:${callback_port}/callback"
dex_password=$(<"${runtime_root}/secrets/dex_password")
printf '%s\n' \
    "issuer: ${issuer}" \
    'storage:' \
    '  type: sqlite3' \
    '  config:' \
    '    file: /var/lib/dex/dex.db' \
    'web:' \
    '  http: 0.0.0.0:5556' \
    'oauth2:' \
    '  skipApprovalScreen: true' \
    'staticClients:' \
    "- id: ${CLIENT_ID}" \
    '  public: true' \
    '  name: Orquesta V11 public test client' \
    '  redirectURIs:' \
    "  - ${redirect_uri}" \
    'connectors:' \
    '- type: ldap' \
    '  id: ldap' \
    '  name: Active Directory' \
    '  config:' \
    '    host: samba:636' \
    '    insecureNoSSL: false' \
    '    insecureSkipVerify: false' \
    '    rootCA: /etc/dex/ldap-ca.crt' \
    '    bindDN: CN=Dex Bridge,CN=Users,DC=v11,DC=example,DC=test' \
    "    bindPW: '${dex_password}'" \
    '    usernamePrompt: Username' \
    '    userSearch:' \
    '      baseDN: CN=Users,DC=v11,DC=example,DC=test' \
    '      filter: (objectClass=person)' \
    '      username: sAMAccountName' \
    '      idAttr: DN' \
    '      emailAttr: userPrincipalName' \
    '      nameAttr: cn' \
    '      preferredUsernameAttr: sAMAccountName' \
    '    groupSearch:' \
    '      baseDN: CN=Users,DC=v11,DC=example,DC=test' \
    '      filter: (objectClass=group)' \
    '      userMatchers:' \
    '      - userAttr: DN' \
    '        groupAttr: member' \
    '      nameAttr: cn' > "${runtime_root}/dex.yaml"
unset dex_password

stage='start_dex'
docker run --detach \
    --name "${dex_container}" \
    --network "${network}" \
    --publish "127.0.0.1:${dex_port}:5556" \
    --user "$(id -u):$(id -g)" \
    --mount "type=bind,src=${runtime_root}/dex.yaml,dst=/etc/dex/config.yaml,readonly" \
    --mount "type=bind,src=${runtime_root}/certificates/ca.crt,dst=/etc/dex/ldap-ca.crt,readonly" \
    --mount "type=bind,src=${runtime_root}/dex-data,dst=/var/lib/dex" \
    "${DEX_IMAGE}" dex serve /etc/dex/config.yaml >/dev/null
wait_for_dex

stage='build_product_probe'
(
    cd "${repo_root}"
    GOFLAGS=-mod=vendor go build -o "${runtime_root}/v11-oidc-probe" \
        ./testdata/v11_dex_samba/probe
)

stage='exercise'
python3 "${repo_root}/testdata/v11_dex_samba/browser_flow.py" \
    --issuer "${issuer}" \
    --client-id "${CLIENT_ID}" \
    --required-group "${REQUIRED_GROUP}" \
    --callback-port "${callback_port}" \
    --probe "${runtime_root}/v11-oidc-probe" \
    --samba-container "${samba_container}" \
    --ldaps-port "${ldaps_port}" \
    --ca "${runtime_root}/certificates/ca.crt" \
    --wrong-ca "${runtime_root}/certificates/wrong-ca.crt" \
    --alice-password-file "${runtime_root}/secrets/alice_password" \
    --disabled-password-file "${runtime_root}/secrets/disabled_password"

stage='complete'
printf 'DEX_IMAGE=%s\n' "${DEX_IMAGE}"
printf 'SAMBA_BASE_IMAGE=debian@sha256:60eac759739651111db372c07be67863818726f754804b8707c90979bda511df\n'
