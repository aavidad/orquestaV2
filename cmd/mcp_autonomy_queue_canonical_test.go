package cmd

import (
	"strings"
	"testing"
	"time"

	"orquesta/agentesapp"
	"orquesta/db"
)

func TestBuildSupervisorOperationalActionsOmiteReinicioRuntimeSiFilaCanonicaYaConvergioSinTarea(t *testing.T) {
	prepararDBTemporalCmd(t)
	prevRows := supervisorPanelRowsBuilder
	prevDetail := supervisorAgentDetailBuilder
	defer func() {
		supervisorPanelRowsBuilder = prevRows
		supervisorAgentDetailBuilder = prevDetail
	}()
	supervisorPanelRowsBuilder = func() ([]agentesapp.Row, error) {
		return []agentesapp.Row{{
			Agente:          &db.Agente{Nombre: "Codex10"},
			EstadoOperativo: "sin_tarea",
		}}, nil
	}
	supervisorAgentDetailBuilder = nil
	now := time.Now().UTC()
	status := apiStatusResponse{
		Autonomia: autonomiaResumen{
			Recent: []autonomyEventSummary{{
				Kind:        "runtime_restart_requested",
				CreatedAt:   now,
				Agent:       "Codex10",
				TargetAgent: "Codex10",
			}},
		},
	}

	actions := buildSupervisorOperationalActions(status, nil)
	if containsSupervisorAction(actions, "seguir_reinicio_runtime", "agente:Codex10") {
		t.Fatalf("no deberia mantener followup zombie de runtime: %+v", actions)
	}
}

func TestBuildSupervisorOperationalActionsOmiteHandoffStaleSiFilaCanonicaYaCerroElFrente(t *testing.T) {
	prepararDBTemporalCmd(t)
	prevRows := supervisorPanelRowsBuilder
	prevDetail := supervisorAgentDetailBuilder
	defer func() {
		supervisorPanelRowsBuilder = prevRows
		supervisorAgentDetailBuilder = prevDetail
	}()
	supervisorPanelRowsBuilder = func() ([]agentesapp.Row, error) {
		return []agentesapp.Row{{
			Agente:          &db.Agente{Nombre: "Codex10"},
			EstadoOperativo: "sin_tarea",
		}}, nil
	}
	supervisorAgentDetailBuilder = nil
	taskID := int64(40)
	now := time.Now().UTC()
	status := apiStatusResponse{
		Autonomia: autonomiaResumen{
			Recent: []autonomyEventSummary{{
				Kind:        "handoff_completed",
				CreatedAt:   now,
				TaskID:      &taskID,
				TargetAgent: "Codex10",
			}},
		},
		TareasActivas: []tareaLite{{
			ID:     taskID,
			Estado: db.TareaEnProgreso,
			Agente: "Codex10",
		}},
	}

	actions := buildSupervisorOperationalActions(status, nil)
	if containsSupervisorAction(actions, "verificar_handoff_consolidado", "tarea:40") {
		t.Fatalf("no deberia mantener handoff stale si la fila canonica ya esta sin tarea: %+v", actions)
	}
}

func TestBuildSupervisorOperationalActionsConsultaOverviewCanonicoSiElPanelNoTraeAlAgente(t *testing.T) {
	prepararDBTemporalCmd(t)
	prevRows := supervisorPanelRowsBuilder
	prevDetail := supervisorAgentDetailBuilder
	defer func() {
		supervisorPanelRowsBuilder = prevRows
		supervisorAgentDetailBuilder = prevDetail
	}()
	supervisorPanelRowsBuilder = func() ([]agentesapp.Row, error) {
		return []agentesapp.Row{{
			Agente:          &db.Agente{Nombre: "Codex1"},
			EstadoOperativo: "trabajando",
		}}, nil
	}
	supervisorAgentDetailBuilder = func(agent string) (*agentesapp.Detail, error) {
		if !strings.EqualFold(agent, "Codex10") {
			return nil, nil
		}
		return &agentesapp.Detail{Row: agentesapp.Row{
			Agente:          &db.Agente{Nombre: "Codex10"},
			EstadoOperativo: "desconocido",
		}}, nil
	}
	now := time.Now().UTC()
	status := apiStatusResponse{
		Autonomia: autonomiaResumen{
			Recent: []autonomyEventSummary{{
				Kind:        "runtime_restart_requested",
				CreatedAt:   now,
				Agent:       "Codex10",
				TargetAgent: "Codex10",
			}},
		},
	}

	actions := buildSupervisorOperationalActions(status, nil)
	if containsSupervisorAction(actions, "seguir_reinicio_runtime", "agente:Codex10") {
		t.Fatalf("no deberia mantener followup zombie cuando el overview canonico ya cerro el agente: %+v", actions)
	}
}
