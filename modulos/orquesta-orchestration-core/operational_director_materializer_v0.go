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
	itemTaskRefs, err := materializer.resolveOperationalDirectorMaterializedTaskRefsV0(ctx, request, items, itemTaskRefs, itemWaveRefs)
	if err != nil {
		return result, err
	}
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
