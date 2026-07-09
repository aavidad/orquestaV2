#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
tmp_root="$(mktemp -d)"
trap 'rm -rf "$tmp_root"' EXIT

inner_ok="$tmp_root/inner_ok.sh"
cat > "$inner_ok" <<'SH'
#!/usr/bin/env bash
set -euo pipefail
mkdir -p "$ORQUESTA_GOLDEN_TASK_RESULT_DIR"
cat > "$ORQUESTA_GOLDEN_TASK_RESULT_DIR/result.json" <<'JSON'
{
  "task_id": "golden-launcher-test",
  "status": "complete",
  "touched_files": ["modulos/a.go", "modulos/b.go"],
  "tests": [{"name": "unit", "status": "passed"}],
  "evidence": {"kind": "synthetic"},
  "metrics": {
    "tokens": {
      "input": 11,
      "output": 7,
      "reasoning": 3,
      "cached_input": 5,
      "total": 21
    },
    "tool_calls": 2
  }
}
JSON
SH
chmod +x "$inner_ok"

ok_result="$tmp_root/ok"
ORQUESTA_GOLDEN_TASK_ID="golden-launcher-test" \
ORQUESTA_GOLDEN_TASK_RESULT_DIR="$ok_result" \
ORQUESTA_GOLDEN_METRICS_INNER_LAUNCHER="$inner_ok" \
bash "$repo_root/scripts/orquesta_golden_metrics_launcher.sh"

python3 - "$ok_result/result.json" <<'PY'
import json
import sys
from pathlib import Path

payload = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
metrics = payload["metrics"]
assert payload["status"] == "complete", payload
assert metrics["metrics_launcher"] == "orquesta_golden_metrics_launcher.v0", metrics
assert metrics["launcher_exit_code"] == 0, metrics
assert metrics["files_touched"] == 2, metrics
assert metrics["elapsed_ms"] >= 0, metrics
assert metrics["input_tokens"] == 11, metrics
assert metrics["output_tokens"] == 7, metrics
assert metrics["reasoning_tokens"] == 3, metrics
assert metrics["cached_input_tokens"] == 5, metrics
assert metrics["total_tokens"] == 21, metrics
assert metrics["tool_calls"] == 2, metrics
PY

inner_usage="$tmp_root/inner_usage.sh"
cat > "$inner_usage" <<'SH'
#!/usr/bin/env bash
set -euo pipefail
mkdir -p "$ORQUESTA_GOLDEN_TASK_RESULT_DIR"
cat > "$ORQUESTA_GOLDEN_TASK_RESULT_DIR/result.json" <<'JSON'
{
  "task_id": "golden-launcher-usage",
  "status": "complete",
  "touched_files": ["modulos/usage.go"],
  "tests": [],
  "evidence": {"usage_path": "codex_usage_accounting.json"}
}
JSON
cat > "$ORQUESTA_GOLDEN_TASK_RESULT_DIR/codex_usage_accounting.json" <<'JSON'
{
  "schema_version": "codex_usage_accounting.v0",
  "usage": {
    "input_tokens": 101,
    "output_tokens": 37,
    "reasoning_tokens": 9,
    "cached_input_tokens": 11,
    "total_tokens": 147
  },
  "quota": {"status": "observed_usage"}
}
JSON
SH
chmod +x "$inner_usage"

usage_result="$tmp_root/usage"
ORQUESTA_GOLDEN_TASK_ID="golden-launcher-usage" \
ORQUESTA_GOLDEN_TASK_RESULT_DIR="$usage_result" \
ORQUESTA_GOLDEN_METRICS_INNER_LAUNCHER="$inner_usage" \
bash "$repo_root/scripts/orquesta_golden_metrics_launcher.sh"

python3 - "$usage_result/result.json" <<'PY'
import json
import sys
from pathlib import Path

payload = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
metrics = payload["metrics"]
assert payload["status"] == "complete", payload
assert metrics["provider_usage_detected"] is True, metrics
assert metrics["provider_usage_source"] == "codex_usage_accounting.json", metrics
assert metrics["input_tokens"] == 101, metrics
assert metrics["output_tokens"] == 37, metrics
assert metrics["reasoning_tokens"] == 9, metrics
assert metrics["cached_input_tokens"] == 11, metrics
assert metrics["total_tokens"] == 147, metrics
PY

inner_fail="$tmp_root/inner_fail.sh"
cat > "$inner_fail" <<'SH'
#!/usr/bin/env bash
set -euo pipefail
exit 17
SH
chmod +x "$inner_fail"

fail_result="$tmp_root/fail"
set +e
ORQUESTA_GOLDEN_TASK_ID="golden-launcher-fail" \
ORQUESTA_GOLDEN_TASK_RESULT_DIR="$fail_result" \
ORQUESTA_GOLDEN_METRICS_INNER_LAUNCHER="$inner_fail" \
bash "$repo_root/scripts/orquesta_golden_metrics_launcher.sh"
exit_code="$?"
set -e

if [[ "$exit_code" -ne 17 ]]; then
  echo "expected exit code 17, got $exit_code" >&2
  exit 1
fi

python3 - "$fail_result/result.json" <<'PY'
import json
import sys
from pathlib import Path

payload = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
metrics = payload["metrics"]
assert payload["task_id"] == "golden-launcher-fail", payload
assert payload["status"] == "failed", payload
assert payload["launch_exit_code"] == 17, payload
assert metrics["launcher_exit_code"] == 17, metrics
assert metrics["metrics_launcher"] == "orquesta_golden_metrics_launcher.v0", metrics
assert metrics["total_tokens"] == 0, metrics
PY

echo "orquesta_golden_metrics_launcher ok"
