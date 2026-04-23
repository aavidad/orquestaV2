package cmd

import (
	"testing"
	"time"

	"orquesta/agentesapp"
	"orquesta/db"
)

func TestProcesarAgentesDegradadosAutonomiaBatchDetalladoIncluyeReanimaciones(t *testing.T) {
	prepararDBTemporalCmd(t)

	prev := runtimeProcessReanimationsBatchFn
	runtimeProcessReanimationsBatchFn = func() apiRuntimeProcessReanimationsResponse {
		return apiRuntimeProcessReanimationsResponse{
			OK:              true,
			Candidates:      3,
			Reactivated:     2,
			CapacityBlocked: 1,
		}
	}
	t.Cleanup(func() {
		runtimeProcessReanimationsBatchFn = prev
	})

	got, err := procesarAgentesDegradadosAutonomiaBatchDetallado()
	if err != nil {
		t.Fatalf("procesarAgentesDegradadosAutonomiaBatchDetallado: %v", err)
	}
	if got.Count != 2 {
		t.Fatalf("deberia sumar reanimaciones automáticas al count: %+v", got)
	}
}

func TestProcesarAgentesDegradadosAutonomiaBatchDetalladoSaltaConSoloCuotaYSinHandles(t *testing.T) {
	prepararDBTemporalCmd(t)

	prev := runtimeProcessReanimationsBatchFn
	prevAgents := serverOperationalListAgentsFetcher
	prevTasks := serverOperationalCountTasksFetcher
	prevNow := statusNowFunc
	prevRowsFetcher := statusRowsFetcher
	defer func() {
		runtimeProcessReanimationsBatchFn = prev
		serverOperationalListAgentsFetcher = prevAgents
		serverOperationalCountTasksFetcher = prevTasks
		statusNowFunc = prevNow
		statusRowsFetcher = prevRowsFetcher
		resetStatusSnapshotCache()
	}()

	resetStatusSnapshotCache()
	now := time.Date(2026, 4, 24, 11, 0, 0, 0, time.UTC)
	statusNowFunc = func() time.Time { return now }
	resetAt := now.Add(20 * time.Minute)
	serverOperationalListAgentsFetcher = func() ([]*db.Agente, error) {
		return []*db.Agente{{Nombre: "Codex1", EstadoCuota: "enfriamiento", ReanimarAt: &resetAt}}, nil
	}
	serverOperationalCountTasksFetcher = func() (map[string]int, error) {
		return map[string]int{string(db.TareaEnProgreso): 2}, nil
	}
	rowsCalls := 0
	statusRowsFetcher = func() ([]agentesapp.Row, error) {
		rowsCalls++
		return nil, nil
	}
	runtimeProcessReanimationsBatchFn = func() apiRuntimeProcessReanimationsResponse {
		return apiRuntimeProcessReanimationsResponse{
			OK:          true,
			Reactivated: 2,
		}
	}

	got, err := procesarAgentesDegradadosAutonomiaBatchDetallado()
	if err != nil {
		t.Fatalf("procesarAgentesDegradadosAutonomiaBatchDetallado: %v", err)
	}
	if got.Count != 2 || got.ReactivatedWithoutRuntime != 0 || got.GhostAssignmentsCompacted != 0 || got.IdleAutoassigned != 0 {
		t.Fatalf("deberia salir pronto conservando solo reanimaciones, got=%+v", got)
	}
	if rowsCalls != 0 {
		t.Fatalf("no deberia construir rows ricas en cuota pura, calls=%d", rowsCalls)
	}
}
