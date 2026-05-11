package orquestacionnucleoapp

import (
	"context"
	"fmt"
	"strings"
	"sync"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

type InMemoryRunStoreV0 struct {
	mu   sync.RWMutex
	runs map[string]orquestacoreworkflow.OrchestrationRunV0
}

func NewInMemoryRunStoreV0(
	runs ...orquestacoreworkflow.OrchestrationRunV0,
) *InMemoryRunStoreV0 {
	store := &InMemoryRunStoreV0{runs: map[string]orquestacoreworkflow.OrchestrationRunV0{}}
	for _, run := range runs {
		store.runs[strings.TrimSpace(run.RunID)] = run
	}
	return store
}

func (store *InMemoryRunStoreV0) LoadRunV0(
	ctx context.Context,
	runRef string,
) (orquestacoreworkflow.OrchestrationRunV0, error) {
	if err := ctx.Err(); err != nil {
		return orquestacoreworkflow.OrchestrationRunV0{}, err
	}
	store.mu.RLock()
	defer store.mu.RUnlock()
	run, ok := store.runs[strings.TrimSpace(runRef)]
	if !ok {
		return orquestacoreworkflow.OrchestrationRunV0{}, RunNotFoundErrorV0{RunRef: runRef}
	}
	return run, nil
}

func (store *InMemoryRunStoreV0) SaveRunV0(
	ctx context.Context,
	run orquestacoreworkflow.OrchestrationRunV0,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	runRef := strings.TrimSpace(run.RunID)
	if runRef == "" {
		return fmt.Errorf("run_id requerido")
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	if store.runs == nil {
		store.runs = map[string]orquestacoreworkflow.OrchestrationRunV0{}
	}
	store.runs[runRef] = run
	return nil
}
