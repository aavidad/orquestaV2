#!/usr/bin/env bash
set -euo pipefail

base_ref="${1:-7576f60bd3b5424a6c1c19e4f319cb634b12f231}"

mapfile -t changed < <(
  {
    git diff --name-only "$base_ref" --
    git ls-files --others --exclude-standard
  } | LC_ALL=C sort -u
)

invalid=()
for path in "${changed[@]}"; do
  case "$path" in
    go.mod|go.sum|vendor/*|architecture_rebuild_test.go|product_manifest_test.go|internal/*|cmd/orquesta/*|config/*|product/*|docs/reconstruccion/*|scripts/check_rebuild_write_set.sh)
      ;;
    *)
      invalid+=("$path")
      ;;
  esac
done

if ((${#invalid[@]} > 0)); then
  printf 'rebuild_write_set_violation\n' >&2
  printf '  %s\n' "${invalid[@]}" >&2
  exit 1
fi

printf 'rebuild_write_set_ok base=%s files=%d\n' "$base_ref" "${#changed[@]}"
