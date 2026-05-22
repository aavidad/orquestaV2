#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

DEFAULT_SEQUENCE="draft_content_block,generate_visual_asset,review_legal,review_pedagogical,review_quality,validate_topic,assemble_topic"
MODE="${ORQUESTA_OPES_DERIVATIVES_SMOKE_MODE:-dry-run-once}"
SMOKE_ID="${SMOKE_ID:-$(date -u +%Y%m%dT%H%M%SZ)}"
SMOKE_OUT_DIR="${SMOKE_OUT_DIR:-/tmp/opes-salidas/derivatives-$SMOKE_ID}"
OPES_BASE_URL_EFFECTIVE="${ORQUESTA_OPES_BASE_URL:-${OPES_BASE_URL:-}}"
ORQUESTA_BASE_URL_EFFECTIVE="${ORQUESTA_BASE_URL:-}"
SEQUENCE="${ORQUESTA_OPES_BRIDGE_JOB_TYPE_SEQUENCE:-$DEFAULT_SEQUENCE}"
LIMIT="${ORQUESTA_OPES_BRIDGE_LIMIT:-1}"
MAX_TICKS="${ORQUESTA_OPES_BRIDGE_MAX_TICKS:-1}"

require_tool() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "falta herramienta requerida: $1" >&2
    exit 2
  fi
}

is_local_url() {
  case "$1" in
    http://127.0.0.1|http://127.0.0.1:*|http://localhost|http://localhost:*|http://[::1]|http://[::1]:*)
      return 0
      ;;
    *)
      return 1
      ;;
  esac
}

require_temporal_opes() {
  if [[ "${ORQUESTA_OPES_DERIVATIVES_SMOKE_CONFIRM:-0}" != "1" ]]; then
    echo "smoke derivados OPES desactivado: exporta ORQUESTA_OPES_DERIVATIVES_SMOKE_CONFIRM=1" >&2
    exit 2
  fi
  if [[ "${ORQUESTA_OPES_TEMPORAL_CONFIRM:-0}" != "1" ]]; then
    echo "falta confirmacion de instancia OPES temporal: exporta ORQUESTA_OPES_TEMPORAL_CONFIRM=1" >&2
    exit 2
  fi
  if [[ -z "$OPES_BASE_URL_EFFECTIVE" ]]; then
    echo "falta ORQUESTA_OPES_BASE_URL u OPES_BASE_URL apuntando a OPES temporal" >&2
    exit 2
  fi
  if ! is_local_url "$OPES_BASE_URL_EFFECTIVE" &&
    [[ "${ORQUESTA_OPES_ALLOW_NONLOCAL_TEMPORAL:-0}" != "1" ]]; then
    echo "OPES_BASE_URL no parece local: $OPES_BASE_URL_EFFECTIVE" >&2
    echo "si es temporal no local, exporta ORQUESTA_OPES_ALLOW_NONLOCAL_TEMPORAL=1" >&2
    exit 2
  fi
}

require_sequence_only() {
  if [[ -n "${ORQUESTA_OPES_BRIDGE_JOB_TYPE:-}" ]]; then
    echo "no mezclar ORQUESTA_OPES_BRIDGE_JOB_TYPE con JOB_TYPE_SEQUENCE en derivados" >&2
    exit 2
  fi
  if [[ -n "${ORQUESTA_OPES_BRIDGE_JOB_REF:-}" ]]; then
    echo "no mezclar ORQUESTA_OPES_BRIDGE_JOB_REF con JOB_TYPE_SEQUENCE en derivados" >&2
    exit 2
  fi
  if [[ -z "$SEQUENCE" ]]; then
    echo "ORQUESTA_OPES_BRIDGE_JOB_TYPE_SEQUENCE no puede estar vacio" >&2
    exit 2
  fi
}

write_metadata() {
  mkdir -p "$SMOKE_OUT_DIR"
  {
    echo "smoke_id=$SMOKE_ID"
    echo "mode=$MODE"
    echo "opes_base_url=$OPES_BASE_URL_EFFECTIVE"
    echo "orquesta_base_url=$ORQUESTA_BASE_URL_EFFECTIVE"
    echo "sequence=$SEQUENCE"
    echo "limit=$LIMIT"
    echo "max_ticks=$MAX_TICKS"
    echo "output_dir=$SMOKE_OUT_DIR"
  } >"$SMOKE_OUT_DIR/metadata.txt"
}

run_dry_run_once() {
  export ORQUESTA_OPES_BASE_URL="$OPES_BASE_URL_EFFECTIVE"
  export ORQUESTA_OPES_BRIDGE_DRY_RUN=1
  export ORQUESTA_OPES_BRIDGE_LIMIT="$LIMIT"
  export ORQUESTA_OPES_BRIDGE_JOB_TYPE_SEQUENCE="$SEQUENCE"
  go run ./cmd/orquesta-server opes-drain-once |
    tee "$SMOKE_OUT_DIR/opes_derivatives_dry_run_summary.json"
}

run_execute_drain_once() {
  if [[ "${ORQUESTA_OPES_DERIVATIVES_EXECUTE:-0}" != "1" ]]; then
    echo "falta confirmacion de efectos: exporta ORQUESTA_OPES_DERIVATIVES_EXECUTE=1" >&2
    exit 2
  fi
  if [[ -z "$ORQUESTA_BASE_URL_EFFECTIVE" ]]; then
    echo "falta ORQUESTA_BASE_URL explicito para crear runs desde derivados" >&2
    exit 2
  fi
  export ORQUESTA_OPES_BASE_URL="$OPES_BASE_URL_EFFECTIVE"
  export ORQUESTA_BASE_URL="$ORQUESTA_BASE_URL_EFFECTIVE"
  export ORQUESTA_OPES_BRIDGE_CONFIRM=1
  export ORQUESTA_OPES_BRIDGE_DRY_RUN=0
  export ORQUESTA_OPES_BRIDGE_LIMIT="$LIMIT"
  export ORQUESTA_OPES_BRIDGE_JOB_TYPE_SEQUENCE="$SEQUENCE"
  go run ./cmd/orquesta-server opes-drain-once |
    tee "$SMOKE_OUT_DIR/opes_derivatives_drain_summary.json"
}

main() {
  require_tool go
  require_tool tee
  require_temporal_opes
  require_sequence_only
  write_metadata
  cd "$repo_root"

  case "$MODE" in
    dry-run-once)
      run_dry_run_once
      ;;
    drain-once)
      run_execute_drain_once
      ;;
    *)
      echo "modo no soportado: $MODE (usa dry-run-once o drain-once)" >&2
      exit 2
      ;;
  esac
}

main "$@"
