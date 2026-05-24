package orquestarails

import "testing"

func TestDetailRailFieldScopeV0(t *testing.T) {
	t.Setenv(DetailProhibitedRailsEnvV0, "on")
	t.Setenv(DetailProhibitedRailsScopeEnvV0, "context_bundle_request.read_set")

	if !TextContainsOperationalSensitiveDetailForFieldV0("context_bundle_request", "read_set", "api_key=valor") {
		t.Fatal("scope exacto no activo")
	}
	if TextContainsOperationalSensitiveDetailForFieldV0("context_bundle_request", "objective", "api_key=valor") {
		t.Fatal("scope de otro campo activo")
	}

	t.Setenv(DetailProhibitedRailsScopeEnvV0, "context_materialization.*")
	if !TextContainsOperationalRawDetailForFieldV0("context_materialization", "content", "prompt=raw") {
		t.Fatal("scope de frontera no activo")
	}

	t.Setenv(DetailProhibitedRailsScopeEnvV0, "*.summary")
	if !TextContainsOperationalSensitiveDetailForFieldV0("director_agent_decision", "summary", "client_secret=valor") {
		t.Fatal("scope wildcard por campo no activo")
	}

	t.Setenv(DetailProhibitedRailsEnvV0, "off")
	if TextContainsOperationalSensitiveDetailForFieldV0("director_agent_decision", "summary", "client_secret=valor") {
		t.Fatal("rail activo con env off")
	}
}

func TestDetailRailExternalMatrixV0(t *testing.T) {
	t.Setenv(DetailProhibitedRailsEnvV0, "on")
	t.Setenv(DetailProhibitedRailsScopeEnvV0, "*")

	allowed := []string{
		"runtime provider model codex git db sql por refs opacas",
		"prompt policy ref y transcript policy ref sin contenido crudo",
		"token budget y secrets policy como politica, no valor",
		"web_application como alias reparable",
		"required test runner por puerto neutral",
	}
	for _, value := range allowed {
		if TextContainsOperationalSensitiveDetailForFieldV0("core_workflow", "summary", value) {
			t.Fatalf("falso positivo operacional: %q", value)
		}
		if TextContainsOperationalRawDetailForFieldV0("context_materialization", "content", value) {
			t.Fatalf("falso positivo raw operacional: %q", value)
		}
	}

	rejected := []string{
		"api_key=valor",
		"client_secret: valor",
		"authorization: bearer valor",
		"-----BEGIN PRIVATE KEY-----",
	}
	for _, value := range rejected {
		if !TextContainsOperationalSensitiveDetailForFieldV0("core_workflow", "summary", value) {
			t.Fatalf("valor sensible aceptado: %q", value)
		}
	}
}

func TestDetailRailRawContentMatrixV0(t *testing.T) {
	t.Setenv(DetailProhibitedRailsEnvV0, "on")
	t.Setenv(DetailProhibitedRailsScopeEnvV0, "context_materialization.content")

	rejected := []string{
		"sk-test-value",
		"postgres://user:pass@host/db",
		"/home/user/private",
		"prompt=raw text",
		"transcript=raw text",
	}
	for _, value := range rejected {
		if !TextContainsOperationalRawDetailForFieldV0("context_materialization", "content", value) {
			t.Fatalf("raw sensible aceptado: %q", value)
		}
	}

	if TextContainsOperationalRawDetailForFieldV0("context_materialization", "content", "prompt policy y transcript policy refs") {
		t.Fatal("politicas opacas de prompt/transcript bloqueadas")
	}
}
