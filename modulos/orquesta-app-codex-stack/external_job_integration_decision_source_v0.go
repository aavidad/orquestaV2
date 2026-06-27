package orquestaappcodexstack

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
	orquestaappchangedirectorsource "orquesta/modulos/orquesta-app-change-director-source"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoragent "orquesta/modulos/orquesta-director-agent"
	orquestadirectoragentworkflow "orquesta/modulos/orquesta-director-agent-workflow"
	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

const (
	externalJobIntegrationTaskSuffixV0 = "-integration-product"
	externalJobIntegrationReasonV0     = "product_not_consolidated_due_write_set_narrowing"
)

type ExternalJobIntegrationDecisionSourceV0 struct {
	AppChangeStore orquestaappchange.AppChangeRecordSourcePortV0
	TaskStore      orquestacionnucleoapp.WorkflowTaskStorePortV0
}

var _ orquestadirectoragentworkflow.DirectorAgentDecisionSourcePortV0 = ExternalJobIntegrationDecisionSourceV0{}

func (source ExternalJobIntegrationDecisionSourceV0) ListDirectorAgentDecisionsV0(
	ctx context.Context,
	request orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0,
) ([]orquestadirectoragent.DirectorAgentDecisionV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	run := request.Run
	if source.AppChangeStore == nil || source.TaskStore == nil || strings.TrimSpace(run.RunID) == "" {
		return nil, nil
	}
	if run.CurrentPhase != orquestacoreworkflow.OrchestrationPhaseProgramacionV0 {
		return nil, nil
	}
	records, err := source.AppChangeStore.ListAppChangeRecordsV0(
		ctx,
		orquestaappchange.AppChangeRecordFilterV0{RunRef: run.RunID},
	)
	if err != nil {
		return nil, err
	}
	decisions := make([]orquestadirectoragent.DirectorAgentDecisionV0, 0, len(records))
	for _, record := range records {
		decision, ok, err := source.integrationDecisionForRecordV0(ctx, run, record)
		if err != nil {
			return nil, err
		}
		if ok {
			decisions = append(decisions, decision)
		}
	}
	return decisions, nil
}

func (source ExternalJobIntegrationDecisionSourceV0) integrationDecisionForRecordV0(
	ctx context.Context,
	run orquestacoreworkflow.OrchestrationRunV0,
	record orquestaappchange.AppChangeRecordV0,
) (orquestadirectoragent.DirectorAgentDecisionV0, bool, error) {
	if !externalJobIntegrationRecordEligibleV0(record) {
		return orquestadirectoragent.DirectorAgentDecisionV0{}, false, nil
	}
	parentRef := orquestaappchangedirectorsource.AppChangeTaskRefV0(record.Request.ChangeRef)
	if parentRef == "" || !codexStackStringInSetV0(run.Tasks, parentRef) {
		return orquestadirectoragent.DirectorAgentDecisionV0{}, false, nil
	}
	integrationRef := externalJobIntegrationTaskRefV0(parentRef)
	if externalJobIntegrationTaskAlreadyKnownV0(run, integrationRef) {
		return orquestadirectoragent.DirectorAgentDecisionV0{}, false, nil
	}
	tasks, err := source.TaskStore.LoadWorkflowTasksV0(ctx, run.RunID, []string{parentRef})
	if err != nil || len(tasks) != 1 {
		return orquestadirectoragent.DirectorAgentDecisionV0{}, false, err
	}
	parent := tasks[0]
	childRefs := compactCodexStackStringsV0(parent.ChildTaskRefs)
	if len(childRefs) == 0 ||
		!workflowTaskHasDomainWorkContractV0(parent) ||
		!externalJobParentWriteSetOnlyCoordinationV0(parent.WriteSet) {
		return orquestadirectoragent.DirectorAgentDecisionV0{}, false, nil
	}
	contractRef := externalJobIntegrationContractRefV0(run)
	if contractRef == "" {
		return orquestadirectoragent.DirectorAgentDecisionV0{}, false, nil
	}
	writeSet := externalJobIntegrationWriteSetV0(record.Request.AllowedWriteSet)
	if len(writeSet) == 0 {
		return orquestadirectoragent.DirectorAgentDecisionV0{}, false, nil
	}
	task := externalJobIntegrationMicrotaskV0(run, record, parent, childRefs, integrationRef, contractRef, writeSet)
	return externalJobIntegrationDecisionV0(run, record, task), true, nil
}

func externalJobIntegrationRecordEligibleV0(record orquestaappchange.AppChangeRecordV0) bool {
	request := record.Request
	return request.ExternalWork != nil && len(request.AllowedWriteSet) > 0
}

func externalJobIntegrationExternalWorkLooksOPESV0(
	appRef string,
	work *orquestaappchange.AppChangeExternalWorkV0,
) bool {
	if work == nil {
		return false
	}
	if strings.TrimSpace(appRef) == "opes" ||
		strings.TrimSpace(work.ProjectRef) == "opes" ||
		strings.TrimSpace(work.ProjectRef) == "project-ref-opes" {
		return true
	}
	for _, ref := range append(append([]string{}, work.InterfaceRefs...), work.WorkRefs...) {
		normalized := strings.ToLower(strings.TrimSpace(ref))
		if normalized == "opes" || strings.HasPrefix(normalized, "opes-") || strings.HasPrefix(normalized, "opes.") {
			return true
		}
	}
	return false
}

