package orquestarails

import "testing"

func TestTextPolicyV0PermiteVocabularioOperativoOpaco(t *testing.T) {
	values := []string{
		"runtime provider model codex git db sql por refs opacas",
		"prompt policy ref y transcript policy ref sin contenido crudo",
		"token budget y secrets policy como politica, no valor",
	}
	for _, value := range values {
		if TextContainsOperationalSensitiveDetailV0(value) {
			t.Fatalf("texto operativo rechazado: %q", value)
		}
	}
}

func TestTextPolicyV0DetalleProhibidoDesactivadoPorDefecto(t *testing.T) {
	t.Setenv(DetailProhibitedRailsEnvV0, "")
	if DetailProhibitedRailsEnabledV0() ||
		TextContainsOperationalSensitiveDetailV0("access_token=valor") {
		t.Fatal("detalle prohibido debe estar desactivado por defecto")
	}
}

func TestTextPolicyV0RechazaPatronesConValorSensible(t *testing.T) {
	t.Setenv(DetailProhibitedRailsEnvV0, "on")
	values := []string{
		"api_key=valor",
		"client_secret: valor",
		"authorization: bearer valor",
		"-----BEGIN PRIVATE KEY-----",
	}
	for _, value := range values {
		if !TextContainsOperationalSensitiveDetailV0(value) {
			t.Fatalf("texto sensible aceptado: %q", value)
		}
	}
}

func TestTextPolicyV0DetalleProhibidoDesactivadoTemporalmente(t *testing.T) {
	t.Setenv(DetailProhibitedRailsEnvV0, "off")
	values := []string{
		"api_key=valor",
		"client_secret: valor",
		"authorization: bearer valor",
		"-----BEGIN PRIVATE KEY-----",
	}
	for _, value := range values {
		if TextContainsOperationalSensitiveDetailV0(value) {
			t.Fatalf("rail temporalmente desactivado rechazo: %q", value)
		}
	}
}

func TestTextPolicyV0AgrupaMarcadoresLegacyDeDetalle(t *testing.T) {
	t.Setenv(DetailProhibitedRailsEnvV0, "on")
	values := []string{
		"oauth token prompt=raw completion=raw",
		`C:\Users\operador\codex`,
		"provider=modelo model=gpt postgres sqlite",
	}
	for _, value := range values {
		if !TextContainsOperationalDetailMarkerV0(value) {
			t.Fatalf("marcador legacy no detectado: %q", value)
		}
	}
}

func TestTextPolicyV0MarcadoresLegacyNoBloqueanConRailsOff(t *testing.T) {
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
