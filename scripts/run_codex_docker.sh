#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
STATE_DIR="${ROOT_DIR}/.codex-docker-home"
WORKSPACE_DIR="${ROOT_DIR}/.codex-sandbox-workspace"
TEMPLATE_CONFIG="${ROOT_DIR}/codex-docker/config.toml"
STATE_SYNC_MANIFEST="${ROOT_DIR}/codex-docker/state-sync-manifest.txt"
HOST_CODEX_HOME="${HOME}/.codex"

mkdir -p "${STATE_DIR}"
chmod 700 "${STATE_DIR}"

if [[ -f "${STATE_DIR}/config.toml" ]] && ! cmp -s "${TEMPLATE_CONFIG}" "${STATE_DIR}/config.toml"; then
  cp -a "${STATE_DIR}/config.toml" "${STATE_DIR}/config.toml.bak"
fi

cp "${TEMPLATE_CONFIG}" "${STATE_DIR}/config.toml"
chmod 600 "${STATE_DIR}/config.toml"

if [[ -f "${HOME}/.codex/auth.json" && ! -f "${STATE_DIR}/auth.json" ]]; then
  cp "${HOME}/.codex/auth.json" "${STATE_DIR}/auth.json"
  chmod 600 "${STATE_DIR}/auth.json"
fi

sync_host_codex_state() {
  [[ -d "${HOST_CODEX_HOME}" ]] || return 0
  [[ -f "${STATE_SYNC_MANIFEST}" ]] || return 0

  copy_file_if_exists() {
    local name="$1"
    if [[ -f "${HOST_CODEX_HOME}/${name}" ]]; then
      cp -a "${HOST_CODEX_HOME}/${name}" "${STATE_DIR}/${name}"
    fi
  }

  copy_dir_if_exists() {
    local name="$1"
    if [[ -d "${HOST_CODEX_HOME}/${name}" ]]; then
      mkdir -p "${STATE_DIR}/${name}"
      cp -a "${HOST_CODEX_HOME}/${name}/." "${STATE_DIR}/${name}/"
    fi
  }

  while IFS= read -r entry || [[ -n "${entry}" ]]; do
    entry="${entry#"${entry%%[![:space:]]*}"}"
    entry="${entry%"${entry##*[![:space:]]}"}"
    [[ -n "${entry}" ]] || continue
    [[ "${entry}" == \#* ]] && continue
    case "${entry}" in
      file:*)
        copy_file_if_exists "${entry#file:}"
        ;;
      dir:*)
        copy_dir_if_exists "${entry#dir:}"
        ;;
    esac
  done < "${STATE_SYNC_MANIFEST}"
}

if [[ "${ORQUESTA_SYNC_HOST_CODEX:-1}" != "0" ]]; then
  sync_host_codex_state
fi

bash "${ROOT_DIR}/scripts/provision_codex_skills.sh" "${STATE_DIR}"

sync_workspace_copy() {
  mkdir -p "${WORKSPACE_DIR}"

  if [[ "${ORQUESTA_REFRESH_WORKSPACE:-0}" != "1" ]] && [[ -d "${WORKSPACE_DIR}/.git" ]]; then
    return 0
  fi

  if command -v rsync >/dev/null 2>&1; then
    rsync -a --delete \
      --exclude '.codex-docker-home/' \
      --exclude '.codex-sandbox-workspace/' \
      --exclude 'backups/' \
      "${ROOT_DIR}/" "${WORKSPACE_DIR}/"
    return 0
  fi

  echo "rsync no disponible; usando copia completa con tar" >&2
  find "${WORKSPACE_DIR}" -mindepth 1 -maxdepth 1 ! -name '.git' -exec rm -rf {} +
  tar -C "${ROOT_DIR}" \
    --exclude='.codex-docker-home' \
    --exclude='.codex-sandbox-workspace' \
    --exclude='backups' \
    -cf - . | tar -C "${WORKSPACE_DIR}" -xf -
}

sync_workspace_copy

export LOCAL_UID="$(id -u)"
export LOCAL_GID="$(id -g)"

cd "${ROOT_DIR}"

docker compose -f docker-compose.codex.yml build codex-sandbox
exec docker compose -f docker-compose.codex.yml run --rm codex-sandbox codex "$@"
