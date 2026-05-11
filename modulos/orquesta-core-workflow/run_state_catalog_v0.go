package orquestacoreworkflow

func OrchestrationPhaseCatalogV0() []OrchestrationPhaseV0 {
	result := make([]OrchestrationPhaseV0, 0, len(orchestrationPhaseCatalogV0))
	for _, phase := range orchestrationPhaseCatalogV0 {
		result = append(result, cloneOrchestrationPhaseV0(phase))
	}
	return result
}

func SupportedOrchestrationPhaseIDsV0() []OrchestrationPhaseIDV0 {
	phases := OrchestrationPhaseCatalogV0()
	result := make([]OrchestrationPhaseIDV0, 0, len(phases))
	for _, phase := range phases {
		result = append(result, phase.ID)
	}
	return result
}

func IsSupportedOrchestrationPhaseV0(phase OrchestrationPhaseIDV0) bool {
	for _, supported := range orchestrationPhaseCatalogV0 {
		if supported.ID == normalizePhaseIDV0(phase) {
			return true
		}
	}
	return false
}

func ValidateOrchestrationPhaseIDV0(phase OrchestrationPhaseIDV0) error {
	if !IsSupportedOrchestrationPhaseV0(phase) {
		return OrchestrationValidationIssueV0{Code: OrchestrationFaseNoSoportadaV0, Field: "phase"}
	}
	return nil
}

var orchestrationPhaseCatalogV0 = []OrchestrationPhaseV0{
	catalogPhaseV0(OrchestrationPhaseDescubrimientoV0, OrchestrationCapacityMediumV0, "solicitud inicial disponible", "alcance inicial definido", "app_spec_ref"),
	catalogPhaseV0(OrchestrationPhaseBrainstormingArquitecturaV0, OrchestrationCapacityHighV0, "alcance inicial definido", "opciones de arquitectura documentadas", "brainstorm_ref"),
	catalogPhaseV0(OrchestrationPhaseVotacionYDecisionV0, OrchestrationCapacityXHighV0, "opciones de arquitectura documentadas", "decision aceptada", "decision_ref"),
	catalogPhaseV0(OrchestrationPhasePlanificacionMicrotareasV0, OrchestrationCapacityHighV0, "decision aceptada", "microtareas acotadas", "plan_ref"),
	catalogPhaseV0(OrchestrationPhaseProgramacionV0, OrchestrationCapacityMediumV0, "microtareas acotadas", "entregas registradas", "delivery_refs"),
	catalogPhaseV0(OrchestrationPhaseDocumentacionV0, OrchestrationCapacityMediumV0, "entregas registradas", "documentacion actualizada", "docs_ref"),
	catalogPhaseV0(OrchestrationPhaseIntegracionV0, OrchestrationCapacityHighV0, "entregas y documentacion disponibles", "integracion preparada", "integration_ref"),
	catalogPhaseV0(OrchestrationPhaseRevisionV0, OrchestrationCapacityHighV0, "integracion preparada", "revision aceptada", "review_ref"),
	catalogPhaseV0(OrchestrationPhaseValidacionFinalV0, OrchestrationCapacityXHighV0, "revision aceptada", "validacion final registrada", "validation_ref"),
	catalogPhaseV0(OrchestrationPhaseCierreV0, OrchestrationCapacityMediumV0, "validacion final registrada", "run cerrado", "closure_ref"),
}

func catalogPhaseV0(id OrchestrationPhaseIDV0, capacity OrchestrationCapacityRecommendationV0, entry string, exit string, evidence string) OrchestrationPhaseV0 {
	return OrchestrationPhaseV0{
		ID:                  id,
		Status:              OrchestrationPhaseStatusPendingV0,
		EntryCriteria:       []string{entry},
		ExitCriteria:        []string{exit},
		EvidenceRequired:    []string{evidence},
		RecommendedCapacity: capacity,
	}
}
