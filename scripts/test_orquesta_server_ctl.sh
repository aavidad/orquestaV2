#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
script="$ROOT/scripts/orquesta_server_ctl.sh"
workdir="$(mktemp -d "${TMPDIR:-/tmp}/orquesta-server-ctl-test.XXXXXX")"
trap 'cleanup' EXIT

cleanup() {
  if [ -n "${fake_pid:-}" ] && kill -0 "$fake_pid" 2>/dev/null; then
    kill -KILL -- "-$fake_pid" 2>/dev/null || kill -KILL "$fake_pid" 2>/dev/null || true
  fi
  rm -rf "$workdir"
}

wait_process_group_gone() {
  pid="$1"
  for _ in 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 19 20; do
    kill -0 -- "-$pid" 2>/dev/null || return 0
    sleep 0.1
  done
  return 1
}

wait_pid_gone() {
  pid="$1"
  for _ in 1 2 3 4 5 6 7 8 9 10; do
    kill -0 "$pid" 2>/dev/null || return 0
    if [ -r "/proc/$pid/stat" ] && [ "$(awk '{print $3}' "/proc/$pid/stat" 2>/dev/null || true)" = "Z" ]; then
      return 0
    fi
    sleep 0.1
  done
  return 1
}

fake_tools="$workdir/tools"
mkdir -p "$fake_tools"
fake_bin="$fake_tools/orquesta-server-fake"
cat >"$fake_bin" <<'SH'
#!/usr/bin/env bash
printf '%s\n' "$@" >"$ORQUESTA_FAKE_ARGS_FILE"
if [ "${ORQUESTA_FAKE_SPAWN_CHILD:-0}" = "1" ]; then
  (trap '' INT TERM; while :; do sleep 1; done) &
  printf '%s\n' "$!" >"$ORQUESTA_FAKE_CHILD_PID_FILE"
fi
trap 'exit 0' INT TERM
while :; do
  read -r -t 1 _ || true
done
SH
chmod +x "$fake_bin"

cat >"$fake_tools/curl" <<'SH'
#!/usr/bin/env bash
binary_real="$(readlink -f "$ORQUESTA_FAKE_BINARY")"
binary_sha="$(sha256sum "$binary_real" | awk '{print $1}')"
binary_name="$(basename "$binary_real")"
commit_ref="$(git -C "$ORQUESTA_FAKE_PROJECT" rev-parse HEAD)"
build_ref="build-ref-orquesta-server-${commit_ref%${commit_ref#????????????}}"
case "${ORQUESTA_FAKE_READINESS_MODE:-ready}" in
  ready)
    printf '%s\n' "{\"schema_version\":\"orquesta_server_readiness.v0\",\"ready\":true,\"liveness_status\":\"ok\",\"status\":\"running\",\"availability_status\":\"running\",\"startup_ready\":true,\"startup_status\":\"startup_ready\",\"runtime_identity\":{\"schema_version\":\"orquesta_server_runtime_identity.v0\",\"binary_path_ref\":\"server-runtime-binary-path\",\"binary_name\":\"$binary_name\",\"binary_sha256\":\"$binary_sha\",\"build_ref\":\"$build_ref\",\"commit_ref\":\"$commit_ref\"}}"
    ;;
  degraded_200)
    printf '%s\n' '{"status":"degraded_identity","availability_status":"degraded_identity","startup_ready":false,"startup_status":"degraded_identity"}'
    ;;
  sha_only_200)
    printf '%s\n' '{"status":"running","binary_sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}'
    ;;
  startup_status_wrong_200)
    printf '%s\n' '{"status":"running","availability_status":"running","startup_ready":true,"startup_status":"startup_degraded"}'
    ;;
  identity_mismatch_200)
    printf '%s\n' "{\"schema_version\":\"orquesta_server_readiness.v0\",\"ready\":true,\"liveness_status\":\"ok\",\"status\":\"running\",\"availability_status\":\"running\",\"startup_ready\":true,\"startup_status\":\"startup_ready\",\"runtime_identity\":{\"schema_version\":\"orquesta_server_runtime_identity.v0\",\"binary_path_ref\":\"server-runtime-binary-path\",\"binary_name\":\"$binary_name\",\"binary_sha256\":\"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb\",\"build_ref\":\"$build_ref\",\"commit_ref\":\"$commit_ref\"}}"
    ;;
  *)
    exit 22
    ;;
esac
SH
chmod +x "$fake_tools/curl"

