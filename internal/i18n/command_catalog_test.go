package i18n_test

import (
	"testing"

	commandcore "orquesta/internal/commands"
	"orquesta/internal/i18n"
)

func TestCommandCatalogKeysExistInBundledLocalesWithoutOwningV21Formatting(t *testing.T) {
	catalog, err := i18n.LoadBundled()
	if err != nil {
		t.Fatal(err)
	}
	for _, definition := range commandcore.CanonicalDefinitions() {
		for _, locale := range []string{"es", "en"} {
			if text := catalog.Text(locale, definition.DescriptionKey); text == definition.DescriptionKey || text == "" {
				t.Errorf("missing %s command description %q", locale, definition.DescriptionKey)
			}
		}
		for _, code := range definition.ErrorCodes {
			key := "error." + code
			for _, locale := range []string{"es", "en"} {
				if text := catalog.Text(locale, key); text == key || text == "" {
					t.Errorf("missing %s stable error key %q", locale, key)
				}
			}
		}
	}
}
