package cmd

import (
	"testing"

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
