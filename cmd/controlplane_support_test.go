package cmd

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"orquesta/coordinacion"
	"orquesta/db"
	"orquesta/reviewapp"
	"orquesta/runtimeagente"
)

func TestProcesarSupervisionAutonomaBatchEncolaSupervisionYRegistraCiclo(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("CodexSupervisor", "admin"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	if err := db.RegistrarAgente("CodexA", "admin"); err != nil {
		t.Fatalf("registrar agente alternativo: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := db.UpsertProyectoAutonomia(&db.ProyectoAutonomia{
		ProyectoID:           proyectoID,
		Enabled:              true,
		ObjetivoGeneral:      "Terminar la app",
		DefinitionOfDoneJSON: `{"done":true}`,
		SupervisorAgente:     "CodexSupervisor",
		ReviewRequired:       true,
		AutoCloseProject:     true,
		EstadoAutonomia:      db.AutonomiaProyectoActiva,
	}); err != nil {
		t.Fatalf("upsert proyecto autonomia: %v", err)
	}
	if err := db.ActivarAsignacion("CodexSupervisor", proyectoID, "supervision"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	if err := db.ActivarAsignacion("CodexA", proyectoID, "supervision alterna"); err != nil {
		t.Fatalf("activar asignacion alterna: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "CodexSupervisor",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "CodexA",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion alterna: %v", err)
	}

	n, err := procesarSupervisionAutonomaBatch()
	if err != nil {
		t.Fatalf("procesar supervision: %v", err)
	}
	if n != 1 {
		t.Fatalf("esperaba 1 supervision, got=%d", n)
	}

	agente := "CodexSupervisor"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	if len(orders) != 1 || orders[0].Tipo != "nudge" || !strings.Contains(orders[0].PayloadJSON, `"accion":"supervisar_proyecto"`) {
		t.Fatalf("supervision no encolada: %+v", orders)
	}
	kind := "supervision"
	cycles, err := db.ListarAutonomiaCiclos(db.FiltroAutonomiaCiclos{ProyectoID: &proyectoID, Kind: &kind, Limit: 10})
	if err != nil {
		t.Fatalf("listar autonomia ciclos: %v", err)
	}
	if len(cycles) != 1 || cycles[0].Agente != "CodexSupervisor" {
		t.Fatalf("ciclo supervision inesperado: %+v", cycles)
	}
	policy, err := db.GetProyectoAutonomia(proyectoID)
	if err != nil {
		t.Fatalf("get proyecto autonomia: %v", err)
	}
	if policy.LastSupervisionAt == nil {
		t.Fatal("last_supervision_at debería informarse")
	}
}

func TestProcesarSupervisionAutonomaBatchArrancaSupervisorPreferidoSinSesion(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("CodexSupervisor", "admin"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := db.UpsertProyectoAutonomia(&db.ProyectoAutonomia{
		ProyectoID:           proyectoID,
		Enabled:              true,
		ObjetivoGeneral:      "Terminar la app",
		DefinitionOfDoneJSON: `{"done":true}`,
		SupervisorAgente:     "CodexSupervisor",
		ReviewRequired:       true,
		AutoCloseProject:     true,
		EstadoAutonomia:      db.AutonomiaProyectoActiva,
	}); err != nil {
		t.Fatalf("upsert proyecto autonomia: %v", err)
	}

	n, err := procesarSupervisionAutonomaBatch()
	if err != nil {
		t.Fatalf("procesar supervision: %v", err)
	}
	if n != 1 {
		t.Fatalf("debería contar supervisión en el mismo ciclo de arranque, got=%d", n)
	}

	asignacion, err := db.GetAsignacionActivaAgente("CodexSupervisor")
	if err != nil {
		agente := "CodexSupervisor"
		estado := "pendiente"
		orders, _ := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
		t.Fatalf("get asignacion activa: %v; orders=%+v", err, orders)
	}
	if asignacion.ProyectoID != proyectoID {
		t.Fatalf("asignacion activa inesperada: %+v", asignacion)
	}

	agente := "CodexSupervisor"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
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
		t.Fatalf("debería encolar start y nudge de supervisión: %+v", orders)
	}
}

func TestProcesarSupervisionAutonomaBatchNoRepiteSupervisionPeriodicaConSupervisorOperativo(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("CodexSupervisor", "admin"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := db.UpsertProyectoAutonomia(&db.ProyectoAutonomia{
		ProyectoID:           proyectoID,
		Enabled:              true,
		ObjetivoGeneral:      "Terminar la app",
		DefinitionOfDoneJSON: `{"done":true}`,
		SupervisorAgente:     "CodexSupervisor",
		ReviewRequired:       true,
		AutoCloseProject:     true,
		EstadoAutonomia:      db.AutonomiaProyectoActiva,
	}); err != nil {
		t.Fatalf("upsert proyecto autonomia: %v", err)
	}
	if err := db.ActivarAsignacion("CodexSupervisor", proyectoID, "supervision"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "CodexSupervisor",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}

	n, err := procesarSupervisionAutonomaBatch()
	if err != nil {
		t.Fatalf("primer ciclo supervision: %v", err)
	}
	if n != 1 {
		t.Fatalf("esperaba bootstrap inicial de supervision, got=%d", n)
	}

	agente := "CodexSupervisor"
	completada := "completada"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &completada})
	if err != nil {
		t.Fatalf("listar orders completadas: %v", err)
	}
	if len(orders) != 0 {
		t.Fatalf("no esperaba completadas antes de materializar la prueba: %+v", orders)
	}

	pendiente := "pendiente"
	orders, err = db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &pendiente})
	if err != nil {
		t.Fatalf("listar orders pendientes: %v", err)
	}
	if len(orders) != 1 {
		t.Fatalf("esperaba una sola order inicial, got=%d", len(orders))
	}
	if _, err := db.DB.Exec(`UPDATE runtime_orders
		SET estado='completada',
		    started_at=datetime('now','-10 minutes'),
		    finished_at=datetime('now','-10 minutes'),
		    updated_at=datetime('now','-10 minutes')
		WHERE id=?`, orders[0].ID); err != nil {
		t.Fatalf("cerrar order inicial: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE proyectos_autonomia
		SET last_supervision_at=datetime('now','-10 minutes')
		WHERE proyecto_id=?`, proyectoID); err != nil {
		t.Fatalf("forzar supervision vencida: %v", err)
	}

	n, err = procesarSupervisionAutonomaBatch()
	if err != nil {
		t.Fatalf("segundo ciclo supervision: %v", err)
	}
	if n != 0 {
		t.Fatalf("no deberia repetir supervision periodica con supervisor operativo, got=%d", n)
	}

	orders, err = db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID})
	if err != nil {
		t.Fatalf("listar todas las orders: %v", err)
	}
	if len(orders) != 1 {
		t.Fatalf("no deberia crear una segunda order periodica: %+v", orders)
	}
}

func TestProcesarSupervisionAutonomaBatchNoRepiteSupervisionPeriodicaSiYaEmitioNudgeReciente(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("CodexSupervisor", "admin"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := db.UpsertProyectoAutonomia(&db.ProyectoAutonomia{
		ProyectoID:           proyectoID,
		Enabled:              true,
		ObjetivoGeneral:      "Terminar la app",
		DefinitionOfDoneJSON: `{"done":true}`,
		SupervisorAgente:     "CodexSupervisor",
		ReviewRequired:       true,
		AutoCloseProject:     true,
		EstadoAutonomia:      db.AutonomiaProyectoActiva,
	}); err != nil {
		t.Fatalf("upsert proyecto autonomia: %v", err)
	}
	if err := db.ActivarAsignacion("CodexSupervisor", proyectoID, "supervision"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "CodexSupervisor",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}

	if _, err := db.DB.Exec(`UPDATE proyectos_autonomia
		SET last_supervision_at=datetime('now','-10 minutes')
		WHERE proyecto_id=?`, proyectoID); err != nil {
		t.Fatalf("forzar supervision vencida: %v", err)
	}

	proyecto, err := db.GetProyecto("orquestador")
	if err != nil {
		t.Fatalf("get proyecto: %v", err)
	}
	if proyecto == nil {
		t.Fatal("proyecto nil")
	}
	encolada, err := encolarNudgeAutonomiaDetallado("CodexSupervisor", proyecto, "supervisar_proyecto", "reciente", "instruction reciente", map[string]any{
		"supervision_cycle_kind": "supervision",
	})
	if err != nil {
		t.Fatalf("encolar nudge reciente: %v", err)
	}
	if !encolada {
		t.Fatal("faltaba nudge reciente")
	}
	agente := "CodexSupervisor"
	estadoPendiente := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estadoPendiente})
	if err != nil {
		t.Fatalf("listar runtime orders pendientes: %v", err)
	}
	if len(orders) != 1 {
		t.Fatalf("faltaba order pendiente inicial: %+v", orders)
	}
	orderID := orders[0].ID
	if _, err := db.DB.Exec(`UPDATE runtime_orders
		SET estado='completada',
		    started_at=datetime('now','-2 minutes'),
		    finished_at=datetime('now','-2 minutes'),
		    updated_at=datetime('now','-2 minutes')
		WHERE id=?`, orderID); err != nil {
		t.Fatalf("cerrar nudge reciente: %v", err)
	}

	n, err := procesarSupervisionAutonomaBatch()
	if err != nil {
		t.Fatalf("segundo ciclo supervision: %v", err)
	}
	if n != 0 {
		t.Fatalf("no deberia repetir supervision periodica si ya emitio un nudge reciente, got=%d", n)
	}

	orders, err = db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID})
	if err != nil {
		t.Fatalf("listar runtime orders: %v", err)
	}
	var supervisionNudges int
	for _, order := range orders {
		if order != nil && order.Tipo == "nudge" && strings.Contains(order.PayloadJSON, `"accion":"supervisar_proyecto"`) {
			supervisionNudges++
		}
	}
	if supervisionNudges != 1 {
		t.Fatalf("no deberia crear un segundo nudge de supervision reciente: %+v", orders)
	}
}

func TestProcesarSupervisionAutonomaBatchNoRepiteSupervisionSiSupervisorYaTieneTrabajoActivo(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("CodexSupervisor", "admin"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := db.UpsertProyectoAutonomia(&db.ProyectoAutonomia{
		ProyectoID:           proyectoID,
		Enabled:              true,
		ObjetivoGeneral:      "Terminar la app",
		DefinitionOfDoneJSON: `{"done":true}`,
		SupervisorAgente:     "CodexSupervisor",
		ReviewRequired:       true,
		AutoCreateTasks:      true,
		AutoCloseProject:     true,
		EstadoAutonomia:      db.AutonomiaProyectoActiva,
	}); err != nil {
		t.Fatalf("upsert proyecto autonomia: %v", err)
	}
	if err := db.ActivarAsignacion("CodexSupervisor", proyectoID, "supervision"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "CodexSupervisor",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	taskID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Supervision ya en curso",
		Descripcion: "Auditoria real del proyecto",
		ProyectoID:  &proyectoID,
		Modulo:      "autonomia",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "orquesta",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(taskID, "CodexSupervisor"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(taskID, "CodexSupervisor"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE proyectos_autonomia
		SET last_supervision_at=datetime('now','-10 minutes')
		WHERE proyecto_id=?`, proyectoID); err != nil {
		t.Fatalf("forzar supervision vencida: %v", err)
	}

	n, err := procesarSupervisionAutonomaBatch()
	if err != nil {
		t.Fatalf("procesar supervision: %v", err)
	}
	if n != 0 {
		t.Fatalf("no deberia repetir supervision si el supervisor ya tiene trabajo activo, got=%d", n)
	}

	agente := "CodexSupervisor"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID})
	if err != nil {
		t.Fatalf("listar runtime orders: %v", err)
	}
	for _, order := range orders {
		if order != nil && order.Tipo == "nudge" && strings.Contains(order.PayloadJSON, `"accion":"supervisar_proyecto"`) {
			t.Fatalf("no deberia crear nudge de supervision sobre supervisor con trabajo activo: %+v", orders)
		}
	}
}

