package main

import (
	"encoding/json"
	"testing"

	orquestamcp "orquesta/modulos/orquesta-mcp"
)

// Smoke real del programa de autonomia por el transporte: se guarda un programa
// con dependencias, se pide su frontera y solo salen los nodos LISTOS. El avance
// se persiste con CAS, asi que dos planificadores no se pisan.
func TestMCPAutonomyProgramGuardaYCalculaLaFronteraV0(t *testing.T) {
	stack := buildCanonicalMCPBootstrapStackForTestV0(t)
	handler, err := buildServerAppHandlerV0(stack)
	if err != nil {
		t.Fatalf("buildServerAppHandlerV0: %v", err)
	}
	server := newLocalHTTPServerForTestV0(t, handler)
	t.Cleanup(server.Close)

	// b depende de a: al calcular la frontera solo puede lanzarse a.
	programa := map[string]any{
		"program_ref": "prog-1",
		"project_ref": "proj-1",
		"root_ref":    "root-1",
		"status":      "active",
		"nodes": []map[string]any{
			{
				"node_ref": "a", "goal_ref": "goal-a", "status": "pending",
				"write_set": []string{"docs/a.md"}, "required_tests": []string{"go test ./a"},
			},
			{
				"node_ref": "b", "goal_ref": "goal-b", "status": "pending",
				"depends_on": []string{"a"},
				"write_set":  []string{"docs/b.md"}, "required_tests": []string{"go test ./b"},
			},
		},
	}

	guardado := llamarAutonomyV0(t, server.URL, map[string]any{
		"action": orquestamcp.MCPAutonomyProgramActionSaveV0, "program": programa,
	})
	if guardado.Estado != orquestamcp.MCPAutonomyProgramEstadoOKV0 {
		t.Fatalf("save: estado=%q errores=%+v", guardado.Estado, guardado.ErroresPublicos)
	}
	if guardado.NodeCount != 2 {
		t.Fatalf("el programa no conservo sus dos nodos: %+v", guardado)
	}

	frontera := llamarAutonomyV0(t, server.URL, map[string]any{
		"action":      orquestamcp.MCPAutonomyProgramActionFrontierV0,
		"project_ref": "proj-1", "root_ref": "root-1", "program_ref": "prog-1",
	})
	if frontera.Estado != orquestamcp.MCPAutonomyProgramEstadoOKV0 {
		t.Fatalf("frontier: estado=%q errores=%+v", frontera.Estado, frontera.ErroresPublicos)
	}
	// Solo 'a' esta listo: 'b' depende de el. Si la frontera lanzara los dos,
	// estariamos ejecutando trabajo cuyas dependencias no existen.
	if len(frontera.LaunchRefs) != 1 {
		t.Fatalf("la frontera lanzo %d nodos, solo 'a' esta listo: %+v", len(frontera.LaunchRefs), frontera.LaunchRefs)
	}

	// Observar no altera nada y devuelve el avance persistido.
	observado := llamarAutonomyV0(t, server.URL, map[string]any{
		"action":      orquestamcp.MCPAutonomyProgramActionObserveV0,
		"project_ref": "proj-1", "root_ref": "root-1", "program_ref": "prog-1",
	})
	if observado.NodeCount != 2 {
		t.Fatalf("observe perdio nodos: %+v", observado)
	}
}

func llamarAutonomyV0(t *testing.T, baseURL string, arguments map[string]any) orquestamcp.MCPAutonomyProgramToolResultV0 {
	t.Helper()
	var raw json.RawMessage
	if err := callMCPJSONRPCBootstrapV0(baseURL, "tools/call", map[string]any{
		"name":      orquestamcp.MCPAutonomyProgramToolNameV0,
		"arguments": arguments,
	}, &raw); err != nil {
		t.Fatalf("tools/call: %v", err)
	}
	if reason := bootstrapMissingPortReasonV0(string(raw)); reason != "" {
		t.Fatalf("puerto sin cablear (%s): %s", reason, string(raw))
	}
	var envelope struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil || len(envelope.Content) != 1 {
		t.Fatalf("envelope inesperado: %v raw=%s", err, string(raw))
	}
	var result orquestamcp.MCPAutonomyProgramToolResultV0
	if err := json.Unmarshal([]byte(envelope.Content[0].Text), &result); err != nil {
		t.Fatalf("decodificando payload: %v text=%s", err, envelope.Content[0].Text)
	}
	return result
}
