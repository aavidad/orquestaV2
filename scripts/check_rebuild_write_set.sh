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
    .gitattributes|AGENTS.md|go.mod|go.sum|vendor/*|architecture_rebuild_test.go|credential_ref_compatibility_test.go|product_manifest_test.go|product_manifest_v17_test.go|product_roadmap_test.go|product_roadmap_v19_test.go|traceability_*_test.go|vendor_patch_test.go|internal/*|cmd/orquesta/*|acceptance/*|config/*|product/*|docs/inventario_bugs_orquesta_2026-06-30.md|docs/reconstruccion/*|scripts/check_rebuild_write_set.sh|scripts/smoke_v11_dex_samba_ad.sh|third_party/patches/README.md|third_party/patches/modernc_sqlite_v1.53.0_runtime_scope.patch|testdata/v11_dex_samba/Dockerfile.samba|testdata/v11_dex_samba/browser_flow.py|testdata/v11_dex_samba/probe/main.go|testdata/v11_dex_samba/samba-entrypoint.sh)
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
