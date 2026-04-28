package cmd

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"orquesta/agentesapp"
	"orquesta/db"
)

func TestProcesarAutoasignacionPremiumSinSesionBatchReutilizaProyectoPausadoReciente(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Gemini1", "programador"); err != nil {
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
	if err := db.UpsertProyectoOperacion(&db.ProyectoOperacion{
		ProyectoID:       proyectoID,
		EstadoOperativo:  db.ProyectoOperativoActivo,
		Motivo:           "microrefactor_loop",
		ObjetivoPct:      100,
		MinAgentes:       1,
		MaxAgentes:       3,
		ResumeAutomatico: true,
	}); err != nil {
		t.Fatalf("upsert proyecto operacion: %v", err)
	}
	if err := db.ActivarAsignacion("Gemini1", proyectoID, "reactivacion_automatica_trabajo_activo"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	if err := db.PausarAsignacion("Gemini1", proyectoID, "sin_trabajo_reactivacion_automatica"); err != nil {
		t.Fatalf("pausar asignacion: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      microcicloRefactorTituloDefault,
		Descripcion: descripcionMicrocicloDefault(&db.Proyecto{Slug: "orquestador", RutaAbs: filepath.Join(tmp, "orquestador")}),
		ProyectoID:  &proyectoID,
		Modulo:      "controlplane",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "orquesta",
		Notas:       microcicloRefactorNotasTag,
	})
	if err != nil {
		t.Fatalf("crear tarea libre: %v", err)
	}

	n, err := procesarAutoasignacionPremiumSinSesionBatch([]agentesapp.Row{{
		Agente:          &db.Agente{Nombre: "Gemini1"},
		EstadoOperativo: "sin_tarea",
	}}, map[string][]*db.Tarea{})
	if err != nil {
		t.Fatalf("procesarAutoasignacionPremiumSinSesionBatch: %v", err)
	}
	if n < 1 {
		t.Fatalf("deberia autoasignar premium idle sin sesion desde proyecto pausado, got=%d", n)
	}

	tarea, err := db.GetTarea(tareaID)
	if err != nil {
		t.Fatalf("get tarea: %v", err)
	}
	if tarea == nil || tarea.Agente == nil || strings.TrimSpace(*tarea.Agente) != "Gemini1" || tarea.Estado != db.TareaEnProgreso {
		t.Fatalf("la tarea libre deberia quedar tomada y arrancada en Gemini1: %+v", tarea)
	}

	agente := "Gemini1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	if len(orders) != 1 || orders[0].Tipo != "start" {
		t.Fatalf("deberia encolar start para premium idle sin sesion: %+v", orders)
	}
	if !strings.Contains(orders[0].PayloadJSON, `"motivo":"premium_idle_autoassigned"`) {
		t.Fatalf("start sin motivo esperado: %s", orders[0].PayloadJSON)
	}
}

func TestProcesarAutoasignacionPremiumSinSesionBatchUsaAsignacionActivaSinRuntime(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Gemini1", "programador"); err != nil {
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
	if err := db.UpsertProyectoOperacion(&db.ProyectoOperacion{
		ProyectoID:       proyectoID,
		EstadoOperativo:  db.ProyectoOperativoActivo,
		Motivo:           "microrefactor_loop",
		ObjetivoPct:      100,
		MinAgentes:       1,
		MaxAgentes:       3,
		ResumeAutomatico: true,
	}); err != nil {
		t.Fatalf("upsert proyecto operacion: %v", err)
	}
	if err := db.ActivarAsignacion("Gemini1", proyectoID, "reactivacion_automatica"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Runtime mailbox/session_resume",
		Descripcion: "Write-set exclusivo: cmd/controlplane_support.go\nTests minimos: go test ./cmd -run 'TestProcesarRuntimeMailboxSessionResumeBatch.*' -count=1",
		ProyectoID:  &proyectoID,
		Modulo:      "controlplane",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "orquesta",
		Notas:       "autonomia:microrefactor_loop",
	})
	if err != nil {
		t.Fatalf("crear tarea libre: %v", err)
	}
	asignacion, err := db.GetAsignacionActivaAgente("Gemini1")
	if err != nil {
		t.Fatalf("get asignacion activa: %v", err)
	}

	n, err := procesarAutoasignacionPremiumSinSesionBatch([]agentesapp.Row{{
		Agente:          &db.Agente{Nombre: "Gemini1"},
		Asignacion:      asignacion,
		EstadoOperativo: "sin_tarea",
	}}, map[string][]*db.Tarea{})
	if err != nil {
		t.Fatalf("procesarAutoasignacionPremiumSinSesionBatch: %v", err)
	}
	if n < 1 {
		t.Fatalf("deberia autoasignar premium idle desde asignacion activa sin runtime, got=%d", n)
	}

	tarea, err := db.GetTarea(tareaID)
	if err != nil {
		t.Fatalf("get tarea: %v", err)
	}
	if tarea == nil || tarea.Agente == nil || strings.TrimSpace(*tarea.Agente) != "Gemini1" || tarea.Estado != db.TareaEnProgreso {
		t.Fatalf("la tarea libre deberia quedar tomada y arrancada en Gemini1: %+v", tarea)
	}

	agente := "Gemini1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	if len(orders) != 1 || orders[0].Tipo != "start" {
		t.Fatalf("deberia encolar start para premium idle desde asignacion activa: %+v", orders)
	}
	if !strings.Contains(orders[0].PayloadJSON, `"motivo":"premium_idle_autoassigned"`) {
		t.Fatalf("start sin motivo esperado: %s", orders[0].PayloadJSON)
	}
}

