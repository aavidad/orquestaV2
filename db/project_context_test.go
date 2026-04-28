package db

import (
	"testing"
	"time"
)

func TestBuildProjectContextSummaryUsaCacheCortaPorAgenteYProyecto(t *testing.T) {
	oldTTL := projectContextSummaryTTL
	oldOperacionFn := projectContextOperacionFn
	oldWorktreeFn := projectContextActiveWorktreeSummaryFn
	oldTareasFn := projectContextActiveTaskSummariesFn
	oldPropuestasFn := projectContextOpenProposalSummariesFn
	projectContextSummaryTTL = 30 * time.Second
	projectContextOperacionFn = func(int64) (*ProyectoOperacion, error) { return nil, nil }
	projectContextActiveWorktreeSummaryFn = func(string, *Proyecto) map[string]any { return nil }
	projectContextActiveTaskSummariesFn = func(string, int64) []map[string]any { return nil }
	projectContextOpenProposalSummariesFn = func(int64) []map[string]any { return nil }
	resetProjectContextSummaryCache()
	t.Cleanup(func() {
		projectContextSummaryTTL = oldTTL
		projectContextOperacionFn = oldOperacionFn
		projectContextActiveWorktreeSummaryFn = oldWorktreeFn
		projectContextActiveTaskSummariesFn = oldTareasFn
		projectContextOpenProposalSummariesFn = oldPropuestasFn
		resetProjectContextSummaryCache()
	})

	var (
		operacionCalls int
		worktreeCalls  int
		tareasCalls    int
		propuestasCall int
	)
	projectContextOperacionFn = func(int64) (*ProyectoOperacion, error) {
		operacionCalls++
		return nil, nil
	}
	projectContextActiveWorktreeSummaryFn = func(string, *Proyecto) map[string]any {
		worktreeCalls++
		return nil
	}
	projectContextActiveTaskSummariesFn = func(string, int64) []map[string]any {
		tareasCalls++
		return nil
	}
	projectContextOpenProposalSummariesFn = func(int64) []map[string]any {
		propuestasCall++
		return nil
	}

	proyecto := &Proyecto{ID: 0, Slug: "orquestador", RutaAbs: "/repo", Tipo: ProyectoRepo}
	BuildProjectContextSummary("QwenCoder1", proyecto)
	BuildProjectContextSummary("QwenCoder1", proyecto)

	if operacionCalls != 1 || worktreeCalls != 1 || tareasCalls != 1 || propuestasCall != 1 {
		t.Fatalf("cache no aplicada: operacion=%d worktree=%d tareas=%d propuestas=%d", operacionCalls, worktreeCalls, tareasCalls, propuestasCall)
	}
}
