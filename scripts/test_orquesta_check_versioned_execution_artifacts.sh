#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
script="$ROOT/scripts/orquesta_check_versioned_execution_artifacts.sh"
workdir="$(mktemp -d "${TMPDIR:-/tmp}/orquesta-versioned-artifacts-test.XXXXXX")"
trap 'rm -rf "$workdir"' EXIT

repo_report="$(cd "$ROOT" && "$script" --json)"
REPORT="$repo_report" python3 - <<'PY'
import json
import os

report = json.loads(os.environ["REPORT"])
if report["status"] != "passed" or report["tracked_candidate_count"] != 68:
    raise SystemExit(f"unexpected repository report: {report}")
if report["categories"] != {
    "historical_evidence": 5,
    "movable_runtime_artifact": 58,
    "retain_pending_reference": 1,
    "test_fixture": 4,
}:
    raise SystemExit(f"unexpected categories: {report}")
PY

fixture="$workdir/fixture"
mkdir -p "$fixture/docs/auditorias" "$fixture/modulos/example/docs"
git -C "$fixture" init -q
git -C "$fixture" config user.email test@example.invalid
git -C "$fixture" config user.name test
printf 'checkpoint\n' >"$fixture/modulos/example/docs/checkpoint_started_goal-ref-task-autoprogramming-fixture.txt"
cat >"$fixture/docs/auditorias/audit.json" <<'JSON'
{"schema_version":"orquesta_s13_versioned_execution_artifact_audit.v0","entries":[{"path":"modulos/example/docs/checkpoint_started_goal-ref-task-autoprogramming-fixture.txt","category":"test_fixture"}]}
JSON
git -C "$fixture" add .
git -C "$fixture" commit -qm fixture

fixture_report="$("$script" --root "$fixture" --audit "$fixture/docs/auditorias/audit.json" --json)"
REPORT="$fixture_report" python3 - <<'PY'
import json
import os

report = json.loads(os.environ["REPORT"])
if report["status"] != "passed" or report["tracked_candidate_count"] != 1:
    raise SystemExit(f"unexpected fixture report: {report}")
PY

printf 'result\n' >"$fixture/modulos/example/docs/orquesta_goal_result_goal-ref-task-autoprogramming-unclassified.json"
git -C "$fixture" add .
set +e
"$script" --root "$fixture" --audit "$fixture/docs/auditorias/audit.json" >/dev/null
status=$?
set -e
if [ "$status" -eq 0 ]; then
  echo "expected unclassified versioned execution artifact to fail" >&2
  exit 1
fi

echo "orquesta_check_versioned_execution_artifacts_ok=true"
