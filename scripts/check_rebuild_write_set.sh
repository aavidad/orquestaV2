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
  if [[ "$base_ref" == "8063d8ce3ed75dd5ec92108ef6e7e73ee8e11cb8" ]]; then
    case "$path" in
      acceptance/fixtures/v21_i18n.json|acceptance/v21_i18n_*.go|acceptance/v21_simplicity_budget_test.go|cmd/orquesta/command.go|cmd/orquesta/main.go|cmd/orquesta/main_test.go|docs/public/en/README.md|docs/public/es/README.md|docs/inventario_bugs_orquesta_2026-06-30.md|docs/reconstruccion/analisis_y_contrato_v21_i18n_total.md|docs/reconstruccion/worksets/v21_i18n_red.json|internal/bootstrap/command_cli.go|internal/bootstrap/command_cli_test.go|internal/bootstrap/command_surfaces_e2e_test.go|internal/bootstrap/command_surfaces_i18n_e2e_test.go|internal/i18n/*|internal/i18n/catalogs/en.json|internal/i18n/catalogs/es.json|internal/interfaces/cli/runner.go|internal/interfaces/mcp/interface.go|internal/interfaces/mcp/v20_commands.go|internal/interfaces/mcp/v20_commands_test.go|internal/interfaces/mcp/v20_test_support_test.go|internal/interfaces/mcp/v21_i18n_test.go|product/evidence/real_codex_mcp_e2e.json|product/evidence/v21_i18n.json|product/evidence/v21_i18n.output.txt|product/roadmap.json|product/traceability/markdown_source_roles.jsonl|product/traceability/pending_sources.json|product/traceability/source_dispositions.jsonl|product_roadmap_v21_test.go|scripts/check_rebuild_write_set.sh|sdk/commands/client.go|sdk/commands/client_test.go|traceability_source_roles_test.go|vendor/golang.org/x/text/currency/*|vendor/golang.org/x/text/feature/plural/*|vendor/golang.org/x/text/internal/catmsg/*|vendor/golang.org/x/text/internal/format/*|vendor/golang.org/x/text/internal/internal.go|vendor/golang.org/x/text/internal/match.go|vendor/golang.org/x/text/internal/number/*|vendor/golang.org/x/text/internal/stringset/*|vendor/golang.org/x/text/message/*|vendor/golang.org/x/text/message/catalog/*|vendor/golang.org/x/text/number/*|vendor/modules.txt)
        ;;
      *)
        invalid+=("$path")
        ;;
    esac
  else
    case "$path" in
      .gitattributes|AGENTS.md|go.mod|go.sum|vendor/*|architecture_rebuild_test.go|credential_ref_compatibility_test.go|product_manifest_test.go|product_manifest_v17_test.go|product_roadmap_test.go|product_roadmap_v19_test.go|product_roadmap_v20_test.go|traceability_*_test.go|vendor_patch_test.go|internal/*|cmd/orquesta/*|sdk/commands/*|acceptance/*|config/*|product/*|docs/inventario_bugs_orquesta_2026-06-30.md|docs/reconstruccion/*|scripts/check_rebuild_write_set.sh|scripts/smoke_v11_dex_samba_ad.sh|third_party/patches/README.md|third_party/patches/modernc_sqlite_v1.53.0_runtime_scope.patch|testdata/v11_dex_samba/Dockerfile.samba|testdata/v11_dex_samba/browser_flow.py|testdata/v11_dex_samba/probe/main.go|testdata/v11_dex_samba/samba-entrypoint.sh)
        ;;
      *)
        invalid+=("$path")
        ;;
    esac
  fi
done

if ((${#invalid[@]} > 0)); then
  printf 'rebuild_write_set_violation\n' >&2
  printf '  %s\n' "${invalid[@]}" >&2
  exit 1
fi

printf 'rebuild_write_set_ok base=%s files=%d\n' "$base_ref" "${#changed[@]}"
