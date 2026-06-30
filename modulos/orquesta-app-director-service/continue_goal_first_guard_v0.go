package orquestaappdirectorservice

import (
	"context"
	"errors"
	"strings"

	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

const (
	continueAppDirectorGoalFirstContainerEvidenceV0    = "evidence-ref-app-director-goal-first-container-continue-v0"
	continueAppDirectorGoalFirstObserveEvidenceV0      = "evidence-ref-app-director-goal-observe-required-v0"
	continueAppDirectorGoalFirstStateMissingEvidenceV0 = "evidence-ref-app-director-goal-state-missing-v0"
	continueAppDirectorGoalFirstObserveCodeV0          = "app_director_goal_first_observe_required"
	continueAppDirectorGoalFirstStateMissingCodeV0     = "app_director_goal_first_state_missing"
)

func continueAppDirectorGoalFirstContainerResultV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
) (ContinueAppDirectorResultV0, bool, error) {
	if request.RunRef == "" {
		return ContinueAppDirectorResultV0{}, false, nil
	}
	if ports.GoalStateStore == nil {
		return continueAppDirectorGoalFirstMarkerResultV0(ctx, request, ports)
	}
	state, err := ports.GoalStateStore.LoadGoalWorkStateV0(ctx, request.RunRef)
	if err != nil {
		if continueAppDirectorGoalStateMissingV0(err) {
			return continueAppDirectorGoalFirstMarkerResultV0(ctx, request, ports)
		}
		return ContinueAppDirectorResultV0{}, true, err
	}
	if _, err := NewAppDirectorGoalStateV0(state); err != nil {
		return ContinueAppDirectorResultV0{}, true, err
	}
	result, err := continueAppDirectorGoalFirstPendingResultV0(
		ctx,
		request,
		ports,
		continueAppDirectorGoalFirstObserveCodeV0,
		"goal_state",
		"run goal-first: observar con ObserveAppDirectorGoalV0; no se ejecuta el loop legacy",
		[]string{
			continueAppDirectorGoalFirstContainerEvidenceV0,
			continueAppDirectorGoalFirstObserveEvidenceV0,
		},
	)
	return result, true, err
}

func continueAppDirectorGoalFirstMarkerResultV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
) (ContinueAppDirectorResultV0, bool, error) {
	if ports.GoalFirstRunMarkerStore == nil {
		return ContinueAppDirectorResultV0{}, false, nil
	}
	marker, err := ports.GoalFirstRunMarkerStore.LoadGoalWorkRunMarkerV0(ctx, request.RunRef)
	if err != nil {
		if continueAppDirectorGoalFirstMarkerMissingV0(err) {
			return ContinueAppDirectorResultV0{}, false, nil
		}
		return ContinueAppDirectorResultV0{}, true, err
	}
	normalized, err := NewAppDirectorGoalFirstRunMarkerV0(marker)
	if err != nil {
		return ContinueAppDirectorResultV0{}, true, err
	}
	if repaired, ok, err := continueAppDirectorRepairGoalStateFromMarkerV0(ctx, request, ports, normalized); err != nil {
		return ContinueAppDirectorResultV0{}, true, err
	} else if ok {
		result, err := continueAppDirectorGoalFirstPendingResultV0(
			ctx,
			request,
			ports,
			continueAppDirectorGoalFirstObserveCodeV0,
			"goal_state",
			"run goal-first: GoalWorkStateV0 reconstruido desde marker; observar con ObserveAppDirectorGoalV0",
			compactStartAppDirectorStringsV0(append(
				[]string{
					continueAppDirectorGoalFirstContainerEvidenceV0,
					continueAppDirectorGoalFirstObserveEvidenceV0,
					"evidence-ref-app-director-goal-state-repaired-from-marker-v0",
				},
				repaired.EvidenceRefs...,
			)),
		)
		return result, true, err
	}
	result, err := continueAppDirectorGoalFirstPendingResultV0(
		ctx,
		request,
		ports,
		continueAppDirectorGoalFirstStateMissingCodeV0,
		"goal_state",
		"run goal-first: falta GoalWorkStateV0 durable; no se ejecuta el loop legacy",
		compactStartAppDirectorStringsV0(append(
			[]string{
				continueAppDirectorGoalFirstContainerEvidenceV0,
				continueAppDirectorGoalFirstStateMissingEvidenceV0,
			},
			normalized.EvidenceRefs...,
		)),
	)
	return result, true, err
}

