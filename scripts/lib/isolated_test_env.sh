#!/usr/bin/env bash
# Perfil reutilizable con raíz privada y lease real del rango de puertos.

orquesta_isolated_test_preflight() {
  local tool
  for tool in python3 flock seq cksum awk; do
    command -v "$tool" >/dev/null 2>&1 || { echo "isolated_test_tool_missing:$tool" >&2; return 2; }
  done
  python3 - <<'PY'
import sys
if sys.version_info < (3, 10): raise SystemExit('python_3_10_or_newer_required')
PY
}

orquesta_private_test_root() {
  python3 - "$1" <<'PY'
import os, pathlib, stat, sys
path=pathlib.Path(sys.argv[1])
if not path.is_absolute() or path == pathlib.Path('/'):
    raise SystemExit('isolated_test_root_invalid')
current=pathlib.Path('/')
for part in path.parts[1:]:
    current /= part
    if os.path.lexists(current) and stat.S_ISLNK(os.lstat(current).st_mode):
        raise SystemExit('isolated_test_root_symlink')
existed=path.exists()
path.mkdir(mode=0o700, parents=True, exist_ok=True)
if not existed: os.chmod(path, 0o700)
current=pathlib.Path('/')
for part in path.parts[1:]:
    current /= part
    info=os.lstat(current)
    if stat.S_ISLNK(info.st_mode): raise SystemExit('isolated_test_root_symlink')
resolved=path.resolve(strict=True)
info=path.stat()
if resolved != path or not stat.S_ISDIR(info.st_mode) or info.st_uid != os.getuid() or stat.S_IMODE(info.st_mode) != 0o700:
    raise SystemExit('isolated_test_root_not_private_owned_canonical')
PY
}

orquesta_release_test_port_lease() {
  if [ -n "${ORQUESTA_TEST_PORT_LOCK_FD:-}" ]; then
    flock -u "$ORQUESTA_TEST_PORT_LOCK_FD" 2>/dev/null || true
    eval "exec ${ORQUESTA_TEST_PORT_LOCK_FD}>&-"
    unset ORQUESTA_TEST_PORT_LOCK_FD
  fi
}

orquesta_acquire_test_port_lease() {
  local root="$1" range="$2" preferred="$3" lease_root candidate offset lock fd
  lease_root="${ORQUESTA_TEST_PORT_LEASE_ROOT:-${ORQUESTA_TEST_CACHE_ROOT:-/srv/orquesta-self/runtime/test-cache}/port-leases}"
  orquesta_private_test_root "$lease_root" || return
  orquesta_release_test_port_lease
  for offset in $(seq 0 255); do
    candidate=$((preferred + offset * (range + 1)))
    [ "$candidate" -ge 1024 ] && [ $((candidate + range)) -le 65535 ] || continue
    lock="$lease_root/ports-${candidate}-$((candidate + range)).lock"
    python3 - "$lock" <<'PY'
import os, stat, sys
path=sys.argv[1]
fd=os.open(path, os.O_CREAT|os.O_RDWR|os.O_NOFOLLOW, 0o600)
info=os.fstat(fd)
if not stat.S_ISREG(info.st_mode) or info.st_uid != os.getuid() or stat.S_IMODE(info.st_mode) != 0o600:
    os.close(fd); raise SystemExit('port_lease_identity_invalid')
os.close(fd)
PY
    [ "$?" -eq 0 ] || return 1
    exec {fd}<>"$lock"
    if flock -n "$fd"; then
      export ORQUESTA_TEST_PORT_LOCK_FD="$fd"
      export ORQUESTA_TEST_PORT_BASE="$candidate"
      export ORQUESTA_TEST_PORT_RANGE="$range"
      export ORQUESTA_TEST_PORT_LOCK_DIR="$lease_root"
      export ORQUESTA_TEST_PORT_LEASE_REF="port-lease-${candidate}-$((candidate + range))"
      export ORQUESTA_TEST_SERVER_ADDR="127.0.0.1:$candidate"
      return 0
    fi
    eval "exec ${fd}>&-"
    [ -z "${ORQUESTA_TEST_PORT_BASE:-}" ] || break
  done
  echo "isolated_test_port_lease_unavailable" >&2
  return 1
}

orquesta_use_isolated_test_env() {
  local root="${1:-}" cache_base port_seed port_base range
  orquesta_isolated_test_preflight || return
  cache_base="${ORQUESTA_TEST_CACHE_ROOT:-/srv/orquesta-self/runtime/test-cache}"
  if [ -z "$root" ]; then root="$cache_base/isolated/${ORQUESTA_TEST_RUN_ID:-run-$(date +%s%N)-$$}"; fi
  umask 077
  orquesta_private_test_root "$root" || return
  mkdir -m 700 -p "$root/tmp" "$root/go-tmp" "$root/go-cache" "$root/go-mod-cache" \
    "$root/go-path" "$root/codex-home" "$root/xdg-runtime" "$root/flaky-cache" "$root/runtime"

  port_seed="$(printf '%s' "$root" | cksum | awk '{print $1}')"
  port_base="${ORQUESTA_TEST_PORT_BASE:-$((20000 + port_seed % 25000))}"
  range="${ORQUESTA_TEST_PORT_RANGE:-63}"
  case "$port_base:$range" in *[!0-9:]*) echo "isolated_test_port_config_invalid" >&2; return 2;; esac
  orquesta_acquire_test_port_lease "$root" "$range" "$port_base" || return

  export TMPDIR="$root/tmp"
  export GOTMPDIR="$root/go-tmp"
  export GOCACHE="$root/go-cache"
  export GOMODCACHE="${ORQUESTA_ISOLATED_TEST_GOMODCACHE:-$root/go-mod-cache}"
  export GOPATH="$root/go-path"
  export CODEX_HOME="$root/codex-home"
  export XDG_RUNTIME_DIR="$root/xdg-runtime"
  export ORQUESTA_FLAKY_HARNESS_CACHE_ROOT="$root/flaky-cache"
  export ORQUESTA_TEST_RUNTIME_ROOT="$root/runtime"
  export ORQUESTA_ISOLATED_TEST_ENV_REQUESTED_ROOT="$root"
  export ORQUESTA_ISOLATED_TEST_ENV_ROOT="$root"
}

orquesta_cleanup_isolated_test_env() {
  local root="${1:-${ORQUESTA_ISOLATED_TEST_ENV_ROOT:-}}"
  [ -n "$root" ] || return 0
  orquesta_private_test_root "$root" || return
  orquesta_release_test_port_lease
  rm -rf -- "$root"
}

if [ "${BASH_SOURCE[0]}" = "$0" ]; then
  orquesta_use_isolated_test_env "${1:-}"
  env | sed -n '/^TMPDIR=/p;/^GOTMPDIR=/p;/^GOCACHE=/p;/^GOMODCACHE=/p;/^GOPATH=/p;/^CODEX_HOME=/p;/^XDG_RUNTIME_DIR=/p;/^ORQUESTA_FLAKY_HARNESS_CACHE_ROOT=/p;/^ORQUESTA_TEST_RUNTIME_ROOT=/p;/^ORQUESTA_TEST_PORT_/p;/^ORQUESTA_ISOLATED_TEST_ENV_/p' | sort
fi
