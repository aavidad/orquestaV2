package orquestaobservability

import "testing"

func TestNormalizeMCPExecutionObservationV0CompactaSinDatosSensibles(t *testing.T) {
	observation := NormalizeMCPExecutionObservationV0(MCPExecutionObservationV0{
		Method:         "orquesta.autoprogramming.prepare_run.v0",
		Profile:        "autoprogramming_long",
		ReasonCode:     MCPExecutionReasonTimeoutV0,
		DurationBucket: "10s_30s",
		CorrelationID:  "corr-mcp-execution-001",
	})
	if observation.SchemaVersion != MCPExecutionObservationSchemaVersionV0 ||
		observation.Method == "" ||
		observation.Profile != "autoprogramming_long" ||
		observation.ReasonCode != MCPExecutionReasonTimeoutV0 ||
		observation.CorrelationID == "" {
		t.Fatalf("observation=%+v", observation)
	}

	redacted := NormalizeMCPExecutionObservationV0(MCPExecutionObservationV0{
		Method:        "/home/alberto/prompts/token.txt",
		Profile:       "secret-profile",
		ReasonCode:    "",
		CorrelationID: "corr-token",
	})
	if redacted.Method != "mcp_unknown" ||
		redacted.Profile != "unknown" ||
		redacted.ReasonCode != MCPExecutionReasonOKV0 ||
		redacted.CorrelationID != "" {
		t.Fatalf("redacted=%+v", redacted)
	}
}
