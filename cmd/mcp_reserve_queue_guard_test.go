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
