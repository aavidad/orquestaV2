#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
smoke_root="${ORQUESTA_SMOKE_ROOT:-$(mktemp -d "${TMPDIR:-/tmp}/orquesta-codex-real-operational-wave.XXXXXX")}"
project_dir="$smoke_root/project"
runtime_dir="$smoke_root/runtime"
codex_home="$smoke_root/codex-home"
code_home="$codex_home/.codex"
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

if [[ -n "${ORQUESTA_OPES_BASE_URL:-}" || -n "${OPES_BASE_URL:-}" ]]; then
  echo "OPES debe estar desactivado para este smoke" >&2
  exit 2
fi

need_cmd go
go_command="$(resolve_command "${ORQUESTA_REQUIRED_TEST_GO_COMMAND:-go}")" || {
  echo "no se pudo resolver ORQUESTA_REQUIRED_TEST_GO_COMMAND" >&2
  exit 127
}

execute_codex="${ORQUESTA_CODEX_REAL_OPERATIONAL_WAVE_EXECUTE_CODEX:-0}"
test_regex='^TestCodexStackOperationalWaveFakeRuntimeV0$'
timeout_value="${ORQUESTA_CODEX_REAL_OPERATIONAL_WAVE_TIMEOUT:-180s}"

mkdir -p "$project_dir" "$runtime_dir" "$code_home" "$output_dir"

if [[ "$execute_codex" == "1" ]]; then
  if [[ "${ORQUESTA_CODEX_REAL_OPERATIONAL_WAVE_CONFIRM:-0}" != "1" ||
    "${ORQUESTA_CODEX_REAL_OPERATIONAL_WAVE_CODEX_EXECUTION_CONFIRMED:-0}" != "1" ]]; then
    echo "confirmacion doble requerida: exporta ORQUESTA_CODEX_REAL_OPERATIONAL_WAVE_CONFIRM=1 y ORQUESTA_CODEX_REAL_OPERATIONAL_WAVE_CODEX_EXECUTION_CONFIRMED=1" >&2
    exit 2
  fi
  if [[ -z "${ORQUESTA_CODEX_HOME:-}" || -z "${ORQUESTA_CODEX_CODE_HOME:-}" ]]; then
    echo "modo real requiere ORQUESTA_CODEX_HOME y ORQUESTA_CODEX_CODE_HOME explicitos" >&2
    exit 2
  fi
  codex_command="$(resolve_command "${ORQUESTA_CODEX_COMMAND:-}")" || {
    echo "falta ORQUESTA_CODEX_COMMAND con ruta/comando Codex real" >&2
    exit 2
  }
  if [[ ! -x "$codex_command" ]]; then
    echo "ORQUESTA_CODEX_COMMAND no es ejecutable: $codex_command" >&2
    exit 2
  fi
  codex_home="$ORQUESTA_CODEX_HOME"
  code_home="$ORQUESTA_CODEX_CODE_HOME"
  test_regex='^TestCodexStackRealOperationalWaveOptInV0$'
  timeout_value="${ORQUESTA_CODEX_REAL_OPERATIONAL_WAVE_TIMEOUT:-900s}"
  export ORQUESTA_CODEX_REAL_OPERATIONAL_WAVE_SMOKE=1
  export ORQUESTA_CODEX_REAL_OPERATIONAL_WAVE_CONFIRM=1
  export ORQUESTA_CODEX_REAL_OPERATIONAL_WAVE_CODEX_EXECUTION_CONFIRMED=1
  export ORQUESTA_CODEX_COMMAND="$codex_command"
  echo "codex_executed=true (opt-in: 3 agentes, WaitAgentRefs/cohorte, review, runner, cierre)"
else
  echo "codex_executed=false (fake runtime: 3 agentes, WaitAgentRefs/cohorte, review, runner, cierre)"
fi

export ORQUESTA_CODEX_PROJECT_WORKDIR="$project_dir"
export ORQUESTA_CODEX_RUNTIME_WORKDIR="$runtime_dir"
export ORQUESTA_CODEX_HOME="$codex_home"
export ORQUESTA_CODEX_CODE_HOME="$code_home"
export ORQUESTA_CODEX_PATH="${ORQUESTA_CODEX_PATH:-$PATH}"
export ORQUESTA_CODEX_APPROVAL_POLICY="${ORQUESTA_CODEX_APPROVAL_POLICY:-never}"
export ORQUESTA_CODEX_SANDBOX="${ORQUESTA_CODEX_SANDBOX:-workspace-write}"
export ORQUESTA_CODEX_MODEL="${ORQUESTA_CODEX_MODEL:-gpt-5.5}"
export ORQUESTA_CODEX_REASONING_EFFORT="${ORQUESTA_CODEX_REASONING_EFFORT:-high}"
export ORQUESTA_CODEX_SMOKE_TIMEOUT_SECONDS="${ORQUESTA_CODEX_SMOKE_TIMEOUT_SECONDS:-720}"
export ORQUESTA_REQUIRED_TEST_GO_COMMAND="$go_command"
export ORQUESTA_REQUIRED_TEST_OUTPUT_DIR="$output_dir"
export ORQUESTA_OPES_BASE_URL=""
export OPES_BASE_URL=""

echo "smoke_root=$smoke_root"
echo "required_test_go_command=$go_command"

(
  cd "$repo_root"
  go test -count=1 -timeout "$timeout_value" \
    ./modulos/orquesta-app-codex-stack \
    -run "$test_regex" \
    -v
)
