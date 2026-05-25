package orquestai18ndocs

const (
	ActiveI18nDocsOwnerModuleV0 = "orquesta-i18n-docs"
	GenerarI18nDocsContractV0   = "GenerarI18nDocsIniciales v0"
)

type ActiveI18nDocsCompositionOwnerV0 struct {
	OwnerModule            string   `json:"owner_module"`
	Contract               string   `json:"contract"`
	ContractVersion        string   `json:"contract_version"`
	BundleStructureVersion string   `json:"bundle_structure_version"`
	LoaderContract         string   `json:"loader_contract"`
	FallbackLocale         string   `json:"fallback_locale"`
	RequiredNamespaces     []string `json:"required_namespaces"`
	RequiredKeysHash       string   `json:"required_keys_hash"`
	WebRequiredKeys        []string `json:"web_required_keys"`
	FactoryRequiredKeys    []string `json:"factory_required_keys"`
	RequiredDocTypes       []string `json:"required_doc_types"`
	CanonicalRefs          []string `json:"canonical_refs"`
	Verification           []string `json:"verification"`
}

func BuildActiveI18nDocsCompositionOwnerV0(seed PlanSeedV0) (ActiveI18nDocsCompositionOwnerV0, []I18nDocsValidationIssueV0) {
	plan := BuildAppI18nDocsPlanV0(seed)
	projection := ActiveI18nDocsOwnerFromPlanV0(plan)
	return projection, ValidateAppI18nDocsPlanV0(plan)
}

func ActiveI18nDocsOwnerFromPlanV0(plan AppI18nDocsPlanV0) ActiveI18nDocsCompositionOwnerV0 {
	projection := ActiveI18nDocsCompositionOwnerV0{
		OwnerModule:            ActiveI18nDocsOwnerModuleV0,
		Contract:               GenerarI18nDocsContractV0,
		ContractVersion:        AppI18nDocsPlanContractVersionV0,
		BundleStructureVersion: I18nBundleStructureVersionV0,
		LoaderContract:         plan.SkeletonLoaderShape.LoaderContract,
		FallbackLocale:         plan.SkeletonLoaderShape.FallbackLocale,
		RequiredNamespaces:     copyStringsV0(plan.SkeletonLoaderShape.RequiredNamespaces),
		RequiredKeysHash:       plan.SkeletonLoaderShape.RequiredKeysHash,
		CanonicalRefs: []string{
			"modulos/orquesta-i18n-docs/docs/contratos.md",
			"modulos/orquesta-i18n-docs/docs/schemas/app_i18n_docs_plan_v0.schema.json",
			"modulos/orquesta-i18n-docs/docs/schemas/i18n_bundle_v0.schema.json",
			"modulos/orquesta-i18n-docs/docs/schemas/docs_bundle_v0.schema.json",
		},
		Verification: []string{
			"go test -count=1 ./modulos/orquesta-i18n-docs ./modulos/orquesta-factory ./modulos/orquesta-web ./modulos/orquesta-mcp",
		},
	}
	if plan.WebBundle != nil {
		projection.WebRequiredKeys = copyStringsV0(plan.WebBundle.RequiredKeys)
	}
	if plan.FactoryBundle != nil {
		projection.FactoryRequiredKeys = copyStringsV0(plan.FactoryBundle.RequiredKeys)
	}
	if plan.DocsBundle != nil {
		projection.RequiredDocTypes = copyStringsV0(plan.DocsBundle.RequiredDocTypes)
	}
	return projection
}

func copyStringsV0(values []string) []string {
	return append([]string(nil), values...)
}
