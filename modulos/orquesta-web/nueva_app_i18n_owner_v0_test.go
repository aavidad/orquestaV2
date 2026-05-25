package orquestaweb

import (
	i18ndocs "orquesta/modulos/orquesta-i18n-docs"
	"testing"
)

func TestNuevaAppI18nActiveOwnerProjectionV0ConsumesI18nDocsOwner(t *testing.T) {
	projection, err := NuevaAppI18nActiveOwnerProjectionV0(NewNuevaAppI18nCatalogV0())
	if err != nil {
		t.Fatalf("owner projection: %v", err)
	}
	if projection.OwnerModule != i18ndocs.ActiveI18nDocsOwnerModuleV0 {
		t.Fatalf("owner module=%q", projection.OwnerModule)
	}
	if projection.FallbackLocale != NuevaAppI18nDefaultLocaleV0 {
		t.Fatalf("fallback=%q", projection.FallbackLocale)
	}
	if projection.RequiredKeysHash == "" || len(projection.RequiredNamespaces) == 0 {
		t.Fatalf("projection incomplete: %+v", projection)
	}
	if len(projection.Locales) != len(NewNuevaAppI18nCatalogV0().SupportedLocales()) {
		t.Fatalf("locales=%+v", projection.Locales)
	}
}
