package main

import (
	"context"
	"errors"
	"fmt"
	"strings"

	orquestaappcodexstack "orquesta/modulos/orquesta-app-codex-stack"
	orquestamcp "orquesta/modulos/orquesta-mcp"
)

// ErrCouncilBudgetUnobservableV0 se devuelve cuando no se puede observar la cuota
// de suficientes miembros. NO se inventa un presupuesto: el reparto de roles en
// caliente se apoya en la cuota real, y sin cuota real no hay reparto en caliente,
// hay adivinacion. Falla cerrado; el operador siempre puede forzar con overrides.
var ErrCouncilBudgetUnobservableV0 = errors.New("council_budget_unobservable")

// councilMemberSourceV0 observa a los miembros declarados en la config canonica y
// su cuota REAL a traves del proveedor de metricas de uso.
type councilMemberSourceV0 struct {
	declarados []serverProjectConfigCouncilMemberV0
	usage      orquestaappcodexstack.CodexStackAgentUsageMetricsProviderPortV0
}

var _ orquestamcp.MCPCouncilMemberSourcePortV0 = councilMemberSourceV0{}

func newCouncilMemberSourceV0(
	declarados []serverProjectConfigCouncilMemberV0,
	usage orquestaappcodexstack.CodexStackAgentUsageMetricsProviderPortV0,
) councilMemberSourceV0 {
	return councilMemberSourceV0{declarados: declarados, usage: usage}
}

func (source councilMemberSourceV0) ObserveCouncilMembersV0(
	ctx context.Context,
) ([]orquestamcp.MCPCouncilMemberV0, error) {
	if len(source.declarados) == 0 {
		return nil, fmt.Errorf("%w: no hay miembros declarados", ErrCouncilBudgetUnobservableV0)
	}
	if source.usage == nil {
		return nil, fmt.Errorf("%w: sin proveedor de metricas de uso", ErrCouncilBudgetUnobservableV0)
	}

	refs := make([]string, 0, len(source.declarados))
	for _, miembro := range source.declarados {
		if ref := strings.TrimSpace(miembro.MemberRef); ref != "" {
			refs = append(refs, ref)
		}
	}
	metricas, err := source.usage.BuildCodexStackAgentUsageMetricsV0(
		ctx,
		orquestaappcodexstack.CodexStackAgentUsageMetricsRequestV0{AgentRefs: refs},
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrCouncilBudgetUnobservableV0, err)
	}
	porRef := map[string]orquestaappcodexstack.CodexStackAgentUsageMetricV0{}
	for _, metrica := range metricas {
		porRef[metrica.AgentRequestID] = metrica
	}

	miembros := make([]orquestamcp.MCPCouncilMemberV0, 0, len(source.declarados))
	for _, declarado := range source.declarados {
		metrica, ok := porRef[strings.TrimSpace(declarado.MemberRef)]
		if !ok {
			// Sin observacion no se inventa un numero: ese miembro no entra.
			continue
		}
		budget, observado := presupuestoObservadoV0(metrica)
		if !observado {
			continue
		}
		miembros = append(miembros, orquestamcp.MCPCouncilMemberV0{
			MemberRef:       declarado.MemberRef,
			FamilyRef:       declarado.FamilyRef,
			BudgetRemaining: budget,
			CapabilityRank:  declarado.CapabilityRank,
		})
	}

	// El consejo necesita al menos dos miembros distintos del autor. Si la cuota
	// solo se observa en uno, no hay consejo: hay una opinion.
	if len(miembros) < 3 {
		return nil, fmt.Errorf(
			"%w: solo se observo la cuota de %d de %d miembros",
			ErrCouncilBudgetUnobservableV0, len(miembros), len(source.declarados),
		)
	}
	return miembros, nil
}

// presupuestoObservadoV0 traduce la cuota REAL a la fraccion [0,1] que el consejo
// usa para repartir roles. Solo cuenta si de verdad se observo.
func presupuestoObservadoV0(
	metrica orquestaappcodexstack.CodexStackAgentUsageMetricV0,
) (float64, bool) {
	if metrica.QuotaLimit <= 0 || metrica.QuotaRemaining < 0 {
		return 0, false
	}
	switch metrica.QuotaStatus {
	case "available", "limited", "exhausted":
	default:
		return 0, false
	}
	fraccion := float64(metrica.QuotaRemaining) / float64(metrica.QuotaLimit)
	if fraccion > 1 {
		fraccion = 1
	}
	return fraccion, true
}
