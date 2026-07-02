#!/usr/bin/env bash
set -euo pipefail

repo_root="$(pwd)"
index_repo=0
dry_run=0
update_tool=1
enable_codebase=0
install_direct_mcp=0
status_only=0

usage() {
  cat <<'USAGE'
Uso: scripts/bootstrap_agent_tooling.sh [--repo PATH] [--enable-codebase] [--index] [--install-direct-mcp] [--status] [--no-update] [--dry-run]

Prepara herramientas canonicas para agentes Orquesta en el usuario actual:
- deja codebase-memory-mcp como herramienta opt-in si se pide explicitamente;
- no instala MCP directo en CODEX_HOME salvo --install-direct-mcp;
- deja norma persistente en ~/.codex/AGENTS.md;
- --status informa configuracion/procesos sin parar nada;
- opcionalmente indexa el worktree aislado;
- no abre UI ni puertos externos.

codebase-memory-mcp solo es obligatorio con --enable-codebase o --index.
USAGE
}

fail() {
  printf 'agent_tooling_bootstrap_failed: %s\n' "$1" >&2
  exit 2
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --repo)
      [ "$#" -ge 2 ] || fail "repo_sin_valor"
      repo_root="$2"
      shift 2
      ;;
    --index)
      index_repo=1
      enable_codebase=1
      shift
      ;;
    --enable-codebase)
      enable_codebase=1
      shift
      ;;
    --install-direct-mcp)
      enable_codebase=1
      install_direct_mcp=1
      shift
      ;;
    --status)
      status_only=1
      shift
      ;;
    --no-update)
      update_tool=0
      shift
      ;;
    --dry-run)
      dry_run=1
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      fail "opcion_incompatible:$1"
      ;;
  esac
done

repo_root="$(cd "$repo_root" && pwd)"
codex_home="${CODEX_HOME:-$HOME/.codex}"
agents_file="$codex_home/AGENTS.md"
ledger_file="$codex_home/log/orquesta-agent-tooling.env"
config_file="$codex_home/config.toml"
tool_bin="${CODEBASE_MEMORY_MCP_BIN:-}"
if [ -z "$tool_bin" ]; then
  tool_bin="$(command -v codebase-memory-mcp || true)"
fi

ledger_value() {
  key="$1"
  file="$2"
  [ -f "$file" ] || return 1
  awk -F= -v key="$key" '$1 == key {print substr($0, length($1) + 2); found=1; exit} END {if (!found) exit 1}' "$file"
}

codebase_config_present() {
  file="$1"
  [ -f "$file" ] || {
    printf '0'
    return 0
  }
  awk '
    /^[[:space:]]*#/ {next}
    /^[[:space:]]*\[mcp_servers\.codebase-memory-mcp\]/ {found=1}
    /codebase-memory-mcp/ {found=1}
    END {print found ? "1" : "0"}
  ' "$file"
}