func TestProcesarSupervisionAutonomaBatchGeneraBacklogInicialSiAutoCreateTasks(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("CodexSupervisor", "admin"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := db.UpsertProyectoAutonomia(&db.ProyectoAutonomia{
		ProyectoID:           proyectoID,
		Enabled:              true,
		ObjetivoGeneral:      "Terminar la app",
		DefinitionOfDoneJSON: `{"done":true}`,
		SupervisorAgente:     "CodexSupervisor",
		ReviewRequired:       true,
		AutoCreateTasks:      true,
		AutoCloseProject:     true,
		EstadoAutonomia:      db.AutonomiaProyectoActiva,
	}); err != nil {
		t.Fatalf("upsert proyecto autonomia: %v", err)
	}
	if err := db.ActivarAsignacion("CodexSupervisor", proyectoID, "supervision"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "CodexSupervisor",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}

	n, err := procesarSupervisionAutonomaBatch()
	if err != nil {
		t.Fatalf("procesar supervision: %v", err)
	}
	if n != 1 {
		t.Fatalf("esperaba 1 supervision, got=%d", n)
	}

	tareas, err := db.ListarTareas(db.FiltroTareas{ProyectoID: &proyectoID})
	if err != nil {
		t.Fatalf("listar tareas: %v", err)
	}
	if len(tareas) < 5 {
		t.Fatalf("debería generar un backlog inicial completo, tareas=%+v", tareas)
	}
	var briefing, investigacion, arquitectura bool
	var briefingActiva bool
	var backlog int
	for _, tarea := range tareas {
		if tarea == nil {
			continue
		}
		switch tarea.Titulo {
		case "Definir briefing funcional de Orquestador":
			briefing = true
			if tarea.Estado == db.TareaEnProgreso && tarea.Agente != nil && *tarea.Agente == "CodexSupervisor" {
				briefingActiva = true
			}
		case "Revisar catálogo y referencias de Orquestador",
			"Revisar catalogo y referencias de Orquestador":
			investigacion = true
		case "Cerrar arquitectura hexagonal y modular de Orquestador",
			"Cerrar arquitectura base de Orquestador":
			arquitectura = true
		case "Autonomía: revisar backlog y abrir siguiente frente útil":
			t.Fatalf("no debería crear tarea semilla cuando el proyecto aún no tiene backlog: %+v", tarea)
		}
		if tarea.Estado == db.TareaBacklog {
			backlog++
		}
	}
	if !briefing || !investigacion || !arquitectura {
		t.Fatalf("faltan tareas base del plan inicial, tareas=%+v", tareas)
	}
	if !briefingActiva {
		t.Fatalf("el briefing debería arrancarse en el supervisor para poner en marcha el plan, tareas=%+v", tareas)
	}
	if backlog == 0 {
		t.Fatalf("el backlog inicial debería dejar trabajo dependiente en backlog, tareas=%+v", tareas)
	}

	kind := "supervision"
	cycles, err := db.ListarAutonomiaCiclos(db.FiltroAutonomiaCiclos{ProyectoID: &proyectoID, Kind: &kind, Limit: 10})
	if err != nil {
		t.Fatalf("listar ciclos: %v", err)
	}
	if len(cycles) != 1 || !strings.Contains(cycles[0].DecisionJSON, `"auto_created_plan":true`) {
		t.Fatalf("ciclo supervision sin traza de auto_create_tasks: %+v", cycles)
	}
}

func TestProcesarSupervisionAutonomaBatchReabreFrenteSiSoloHayHistoricoCerrado(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("CodexSupervisor", "admin"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := db.UpsertProyectoAutonomia(&db.ProyectoAutonomia{
		ProyectoID:           proyectoID,
		Enabled:              true,
		ObjetivoGeneral:      "Terminar la app",
		DefinitionOfDoneJSON: `{"done":true}`,
		SupervisorAgente:     "CodexSupervisor",
		ReviewRequired:       true,
		AutoCreateTasks:      true,
		AutoCloseProject:     false,
		EstadoAutonomia:      db.AutonomiaProyectoActiva,
	}); err != nil {
		t.Fatalf("upsert proyecto autonomia: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Frente anterior",
		Descripcion: "Ya completado",
		ProyectoID:  &proyectoID,
		Modulo:      "core",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea historica: %v", err)
	}
	if err := db.TomarTarea(tareaID, "CodexSupervisor"); err != nil {
		t.Fatalf("tomar tarea historica: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "CodexSupervisor"); err != nil {
		t.Fatalf("iniciar tarea historica: %v", err)
	}
	if err := db.CompletarTarea(tareaID, "CodexSupervisor", "hecho"); err != nil {
		t.Fatalf("completar tarea historica: %v", err)
	}
	if err := db.ActivarAsignacion("CodexSupervisor", proyectoID, "supervision"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "CodexSupervisor",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}

	n, err := procesarSupervisionAutonomaBatch()
	if err != nil {
		t.Fatalf("procesar supervision: %v", err)
	}
	if n != 1 {
		t.Fatalf("esperaba 1 supervision con reapertura de frente, got=%d", n)
	}

	tareas, err := db.ListarTareas(db.FiltroTareas{ProyectoID: &proyectoID})
	if err != nil {
		t.Fatalf("listar tareas: %v", err)
	}
	if len(tareas) != 2 {
		t.Fatalf("debería conservar histórico y abrir un frente nuevo, tareas=%+v", tareas)
	}
	var abiertas int
	for _, tarea := range tareas {
		if tarea == nil {
			continue
		}
		if tarea.Estado == db.TareaEnProgreso && tarea.Titulo == "Autonomía: revisar backlog y abrir siguiente frente útil" {
			abiertas++
		}
	}
	if abiertas != 1 {
		t.Fatalf("debería abrir exactamente una nueva tarea semilla en progreso, tareas=%+v", tareas)
	}
}

func TestProcesarSupervisionAutonomaBatchNoCreaTareaSemillaSiAutoCreateTasksDesactivado(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("CodexSupervisor", "admin"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := db.UpsertProyectoAutonomia(&db.ProyectoAutonomia{
		ProyectoID:           proyectoID,
		Enabled:              true,
		ObjetivoGeneral:      "Terminar la app",
		DefinitionOfDoneJSON: `{"done":true}`,
		SupervisorAgente:     "CodexSupervisor",
		ReviewRequired:       true,
		AutoCreateTasks:      false,
		AutoCloseProject:     true,
		EstadoAutonomia:      db.AutonomiaProyectoActiva,
	}); err != nil {
		t.Fatalf("upsert proyecto autonomia: %v", err)
	}
	if err := db.ActivarAsignacion("CodexSupervisor", proyectoID, "supervision"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "CodexSupervisor",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}

	if _, err := procesarSupervisionAutonomaBatch(); err != nil {
		t.Fatalf("procesar supervision: %v", err)
	}

	tareas, err := db.ListarTareas(db.FiltroTareas{ProyectoID: &proyectoID})
	if err != nil {
		t.Fatalf("listar tareas: %v", err)
	}
	if len(tareas) != 0 {
		t.Fatalf("no debería crear tarea semilla con auto_create_tasks desactivado: %+v", tareas)
	}
}

func TestProcesarReviewGatesBatchCreaGateYEncolaRevision(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("CodexReviewer", "admin"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	if err := db.RegistrarAgente("CodexA", "admin"); err != nil {
		t.Fatalf("registrar agente alternativo: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := db.UpsertProyectoAutonomia(&db.ProyectoAutonomia{
		ProyectoID:           proyectoID,
		Enabled:              true,
		ObjetivoGeneral:      "Terminar la app",
		DefinitionOfDoneJSON: `{"done":true}`,
		ReviewerAgente:       "CodexReviewer",
		ReviewRequired:       true,
		AutoCloseProject:     true,
		EstadoAutonomia:      db.AutonomiaProyectoActiva,
	}); err != nil {
		t.Fatalf("upsert proyecto autonomia: %v", err)
	}
	if err := db.ActivarAsignacion("CodexReviewer", proyectoID, "review"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	if err := db.ActivarAsignacion("CodexA", proyectoID, "review alterna"); err != nil {
		t.Fatalf("activar asignacion alterna: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "CodexReviewer",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "CodexA",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion alterna: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:     "Cerrar autonomia",
		ProyectoID: &proyectoID,
		Prioridad:  db.PrioridadAlta,
		CreadoPor:  "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "CodexReviewer"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "CodexReviewer"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	if err := db.CompletarTarea(tareaID, "CodexReviewer", "abc123"); err != nil {
		t.Fatalf("completar tarea: %v", err)
	}
	worktree, err := db.CoordinationWorktreeSQLRepository{}.Create(&coordinacion.Worktree{
		ProjectID: proyectoID,
		TaskID:    &tareaID,
		Agent:     "CodexReviewer",
		Name:      "orquestador-codexreviewer-t1",
		Path:      filepath.Join(tmp, "wt-review"),
		Branch:    "orq/orquestador/CodexReviewer/t1",
		BaseRef:   "master",
		State:     coordinacion.WorktreeActive,
		Reason:    "review_test",
	})
	if err != nil {
		t.Fatalf("crear worktree activa: %v", err)
	}

	n, err := procesarReviewGatesBatch()
	if err != nil {
		t.Fatalf("procesar review gates: %v", err)
	}
	if n != 2 {
		t.Fatalf("esperaba 2 acciones de review (gate + nudge), got=%d", n)
	}

	gates, err := db.ListarReviewGates(db.FiltroReviewGates{ProyectoID: &proyectoID, Limit: 10})
	if err != nil {
		t.Fatalf("listar review gates: %v", err)
	}
	if len(gates) != 1 {
		t.Fatalf("esperaba 1 review gate, got=%d", len(gates))
	}
	if gates[0].ReviewerAgente != "CodexReviewer" || gates[0].Estado != db.ReviewGateEnRevision {
		t.Fatalf("review gate inesperado: %+v", gates[0])
	}
	if gates[0].TareaID == nil || *gates[0].TareaID != tareaID {
		t.Fatalf("el gate debería quedar ligado a la tarea revisada, gate=%+v", gates[0])
	}
	if gates[0].WorktreeID == nil || *gates[0].WorktreeID != worktree.ID {
		t.Fatalf("el gate debería quedar ligado al worktree activo, gate=%+v", gates[0])
	}
	agente := "CodexReviewer"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	if len(orders) != 1 || orders[0].Tipo != "nudge" || !strings.Contains(orders[0].PayloadJSON, `"accion":"ejecutar_review_gate"`) {
		t.Fatalf("review nudge inesperado: %+v", orders)
	}
	policy, err := db.GetProyectoAutonomia(proyectoID)
	if err != nil {
		t.Fatalf("get proyecto autonomia: %v", err)
	}
	if policy.EstadoAutonomia != db.AutonomiaProyectoEsperandoReview {
		t.Fatalf("estado autonomia inesperado: %s", policy.EstadoAutonomia)
	}
}

func TestProcesarReviewGatesBatchCreaSolicitudMergeTrasGateAprobadoReactivaProyecto(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := db.UpsertProyectoAutonomia(&db.ProyectoAutonomia{
		ProyectoID:           proyectoID,
		Enabled:              true,
		ObjetivoGeneral:      "Terminar la app",
		DefinitionOfDoneJSON: `{"done":true}`,
		ReviewerAgente:       "CodexReviewer",
		ReviewRequired:       true,
		EstadoAutonomia:      db.AutonomiaProyectoActiva,
	}); err != nil {
		t.Fatalf("upsert proyecto autonomia: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:     "Cerrar frente para merge",
		ProyectoID: &proyectoID,
		Prioridad:  db.PrioridadAlta,
		CreadoPor:  "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "CodexReviewer"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "CodexReviewer"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	if err := db.CompletarTarea(tareaID, "CodexReviewer", "abc123"); err != nil {
		t.Fatalf("completar tarea: %v", err)
	}
	worktree, err := db.CoordinationWorktreeSQLRepository{}.Create(&coordinacion.Worktree{
		ProjectID: proyectoID,
		TaskID:    &tareaID,
		Agent:     "CodexReviewer",
		Name:      "orquestador-codexreviewer-merge",
		Path:      filepath.Join(tmp, "wt-merge"),
		Branch:    "orq/orquestador/CodexReviewer/t-merge",
		BaseRef:   "master",
		State:     coordinacion.WorktreeActive,
		Reason:    "merge_test",
	})
	if err != nil {
		t.Fatalf("crear worktree activa: %v", err)
	}
	now := time.Now().UTC()
	if _, err := db.CrearReviewGate(&db.ReviewGate{
		ProyectoID:     &proyectoID,
		TareaID:        &tareaID,
		WorktreeID:     &worktree.ID,
		RequestedBy:    "orquesta",
		ReviewerAgente: "CodexReviewer",
		Estado:         db.ReviewGateAprobado,
		ResolvedAt:     &now,
	}); err != nil {
		t.Fatalf("crear review gate aprobado: %v", err)
	}

	n, err := procesarReviewGatesBatch()
	if err != nil {
		t.Fatalf("procesar review gates: %v", err)
	}
	if n != 1 {
		t.Fatalf("esperaba 1 acción de auto-merge, got=%d", n)
	}
	merges, err := db.ListarGitMerges(&proyectoID, "")
	if err != nil {
		t.Fatalf("listar merges: %v", err)
	}
	if len(merges) != 1 {
		t.Fatalf("debería crear una única solicitud de merge, merges=%+v", merges)
	}
	if merges[0].SourceBranch != "orq/orquestador/CodexReviewer/t-merge" || merges[0].TargetBranch != "master" {
		t.Fatalf("solicitud de merge inesperada: %+v", merges[0])
	}
	if merges[0].Estado != "aprobado" {
		t.Fatalf("la solicitud automática debería quedar aprobada, merge=%+v", merges[0])
	}
	if !strings.Contains(merges[0].Notas, "review gate") {
		t.Fatalf("la solicitud de merge debería dejar trazabilidad del gate, merge=%+v", merges[0])
	}

	n, err = procesarReviewGatesBatch()
	if err != nil {
		t.Fatalf("reprocesar review gates: %v", err)
	}
	if n != 0 {
		t.Fatalf("no debería duplicar la solicitud de merge, got=%d", n)
	}
	merges, err = db.ListarGitMerges(&proyectoID, "")
	if err != nil {
		t.Fatalf("listar merges tras reproceso: %v", err)
	}
	if len(merges) != 1 {
		t.Fatalf("no debería duplicar merges, merges=%+v", merges)
	}
}

func TestProcesarReviewGatesBatchArrancaReviewerPreferidoSinSesion(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("CodexReviewer", "admin"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := db.UpsertProyectoAutonomia(&db.ProyectoAutonomia{
		ProyectoID:           proyectoID,
		Enabled:              true,
		ObjetivoGeneral:      "Terminar la app",
		DefinitionOfDoneJSON: `{"done":true}`,
		ReviewerAgente:       "CodexReviewer",
		ReviewRequired:       true,
		AutoCloseProject:     true,
		EstadoAutonomia:      db.AutonomiaProyectoActiva,
	}); err != nil {
		t.Fatalf("upsert proyecto autonomia: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:     "Cerrar autonomia",
		ProyectoID: &proyectoID,
		Prioridad:  db.PrioridadAlta,
		CreadoPor:  "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "CodexReviewer"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "CodexReviewer"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	if err := db.CompletarTarea(tareaID, "CodexReviewer", "abc123"); err != nil {
		t.Fatalf("completar tarea: %v", err)
	}

	n, err := procesarReviewGatesBatch()
	if err != nil {
		t.Fatalf("procesar review gates: %v", err)
	}
	if n != 0 {
		t.Fatalf("no debería contar review hasta que el reviewer arranque, got=%d", n)
	}

	asignacion, err := db.GetAsignacionActivaAgente("CodexReviewer")
	if err != nil {
		agente := "CodexReviewer"
		estado := "pendiente"
		orders, _ := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
		t.Fatalf("get asignacion activa: %v; orders=%+v", err, orders)
	}
	if asignacion.ProyectoID != proyectoID {
		t.Fatalf("asignacion activa inesperada: %+v", asignacion)
	}

	agente := "CodexReviewer"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	if len(orders) != 1 || orders[0].Tipo != "start" {
		t.Fatalf("debería encolar start para reviewer preferido: %+v", orders)
	}
}

func TestProcesarReviewGatesBatchReabreCorreccionTrasCambiosPedidos(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("CodexSupervisor", "admin"); err != nil {
		t.Fatalf("registrar supervisor: %v", err)
	}
	if err := db.RegistrarAgente("CodexReviewer", "admin"); err != nil {
		t.Fatalf("registrar reviewer: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := db.UpsertProyectoAutonomia(&db.ProyectoAutonomia{
		ProyectoID:           proyectoID,
		Enabled:              true,
		ObjetivoGeneral:      "Terminar la app",
		DefinitionOfDoneJSON: `{"done":true}`,
		SupervisorAgente:     "CodexSupervisor",
		ReviewerAgente:       "CodexReviewer",
		ReviewRequired:       true,
		AutoCloseProject:     true,
		EstadoAutonomia:      db.AutonomiaProyectoEsperandoReview,
	}); err != nil {
		t.Fatalf("upsert proyecto autonomia: %v", err)
	}
	if err := db.ActivarAsignacion("CodexSupervisor", proyectoID, "supervision"); err != nil {
		t.Fatalf("activar asignacion supervisor: %v", err)
	}
	if err := db.ActivarAsignacion("CodexReviewer", proyectoID, "review"); err != nil {
		t.Fatalf("activar asignacion reviewer: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "CodexSupervisor",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion supervisor: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "CodexReviewer",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion reviewer: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:     "Ultimo frente",
		ProyectoID: &proyectoID,
		Prioridad:  db.PrioridadAlta,
		CreadoPor:  "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "CodexReviewer"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "CodexReviewer"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	if err := db.CompletarTarea(tareaID, "CodexReviewer", "abc123"); err != nil {
		t.Fatalf("completar tarea: %v", err)
	}
	gateID, err := db.CrearReviewGate(&db.ReviewGate{
		ProyectoID:     &proyectoID,
		RequestedBy:    "CodexReviewer",
		ReviewerAgente: "CodexReviewer",
		Estado:         db.ReviewGateCambiosPed,
		SeverityMax:    "high",
		FindingsJSON:   `{"findings":["faltan tests","ajustar arquitectura"]}`,
	})
	if err != nil {
		t.Fatalf("crear review gate: %v", err)
	}

	n, err := procesarReviewGatesBatch()
	if err != nil {
		t.Fatalf("procesar review gates: %v", err)
	}
	if n != 1 {
		t.Fatalf("esperaba 1 acción de corrección, got=%d", n)
	}

	agente := "CodexSupervisor"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	if len(orders) != 1 || orders[0].Tipo != "nudge" || !strings.Contains(orders[0].PayloadJSON, `"accion":"resolver_review_feedback"`) {
		todas, _ := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{ProyectoID: &proyectoID, Estado: &estado})
		t.Fatalf("nudge de corrección inesperado: agente=%+v todas=%+v", orders, todas)
	}
	if !strings.Contains(orders[0].PayloadJSON, `"gate_id":`) || !strings.Contains(orders[0].PayloadJSON, `"review_findings":"`) {
		t.Fatalf("payload de corrección incompleto: %s", orders[0].PayloadJSON)
	}
	gate, err := db.GetReviewGate(gateID)
	if err != nil {
		t.Fatalf("get review gate: %v", err)
	}
	if gate == nil || gate.Estado != db.ReviewGateEnRevision {
		t.Fatalf("gate debería reabrirse a en_revision: %+v", gate)
	}
	policy, err := db.GetProyectoAutonomia(proyectoID)
	if err != nil {
		t.Fatalf("get proyecto autonomia: %v", err)
	}
	if policy.EstadoAutonomia != db.AutonomiaProyectoActiva {
		t.Fatalf("estado autonomia debería volver a activo, got=%s", policy.EstadoAutonomia)
	}
	kind := "review_feedback"
	cycles, err := supervisionService.ListCycles("orquestador", &kind, 5)
	if err != nil {
		t.Fatalf("listar ciclos review_feedback: %v", err)
	}
	if len(cycles) != 1 || cycles[0] == nil || cycles[0].Agente != "CodexSupervisor" {
		t.Fatalf("ciclo review_feedback inesperado: %+v", cycles)
	}
}

func TestProcesarReviewGatesBatchEncolaCorreccionTrasCambiosPedidos(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("CodexReviewer", "admin"); err != nil {
		t.Fatalf("registrar reviewer: %v", err)
	}
	if err := db.RegistrarAgente("CodexSupervisor", "admin"); err != nil {
		t.Fatalf("registrar supervisor: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := db.UpsertProyectoAutonomia(&db.ProyectoAutonomia{
		ProyectoID:           proyectoID,
		Enabled:              true,
		ObjetivoGeneral:      "Terminar la app",
		DefinitionOfDoneJSON: `{"done":true}`,
		SupervisorAgente:     "CodexSupervisor",
		ReviewerAgente:       "CodexReviewer",
		ReviewRequired:       true,
		AutoCloseProject:     true,
		EstadoAutonomia:      db.AutonomiaProyectoEsperandoReview,
	}); err != nil {
		t.Fatalf("upsert proyecto autonomia: %v", err)
	}
	if err := db.ActivarAsignacion("CodexReviewer", proyectoID, "review"); err != nil {
		t.Fatalf("activar asignacion reviewer: %v", err)
	}
	if err := db.ActivarAsignacion("CodexSupervisor", proyectoID, "correccion"); err != nil {
		t.Fatalf("activar asignacion supervisor: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "CodexReviewer",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion reviewer: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "CodexSupervisor",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion supervisor: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:     "Ultimo frente",
		ProyectoID: &proyectoID,
		Prioridad:  db.PrioridadAlta,
		CreadoPor:  "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "CodexReviewer"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "CodexReviewer"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	if err := db.CompletarTarea(tareaID, "CodexReviewer", "abc123"); err != nil {
		t.Fatalf("completar tarea: %v", err)
	}
	if _, err := db.CrearReviewGate(&db.ReviewGate{
		ProyectoID:     &proyectoID,
		RequestedBy:    "orquesta",
		ReviewerAgente: "CodexReviewer",
		Estado:         db.ReviewGateCambiosPed,
		SeverityMax:    "high",
		FindingsJSON:   `{"findings":["falta cerrar el refactor server-first"]}`,
	}); err != nil {
		t.Fatalf("crear review gate cambios_pedidos: %v", err)
	}

	n, err := procesarReviewGatesBatch()
	if err != nil {
		t.Fatalf("procesar review gates: %v", err)
	}
	if n != 1 {
		t.Fatalf("esperaba 1 nudge de correccion, got=%d", n)
	}

	agente := "CodexSupervisor"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	if len(orders) != 1 || orders[0].Tipo != "nudge" || !strings.Contains(orders[0].PayloadJSON, `"accion":"resolver_review_feedback"`) {
		todas, _ := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{ProyectoID: &proyectoID, Estado: &estado})
		t.Fatalf("nudge de correccion inesperado: agente=%+v todas=%+v", orders, todas)
	}
	if !strings.Contains(orders[0].PayloadJSON, `"gate_id":1`) {
		t.Fatalf("payload de correccion sin gate_id: %s", orders[0].PayloadJSON)
	}
	kind := "review_feedback"
	cycles, err := db.ListarAutonomiaCiclos(db.FiltroAutonomiaCiclos{ProyectoID: &proyectoID, Kind: &kind, Limit: 10})
	if err != nil {
		t.Fatalf("listar ciclos review_feedback: %v", err)
	}
	if len(cycles) != 1 || cycles[0].Agente != "CodexSupervisor" {
		t.Fatalf("ciclo review_feedback inesperado: %+v", cycles)
	}
	gates, err := db.ListarReviewGates(db.FiltroReviewGates{ProyectoID: &proyectoID, Limit: 10})
	if err != nil {
		t.Fatalf("listar gates: %v", err)
	}
	if len(gates) != 1 || gates[0].Estado != db.ReviewGateEnRevision {
		t.Fatalf("gate deberia reabrirse en_revision: %+v", gates)
	}
	policy, err := db.GetProyectoAutonomia(proyectoID)
	if err != nil {
		t.Fatalf("get proyecto autonomia: %v", err)
	}
	if policy.EstadoAutonomia != db.AutonomiaProyectoActiva {
		t.Fatalf("el proyecto deberia reactivarse para corregir, got=%s", policy.EstadoAutonomia)
	}
}

func TestProcesarReviewGatesBatchCreaSolicitudMergeTrasGateAprobado(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("CodexReviewer", "admin"); err != nil {
		t.Fatalf("registrar reviewer: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := db.UpsertProyectoAutonomia(&db.ProyectoAutonomia{
		ProyectoID:           proyectoID,
		Enabled:              true,
		ObjetivoGeneral:      "Terminar la app",
		DefinitionOfDoneJSON: `{"done":true}`,
		ReviewerAgente:       "CodexReviewer",
		ReviewRequired:       true,
		AutoCloseProject:     true,
		EstadoAutonomia:      db.AutonomiaProyectoEsperandoReview,
	}); err != nil {
		t.Fatalf("upsert proyecto autonomia: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:     "Ultimo frente",
		ProyectoID: &proyectoID,
		Prioridad:  db.PrioridadAlta,
		CreadoPor:  "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "CodexReviewer"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "CodexReviewer"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	if err := db.CompletarTarea(tareaID, "CodexReviewer", "abc123"); err != nil {
		t.Fatalf("completar tarea: %v", err)
	}
	worktree, err := db.CoordinationWorktreeSQLRepository{}.Create(&coordinacion.Worktree{
		ProjectID: proyectoID,
		TaskID:    &tareaID,
		Agent:     "CodexReviewer",
		Name:      "orquestador-codexreviewer-t3",
		Path:      filepath.Join(tmp, "wt-approved"),
		Branch:    "orq/orquestador/CodexReviewer/t3",
		BaseRef:   "master",
		State:     coordinacion.WorktreeActive,
		Reason:    "review_approved",
	})
	if err != nil {
		t.Fatalf("crear worktree activa: %v", err)
	}
	now := time.Now().UTC()
	if _, err := db.CrearReviewGate(&db.ReviewGate{
		ProyectoID:     &proyectoID,
		TareaID:        &tareaID,
		WorktreeID:     &worktree.ID,
		RequestedBy:    "orquesta",
		ReviewerAgente: "CodexReviewer",
		Estado:         db.ReviewGateAprobado,
		SeverityMax:    "high",
		ResolvedAt:     &now,
	}); err != nil {
		t.Fatalf("crear review gate aprobado: %v", err)
	}

	n, err := procesarReviewGatesBatch()
	if err != nil {
		t.Fatalf("procesar review gates: %v", err)
	}
	if n != 1 {
		t.Fatalf("esperaba 1 creación de solicitud de merge, got=%d", n)
	}

	merges, err := db.ListarGitMerges(&proyectoID, "")
	if err != nil {
		t.Fatalf("listar git merges: %v", err)
	}
	if len(merges) != 1 {
		t.Fatalf("debería crear una solicitud de merge, merges=%+v", merges)
	}
	if merges[0].SourceBranch != "orq/orquestador/CodexReviewer/t3" || merges[0].TargetBranch != "master" || merges[0].Estado != "aprobado" {
		t.Fatalf("solicitud de merge inesperada: %+v", merges[0])
	}
	if !strings.Contains(merges[0].MetadataJSON, `"review_gate":`) || !strings.Contains(merges[0].MetadataJSON, `"auto_created":true`) {
		t.Fatalf("metadata de merge incompleta: %s", merges[0].MetadataJSON)
	}
	policy, err := db.GetProyectoAutonomia(proyectoID)
	if err != nil {
		t.Fatalf("get proyecto autonomia: %v", err)
	}
	if policy.EstadoAutonomia != db.AutonomiaProyectoActiva {
		t.Fatalf("estado autonomia debería volver a activo tras review aprobada, got=%s", policy.EstadoAutonomia)
	}
}

func TestProyectoTerminadoAutonomamenteEsperaReviewAprobada(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := db.UpsertProyectoAutonomia(&db.ProyectoAutonomia{
		ProyectoID:           proyectoID,
		Enabled:              true,
		ObjetivoGeneral:      "Terminar la app",
		DefinitionOfDoneJSON: `{"done":true}`,
		ReviewRequired:       true,
		AutoCloseProject:     true,
		EstadoAutonomia:      db.AutonomiaProyectoActiva,
	}); err != nil {
		t.Fatalf("upsert proyecto autonomia: %v", err)
	}
	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:     "Ultimo frente",
		ProyectoID: &proyectoID,
		Prioridad:  db.PrioridadAlta,
		CreadoPor:  "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	if err := db.CompletarTarea(tareaID, "Codex1", "abc123"); err != nil {
		t.Fatalf("completar tarea: %v", err)
	}

	proyecto, err := db.GetProyecto(strconv.FormatInt(proyectoID, 10))
	if err != nil {
		t.Fatalf("get proyecto: %v", err)
	}
	terminado, _, err := proyectoTerminadoAutonomamente(proyecto)
	if err != nil {
		t.Fatalf("proyectoTerminadoAutonomamente sin review: %v", err)
	}
	if terminado {
		t.Fatal("no debería cerrar proyecto sin review aprobada")
	}
	now := time.Now().UTC()
	if _, err := db.CrearReviewGate(&db.ReviewGate{
		ProyectoID:  &proyectoID,
		RequestedBy: "orquesta",
		Estado:      db.ReviewGateAprobado,
		ResolvedAt:  &now,
	}); err != nil {
		t.Fatalf("crear review gate aprobado: %v", err)
	}
	terminado, motivo, err := proyectoTerminadoAutonomamente(proyecto)
	if err != nil {
		t.Fatalf("proyectoTerminadoAutonomamente con review: %v", err)
	}
	if !terminado {
		t.Fatalf("debería cerrar con review aprobada, motivo=%s", motivo)
	}
	if !strings.Contains(motivo, "review gate") {
		t.Fatalf("motivo de cierre sin referencia a review: %s", motivo)
	}
}

func TestProyectoTerminadoAutonomamenteEsperaMergeFusionadoSiExisteSolicitud(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)
	repo := prepararRepoGitAutonomia(t, filepath.Join(tmp, "orquestador"))

	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: repo,
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := db.UpsertProyectoAutonomia(&db.ProyectoAutonomia{
		ProyectoID:           proyectoID,
		Enabled:              true,
		ObjetivoGeneral:      "Terminar la app",
		DefinitionOfDoneJSON: `{"done":true}`,
		ReviewRequired:       true,
		AutoCloseProject:     true,
		EstadoAutonomia:      db.AutonomiaProyectoActiva,
	}); err != nil {
		t.Fatalf("upsert proyecto autonomia: %v", err)
	}
	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:     "Ultimo frente",
		ProyectoID: &proyectoID,
		Prioridad:  db.PrioridadAlta,
		CreadoPor:  "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	if err := db.CompletarTarea(tareaID, "Codex1", "abc123"); err != nil {
		t.Fatalf("completar tarea: %v", err)
	}
	now := time.Now().UTC()
	if _, err := db.CrearReviewGate(&db.ReviewGate{
		ProyectoID:  &proyectoID,
		RequestedBy: "orquesta",
		Estado:      db.ReviewGateAprobado,
		ResolvedAt:  &now,
	}); err != nil {
		t.Fatalf("crear review gate aprobado: %v", err)
	}
	if _, err := db.GuardarGitMerge(&db.GitMerge{
		ProyectoID:   proyectoID,
		SourceBranch: "feature/x",
		TargetBranch: "master",
		RequestedBy:  "orquesta",
		Estado:       "aprobado",
		MetadataJSON: `{"auto_created":true,"source":"review_gate_approved"}`,
	}); err != nil {
		t.Fatalf("guardar git merge: %v", err)
	}

	proyecto, err := db.GetProyecto(strconv.FormatInt(proyectoID, 10))
	if err != nil {
		t.Fatalf("get proyecto: %v", err)
	}
	terminado, motivo, err := proyectoTerminadoAutonomamente(proyecto)
	if err != nil {
		t.Fatalf("proyectoTerminadoAutonomamente esperando merge: %v", err)
	}
	if terminado {
		t.Fatalf("no debería cerrar con merge aprobado pendiente, motivo=%s", motivo)
	}
	if !strings.Contains(motivo, "esperando integración merge") {
		t.Fatalf("motivo inesperado esperando merge: %s", motivo)
	}

	merges, err := db.ListarGitMerges(&proyectoID, "")
	if err != nil {
		t.Fatalf("listar merges: %v", err)
	}
	if len(merges) != 1 {
		t.Fatalf("esperaba 1 merge, got=%d", len(merges))
	}
	merges[0].Estado = "fusionado"
	merges[0].CommitMerge = "def456"
	if _, err := db.GuardarGitMerge(merges[0]); err != nil {
		t.Fatalf("actualizar merge fusionado: %v", err)
	}
	terminado, motivo, err = proyectoTerminadoAutonomamente(proyecto)
	if err != nil {
		t.Fatalf("proyectoTerminadoAutonomamente con merge fusionado: %v", err)
	}
	if !terminado {
		t.Fatalf("debería cerrar con merge fusionado, motivo=%s", motivo)
	}
	if !strings.Contains(motivo, "fusionado") {
		t.Fatalf("motivo de cierre sin merge fusionado: %s", motivo)
	}
}

func TestProyectoTerminadoAutonomamenteRespetaAutoCloseProjectFalse(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := db.UpsertProyectoAutonomia(&db.ProyectoAutonomia{
		ProyectoID:           proyectoID,
		Enabled:              true,
		ObjetivoGeneral:      "Terminar la app",
		DefinitionOfDoneJSON: `{"done":true}`,
		ReviewRequired:       false,
		AutoCloseProject:     false,
		EstadoAutonomia:      db.AutonomiaProyectoActiva,
	}); err != nil {
		t.Fatalf("upsert proyecto autonomia: %v", err)
	}
	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:     "Ultimo frente",
		ProyectoID: &proyectoID,
		Prioridad:  db.PrioridadAlta,
		CreadoPor:  "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	if err := db.CompletarTarea(tareaID, "Codex1", "abc123"); err != nil {
		t.Fatalf("completar tarea: %v", err)
	}

	proyecto, err := db.GetProyecto(strconv.FormatInt(proyectoID, 10))
	if err != nil {
		t.Fatalf("get proyecto: %v", err)
	}
	terminado, _, err := proyectoTerminadoAutonomamente(proyecto)
	if err != nil {
		t.Fatalf("proyectoTerminadoAutonomamente auto_close=false: %v", err)
	}
	if terminado {
		t.Fatal("no debería considerar terminado un proyecto con auto_close_project=false")
	}
}

func TestProyectoTerminadoAutonomamenteRespetaAutoCloseProject(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := db.UpsertProyectoAutonomia(&db.ProyectoAutonomia{
		ProyectoID:           proyectoID,
		Enabled:              true,
		ObjetivoGeneral:      "Terminar la app",
		DefinitionOfDoneJSON: `{"done":true}`,
		ReviewRequired:       true,
		AutoCloseProject:     false,
		EstadoAutonomia:      db.AutonomiaProyectoActiva,
	}); err != nil {
		t.Fatalf("upsert proyecto autonomia: %v", err)
	}
	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:     "Ultimo frente",
		ProyectoID: &proyectoID,
		Prioridad:  db.PrioridadAlta,
		CreadoPor:  "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	if err := db.CompletarTarea(tareaID, "Codex1", "abc123"); err != nil {
		t.Fatalf("completar tarea: %v", err)
	}
	now := time.Now().UTC()
	if _, err := db.CrearReviewGate(&db.ReviewGate{
		ProyectoID:  &proyectoID,
		RequestedBy: "orquesta",
		Estado:      db.ReviewGateAprobado,
		ResolvedAt:  &now,
	}); err != nil {
		t.Fatalf("crear review gate aprobado: %v", err)
	}

	proyecto, err := db.GetProyecto(strconv.FormatInt(proyectoID, 10))
	if err != nil {
		t.Fatalf("get proyecto: %v", err)
	}
	terminado, motivo, err := proyectoTerminadoAutonomamente(proyecto)
	if err != nil {
		t.Fatalf("proyectoTerminadoAutonomamente: %v", err)
	}
	if terminado {
		t.Fatalf("no debería cerrar con auto_close_project desactivado, motivo=%s", motivo)
	}
}

func TestConstruirInstruccionSupervisionRespetaAutoCreateTasks(t *testing.T) {
	proyecto := &db.Proyecto{ID: 1, Slug: "orquestador"}
	supervisor := &db.Agente{Nombre: "CodexSupervisor", Rol: "admin"}

	conBacklog := construirInstruccionSupervision(&db.ProyectoAutonomia{
		Enabled:         true,
		ObjetivoGeneral: "Terminar la app",
		AutoCreateTasks: true,
	}, proyecto, supervisor, "faltan frentes")
	if !strings.Contains(conBacklog, "crea o reajusta tareas") {
		t.Fatalf("la instrucción debería permitir crear backlog automáticamente: %s", conBacklog)
	}

	sinBacklog := construirInstruccionSupervision(&db.ProyectoAutonomia{
		Enabled:         true,
		ObjetivoGeneral: "Terminar la app",
		AutoCreateTasks: false,
	}, proyecto, supervisor, "faltan frentes")
	if strings.Contains(sinBacklog, "crea o reajusta tareas") {
		t.Fatalf("la instrucción no debería permitir crear backlog automáticamente: %s", sinBacklog)
	}
	if !strings.Contains(sinBacklog, "sin crear tareas nuevas automáticamente") {
		t.Fatalf("la instrucción debería reflejar el modo sin auto_create_tasks: %s", sinBacklog)
	}
}

func TestProcesarAutonomiaAgentesBatchEncolaPausePorReasignacion(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoA, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador-a",
		Nombre:  "Orquestador A",
		RutaAbs: filepath.Join(tmp, "orquestador-a"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto A: %v", err)
	}
	proyectoB, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador-b",
		Nombre:  "Orquestador B",
		RutaAbs: filepath.Join(tmp, "orquestador-b"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto B: %v", err)
	}
	if err := db.ActivarAsignacion("Codex1", proyectoB, "cambio de frente"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoA,
		CWD:         filepath.Join(tmp, "orquestador-a"),
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}

	n, err := procesarAutonomiaAgentesBatch()
	if err != nil {
		t.Fatalf("procesar autonomia: %v", err)
	}
	if n != 1 {
		t.Fatalf("esperaba 1 decision autonoma, got=%d", n)
	}

	agente := "Codex1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, Estado: &estado})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	if len(orders) != 1 || orders[0].Tipo != "pause" {
		t.Fatalf("pause no encolada: %+v", orders)
	}
	if !strings.Contains(orders[0].PayloadJSON, `"accion":"pause"`) {
		t.Fatalf("payload pause inesperado: %s", orders[0].PayloadJSON)
	}
}

func TestProcesarCierreProyectoSesionRespetaAutoCloseProject(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := db.UpsertProyectoAutonomia(&db.ProyectoAutonomia{
		ProyectoID:           proyectoID,
		Enabled:              true,
		ObjetivoGeneral:      "Terminar la app",
		DefinitionOfDoneJSON: `{"done":true}`,
		ReviewRequired:       true,
		AutoCloseProject:     false,
		EstadoAutonomia:      db.AutonomiaProyectoActiva,
	}); err != nil {
		t.Fatalf("upsert proyecto autonomia: %v", err)
	}
	if err := db.ActivarAsignacion("Codex1", proyectoID, "frente principal"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:     "Ultimo frente",
		ProyectoID: &proyectoID,
		Prioridad:  db.PrioridadAlta,
		CreadoPor:  "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	if err := db.CompletarTarea(tareaID, "Codex1", "abc123"); err != nil {
		t.Fatalf("completar tarea: %v", err)
	}
	now := time.Now().UTC()
	if _, err := db.CrearReviewGate(&db.ReviewGate{
		ProyectoID:  &proyectoID,
		RequestedBy: "orquesta",
		Estado:      db.ReviewGateAprobado,
		ResolvedAt:  &now,
	}); err != nil {
		t.Fatalf("crear review gate aprobado: %v", err)
	}

	n, err := procesarCierreProyectoSesion(sesion)
	if err != nil {
		t.Fatalf("procesar cierre proyecto: %v", err)
	}
	if n != 0 {
		t.Fatalf("no debería cerrar con auto_close_project desactivado, got=%d", n)
	}
	op, err := db.GetProyectoOperacion(proyectoID)
	if err != nil {
		t.Fatalf("get proyecto operacion: %v", err)
	}
	if op.EstadoOperativo == db.ProyectoOperativoCerrado {
		t.Fatalf("el proyecto no debería quedar cerrado: %+v", op)
	}
}

func TestProcesarAutonomiaAgentesBatchEncolaNudgePorPropuestasPendientes(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if err := db.ActivarAsignacion("Codex1", proyectoID, "frente principal"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	if _, err := db.CrearPropuesta(&db.Propuesta{
		Titulo:       "Nueva política",
		Descripcion:  "Pendiente de votar",
		ProyectoID:   &proyectoID,
		Tipo:         "otro",
		PropuestoPor: "alberto",
		Distribuidor: "claude",
	}); err != nil {
		t.Fatalf("crear propuesta: %v", err)
	}

	n, err := procesarAutonomiaAgentesBatch()
	if err != nil {
		t.Fatalf("procesar autonomia: %v", err)
	}
	if n != 1 {
		t.Fatalf("esperaba 1 decision autonoma, got=%d", n)
	}

	agente := "Codex1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	if len(orders) != 1 || orders[0].Tipo != "nudge" {
		t.Fatalf("nudge no encolado: %+v", orders)
	}
	if !strings.Contains(orders[0].PayloadJSON, `"kind":"autonomia"`) ||
		!strings.Contains(orders[0].PayloadJSON, `"accion":"votar_propuestas_pendientes"`) {
		t.Fatalf("payload nudge inesperado: %s", orders[0].PayloadJSON)
	}
}

func TestProcesarAutonomiaAgentesBatchNoEncolaNudgePorContinuarTrabajoEnSesionActiva(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if err := db.ActivarAsignacion("Codex1", proyectoID, "frente principal"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	if _, err := db.UpsertConector(&db.Conector{
		Slug:       "codex-remote",
		Nombre:     "Codex Remote",
		Transporte: "api",
		Comando:    "http://localhost:9999",
		Activo:     true,
	}); err != nil {
		t.Fatalf("upsert conector: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Seguir implementando",
		Descripcion: "Trabajo activo",
		ProyectoID:  &proyectoID,
		Modulo:      "core",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}

	n, err := procesarAutonomiaAgentesBatch()
	if err != nil {
		t.Fatalf("procesar autonomia: %v", err)
	}
	if n != 0 {
		t.Fatalf("no deberia forzar nudge pasivo en sesion activa, got=%d", n)
	}

	agente := "Codex1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	if len(orders) != 0 {
		t.Fatalf("no deberia crear nudge pasivo: %+v", orders)
	}
}

func TestProcesarAutonomiaAgentesBatchNoEncolaNudgePorEsperarOPedirTareaEnSesionActiva(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if err := db.ActivarAsignacion("Codex1", proyectoID, "frente principal"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}

	n, err := procesarAutonomiaAgentesBatch()
	if err != nil {
		t.Fatalf("procesar autonomia: %v", err)
	}
	if n != 0 {
		t.Fatalf("no deberia forzar nudge pasivo en sesion activa, got=%d", n)
	}

	agente := "Codex1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	if len(orders) != 0 {
		t.Fatalf("no deberia crear nudge pasivo: %+v", orders)
	}
}

func TestProcesarAutonomiaAgentesBatchAutoasignaTrabajoASesionActivaIdle(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.ConfigSet("server_autobootstrap_project_slug", "orquestador"); err != nil {
		t.Fatalf("config project slug: %v", err)
	}
	if err := db.ConfigSet("server_autobootstrap_worker_agents", "Codex1"); err != nil {
		t.Fatalf("config worker agents: %v", err)
	}
	if err := db.RegistrarAgente("CodexSupervisor", "admin"); err != nil {
		t.Fatalf("registrar supervisor: %v", err)
	}
	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := db.UpsertProyectoAutonomia(&db.ProyectoAutonomia{
		ProyectoID:        proyectoID,
		Enabled:           true,
		MaxWorkers:        1,
		SupervisorAgente:  "CodexSupervisor",
		ReserveSupervisor: true,
		EstadoAutonomia:   db.AutonomiaProyectoActiva,
	}); err != nil {
		t.Fatalf("upsert autonomia: %v", err)
	}
	if err := db.ActivarAsignacion("CodexSupervisor", proyectoID, "supervision"); err != nil {
		t.Fatalf("activar asignacion supervisor: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "CodexSupervisor",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion supervisor: %v", err)
	}
	if err := db.ActivarAsignacion("Codex1", proyectoID, "frente principal"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Siguiente frente útil",
		Descripcion: "Debe entrar en sesión activa idle",
		ProyectoID:  &proyectoID,
		Modulo:      "planocontrol",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}

	n, err := procesarAutonomiaSesionActiva(sesion)
	if err != nil {
		t.Fatalf("procesar autonomia: %v", err)
	}
	if n != 1 {
		t.Fatalf("deberia autoasignar y encolar nudge útil, got=%d", n)
	}

	tarea, err := db.GetTarea(tareaID)
	if err != nil {
		t.Fatalf("get tarea: %v", err)
	}
	if tarea.Agente == nil || *tarea.Agente != "Codex1" || tarea.Estado != db.TareaAsignada {
		t.Fatalf("la tarea deberia quedar autoasignada a la sesion viva: %+v", tarea)
	}

	agente := "Codex1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	if len(orders) != 1 || orders[0].Tipo != "nudge" {
		t.Fatalf("deberia encolar un nudge útil tras autoasignar: %+v", orders)
	}
	if !strings.Contains(orders[0].PayloadJSON, `"accion":"continuar_trabajo"`) || !strings.Contains(orders[0].PayloadJSON, `"tarea_id":`) {
		t.Fatalf("payload nudge inesperado: %s", orders[0].PayloadJSON)
	}
	if !strings.Contains(orders[0].PayloadJSON, `"instruction":"toma tarea asignada y sigue"`) {
		t.Fatalf("instruction de autoasignacion demasiado larga o inesperada: %s", orders[0].PayloadJSON)
	}
}

func TestProcesarAutonomiaAgentesBatchNoEncolaNudgePorSupervisarProyectoEnSesionActiva(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if err := db.ActivarAsignacion("Codex1", proyectoID, "supervision"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	if _, err := db.UpsertProyectoAutonomia(&db.ProyectoAutonomia{
		ProyectoID:           proyectoID,
		Enabled:              true,
		ObjetivoGeneral:      "Terminar la app",
		DefinitionOfDoneJSON: `{"done":true}`,
		SupervisorAgente:     "Codex1",
		EstadoAutonomia:      db.AutonomiaProyectoActiva,
	}); err != nil {
		t.Fatalf("upsert proyecto autonomia: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}

	n, err := procesarAutonomiaAgentesBatch()
	if err != nil {
		t.Fatalf("procesar autonomia: %v", err)
	}
	if n != 0 {
		t.Fatalf("no deberia forzar nudge de supervisor en sesion activa, got=%d", n)
	}

	agente := "Codex1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	if len(orders) != 0 {
		t.Fatalf("no deberia crear nudge supervisor pasivo: %+v", orders)
	}
}

func TestProcesarAutonomiaAgentesBatchNoRepiteNudgeRecienteMaterializado(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if err := db.ActivarAsignacion("Codex1", proyectoID, "frente principal"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Seguir implementando",
		Descripcion: "Trabajo activo",
		ProyectoID:  &proyectoID,
		Modulo:      "core",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}

	n, err := procesarAutonomiaAgentesBatch()
	if err != nil {
		t.Fatalf("procesar autonomia inicial: %v", err)
	}
	if n != 0 {
		t.Fatalf("no deberia forzar nudge pasivo inicial, got=%d", n)
	}
	if _, err := db.ProcesarRuntimeOrdersBatch(); err != nil {
		t.Fatalf("procesar runtime orders: %v", err)
	}
	if _, err := procesarRuntimeMailboxBatch(); err != nil {
		t.Fatalf("procesar runtime mailbox: %v", err)
	}

	n, err = procesarAutonomiaAgentesBatch()
	if err != nil {
		t.Fatalf("procesar autonomia repetida: %v", err)
	}
	if n != 0 {
		t.Fatalf("no deberia repetir el nudge reciente, got=%d", n)
	}

	agente := "Codex1"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	var nudges, sendInstructions int
	for _, order := range orders {
		if order == nil {
			continue
		}
		switch order.Tipo {
		case "nudge":
			nudges++
		case "send_instruction":
			sendInstructions++
			if order.HandleID == nil || *order.HandleID != handle.ID {
				t.Fatalf("send_instruction sin handle esperado: %+v", order)
			}
		}
	}
	if nudges != 0 || sendInstructions != 0 {
		t.Fatalf("ordenes de autonomia duplicadas: nudges=%d send_instruction=%d orders=%+v", nudges, sendInstructions, orders)
	}
}

func TestResetReanimacionEncolaResumeCuandoHayHandlePausado(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if err := db.ActivarAsignacion("Codex1", proyectoID, "frente principal"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	if _, err := db.UpsertConector(&db.Conector{
		Slug:       "codex-remote",
		Nombre:     "Codex Remote",
		Transporte: "api",
		Comando:    "http://localhost:9999",
		Activo:     true,
	}); err != nil {
		t.Fatalf("upsert conector: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Retomar tras enfriamiento",
		Descripcion: "Trabajo activo",
		ProyectoID:  &proyectoID,
		Modulo:      "core",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET estado='pausado' WHERE id=?`, handle.ID); err != nil {
		t.Fatalf("pausar handle: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE agentes SET reanimar_at=CURRENT_TIMESTAMP, estado_cuota='enfriamiento', motivo_pausa='cuota' WHERE nombre='Codex1'`); err != nil {
		t.Fatalf("marcar reanimacion: %v", err)
	}

	if err := (dbAutomationService{}).ResetReanimacion("Codex1"); err != nil {
		t.Fatalf("reset reanimacion: %v", err)
	}

	agente := "Codex1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	if len(orders) != 1 || orders[0].Tipo != "resume" {
		t.Fatalf("resume no encolado: %+v", orders)
	}
	if !strings.Contains(orders[0].PayloadJSON, `"accion":"resume"`) {
		t.Fatalf("payload resume inesperado: %s", orders[0].PayloadJSON)
	}
}

func TestProcesarRuntimeMailboxInteractivoBatchOmiteHandlesSinInputInteractivo(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	if _, err := db.DB.Exec(
		`UPDATE runtime_handles SET capabilities_json=?, metadata_json=? WHERE id=?`,
		`{"can_send_input":false}`,
		`{"driver":"process_pty_cli","rendered_command":"codex-perfil Codex1","can_send_input":false}`,
		handle.ID,
	); err != nil {
		t.Fatalf("update handle: %v", err)
	}

	msgID, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "autonomia",
		PayloadJSON: `{"accion":"continuar_trabajo","motivo":"seguir frente"}`,
	})
	if err != nil {
		t.Fatalf("mailbox: %v", err)
	}

	n, err := procesarRuntimeMailboxInteractivoBatch()
	if err != nil {
		t.Fatalf("procesar mailbox interactivo: %v", err)
	}
	if n != 0 {
		t.Fatalf("no deberia crear send_instruction interactiva, got=%d", n)
	}

	agente := "Codex1"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	for _, order := range orders {
		if order != nil && order.Tipo == "send_instruction" {
			t.Fatalf("no deberia crear send_instruction para handle sin input interactivo: %+v", order)
		}
	}

	estado := "pendiente"
	mailbox, err := db.ListarRuntimeMailbox(db.FiltroRuntimeMailbox{ToAgente: &agente, Estado: &estado})
	if err != nil {
		t.Fatalf("listar mailbox: %v", err)
	}
	if len(mailbox) != 1 || mailbox[0].ID != msgID {
		t.Fatalf("mailbox pendiente inesperada: %+v", mailbox)
	}
}

func TestProcesarRuntimeMailboxSupervisorLocalBatchEncolaSendInstructionParaCodexSupervisado(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	metaJSON := `{"driver":"process_pty_cli","stdin_path":"` + filepath.Join(tmp, "pty.stdin") + `","supervisor_ref":"` + filepath.Join(tmp, "supervisor.ref") + `","rendered_command":"codex-perfil Codex1","working_dir":"` + filepath.Join(tmp, "orquestador") + `","external_session_id":"sess-supervisor-local","can_send_input":false}`
	capsJSON := `{"can_send_input":false,"mailbox_delivery_mode":"` + runtimeagente.MailboxDeliveryBootstrapOnly + `"}`
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET capabilities_json=?, metadata_json=? WHERE id=?`, capsJSON, metaJSON, handle.ID); err != nil {
		t.Fatalf("update handle: %v", err)
	}

	msgID, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "autonomia",
		PayloadJSON: `{"accion":"continuar_trabajo","motivo":"seguir frente"}`,
	})
	if err != nil {
		t.Fatalf("mailbox: %v", err)
	}

	n, err := procesarRuntimeMailboxSupervisorLocalBatch()
	if err != nil {
		t.Fatalf("procesar mailbox supervisor local: %v", err)
	}
	if n != 1 {
		t.Fatalf("deberia crear una sola send_instruction por supervisor local, got=%d", n)
	}

	agente := "Codex1"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	var send *db.RuntimeOrder
	for _, order := range orders {
		if order != nil && order.Tipo == "send_instruction" {
			send = order
			break
		}
	}
	if send == nil {
		t.Fatal("faltaba send_instruction supervisor local")
	}
	if !strings.Contains(send.PayloadJSON, `"mailbox_id":`+strconv.FormatInt(msgID, 10)) {
		t.Fatalf("send_instruction sin mailbox_id: %s", send.PayloadJSON)
	}
	if !strings.Contains(send.PayloadJSON, `"delivery_attempt_signature":"supervisor_local|handle:`) {
		t.Fatalf("firma de intento supervisor_local inesperada: %s", send.PayloadJSON)
	}
	if !strings.Contains(send.PayloadJSON, `"external_session_id":"sess-supervisor-local"`) {
		t.Fatalf("send_instruction sin external_session_id viva: %s", send.PayloadJSON)
	}

	pendiente := "pendiente"
	mailboxPendiente, err := db.ListarRuntimeMailbox(db.FiltroRuntimeMailbox{ToAgente: &agente, Estado: &pendiente})
	if err != nil {
		t.Fatalf("listar mailbox pendiente: %v", err)
	}
	if len(mailboxPendiente) != 1 || mailboxPendiente[0].ID != msgID {
		t.Fatalf("la mailbox durable debe seguir pendiente hasta entrega real: %+v", mailboxPendiente)
	}
}

func TestProcesarRuntimeMailboxBatchCoordinaRestartParaHandlesBootstrapOnly(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	if _, err := db.DB.Exec(
		`UPDATE runtime_handles SET transporte='cli', handle_kind='process', capabilities_json=?, metadata_json=? WHERE id=?`,
		`{"can_send_input":false,"mailbox_delivery_mode":"coordinated_restart"}`,
		`{"driver":"process_pty_cli","rendered_command":"codex-perfil Codex1","can_send_input":false,"mailbox_delivery_mode":"coordinated_restart"}`,
		handle.ID,
	); err != nil {
		t.Fatalf("update handle: %v", err)
	}

	msgViejo, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "autonomia",
		PayloadJSON: `{"accion":"continuar_trabajo","motivo":"mensaje viejo"}`,
	})
	if err != nil {
		t.Fatalf("mailbox vieja: %v", err)
	}
	msgNuevo, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "autonomia",
		PayloadJSON: `{"accion":"continuar_trabajo","motivo":"mensaje nuevo"}`,
	})
	if err != nil {
		t.Fatalf("mailbox nueva: %v", err)
	}

	n, err := procesarRuntimeMailboxBatch()
	if err != nil {
		t.Fatalf("procesar mailbox batch: %v", err)
	}
	if n != 0 {
		t.Fatalf("codex-cli no deberia reiniciarse por mailbox rutinario, got=%d", n)
	}

	agente := "Codex1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar runtime orders: %v", err)
	}
	if len(orders) != 0 {
		t.Fatalf("no deberia crear ordenes de restart: %+v", orders)
	}

	pendiente := "pendiente"
	consumido := "consumido"
	mailboxPendiente, err := db.ListarRuntimeMailbox(db.FiltroRuntimeMailbox{ToAgente: &agente, ProyectoID: &proyectoID, Estado: &pendiente})
	if err != nil {
		t.Fatalf("listar mailbox pendiente: %v", err)
	}
	if len(mailboxPendiente) != 1 || mailboxPendiente[0].ID != msgNuevo {
		t.Fatalf("deberia quedar solo la mailbox mas reciente pendiente: %+v", mailboxPendiente)
	}
	mailboxConsumido, err := db.ListarRuntimeMailbox(db.FiltroRuntimeMailbox{ToAgente: &agente, ProyectoID: &proyectoID, Estado: &consumido})
	if err != nil {
		t.Fatalf("listar mailbox consumido: %v", err)
	}
	if len(mailboxConsumido) != 1 || mailboxConsumido[0].ID != msgViejo {
		t.Fatalf("mailbox consumida inesperada: %+v", mailboxConsumido)
	}
}

