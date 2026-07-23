package i18n_test

import (
	"testing"

	commandcore "orquesta/internal/commands"
	"orquesta/internal/i18n"
)

func TestCommandCatalogKeysExistInEveryBundledLocale(t *testing.T) {
	assertCommandCatalogKeysExistInEveryBundledLocale(t)
}

// Kept while V20 evidence names this contract; V21 adds stricter formatting
// coverage without invalidating the previously accredited test subject.
func TestCommandCatalogKeysExistInBundledLocalesWithoutOwningV21Formatting(t *testing.T) {
	assertCommandCatalogKeysExistInEveryBundledLocale(t)
}

func assertCommandCatalogKeysExistInEveryBundledLocale(t *testing.T) {
	t.Helper()
	catalog, err := i18n.LoadBundled()
	if err != nil {
		t.Fatal(err)
	}
	for _, definition := range commandcore.CanonicalDefinitions() {
		for _, locale := range catalog.Locales() {
			if text, textErr := catalog.Text(locale, definition.DescriptionKey); textErr != nil || text == "" {
				t.Errorf("missing %s command description %q: %v", locale, definition.DescriptionKey, textErr)
			}
		}
		for _, code := range definition.ErrorCodes {
			key := "error." + code
			for _, locale := range catalog.Locales() {
				if text, textErr := catalog.Text(locale, key); textErr != nil || text == "" {
					t.Errorf("missing %s stable error key %q: %v", locale, key, textErr)
				}
			}
		}
	}
}
