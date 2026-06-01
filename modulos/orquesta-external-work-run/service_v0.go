package orquestaexternalworkrun

import (
	"context"
	"strings"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func StartExternalWorkRunV0(
	ctx context.Context,
	request StartExternalWorkRunRequestV0,
	ports StartExternalWorkRunPortsV0,
	config StartExternalWorkRunConfigV0,
) (StartExternalWorkRunResultV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	request = normalizeStartExternalWorkRunRequestV0(request, config)
	if issues := validateStartExternalWorkRunRequestV0(request, ports); len(issues) > 0 {
		return invalidStartExternalWorkRunResultV0(request, issues), nil
	}
	existingRun, runExists, err := loadExistingExternalWorkRunV0(ctx, request, ports)
	if err != nil {
		return StartExternalWorkRunResultV0{}, err
	}
	if runExists {
		if issues := validateExistingExternalWorkRunV0(existingRun, request); len(issues) > 0 {
			return invalidStartExternalWorkRunResultV0(request, issues), nil
		}
	}

	evidenceRefs, err := ensureOperationalRunV0(ctx, request, ports)
	if err != nil {
		return StartExternalWorkRunResultV0{}, err
	}
	if err := enqueueExternalWorkRunV0(ctx, request, ports.RunQueue); err != nil {
		return StartExternalWorkRunResultV0{}, err
	}
	changeResult, err := requestExternalWorkAppChangeOnceV0(ctx, request, ports.AppChange)
	if err != nil {
		return StartExternalWorkRunResultV0{}, err
	}
	if changeResult.Status != orquestaappchange.AppChangeStatusAcceptedV0 {
		return invalidStartExternalWorkRunResultV0(request, appChangeResultIssuesV0(changeResult)), nil
	}
	return acceptedStartExternalWorkRunResultV0(
		request,
		changeResult,
		append(evidenceRefs, "evidence-ref-external-work-run-queued"),
	), nil
}

func loadExistingExternalWorkRunV0(
	ctx context.Context,
	request StartExternalWorkRunRequestV0,
	ports StartExternalWorkRunPortsV0,
) (orquestacoreworkflow.OrchestrationRunV0, bool, error) {
	run, err := ports.RunStore.LoadRunV0(ctx, request.RunRef)
	if err == nil {
		return run, true, nil
	}
	if orquestacionnucleoapp.IsRunNotFoundErrorV0(err) {
		return orquestacoreworkflow.OrchestrationRunV0{}, false, nil
	}
	return orquestacoreworkflow.OrchestrationRunV0{}, false, err
}

func validateExistingExternalWorkRunV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	request StartExternalWorkRunRequestV0,
) []ExternalWorkRunIssueV0 {
	if strings.TrimSpace(run.RunID) != request.RunRef ||
		strings.TrimSpace(run.ProjectRef) != request.ProjectRef ||
		strings.TrimSpace(run.AppSpecRef) != request.AppSpecRef {
		return []ExternalWorkRunIssueV0{
			externalWorkRunIssueV0(ErrExternalWorkRunExistingRunConflictV0, "run_ref"),
		}
	}
	return nil
}

func ensureOperationalRunV0(
	ctx context.Context,
	request StartExternalWorkRunRequestV0,
	ports StartExternalWorkRunPortsV0,
) ([]string, error) {
	run, err := ports.RunStore.LoadRunV0(ctx, request.RunRef)
	if err != nil && !orquestacionnucleoapp.IsRunNotFoundErrorV0(err) {
		return nil, err
	}
	evidenceRefs := []string{}
	if orquestacionnucleoapp.IsRunNotFoundErrorV0(err) {
		var startRefs []string
		run, startRefs, err = createExternalWorkRunV0(ctx, request, ports)
		if err != nil {
			return nil, err
		}
		evidenceRefs = append(evidenceRefs, startRefs...)
	}
	run, phaseRefs, err := ensureExternalWorkProgrammingPhaseV0(ctx, run, request, ports)
	if err != nil {
		return nil, err
	}
	_ = run
	return append(evidenceRefs, phaseRefs...), nil
}

