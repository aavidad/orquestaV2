#!/usr/bin/env bash
set -euo pipefail
umask 077

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
SCRIPT="$ROOT/scripts/orquesta_profile_server.sh"
FIRECRACKER_PROBE="$ROOT/scripts/lib/firecracker_launcher_probe.py"
MAINTENANCE_MARKER_READER="$ROOT/scripts/lib/profile_maintenance_marker.py"
TEST_PARENT="$(python3 - <<'PY'
import os
import pwd

print(pwd.getpwuid(os.geteuid()).pw_dir)
PY
)"
TEST_ROOT="$(mktemp -d "$TEST_PARENT/.orquesta-profile-server-test.XXXXXX")"
BASE="$TEST_ROOT/runtime"
ACCOUNTS="$TEST_ROOT/accounts"
FAKE_SOURCE="$TEST_ROOT/fake-server.go"
FAKE_BINARY="$TEST_ROOT/orquesta-fake"
REAL_BINARY="$TEST_ROOT/orquesta-real"
REAL_CLI_PROBE_ROOT="$TEST_ROOT/real-cli-probe"
GO_CACHE="$TEST_ROOT/go-cache"
REPOSITORY="$TEST_ROOT/repository"
FAKE_BWRAP_9000="$TEST_ROOT/fake-bwrap-9000"
FAKE_BWRAP_65536="$TEST_ROOT/fake-bwrap-65536"
FAKE_BWRAP_FAILED="$TEST_ROOT/fake-bwrap-failed"
FAKE_TOOLCHAIN="$TEST_ROOT/go-toolchain"
UNSAFE_TOOLCHAIN="$TEST_ROOT/go-toolchain-unsafe"
LAUNCHER_ROOT="$TEST_ROOT/firecracker-launcher"
LAUNCHER_FIXTURE_PID=""
mkdir -m 700 "$BASE" "$ACCOUNTS" "$GO_CACHE"

cleanup() {
  if [ -n "$LAUNCHER_FIXTURE_PID" ] &&
    kill -0 "$LAUNCHER_FIXTURE_PID" 2>/dev/null; then
    kill -TERM "$LAUNCHER_FIXTURE_PID" 2>/dev/null || true
  fi
  rm -f -- "$BASE/.profile-locks/"*.maintenance 2>/dev/null || true
  for profile in CodexA CodexB; do
    "$SCRIPT" stop --profile "$profile" --runtime-base "$BASE" \
      >/dev/null 2>&1 || true
  done
  if [ -n "${LEAK_PID:-}" ] && kill -0 "$LEAK_PID" 2>/dev/null; then
    kill -TERM "$LEAK_PID" 2>/dev/null || true
  fi
  rm -rf -- "$TEST_ROOT"
}
trap cleanup EXIT

cat >"$FAKE_SOURCE" <<'GO'
package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

func main() {
	if len(os.Args) != 4 || os.Args[1] != "serve" || os.Args[2] != "--config" {
		os.Exit(2)
	}
	values := readConfig(os.Args[3])
	tokenPath := values["identity.local_token_path"]
	if err := os.WriteFile(tokenPath, []byte("test-profile-token\n"), 0o600); err != nil {
		panic(err)
	}
	auth, err := os.ReadFile(filepath.Join(values["runtime.codex.account_home_root"], values["runtime.codex.account_profile"], "auth.json"))
	if err != nil {
		panic(err)
	}
	sum := sha256.Sum256(auth)
	observed := map[string]any{
		"home":        os.Getenv("HOME"),
		"codex_home":  os.Getenv("CODEX_HOME"),
		"auth_sha":    hex.EncodeToString(sum[:]),
		"environment": os.Environ(),
		"pid":         os.Getpid(),
	}
	encoded, _ := json.Marshal(observed)
	if err := os.WriteFile(filepath.Join(os.Getenv("HOME"), "observed.json"), encoded, 0o600); err != nil {
		panic(err)
	}
	handler := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost ||
			request.URL.Path != "/api/v1/commands/orquesta.system.status" ||
			request.Header.Get("Authorization") != "Bearer test-profile-token" {
			http.Error(writer, "unauthorized", http.StatusUnauthorized)
			return
		}
		var input struct {
			Version    string         `json:"version"`
			RequestRef string         `json:"request_ref"`
			ProjectRef string         `json:"project_ref"`
			Payload    map[string]any `json:"payload"`
		}
		if json.NewDecoder(request.Body).Decode(&input) != nil ||
			input.Version != "1" ||
			!validReadinessRef(input.RequestRef) ||
			input.ProjectRef != values["project.default"] {
			http.Error(writer, "bad request", http.StatusBadRequest)
			return
		}
		count, err := os.OpenFile(
			filepath.Join(os.Getenv("HOME"), "readiness-count.txt"),
			os.O_CREATE|os.O_WRONLY|os.O_APPEND,
			0o600,
		)
		if err != nil {
			panic(err)
		}
		if _, err := count.WriteString("request\n"); err != nil {
			panic(err)
		}
		if err := count.Close(); err != nil {
			panic(err)
		}
		if _, err := os.Stat(filepath.Join(os.Getenv("HOME"), "slow-readiness")); err == nil {
			time.Sleep(750 * time.Millisecond)
		} else if !os.IsNotExist(err) {
			panic(err)
		}
		if err := os.WriteFile(filepath.Join(os.Getenv("HOME"), "readiness-ref.txt"), []byte(input.RequestRef), 0o600); err != nil {
			panic(err)
		}
		writer.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(writer).Encode(map[string]any{
			"command_id": "orquesta.system.status", "command_version": "1",
			"request_ref": input.RequestRef,
			"data": map[string]int{
				"goals": 0, "running_goals": 0,
				"pending_actions": 0, "quarantined_actions": 0,
			},
			"audit_ref": "audit:profile-readiness",
		}); err != nil {
			panic(err)
		}
	})
	listener, err := net.Listen("tcp", values["server.listen"])
	if err != nil {
		panic(err)
	}
	server := &http.Server{Handler: handler}
	go server.Serve(listener)
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM)
	<-stop
	server.Close()
}

