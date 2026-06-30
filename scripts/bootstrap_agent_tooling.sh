#!/usr/bin/env bash
set -euo pipefail

repo_root="$(pwd)"
index_repo=0
dry_run=0
update_tool=1
enable_codebase=0

usage() {
  cat <<'USAGE'
Uso: scripts/bootstrap_agent_tooling.sh [--repo PATH] [--enable-codebase] [--index] [--no-update] [--dry-run]

Prepara herramientas canonicas para agentes Orquesta en el usuario actual:
- deja codebase-memory-mcp como herramienta opt-in si se pide explicitamente;
- deja norma persistente en ~/.codex/AGENTS.md;
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
tool_bin="${CODEBASE_MEMORY_MCP_BIN:-}"
if [ -z "$tool_bin" ]; then
  tool_bin="$(command -v codebase-memory-mcp || true)"
fi
if [ "$enable_codebase" -eq 1 ]; then
  [ -n "$tool_bin" ] || fail "codebase-memory-mcp_no_encontrado: instala el binario en PATH o define CODEBASE_MEMORY_MCP_BIN"
  [ -x "$tool_bin" ] || fail "codebase-memory-mcp_no_ejecutable:$tool_bin"
fi

codex_home="${CODEX_HOME:-$HOME/.codex}"
agents_file="$codex_home/AGENTS.md"
mkdir -p "$codex_home"

if [ "$dry_run" -eq 1 ]; then
  printf 'agent_tooling_bootstrap_dry_run: repo=%s tool=%s codex_home=%s codebase=%s index=%s update=%s\n' \
    "$repo_root" "$tool_bin" "$codex_home" "$enable_codebase" "$index_repo" "$update_tool"
  exit 0
fi

if [ "$enable_codebase" -eq 1 ] && [ "$update_tool" -eq 1 ]; then
  "$tool_bin" update -y >/dev/null 2>&1 || {
    printf 'agent_tooling_update_warning: codebase-memory-mcp_update_failed; continuing_with_installed_version\n' >&2
  }
fi

if [ "$enable_codebase" -eq 1 ]; then
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

version=""
if [ "$enable_codebase" -eq 1 ]; then
  version="$("$tool_bin" --version 2>/dev/null || true)"
fi
mkdir -p "$codex_home/log"
printf 'tool=codebase-memory-mcp\nenabled=%s\nversion=%s\nrepo=%s\nui=false\n' \
  "$enable_codebase" "$version" "$repo_root" >"$codex_home/log/orquesta-agent-tooling.env"

if [ "$index_repo" -eq 1 ]; then
  escaped_repo="${repo_root//\\/\\\\}"
  escaped_repo="${escaped_repo//\"/\\\"}"
  "$tool_bin" cli index_repository "{\"repo_path\":\"$escaped_repo\",\"mode\":\"fast\",\"persistence\":false}"
fi

printf 'agent_tooling_bootstrap_ok: tool=%s enabled=%s version=%s repo=%s index=%s\n' \
  "$tool_bin" "$enable_codebase" "$version" "$repo_root" "$index_repo"
