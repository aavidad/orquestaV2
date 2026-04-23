package cmd

import (
	"context"
	"testing"
	"time"

	"orquesta/db"
	"orquesta/planocontrol"
)

func TestLifecycleHookWakeControlPlaneWarmAndInvalidateStatus(t *testing.T) {
	prepararDBTemporalCmd(t)
	resetAutonomiaActiveSessionsObservationGate()
	resetStatusSnapshotCache()
	now := time.Now().UTC()
	storeStatusSnapshotWithTTL(apiStatusResponse{
		AgentesActivos: []*db.Agente{{Nombre: "Codex1", Activo: true}},
		Generado:       now.Format(time.RFC3339),
	}, now, time.Minute)
	before := statusCacheState.expires
	if before.IsZero() {
		t.Fatal("snapshot previo no cargado")
	}
	if !allowAutonomiaActiveSessionsObservation(time.Now().UTC()) {
		t.Fatal("primer acceso al gate deberia permitirse")
	}
	if allowAutonomiaActiveSessionsObservation(time.Now().UTC()) {
		t.Fatal("segundo acceso inmediato no deberia permitirse sin reset")
	}

	runner := &planocontrol.Runner{
		Automation:            dbAutomationService{},
		ControlPlaneWarmCada:  time.Hour,
		ControlPlaneColdCada:  time.Hour,
		NotificationRetryCada: time.Hour,
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
		runner.Wait()
	}()
	runner.StartNonResidentWorker(ctx)
	unregister := registerActiveControlPlaneRunner(runner)
	defer unregister()

	db.EmitirHookCicloVida("Codex1", db.HookTaskFinish, 7, "tarea", 42, "Codex1", "fin")

	if !allowAutonomiaActiveSessionsObservation(time.Now().UTC()) {
		t.Fatal("hook lifecycle deberia despertar warm y resetear el gate")
	}
	statusCacheState.mu.Lock()
	after := statusCacheState.expires
	statusCacheState.mu.Unlock()
	if !after.Before(before) {
		t.Fatalf("el hook deberia invalidar el snapshot: before=%s after=%s", before, after)
	}
}