func validReadinessRef(value string) bool {
	const prefix = "request:profile-server-readiness:"
	suffix := strings.TrimPrefix(value, prefix)
	decoded, err := hex.DecodeString(suffix)
	return strings.HasPrefix(value, prefix) && err == nil && len(decoded) == sha256.Size
}

func readConfig(path string) map[string]string {
	handle, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer handle.Close()
	values := make(map[string]string)
	section := ""
	scanner := bufio.NewScanner(handle)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.TrimSuffix(strings.TrimPrefix(line, "["), "]")
			continue
		}
		key, raw, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		key = strings.TrimSpace(key)
		raw = strings.Trim(strings.TrimSpace(raw), `"`)
		values[section+"."+key] = raw
	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}
	return values
}
GO

GOCACHE="$GO_CACHE" go build -o "$FAKE_BINARY" "$FAKE_SOURCE"
chmod 755 "$FAKE_BINARY"

# Acredita la identidad y el verbo contra el binario real sin arrancar runtime:
# `serve --help` termina en el parser antes de abrir red, estado o SQLite.
mkdir -m 700 "$REAL_CLI_PROBE_ROOT"
GOCACHE="$GO_CACHE" GOFLAGS=-mod=vendor \
  go build -o "$REAL_BINARY" "$ROOT/cmd/orquesta"
chmod 755 "$REAL_BINARY"
go version -m "$REAL_BINARY" |
  grep -Eq '^[[:space:]]*path[[:space:]]+orquesta/cmd/orquesta$'
(
  cd "$REAL_CLI_PROBE_ROOT"
  env -i \
    "HOME=$REAL_CLI_PROBE_ROOT" \
    "PATH=/usr/local/bin:/usr/bin:/bin" \
    "$REAL_BINARY" serve --help
) >"$TEST_ROOT/real-cli-probe.out" 2>"$TEST_ROOT/real-cli-probe.err"
grep -Fq 'orquesta serve [--config' "$TEST_ROOT/real-cli-probe.out"
if [ -s "$TEST_ROOT/real-cli-probe.err" ]; then
  sed -n '1,20p' "$TEST_ROOT/real-cli-probe.err" >&2
  exit 1
fi
[ -z "$(find "$REAL_CLI_PROBE_ROOT" -mindepth 1 -print -quit)" ]

mkdir -m 700 "$FAKE_TOOLCHAIN" "$FAKE_TOOLCHAIN/bin"
printf '#!/bin/sh\nexit 97\n' >"$FAKE_TOOLCHAIN/bin/go"
chmod 700 "$FAKE_TOOLCHAIN/bin/go"
mkdir -m 700 \
  "$UNSAFE_TOOLCHAIN" \
  "$UNSAFE_TOOLCHAIN/bin" \
  "$UNSAFE_TOOLCHAIN/pkg" \
  "$UNSAFE_TOOLCHAIN/pkg/tool"
printf '#!/bin/sh\nexit 97\n' >"$UNSAFE_TOOLCHAIN/bin/go"
printf 'unsafe tool\n' >"$UNSAFE_TOOLCHAIN/pkg/tool/compile"
chmod 700 "$UNSAFE_TOOLCHAIN/bin/go"
chmod 720 "$UNSAFE_TOOLCHAIN/pkg/tool/compile"

write_fake_bwrap() {
  command_path="$1"
  capacity="$2"
  cat >"$command_path" <<EOF
#!/usr/bin/env python3
import os
import sys

capacity = $capacity
if len(sys.argv) != 5 or sys.argv[1] != "--args" or sys.argv[3:] != ["--", "/bin/true"]:
    raise SystemExit(97)
with open("/proc/self/fd/" + sys.argv[2], "rb") as arguments:
    values = [item for item in arguments.read().split(b"\\0") if item]
expected = [
    b"--unshare-all", b"--unshare-user", b"--disable-userns",
    b"--assert-userns-disabled", b"--die-with-parent", b"--new-session",
    b"--clearenv", b"--ro-bind", b"/", b"/",
]
if values[:len(expected)] != expected:
    raise SystemExit(98)
count = len(values)
count += len(sys.argv) - 1
if count > capacity:
    print("bwrap: Exceeded maximum number of arguments %d" % capacity, file=sys.stderr)
    raise SystemExit(1)
EOF
  chmod 755 "$command_path"
}

write_fake_bwrap "$FAKE_BWRAP_9000" 9000
write_fake_bwrap "$FAKE_BWRAP_65536" 65536
cat >"$FAKE_BWRAP_FAILED" <<'PY'
#!/usr/bin/env python3
import sys

print("bwrap: probe failure", file=sys.stderr)
raise SystemExit(1)
PY
chmod 755 "$FAKE_BWRAP_FAILED"

available_port() {
  python3 - <<'PY'
import socket

with socket.socket() as listener:
    listener.bind(("127.0.0.1", 0))
    print(listener.getsockname()[1])
PY
}