codebase_process_count() {
  if ! command -v ps >/dev/null 2>&1 || ! command -v awk >/dev/null 2>&1; then
    printf 'unknown'
    return 0
  fi
  ps -eo args= | awk '
    {
      command=$0
      n=split(command, fields, /[[:space:]]+/)
      if (n < 1) next
      first=fields[1]
      sub(/^.*\//, "", first)
      if (first == "codebase-memory-mcp" || command ~ /\/codebase-memory-mcp([[:space:]]|$)/) count++
    }
    END {print count + 0}
  '
}

print_status() {
  ledger_enabled="$(ledger_value enabled "$ledger_file" 2>/dev/null || printf 'unknown')"
  direct_mcp="$(ledger_value direct_mcp "$ledger_file" 2>/dev/null || printf 'unknown')"
  broker_state_dir="$(ledger_value broker_state_dir "$ledger_file" 2>/dev/null || printf '%s' "${ORQUESTA_CODEBASE_BROKER_STATE_DIR:-}")"
  config_has_mcp="$(codebase_config_present "$config_file")"
  live_processes="$(codebase_process_count)"
  estado="ok"
  next_action="none"
  if [ "$config_has_mcp" = "1" ] && [ "$direct_mcp" != "true" ]; then
    estado="attention_required"
    next_action="remove_direct_mcp_config_or_enable_explicit_opt_in"
  fi
  if [ "$live_processes" != "0" ] && [ "$live_processes" != "unknown" ]; then
    estado="attention_required"
    next_action="stop_orphan_codebase_memory_mcp_processes"
  fi
  printf 'agent_tooling_status: estado=%s repo=%s codex_home=%s tool=%s ledger_enabled=%s direct_mcp=%s broker_state_dir=%s config_codebase_memory_mcp=%s live_codebase_memory_mcp_processes=%s next_action=%s policy=broker_only\n' \
    "$estado" "$repo_root" "$codex_home" "$tool_bin" "$ledger_enabled" "$direct_mcp" "$broker_state_dir" "$config_has_mcp" "$live_processes" "$next_action"
}

if [ "$status_only" -eq 1 ]; then
  print_status
  exit 0
fi

if [ "$enable_codebase" -eq 1 ]; then
  [ -n "$tool_bin" ] || fail "codebase-memory-mcp_no_encontrado: instala el binario en PATH o define CODEBASE_MEMORY_MCP_BIN"
  [ -x "$tool_bin" ] || fail "codebase-memory-mcp_no_ejecutable:$tool_bin"
fi

mkdir -p "$codex_home"

if [ "$dry_run" -eq 1 ]; then
  printf 'agent_tooling_bootstrap_dry_run: repo=%s tool=%s codex_home=%s codebase=%s index=%s update=%s direct_mcp=%s\n' \
    "$repo_root" "$tool_bin" "$codex_home" "$enable_codebase" "$index_repo" "$update_tool" "$install_direct_mcp"
  exit 0
fi

if [ "$enable_codebase" -eq 1 ] && [ "$update_tool" -eq 1 ]; then
  "$tool_bin" update -y >/dev/null 2>&1 || {
    printf 'agent_tooling_update_warning: codebase-memory-mcp_update_failed; continuing_with_installed_version\n' >&2
  }
fi

if [ "$enable_codebase" -eq 1 ] && [ "$install_direct_mcp" -eq 1 ]; then
  "$tool_bin" install -y --ui=false >/dev/null
fi

if ! grep -q 'orquesta-agent-tooling:start' "$agents_file" 2>/dev/null; then
  cat >>"$agents_file" <<'EOF'

<!-- orquesta-agent-tooling:start -->
# Orquesta Agent Tooling

When working in Orquesta repositories:
- use codebase-memory-mcp only when it clearly helps with symbol relationships,
  callers/callees, hotspots or architecture summaries;
- use grep/rg for exact strings, Markdown, configs and incident documents;
- do not use codebase-memory-mcp by default in subagents or broad parallel work;
- do not index repositories or leave multiple codebase-memory-mcp instances
  running without explicit operator approval;
- use compact communication; prefer caveman-style summaries if the skill exists;
- do not load whole courses/temarios as context; use domain RAG or compact refs.
<!-- orquesta-agent-tooling:end -->
EOF
fi

if ! grep -q 'orquesta-agent-tooling-codebase-broker:start' "$agents_file" 2>/dev/null; then
  cat >>"$agents_file" <<'EOF'

<!-- orquesta-agent-tooling-codebase-broker:start -->
# Orquesta Codebase Broker Guard

For Orquesta repositories:
- do not configure or launch codebase-memory-mcp directly from subagents;
- use Orquesta broker endpoints/tools for code context by default;
- direct MCP install requires explicit operator opt-in and --install-direct-mcp;
- run scripts/bootstrap_agent_tooling.sh --status before long sessions when
  investigating duplicate codebase-memory-mcp processes;
- if --status reports live_codebase_memory_mcp_processes above zero without an
  active Orquesta broker lease, stop the orphan processes cooperatively or start
  the configured Orquesta watchdog instead of leaving them attached to sessions.
<!-- orquesta-agent-tooling-codebase-broker:end -->
EOF
fi

version=""
if [ "$enable_codebase" -eq 1 ]; then
  version="$("$tool_bin" --version 2>/dev/null || true)"
fi
mkdir -p "$codex_home/log"
direct_mcp="false"
if [ "$install_direct_mcp" -eq 1 ]; then
  direct_mcp="true"
fi
printf 'tool=codebase-memory-mcp\nenabled=%s\nversion=%s\nrepo=%s\nui=false\ndirect_mcp=%s\nbroker_policy=central_only\nbroker_state_dir=%s\n' \
  "$enable_codebase" "$version" "$repo_root" "$direct_mcp" "${ORQUESTA_CODEBASE_BROKER_STATE_DIR:-}" >"$ledger_file"

if [ "$index_repo" -eq 1 ]; then
  escaped_repo="${repo_root//\\/\\\\}"
  escaped_repo="${escaped_repo//\"/\\\"}"
  "$tool_bin" cli index_repository "{\"repo_path\":\"$escaped_repo\",\"mode\":\"fast\",\"persistence\":false}"
fi

printf 'agent_tooling_bootstrap_ok: tool=%s enabled=%s version=%s repo=%s index=%s direct_mcp=%s\n' \
  "$tool_bin" "$enable_codebase" "$version" "$repo_root" "$index_repo" "$direct_mcp"
