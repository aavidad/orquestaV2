package orquestastatefile

import (
	"context"
	"reflect"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
)

const materialProgressStateDocumentSchemaV0 = "orquesta_state_file.material_progress_state.v0"

type materialProgressStateDocumentV0 struct {
	SchemaVersion string                                          `json:"schema_version"`
	RunRef        string                                          `json:"run_ref"`
	GoalRef       string                                          `json:"goal_ref"`
	State         orquestaautoprogramming.MaterialProgressStateV0 `json:"state"`
}

func (store *StoreV0) LoadMaterialProgressStateV0(
	ctx context.Context,
	runRef string,
	goalRef string,
) (orquestaautoprogramming.MaterialProgressStateV0, error) {
	ctx = contextOrBackgroundV0(ctx)
	runRef, goalRef = normalizeRefV0(runRef), normalizeRefV0(goalRef)
	if err := ctx.Err(); err != nil {
		return orquestaautoprogramming.MaterialProgressStateV0{}, err
	}
	if runRef == "" || goalRef == "" {
		return orquestaautoprogramming.MaterialProgressStateV0{}, invalidErrorV0("material_progress_state.scope", "run_ref y goal_ref requeridos")
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	path := store.materialProgressStatePathV0(runRef, goalRef)
	var document materialProgressStateDocumentV0
	var found bool
	err := withProcessFileLockV0(ctx, path+".lock", func() error {
		var err error
		document, found, err = readJSONFileV0[materialProgressStateDocumentV0](path)
		return err
	})
	if err != nil {
		return orquestaautoprogramming.MaterialProgressStateV0{}, err
	}
	if !found {
		return orquestaautoprogramming.MaterialProgressStateV0{}, orquestaautoprogramming.MaterialProgressStateNotFoundErrorV0{
			RunRef:  runRef,
			GoalRef: goalRef,
		}
	}
	return validateMaterialProgressStateDocumentV0(document, runRef, goalRef)
}

func (store *StoreV0) CompareAndSwapMaterialProgressStateV0(
	ctx context.Context,
	expectedVersion uint64,
	state orquestaautoprogramming.MaterialProgressStateV0,
) (orquestaautoprogramming.MaterialProgressStateV0, error) {
	ctx = contextOrBackgroundV0(ctx)
	if err := ctx.Err(); err != nil {
		return orquestaautoprogramming.MaterialProgressStateV0{}, err
	}
	state = orquestaautoprogramming.NormalizeMaterialProgressStateV0(state)
	if state.RunRef == "" || state.GoalRef == "" {
		return orquestaautoprogramming.MaterialProgressStateV0{}, invalidErrorV0("material_progress_state.scope", "run_ref y goal_ref requeridos")
	}
	state.StoreVersion = expectedVersion + 1
	if validation := orquestaautoprogramming.ValidateMaterialProgressStateV0(state); !validation.Accepted {
		return orquestaautoprogramming.MaterialProgressStateV0{}, invalidErrorV0("material_progress_state", "estado invalido")
	} else {
		state = validation.State
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	path := store.materialProgressStatePathV0(state.RunRef, state.GoalRef)
	var saved orquestaautoprogramming.MaterialProgressStateV0
	err := withProcessFileLockV0(ctx, path+".lock", func() error {
		document, found, err := readJSONFileV0[materialProgressStateDocumentV0](path)
		if err != nil {
			return err
		}
		if !found {
			if expectedVersion != 0 {
				return materialProgressStateConflictV0(state, expectedVersion, 0)
			}
			saved = state
			return store.writeMaterialProgressStateV0(path, saved)
		}
		existing, err := validateMaterialProgressStateDocumentV0(document, state.RunRef, state.GoalRef)
		if err != nil {
			return err
		}
		if reflect.DeepEqual(existing, state) {
			saved = existing
			return nil
		}
		if existing.StoreVersion != expectedVersion || materialProgressStateRegressesV0(existing, state) {
			return materialProgressStateConflictV0(state, expectedVersion, existing.StoreVersion)
		}
		saved = state
		return store.writeMaterialProgressStateV0(path, saved)
	})
	if err != nil {
		return orquestaautoprogramming.MaterialProgressStateV0{}, err
	}
	return saved, nil
}

func (store *StoreV0) writeMaterialProgressStateV0(path string, state orquestaautoprogramming.MaterialProgressStateV0) error {
	return writeJSONAtomicV0(path, materialProgressStateDocumentV0{
		SchemaVersion: materialProgressStateDocumentSchemaV0,
		RunRef:        state.RunRef,
		GoalRef:       state.GoalRef,
		State:         state,
	})
}

func validateMaterialProgressStateDocumentV0(
	document materialProgressStateDocumentV0,
	runRef string,
	goalRef string,
) (orquestaautoprogramming.MaterialProgressStateV0, error) {
	if document.SchemaVersion != materialProgressStateDocumentSchemaV0 ||
		normalizeRefV0(document.RunRef) != runRef || normalizeRefV0(document.GoalRef) != goalRef {
		return orquestaautoprogramming.MaterialProgressStateV0{}, storeErrorV0("material_progress_state", "documento persistido inconsistente")
	}
	validation := orquestaautoprogramming.ValidateMaterialProgressStateV0(document.State)
	if !validation.Accepted || validation.State.RunRef != runRef || validation.State.GoalRef != goalRef {
		return orquestaautoprogramming.MaterialProgressStateV0{}, storeErrorV0("material_progress_state", "estado persistido invalido")
	}
	return validation.State, nil
}

func materialProgressStateRegressesV0(
	current orquestaautoprogramming.MaterialProgressStateV0,
	next orquestaautoprogramming.MaterialProgressStateV0,
) bool {
	if !reflect.DeepEqual(next.Policy, current.Policy) ||
		next.BaselineRef != current.BaselineRef ||
		next.WriteSetSHA256 != current.WriteSetSHA256 ||
		next.ContextRevisionRef != current.ContextRevisionRef {
		return true
	}
	if next.LastCheckpoint.Sequence < current.LastCheckpoint.Sequence ||
		next.LastCheckpoint.TokensAccumulated < current.LastCheckpoint.TokensAccumulated {
		return true
	}
	return next.LastCheckpoint.Sequence == current.LastCheckpoint.Sequence
}

func materialProgressStateConflictV0(
	state orquestaautoprogramming.MaterialProgressStateV0,
	expectedVersion uint64,
	actualVersion uint64,
) error {
	return orquestaautoprogramming.MaterialProgressStateCASConflictErrorV0{
		RunRef: state.RunRef, GoalRef: state.GoalRef,
		ExpectedVersion: expectedVersion, ActualVersion: actualVersion,
	}
}
