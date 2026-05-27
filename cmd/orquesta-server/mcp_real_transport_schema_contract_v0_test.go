package main

import (
	"testing"

	orquestamcp "orquesta/modulos/orquesta-mcp"
	operator "orquesta/modulos/orquesta-operator-mcp"
)

func TestMCPRealTransportV0InputSchemaSaleDeDTOCanonico(t *testing.T) {
	handler, err := newMCPRealHTTPHandlerV0(orquestamcp.MCPTransportBindingsV0{})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	server := newLocalHTTPServerForTestV0(t, handler)
	defer server.Close()

	var tools mcpToolListResultV0
	callMCPJSONRPCTestV0(t, server.URL+mcpRealHTTPPathV0, "tools/list", map[string]any{}, &tools)
	for _, name := range []string{
		orquestamcp.MCPDomainWorkToolNameV0,
		orquestamcp.MCPAppVCSToolNameV0,
		operator.OperatorMCPBurstToolNameV0,
		orquestamcp.MCPOperatorFriendlyStatusToolNameV0,
	} {
		assertMCPRealToolSchemaFromDTOV0(t, tools, name)
	}
}

func assertMCPRealToolSchemaFromDTOV0(t *testing.T, tools mcpToolListResultV0, name string) {
	t.Helper()
	tool := mcpToolByNameTestV0(tools.Tools, name)
	if tool == nil {
		t.Fatalf("tool no registrado: %s", name)
	}
	if tool.InputSchema["x-orquesta-schema_status"] != "ok" {
		t.Fatalf("schema no canonico %s: %+v", name, tool.InputSchema)
	}
	properties, _ := tool.InputSchema["properties"].(map[string]any)
	fields, ok := orquestamcp.MCPTransportToolInputFieldsV0(name)
	if !ok {
		t.Fatalf("sin contrato DTO: %s", name)
	}
	for _, field := range fields {
		if _, ok := properties[field.Name]; !ok {
			t.Fatalf("schema %s no expone campo DTO %s: %+v", name, field.Name, properties)
		}
	}
}
