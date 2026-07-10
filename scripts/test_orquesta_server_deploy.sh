#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
script="$ROOT/scripts/orquesta_server_deploy.sh"
workdir="$(mktemp -d "${TMPDIR:-/tmp}/orquesta-server-deploy-test.XXXXXX")"
trap 'rm -rf "$workdir"' EXIT

git_id() {
  git -c user.name="Deploy Test" -c user.email="deploy-test@example.invalid" "$@"
}

make_repo() {
  repo="$1"
  mkdir -p "$repo/cmd/orquesta-server"
  git_id -C "$repo" init -q
  printf 'module example.invalid/orquesta-deploy-test\n\ngo 1.22\n' >"$repo/go.mod"
  printf 'package main\nfunc main(){}\n' >"$repo/cmd/orquesta-server/main.go"
  git_id -C "$repo" add .
  git_id -C "$repo" commit -q -m initial
}

make_ctl() {
  ctl="$1"
  status_sha="$2"
  cat >"$ctl" <<SH
#!/usr/bin/env bash
set -euo pipefail
case "\${1:-}" in
start)
  printf 'start %s\n' "\${ORQUESTA_CTL_BINARY:-}" >>"$workdir/ctl.log"
  ;;
stop)
  printf 'stop %s\n' "\${ORQUESTA_CTL_BINARY:-}" >>"$workdir/ctl.log"
  ;;
status)
  printf '{"status":"running","readiness":true,"supervisor_ok":true'
  if [ -n "$status_sha" ]; then
    printf ',"binary_sha256":"%s"' "$status_sha"
  fi
  printf '}\n'
  ;;
*)
  exit 2
  ;;
esac
SH
  chmod +x "$ctl"
}

make_ctl_nested_runtime_identity() {
  ctl="$1"
  status_sha="$2"
  cat >"$ctl" <<SH
#!/usr/bin/env bash
set -euo pipefail
case "\${1:-}" in
start)
  printf 'start %s\n' "\${ORQUESTA_CTL_BINARY:-}" >>"$workdir/ctl.log"
  ;;
stop)
  printf 'stop %s\n' "\${ORQUESTA_CTL_BINARY:-}" >>"$workdir/ctl.log"
  ;;
status)
  printf '{"status":"running","readiness":true,"supervisor_ok":true,"runtime_identity":{"binary_sha256":"%s"}}\n' "$status_sha"
  ;;
*)
  exit 2
  ;;
esac
SH
  chmod +x "$ctl"
}

make_ctl_status_fails() {
  ctl="$1"
  cat >"$ctl" <<SH
#!/usr/bin/env bash
set -euo pipefail
case "\${1:-}" in
start)
  printf 'start %s\n' "\${ORQUESTA_CTL_BINARY:-}" >>"$workdir/ctl.log"
  ;;
stop)
  printf 'stop %s\n' "\${ORQUESTA_CTL_BINARY:-}" >>"$workdir/ctl.log"
  ;;
status)
  echo "status failed" >&2
  exit 7
  ;;
*)
  exit 2
  ;;
esac
SH
  chmod +x "$ctl"
}

make_ctl_stop_fails() {
  ctl="$1"
  cat >"$ctl" <<SH
#!/usr/bin/env bash
set -euo pipefail
case "\${1:-}" in
stop)
  echo "stop failed" >&2
  exit 9
  ;;
start)
  printf 'start %s\n' "\${ORQUESTA_CTL_BINARY:-}" >>"$workdir/ctl.log"
  ;;
status)
  printf '{"status":"running","readiness":true,"supervisor_ok":true,"binary_sha256":"%s"}\n' "$(printf deploy-test-binary | sha256sum | awk '{print $1}')"
  ;;
*)
  exit 2
  ;;
esac
SH
  chmod +x "$ctl"
}

run_deploy() {
  case_dir="$1"
  repo="$case_dir/repo"
  wt="$case_dir/worktree"
  state="$case_dir/state"
  bin="$case_dir/runtime/orquesta-server"
  ctl="$case_dir/ctl.sh"
  mkdir -p "$state"
  ORQUESTA_DEPLOY_REPO="$repo" \
  ORQUESTA_DEPLOY_WORKTREE="$wt" \
  ORQUESTA_DEPLOY_REF="${ORQUESTA_DEPLOY_REF:-HEAD}" \
  ORQUESTA_DEPLOY_CTL="$ctl" \
  ORQUESTA_DEPLOY_BINARY="$bin" \
  ORQUESTA_DEPLOY_STATE_DIR="$state" \
  ORQUESTA_DEPLOY_BUILD_CMD='printf deploy-test-binary >"$ORQUESTA_DEPLOY_BUILD_OUT"; chmod +x "$ORQUESTA_DEPLOY_BUILD_OUT"' \
    bash "$script"
}

