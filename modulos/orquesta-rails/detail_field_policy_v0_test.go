package orquestarails

import "testing"

func TestDetailRailFieldScopeV0DetectaSecretosEfectivosPeroNoReactivaRailsBlandos(t *testing.T) {
	t.Setenv(RailsModeEnvV0, RailsModeEnforcedV0)
	t.Setenv(DetailProhibitedRailsEnvV0, "on")
	t.Setenv(DetailProhibitedRailsScopeEnvV0, "context_bundle_request.read_set")

	if !TextContainsOperationalSensitiveDetailForFieldV0("context_bundle_request", "objective", "api_key=valor") {
		t.Fatal("secreto efectivo debe detectarse sin depender del scope blando")
	}
	if !TextContainsOperationalSensitiveDetailForFieldV0("context_bundle_request", "read_set", "api_key=valor") {
		t.Fatal("secreto efectivo debe detectarse en scope exacto")
	}

	t.Setenv(DetailProhibitedRailsScopeEnvV0, "context_materialization.*")
	if !TextContainsOperationalRawDetailForFieldV0("context_materialization", "content", "prompt=raw") {
		t.Fatal("prompt crudo debe detectarse como dato sensible efectivo")
	}

	t.Setenv(DetailProhibitedRailsScopeEnvV0, "*.summary")
	if !TextContainsOperationalSensitiveDetailForFieldV0("director_agent_decision", "summary", "client_secret=valor") {
		t.Fatal("client_secret efectivo debe detectarse")
	}

	t.Setenv(DetailProhibitedRailsEnvV0, "off")
	if !TextContainsOperationalSensitiveDetailForFieldV0("director_agent_decision", "summary", "client_secret=valor") {
		t.Fatal("env off no debe apagar secretos efectivos")
	}
}

func TestDetailRailExternalMatrixV0PermaneceDormida(t *testing.T) {
	t.Setenv(RailsModeEnvV0, RailsModeEnforcedV0)
	t.Setenv(DetailProhibitedRailsEnvV0, "on")
	t.Setenv(DetailProhibitedRailsScopeEnvV0, "*")

	allowed := []string{
		"runtime provider model codex git db sql por refs opacas",
		"process ref y pid policy como vocabulario, no valor",
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
			t.Fatalf("valor sensible no detectado: %q", value)
		}
	}
}

func TestDetailRailRawContentMatrixV0PermaneceDormida(t *testing.T) {
	t.Setenv(RailsModeEnvV0, RailsModeEnforcedV0)
	t.Setenv(DetailProhibitedRailsEnvV0, "on")
	t.Setenv(DetailProhibitedRailsScopeEnvV0, "context_materialization.content")

	rejected := []string{
		"sk-test-value",
		"postgres://user:pass@host/db",
		"/home/user/private",
		"pid=1234",
		"process_ref=runtime-ref-001",
		"prompt=raw text",
		"transcript=raw text",
	}
	for _, value := range rejected {
		want := value == "sk-test-value" ||
			value == "postgres://user:pass@host/db" ||
			value == "prompt=raw text" ||
			value == "transcript=raw text"
		got := TextContainsOperationalRawDetailForFieldV0("context_materialization", "content", value)
		if got != want {
			t.Fatalf("raw detail=%v want %v para %q", got, want, value)
		}
	}

	if TextContainsOperationalRawDetailForFieldV0("context_materialization", "content", "prompt policy y transcript policy refs") {
		t.Fatal("politicas opacas de prompt/transcript bloqueadas")
	}
}
