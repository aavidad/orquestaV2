package orquestastatefile

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

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
	_, err := store.CompareAndSwapGoalWorkStateV0(ctx, state.StoreVersion, state)
	return err
}

func (store *StoreV0) CompareAndSwapGoalWorkStateV0(
	ctx context.Context,
	expectedVersion uint64,
	state orquestagoal.GoalWorkStateV0,
) (orquestagoal.GoalWorkStateV0, error) {
	ctx = contextOrBackgroundV0(ctx)
	if err := ctx.Err(); err != nil {
		return orquestagoal.GoalWorkStateV0{}, err
	}
	normalized, err := orquestagoal.NewGoalWorkStateV0(state)
	if err != nil {
		return orquestagoal.GoalWorkStateV0{}, invalidErrorV0("app_director_goal_state", err.Error())
	}
	if normalized.StoreVersion != expectedVersion {
		return orquestagoal.GoalWorkStateV0{}, storeErrorV0("app_director_goal_state.store_version", "CAS expected_version inconsistente")
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	path := store.appDirectorGoalStatePathV0(normalized.RunRef)
	err = withProcessFileLockV0(ctx, path+".lock", func() error {
		var saveErr error
		normalized, saveErr = store.saveAppDirectorGoalStateCASLockedV0(normalized, expectedVersion)
		return saveErr
	})
	return normalized, err
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
	path := store.appDirectorGoalStatePathV0(runRef)
	var document appDirectorGoalStateDocumentV0
	var ok bool
	err := withProcessFileLockV0(ctx, path+".lock", func() error {
		var readErr error
		document, ok, readErr = readJSONFileV0[appDirectorGoalStateDocumentV0](path)
		return readErr
	})
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

func (store *StoreV0) ListGoalWorkStatesV0(
	ctx context.Context,
	request orquestagoal.GoalWorkStateListRequestV0,
) ([]orquestagoal.GoalWorkStateV0, error) {
	ctx = contextOrBackgroundV0(ctx)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	request = orquestagoal.NormalizeGoalWorkStateListRequestV0(request)
	store.mu.Lock()
	defer store.mu.Unlock()
	var states []orquestagoal.GoalWorkStateV0
	var err error
	if len(request.RunRefs) > 0 {
		states, err = store.listAppDirectorGoalStatesByRunRefsLockedV0(ctx, request)
	} else {
		states, err = store.listAppDirectorGoalStatesFromDirLockedV0(ctx, request)
	}
	if err != nil {
		return nil, err
	}
	sort.Slice(states, func(i, j int) bool {
		return strings.TrimSpace(states[i].RunRef) < strings.TrimSpace(states[j].RunRef)
	})
	if request.MaxItems > 0 && len(states) > request.MaxItems {
		states = states[:request.MaxItems]
	}
	if states == nil {
		return []orquestagoal.GoalWorkStateV0{}, nil
	}
	return states, nil
}

func (store *StoreV0) saveAppDirectorGoalStateCASLockedV0(
	state orquestagoal.GoalWorkStateV0,
	expectedVersion uint64,
) (orquestagoal.GoalWorkStateV0, error) {
	runRef := normalizeRefV0(state.RunRef)
	if existingDocument, ok, err := readJSONFileV0[appDirectorGoalStateDocumentV0](store.appDirectorGoalStatePathV0(runRef)); err != nil {
		return orquestagoal.GoalWorkStateV0{}, err
	} else if ok {
		existing, err := validateAppDirectorGoalStateDocumentV0(existingDocument, runRef)
		if err != nil {
			return orquestagoal.GoalWorkStateV0{}, err
		}
		if !reflect.DeepEqual(existing.Spec, state.Spec) {
			return orquestagoal.GoalWorkStateV0{}, storeErrorV0("app_director_goal_state.spec", "goal spec congelada no puede cambiar")
		}
		if existing.StoreVersion != expectedVersion {
			return orquestagoal.GoalWorkStateV0{}, orquestagoal.GoalWorkStateCASConflictErrorV0{
				RunRef: runRef, ExpectedVersion: expectedVersion, CurrentVersion: existing.StoreVersion,
			}
		}
	} else if expectedVersion != 0 {
		return orquestagoal.GoalWorkStateV0{}, orquestagoal.GoalWorkStateCASConflictErrorV0{
			RunRef: runRef, ExpectedVersion: expectedVersion,
		}
	}
	state.StoreVersion = expectedVersion + 1
	err := writeJSONAtomicV0(store.appDirectorGoalStatePathV0(runRef), appDirectorGoalStateDocumentV0{
		SchemaVersion: orquestagoal.GoalWorkStateSchemaV0,
		RunRef:        runRef,
		GoalRef:       state.GoalRef,
		State:         state,
	})
	return state, err
}

func (store *StoreV0) listAppDirectorGoalStatesByRunRefsLockedV0(
	ctx context.Context,
	request orquestagoal.GoalWorkStateListRequestV0,
) ([]orquestagoal.GoalWorkStateV0, error) {
	out := make([]orquestagoal.GoalWorkStateV0, 0, len(request.RunRefs))
	for _, runRef := range request.RunRefs {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		state, ok, err := store.readAppDirectorGoalStateFileLockedV0(runRef)
		if err != nil {
			return nil, err
		}
		if !ok || !orquestagoal.GoalWorkStateMatchesListRequestV0(state, request) {
			continue
		}
		out = append(out, state)
	}
	return out, nil
}

func (store *StoreV0) listAppDirectorGoalStatesFromDirLockedV0(
	ctx context.Context,
	request orquestagoal.GoalWorkStateListRequestV0,
) ([]orquestagoal.GoalWorkStateV0, error) {
	dir := filepath.Join(store.rootDir, appDirectorGoalStatesDirV0)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	out := make([]orquestagoal.GoalWorkStateV0, 0, len(entries))
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		document, ok, err := readJSONFileV0[appDirectorGoalStateDocumentV0](filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		state, err := validateAppDirectorGoalStateDocumentV0(document, document.RunRef)
		if err != nil {
			return nil, err
		}
		if !orquestagoal.GoalWorkStateMatchesListRequestV0(state, request) {
			continue
		}
		out = append(out, state)
	}
	return out, nil
}

func (store *StoreV0) readAppDirectorGoalStateFileLockedV0(
	runRef string,
) (orquestagoal.GoalWorkStateV0, bool, error) {
	runRef = normalizeRefV0(runRef)
	if runRef == "" {
		return orquestagoal.GoalWorkStateV0{}, false, nil
	}
	document, ok, err := readJSONFileV0[appDirectorGoalStateDocumentV0](store.appDirectorGoalStatePathV0(runRef))
	if err != nil || !ok {
		return orquestagoal.GoalWorkStateV0{}, ok, err
	}
	state, err := validateAppDirectorGoalStateDocumentV0(document, runRef)
	if err != nil {
		return orquestagoal.GoalWorkStateV0{}, false, err
	}
	return state, true, nil
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