func TestProcesarAutoasignacionPremiumSinSesionBatchNoBloqueaPorHandleActivaStale(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Gemini1", "programador"); err != nil {
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
	if err := db.UpsertProyectoOperacion(&db.ProyectoOperacion{
		ProyectoID:       proyectoID,
		EstadoOperativo:  db.ProyectoOperativoActivo,
		Motivo:           "microrefactor_loop",
		ObjetivoPct:      100,
		MinAgentes:       1,
		MaxAgentes:       3,
		ResumeAutomatico: true,
	}); err != nil {
		t.Fatalf("upsert proyecto operacion: %v", err)
	}
	if err := db.ActivarAsignacion("Gemini1", proyectoID, "reactivacion_automatica"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Retomar frente premium con handle stale",
		Descripcion: "El batch no debe tratar una handle antigua como runtime util.",
		ProyectoID:  &proyectoID,
		Modulo:      "controlplane",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "orquesta",
	})
	if err != nil {
		t.Fatalf("crear tarea libre: %v", err)
	}
	asignacion, err := db.GetAsignacionActivaAgente("Gemini1")
	if err != nil {
		t.Fatalf("get asignacion activa: %v", err)
	}

	n, err := procesarAutoasignacionPremiumSinSesionBatch([]agentesapp.Row{{
		Agente:          &db.Agente{Nombre: "Gemini1"},
		Asignacion:      asignacion,
		EstadoOperativo: "sin_tarea",
		Handle: &db.RuntimeHandle{
			ID:         91,
			Agente:     "Gemini1",
			ProyectoID: &proyectoID,
			Estado:     "activo",
			UpdatedAt:  time.Now().UTC().Add(-20 * time.Minute),
		},
	}}, map[string][]*db.Tarea{})
	if err != nil {
		t.Fatalf("procesarAutoasignacionPremiumSinSesionBatch: %v", err)
	}
	if n < 1 {
		t.Fatalf("deberia ignorar handle stale y autoasignar igual, got=%d", n)
	}

	tarea, err := db.GetTarea(tareaID)
	if err != nil {
		t.Fatalf("get tarea: %v", err)
	}
	if tarea == nil || tarea.Agente == nil || strings.TrimSpace(*tarea.Agente) != "Gemini1" || tarea.Estado != db.TareaEnProgreso {
		t.Fatalf("la tarea libre deberia quedar tomada y arrancada en Gemini1: %+v", tarea)
	}
}