cat >"$fake_tools/go" <<'SH'
#!/usr/bin/env bash
[ "${1:-}" = "version" ] && [ "${2:-}" = "-m" ] || exit 2
commit_ref="$(git -C "$ORQUESTA_FAKE_PROJECT" rev-parse HEAD)"
case "${ORQUESTA_FAKE_BUILD_MODE:-exact}" in
  mismatch) commit_ref="cccccccccccccccccccccccccccccccccccccccc" ;;
esac
printf '%s: go1.25.0\n' "${3:-binary}"
printf '\tbuild\tvcs.revision=%s\n' "$commit_ref"
if [ "${ORQUESTA_FAKE_BUILD_MODE:-exact}" = "modified" ]; then
  printf '\tbuild\tvcs.modified=true\n'
else
  printf '\tbuild\tvcs.modified=false\n'
fi
SH
chmod +x "$fake_tools/go"

init_canonical_repo() {
  repo="$1"
  mkdir -p "$repo"
  git -C "$repo" init -q
  git -C "$repo" -c user.name="Ctl Test" -c user.email="ctl-test@example.invalid" commit --allow-empty -q -m initial
  git -C "$repo" branch -M trabajo/plataforma-agentes
  branch="$(git -C "$repo" branch --show-current)"
  git -C "$repo" remote add origin git@github.com:aavidad/orquestador.git
  git -C "$repo" update-ref "refs/remotes/origin/$branch" HEAD
  git -C "$repo" branch --set-upstream-to "origin/$branch" >/dev/null 2>&1
}

run_ctl_start() {
  root="$1"
  project="$2"
  mode="${3:-ready}"
  binary="${4:-$fake_bin}"
  spawn_child="${5:-0}"
  ORQUESTA_CTL_HOME="$root" \
  ORQUESTA_CTL_BINARY="$binary" \
  ORQUESTA_CTL_GO_BINARY="$fake_tools/go" \
  ORQUESTA_CTL_USER="$(id -un)" \
  ORQUESTA_CTL_ADDR="127.0.0.1:1" \
  ORQUESTA_CTL_WORKDIR="$project" \
  ORQUESTA_CTL_STARTUP_SLEEP="0" \
  ORQUESTA_FAKE_ARGS_FILE="$root/args.txt" \
  ORQUESTA_FAKE_BINARY="$binary" \
  ORQUESTA_FAKE_PROJECT="$project" \
  ORQUESTA_FAKE_CHILD_PID_FILE="$root/child.pid" \
  ORQUESTA_FAKE_SPAWN_CHILD="$spawn_child" \
  ORQUESTA_FAKE_READINESS_MODE="$mode" \
  PATH="$fake_tools:$PATH" \
    bash "$script" start
}

run_start_case() {
  case_name="$1"
  expected_config="$2"
  root="$workdir/$case_name"
  mkdir -p "$root/state"
  init_canonical_repo "$root/project"
  if [ "$expected_config" = "auto" ]; then
    expected_config="$root/project/orquesta.config.json"
    printf '%s\n' '{"schema_version":"orquesta_config.v0"}' >"$expected_config"
  fi

  run_ctl_start "$root" "$root/project" ready >/dev/null
  fake_pid="$(cat "$root/server.pid")"
  grep -qx 'run' "$root/args.txt"
  if [ "$expected_config" = "none" ]; then
    if grep -q -- '--config' "$root/args.txt"; then
      echo "config arg inesperado en $case_name" >&2
      exit 1
    fi
  else
    grep -qx -- '--config' "$root/args.txt"
    grep -qx "$expected_config" "$root/args.txt"
  fi
	# El fake hereda SIGINT ignorado de nohup; TERM verifica su trap sin esperar
	# el escalado que el ctl reserva para un fallo real de readiness.
	kill -TERM "$fake_pid" 2>/dev/null || true
	if ! wait_pid_gone "$fake_pid"; then
		echo "hijo fake filtrado tras caso ready: pid=$fake_pid" >&2
		exit 1
	fi
  fake_pid=""
}

run_invalid_workdir_case() {
  case_name="$1"
  project_setup="$2"
  root="$workdir/$case_name"
  mkdir -p "$root/state"
  project="$root/project"
  case "$project_setup" in
    missing) ;;
    file) printf 'not-dir\n' >"$project" ;;
    non-git) mkdir -p "$project" ;;
    *) exit 2 ;;
  esac
  set +e
  run_ctl_start "$root" "$project" ready >"$root/out.txt" 2>"$root/err.txt"
  code=$?
  set -e
  [ "$code" -ne 0 ]
  grep -q 'reason_code=ctl_workdir_invalid' "$root/err.txt"
}

