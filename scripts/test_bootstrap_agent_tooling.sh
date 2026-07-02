#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

FAKE_BIN="$TMP/codebase-memory-mcp"
FAKE_PS="$TMP/ps"
CALLS="$TMP/calls.log"
cat >"$FAKE_BIN" <<'FAKE'
#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "$*" >>"${CODEBASE_MEMORY_MCP_FAKE_CALLS:?}"
case "${1:-}" in
  --version)
    printf 'codebase-memory-mcp fake-0.0.1\n'
    ;;
  update|install)
    exit 0
    ;;
  cli)
    printf '{"ok":true}\n'
    ;;
  *)
    exit 0
    ;;
esac
FAKE
chmod +x "$FAKE_BIN"
cat >"$FAKE_PS" <<'FAKE_PS'
#!/usr/bin/env bash
set -euo pipefail
if [ "${CODEBASE_MEMORY_MCP_FAKE_PS_LIVE:-0}" = "1" ]; then
  printf '/home/alberto/.local/bin/codebase-memory-mcp --stdio\n'
fi
FAKE_PS
chmod +x "$FAKE_PS"

run_bootstrap() {
  CODEX_HOME="$TMP/codex-home" \
  CODEBASE_MEMORY_MCP_BIN="$FAKE_BIN" \
  CODEBASE_MEMORY_MCP_FAKE_CALLS="$CALLS" \
  PATH="$TMP:$PATH" \
  "$ROOT/scripts/bootstrap_agent_tooling.sh" --repo "$ROOT" "$@"
}

assert_no_install_by_default() {
  : >"$CALLS"
  rm -rf "$TMP/codex-home"
  run_bootstrap --enable-codebase --no-update >"$TMP/bootstrap-default.out"
  if grep -qx 'install -y --ui=false' "$CALLS"; then
    printf 'bootstrap_test_failed: install_direct_mcp_default_on\n' >&2
    exit 1
  fi
  grep -q '^direct_mcp=false$' "$TMP/codex-home/log/orquesta-agent-tooling.env"
}

assert_direct_install_requires_opt_in() {
  : >"$CALLS"
  rm -rf "$TMP/codex-home"
  run_bootstrap --enable-codebase --install-direct-mcp --no-update >"$TMP/bootstrap-direct.out"
  grep -qx 'install -y --ui=false' "$CALLS"
  grep -q '^direct_mcp=true$' "$TMP/codex-home/log/orquesta-agent-tooling.env"
}

assert_status_reports_direct_config_attention() {
  mkdir -p "$TMP/codex-home"
  cat >"$TMP/codex-home/config.toml" <<'TOML'
[mcp_servers.codebase-memory-mcp]
command = "codebase-memory-mcp"
TOML
  cat >"$TMP/codex-home/log-orquesta-agent-tooling.env" <<'EOF_LEDGER'
enabled=1
direct_mcp=false
EOF_LEDGER
  mkdir -p "$TMP/codex-home/log"
  mv "$TMP/codex-home/log-orquesta-agent-tooling.env" "$TMP/codex-home/log/orquesta-agent-tooling.env"
  status="$(run_bootstrap --status)"
  [[ "$status" == *"estado=attention_required"* ]] || {
    printf 'bootstrap_test_failed: status_estado_unexpected:%s\n' "$status" >&2
    exit 1
  }
  [[ "$status" == *"config_codebase_memory_mcp=1"* ]] || {
    printf 'bootstrap_test_failed: status_config_unexpected:%s\n' "$status" >&2
    exit 1
  }
  [[ "$status" == *"direct_mcp=false"* ]] || {
    printf 'bootstrap_test_failed: status_direct_mcp_unexpected:%s\n' "$status" >&2
    exit 1
  }
  [[ "$status" == *"next_action=remove_direct_mcp_config_or_enable_explicit_opt_in"* ]] || {
    printf 'bootstrap_test_failed: status_next_action_unexpected:%s\n' "$status" >&2
    exit 1
  }
}

assert_status_reports_live_process_action() {
  rm -rf "$TMP/codex-home"
  mkdir -p "$TMP/codex-home/log"
  cat >"$TMP/codex-home/log/orquesta-agent-tooling.env" <<'EOF_LEDGER'
enabled=1
direct_mcp=false
EOF_LEDGER
  status="$(CODEBASE_MEMORY_MCP_FAKE_PS_LIVE=1 run_bootstrap --status)"
  [[ "$status" == *"estado=attention_required"* ]] || {
    printf 'bootstrap_test_failed: live_status_estado_unexpected:%s\n' "$status" >&2
    exit 1
  }
  [[ "$status" == *"live_codebase_memory_mcp_processes=1"* ]] || {
    printf 'bootstrap_test_failed: live_status_count_unexpected:%s\n' "$status" >&2
    exit 1
  }
  [[ "$status" == *"next_action=stop_orphan_codebase_memory_mcp_processes"* ]] || {
    printf 'bootstrap_test_failed: live_status_next_action_unexpected:%s\n' "$status" >&2
    exit 1
  }
}

assert_no_install_by_default
assert_direct_install_requires_opt_in
assert_status_reports_direct_config_attention
assert_status_reports_live_process_action

printf 'bootstrap_agent_tooling_test_ok\n'
