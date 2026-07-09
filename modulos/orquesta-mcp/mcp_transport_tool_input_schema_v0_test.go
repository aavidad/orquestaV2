package orquestamcp

import (
	"strings"
	"testing"

	channel "orquesta/modulos/orquesta-operator-director-channel"
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

func TestMCPInternalContractSurfaceInventoryV0CubreCamposCanonicos(t *testing.T) {
	transport := newFakeMCPTransportV0()
	if err := RegisterMCPTransportV0(transport, MCPTransportBindingsV0{}); err != nil {
		t.Fatalf("register transport: %v", err)
	}
	inventory := mcpInternalContractSurfaceInventoryForTestV0()
	known := map[string]mcpContractSurfaceInventoryEntryForTestV0{}
	for _, entry := range inventory {
		known[entry.ToolName] = entry
	}
	for name := range transport.tools {
		if mcpInternalToolRequiresInventoryForTestV0(name) {
			if _, ok := known[name]; !ok {
				t.Fatalf("tool interna sin inventario de contrato: %s", name)
			}
		}
	}
	for _, entry := range inventory {
		tool, ok := transport.tools[entry.ToolName]
		if !ok {
			t.Fatalf("tool inventariada no registrada: %+v", entry)
		}
		fields, ok := MCPTransportToolInputFieldsV0(entry.ToolName)
		if !ok || len(fields) == 0 {
			t.Fatalf("tool inventariada sin DTO canonico: %+v", entry)
		}
		announced := map[string]bool{}
		for _, field := range mcpToolInputFieldNamesFromShapeTestV0(tool.InputShape) {
			announced[field] = true
		}
		for _, field := range fields {
			if !announced[field.Name] {
				t.Fatalf("contrato %s tool %s no anuncia campo canonico %q en input_schema=%q",
					entry.ContractRef, entry.ToolName, field.Name, tool.InputShape)
			}
		}
	}
}

func TestMCPInternalToolRequiresInventoryV0CubreFamiliasE3PorPrefijo(t *testing.T) {
	for _, name := range []string{
		"orquesta.nueva_app.future.v0",
		"orquesta.director.human_work.future.v0",
		"orquesta.operator.director.future.v0",
		MCPAutoprogrammingPrepareRunToolNameV0,
		MCPAutoprogrammingStatusToolNameV0,
	} {
		if !mcpInternalToolRequiresInventoryForTestV0(name) {
			t.Fatalf("tool interna E3 sin exigencia de inventario: %s", name)
		}
	}
	if mcpInternalToolRequiresInventoryForTestV0(MCPAutoprogrammingObserveGoalToolNameV0) {
		t.Fatalf("observe_goal no pertenece al inventario E3 actual")
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

type mcpContractSurfaceInventoryEntryForTestV0 struct {
	ContractRef string
	ToolName    string
}

func mcpInternalContractSurfaceInventoryForTestV0() []mcpContractSurfaceInventoryEntryForTestV0 {
	return []mcpContractSurfaceInventoryEntryForTestV0{
		{ContractRef: "nueva_app.solicitar.v0", ToolName: MCPNuevaAppToolNameV0},
		{ContractRef: "nueva_app.wizard.v0", ToolName: MCPNuevaAppWizardToolNameV0},
		{ContractRef: "nueva_app.wizard_bot.v0", ToolName: MCPNuevaAppWizardBotToolNameV0},
		{ContractRef: "autoprogramming.prepare_run.v0", ToolName: MCPAutoprogrammingPrepareRunToolNameV0},
		{ContractRef: "autoprogramming.status.v0", ToolName: MCPAutoprogrammingStatusToolNameV0},
		{ContractRef: "operator_director.review_plan.v0", ToolName: MCPHumanDirectorWorkReviewPlanToolNameV0},
		{ContractRef: "operator_director.message.v0", ToolName: channel.OperatorDirectorMessageToolNameV0},
	}
}

func mcpInternalToolRequiresInventoryForTestV0(name string) bool {
	name = strings.TrimSpace(name)
	for _, prefix := range []string{
		"orquesta.nueva_app.",
		"orquesta.director.human_work.",
		"orquesta.operator.director.",
	} {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	switch name {
	case
		MCPAutoprogrammingPrepareRunToolNameV0,
		MCPAutoprogrammingStatusToolNameV0:
		return true
	default:
		return false
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
