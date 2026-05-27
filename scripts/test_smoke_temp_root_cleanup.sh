#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck source=scripts/lib/smoke_common.sh
source "$ROOT/scripts/lib/smoke_common.sh"

valid_root="$(mktemp -d "${TMPDIR:-/tmp}/orquesta-smoke-cleanup-test-valid.XXXXXX")"
smoke_temp_root_prepare "$valid_root" "test"
touch "$valid_root/child.txt"
smoke_temp_root_cleanup "$valid_root" 0
if [[ -e "$valid_root" ]]; then
  echo "valid root was not removed" >&2
  exit 1
fi

unmarked_root="$(mktemp -d "${TMPDIR:-/tmp}/orquesta-smoke-cleanup-test-unmarked.XXXXXX")"
touch "$unmarked_root/child.txt"
smoke_temp_root_cleanup "$unmarked_root" 0
if [[ ! -d "$unmarked_root" ]]; then
  echo "unmarked root was removed" >&2
  exit 1
fi
{
  echo "schema_version=orquesta_smoke_temp_root.v0"
  echo "source=test-cleanup"
} >"$(smoke_temp_root_marker_path "$unmarked_root")"
smoke_temp_root_cleanup "$unmarked_root" 0

if smoke_temp_root_prepare "$ROOT" "test-forbidden-project" 2>/dev/null; then
  echo "project root was accepted as smoke temp root" >&2
  exit 1
fi

echo "smoke_temp_root_cleanup_ok=true"
