package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"orquesta/db"
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
	if n != 0 {
		t.Fatalf("no debería contar supervisión hasta que el agente arranque, got=%d", n)
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
	if len(orders) != 1 || orders[0].Tipo != "start" {
		t.Fatalf("debería encolar start para supervisor preferido: %+v", orders)
	}
}

func TestProcesarSupervisionAutonomaBatchCreaTareaSemillaSiAutoCreateTasks(t *testing.T) {
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
	if len(tareas) != 1 {
		t.Fatalf("debería crear una tarea semilla, tareas=%+v", tareas)
	}
	if tareas[0].Titulo != "Autonomía: revisar backlog y abrir siguiente frente útil" {
		t.Fatalf("tarea semilla inesperada: %+v", tareas[0])
	}
	if tareas[0].Estado != db.TareaEnProgreso {
		t.Fatalf("la tarea semilla debería arrancarse, got=%s", tareas[0].Estado)
	}
	if tareas[0].Agente == nil || *tareas[0].Agente != "CodexSupervisor" {
		t.Fatalf("la tarea semilla debería quedar en el supervisor, tarea=%+v", tareas[0])
	}

	kind := "supervision"
	cycles, err := db.ListarAutonomiaCiclos(db.FiltroAutonomiaCiclos{ProyectoID: &proyectoID, Kind: &kind, Limit: 10})
	if err != nil {
		t.Fatalf("listar ciclos: %v", err)
	}
	if len(cycles) != 1 || !strings.Contains(cycles[0].DecisionJSON, `"auto_created_task":true`) {
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

func TestProcesarAutonomiaAgentesBatchEncolaNudgePorContinuarTrabajo(t *testing.T) {
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
		!strings.Contains(orders[0].PayloadJSON, `"accion":"continuar_trabajo"`) {
		t.Fatalf("payload nudge inesperado: %s", orders[0].PayloadJSON)
	}
}

func TestProcesarAutonomiaAgentesBatchEncolaNudgePorEsperarOPedirTarea(t *testing.T) {
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
		!strings.Contains(orders[0].PayloadJSON, `"accion":"esperar_o_pedir_tarea"`) {
		t.Fatalf("payload nudge inesperado: %s", orders[0].PayloadJSON)
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
	if n != 1 {
		t.Fatalf("esperaba 1 decision autonoma inicial, got=%d", n)
	}
	if _, err := db.ProcesarRuntimeOrdersBatch(); err != nil {
		t.Fatalf("procesar runtime orders: %v", err)
	}
	if _, err := procesarRuntimeTranscriptBatch(); err != nil {
		t.Fatalf("procesar runtime transcript: %v", err)
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
	if nudges != 1 || sendInstructions != 1 {
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

func TestProcesarRuntimeTranscriptBatchMaterializaAutonomiaMailbox(t *testing.T) {
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
	if _, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "orquesta",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "autonomia",
		PayloadJSON: `{"accion":"continuar_trabajo","motivo":"seguir el frente activo"}`,
	}); err != nil {
		t.Fatalf("crear mailbox: %v", err)
	}

	n, err := procesarRuntimeTranscriptBatch()
	if err != nil {
		t.Fatalf("procesar transcript batch: %v", err)
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
	mailboxEstado := "consumido"
	mailbox, err := db.ListarRuntimeMailbox(db.FiltroRuntimeMailbox{ToAgente: &agente, ProyectoID: &proyectoID, Estado: &mailboxEstado})
	if err != nil {
		t.Fatalf("listar mailbox: %v", err)
	}
	if len(mailbox) != 1 {
		t.Fatalf("mailbox no consumido tras materialización: %+v", mailbox)
	}
}
