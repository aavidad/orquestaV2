package orquestamcp

import "testing"

func TestMCPAutoprogrammingPrepareRunDescriptorV0EsAdaptadorOptIn(t *testing.T) {
	descriptor := MCPAutoprogrammingPrepareRunDescriptorV0()

	if descriptor.Name != MCPAutoprogrammingPrepareRunToolNameV0 ||
		descriptor.ResourceURI != MCPAutoprogrammingPrepareRunResourceURIV0 ||
		descriptor.InputSchema == "" ||
		descriptor.Output == "" {
		t.Fatalf("descriptor=%+v", descriptor)
	}
	if len(descriptor.Invariantes) == 0 {
		t.Fatalf("invariantes vacias")
	}
}

func TestNewMCPAutoprogrammingPrepareRunErrorResultV0NormalizaErrorPublico(t *testing.T) {
	result := NewMCPAutoprogrammingPrepareRunErrorResultV0(
		MCPAutoprogrammingPrepareRunToolInputV0{
			RequestID:     "request-prepare-run-001",
			CorrelationID: "corr-prepare-run-001",
		},
		"",
		"executor",
		"",
	)

	if result.Estado != MCPAutoprogrammingPrepareRunEstadoErrorV0 ||
		result.Accepted ||
		result.RequestID != "request-prepare-run-001" ||
		result.CorrelationID != "corr-prepare-run-001" ||
		len(result.Errores) != 1 ||
		result.Errores[0].Code == "" ||
		result.Errores[0].Message == "" {
		t.Fatalf("result=%+v", result)
	}
}