func TestProcesarRuntimeMailboxBatchNoDuplicaReinicioCoordinadoAbierto(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	if _, err := db.DB.Exec(
		`UPDATE runtime_handles SET transporte='cli', handle_kind='process', capabilities_json=?, metadata_json=? WHERE id=?`,
		`{"can_send_input":false,"mailbox_delivery_mode":"coordinated_restart"}`,
		`{"driver":"process_pty_cli","rendered_command":"codex-perfil Codex1","can_send_input":false,"mailbox_delivery_mode":"coordinated_restart"}`,
		handle.ID,
	); err != nil {
		t.Fatalf("update handle: %v", err)
	}
	if _, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "instruction",
		PayloadJSON: `{"texto":"seguir"}`,
	}); err != nil {
		t.Fatalf("mailbox: %v", err)
	}
	if _, err := runtimesService.EnqueueRuntimeOrder(&db.RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		HandleID:    &handle.ID,
		Tipo:        agenteControlAccionStop,
		PayloadJSON: `{"accion":"stop","proyecto":"orquestador","motivo":"ya pendiente"}`,
	}); err != nil {
		t.Fatalf("encolar stop: %v", err)
	}

	n, err := procesarRuntimeMailboxBatch()
	if err != nil {
		t.Fatalf("procesar mailbox batch: %v", err)
	}
	if n != 0 {
		t.Fatalf("no deberia duplicar reinicio coordinado, got=%d", n)
	}

	agente := "Codex1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar runtime orders: %v", err)
	}
	if len(orders) != 1 || orders[0].Tipo != agenteControlAccionStop {
		t.Fatalf("ordenes pendientes inesperadas: %+v", orders)
	}
}

