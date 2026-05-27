package orquestaappcodexstack

import (
	"fmt"
	"strings"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func autoprogrammingBridgeRunV0(
	request AutoprogrammingBridgeRequestV0,
	work orquestaautoprogramming.AutoprogrammingProgrammableWorkV0,
) orquestacoreworkflow.OrchestrationRunV0 {
	phases := orquestacoreworkflow.OrchestrationPhaseCatalogV0()
	for index := range phases {
		if phases[index].ID != orquestacoreworkflow.OrchestrationPhaseProgramacionV0 {
			continue
		}
		phases[index].Status = orquestacoreworkflow.OrchestrationPhaseStatusActiveV0
		phases[index].OpenedAt = request.OccurredAt
	}
	taskRefs := make([]string, 0, len(work.Tasks))
	for _, task := range work.Tasks {
		taskRefs = append(taskRefs, task.TaskID)
	}
	return orquestacoreworkflow.OrchestrationRunV0{
		SchemaVersion:     orquestacoreworkflow.OrchestrationRunSchemaVersionV0,
		RunID:             work.RequestRef,
		ProjectRef:        work.ProjectRef,
		AppSpecRef:        "app-spec-ref-autoprogramming-" + autoprogrammingBridgeHashRefV0(work.RequestRef),
		Status:            orquestacoreworkflow.OrchestrationRunStatusActiveV0,
		CurrentPhase:      orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		Phases:            phases,
		Tasks:             compactStringsV0(taskRefs),
		FunctionContracts: autoprogrammingBridgeFunctionRefsV0(work.Tasks),
	}
}

func autoprogrammingBridgeValidateExistingRunV0(
	existing orquestacoreworkflow.OrchestrationRunV0,
	expected orquestacoreworkflow.OrchestrationRunV0,
) error {
	if strings.TrimSpace(existing.RunID) != strings.TrimSpace(expected.RunID) ||
		strings.TrimSpace(existing.ProjectRef) != strings.TrimSpace(expected.ProjectRef) {
		return fmt.Errorf("autoprogramming run existente incompatible: %s", expected.RunID)
	}
	if !autoprogrammingBridgeCompatibleAppSpecRefV0(existing.AppSpecRef, expected.AppSpecRef) {
		return fmt.Errorf("autoprogramming run existente incompatible: %s", expected.RunID)
	}
	for _, taskRef := range compactStringsV0(expected.Tasks) {
		if !stringInSetV0(existing.Tasks, taskRef) {
			return fmt.Errorf("autoprogramming run existente sin task esperada: %s", taskRef)
		}
	}
	for _, contractRef := range compactStringsV0(expected.FunctionContracts) {
		if !stringInSetV0(existing.FunctionContracts, contractRef) {
			return fmt.Errorf("autoprogramming run existente sin contrato esperado: %s", contractRef)
		}
	}
	return nil
}

func autoprogrammingBridgeCompatibleAppSpecRefV0(existing string, expected string) bool {
	existing = strings.TrimSpace(existing)
	expected = strings.TrimSpace(expected)
	if existing == expected {
		return true
	}
	return strings.HasPrefix(existing, "app-spec-ref-autoprogramming-") &&
		strings.HasPrefix(expected, "app-spec-ref-autoprogramming-")
}

func autoprogrammingBridgeFunctionRefsV0(
	tasks []orquestacoreworkflow.WorkflowTaskV0,
) []string {
	refs := make([]string, 0)
	for _, task := range tasks {
		for _, ref := range task.FunctionContractRefs {
			if strings.TrimSpace(ref.ContractRef) != "" {
				refs = append(refs, ref.ContractRef)
				continue
			}
			refs = append(refs, ref.FunctionName)
		}
	}
	return compactStringsV0(refs)
}

func autoprogrammingBridgeHashRefV0(value string) string {
	return codexStackDeterministicDigestV0(value)
}
