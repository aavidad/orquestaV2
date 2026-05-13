package orquestamcp

import (
	"context"
	"encoding/json"
	"testing"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

func TestMCPDomainWorkTransportV0InvocaExecutorYQuedaOptIn(t *testing.T) {
	creator := &fakeMCPDomainWorkCreatorV0{
		job: orquestadomainwork.DomainWorkJobV0{
			SchemaVersion: orquestadomainwork.DomainWorkJobSchemaV0,
			Status:        orquestadomainwork.DomainWorkStatusAcceptedV0,
			JobRef:        "job-domain-transport-001",
		},
	}
	handler := mcpDomainWorkTransportHandlerV0(MCPDomainWorkToolExecutorV0{JobCreator: creator})
	raw, err := json.Marshal(MCPDomainWorkToolInputV0{Action: MCPDomainWorkActionCreateJobV0})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	output, err := handler(context.Background(), raw)
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	var result MCPDomainWorkToolResultV0
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if result.Estado != MCPDomainWorkEstadoOKV0 ||
		result.Job == nil ||
		result.Job.JobRef != "job-domain-transport-001" {
		t.Fatalf("result=%+v", result)
	}
	assertTransportPayloadSaneadoMCPTestV0(t, output, 800)

	output, err = mcpDomainWorkTransportHandlerV0(nil)(context.Background(), raw)
	if err != nil {
		t.Fatalf("handler unbound: %v", err)
	}
	var unbound MCPTransportToolErrorV0
	if err := json.Unmarshal(output, &unbound); err != nil {
		t.Fatalf("decode unbound: %v", err)
	}
	if unbound.ErrorCode != MCPTransportToolUnboundV0 {
		t.Fatalf("unbound=%+v", unbound)
	}
}
