#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
smoke_root="${ORQUESTA_SMOKE_ROOT:-$(mktemp -d "${TMPDIR:-/tmp}/orquesta-codex-real-required-test-runner.XXXXXX")}"
project_dir="$smoke_root/project"
runtime_dir="$smoke_root/runtime"
codex_home="$smoke_root/codex-home"
output_dir="$smoke_root/required-test-output"
keep_dir="${ORQUESTA_KEEP_SMOKE_DIR:-0}"

cleanup() {
  if [[ "$keep_dir" == "1" ]]; then
    echo "directorio conservado: $smoke_root" >&2
  else
    rm -rf "$smoke_root"
  fi
}
trap cleanup EXIT

need_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "falta comando requerido: $1" >&2
    exit 127
  fi
}

resolve_command() {
  local raw="$1"
  if [[ -z "$raw" ]]; then
    return 1
  fi
  if [[ "$raw" == /* ]]; then
    printf '%s' "$raw"
    return 0
  fi
  command -v "$raw"
}

if [[ "${ORQUESTA_CODEX_REAL_REQUIRED_TEST_RUNNER_CONFIRM:-0}" != "1" ]]; then
  echo "smoke desactivado: exporta ORQUESTA_CODEX_REAL_REQUIRED_TEST_RUNNER_CONFIRM=1" >&2
  exit 2
fi
if [[ "${ORQUESTA_CODEX_REAL_REQUIRED_TEST_RUNNER_NO_CODEX_EXECUTION_CONFIRMED:-0}" != "1" ]]; then
  echo "segunda confirmacion requerida: exporta ORQUESTA_CODEX_REAL_REQUIRED_TEST_RUNNER_NO_CODEX_EXECUTION_CONFIRMED=1" >&2
  exit 2
fi

need_cmd go

codex_command="$(resolve_command "${ORQUESTA_CODEX_COMMAND:-}")" || {
  echo "falta ORQUESTA_CODEX_COMMAND con ruta/comando Codex real; solo se valida configuracion, no se ejecuta Codex" >&2
  exit 2
}
if [[ ! -x "$codex_command" ]]; then
  echo "ORQUESTA_CODEX_COMMAND no es ejecutable: $codex_command" >&2
  exit 2
fi
go_command="$(resolve_command "${ORQUESTA_REQUIRED_TEST_GO_COMMAND:-go}")" || {
  echo "no se pudo resolver ORQUESTA_REQUIRED_TEST_GO_COMMAND" >&2
  exit 127
}

mkdir -p "$project_dir" "$runtime_dir" "$codex_home/.codex" "$output_dir"

export ORQUESTA_CODEX_REAL_REQUIRED_TEST_RUNNER_SMOKE=1
export ORQUESTA_CODEX_REAL_REQUIRED_TEST_RUNNER_CONFIRM=1
export ORQUESTA_CODEX_REAL_REQUIRED_TEST_RUNNER_NO_CODEX_EXECUTION_CONFIRMED=1
export ORQUESTA_CODEX_COMMAND="$codex_command"
export ORQUESTA_CODEX_PROJECT_WORKDIR="$project_dir"
export ORQUESTA_CODEX_RUNTIME_WORKDIR="$runtime_dir"
export ORQUESTA_CODEX_HOME="$codex_home"
export ORQUESTA_CODEX_CODE_HOME="$codex_home/.codex"
export ORQUESTA_CODEX_PATH="${ORQUESTA_CODEX_PATH:-$PATH}"
export ORQUESTA_CODEX_APPROVAL_POLICY="${ORQUESTA_CODEX_APPROVAL_POLICY:-never}"
export ORQUESTA_CODEX_SANDBOX="${ORQUESTA_CODEX_SANDBOX:-workspace-write}"
export ORQUESTA_CODEX_MODEL="${ORQUESTA_CODEX_MODEL:-gpt-5.5}"
export ORQUESTA_CODEX_REASONING_EFFORT="${ORQUESTA_CODEX_REASONING_EFFORT:-medium}"
export ORQUESTA_CODEX_SMOKE_TIMEOUT_SECONDS="${ORQUESTA_CODEX_SMOKE_TIMEOUT_SECONDS:-180}"
export ORQUESTA_REQUIRED_TEST_GO_COMMAND="$go_command"
export ORQUESTA_REQUIRED_TEST_OUTPUT_DIR="$output_dir"
export ORQUESTA_OPES_BASE_URL=""
export OPES_BASE_URL=""

echo "smoke_root=$smoke_root"
echo "codex_config_validated=$codex_command"
echo "required_test_go_command=$go_command"
echo "codex_executed=false (el test valida config y snapshot vacio; no llama LaunchV0)"

(
  cd "$repo_root"
  go test -count=1 -timeout "${ORQUESTA_CODEX_REAL_REQUIRED_TEST_RUNNER_TIMEOUT:-240s}" \
    ./modulos/orquesta-app-codex-stack \
    -run '^TestCodexStackRealRequiredTestRunnerOptInV0$' \
    -v
)
