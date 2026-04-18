package cmd

import "testing"

func TestRuntimeTranscriptTextoPareceConsultaCLINuevasClasificaciones(t *testing.T) {
	for _, classification := range []string{"cli_query", "credentials_request"} {
		if !runtimeTranscriptTextoPareceConsultaCLI("irrelevante", classification) {
			t.Fatalf("classification %s deberia tratarse como consulta CLI", classification)
		}
	}
}

func TestColetillaRespuestaSignalTranscriptNuevasClasificaciones(t *testing.T) {
	if got := coletillaRespuestaSignalTranscript("cli_query", ""); got == "" {
		t.Fatalf("cli_query deberia producir coletilla operativa")
	}
	if got := coletillaRespuestaSignalTranscript("credentials_request", ""); got == "" {
		t.Fatalf("credentials_request deberia producir coletilla operativa")
	}
}