func TestProcesarRuntimeMailboxSessionResumeBatchEncolaSendInstructionSinConsumirMailbox(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	metaJSON := `{"driver":"process_pty_cli","rendered_command":"codex-perfil Codex1","working_dir":"` + filepath.Join(tmp, "orquestador") + `","external_session_id":"sess-batch-resume","can_send_input":false}`
	capsJSON := `{"can_send_input":false,"mailbox_delivery_mode":"` + runtimeagente.MailboxDeliverySessionResume + `"}`
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET capabilities_json=?, metadata_json=? WHERE id=?`, capsJSON, metaJSON, handle.ID); err != nil {
		t.Fatalf("update handle: %v", err)
	}

	msgID, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "instruction",
		PayloadJSON: `{"texto":"instruccion session resume"}`,
	})
	if err != nil {
		t.Fatalf("mailbox: %v", err)
	}

	n, err := procesarRuntimeMailboxSessionResumeBatch()
	if err != nil {
		t.Fatalf("procesar mailbox session resume: %v", err)
	}
	if n != 1 {
		t.Fatalf("deberia crear una sola send_instruction por session_resume, got=%d", n)
	}

	agente := "Codex1"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	var send *db.RuntimeOrder
	for _, order := range orders {
		if order != nil && order.Tipo == "send_instruction" {
			send = order
			break
		}
	}
	if send == nil {
		t.Fatalf("faltaba send_instruction")
	}
	if !strings.Contains(send.PayloadJSON, `"mailbox_id":`+strconv.FormatInt(msgID, 10)) {
		t.Fatalf("send_instruction sin mailbox_id: %s", send.PayloadJSON)
	}
	if !strings.Contains(send.PayloadJSON, `"external_session_id":"sess-batch-resume"`) {
		t.Fatalf("send_instruction sin external_session_id: %s", send.PayloadJSON)
	}

	pendiente := "pendiente"
	mailboxPendiente, err := db.ListarRuntimeMailbox(db.FiltroRuntimeMailbox{ToAgente: &agente, Estado: &pendiente})
	if err != nil {
		t.Fatalf("listar mailbox pendiente: %v", err)
	}
	if len(mailboxPendiente) != 1 || mailboxPendiente[0].ID != msgID {
		t.Fatalf("la mailbox debe seguir pendiente hasta entregar de verdad: %+v", mailboxPendiente)
	}
}

func TestProcesarRuntimeMailboxSessionResumeBatchEncolaSendInstructionParaWatchdog(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	metaJSON := `{"driver":"process_pty_cli","rendered_command":"codex-perfil Codex1","working_dir":"` + filepath.Join(tmp, "orquestador") + `","external_session_id":"sess-watchdog"}`
	capsJSON := `{"can_send_input":false,"mailbox_delivery_mode":"` + runtimeagente.MailboxDeliverySessionResume + `"}`
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET capabilities_json=?, metadata_json=? WHERE id=?`, capsJSON, metaJSON, handle.ID); err != nil {
		t.Fatalf("update handle: %v", err)
	}

	msgID, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "watchdog",
		PayloadJSON: `{"texto":"confirma estado o reanuda tick"}`,
	})
	if err != nil {
		t.Fatalf("mailbox watchdog: %v", err)
	}

	n, err := procesarRuntimeMailboxSessionResumeBatch()
	if err != nil {
		t.Fatalf("procesar mailbox session resume: %v", err)
	}
	if n != 1 {
		t.Fatalf("deberia crear una sola send_instruction para watchdog, got=%d", n)
	}

	agente := "Codex1"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	var send *db.RuntimeOrder
	for _, order := range orders {
		if order != nil && order.Tipo == "send_instruction" {
			send = order
			break
		}
	}
	if send == nil {
		t.Fatalf("faltaba send_instruction")
	}
	if !strings.Contains(send.PayloadJSON, `"mailbox_id":`+strconv.FormatInt(msgID, 10)) {
		t.Fatalf("send_instruction sin mailbox_id: %s", send.PayloadJSON)
	}
	if !strings.Contains(send.PayloadJSON, `"texto":"confirma estado o reanuda tick"`) {
		t.Fatalf("send_instruction watchdog sin texto: %s", send.PayloadJSON)
	}
}

func TestProcesarRuntimeMailboxSessionResumeBatchRespetaLeaseBootstrapPendiente(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	metaJSON := `{"driver":"process_pty_cli","rendered_command":"codex-perfil Codex1","working_dir":"` + filepath.Join(tmp, "orquestador") + `","external_session_id":"sess-bootstrap-resume","can_send_input":false}`
	capsJSON := `{"can_send_input":false,"mailbox_delivery_mode":"` + runtimeagente.MailboxDeliverySessionResume + `"}`
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET capabilities_json=?, metadata_json=? WHERE id=?`, capsJSON, metaJSON, handle.ID); err != nil {
		t.Fatalf("update handle: %v", err)
	}

	msgID, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "autonomia",
		PayloadJSON: `{"texto":"arranque autonomo"}`,
	})
	if err != nil {
		t.Fatalf("mailbox: %v", err)
	}
	startID, err := db.EncolarRuntimeOrder(&db.RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "start",
		PayloadJSON: `{"proyecto":"orquestador"}`,
	})
	if err != nil {
		t.Fatalf("encolar start: %v", err)
	}
	resumeID, err := db.EncolarRuntimeOrder(&db.RuntimeOrder{
		Agente:     "Codex1",
		ProyectoID: &proyectoID,
		Tipo:       "resume",
		ResultadoJSON: fmt.Sprintf(
			`{"lease_state":"waiting_for_evidence","start_order_id":%d,"mailbox_ids":[%d],"sesion_id":%d}`,
			startID, msgID, sesion.ID,
		),
	})
	if err != nil {
		t.Fatalf("encolar resume bootstrap: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE runtime_orders SET estado='ejecutando', started_at=CURRENT_TIMESTAMP, updated_at=CURRENT_TIMESTAMP WHERE id=?`, resumeID); err != nil {
		t.Fatalf("marcar resume ejecutando: %v", err)
	}

	n, err := procesarRuntimeMailboxSessionResumeBatch()
	if err != nil {
		t.Fatalf("procesar mailbox session resume: %v", err)
	}
	if n != 0 {
		t.Fatalf("no deberia crear send_instruction si el mailbox ya esta cubierto por bootstrap, got=%d", n)
	}

	agente := "Codex1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	for _, order := range orders {
		if order != nil && order.Tipo == "send_instruction" {
			t.Fatalf("no deberia existir send_instruction duplicada: %+v", order)
		}
	}
}

func TestProcesarRuntimeMailboxSessionResumeBatchNoRematerializaMailboxOnlyEnMismaSesion(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	metaJSON := `{"driver":"process_pty_cli","rendered_command":"codex-perfil Codex1","working_dir":"` + filepath.Join(tmp, "orquestador") + `","external_session_id":"sess-dedupe","can_send_input":false}`
	capsJSON := `{"can_send_input":false,"mailbox_delivery_mode":"` + runtimeagente.MailboxDeliverySessionResume + `"}`
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET capabilities_json=?, metadata_json=? WHERE id=?`, capsJSON, metaJSON, handle.ID); err != nil {
		t.Fatalf("update handle: %v", err)
	}

	msgID, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "instruction",
		PayloadJSON: `{"texto":"instruccion durable"}`,
	})
	if err != nil {
		t.Fatalf("mailbox: %v", err)
	}

	payload := fmt.Sprintf(`{"to_agente":"Codex1","from_agente":"server","texto":"instruccion durable","mailbox_id":%d,"mailbox_kind":"instruction","external_session_id":"sess-dedupe","delivery_attempt_signature":"session_resume|handle:%d|session:sess-dedupe"}`, msgID, handle.ID)
	resultado := fmt.Sprintf(`{"ok":true,"mailbox_id":%d,"deferred":true,"deferred_reason":"runtime no disponible para entrega inmediata","mailbox_only":true}`, msgID)
	if _, err := db.EncolarRuntimeOrder(&db.RuntimeOrder{
		Agente:        "Codex1",
		ProyectoID:    &proyectoID,
		RuntimeID:     handle.RuntimeID,
		HandleID:      &handle.ID,
		Tipo:          "send_instruction",
		PayloadJSON:   payload,
		ResultadoJSON: resultado,
		Estado:        "completada",
	}); err != nil {
		t.Fatalf("encolar send_instruction previa: %v", err)
	}

	n, err := procesarRuntimeMailboxSessionResumeBatch()
	if err != nil {
		t.Fatalf("procesar mailbox session resume: %v", err)
	}
	if n != 0 {
		t.Fatalf("no deberia rematerializar send_instruction mailbox_only en la misma sesion, got=%d", n)
	}

	agente := "Codex1"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	var sendCount int
	for _, order := range orders {
		if order != nil && order.Tipo == "send_instruction" {
			sendCount++
		}
	}
	if sendCount != 1 {
		t.Fatalf("no deberia crear una nueva send_instruction: %d", sendCount)
	}
}

func TestProcesarRuntimeMailboxSessionResumeBatchPermiteReintentoDeMailboxOnlyLegacySinFirma(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	metaJSON := `{"driver":"process_pty_cli","rendered_command":"codex-perfil Codex1","working_dir":"` + filepath.Join(tmp, "orquestador") + `","external_session_id":"sess-live","can_send_input":false}`
	capsJSON := `{"can_send_input":false,"mailbox_delivery_mode":"` + runtimeagente.MailboxDeliverySessionResume + `"}`
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET capabilities_json=?, metadata_json=? WHERE id=?`, capsJSON, metaJSON, handle.ID); err != nil {
		t.Fatalf("update handle: %v", err)
	}

	msgID, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "instruction",
		PayloadJSON: `{"texto":"legacy mailbox"}`,
	})
	if err != nil {
		t.Fatalf("mailbox: %v", err)
	}

	payload := fmt.Sprintf(`{"to_agente":"Codex1","from_agente":"server","texto":"legacy mailbox","mailbox_id":%d,"mailbox_kind":"instruction","external_session_id":"sess-live"}`, msgID)
	resultado := fmt.Sprintf(`{"ok":true,"mailbox_id":%d,"deferred":true,"deferred_reason":"runtime no disponible para entrega inmediata","mailbox_only":true}`, msgID)
	if _, err := db.EncolarRuntimeOrder(&db.RuntimeOrder{
		Agente:        "Codex1",
		ProyectoID:    &proyectoID,
		RuntimeID:     handle.RuntimeID,
		HandleID:      &handle.ID,
		Tipo:          "send_instruction",
		PayloadJSON:   payload,
		ResultadoJSON: resultado,
		Estado:        "completada",
	}); err != nil {
		t.Fatalf("encolar send_instruction legacy: %v", err)
	}

	n, err := procesarRuntimeMailboxSessionResumeBatch()
	if err != nil {
		t.Fatalf("procesar mailbox session resume: %v", err)
	}
	if n != 1 {
		t.Fatalf("deberia permitir un reintento cuando falta firma de intento, got=%d", n)
	}

	agente := "Codex1"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	var sendCount int
	var newest *db.RuntimeOrder
	for _, order := range orders {
		if order != nil && order.Tipo == "send_instruction" {
			sendCount++
			if newest == nil || order.ID > newest.ID {
				newest = order
			}
		}
	}
	if sendCount != 2 {
		t.Fatalf("deberia crear una nueva send_instruction para el retry util, got=%d", sendCount)
	}
	if newest == nil || !strings.Contains(newest.PayloadJSON, `"delivery_attempt_signature":"session_resume|handle:`) {
		t.Fatalf("la orden nueva deberia persistir la firma del intento: %+v", newest)
	}
}

func TestProcesarRuntimeMailboxSessionResumeBatchRematerializaMailboxOnlyCaducado(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)
	old := time.Now().UTC().Add(-2 * runtimeMailboxOnlyDedupeTTL())

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	metaJSON := `{"driver":"process_pty_cli","rendered_command":"codex-perfil Codex1","working_dir":"` + filepath.Join(tmp, "orquestador") + `","external_session_id":"sess-expirada","can_send_input":false}`
	capsJSON := `{"can_send_input":false,"mailbox_delivery_mode":"` + runtimeagente.MailboxDeliverySessionResume + `"}`
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET capabilities_json=?, metadata_json=? WHERE id=?`, capsJSON, metaJSON, handle.ID); err != nil {
		t.Fatalf("update handle: %v", err)
	}

	msgID, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "instruction",
		PayloadJSON: `{"texto":"instruccion durable caducada"}`,
	})
	if err != nil {
		t.Fatalf("mailbox: %v", err)
	}

	payload := fmt.Sprintf(`{"to_agente":"Codex1","from_agente":"server","texto":"instruccion durable caducada","mailbox_id":%d,"mailbox_kind":"instruction","external_session_id":"sess-expirada","delivery_attempt_signature":"session_resume|handle:%d|session:sess-expirada"}`, msgID, handle.ID)
	resultado := fmt.Sprintf(`{"ok":true,"mailbox_id":%d,"deferred":true,"deferred_reason":"runtime no disponible para entrega inmediata","mailbox_only":true}`, msgID)
	orderID, err := int64(0), error(nil)
	if orderID, err = db.EncolarRuntimeOrder(&db.RuntimeOrder{
		Agente:        "Codex1",
		ProyectoID:    &proyectoID,
		RuntimeID:     handle.RuntimeID,
		HandleID:      &handle.ID,
		Tipo:          "send_instruction",
		PayloadJSON:   payload,
		ResultadoJSON: resultado,
		Estado:        "completada",
		FinishedAt:    &old,
		UpdatedAt:     old,
	}); err != nil {
		t.Fatalf("encolar send_instruction previa: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE runtime_orders SET finished_at=?, updated_at=? WHERE id=?`,
		old,
		old,
		orderID,
	); err != nil {
		t.Fatalf("envejecer send_instruction previa: %v", err)
	}

	n, err := procesarRuntimeMailboxSessionResumeBatch()
	if err != nil {
		t.Fatalf("procesar mailbox session resume: %v", err)
	}
	if n != 1 {
		t.Fatalf("deberia rematerializar send_instruction mailbox_only caducada, got=%d", n)
	}

	agente := "Codex1"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	var sendCount int
	for _, order := range orders {
		if order != nil && order.Tipo == "send_instruction" {
			sendCount++
		}
	}
	if sendCount != 2 {
		t.Fatalf("deberia crear una nueva send_instruction tras caducar el intento previo: %d", sendCount)
	}
}

func TestProcesarRuntimeMailboxSessionResumeBatchNoRematerializaGuidanceCaducadaEnMismaSesion(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)
	old := time.Now().UTC().Add(-2 * runtimeMailboxOnlyDedupeTTL())

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	metaJSON := `{"driver":"process_pty_cli","rendered_command":"codex-perfil Codex1","working_dir":"` + filepath.Join(tmp, "orquestador") + `","external_session_id":"sess-guidance","can_send_input":false}`
	capsJSON := `{"can_send_input":false,"mailbox_delivery_mode":"` + runtimeagente.MailboxDeliverySessionResume + `"}`
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET capabilities_json=?, metadata_json=? WHERE id=?`, capsJSON, metaJSON, handle.ID); err != nil {
		t.Fatalf("update handle: %v", err)
	}

	msgID, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "autonomia",
		PayloadJSON: `{"texto":"reevalua bloqueo y sigue"}`,
	})
	if err != nil {
		t.Fatalf("mailbox: %v", err)
	}

	payload := fmt.Sprintf(`{"to_agente":"Codex1","from_agente":"server","texto":"reevalua bloqueo y sigue","mailbox_id":%d,"mailbox_kind":"autonomia","external_session_id":"sess-guidance","delivery_attempt_signature":"session_resume|handle:%d|session:sess-guidance"}`, msgID, handle.ID)
	resultado := fmt.Sprintf(`{"ok":true,"mailbox_id":%d,"deferred":true,"deferred_reason":"session_resume timeout: Perfil activo: Codex1","mailbox_only":true}`, msgID)
	orderID, err := db.EncolarRuntimeOrder(&db.RuntimeOrder{
		Agente:        "Codex1",
		ProyectoID:    &proyectoID,
		RuntimeID:     handle.RuntimeID,
		HandleID:      &handle.ID,
		Tipo:          "send_instruction",
		PayloadJSON:   payload,
		ResultadoJSON: resultado,
		Estado:        "completada",
		FinishedAt:    &old,
		UpdatedAt:     old,
	})
	if err != nil {
		t.Fatalf("encolar send_instruction previa: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE runtime_orders SET finished_at=?, updated_at=? WHERE id=?`, old, old, orderID); err != nil {
		t.Fatalf("envejecer send_instruction previa: %v", err)
	}

	n, err := procesarRuntimeMailboxSessionResumeBatch()
	if err != nil {
		t.Fatalf("procesar mailbox session resume: %v", err)
	}
	if n != 0 {
		t.Fatalf("no deberia rematerializar guidance caducada en la misma sesion, got=%d", n)
	}

	agente := "Codex1"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	var sendCount int
	for _, order := range orders {
		if order != nil && order.Tipo == "send_instruction" {
			sendCount++
		}
	}
	if sendCount != 1 {
		t.Fatalf("no deberia crear una nueva send_instruction guidance: %d", sendCount)
	}
}

func TestEncolarSendInstructionDesdeRuntimeMailboxCompactaNudgeCodex(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex4", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex4",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	metaJSON := `{"driver":"process_pty_cli","rendered_command":"'/home/alberto/Trabajo/codex-perfiles/bin/codex-perfil' 'Codex4'","working_dir":"` + filepath.Join(tmp, "orquestador") + `","external_session_id":"sess-codex4","can_send_input":false}`
	capsJSON := `{"can_send_input":false,"mailbox_delivery_mode":"` + runtimeagente.MailboxDeliverySessionResume + `"}`
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET capabilities_json=?, metadata_json=?, handle_ref=? WHERE id=?`, capsJSON, metaJSON, strconv.Itoa(os.Getpid()), handle.ID); err != nil {
		t.Fatalf("update handle: %v", err)
	}
	handle, err = db.GetRuntimeHandle(handle.ID)
	if err != nil || handle == nil {
		t.Fatalf("refrescar handle: %+v err=%v", handle, err)
	}

	msgID, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex4",
		ProyectoID:  &proyectoID,
		Kind:        "nudge",
		PayloadJSON: `{"texto":"Se te ha asignado automaticamente la tarea #411. Entra en Orquesta, revisa el contexto vivo y continua hasta cerrarla."}`,
	})
	if err != nil {
		t.Fatalf("mailbox: %v", err)
	}

	orderID, err := encolarSendInstructionDesdeRuntimeMailbox(&db.RuntimeMailboxMessage{
		ID:          msgID,
		FromAgente:  "server",
		ToAgente:    "Codex4",
		ProyectoID:  &proyectoID,
		Kind:        "nudge",
		PayloadJSON: `{"texto":"Se te ha asignado automaticamente la tarea #411. Entra en Orquesta, revisa el contexto vivo y continua hasta cerrarla."}`,
	}, handle, "Se te ha asignado automaticamente la tarea #411. Entra en Orquesta, revisa el contexto vivo y continua hasta cerrarla.", "sess-codex4")
	if err != nil {
		t.Fatalf("encolar send_instruction: %v", err)
	}
	order, err := db.GetRuntimeOrder(orderID)
	if err != nil || order == nil {
		t.Fatalf("get order: %+v err=%v", order, err)
	}
	if !strings.Contains(order.PayloadJSON, `"texto":"toma tarea asignada y sigue"`) {
		t.Fatalf("payload no compactado para Codex: %s", order.PayloadJSON)
	}
}

func TestProcesarRuntimeMailboxSessionResumeBatchCoalesceNudgeAunqueHayaOrdenAbierta(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	metaJSON := `{"driver":"process_pty_cli","rendered_command":"codex-perfil Codex1","working_dir":"` + filepath.Join(tmp, "orquestador") + `","external_session_id":"sess-nudge"}`
	capsJSON := `{"can_send_input":false,"mailbox_delivery_mode":"` + runtimeagente.MailboxDeliverySessionResume + `"}`
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET capabilities_json=?, metadata_json=? WHERE id=?`, capsJSON, metaJSON, handle.ID); err != nil {
		t.Fatalf("update handle: %v", err)
	}

	if _, err := db.EncolarRuntimeOrder(&db.RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		RuntimeID:   handle.RuntimeID,
		HandleID:    &handle.ID,
		Tipo:        "send_instruction",
		PayloadJSON: `{"to_agente":"Codex1","texto":"instruccion abierta"}`,
	}); err != nil {
		t.Fatalf("encolar send_instruction existente: %v", err)
	}

	viejoID, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "nudge",
		PayloadJSON: `{"texto":"nudge viejo"}`,
	})
	if err != nil {
		t.Fatalf("mailbox vieja: %v", err)
	}
	nuevoID, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "nudge",
		PayloadJSON: `{"texto":"nudge nuevo"}`,
	})
	if err != nil {
		t.Fatalf("mailbox nueva: %v", err)
	}

	n, err := procesarRuntimeMailboxSessionResumeBatch()
	if err != nil {
		t.Fatalf("procesar mailbox session resume: %v", err)
	}
	if n != 0 {
		t.Fatalf("no deberia crear send_instruction nueva con orden abierta, got=%d", n)
	}

	agente := "Codex1"
	pendiente := "pendiente"
	mailboxPendiente, err := db.ListarRuntimeMailbox(db.FiltroRuntimeMailbox{ToAgente: &agente, ProyectoID: &proyectoID, Estado: &pendiente})
	if err != nil {
		t.Fatalf("listar mailbox pendiente: %v", err)
	}
	if len(mailboxPendiente) != 1 || mailboxPendiente[0].ID != nuevoID {
		t.Fatalf("la mailbox coalescible deberia conservar solo la ultima: %+v", mailboxPendiente)
	}

	consumido := "consumido"
	mailboxConsumida, err := db.ListarRuntimeMailbox(db.FiltroRuntimeMailbox{ToAgente: &agente, ProyectoID: &proyectoID, Estado: &consumido})
	if err != nil {
		t.Fatalf("listar mailbox consumida: %v", err)
	}
	foundOld := false
	for _, msg := range mailboxConsumida {
		if msg != nil && msg.ID == viejoID {
			foundOld = true
			break
		}
	}
	if !foundOld {
		t.Fatalf("la mailbox vieja deberia quedar consumida tras coalesce: %+v", mailboxConsumida)
	}
}

func TestProcesarRuntimeMailboxBatchConsumeWatchdogSinHandleActivo(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}

	msgID, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "watchdog",
		PayloadJSON: `{"texto":"heartbeat obsoleto"}`,
	})
	if err != nil {
		t.Fatalf("mailbox watchdog: %v", err)
	}

	n, err := procesarRuntimeMailboxBatch()
	if err != nil {
		t.Fatalf("procesar mailbox: %v", err)
	}
	if n != 1 {
		t.Fatalf("deberia reconciliar exactamente un watchdog sin handle, got=%d", n)
	}

	pendiente := "pendiente"
	toAgente := "Codex1"
	pendientes, err := db.ListarRuntimeMailbox(db.FiltroRuntimeMailbox{
		ToAgente:   &toAgente,
		ProyectoID: &proyectoID,
		Estado:     &pendiente,
	})
	if err != nil {
		t.Fatalf("listar mailbox pendiente: %v", err)
	}
	if len(pendientes) != 0 {
		t.Fatalf("no deberia quedar watchdog pendiente sin handle: %+v", pendientes)
	}

	consumido := "consumido"
	consumidos, err := db.ListarRuntimeMailbox(db.FiltroRuntimeMailbox{
		ToAgente:   &toAgente,
		ProyectoID: &proyectoID,
		Estado:     &consumido,
	})
	if err != nil {
		t.Fatalf("listar mailbox consumido: %v", err)
	}
	if len(consumidos) != 1 || consumidos[0].ID != msgID || consumidos[0].Kind != "watchdog" {
		t.Fatalf("watchdog consumido inesperado: %+v", consumidos)
	}

	logs, err := db.ListarAuditoria(db.FiltroAuditoria{Limite: 100})
	if err != nil {
		t.Fatalf("listar auditoria: %v", err)
	}
	found := false
	for _, item := range logs {
		if item != nil && item.Accion == "runtime_mailbox_watchdog_sin_handle" && item.EntidadID == msgID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("faltaba auditoria de watchdog sin handle")
	}
}

