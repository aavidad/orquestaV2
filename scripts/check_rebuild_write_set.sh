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
      acceptance/fixtures/v21_i18n.json|acceptance/v21_i18n_*.go|acceptance/v21_simplicity_budget_test.go|cmd/orquesta/command.go|cmd/orquesta/main.go|cmd/orquesta/main_test.go|docs/public/en/README.md|docs/public/es/README.md|docs/inventario_bugs_orquesta_2026-06-30.md|docs/reconstruccion/analisis_y_contrato_v21_i18n_total.md|docs/reconstruccion/worksets/v21_i18n_red.json|internal/bootstrap/command_cli.go|internal/bootstrap/command_cli_test.go|internal/bootstrap/command_surfaces_e2e_test.go|internal/bootstrap/command_surfaces_i18n_e2e_test.go|internal/i18n/*|internal/i18n/catalogs/en.json|internal/i18n/catalogs/es.json|internal/interfaces/cli/runner.go|internal/interfaces/mcp/interface.go|internal/interfaces/mcp/v20_commands.go|internal/interfaces/mcp/v20_commands_test.go|internal/interfaces/mcp/v20_test_support_test.go|internal/interfaces/mcp/v21_i18n_test.go|product/evidence/real_codex_mcp_e2e.json|product/evidence/v21_i18n.json|product/evidence/v21_i18n.output.txt|product/roadmap.json|product/traceability/markdown_source_roles.jsonl|product/traceability/pending_sources.json|product/traceability/source_dispositions.jsonl|product_roadmap_v20_test.go|product_roadmap_v21_test.go|scripts/check_rebuild_write_set.sh|sdk/commands/client.go|sdk/commands/client_test.go|traceability_source_roles_test.go|vendor/golang.org/x/text/currency/*|vendor/golang.org/x/text/feature/plural/*|vendor/golang.org/x/text/internal/catmsg/*|vendor/golang.org/x/text/internal/format/*|vendor/golang.org/x/text/internal/internal.go|vendor/golang.org/x/text/internal/match.go|vendor/golang.org/x/text/internal/number/*|vendor/golang.org/x/text/internal/stringset/*|vendor/golang.org/x/text/message/*|vendor/golang.org/x/text/message/catalog/*|vendor/golang.org/x/text/number/*|vendor/modules.txt)
        ;;
      *)
        invalid+=("$path")
        ;;
    esac
  elif [[ "$base_ref" == "665ef446a32a7f0512255640fa99b2e7edd9f29d" ]]; then
    case "$path" in
      acceptance/fixtures/v22_codex_e2e.json|acceptance/v20_simplicity_budget_test.go|acceptance/v21_i18n_test.go|acceptance/v21_simplicity_budget_test.go|acceptance/v22_codex_e2e_support_test.go|acceptance/v22_codex_e2e_test.go|acceptance/v22_codex_real_e2e_test.go|acceptance/v22_codex_real_process_linux_test.go|acceptance/v22_simplicity_budget_test.go|\
      config/generated/orquesta.schema.json|config/generated/reference.md|config/generated/ui.json|config/orquesta.toml.example|config/registry.json|\
      docs/inventario_bugs_orquesta_2026-06-30.md|docs/reconstruccion/analisis_y_contrato_v22_codex_e2e.md|docs/reconstruccion/handoff_v22_sesion_2026-07-23.md|docs/reconstruccion/manual_revision_claude_v22.md|\
      internal/adapters/agent/codex/adapter.go|internal/adapters/agent/codex/adapter_integration_test.go|internal/adapters/agent/codex/config.go|internal/adapters/agent/codex/control_linux_test.go|internal/adapters/agent/codex/credential.go|internal/adapters/agent/codex/persistence.go|internal/adapters/agent/codex/process.go|internal/adapters/agent/codex/prompt.go|internal/adapters/agent/codex/prompt_test.go|internal/adapters/agent/codex/session.go|internal/adapters/agent/codex/session_test.go|\
      internal/adapters/auth/executiontoken/broker.go|internal/adapters/auth/executiontoken/errors.go|internal/adapters/auth/executiontoken/provider.go|internal/adapters/auth/executiontoken/provider_test.go|internal/adapters/auth/executiontoken/routed_provider.go|internal/adapters/auth/executiontoken/routed_provider_test.go|\
      internal/adapters/state/sqlite/app_specs_test.go|internal/adapters/state/sqlite/claim.go|internal/adapters/state/sqlite/command_audit_test.go|internal/adapters/state/sqlite/director_test.go|internal/adapters/state/sqlite/execution_session.go|internal/adapters/state/sqlite/execution_session_test.go|internal/adapters/state/sqlite/identity_access.go|internal/adapters/state/sqlite/identity_runtime_test.go|internal/adapters/state/sqlite/mailbox.go|internal/adapters/state/sqlite/mailbox_delivery.go|internal/adapters/state/sqlite/mailbox_resolution.go|internal/adapters/state/sqlite/mailbox_schema_test.go|internal/adapters/state/sqlite/mailbox_test.go|internal/adapters/state/sqlite/migrations/016_post_artifact_mailbox.sql|internal/adapters/state/sqlite/mutations.go|internal/adapters/state/sqlite/recovery_v09_test.go|internal/adapters/state/sqlite/recovery_v10_test.go|internal/adapters/state/sqlite/recovery_validation.go|internal/adapters/state/sqlite/recovery_validation_v10.go|internal/adapters/state/sqlite/recovery_validation_v21.go|internal/adapters/state/sqlite/recovery_validation_versions.go|internal/adapters/state/sqlite/repository_test.go|internal/adapters/state/sqlite/v10_migration_test.go|internal/adapters/state/sqlite/v16_workspace_git_test.go|internal/adapters/state/sqlite/v18_migration_test.go|internal/adapters/state/sqlite/v19_migration_test.go|internal/adapters/state/sqlite/validate.go|\
      internal/application/access.go|internal/application/effect_admission.go|internal/application/effect_target_test.go|internal/application/execution_session.go|internal/application/execution_session_test.go|internal/application/mailbox_admission.go|internal/application/mailbox_delivery.go|internal/application/mailbox_test.go|internal/application/mailbox_validation.go|internal/application/orchestrator.go|internal/application/post_artifact_delivery.go|internal/application/processing.go|internal/application/state.go|internal/application/test_support_test.go|internal/application/validation.go|\
      internal/bootstrap/codex_prompt.go|internal/bootstrap/codex_prompt_test.go|internal/bootstrap/execution_authority_test.go|internal/bootstrap/execution_session.go|internal/bootstrap/execution_session_test.go|internal/bootstrap/runtime.go|\
      internal/commands/definitions_generated.go|internal/commands/dispatcher.go|internal/commands/dispatcher_test.go|internal/commands/handlers_goal.go|internal/commands/model.go|internal/commands/output.go|internal/commands/registry.json|internal/commands/v22_public_projection_test.go|\
      internal/config/config_test.go|internal/config/keys_generated.go|internal/config/registry.go|internal/config/value.go|\
      internal/i18n/catalog_test.go|internal/i18n/catalogs/en.json|internal/i18n/catalogs/es.json|internal/i18n/manifest.go|internal/i18n/manifest.json|internal/identity/contracts_test.go|internal/identity/membership.go|internal/identity/policy.go|internal/interfaces/mcp/v22_public_projection_test.go|internal/ports/agent.go|internal/ports/execution_session.go|\
      product/evidence/v22_codex_e2e.json|product/evidence/v22_codex_e2e.output.txt|product/roadmap.json|product/traceability/markdown_source_roles.jsonl|product/traceability/pending_sources.json|product/traceability/source_dispositions.jsonl|product_roadmap_v22_test.go|scripts/check_rebuild_write_set.sh)
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