func TestResolverProyectoAutoasignacionPremiumSinSesionUsaAutobootstrapParaSupervisor(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.ConfigSet("server_autobootstrap_enabled", "true"); err != nil {
		t.Fatalf("config enabled: %v", err)
	}
	if err := db.ConfigSet("server_autobootstrap_project_slug", "orquestador"); err != nil {
		t.Fatalf("config slug: %v", err)
	}
	if err := db.ConfigSet("server_autobootstrap_project_name", "Orquestador"); err != nil {
		t.Fatalf("config name: %v", err)
	}
	if err := db.ConfigSet("server_autobootstrap_project_path", filepath.Join(tmp, "orquestador")); err != nil {
		t.Fatalf("config path: %v", err)
	}
	if err := db.ConfigSet("server_autobootstrap_supervisor_agent", "Codex1"); err != nil {
		t.Fatalf("config supervisor: %v", err)
	}
	if err := db.ConfigSet("server_autobootstrap_worker_agents", "Codex2,Codex3,Codex4"); err != nil {
		t.Fatalf("config workers: %v", err)
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
		MaxWorkers:       3,
		SupervisorAgente: "Codex1",
		EstadoAutonomia:  db.AutonomiaProyectoActiva,
	}); err != nil {
		t.Fatalf("upsert autonomia: %v", err)
	}

	proyecto, err := resolverProyectoAutoasignacionPremiumSinSesion("Codex1", agentesapp.Row{
		Agente:          &db.Agente{Nombre: "Codex1"},
		EstadoOperativo: "disponible",
	})
	if err != nil {
		t.Fatalf("resolver proyecto: %v", err)
	}
	if proyecto == nil || proyecto.ID != proyectoID {
		t.Fatalf("deberia resolver proyecto autobootstrap para supervisor, got=%+v", proyecto)
	}
}

func TestProcesarAutoasignacionPremiumSinSesionBatchAbreMicrotareaParaAsignacionActivaSinRuntimeNiTareaActiva(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("GeminiSeed", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador-seed-idle-direct",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if err := db.UpsertProyectoOperacion(&db.ProyectoOperacion{
		ProyectoID:       proyectoID,
		EstadoOperativo:  db.ProyectoOperativoActivo,
		Motivo:           "microrefactor_loop",
		ObjetivoPct:      100,
		MinAgentes:       1,
		MaxAgentes:       3,
		ResumeAutomatico: true,
	}); err != nil {
		t.Fatalf("upsert proyecto operacion: %v", err)
	}
	if _, err := db.UpsertProyectoAutonomia(&db.ProyectoAutonomia{
		ProyectoID:       proyectoID,
		Enabled:          true,
		MaxWorkers:       1,
		SupervisorAgente: "orquesta",
		EstadoAutonomia:  db.AutonomiaProyectoActiva,
	}); err != nil {
		t.Fatalf("upsert autonomia: %v", err)
	}
	if err := db.ActivarAsignacion("GeminiSeed", proyectoID, "frente nucleo"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	asignacion, err := db.GetAsignacionActivaAgente("GeminiSeed")
	if err != nil {
		t.Fatalf("get asignacion activa: %v", err)
	}

	n, err := procesarAutoasignacionPremiumSinSesionBatch([]agentesapp.Row{{
		Agente:          &db.Agente{Nombre: "GeminiSeed"},
		Asignacion:      asignacion,
		EstadoOperativo: "sin_tarea",
	}}, map[string][]*db.Tarea{})
	if err != nil {
		t.Fatalf("procesarAutoasignacionPremiumSinSesionBatch: %v", err)
	}
	if n < 1 {
		t.Fatalf("deberia abrir backlog premium util para asignado sin tarea activa, got=%d", n)
	}

	tareas, err := db.ListarTareas(db.FiltroTareas{ProyectoID: &proyectoID})
	if err != nil {
		t.Fatalf("listar tareas: %v", err)
	}
	var abierta *db.Tarea
	for _, tarea := range tareas {
		if tarea == nil {
			continue
		}
		if tarea.Agente != nil && strings.TrimSpace(*tarea.Agente) == "GeminiSeed" && strings.Contains(strings.TrimSpace(tarea.Notas), microcicloRefactorNotasTag) {
			abierta = tarea
			break
		}
	}
	if abierta == nil {
		t.Fatalf("deberia dejar una microtarea premium activa para GeminiSeed, tareas=%+v", tareas)
	}
	if abierta.Estado != db.TareaAsignada && abierta.Estado != db.TareaEnProgreso {
		t.Fatalf("la microtarea premium deberia quedar activa para GeminiSeed: %+v", abierta)
	}
	if (!strings.Contains(abierta.Descripcion, "WRITE_SET:") && !strings.Contains(abierta.Descripcion, "Write-set exclusivo:")) ||
		(!strings.Contains(abierta.Descripcion, "Tests minimos:") && !strings.Contains(abierta.Descripcion, "Tests minimos del slice:")) {
		t.Fatalf("la microtarea premium deberia nacer con contrato accionable: %+v", abierta)
	}
	if !strings.Contains(abierta.Descripcion, "cmd/controlplane_support.go") ||
		!strings.Contains(abierta.Descripcion, "db/controlplane_entities.go") ||
		!strings.Contains(abierta.Descripcion, "runtimeagente/driver.go") ||
		!strings.Contains(abierta.Descripcion, "TestResolverBootstrapRuntimeLeasePendiente") {
		t.Fatalf("la microtarea premium deberia apuntar al write-set y tests del nucleo: %+v", abierta)
	}

	agente := "GeminiSeed"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	if len(orders) != 1 || orders[0].Tipo != "start" {
		t.Fatalf("deberia encolar start para backlog premium generado, got=%+v", orders)
	}
	if !strings.Contains(orders[0].PayloadJSON, `"motivo":"premium_idle_autoassigned"`) {
		t.Fatalf("start sin motivo esperado: %s", orders[0].PayloadJSON)
	}
}