func TestProcesarRuntimeMailboxBatchConsumeAgenteSinVida(t *testing.T) {
	prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex6", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	msgID, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex6",
		Kind:        "nudge",
		PayloadJSON: `{"texto":"fuera de flota activa"}`,
	})
	if err != nil {
		t.Fatalf("mailbox: %v", err)
	}

	n, err := procesarRuntimeMailboxBatch()
	if err != nil {
		t.Fatalf("procesar runtime mailbox: %v", err)
	}
	if n != 1 {
		t.Fatalf("deberia reconciliar exactamente un mailbox zombie, got=%d", n)
	}

	agente := "Codex6"
	estado := "pendiente"
	pendientes, err := db.ListarRuntimeMailbox(db.FiltroRuntimeMailbox{ToAgente: &agente, Estado: &estado})
	if err != nil {
		t.Fatalf("listar mailbox pendiente: %v", err)
	}
	if len(pendientes) != 0 {
		t.Fatalf("no deberia quedar mailbox pendiente para agente sin vida: %+v", pendientes)
	}
	estado = "consumido"
	consumidos, err := db.ListarRuntimeMailbox(db.FiltroRuntimeMailbox{ToAgente: &agente, Estado: &estado})
	if err != nil {
		t.Fatalf("listar mailbox consumido: %v", err)
	}
	if len(consumidos) != 1 || consumidos[0].ID != msgID {
		t.Fatalf("mailbox consumido inesperado: %+v", consumidos)
	}
}

func TestProcesarRuntimeMailboxBatchNoConsumeAgenteSinHandlePeroConTrabajo(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex6", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if err := db.ActivarAsignacion("Codex6", proyectoID, "frente activo"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Trabajo real",
		Descripcion: "No debe consumirse el mailbox",
		ProyectoID:  &proyectoID,
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Codex6"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	msgID, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex6",
		ProyectoID:  &proyectoID,
		Kind:        "instruction",
		PayloadJSON: `{"texto":"retoma el trabajo"}`,
	})
	if err != nil {
		t.Fatalf("mailbox: %v", err)
	}

	n, err := procesarRuntimeMailboxBatch()
	if err != nil {
		t.Fatalf("procesar runtime mailbox: %v", err)
	}
	if n != 0 {
		t.Fatalf("no deberia reconciliar mailbox de agente con trabajo real, got=%d", n)
	}

	agente := "Codex6"
	estado := "pendiente"
	pendientes, err := db.ListarRuntimeMailbox(db.FiltroRuntimeMailbox{ToAgente: &agente, Estado: &estado})
	if err != nil {
		t.Fatalf("listar mailbox pendiente: %v", err)
	}
	if len(pendientes) != 1 || pendientes[0].ID != msgID {
		t.Fatalf("mailbox pendiente inesperado: %+v", pendientes)
	}
}

func TestProcesarRuntimeMailboxBatchConsumeAgenteFueraDeFlotaConAsignacionAutomatica(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.ConfigSet("server_autobootstrap_supervisor_agent", "Codex1"); err != nil {
		t.Fatalf("config supervisor: %v", err)
	}
	if err := db.ConfigSet("server_autobootstrap_worker_agents", "Codex2,Codex3,Codex4,Codex5"); err != nil {
		t.Fatalf("config workers: %v", err)
	}
	if err := db.RegistrarAgente("Codex6", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if err := db.ActivarAsignacion("Codex6", proyectoID, "reactivacion_automatica"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	msgID, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex6",
		ProyectoID:  &proyectoID,
		Kind:        "autonomia",
		PayloadJSON: `{"texto":"fuera de flota activa"}`,
	})
	if err != nil {
		t.Fatalf("mailbox: %v", err)
	}

	n, err := procesarRuntimeMailboxBatch()
	if err != nil {
		t.Fatalf("procesar runtime mailbox: %v", err)
	}
	if n != 1 {
		t.Fatalf("deberia reconciliar mailbox de agente fuera de flota, got=%d", n)
	}

	agente := "Codex6"
	estado := "consumido"
	consumidos, err := db.ListarRuntimeMailbox(db.FiltroRuntimeMailbox{ToAgente: &agente, Estado: &estado})
	if err != nil {
		t.Fatalf("listar consumidos: %v", err)
	}
	if len(consumidos) != 1 || consumidos[0].ID != msgID {
		t.Fatalf("mailbox consumido inesperado: %+v", consumidos)
	}
}

func TestProcesarRuntimeMailboxBatchConsumeWatchdogEnEnfriamiento(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET estado='pausado' WHERE id=?`, handle.ID); err != nil {
		t.Fatalf("pausar handle: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE agentes SET estado_cuota='enfriamiento', motivo_pausa='cuota' WHERE nombre='Codex1'`); err != nil {
		t.Fatalf("marcar enfriamiento: %v", err)
	}

	msgID, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "watchdog",
		PayloadJSON: `{"texto":"no deberia despertar al agente en enfriamiento"}`,
	})
	if err != nil {
		t.Fatalf("mailbox watchdog: %v", err)
	}

	n, err := procesarRuntimeMailboxBatch()
	if err != nil {
		t.Fatalf("procesar mailbox: %v", err)
	}
	if n != 1 {
		t.Fatalf("deberia reconciliar exactamente un watchdog en enfriamiento, got=%d", n)
	}

	pendiente := "pendiente"
	toAgente := "Codex1"
	pendientes, err := db.ListarRuntimeMailbox(db.FiltroRuntimeMailbox{
		ToAgente:   &toAgente,
		ProyectoID: &proyectoID,
		Estado:     &pendiente,
	})
	if err != nil {
		t.Fatalf("listar mailbox pendiente: %v", err)
	}
	if len(pendientes) != 0 {
		t.Fatalf("no deberia quedar watchdog pendiente en enfriamiento: %+v", pendientes)
	}

	consumido := "consumido"
	consumidos, err := db.ListarRuntimeMailbox(db.FiltroRuntimeMailbox{
		ToAgente:   &toAgente,
		ProyectoID: &proyectoID,
		Estado:     &consumido,
	})
	if err != nil {
		t.Fatalf("listar mailbox consumido: %v", err)
	}
	if len(consumidos) != 1 || consumidos[0].ID != msgID {
		t.Fatalf("watchdog consumido inesperado: %+v", consumidos)
	}

	logs, err := db.ListarAuditoria(db.FiltroAuditoria{Limite: 20})
	if err != nil {
		t.Fatalf("listar auditoria: %v", err)
	}
	found := false
	for _, item := range logs {
		if item != nil && item.Accion == "runtime_mailbox_watchdog_enfriamiento" && item.EntidadID == msgID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("faltaba auditoria de watchdog en enfriamiento")
	}
}

func TestProcesarRuntimeMailboxBatchConsumeGovernanceRefreshEnEnfriamiento(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET estado='pausado' WHERE id=?`, handle.ID); err != nil {
		t.Fatalf("pausar handle: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE agentes SET estado_cuota='enfriamiento', motivo_pausa='cuota' WHERE nombre='Codex1'`); err != nil {
		t.Fatalf("marcar enfriamiento: %v", err)
	}

	msgID, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        db.MailboxKindGovernanceRefresh,
		PayloadJSON: `{"motivo":"quota_cooldown"}`,
	})
	if err != nil {
		t.Fatalf("mailbox governance_refresh: %v", err)
	}

	n, err := procesarRuntimeMailboxBatch()
	if err != nil {
		t.Fatalf("procesar mailbox: %v", err)
	}
	if n != 1 {
		t.Fatalf("deberia reconciliar exactamente un governance_refresh en enfriamiento, got=%d", n)
	}

	pendiente := "pendiente"
	toAgente := "Codex1"
	pendientes, err := db.ListarRuntimeMailbox(db.FiltroRuntimeMailbox{
		ToAgente:   &toAgente,
		ProyectoID: &proyectoID,
		Estado:     &pendiente,
	})
	if err != nil {
		t.Fatalf("listar mailbox pendiente: %v", err)
	}
	if len(pendientes) != 0 {
		t.Fatalf("no deberia quedar governance_refresh pendiente en enfriamiento: %+v", pendientes)
	}

	consumido := "consumido"
	consumidos, err := db.ListarRuntimeMailbox(db.FiltroRuntimeMailbox{
		ToAgente:   &toAgente,
		ProyectoID: &proyectoID,
		Estado:     &consumido,
	})
	if err != nil {
		t.Fatalf("listar mailbox consumido: %v", err)
	}
	if len(consumidos) != 1 || consumidos[0].ID != msgID {
		t.Fatalf("governance_refresh consumido inesperado: %+v", consumidos)
	}

	logs, err := db.ListarAuditoria(db.FiltroAuditoria{Limite: 20})
	if err != nil {
		t.Fatalf("listar auditoria: %v", err)
	}
	found := false
	for _, item := range logs {
		if item != nil && item.Accion == "runtime_mailbox_refresh_enfriamiento" && item.EntidadID == msgID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("faltaba auditoria de governance_refresh en enfriamiento")
	}
}

func TestProcesarRuntimeMailboxInteractivoBatchNoDuplicaSendInstructionAbierta(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	if handle.RuntimeID == nil {
		t.Fatalf("runtime handle sin runtime asociado: %+v", handle)
	}

	existenteID, err := db.EncolarRuntimeOrder(&db.RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		RuntimeID:   handle.RuntimeID,
		HandleID:    &handle.ID,
		Tipo:        "send_instruction",
		PayloadJSON: `{"to_agente":"Codex1","texto":"instruccion abierta"}`,
	})
	if err != nil {
		t.Fatalf("encolar send_instruction existente: %v", err)
	}
	msgID, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "instruction",
		PayloadJSON: `{"texto":"otra instruccion"}`,
	})
	if err != nil {
		t.Fatalf("mailbox: %v", err)
	}

	n, err := procesarRuntimeMailboxInteractivoBatch()
	if err != nil {
		t.Fatalf("procesar mailbox interactivo: %v", err)
	}
	if n != 0 {
		t.Fatalf("no deberia crear send_instruction nueva, got=%d", n)
	}

	agente := "Codex1"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	var sendInstructions []int64
	for _, order := range orders {
		if order != nil && order.Tipo == "send_instruction" {
			sendInstructions = append(sendInstructions, order.ID)
		}
	}
	if len(sendInstructions) != 1 || sendInstructions[0] != existenteID {
		t.Fatalf("send_instruction inesperadas: %+v", sendInstructions)
	}

	estado := "pendiente"
	mailbox, err := db.ListarRuntimeMailbox(db.FiltroRuntimeMailbox{ToAgente: &agente, Estado: &estado})
	if err != nil {
		t.Fatalf("listar mailbox: %v", err)
	}
	if len(mailbox) != 1 || mailbox[0].ID != msgID {
		t.Fatalf("mailbox pendiente inesperada: %+v", mailbox)
	}
}

func TestProcesarRuntimeMailboxInteractivoBatchCoalesceAutonomiaPendiente(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	if handle.RuntimeID == nil {
		t.Fatalf("runtime handle sin runtime asociado: %+v", handle)
	}
	capsJSON := `{"can_send_input":true,"mailbox_delivery_mode":"` + runtimeagente.MailboxDeliveryInteractive + `"}`
	metaJSON := `{"driver":"process_pty_cli","rendered_command":"cat-cli Codex1","working_dir":"` + filepath.Join(tmp, "orquestador") + `","can_send_input":true,"mailbox_delivery_mode":"` + runtimeagente.MailboxDeliveryInteractive + `"}`
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET capabilities_json=?, metadata_json=? WHERE id=?`, capsJSON, metaJSON, handle.ID); err != nil {
		t.Fatalf("update handle interactivo: %v", err)
	}

	msgViejo, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "autonomia",
		PayloadJSON: `{"accion":"continuar_trabajo","motivo":"mensaje viejo"}`,
	})
	if err != nil {
		t.Fatalf("mailbox vieja: %v", err)
	}
	msgNuevo, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "autonomia",
		PayloadJSON: `{"accion":"continuar_trabajo","motivo":"mensaje nuevo"}`,
	})
	if err != nil {
		t.Fatalf("mailbox nueva: %v", err)
	}

	n, err := procesarRuntimeMailboxInteractivoBatch()
	if err != nil {
		t.Fatalf("procesar mailbox interactivo: %v", err)
	}
	if n != 1 {
		t.Fatalf("deberia crear una sola send_instruction, got=%d", n)
	}

	agente := "Codex1"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	var send *db.RuntimeOrder
	for _, order := range orders {
		if order != nil && order.Tipo == "send_instruction" {
			send = order
			break
		}
	}
	if send == nil {
		t.Fatalf("faltaba send_instruction")
	}
	if !strings.Contains(send.PayloadJSON, "mensaje nuevo") {
		t.Fatalf("deberia usar la autonomia mas reciente: %s", send.PayloadJSON)
	}

	pendiente := "pendiente"
	consumido := "consumido"
	mailboxPendiente, err := db.ListarRuntimeMailbox(db.FiltroRuntimeMailbox{ToAgente: &agente, Estado: &pendiente})
	if err != nil {
		t.Fatalf("listar mailbox pendiente: %v", err)
	}
	if len(mailboxPendiente) != 1 || mailboxPendiente[0].ID != msgNuevo {
		t.Fatalf("deberia quedar pendiente la autonomia vigente hasta entrega real: %+v", mailboxPendiente)
	}
	mailboxConsumido, err := db.ListarRuntimeMailbox(db.FiltroRuntimeMailbox{ToAgente: &agente, Estado: &consumido})
	if err != nil {
		t.Fatalf("listar mailbox consumido: %v", err)
	}
	if len(mailboxConsumido) != 1 {
		t.Fatalf("solo deberia quedar consumida la autonomia supersedida: %+v", mailboxConsumido)
	}
	idsConsumidos := map[int64]struct{}{}
	for _, msg := range mailboxConsumido {
		if msg != nil {
			idsConsumidos[msg.ID] = struct{}{}
		}
	}
	if _, ok := idsConsumidos[msgViejo]; !ok {
		t.Fatalf("faltaba autonomia vieja consumida")
	}
	if _, ok := idsConsumidos[msgNuevo]; ok {
		t.Fatalf("la autonomia nueva no deberia marcarse consumida antes de la entrega real")
	}
}

func TestProcesarRuntimeMailboxInteractivoBatchNoRematerializaMailboxOnlyEnMismoHandle(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "cat-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	if handle.RuntimeID == nil {
		t.Fatalf("runtime handle sin runtime asociado: %+v", handle)
	}
	capsJSON := `{"can_send_input":true,"mailbox_delivery_mode":"` + runtimeagente.MailboxDeliveryInteractive + `"}`
	metaJSON := `{"driver":"process_pty_cli","rendered_command":"cat-cli Codex1","working_dir":"` + filepath.Join(tmp, "orquestador") + `","can_send_input":true}`
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET capabilities_json=?, metadata_json=? WHERE id=?`, capsJSON, metaJSON, handle.ID); err != nil {
		t.Fatalf("update handle: %v", err)
	}

	msgID, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "instruction",
		PayloadJSON: `{"texto":"instruccion interactiva durable"}`,
	})
	if err != nil {
		t.Fatalf("mailbox: %v", err)
	}

	payload := fmt.Sprintf(`{"to_agente":"Codex1","from_agente":"server","texto":"instruccion interactiva durable","mailbox_id":%d,"mailbox_kind":"instruction","delivery_attempt_signature":"interactive|handle:%d|session:"}`, msgID, handle.ID)
	resultado := fmt.Sprintf(`{"ok":true,"mailbox_id":%d,"deferred":true,"deferred_reason":"runtime no disponible para entrega inmediata","mailbox_only":true}`, msgID)
	if _, err := db.EncolarRuntimeOrder(&db.RuntimeOrder{
		Agente:        "Codex1",
		ProyectoID:    &proyectoID,
		RuntimeID:     handle.RuntimeID,
		HandleID:      &handle.ID,
		Tipo:          "send_instruction",
		PayloadJSON:   payload,
		ResultadoJSON: resultado,
		Estado:        "completada",
	}); err != nil {
		t.Fatalf("encolar send_instruction previa: %v", err)
	}

	n, err := procesarRuntimeMailboxInteractivoBatch()
	if err != nil {
		t.Fatalf("procesar mailbox interactivo: %v", err)
	}
	if n != 0 {
		t.Fatalf("no deberia rematerializar send_instruction mailbox_only en el mismo handle, got=%d", n)
	}

	agente := "Codex1"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	var sendCount int
	for _, order := range orders {
		if order != nil && order.Tipo == "send_instruction" {
			sendCount++
		}
	}
	if sendCount != 1 {
		t.Fatalf("no deberia crear una nueva send_instruction: %d", sendCount)
	}
}

func TestProcesarRuntimeMailboxInteractivoBatchCoalesceInstructionPendiente(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	if handle.RuntimeID == nil {
		t.Fatalf("runtime handle sin runtime asociado: %+v", handle)
	}
	capsJSON := `{"can_send_input":true,"mailbox_delivery_mode":"` + runtimeagente.MailboxDeliveryInteractive + `"}`
	metaJSON := `{"driver":"process_pty_cli","rendered_command":"cat-cli Codex1","working_dir":"` + filepath.Join(tmp, "orquestador") + `","can_send_input":true,"mailbox_delivery_mode":"` + runtimeagente.MailboxDeliveryInteractive + `"}`
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET capabilities_json=?, metadata_json=? WHERE id=?`, capsJSON, metaJSON, handle.ID); err != nil {
		t.Fatalf("update handle interactivo: %v", err)
	}

	msgViejo, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "instruction",
		PayloadJSON: `{"texto":"instruccion vieja"}`,
	})
	if err != nil {
		t.Fatalf("mailbox vieja: %v", err)
	}
	msgNuevo, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "instruction",
		PayloadJSON: `{"texto":"instruccion nueva"}`,
	})
	if err != nil {
		t.Fatalf("mailbox nueva: %v", err)
	}

	n, err := procesarRuntimeMailboxInteractivoBatch()
	if err != nil {
		t.Fatalf("procesar mailbox interactivo: %v", err)
	}
	if n != 1 {
		t.Fatalf("deberia crear una sola send_instruction, got=%d", n)
	}

	agente := "Codex1"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	var send *db.RuntimeOrder
	for _, order := range orders {
		if order != nil && order.Tipo == "send_instruction" {
			send = order
			break
		}
	}
	if send == nil {
		t.Fatalf("faltaba send_instruction")
	}
	if !strings.Contains(send.PayloadJSON, "instruccion nueva") {
		t.Fatalf("deberia usar la instruction mas reciente: %s", send.PayloadJSON)
	}

	pendiente := "pendiente"
	consumido := "consumido"
	mailboxPendiente, err := db.ListarRuntimeMailbox(db.FiltroRuntimeMailbox{ToAgente: &agente, Estado: &pendiente})
	if err != nil {
		t.Fatalf("listar mailbox pendiente: %v", err)
	}
	if len(mailboxPendiente) != 1 || mailboxPendiente[0].ID != msgNuevo {
		t.Fatalf("deberia quedar pendiente la instruction vigente hasta entrega real: %+v", mailboxPendiente)
	}
	mailboxConsumido, err := db.ListarRuntimeMailbox(db.FiltroRuntimeMailbox{ToAgente: &agente, Estado: &consumido})
	if err != nil {
		t.Fatalf("listar mailbox consumido: %v", err)
	}
	idsConsumidos := map[int64]struct{}{}
	for _, msg := range mailboxConsumido {
		if msg != nil {
			idsConsumidos[msg.ID] = struct{}{}
		}
	}
	if _, ok := idsConsumidos[msgViejo]; !ok {
		t.Fatalf("faltaba instruction vieja consumida")
	}
	if _, ok := idsConsumidos[msgNuevo]; ok {
		t.Fatalf("la instruction nueva no deberia marcarse consumida antes de la entrega real")
	}
}

func TestEncolarNudgeAutonomiaDetalladoNoDuplicaMailboxPendienteEnHandleNoInteractivo(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	if _, err := db.DB.Exec(
		`UPDATE runtime_handles SET capabilities_json=?, metadata_json=? WHERE id=?`,
		`{"can_send_input":false}`,
		`{"driver":"process_pty_cli","rendered_command":"codex-perfil Codex1","can_send_input":false}`,
		handle.ID,
	); err != nil {
		t.Fatalf("update handle: %v", err)
	}

	runtimeOrderID := int64(91)
	if _, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:     "server",
		ToAgente:       "Codex1",
		ProyectoID:     &proyectoID,
		RuntimeOrderID: &runtimeOrderID,
		Kind:           "autonomia",
		PayloadJSON:    `{"accion":"continuar_trabajo","motivo":"seguir frente","texto":"seguir"}`,
	}); err != nil {
		t.Fatalf("crear mailbox existente: %v", err)
	}

	proyecto, err := db.GetProyecto("orquestador")
	if err != nil {
		t.Fatalf("get proyecto: %v", err)
	}
	encolada, err := encolarNudgeAutonomiaDetallado("Codex1", proyecto, "continuar_trabajo", "seguir frente", "sigue trabajando", nil)
	if err != nil {
		t.Fatalf("encolar nudge autonomia: %v", err)
	}
	if encolada {
		t.Fatal("no deberia encolar un nuevo nudge cuando ya existe mailbox pendiente equivalente")
	}

	agente := "Codex1"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	if len(orders) != 0 {
		t.Fatalf("no deberia crear orden adicional: %+v", orders)
	}

	estado := "pendiente"
	mailbox, err := db.ListarRuntimeMailbox(db.FiltroRuntimeMailbox{ToAgente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar mailbox: %v", err)
	}
	if len(mailbox) != 1 {
		t.Fatalf("mailbox inesperado tras dedupe: %+v", mailbox)
	}
}

func TestResetReanimacionEncolaStartCuandoNoHayRuntimePeroSiTrabajo(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if err := db.ActivarAsignacion("Codex1", proyectoID, "frente principal"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Retomar sin runtime",
		Descripcion: "Trabajo asignado",
		ProyectoID:  &proyectoID,
		Modulo:      "core",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE agentes SET reanimar_at=CURRENT_TIMESTAMP, estado_cuota='enfriamiento', motivo_pausa='cuota' WHERE nombre='Codex1'`); err != nil {
		t.Fatalf("marcar reanimacion: %v", err)
	}

	if err := (dbAutomationService{}).ResetReanimacion("Codex1"); err != nil {
		t.Fatalf("reset reanimacion: %v", err)
	}

	agente := "Codex1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	if len(orders) != 1 || orders[0].Tipo != "start" {
		t.Fatalf("start no encolado: %+v", orders)
	}
	if !strings.Contains(orders[0].PayloadJSON, `"accion":"start"`) {
		t.Fatalf("payload start inesperado: %s", orders[0].PayloadJSON)
	}
}

