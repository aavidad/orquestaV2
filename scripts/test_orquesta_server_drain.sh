#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck source=scripts/lib/isolated_test_env.sh
source "$ROOT/scripts/lib/isolated_test_env.sh"

test_cache_root="${ORQUESTA_DRAIN_TEST_CACHE_ROOT:-$ROOT/.orquesta-runtime/test-cache/f3-rework/shell}"
mkdir -p "$test_cache_root"
test_root="$(mktemp -d "$test_cache_root/orquesta-server-drain-test.XXXXXX")"
cleanup() {
  chmod -R u+w "$test_root" 2>/dev/null || true
  rm -rf "$test_root"
}
trap cleanup EXIT

orquesta_use_isolated_test_env "$test_root/env"

state_root="$test_root/state-root"
runtime_root="$test_root/runtime-root"
usage_app="$test_root/uso-app"
receipt="$test_root/receipt.json"
backup="$test_root/backup"
ps_file="$test_root/ps.txt"
fake_bin="$test_root/fake-bin"
kill_log="$test_root/kill.log"
curl_log="$test_root/curl.log"
tmux_log="$test_root/tmux.log"

mkdir -p "$state_root/state" "$state_root/logs" "$state_root/runtime" "$runtime_root" "$usage_app"
printf '{"schema_version":"state"}\n' >"$state_root/state/state.json"
printf '1234\n' >"$state_root/server.pid"
printf 'http://127.0.0.1:19071\n' >"$state_root/runtime/base_url.txt"
mkdir -p "$fake_bin"
cat >"$fake_bin/kill-fake" <<EOF
#!/usr/bin/env bash
printf '%s\n' "\$*" >>'$kill_log'
exit 0
EOF
cat >"$fake_bin/curl-fake" <<EOF
#!/usr/bin/env bash
printf '%s\n' "\$*" >>'$curl_log'
exit 0
EOF
cat >"$fake_bin/tmux-fake" <<EOF
#!/usr/bin/env bash
printf '%s\n' "\$*" >>'$tmux_log'
exit 0
EOF
chmod +x "$fake_bin/kill-fake" "$fake_bin/curl-fake" "$fake_bin/tmux-fake"

cat >"$ps_file" <<EOF
 1234     1 berserk /srv/orquesta-self/runtime/orquesta-server-claude run ORQUESTA_SERVER_STATE_DIR=$state_root/state ORQUESTA_CODEX_RUNTIME_WORKDIR=$runtime_root
 5678     1 berserk /srv/orquesta-self/runtime/orquesta-server-claude run ORQUESTA_SERVER_STATE_DIR=$usage_app/state ORQUESTA_CODEX_RUNTIME_WORKDIR=$usage_app/runtime
 9012     1 berserk /srv/orquesta-self/runtime/orquesta-server-claude run --foreign-profile
EOF

bash -n "$ROOT/scripts/orquesta_server_drain.sh" "$ROOT/scripts/lib/isolated_test_env.sh"

ORQUESTA_DRAIN_RUNTIME_ROOT="$runtime_root" \
ORQUESTA_DRAIN_STATE_ROOT="$state_root" \
ORQUESTA_DRAIN_PROTECTED_APP_ROOT="$usage_app" \
ORQUESTA_DRAIN_RECEIPT="$receipt" \
ORQUESTA_DRAIN_BACKUP_DIR="$backup" \
ORQUESTA_DRAIN_PS_COMMAND="cat '$ps_file'" \
  "$ROOT/scripts/orquesta_server_drain.sh" --dry-run >/dev/null

receipt_no_args="$test_root/receipt-no-args.json"
backup_no_args="$test_root/backup-no-args"

ORQUESTA_DRAIN_RUNTIME_ROOT="$runtime_root" \
ORQUESTA_DRAIN_STATE_ROOT="$state_root" \
ORQUESTA_DRAIN_PROTECTED_APP_ROOT="$usage_app" \
ORQUESTA_DRAIN_RECEIPT="$receipt_no_args" \
ORQUESTA_DRAIN_BACKUP_DIR="$backup_no_args" \
ORQUESTA_DRAIN_PS_COMMAND="cat '$ps_file'" \
  "$ROOT/scripts/orquesta_server_drain.sh" >/dev/null

python3 - "$receipt" "$backup" "$state_root" "$usage_app" "$receipt_no_args" <<'PY'
import json
import os
import sys