prepare_account() {
  profile="$1"
  account="$2"
  mkdir -m 700 "$ACCOUNTS/$profile"
  printf '{"account":"%s"}\n' "$account" >"$ACCOUNTS/$profile/auth.json"
  chmod 600 "$ACCOUNTS/$profile/auth.json"
}

write_config() {
  profile="$1"
  port="$2"
  max_concurrent="${3:-1}"
  auth_max_document_bytes="$4"
  go_toolchain_root="${5-}"
  runtime_root="$BASE/$profile"
  config="$TEST_ROOT/$profile.toml"
  cat >"$config" <<EOF
[server]
listen = "127.0.0.1:$port"

[state.sqlite]
path = "$runtime_root/state/orquesta.sqlite"

[artifact.filesystem]
root = "$runtime_root/artifacts"

[credentials.local]
path = "$runtime_root/secrets/credentials.json"

[runtime.codex]
command = "$FAKE_BINARY"
max_concurrent_executions = $max_concurrent
work_root = "$runtime_root/work"
cache_root = "$runtime_root/cache/codex-go"
go_toolchain_root = "$go_toolchain_root"
env_allowlist = ["PATH", "HOME", "CODEX_HOME"]
account_home_root = "$ACCOUNTS"
account_profile = "$profile"
account_auth_max_document_bytes = $auth_max_document_bytes

[workspace.local]
root = "$runtime_root/workspaces"

[identity]
local_token_path = "$runtime_root/secrets/local-owner.token"

[project]
default = "project:default"

[config]
effective_path = "$runtime_root/effective-config.json"
EOF
  chmod 600 "$config"
}

start_profile() {
  profile="$1"
  "$SCRIPT" start \
    --profile "$profile" \
    --runtime-base "$BASE" \
    --source-codex-home "$ACCOUNTS/$profile" \
    --binary "$FAKE_BINARY" \
    --config "$TEST_ROOT/$profile.toml" \
    --exec-path "$(dirname "$(realpath "$(command -v go)")"):/usr/local/bin:/usr/bin"
}

start_profile_with_driver() {
  driver="$1"
  profile="$2"
  "$driver" start \
    --profile "$profile" \
    --runtime-base "$BASE" \
    --source-codex-home "$ACCOUNTS/$profile" \
    --binary "$FAKE_BINARY" \
    --config "$TEST_ROOT/$profile.toml" \
    --exec-path "$(dirname "$(realpath "$(command -v go)")"):/usr/local/bin:/usr/bin"
}

start_profile_with_maintenance() {
  profile="$1"
  maintenance_ref="$2"
  "$SCRIPT" start \
    --profile "$profile" \
    --runtime-base "$BASE" \
    --source-codex-home "$ACCOUNTS/$profile" \
    --binary "$FAKE_BINARY" \
    --config "$TEST_ROOT/$profile.toml" \
    --exec-path "$(dirname "$(realpath "$(command -v go)")"):/usr/local/bin:/usr/bin" \
    --maintenance-ref "$maintenance_ref"
}

write_maintenance_marker() {
  profile="$1"
  maintenance_ref="$2"
  marker="$BASE/.profile-locks/$profile.maintenance"
  temporary="$BASE/.profile-locks/.$profile.maintenance.tmp"
  exec 6<>"$BASE/.profile-locks/$profile.control.lock"
  flock 6
  printf 'maintenance_ref=%s\n' "$maintenance_ref" >"$temporary"
  chmod 600 -- "$temporary"
  mv -f -- "$temporary" "$marker"
  flock -u 6
  exec 6>&-
}

remove_maintenance_marker() {
  profile="$1"
  exec 6<>"$BASE/.profile-locks/$profile.control.lock"
  flock 6
  rm -f -- "$BASE/.profile-locks/$profile.maintenance"
  flock -u 6
  exec 6>&-
}

start_profile_with_nofile() {
  profile="$1"
  nofile="$2"
  (
    ulimit -S -n "$nofile"
    start_profile "$profile"
  )
}

enable_bubblewrap_attestor() {
  profile="$1"
  max_concurrent="$2"
  bubblewrap_command="${3:-$FAKE_BWRAP_65536}"
  cat >>"$TEST_ROOT/$profile.toml" <<EOF

[repository.local]
seed_path = "$REPOSITORY"

[test_attestor]
provider = "bubblewrap"
max_subject_bytes = 536870912
max_concurrent_runs = $max_concurrent

[test_attestor.bubblewrap]
command = "$bubblewrap_command"
EOF
  chmod 600 "$TEST_ROOT/$profile.toml"
}

enable_microvm_attestor() {
  profile="$1"
  launcher_socket="$2"
  cat >>"$TEST_ROOT/$profile.toml" <<EOF

[test_attestor]
provider = "microvm"
max_subject_bytes = 536870912
max_concurrent_runs = 16

[test_attestor.microvm]
launcher_socket = "$launcher_socket"
expected_asset_digest = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
guest_memory_mib = 4096
EOF
  chmod 600 "$TEST_ROOT/$profile.toml"
}

