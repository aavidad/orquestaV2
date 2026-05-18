package orquestacionnucleoapp

import (
	"context"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
)

type OperationalDirectorPlanMaterializerV0 struct {
	RunStore    RunStorePortV0
	EventSink   EventSinkPortV0
	TaskWriter  WorkflowTaskWriterPortV0
	RequestedBy string
}

type OperationalDirectorPlanMaterializeRequestV0 struct {
	Plan                 orquestadirectoroperativo.OperationalDirectorPlanV0
	FunctionContractRefs []orquestacoreworkflow.WorkflowFunctionContractRefV0
	TargetPhaseID        orquestacoreworkflow.OrchestrationPhaseIDV0
	OccurredAt           string
	CorrelationID        string
	MaxItems             int
}

type OperationalDirectorPlanMaterializeResultV0 struct {
	Tasks       []orquestacoreworkflow.WorkflowTaskV0
	Commands    []orquestacoreworkflow.OrchestrationCommandV0
	EventsCount int
	Issues      []ErrorV0
}

func (materializer OperationalDirectorPlanMaterializerV0) MaterializeOperationalDirectorPlanV0(
	ctx context.Context,
	request OperationalDirectorPlanMaterializeRequestV0,
) (OperationalDirectorPlanMaterializeResultV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	request = normalizeOperationalDirectorPlanMaterializeRequestV0(request)
	if issues := materializer.validateOperationalDirectorPlanMaterializerV0(request); len(issues) > 0 {
		return OperationalDirectorPlanMaterializeResultV0{Issues: issues}, nil
	}
	work := orquestadirectoroperativo.BuildOperationalDirectorWaveWorkV0(request.Plan)
	if len(work.Issues) > 0 {
		return OperationalDirectorPlanMaterializeResultV0{
			Issues: operationalDirectorMaterializerIssuesFromPlanV0(work.Issues),
		}, nil
	}
	items := operationalDirectorMaterializableItemsV0(work, request.MaxItems)
	if len(items) == 0 {
		return OperationalDirectorPlanMaterializeResultV0{
			Issues: []ErrorV0{errorV0(
				ErrNucleoOrquestacionInvalidoV0,
				"operational_director_plan.items",
				"plan sin items materializables",
			)},
		}, nil
	}
	result := OperationalDirectorPlanMaterializeResultV0{
		Tasks:    make([]orquestacoreworkflow.WorkflowTaskV0, 0, len(items)),
		Commands: make([]orquestacoreworkflow.OrchestrationCommandV0, 0, len(items)),
	}
	itemTaskRefs := operationalDirectorMaterializedTaskRefsV0(request.Plan, items)
	itemWaveRefs := operationalDirectorMaterializedWaveRefsV0(work)
	for _, item := range items {
		task, err := operationalDirectorWorkflowTaskFromItemV0(request, item, itemTaskRefs, itemWaveRefs)
		if err != nil {
			result.Issues = append(result.Issues, errorV0(
				ErrNucleoOrquestacionInvalidoV0,
				"operational_director_plan.item",
				err.Error(),
			))
			return result, nil
		}
		command, err := operationalDirectorCreateMicrotaskCommandV0(materializer, request, task)
		if err != nil {
			result.Issues = append(result.Issues, errorV0(
				ErrNucleoOrquestacionInvalidoV0,
				"operational_director_plan.command",
				err.Error(),
			))
			return result, nil
		}
		if err := materializer.preflightOperationalDirectorCreateMicrotaskCommandV0(ctx, command); err != nil {
			return result, err
		}
		if err := materializer.TaskWriter.SaveWorkflowTaskV0(ctx, task); err != nil {
			return result, err
		}
		commandResult, err := HandleStoredWorkflowCommandV0(
			ctx,
			materializer.RunStore,
			materializer.EventSink,
			command,
		)
		if err != nil {
			return result, err
		}
		result.Tasks = append(result.Tasks, task)
		result.Commands = append(result.Commands, command)
		result.EventsCount += len(commandResult.Events)
	}
	return result, nil
}

func (materializer OperationalDirectorPlanMaterializerV0) preflightOperationalDirectorCreateMicrotaskCommandV0(
	ctx context.Context,
	command orquestacoreworkflow.OrchestrationCommandV0,
) error {
	run, err := materializer.RunStore.LoadRunV0(ctx, command.RunID)
	if err != nil {
		return err
	}
	_, err = orquestacoreworkflow.HandleCommandV0(run, command)
	return err
}

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

func operationalDirectorWorkflowTaskFromItemV0(
	request OperationalDirectorPlanMaterializeRequestV0,
	item orquestadirectoroperativo.OperationalDirectorWorkItemV0,
	itemTaskRefs map[string]string,
	itemWaveRefs map[string]string,
) (orquestacoreworkflow.WorkflowTaskV0, error) {
	criteria := operationalDirectorWorkflowTaskCriteriaV0(request, item, itemTaskRefs, itemWaveRefs)
	waveRef := strings.TrimSpace(itemWaveRefs[item.ItemID])
	task := orquestacoreworkflow.WorkflowTaskV0{
		SchemaVersion:        orquestacoreworkflow.WorkflowTaskSchemaVersionV0,
		TaskID:               itemTaskRefs[item.ItemID],
		RunID:                request.Plan.RunRef,
		PhaseID:              request.TargetPhaseID,
		Title:                item.Title,
		Summary:              "Materializar item del Director Operativo.",
		WriteSet:             append([]string(nil), item.WriteSet...),
		AcceptanceCriteria:   criteria,
		RequiredTests:        append([]string(nil), item.RequiredTests...),
		DependsOn:            operationalDirectorWorkflowTaskDependsOnV0(item.DependsOn, itemTaskRefs),
		ParentTaskRef:        strings.TrimSpace(itemTaskRefs[item.ParentItemID]),
		CohortRef:            operationalDirectorCohortRefV0(request.Plan, waveRef),
		WaveRef:              waveRef,
		DelegationDepth:      item.DelegationDepth,
		MaxChildAgents:       item.MaxChildItems,
		ChildTaskRefs:        operationalDirectorWorkflowTaskChildRefsV0(item.ChildItemIDs, itemTaskRefs),
		FunctionContractRefs: append([]orquestacoreworkflow.WorkflowFunctionContractRefV0(nil), request.FunctionContractRefs...),
	}
	return orquestacoreworkflow.NewWorkflowTaskV0(task)
}

