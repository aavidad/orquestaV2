#!/usr/bin/env bash
set -euo pipefail

repo_root="$(pwd)"
index_repo=0
dry_run=0
update_tool=1

usage() {
  cat <<'USAGE'
Uso: scripts/bootstrap_agent_tooling.sh [--repo PATH] [--index] [--no-update] [--dry-run]

Prepara herramientas canonicas para agentes Orquesta en el usuario actual:
- instala/configura codebase-memory-mcp para Codex;
- deja norma persistente en ~/.codex/AGENTS.md;
- opcionalmente indexa el worktree aislado;
- no abre UI ni puertos externos.

Requiere que codebase-memory-mcp exista en PATH o CODEBASE_MEMORY_MCP_BIN.
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
[ -n "$tool_bin" ] || fail "codebase-memory-mcp_no_encontrado: instala el binario en PATH o define CODEBASE_MEMORY_MCP_BIN"
[ -x "$tool_bin" ] || fail "codebase-memory-mcp_no_ejecutable:$tool_bin"

codex_home="${CODEX_HOME:-$HOME/.codex}"
agents_file="$codex_home/AGENTS.md"
mkdir -p "$codex_home"

if [ "$dry_run" -eq 1 ]; then
  printf 'agent_tooling_bootstrap_dry_run: repo=%s tool=%s codex_home=%s index=%s update=%s\n' \
    "$repo_root" "$tool_bin" "$codex_home" "$index_repo" "$update_tool"
  exit 0
fi

if [ "$update_tool" -eq 1 ]; then
  "$tool_bin" update -y >/dev/null 2>&1 || {
    printf 'agent_tooling_update_warning: codebase-memory-mcp_update_failed; continuing_with_installed_version\n' >&2
  }
fi

"$tool_bin" install -y --ui=false >/dev/null

if ! grep -q 'orquesta-agent-tooling:start' "$agents_file" 2>/dev/null; then
  cat >>"$agents_file" <<'EOF'

<!-- orquesta-agent-tooling:start -->
# Orquesta Agent Tooling

When working in Orquesta repositories:
- use codebase-memory-mcp first for code discovery and architecture relations;
- use grep/rg for exact strings, Markdown, configs and incident documents;
- use compact communication; prefer caveman-style summaries if the skill exists;
- do not load whole courses/temarios as context; use domain RAG or compact refs.
<!-- orquesta-agent-tooling:end -->
EOF
fi

version="$("$tool_bin" --version 2>/dev/null || true)"
mkdir -p "$codex_home/log"
printf 'tool=codebase-memory-mcp\nversion=%s\nrepo=%s\nui=false\n' \
  "$version" "$repo_root" >"$codex_home/log/orquesta-agent-tooling.env"

if [ "$index_repo" -eq 1 ]; then
  escaped_repo="${repo_root//\\/\\\\}"
  escaped_repo="${escaped_repo//\"/\\\"}"
  "$tool_bin" cli index_repository "{\"repo_path\":\"$escaped_repo\",\"mode\":\"fast\",\"persistence\":false}"
fi

printf 'agent_tooling_bootstrap_ok: tool=%s version=%s repo=%s index=%s\n' \
  "$tool_bin" "$version" "$repo_root" "$index_repo"
