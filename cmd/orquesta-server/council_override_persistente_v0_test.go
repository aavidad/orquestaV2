package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	orquestamcp "orquesta/modulos/orquesta-mcp"
)

func miembrosConsejoParaTestV0() []map[string]any {
	return []map[string]any{
		{"member_ref": "autor", "family_ref": "familia-a", "budget_remaining": 0.90, "capability_rank": 2},
		{"member_ref": "sobrado", "family_ref": "familia-a", "budget_remaining": 0.80, "capability_rank": 3},
		{"member_ref": "justito", "family_ref": "familia-b", "budget_remaining": 0.04, "capability_rank": 1},
		{"member_ref": "medio", "family_ref": "familia-b", "budget_remaining": 0.50, "capability_rank": 5},
	}
}

// El operador pidio DOS alcances para el override: la peticion concreta y un
// ajuste persistente que valga para todos los consejos hasta que lo cambie.
// Precedencia: peticion > persistente > automatico.
func TestCouncilOverridePersistenteYSuPrecedenciaV0(t *testing.T) {
	stack := stackConConsejoInyectableParaTestV0(t, buildCanonicalMCPBootstrapStackForTestV0(t))
	handler, err := buildServerAppHandlerV0(stack)
	if err != nil {
		t.Fatalf("buildServerAppHandlerV0: %v", err)
	}
	// El ejecutor lee el fichero EN CADA LLAMADA: un ajuste persistente nuevo
	// entra sin reiniciar el servidor.
	escribirOverridesPersistentesV0(t, os.Getenv(envServerStateDirV0), []orquestamcp.MCPCouncilOverrideV0{{
		Role: "revisor", MemberRef: "medio", ForcedBy: "operador", Reason: "ajuste persistente",
	}})
	server := newLocalHTTPServerForTestV0(t, handler)
	t.Cleanup(server.Close)

	// Sin override en la peticion: manda el persistente, no el automatico.
	persistente := llamarCouncilV0(t, server.URL, map[string]any{
		"action": orquestamcp.MCPCouncilActionAssignV0, "council_ref": "c1",
		"author_ref": "autor", "members": miembrosConsejoParaTestV0(),
	})
	revisor := asientoMCPV0(t, persistente, "revisor")
	if revisor.MemberRef != "medio" {
		t.Fatalf("el override persistente no se aplico: revisor = %s", revisor.MemberRef)
	}
	if !revisor.Forced || revisor.AutomaticMemberRef != "sobrado" {
		t.Fatalf("el persistente no dejo evidencia auditable: %+v", revisor)
	}

	// Con override en la peticion: pisa al persistente.
	deLaPeticion := llamarCouncilV0(t, server.URL, map[string]any{
		"action": orquestamcp.MCPCouncilActionAssignV0, "council_ref": "c2",
		"author_ref": "autor", "members": miembrosConsejoParaTestV0(),
		"overrides": []map[string]any{{
			"role": "revisor", "member_ref": "sobrado", "forced_by": "operador", "reason": "solo por hoy",
		}},
	})
	revisorPeticion := asientoMCPV0(t, deLaPeticion, "revisor")
	if revisorPeticion.MemberRef != "sobrado" {
		t.Fatalf("el override de la peticion debe pisar al persistente: %s", revisorPeticion.MemberRef)
	}
}

// La superficie publica devuelve CODIGOS TIPADOS, nunca el texto interno del
// error: volcar err.Error() al exterior es fuga de detalle.
func TestCouncilNoFiltraTextoInternoEnLaSuperficieV0(t *testing.T) {
	stack := stackConConsejoInyectableParaTestV0(t, buildCanonicalMCPBootstrapStackForTestV0(t))
	handler, err := buildServerAppHandlerV0(stack)
	if err != nil {
		t.Fatalf("buildServerAppHandlerV0: %v", err)
	}
	server := newLocalHTTPServerForTestV0(t, handler)
	t.Cleanup(server.Close)

	invalido := llamarCouncilV0(t, server.URL, map[string]any{
		"action": orquestamcp.MCPCouncilActionAssignV0, "council_ref": "c3",
		"author_ref": "autor", "members": miembrosConsejoParaTestV0(),
		"overrides": []map[string]any{{"role": "revisor", "member_ref": "autor", "forced_by": "operador"}},
	})
	if invalido.Estado != orquestamcp.MCPCouncilEstadoErrorV0 {
		t.Fatalf("forzar al autor como revisor debe fallar: %+v", invalido)
	}
	if len(invalido.ErroresPublicos) == 0 {
		t.Fatal("el error no trae codigo publico tipado")
	}
	if invalido.ErroresPublicos[0].Code != "council_autor_no_se_revisa_a_si_mismo" {
		t.Fatalf("codigo publico inesperado: %q", invalido.ErroresPublicos[0].Code)
	}
	// El dominio adorna sus errores con detalle interno ("no puede ocupar el rol
	// ... de su propia entrega"). Nada de eso puede asomar por la superficie.
	if strings.Contains(invalido.Rationale, "no puede ocupar el rol") {
		t.Fatalf("la superficie filtro texto interno del error: %q", invalido.Rationale)
	}
}

func escribirOverridesPersistentesV0(t *testing.T, stateDir string, overrides []orquestamcp.MCPCouncilOverrideV0) {
	t.Helper()
	if err := os.MkdirAll(stateDir, 0o700); err != nil {
		t.Fatalf("creando state dir: %v", err)
	}
	bytes, err := json.Marshal(councilPersistentOverridesV0{
		SchemaVersion: "orquesta_council_overrides.v0",
		Overrides:     overrides,
	})
	if err != nil {
		t.Fatalf("serializando overrides: %v", err)
	}
	if err := os.WriteFile(filepath.Join(stateDir, councilOverridesFileNameV0), bytes, 0o600); err != nil {
		t.Fatalf("escribiendo overrides persistentes: %v", err)
	}
}
