package main

import (
	"strings"

	orquestamcp "orquesta/modulos/orquesta-mcp"
)

func (registry *mcpRealTransportRegistryV0) toolListV0() mcpToolListResultV0 {
	names := sortedKeysMCPRealV0(registry.tools)
	tools := make([]mcpToolDescriptorV0, 0, len(names))
	for _, name := range names {
		tool := registry.tools[name]
		tools = append(tools, mcpToolDescriptorV0{
			Name:        tool.Name,
			Description: mcpToolDescriptionV0(tool),
			InputSchema: mcpToolInputSchemaV0(tool),
		})
	}
	return mcpToolListResultV0{Tools: tools}
}

func mcpToolDescriptionV0(tool orquestamcp.MCPTransportToolEnvelopeV0) string {
	parts := []string{strings.TrimSpace(tool.ResourceURI)}
	if strings.TrimSpace(tool.InputShape) != "" {
		parts = append(parts, "input: "+strings.TrimSpace(tool.InputShape))
	}
	if strings.TrimSpace(tool.OutputShape) != "" {
		parts = append(parts, "output: "+strings.TrimSpace(tool.OutputShape))
	}
	switch strings.TrimSpace(tool.Name) {
	case "orquesta.status.v0":
		parts = append(parts, "Use this for general Orquesta status, como va, queue summary, active projects and safe next actions. No internal refs required.")
	case "orquesta.tasks.list.v0":
		parts = append(parts, "Use this to list queued/active/completed visible tasks. No internal refs required.")
	case "orquesta.projects.list.v0":
		parts = append(parts, "Use this to list active projects visible from the queue. No internal refs required.")
	case "orquesta.agents.list.v0":
		parts = append(parts, "Use this to list agents for visible runs. No internal refs required.")
	case "orquesta.operator.command.v0":
		parts = append(parts, "Use this for human natural-language operational commands when no more specific Orquesta tool is obvious.")
	case "orquesta.operator.status.query.v0", "orquesta.operator.directed_query.v0":
		parts = append(parts, "Low-level operator connector tool. Do not use for general status; prefer orquesta.status.v0 unless connector refs are already known.")
	}
	return strings.Join(compactMCPRealStringsV0(parts), "\n")
}

func mcpToolInputSchemaV0(tool orquestamcp.MCPTransportToolEnvelopeV0) map[string]any {
	description := strings.TrimSpace(tool.InputShape)
	if description == "" {
		description = "JSON envelope"
	}
	schema := map[string]any{
		"type":                 "object",
		"description":          description,
		"additionalProperties": true,
		"x-orquesta-public-errors": []string{
			orquestamcp.MCPTransportToolUnboundV0,
			orquestamcp.MCPTransportPortUnavailableV0,
			orquestamcp.MCPTransportSchemaStaleV0,
		},
	}
	properties, required, schemaStatus := mcpToolSchemaFieldsV0(tool)
	schema["x-orquesta-schema_status"] = schemaStatus
	if schemaStatus != "ok" {
		schema["x-orquesta-reason_code"] = orquestamcp.MCPTransportSchemaStaleV0
	}
	if len(properties) > 0 {
		schema["properties"] = properties
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

func mcpToolSchemaFieldsV0(
	tool orquestamcp.MCPTransportToolEnvelopeV0,
) (map[string]any, []string, string) {
	if properties, required, ok := mcpCanonicalToolSchemaV0(tool.Name); ok {
		return properties, required, "ok"
	}
	fields := mcpInputShapeFieldsV0(tool.InputShape)
	properties := map[string]any{}
	required := []string{}
	for _, field := range fields {
		properties[field.name] = mcpInputShapeFieldSchemaV0(field)
		if field.required {
			required = append(required, field.name)
		}
	}
	return properties, required, "schema_stale"
}

func mcpCanonicalToolSchemaV0(name string) (map[string]any, []string, bool) {
	fields, ok := orquestamcp.MCPTransportToolInputFieldsV0(name)
	if !ok {
		return nil, nil, false
	}
	properties := map[string]any{}
	required := []string{}
	for _, field := range fields {
		properties[field.Name] = mcpCanonicalInputFieldSchemaV0(field)
		if field.Required {
			required = append(required, field.Name)
		}
	}
	return properties, required, true
}

func mcpCanonicalInputFieldSchemaV0(
	field orquestamcp.MCPTransportToolInputFieldV0,
) map[string]any {
	schema := map[string]any{"type": field.Type}
	if len(field.Enum) > 0 {
		schema["enum"] = field.Enum
	}
	return schema
}
