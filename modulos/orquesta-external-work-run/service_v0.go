package orquestaexternalworkrun

import (
	"context"
	"reflect"
	"strings"
	"time"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
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

func enqueueExternalWorkRunV0(
	ctx context.Context,
	request StartExternalWorkRunRequestV0,
	writer orquestarunqueue.RunQueuePriorityWriterPortV0,
) error {
	occurredAt, err := time.Parse(time.RFC3339, request.OccurredAt)
	if err != nil {
		return err
	}
	_, err = writer.SetRunPriorityV0(ctx, orquestarunqueue.RunQueuePriorityCommandV0{
		RunRef:         request.RunRef,
		QueueRef:       request.QueueRef,
		AppRef:         request.AppChangeRequest.AppRef,
		PriorityScore:  request.PriorityScore,
		UpdatedAt:      occurredAt,
		RequestedBy:    request.RequestedBy,
		Reason:         "external_work_run_created",
		IdempotencyKey: "idem-external-work-run-queue-" + request.RunRef,
		EvidenceRefs: []string{
			"evidence-ref-external-work-run-queued",
			request.AppChangeRequest.ChangeRef,
		},
	})
	return err
}

func requestExternalWorkAppChangeOnceV0(
	ctx context.Context,
	request StartExternalWorkRunRequestV0,
	ports orquestaappchange.AppChangePortsV0,
) (orquestaappchange.AppChangeResultV0, error) {
	if source, ok := ports.Store.(orquestaappchange.AppChangeRecordSourcePortV0); ok {
		result, found, err := existingExternalWorkAppChangeResultV0(ctx, request, source)
		if err != nil || found {
			return result, err
		}
	}
	return orquestaappchange.RequestAppChangeV0(ctx, request.AppChangeRequest, ports)
}

func existingExternalWorkAppChangeResultV0(
	ctx context.Context,
	request StartExternalWorkRunRequestV0,
	source orquestaappchange.AppChangeRecordSourcePortV0,
) (orquestaappchange.AppChangeResultV0, bool, error) {
	records, err := source.ListAppChangeRecordsV0(
		ctx,
		orquestaappchange.AppChangeRecordFilterV0{RunRef: request.RunRef},
	)
	if err != nil {
		return orquestaappchange.AppChangeResultV0{}, false, err
	}
	for _, record := range records {
		if strings.TrimSpace(record.Request.ChangeRef) != request.AppChangeRequest.ChangeRef {
			continue
		}
		if !reflect.DeepEqual(
			orquestaappchange.PrepareAppChangeRequestV0(record.Request),
			request.AppChangeRequest,
		) {
			return orquestaappchange.AppChangeResultV0{
				SchemaVersion: orquestaappchange.AppChangeResultSchemaV0,
				Status:        orquestaappchange.AppChangeStatusInvalidV0,
				RequestID:     request.RequestID,
				CorrelationID: request.CorrelationID,
				RunRef:        request.RunRef,
				AppRef:        request.AppChangeRequest.AppRef,
				ChangeRef:     request.AppChangeRequest.ChangeRef,
				Issues: []orquestaappchange.AppChangeIssueV0{{
					Code:  ErrExternalWorkRunExistingChangeConflictV0,
					Field: "change_ref",
				}},
			}, true, nil
		}
		return orquestaappchange.AppChangeResultV0{
			SchemaVersion:       orquestaappchange.AppChangeResultSchemaV0,
			Status:              orquestaappchange.AppChangeStatusAcceptedV0,
			RequestID:           request.AppChangeRequest.RequestID,
			CorrelationID:       request.AppChangeRequest.CorrelationID,
			RunRef:              request.RunRef,
			AppRef:              request.AppChangeRequest.AppRef,
			ChangeRef:           request.AppChangeRequest.ChangeRef,
			DirectorQuestionRef: "question-ref-app-change-" + request.AppChangeRequest.ChangeRef,
			EvidenceRefs:        []string{"evidence-ref-external-work-app-change-existing"},
		}, true, nil
	}
	return orquestaappchange.AppChangeResultV0{}, false, nil
}
