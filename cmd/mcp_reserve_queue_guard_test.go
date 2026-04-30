package cmd

import (
	"testing"
	"time"

	"orquesta/agentesapp"
	"orquesta/db"
)

func TestBuildSupervisorOperationalActionsFastNoReservaOtraTareaSiYaHayReservada(t *testing.T) {
	status := apiStatusResponse{
		AgentesActivos: []*db.Agente{
			{Nombre: "Codex1", Rol: "programador", Activo: true, EstadoCuota: "activo"},
			{Nombre: "Codex2", Rol: "programador", Activo: true, EstadoCuota: "activo"},
		},
		AgentesTrabajando: []*db.Agente{
			{Nombre: "Codex1", Rol: "programador", Activo: true, EstadoCuota: "activo"},
			{Nombre: "Codex2", Rol: "programador", Activo: true, EstadoCuota: "activo"},
		},
		TareasPorEstado: map[string]int{
			string(db.TareaLibre):    8,
			string(db.TareaAsignada): 1,
		},
		TareasReservadas: []tareaLite{
			{ID: 4, Estado: db.TareaAsignada, Agente: "Codex2"},
		},
	}

	actions := buildSupervisorOperationalActionsFast(status, nil)
	if containsSupervisorAction(actions, "reservar_tarea_libre", "backlog:libre") {
		t.Fatalf("no deberia sugerir otra reserva si ya existe una viva: %+v", actions)
	}
}

func TestBuildSupervisorOperationalActionsFastNoReservaSinWorkerVisible(t *testing.T) {
	prevConfigGet := statusConfigGet
	prevAutonomy := statusListAutonomyFetcher
	t.Cleanup(func() {
		statusConfigGet = prevConfigGet
		statusListAutonomyFetcher = prevAutonomy
	})

	statusConfigGet = func(string) (string, error) { return "Codex2", nil }
	statusListAutonomyFetcher = func() ([]*db.ProyectoAutonomia, error) { return nil, nil }

	status := apiStatusResponse{
		AgentesActivos: []*db.Agente{
			{Nombre: "Codex2", Rol: "programador", Activo: true, EstadoCuota: "activo"},
		},
		TareasPorEstado: map[string]int{
			string(db.TareaLibre): 8,
		},
	}

	actions := buildSupervisorOperationalActionsFast(status, nil)
	if containsSupervisorAction(actions, "reservar_tarea_libre", "backlog:libre") {
		t.Fatalf("no deberia sugerir reserva autoaplicable sin worker visible: %+v", actions)
	}
}

func TestBuildSupervisorOperationalActionsFastNoReplanificaSinWorkerVisible(t *testing.T) {
	prevConfigGet := statusConfigGet
	prevAutonomy := statusListAutonomyFetcher
	t.Cleanup(func() {
		statusConfigGet = prevConfigGet
		statusListAutonomyFetcher = prevAutonomy
	})

	statusConfigGet = func(string) (string, error) { return "Codex2", nil }
	statusListAutonomyFetcher = func() ([]*db.ProyectoAutonomia, error) { return nil, nil }

	status := apiStatusResponse{
		AgentesActivos: []*db.Agente{
			{Nombre: "Codex2", Rol: "programador", Activo: true, EstadoCuota: "activo"},
		},
		AgentesQuotaBlocked: []*db.Agente{
			{Nombre: "Codex3", Rol: "programador", EstadoCuota: "enfriamiento"},
		},
		TareasActivas: []tareaLite{
			{ID: 40, Estado: db.TareaEnProgreso, Agente: "Codex3"},
		},
	}

	actions := buildSupervisorOperationalActionsFast(status, nil)
	if containsSupervisorAction(actions, "replanificar_por_cuota", "tarea:40") {
		t.Fatalf("no deberia sugerir replanificacion autoaplicable sin worker visible: %+v", actions)
	}
}

func TestBuildSupervisorOperationalActionsFastNoReservaSobreAgenteConRecoveryManualPendiente(t *testing.T) {
	prepararDBTemporalCmd(t)
	prevRows := supervisorPanelRowsBuilder
	defer func() { supervisorPanelRowsBuilder = prevRows }()
	supervisorPanelRowsBuilder = func() ([]agentesapp.Row, error) {
		return []agentesapp.Row{{
			Agente:          &db.Agente{Nombre: "Codex10"},
			EstadoOperativo: "trabajando",
		}}, nil
	}
	taskID := int64(40)
	status := apiStatusResponse{
		AgentesActivos: []*db.Agente{
			{Nombre: "Codex10", Rol: "programador", Activo: true, EstadoCuota: "activo"},
		},
		AgentesTrabajando: []*db.Agente{
			{Nombre: "Codex10", Rol: "programador", Activo: true, EstadoCuota: "activo"},
		},
		TareasActivas: []tareaLite{
			{ID: taskID, Estado: db.TareaEnProgreso, Agente: "Codex10", Prioridad: db.PrioridadAlta},
		},
		TareasPorEstado: map[string]int{
			string(db.TareaLibre): 4,
		},
		Autonomia: autonomiaResumen{
			Recent: []autonomyEventSummary{{
				Kind:        "runtime_restart_requested",
				CreatedAt:   time.Now().UTC(),
				TaskID:      &taskID,
				TargetAgent: "Codex10",
			}},
		},
	}

	actions := buildSupervisorOperationalActionsFast(status, nil)
	if !containsSupervisorAction(actions, "seguir_reinicio_runtime", "tarea:40") {
		t.Fatalf("deberia mantener followup manual de reinicio: %+v", actions)
	}
	if containsSupervisorAction(actions, "reservar_tarea_libre", "backlog:libre") {
		t.Fatalf("no deberia abrir backlog nuevo sobre un agente con recovery manual pendiente: %+v", actions)
	}
}
