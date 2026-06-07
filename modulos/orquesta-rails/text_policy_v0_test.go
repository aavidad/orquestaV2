package orquestarails

import "testing"

func TestTextPolicyV0PermiteVocabularioOperativoOpaco(t *testing.T) {
	values := []string{
		"runtime provider model codex git db sql por refs opacas",
		"prompt policy ref y transcript policy ref sin contenido crudo",
		"token budget y secrets policy como politica, no valor",
		"la clave del ejercicio es reutilizar temas comunes ya revisados",
		"capacity_limited fue un diagnostico viejo y no debe cortar entregas",
		"el provider visual y el model editorial son roles de composicion",
		"clave, token, prompt y transcript como palabras docentes sin valor secreto",
	}
	for _, value := range values {
		if TextContainsOperationalSensitiveDetailV0(value) {
			t.Fatalf("texto operativo rechazado: %q", value)
		}
	}
}

func TestTextPolicyV0NoUsaPalabrasGenericasComoRailSensible(t *testing.T) {
	values := []string{
		"clave",
		"clave de estudio",
		"palabra clave del tema",
		"secret como nombre de politica sin valor",
		"secreto profesional explicado como concepto juridico",
		"capacity provider model runtime token prompt transcript",
	}
	for _, value := range values {
		if TextContainsOperationalSensitiveDetailV0(value) {
			t.Fatalf("palabra generica tratada como secreto: %q", value)
		}
	}
}

func TestTextPolicyV0DetalleProhibidoDesactivadoPorDefecto(t *testing.T) {
	t.Setenv(RailsModeEnvV0, "")
	t.Setenv(DetailProhibitedRailsEnvV0, "")
	if DetailProhibitedRailsEnabledV0() {
		t.Fatal("detalle prohibido debe estar desactivado por defecto")
	}
	if !TextContainsOperationalSensitiveDetailV0("access_token=valor") {
		t.Fatal("secreto efectivo debe detectarse aunque el rail blando este apagado")
	}
}

func TestTextPolicyV0DetalleProhibidoOfflineAunqueSePidieraOn(t *testing.T) {
	t.Setenv(RailsModeEnvV0, "")
	t.Setenv(DetailProhibitedRailsEnvV0, "on")
	if DetailProhibitedRailsEnabledV0() {
		t.Fatal("modo rails offline debe impedir bloqueo aunque detalle este on")
	}
	if !TextContainsOperationalSensitiveDetailV0("access_token=valor") {
		t.Fatal("secreto efectivo debe detectarse aunque rails este offline")
	}
}

func TestTextPolicyV0DetectaPatronesConValorSensibleAunqueRailsEsteOffline(t *testing.T) {
	t.Setenv(RailsModeEnvV0, RailsModeEnforcedV0)
	t.Setenv(DetailProhibitedRailsEnvV0, "on")
	values := []string{
		"api_key=valor",
		"client_secret: valor",
		"authorization: bearer valor",
		"-----BEGIN PRIVATE KEY-----",
		"prompt=raw",
		"transcript=raw",
		"postgres://user:pass@host/db",
	}
	for _, value := range values {
		if !TextContainsOperationalSensitiveDetailV0(value) {
			t.Fatalf("texto sensible no detectado: %q", value)
		}
	}
}

func TestTextPolicyV0DetalleProhibidoDesactivadoTemporalmente(t *testing.T) {
	t.Setenv(RailsModeEnvV0, RailsModeEnforcedV0)
	t.Setenv(DetailProhibitedRailsEnvV0, "off")
	values := []string{
		"api_key=valor",
		"client_secret: valor",
		"authorization: bearer valor",
		"-----BEGIN PRIVATE KEY-----",
	}
	for _, value := range values {
		if !TextContainsOperationalSensitiveDetailV0(value) {
			t.Fatalf("secreto efectivo no detectado con rail blando off: %q", value)
		}
	}
}

func TestTextPolicyV0MarcadoresLegacyNoBloqueanAunqueSePidaEnforced(t *testing.T) {
	t.Setenv(RailsModeEnvV0, RailsModeEnforcedV0)
	t.Setenv(DetailProhibitedRailsEnvV0, "on")
	values := []string{
		"oauth token prompt=raw completion=raw",
		`C:\Users\operador\codex`,
		"provider=modelo model=gpt postgres sqlite",
	}
	for _, value := range values {
		if TextContainsOperationalDetailMarkerV0(value) {
			t.Fatalf("marcador legacy bloqueo con rails quitados: %q", value)
		}
	}
}

func TestTextPolicyV0MarcadoresLegacyNoBloqueanConRailsOff(t *testing.T) {
	t.Setenv(RailsModeEnvV0, RailsModeEnforcedV0)
	t.Setenv(DetailProhibitedRailsEnvV0, "off")
	if TextContainsOperationalDetailMarkerV0("oauth token prompt=raw completion=raw /home/alberto") ||
		BytesContainOperationalDetailMarkerV0([]byte(`{"note":"token=valor"}`)) {
		t.Fatal("marcador legacy bloqueo con rails off")
	}
}

func TestTextPolicyV0ConservaListaParaReactivacionFutura(t *testing.T) {
	if len(OperationalSensitiveFragmentsV0) == 0 {
		t.Fatal("lista de fragmentos sensibles vacia")
	}
	for _, want := range []string{"api_key=", "access_token=", "bearer ", "-----begin "} {
		found := false
		for _, got := range OperationalSensitiveFragmentsV0 {
			if got == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("fragmento sensible no conservado: %q", want)
		}
	}
	for _, want := range []string{"token", "oauth", "prompt=", "transcript", "/home/"} {
		found := false
		for _, got := range OperationalDetailMarkersV0 {
			if got == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("marcador legacy no conservado: %q", want)
		}
	}
}