start_launcher_fixture() {
  launcher_socket="$1"
  observation="$2"
  ready_file="$3"
  unlink_after="${4:-no}"
  connection_count="${5:-1}"
  python3 - \
    "$launcher_socket" \
    "$observation" \
    "$ready_file" \
    "$unlink_after" \
    "$connection_count" <<'PY' &
import os
import socket
import sys

socket_path, observation_path, ready_path, unlink_after, connection_count_text = sys.argv[1:]
connection_count = int(connection_count_text)
listener = socket.socket(socket.AF_UNIX, socket.SOCK_SEQPACKET)
listener.bind(socket_path)
os.chmod(socket_path, 0o660)
listener.listen(1)
with open(ready_path, "x", encoding="ascii") as ready:
    ready.write("ready\n")
payload_bytes = 0
for _ in range(connection_count):
    connection, _ = listener.accept()
    payload_bytes += len(connection.recv(1))
    connection.close()
with open(observation_path, "x", encoding="ascii") as observation:
    observation.write(str(payload_bytes) + "\n")
listener.close()
if unlink_after == "yes":
    os.unlink(socket_path)
PY
  launcher_pid="$!"
  LAUNCHER_FIXTURE_PID="$launcher_pid"
  for _ in $(seq 1 100); do
    [ -e "$ready_file" ] && [ -S "$launcher_socket" ] && return
    kill -0 "$launcher_pid" 2>/dev/null || return 1
    sleep 0.01
  done
  return 1
}

expect_probe_status() {
  expected_status="$1"
  shift
  set +e
  "$@" >/dev/null 2>&1
  observed_status="$?"
  set -e
  [ "$observed_status" -eq "$expected_status" ]
}

expect_failure() {
  expected="$1"
  shift
  set +e
  "$@" >"$TEST_ROOT/failure.out" 2>"$TEST_ROOT/failure.err"
  result=$?
  set -e
  [ "$result" -ne 0 ]
  if ! grep -q "reason_code=$expected" "$TEST_ROOT/failure.err"; then
    sed -n '1,20p' "$TEST_ROOT/failure.err" >&2
    exit 1
  fi
  if grep -E '/home/|/tmp/|auth.json|token' "$TEST_ROOT/failure.err"; then
    echo "el error publicó una ruta o nombre sensible" >&2
    exit 1
  fi
}

assert_observed_environment() {
  profile="$1"
  python3 - "$BASE/$profile/daemon-home/observed.json" \
    "$BASE/$profile/daemon-home" <<'PY'
import json
import sys

with open(sys.argv[1], encoding="utf-8") as handle:
    value = json.load(handle)
expected_home = sys.argv[2]
if value["home"] != expected_home or value["codex_home"] != expected_home:
    raise SystemExit("daemon home incorrecto")
names = {entry.split("=", 1)[0] for entry in value["environment"]}
if names != {"HOME", "CODEX_HOME", "PATH"}:
    raise SystemExit(f"entorno heredado: {sorted(names)}")
PY
}

bash -n "$SCRIPT"
PYTHONPYCACHEPREFIX="$TEST_ROOT/pycache" \
  python3 -m py_compile "$FIRECRACKER_PROBE" "$MAINTENANCE_MARKER_READER"
grep -Fq '[ "$(uname -s 2>/dev/null)" = "Linux" ] || fail "platform_unsupported"' \
  "$SCRIPT"

# Status de un perfil inexistente observa sin crear locks ni runtime.
set +e
missing_status_output="$(
  "$SCRIPT" status --profile CodexMissing --runtime-base "$BASE"
)"
missing_status_code="$?"
set -e
[ "$missing_status_code" -eq 3 ]
grep -q "status=stopped profile=CodexMissing" <<<"$missing_status_output"
[ ! -e "$BASE/.profile-locks" ]
[ ! -e "$BASE/CodexMissing" ]

# La sonda UDS no envía ninguna petición: solo conecta, acredita peer y cierra.
mkdir -m 700 "$LAUNCHER_ROOT"
launcher_uid="$(id -u)"
launcher_gid="$(id -g)"
PROFILE_TEST_DRIVER_ROOT="$TEST_ROOT/profile-test-driver"
mkdir -m 700 "$PROFILE_TEST_DRIVER_ROOT" "$PROFILE_TEST_DRIVER_ROOT/lib"
cp -- "$SCRIPT" "$PROFILE_TEST_DRIVER_ROOT/orquesta_profile_server.sh"
cp -- \
  "$FIRECRACKER_PROBE" \
  "$MAINTENANCE_MARKER_READER" \
  "$ROOT/scripts/lib/pidfd_signal.py" \
  "$PROFILE_TEST_DRIVER_ROOT/lib/"
# Producción conserva peer root:root. La copia aislada solo permite que el
# fixture no privilegiado recorra start/status/cleanup con el mismo protocolo.
sed -i \
  -e "s/--trusted-uid 0/--trusted-uid $launcher_uid/" \
  -e "s/--trusted-gid 0/--trusted-gid $launcher_gid/" \
  "$PROFILE_TEST_DRIVER_ROOT/orquesta_profile_server.sh"
chmod 755 \
  "$PROFILE_TEST_DRIVER_ROOT/orquesta_profile_server.sh" \
  "$PROFILE_TEST_DRIVER_ROOT/lib/firecracker_launcher_probe.py" \
  "$PROFILE_TEST_DRIVER_ROOT/lib/profile_maintenance_marker.py" \
  "$PROFILE_TEST_DRIVER_ROOT/lib/pidfd_signal.py"
connectable_socket="$LAUNCHER_ROOT/connectable.sock"
connectable_observation="$LAUNCHER_ROOT/connectable.observed"
start_launcher_fixture \
  "$connectable_socket" \
  "$connectable_observation" \
  "$LAUNCHER_ROOT/connectable.ready"
"$FIRECRACKER_PROBE" \
  --socket "$connectable_socket" \
  --trusted-uid "$launcher_uid" \
  --trusted-gid "$launcher_gid"
