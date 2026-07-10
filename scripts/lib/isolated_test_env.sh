#!/usr/bin/env bash
# Perfil reutilizable para tests/smokes aislados de Orquesta.

orquesta_use_isolated_test_env() {
  local root="${1:-}"
  local requested_root=""
  if [ -z "$root" ]; then
    root="$(mktemp -d "${TMPDIR:-/tmp}/orquesta-test-env.XXXXXX")"
  fi
  requested_root="$root"
  if ! mkdir -p "$root" 2>/dev/null; then
    local fallback_base="${ORQUESTA_ISOLATED_TEST_ENV_FALLBACK_ROOT:-${PWD:-/tmp}/.orquesta-runtime/isolated-test-env}"
    local root_key
    root_key="$(printf '%s' "$requested_root" | sed 's#[^A-Za-z0-9_.-]#_#g')"
    root="$fallback_base/$root_key"
  fi
  mkdir -p \
    "$root/tmp" \
    "$root/go-tmp" \
    "$root/go-cache" \
    "$root/go-mod-cache" \
    "$root/go-path" \
    "$root/codex-home" \
    "$root/xdg-runtime" \
    "$root/flaky-cache" \
    "$root/ports" \
    "$root/runtime"

  local seed_gomodcache="${ORQUESTA_ISOLATED_TEST_ENV_SEED_GOMODCACHE:-/srv/orquesta-self/runtime/go-mod-cache}"
  if [ -d "$seed_gomodcache" ] && [ ! -d "$root/go-mod-cache/golang.org/x/text@v0.38.0" ]; then
    cp -a "$seed_gomodcache/." "$root/go-mod-cache/" 2>/dev/null || true
    chmod -R u+w "$root/go-mod-cache" 2>/dev/null || true
  fi

  export TMPDIR="$root/tmp"
  export GOTMPDIR="$root/go-tmp"
  export GOCACHE="$root/go-cache"
  export GOMODCACHE="$root/go-mod-cache"
  export GOPATH="$root/go-path"
  export CODEX_HOME="$root/codex-home"
  export XDG_RUNTIME_DIR="$root/xdg-runtime"
  export ORQUESTA_FLAKY_HARNESS_CACHE_ROOT="$root/flaky-cache"
  export ORQUESTA_TEST_RUNTIME_ROOT="$root/runtime"
  export ORQUESTA_TEST_PORT_BASE="${ORQUESTA_TEST_PORT_BASE:-39100}"
  export ORQUESTA_TEST_PORT_RANGE="${ORQUESTA_TEST_PORT_RANGE:-99}"
  export ORQUESTA_TEST_PORT_LOCK_DIR="$root/ports"
  export ORQUESTA_ISOLATED_TEST_ENV_REQUESTED_ROOT="$requested_root"
  export ORQUESTA_ISOLATED_TEST_ENV_ROOT="$root"
}

if [ "${BASH_SOURCE[0]}" = "$0" ]; then
  orquesta_use_isolated_test_env "${1:-}"
  printf 'TMPDIR=%s\n' "$TMPDIR"
  printf 'GOTMPDIR=%s\n' "$GOTMPDIR"
  printf 'GOCACHE=%s\n' "$GOCACHE"
  printf 'GOMODCACHE=%s\n' "$GOMODCACHE"
  printf 'GOPATH=%s\n' "$GOPATH"
  printf 'CODEX_HOME=%s\n' "$CODEX_HOME"
  printf 'XDG_RUNTIME_DIR=%s\n' "$XDG_RUNTIME_DIR"
  printf 'ORQUESTA_FLAKY_HARNESS_CACHE_ROOT=%s\n' "$ORQUESTA_FLAKY_HARNESS_CACHE_ROOT"
  printf 'ORQUESTA_TEST_RUNTIME_ROOT=%s\n' "$ORQUESTA_TEST_RUNTIME_ROOT"
  printf 'ORQUESTA_TEST_PORT_BASE=%s\n' "$ORQUESTA_TEST_PORT_BASE"
  printf 'ORQUESTA_TEST_PORT_RANGE=%s\n' "$ORQUESTA_TEST_PORT_RANGE"
  printf 'ORQUESTA_TEST_PORT_LOCK_DIR=%s\n' "$ORQUESTA_TEST_PORT_LOCK_DIR"
  printf 'ORQUESTA_ISOLATED_TEST_ENV_REQUESTED_ROOT=%s\n' "$ORQUESTA_ISOLATED_TEST_ENV_REQUESTED_ROOT"
  printf 'ORQUESTA_ISOLATED_TEST_ENV_ROOT=%s\n' "$ORQUESTA_ISOLATED_TEST_ENV_ROOT"
fi