func continueAppDirectorRepairGoalStateFromMarkerV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
	marker AppDirectorGoalFirstRunMarkerV0,
) (AppDirectorGoalStateV0, bool, error) {
	if ports.GoalStateStore == nil || marker.Spec == nil || marker.LaunchReceipt == nil {
		return AppDirectorGoalStateV0{}, false, nil
	}
	state, err := orquestagoal.NewGoalWorkStateFromLaunchV0(orquestagoal.GoalWorkStateFromLaunchRequestV0{
		RunRef:        request.RunRef,
		Spec:          *marker.Spec,
		LaunchReceipt: *marker.LaunchReceipt,
		EvidenceRefs: compactStartAppDirectorStringsV0(append(
			[]string{"evidence-ref-app-director-goal-state-repaired-from-marker-v0"},
			marker.EvidenceRefs...,
		)),
	})
	if err != nil {
		return AppDirectorGoalStateV0{}, false, err
	}
	if err := ports.GoalStateStore.SaveGoalWorkStateV0(ctx, state); err != nil {
		return AppDirectorGoalStateV0{}, false, err
	}
	return state, true, nil
}

func continueAppDirectorGoalFirstPendingResultV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
	code string,
	field string,
	message string,
	evidenceRefs []string,
) (ContinueAppDirectorResultV0, error) {
	if ports.RunStore == nil {
		return ContinueAppDirectorResultV0{}, AppDirectorServiceIssueV0{Field: "ports.run_store"}
	}
	run, err := ports.RunStore.LoadRunV0(ctx, request.RunRef)
	if err != nil {
		return ContinueAppDirectorResultV0{}, err
	}
	return ContinueAppDirectorResultV0{
		SchemaVersion: ContinueAppDirectorResultSchemaVersionV0,
		Status:        ContinueAppDirectorStatusPendingV0,
		CorrelationID: request.CorrelationID,
		Run:           run,
		LoopStatus:    orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0,
		StartedAgents: append([]string(nil), run.StartedAgents...),
		OperationalClosureIssues: []orquestacionnucleoapp.ErrorV0{{
			Code:    code,
			Field:   field,
			Message: message,
		}},
		EvidenceRefs: compactStartAppDirectorStringsV0(evidenceRefs),
	}, nil
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

func continueAppDirectorGoalFirstMarkerMissingV0(err error) bool {
	if err == nil {
		return false
	}
	var coreIssue orquestacionnucleoapp.ErrorV0
	if errors.As(err, &coreIssue) &&
		strings.TrimSpace(coreIssue.Code) == orquestacionnucleoapp.ErrNucleoOrquestacionStoreV0 &&
		strings.TrimSpace(coreIssue.Field) == "app_director_goal_marker" &&
		continueAppDirectorGoalStateMissingTextV0(coreIssue.Message) {
		return true
	}
	var serviceIssue AppDirectorServiceIssueV0
	if errors.As(err, &serviceIssue) && strings.TrimSpace(serviceIssue.Field) == "goal_first_run_marker" {
		return true
	}
	message := err.Error()
	return (strings.Contains(message, "app_director_goal_marker") ||
		strings.Contains(message, "goal_first_run_marker")) &&
		continueAppDirectorGoalStateMissingTextV0(message)
}

func continueAppDirectorGoalStateMissingTextV0(value string) bool {
	lower := strings.ToLower(strings.TrimSpace(value))
	return strings.Contains(lower, "not_found") ||
		strings.Contains(lower, "not found") ||
		strings.Contains(lower, "no encontrado") ||
		strings.Contains(lower, "no encontrada")
}
