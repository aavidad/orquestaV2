package orquestamcp

import i18ndocs "orquesta/modulos/orquesta-i18n-docs"

const mcpBacklogT75RefV0 = "docs/autoprogramacion_orquesta_pendientes_2026-05-23.md#T75-i18n-docs-active-composition-owner"

func mcpI18nDocsSharedContractSourceV0() mcpSharedContractSourceV0 {
	owner, _ := i18ndocs.BuildActiveI18nDocsCompositionOwnerV0(i18ndocs.PlanSeedV0{})
	return mcpSharedContractSourceV0{
		Name:       "GenerarI18nDocsIniciales",
		Slug:       "generar-i18n-docs-iniciales",
		Version:    owner.ContractVersion,
		Owner:      owner.OwnerModule,
		MCPRole:    "contract_reference",
		SummaryKey: "mcp.contracts.generar_i18n_docs_iniciales.summary.v0",
		CanonicalRefs: []string{
			"orquesta-i18n-docs/docs/contratos.md",
			"orquesta-i18n-docs/docs/schemas/app_i18n_docs_plan_v0.schema.json",
			"orquesta-i18n-docs/docs/schemas/i18n_bundle_v0.schema.json",
			"orquesta-i18n-docs/docs/schemas/docs_bundle_v0.schema.json",
		},
		Input:  "AppSpecV0 validada + locales + salidas solicitadas",
		Output: "AppI18nDocsPlanV0",
		PublicErrors: []string{
			"app_spec_requerida",
			"app_spec_no_validada",
			"idioma_ui_requerido",
			"idioma_docs_requerido",
			"idioma_invalido",
			"default_locale_fuera_de_catalogo",
			"salida_requerida_incompatible",
			"plantilla_no_disponible",
			"locale_sin_catalogo",
			"clave_requerida_faltante",
			"clave_inestable",
			"estructura_bundle_incompatible",
			"documento_sin_idioma",
			"tipo_documento_faltante",
			"contenido_sin_clave_i18n",
		},
		Guardrails: []string{
			"textos_visibles_por_catalogos",
			"documento_declara_locale",
			"fallback_locale_de_owner_activo",
			"required_keys_hash_de_owner_activo",
			"plan_serializable_no_filesystem",
			"sin_llm_persistencia_apps_tareas_backlog",
			"owner_activo_i18n_docs",
		},
		ProgressKey:  "mcp.contracts.progress.builder_harness_relacional_vigente.v0",
		BacklogRefs:  []string{mcpBacklogT75RefV0},
		Verification: owner.Verification,
	}
}
