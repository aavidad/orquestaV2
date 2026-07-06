package orquestaappcodexstack

import (
	"context"
	"errors"
	"strings"

	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
	orquestaruncoordinator "orquesta/modulos/orquesta-run-coordinator"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
)

const codexSupervisorOperationalPlanStateNeedsReplanOutcomeV0 = "needs_replan"
const codexSupervisorRunEventsOversizedOutcomeV0 = "run_oversized"

func codexSupervisorRecoverableOperationalPlanStateErrorV0(err error) bool {
	if err == nil {
		return false
	}
	var serviceIssue orquestaappdirectorservice.AppDirectorServiceIssueV0
	if errors.As(err, &serviceIssue) &&
		strings.TrimSpace(serviceIssue.Field) == "operational_director_plan_state.active_step" {
		return true
	}
	var coreIssue orquestacionnucleoapp.ErrorV0
	if errors.As(err, &coreIssue) &&
		coreIssue.Code == orquestacionnucleoapp.ErrNucleoOrquestacionInvalidoV0 {
		field := strings.TrimSpace(coreIssue.Field)
		message := strings.TrimSpace(coreIssue.Message)
		return field == "active_step_id" ||
			field == "operational_director_plan_state.active_step" ||
			(field == "operational_director_plan_state" && strings.Contains(message, "active_step_id"))
	}
	return false
}

func codexSupervisorRunEventsBudgetExceededV0(err error) bool {
	if err == nil {
		return false
	}
	var coreIssue orquestacionnucleoapp.ErrorV0
	if !errors.As(err, &coreIssue) ||
		coreIssue.Code != orquestacionnucleoapp.ErrNucleoOrquestacionStoreV0 ||
		strings.TrimSpace(coreIssue.Field) != "events.budget" {
		return false
	}
	switch strings.TrimSpace(coreIssue.Message) {
	case "events_full_history_budget_exceeded", "events_run_limit_exceeded":
		return true
	default:
		return strings.Contains(err.Error(), "events_full_history_budget_exceeded") ||
			strings.Contains(err.Error(), "events_run_limit_exceeded")
	}
}

func codexSupervisorOperationalPlanStateNeedsReplanDiagnosticV0(
	runRef string,
	err error,
) orquestaruncoordinator.RunDrainDiagnosticV0 {
	message := ""
	if err != nil {
		message = strings.TrimSpace(err.Error())
	}
	return orquestaruncoordinator.RunDrainDiagnosticV0{
		Kind:   "operational_plan_state_active_step_needs_replan",
		Status: codexSupervisorOperationalPlanStateNeedsReplanOutcomeV0,
		RunRef: strings.TrimSpace(runRef),
		Error:  message,
		EvidenceRefs: []string{
			"evidence-ref-codex-supervisor-operational-plan-state-active-step-needs-replan",
		},
	}
}

func codexSupervisorRunEventsOversizedDiagnosticV0(
	runRef string,
	err error,
) orquestaruncoordinator.RunDrainDiagnosticV0 {
	message := ""
	if err != nil {
		message = strings.TrimSpace(err.Error())
	}
	return orquestaruncoordinator.RunDrainDiagnosticV0{
		Kind:       "run_events_budget_exceeded",
		Status:     codexSupervisorRunEventsOversizedOutcomeV0,
		RunRef:     strings.TrimSpace(runRef),
		Error:      message,
		TargetPort: "run_supervisor",
		EvidenceRefs: []string{
			"evidence-ref-codex-supervisor-run-events-budget-exceeded",
		},
	}
}

func (stack StackV0) markQueuedCandidateOperationalPlanStateNeedsReplanV0(
	ctx context.Context,
	command orquestaruncoordinator.RunCoordinatorTickCommandV0,
	candidate orquestarunqueue.RunSchedulingCandidateV0,
) error {
	if stack.Stores.RunQueue == nil {
		return nil
	}
	updatedAt := command.OccurredAt
	if updatedAt.IsZero() {
		updatedAt = stackNowV0(stack.Clock)
	}
	_, err := stack.Stores.RunQueue.SetRunPriorityV0(ctx, orquestarunqueue.RunQueuePriorityCommandV0{
		RunRef:           candidate.RunRef,
		QueueRef:         command.QueueRef,
		AppRef:           candidate.AppRef,
		Status:           orquestarunqueue.RunStatusStoppedV0,
		PriorityScore:    candidate.PriorityScore,
		UpdatedAt:        updatedAt,
		FairnessGroupRef: candidate.FairnessGroupRef,
		AttemptGroup:     candidate.AttemptGroup,
		ParentRunRef:     candidate.ParentRunRef,
		SupersedesRunRef: candidate.SupersedesRunRef,
		RescueReason:     candidate.RescueReason,
		RequestedBy:      "orquesta-app-codex-stack",
		Reason:           "operational_plan_state_active_step_needs_replan",
		IdempotencyKey: "orquesta-app-codex-stack-operational-plan-needs-replan:" +
			strings.TrimSpace(candidate.RunRef),
		EvidenceRefs: compactStringsV0(append(
			candidate.EvidenceRefs,
			"evidence-ref-codex-supervisor-operational-plan-state-active-step-needs-replan",
			"evidence-ref-codex-supervisor-operational-plan-needs-replan",
		)),
		WorksetClaims: candidate.WorksetClaims,
	})
	return err
}

