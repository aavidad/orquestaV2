#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
tmp_root="$(mktemp -d)"
trap 'rm -rf "$tmp_root"' EXIT

fake_launcher="$tmp_root/fake_launcher.sh"
cat > "$fake_launcher" <<'SH'
#!/usr/bin/env bash
set -euo pipefail
python3 - "$ORQUESTA_GOLDEN_TASK_REQUEST" "$ORQUESTA_GOLDEN_TASK_RESULT_DIR/result.json" <<'PY'
import json
import sys
from pathlib import Path

request = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
task = request["task"]
result = {
    "task_id": task["task_id"],
    "status": "complete",
    "touched_files": task.get("expected_files", []),
    "artifact_paths": task.get("expected_files", []),
    "tests": [{"command": command, "status": "passed"} for command in task.get("required_tests", [])],
    "evidence": {"fake_launcher": True},
    "metrics": {"input_tokens": 10, "output_tokens": 5, "total_tokens": 15},
}
Path(sys.argv[2]).write_text(json.dumps(result, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
PY
SH
chmod +x "$fake_launcher"

run_root="$tmp_root/run"
output="$tmp_root/eval.json"
ORQUESTA_GOLDEN_EVALS_CONFIRM=isolated \
"$repo_root/scripts/orquesta_golden_evals.sh" \
  --run \
  --task golden-new-app-smoke-v0 \
  --results-dir "$run_root" \
  --launcher-command "$fake_launcher" \
  --output "$output"

python3 - "$output" <<'PY'
import json
import sys
from pathlib import Path

payload = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
assert payload["status"] == "passed", payload
assert payload["summary"]["total"] == 1, payload["summary"]
assert payload["summary"]["passed"] == 1, payload["summary"]
assert payload["summary"]["failed"] == 0, payload["summary"]
assert payload["summary"]["score"] == 1.0, payload["summary"]
assert [task["task_id"] for task in payload["tasks"]] == ["golden-new-app-smoke-v0"], payload["tasks"]
assert payload["summary"]["metrics"]["tasks_with_metrics"] == 1, payload["summary"]["metrics"]
PY

evaluate_output="$tmp_root/eval-existing.json"
"$repo_root/scripts/orquesta_golden_evals.sh" \
  --evaluate \
  --task golden-new-app-smoke-v0 \
  --results-dir "$run_root/results" \
  --output "$evaluate_output"

python3 - "$evaluate_output" <<'PY'
import json
import sys
from pathlib import Path

payload = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
assert payload["status"] == "passed", payload
assert payload["summary"]["total"] == 1, payload["summary"]
assert [task["task_id"] for task in payload["tasks"]] == ["golden-new-app-smoke-v0"], payload["tasks"]
PY

echo "orquesta_golden_evals ok"
