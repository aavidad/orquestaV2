#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
tmp_root="$(mktemp -d)"
trap 'rm -rf "$tmp_root"' EXIT

baseline_ok="$tmp_root/baseline_ok.sh"
variant_ok="$tmp_root/variant_ok.sh"
baseline_fail_ok="$tmp_root/baseline_fail_ok.sh"
variant_fail="$tmp_root/variant_fail.sh"

cat > "$baseline_ok" <<'SH'
#!/usr/bin/env bash
set -euo pipefail
mkdir -p "$ORQUESTA_GOLDEN_TASK_RESULT_DIR"
cat > "$ORQUESTA_GOLDEN_TASK_RESULT_DIR/result.json" <<'JSON'
{
  "task_id": "golden-ab-test",
  "status": "complete",
  "touched_files": ["app/a.go", "app/b.go"],
  "artifact_paths": ["app/a.go"],
  "tests": [{"command": "go test ./...", "status": "passed"}],
  "evidence": {"baseline": true},
  "metrics": {
    "input_tokens": 70,
    "output_tokens": 20,
    "reasoning_tokens": 10,
    "total_tokens": 100,
    "files_touched": 2,
    "new_files_count": 1,
    "lines_added": 40,
    "rework_count": 1
  }
}
JSON
SH
chmod +x "$baseline_ok"

cat > "$variant_ok" <<'SH'
#!/usr/bin/env bash
set -euo pipefail
test "$GOLDEN_AB_ARM" = "variant"
test "$GOLDEN_AGENT_SKILL_REFS" = "skill-ref-orquesta-programacion-minima-v0"
mkdir -p "$ORQUESTA_GOLDEN_TASK_RESULT_DIR"
cat > "$ORQUESTA_GOLDEN_TASK_RESULT_DIR/result.json" <<'JSON'
{
  "task_id": "golden-ab-test",
  "status": "complete",
  "touched_files": ["app/a.go"],
  "artifact_paths": ["app/a.go"],
  "tests": [{"command": "go test ./...", "status": "passed"}],
  "evidence": {"variant": true},
  "metrics": {
    "tokens": {
      "input": 50,
      "output": 15,
      "reasoning": 5,
      "total": 70
    },
    "files_touched": 1,
    "new_files_count": 0,
    "lines_added": 18,
    "rework_count": 0
  }
}
JSON
SH
chmod +x "$variant_ok"

ok_result="$tmp_root/ok"
ORQUESTA_GOLDEN_TASK_ID="golden-ab-test" \
ORQUESTA_GOLDEN_TASK_REQUEST="$tmp_root/request.json" \
ORQUESTA_GOLDEN_TASK_RESULT_DIR="$ok_result" \
bash "$repo_root/scripts/orquesta_golden_ab_launcher.sh" \
  --baseline-command "$baseline_ok" \
  --variant-command "$variant_ok" \
  --variant-skill-ref skill-ref-orquesta-programacion-minima-v0

python3 - "$ok_result/result.json" <<'PY'
import json
import sys
from pathlib import Path

payload = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
metrics = payload["metrics"]
evidence = payload["evidence"]["ab_comparison"]
assert payload["schema_version"] == "orquesta_golden_ab_result.v0", payload
assert payload["status"] == "complete", payload
assert payload["primary_arm"] == "variant", payload
assert payload["touched_files"] == ["app/a.go"], payload
assert metrics["ab_launcher"] == "orquesta_golden_ab_launcher.v0", metrics
assert metrics["total_tokens"] == 70, metrics
assert metrics["baseline_total_tokens"] == 100, metrics
assert metrics["variant_total_tokens"] == 70, metrics
assert metrics["token_delta"] == -30, metrics
assert metrics["files_touched_delta"] == -1, metrics
assert metrics["lines_added_delta"] == -22, metrics
assert metrics["rework_delta"] == -1, metrics
assert evidence["failed_arms"] == [], evidence
assert evidence["variant_skill_refs"] == ["skill-ref-orquesta-programacion-minima-v0"], evidence
PY

cat > "$baseline_fail_ok" <<'SH'
#!/usr/bin/env bash
set -euo pipefail
mkdir -p "$ORQUESTA_GOLDEN_TASK_RESULT_DIR"
cat > "$ORQUESTA_GOLDEN_TASK_RESULT_DIR/result.json" <<'JSON'
{
  "task_id": "golden-ab-fail",
  "status": "complete",
  "touched_files": [],
  "tests": [],
  "evidence": {},
  "metrics": {"total_tokens": 12}
}
JSON
SH
chmod +x "$baseline_fail_ok"

cat > "$variant_fail" <<'SH'
#!/usr/bin/env bash
set -euo pipefail
exit 12
SH
chmod +x "$variant_fail"

fail_result="$tmp_root/fail"
ORQUESTA_GOLDEN_TASK_ID="golden-ab-fail" \
ORQUESTA_GOLDEN_TASK_REQUEST="$tmp_root/request.json" \
ORQUESTA_GOLDEN_TASK_RESULT_DIR="$fail_result" \
bash "$repo_root/scripts/orquesta_golden_ab_launcher.sh" \
  --baseline-command "$baseline_fail_ok" \
  --variant-command "$variant_fail"

python3 - "$fail_result/result.json" <<'PY'
import json
import sys
from pathlib import Path

payload = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
metrics = payload["metrics"]
evidence = payload["evidence"]["ab_comparison"]
assert payload["status"] == "failed", payload
assert payload["launch_exit_code"] == 12, payload
assert metrics["baseline_launcher_exit_code"] == 0, metrics
assert metrics["variant_launcher_exit_code"] == 12, metrics
assert metrics["ab_failed_arms_count"] == 1, metrics
assert evidence["failed_arms"] == ["variant"], evidence
assert payload["arms"]["variant"]["status"] == "failed", payload["arms"]["variant"]
PY

echo "orquesta_golden_ab_launcher ok"
