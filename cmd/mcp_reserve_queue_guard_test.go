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

func TestBuildSupervisorOperationalActionsFastNoReservaSobreAgenteConFrenteYaVisible(t *testing.T) {
	status := apiStatusResponse{
		AgentesActivos: []*db.Agente{
			{Nombre: "Codex10", Rol: "programador", Activo: true, EstadoCuota: "activo"},
		},
		AgentesTrabajando: []*db.Agente{
			{Nombre: "Codex10", Rol: "programador", Activo: true, EstadoCuota: "activo"},
		},
		TareasActivas: []tareaLite{
			{ID: 40, Estado: db.TareaEnProgreso, Agente: "Codex10", Prioridad: db.PrioridadAlta},
		},
		TareasPorEstado: map[string]int{
			string(db.TareaLibre): 4,
		},
	}

	actions := buildSupervisorOperationalActionsFast(status, nil)
	if containsSupervisorAction(actions, "reservar_tarea_libre", "backlog:libre") {
		t.Fatalf("no deberia reservar otra tarea sobre un agente que ya tiene un frente visible: %+v", actions)
	}
}

func TestBuildSupervisorOperationalActionsAbreFrenteConAgenteIdleDelPanelCanonico(t *testing.T) {
	prevRows := supervisorPanelRowsBuilder
	defer func() { supervisorPanelRowsBuilder = prevRows }()
	supervisorPanelRowsBuilder = func() ([]agentesapp.Row, error) {
		return []agentesapp.Row{
			{
				Agente:           &db.Agente{Nombre: "Codex10", Rol: "programador", Activo: true, Habilitado: true},
				EstadoOperativo:  "trabajando",
				WorkerAlive:      true,
				WorkerState:      "ready",
				OpenTasks:        1,
				CurrentTask:      &agentesapp.TaskFocus{TaskID: 40, Title: "runtime mailbox", State: db.TareaEnProgreso},
			},
			{
				Agente:           &db.Agente{Nombre: "Codex11", Rol: "programador", Activo: false, Habilitado: true},
				EstadoOperativo:  "sin_tarea",
			},
		}, nil
	}

	status := apiStatusResponse{
		AgentesActivos: []*db.Agente{
			{Nombre: "Codex10", Rol: "programador", Activo: true, EstadoCuota: "activo"},
		},
		AgentesTrabajando: []*db.Agente{
			{Nombre: "Codex10", Rol: "programador", Activo: true, EstadoCuota: "activo"},
		},
		TareasActivas: []tareaLite{
			{ID: 40, Estado: db.TareaEnProgreso, Agente: "Codex10", Prioridad: db.PrioridadAlta},
		},
		TareasPorEstado: map[string]int{
			string(db.TareaLibre): 4,
		},
	}

	actions := buildSupervisorOperationalActions(status, nil)
	item := findSupervisorAction(actions, "asignar_tarea_libre", "backlog:libre")
	if item == nil {
		t.Fatalf("deberia abrir frente con agente idle del panel canonico: %+v", actions)
	}
	if item.Assignee != "codex11" {
		t.Fatalf("assignee inesperado: %+v", item)
	}
}

func TestBuildSupervisorOperationalActionsPrefiereAbrirFrenteSobreReservaSiHayIdleLanzable(t *testing.T) {
	prevRows := supervisorPanelRowsBuilder
	defer func() { supervisorPanelRowsBuilder = prevRows }()
	supervisorPanelRowsBuilder = func() ([]agentesapp.Row, error) {
		return []agentesapp.Row{
			{
				Agente:          &db.Agente{Nombre: "Codex10", Rol: "programador", Activo: true, Habilitado: true},
				EstadoOperativo: "trabajando",
				WorkerAlive:     true,
				WorkerState:     "ready",
				OpenTasks:       1,
				CurrentTask:     &agentesapp.TaskFocus{TaskID: 40, Title: "runtime mailbox", State: db.TareaEnProgreso},
			},
			{
				Agente:          &db.Agente{Nombre: "Codex11", Rol: "programador", Activo: false, Habilitado: true},
				EstadoOperativo: "sin_tarea",
			},
		}, nil
	}

	status := apiStatusResponse{
		AgentesActivos: []*db.Agente{
			{Nombre: "Codex10", Rol: "programador", Activo: true, EstadoCuota: "activo"},
		},
		AgentesTrabajando: []*db.Agente{
			{Nombre: "Codex10", Rol: "programador", Activo: true, EstadoCuota: "activo"},
		},
		TareasActivas: []tareaLite{
			{ID: 40, Estado: db.TareaEnProgreso, Agente: "Codex10", Prioridad: db.PrioridadAlta},
		},
		TareasPorEstado: map[string]int{
			string(db.TareaLibre): 2,
		},
	}

	actions := buildSupervisorOperationalActions(status, nil)
	if reserve := findSupervisorAction(actions, "reservar_tarea_libre", "backlog:libre"); reserve != nil {
		t.Fatalf("no deberia reservar si ya va a abrir frente con agente idle lanzable: %+v", actions)
	}
	if assign := findSupervisorAction(actions, "asignar_tarea_libre", "backlog:libre"); assign == nil {
		t.Fatalf("deberia abrir frente nuevo con agente idle lanzable: %+v", actions)
	}
}

func TestSupervisorIdleLaunchCandidatesExcluyeSupervisoresConfigurados(t *testing.T) {
	prevRows := supervisorPanelRowsBuilder
	defer func() { supervisorPanelRowsBuilder = prevRows }()
	supervisorPanelRowsBuilder = func() ([]agentesapp.Row, error) {
		return []agentesapp.Row{
			{
				Agente:          &db.Agente{Nombre: "Codex1", Rol: "supervisor", Habilitado: true},
				EstadoOperativo: "sin_tarea",
			},
			{
				Agente:          &db.Agente{Nombre: "Codex11", Rol: "programador", Habilitado: true},
				EstadoOperativo: "sin_tarea",
			},
		}, nil
	}

	got := supervisorIdleLaunchCandidates()
	if len(got) != 1 || got[0] != "codex11" {
		t.Fatalf("candidatos launch idle inesperados: %+v", got)
	}
}
