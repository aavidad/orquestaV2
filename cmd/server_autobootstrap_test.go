package cmd

import (
	"path/filepath"
	"strings"
	"testing"

	"orquesta/db"
)

func TestBootstrapServerAutonomyConfiguraProyectoYArranque(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)
	t.Setenv("PWD", tmp)

	if err := db.ConfigSet("server_autobootstrap_enabled", "true"); err != nil {
		t.Fatalf("config autobootstrap enabled: %v", err)
	}
	if err := db.ConfigSet("server_autobootstrap_project_slug", "orquestador"); err != nil {
		t.Fatalf("config slug: %v", err)
	}
	if err := db.ConfigSet("server_autobootstrap_project_name", "Orquestador"); err != nil {
		t.Fatalf("config name: %v", err)
	}
	if err := db.ConfigSet("server_autobootstrap_project_path", filepath.Join(tmp, "repo")); err != nil {
		t.Fatalf("config path: %v", err)
	}
	if err := db.ConfigSet("server_autobootstrap_supervisor_agent", "Codex1"); err != nil {
		t.Fatalf("config supervisor: %v", err)
	}
	if err := db.ConfigSet("server_autobootstrap_worker_agents", "Codex2,Codex3,Codex4,Codex5"); err != nil {
		t.Fatalf("config workers: %v", err)
	}

	if err := bootstrapServerAutonomy(); err != nil {
		t.Fatalf("bootstrap server autonomy: %v", err)
	}

	proyecto, err := db.GetProyecto("orquestador")
	if err != nil {
		t.Fatalf("get proyecto: %v", err)
	}
	if proyecto == nil || proyecto.ID == 0 {
		t.Fatal("proyecto no creado")
	}

	policy, err := db.GetProyectoAutonomia(proyecto.ID)
	if err != nil {
		t.Fatalf("get proyecto autonomia: %v", err)
	}
	if policy == nil || !policy.Enabled || !policy.ReserveSupervisor || !policy.AutoCreateTasks {
		t.Fatalf("policy inesperada: %+v", policy)
	}
	if policy.SupervisorAgente != "Codex1" || policy.MaxWorkers != 4 {
		t.Fatalf("policy supervisor/workers inesperada: %+v", policy)
	}

	for _, nombre := range []string{"Codex1", "Codex2", "Codex3", "Codex4", "Codex5"} {
		asignacion, err := db.GetAsignacionActivaAgente(nombre)
		if err != nil {
			t.Fatalf("get asignacion activa %s: %v", nombre, err)
		}
		if asignacion.ProyectoID != proyecto.ID {
			t.Fatalf("asignacion activa inesperada para %s: %+v", nombre, asignacion)
		}
	}

	agente := "Codex1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyecto.ID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar orders supervisor: %v", err)
	}
	var hasStart, hasNudge bool
	for _, order := range orders {
		if order == nil {
			continue
		}
		if order.Tipo == "start" {
			hasStart = true
		}
		if order.Tipo == "nudge" && strings.Contains(order.PayloadJSON, `"accion":"supervisar_proyecto"`) {
			hasNudge = true
		}
	}
	if !hasStart || !hasNudge {
		t.Fatalf("orders supervisor insuficientes: %+v", orders)
	}

	tareas, err := db.ListarTareas(db.FiltroTareas{ProyectoID: &proyecto.ID})
	if err != nil {
		t.Fatalf("listar tareas: %v", err)
	}
	if len(tareas) == 0 {
		t.Fatal("el bootstrap deberia sembrar backlog real")
	}
}
