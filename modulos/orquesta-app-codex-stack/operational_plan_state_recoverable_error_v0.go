package orquestaappcodexstack

import (
	"context"
	"errors"
	"strings"

	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruncoordinator "orquesta/modulos/orquesta-run-coordinator"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
)

const codexSupervisorOperationalPlanStateNeedsReplanOutcomeV0 = "needs_replan"

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