func operationalDirectorWorkflowTaskCriteriaV0(
	request OperationalDirectorPlanMaterializeRequestV0,
	item orquestadirectoroperativo.OperationalDirectorWorkItemV0,
	itemTaskRefs map[string]string,
	itemWaveRefs map[string]string,
) []string {
	criteria := []string{"objetivo_actual: " + request.Plan.Objective}
	criteria = operationalDirectorAppendCriterionV0(criteria, "operational_director.plan_ref", request.Plan.PlanRef)
	criteria = operationalDirectorAppendCriterionV0(criteria, "operational_director.request_ref", request.Plan.RequestRef)
	criteria = operationalDirectorAppendCriterionV0(criteria, "operational_director.wave_ref", itemWaveRefs[item.ItemID])
	criteria = operationalDirectorAppendCriterionV0(criteria, "operational_director.source_step_ref", item.SourceStepID)
	criteria = operationalDirectorAppendCriterionV0(criteria, "operational_director.source_item_ref", item.ItemID)
	criteria = operationalDirectorAppendCriterionV0(criteria, "operational_director.parent_item_ref", item.ParentItemID)
	criteria = operationalDirectorAppendCriterionV0(criteria, "operational_director.parent_task_ref", itemTaskRefs[item.ParentItemID])
	criteria = operationalDirectorAppendCriterionV0(criteria, "operational_director.child_item_refs", strings.Join(item.ChildItemIDs, ","))
	criteria = append(criteria, item.AcceptanceCriteria...)
	return criteria
}

func operationalDirectorCohortRefV0(
	plan orquestadirectoroperativo.OperationalDirectorPlanV0,
	waveRef string,
) string {
	waveRef = strings.TrimSpace(waveRef)
	if waveRef == "" {
		return ""
	}
	return "cohort-operational-director-" +
		operationalDirectorSafeRefPartV0(plan.RequestRef) +
		"-" +
		operationalDirectorSafeRefPartV0(waveRef)
}

func operationalDirectorAppendCriterionV0(criteria []string, key string, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return criteria
	}
	return append(criteria, key+": "+value)
}

func operationalDirectorWorkflowTaskDependsOnV0(
	itemDependsOn []string,
	itemTaskRefs map[string]string,
) []string {
	out := make([]string, 0, len(itemDependsOn))
	for _, itemRef := range itemDependsOn {
		if taskRef := strings.TrimSpace(itemTaskRefs[itemRef]); taskRef != "" {
			out = append(out, taskRef)
		}
	}
	return out
}

func operationalDirectorWorkflowTaskChildRefsV0(
	itemChildIDs []string,
	itemTaskRefs map[string]string,
) []string {
	out := make([]string, 0, len(itemChildIDs))
	for _, itemRef := range itemChildIDs {
		if taskRef := strings.TrimSpace(itemTaskRefs[itemRef]); taskRef != "" {
			out = append(out, taskRef)
		}
	}
	return out
}

func operationalDirectorCreateMicrotaskCommandV0(
	materializer OperationalDirectorPlanMaterializerV0,
	request OperationalDirectorPlanMaterializeRequestV0,
	task orquestacoreworkflow.WorkflowTaskV0,
) (orquestacoreworkflow.OrchestrationCommandV0, error) {
	suffix := operationalDirectorSafeRefPartV0(task.TaskID)
	return orquestacoreworkflow.NewCreateMicrotaskCommandV0(
		orquestacoreworkflow.OrchestrationCommandMetaV0{
			CommandID:      "cmd-operational-director-create-" + suffix,
			RunID:          request.Plan.RunRef,
			IdempotencyKey: "idem-operational-director-create-" + suffix,
			CorrelationID:  request.CorrelationID,
			RequestedBy:    operationalDirectorRequestedByV0(materializer.RequestedBy),
			OccurredAt:     request.OccurredAt,
		},
		orquestacoreworkflow.CreateMicrotaskCommandPayloadV0{Task: task},
	)
}

func operationalDirectorTaskRefV0(
	plan orquestadirectoroperativo.OperationalDirectorPlanV0,
	item orquestadirectoroperativo.OperationalDirectorWorkItemV0,
) string {
	return "task-operational-director-" +
		operationalDirectorSafeRefPartV0(plan.RequestRef) +
		"-" +
		operationalDirectorSafeRefPartV0(item.SourceStepID)
}

func operationalDirectorSafeRefPartV0(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "\\", "-")
	value = strings.ReplaceAll(value, "/", "-")
	value = strings.ReplaceAll(value, " ", "-")
	return value
}

func operationalDirectorRequestedByV0(value string) string {
	value = strings.TrimSpace(value)
	if value != "" {
		return value
	}
	return "orquesta-operational-director-materializer"
}