receipt_path, backup, state_root, usage_app, receipt_no_args_path = sys.argv[1:6]
with open(receipt_path, encoding="utf-8") as fh:
    receipt = json.load(fh)
with open(receipt_no_args_path, encoding="utf-8") as fh:
    receipt_no_args = json.load(fh)

assert receipt["schema_version"] == "orquesta_server_drain_receipt.v0"
assert receipt["backup_prepared_before_stop"] is True
assert receipt["dry_run"] is True
assert receipt["drain_status"] == "refused"
assert receipt["backup_dir"] == backup
assert os.path.exists(os.path.join(backup, "manifest.json"))
assert os.path.exists(os.path.join(backup, "state"))
assert receipt["inventory_before"]["targets"][0]["pid"] == "1234"
assert receipt["inventory_before"]["protected"][0]["pid"] == "5678"
assert receipt["inventory_before"]["protected"][0]["skip_reason"] == "protected_uso_app"
assert receipt["inventory_before"]["skipped"][0]["pid"] == "9012"
assert receipt["inventory_before"]["skipped"][0]["skip_reason"] == "identity_not_managed"
assert any(action == "dry_run_skip pid=1234 signal=INT" for action in receipt["actions"])
assert receipt["protected_app_root"] == usage_app
assert receipt["state_root"] == state_root
assert "inventory_after" in receipt
assert receipt_no_args["dry_run"] is True
assert receipt_no_args["drain_status"] == "refused"
assert any(action == "dry_run_skip pid=1234 signal=INT" for action in receipt_no_args["actions"])
PY

receipt_confirm="$test_root/receipt-confirm.json"
backup_confirm="$test_root/backup-confirm"
ORQUESTA_DRAIN_RUNTIME_ROOT="$runtime_root" \
ORQUESTA_DRAIN_STATE_ROOT="$state_root" \
ORQUESTA_DRAIN_PROTECTED_APP_ROOT="$usage_app" \
ORQUESTA_DRAIN_RECEIPT="$receipt_confirm" \
ORQUESTA_DRAIN_BACKUP_DIR="$backup_confirm" \
ORQUESTA_DRAIN_PS_COMMAND="cat '$ps_file'" \
ORQUESTA_DRAIN_WAIT_SECONDS=0 \
ORQUESTA_DRAIN_KILL_COMMAND="$fake_bin/kill-fake" \
ORQUESTA_DRAIN_CURL_COMMAND="$fake_bin/curl-fake" \
ORQUESTA_DRAIN_TMUX_COMMAND="$fake_bin/tmux-fake" \
ORQUESTA_DRAIN_TMUX_LIST_COMMAND="printf '%s\n' orquesta-goal-test ignored-session" \
  "$ROOT/scripts/orquesta_server_drain.sh" --drain --confirm-drain orquesta-server-drain >/dev/null

python3 - "$receipt_confirm" "$kill_log" "$curl_log" "$tmux_log" <<'PY'
import json
import sys

receipt_path, kill_log, curl_log, tmux_log = sys.argv[1:5]
with open(receipt_path, encoding="utf-8") as fh:
    receipt = json.load(fh)
with open(kill_log, encoding="utf-8") as fh:
    kill_lines = [line.strip() for line in fh if line.strip()]
with open(curl_log, encoding="utf-8") as fh:
    curl_text = fh.read()
with open(tmux_log, encoding="utf-8") as fh:
    tmux_lines = [line.strip() for line in fh if line.strip()]

assert receipt["dry_run"] is False
assert receipt["drain_status"] == "residual"
assert any("action=shutdown_http" in action for action in receipt["actions"])
assert any(action == "send pid=1234 signal=INT" for action in receipt["actions"])
assert any(action == "send pid=1234 signal=TERM" for action in receipt["actions"])
assert any(action == "send session=orquesta-goal-test action=tmux_kill_session" for action in receipt["actions"])
assert "-X POST" in curl_text
assert "/api/v0/server/shutdown" in curl_text
assert kill_lines == ["-INT 1234", "-TERM 1234"]
assert tmux_lines == ["kill-session -t orquesta-goal-test"]
PY

if grep -R "/api/v0/runs/control" "$ROOT/scripts/orquesta_server_drain.sh"; then
  echo "drain no debe depender de runs/control" >&2
  exit 1
fi

echo "test_orquesta_server_drain=ok"
