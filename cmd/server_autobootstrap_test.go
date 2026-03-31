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

func TestBootstrapServerAutonomyNoDuplicaBootstrapEnAgenteYaOperativo(t *testing.T) {
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
	if err := db.ConfigSet("server_autobootstrap_worker_agents", "Codex2,Codex3"); err != nil {
		t.Fatalf("config workers: %v", err)
	}

	if err := bootstrapServerAutonomy(); err != nil {
		t.Fatalf("bootstrap inicial: %v", err)
	}

	proyecto, err := db.GetProyecto("orquestador")
	if err != nil {
		t.Fatalf("get proyecto: %v", err)
	}
	if proyecto == nil {
		t.Fatal("proyecto no creado")
	}

	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex3",
		ProyectoID:  &proyecto.ID,
		CWD:         filepath.Join(tmp, "repo"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion worker: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil {
		t.Fatalf("get handle sesion: %v", err)
	}
	if handle == nil {
		t.Fatal("handle no creado")
	}
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET estado='activo', last_seen_at=CURRENT_TIMESTAMP WHERE id=?`, handle.ID); err != nil {
		t.Fatalf("activar handle: %v", err)
	}

	if err := bootstrapServerAutonomy(); err != nil {
		t.Fatalf("bootstrap repetido: %v", err)
	}

	agente := "Codex3"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyecto.ID})
	if err != nil {
		t.Fatalf("listar orders worker: %v", err)
	}
	var nudgesBootstrap int
	for _, order := range orders {
		if order == nil || order.Tipo != "nudge" {
			continue
		}
		if strings.Contains(order.PayloadJSON, `"bootstrap_kind":"server_autobootstrap"`) &&
			strings.Contains(order.PayloadJSON, `"accion":"esperar_o_pedir_tarea"`) {
			nudgesBootstrap++
		}
	}
	if nudgesBootstrap != 1 {
		t.Fatalf("bootstrap duplicado para worker operativo: %d", nudgesBootstrap)
	}
}

func TestAgenteYaBootstrappeadoServidorDetectaMailboxDurablePendiente(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)
	t.Setenv("PWD", tmp)

	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "repo"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if err := db.RegistrarAgente("Codex3", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	if err := db.ActivarAsignacion("Codex3", proyectoID, "test"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	msgID, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex3",
		ProyectoID:  &proyectoID,
		Kind:        "autonomia",
		PayloadJSON: `{"bootstrap":true,"bootstrap_kind":"server_autobootstrap","accion":"esperar_o_pedir_tarea"}`,
	})
	if err != nil {
		t.Fatalf("encolar mailbox: %v", err)
	}
	if msgID == 0 {
		t.Fatal("mailbox no creada")
	}

	ok, err := agenteYaBootstrappeadoServidor("Codex3", proyectoID)
	if err != nil {
		t.Fatalf("agenteYaBootstrappeadoServidor: %v", err)
	}
	if !ok {
		t.Fatal("deberia detectar bootstrap durable pendiente")
	}
}
