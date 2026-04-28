package cmd

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"orquesta/db"
)

func TestImprimirWorkspaceControlMuestraResumenGlobal(t *testing.T) {
	buf := &bytes.Buffer{}

	now := time.Date(2026, 4, 24, 16, 0, 0, 0, time.UTC)
	report := &workspaceControlReport{
		ActiveProjects:      1,
		TaskCounts:          map[string]int{"en_progreso": 2},
		DeudaDispatch:       deudaDispatchResumen{Total: 1, Pendientes: 1},
		Autonomia:           autonomiaResumen{Supervisando: 1, Count: 2},
		WorkersConectados:   2,
		WorkersTrabajando:   1,
		SupervisoresActivos: 1,
		AutonomyHighlights:  []string{"task_reassigned=2", "integracion_bloqueada=15", "riesgo_top=orquestador(15)", "último 2026-04-24 16:00:00"},
		AutonomySurface: &autonomySurface{
			Events:     2,
			LastAt:     &now,
			ByKind:     map[string]int{"task_reassigned": 2},
			Highlights: []string{"task_reassigned=2", "último 2026-04-24 16:00:00"},
			Projects: []autonomyProjectSurface{{
				Project:    "orquestador",
				Events:     2,
				LastAt:     &now,
				ByKind:     map[string]int{"task_reassigned": 2},
				Highlights: []string{"task_reassigned=2"},
			}},
		},
		AutonomyRecent: []autonomySurfaceRecentItem{{
			Project: "orquestador",
			autonomyEventSummary: autonomyEventSummary{
				Kind:        "task_reassigned",
				TargetAgent: "Codex1",
				CreatedAt:   now,
			},
		}},
		AutonomyProjects: []workspaceAutonomyProjectSummary{{
			Project:    "orquestador",
			Events:     2,
			Blocking:   15,
			LastAt:     &now,
			Highlights: []string{"task_reassigned=2", "riesgo=critico", "integracion_bloqueada=15", "bloqueadas=1", "review_gates=1", "runtime_orders=1", "mailbox_rt=1", "propuestas_abiertas=1"},
		}},
		Projects: []*apiProyectoCockpit{{
			Proyecto:                &db.Proyecto{Slug: "orquestador"},
			AutonomyEvents:          2,
			RuntimeMailboxPendiente: 1,
			RuntimeOrdersAbiertas:   1,
		}},
	}

	if err := renderWorkspaceControl(buf, report); err != nil {
		t.Fatalf("renderWorkspaceControl: %v", err)
	}
	out := buf.String()
	for _, token := range []string{"Workspace:  proyectos=1", "Workers:    conectados=2", "Autonomy:", "Highlights: task_reassigned=2 | integracion_bloqueada=15 | riesgo_top=orquestador(15)", "Resumen por proyecto:", "Timeline:", "task_reassigned · destino=Codex1", "orquestador: 2 evento(s) · integracion_bloqueada=15", "riesgo=critico | integracion_bloqueada=15 | bloqueadas=1 | review_gates=1 | runtime_orders=1 | mailbox_rt=1 | propuestas_abiertas=1", "Proyecto:   orquestador"} {
		if !strings.Contains(out, token) {
			t.Fatalf("falta %q en salida: %s", token, out)
		}
	}
}

func TestImprimirWorkspaceControlMuestraResumenCompactoSinAutonomySurface(t *testing.T) {
	buf := &bytes.Buffer{}

	now := time.Date(2026, 4, 24, 16, 0, 0, 0, time.UTC)
	report := &workspaceControlReport{
		ActiveProjects: 1,
		AutonomyProjects: []workspaceAutonomyProjectSummary{{
			Project:    "orquestador",
			Events:     1,
			Blocking:   5,
			LastAt:     &now,
			Highlights: []string{"task_reassigned=1", "riesgo=alto", "integracion_bloqueada=5", "bloqueadas=1"},
		}},
	}

	if err := renderWorkspaceControl(buf, report); err != nil {
		t.Fatalf("renderWorkspaceControl: %v", err)
	}
	out := buf.String()
	for _, token := range []string{"Resumen por proyecto:", "orquestador: 1 evento(s) · integracion_bloqueada=5", "task_reassigned=1", "riesgo=alto", "integracion_bloqueada=5", "bloqueadas=1"} {
		if !strings.Contains(out, token) {
			t.Fatalf("falta %q en salida: %s", token, out)
		}
	}
}
