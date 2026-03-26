package db

import (
	"path/filepath"
	"strings"
	"testing"
)

func restringirPlanificadorATestAgentes(t *testing.T, permitidos ...string) {
	t.Helper()
	args := make([]any, 0, len(permitidos))
	placeholders := make([]string, 0, len(permitidos))
	for _, nombre := range permitidos {
		placeholders = append(placeholders, "?")
		args = append(args, nombre)
	}
	query := `UPDATE agentes SET habilitado=0 WHERE rol='programador'`
	if len(placeholders) > 0 {
		query += ` AND nombre NOT IN (` + strings.Join(placeholders, ",") + `)`
	}
	if _, err := DB.Exec(query, args...); err != nil {
		t.Fatalf("restringir agentes planificables: %v", err)
	}
}

func TestPlanificarTareasAutomaticamenteAutoasignaYEncolaStart(t *testing.T) {
	tmp := prepararDBTemporal(t)
	restringirPlanificadorATestAgentes(t, "Codex1")

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	if _, err := DB.Exec(`UPDATE agentes SET estado_sesion='disponible' WHERE nombre='Codex1'`); err != nil {
		t.Fatalf("marcar agente disponible: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if err := ActivarAsignacion("Codex1", proyectoID, "asignacion test"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	tareaID, err := CrearTarea(&Tarea{
		Titulo:      "Implementar arranque autonomo",
		Descripcion: "Cobertura del planificador",
		ProyectoID:  &proyectoID,
		Modulo:      "orquestador",
		Prioridad:   PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}

	if err := PlanificarTareasAutomaticamente(); err != nil {
		t.Fatalf("planificar: %v", err)
	}

	tarea, err := GetTarea(tareaID)
	if err != nil {
		t.Fatalf("get tarea: %v", err)
	}
	if tarea.Agente == nil || *tarea.Agente != "Codex1" || tarea.Estado != TareaAsignada {
		t.Fatalf("tarea no autoasignada correctamente: %+v", tarea)
	}

	agente := "Codex1"
	estado := "pendiente"
	orders, err := ListarRuntimeOrders(FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar runtime orders: %v", err)
	}
	if len(orders) != 1 || orders[0].Tipo != "start" {
		t.Fatalf("runtime orders inesperadas: %+v", orders)
	}
	if !strings.Contains(orders[0].PayloadJSON, `"proyecto":"orquestador"`) {
		t.Fatalf("payload start inesperado: %s", orders[0].PayloadJSON)
	}
}

func TestPlanificarTareasAutomaticamenteEncolaStartParaTrabajoYaAsignado(t *testing.T) {
	tmp := prepararDBTemporal(t)
	restringirPlanificadorATestAgentes(t, "Codex1")

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	if _, err := DB.Exec(`UPDATE agentes SET estado_sesion='disponible' WHERE nombre='Codex1'`); err != nil {
		t.Fatalf("marcar agente disponible: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if err := ActivarAsignacion("Codex1", proyectoID, "asignacion test"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	tareaID, err := CrearTarea(&Tarea{
		Titulo:      "Retomar trabajo ya asignado",
		Descripcion: "Cobertura de autoarranque",
		ProyectoID:  &proyectoID,
		Modulo:      "orquestador",
		Prioridad:   PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := TomarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}

	if err := PlanificarTareasAutomaticamente(); err != nil {
		t.Fatalf("planificar: %v", err)
	}

	agente := "Codex1"
	estado := "pendiente"
	orders, err := ListarRuntimeOrders(FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar runtime orders: %v", err)
	}
	if len(orders) != 1 || orders[0].Tipo != "start" {
		t.Fatalf("runtime orders inesperadas: %+v", orders)
	}
}

func TestPlanificarTareasAutomaticamenteNoDuplicaStartNiHandleActivo(t *testing.T) {
	tmp := prepararDBTemporal(t)
	restringirPlanificadorATestAgentes(t, "Codex1")

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	if _, err := DB.Exec(`UPDATE agentes SET estado_sesion='disponible' WHERE nombre='Codex1'`); err != nil {
		t.Fatalf("marcar agente disponible: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if err := ActivarAsignacion("Codex1", proyectoID, "asignacion test"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	tareaID, err := CrearTarea(&Tarea{
		Titulo:      "No duplicar start",
		Descripcion: "Cobertura de deduplicacion",
		ProyectoID:  &proyectoID,
		Modulo:      "orquestador",
		Prioridad:   PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := TomarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}

	if _, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "start",
		PayloadJSON: `{"proyecto":"orquestador","motivo":"preexistente"}`,
	}); err != nil {
		t.Fatalf("encolar start previo: %v", err)
	}

	if err := PlanificarTareasAutomaticamente(); err != nil {
		t.Fatalf("planificar con start pendiente: %v", err)
	}

	agente := "Codex1"
	estado := "pendiente"
	orders, err := ListarRuntimeOrders(FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar runtime orders: %v", err)
	}
	if len(orders) != 1 {
		t.Fatalf("se esperaba una sola start pendiente: %+v", orders)
	}

	if _, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_orders SET estado='completada' WHERE agente=? AND proyecto_id=? AND tipo='start'`, "Codex1", proyectoID); err != nil {
		t.Fatalf("cerrar start previa: %v", err)
	}

	if err := PlanificarTareasAutomaticamente(); err != nil {
		t.Fatalf("planificar con handle activo: %v", err)
	}

	orders, err = ListarRuntimeOrders(FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID})
	if err != nil {
		t.Fatalf("listar runtime orders final: %v", err)
	}
	var starts int
	for _, order := range orders {
		if order.Tipo == "start" {
			starts++
		}
	}
	if starts != 1 {
		t.Fatalf("no deberia duplicar starts con handle activo: %+v", orders)
	}
}

func TestPlanificarTareasAutomaticamenteNoDuplicaSiHayBootstrapPendiente(t *testing.T) {
	tmp := prepararDBTemporal(t)
	restringirPlanificadorATestAgentes(t, "Codex1")

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	if _, err := DB.Exec(`UPDATE agentes SET estado_sesion='disponible' WHERE nombre='Codex1'`); err != nil {
		t.Fatalf("marcar agente disponible: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if err := ActivarAsignacion("Codex1", proyectoID, "asignacion test"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	tareaID, err := CrearTarea(&Tarea{
		Titulo:      "No duplicar bootstrap",
		Descripcion: "Cobertura de continuidad",
		ProyectoID:  &proyectoID,
		Modulo:      "orquestador",
		Prioridad:   PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := TomarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}

	if _, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "resume",
		PayloadJSON: `{"proyecto":"orquestador","motivo":"continuidad_pendiente"}`,
	}); err != nil {
		t.Fatalf("encolar resume previo: %v", err)
	}

	if err := PlanificarTareasAutomaticamente(); err != nil {
		t.Fatalf("planificar con resume pendiente: %v", err)
	}

	agente := "Codex1"
	orders, err := ListarRuntimeOrders(FiltroRuntimeOrders{Agente: &agente})
	if err != nil {
		t.Fatalf("listar runtime orders: %v", err)
	}
	if len(orders) != 1 || orders[0].Tipo != "resume" {
		t.Fatalf("no deberia crear start si ya hay continuidad pendiente: %+v", orders)
	}
}

func TestPlanificarTareasAutomaticamenteReactivaAsignacionPausadaConTrabajo(t *testing.T) {
	tmp := prepararDBTemporal(t)
	restringirPlanificadorATestAgentes(t, "Codex1")

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	if _, err := DB.Exec(`UPDATE agentes SET estado_sesion='disponible' WHERE nombre='Codex1'`); err != nil {
		t.Fatalf("marcar agente disponible: %v", err)
	}
	proyectoA, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador-a",
		Nombre:  "Orquestador A",
		RutaAbs: filepath.Join(tmp, "orquestador-a"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto A: %v", err)
	}
	proyectoB, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador-b",
		Nombre:  "Orquestador B",
		RutaAbs: filepath.Join(tmp, "orquestador-b"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto B: %v", err)
	}
	if err := ActivarAsignacion("Codex1", proyectoA, "frente original"); err != nil {
		t.Fatalf("activar asignacion A: %v", err)
	}
	if err := PausarAsignacion("Codex1", proyectoA, "esperando humano"); err != nil {
		t.Fatalf("pausar asignacion A: %v", err)
	}
	if err := ActivarAsignacion("Codex1", proyectoB, "frente temporal"); err != nil {
		t.Fatalf("activar asignacion B: %v", err)
	}
	tareaID, err := CrearTarea(&Tarea{
		Titulo:      "Retomar frente pausado",
		Descripcion: "El proyecto A vuelve a tener trabajo",
		ProyectoID:  &proyectoA,
		Modulo:      "core",
		Prioridad:   PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}

	if err := PlanificarTareasAutomaticamente(); err != nil {
		t.Fatalf("planificar: %v", err)
	}

	proyectoActivoID, err := ObtenerProyectoActivoAgente("Codex1")
	if err != nil {
		t.Fatalf("obtener proyecto activo: %v", err)
	}
	if proyectoActivoID != proyectoA {
		t.Fatalf("deberia haber reactivado el proyecto A, got=%d want=%d", proyectoActivoID, proyectoA)
	}
	tarea, err := GetTarea(tareaID)
	if err != nil {
		t.Fatalf("get tarea: %v", err)
	}
	if tarea.Agente == nil || *tarea.Agente != "Codex1" || tarea.Estado != TareaAsignada {
		t.Fatalf("tarea no reasignada al frente reactivado: %+v", tarea)
	}
	agente := "Codex1"
	asignaciones, err := ListarAsignaciones(FiltroAsignaciones{Agente: &agente})
	if err != nil {
		t.Fatalf("listar asignaciones: %v", err)
	}
	if len(asignaciones) < 2 {
		t.Fatalf("asignaciones insuficientes: %+v", asignaciones)
	}
	if asignaciones[0].ProyectoID != proyectoA || asignaciones[0].Estado != AsignacionActiva {
		t.Fatalf("asignacion A no reactivada: %+v", asignaciones)
	}
	if asignaciones[1].ProyectoID != proyectoB || asignaciones[1].Estado != AsignacionPausada {
		t.Fatalf("asignacion B no pausada tras retorno: %+v", asignaciones)
	}
}

func TestPlanificarTareasAutomaticamenteNoArrancaSiHaySesionActivaEnOtroProyecto(t *testing.T) {
	tmp := prepararDBTemporal(t)
	restringirPlanificadorATestAgentes(t, "Codex1")

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	if _, err := DB.Exec(`UPDATE agentes SET estado_sesion='disponible' WHERE nombre='Codex1'`); err != nil {
		t.Fatalf("marcar agente disponible: %v", err)
	}
	proyectoA, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador-a",
		Nombre:  "Orquestador A",
		RutaAbs: filepath.Join(tmp, "orquestador-a"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto A: %v", err)
	}
	proyectoB, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador-b",
		Nombre:  "Orquestador B",
		RutaAbs: filepath.Join(tmp, "orquestador-b"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto B: %v", err)
	}
	if err := ActivarAsignacion("Codex1", proyectoA, "asignacion principal"); err != nil {
		t.Fatalf("activar asignacion A: %v", err)
	}
	tareaID, err := CrearTarea(&Tarea{
		Titulo:      "No desalojar sesion de otro proyecto",
		Descripcion: "Cobertura cross-project",
		ProyectoID:  &proyectoA,
		Modulo:      "orquestador",
		Prioridad:   PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := TomarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}

	if _, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoB,
		CWD:         filepath.Join(tmp, "orquestador-b"),
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion otro proyecto: %v", err)
	}

	if err := PlanificarTareasAutomaticamente(); err != nil {
		t.Fatalf("planificar con sesion activa en otro proyecto: %v", err)
	}

	agente := "Codex1"
	orders, err := ListarRuntimeOrders(FiltroRuntimeOrders{Agente: &agente})
	if err != nil {
		t.Fatalf("listar runtime orders: %v", err)
	}
	if len(orders) != 0 {
		t.Fatalf("no deberia crear start con sesion activa en otro proyecto: %+v", orders)
	}
}

func TestPlanificarTareasAutomaticamenteIncluyeAgenteRecienRegistradoSinSesion(t *testing.T) {
	tmp := prepararDBTemporal(t)
	restringirPlanificadorATestAgentes(t, "CodexNuevo")

	if err := RegistrarAgente("CodexNuevo", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "app-demo",
		Nombre:  "App Demo",
		RutaAbs: filepath.Join(tmp, "app-demo"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if err := ActivarAsignacion("CodexNuevo", proyectoID, "onboarding automatico"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	tareaID, err := CrearTarea(&Tarea{
		Titulo:      "Primera tarea autonoma",
		Descripcion: "Debe entrar en la rueda sin sesion previa",
		ProyectoID:  &proyectoID,
		Modulo:      "bootstrap",
		Prioridad:   PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}

	if err := PlanificarTareasAutomaticamente(); err != nil {
		t.Fatalf("planificar: %v", err)
	}

	tarea, err := GetTarea(tareaID)
	if err != nil {
		t.Fatalf("get tarea: %v", err)
	}
	if tarea.Agente == nil || *tarea.Agente != "CodexNuevo" || tarea.Estado != TareaAsignada {
		t.Fatalf("tarea no autoasignada al agente nuevo: %+v", tarea)
	}

	agente := "CodexNuevo"
	estado := "pendiente"
	orders, err := ListarRuntimeOrders(FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar runtime orders: %v", err)
	}
	if len(orders) != 1 || orders[0].Tipo != "start" {
		t.Fatalf("start no encolada para agente nuevo: %+v", orders)
	}
}

func TestPlanificarTareasAutomaticamenteNoSeleccionaAgenteEnEnfriamiento(t *testing.T) {
	tmp := prepararDBTemporal(t)
	restringirPlanificadorATestAgentes(t, "CodexPausado")

	if err := RegistrarAgente("CodexPausado", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	if _, err := DB.Exec(`UPDATE agentes SET estado_cuota='enfriamiento', reanimar_at=CURRENT_TIMESTAMP WHERE nombre='CodexPausado'`); err != nil {
		t.Fatalf("marcar enfriamiento: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "app-enfriada",
		Nombre:  "App Enfriada",
		RutaAbs: filepath.Join(tmp, "app-enfriada"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if err := ActivarAsignacion("CodexPausado", proyectoID, "pausa por cuota"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	tareaID, err := CrearTarea(&Tarea{
		Titulo:      "No arrancar en enfriamiento",
		Descripcion: "Debe esperar a reanimacion",
		ProyectoID:  &proyectoID,
		Modulo:      "scheduler",
		Prioridad:   PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}

	if err := PlanificarTareasAutomaticamente(); err != nil {
		t.Fatalf("planificar: %v", err)
	}

	tarea, err := GetTarea(tareaID)
	if err != nil {
		t.Fatalf("get tarea: %v", err)
	}
	if tarea.Agente != nil || tarea.Estado != TareaLibre {
		t.Fatalf("la tarea no deberia asignarse mientras el agente esta enfriando: %+v", tarea)
	}

	agente := "CodexPausado"
	orders, err := ListarRuntimeOrders(FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID})
	if err != nil {
		t.Fatalf("listar runtime orders: %v", err)
	}
	if len(orders) != 0 {
		t.Fatalf("no deberia crear runtime orders para agente en enfriamiento: %+v", orders)
	}
}

func TestResolverProyectoPlanificableAgenteReactivaProyectoTrasDesbloqueoHumano(t *testing.T) {
	tmp := prepararDBTemporal(t)
	restringirPlanificadorATestAgentes(t, "Codex1")

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	if _, err := DB.Exec(`UPDATE agentes SET estado_sesion='disponible' WHERE nombre='Codex1'`); err != nil {
		t.Fatalf("marcar agente disponible: %v", err)
	}
	proyectoA, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador-a",
		Nombre:  "Orquestador A",
		RutaAbs: filepath.Join(tmp, "orquestador-a"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto A: %v", err)
	}
	proyectoB, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador-b",
		Nombre:  "Orquestador B",
		RutaAbs: filepath.Join(tmp, "orquestador-b"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto B: %v", err)
	}
	if err := ActivarAsignacion("Codex1", proyectoA, "frente original"); err != nil {
		t.Fatalf("activar asignacion A: %v", err)
	}
	tareaA, err := CrearTarea(&Tarea{
		Titulo:      "Esperando desbloqueo humano",
		Descripcion: "Bloqueo del frente A",
		ProyectoID:  &proyectoA,
		Modulo:      "core",
		Prioridad:   PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea A: %v", err)
	}
	if err := TomarTarea(tareaA, "Codex1"); err != nil {
		t.Fatalf("tomar tarea A: %v", err)
	}
	if err := IniciarTarea(tareaA, "Codex1"); err != nil {
		t.Fatalf("iniciar tarea A: %v", err)
	}
	if err := BloquearTarea(tareaA, "Codex1", "esperando respuesta humana"); err != nil {
		t.Fatalf("bloquear tarea A: %v", err)
	}
	if err := MarcarProyectoEsperandoHumano(proyectoA, "esperando respuesta humana"); err != nil {
		t.Fatalf("marcar proyecto esperando humano: %v", err)
	}
	if err := PausarAsignacion("Codex1", proyectoA, "bloqueo_humano"); err != nil {
		t.Fatalf("pausar asignacion A: %v", err)
	}
	if err := ActivarAsignacion("Codex1", proyectoB, "frente temporal"); err != nil {
		t.Fatalf("activar asignacion B: %v", err)
	}
	if err := DesbloquearTarea(tareaA, "alberto", "respuesta recibida"); err != nil {
		t.Fatalf("desbloquear tarea A: %v", err)
	}

	proyectoPlanificable, err := ResolverProyectoPlanificableAgente("Codex1")
	if err != nil {
		t.Fatalf("resolver proyecto planificable: %v", err)
	}
	if proyectoPlanificable != proyectoA {
		t.Fatalf("deberia volver al proyecto A tras desbloqueo, got=%d want=%d", proyectoPlanificable, proyectoA)
	}

	asignacionActiva, err := GetAsignacionActivaAgente("Codex1")
	if err != nil {
		t.Fatalf("get asignacion activa: %v", err)
	}
	if asignacionActiva == nil || asignacionActiva.ProyectoID != proyectoA {
		t.Fatalf("asignacion activa inesperada: %+v", asignacionActiva)
	}
	op, err := GetProyectoOperacion(proyectoA)
	if err != nil {
		t.Fatalf("get proyecto operacion: %v", err)
	}
	if op.EstadoOperativo != ProyectoOperativoActivo {
		t.Fatalf("el proyecto A deberia quedar reactivado automaticamente: %+v", op)
	}
}

func TestResolverProyectoPlanificableAgenteAsignaSegunPoliticaOperativaProyecto(t *testing.T) {
	tmp := prepararDBTemporal(t)
	restringirPlanificadorATestAgentes(t, "CodexLibre", "CodexOcupado")

	if err := RegistrarAgente("CodexLibre", "programador"); err != nil {
		t.Fatalf("registrar agente libre: %v", err)
	}
	if err := RegistrarAgente("CodexOcupado", "programador"); err != nil {
		t.Fatalf("registrar agente ocupado: %v", err)
	}
	if _, err := DB.Exec(`UPDATE agentes SET estado_sesion='disponible' WHERE nombre IN ('CodexLibre','CodexOcupado')`); err != nil {
		t.Fatalf("marcar agentes disponibles: %v", err)
	}

	proyectoA, err := UpsertProyecto(&Proyecto{
		Slug:    "proyecto-a",
		Nombre:  "Proyecto A",
		RutaAbs: filepath.Join(tmp, "proyecto-a"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto A: %v", err)
	}
	proyectoB, err := UpsertProyecto(&Proyecto{
		Slug:    "proyecto-b",
		Nombre:  "Proyecto B",
		RutaAbs: filepath.Join(tmp, "proyecto-b"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto B: %v", err)
	}
	if err := UpsertProyectoOperacion(&ProyectoOperacion{
		ProyectoID:       proyectoA,
		EstadoOperativo:  ProyectoOperativoActivo,
		ObjetivoPct:      80,
		Prioridad:        200,
		ResumeAutomatico: true,
	}); err != nil {
		t.Fatalf("upsert proyecto operacion A: %v", err)
	}
	if err := UpsertProyectoOperacion(&ProyectoOperacion{
		ProyectoID:       proyectoB,
		EstadoOperativo:  ProyectoOperativoActivo,
		ObjetivoPct:      20,
		Prioridad:        100,
		ResumeAutomatico: true,
	}); err != nil {
		t.Fatalf("upsert proyecto operacion B: %v", err)
	}
	if err := ActivarAsignacion("CodexOcupado", proyectoB, "ocupado en B"); err != nil {
		t.Fatalf("activar asignacion B: %v", err)
	}
	if _, err := CrearTarea(&Tarea{
		Titulo:      "Trabajo A",
		Descripcion: "Proyecto priorizado",
		ProyectoID:  &proyectoA,
		Modulo:      "core",
		Prioridad:   PrioridadAlta,
		CreadoPor:   "alberto",
	}); err != nil {
		t.Fatalf("crear tarea A: %v", err)
	}
	if _, err := CrearTarea(&Tarea{
		Titulo:      "Trabajo B",
		Descripcion: "Proyecto menos prioritario",
		ProyectoID:  &proyectoB,
		Modulo:      "core",
		Prioridad:   PrioridadAlta,
		CreadoPor:   "alberto",
	}); err != nil {
		t.Fatalf("crear tarea B: %v", err)
	}

	proyectoPlanificable, err := ResolverProyectoPlanificableAgente("CodexLibre")
	if err != nil {
		t.Fatalf("resolver proyecto planificable: %v", err)
	}
	if proyectoPlanificable != proyectoA {
		t.Fatalf("deberia elegir el proyecto A por politica operativa, got=%d want=%d", proyectoPlanificable, proyectoA)
	}
	asignacionActiva, err := GetAsignacionActivaAgente("CodexLibre")
	if err != nil {
		t.Fatalf("get asignacion activa libre: %v", err)
	}
	if asignacionActiva == nil || asignacionActiva.ProyectoID != proyectoA {
		t.Fatalf("asignacion activa inesperada tras politicas: %+v", asignacionActiva)
	}
}

func TestResolverProyectoPlanificableAgenteNoReactivaProyectoSinResumeAutomatico(t *testing.T) {
	tmp := prepararDBTemporal(t)
	restringirPlanificadorATestAgentes(t, "Codex1")

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	if _, err := DB.Exec(`UPDATE agentes SET estado_sesion='disponible' WHERE nombre='Codex1'`); err != nil {
		t.Fatalf("marcar agente disponible: %v", err)
	}
	proyectoA, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador-a",
		Nombre:  "Orquestador A",
		RutaAbs: filepath.Join(tmp, "orquestador-a"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto A: %v", err)
	}
	proyectoB, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador-b",
		Nombre:  "Orquestador B",
		RutaAbs: filepath.Join(tmp, "orquestador-b"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto B: %v", err)
	}
	if err := UpsertProyectoOperacion(&ProyectoOperacion{
		ProyectoID:       proyectoA,
		EstadoOperativo:  ProyectoOperativoEsperandoHumano,
		Motivo:           "esperando respuesta humana",
		ObjetivoPct:      100,
		Prioridad:        300,
		ResumeAutomatico: false,
	}); err != nil {
		t.Fatalf("upsert proyecto operacion A: %v", err)
	}
	if err := ActivarAsignacion("Codex1", proyectoA, "frente original"); err != nil {
		t.Fatalf("activar asignacion A: %v", err)
	}
	tareaA, err := CrearTarea(&Tarea{
		Titulo:      "Esperando desbloqueo humano",
		Descripcion: "Bloqueo del frente A",
		ProyectoID:  &proyectoA,
		Modulo:      "core",
		Prioridad:   PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea A: %v", err)
	}
	if err := TomarTarea(tareaA, "Codex1"); err != nil {
		t.Fatalf("tomar tarea A: %v", err)
	}
	if err := IniciarTarea(tareaA, "Codex1"); err != nil {
		t.Fatalf("iniciar tarea A: %v", err)
	}
	if err := BloquearTarea(tareaA, "Codex1", "esperando respuesta humana"); err != nil {
		t.Fatalf("bloquear tarea A: %v", err)
	}
	if err := PausarAsignacion("Codex1", proyectoA, "bloqueo_humano"); err != nil {
		t.Fatalf("pausar asignacion A: %v", err)
	}
	if err := ActivarAsignacion("Codex1", proyectoB, "frente temporal"); err != nil {
		t.Fatalf("activar asignacion B: %v", err)
	}
	if _, err := CrearTarea(&Tarea{
		Titulo:      "Trabajo B",
		Descripcion: "frente temporal",
		ProyectoID:  &proyectoB,
		Modulo:      "core",
		Prioridad:   PrioridadMedia,
		CreadoPor:   "alberto",
	}); err != nil {
		t.Fatalf("crear tarea B: %v", err)
	}
	if err := DesbloquearTarea(tareaA, "alberto", "respuesta recibida"); err != nil {
		t.Fatalf("desbloquear tarea A: %v", err)
	}

	proyectoPlanificable, err := ResolverProyectoPlanificableAgente("Codex1")
	if err != nil {
		t.Fatalf("resolver proyecto planificable: %v", err)
	}
	if proyectoPlanificable != proyectoB {
		t.Fatalf("no deberia reactivar automaticamente el proyecto A, got=%d want=%d", proyectoPlanificable, proyectoB)
	}

	asignacionActiva, err := GetAsignacionActivaAgente("Codex1")
	if err != nil {
		t.Fatalf("get asignacion activa: %v", err)
	}
	if asignacionActiva == nil || asignacionActiva.ProyectoID != proyectoB {
		t.Fatalf("asignacion activa inesperada: %+v", asignacionActiva)
	}
	op, err := GetProyectoOperacion(proyectoA)
	if err != nil {
		t.Fatalf("get proyecto operacion: %v", err)
	}
	if op.EstadoOperativo != ProyectoOperativoEsperandoHumano {
		t.Fatalf("el proyecto A no deberia reactivarse automaticamente: %+v", op)
	}
}
