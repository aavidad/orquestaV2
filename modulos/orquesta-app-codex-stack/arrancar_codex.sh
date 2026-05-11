#!/usr/bin/env bash
set -euo pipefail

die() {
  printf 'orquesta-app-codex-stack: %s\n' "$*" >&2
  exit 2
}

[ "${ORQUESTA_CODEX_STACK_OPT_IN:-}" = "1" ] || die "set ORQUESTA_CODEX_STACK_OPT_IN=1 to run the external Codex stack"
[ -n "${ORQUESTA_CODEX_STACK_COMMAND:-}" ] || die "set ORQUESTA_CODEX_STACK_COMMAND with the explicit smoke/server command"
[ -n "${ORQUESTA_CODEX_COMMAND:-}" ] || die "set ORQUESTA_CODEX_COMMAND explicitly"
[ -n "${ORQUESTA_CODEX_HOME:-}" ] || die "set ORQUESTA_CODEX_HOME explicitly"
[ -n "${ORQUESTA_CODEX_CODE_HOME:-}" ] || die "set ORQUESTA_CODEX_CODE_HOME explicitly"
[ -n "${ORQUESTA_CODEX_PATH:-}" ] || die "set ORQUESTA_CODEX_PATH explicitly"
[ -n "${ORQUESTA_CODEX_APPROVAL_POLICY:-}" ] || die "set ORQUESTA_CODEX_APPROVAL_POLICY explicitly"
[ -n "${ORQUESTA_CODEX_SANDBOX:-}" ] || die "set ORQUESTA_CODEX_SANDBOX explicitly"

case "${ORQUESTA_CODEX_STACK_DB_DSN:-}" in
  "")
    ;;
  default|sqlite|postgres|memory)
    die "ORQUESTA_CODEX_STACK_DB_DSN must be an operator-owned DSN/ref, not a symbolic default"
    ;;
esac

case "${ORQUESTA_CODEX_MODEL:-}" in
  "")
    ;;
  default)
    die "ORQUESTA_CODEX_MODEL must be explicit when used"
    ;;
esac

export PATH="$ORQUESTA_CODEX_PATH"
exec "${SHELL:-/bin/sh}" -lc "$ORQUESTA_CODEX_STACK_COMMAND"