func (stack StackV0) markQueuedCandidateRunEventsOversizedV0(
	ctx context.Context,
	command orquestaruncoordinator.RunCoordinatorTickCommandV0,
	candidate orquestarunqueue.RunSchedulingCandidateV0,
) error {
	if err := stack.markRunEventsOversizedRunControlV0(ctx, candidate.RunRef); err != nil {
		return err
	}
	if stack.Stores.RunQueue == nil {
		return nil
	}
	updatedAt := command.OccurredAt
	if updatedAt.IsZero() {
		updatedAt = stackNowV0(stack.Clock)
	}
	_, err := stack.Stores.RunQueue.SetRunPriorityV0(ctx, orquestarunqueue.RunQueuePriorityCommandV0{
		RunRef:           candidate.RunRef,
		QueueRef:         command.QueueRef,
		AppRef:           candidate.AppRef,
		Status:           orquestarunqueue.RunStatusStoppedV0,
		PriorityScore:    candidate.PriorityScore,
		UpdatedAt:        updatedAt,
		FairnessGroupRef: candidate.FairnessGroupRef,
		AttemptGroup:     candidate.AttemptGroup,
		ParentRunRef:     candidate.ParentRunRef,
		SupersedesRunRef: candidate.SupersedesRunRef,
		RescueReason:     "run_oversized_events_budget",
		RequestedBy:      "orquesta-app-codex-stack",
		Reason:           "run_oversized_events_budget",
		IdempotencyKey: "orquesta-app-codex-stack-run-events-budget:" +
			strings.TrimSpace(candidate.RunRef),
		EvidenceRefs: compactStringsV0(append(
			candidate.EvidenceRefs,
			"evidence-ref-codex-supervisor-run-events-budget-exceeded",
			"evidence-ref-codex-supervisor-run-oversized-parked",
		)),
		WorksetClaims: candidate.WorksetClaims,
	})
	return err
}

func (stack StackV0) markRunEventsOversizedRunControlV0(
	ctx context.Context,
	runRef string,
) error {
	if stack.Stores.RunControl == nil {
		return nil
	}
	runRef = strings.TrimSpace(runRef)
	if runRef == "" {
		return nil
	}
	state, err := stack.Stores.RunControl.ReadRunControlStateV0(
		ctx,
		orquestaruncontrol.RunControlReadRequestV0{RunRef: runRef},
	)
	if err == nil &&
		orquestaruncontrol.NormalizeRunControlStatusV0(state.Status) == orquestaruncontrol.RunControlStatusCanceledV0 {
		return nil
	}
	if err != nil {
		var notFound orquestaruncontrol.RunControlStateNotFoundErrorV0
		if !errors.As(err, &notFound) {
			return err
		}
	}
	evidenceRefs := []string{
		"evidence-ref-codex-supervisor-run-events-budget-exceeded",
		"evidence-ref-codex-supervisor-run-oversized-parked",
	}
	if err == nil {
		evidenceRefs = compactStringsV0(append(state.EvidenceRefs, evidenceRefs...))
	}
	_, err = stack.Stores.RunControl.CompleteRunControlV0(ctx, orquestaruncontrol.CompleteRunControlCommandV0{
		RunRef:       runRef,
		TargetStatus: orquestaruncontrol.RunControlStatusStoppedV0,
		RequestedBy:  "orquesta-app-codex-stack",
		Reason:       "run_oversized_events_budget",
		IdempotencyKey: "orquesta-app-codex-stack-run-events-budget:" +
			runRef,
		EvidenceRefs: evidenceRefs,
	})
	return err
}
