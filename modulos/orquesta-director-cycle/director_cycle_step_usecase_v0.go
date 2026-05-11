package orquestadirectorcycle

import (
	"context"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorcycleoutbox "orquesta/modulos/orquesta-director-cycle-outbox"
	orquestadirectorrunner "orquesta/modulos/orquesta-director-runner"
	orquestadirectorscheduler "orquesta/modulos/orquesta-director-scheduler"
	orquestadirectortickinput "orquesta/modulos/orquesta-director-tick-input"
)

func ExecuteDirectorCycleStepV0(
	ctx context.Context,
	input DirectorCycleStepInputV0,
) (DirectorCycleStepResultV0, error) {
	input = normalizeDirectorCycleStepInputV0(input)
	result := newDirectorCycleStepResultV0(input)
	if err := validateDirectorCycleStepInputV0(ctx, input); err != nil {
		return resultWithCycleStepErrorV0(result, err)
	}
	pendingBefore, err := listCycleStepPendingOutboxV0(ctx, input)
	if err != nil {
		return resultWithCycleStepErrorV0(result, err)
	}
	result.PendingOutboxBeforeRefs = pendingBefore.PendingOutboxRefs
	schedulerInput, err := buildCycleStepSchedulerInputV0(input, pendingBefore.PendingOutboxRefs)
	if err != nil {
		return resultWithCycleStepIssueV0(result, input, ErrDirectorCycleStepTickInputV0, "tick_input", "tick input fallo", false)
	}
	runnerResult, err := orquestadirectorrunner.RunDirectorCycleV0(ctx, orquestadirectorrunner.DirectorCycleInputV0{
		Scheduler:      input.Scheduler,
		Workflow:       input.Workflow,
		CycleRef:       input.CycleRef,
		RunRef:         input.RunRef,
		SchedulerInput: schedulerInput,
		MaxCommands:    input.MaxCommands,
		MaxOutbox:      input.MaxOutbox,
		CorrelationID:  input.CorrelationID,
		EvidenceRefs:   input.EvidenceRefs,
	})
	copyRunnerResultToStepV0(&result, runnerResult)
	if err != nil {
		return resultWithCycleStepRunnerErrorV0(result, input, runnerResult, err)
	}
	outboxRecord, err := recordCycleStepOutboxV0(ctx, input, runnerResult.Outbox)
	if err != nil {
		return resultWithCycleStepErrorV0(result, err)
	}
	result.OutboxSavedCount = outboxRecord.SavedCount
	result.OutboxPendingAfterCount = outboxRecord.PendingCount
	result.PendingOutboxAfterRefs = outboxRecord.PendingOutboxRefs
	return result, nil
}

func listCycleStepPendingOutboxV0(
	ctx context.Context,
	input DirectorCycleStepInputV0,
) (orquestadirectorcycleoutbox.DirectorCycleOutboxRecordResultV0, error) {
	result, err := orquestadirectorcycleoutbox.RecordDirectorCycleOutboxV0(ctx, orquestadirectorcycleoutbox.DirectorCycleOutboxRecordInputV0{
		Ledger:        input.OutboxLedger,
		RunRef:        input.RunRef,
		CorrelationID: input.CorrelationID,
	})
	if err != nil {
		return result, cycleStepErrorV0(input, ErrDirectorCycleStepOutboxV0, "outbox ledger fallo", "outbox_ledger", true, nil)
	}
	return result, nil
}

func recordCycleStepOutboxV0(
	ctx context.Context,
	input DirectorCycleStepInputV0,
	messages []orquestacoreworkflow.OutboxMessageV0,
) (orquestadirectorcycleoutbox.DirectorCycleOutboxRecordResultV0, error) {
	result, err := orquestadirectorcycleoutbox.RecordDirectorCycleOutboxV0(ctx, orquestadirectorcycleoutbox.DirectorCycleOutboxRecordInputV0{
		Ledger:        input.OutboxLedger,
		RunRef:        input.RunRef,
		Messages:      messages,
		CorrelationID: input.CorrelationID,
	})
	if err != nil {
		return result, cycleStepErrorV0(input, ErrDirectorCycleStepOutboxV0, "outbox ledger fallo", "outbox_ledger", true, nil)
	}
	return result, nil
}

func buildCycleStepSchedulerInputV0(
	input DirectorCycleStepInputV0,
	pendingOutboxRefs []string,
) (orquestadirectorscheduler.DirectorSchedulerTickInputV0, error) {
	return orquestadirectortickinput.BuildDirectorSchedulerTickInputV0(orquestadirectortickinput.DirectorTickInputBuildRequestV0{
		TickRef:                       input.TickRef,
		OccurredAt:                    input.OccurredAt,
		Run:                           input.Run,
		PendingOutboxRefs:             pendingOutboxRefs,
		LeaseActionCandidates:         input.LeaseActionCandidates,
		PhaseArtifactCandidates:       input.PhaseArtifactCandidates,
		DeliveryCandidates:            input.DeliveryCandidates,
		ReviewGateCandidates:          input.ReviewGateCandidates,
		ProgressSupervisionCandidates: input.ProgressSupervisionCandidates,
		ReplanFollowupCandidates:      input.ReplanFollowupCandidates,
		WorkClaims:                    input.WorkClaims,
		WorkCandidates:                input.WorkCandidates,
		EvidenceRefs:                  input.EvidenceRefs,
	})
}
