#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
tmp_root="$(mktemp -d "${TMPDIR:-/tmp}/orquesta-nightly-test.XXXXXX")"
trap 'rm -rf "$tmp_root"' EXIT

results_dir="$tmp_root/results"
fake_preflight="$tmp_root/fake-preflight.sh"
fake_real="$tmp_root/fake-real.sh"

cat >"$fake_preflight" <<'SH'
#!/usr/bin/env bash
set -euo pipefail
if [[ "${ORQUESTA_GOAL_FIRST_SMOKE_PREFLIGHT_ONLY:-0}" != "1" ]]; then
  echo "preflight flag missing" >&2
  exit 9
fi
echo "smoke_goal_first_app_server_preflight=ok"
echo "codex_command=/tmp/fake-codex"
echo "goal_backend=app_server_tmux"
SH
chmod +x "$fake_preflight"

ORQUESTA_NIGHTLY_RESULTS_DIR="$results_dir" \
ORQUESTA_NIGHTLY_DATE_ID="20990101" \
ORQUESTA_NIGHTLY_SMOKE_SCRIPT="$fake_preflight" \
  "$root/scripts/orquesta_smoke_nightly.sh" >"$tmp_root/preflight.out"

python3 - "$results_dir/resultado_20990101.json" <<'PY'
import json
import sys

with open(sys.argv[1], encoding="utf-8") as fh:
    payload = json.load(fh)
assert payload["mode"] == "preflight", payload
assert payload["exit_code"] == 0, payload
assert payload["phase_reached"] == "preflight_ok", payload
assert payload["real_confirmed"] is False, payload
PY

cat >"$fake_real" <<'SH'
#!/usr/bin/env bash
set -euo pipefail
if [[ "${ORQUESTA_GOAL_FIRST_SMOKE_PREFLIGHT_ONLY:-1}" == "1" ]]; then
  echo "real mode was not enabled" >&2
  exit 8
fi
if [[ "${ORQUESTA_CODEX_GOAL_FIRST_APP_SERVER_REAL_CONFIRM:-0}" != "1" ||
  "${ORQUESTA_CODEX_GOAL_FIRST_APP_SERVER_CODEX_EXECUTION_CONFIRMED:-0}" != "1" ]]; then
  echo "underlying real confirmations missing" >&2
  exit 7
fi
echo "Codex app-server preparado para tmux"
echo "compilando servidor temporal..."
echo "arrancando orquesta-server..."
echo "servidor listo: http://127.0.0.1:12345"
echo "POST /api/v0/apps/director -> HTTP 200"
echo "run_ref=run-nightly-test"
echo "goal_ref=goal-nightly-test"
echo "external_goal_ref=external-nightly-test"
echo "poll=1 mode=goal_first goal_status=complete run_status=cerrada closure_status=accepted closure_accepted=true"
echo "smoke_goal_first_app_server_real=ok"
echo "artifact_refs=2"
echo "evidence_refs=3"
echo "app_server_tmux_shutdown_ready=true"
echo "smoke_root=/tmp/nightly-fake"
SH
chmod +x "$fake_real"

ORQUESTA_NIGHTLY_RESULTS_DIR="$results_dir" \
ORQUESTA_NIGHTLY_DATE_ID="20990102" \
ORQUESTA_NIGHTLY_REAL_CONFIRM=1 \
ORQUESTA_NIGHTLY_SMOKE_SCRIPT="$fake_real" \
  "$root/scripts/orquesta_smoke_nightly.sh" >"$tmp_root/real.out"

python3 - "$results_dir/resultado_20990102.json" <<'PY'
import json
import sys

with open(sys.argv[1], encoding="utf-8") as fh:
    payload = json.load(fh)
assert payload["mode"] == "real", payload
assert payload["exit_code"] == 0, payload
assert payload["phase_reached"] == "shutdown_verified", payload
assert payload["real_confirmed"] is True, payload
assert payload["refs"]["run_ref"] == "run-nightly-test", payload
assert payload["refs"]["artifact_refs"] == "2", payload
PY

if ORQUESTA_NIGHTLY_RESULTS_DIR="$results_dir" \
  ORQUESTA_NIGHTLY_DATE_ID="20990103" \
  ORQUESTA_NIGHTLY_SMOKE_SCRIPT="$fake_preflight" \
  OPES_BASE_URL="http://127.0.0.1:1" \
    "$root/scripts/orquesta_smoke_nightly.sh" >"$tmp_root/opes.out" 2>&1; then
  echo "nightly accepted OPES env" >&2
  exit 1
fi

python3 - "$results_dir/resultado_20990103.json" <<'PY'
import json
import sys

with open(sys.argv[1], encoding="utf-8") as fh:
    payload = json.load(fh)
assert payload["status"] == "blocked", payload
assert payload["phase_reached"] == "blocked_opes_environment", payload
PY

echo "orquesta_smoke_nightly_guard_ok=true"
