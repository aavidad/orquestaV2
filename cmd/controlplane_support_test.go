package cmd

import (
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"orquesta/db"
)

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
