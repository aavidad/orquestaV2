package orquestamcp

import (
	"strings"
	"testing"
)

func TestMCPTransportToolInputSchemaV0CubreToolsRegistrados(t *testing.T) {
	transport := newFakeMCPTransportV0()
	if err := RegisterMCPTransportV0(transport, MCPTransportBindingsV0{}); err != nil {
		t.Fatalf("register transport: %v", err)
	}
	for name, tool := range transport.tools {
		fields, ok := MCPTransportToolInputFieldsV0(name)
		if !ok || len(fields) == 0 {
			t.Fatalf("tool sin DTO canonico: %s", name)
		}
		if strings.TrimSpace(tool.OutputShape) == "" {
			t.Fatalf("tool sin output shape: %s", name)
		}
		assertMCPToolDescriptorFieldsMatchDTOV0(t, name, tool.InputShape, fields)
		assertTransportPayloadSaneadoMCPTestV0(t, tool, 2500)
	}
}

func assertMCPToolDescriptorFieldsMatchDTOV0(
	t *testing.T,
	name string,
	inputShape string,
	fields []MCPTransportToolInputFieldV0,
) {
	t.Helper()
	known := map[string]bool{}
	for _, field := range fields {
		known[field.Name] = true
	}
	for _, field := range mcpToolInputFieldNamesFromShapeTestV0(inputShape) {
		if !known[field] {
			t.Fatalf("tool %s anuncia campo stale %q en input_schema=%q", name, field, inputShape)
		}
	}
}

func mcpToolInputFieldNamesFromShapeTestV0(shape string) []string {
	inside, ok := mcpTextBetweenFirstBracesTestV0(shape)
	if !ok {
		return nil
	}
	out := []string{}
	for _, token := range mcpSplitTopLevelTestV0(inside, ',') {
		out = append(out, mcpToolInputNamesFromTokenTestV0(token)...)
	}
	return out
}

func mcpTextBetweenFirstBracesTestV0(text string) (string, bool) {
	start := strings.Index(text, "{")
	if start < 0 {
		return "", false
	}
	depth := 0
	for idx := start; idx < len(text); idx++ {
		switch text[idx] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return text[start+1 : idx], true
			}
		}
	}
	return "", false
}

func mcpSplitTopLevelTestV0(text string, sep rune) []string {
	parts := []string{}
	depth := 0
	start := 0
	for idx, char := range text {
		switch char {
		case '{', '(', '[':
			depth++
		case '}', ')', ']':
			if depth > 0 {
				depth--
			}
		default:
			if char == sep && depth == 0 {
				parts = append(parts, text[start:idx])
				start = idx + len(string(char))
			}
		}
	}
	parts = append(parts, text[start:])
	return parts
}

func mcpToolInputNamesFromTokenTestV0(token string) []string {
	namePart := strings.TrimSpace(token)
	if idx := strings.Index(namePart, ":"); idx >= 0 {
		namePart = namePart[:idx]
	}
	if idx := strings.Index(namePart, "["); idx >= 0 {
		namePart = namePart[:idx]
	}
	names := []string{}
	for _, piece := range strings.Split(namePart, "|") {
		piece = strings.TrimSuffix(strings.TrimSpace(piece), "?")
		if piece == "" || strings.ContainsAny(piece, "{}()") {
			continue
		}
		names = append(names, piece)
	}
	return names
}