func externalJobIntegrationSubrolesRequiredCountV0(
	fields []orquestadomainwork.DomainWorkFieldV0,
) int {
	for _, field := range fields {
		if strings.TrimSpace(field.Name) != "subroles_required" {
			continue
		}
		if len(field.Values) > 0 {
			return len(field.Values)
		}
		if count, err := strconv.Atoi(strings.TrimSpace(string(field.ValueJSON))); err == nil {
			return count
		}
		if strings.EqualFold(strings.TrimSpace(string(field.ValueJSON)), "true") {
			return 6
		}
		if count, err := strconv.Atoi(strings.TrimSpace(field.Value)); err == nil {
			return count
		}
		if strings.EqualFold(strings.TrimSpace(field.Value), "true") {
			return 6
		}
	}
	return 0
}

func externalJobIntegrationTaskAlreadyKnownV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	taskRef string,
) bool {
	return codexStackStringInSetV0(run.Tasks, taskRef) ||
		codexStackStringInSetV0(run.DeliveredTasks, taskRef) ||
		codexStackStringInSetV0(run.ClosedTasks, taskRef)
}

func externalJobIntegrationContractRefV0(run orquestacoreworkflow.OrchestrationRunV0) string {
	for _, ref := range run.FunctionContracts {
		ref = strings.TrimSpace(ref)
		if strings.HasPrefix(ref, "contract:function:app-change:") {
			return ref
		}
	}
	for _, ref := range run.FunctionContracts {
		if strings.TrimSpace(ref) != "" {
			return strings.TrimSpace(ref)
		}
	}
	return ""
}

func externalJobIntegrationWriteSetV0(allowed []string) []string {
	out := make([]string, 0, len(allowed)*2)
	for _, entry := range allowed {
		entry = strings.Trim(strings.TrimSpace(entry), "/")
		if entry == "" {
			continue
		}
		out = append(out, entry, entry+"/coordinacion")
	}
	return compactCodexStackStringsV0(out)
}

func externalJobIntegrationTaskRefV0(parentRef string) string {
	parentRef = strings.TrimSpace(parentRef)
	if parentRef == "" {
		return ""
	}
	return parentRef + externalJobIntegrationTaskSuffixV0
}

func externalJobIntegrationMicrotaskV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	record orquestaappchange.AppChangeRecordV0,
	parent orquestacoreworkflow.WorkflowTaskV0,
	childRefs []string,
	taskRef string,
	contractRef string,
	writeSet []string,
) orquestadirectoragent.DirectorAgentMicrotaskV0 {
	dependsOn := compactCodexStackStringsV0(append([]string{parent.TaskID}, childRefs...))
	return orquestadirectoragent.DirectorAgentMicrotaskV0{
		SchemaVersion:   orquestadirectoragent.DirectorAgentMicrotaskSchemaVersionV0,
		TaskID:          taskRef,
		RunID:           run.RunID,
		PhaseID:         string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
		WorkProfileKind: string(orquestacoreworkflow.WorkProfileDomainWorkV0),
		Title:           "Integrar producto externo canonico",
		Summary: "Consolidar los artefactos utiles de subtrabajos externos en el producto canonico autorizado; " +
			"no rehacer material valido y dejar bloqueo causal si falta evidencia.",
		WriteSet: writeSet,
		AcceptanceCriteria: []string{
			"Consolidar Markdown canonico, informe de extension o checkpoint bajo el write-set de producto autorizado.",
			"Usar entregas de subtrabajos como insumo y conservar evidencia de reutilizacion, rework o bloqueo.",
			"No marcar el job externo como cerrado si faltan artefactos, pruebas o validaciones exigidas por el contrato externo.",
		},
		RequiredTests: []string{"validar integracion canonica de producto externo"},
		DependsOn:     dependsOn,
		ContextRefs: compactCodexStackStringsV0([]string{
			"external-job-integration-required",
			externalJobIntegrationReasonV0,
			strings.TrimSpace(record.Request.ExternalWork.JobRef),
		}),
		ParentTaskRef:   parent.TaskID,
		CohortRef:       parent.CohortRef,
		WaveRef:         parent.WaveRef,
		DelegationDepth: parent.DelegationDepth + 1,
		MaxChildAgents:  0,
		ChildTaskRefs:   nil,
		FunctionContractRefs: []orquestadirectoragent.DirectorAgentFunctionContractRefV0{{
			ContractRef:  contractRef,
			FunctionName: "ApplyExternalDomainWorkV0",
		}},
	}
}

func externalJobIntegrationDecisionV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	record orquestaappchange.AppChangeRecordV0,
	task orquestadirectoragent.DirectorAgentMicrotaskV0,
) orquestadirectoragent.DirectorAgentDecisionV0 {
	suffix := strings.TrimPrefix(task.TaskID, "task-ref-")
	decisionRef := "director-decision-external-job-integration-" + suffix
	return orquestadirectoragent.DirectorAgentDecisionV0{
		SchemaVersion:   orquestadirectoragent.DirectorAgentDecisionSchemaVersionV0,
		DecisionRef:     decisionRef,
		RunID:           run.RunID,
		PhaseID:         string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
		CommandType:     orquestadirectoragent.DirectorAgentCommandCreateMicrotaskV0,
		CommandRef:      "command-ref-external-job-integration-" + suffix,
		Summary:         fmt.Sprintf("Crear integrador de producto para job externo %s.", strings.TrimSpace(record.Request.ExternalWork.JobRef)),
		EvidenceRefs:    []string{externalJobIntegrationReasonV0, task.ParentTaskRef},
		CreateMicrotask: &orquestadirectoragent.DirectorAgentCreateMicrotaskCommandV0{Task: task},
	}
}
