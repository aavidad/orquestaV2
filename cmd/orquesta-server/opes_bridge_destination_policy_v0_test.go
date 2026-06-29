package main

import (
	"strings"
	"testing"
)

func TestOPESDrainConfigV0RechazaCredencialesYQueryEnDestino(t *testing.T) {
	t.Setenv("ORQUESTA_BASE_URL", "http://127.0.0.1:8787")
	t.Setenv("ORQUESTA_OPES_BASE_URL", "http://user:pass@127.0.0.1:18082")
	if _, err := opesDrainConfigFromEnvV0(); err == nil ||
		!strings.Contains(err.Error(), "credentials_not_allowed") {
		t.Fatalf("err=%v", err)
	}

	t.Setenv("ORQUESTA_OPES_BASE_URL", "http://127.0.0.1:18082?token=secret")
	if _, err := opesDrainConfigFromEnvV0(); err == nil ||
		!strings.Contains(err.Error(), "query_not_allowed") {
		t.Fatalf("err=%v", err)
	}
}

func TestOPESDrainConfigV0RequiereConfirmacionParaDestinoNoLocal(t *testing.T) {
	t.Setenv("ORQUESTA_BASE_URL", "http://127.0.0.1:8787")
	t.Setenv("ORQUESTA_OPES_BASE_URL", "https://opes.example.test")

	if _, err := opesDrainConfigFromEnvV0(); err == nil ||
		!strings.Contains(err.Error(), "confirmation_required") {
		t.Fatalf("err=%v", err)
	}

	t.Setenv("ORQUESTA_OPES_TEMPORAL_CONFIRM", "1")
	config, err := opesDrainConfigFromEnvV0()
	if err != nil {
		t.Fatalf("config: %v", err)
	}
	if config.Destination.OPESDestination.Category != "temporal" ||
		config.Destination.OrquestaDestination.Category != "loopback" ||
		config.Destination.DestinationEvidenceRef == "" {
		t.Fatalf("destination=%+v", config.Destination)
	}

	t.Setenv("ORQUESTA_OPES_BRIDGE_DESTINATION_EVIDENCE_REF", "bad ref with spaces")
	if _, err := opesDrainConfigFromEnvV0(); err == nil ||
		!strings.Contains(err.Error(), "evidence_ref_invalid") {
		t.Fatalf("err=%v", err)
	}
}

func TestOPESDrainConfigV0BridgeConfirmNoSustituyeTemporalConfirm(t *testing.T) {
	t.Setenv("ORQUESTA_BASE_URL", "http://127.0.0.1:8787")
	t.Setenv("ORQUESTA_OPES_BASE_URL", "https://opes.example.test")
	t.Setenv("ORQUESTA_OPES_BRIDGE_CONFIRM", "1")

	if _, err := opesDrainConfigFromEnvV0(); err == nil ||
		!strings.Contains(err.Error(), "confirmation_required") {
		t.Fatalf("err=%v", err)
	}
}

func TestOPESDrainConfigV0RechazaConfirmacionProductiva(t *testing.T) {
	t.Setenv("ORQUESTA_BASE_URL", "http://127.0.0.1:8787")
	t.Setenv("ORQUESTA_OPES_BASE_URL", "https://opes.example.com")
	t.Setenv("ORQUESTA_OPES_BRIDGE_PRODUCTIVE_CONFIRM", "1")

	if _, err := opesDrainConfigFromEnvV0(); err == nil ||
		!strings.Contains(err.Error(), "productive_not_allowed") {
		t.Fatalf("err=%v", err)
	}

	t.Setenv("ORQUESTA_OPES_TEMPORAL_CONFIRM", "1")
	t.Setenv("ORQUESTA_OPES_BRIDGE_DESTINATION_EVIDENCE_REF", "evidence-ref-opes-prod-001")
	if _, err := opesDrainConfigFromEnvV0(); err == nil ||
		!strings.Contains(err.Error(), "productive_not_allowed") {
		t.Fatalf("err=%v", err)
	}
}
