package orquestaappcodexstack

import (
	"context"
	"fmt"
	"strings"

	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
)

func (stack *StackV0) ObserveAppDirectorGoalV0(
	ctx context.Context,
	request orquestaappdirectorservice.ObserveAppDirectorGoalRequestV0,
) (orquestaappdirectorservice.ObserveAppDirectorGoalResultV0, error) {
	coordinator := stack.goalFirstObservationCoordinatorV0()
	if coordinator == nil {
		return orquestaappdirectorservice.ObserveAppDirectorGoalResultV0{}, fmt.Errorf("goal_first_observation_coordinator_unavailable")
	}
	release, err := coordinator.acquireV0(ctx, request.RunRef)
	if err != nil {
		return orquestaappdirectorservice.ObserveAppDirectorGoalResultV0{}, err
	}
	defer release()
	return stack.observeAppDirectorGoalSerializedV0(ctx, request)
}

func (stack *StackV0) observeAppDirectorGoalSerializedV0(
	ctx context.Context,
	request orquestaappdirectorservice.ObserveAppDirectorGoalRequestV0,
) (orquestaappdirectorservice.ObserveAppDirectorGoalResultV0, error) {
	result, err := orquestaappdirectorservice.ObserveAppDirectorGoalV0(ctx, request, stack.Ports)
	if err != nil {
		return result, err
	}
	if submitted, err := stack.submitDomainWorkArtifactAfterGoalObservationV0(ctx, request, result); err != nil || submitted {
		if err != nil {
			return result, err
		}
		result, err = orquestaappdirectorservice.ObserveAppDirectorGoalV0(ctx, request, stack.Ports)
		if err != nil {
			return result, err
		}
	}
	if err := stack.syncGoalFirstQueueAfterObservationV0(ctx, request, result); err != nil {
		return result, err
	}
	return result, nil
}

func (stack *StackV0) syncGoalFirstQueueAfterObservationV0(
	ctx context.Context,
	request orquestaappdirectorservice.ObserveAppDirectorGoalRequestV0,
	result orquestaappdirectorservice.ObserveAppDirectorGoalResultV0,
) error {
	if stack.Stores.RunQueue == nil {
		return nil
	}
	status, ok := goalFirstQueueStatusForRunV0(result.Run.Status)
	if !ok {
		return nil
	}
	runRef := firstNonEmptyQueuedSourceV0(result.Run.RunID, result.RunRef, request.RunRef)
	if strings.TrimSpace(runRef) == "" {
		return nil
	}
	queue := normalizeRunQueueConfigV0(stack.RunQueue)
	candidate, err := stack.goalFirstQueueCandidateV0(ctx, queue.QueueRef, runRef)
	if err != nil {
		return err
	}
	_, err = stack.Stores.RunQueue.SetRunPriorityV0(ctx, goalFirstQueueCommandV0(
		*stack,
		queue,
		candidate,
		result,
		runRef,
		status,
	))
	return err
}

func goalFirstQueueStatusForRunV0(
	status orquestacoreworkflow.OrchestrationRunStatusV0,
) (string, bool) {
	switch status {
	case orquestacoreworkflow.OrchestrationRunStatusClosedV0:
		return orquestarunqueue.RunStatusClosedV0, true
	case orquestacoreworkflow.OrchestrationRunStatusBlockedV0:
		return orquestarunqueue.RunStatusStoppedV0, true
	default:
		return "", false
	}
}

func (stack StackV0) goalFirstQueueCandidateV0(
	ctx context.Context,
	queueRef string,
	runRef string,
) (orquestarunqueue.RunSchedulingCandidateV0, error) {
	candidates, err := stack.Stores.RunQueue.ListRunSchedulingCandidatesV0(
		ctx,
		orquestarunqueue.RunQueueReadRequestV0{
			QueueRef:             queueRef,
			IncludeNonExecutable: true,
		},
	)
	if err != nil {
		return orquestarunqueue.RunSchedulingCandidateV0{}, err
	}
	runRef = strings.TrimSpace(runRef)
	for _, candidate := range candidates {
		if strings.TrimSpace(candidate.RunRef) == runRef {
			return candidate, nil
		}
	}
	return orquestarunqueue.RunSchedulingCandidateV0{}, nil
}

func goalFirstQueueCommandV0(
	stack StackV0,
	queue RunQueueConfigV0,
	candidate orquestarunqueue.RunSchedulingCandidateV0,
	result orquestaappdirectorservice.ObserveAppDirectorGoalResultV0,
	runRef string,
	status string,
) orquestarunqueue.RunQueuePriorityCommandV0 {
	if strings.TrimSpace(candidate.RunRef) == "" {
		candidate.RunRef = runRef
		candidate.AppRef = firstNonEmptyQueuedSourceV0(
			result.Run.AppSpecRef,
			result.Run.ProjectRef,
			runRef,
		)
		candidate.PriorityScore = queue.DefaultPriorityScore
	}
	evidenceRefs := append([]string(nil), candidate.EvidenceRefs...)
	evidenceRefs = append(evidenceRefs, result.EvidenceRefs...)
	evidenceRefs = compactStringsV0(append(evidenceRefs, "evidence-ref-goal-first-queue-terminal-sync"))
	return orquestarunqueue.RunQueuePriorityCommandV0{
		RunRef:           runRef,
		QueueRef:         queue.QueueRef,
		AppRef:           candidate.AppRef,
		Status:           status,
		PriorityScore:    candidate.PriorityScore,
		UpdatedAt:        stackNowV0(stack.Clock),
		FairnessGroupRef: candidate.FairnessGroupRef,
		AttemptGroup:     candidate.AttemptGroup,
		ParentRunRef:     candidate.ParentRunRef,
		SupersedesRunRef: candidate.SupersedesRunRef,
		RescueReason:     candidate.RescueReason,
		RequestedBy:      "orquesta-app-codex-stack-goal-first",
		Reason:           "goal_first_terminal_observed",
		IdempotencyKey: "idem-run-queue-goal-first-terminal-" +
			codexStackOperationalClosureSafeRefV0(runRef) + "-" + status,
		EvidenceRefs:  evidenceRefs,
		WorksetClaims: candidate.WorksetClaims,
	}
}