wait "$LAUNCHER_FIXTURE_PID"
LAUNCHER_FIXTURE_PID=""
[ "$(<"$connectable_observation")" = "0" ]

# Ausencia y un inode UDS sin listener fallan como indisponibilidad.
expect_probe_status 41 \
  "$FIRECRACKER_PROBE" \
  --socket "$LAUNCHER_ROOT/missing.sock" \
  --trusted-uid "$launcher_uid" \
  --trusted-gid "$launcher_gid"
rejected_socket="$LAUNCHER_ROOT/rejected.sock"
python3 - "$rejected_socket" <<'PY'
import os
import socket
import sys

listener = socket.socket(socket.AF_UNIX, socket.SOCK_SEQPACKET)
listener.bind(sys.argv[1])
os.chmod(sys.argv[1], 0o660)
listener.close()
PY
expect_probe_status 41 \
  "$FIRECRACKER_PROBE" \
  --socket "$rejected_socket" \
  --trusted-uid "$launcher_uid" \
  --trusted-gid "$launcher_gid"

# Symlink y peer con GID distinto no pueden acreditar el launcher configurado.
identity_socket="$LAUNCHER_ROOT/identity.sock"
start_launcher_fixture \
  "$identity_socket" \
  "$LAUNCHER_ROOT/identity.observed" \
  "$LAUNCHER_ROOT/identity.ready"
expect_probe_status 43 \
  "$FIRECRACKER_PROBE" \
  --socket "$identity_socket" \
  --trusted-uid "$launcher_uid" \
  --trusted-gid "$((launcher_gid + 1))"
wait "$LAUNCHER_FIXTURE_PID"
LAUNCHER_FIXTURE_PID=""
symlink_target="$LAUNCHER_ROOT/symlink-target.sock"
start_launcher_fixture \
  "$symlink_target" \
  "$LAUNCHER_ROOT/symlink.observed" \
  "$LAUNCHER_ROOT/symlink.ready"
ln -s "$symlink_target" "$LAUNCHER_ROOT/symlink.sock"
expect_probe_status 42 \
  "$FIRECRACKER_PROBE" \
  --socket "$LAUNCHER_ROOT/symlink.sock" \
  --trusted-uid "$launcher_uid" \
  --trusted-gid "$launcher_gid"
kill -TERM "$LAUNCHER_FIXTURE_PID"
wait "$LAUNCHER_FIXTURE_PID" 2>/dev/null || true
LAUNCHER_FIXTURE_PID=""

prepare_account CodexA account-a
prepare_account CodexB account-b
write_config CodexA "$(available_port)" 1 1048576
write_config CodexB "$(available_port)" 1 1048576

# Dos cuentas arrancan desde el mismo cwd sin compartir HOME, auth ni estado.
start_profile CodexA >"$TEST_ROOT/start-a-1.out" &
start_one_pid="$!"
start_profile CodexA >"$TEST_ROOT/start-a-2.out" &
start_two_pid="$!"
wait "$start_one_pid"
wait "$start_two_pid"
pid_a="$(<"$BASE/CodexA/run/server.pid")"
grep -q "status=running profile=CodexA pid=$pid_a" \
  "$TEST_ROOT/start-a-1.out"
grep -q "status=running profile=CodexA pid=$pid_a" \
  "$TEST_ROOT/start-a-2.out"
start_profile CodexB >"$TEST_ROOT/start-b.out"
assert_observed_environment CodexA
assert_observed_environment CodexB
[ "$(stat -Lc '%a' -- "$BASE/CodexA/cache/codex-go")" = 700 ]
[ "$(stat -Lc '%a' -- "$BASE/CodexB/cache/codex-go")" = 700 ]
[ "$(realpath -e -- "$BASE/CodexA/cache/codex-go")" != \
  "$(realpath -e -- "$BASE/CodexB/cache/codex-go")" ]
grep -Fq 'go_toolchain_root = ""' "$BASE/CodexA/config/orquesta.toml"
grep -Fq 'go_toolchain_root = ""' "$BASE/CodexB/config/orquesta.toml"
readiness_ref_a="$(<"$BASE/CodexA/daemon-home/readiness-ref.txt")"
[[ "$readiness_ref_a" =~ ^request:profile-server-readiness:[0-9a-f]{64}$ ]]
[ ! -e "$BASE/CodexA/daemon-home/auth.json" ]
[ ! -e "$BASE/CodexA/daemon-home/.codex" ]
[ "$(sha256sum "$ACCOUNTS/CodexA/auth.json" | awk '{print $1}')" != \
  "$(sha256sum "$ACCOUNTS/CodexB/auth.json" | awk '{print $1}')" ]

# Start y stop son idempotentes, y status conserva la identidad del PID.
start_profile CodexA >"$TEST_ROOT/start-a-replay.out"
[ "$(<"$BASE/CodexA/run/server.pid")" = "$pid_a" ]
"$SCRIPT" status --profile CodexA --runtime-base "$BASE" |
  grep -q "status=running profile=CodexA pid=$pid_a"

# El lease de perfil sigue retenido después de que el wrapper start termine.
exec 7<>"$BASE/.profile-locks/CodexA.lease.lock"
if flock -n 7; then
  echo "el daemon no retuvo el lease de perfil" >&2
  exit 1
fi
exec 7>&-

# Un contrato inseguro no arranca ni reemplaza al daemon vivo.
write_config CodexA "$(available_port)" 70 1048576
expect_failure orquesta_config_invalid start_profile CodexA
[ "$(<"$BASE/CodexA/run/server.pid")" = "$pid_a" ]
if [ "$(id -u)" -ne 0 ]; then
  write_config CodexA "$(available_port)" 1 1048576 "$FAKE_TOOLCHAIN"
  expect_failure orquesta_config_invalid start_profile CodexA
  [ "$(<"$BASE/CodexA/run/server.pid")" = "$pid_a" ]
