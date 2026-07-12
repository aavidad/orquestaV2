package main

import (
	"encoding/json"
	"os"
	"testing"

	orquestaappcodexstack "orquesta/modulos/orquesta-app-codex-stack"

	orquestamcp "orquesta/modulos/orquesta-mcp"
)

// Smoke real del consejo por el transporte: el operador convoca, ve los roles que
// salieron por presupuesto, FUERZA uno, y el consejo decide. Hasta hoy el consejo
// existia solo en un .md y nunca se habia visto actuar.
func TestMCPCouncilConvocaFuerzaRolYDecidePorTransporteV0(t *testing.T) {
	stack := stackConConsejoInyectableParaTestV0(t, buildCanonicalMCPBootstrapStackForTestV0(t))
	handler, err := buildServerAppHandlerV0(stack)
	if err != nil {
		t.Fatalf("buildServerAppHandlerV0: %v", err)
	}
	server := newLocalHTTPServerForTestV0(t, handler)
	t.Cleanup(server.Close)

	miembros := []map[string]any{
		{"member_ref": "autor", "family_ref": "familia-a", "budget_remaining": 0.90, "capability_rank": 2},
		{"member_ref": "sobrado", "family_ref": "familia-a", "budget_remaining": 0.80, "capability_rank": 3},
		{"member_ref": "justito", "family_ref": "familia-b", "budget_remaining": 0.04, "capability_rank": 1},
		{"member_ref": "medio", "family_ref": "familia-b", "budget_remaining": 0.50, "capability_rank": 5},
	}

	automatico := llamarCouncilV0(t, server.URL, map[string]any{
		"action":      orquestamcp.MCPCouncilActionAssignV0,
		"council_ref": "consejo-mcp-1",
		"author_ref":  "autor",
		"members":     miembros,
	})
	if automatico.Estado != orquestamcp.MCPCouncilEstadoOKV0 {
		t.Fatalf("assign: estado=%q errores=%+v", automatico.Estado, automatico.ErroresPublicos)
	}
	revisor := asientoMCPV0(t, automatico, "revisor")
	if revisor.MemberRef != "sobrado" {
		t.Fatalf("el revisor deberia ser el de mas presupuesto: %s", revisor.MemberRef)
	}
	consultor := asientoMCPV0(t, automatico, "consultor")
	if consultor.Material != "masticado" {
		t.Fatalf("el consultor decide sobre material masticado, no lee diffs: %s", consultor.Material)
	}
	if consultor.BudgetWarning == "" {
		t.Fatal("un consultor al 4% de cuota debe avisar por el transporte")
	}

	// El operador fuerza el rol. Este es el "medio de hacerlo" que pidio.
	forzado := llamarCouncilV0(t, server.URL, map[string]any{
		"action":      orquestamcp.MCPCouncilActionAssignV0,
		"council_ref": "consejo-mcp-2",
		"author_ref":  "autor",
		"members":     miembros,
		"overrides": []map[string]any{{
			"role": "revisor", "member_ref": "medio", "forced_by": "operador", "reason": "lo quiero yo",
		}},
	})
	forzadoRevisor := asientoMCPV0(t, forzado, "revisor")
	if forzadoRevisor.MemberRef != "medio" {
		t.Fatalf("el override del operador no gano: %s", forzadoRevisor.MemberRef)
	}
	if !forzadoRevisor.Forced || forzadoRevisor.AutomaticMemberRef != "sobrado" {
		t.Fatalf("el override no dejo evidencia auditable: %+v", forzadoRevisor)
	}

	// Y el override NO puede saltarse la integridad, tampoco por el transporte.
	invalido := llamarCouncilV0(t, server.URL, map[string]any{
		"action":      orquestamcp.MCPCouncilActionAssignV0,
		"council_ref": "consejo-mcp-3",
		"author_ref":  "autor",
		"members":     miembros,
		"overrides": []map[string]any{{
			"role": "revisor", "member_ref": "autor", "forced_by": "operador",
		}},
	})
	if invalido.Estado != orquestamcp.MCPCouncilEstadoErrorV0 {
		t.Fatalf("el autor no puede revisarse a si mismo ni por MCP: %+v", invalido)
	}

	// El veto de seguridad no lo levanta la mayoria, tampoco por el transporte.
	critico := llamarCouncilV0(t, server.URL, map[string]any{
		"action":            orquestamcp.MCPCouncilActionDecideV0,
		"council_ref":       "consejo-mcp-4",
		"author_ref":        "autor",
		"security_critical": true,
		"members":           miembros,
		"ballots": []map[string]any{
			{"member_ref": "sobrado", "vote": "approve"},
			{"member_ref": "justito", "vote": "approve"},
			{"member_ref": "medio", "vote": "block"},
		},
	})
	if critico.Outcome != "council_decision_blocked" {
		t.Fatalf("el veto de seguridad debe bloquear pese a dos aprobaciones: %+v", critico)
	}
	if critico.Approvals < 2 {
		t.Fatal("el escenario no tenia mayoria que vencer; no probaria nada")
	}
}

func llamarCouncilV0(t *testing.T, baseURL string, arguments map[string]any) orquestamcp.MCPCouncilToolResultV0 {
	t.Helper()
	var raw json.RawMessage
	if err := callMCPJSONRPCBootstrapV0(baseURL, "tools/call", map[string]any{
		"name":      orquestamcp.MCPCouncilToolNameV0,
		"arguments": arguments,
	}, &raw); err != nil {
		t.Fatalf("tools/call %s: %v", orquestamcp.MCPCouncilToolNameV0, err)
	}
	if reason := bootstrapMissingPortReasonV0(string(raw)); reason != "" {
		t.Fatalf("el consejo respondio con puerto sin cablear (%s): %s", reason, string(raw))
	}
	var envelope struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil || len(envelope.Content) != 1 {
		t.Fatalf("envelope inesperado: %v raw=%s", err, string(raw))
	}
	var result orquestamcp.MCPCouncilToolResultV0
	if err := json.Unmarshal([]byte(envelope.Content[0].Text), &result); err != nil {
		t.Fatalf("decodificando payload: %v text=%s", err, envelope.Content[0].Text)
	}
	return result
}

func asientoMCPV0(t *testing.T, result orquestamcp.MCPCouncilToolResultV0, role string) orquestamcp.MCPCouncilSeatV0 {
	t.Helper()
	for _, seat := range result.Seats {
		if seat.Role == role {
			return seat
		}
	}
	t.Fatalf("el consejo no asigno el rol %s: %+v", role, result.Seats)
	return orquestamcp.MCPCouncilSeatV0{}
}

// stackConConsejoInyectableParaTestV0 sustituye el consejo por un ejecutor SIN
// fuente acreditada, para poder inyectar miembros desde el test. En produccion la
// fuente manda siempre y los miembros del input se ignoran: permitir inyeccion
// ahi seria el bypass que Codex encontro.
func stackConConsejoInyectableParaTestV0(
	t *testing.T,
	stack orquestaappcodexstack.StackV0,
) orquestaappcodexstack.StackV0 {
	t.Helper()
	executor, err := newCouncilExecutorV0(os.Getenv(envServerStateDirV0))
	if err != nil {
		t.Fatalf("newCouncilExecutorV0: %v", err)
	}
	stack.MCPTransportBindings.Council = orquestamcp.MCPCouncilToolExecutorV0{
		Council:           executor,
		ClassifyPublicErr: councilPublicErrorClassifierV0,
	}
	return stack
}