assert_receipt_status() {
  receipt="$1"
  status="$2"
  reason="$3"
  python3 - "$receipt" "$status" "$reason" <<'PY'
import json, sys
path, status, reason = sys.argv[1:4]
data = json.load(open(path))
if data.get("status") != status:
    raise SystemExit(f"status {data.get('status')} != {status}")
if reason and data.get("reason_code") != reason:
    raise SystemExit(f"reason {data.get('reason_code')} != {reason}")
if "remote_url" not in data:
    raise SystemExit("remote_url missing")
PY
}

test_success() {
  case_dir="$workdir/success"
  make_repo "$case_dir/repo"
  mkdir -p "$case_dir"
  make_ctl "$case_dir/ctl.sh" "$(printf deploy-test-binary | sha256sum | awk '{print $1}')"
  run_deploy "$case_dir" >/tmp/orquesta-deploy-success.out
  grep -q 'orquesta_server_deploy=ok' /tmp/orquesta-deploy-success.out
  grep -q "stop $case_dir/runtime/orquesta-server" "$workdir/ctl.log"
  grep -q "start $case_dir/runtime/orquesta-server" "$workdir/ctl.log"
  [ -x "$case_dir/runtime/orquesta-server" ]
  grep -q 'deploy-test-binary' "$case_dir/runtime/orquesta-server"
  assert_receipt_status "$case_dir/state/orquesta_server_deploy_receipt_v0.json" ok ""
}

test_success_nested_runtime_identity() {
	case_dir="$workdir/success-nested-runtime-identity"
	make_repo "$case_dir/repo"
  mkdir -p "$case_dir"
  make_ctl_nested_runtime_identity "$case_dir/ctl.sh" "$(printf deploy-test-binary | sha256sum | awk '{print $1}')"
  run_deploy "$case_dir" >/tmp/orquesta-deploy-success-nested.out
  grep -q 'orquesta_server_deploy=ok' /tmp/orquesta-deploy-success-nested.out
	assert_receipt_status "$case_dir/state/orquesta_server_deploy_receipt_v0.json" ok ""
}

test_success_preserves_branch_worktree() {
	case_dir="$workdir/success-branch"
	make_repo "$case_dir/repo"
	git_id -C "$case_dir/repo" branch -M trabajo/plataforma-agentes
	mkdir -p "$case_dir"
	make_ctl "$case_dir/ctl.sh" "$(printf deploy-test-binary | sha256sum | awk '{print $1}')"
	ORQUESTA_DEPLOY_REF=trabajo/plataforma-agentes run_deploy "$case_dir" >/tmp/orquesta-deploy-success-branch.out
	grep -q 'orquesta_server_deploy=ok' /tmp/orquesta-deploy-success-branch.out
	[ "$(git -C "$case_dir/worktree" symbolic-ref --short -q HEAD)" = "trabajo/plataforma-agentes" ]
	assert_receipt_status "$case_dir/state/orquesta_server_deploy_receipt_v0.json" ok ""
}

test_deploy_config_missing() {
	case_dir="$workdir/config-missing"
	make_repo "$case_dir/repo"
  mkdir -p "$case_dir"
  make_ctl "$case_dir/ctl.sh" ""
  set +e
  ORQUESTA_DEPLOY_REQUIRE_CONFIG=1 run_deploy "$case_dir" >/tmp/orquesta-deploy-config.out 2>&1
  code=$?
  set -e
  [ "$code" -ne 0 ]
  grep -q 'reason_code=deploy_config_missing' /tmp/orquesta-deploy-config.out
  assert_receipt_status "$case_dir/state/orquesta_server_deploy_receipt_v0.json" failed deploy_config_missing
}

