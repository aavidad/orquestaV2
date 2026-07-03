package orquestamcp

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestMCPTransportToolEnvelopeMarshalJSONV0CompactaShapesSinMutarContrato(t *testing.T) {
	longShape := "ok:{" + strings.Repeat("field?,", 40) + "evidence_refs?}"
	tool := MCPTransportToolEnvelopeV0{
		Name:            "orquesta.test.long_shape.v0",
		Version:         "v0",
		ResourceURI:     "orquesta://contracts/test-long-shape/v0",
		InputShape:      "envelope:{request_id?,correlation_id?}",
		OutputShape:     longShape,
		Mode:            MCPTransportModeOptInV0,
		OutputBudget:    MCPTransportToolOutputBudgetV0(MCPTransportOutputFreshnessLiveV0),
		ExecutionBudget: MCPTransportToolExecutionBudgetV0(MCPTransportExecutionProfileDefaultToolV0),
	}

	raw, err := json.Marshal(tool)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(raw), `"output_shape":"shape_ref:orquesta://contracts/test-long-shape/v0#output"`) {
		t.Fatalf("shape no compactado: %s", raw)
	}
	if tool.OutputShape != longShape {
		t.Fatalf("contrato interno mutado")
	}
}
