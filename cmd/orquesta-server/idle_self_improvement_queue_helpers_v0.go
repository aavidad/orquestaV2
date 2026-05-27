package main

import (
	"context"
	"errors"
	"strings"
	"time"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
)

func (supervisor serverStackSupervisorV0) markIdleSelfImprovementQueueCandidateClosedV0(
	ctx context.Context,
	queueRef string,
	candidates []orquestarunqueue.RunSchedulingCandidateV0,
	runRef string,
) error {
	if supervisor.stack == nil || supervisor.stack.Stores.RunQueue == nil {
		return nil
	}
	priority := idleSelfImprovementCandidatePriorityScoreV0(candidates, runRef)
	_, err := supervisor.stack.Stores.RunQueue.SetRunPriorityV0(ctx, orquestarunqueue.RunQueuePriorityCommandV0{
		RunRef:         runRef,
		QueueRef:       queueRef,
		Status:         orquestarunqueue.RunStatusClosedV0,
		PriorityScore:  priority,
		UpdatedAt:      time.Now().UTC(),
		RequestedBy:    "orquesta-server-idle-self-improvement",
		Reason:         "prepared_closed_run_queue_candidate_synced",
		IdempotencyKey: "idem-idle-self-improvement-close-sync-" + runRef,
		EvidenceRefs: []string{
			"evidence-ref-idle-self-improvement-queue-candidate-closed-synced",
		},
	})
	return err
}

func (supervisor serverStackSupervisorV0) idleSelfImprovementPreparedRunPersistedV0(
	ctx context.Context,
	runRef string,
) bool {
	runRef = strings.TrimSpace(runRef)
	if runRef == "" || supervisor.stack == nil || supervisor.stack.Stores.RunStore == nil {
		return false
	}
	_, err := supervisor.stack.Stores.RunStore.LoadRunV0(ctx, runRef)
	return err == nil
}

func (supervisor serverStackSupervisorV0) idleSelfImprovementPreparedRunStatusV0(
	ctx context.Context,
	runRef string,
) (orquestacoreworkflow.OrchestrationRunStatusV0, bool) {
	runRef = strings.TrimSpace(runRef)
	if runRef == "" || supervisor.stack == nil || supervisor.stack.Stores.RunStore == nil {
		return "", false
	}
	run, err := supervisor.stack.Stores.RunStore.LoadRunV0(ctx, runRef)
	if err != nil {
		return "", false
	}
	return run.Status, true
}

func (supervisor serverStackSupervisorV0) idleSelfImprovementPreparedRunControlStateV0(
	ctx context.Context,
	runRef string,
) (orquestaruncontrol.RunControlStateV0, bool) {
	runRef = strings.TrimSpace(runRef)
	if runRef == "" || supervisor.stack == nil || supervisor.stack.Stores.RunControl == nil {
		return orquestaruncontrol.RunControlStateV0{}, false
	}
	state, err := supervisor.stack.Stores.RunControl.ReadRunControlStateV0(
		ctx,
		orquestaruncontrol.RunControlReadRequestV0{RunRef: runRef},
	)
	if err != nil {
		var notFound orquestaruncontrol.RunControlStateNotFoundErrorV0
		if errors.As(err, &notFound) {
			return orquestaruncontrol.RunControlStateV0{}, false
		}
		return orquestaruncontrol.RunControlStateV0{}, false
	}
	return state, true
}

func idleSelfImprovementQueueCandidateStatusV0(
	candidates []orquestarunqueue.RunSchedulingCandidateV0,
	runRef string,
) (string, bool) {
	for _, candidate := range candidates {
		if strings.TrimSpace(candidate.RunRef) == runRef {
			return strings.TrimSpace(candidate.Status), true
		}
	}
	return "", false
}

func idleSelfImprovementCandidatePriorityScoreV0(
	candidates []orquestarunqueue.RunSchedulingCandidateV0,
	runRef string,
) int {
	for _, candidate := range candidates {
		if strings.TrimSpace(candidate.RunRef) == strings.TrimSpace(runRef) {
			return candidate.PriorityScore
		}
	}
	return 0
}

func idleSelfImprovementCanRequeueNonExecutableStatusV0(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case orquestarunqueue.RunStatusDeliveredV0, orquestarunqueue.RunStatusStoppedV0:
		return true
	default:
		return false
	}
}

func firstPrepareRunIssueMessageServerStackV0(issues []orquestamcp.MCPValidationIssueV0) string {
	for _, issue := range issues {
		if message := strings.TrimSpace(issue.Message); message != "" {
			return message
		}
		if code := strings.TrimSpace(issue.Code); code != "" {
			return code
		}
	}
	return ""
}

func firstNonEmptyServerStackV0(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