func TestProcesarAutoasignacionPremiumSinSesionBatchOmiteAgenteRetirado(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("GeminiRetirado", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	if err := db.RetirarAgente("GeminiRetirado"); err != nil {
		t.Fatalf("retirar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador-retirado",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if err := db.UpsertProyectoOperacion(&db.ProyectoOperacion{
		ProyectoID:       proyectoID,
		EstadoOperativo:  db.ProyectoOperativoActivo,
		Motivo:           "microrefactor_loop",
		ObjetivoPct:      100,
		MinAgentes:       1,
		MaxAgentes:       3,
		ResumeAutomatico: true,
	}); err != nil {
		t.Fatalf("upsert proyecto operacion: %v", err)
	}
	if err := db.ActivarAsignacion("GeminiRetirado", proyectoID, "frente premium"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      microcicloRefactorTituloDefault,
		Descripcion: descripcionMicrocicloDefault(&db.Proyecto{Slug: "orquestador-retirado", RutaAbs: filepath.Join(tmp, "orquestador")}),
		ProyectoID:  &proyectoID,
		Modulo:      "controlplane",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "orquesta",
		Notas:       microcicloRefactorNotasTag,
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}

	n, err := procesarAutoasignacionPremiumSinSesionBatch([]agentesapp.Row{{
		Agente:           &db.Agente{Nombre: "GeminiRetirado", Habilitado: false},
		EstadoOperativo:  "retirado",
		DetalleOperativo: "agente retirado",
	}}, map[string][]*db.Tarea{})
	if err != nil {
		t.Fatalf("procesarAutoasignacionPremiumSinSesionBatch: %v", err)
	}
	if n != 0 {
		t.Fatalf("no deberia procesar agente retirado, got=%d", n)
	}

	tarea, err := db.GetTarea(tareaID)
	if err != nil {
		t.Fatalf("get tarea: %v", err)
	}
	if tarea == nil || tarea.Agente != nil || tarea.Estado != db.TareaLibre {
		t.Fatalf("la tarea no deberia tocarse para agente retirado: %+v", tarea)
	}

	agente := "GeminiRetirado"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	if len(orders) != 0 {
		t.Fatalf("no deberia encolar start para agente retirado: %+v", orders)
	}
}

func TestProcesarAutoasignacionPremiumSinSesionBatchRespetaCooldownReactivacionReciente(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("GeminiCooldown", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador-cooldown",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if err := db.UpsertProyectoOperacion(&db.ProyectoOperacion{
		ProyectoID:       proyectoID,
		EstadoOperativo:  db.ProyectoOperativoActivo,
		Motivo:           "microrefactor_loop",
		ObjetivoPct:      100,
		MinAgentes:       1,
		MaxAgentes:       3,
		ResumeAutomatico: true,
	}); err != nil {
		t.Fatalf("upsert proyecto operacion: %v", err)
	}
	if err := db.ActivarAsignacion("GeminiCooldown", proyectoID, "frente premium"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      microcicloRefactorTituloDefault,
		Descripcion: descripcionMicrocicloDefault(&db.Proyecto{Slug: "orquestador-cooldown", RutaAbs: filepath.Join(tmp, "orquestador")}),
		ProyectoID:  &proyectoID,
		Modulo:      "controlplane",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "orquesta",
		Notas:       microcicloRefactorNotasTag,
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	payload := `{"accion":"start","motivo":"premium_idle_autoassigned"}`
	if _, err := db.DB.Exec(`
		INSERT INTO runtime_orders (agente, proyecto_id, tipo, estado, payload_json, created_at, updated_at)
		VALUES (?, ?, 'start', 'fallida', ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, "GeminiCooldown", proyectoID, payload); err != nil {
		t.Fatalf("insert runtime order reciente: %v", err)
	}

	n, err := procesarAutoasignacionPremiumSinSesionBatch([]agentesapp.Row{{
		Agente:          &db.Agente{Nombre: "GeminiCooldown", Habilitado: true},
		EstadoOperativo: "sin_tarea",
	}}, map[string][]*db.Tarea{})
	if err != nil {
		t.Fatalf("procesarAutoasignacionPremiumSinSesionBatch: %v", err)
	}
	if n != 0 {
		t.Fatalf("deberia respetar cooldown de reactivacion reciente, got=%d", n)
	}

	tarea, err := db.GetTarea(tareaID)
	if err != nil {
		t.Fatalf("get tarea: %v", err)
	}
	if tarea == nil || tarea.Agente != nil || tarea.Estado != db.TareaLibre {
		t.Fatalf("la tarea no deberia tocarse durante cooldown: %+v", tarea)
	}

	agente := "GeminiCooldown"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	if len(orders) != 0 {
		t.Fatalf("no deberia encolar start nuevo durante cooldown: %+v", orders)
	}
}

func TestResolverProyectoAutoasignacionPremiumSinSesionPrefiereProyectoPremiumSobrePausaMasReciente(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Gemini1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoPremiumID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto premium: %v", err)
	}
	if err := db.UpsertProyectoOperacion(&db.ProyectoOperacion{
		ProyectoID:       proyectoPremiumID,
		EstadoOperativo:  db.ProyectoOperativoActivo,
		Motivo:           "microrefactor_loop",
		ObjetivoPct:      100,
		MinAgentes:       1,
		MaxAgentes:       3,
		ResumeAutomatico: true,
	}); err != nil {
		t.Fatalf("upsert proyecto operacion premium: %v", err)
	}
	proyectoAPIID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "api",
		Nombre:  "API",
		RutaAbs: filepath.Join(tmp, "api"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto api: %v", err)
	}
	if err := db.ActivarAsignacion("Gemini1", proyectoPremiumID, "reactivacion_automatica_trabajo_activo"); err != nil {
		t.Fatalf("activar asignacion premium: %v", err)
	}
	if err := db.PausarAsignacion("Gemini1", proyectoPremiumID, "sin_trabajo_reactivacion_automatica"); err != nil {
		t.Fatalf("pausar asignacion premium: %v", err)
	}
	if err := db.ActivarAsignacion("Gemini1", proyectoAPIID, "reactivacion_automatica_trabajo_activo"); err != nil {
		t.Fatalf("activar asignacion api: %v", err)
	}
	if err := db.PausarAsignacion("Gemini1", proyectoAPIID, "sin_trabajo_espera_automatica"); err != nil {
		t.Fatalf("pausar asignacion api: %v", err)
	}

	proyecto, err := resolverProyectoAutoasignacionPremiumSinSesion("Gemini1", agentesapp.Row{
		Agente:          &db.Agente{Nombre: "Gemini1"},
		EstadoOperativo: "sin_tarea",
	})
	if err != nil {
		t.Fatalf("resolverProyectoAutoasignacionPremiumSinSesion: %v", err)
	}
	if proyecto == nil || proyecto.ID != proyectoPremiumID {
		t.Fatalf("deberia preferir el proyecto premium con continuidad frente a la pausa mas reciente: %+v", proyecto)
	}
}

func TestProcesarAutoasignacionPremiumSinSesionBatchReanudaWorkerVivoConAsignacionPausada(t *testing.T) {
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
	if err := db.UpsertProyectoOperacion(&db.ProyectoOperacion{
		ProyectoID:       proyectoID,
		EstadoOperativo:  db.ProyectoOperativoActivo,
		Motivo:           "microrefactor_loop",
		ObjetivoPct:      100,
		MinAgentes:       1,
		MaxAgentes:       3,
		ResumeAutomatico: true,
	}); err != nil {
		t.Fatalf("upsert proyecto operacion: %v", err)
	}
	if err := db.ActivarAsignacion("Codex4", proyectoID, "reactivacion_automatica_trabajo_activo"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	if err := db.PausarAsignacion("Codex4", proyectoID, "sin_trabajo_reactivacion_automatica"); err != nil {
		t.Fatalf("pausar asignacion: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      microcicloRefactorTituloDefault,
		Descripcion: descripcionMicrocicloDefault(&db.Proyecto{Slug: "orquestador", RutaAbs: filepath.Join(tmp, "orquestador")}),
		ProyectoID:  &proyectoID,
		Modulo:      "controlplane",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "orquesta",
		Notas:       microcicloRefactorNotasTag,
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	now := time.Now().UTC()
	row := agentesapp.Row{
		Agente:          &db.Agente{Nombre: "Codex4", Habilitado: true},
		EstadoOperativo: "disponible",
		DetalleOperativo:"worker running",
		WorkerAlive:     true,
		WorkerState:     "running",
		WorkerDriver:    "tmux_cli_session",
		WorkerHeartbeat: &now,
		WorkerUpdatedAt: &now,
		Runtime:         &db.RuntimeInstance{LogicalState: "activo", UpdatedAt: now},
		Handle:          &db.RuntimeHandle{Estado: "activo", Transporte: "tmux", LastSeenAt: &now},
	}

	n, err := procesarAutoasignacionPremiumSinSesionBatch([]agentesapp.Row{row}, map[string][]*db.Tarea{})
	if err != nil {
		t.Fatalf("procesarAutoasignacionPremiumSinSesionBatch: %v", err)
	}
	if n < 1 {
		t.Fatalf("deberia reactivar worker vivo con asignacion pausada, got=%d", n)
	}

	tarea, err := db.GetTarea(tareaID)
	if err != nil {
		t.Fatalf("get tarea: %v", err)
	}
	if tarea == nil || tarea.Agente == nil || strings.TrimSpace(*tarea.Agente) != "Codex4" {
		t.Fatalf("la tarea deberia quedar reasignada al worker vivo: %+v", tarea)
	}

	asignacion, err := db.GetAsignacionActivaAgente("Codex4")
	if err != nil {
		t.Fatalf("get asignacion activa: %v", err)
	}
	if asignacion == nil || asignacion.ProyectoID != proyectoID {
		t.Fatalf("la asignacion deberia reactivarse para Codex4: %+v", asignacion)
	}

	agente := "Codex4"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	if len(orders) != 1 || orders[0].Tipo != "nudge" {
		t.Fatalf("deberia encolar nudge sobre worker ya vivo: %+v", orders)
	}
	if !strings.Contains(orders[0].PayloadJSON, `"motivo_autoasignacion":"runtime_idle_paused_assignment"`) {
		t.Fatalf("nudge sin motivo esperado: %s", orders[0].PayloadJSON)
	}
}

func TestResolverProyectoAutoasignacionPremiumSinSesionPrefiereAsignacionPausadaSobreSesionVieja(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Gemini1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoPremiumID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto premium: %v", err)
	}
	if err := db.UpsertProyectoOperacion(&db.ProyectoOperacion{
		ProyectoID:       proyectoPremiumID,
		EstadoOperativo:  db.ProyectoOperativoActivo,
		Motivo:           "microrefactor_loop",
		ObjetivoPct:      100,
		MinAgentes:       1,
		MaxAgentes:       3,
		ResumeAutomatico: true,
	}); err != nil {
		t.Fatalf("upsert proyecto operacion premium: %v", err)
	}
	proyectoAPIID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "api",
		Nombre:  "API",
		RutaAbs: filepath.Join(tmp, "api"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto api: %v", err)
	}
	if err := db.ActivarAsignacion("Gemini1", proyectoPremiumID, "reactivacion_automatica_trabajo_activo"); err != nil {
		t.Fatalf("activar asignacion premium: %v", err)
	}
	if err := db.PausarAsignacion("Gemini1", proyectoPremiumID, "sin_trabajo_reactivacion_automatica"); err != nil {
		t.Fatalf("pausar asignacion premium: %v", err)
	}
	if _, err := db.DB.Exec(`INSERT INTO sesiones (agente, proyecto_id, activa, inicio, herramienta, estado) VALUES (?, ?, 0, CURRENT_TIMESTAMP, 'gemini-cli', 'cerrada')`, "Gemini1", proyectoAPIID); err != nil {
		t.Fatalf("insertar sesion vieja: %v", err)
	}

	proyecto, err := resolverProyectoAutoasignacionPremiumSinSesion("Gemini1", agentesapp.Row{
		Agente:          &db.Agente{Nombre: "Gemini1"},
		EstadoOperativo: "sin_tarea",
	})
	if err != nil {
		t.Fatalf("resolverProyectoAutoasignacionPremiumSinSesion: %v", err)
	}
	if proyecto == nil || proyecto.ID != proyectoPremiumID {
		t.Fatalf("deberia preferir la asignacion pausada premium sobre la sesion vieja: %+v", proyecto)
	}
}
