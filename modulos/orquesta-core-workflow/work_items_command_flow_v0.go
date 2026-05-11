package orquestacoreworkflow

import "strings"

func validateCreateMicrotaskFunctionRefsRequiredV0(task WorkflowTaskV0) error {
	if len(task.FunctionContractRefs) == 0 {
		return commandErrorV0(ErrPayloadInvalidoV0, "payload.task.function_contract_refs")
	}
	for _, ref := range task.FunctionContractRefs {
		if strings.TrimSpace(ref.ContractRef) == "" {
			return commandErrorV0(ErrPayloadInvalidoV0, "payload.task.function_contract_refs")
		}
	}
	return nil
}

func ensureCreateMicrotaskCommandPlanningCurrentV0(current OrchestrationRunV0) error {
	if !createMicrotaskPlanningPhaseCurrentV0(current) {
		return commandErrorV0(ErrTransicionInvalidaV0, "phase.status")
	}
	return nil
}

func ensureCreateMicrotaskEventPlanningCurrentV0(current OrchestrationRunV0) error {
	if !createMicrotaskPlanningPhaseCurrentV0(current) {
		return eventErrorV0(ErrSecuenciaInvalidaV0, "phase.status")
	}
	return nil
}

func createMicrotaskPlanningPhaseCurrentV0(current OrchestrationRunV0) bool {
	phase := normalizePhaseIDV0(current.CurrentPhase)
	if phase != OrchestrationPhasePlanificacionMicrotareasV0 &&
		phase != OrchestrationPhaseProgramacionV0 {
		return false
	}
	return phaseIsCurrentAndActiveV0(current, phase)
}

func ensureCreateMicrotaskCommandContractsReadyV0(current OrchestrationRunV0, task WorkflowTaskV0) error {
	if !functionContractRefsAlreadyPublishedV0(current, explicitFunctionContractRefsV0(task.FunctionContractRefs)) {
		return commandErrorV0(ErrTransicionInvalidaV0, "payload.task.function_contract_refs")
	}
	return nil
}

func ensureMicrotaskEventContractsReadyV0(current OrchestrationRunV0, refs []string) error {
	if !functionContractRefsAlreadyPublishedV0(current, compactStringsV0(refs)) {
		return eventErrorV0(ErrSecuenciaInvalidaV0, "payload.function_contract_refs")
	}
	return nil
}

func functionContractRefsAlreadyPublishedV0(current OrchestrationRunV0, refs []string) bool {
	if len(refs) == 0 {
		return false
	}
	for _, ref := range refs {
		if !functionContractAlreadyReflectedV0(current, ref) {
			return false
		}
	}
	return true
}

func explicitFunctionContractRefsV0(refs []WorkflowFunctionContractRefV0) []string {
	result := make([]string, 0, len(refs))
	for _, ref := range refs {
		result = appendUniqueCompactRefV0(result, ref.ContractRef)
	}
	return result
}
