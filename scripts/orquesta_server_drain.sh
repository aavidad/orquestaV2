#!/usr/bin/env bash
# Entrada de operador F3-R2. Dry-run por defecto; la lógica de proceso vive
# fuera del core en un adaptador local fail-closed.

set -Eeuo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PYTHON="${ORQUESTA_DRAIN_PYTHON:-python3}"

if [ -n "${ORQUESTA_DRAIN_PYTHON+x}" ] && [ "${DRAIN_TEST_MODE:-0}" != "1" ]; then
  echo "orquesta_server_drain=not_ok drain_status=refused preflight_error=test_override_forbidden_in_real_mode" >&2
  exit 2
fi
command -v "$PYTHON" >/dev/null 2>&1 || {
  echo "orquesta_server_drain=not_ok drain_status=refused preflight_error=python3_missing" >&2
  exit 2
}
"$PYTHON" - <<'PY'
import os, signal, sys
if sys.version_info < (3, 10):
    raise SystemExit("python_3_10_or_newer_required")
if not hasattr(os, "pidfd_open") or not hasattr(signal, "pidfd_send_signal"):
    raise SystemExit("python_pidfd_api_required")
PY

for tool in flock tmux curl; do
  command -v "$tool" >/dev/null 2>&1 || {
    echo "orquesta_server_drain=not_ok drain_status=refused preflight_error=required_tool_missing:$tool" >&2
    exit 2
  }
done

exec "$PYTHON" "$ROOT/scripts/lib/orquesta_drain_runtime.py" "$@"
