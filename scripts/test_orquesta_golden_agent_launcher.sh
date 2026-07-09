#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
tmp_root="$(mktemp -d)"
trap 'rm -rf "$tmp_root"' EXIT

request="$tmp_root/request.json"
cat > "$request" <<'JSON'
{
  "schema_version": "orquesta_golden_task_request.v0",
  "task_set_ref": "golden-test-set",
  "task": {
    "task_id": "golden-agent-test",
    "task_class": "script_contract_fix",
    "objective": "Corregir una regresion con diff minimo y test focal.",
    "write_set": ["scripts", "docs/runbooks"],
    "expected_files": ["scripts/test_autoprogramming_fast.sh"],
    "required_tests": ["bash -n scripts/test_autoprogramming_fast.sh"],
    "forbidden_path_prefixes": ["OPES"]
  },
  "result_contract": {
    "result_path": "results/golden-agent-test/result.json",
    "required_status": "complete",
    "required_checks": ["tests_pass", "expected_files_present", "writeset_only"]
  }
}
JSON

agent_ok="$tmp_root/agent_ok.sh"
cat > "$agent_ok" <<'SH'
#!/usr/bin/env bash
set -euo pipefail
test "$GOLDEN_AGENT_TASK_ID" = "golden-agent-test"
test "$GOLDEN_AGENT_SKILL_REFS" = "skill-ref-orquesta-programacion-minima-v0"
grep -q "Corregir una regresion" "$GOLDEN_AGENT_PROMPT_PATH"
grep -q "scripts/test_autoprogramming_fast.sh" "$GOLDEN_AGENT_PROMPT_PATH"
cat > "$GOLDEN_AGENT_RESULT_PATH" <<'JSON'
{
  "task_id": "golden-agent-test",
  "status": "complete",
  "touched_files": ["scripts/test_autoprogramming_fast.sh"],
  "artifact_paths": ["scripts/test_autoprogramming_fast.sh"],
  "tests": [{"command": "bash -n scripts/test_autoprogramming_fast.sh", "status": "passed"}],
  "evidence": {"agent": "fake-ok"},
  "usage_path": "codex_usage_accounting.json"
}
JSON
cat > "$GOLDEN_AGENT_RESULT_DIR/codex_usage_accounting.json" <<'JSON'
{"usage":{"input_tokens":10,"output_tokens":5,"reasoning_tokens":1,"total_tokens":16}}
JSON
SH
chmod +x "$agent_ok"

ok_result="$tmp_root/ok"
ORQUESTA_GOLDEN_TASK_ID="golden-agent-test" \
ORQUESTA_GOLDEN_TASK_REQUEST="$request" \
ORQUESTA_GOLDEN_TASK_RESULT_DIR="$ok_result" \
bash "$repo_root/scripts/orquesta_golden_agent_launcher.sh" \
  --agent-command "$agent_ok" \
  --skill-ref skill-ref-orquesta-programacion-minima-v0

python3 - "$ok_result/result.json" "$ok_result/agent_prompt.md" <<'PY'
import json
import sys
from pathlib import Path

payload = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
prompt = Path(sys.argv[2]).read_text(encoding="utf-8")
evidence = payload["evidence"]["golden_agent_launcher"]
assert payload["status"] == "complete", payload
assert payload["task_id"] == "golden-agent-test", payload
assert payload["usage_path"] == "codex_usage_accounting.json", payload
assert evidence["agent_exit_code"] == 0, evidence
assert evidence["skill_refs"] == ["skill-ref-orquesta-programacion-minima-v0"], evidence
assert evidence["issues"] == [], evidence
assert "Write Set" in prompt and "scripts" in prompt, prompt
PY

agent_missing="$tmp_root/agent_missing.sh"
cat > "$agent_missing" <<'SH'
#!/usr/bin/env bash
set -euo pipefail
echo "no result"
SH
chmod +x "$agent_missing"

missing_result="$tmp_root/missing"
ORQUESTA_GOLDEN_TASK_ID="golden-agent-test" \
ORQUESTA_GOLDEN_TASK_REQUEST="$request" \
ORQUESTA_GOLDEN_TASK_RESULT_DIR="$missing_result" \
bash "$repo_root/scripts/orquesta_golden_agent_launcher.sh" \
  --agent-command "$agent_missing"

python3 - "$missing_result/result.json" <<'PY'
import json
import sys
from pathlib import Path

payload = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
evidence = payload["evidence"]["golden_agent_launcher"]
assert payload["status"] == "failed", payload
assert "result_json_missing" in payload["launcher_issues"], payload
assert evidence["agent_exit_code"] == 0, evidence
PY

agent_fail="$tmp_root/agent_fail.sh"
cat > "$agent_fail" <<'SH'
#!/usr/bin/env bash
set -euo pipefail
cat > "$GOLDEN_AGENT_RESULT_PATH" <<'JSON'
{
  "task_id": "golden-agent-test",
  "status": "complete",
  "touched_files": [],
  "tests": [],
  "evidence": {}
}
JSON
exit 23
SH
chmod +x "$agent_fail"

fail_result="$tmp_root/fail"
ORQUESTA_GOLDEN_TASK_ID="golden-agent-test" \
ORQUESTA_GOLDEN_TASK_REQUEST="$request" \
ORQUESTA_GOLDEN_TASK_RESULT_DIR="$fail_result" \
bash "$repo_root/scripts/orquesta_golden_agent_launcher.sh" \
  --agent-command "$agent_fail"

python3 - "$fail_result/result.json" <<'PY'
import json
import sys
from pathlib import Path

payload = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
evidence = payload["evidence"]["golden_agent_launcher"]
assert payload["status"] == "failed", payload
assert payload["agent_exit_code"] == 23, payload
assert "agent_command_failed" in payload["launcher_issues"], payload
assert evidence["agent_exit_code"] == 23, evidence
PY

echo "orquesta_golden_agent_launcher ok"
