package orquestaweb

import (
	"fmt"
	i18ndocs "orquesta/modulos/orquesta-i18n-docs"
)

type NuevaAppI18nOwnerProjectionV0 struct {
	OwnerModule        string   `json:"owner_module"`
	Contract           string   `json:"contract"`
	LoaderContract     string   `json:"loader_contract"`
	FallbackLocale     string   `json:"fallback_locale"`
	RequiredNamespaces []string `json:"required_namespaces"`
	RequiredKeysHash   string   `json:"required_keys_hash"`
	Locales            []string `json:"locales"`
}

func NuevaAppI18nActiveOwnerProjectionV0(catalog NuevaAppI18nCatalogV0) (NuevaAppI18nOwnerProjectionV0, error) {
	if catalog.defaultLocale == "" {
		catalog = NewNuevaAppI18nCatalogV0()
	}
	owner, issues := i18ndocs.BuildActiveI18nDocsCompositionOwnerV0(i18ndocs.PlanSeedV0{
		AppTitle:           "Nueva app",
		PrimaryActionLabel: "Solicitar app",
		UIDefaultLocale:    catalog.DefaultLocale(),
		DocsDefaultLocale:  catalog.DefaultLocale(),
		UILocales:          catalog.SupportedLocales(),
		DocsLocales:        catalog.SupportedLocales(),
	})
	if len(issues) != 0 {
		return NuevaAppI18nOwnerProjectionV0{}, fmt.Errorf("i18n owner projection invalid: %s", issues[0].Code)
	}
	if owner.FallbackLocale != catalog.DefaultLocale() {
		return NuevaAppI18nOwnerProjectionV0{}, fmt.Errorf("i18n owner fallback mismatch: %s", owner.FallbackLocale)
	}
	return NuevaAppI18nOwnerProjectionV0{
		OwnerModule:        owner.OwnerModule,
		Contract:           owner.Contract,
		LoaderContract:     owner.LoaderContract,
		FallbackLocale:     owner.FallbackLocale,
		RequiredNamespaces: append([]string(nil), owner.RequiredNamespaces...),
		RequiredKeysHash:   owner.RequiredKeysHash,
		Locales:            catalog.SupportedLocales(),
	}, nil
}
