#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)"
query="${script_dir}/consultar_lecciones_legacy.sh"
index="${script_dir}/../product/knowledge/legacy_preflight_patterns.jsonl"

bash -n "$query"
jq -e -s '
  length == 13 and
  (map(.pattern_id) | unique | length) == 13 and
  all(.[];
    .schema_version == 1 and
    (.pattern_id | test("^LEGACY-[0-9]{3}$")) and
    (.title | length > 10) and
    (.capability_ids | length > 0) and
    (.path_prefixes | length > 0) and
    (.operations | length > 0) and
    (.symptoms | length > 0) and
    (.evidence_refs | length > 0) and
    (.current_state == "enforced" or .current_state == "partial")
  )
' "$index" >/dev/null

"$query" --capability ORC-03 --operation plan --json |
  jq -e 'length == 1 and .[0].pattern_id == "LEGACY-001"' >/dev/null
"$query" --path internal/config/manager.go --operation config --json |
  jq -e 'length == 1 and .[0].pattern_id == "LEGACY-007"' >/dev/null
"$query" --operation shutdown --json |
  jq -e 'length == 1 and .[0].pattern_id == "LEGACY-003"' >/dev/null

if "$query" --capability WIZ-99 >/dev/null 2>&1; then
  printf 'Una consulta sin coincidencias no debe devolver éxito.\n' >&2
  exit 1
fi

printf 'status=passed patterns=13 queries=4\n'
