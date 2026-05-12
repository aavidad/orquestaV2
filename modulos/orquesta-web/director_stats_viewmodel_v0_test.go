package orquestaweb

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestWebDirectorStatsPanelV0ProyectaRunCountsProgresoYErrores(t *testing.T) {
	result := directorStatsResultForWebTestV0()
	result.Errores = []WebDirectorStatsPublicIssueV0{{
		Code:    "run_parcial",
		Field:   "run_ref",
		Message: "Vista parcial",
	}}
	result.Stats.Progress.Issues = []WebDirectorStatsPublicIssueV0{{
		Code:  "progress_source_error",
		Field: "progress_source",
	}}

	panel := NewWebDirectorStatsPanelV0("es", result)

	if panel.SchemaVersion != WebDirectorStatsPanelSchemaV0 ||
		panel.Locale != "es-ES" ||
		panel.Estado != WebDirectorStatsInboundEstadoOKV0 ||
		panel.RunRef != "run-ref-web-stats-001" ||
		panel.Resumen.PercentComplete != 50 ||
		panel.Resumen.ProgressingAgents != 1 ||
		panel.Resumen.UsageTotalTokens != 1500 {
		t.Fatalf("panel=%+v", panel)
	}
	if panel.Counts.TasksTotal != 2 ||
		panel.Counts.TasksDelivered != 1 ||
		panel.Counts.Brainstorms != 1 ||
		panel.Counts.AgentsStarted != 1 ||
		panel.Counts.AgentsDelivered != 1 {
		t.Fatalf("counts=%+v", panel.Counts)
	}
	if len(panel.Tareas) != 1 ||
		panel.Tareas[0].TaskRef != "task-ref-web-stats-001" ||
		panel.Tareas[0].AgentRequestID != "agent-ref-web-stats-001" {
		t.Fatalf("tareas=%+v", panel.Tareas)
	}
	if len(panel.Agentes) != 1 ||
		panel.Agentes[0].TaskRef != "task-ref-web-stats-001" ||
		panel.Agentes[0].ProgressStatus != "progressing" ||
		!panel.Agentes[0].InFlight ||
		panel.Agentes[0].NoProgressTicks != 2 ||
		panel.Agentes[0].RepeatedActionCount != 1 ||
		panel.Agentes[0].ModelAlias != "gpt-5.5" ||
		panel.Agentes[0].QuotaRemaining != 90 ||
		panel.Agentes[0].TotalTokens != 1500 {
		t.Fatalf("agentes=%+v", panel.Agentes)
	}
	if len(panel.ErroresPublicos) != 2 ||
		panel.ErroresPublicos[1].Code != "progress_source_error" {
		t.Fatalf("errores=%+v", panel.ErroresPublicos)
	}
}

func TestWebDirectorStatsPanelV0NoExponeDetallesDeProcesoRuntimeNiProveedor(t *testing.T) {
	raw, err := json.Marshal(NewWebDirectorStatsPanelV0("en-US", directorStatsResultForWebTestV0()))
	if err != nil {
		t.Fatalf("marshal panel: %v", err)
	}
	got := strings.ToLower(string(raw))
	for _, forbidden := range []string{
		"process_ref",
		"session_ref",
		"runtime_provider",
		"provider",
		"oauth",
		"home",
		"transcript",
		"dsn",
		"sql",
	} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("panel expone detalle prohibido %q: %s", forbidden, got)
		}
	}
}

func TestWebDirectorStatsPanelV0ErroresPublicosSinStats(t *testing.T) {
	panel := NewWebDirectorStatsPanelV0("en", WebDirectorStatsInboundResultV0{
		Estado: WebDirectorStatsInboundEstadoErrorV0,
		RunRef: "run-ref-missing",
		Errores: []WebDirectorStatsPublicIssueV0{{
			Code:  "run_no_disponible",
			Field: "run_ref",
		}},
	})

	if panel.Estado != WebDirectorStatsInboundEstadoErrorV0 ||
		panel.Locale != "en-US" ||
		panel.Textos.Title != "Director statistics" ||
		len(panel.ErroresPublicos) != 1 ||
		panel.ErroresPublicos[0].Code != "run_no_disponible" {
		t.Fatalf("panel=%+v", panel)
	}
	if panel.Counts.TasksTotal != 0 || len(panel.Tareas) != 0 || len(panel.Agentes) != 0 {
		t.Fatalf("no debe inventar stats: %+v", panel)
	}
}

func directorStatsResultForWebTestV0() WebDirectorStatsInboundResultV0 {
	return WebDirectorStatsInboundResultV0{
		Estado:        WebDirectorStatsInboundEstadoOKV0,
		RequestID:     "request-ref-web-stats-001",
		CorrelationID: "corr-web-stats-001",
		RunRef:        "run-ref-web-stats-001",
		Stats: &WebDirectorRunStatsContractV0{
			SchemaVersion: "director_run_stats.v0",
			RunRef:        "run-ref-web-stats-001",
			Status:        "running",
			CurrentPhase:  "programacion",
			Counts: map[string]int{
				"tasks_total":      2,
				"tasks_closed":     1,
				"tasks_delivered":  1,
				"brainstorms":      1,
				"agents_started":   1,
				"agents_delivered": 1,
			},
			Progress: WebDirectorProgressStatsContractV0{
				SourceStatus:      "loaded",
				PercentComplete:   50,
				TasksTotal:        2,
				TasksClosed:       1,
				TasksObserved:     1,
				ObservedAgents:    1,
				ProgressingAgents: 1,
				Tasks: []WebDirectorTaskProgressV0{{
					TaskRef:        " task-ref-web-stats-001 ",
					Status:         "in_progress",
					AgentRequestID: " agent-ref-web-stats-001 ",
					LastReportRef:  "report-ref-web-stats-001",
					EvidenceRefs:   []string{"evidence-ref-web-stats-001", "evidence-ref-web-stats-001"},
				}},
			},
			UsageSummary: &WebDirectorRunUsageSummaryV0{
				AgentsObserved: 1,
				QuotaStatus:    "available",
				TotalTokens:    1500,
				CostMicros:     500,
			},
			Agents: []WebDirectorAgentStatsContractV0{{
				AgentRequestID: "agent-ref-web-stats-001",
				Status:         "running",
				InFlight:       true,
				LastProgress: &WebDirectorAgentProgressV0{
					TaskRef:             "task-ref-web-stats-001",
					ReportRef:           " report-ref-web-stats-001 ",
					Status:              "progressing",
					NoProgressTicks:     2,
					RepeatedActionCount: 1,
				},
				Usage: &WebDirectorAgentUsageV0{
					ModelAlias:     " gpt-5.5 ",
					CapacityLevel:  "xhigh",
					QuotaStatus:    "available",
					QuotaRemaining: 90,
					QuotaLimit:     100,
					TotalTokens:    1500,
				},
			}},
		},
	}
}