func createExternalWorkRunV0(
	ctx context.Context,
	request StartExternalWorkRunRequestV0,
	ports StartExternalWorkRunPortsV0,
) (orquestacoreworkflow.OrchestrationRunV0, []string, error) {
	command, err := orquestacoreworkflow.NewStartRunCommandV0(
		externalWorkRunCommandMetaV0(request, "start"),
		orquestacoreworkflow.StartRunCommandPayloadV0{
			ProjectRef: request.ProjectRef,
			AppSpecRef: request.AppSpecRef,
		},
	)
	if err != nil {
		return orquestacoreworkflow.OrchestrationRunV0{}, nil, err
	}
	run, events, err := applyExternalWorkRunCommandV0(
		orquestacoreworkflow.OrchestrationRunV0{},
		command,
	)
	if err != nil {
		return orquestacoreworkflow.OrchestrationRunV0{}, nil, err
	}
	if err := persistExternalWorkRunEventsV0(ctx, ports, run, events); err != nil {
		return orquestacoreworkflow.OrchestrationRunV0{}, nil, err
	}
	return run, []string{"evidence-ref-external-work-run-started"}, nil
}

func ensureExternalWorkProgrammingPhaseV0(
	ctx context.Context,
	run orquestacoreworkflow.OrchestrationRunV0,
	request StartExternalWorkRunRequestV0,
	ports StartExternalWorkRunPortsV0,
) (orquestacoreworkflow.OrchestrationRunV0, []string, error) {
	if run.CurrentPhase == orquestacoreworkflow.OrchestrationPhaseProgramacionV0 {
		return run, nil, nil
	}
	command, err := orquestacoreworkflow.NewOpenPhaseCommandV0(
		externalWorkRunCommandMetaV0(request, "open-programacion"),
		orquestacoreworkflow.OpenPhaseCommandPayloadV0{
			PhaseID: string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			Reason:  "trabajo externo ya especificado",
		},
	)
	if err != nil {
		return run, nil, err
	}
	next, events, err := applyExternalWorkRunCommandV0(run, command)
	if err != nil {
		return run, nil, err
	}
	if err := persistExternalWorkRunEventsV0(ctx, ports, next, events); err != nil {
		return run, nil, err
	}
	return next, []string{"evidence-ref-external-work-run-programacion"}, nil
}

func applyExternalWorkRunCommandV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	command orquestacoreworkflow.OrchestrationCommandV0,
) (orquestacoreworkflow.OrchestrationRunV0, []orquestacoreworkflow.OrchestrationEventV0, error) {
	result, err := orquestacoreworkflow.HandleCommandV0(run, command)
	if err != nil {
		return run, nil, err
	}
	next := run
	for _, event := range result.Events {
		next, err = orquestacoreworkflow.ApplyEventV0(next, event)
		if err != nil {
			return run, nil, err
		}
	}
	return next, result.Events, nil
}

func persistExternalWorkRunEventsV0(
	ctx context.Context,
	ports StartExternalWorkRunPortsV0,
	run orquestacoreworkflow.OrchestrationRunV0,
	events []orquestacoreworkflow.OrchestrationEventV0,
) error {
	if len(events) > 0 {
		if err := ports.EventSink.AppendRunEventsV0(ctx, run.RunID, events); err != nil {
			return err
		}
	}
	return ports.RunStore.SaveRunV0(ctx, run)
}

func externalWorkRunCommandMetaV0(
	request StartExternalWorkRunRequestV0,
	action string,
) orquestacoreworkflow.OrchestrationCommandMetaV0 {
	key := compactExternalWorkRunRefV0(action + "-" + request.RunRef)
	return orquestacoreworkflow.OrchestrationCommandMetaV0{
		CommandID:      "cmd-external-work-" + key,
		RunID:          request.RunRef,
		IdempotencyKey: "idem-external-work-" + key,
		CorrelationID:  request.CorrelationID,
		RequestedBy:    request.RequestedBy,
		OccurredAt:     request.OccurredAt,
	}
}
