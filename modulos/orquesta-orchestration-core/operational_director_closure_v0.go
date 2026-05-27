package orquestacionnucleoapp

import (
	"context"
	"fmt"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

type OperationalDirectorClosureV0 struct {
	RunStore                  RunStorePortV0
	EventSink                 EventSinkPortV0
	EventReader               RunEventReaderPortV0
	TaskStore                 WorkflowTaskStorePortV0
	RequiredTestEvidenceStore RequiredTestEvidenceReaderPortV0
	RequestedBy               string
}

type OperationalDirectorClosureRequestV0 struct {
	RunRef                   string
	TaskID                   string
	DeliveryRef              string
	AcceptedReviewRef        string
	ValidationRef            string
	ClosureRef               string
	OccurredAt               string
	CorrelationID            string
	RequestedBy              string
	Summary                  string
	RequiredTestEvidenceRefs []string
	EvidenceRefs             []string
}

type OperationalDirectorClosureResultV0 struct {
	Run         orquestacoreworkflow.OrchestrationRunV0
	Commands    []orquestacoreworkflow.OrchestrationCommandV0
	EventsCount int
	Issues      []ErrorV0
}

func (closer OperationalDirectorClosureV0) CloseOperationalDirectorRunV0(
	ctx context.Context,
	request OperationalDirectorClosureRequestV0,
) (OperationalDirectorClosureResultV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	request = normalizeOperationalDirectorClosureRequestV0(request)
	if request.RequestedBy == "" {
		request.RequestedBy = strings.TrimSpace(closer.RequestedBy)
	}
	if issues := closer.validateOperationalDirectorClosureV0(request); len(issues) > 0 {
		return OperationalDirectorClosureResultV0{Issues: issues}, nil
	}
	run, err := closer.RunStore.LoadRunV0(ctx, request.RunRef)
	if err != nil {
		return OperationalDirectorClosureResultV0{}, err
	}
	result := OperationalDirectorClosureResultV0{Run: run}
	if run.Status == orquestacoreworkflow.OrchestrationRunStatusClosedV0 {
		result.Issues = append(result.Issues, operationalDirectorClosedRunRequestIssuesV0(run, request)...)
		return result, nil
	}
	tasks, issues, err := closer.operationalDirectorClosureTasksV0(ctx, run, request)
	if err != nil || len(issues) > 0 {
		result.Issues = append(result.Issues, issues...)
		return result, err
	}
	if issues := operationalDirectorClosureReadinessIssuesV0(run, tasks, request); len(issues) > 0 {
		result.Issues = append(result.Issues, issues...)
		return result, nil
	}
	events, issues, err := closer.operationalDirectorClosureEventsV0(ctx, request)
	if err != nil || len(issues) > 0 {
		result.Issues = append(result.Issues, issues...)
		return result, err
	}
	if issues := operationalDirectorClosureCausalIssuesV0(events, tasks, request); len(issues) > 0 {
		result.Issues = append(result.Issues, issues...)
		return result, nil
	}
	trace := operationalDirectorClosureTraceFromEventsV0(events)
	if issues, err := closer.operationalDirectorClosureRequiredTestIssuesV0(ctx, tasks, trace, request); err != nil || len(issues) > 0 {
		result.Issues = append(result.Issues, issues...)
		return result, err
	}
	if !operationalDirectorClosureReflectedV0(run.ClosedTasks, request.TaskID) {
		if operationalDirectorClosurePhaseV0(run.CurrentPhase) != orquestacoreworkflow.OrchestrationPhaseRevisionV0 {
			if err := closer.applyOperationalDirectorClosureCommandV0(ctx, request, &result, operationalDirectorOpenRevisionForCloseTaskCommandV0); err != nil {
				return result, err
			}
			run = result.Run
		}
		if err := closer.applyOperationalDirectorClosureCommandV0(ctx, request, &result, operationalDirectorCloseTaskCommandV0); err != nil {
			return result, err
		}
	}
	run = result.Run
	if openTasks := operationalDirectorClosureOpenTasksV0(run); len(openTasks) > 0 {
		result.Issues = append(result.Issues, errorV0(
			ErrNucleoOrquestacionInvalidoV0,
			"run.open_tasks",
			fmt.Sprintf("microtareas abiertas: %s", strings.Join(openTasks, ",")),
		))
		return result, nil
	}
	if !operationalDirectorClosureReflectedV0(run.Validations, request.ValidationRef) {
		currentPhase := operationalDirectorClosurePhaseV0(run.CurrentPhase)
		if currentPhase != orquestacoreworkflow.OrchestrationPhaseValidacionFinalV0 &&
			currentPhase != orquestacoreworkflow.OrchestrationPhaseCierreV0 {
			if err := closer.applyOperationalDirectorClosureCommandV0(ctx, request, &result, operationalDirectorOpenFinalValidationCommandV0); err != nil {
				return result, err
			}
		}
		if err := closer.applyOperationalDirectorClosureCommandV0(ctx, request, &result, operationalDirectorRegisterFinalValidationCommandV0); err != nil {
			return result, err
		}
	}
	run = result.Run
	if operationalDirectorClosurePhaseV0(run.CurrentPhase) != orquestacoreworkflow.OrchestrationPhaseCierreV0 {
		if err := closer.applyOperationalDirectorClosureCommandV0(ctx, request, &result, operationalDirectorOpenClosureCommandV0); err != nil {
			return result, err
		}
	}
	if !operationalDirectorClosureReflectedV0(result.Run.Closures, request.ClosureRef) {
		if err := closer.applyOperationalDirectorClosureCommandV0(ctx, request, &result, operationalDirectorCloseRunCommandV0); err != nil {
			return result, err
		}
	}
	return result, nil
}

func (closer OperationalDirectorClosureV0) applyOperationalDirectorClosureCommandV0(
	ctx context.Context,
	request OperationalDirectorClosureRequestV0,
	result *OperationalDirectorClosureResultV0,
	build operationalDirectorClosureCommandBuilderV0,
) error {
	command, err := build(request)
	if err != nil {
		return err
	}
	commandResult, err := HandleStoredWorkflowCommandV0(ctx, closer.RunStore, closer.EventSink, command)
	if err != nil {
		return err
	}
	run, err := closer.RunStore.LoadRunV0(ctx, request.RunRef)
	if err != nil {
		return err
	}
	result.Run = run
	result.Commands = append(result.Commands, command)
	result.EventsCount += len(commandResult.Events)
	return nil
}

func (closer OperationalDirectorClosureV0) operationalDirectorClosureTasksV0(
	ctx context.Context,
	run orquestacoreworkflow.OrchestrationRunV0,
	request OperationalDirectorClosureRequestV0,
) ([]orquestacoreworkflow.WorkflowTaskV0, []ErrorV0, error) {
	if closer.TaskStore == nil {
		return nil, []ErrorV0{errorV0(ErrNucleoOrquestacionInvalidoV0, "task_store", "task_store requerido")}, nil
	}
	tasks, err := closer.TaskStore.LoadWorkflowTasksV0(ctx, run.RunID, run.Tasks)
	if err != nil {
		return nil, nil, err
	}
	if !operationalDirectorClosureTaskLoadedV0(tasks, request.TaskID) {
		return tasks, []ErrorV0{errorV0(ErrNucleoOrquestacionInvalidoV0, "task_id", "microtarea no encontrada")}, nil
	}
	return tasks, nil, nil
}

func (closer OperationalDirectorClosureV0) operationalDirectorClosureEventsV0(
	ctx context.Context,
	request OperationalDirectorClosureRequestV0,
) ([]orquestacoreworkflow.OrchestrationEventV0, []ErrorV0, error) {
	reader := closer.EventReader
	if reader == nil {
		reader, _ = closer.EventSink.(RunEventReaderPortV0)
	}
	if reader == nil {
		return nil, []ErrorV0{errorV0(ErrNucleoOrquestacionInvalidoV0, "event_reader", "event_reader requerido")}, nil
	}
	events, err := loadRunEventsWithBudgetV0(ctx, reader, request.RunRef)
	if err != nil {
		return nil, nil, err
	}
	if len(events) == 0 {
		return nil, []ErrorV0{errorV0(ErrNucleoOrquestacionInvalidoV0, "events", "historial de eventos requerido")}, nil
	}
	return events, nil, nil
}

func operationalDirectorClosureReadinessIssuesV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	tasks []orquestacoreworkflow.WorkflowTaskV0,
	request OperationalDirectorClosureRequestV0,
) []ErrorV0 {
	issues := make([]ErrorV0, 0)
	if !operationalDirectorClosureReflectedV0(run.Tasks, request.TaskID) {
		issues = append(issues, errorV0(ErrNucleoOrquestacionInvalidoV0, "task_id", "microtarea no pertenece al run"))
	}
	if !operationalDirectorClosureReflectedV0(run.Deliveries, request.DeliveryRef) {
		issues = append(issues, errorV0(ErrNucleoOrquestacionInvalidoV0, "delivery_ref", "entrega no registrada"))
	}
	if !operationalDirectorClosureReflectedV0(run.AcceptedReviews, request.AcceptedReviewRef) {
		issues = append(issues, errorV0(ErrNucleoOrquestacionInvalidoV0, "accepted_review_ref", "review aceptada no registrada"))
	}
	return issues
}
