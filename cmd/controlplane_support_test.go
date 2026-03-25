package cmd

import (
	"path/filepath"
	"strings"
	"testing"

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
