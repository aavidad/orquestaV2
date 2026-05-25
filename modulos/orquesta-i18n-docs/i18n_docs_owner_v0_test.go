package orquestai18ndocs

import "testing"

func TestActiveI18nDocsCompositionOwnerV0DeclaresCanonicalProjection(t *testing.T) {
	owner, issues := BuildActiveI18nDocsCompositionOwnerV0(PlanSeedV0{
		AppTitle:          "Panel de reservas",
		UIDefaultLocale:   "es-ES",
		DocsDefaultLocale: "es-ES",
		UILocales:         []string{"es-ES", "en-US"},
		DocsLocales:       []string{"es-ES", "en-US"},
	})
	if len(issues) != 0 {
		t.Fatalf("owner projection should be valid: %+v", issues)
	}
	if owner.OwnerModule != ActiveI18nDocsOwnerModuleV0 || owner.Contract != GenerarI18nDocsContractV0 {
		t.Fatalf("owner identity: %+v", owner)
	}
	if owner.LoaderContract != i18nLoaderContractV0 || owner.FallbackLocale != "es-ES" {
		t.Fatalf("loader projection: %+v", owner)
	}
	if owner.RequiredKeysHash == "" || len(owner.WebRequiredKeys) == 0 || len(owner.FactoryRequiredKeys) == 0 {
		t.Fatalf("required keys projection incomplete: %+v", owner)
	}
	if len(owner.RequiredDocTypes) != len(requiredDocTypesV0) {
		t.Fatalf("doc types projection: %+v", owner.RequiredDocTypes)
	}
	if len(owner.CanonicalRefs) == 0 || len(owner.Verification) == 0 {
		t.Fatalf("owner lacks evidence refs: %+v", owner)
	}
}