test_deploy_not_fast_forward() {
  case_dir="$workdir/not-ff"
  make_repo "$case_dir/repo"
  git clone -q "$case_dir/repo" "$case_dir/worktree"
  printf 'local\n' >"$case_dir/worktree/local.txt"
  git_id -C "$case_dir/worktree" add local.txt
  git_id -C "$case_dir/worktree" commit -q -m local
  printf 'remote\n' >"$case_dir/repo/remote.txt"
  git_id -C "$case_dir/repo" add remote.txt
  git_id -C "$case_dir/repo" commit -q -m remote
  make_ctl "$case_dir/ctl.sh" ""
  set +e
  ORQUESTA_DEPLOY_REF=HEAD run_deploy "$case_dir" >/tmp/orquesta-deploy-not-ff.out 2>&1
  code=$?
  set -e
  [ "$code" -ne 0 ]
  grep -q 'reason_code=deploy_not_fast_forward' /tmp/orquesta-deploy-not-ff.out
  assert_receipt_status "$case_dir/state/orquesta_server_deploy_receipt_v0.json" failed deploy_not_fast_forward
}

test_deploy_runtime_identity_mismatch() {
  case_dir="$workdir/sha-mismatch"
  make_repo "$case_dir/repo"
  mkdir -p "$case_dir"
  make_ctl "$case_dir/ctl.sh" "$(printf wrong-binary | sha256sum | awk '{print $1}')"
  set +e
  run_deploy "$case_dir" >/tmp/orquesta-deploy-sha.out 2>&1
  code=$?
  set -e
  [ "$code" -ne 0 ]
  grep -q 'reason_code=deploy_runtime_identity_mismatch' /tmp/orquesta-deploy-sha.out
  assert_receipt_status "$case_dir/state/orquesta_server_deploy_receipt_v0.json" failed deploy_runtime_identity_mismatch
}

test_deploy_runtime_identity_missing() {
  case_dir="$workdir/sha-missing"
  make_repo "$case_dir/repo"
  mkdir -p "$case_dir"
  make_ctl "$case_dir/ctl.sh" ""
  set +e
  run_deploy "$case_dir" >/tmp/orquesta-deploy-sha-missing.out 2>&1
  code=$?
  set -e
  [ "$code" -ne 0 ]
  grep -q 'reason_code=deploy_runtime_identity_missing' /tmp/orquesta-deploy-sha-missing.out
  assert_receipt_status "$case_dir/state/orquesta_server_deploy_receipt_v0.json" failed deploy_runtime_identity_missing
}

test_deploy_status_failed() {
  case_dir="$workdir/status-failed"
  make_repo "$case_dir/repo"
  mkdir -p "$case_dir"
  make_ctl_status_fails "$case_dir/ctl.sh"
  set +e
  run_deploy "$case_dir" >/tmp/orquesta-deploy-status-failed.out 2>&1
  code=$?
  set -e
  [ "$code" -ne 0 ]
  grep -q 'reason_code=deploy_status_failed' /tmp/orquesta-deploy-status-failed.out
  assert_receipt_status "$case_dir/state/orquesta_server_deploy_receipt_v0.json" failed deploy_status_failed
}

test_deploy_stop_failed() {
  case_dir="$workdir/stop-failed"
  make_repo "$case_dir/repo"
  mkdir -p "$case_dir"
  old_bin="$case_dir/runtime/orquesta-server"
  mkdir -p "$(dirname "$old_bin")"
  printf old-binary >"$old_bin"
  chmod +x "$old_bin"
  make_ctl_stop_fails "$case_dir/ctl.sh"
  set +e
  run_deploy "$case_dir" >/tmp/orquesta-deploy-stop-failed.out 2>&1
  code=$?
  set -e
  [ "$code" -ne 0 ]
  grep -q 'reason_code=deploy_stop_failed' /tmp/orquesta-deploy-stop-failed.out
  grep -q 'old-binary' "$old_bin"
  assert_receipt_status "$case_dir/state/orquesta_server_deploy_receipt_v0.json" failed deploy_stop_failed
}

bash -n "$script"
test_success
test_success_nested_runtime_identity
test_success_preserves_branch_worktree
test_deploy_config_missing
test_deploy_not_fast_forward
test_deploy_runtime_identity_mismatch
test_deploy_runtime_identity_missing
test_deploy_status_failed
test_deploy_stop_failed
echo "orquesta_server_deploy_tests=ok"