fi
write_config CodexA "$(available_port)" 1 1048576 "$UNSAFE_TOOLCHAIN"
expect_failure orquesta_config_invalid start_profile CodexA
[ "$(<"$BASE/CodexA/run/server.pid")" = "$pid_a" ]
write_config CodexA "$(available_port)" 1 1048576

"$SCRIPT" stop --profile CodexA --runtime-base "$BASE" >/dev/null
"$SCRIPT" stop --profile CodexA --runtime-base "$BASE" >/dev/null
exec 7<>"$BASE/.profile-locks/CodexA.lease.lock"
flock -n 7
flock -u 7
exec 7>&-

# Un marker canónico excluye toda mutación salvo la ligada a su ref. Status
# sigue siendo observación read-only y el marker nunca lo retira este script.
MAINTENANCE_REF="$(
  printf '%s' "profile-maintenance-CodexA" | sha256sum | awk '{print $1}'
)"
readonly MAINTENANCE_REF
OTHER_MAINTENANCE_REF="$(
  printf '%s' "profile-maintenance-other" | sha256sum | awk '{print $1}'
)"
readonly OTHER_MAINTENANCE_REF
expect_failure maintenance_ref_invalid \
  "$SCRIPT" stop \
  --profile CodexA \
  --runtime-base "$BASE" \
  --maintenance-ref not-a-sha256
expect_failure maintenance_marker_missing \
  "$SCRIPT" stop \
  --profile CodexA \
  --runtime-base "$BASE" \
  --maintenance-ref "$MAINTENANCE_REF"
write_maintenance_marker CodexA "$MAINTENANCE_REF"
maintenance_marker="$BASE/.profile-locks/CodexA.maintenance"
[ "$(stat -Lc '%a:%u:%h' -- "$maintenance_marker")" = \
  "600:$(id -u):1" ]
set +e
maintenance_status_output="$(
  "$SCRIPT" status --profile CodexA --runtime-base "$BASE"
)"
maintenance_status_code="$?"
set -e
[ "$maintenance_status_code" -eq 3 ]
grep -q "status=stopped profile=CodexA" <<<"$maintenance_status_output"
expect_failure profile_maintenance_active start_profile CodexA
expect_failure maintenance_ref_mismatch \
  start_profile_with_maintenance CodexA "$OTHER_MAINTENANCE_REF"
start_profile_with_maintenance CodexA "$MAINTENANCE_REF" >/dev/null
maintenance_pid="$(<"$BASE/CodexA/run/server.pid")"
"$SCRIPT" status --profile CodexA --runtime-base "$BASE" |
  grep -q "status=running profile=CodexA pid=$maintenance_pid"
expect_failure profile_maintenance_active \
  "$SCRIPT" stop --profile CodexA --runtime-base "$BASE"
expect_failure maintenance_ref_mismatch \
  "$SCRIPT" stop \
  --profile CodexA \
  --runtime-base "$BASE" \
  --maintenance-ref "$OTHER_MAINTENANCE_REF"
kill -0 "$maintenance_pid"

# Modo, hardlink, symlink, owner y framing alterados fallan cerrados. Cada
# negativo conserva tanto el daemon como el marker para una reparación segura.
chmod 640 -- "$maintenance_marker"
expect_failure maintenance_marker_invalid \
  "$SCRIPT" stop \
  --profile CodexA \
  --runtime-base "$BASE" \
  --maintenance-ref "$MAINTENANCE_REF"
chmod 600 -- "$maintenance_marker"
ln -- "$maintenance_marker" "$maintenance_marker.hardlink"
expect_failure maintenance_marker_invalid \
  "$SCRIPT" stop \
  --profile CodexA \
  --runtime-base "$BASE" \
  --maintenance-ref "$MAINTENANCE_REF"
rm -- "$maintenance_marker.hardlink"
mv -- "$maintenance_marker" "$maintenance_marker.target"
ln -s -- "$maintenance_marker.target" "$maintenance_marker"
expect_failure maintenance_marker_invalid \
  "$SCRIPT" stop \
  --profile CodexA \
  --runtime-base "$BASE" \
  --maintenance-ref "$MAINTENANCE_REF"
rm -- "$maintenance_marker"
mv -- "$maintenance_marker.target" "$maintenance_marker"
if [ "$(id -u)" -eq 0 ]; then
  chown 1 -- "$maintenance_marker"
  expect_failure maintenance_marker_invalid \
    "$SCRIPT" stop \
    --profile CodexA \
    --runtime-base "$BASE" \
    --maintenance-ref "$MAINTENANCE_REF"
  chown 0 -- "$maintenance_marker"
fi
printf 'maintenance_ref=%s\nextra\n' "$MAINTENANCE_REF" \
  >"$maintenance_marker"
expect_failure maintenance_marker_invalid \
  "$SCRIPT" stop \
  --profile CodexA \
  --runtime-base "$BASE" \
  --maintenance-ref "$MAINTENANCE_REF"
write_maintenance_marker CodexA "$MAINTENANCE_REF"
kill -0 "$maintenance_pid"
"$SCRIPT" stop \
  --profile CodexA \
  --runtime-base "$BASE" \
  --maintenance-ref "$MAINTENANCE_REF" >/dev/null