func TestResetReanimacionConservaCooldownSiFallaReactivacion(t *testing.T) {
	prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	if _, err := db.DB.Exec(`INSERT INTO asignaciones (agente, proyecto_id, estado, nota) VALUES ('Codex1', 999999, 'activa', 'forzar error de proyecto')`); err != nil {
		t.Fatalf("insert asignacion inconsistente: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE agentes SET reanimar_at=CURRENT_TIMESTAMP, estado_cuota='enfriamiento', motivo_pausa='cuota' WHERE nombre='Codex1'`); err != nil {
		t.Fatalf("marcar reanimacion: %v", err)
	}

	if err := (dbAutomationService{}).ResetReanimacion("Codex1"); err == nil {
		t.Fatalf("esperaba error al reactivar con proyecto inconsistente")
	}

	agente, err := db.GetAgente("Codex1")
	if err != nil {
		t.Fatalf("get agente: %v", err)
	}
	if agente == nil {
		t.Fatalf("agente nil")
	}
	if !strings.EqualFold(strings.TrimSpace(agente.EstadoCuota), "enfriamiento") {
		t.Fatalf("estado_cuota inesperado: %+v", agente)
	}
	if agente.ReanimarAt == nil {
		t.Fatalf("reanimar_at no deberia limpiarse si falla la reactivacion: %+v", agente)
	}
	if strings.TrimSpace(agente.MotivoPausa) == "" {
		t.Fatalf("motivo_pausa no deberia vaciarse si falla la reactivacion: %+v", agente)
	}
}

func TestResetReanimacionSostieneCooldownSiLaCuotaVisibleSigueAgotada(t *testing.T) {
	prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	sesionID, err := db.IniciarSesion("Codex1")
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	now := time.Now().UTC()
	resetPrimary := now.Add(30 * time.Minute)
	resetSecondary := now.Add(72 * time.Hour)
	raw := `{"rate_limits":{"primary":{"used_percent":20,"window_minutes":300,"resets_at":` + strconv.FormatInt(resetPrimary.Unix(), 10) + `},"secondary":{"used_percent":100,"window_minutes":10080,"resets_at":` + strconv.FormatInt(resetSecondary.Unix(), 10) + `}}}`
	if _, err := db.RegistrarPresupuestoSesion(&db.PresupuestoSesion{
		SesionID:        sesionID,
		WindowKind:      "5h",
		BudgetSource:    "codex_token_count_observed",
		RawSnapshotJSON: raw,
		CheckedAt:       now,
	}); err != nil {
		t.Fatalf("registrar presupuesto: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE agentes SET reanimar_at=CURRENT_TIMESTAMP, estado_cuota='enfriamiento', motivo_pausa='cuota' WHERE nombre='Codex1'`); err != nil {
		t.Fatalf("marcar reanimacion: %v", err)
	}

	if err := (dbAutomationService{}).ResetReanimacion("Codex1"); err != nil {
		t.Fatalf("reset reanimacion: %v", err)
	}

	agente, err := db.GetAgente("Codex1")
	if err != nil {
		t.Fatalf("get agente: %v", err)
	}
	if agente == nil {
		t.Fatalf("agente nil")
	}
	if strings.TrimSpace(agente.EstadoCuota) != "enfriamiento" {
		t.Fatalf("estado_cuota inesperado: %+v", agente)
	}
	if agente.ReanimarAt == nil || agente.ReanimarAt.Before(resetSecondary.Add(-time.Minute)) {
		t.Fatalf("deberia sostener cooldown hasta el reset visible semanal: %+v want>=%s", agente, resetSecondary.Format(time.RFC3339))
	}
	estado := "pendiente"
	nombre := "Codex1"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &nombre, Estado: &estado})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	if len(orders) != 0 {
		t.Fatalf("no deberia reactivar ni encolar ordenes mientras la cuota siga agotada: %+v", orders)
	}
}

func TestProcesarAutonomiaAgentesBatchEncolaPausePorBloqueoHumano(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if err := db.ActivarAsignacion("Codex1", proyectoID, "frente principal"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Esperando respuesta humana",
		Descripcion: "Bloqueo operativo",
		ProyectoID:  &proyectoID,
		Modulo:      "core",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	if err := db.BloquearTarea(tareaID, "Codex1", "esperando respuesta humana"); err != nil {
		t.Fatalf("bloquear tarea: %v", err)
	}

	n, err := procesarAutonomiaAgentesBatch()
	if err != nil {
		t.Fatalf("procesar autonomia: %v", err)
	}
	if n != 1 {
		t.Fatalf("esperaba 1 decision autonoma, got=%d", n)
	}

	agente := "Codex1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	if len(orders) != 1 || orders[0].Tipo != "pause" {
		t.Fatalf("pause no encolada: %+v", orders)
	}
	if !strings.Contains(orders[0].PayloadJSON, "bloqueo_humano:esperando respuesta humana") {
		t.Fatalf("payload pause inesperado: %s", orders[0].PayloadJSON)
	}
	op, err := db.GetProyectoOperacion(proyectoID)
	if err != nil {
		t.Fatalf("get proyecto operacion: %v", err)
	}
	if op.EstadoOperativo != db.ProyectoOperativoEsperandoHumano {
		t.Fatalf("estado operativo inesperado: %+v", op)
	}
}

func TestProcesarAutonomiaAgentesBatchNoDuplicaPauseSiYaEstaPausadoPorCuota(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if err := db.ActivarAsignacion("Codex1", proyectoID, "frente principal"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Agente en enfriamiento",
		Descripcion: "No debe crear pause duplicada",
		ProyectoID:  &proyectoID,
		Modulo:      "core",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET estado='pausado' WHERE id=?`, handle.ID); err != nil {
		t.Fatalf("pausar handle: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE agentes SET reanimar_at=CURRENT_TIMESTAMP, estado_cuota='enfriamiento', motivo_pausa='cuota' WHERE nombre='Codex1'`); err != nil {
		t.Fatalf("marcar enfriamiento: %v", err)
	}

	n, err := procesarAutonomiaAgentesBatch()
	if err != nil {
		t.Fatalf("procesar autonomia: %v", err)
	}
	if n != 0 {
		t.Fatalf("no deberia crear decision nueva, got=%d", n)
	}

	agente := "Codex1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil && err != sql.ErrNoRows {
		t.Fatalf("listar orders: %v", err)
	}
	if len(orders) != 0 {
		t.Fatalf("no deberia crear pause duplicada: %+v", orders)
	}
}

