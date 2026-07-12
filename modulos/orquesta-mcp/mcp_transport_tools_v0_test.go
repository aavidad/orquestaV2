package orquestamcp

import (
	"context"
	"encoding/json"
	"testing"
)

func TestMCPDocumentPlanExpandTransportV0SoloSeAnunciaConBinding(t *testing.T) {
	for _, tool := range MCPBoundTransportToolsV0(MCPTransportBindingsV0{}) {
		if tool.Name == MCPDocumentPlanExpandToolNameV0 {
			t.Fatalf("tool opt-in sin binding anunciada")
		}
	}

	for _, tool := range MCPTransportToolsV0(MCPTransportBindingsV0{}) {
		if tool.Name != MCPDocumentPlanExpandToolNameV0 {
			continue
		}
		output, err := tool.Handler(context.Background(), json.RawMessage(`{}`))
		if err != nil {
			t.Fatalf("handler nil binding: %v", err)
		}
		var public MCPTransportToolErrorV0
		if err := json.Unmarshal(output, &public); err != nil {
			t.Fatalf("decode nil binding: %v", err)
		}
		if public.ErrorCode != MCPTransportToolUnboundV0 {
			t.Fatalf("nil binding=%+v", public)
		}
		return
	}
	t.Fatalf("tool no declarada")
}
