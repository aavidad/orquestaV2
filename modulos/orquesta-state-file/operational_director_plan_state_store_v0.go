package orquestastatefile

import (
	"context"

	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

type operationalDirectorPlanStateDocumentV0 struct {
	SchemaVersion string                                               `json:"schema_version"`
	RunRef        string                                               `json:"run_ref"`
	PlanRef       string                                               `json:"plan_ref"`
	StateRef      string                                               `json:"state_ref"`
	State         orquestacionnucleoapp.OperationalDirectorPlanStateV0 `json:"state"`
}

func (store *StoreV0) SaveOperationalDirectorPlanStateV0(
	ctx context.Context,
	state orquestacionnucleoapp.OperationalDirectorPlanStateV0,
) error {
	ctx = contextOrBackgroundV0(ctx)
	if err := ctx.Err(); err != nil {
		return err
	}
	normalized, err := orquestacionnucleoapp.NewOperationalDirectorPlanStateV0(state)
	if err != nil {
		return invalidErrorV0("operational_director_plan_state", err.Error())
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	return store.saveOperationalDirectorPlanStateLockedV0(normalized)
}

func (store *StoreV0) LoadOperationalDirectorPlanStateV0(
	ctx context.Context,
	runRef string,
	planRef string,
) (orquestacionnucleoapp.OperationalDirectorPlanStateV0, error) {
	ctx = contextOrBackgroundV0(ctx)
	if err := ctx.Err(); err != nil {
		return orquestacionnucleoapp.OperationalDirectorPlanStateV0{}, err
	}
	runRef = normalizeRefV0(runRef)
	planRef = normalizeRefV0(planRef)
	if runRef == "" {
		return orquestacionnucleoapp.OperationalDirectorPlanStateV0{}, invalidErrorV0("run_ref", "run_ref requerido")
	}
	if planRef == "" {
		return orquestacionnucleoapp.OperationalDirectorPlanStateV0{}, invalidErrorV0("plan_ref", "plan_ref requerido")
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	document, ok, err := readJSONFileV0[operationalDirectorPlanStateDocumentV0](store.operationalDirectorPlanStatePathV0(runRef, planRef))
	if err != nil {
		return orquestacionnucleoapp.OperationalDirectorPlanStateV0{}, err
	}
	if !ok {
		return orquestacionnucleoapp.OperationalDirectorPlanStateV0{}, storeErrorV0("operational_director_plan_state", "estado de plan no encontrado")
	}
	state, err := validateOperationalDirectorPlanStateDocumentV0(document, runRef, planRef)
	if err != nil {
		return orquestacionnucleoapp.OperationalDirectorPlanStateV0{}, err
	}
	return state, nil
}

func (store *StoreV0) saveOperationalDirectorPlanStateLockedV0(
	state orquestacionnucleoapp.OperationalDirectorPlanStateV0,
) error {
	runRef := normalizeRefV0(state.RunRef)
	planRef := normalizeRefV0(state.PlanRef)
	return writeJSONAtomicV0(store.operationalDirectorPlanStatePathV0(runRef, planRef), operationalDirectorPlanStateDocumentV0{
		SchemaVersion: operationalDirectorPlanStateSchemaV0,
		RunRef:        runRef,
		PlanRef:       planRef,
		StateRef:      state.StateRef,
		State:         state,
	})
}

func validateOperationalDirectorPlanStateDocumentV0(
	document operationalDirectorPlanStateDocumentV0,
	expectedRunRef string,
	expectedPlanRef string,
) (orquestacionnucleoapp.OperationalDirectorPlanStateV0, error) {
	if document.SchemaVersion != operationalDirectorPlanStateSchemaV0 {
		return orquestacionnucleoapp.OperationalDirectorPlanStateV0{}, storeErrorV0("operational_director_plan_state.schema_version", "schema_version invalida")
	}
	if document.RunRef != expectedRunRef || document.PlanRef != expectedPlanRef {
		return orquestacionnucleoapp.OperationalDirectorPlanStateV0{}, storeErrorV0("operational_director_plan_state.ref", "ref inconsistente")
	}
	state, err := orquestacionnucleoapp.NewOperationalDirectorPlanStateV0(document.State)
	if err != nil {
		return orquestacionnucleoapp.OperationalDirectorPlanStateV0{}, err
	}
	if document.StateRef != state.StateRef ||
		state.RunRef != expectedRunRef ||
		state.PlanRef != expectedPlanRef {
		return orquestacionnucleoapp.OperationalDirectorPlanStateV0{}, storeErrorV0("operational_director_plan_state.state_ref", "ref interna inconsistente")
	}
	return state, nil
}