func TestProcesarAutonomiaAgentesBatchAparcaSesionBloqueadaYPausaAsignacion(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if err := db.ActivarAsignacion("Codex1", proyectoID, "frente principal"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Consulta a humano",
		Descripcion: "Esperando desbloqueo",
		ProyectoID:  &proyectoID,
		Modulo:      "core",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	if err := db.BloquearTarea(tareaID, "Codex1", "esperando decision humana"); err != nil {
		t.Fatalf("bloquear tarea: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE sesiones SET estado='pausada' WHERE id=?`, sesion.ID); err != nil {
		t.Fatalf("pausar sesion: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET estado='pausado' WHERE id=?`, handle.ID); err != nil {
		t.Fatalf("pausar handle: %v", err)
	}

	n, err := procesarAutonomiaAgentesBatch()
	if err != nil {
		t.Fatalf("procesar autonomia: %v", err)
	}
	if n != 1 {
		t.Fatalf("esperaba 1 decision autonoma, got=%d", n)
	}

	sesionRecargada, err := db.GetSesionByID(sesion.ID)
	if err != nil {
		t.Fatalf("get sesion: %v", err)
	}
	if sesionRecargada.Activa {
		t.Fatalf("la sesion deberia quedar aparcada: %+v", sesionRecargada)
	}
	agenteRecargado, err := db.GetAgente("Codex1")
	if err != nil {
		t.Fatalf("get agente: %v", err)
	}
	if agenteRecargado.Activo || agenteRecargado.EstadoSesion != "" {
		t.Fatalf("agente no liberado tras aparcado: %+v", agenteRecargado)
	}
	handles, err := db.ListarRuntimeHandles(&agenteRecargado.Nombre)
	if err != nil {
		t.Fatalf("listar handles: %v", err)
	}
	if len(handles) == 0 || handles[0].Estado != "cerrado" {
		t.Fatalf("handle no cerrado tras aparcado: %+v", handles)
	}
	agente := "Codex1"
	asignaciones, err := db.ListarAsignaciones(db.FiltroAsignaciones{Agente: &agente})
	if err != nil {
		t.Fatalf("listar asignaciones: %v", err)
	}
	if len(asignaciones) == 0 || asignaciones[0].Estado != db.AsignacionPausada {
		t.Fatalf("asignacion no pausada: %+v", asignaciones)
	}
	op, err := db.GetProyectoOperacion(proyectoID)
	if err != nil {
		t.Fatalf("get proyecto operacion: %v", err)
	}
	if op.EstadoOperativo != db.ProyectoOperativoEsperandoHumano {
		t.Fatalf("estado operativo inesperado tras aparcado: %+v", op)
	}
}

func TestProcesarAutonomiaAgentesBatchRespetaEstadoOperativoProyectoEsperandoHumano(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if err := db.ActivarAsignacion("Codex1", proyectoID, "frente principal"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	if err := db.MarcarProyectoEsperandoHumano(proyectoID, "esperando decision externa"); err != nil {
		t.Fatalf("marcar proyecto esperando humano: %v", err)
	}

	n, err := procesarAutonomiaAgentesBatch()
	if err != nil {
		t.Fatalf("procesar autonomia: %v", err)
	}
	if n != 1 {
		t.Fatalf("esperaba 1 decision autonoma, got=%d", n)
	}

	agente := "Codex1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	if len(orders) != 1 || orders[0].Tipo != "pause" {
		t.Fatalf("pause no encolada por estado operativo del proyecto: %+v", orders)
	}
	if !strings.Contains(orders[0].PayloadJSON, "bloqueo_humano:esperando decision externa") {
		t.Fatalf("payload pause inesperado: %s", orders[0].PayloadJSON)
	}
}

func TestProcesarAutonomiaAgentesBatchCierraProyectoTerminado(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := db.UpsertProyectoAutonomia(&db.ProyectoAutonomia{
		ProyectoID:       proyectoID,
		Enabled:          true,
		ObjetivoGeneral:  "Terminar la app",
		AutoCloseProject: true,
		EstadoAutonomia:  db.AutonomiaProyectoActiva,
	}); err != nil {
		t.Fatalf("upsert proyecto autonomia: %v", err)
	}
	if err := db.ActivarAsignacion("Codex1", proyectoID, "frente principal"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Entrega final",
		Descripcion: "Proyecto listo",
		ProyectoID:  &proyectoID,
		Modulo:      "core",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	if err := db.CompletarTarea(tareaID, "Codex1", "entrega completada"); err != nil {
		t.Fatalf("completar tarea: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}

	n, err := procesarAutonomiaAgentesBatch()
	if err != nil {
		t.Fatalf("procesar autonomia: %v", err)
	}
	if n != 1 {
		t.Fatalf("esperaba 1 decision autonoma, got=%d", n)
	}

	op, err := db.GetProyectoOperacion(proyectoID)
	if err != nil {
		t.Fatalf("get proyecto operacion: %v", err)
	}
	if op.EstadoOperativo != db.ProyectoOperativoCerrado {
		t.Fatalf("proyecto no marcado cerrado: %+v", op)
	}
	agente := "Codex1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	if len(orders) != 1 || orders[0].Tipo != "pause" || !strings.Contains(orders[0].PayloadJSON, "proyecto_terminado:") {
		t.Fatalf("pause de cierre inesperada: %+v", orders)
	}
}

func TestProcesarAutonomiaAgentesBatchRecuperaRuntimeRemotoDegradado(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if err := db.ActivarAsignacion("Codex1", proyectoID, "frente principal"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	conectorID, err := db.UpsertConector(&db.Conector{
		Slug:       "codex-remote",
		Nombre:     "Codex Remote",
		Transporte: "api",
		Comando:    "http://localhost:9999",
		Activo:     true,
	})
	if err != nil {
		t.Fatalf("upsert conector: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Recuperar runtime remoto",
		Descripcion: "Trabajo activo",
		ProyectoID:  &proyectoID,
		Modulo:      "core",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:             "Codex1",
		ConectorID:         &conectorID,
		ProyectoID:         &proyectoID,
		CWD:                filepath.Join(tmp, "orquestador"),
		Herramienta:        "codex-remote",
		ExternalSessionID:  "sess-remote-degraded",
		ResumenContinuidad: "continuidad activa",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("get handle: %+v err=%v", handle, err)
	}
	if _, err := db.DB.Exec(`
		UPDATE runtime_handles
		SET transporte='api',
		    estado='fallido',
		    metadata_json='{\"remote_sync_failures\":3,\"remote_last_error\":\"adapter down\"}'
		WHERE id = ?`, handle.ID); err != nil {
		t.Fatalf("degradar handle: %v", err)
	}
	runtime, err := db.GetRuntimeBySesionID(sesion.ID)
	if err != nil || runtime == nil {
		t.Fatalf("get runtime: %+v err=%v", runtime, err)
	}
	if _, err := db.DB.Exec(`
		UPDATE runtime_instances
		SET logical_state='degradado', process_state='remote_status_error'
		WHERE id = ?`, runtime.ID); err != nil {
		t.Fatalf("degradar runtime: %v", err)
	}

	sesionActual, err := db.GetSesionByID(sesion.ID)
	if err != nil || sesionActual == nil {
		t.Fatalf("get sesion actual: %+v err=%v", sesionActual, err)
	}
	n, err := procesarRecuperacionRuntimeDegradadoSesion(sesionActual)
	if err != nil {
		t.Fatalf("procesar recuperacion degradada: %v", err)
	}
	if n != 2 {
		t.Fatalf("esperaba 2 decisiones autonomas (checkpoint + resume), got=%d", n)
	}

	agente := "Codex1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	if len(orders) != 2 {
		t.Fatalf("orders autonomas inesperadas: %+v", orders)
	}
	var checkpointFound, resumeFound bool
	for _, order := range orders {
		if order == nil {
			continue
		}
		switch order.Tipo {
		case "checkpoint":
			checkpointFound = strings.Contains(order.PayloadJSON, "remote_runtime_degraded")
		case "resume":
			resumeFound = strings.Contains(order.PayloadJSON, `"motivo":"remote_runtime_degraded"`)
		}
	}
	if !checkpointFound || !resumeFound {
		t.Fatalf("recuperacion autonoma remota incompleta: %+v", orders)
	}
}

func TestProcesarAutonomiaAgentesBatchRecuperaRuntimeRemotoDegradadoSinSesionExternaRelanzaStart(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if err := db.ActivarAsignacion("Codex1", proyectoID, "frente principal"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	conectorID, err := db.UpsertConector(&db.Conector{
		Slug:       "codex-remote",
		Nombre:     "Codex Remote",
		Transporte: "api",
		Comando:    "http://localhost:9999",
		Activo:     true,
	})
	if err != nil {
		t.Fatalf("upsert conector: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:             "Codex1",
		ConectorID:         &conectorID,
		ProyectoID:         &proyectoID,
		CWD:                filepath.Join(tmp, "orquestador"),
		Herramienta:        "codex-remote",
		ResumenContinuidad: "continuidad activa",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("get handle: %+v err=%v", handle, err)
	}
	if _, err := db.DB.Exec(`
		UPDATE runtime_handles
		SET transporte='api',
		    handle_kind='session',
		    handle_ref=?,
		    estado='fallido',
		    metadata_json='{"remote_sync_failures":3,"remote_last_error":"adapter down"}'
		WHERE id = ?`, strconv.FormatInt(sesion.ID, 10), handle.ID); err != nil {
		t.Fatalf("degradar handle: %v", err)
	}
	runtime, err := db.GetRuntimeBySesionID(sesion.ID)
	if err != nil || runtime == nil {
		t.Fatalf("get runtime: %+v err=%v", runtime, err)
	}
	if _, err := db.DB.Exec(`
		UPDATE runtime_instances
		SET logical_state='degradado', process_state='remote_status_error'
		WHERE id = ?`, runtime.ID); err != nil {
		t.Fatalf("degradar runtime: %v", err)
	}

	sesionActual, err := db.GetSesionByID(sesion.ID)
	if err != nil || sesionActual == nil {
		t.Fatalf("get sesion actual: %+v err=%v", sesionActual, err)
	}
	n, err := procesarRecuperacionRuntimeDegradadoSesion(sesionActual)
	if err != nil {
		t.Fatalf("procesar recuperacion degradada: %v", err)
	}
	if n != 2 {
		t.Fatalf("esperaba 2 decisiones autonomas (checkpoint + start), got=%d", n)
	}

	agente := "Codex1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	if len(orders) != 2 {
		t.Fatalf("orders autonomas inesperadas: %+v", orders)
	}
	var checkpointFound, startFound bool
	for _, order := range orders {
		if order == nil {
			continue
		}
		switch order.Tipo {
		case "checkpoint":
			checkpointFound = strings.Contains(order.PayloadJSON, "remote_runtime_degraded")
		case "start":
			startFound = strings.Contains(order.PayloadJSON, `"motivo":"remote_runtime_degraded"`)
		}
	}
	if !checkpointFound || !startFound {
		t.Fatalf("recuperacion autonoma remota por start incompleta: %+v", orders)
	}
}

func TestProcesarAutonomiaAgentesBatchRecuperaRuntimeLocalFallidoRelanzaStart(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if err := db.ActivarAsignacion("Codex1", proyectoID, "frente principal"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Recuperar runtime local",
		Descripcion: "Hay trabajo real pendiente",
		ProyectoID:  &proyectoID,
		Modulo:      "controlplane",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	conector, err := db.GetConector("codex-cli")
	if err != nil || conector == nil {
		t.Fatalf("get conector codex-cli: %+v err=%v", conector, err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:             "Codex1",
		ConectorID:         &conector.ID,
		ProyectoID:         &proyectoID,
		CWD:                filepath.Join(tmp, "orquestador"),
		Herramienta:        "codex-cli",
		ResumenContinuidad: "continuidad activa",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("get handle: %+v err=%v", handle, err)
	}
	if _, err := db.DB.Exec(`
		UPDATE runtime_handles
		SET transporte='cli',
		    handle_kind='process',
		    handle_ref='999999',
		    estado='fallido'
		WHERE id = ?`, handle.ID); err != nil {
		t.Fatalf("degradar handle local: %v", err)
	}
	runtime, err := db.GetRuntimeBySesionID(sesion.ID)
	if err != nil || runtime == nil {
		t.Fatalf("get runtime: %+v err=%v", runtime, err)
	}
	if _, err := db.DB.Exec(`
		UPDATE runtime_instances
		SET logical_state='fallido', process_state='fallido'
		WHERE id = ?`, runtime.ID); err != nil {
		t.Fatalf("degradar runtime local: %v", err)
	}

	sesionActual, err := db.GetSesionByID(sesion.ID)
	if err != nil || sesionActual == nil {
		t.Fatalf("get sesion actual: %+v err=%v", sesionActual, err)
	}
	n, err := procesarRecuperacionRuntimeDegradadoSesion(sesionActual)
	if err != nil {
		t.Fatalf("procesar recuperacion local fallida: %v", err)
	}
	if n != 1 {
		t.Fatalf("esperaba 1 decision autonoma (start), got=%d", n)
	}

	agente := "Codex1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	if len(orders) != 1 || orders[0].Tipo != "start" {
		t.Fatalf("recuperacion autonoma local inesperada: %+v", orders)
	}
	if !strings.Contains(orders[0].PayloadJSON, `"motivo":"local_runtime_failed"`) {
		t.Fatalf("start de recuperacion local sin motivo esperado: %s", orders[0].PayloadJSON)
	}
}

func TestProcesarAutonomiaAgentesBatchNoRelanzaRuntimeLocalFallidoSinTrabajoArrancable(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if err := db.ActivarAsignacion("Codex1", proyectoID, "frente principal"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	conector, err := db.GetConector("codex-cli")
	if err != nil || conector == nil {
		t.Fatalf("get conector codex-cli: %+v err=%v", conector, err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:             "Codex1",
		ConectorID:         &conector.ID,
		ProyectoID:         &proyectoID,
		CWD:                filepath.Join(tmp, "orquestador"),
		Herramienta:        "codex-cli",
		ResumenContinuidad: "continuidad activa",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("get handle: %+v err=%v", handle, err)
	}
	if _, err := db.DB.Exec(`
		UPDATE runtime_handles
		SET transporte='cli',
		    handle_kind='process',
		    handle_ref='999999',
		    estado='fallido'
		WHERE id = ?`, handle.ID); err != nil {
		t.Fatalf("degradar handle local: %v", err)
	}
	runtime, err := db.GetRuntimeBySesionID(sesion.ID)
	if err != nil || runtime == nil {
		t.Fatalf("get runtime: %+v err=%v", runtime, err)
	}
	if _, err := db.DB.Exec(`
		UPDATE runtime_instances
		SET logical_state='fallido', process_state='fallido'
		WHERE id = ?`, runtime.ID); err != nil {
		t.Fatalf("degradar runtime local: %v", err)
	}

	sesionActual, err := db.GetSesionByID(sesion.ID)
	if err != nil || sesionActual == nil {
		t.Fatalf("get sesion actual: %+v err=%v", sesionActual, err)
	}
	n, err := procesarRecuperacionRuntimeDegradadoSesion(sesionActual)
	if err != nil {
		t.Fatalf("procesar recuperacion local fallida sin trabajo: %v", err)
	}
	if n != 0 {
		t.Fatalf("sin trabajo arrancable no deberia relanzar start, got=%d", n)
	}

	agente := "Codex1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	if len(orders) != 0 {
		t.Fatalf("sin trabajo arrancable no deberia encolar ordenes: %+v", orders)
	}
}

func TestProcesarAutonomiaAgentesBatchNoRelanzaRuntimeLocalSiSigueVivo(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if err := db.ActivarAsignacion("Codex1", proyectoID, "frente principal"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Runtime local vivo",
		Descripcion: "No debe relanzarse si el proceso observado sigue vivo",
		ProyectoID:  &proyectoID,
		Modulo:      "controlplane",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	conector, err := db.GetConector("codex-cli")
	if err != nil || conector == nil {
		t.Fatalf("get conector codex-cli: %+v err=%v", conector, err)
	}
	pid := int64(os.Getpid())
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:             "Codex1",
		ConectorID:         &conector.ID,
		ProyectoID:         &proyectoID,
		CWD:                filepath.Join(tmp, "orquestador"),
		Herramienta:        "codex-cli",
		PID:                &pid,
		ResumenContinuidad: "continuidad activa",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("get handle: %+v err=%v", handle, err)
	}
	runtime, err := db.GetRuntimeBySesionID(sesion.ID)
	if err != nil || runtime == nil {
		t.Fatalf("get runtime: %+v err=%v", runtime, err)
	}
	if _, err := db.DB.Exec(`
		UPDATE runtime_handles
		SET transporte='cli',
		    handle_kind='process',
		    handle_ref=?,
		    estado='fallido'
		WHERE id = ?`, strconv.FormatInt(pid, 10), handle.ID); err != nil {
		t.Fatalf("degradar handle local vivo: %v", err)
	}
	if _, err := db.DB.Exec(`
		UPDATE runtime_instances
		SET pid=?, logical_state='fallido', process_state='fallido'
		WHERE id = ?`, pid, runtime.ID); err != nil {
		t.Fatalf("degradar runtime local vivo: %v", err)
	}

	sesionActual, err := db.GetSesionByID(sesion.ID)
	if err != nil || sesionActual == nil {
		t.Fatalf("get sesion actual: %+v err=%v", sesionActual, err)
	}
	n, err := procesarRecuperacionRuntimeDegradadoSesion(sesionActual)
	if err != nil {
		t.Fatalf("procesar recuperacion local viva: %v", err)
	}
	if n != 0 {
		t.Fatalf("no deberia relanzar start si el proceso local sigue vivo, got=%d", n)
	}

	handle, err = db.GetRuntimeHandle(handle.ID)
	if err != nil || handle == nil {
		t.Fatalf("get handle tras recuperacion: %+v err=%v", handle, err)
	}
	if handle.Estado != "activo" {
		t.Fatalf("el handle vivo deberia revivir a activo: %+v", handle)
	}
	runtime, err = db.GetRuntime(runtime.ID)
	if err != nil || runtime == nil {
		t.Fatalf("get runtime tras recuperacion: %+v err=%v", runtime, err)
	}
	if strings.EqualFold(strings.TrimSpace(runtime.ProcessState), "fallido") {
		t.Fatalf("el runtime vivo no deberia seguir fallido: %+v", runtime)
	}

	agente := "Codex1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	if len(orders) != 0 {
		t.Fatalf("no deberia encolar ordenes al revivir handle vivo: %+v", orders)
	}
}

func TestProcesarAutonomiaAgentesBatchBloqueaProyectoPorCircuitoAbiertoConector(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if err := db.ActivarAsignacion("Codex1", proyectoID, "frente principal"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	if _, err := db.UpsertConector(&db.Conector{
		Slug:       "codex-remote",
		Nombre:     "Codex Remote",
		Transporte: "api",
		Comando:    "http://localhost:9999",
		Activo:     true,
	}); err != nil {
		t.Fatalf("upsert conector: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:             "Codex1",
		ProyectoID:         &proyectoID,
		CWD:                filepath.Join(tmp, "orquestador"),
		Herramienta:        "codex-remote",
		ExternalSessionID:  "sess-remote-blocked",
		ResumenContinuidad: "continuidad activa",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("get handle: %+v err=%v", handle, err)
	}
	if _, err := db.DB.Exec(`
		UPDATE runtime_handles
		SET transporte='api', estado='fallido', metadata_json='{"conector":"codex-remote","remote_sync_failures":3}'
		WHERE id = ?`, handle.ID); err != nil {
		t.Fatalf("degradar handle: %v", err)
	}
	runtime, err := db.GetRuntimeBySesionID(sesion.ID)
	if err != nil || runtime == nil {
		t.Fatalf("get runtime: %+v err=%v", runtime, err)
	}
	if _, err := db.DB.Exec(`
		UPDATE runtime_instances
		SET connector='codex-remote', logical_state='degradado', process_state='remote_status_error'
		WHERE id = ?`, runtime.ID); err != nil {
		t.Fatalf("degradar runtime: %v", err)
	}
	conector, err := db.GetConector("codex-remote")
	if err != nil {
		t.Fatalf("get conector: %v", err)
	}
	cooldown := time.Now().UTC().Add(time.Minute)
	if err := db.UpsertConectorOperacion(&db.ConectorOperacion{
		ConectorID:         conector.ID,
		EstadoOperativo:    db.ConectorOperativoCircuitoAbierto,
		Motivo:             "remote_connector_unavailable",
		FallosConsecutivos: 3,
		CooldownUntil:      &cooldown,
	}); err != nil {
		t.Fatalf("upsert conector operacion: %v", err)
	}

	n, err := procesarAutonomiaAgentesBatch()
	if err != nil {
		t.Fatalf("procesar autonomia: %v", err)
	}
	if n != 1 {
		t.Fatalf("esperaba 1 decision autonoma, got=%d", n)
	}

	op, err := db.GetProyectoOperacion(proyectoID)
	if err != nil {
		t.Fatalf("get proyecto operacion: %v", err)
	}
	if op.EstadoOperativo != db.ProyectoOperativoBloqueadoExterno || !strings.Contains(op.Motivo, "conector:codex-remote:circuito_abierto") {
		t.Fatalf("proyecto no bloqueado por conector: %+v", op)
	}
	agente := "Codex1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	if len(orders) != 1 || orders[0].Tipo != "pause" || !strings.Contains(orders[0].PayloadJSON, "conector:codex-remote:circuito_abierto") {
		t.Fatalf("el daemon deberia pausar y no relanzar mientras el circuito esta abierto: %+v", orders)
	}
}

func TestProcesarRuntimeTranscriptBatchEncolaGuiaAutomatica(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
		Branch:      "main",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	logPath := filepath.Join(tmp, "codex-transcript.log")
	if err := os.WriteFile(logPath, []byte("¿me dejas seguir con el refactor?\n"), 0o600); err != nil {
		t.Fatalf("write log: %v", err)
	}
	metaJSON, _ := json.Marshal(map[string]any{
		"log_path":       logPath,
		"driver":         "process_pty_cli",
		"stdin_path":     filepath.Join(tmp, "pty.stdin"),
		"supervisor_ref": filepath.Join(tmp, "supervisor.ref"),
		"rendered_command": filepath.Join(
			tmp, "codex-perfiles", "bin", "codex-perfil",
		) + " Codex1",
		"can_send_input": false,
	})
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET metadata_json=? WHERE id=?`, string(metaJSON), handle.ID); err != nil {
		t.Fatalf("update handle metadata: %v", err)
	}

	n, err := procesarRuntimeTranscriptBatch()
	if err != nil {
		t.Fatalf("procesar transcript batch: %v", err)
	}
	if n < 2 {
		t.Fatalf("esperaba ingestión + señal procesada, got=%d", n)
	}

	agente := "Codex1"
	transcript, err := db.ListarRuntimeTranscript(db.FiltroRuntimeTranscript{Agente: &agente, Limit: 10})
	if err != nil {
		t.Fatalf("listar transcript: %v", err)
	}
	if len(transcript) == 0 || transcript[0].HandledAt == nil {
		t.Fatalf("se esperaba transcript gestionado: %+v", transcript)
	}
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar runtime orders: %v", err)
	}
	if len(orders) != 1 || orders[0].Tipo != "send_instruction" {
		t.Fatalf("send_instruction no encolada: %+v", orders)
	}
	if !strings.Contains(orders[0].PayloadJSON, `"classification":"approval_request"`) {
		t.Fatalf("payload de guía automática inesperado: %s", orders[0].PayloadJSON)
	}
	if !strings.Contains(orders[0].PayloadJSON, "Aprobado automaticamente") {
		t.Fatalf("la guía automática debería aprobar explícitamente acciones normales: %s", orders[0].PayloadJSON)
	}
	kind := "auto_guidance_sent"
	events, err := db.ListarRuntimeEvents(db.FiltroRuntimeEvents{
		Agente:     &agente,
		ProyectoID: &proyectoID,
		Kind:       &kind,
		Limit:      5,
	})
	if err != nil {
		t.Fatalf("listar runtime events: %v", err)
	}
	if len(events) != 1 || events[0] == nil {
		t.Fatalf("evento auto_guidance_sent inesperado: %+v", events)
	}
	if events[0].Payload == nil {
		t.Fatalf("el evento debería exponer payload parseado: %+v", events[0])
	}
	if got, _ := events[0].Payload["classification"].(string); got != "approval_request" {
		t.Fatalf("clasificación del evento inesperada: %+v", events[0].Payload)
	}
	if got, _ := events[0].Payload["signal_text"].(string); !strings.Contains(got, "refactor") {
		t.Fatalf("signal_text del evento inesperado: %+v", events[0].Payload)
	}
	instruction, _ := events[0].Payload["instruction"].(map[string]any)
	if instruction == nil {
		t.Fatalf("el evento debería incluir instruction útil: %+v", events[0].Payload)
	}
	if got, _ := instruction["to_agente"].(string); got != "Codex1" {
		t.Fatalf("instruction.to_agente inesperado: %+v", instruction)
	}
	if got, _ := instruction["texto"].(string); !strings.Contains(got, "Aprobado automaticamente") {
		t.Fatalf("instruction.texto sin guía útil: %+v", instruction)
	}
	var notif *db.EventoNotificacion
	for {
		select {
		case item := <-db.CanalNotificaciones:
			if item.Tipo == "runtime_auto_guidance" && item.Agente == "Codex1" {
				copy := item
				notif = &copy
			}
		default:
			goto notificationsChecked
		}
	}

notificationsChecked:
	if notif == nil {
		t.Fatalf("faltaba notificación runtime_auto_guidance en el canal")
	}
	if !strings.Contains(notif.Texto, "refactor") {
		t.Fatalf("notificación runtime_auto_guidance sin contexto útil: %+v", notif)
	}
	if notif.Payload == nil {
		t.Fatalf("notificación runtime_auto_guidance sin payload útil: %+v", notif)
	}
	if got, _ := notif.Payload["classification"].(string); got != "approval_request" {
		t.Fatalf("payload de notificación inesperado: %+v", notif.Payload)
	}
}

func TestConstruirRespuestaSignalTranscriptAplicaPoliticaPermisos(t *testing.T) {
	t.Parallel()

	cases := []struct {
		nombre         string
		classification string
		texto          string
		wantContains   []string
	}{
		{
			nombre:         "aprueba_trabajo_normal",
			classification: "approval_request",
			texto:          "me dejas seguir con el refactor y ejecutar go test ./...?",
			wantContains:   []string{"Aprobado automaticamente", "go test", "No necesitas confirmacion humana"},
		},
		{
			nombre:         "rechaza_accion_destructiva",
			classification: "approval_request",
			texto:          "quieres que haga git reset --hard y rm -rf de los cambios?",
			wantContains:   []string{"No autorizado automaticamente", "git reset --hard", "workspace"},
		},
		{
			nombre:         "detecta_dependencia_externa",
			classification: "waiting_human",
			texto:          "quedo a la espera, faltan credenciales oauth y token externo",
			wantContains:   []string{"No inventes credenciales", "otro frente util"},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.nombre, func(t *testing.T) {
			t.Parallel()
			resp := construirRespuestaSignalTranscript(&db.RuntimeTranscriptEntry{
				Classification: tc.classification,
				NormalizedText: tc.texto,
				Text:           tc.texto,
			})
			for _, token := range tc.wantContains {
				if !strings.Contains(resp, token) {
					t.Fatalf("respuesta sin %q:\n%s", token, resp)
				}
			}
		})
	}
}

func TestProcesarRuntimeTranscriptBatchDespiertaSupervisorPorSignal(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar worker: %v", err)
	}
	if err := db.RegistrarAgente("CodexSupervisor", "admin"); err != nil {
		t.Fatalf("registrar supervisor: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := db.UpsertProyectoAutonomia(&db.ProyectoAutonomia{
		ProyectoID:           proyectoID,
		Enabled:              true,
		ObjetivoGeneral:      "Terminar la app",
		DefinitionOfDoneJSON: `{"done":true}`,
		SupervisorAgente:     "CodexSupervisor",
		EstadoAutonomia:      db.AutonomiaProyectoActiva,
	}); err != nil {
		t.Fatalf("upsert proyecto autonomia: %v", err)
	}
	if err := db.ActivarAsignacion("CodexSupervisor", proyectoID, "supervision"); err != nil {
		t.Fatalf("activar asignacion supervisor: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "CodexSupervisor",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
		Branch:      "main",
	}); err != nil {
		t.Fatalf("iniciar sesion supervisor: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
		Branch:      "main",
	})
	if err != nil {
		t.Fatalf("iniciar sesion worker: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle worker: %+v err=%v", handle, err)
	}
	logPath := filepath.Join(tmp, "codex-transcript-supervisor.log")
	if err := os.WriteFile(logPath, []byte("quedo a la espera de tu respuesta\n"), 0o600); err != nil {
		t.Fatalf("write log: %v", err)
	}
	metaJSON, _ := json.Marshal(map[string]any{
		"log_path":         logPath,
		"driver":           "process_pty_cli",
		"stdin_path":       filepath.Join(tmp, "pty.stdin"),
		"supervisor_ref":   filepath.Join(tmp, "supervisor.ref"),
		"rendered_command": filepath.Join(tmp, "codex-perfiles", "bin", "codex-perfil") + " Codex1",
		"can_send_input":   false,
	})
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET metadata_json=? WHERE id=?`, string(metaJSON), handle.ID); err != nil {
		t.Fatalf("update handle metadata: %v", err)
	}

	n, err := procesarRuntimeTranscriptBatch()
	if err != nil {
		t.Fatalf("procesar transcript batch: %v", err)
	}
	if n < 2 {
		t.Fatalf("esperaba ingestión + señal procesada, got=%d", n)
	}

	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar runtime orders: %v", err)
	}
	var workerGuide *db.RuntimeOrder
	var supervisorNudge *db.RuntimeOrder
	for _, order := range orders {
		if order == nil {
			continue
		}
		switch {
		case order.Agente == "Codex1" && order.Tipo == "send_instruction":
			workerGuide = order
		case order.Agente == "CodexSupervisor" && order.Tipo == "nudge":
			supervisorNudge = order
		}
	}
	if workerGuide == nil || !strings.Contains(workerGuide.PayloadJSON, `"classification":"waiting_human"`) {
		t.Fatalf("guía automática del worker inesperada: %+v", workerGuide)
	}
	if supervisorNudge == nil || !strings.Contains(supervisorNudge.PayloadJSON, `"accion":"inspeccionar_transcript_signal"`) {
		t.Fatalf("nudge al supervisor inesperado: %+v", supervisorNudge)
	}
}

func TestProcesarRuntimeTranscriptBatchDespiertaReviewerPorReadyForReview(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar worker: %v", err)
	}
	if err := db.RegistrarAgente("CodexReviewer", "admin"); err != nil {
		t.Fatalf("registrar reviewer: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := db.UpsertProyectoAutonomia(&db.ProyectoAutonomia{
		ProyectoID:           proyectoID,
		Enabled:              true,
		ObjetivoGeneral:      "Terminar la app",
		DefinitionOfDoneJSON: `{"done":true}`,
		ReviewerAgente:       "CodexReviewer",
		ReviewRequired:       true,
		EstadoAutonomia:      db.AutonomiaProyectoActiva,
	}); err != nil {
		t.Fatalf("upsert proyecto autonomia: %v", err)
	}
	if err := db.ActivarAsignacion("CodexReviewer", proyectoID, "review"); err != nil {
		t.Fatalf("activar asignacion reviewer: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "CodexReviewer",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
		Branch:      "main",
	}); err != nil {
		t.Fatalf("iniciar sesion reviewer: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
		Branch:      "main",
	})
	if err != nil {
		t.Fatalf("iniciar sesion worker: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle worker: %+v err=%v", handle, err)
	}
	logPath := filepath.Join(tmp, "codex-transcript-review.log")
	if err := os.WriteFile(logPath, []byte("está listo para revisión final\n"), 0o600); err != nil {
		t.Fatalf("write log: %v", err)
	}
	metaJSON, _ := json.Marshal(map[string]any{
		"log_path":         logPath,
		"driver":           "process_pty_cli",
		"stdin_path":       filepath.Join(tmp, "pty.stdin"),
		"supervisor_ref":   filepath.Join(tmp, "supervisor.ref"),
		"rendered_command": filepath.Join(tmp, "codex-perfiles", "bin", "codex-perfil") + " Codex1",
		"can_send_input":   false,
	})
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET metadata_json=? WHERE id=?`, string(metaJSON), handle.ID); err != nil {
		t.Fatalf("update handle metadata: %v", err)
	}

	n, err := procesarRuntimeTranscriptBatch()
	if err != nil {
		t.Fatalf("procesar transcript batch: %v", err)
	}
	if n < 2 {
		t.Fatalf("esperaba ingestión + señal procesada, got=%d", n)
	}

	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar runtime orders: %v", err)
	}
	var workerGuide *db.RuntimeOrder
	var reviewerNudge *db.RuntimeOrder
	for _, order := range orders {
		if order == nil {
			continue
		}
		switch {
		case order.Agente == "Codex1" && order.Tipo == "send_instruction":
			workerGuide = order
		case order.Agente == "CodexReviewer" && order.Tipo == "nudge":
			reviewerNudge = order
		}
	}
	if workerGuide == nil || !strings.Contains(workerGuide.PayloadJSON, `"classification":"ready_for_review"`) {
		t.Fatalf("guía automática del worker inesperada: %+v", workerGuide)
	}
	if reviewerNudge == nil || !strings.Contains(reviewerNudge.PayloadJSON, `"accion":"inspeccionar_ready_for_review"`) {
		t.Fatalf("nudge al reviewer inesperado: %+v", reviewerNudge)
	}
}

func TestProcesarRuntimeTranscriptBatchNeedsReplanAbreTareaYDespiertaSupervisor(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar worker: %v", err)
	}
	if err := db.RegistrarAgente("CodexSupervisor", "admin"); err != nil {
		t.Fatalf("registrar supervisor: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := db.UpsertProyectoAutonomia(&db.ProyectoAutonomia{
		ProyectoID:           proyectoID,
		Enabled:              true,
		ObjetivoGeneral:      "Terminar la app",
		DefinitionOfDoneJSON: `{"done":true}`,
		SupervisorAgente:     "CodexSupervisor",
		ReviewRequired:       true,
		EstadoAutonomia:      db.AutonomiaProyectoActiva,
	}); err != nil {
		t.Fatalf("upsert proyecto autonomia: %v", err)
	}
	if err := db.ActivarAsignacion("CodexSupervisor", proyectoID, "supervision"); err != nil {
		t.Fatalf("activar asignacion supervisor: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "CodexSupervisor",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
		Branch:      "main",
	}); err != nil {
		t.Fatalf("iniciar sesion supervisor: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
		Branch:      "main",
	})
	if err != nil {
		t.Fatalf("iniciar sesion worker: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle worker: %+v err=%v", handle, err)
	}
	logPath := filepath.Join(tmp, "codex-transcript-needs-replan.log")
	if err := os.WriteFile(logPath, []byte("¿Qué hago ahora? no tengo claro el siguiente paso\n"), 0o600); err != nil {
		t.Fatalf("write log: %v", err)
	}
	metaJSON, _ := json.Marshal(map[string]any{"log_path": logPath})
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET metadata_json=? WHERE id=?`, string(metaJSON), handle.ID); err != nil {
		t.Fatalf("update handle metadata: %v", err)
	}

	n, err := procesarRuntimeTranscriptBatch()
	if err != nil {
		t.Fatalf("procesar transcript batch: %v", err)
	}
	if n < 2 {
		t.Fatalf("esperaba ingestión + señal procesada, got=%d", n)
	}

	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar runtime orders: %v", err)
	}
	var workerGuide *db.RuntimeOrder
	var supervisorNudge *db.RuntimeOrder
	for _, order := range orders {
		if order == nil {
			continue
		}
		switch {
		case order.Agente == "Codex1" && order.Tipo == "send_instruction":
			workerGuide = order
		case order.Agente == "CodexSupervisor" && order.Tipo == "nudge":
			supervisorNudge = order
		}
	}
	if workerGuide == nil || !strings.Contains(workerGuide.PayloadJSON, `"classification":"needs_replan"`) {
		t.Fatalf("guía automática del worker inesperada: %+v", workerGuide)
	}
	if supervisorNudge == nil || !strings.Contains(supervisorNudge.PayloadJSON, `"accion":"inspeccionar_transcript_signal"`) {
		t.Fatalf("nudge al supervisor inesperado: %+v", supervisorNudge)
	}
	tareas, err := db.ListarTareas(db.FiltroTareas{ProyectoID: &proyectoID})
	if err != nil {
		t.Fatalf("listar tareas: %v", err)
	}
	var replanTask *db.Tarea
	for _, tarea := range tareas {
		if tarea == nil || tarea.Titulo != autonomiaReplanTaskTitle {
			continue
		}
		replanTask = tarea
		break
	}
	if replanTask == nil {
		t.Fatalf("debería crear una tarea explícita de replanificación, tareas=%+v", tareas)
	}
	if replanTask.Estado != db.TareaEnProgreso {
		t.Fatalf("la tarea de replan debería arrancarse para el supervisor, got=%s", replanTask.Estado)
	}
	if replanTask.Agente == nil || *replanTask.Agente != "CodexSupervisor" {
		t.Fatalf("la tarea de replan debería quedar en el supervisor, tarea=%+v", replanTask)
	}
	if !strings.Contains(replanTask.Notas, "autonomia:needs_replan") {
		t.Fatalf("la tarea de replan debería quedar marcada como needs_replan, tarea=%+v", replanTask)
	}
}

func TestProcesarRuntimeTranscriptBatchNoGuiaAlWorkerEnRuntimePanic(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar worker: %v", err)
	}
	if err := db.RegistrarAgente("CodexSupervisor", "admin"); err != nil {
		t.Fatalf("registrar supervisor: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := db.UpsertProyectoAutonomia(&db.ProyectoAutonomia{
		ProyectoID:       proyectoID,
		Enabled:          true,
		ObjetivoGeneral:  "Terminar la app",
		SupervisorAgente: "CodexSupervisor",
		EstadoAutonomia:  db.AutonomiaProyectoActiva,
	}); err != nil {
		t.Fatalf("upsert proyecto autonomia: %v", err)
	}
	if err := db.ActivarAsignacion("CodexSupervisor", proyectoID, "supervision"); err != nil {
		t.Fatalf("activar asignacion supervisor: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "CodexSupervisor",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
		Branch:      "main",
	}); err != nil {
		t.Fatalf("iniciar sesion supervisor: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
		Branch:      "main",
	})
	if err != nil {
		t.Fatalf("iniciar sesion worker: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle worker: %+v err=%v", handle, err)
	}
	logPath := filepath.Join(tmp, "codex-transcript-panic.log")
	if err := os.WriteFile(logPath, []byte("thread 'main' panicked at src/ui.rs:1:1\n"), 0o600); err != nil {
		t.Fatalf("write log: %v", err)
	}
	metaJSON, _ := json.Marshal(map[string]any{
		"log_path":         logPath,
		"driver":           "process_pty_cli",
		"stdin_path":       filepath.Join(tmp, "pty.stdin"),
		"supervisor_ref":   filepath.Join(tmp, "supervisor.ref"),
		"rendered_command": filepath.Join(tmp, "codex-perfiles", "bin", "codex-perfil") + " Codex1",
		"can_send_input":   false,
	})
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET metadata_json=? WHERE id=?`, string(metaJSON), handle.ID); err != nil {
		t.Fatalf("update handle metadata: %v", err)
	}

	n, err := procesarRuntimeTranscriptBatch()
	if err != nil {
		t.Fatalf("procesar transcript batch: %v", err)
	}
	if n < 2 {
		t.Fatalf("esperaba ingestión + señal procesada, got=%d", n)
	}

	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar runtime orders: %v", err)
	}
	var workerGuide *db.RuntimeOrder
	var supervisorNudge *db.RuntimeOrder
	for _, order := range orders {
		if order == nil {
			continue
		}
		switch {
		case order.Agente == "Codex1" && order.Tipo == "send_instruction":
			workerGuide = order
		case order.Agente == "CodexSupervisor" && order.Tipo == "nudge":
			supervisorNudge = order
		}
	}
	if workerGuide != nil {
		t.Fatalf("no debería guiar al worker tras runtime_panic: %+v", workerGuide)
	}
	if supervisorNudge == nil || !strings.Contains(supervisorNudge.PayloadJSON, `"accion":"inspeccionar_transcript_signal"`) {
		t.Fatalf("nudge al supervisor inesperado: %+v", supervisorNudge)
	}
	agenteInfo, err := db.GetAgente("Codex1")
	if err != nil || agenteInfo == nil {
		t.Fatalf("get agente: agente=%+v err=%v", agenteInfo, err)
	}
	if !strings.Contains(agenteInfo.MotivoPausa, "runtime_panic") {
		t.Fatalf("motivo_pausa sin trazabilidad de runtime_panic: %+v", agenteInfo)
	}
	var estadoCuota string
	var reanimarAt sql.NullTime
	if err := db.DB.QueryRow(`SELECT estado_cuota, reanimar_at FROM agentes WHERE nombre=?`, "Codex1").Scan(&estadoCuota, &reanimarAt); err != nil {
		t.Fatalf("leer estado persistido de agente: %v", err)
	}
	if estadoCuota != "enfriamiento" {
		t.Fatalf("runtime_panic deberia persistir enfriamiento, got=%s agente=%+v", estadoCuota, agenteInfo)
	}
	if !reanimarAt.Valid {
		t.Fatalf("runtime_panic deberia persistir reanimar_at, agente=%+v", agenteInfo)
	}
	handle, err = db.GetRuntimeHandle(handle.ID)
	if err != nil || handle == nil {
		t.Fatalf("reload handle: %+v err=%v", handle, err)
	}
	if !strings.Contains(handle.MetadataJSON, `"disable_supervisor_hot_input":true`) {
		t.Fatalf("runtime_panic deberia degradar supervisor_local en el handle: %s", handle.MetadataJSON)
	}
}

func TestEnfriarAgentePorRuntimePanicNoDegradaHandleDeGeneracionNueva(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
		Branch:      "main",
	})
	if err != nil {
		t.Fatalf("iniciar sesion worker: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle worker: %+v err=%v", handle, err)
	}
	startedAt := time.Now().UTC()
	metaJSON, _ := json.Marshal(map[string]any{
		"log_path":         filepath.Join(tmp, "panic.log"),
		"driver":           "process_pty_cli",
		"stdin_path":       filepath.Join(tmp, "pty.stdin"),
		"supervisor_ref":   filepath.Join(tmp, "supervisor.ref"),
		"rendered_command": filepath.Join(tmp, "codex-perfiles", "bin", "codex-perfil") + " Codex1",
		"can_send_input":   false,
		"started_at":       startedAt.Format(time.RFC3339Nano),
	})
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET metadata_json=? WHERE id=?`, string(metaJSON), handle.ID); err != nil {
		t.Fatalf("update handle metadata: %v", err)
	}

	item := &db.RuntimeTranscriptEntry{
		ID:             10576,
		Agente:         "Codex1",
		HandleID:       &handle.ID,
		CreatedAt:      startedAt.Add(-2 * time.Second),
		Stream:         "pty_out",
		Classification: "runtime_panic",
	}
	if _, err := enfriarAgentePorRuntimePanic("Codex1", item); err != nil {
		t.Fatalf("enfriar agente por runtime panic: %v", err)
	}
	handle, err = db.GetRuntimeHandle(handle.ID)
	if err != nil || handle == nil {
		t.Fatalf("reload handle: %+v err=%v", handle, err)
	}
	if strings.Contains(handle.MetadataJSON, `"disable_supervisor_hot_input":true`) {
		t.Fatalf("no deberia degradar el handle actual por un panic de la generacion anterior: %s", handle.MetadataJSON)
	}
}

func TestProcesarRuntimeTranscriptBatchResuelveReviewGateDesdeReviewer(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("CodexReviewer", "admin"); err != nil {
		t.Fatalf("registrar reviewer: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := db.UpsertProyectoAutonomia(&db.ProyectoAutonomia{
		ProyectoID:           proyectoID,
		Enabled:              true,
		ObjetivoGeneral:      "Terminar la app",
		DefinitionOfDoneJSON: `{"done":true}`,
		ReviewerAgente:       "CodexReviewer",
		ReviewRequired:       true,
		EstadoAutonomia:      db.AutonomiaProyectoActiva,
	}); err != nil {
		t.Fatalf("upsert proyecto autonomia: %v", err)
	}
	if err := db.ActivarAsignacion("CodexReviewer", proyectoID, "review"); err != nil {
		t.Fatalf("activar asignacion reviewer: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:     "Cerrar review",
		ProyectoID: &proyectoID,
		Prioridad:  db.PrioridadAlta,
		CreadoPor:  "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "CodexReviewer"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "CodexReviewer"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	if err := db.CompletarTarea(tareaID, "CodexReviewer", "abc123"); err != nil {
		t.Fatalf("completar tarea: %v", err)
	}
	gateID, err := db.CrearReviewGate(&db.ReviewGate{
		ProyectoID:     &proyectoID,
		TareaID:        &tareaID,
		RequestedBy:    "orquesta",
		ReviewerAgente: "CodexReviewer",
		Estado:         db.ReviewGateEnRevision,
		SeverityMax:    "high",
	})
	if err != nil {
		t.Fatalf("crear review gate: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "CodexReviewer",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
		Branch:      "main",
	})
	if err != nil {
		t.Fatalf("iniciar sesion reviewer: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle reviewer: %+v err=%v", handle, err)
	}
	logPath := filepath.Join(tmp, "codex-transcript-review-approved.log")
	if err := os.WriteFile(logPath, []byte("Review aprobada, LGTM\n"), 0o600); err != nil {
		t.Fatalf("write log: %v", err)
	}
	metaJSON, _ := json.Marshal(map[string]any{"log_path": logPath})
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET metadata_json=? WHERE id=?`, string(metaJSON), handle.ID); err != nil {
		t.Fatalf("update handle metadata: %v", err)
	}

	n, err := procesarRuntimeTranscriptBatch()
	if err != nil {
		t.Fatalf("procesar transcript batch: %v", err)
	}
	if n < 2 {
		t.Fatalf("esperaba ingestión + resolución de gate, got=%d", n)
	}
	gate, err := db.GetReviewGate(gateID)
	if err != nil {
		t.Fatalf("get review gate: %v", err)
	}
	if gate == nil || gate.Estado != db.ReviewGateAprobado {
		t.Fatalf("el review gate debería quedar aprobado, gate=%+v", gate)
	}
	if !strings.Contains(gate.FindingsJSON, `"classification":"review_approved"`) {
		t.Fatalf("los findings deberían reflejar el transcript que resolvió el gate: %s", gate.FindingsJSON)
	}
	agente := "CodexReviewer"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar runtime orders: %v", err)
	}
	if len(orders) != 0 {
		t.Fatalf("no debería dejar nudges o instrucciones pendientes al resolver el gate directamente: %+v", orders)
	}
}

func TestAsegurarTareaReplanAutonomiaSignalCreaFrenteParaSupervisor(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	policy := &db.ProyectoAutonomia{
		ProyectoID:           proyectoID,
		Enabled:              true,
		ObjetivoGeneral:      "Terminar la app sin intervención humana",
		DefinitionOfDoneJSON: `{"tests":"green"}`,
		SupervisorAgente:     "CodexSupervisor",
	}
	proyecto, err := db.GetProyecto(strconv.FormatInt(proyectoID, 10))
	if err != nil {
		t.Fatalf("get proyecto: %v", err)
	}
	supervisor := &db.Agente{Nombre: "CodexSupervisor", Rol: "admin"}
	item := &db.RuntimeTranscriptEntry{
		ID:             77,
		ProyectoID:     &proyectoID,
		Agente:         "CodexWorker",
		Classification: "needs_replan",
		Text:           "No tengo claro el siguiente paso útil",
	}

	note, err := asegurarTareaReplanAutonomiaSignal(item, policy, proyecto, supervisor)
	if err != nil {
		t.Fatalf("asegurarTareaReplanAutonomiaSignal: %v", err)
	}
	if !strings.HasPrefix(note, "replan_task_created:") {
		t.Fatalf("nota inesperada: %s", note)
	}
	tareas, err := db.ListarTareas(db.FiltroTareas{ProyectoID: &proyectoID})
	if err != nil {
		t.Fatalf("listar tareas: %v", err)
	}
	if len(tareas) != 1 {
		t.Fatalf("debería crear exactamente una tarea de replan, tareas=%+v", tareas)
	}
	if tareas[0].Titulo != autonomiaReplanTaskTitle {
		t.Fatalf("título de replan inesperado: %+v", tareas[0])
	}
	if tareas[0].Estado != db.TareaEnProgreso {
		t.Fatalf("la tarea de replan debería arrancarse para el supervisor, tarea=%+v", tareas[0])
	}
	if tareas[0].Agente == nil || *tareas[0].Agente != "CodexSupervisor" {
		t.Fatalf("la tarea de replan debería quedar en el supervisor, tarea=%+v", tareas[0])
	}
}

func TestAsegurarSolicitudMergeDesdeGateAprobadoCreaSolicitud(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:     "Cerrar review",
		ProyectoID: &proyectoID,
		Prioridad:  db.PrioridadAlta,
		CreadoPor:  "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	worktree, err := db.CoordinationWorktreeSQLRepository{}.Create(&coordinacion.Worktree{
		ProjectID: proyectoID,
		TaskID:    &tareaID,
		Agent:     "CodexReviewer",
		Name:      "orquestador-codexreviewer-t1",
		Path:      filepath.Join(tmp, "wt-review-merge"),
		Branch:    "orq/orquestador/CodexReviewer/t1",
		BaseRef:   "master",
		State:     coordinacion.WorktreeActive,
		Reason:    "review_merge",
	})
	if err != nil {
		t.Fatalf("crear worktree: %v", err)
	}
	proyecto, err := db.GetProyecto(strconv.FormatInt(proyectoID, 10))
	if err != nil {
		t.Fatalf("get proyecto: %v", err)
	}
	gate := &reviewapp.Gate{
		ID:             91,
		ProyectoID:     proyectoID,
		ProyectoSlug:   proyecto.Slug,
		TareaID:        &tareaID,
		WorktreeID:     &worktree.ID,
		ReviewerAgente: "CodexReviewer",
		Estado:         reviewapp.GateStateApproved,
	}

	created, err := asegurarSolicitudMergeDesdeGateAprobado(proyecto, gate)
	if err != nil {
		t.Fatalf("asegurarSolicitudMergeDesdeGateAprobado: %v", err)
	}
	if !created {
		t.Fatal("debería crear una solicitud de merge para un gate aprobado con worktree válido")
	}
	merges, err := db.ListarGitMerges(&proyectoID, "")
	if err != nil {
		t.Fatalf("listar merges: %v", err)
	}
	if len(merges) != 1 {
		t.Fatalf("debería crear exactamente una solicitud de merge, merges=%+v", merges)
	}
	if merges[0].SourceBranch != worktree.Branch || merges[0].TargetBranch != "master" || merges[0].Estado != "aprobado" {
		t.Fatalf("solicitud de merge inesperada: %+v", merges[0])
	}
	if !strings.Contains(merges[0].MetadataJSON, `"source":"review_gate_approved"`) {
		t.Fatalf("metadata de merge inesperada: %s", merges[0].MetadataJSON)
	}
}

func TestProcesarGitMergesBatchFusionaCierraWorktreeYCompletaTarea(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)
	repo := prepararRepoGitAutonomia(t, filepath.Join(tmp, "orquestador"))

	if err := db.RegistrarAgente("CodexReviewer", "admin"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: repo,
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := db.UpsertProyectoAutonomia(&db.ProyectoAutonomia{
		ProyectoID:           proyectoID,
		Enabled:              true,
		ObjetivoGeneral:      "Terminar la app",
		DefinitionOfDoneJSON: `{"done":true}`,
		ReviewRequired:       true,
		AutoCloseProject:     true,
		EstadoAutonomia:      db.AutonomiaProyectoActiva,
	}); err != nil {
		t.Fatalf("upsert proyecto autonomia: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:     "Integrar cambio",
		ProyectoID: &proyectoID,
		Prioridad:  db.PrioridadAlta,
		CreadoPor:  "orquesta",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "CodexReviewer"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "CodexReviewer"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	worktree, err := newCoordinationService().PrepareWorktree(coordinacion.PrepareWorktreeInput{
		ProjectRef: "orquestador",
		Agent:      "CodexReviewer",
		TaskID:     &tareaID,
		Reason:     "review_merge",
	})
	if err != nil {
		t.Fatalf("prepare worktree: %v", err)
	}
	cmdGitAutonomia(t, worktree.Path, "config", "user.name", "Orquesta Test")
	cmdGitAutonomia(t, worktree.Path, "config", "user.email", "orquesta@example.test")
	if err := os.WriteFile(filepath.Join(worktree.Path, "feature.txt"), []byte("hola\n"), 0o644); err != nil {
		t.Fatalf("write feature: %v", err)
	}
	cmdGitAutonomia(t, worktree.Path, "add", "feature.txt")
	cmdGitAutonomia(t, worktree.Path, "commit", "-m", "feat: add feature")
	cmdGitAutonomia(t, repo, "checkout", "--detach")

	metadataJSON, err := json.Marshal(map[string]any{
		"auto_created": true,
		"source":       "review_gate_approved",
		"review_gate":  91,
		"worktree_id":  worktree.ID,
		"tarea_id":     tareaID,
	})
	if err != nil {
		t.Fatalf("marshal metadata: %v", err)
	}
	if _, err := db.GuardarGitMerge(&db.GitMerge{
		ProyectoID:   proyectoID,
		SourceBranch: worktree.Branch,
		TargetBranch: "master",
		RequestedBy:  "CodexReviewer",
		Estado:       "aprobado",
		MetadataJSON: string(metadataJSON),
	}); err != nil {
		t.Fatalf("guardar git merge: %v", err)
	}

	n, err := procesarGitMergesBatch()
	if err != nil {
		t.Fatalf("procesar git merges: %v", err)
	}
	if n != 1 {
		t.Fatalf("esperaba 1 merge procesado, got=%d", n)
	}
	merges, err := db.ListarGitMerges(&proyectoID, "")
	if err != nil {
		t.Fatalf("listar merges: %v", err)
	}
	if len(merges) != 1 || merges[0].Estado != "fusionado" {
		if len(merges) == 0 {
			t.Fatalf("merge no fusionado correctamente: lista vacía")
		}
		t.Fatalf("merge no fusionado correctamente: %+v", *merges[0])
	}
	tarea, err := db.GetTarea(tareaID)
	if err != nil {
		t.Fatalf("get tarea: %v", err)
	}
	if tarea.Estado != db.TareaCompletada || strings.TrimSpace(tarea.CommitCierre) != strings.TrimSpace(merges[0].CommitMerge) {
		t.Fatalf("tarea no completada con commit de merge: %+v merge=%+v", tarea, merges[0])
	}
	wt, err := db.CoordinationWorktreeRepository().GetByID(worktree.ID)
	if err != nil {
		t.Fatalf("get worktree: %v", err)
	}
	if wt.State != coordinacion.WorktreeClosed {
		t.Fatalf("worktree debería quedar cerrada: %+v", wt)
	}
	cmdGitAutonomia(t, repo, "checkout", "master")
	if _, err := os.Stat(worktree.Path); !os.IsNotExist(err) {
		t.Fatalf("la ruta de worktree debería haberse eliminado: err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(repo, "feature.txt")); err != nil {
		t.Fatalf("feature.txt debería quedar fusionado en repo base: %v", err)
	}
}

func TestProcesarRuntimeMailboxBatchMaterializaAutonomiaMailbox(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
		Branch:      "main",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	metaJSON := `{"driver":"process_pty_cli","stdin_path":"` + filepath.Join(tmp, "pty.stdin") + `","supervisor_ref":"` + filepath.Join(tmp, "supervisor.ref") + `","rendered_command":"codex-perfil Codex1","working_dir":"` + filepath.Join(tmp, "orquestador") + `","external_session_id":"sess-batch-main","can_send_input":false}`
	capsJSON := `{"can_send_input":false,"mailbox_delivery_mode":"` + runtimeagente.MailboxDeliveryBootstrapOnly + `"}`
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET capabilities_json=?, metadata_json=? WHERE id=?`, capsJSON, metaJSON, handle.ID); err != nil {
		t.Fatalf("update handle supervisor local: %v", err)
	}
	if _, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "orquesta",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "autonomia",
		PayloadJSON: `{"accion":"continuar_trabajo","motivo":"seguir el frente activo"}`,
	}); err != nil {
		t.Fatalf("crear mailbox: %v", err)
	}

	n, err := procesarRuntimeMailboxBatch()
	if err != nil {
		t.Fatalf("procesar mailbox batch: %v", err)
	}
	if n < 1 {
		t.Fatalf("esperaba al menos una materialización de mailbox, got=%d", n)
	}
	agente := "Codex1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar runtime orders: %v", err)
	}
	if len(orders) != 1 || orders[0].Tipo != "send_instruction" {
		t.Fatalf("autonomia no materializada en send_instruction: %+v", orders)
	}
	if orders[0].HandleID == nil || *orders[0].HandleID != handle.ID {
		t.Fatalf("send_instruction sin handle activo: %+v", orders[0])
	}
	if !strings.Contains(orders[0].PayloadJSON, `"delivery_attempt_signature":"supervisor_local|handle:`) {
		t.Fatalf("autonomia del batch general deberia materializarse por supervisor_local: %s", orders[0].PayloadJSON)
	}
	mailboxEstado := "pendiente"
	mailbox, err := db.ListarRuntimeMailbox(db.FiltroRuntimeMailbox{ToAgente: &agente, ProyectoID: &proyectoID, Estado: &mailboxEstado})
	if err != nil {
		t.Fatalf("listar mailbox: %v", err)
	}
	if len(mailbox) != 1 {
		t.Fatalf("la mailbox durable debe seguir pendiente tras materialización: %+v", mailbox)
	}
}

