package orquestastatefile

import (
	"context"

	orquestagoal "orquesta/modulos/orquesta-goal"
)

type appDirectorGoalStateDocumentV0 struct {
	SchemaVersion string                       `json:"schema_version"`
	RunRef        string                       `json:"run_ref"`
	GoalRef       string                       `json:"goal_ref"`
	State         orquestagoal.GoalWorkStateV0 `json:"state"`
}

func (store *StoreV0) SaveGoalWorkStateV0(
	ctx context.Context,
	state orquestagoal.GoalWorkStateV0,
) error {
	ctx = contextOrBackgroundV0(ctx)
	if err := ctx.Err(); err != nil {
		return err
	}
	normalized, err := orquestagoal.NewGoalWorkStateV0(state)
	if err != nil {
		return invalidErrorV0("app_director_goal_state", err.Error())
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	return store.saveAppDirectorGoalStateLockedV0(normalized)
}

func (store *StoreV0) LoadGoalWorkStateV0(
	ctx context.Context,
	runRef string,
) (orquestagoal.GoalWorkStateV0, error) {
	ctx = contextOrBackgroundV0(ctx)
	if err := ctx.Err(); err != nil {
		return orquestagoal.GoalWorkStateV0{}, err
	}
	runRef = normalizeRefV0(runRef)
	if runRef == "" {
		return orquestagoal.GoalWorkStateV0{}, invalidErrorV0("run_ref", "run_ref requerido")
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	document, ok, err := readJSONFileV0[appDirectorGoalStateDocumentV0](store.appDirectorGoalStatePathV0(runRef))
	if err != nil {
		return orquestagoal.GoalWorkStateV0{}, err
	}
	if !ok {
		return orquestagoal.GoalWorkStateV0{}, storeErrorV0("app_director_goal_state", "estado de goal no encontrado")
	}
	state, err := validateAppDirectorGoalStateDocumentV0(document, runRef)
	if err != nil {
		return orquestagoal.GoalWorkStateV0{}, err
	}
	return state, nil
}

func (store *StoreV0) saveAppDirectorGoalStateLockedV0(
	state orquestagoal.GoalWorkStateV0,
) error {
	runRef := normalizeRefV0(state.RunRef)
	return writeJSONAtomicV0(store.appDirectorGoalStatePathV0(runRef), appDirectorGoalStateDocumentV0{
		SchemaVersion: orquestagoal.GoalWorkStateSchemaV0,
		RunRef:        runRef,
		GoalRef:       state.GoalRef,
		State:         state,
	})
}

func validateAppDirectorGoalStateDocumentV0(
	document appDirectorGoalStateDocumentV0,
	expectedRunRef string,
) (orquestagoal.GoalWorkStateV0, error) {
	if document.SchemaVersion != orquestagoal.GoalWorkStateSchemaV0 {
		return orquestagoal.GoalWorkStateV0{}, storeErrorV0("app_director_goal_state.schema_version", "schema_version invalida")
	}
	if document.RunRef != expectedRunRef {
		return orquestagoal.GoalWorkStateV0{}, storeErrorV0("app_director_goal_state.ref", "ref inconsistente")
	}
	state, err := orquestagoal.NewGoalWorkStateV0(document.State)
	if err != nil {
		return orquestagoal.GoalWorkStateV0{}, err
	}
	if state.RunRef != expectedRunRef || state.GoalRef != document.GoalRef {
		return orquestagoal.GoalWorkStateV0{}, storeErrorV0("app_director_goal_state.state_ref", "ref interna inconsistente")
	}
	return state, nil
}
