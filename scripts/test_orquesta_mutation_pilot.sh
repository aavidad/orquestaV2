#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
tmp_root="$(mktemp -d "${TMPDIR:-/tmp}/orquesta-mutation-pilot-test.XXXXXX")"
trap 'rm -rf "$tmp_root"' EXIT

results_dir="$tmp_root/nightly"
fake_tool="$tmp_root/fake-go-mutesting.sh"
invocations="$tmp_root/invocations.log"

cat >"$fake_tool" <<'SH'
#!/usr/bin/env bash
set -euo pipefail

if [[ "$#" -ne 1 ]]; then
  echo "fake tool expected exactly one package argument" >&2
  exit 9
fi

printf '%s\n' "$1" >>"$ORQUESTA_MUTATION_PILOT_FAKE_INVOCATIONS"

case "$1" in
  ./modulos/orquesta-estado-vivo)
    echo "mutation_score=0.75"
    echo "mutations_total=4"
    echo "mutants_killed=3"
    echo "mutants_survived=1"
    echo "SURVIVED: modulos/orquesta-estado-vivo/reglas_precedencia_v0.go:42 cambio de precedencia wait_external"
    ;;
  ./modulos/orquesta-goal)
    echo "The mutation score is 80% (4 killed, 1 survived, total is 5)"
    echo "SURVIVED: modulos/orquesta-goal/validation_v0.go:120 validacion de write-set debilitada"
    ;;
  *)
    echo "paquete fuera de alcance: $1" >&2
    exit 8
    ;;
esac
SH
chmod +x "$fake_tool"

if ORQUESTA_NIGHTLY_RESULTS_DIR="$results_dir" \
  ORQUESTA_MUTATION_PILOT_TOOL="$fake_tool" \
  "$root/scripts/orquesta_mutation_pilot.sh" >"$tmp_root/missing-confirm.out" 2>"$tmp_root/missing-confirm.err"; then
  echo "mutation pilot accepted missing opt-in confirmation" >&2
  exit 1
fi
grep -q "ORQUESTA_MUTATION_PILOT_CONFIRM" "$tmp_root/missing-confirm.err"

if ORQUESTA_MUTATION_PILOT_CONFIRM=MUTATION_PILOT_OPT_IN \
  ORQUESTA_NIGHTLY_RESULTS_DIR="$results_dir" \
  ORQUESTA_MUTATION_PILOT_TOOL="$fake_tool" \
  "$root/scripts/orquesta_mutation_pilot.sh" ./modulos/orquesta-estado-vivo \
    >"$tmp_root/arg.out" 2>"$tmp_root/arg.err"; then
  echo "mutation pilot accepted package arguments" >&2
  exit 1
fi
grep -q "no acepta argumentos" "$tmp_root/arg.err"

ORQUESTA_MUTATION_PILOT_FAKE_INVOCATIONS="$invocations" \
ORQUESTA_MUTATION_PILOT_CONFIRM=MUTATION_PILOT_OPT_IN \
ORQUESTA_NIGHTLY_RESULTS_DIR="$results_dir" \
ORQUESTA_MUTATION_PILOT_DATE_ID="20990104" \
ORQUESTA_MUTATION_PILOT_TOOL="$fake_tool" \
  "$root/scripts/orquesta_mutation_pilot.sh" >"$tmp_root/run.out"

result_json="$results_dir/mutation-testing-pilot/resultado_mutation_20990104.json"
test -f "$result_json"
grep -q "mutation_pilot_result=$result_json" "$tmp_root/run.out"

python3 - "$result_json" "$invocations" <<'PY'
import json
import sys
from pathlib import Path

payload = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
invocations = Path(sys.argv[2]).read_text(encoding="utf-8").splitlines()

expected_packages = [
    "./modulos/orquesta-estado-vivo",
    "./modulos/orquesta-goal",
]

if payload["schema_version"] != "orquesta_mutation_pilot.v0":
    raise SystemExit("unexpected schema")
if payload["status"] != "valid_pilot_result":
    raise SystemExit(f"unexpected status {payload['status']!r}")
if payload["date_utc"] != "20990104":
    raise SystemExit("date id was not preserved")
if payload["target_packages"] != expected_packages:
    raise SystemExit(f"unexpected target packages: {payload['target_packages']!r}")
if invocations != expected_packages:
    raise SystemExit(f"tool invoked with unexpected packages: {invocations!r}")
if payload["policy"]["ci_default"]:
    raise SystemExit("mutation pilot must not be CI default")
if payload["policy"]["go_mod_dependency_required"]:
    raise SystemExit("mutation pilot must not require go.mod dependency")

packages = {entry["package"]: entry for entry in payload["packages"]}
estado = packages["./modulos/orquesta-estado-vivo"]
goal = packages["./modulos/orquesta-goal"]

if estado["score"] != 0.75 or estado["mutants_survived"] != 1:
    raise SystemExit(f"estado-vivo score not parsed: {estado!r}")
if goal["score"] != 0.8 or goal["mutations_total"] != 5:
    raise SystemExit(f"goal score not parsed: {goal!r}")
if len(payload["surviving_mutants_top"]) != 2:
    raise SystemExit(f"survivor top not collected: {payload['surviving_mutants_top']!r}")
PY

echo "orquesta_mutation_pilot_shell_test_ok=true"
