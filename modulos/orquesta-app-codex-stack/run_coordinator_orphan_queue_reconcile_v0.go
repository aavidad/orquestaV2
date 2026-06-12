package orquestaappcodexstack

import (
	"context"
	"errors"
	"strings"

	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
	orquestaruncoordinator "orquesta/modulos/orquesta-run-coordinator"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
)

func (stack StackV0) reconcileQueuedOrphanExecutableRunsV0(
	ctx context.Context,
	command orquestaruncoordinator.RunCoordinatorTickCommandV0,
) error {
	if stack.Stores.RunQueue == nil || stack.Stores.RunControl == nil || stack.Ports.RunStore == nil {
		return nil
	}
	candidates, err := stack.Stores.RunQueue.ListRunSchedulingCandidatesV0(
		ctx,
		orquestarunqueue.RunQueueReadRequestV0{
			QueueRef: strings.TrimSpace(command.QueueRef),
			AppRefs:  append([]string(nil), command.AppRefs...),
		},
	)
	if err != nil {
		return err
	}
	for _, candidate := range candidates {
		if !codexStackQueueStatusCanRetireOrphanV0(candidate.Status) {
			continue
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if _, err := stack.Ports.RunStore.LoadRunV0(ctx, candidate.RunRef); err == nil {
			continue
		} else if !orquestacionnucleoapp.IsRunNotFoundErrorV0(err) {
			return err
		}
		if err := stack.retireQueuedOrphanExecutableRunV0(ctx, command, candidate); err != nil {
			return err
		}
	}
	return nil
}

func codexStackQueueStatusCanRetireOrphanV0(status string) bool {
	switch strings.TrimSpace(status) {
	case orquestarunqueue.RunStatusReadyV0, orquestarunqueue.RunStatusRunningV0:
		return true
	default:
		return false
	}
}

func (stack StackV0) retireQueuedOrphanExecutableRunV0(
	ctx context.Context,
	command orquestaruncoordinator.RunCoordinatorTickCommandV0,
	candidate orquestarunqueue.RunSchedulingCandidateV0,
) error {
	target := orquestaruncontrol.RunControlStatusStoppedV0
	queueStatus := orquestarunqueue.RunStatusStoppedV0
	state, err := stack.Stores.RunControl.ReadRunControlStateV0(
		ctx,
		orquestaruncontrol.RunControlReadRequestV0{RunRef: candidate.RunRef},
	)
	if err != nil {
		var notFound orquestaruncontrol.RunControlStateNotFoundErrorV0
		if !errors.As(err, &notFound) {
			return err
		}
		state = orquestaruncontrol.DefaultRunControlStateV0(candidate.RunRef)
	}
	switch orquestaruncontrol.NormalizeRunControlStatusV0(state.Status) {
	case orquestaruncontrol.RunControlStatusCancelRequestedV0, orquestaruncontrol.RunControlStatusCanceledV0:
		target = orquestaruncontrol.RunControlStatusCanceledV0
		queueStatus = orquestarunqueue.RunStatusCanceledV0
	}
	evidence := compactStringsV0(append(
		append([]string(nil), state.EvidenceRefs...),
		"evidence-ref-run-queue-orphan-executable-auto-retired",
		"evidence-ref-run-store-missing",
	))
	completed, err := stack.Stores.RunControl.CompleteRunControlV0(ctx, orquestaruncontrol.CompleteRunControlCommandV0{
		RunRef:         candidate.RunRef,
		TargetStatus:   target,
		RequestedBy:    "orquesta-app-codex-stack-run-queue-reconciler",
		Reason:         "queued_executable_run_missing_from_run_store",
		IdempotencyKey: "idem-run-queue-orphan-retire-" + codexStackOperationalClosureSafeRefV0(candidate.RunRef),
		EvidenceRefs:   evidence,
	})
	if err != nil {
		return err
	}
	queueEvidence := compactStringsV0(append(
		append([]string(nil), candidate.EvidenceRefs...),
		completed.EvidenceRefs...,
	))
	_, err = stack.Stores.RunQueue.SetRunPriorityV0(ctx, orquestarunqueue.RunQueuePriorityCommandV0{
		RunRef:           candidate.RunRef,
		QueueRef:         strings.TrimSpace(command.QueueRef),
		AppRef:           candidate.AppRef,
		Status:           queueStatus,
		PriorityScore:    candidate.PriorityScore,
		UpdatedAt:        command.OccurredAt,
		FairnessGroupRef: candidate.FairnessGroupRef,
		AttemptGroup:     candidate.AttemptGroup,
		ParentRunRef:     candidate.ParentRunRef,
		SupersedesRunRef: candidate.SupersedesRunRef,
		RescueReason:     candidate.RescueReason,
		RequestedBy:      "orquesta-app-codex-stack-run-queue-reconciler",
		Reason:           "queued_executable_run_missing_from_run_store",
		IdempotencyKey:   "idem-run-queue-orphan-retire-" + codexStackOperationalClosureSafeRefV0(candidate.RunRef),
		EvidenceRefs:     queueEvidence,
		WorksetClaims:    candidate.WorksetClaims,
	})
	return err
}
