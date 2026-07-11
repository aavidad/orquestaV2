package main

import (
	"testing"

	orquestamcp "orquesta/modulos/orquesta-mcp"
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
	if len(tools.Tools) == 0 {
		t.Fatalf("tools/list no devolvio herramientas")
	}
	for _, tool := range tools.Tools {
		assertMCPRealToolSchemaFromDTOV0(t, tools, tool.Name)
	}
}

func TestMCPRealTransportV0ToolsListExponeAcceptanceChecksV0(t *testing.T) {
	handler, err := newMCPRealHTTPHandlerV0(orquestamcp.MCPTransportBindingsV0{})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	server := newLocalHTTPServerForTestV0(t, handler)
	defer server.Close()

	var tools mcpToolListResultV0
	callMCPJSONRPCTestV0(t, server.URL+mcpRealHTTPPathV0, "tools/list", map[string]any{}, &tools)
	for _, name := range []string{
		orquestamcp.MCPHumanDirectorWorkReviewPlanToolNameV0,
		orquestamcp.MCPAutoprogrammingSelfImprovementToolNameV0,
	} {
		assertMCPRealAcceptanceChecksSchemaV0(t, mcpToolByNameTestV0(tools.Tools, name), name)
	}
}

func assertMCPRealAcceptanceChecksSchemaV0(t *testing.T, tool *mcpToolDescriptorV0, name string) {
	t.Helper()
	if tool == nil {
		t.Fatalf("tool no registrado: %s", name)
	}
	properties, _ := tool.InputSchema["properties"].(map[string]any)
	var checkSchema map[string]any
	if name == orquestamcp.MCPHumanDirectorWorkReviewPlanToolNameV0 {
		workIntake, _ := properties["work_intake"].(map[string]any)
		workIntakeProperties, _ := workIntake["properties"].(map[string]any)
		request, _ := workIntakeProperties["request"].(map[string]any)
		requestProperties, _ := request["properties"].(map[string]any)
		checkSchema, _ = requestProperties["acceptance_checks"].(map[string]any)
	} else {
		proposal, _ := properties["proposal"].(map[string]any)
		proposalProperties, _ := proposal["properties"].(map[string]any)
		checkSchema, _ = proposalProperties["acceptance_checks"].(map[string]any)
	}
	items, _ := checkSchema["items"].(map[string]any)
	itemProperties, _ := items["properties"].(map[string]any)
	for _, field := range []string{"criterion_ref", "description", "command"} {
		if _, ok := itemProperties[field]; !ok {
			t.Fatalf("schema %s no expone acceptance_checks.%s: %+v", name, field, tool.InputSchema)
		}
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