func TestProcesarAutonomiaSesionActivaPersisteEnfriamientoPorCuota(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex5", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if err := db.ActivarAsignacion("Codex5", proyectoID, "frente agotado"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex5",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Frente detenido por cuota",
		Descripcion: "Debe pausar y persistir enfriamiento",
		ProyectoID:  &proyectoID,
		Modulo:      "web",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "tester",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Codex5"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	started := time.Now().UTC().Add(-10 * time.Minute)
	reset := time.Now().UTC().Add(90 * time.Minute)
	credits := 0.0
	if _, err := db.RegistrarPresupuestoSesion(&db.PresupuestoSesion{
		SesionID:         sesion.ID,
		WindowKind:       "5h",
		WindowStartedAt:  &started,
		ResetAt:          &reset,
		RemainingCredits: &credits,
		BudgetSource:     "codex_token_count_observed",
		CheckedAt:        time.Now().UTC(),
	}); err != nil {
		t.Fatalf("registrar presupuesto: %v", err)
	}

	n, err := procesarAutonomiaSesionActiva(sesion)
	if err != nil {
		t.Fatalf("procesar autonomia sesion: %v", err)
	}
	if n != 1 {
		t.Fatalf("esperaba 1 accion autonoma, got=%d", n)
	}
	agente, err := db.GetAgente("Codex5")
	if err != nil {
		t.Fatalf("get agente: %v", err)
	}
	if strings.TrimSpace(agente.EstadoCuota) != "enfriamiento" {
		t.Fatalf("deberia quedar en enfriamiento: %+v", agente)
	}
	if agente.ReanimarAt == nil || agente.ReanimarAt.Before(reset.Add(-time.Minute)) || agente.ReanimarAt.After(reset.Add(time.Minute)) {
		t.Fatalf("reanimar_at deberia alinearse con reset del presupuesto: %+v reset=%s", agente, reset)
	}
	agenteNombre := "Codex5"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agenteNombre, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar runtime orders: %v", err)
	}
	if len(orders) != 1 || orders[0].Tipo != "pause" {
		t.Fatalf("deberia encolar una pause: %+v", orders)
	}
}

func TestProcesarPresupuestoSesionObservadoBatchToleraHandleRoto(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)
	base := filepath.Join(tmp, "codex-perfiles")
	if err := os.MkdirAll(filepath.Join(base, "bin"), 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	sessionsDir := filepath.Join(base, "homes", "CodexBueno", "sessions", "2026", "03", "31")
	if err := os.MkdirAll(sessionsDir, 0o755); err != nil {
		t.Fatalf("mkdir sessions bueno: %v", err)
	}
	authPath := filepath.Join(base, "homes", "CodexBueno", "auth.json")
	if err := os.MkdirAll(filepath.Dir(authPath), 0o755); err != nil {
		t.Fatalf("mkdir auth bueno: %v", err)
	}
	if err := os.WriteFile(authPath, []byte(`{"profile":{"email":"bueno@example.com","name":"Cuenta Buena"}}`), 0o600); err != nil {
		t.Fatalf("write auth bueno: %v", err)
	}
	resetPrimary := time.Date(2026, 3, 31, 14, 0, 0, 0, time.UTC)
	resetSecondary := time.Date(2026, 4, 6, 7, 26, 0, 0, time.UTC)
	sessionPath := filepath.Join(sessionsDir, "rollout-test.jsonl")
	lines := []string{
		`{"timestamp":"2026-03-31T10:00:00Z","type":"session_meta","payload":{"id":"sess-buena","timestamp":"2026-03-31T10:00:00Z","cwd":"` + filepath.Join(tmp, "orquestador") + `"}}`,
		`{"timestamp":"2026-03-31T10:05:00Z","type":"event_msg","payload":{"type":"token_count","rate_limits":{"primary":{"used_percent":12,"window_minutes":300,"resets_at":` + strconv.FormatInt(resetPrimary.Unix(), 10) + `},"secondary":{"used_percent":43,"window_minutes":10080,"resets_at":` + strconv.FormatInt(resetSecondary.Unix(), 10) + `},"credits":null,"plan_type":"plus"}}}`,
	}
	if err := os.WriteFile(sessionPath, []byte(lines[0]+"\n"+lines[1]+"\n"), 0o600); err != nil {
		t.Fatalf("write session bueno: %v", err)
	}

	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	for _, nombre := range []string{"CodexBueno", "CodexRoto"} {
		if err := db.RegistrarAgente(nombre, "programador"); err != nil {
			t.Fatalf("registrar %s: %v", nombre, err)
		}
		sesion, err := db.IniciarSesionContexto(db.SesionInicio{
			Agente:      nombre,
			ProyectoID:  &proyectoID,
			CWD:         filepath.Join(tmp, "orquestador"),
			Herramienta: "codex-cli",
		})
		if err != nil {
			t.Fatalf("iniciar sesion %s: %v", nombre, err)
		}
		if err := db.UpsertRuntimeHandleDesdeSesion(sesion); err != nil {
			t.Fatalf("upsert handle %s: %v", nombre, err)
		}
	}
	bueno, err := db.GetRuntimeHandleActivoAgenteProyecto("CodexBueno", &proyectoID)
	if err != nil || bueno == nil {
		t.Fatalf("get handle bueno: %v %+v", err, bueno)
	}
	rendidoBueno := filepath.Join(base, "bin", "codex-perfil") + " CodexBueno"
	metaBueno := map[string]any{
		"rendered_command":    rendidoBueno,
		"working_dir":         filepath.Join(tmp, "orquestador"),
		"external_session_id": "sess-buena",
		"started_at":          time.Date(2026, 3, 31, 10, 0, 0, 0, time.UTC).Format(time.RFC3339),
	}
	metaBuenoJSON, _ := json.Marshal(metaBueno)
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET metadata_json=? WHERE id=?`, string(metaBuenoJSON), bueno.ID); err != nil {
		t.Fatalf("update handle bueno: %v", err)
	}

	roto, err := db.GetRuntimeHandleActivoAgenteProyecto("CodexRoto", &proyectoID)
	if err != nil || roto == nil {
		t.Fatalf("get handle roto: %v %+v", err, roto)
	}
	badRoot := filepath.Join(tmp, "codex-roto")
	badSessionsDir := filepath.Join(badRoot, "homes", "CodexRoto", "sessions", "2026", "03", "31")
	if err := os.MkdirAll(filepath.Join(badRoot, "bin"), 0o755); err != nil {
		t.Fatalf("mkdir bin roto: %v", err)
	}
	if err := os.MkdirAll(badSessionsDir, 0o755); err != nil {
		t.Fatalf("mkdir sessions roto: %v", err)
	}
	if err := os.Chmod(badSessionsDir, 0o000); err != nil {
		t.Fatalf("chmod sessions roto: %v", err)
	}
	defer func() { _ = os.Chmod(badSessionsDir, 0o755) }()
	rendidoRoto := filepath.Join(badRoot, "bin", "codex-perfil") + " CodexRoto"
	metaRoto := map[string]any{
		"rendered_command":    rendidoRoto,
		"working_dir":         filepath.Join(tmp, "orquestador"),
		"external_session_id": "sess-rota",
		"started_at":          time.Date(2026, 3, 31, 10, 0, 0, 0, time.UTC).Format(time.RFC3339),
	}
	metaRotoJSON, _ := json.Marshal(metaRoto)
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET metadata_json=? WHERE id=?`, string(metaRotoJSON), roto.ID); err != nil {
		t.Fatalf("update handle roto: %v", err)
	}

	procesados, err := procesarPresupuestoSesionObservadoBatch()
	if err != nil {
		t.Fatalf("procesar presupuesto observado: %v", err)
	}
	if procesados != 1 {
		t.Fatalf("deberia procesar solo el handle sano: %d", procesados)
	}
	agenteBueno, err := db.GetAgente("CodexBueno")
	if err != nil {
		t.Fatalf("get agente bueno: %v", err)
	}
	if agenteBueno.PresupuestoCheckedAt == nil || agenteBueno.CuentaEmail != "bueno@example.com" {
		t.Fatalf("deberia persistir telemetria del handle sano: %+v", agenteBueno)
	}
}

func prepararRepoGitAutonomia(t *testing.T, repo string) string {
	t.Helper()
	cmdGitAutonomia(t, "", "init", "-b", "master", repo)
	cmdGitAutonomia(t, repo, "config", "user.name", "Orquesta Test")
	cmdGitAutonomia(t, repo, "config", "user.email", "orquesta@example.test")
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("base\n"), 0o644); err != nil {
		t.Fatalf("write README: %v", err)
	}
	cmdGitAutonomia(t, repo, "add", "README.md")
	cmdGitAutonomia(t, repo, "commit", "-m", "base")
	return repo
}

func cmdGitAutonomia(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	if strings.TrimSpace(dir) != "" {
		cmd.Dir = dir
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out))
}
