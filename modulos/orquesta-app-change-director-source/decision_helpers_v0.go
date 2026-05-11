package orquestaappchangedirectorsource

import (
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoragent "orquesta/modulos/orquesta-director-agent"
)

func appChangeDecisionV0(
	runRef string,
	phase orquestacoreworkflow.OrchestrationPhaseIDV0,
	action string,
	commandType string,
	refs appChangeRefSetV0,
) orquestadirectoragent.DirectorAgentDecisionV0 {
	return orquestadirectoragent.DirectorAgentDecisionV0{
		SchemaVersion: orquestadirectoragent.DirectorAgentDecisionSchemaVersionV0,
		DecisionRef:   appChangeDecisionRefV0(action, refs),
		RunID:         runRef,
		PhaseID:       string(phase),
		CommandType:   commandType,
		CommandRef:    appChangeCommandRefV0(action, refs),
		Summary:       appChangeSummaryForActionV0(action),
		EvidenceRefs:  []string{refs.EvidenceRef},
	}
}

func appChangeOpenPhaseV0(
	runRef string,
	current orquestacoreworkflow.OrchestrationPhaseIDV0,
	next orquestacoreworkflow.OrchestrationPhaseIDV0,
	action string,
	refs appChangeRefSetV0,
) orquestadirectoragent.DirectorAgentDecisionV0 {
	decision := appChangeDecisionV0(
		runRef,
		current,
		action,
		orquestadirectoragent.DirectorAgentCommandOpenPhaseV0,
		refs,
	)
	decision.OpenPhase = &orquestadirectoragent.DirectorAgentOpenPhaseCommandV0{
		PhaseID: string(next),
		Reason:  "Avanzar replanificacion del cambio.",
	}
	return decision
}

func appChangeSummaryForActionV0(action string) string {
	switch action {
	case "answer":
		return "Responder cambio solicitado."
	case "vote":
		return "Solicitar votacion de replanificacion."
	case "accept":
		return "Aceptar replanificacion compacta."
	case "contract":
		return "Publicar contrato funcional del cambio."
	case "task":
		return "Crear microtarea del cambio."
	default:
		return "Avanzar replanificacion del cambio."
	}
}
