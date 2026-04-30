package cmd

import (
	"testing"

	"orquesta/db"
)

func TestApplySupervisorRecommendedActionInternalToleraReservaStaleDesdeSafeQueue(t *testing.T) {
	withTempOrquestaDB(t, func() {
		taskID, err := db.CrearTarea(&db.Tarea{
			Titulo:      "Reserva stale",
			Descripcion: "La cola llega tarde",
			Modulo:      "openclaw",
			Prioridad:   db.PrioridadAlta,
			CreadoPor:   "test",
		})
		if err != nil {
			t.Fatalf("crear tarea: %v", err)
		}
		if err := db.RegistrarAgente("Codex4", "programador"); err != nil {
			t.Fatalf("registrar agente: %v", err)
		}
		if _, err := db.DB.Exec(`UPDATE tareas SET estado='asignada', agente='Codex4' WHERE id=?`, taskID); err != nil {
			t.Fatalf("preasignar tarea: %v", err)
		}

		prev := supervisorActionSnapshotBuilder
		supervisorActionSnapshotBuilder = func(supervisor string) (map[string]any, error) {
			action := supervisorRecommendedAction{
				Kind:     "dispatch",
				Target:   "tarea:" + itoa(taskID),
				Action:   "reservar_tarea_libre",
				Assignee: "Codex4",
			}
			return map[string]any{
				"safe_action_queue": []supervisorRecommendedAction{action},
				"next_safe_action":  action,
			}, nil
		}
		t.Cleanup(func() { supervisorActionSnapshotBuilder = prev })

		result, err := applySupervisorRecommendedActionInternal("OpenClaw", "reservar_tarea_libre", "tarea:"+itoa(taskID), "Codex4")
		if err != nil {
			t.Fatalf("apply stale reserve: %v", err)
		}
		if ok, _ := result["ok"].(bool); !ok {
			t.Fatalf("resultado no-ok: %#v", result)
		}
		if noop, _ := result["noop"].(bool); !noop {
			t.Fatalf("deberia marcar noop: %#v", result)
		}
		if stale, _ := result["stale_action"].(bool); !stale {
			t.Fatalf("deberia marcar stale_action: %#v", result)
		}
	})
}

func TestApplySupervisorRecommendedActionInternalToleraRebalanceoStaleDesdeSafeQueue(t *testing.T) {
	withTempOrquestaDB(t, func() {
		taskID, err := db.CrearTarea(&db.Tarea{
			Titulo:      "Rebalanceo stale",
			Descripcion: "La reserva ya cambió",
			Modulo:      "openclaw",
			Prioridad:   db.PrioridadAlta,
			CreadoPor:   "test",
		})
		if err != nil {
			t.Fatalf("crear tarea: %v", err)
		}
		if err := db.RegistrarAgente("Codex3", "programador"); err != nil {
			t.Fatalf("registrar Codex3: %v", err)
		}
		if err := db.RegistrarAgente("Codex4", "programador"); err != nil {
			t.Fatalf("registrar Codex4: %v", err)
		}
		if _, err := db.DB.Exec(`UPDATE tareas SET estado='en_progreso', agente='Codex4' WHERE id=?`, taskID); err != nil {
			t.Fatalf("poner tarea en progreso: %v", err)
		}

		prev := supervisorActionSnapshotBuilder
		supervisorActionSnapshotBuilder = func(supervisor string) (map[string]any, error) {
			action := supervisorRecommendedAction{
				Kind:     "dispatch",
				Target:   "tarea:" + itoa(taskID),
				Action:   "rebalancear_reserva",
				Assignee: "Codex3",
			}
			return map[string]any{
				"safe_action_queue": []supervisorRecommendedAction{action},
				"next_safe_action":  action,
			}, nil
		}
		t.Cleanup(func() { supervisorActionSnapshotBuilder = prev })

		result, err := applySupervisorRecommendedActionInternal("OpenClaw", "rebalancear_reserva", "tarea:"+itoa(taskID), "Codex3")
		if err != nil {
			t.Fatalf("apply stale rebalance: %v", err)
		}
		if ok, _ := result["ok"].(bool); !ok {
			t.Fatalf("resultado no-ok: %#v", result)
		}
		if noop, _ := result["noop"].(bool); !noop {
			t.Fatalf("deberia marcar noop: %#v", result)
		}
		if stale, _ := result["stale_action"].(bool); !stale {
			t.Fatalf("deberia marcar stale_action: %#v", result)
		}
	})
}

func TestApplySupervisorRecommendedActionInternalToleraSafeActionSinAssigneeDesdeQueue(t *testing.T) {
	prev := supervisorActionSnapshotBuilder
	supervisorActionSnapshotBuilder = func(supervisor string) (map[string]any, error) {
		action := supervisorRecommendedAction{
			Kind:   "dispatch",
			Target: "backlog:libre",
			Action: "reservar_tarea_libre",
		}
		return map[string]any{
			"safe_action_queue": []supervisorRecommendedAction{action},
			"next_safe_action":  action,
		}, nil
	}
	t.Cleanup(func() { supervisorActionSnapshotBuilder = prev })

	result, err := applySupervisorRecommendedActionInternal("OpenClaw", "reservar_tarea_libre", "backlog:libre", "")
	if err != nil {
		t.Fatalf("apply safe action sin assignee: %v", err)
	}
	if ok, _ := result["ok"].(bool); !ok {
		t.Fatalf("resultado no-ok: %#v", result)
	}
	if noop, _ := result["noop"].(bool); !noop {
		t.Fatalf("deberia marcar noop: %#v", result)
	}
	if stale, _ := result["stale_action"].(bool); !stale {
		t.Fatalf("deberia marcar stale_action: %#v", result)
	}
}
