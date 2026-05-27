#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck source=scripts/lib/go_tool.sh
source "$ROOT/scripts/lib/go_tool.sh"
# shellcheck source=scripts/lib/smoke_common.sh
source "$ROOT/scripts/lib/smoke_common.sh"

workdir="$(mktemp -d)"
smoke_temp_root_prepare "$workdir" "generated"
trap 'smoke_temp_root_cleanup "$workdir" 0' EXIT

mkdir -p "$workdir/usr-local-go/bin"
cat >"$workdir/usr-local-go/bin/go" <<'FAKEGO'
#!/bin/sh
echo "fake go $*"
FAKEGO
chmod +x "$workdir/usr-local-go/bin/go"

mkdir -p "$workdir/empty-bin"
PATH="$workdir/empty-bin"
ORQUESTA_GO_TOOL_CANDIDATES="$workdir/usr-local-go/bin/go"
orquesta_go_tool_ensure_path

resolved="$(command -v go)"
if [[ "$resolved" != "$workdir/usr-local-go/bin/go" ]]; then
  echo "expected go from fallback candidate, got: ${resolved:-<missing>}" >&2
  exit 1
fi

output="$(go version)"
if [[ "$output" != "fake go version" ]]; then
  echo "unexpected go output: $output" >&2
  exit 1
fi
