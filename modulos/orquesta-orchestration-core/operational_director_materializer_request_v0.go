package orquestacionnucleoapp

import (
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
)

func normalizeOperationalDirectorPlanMaterializeRequestV0(
	request OperationalDirectorPlanMaterializeRequestV0,
) OperationalDirectorPlanMaterializeRequestV0 {
	request.OccurredAt = strings.TrimSpace(request.OccurredAt)
	request.CorrelationID = strings.TrimSpace(request.CorrelationID)
	request.TargetPhaseID = operationalDirectorTargetPhaseIDV0(request.TargetPhaseID)
	request.FunctionContractRefs = normalizeOperationalDirectorFunctionRefsV0(request.FunctionContractRefs)
	if request.MaxItems < 0 {
		request.MaxItems = 0
	}
	if request.Plan.MaxParallelAgents > 0 &&
		(request.MaxItems == 0 || request.MaxItems > request.Plan.MaxParallelAgents) {
		request.MaxItems = request.Plan.MaxParallelAgents
	}
	return request
}

func operationalDirectorTargetPhaseIDV0(
	phase orquestacoreworkflow.OrchestrationPhaseIDV0,
) orquestacoreworkflow.OrchestrationPhaseIDV0 {
	phase = orquestacoreworkflow.OrchestrationPhaseIDV0(strings.TrimSpace(string(phase)))
	if phase != "" {
		return phase
	}
	return orquestacoreworkflow.OrchestrationPhaseProgramacionV0
}

func normalizeOperationalDirectorFunctionRefsV0(
	refs []orquestacoreworkflow.WorkflowFunctionContractRefV0,
) []orquestacoreworkflow.WorkflowFunctionContractRefV0 {
	out := make([]orquestacoreworkflow.WorkflowFunctionContractRefV0, 0, len(refs))
	for _, ref := range refs {
		compact := orquestacoreworkflow.WorkflowFunctionContractRefV0{
			ContractRef:  strings.TrimSpace(ref.ContractRef),
			FunctionName: strings.TrimSpace(ref.FunctionName),
		}
		if compact.ContractRef == "" && compact.FunctionName == "" {
			continue
		}
		out = append(out, compact)
	}
	return out
}

func (materializer OperationalDirectorPlanMaterializerV0) validateOperationalDirectorPlanMaterializerV0(
	request OperationalDirectorPlanMaterializeRequestV0,
) []ErrorV0 {
	issues := make([]ErrorV0, 0)
	if materializer.RunStore == nil {
		issues = append(issues, errorV0(ErrNucleoOrquestacionInvalidoV0, "run_store", "run_store requerido"))
	}
	if materializer.TaskWriter == nil {
		issues = append(issues, errorV0(ErrNucleoOrquestacionInvalidoV0, "workflow_task_writer", "workflow_task_writer requerido"))
	}
	if strings.TrimSpace(request.Plan.RunRef) == "" {
		issues = append(issues, errorV0(ErrNucleoOrquestacionInvalidoV0, "plan.run_ref", "run_ref requerido"))
	}
	if request.Plan.Status != orquestadirectoroperativo.OperationalDirectorPlanReadyV0 {
		issues = append(issues, errorV0(ErrNucleoOrquestacionInvalidoV0, "plan.status", "plan no listo para lanzar"))
	}
	if len(request.FunctionContractRefs) == 0 {
		issues = append(issues, errorV0(ErrNucleoOrquestacionInvalidoV0, "function_contract_refs", "contrato funcional requerido"))
	}
	if strings.TrimSpace(request.OccurredAt) == "" {
		issues = append(issues, errorV0(ErrNucleoOrquestacionInvalidoV0, "occurred_at", "occurred_at requerido"))
	}
	if err := orquestacoreworkflow.ValidateOrchestrationPhaseIDV0(request.TargetPhaseID); err != nil {
		issues = append(issues, errorV0(ErrNucleoOrquestacionInvalidoV0, "target_phase_id", err.Error()))
	}
	return issues
}

func operationalDirectorMaterializerIssuesFromPlanV0(
	issues []orquestadirectoroperativo.OperationalDirectorIssueV0,
) []ErrorV0 {
	out := make([]ErrorV0, 0, len(issues))
	for _, issue := range issues {
		out = append(out, errorV0(issue.Code, issue.Field, issue.Message))
	}
	return out
}

func operationalDirectorMaterializableItemsV0(
	work orquestadirectoroperativo.OperationalDirectorWaveWorkV0,
	maxItems int,
) []orquestadirectoroperativo.OperationalDirectorWorkItemV0 {
	if !work.ReadyToLaunch {
		return nil
	}
	items := make([]orquestadirectoroperativo.OperationalDirectorWorkItemV0, 0)
	for _, wave := range work.Waves {
		for _, item := range wave.Items {
			if item.Kind != orquestadirectoroperativo.OperationalDirectorStepLaunchSubagentsV0 {
				continue
			}
			items = append(items, item)
			if maxItems > 0 && len(items) >= maxItems {
				return items
			}
		}
	}
	return items
}

func operationalDirectorMaterializedTaskRefsV0(
	plan orquestadirectoroperativo.OperationalDirectorPlanV0,
	items []orquestadirectoroperativo.OperationalDirectorWorkItemV0,
) map[string]string {
	refs := make(map[string]string, len(items))
	for _, item := range items {
		refs[item.ItemID] = operationalDirectorTaskRefV0(plan, item)
	}
	return refs
}

func operationalDirectorMaterializedWaveRefsV0(
	work orquestadirectoroperativo.OperationalDirectorWaveWorkV0,
) map[string]string {
	refs := make(map[string]string)
	for _, wave := range work.Waves {
		for _, item := range wave.Items {
			if strings.TrimSpace(item.ItemID) == "" || strings.TrimSpace(wave.WaveID) == "" {
				continue
			}
			refs[item.ItemID] = wave.WaveID
		}
	}
	return refs
}
