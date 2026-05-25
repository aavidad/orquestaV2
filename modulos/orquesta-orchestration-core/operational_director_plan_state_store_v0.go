package orquestacionnucleoapp

import (
	"context"
	"strings"
	"sync"
)

type InMemoryOperationalDirectorPlanStateStoreV0 struct {
	mu     sync.Mutex
	states map[string]map[string]OperationalDirectorPlanStateV0
}

var _ OperationalDirectorPlanStateWriterPortV0 = (*InMemoryOperationalDirectorPlanStateStoreV0)(nil)
var _ OperationalDirectorPlanStateStorePortV0 = (*InMemoryOperationalDirectorPlanStateStoreV0)(nil)

func NewInMemoryOperationalDirectorPlanStateStoreV0(
	states ...OperationalDirectorPlanStateV0,
) *InMemoryOperationalDirectorPlanStateStoreV0 {
	store := &InMemoryOperationalDirectorPlanStateStoreV0{
		states: map[string]map[string]OperationalDirectorPlanStateV0{},
	}
	for _, state := range states {
		_ = store.saveOperationalDirectorPlanStateV0(state)
	}
	return store
}

func (store *InMemoryOperationalDirectorPlanStateStoreV0) SaveOperationalDirectorPlanStateV0(
	ctx context.Context,
	state OperationalDirectorPlanStateV0,
) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	return store.saveOperationalDirectorPlanStateV0(state)
}

func (store *InMemoryOperationalDirectorPlanStateStoreV0) LoadOperationalDirectorPlanStateV0(
	ctx context.Context,
	runRef string,
	planRef string,
) (OperationalDirectorPlanStateV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return OperationalDirectorPlanStateV0{}, err
	}
	runRef = strings.TrimSpace(runRef)
	planRef = strings.TrimSpace(planRef)
	if runRef == "" {
		return OperationalDirectorPlanStateV0{}, errorV0(ErrNucleoOrquestacionInvalidoV0, "run_ref", "run_ref requerido")
	}
	if planRef == "" {
		return OperationalDirectorPlanStateV0{}, errorV0(ErrNucleoOrquestacionInvalidoV0, "plan_ref", "plan_ref requerido")
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	state, ok := store.states[runRef][planRef]
	if !ok {
		return OperationalDirectorPlanStateV0{}, errorV0(ErrNucleoOrquestacionStoreV0, "operational_director_plan_state", "estado de plan no encontrado")
	}
	return state, nil
}

func (store *InMemoryOperationalDirectorPlanStateStoreV0) saveOperationalDirectorPlanStateV0(
	state OperationalDirectorPlanStateV0,
) error {
	normalized, err := NewOperationalDirectorPlanStateV0(state)
	if err != nil {
		return err
	}
	if store.states[normalized.RunRef] == nil {
		store.states[normalized.RunRef] = map[string]OperationalDirectorPlanStateV0{}
	}
	store.states[normalized.RunRef][normalized.PlanRef] = normalized
	return nil
}
