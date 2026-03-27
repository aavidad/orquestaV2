#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
STATE_DIR="${ROOT_DIR}/.codex-docker-home"
TEMPLATE_CONFIG="${ROOT_DIR}/codex-docker/config.toml"

mkdir -p "${STATE_DIR}"
chmod 700 "${STATE_DIR}"

if [[ ! -f "${STATE_DIR}/config.toml" ]]; then
  cp "${TEMPLATE_CONFIG}" "${STATE_DIR}/config.toml"
  chmod 600 "${STATE_DIR}/config.toml"
fi

if [[ -f "${HOME}/.codex/auth.json" && ! -f "${STATE_DIR}/auth.json" ]]; then
  cp "${HOME}/.codex/auth.json" "${STATE_DIR}/auth.json"
  chmod 600 "${STATE_DIR}/auth.json"
fi

export LOCAL_UID="$(id -u)"
export LOCAL_GID="$(id -g)"

cd "${ROOT_DIR}"

docker compose -f docker-compose.codex.yml build codex-sandbox
exec docker compose -f docker-compose.codex.yml run --rm codex-sandbox codex "$@"
