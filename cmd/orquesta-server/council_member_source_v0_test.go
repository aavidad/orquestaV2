package main

import (
	"context"
	"errors"
	"testing"

	orquestaappcodexstack "orquesta/modulos/orquesta-app-codex-stack"
)

type usoFalsoV0 struct {
	metricas []orquestaappcodexstack.CodexStackAgentUsageMetricV0
	err      error
}

func (fake usoFalsoV0) BuildCodexStackAgentUsageMetricsV0(
	context.Context, orquestaappcodexstack.CodexStackAgentUsageMetricsRequestV0,
) ([]orquestaappcodexstack.CodexStackAgentUsageMetricV0, error) {
	return fake.metricas, fake.err
}

func declaradosV0() []serverProjectConfigCouncilMemberV0 {
	return []serverProjectConfigCouncilMemberV0{
		{MemberRef: "agente-a", FamilyRef: "familia-a", CapabilityRank: 3},
		{MemberRef: "agente-b", FamilyRef: "familia-b", CapabilityRank: 5},
		{MemberRef: "agente-c", FamilyRef: "familia-b", CapabilityRank: 1},
	}
}

// El presupuesto se OBSERVA, no se declara. Si se declarara, cualquiera podria
// fabricar el reparto de roles escribiendo un numero en un fichero.
func TestCouncilMemberSourceObservaLaCuotaRealV0(t *testing.T) {
	source := newCouncilMemberSourceV0(declaradosV0(), usoFalsoV0{
		metricas: []orquestaappcodexstack.CodexStackAgentUsageMetricV0{
			{AgentRequestID: "agente-a", QuotaStatus: "available", QuotaRemaining: 800, QuotaLimit: 1000},
			{AgentRequestID: "agente-b", QuotaStatus: "limited", QuotaRemaining: 500, QuotaLimit: 1000},
			{AgentRequestID: "agente-c", QuotaStatus: "exhausted", QuotaRemaining: 40, QuotaLimit: 1000},
		},
	})

	miembros, err := source.ObserveCouncilMembersV0(context.Background())
	if err != nil {
		t.Fatalf("ObserveCouncilMembersV0: %v", err)
	}
	if len(miembros) != 3 {
		t.Fatalf("se observaron %d miembros de 3", len(miembros))
	}
	porRef := map[string]float64{}
	for _, miembro := range miembros {
		porRef[miembro.MemberRef] = miembro.BudgetRemaining
	}
	if porRef["agente-a"] != 0.8 || porRef["agente-b"] != 0.5 || porRef["agente-c"] != 0.04 {
		t.Fatalf("la cuota real no se tradujo a presupuesto: %+v", porRef)
	}
}

// Sin cuota observable NO se inventa un numero. El reparto en caliente se apoya en
// la cuota real; sin ella no hay reparto en caliente, hay adivinacion.
func TestCouncilMemberSourceFallaCerradoSinCuotaObservableV0(t *testing.T) {
	casos := map[string]usoFalsoV0{
		"sin metricas":        {metricas: nil},
		"error del proveedor": {err: errors.New("el proveedor se cayo")},
		"cuota no reportada": {metricas: []orquestaappcodexstack.CodexStackAgentUsageMetricV0{
			{AgentRequestID: "agente-a", QuotaStatus: "unknown"},
			{AgentRequestID: "agente-b", QuotaStatus: "not_configured"},
			{AgentRequestID: "agente-c", QuotaStatus: "available", QuotaRemaining: 10, QuotaLimit: 100},
		}},
	}
	for nombre, uso := range casos {
		source := newCouncilMemberSourceV0(declaradosV0(), uso)
		if _, err := source.ObserveCouncilMembersV0(context.Background()); !errors.Is(err, ErrCouncilBudgetUnobservableV0) {
			t.Fatalf("%s: deberia fallar cerrado, no inventar presupuestos: %v", nombre, err)
		}
	}

	sinProveedor := newCouncilMemberSourceV0(declaradosV0(), nil)
	if _, err := sinProveedor.ObserveCouncilMembersV0(context.Background()); !errors.Is(err, ErrCouncilBudgetUnobservableV0) {
		t.Fatal("sin proveedor de metricas no se puede repartir por presupuesto")
	}
}
