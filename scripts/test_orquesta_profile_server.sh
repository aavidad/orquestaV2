#!/usr/bin/env bash
set -euo pipefail
umask 077

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
SCRIPT="$ROOT/scripts/orquesta_profile_server.sh"
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
GO_CACHE="$TEST_ROOT/go-cache"
REPOSITORY="$TEST_ROOT/repository"
FAKE_BWRAP_9000="$TEST_ROOT/fake-bwrap-9000"
FAKE_BWRAP_65536="$TEST_ROOT/fake-bwrap-65536"
FAKE_BWRAP_FAILED="$TEST_ROOT/fake-bwrap-failed"
FAKE_TOOLCHAIN="$TEST_ROOT/go-toolchain"
UNSAFE_TOOLCHAIN="$TEST_ROOT/go-toolchain-unsafe"
mkdir -m 700 "$BASE" "$ACCOUNTS" "$GO_CACHE"

cleanup() {
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
		"home":       os.Getenv("HOME"),
		"codex_home": os.Getenv("CODEX_HOME"),
		"auth_sha":   hex.EncodeToString(sum[:]),
		"environment": os.Environ(),
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

expect_failure() {
  expected="$1"
  shift
  set +e
  "$@" >"$TEST_ROOT/failure.out" 2>"$TEST_ROOT/failure.err"
  result=$?
  set -e
  [ "$result" -ne 0 ]
  grep -q "reason_code=$expected" "$TEST_ROOT/failure.err"
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
grep -Fq '[ "$(uname -s 2>/dev/null)" = "Linux" ] || fail "platform_unsupported"' \
  "$SCRIPT"

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

# Un abuelo escribible invalida la cadena aunque root y perfil sigan en 0700.
chmod 722 "$TEST_ROOT"
expect_failure account_home_ancestor_insecure start_profile CodexA
chmod 700 "$TEST_ROOT"

printf 'PASS test_orquesta_profile_server\n'