run_readiness_negative_case() {
  mode="$1"
  spawn_child="${2:-0}"
  root="$workdir/readiness-$mode"
  mkdir -p "$root/state"
  init_canonical_repo "$root/project"
  set +e
  run_ctl_start "$root" "$root/project" "$mode" "$fake_bin" "$spawn_child" >"$root/out.txt" 2>"$root/err.txt"
  code=$?
  set -e
  [ "$code" -ne 0 ]
  grep -q 'reason_code=startup_not_ready' "$root/err.txt"
  pid="$(cat "$root/server.pid")"
  wait_pid_gone "$pid"
  wait_process_group_gone "$pid" || {
    echo "daemon o hijo filtrado tras readiness $mode: pid=$pid" >&2
    exit 1
  }
}

run_binary_identity_negative_case() {
  mode="$1"
  expected_reason="$2"
  root="$workdir/binary-$mode"
  mkdir -p "$root/state"
  init_canonical_repo "$root/project"
  set +e
  ORQUESTA_FAKE_BUILD_MODE="$mode" run_ctl_start "$root" "$root/project" ready >"$root/out.txt" 2>"$root/err.txt"
  code=$?
  set -e
  [ "$code" -ne 0 ]
  grep -q "reason_code=$expected_reason" "$root/err.txt"
  [ ! -e "$root/server.pid" ]
}

run_binary_symlink_case() {
  root="$workdir/binary-symlink"
  mkdir -p "$root/state"
  init_canonical_repo "$root/project"
  ln -s "$fake_bin" "$root/orquesta-server-current"
  run_ctl_start "$root" "$root/project" ready "$root/orquesta-server-current" >/dev/null
  fake_pid="$(cat "$root/server.pid")"
  kill -TERM "$fake_pid" 2>/dev/null || true
  wait_pid_gone "$fake_pid"
  fake_pid=""
}

run_marked_workdir_case() {
  marker="$1"
  expected_reason="$2"
  root="$workdir/marked-${marker//./-}"
  mkdir -p "$root/state"
  init_canonical_repo "$root/project"
  : >"$root/project/$marker"
  set +e
  run_ctl_start "$root" "$root/project" ready >"$root/out.txt" 2>"$root/err.txt"
  code=$?
  set -e
  [ "$code" -ne 0 ]
  grep -q "reason_code=$expected_reason" "$root/err.txt"
}

run_remote_negative_case() {
  root="$workdir/local-remote"
  mkdir -p "$root/state"
  init_canonical_repo "$root/project"
  git -C "$root/project" remote set-url origin "$root/local.bundle"
  set +e
  run_ctl_start "$root" "$root/project" ready >"$root/out.txt" 2>"$root/err.txt"
  code=$?
  set -e
  [ "$code" -ne 0 ]
  grep -q 'reason_code=ctl_workdir_remote_not_canonical' "$root/err.txt"
}

run_symlink_negative_case() {
  root="$workdir/symlink"
  mkdir -p "$root/state"
  init_canonical_repo "$root/project-real"
  ln -s "$root/project-real" "$root/project"
  set +e
  run_ctl_start "$root" "$root/project" ready >"$root/out.txt" 2>"$root/err.txt"
  code=$?
  set -e
  [ "$code" -ne 0 ]
  grep -q 'reason_code=ctl_workdir_invalid' "$root/err.txt"
}

bash -n "$script"
run_start_case "without-config" "none"
run_start_case "with-auto-config" "auto"
run_invalid_workdir_case "missing-workdir" "missing"
run_invalid_workdir_case "file-workdir" "file"
run_invalid_workdir_case "nongit-workdir" "non-git"
run_marked_workdir_case ".orquesta-retired" "ctl_workdir_invalid"
run_marked_workdir_case ".orquesta-stale" "ctl_workdir_stale"
run_remote_negative_case
run_symlink_negative_case
run_binary_symlink_case
run_binary_identity_negative_case "mismatch" "ctl_binary_commit_mismatch"
run_binary_identity_negative_case "modified" "ctl_binary_not_reproducible"
run_readiness_negative_case "degraded_200" 1
run_readiness_negative_case "sha_only_200"
run_readiness_negative_case "startup_status_wrong_200"
run_readiness_negative_case "identity_mismatch_200"
