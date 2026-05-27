package orquestaweb

import "testing"

func TestNewWebDirectorStatsPanelV0MantieneActividadPorProcesoRegistrado(t *testing.T) {
	panel := NewWebDirectorStatsPanelV0("es", WebDirectorStatsInboundResultV0{
		Estado: WebDirectorStatsInboundEstadoOKV0,
		RunRef: "run-ref-web-live-process-001",
		Stats: &WebDirectorRunStatsContractV0{
			RunRef: "run-ref-web-live-process-001",
			Status: "running",
			Counts: map[string]int{
				"tasks_total":  1,
				"tasks_closed": 0,
			},
			Progress: WebDirectorProgressStatsContractV0{
				SourceStatus:    "not_configured",
				PercentComplete: 1,
				TasksTotal:      1,
				TasksClosed:     0,
				TasksObserved:   1,
				Tasks: []WebDirectorTaskProgressV0{{
					TaskRef:        "task-ref-web-live-process-001",
					Status:         "in_progress",
					AgentRequestID: "agent-ref-task-ref-web-live-process-001",
					ProgressStatus: "process_registered",
				}},
			},
		},
	})

	if panel.Progress.PercentComplete != 1 ||
		panel.Resumen.PercentComplete != 1 ||
		panel.Resumen.TasksClosed != 0 ||
		panel.Resumen.TasksObserved != 1 ||
		len(panel.Tasks) != 1 ||
		panel.Tasks[0].ProgressStatus != "process_registered" {
		t.Fatalf("panel=%+v", panel)
	}
}
