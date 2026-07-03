package orquestaappcodexstack

import (
	"context"
	"strings"

	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestaservershutdown "orquesta/modulos/orquesta-server-shutdown"
)

const (
	goalFirstBackendGoneWithoutResultReasonV0   = "goal_backend_gone_without_result"
	goalFirstBackendGoneWithoutResultEvidenceV0 = "evidence-ref-goal-backend-gone-without-result"
)

type goalFirstReconciledObserverV0 struct {
	Inner                    orquestagoal.GoalWorkObservationPortV0
	StateStore               orquestagoal.GoalWorkStateStorePortV0
	MaterializedResultSource stackGoalMaterializedRefsSourceV0
}

func goalFirstReconciledObserverFromConfigV0(config ConfigV0) orquestagoal.GoalWorkObservationPortV0 {
	if config.AppGoalObserver == nil {
		return nil
	}
	return goalFirstReconciledObserverV0{
		Inner:      config.AppGoalObserver,
		StateStore: config.Stores.AppGoalStateStore,
		MaterializedResultSource: stackGoalMaterializedRefsSourceV0{
			Config: config,
		},
	}
}

func (observer goalFirstReconciledObserverV0) ObserveGoalWorkV0(
	ctx context.Context,
	request orquestagoal.GoalObservationRequestV0,
) (orquestagoal.GoalWorkResultV0, error) {
	state, stateOK := observer.goalStateForObservationV0(ctx, request)
	if stateOK {
		if result, ok, err := observer.MaterializedResultSource.LoadTerminalGoalMaterializedResultV0(ctx, state); err != nil {
			return orquestagoal.GoalWorkResultV0{}, err
		} else if ok {
			return result, nil
		}
		if observer.backendGoneWithoutResultV0(ctx, state) {
			return goalFirstBackendGoneWithoutResultV0(state), nil
		}
	}
	if observer.Inner == nil {
		return orquestagoal.GoalWorkResultV0{}, nil
	}
	return observer.Inner.ObserveGoalWorkV0(ctx, request)
}

func (observer goalFirstReconciledObserverV0) ReadActiveShutdownWorkV0(
	ctx context.Context,
	request orquestaservershutdown.ActiveShutdownWorkRequestV0,
) (orquestaservershutdown.ActiveShutdownWorkResultV0, error) {
	reader, ok := observer.Inner.(orquestaservershutdown.ActiveShutdownWorkReaderPortV0)
	if !ok || reader == nil {
		return orquestaservershutdown.ActiveShutdownWorkResultV0{}, nil
	}
	return reader.ReadActiveShutdownWorkV0(ctx, request)
}

func (observer goalFirstReconciledObserverV0) CleanupActiveShutdownWorkV0(
	ctx context.Context,
	command orquestaservershutdown.ActiveShutdownWorkCleanupCommandV0,
) (orquestaservershutdown.ActiveShutdownWorkCleanupResultV0, error) {
	cleaner, ok := observer.Inner.(orquestaservershutdown.ActiveShutdownWorkCleanerPortV0)
	if !ok || cleaner == nil {
		return orquestaservershutdown.ActiveShutdownWorkCleanupResultV0{}, nil
	}
	return cleaner.CleanupActiveShutdownWorkV0(ctx, command)
}

func (observer goalFirstReconciledObserverV0) goalStateForObservationV0(
	ctx context.Context,
	request orquestagoal.GoalObservationRequestV0,
) (orquestagoal.GoalWorkStateV0, bool) {
	lister, ok := observer.StateStore.(orquestagoal.GoalWorkStateListPortV0)
	if !ok || lister == nil {
		return orquestagoal.GoalWorkStateV0{}, false
	}
	states, err := lister.ListGoalWorkStatesV0(ctx, orquestagoal.GoalWorkStateListRequestV0{
		Statuses: []string{
			orquestagoal.GoalStatusRunningV0,
			orquestagoal.GoalStatusCompleteV0,
			orquestagoal.GoalStatusBlockedV0,
			orquestagoal.GoalStatusInvalidV0,
		},
		MaxItems: 200,
	})
	if err != nil {
		return orquestagoal.GoalWorkStateV0{}, false
	}
	for _, state := range states {
		if goalFirstStateMatchesObservationV0(state, request) {
			return state, true
		}
	}
	return orquestagoal.GoalWorkStateV0{}, false
}

func (observer goalFirstReconciledObserverV0) backendGoneWithoutResultV0(
	ctx context.Context,
	state orquestagoal.GoalWorkStateV0,
) bool {
	reader, ok := observer.Inner.(orquestaservershutdown.ActiveShutdownWorkReaderPortV0)
	if !ok || reader == nil {
		return false
	}
	active, err := reader.ReadActiveShutdownWorkV0(ctx, orquestaservershutdown.ActiveShutdownWorkRequestV0{
		MaxItems: 200,
	})
	if err != nil {
		return false
	}
	// Un ActiveWork de backend describe la sesion del app-server (work_ref =
	// sesion tmux), no refs por goal: la correspondencia por goal_ref nunca
	// casa y decaia goals con backend vivo. La unica evidencia segura de
	// backend caido es la ausencia total de trabajos activos.
	return len(active.ActiveWorks) == 0
}

func goalFirstStateMatchesObservationV0(
	state orquestagoal.GoalWorkStateV0,
	request orquestagoal.GoalObservationRequestV0,
) bool {
	goalRef := strings.TrimSpace(request.GoalRef)
	externalGoalRef := strings.TrimSpace(request.ExternalGoalRef)
	return goalRef != "" && strings.TrimSpace(state.GoalRef) == goalRef ||
		externalGoalRef != "" && strings.TrimSpace(state.ExternalGoalRef) == externalGoalRef
}

func goalFirstBackendGoneWithoutResultV0(
	state orquestagoal.GoalWorkStateV0,
) orquestagoal.GoalWorkResultV0 {
	return orquestagoal.NormalizeGoalWorkResultV0(orquestagoal.GoalWorkResultV0{
		SchemaVersion:   orquestagoal.GoalWorkResultSchemaV0,
		Status:          orquestagoal.GoalStatusBlockedV0,
		GoalRef:         state.GoalRef,
		ExternalGoalRef: state.ExternalGoalRef,
		Summary:         goalFirstBackendGoneWithoutResultReasonV0,
		EvidenceRefs:    []string{goalFirstBackendGoneWithoutResultEvidenceV0},
		Issues: []orquestagoal.GoalWorkIssueV0{{
			Code:  goalFirstBackendGoneWithoutResultReasonV0,
			Field: "goal_backend",
		}},
	})
}
