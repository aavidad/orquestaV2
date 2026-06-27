package orquestastatefile

import (
	"context"
	"strings"

	orquestagoal "orquesta/modulos/orquesta-goal"
)

type appDirectorGoalMarkerDocumentV0 struct {
	SchemaVersion string                           `json:"schema_version"`
	RunRef        string                           `json:"run_ref"`
	GoalRef       string                           `json:"goal_ref,omitempty"`
	Marker        orquestagoal.GoalWorkRunMarkerV0 `json:"marker"`
}

func (store *StoreV0) SaveGoalWorkRunMarkerV0(
	ctx context.Context,
	marker orquestagoal.GoalWorkRunMarkerV0,
) error {
	ctx = contextOrBackgroundV0(ctx)
	if err := ctx.Err(); err != nil {
		return err
	}
	normalized, err := orquestagoal.NewGoalWorkRunMarkerV0(marker)
	if err != nil {
		return invalidErrorV0("app_director_goal_marker", err.Error())
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	return store.saveAppDirectorGoalMarkerLockedV0(normalized)
}

func (store *StoreV0) LoadGoalWorkRunMarkerV0(
	ctx context.Context,
	runRef string,
) (orquestagoal.GoalWorkRunMarkerV0, error) {
	ctx = contextOrBackgroundV0(ctx)
	if err := ctx.Err(); err != nil {
		return orquestagoal.GoalWorkRunMarkerV0{}, err
	}
	runRef = normalizeRefV0(runRef)
	if runRef == "" {
		return orquestagoal.GoalWorkRunMarkerV0{}, invalidErrorV0("run_ref", "run_ref requerido")
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	document, ok, err := readJSONFileV0[appDirectorGoalMarkerDocumentV0](store.appDirectorGoalMarkerPathV0(runRef))
	if err != nil {
		return orquestagoal.GoalWorkRunMarkerV0{}, err
	}
	if !ok {
		return orquestagoal.GoalWorkRunMarkerV0{}, storeErrorV0("app_director_goal_marker", "marcador goal-first no encontrado")
	}
	marker, err := validateAppDirectorGoalMarkerDocumentV0(document, runRef)
	if err != nil {
		return orquestagoal.GoalWorkRunMarkerV0{}, err
	}
	return marker, nil
}

func (store *StoreV0) saveAppDirectorGoalMarkerLockedV0(
	marker orquestagoal.GoalWorkRunMarkerV0,
) error {
	runRef := normalizeRefV0(marker.RunRef)
	return writeJSONAtomicV0(store.appDirectorGoalMarkerPathV0(runRef), appDirectorGoalMarkerDocumentV0{
		SchemaVersion: orquestagoal.GoalWorkRunMarkerSchemaV0,
		RunRef:        runRef,
		GoalRef:       strings.TrimSpace(marker.GoalRef),
		Marker:        marker,
	})
}

func validateAppDirectorGoalMarkerDocumentV0(
	document appDirectorGoalMarkerDocumentV0,
	runRef string,
) (orquestagoal.GoalWorkRunMarkerV0, error) {
	if document.SchemaVersion != orquestagoal.GoalWorkRunMarkerSchemaV0 {
		return orquestagoal.GoalWorkRunMarkerV0{}, storeErrorV0("app_director_goal_marker.schema_version", "schema_version invalida")
	}
	if normalizeRefV0(document.RunRef) != normalizeRefV0(runRef) {
		return orquestagoal.GoalWorkRunMarkerV0{}, storeErrorV0("app_director_goal_marker.ref", "ref inconsistente")
	}
	marker, err := orquestagoal.NewGoalWorkRunMarkerV0(document.Marker)
	if err != nil {
		return orquestagoal.GoalWorkRunMarkerV0{}, err
	}
	if normalizeRefV0(marker.RunRef) != normalizeRefV0(runRef) {
		return orquestagoal.GoalWorkRunMarkerV0{}, storeErrorV0("app_director_goal_marker.state_ref", "ref interna inconsistente")
	}
	return marker, nil
}
