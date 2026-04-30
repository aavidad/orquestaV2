package cmd

import (
	"fmt"
	"testing"
	"time"

	"orquesta/db"
)

func TestApplySupervisorNextActionUsaSnapshotCanonicoDelOperador(t *testing.T) {
	prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar Codex1: %v", err)
	}
	if err := db.RegistrarAgente("Codex2", "programador"); err != nil {
		t.Fatalf("registrar Codex2: %v", err)
	}
	proyectoID := insertTestProyecto(t, "orquestador", "orquestador", t.TempDir())
	taskID, err := db.CrearTarea(&db.Tarea{
		Titulo:     "Cerrar arquitectura hexagonal y modular de Orquestador",
		ProyectoID: &proyectoID,
		Estado:     db.TareaLibre,
		Prioridad:  db.PrioridadAlta,
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := tareasService.Take(taskID, "Codex1"); err != nil {
		t.Fatalf("reservar tarea: %v", err)
	}

	prevBuilder := supervisorActionSnapshotBuilder
	prevStatusService := statusService
	prevNow := statusNowFunc
	defer func() {
		supervisorActionSnapshotBuilder = prevBuilder
		statusService = prevStatusService
		statusNowFunc = prevNow
	}()

	now := time.Now().UTC()
	statusNowFunc = func() time.Time { return now }
	statusService = stubStatusService{response: apiStatusResponse{}}
	storeStatusSnapshotWithTTL(apiStatusResponse{}, now, 5*time.Minute)

	supervisorActionSnapshotBuilder = func(supervisor string) (map[string]any, error) {
		return map[string]any{
			"safe_action_queue": []supervisorRecommendedAction{{
				Kind:     "dispatch",
				Target:   fmt.Sprintf("tarea:%d", taskID),
				Action:   "rebalancear_reserva",
				Assignee: "Codex2",
				Priority: "media",
			}},
			"next_safe_action": supervisorRecommendedAction{
				Kind:     "dispatch",
				Target:   fmt.Sprintf("tarea:%d", taskID),
				Action:   "rebalancear_reserva",
				Assignee: "Codex2",
				Priority: "media",
			},
		}, nil
	}

	result, err := applySupervisorNextAction("OpenClaw")
	if err != nil {
		t.Fatalf("applySupervisorNextAction: %v", err)
	}
	if got, _ := result["task_id"].(int64); got != taskID {
		t.Fatalf("task_id inesperado: %#v", result)
	}
	tarea, err := db.GetTarea(taskID)
	if err != nil {
		t.Fatalf("get tarea: %v", err)
	}
	if tarea == nil || tarea.Agente == nil || *tarea.Agente != "Codex2" || tarea.Estado != db.TareaAsignada {
		t.Fatalf("la tarea no se rebalanceo segun snapshot canonico: %+v", tarea)
	}
}

func TestBuildSupervisorActionSnapshotOmiteSafeQueueSiHayRecoveryPendiente(t *testing.T) {
	prevBuilder := supervisorActionSnapshotBuilder
	t.Cleanup(func() {
		supervisorActionSnapshotBuilder = prevBuilder
	})

	supervisorActionSnapshotBuilder = func(supervisor string) (map[string]any, error) {
		return map[string]any{
			"action_queue": []supervisorRecommendedAction{{
				Kind:     "dispatch",
				Target:   "backlog:libre",
				Action:   "asignar_tarea_libre",
				Assignee: "codex13",
				Priority: "media",
			}},
			"safe_action_queue": []supervisorRecommendedAction{{
				Kind:     "dispatch",
				Target:   "backlog:libre",
				Action:   "asignar_tarea_libre",
				Assignee: "codex13",
				Priority: "media",
			}},
			"next_safe_action": supervisorRecommendedAction{
				Kind:     "dispatch",
				Target:   "backlog:libre",
				Action:   "asignar_tarea_libre",
				Assignee: "codex13",
				Priority: "media",
			},
			"server_operational": serverOperationalInfo{
				State:       "ready",
				Operational: true,
				Reason:      "control_plane_responsive",
				Recovery: &serverOperationalRecoveryHint{
					Kind:            "compaction_debt",
					SuggestedAction: "compact_or_reassign_active_tasks",
				},
			},
		}, nil
	}

	snapshot, err := buildSupervisorActionSnapshot("OpenClaw")
	if err != nil {
		t.Fatalf("buildSupervisorActionSnapshot: %v", err)
	}
	safe, _ := snapshot["safe_action_queue"].([]supervisorRecommendedAction)
	if len(safe) != 0 {
		t.Fatalf("safe_action_queue deberia vaciarse con recovery pendiente: %#v", safe)
	}
	if nextSafe, ok := snapshot["next_safe_action"].(supervisorRecommendedAction); ok {
		t.Fatalf("next_safe_action deberia quedar nil con recovery pendiente: %#v", nextSafe)
	}
}
