package orquestaappdirectorservice

import (
	"context"
	"errors"
	"strings"

	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

const (
	continueAppDirectorGoalFirstContainerEvidenceV0 = "evidence-ref-app-director-goal-first-container-continue-v0"
	continueAppDirectorGoalFirstObserveEvidenceV0   = "evidence-ref-app-director-goal-observe-required-v0"
	continueAppDirectorGoalFirstObserveCodeV0       = "app_director_goal_first_observe_required"
)

func continueAppDirectorGoalFirstContainerResultV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
) (ContinueAppDirectorResultV0, bool, error) {
	if request.RunRef == "" || ports.GoalStateStore == nil {
		return ContinueAppDirectorResultV0{}, false, nil
	}
	state, err := ports.GoalStateStore.LoadGoalWorkStateV0(ctx, request.RunRef)
	if err != nil {
		if continueAppDirectorGoalStateMissingV0(err) {
			return ContinueAppDirectorResultV0{}, false, nil
		}
		return ContinueAppDirectorResultV0{}, true, err
	}
	if _, err := NewAppDirectorGoalStateV0(state); err != nil {
		return ContinueAppDirectorResultV0{}, true, err
	}
	if ports.RunStore == nil {
		return ContinueAppDirectorResultV0{}, true, AppDirectorServiceIssueV0{Field: "ports.run_store"}
	}
	run, err := ports.RunStore.LoadRunV0(ctx, request.RunRef)
	if err != nil {
		return ContinueAppDirectorResultV0{}, true, err
	}
	return ContinueAppDirectorResultV0{
		SchemaVersion: ContinueAppDirectorResultSchemaVersionV0,
		Status:        ContinueAppDirectorStatusPendingV0,
		CorrelationID: request.CorrelationID,
		Run:           run,
		LoopStatus:    orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0,
		StartedAgents: append([]string(nil), run.StartedAgents...),
		OperationalClosureIssues: []orquestacionnucleoapp.ErrorV0{{
			Code:    continueAppDirectorGoalFirstObserveCodeV0,
			Field:   "goal_state",
			Message: "run goal-first: observar con ObserveAppDirectorGoalV0; no se ejecuta el loop legacy",
		}},
		EvidenceRefs: []string{
			continueAppDirectorGoalFirstContainerEvidenceV0,
			continueAppDirectorGoalFirstObserveEvidenceV0,
		},
	}, true, nil
}

func continueAppDirectorGoalStateMissingV0(err error) bool {
	if err == nil {
		return false
	}
	var serviceIssue AppDirectorServiceIssueV0
	if errors.As(err, &serviceIssue) && strings.TrimSpace(serviceIssue.Field) == "goal_state" {
		return true
	}
	var coreIssue orquestacionnucleoapp.ErrorV0
	if errors.As(err, &coreIssue) &&
		strings.TrimSpace(coreIssue.Code) == orquestacionnucleoapp.ErrNucleoOrquestacionStoreV0 &&
		strings.TrimSpace(coreIssue.Field) == "app_director_goal_state" &&
		continueAppDirectorGoalStateMissingTextV0(coreIssue.Message) {
		return true
	}
	message := err.Error()
	return (strings.Contains(message, "goal_state") || strings.Contains(message, "app_director_goal_state")) &&
		continueAppDirectorGoalStateMissingTextV0(message)
}

func continueAppDirectorGoalStateMissingTextV0(value string) bool {
	lower := strings.ToLower(strings.TrimSpace(value))
	return strings.Contains(lower, "not_found") ||
		strings.Contains(lower, "not found") ||
		strings.Contains(lower, "no encontrado") ||
		strings.Contains(lower, "no encontrada")
}
