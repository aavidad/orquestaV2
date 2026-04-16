#!/usr/bin/env bash
set -euo pipefail

CODEX_HOME_DIR="${1:-${CODEX_HOME:-${HOME}/.codex}}"
SKILLS_DIR="${CODEX_HOME_DIR}/skills"
INSTALL_CAVEMAN="${ORQUESTA_INSTALL_CAVEMAN:-1}"
CAVEMAN_SOURCE_DIR="${ORQUESTA_CAVEMAN_SOURCE_DIR:-}"
CAVEMAN_REPO_URL="${ORQUESTA_CAVEMAN_REPO_URL:-https://github.com/JuliusBrussee/caveman.git}"
CAVEMAN_REF="${ORQUESTA_CAVEMAN_REF:-main}"

mkdir -p "${SKILLS_DIR}"

copy_skill_dir() {
  local src_root="$1"
  local skill_name="$2"
  local src="${src_root}/skills/${skill_name}"
  local dest="${SKILLS_DIR}/${skill_name}"

  if [[ ! -f "${src}/SKILL.md" ]]; then
    return 1
  fi

  rm -rf "${dest}"
  mkdir -p "${dest}"
  cp -a "${src}/." "${dest}/"
}

install_caveman_from_local() {
  local root="$1"
  copy_skill_dir "${root}" caveman
  copy_skill_dir "${root}" compress
}

install_caveman_from_git() {
  local tmpdir
  tmpdir="$(mktemp -d)"
  trap 'rm -rf "${tmpdir}"' EXIT
  git clone --depth 1 --branch "${CAVEMAN_REF}" "${CAVEMAN_REPO_URL}" "${tmpdir}/caveman" >/dev/null 2>&1
  install_caveman_from_local "${tmpdir}/caveman"
}

if [[ "${INSTALL_CAVEMAN}" == "1" ]]; then
  if [[ -f "${SKILLS_DIR}/caveman/SKILL.md" && -f "${SKILLS_DIR}/compress/SKILL.md" ]]; then
    echo "skills caveman/compress ya presentes en ${SKILLS_DIR}"
    exit 0
  fi

  if [[ -n "${CAVEMAN_SOURCE_DIR}" && -f "${CAVEMAN_SOURCE_DIR}/skills/caveman/SKILL.md" ]]; then
    install_caveman_from_local "${CAVEMAN_SOURCE_DIR}"
    echo "skills caveman/compress instalados desde fuente local en ${SKILLS_DIR}"
    exit 0
  fi

  install_caveman_from_git
  echo "skills caveman/compress instalados desde ${CAVEMAN_REPO_URL} en ${SKILLS_DIR}"
fi