[ -f "$maintenance_marker" ]
remove_maintenance_marker CodexA
[ ! -e "$maintenance_marker" ]

# El perfil persiste y una reautenticación legítima no se copia ni se rechaza.
printf '{"account":"account-a-refreshed"}\n' >"$ACCOUNTS/CodexA/auth.json"
chmod 600 "$ACCOUNTS/CodexA/auth.json"
write_config CodexA "$(available_port)" 1 1048576
start_profile CodexA >/dev/null
readiness_ref_restart="$(<"$BASE/CodexA/daemon-home/readiness-ref.txt")"
[ "$readiness_ref_restart" != "$readiness_ref_a" ]
"$SCRIPT" stop --profile CodexA --runtime-base "$BASE" >/dev/null

# El límite de auth procede del snapshot TOML, no de un 1 MiB local.
python3 - "$ACCOUNTS/CodexA/auth.json" <<'PY'
import json
import sys

with open(sys.argv[1], "w", encoding="utf-8") as handle:
    json.dump({"account": "account-a-large", "padding": "x" * 1100000}, handle)
PY
chmod 600 "$ACCOUNTS/CodexA/auth.json"
write_config CodexA "$(available_port)" 1 2097152
start_profile CodexA >/dev/null
"$SCRIPT" stop --profile CodexA --runtime-base "$BASE" >/dev/null
write_config CodexA "$(available_port)" 1 1048576
expect_failure source_auth_too_large start_profile CodexA
printf '{"account":"account-a-restored"}\n' >"$ACCOUNTS/CodexA/auth.json"
chmod 600 "$ACCOUNTS/CodexA/auth.json"
write_config CodexA "$(available_port)" 1 1048576
sed -i '/account_auth_max_document_bytes/d' "$TEST_ROOT/CodexA.toml"
expect_failure orquesta_config_invalid start_profile CodexA
write_config CodexA "$(available_port)" 1 1048576

# Traversal, symlinks y permisos amplios fallan antes de lanzar.
expect_failure profile_invalid \
  "$SCRIPT" status --profile ../CodexA --runtime-base "$BASE"
chmod 775 "$ACCOUNTS/CodexA"
expect_failure source_code_home_not_private start_profile CodexA
chmod 700 "$ACCOUNTS/CodexA"
mv "$ACCOUNTS/CodexA/auth.json" "$ACCOUNTS/CodexA/auth.real"
ln -s auth.real "$ACCOUNTS/CodexA/auth.json"
expect_failure source_auth_not_private start_profile CodexA
rm "$ACCOUNTS/CodexA/auth.json"
mv "$ACCOUNTS/CodexA/auth.real" "$ACCOUNTS/CodexA/auth.json"

# El binding cuenta/config y max=1 son obligatorios.
write_config CodexA "$(available_port)" 70 1048576
expect_failure orquesta_config_invalid start_profile CodexA

"$SCRIPT" stop --profile CodexB --runtime-base "$BASE" >/dev/null

# Una autorización durable lenta no provoca una tormenta de health-checks:
# después de que el puerto acepte, el bootstrap espera una única respuesta.
rm -f -- "$BASE/CodexB/daemon-home/readiness-count.txt"
touch "$BASE/CodexB/daemon-home/slow-readiness"
chmod 600 "$BASE/CodexB/daemon-home/slow-readiness"
write_config CodexB "$(available_port)" 1 1048576
start_profile CodexB >/dev/null
[ "$(wc -l <"$BASE/CodexB/daemon-home/readiness-count.txt")" -eq 1 ]
"$SCRIPT" stop --profile CodexB --runtime-base "$BASE" >/dev/null

# El preflight usa solo blobs regulares del HEAD autorizado y replica las
# reservas NOFILE y el número real de argumentos del attestor bubblewrap.
mkdir -m 700 "$REPOSITORY"
git -C "$REPOSITORY" init -q
for entry in $(seq 1 20); do
  printf 'tracked-%s\n' "$entry" >"$REPOSITORY/tracked-$entry"
done
git -C "$REPOSITORY" add .
git -C "$REPOSITORY" \
  -c user.name=Orquesta \
  -c user.email=orquesta.invalid \
  commit -qm "fixture"
for entry in $(seq 1 100); do
  printf 'untracked-%s\n' "$entry" >"$REPOSITORY/untracked-$entry"
done

write_config CodexA "$(available_port)" 1 1048576
enable_bubblewrap_attestor CodexA 1
start_profile_with_nofile CodexA 128 >/dev/null
"$SCRIPT" stop --profile CodexA --runtime-base "$BASE" >/dev/null

write_config CodexA "$(available_port)" 1 1048576
enable_bubblewrap_attestor CodexA 2
expect_failure test_attestor_nofile_insufficient \
  start_profile_with_nofile CodexA 128
grep -q 'action=increase_process_nofile_limit' "$TEST_ROOT/failure.err"
[ ! -e "$BASE/CodexA/run/server.pid" ]

# 1800 blobs fuerzan más de 9000 argumentos; el mismo HEAD cabe en 65536.
for entry in $(seq 21 1820); do
  printf 'tracked-%s\n' "$entry" >"$REPOSITORY/tracked-$entry"
done
mkdir "$REPOSITORY/nested"
ln -s ../tracked-1 "$REPOSITORY/nested/link"
git -C "$REPOSITORY" add -- 'tracked-*' nested/link
git -C "$REPOSITORY" \
  -c user.name=Orquesta \
  -c user.email=orquesta.invalid \
  commit -qm "fixture amplia"

