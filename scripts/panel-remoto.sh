#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CERT_DIR="${ROOT_DIR}/certs/panel-mtls"

HOST="${HOST:-0.0.0.0}"
PORT="${PORT:-8443}"
TLS_CERT="${TLS_CERT:-${CERT_DIR}/server.crt}"
TLS_KEY="${TLS_KEY:-${CERT_DIR}/server.key}"
TLS_CLIENT_CA="${TLS_CLIENT_CA:-${CERT_DIR}/ca.crt}"

if [[ ! -f "${TLS_CERT}" ]]; then
  echo "Falta TLS_CERT: ${TLS_CERT}" >&2
  exit 1
fi

if [[ ! -f "${TLS_KEY}" ]]; then
  echo "Falta TLS_KEY: ${TLS_KEY}" >&2
  exit 1
fi

if [[ ! -f "${TLS_CLIENT_CA}" ]]; then
  echo "Falta TLS_CLIENT_CA: ${TLS_CLIENT_CA}" >&2
  exit 1
fi

echo "Arrancando panel remoto Orquesta"
echo "  host: ${HOST}"
echo "  port: ${PORT}"
echo "  cert: ${TLS_CERT}"
echo "  key:  ${TLS_KEY}"
echo "  ca:   ${TLS_CLIENT_CA}"
echo

exec "${ROOT_DIR}/orquesta" serve \
  --host "${HOST}" \
  --puerto "${PORT}" \
  --tls-cert "${TLS_CERT}" \
  --tls-key "${TLS_KEY}" \
  --tls-client-ca "${TLS_CLIENT_CA}"
