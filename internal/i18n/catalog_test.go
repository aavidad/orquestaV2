package i18n

import "testing"

func TestBundledCatalogHasParityAndSpanishFallback(t *testing.T) {
	catalog, err := LoadBundled()
	if err != nil {
		t.Fatalf("LoadBundled() error = %v", err)
	}

	if got := catalog.Text("en", "tool.goals.create.description"); got != "Create a durable Goal for the selected project." {
		t.Fatalf("english text = %q", got)
	}
	if got := catalog.Text("es", "tool.goals.amend.description"); got != "Crea un sucesor causal confirmado para un Goal terminal." {
		t.Fatalf("amendment text = %q", got)
	}
	if got := catalog.Text("en", "tool.goals.amend.description"); got != "Create a confirmed causal successor for a terminal Goal." {
		t.Fatalf("english amendment text = %q", got)
	}
	if got := catalog.Text("gl", "error.not_found"); got != "No se encontró el recurso solicitado." {
		t.Fatalf("fallback text = %q", got)
	}
	if got := catalog.Text("es-ES", "error.invalid_request"); got != "La solicitud no es válida." {
		t.Fatalf("regional text = %q", got)
	}
	if got := catalog.Text("es", "missing.key"); got != "missing.key" {
		t.Fatalf("missing key = %q", got)
	}
}

func TestValidateParityRejectsMissingAndExtraKeys(t *testing.T) {
	missing := map[string]map[string]string{
		"es": {"a": "A"},
		"en": {},
	}
	if err := validateParity(missing); err == nil {
		t.Fatal("validateParity() accepted missing key")
	}

	extra := map[string]map[string]string{
		"es": {"a": "A"},
		"en": {"a": "A", "b": "B"},
	}
	if err := validateParity(extra); err == nil {
		t.Fatal("validateParity() accepted extra key")
	}
}