write_config CodexA "$(available_port)" 1 1048576
enable_bubblewrap_attestor CodexA 1 "$FAKE_BWRAP_9000"
expect_failure test_attestor_bubblewrap_arguments_insufficient \
  start_profile_with_nofile CodexA 4096
grep -q 'action=configure_bubblewrap_with_sufficient_argument_capacity' \
  "$TEST_ROOT/failure.err"
[ ! -e "$BASE/CodexA/run/server.pid" ]

write_config CodexA "$(available_port)" 1 1048576
enable_bubblewrap_attestor CodexA 1 "$FAKE_BWRAP_FAILED"
expect_failure test_attestor_bubblewrap_probe_failed \
  start_profile_with_nofile CodexA 4096
grep -q 'action=repair_configured_bubblewrap' "$TEST_ROOT/failure.err"
[ ! -e "$BASE/CodexA/run/server.pid" ]

write_config CodexA "$(available_port)" 1 1048576
enable_bubblewrap_attestor CodexA 1 "$FAKE_BWRAP_65536"
start_profile_with_nofile CodexA 4096 >/dev/null
"$SCRIPT" stop --profile CodexA --runtime-base "$BASE" >/dev/null

# Un provider distinto de bubblewrap no queda sujeto a ningún preflight suyo.
write_config CodexA "$(available_port)" 1 1048576
start_profile_with_nofile CodexA 80 >/dev/null
"$SCRIPT" stop --profile CodexA --runtime-base "$BASE" >/dev/null

# MicroVM falla antes de publicar PID/running cuando el launcher no escucha.
write_config CodexA "$(available_port)" 1 1048576
profile_missing_socket="/run/orquesta-profile-server-test-$$.sock"
[ ! -e "$profile_missing_socket" ]
enable_microvm_attestor CodexA "$profile_missing_socket"
expect_failure test_attestor_microvm_launcher_unavailable start_profile CodexA
[ ! -e "$BASE/CodexA/run/server.pid" ]

# Un launcher vivo permite el start, pero status deja de publicar running si
# ese mismo UDS desaparece después de las sondas de arranque.
status_socket="$LAUNCHER_ROOT/disappears-before-status.sock"
status_observation="$LAUNCHER_ROOT/disappears-before-status.observed"
start_launcher_fixture \
  "$status_socket" \
  "$status_observation" \
  "$LAUNCHER_ROOT/disappears-before-status.ready" \
  yes \
  2
write_config CodexA "$(available_port)" 1 1048576
enable_microvm_attestor CodexA "$status_socket"
start_profile_with_driver \
  "$PROFILE_TEST_DRIVER_ROOT/orquesta_profile_server.sh" \
  CodexA >/dev/null
status_daemon_pid="$(<"$BASE/CodexA/run/server.pid")"
wait "$LAUNCHER_FIXTURE_PID"
LAUNCHER_FIXTURE_PID=""
[ "$(<"$status_observation")" = "0" ]
expect_failure test_attestor_microvm_launcher_unavailable \
  "$PROFILE_TEST_DRIVER_ROOT/orquesta_profile_server.sh" \
  status \
  --profile CodexA \
  --runtime-base "$BASE"
kill -0 "$status_daemon_pid"
"$PROFILE_TEST_DRIVER_ROOT/orquesta_profile_server.sh" \
  stop \
  --profile CodexA \
  --runtime-base "$BASE" >/dev/null
if kill -0 "$status_daemon_pid" 2>/dev/null; then
  echo "el daemon sobrevivió al stop posterior al status microVM" >&2
  exit 1
fi

# Si el launcher muere tras la primera sonda, el daemon ya arrancado se cierra
# antes de publicar running y se retira toda su identidad de proceso.
race_socket="$LAUNCHER_ROOT/disappears-after-preflight.sock"
start_launcher_fixture \
  "$race_socket" \
  "$LAUNCHER_ROOT/disappears-after-preflight.observed" \
  "$LAUNCHER_ROOT/disappears-after-preflight.ready" \
  yes
write_config CodexA "$(available_port)" 1 1048576
enable_microvm_attestor CodexA "$race_socket"
expect_failure test_attestor_microvm_launcher_unavailable \
  start_profile_with_driver \
  "$PROFILE_TEST_DRIVER_ROOT/orquesta_profile_server.sh" \
  CodexA
wait "$LAUNCHER_FIXTURE_PID"
LAUNCHER_FIXTURE_PID=""
race_daemon_pid="$(
  python3 - "$BASE/CodexA/daemon-home/observed.json" <<'PY'
import json
import sys

with open(sys.argv[1], encoding="utf-8") as handle:
    print(json.load(handle)["pid"])
PY
)"
[[ "$race_daemon_pid" =~ ^[1-9][0-9]*$ ]]
for _ in $(seq 1 100); do
  kill -0 "$race_daemon_pid" 2>/dev/null || break
  sleep 0.01
done
if kill -0 "$race_daemon_pid" 2>/dev/null; then
  echo "el daemon sobrevivió al fallo final del launcher microVM" >&2
  exit 1
fi
[ ! -e "$BASE/CodexA/run/server.pid" ]
[ ! -e "$BASE/CodexA/run/server.start_ref" ]
[ ! -e "$BASE/CodexA/run/server.binary_id" ]

# Un abuelo escribible invalida la cadena aunque root y perfil sigan en 0700.
chmod 722 "$TEST_ROOT"
expect_failure account_home_ancestor_insecure start_profile CodexA
chmod 700 "$TEST_ROOT"

printf 'PASS test_orquesta_profile_server\n'
