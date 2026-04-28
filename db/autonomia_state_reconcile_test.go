package db

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestReconciliarEstadoAutonomiaCanonicalizaRuntimeOrderHandoffPendiente(t *testing.T) {
	prepararDBTemporal(t)
	if _, err := DB.Exec(`DELETE FROM agentes WHERE nombre IN ('codex1','codexpg1')`); err != nil {
		t.Fatalf("limpiar aliases previos: %v", err)
	}
	for _, nombre := range []string{"Codex1", "CodexPg1"} {
		if err := RegistrarAgente(nombre, "programador"); err != nil {
			t.Fatalf("RegistrarAgente %s: %v", nombre, err)
		}
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "demo-reconcile-runtime-order",
		Nombre:  "Demo Reconcile Runtime Order",
		RutaAbs: t.TempDir(),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("UpsertProyecto: %v", err)
	}
	payload := `{"agente_origen":"codex1","agente_destino":"codexpg1","tarea_id":24,"motivo":"test"}`
	id, err := insertReturningID(`
		INSERT INTO runtime_orders (
			agente, proyecto_id, tipo, payload_json, resultado_json, estado
		) VALUES (?,?,?,?,?,?)`,
		"CodexPg1", proyectoID, "handoff", payload, "{}", "pendiente",
	)
	if err != nil {
		t.Fatalf("insert runtime_order: %v", err)
	}

	if err := reconciliarAsignacionesDuplicadasAgenteProyecto(); err != nil {
		t.Fatalf("reconciliarAsignacionesDuplicadasAgenteProyecto: %v", err)
	}

	order, err := GetRuntimeOrder(id)
	if err != nil {
		t.Fatalf("GetRuntimeOrder: %v", err)
	}
	if order == nil {
		t.Fatalf("runtime order nil")
	}
	var got map[string]any
	if err := json.Unmarshal([]byte(order.PayloadJSON), &got); err != nil {
		t.Fatalf("payload json: %v", err)
	}
	if strings.TrimSpace(order.Agente) != "CodexPg1" {
		t.Fatalf("agente inesperado: %q", order.Agente)
	}
	if strings.TrimSpace(stringFromMap(got, "agente_origen", "")) != "Codex1" {
		t.Fatalf("agente_origen no canonico: %s", order.PayloadJSON)
	}
	if strings.TrimSpace(stringFromMap(got, "agente_destino", "")) != "CodexPg1" {
		t.Fatalf("agente_destino no canonico: %s", order.PayloadJSON)
	}
}

func TestReconciliarEstadoAutonomiaCompactaAsignacionesDuplicadas(t *testing.T) {
	prepararDBTemporal(t)
	if err := RegistrarAgente("Codex2", "programador"); err != nil {
		t.Fatalf("RegistrarAgente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "demo-reconcile-asignaciones",
		Nombre:  "Demo Reconcile Asignaciones",
		RutaAbs: t.TempDir(),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("UpsertProyecto: %v", err)
	}
	if _, err := DB.Exec(`INSERT INTO asignaciones (agente, proyecto_id, estado, nota) VALUES (?,?,?,?)`, "Codex2", proyectoID, "activa", "nueva"); err != nil {
		t.Fatalf("insert asignacion activa: %v", err)
	}
	if _, err := DB.Exec(`INSERT INTO asignaciones (agente, proyecto_id, estado, nota) VALUES (?,?,?,?)`, "Codex2", proyectoID, "pausada", "vieja"); err != nil {
		t.Fatalf("insert asignacion pausada: %v", err)
	}

	if err := reconciliarRuntimeOrdersAliasNoCanonico(); err != nil {
		t.Fatalf("reconciliarRuntimeOrdersAliasNoCanonico: %v", err)
	}

	var abiertas int
	if err := DB.QueryRow(`
		SELECT COUNT(*)
		FROM asignaciones
		WHERE agente = 'Codex2'
		  AND proyecto_id = ?
		  AND estado IN ('planificada','activa','pausada')`, proyectoID,
	).Scan(&abiertas); err != nil {
		t.Fatalf("count asignaciones abiertas: %v", err)
	}
	if abiertas != 1 {
		t.Fatalf("asignaciones abiertas inesperadas: %d", abiertas)
	}

	var cerradasDuplicadas int
	if err := DB.QueryRow(`
		SELECT COUNT(*)
		FROM asignaciones
		WHERE agente = 'Codex2'
		  AND proyecto_id = ?
		  AND estado = 'cerrada'
		  AND nota LIKE 'duplicada_compactada:%'`, proyectoID,
	).Scan(&cerradasDuplicadas); err != nil {
		t.Fatalf("count asignaciones cerradas duplicadas: %v", err)
	}
	if cerradasDuplicadas != 1 {
		t.Fatalf("cierres por compactacion inesperados: %d", cerradasDuplicadas)
	}
}

func TestReconciliarEstadoAutonomiaNormalizaAsignacionSupervisorReservadoActiva(t *testing.T) {
	prepararDBTemporal(t)
	if err := RegistrarAgente("Codex2", "programador"); err != nil {
		t.Fatalf("RegistrarAgente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "demo-supervisor-note",
		Nombre:  "Demo Supervisor Note",
		RutaAbs: t.TempDir(),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("UpsertProyecto: %v", err)
	}
	if _, err := UpsertProyectoAutonomia(&ProyectoAutonomia{
		ProyectoID:        proyectoID,
		Enabled:           true,
		SupervisorAgente:  "Codex2",
		ReserveSupervisor: true,
		EstadoAutonomia:   AutonomiaProyectoActiva,
	}); err != nil {
		t.Fatalf("UpsertProyectoAutonomia: %v", err)
	}
	if err := ActivarAsignacion("Codex2", proyectoID, "handoff_recibido_desde_Codex1"); err != nil {
		t.Fatalf("ActivarAsignacion: %v", err)
	}

	if err := reconciliarAsignacionesSupervisorReservadoActivas(); err != nil {
		t.Fatalf("reconciliarAsignacionesSupervisorReservadoActivas: %v", err)
	}

	estado := AsignacionActiva
	asignaciones, err := ListarAsignaciones(FiltroAsignaciones{ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("ListarAsignaciones: %v", err)
	}
	if len(asignaciones) != 1 || strings.TrimSpace(asignaciones[0].Nota) != "supervision_automatica" {
		t.Fatalf("nota supervisor inesperada: %+v", asignaciones)
	}
}

func TestReconciliarEstadoAutonomiaCancelaOrdenYMailboxWorkerEnSupervisorReservado(t *testing.T) {
	prepararDBTemporal(t)
	if err := RegistrarAgente("Codex2", "programador"); err != nil {
		t.Fatalf("RegistrarAgente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "demo-supervisor-worker-order",
		Nombre:  "Demo Supervisor Worker Order",
		RutaAbs: t.TempDir(),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("UpsertProyecto: %v", err)
	}
	if _, err := UpsertProyectoAutonomia(&ProyectoAutonomia{
		ProyectoID:        proyectoID,
		Enabled:           true,
		SupervisorAgente:  "Codex2",
		ReserveSupervisor: true,
		EstadoAutonomia:   AutonomiaProyectoActiva,
	}); err != nil {
		t.Fatalf("UpsertProyectoAutonomia: %v", err)
	}
	tareaID, err := CrearTarea(&Tarea{
		Titulo:      "Cerrar Orquesta al 100%",
		Descripcion: "Worker task mal dirigida al supervisor",
		ProyectoID:  &proyectoID,
		Prioridad:   PrioridadAlta,
		CreadoPor:   "orquesta",
		Notas:       "autonomia:finish_app",
	})
	if err != nil {
		t.Fatalf("CrearTarea: %v", err)
	}
	payload := `{"kind":"pipeline_local","source":"pipeline_local","accion":"implementar","tarea_objetivo_id":` + jsonNumber(tareaID) + `}`
	mailboxID, err := insertReturningID(`
		INSERT INTO runtime_mailbox (
			from_agente, to_agente, proyecto_id, kind, payload_json, estado
		) VALUES (?,?,?,?,?,?)`,
		"server", "Codex2", proyectoID, "pipeline_local", payload, "pendiente",
	)
	if err != nil {
		t.Fatalf("insert runtime_mailbox: %v", err)
	}
	orderID, err := insertReturningID(`
		INSERT INTO runtime_orders (
			agente, proyecto_id, tipo, payload_json, resultado_json, estado
		) VALUES (?,?,?,?,?,?)`,
		"Codex2", proyectoID, "send_instruction", payload, "{}", "pendiente",
	)
	if err != nil {
		t.Fatalf("insert runtime_order: %v", err)
	}

	if err := reconciliarRuntimeMailboxWorkerEnSupervisorReservado(); err != nil {
		t.Fatalf("reconciliarRuntimeMailboxWorkerEnSupervisorReservado: %v", err)
	}
	if err := reconciliarRuntimeOrdersWorkerEnSupervisorReservado(); err != nil {
		t.Fatalf("reconciliarRuntimeOrdersWorkerEnSupervisorReservado: %v", err)
	}

	var mailboxEstado string
	if err := DB.QueryRow(`SELECT estado FROM runtime_mailbox WHERE id=?`, mailboxID).Scan(&mailboxEstado); err != nil {
		t.Fatalf("mailbox estado: %v", err)
	}
	if mailboxEstado != "cancelado" {
		t.Fatalf("mailbox deberia quedar cancelado, got=%q", mailboxEstado)
	}
	var orderEstado, orderError string
	if err := DB.QueryRow(`SELECT estado, COALESCE(error_text,'') FROM runtime_orders WHERE id=?`, orderID).Scan(&orderEstado, &orderError); err != nil {
		t.Fatalf("order estado: %v", err)
	}
	if orderEstado != "cancelada" || !strings.Contains(orderError, "supervisor_reserved_worker_task") {
		t.Fatalf("orden deberia quedar cancelada por supervisor reservado, got estado=%q error=%q", orderEstado, orderError)
	}
}

func TestReconciliarEstadoAutonomiaCancelaOrdenYMailboxWorkerEnReviewerReservado(t *testing.T) {
	prepararDBTemporal(t)
	if err := RegistrarAgente("Codex3", "programador"); err != nil {
		t.Fatalf("RegistrarAgente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "demo-reviewer-worker-order",
		Nombre:  "Demo Reviewer Worker Order",
		RutaAbs: t.TempDir(),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("UpsertProyecto: %v", err)
	}
	if _, err := UpsertProyectoAutonomia(&ProyectoAutonomia{
		ProyectoID:      proyectoID,
		Enabled:         true,
		ReviewerAgente:  "Codex3",
		ReserveReviewer: true,
		EstadoAutonomia: AutonomiaProyectoActiva,
	}); err != nil {
		t.Fatalf("UpsertProyectoAutonomia: %v", err)
	}
	tareaID, err := CrearTarea(&Tarea{
		Titulo:      "Cerrar Orquesta al 100%",
		Descripcion: "Worker task mal dirigida al reviewer",
		ProyectoID:  &proyectoID,
		Prioridad:   PrioridadAlta,
		CreadoPor:   "orquesta",
		Notas:       "autonomia:finish_app",
	})
	if err != nil {
		t.Fatalf("CrearTarea: %v", err)
	}
	payload := `{"kind":"pipeline_local","source":"pipeline_local","accion":"implementar","tarea_objetivo_id":` + jsonNumber(tareaID) + `}`
	mailboxID, err := insertReturningID(`
		INSERT INTO runtime_mailbox (
			from_agente, to_agente, proyecto_id, kind, payload_json, estado
		) VALUES (?,?,?,?,?,?)`,
		"server", "Codex3", proyectoID, "pipeline_local", payload, "pendiente",
	)
	if err != nil {
		t.Fatalf("insert runtime_mailbox: %v", err)
	}
	orderID, err := insertReturningID(`
		INSERT INTO runtime_orders (
			agente, proyecto_id, tipo, payload_json, resultado_json, estado
		) VALUES (?,?,?,?,?,?)`,
		"Codex3", proyectoID, "send_instruction", payload, "{}", "pendiente",
	)
	if err != nil {
		t.Fatalf("insert runtime_order: %v", err)
	}

	if err := reconciliarRuntimeMailboxWorkerEnSupervisorReservado(); err != nil {
		t.Fatalf("reconciliarRuntimeMailboxWorkerEnSupervisorReservado: %v", err)
	}
	if err := reconciliarRuntimeOrdersWorkerEnSupervisorReservado(); err != nil {
		t.Fatalf("reconciliarRuntimeOrdersWorkerEnSupervisorReservado: %v", err)
	}

	var mailboxEstado string
	if err := DB.QueryRow(`SELECT estado FROM runtime_mailbox WHERE id=?`, mailboxID).Scan(&mailboxEstado); err != nil {
		t.Fatalf("mailbox estado: %v", err)
	}
	if mailboxEstado != "cancelado" {
		t.Fatalf("mailbox deberia quedar cancelado, got=%q", mailboxEstado)
	}
	var orderEstado, orderError string
	if err := DB.QueryRow(`SELECT estado, COALESCE(error_text,'') FROM runtime_orders WHERE id=?`, orderID).Scan(&orderEstado, &orderError); err != nil {
		t.Fatalf("order estado: %v", err)
	}
	if orderEstado != "cancelada" || !strings.Contains(orderError, "supervisor_reserved_worker_task") {
		t.Fatalf("orden deberia quedar cancelada por reviewer reservado, got estado=%q error=%q", orderEstado, orderError)
	}
}

func TestReconciliarEstadoAutonomiaCancelaPipelineFantasmaSinTareaNiAsignacionActiva(t *testing.T) {
	prepararDBTemporal(t)
	if err := RegistrarAgente("Codex4", "programador"); err != nil {
		t.Fatalf("RegistrarAgente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "demo-pipeline-fantasma",
		Nombre:  "Demo Pipeline Fantasma",
		RutaAbs: t.TempDir(),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("UpsertProyecto: %v", err)
	}
	if err := ActivarAsignacion("Codex4", proyectoID, "handoff_temporal"); err != nil {
		t.Fatalf("ActivarAsignacion: %v", err)
	}
	if err := PausarAsignacion("Codex4", proyectoID, "handoff_cedido"); err != nil {
		t.Fatalf("PausarAsignacion: %v", err)
	}
	payload := `{"kind":"pipeline_local","source":"pipeline_local","accion":"especificar","tarea_objetivo_id":0}`
	mailboxID, err := insertReturningID(`
		INSERT INTO runtime_mailbox (
			from_agente, to_agente, proyecto_id, kind, payload_json, estado
		) VALUES (?,?,?,?,?,?)`,
		"server", "Codex4", proyectoID, "pipeline_local", payload, "pendiente",
	)
	if err != nil {
		t.Fatalf("insert runtime_mailbox: %v", err)
	}
	orderID, err := insertReturningID(`
		INSERT INTO runtime_orders (
			agente, proyecto_id, tipo, payload_json, resultado_json, estado
		) VALUES (?,?,?,?,?,?)`,
		"Codex4", proyectoID, "send_instruction", payload, "{}", "pendiente",
	)
	if err != nil {
		t.Fatalf("insert runtime_order: %v", err)
	}

	if err := reconciliarRuntimeMailboxPipelineFantasma(); err != nil {
		t.Fatalf("reconciliarRuntimeMailboxPipelineFantasma: %v", err)
	}
	if err := reconciliarRuntimeOrdersPipelineFantasma(); err != nil {
		t.Fatalf("reconciliarRuntimeOrdersPipelineFantasma: %v", err)
	}

	var mailboxEstado string
	if err := DB.QueryRow(`SELECT estado FROM runtime_mailbox WHERE id=?`, mailboxID).Scan(&mailboxEstado); err != nil {
		t.Fatalf("mailbox estado: %v", err)
	}
	if mailboxEstado != "cancelado" {
		t.Fatalf("mailbox fantasma deberia quedar cancelado, got=%q", mailboxEstado)
	}
	var orderEstado, orderError string
	if err := DB.QueryRow(`SELECT estado, COALESCE(error_text,'') FROM runtime_orders WHERE id=?`, orderID).Scan(&orderEstado, &orderError); err != nil {
		t.Fatalf("order estado: %v", err)
	}
	if orderEstado != "cancelada" || !strings.Contains(orderError, "ghost_pipeline_without_task") {
		t.Fatalf("orden fantasma deberia quedar cancelada, got estado=%q error=%q", orderEstado, orderError)
	}
}

func TestReconciliarEstadoAutonomiaActualizaNotasFinishAppConPolicyViva(t *testing.T) {
	prepararDBTemporal(t)
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "demo-finish-app-policy",
		Nombre:  "Demo Finish App Policy",
		RutaAbs: t.TempDir(),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("UpsertProyecto: %v", err)
	}
	if _, err := UpsertProyectoAutonomia(&ProyectoAutonomia{
		ProyectoID:        proyectoID,
		Enabled:           true,
		SupervisorAgente:  "Codex2",
		ReviewerAgente:    "Codex3",
		MaxWorkers:        4,
		AutoCreateTasks:   true,
		ReserveSupervisor: true,
		ReserveReviewer:   true,
		EstadoAutonomia:   AutonomiaProyectoActiva,
	}); err != nil {
		t.Fatalf("UpsertProyectoAutonomia: %v", err)
	}
	tareaID, err := CrearTarea(&Tarea{
		Titulo:      "Cerrar app completa",
		Descripcion: "Finish app con policy stale en notas",
		ProyectoID:  &proyectoID,
		Prioridad:   PrioridadAlta,
		CreadoPor:   "orquesta",
		Notas: `autonomia:finish_app
autonomia_persistente_v1:
  enabled: true
  supervisor_agente: Codex1
  reviewer_agente: Codex2
  max_workers: 4
  auto_create_tasks: true
handoff Codex1→Codex4: stale`,
	})
	if err != nil {
		t.Fatalf("CrearTarea: %v", err)
	}

	if err := ReconciliarEstadoAutonomia(); err != nil {
		t.Fatalf("ReconciliarEstadoAutonomia: %v", err)
	}

	tarea, err := GetTarea(tareaID)
	if err != nil {
		t.Fatalf("GetTarea: %v", err)
	}
	if tarea == nil {
		t.Fatalf("tarea nil")
	}
	if !strings.Contains(tarea.Notas, "supervisor_agente: Codex2") {
		t.Fatalf("supervisor no reconciliado en notas: %s", tarea.Notas)
	}
	if !strings.Contains(tarea.Notas, "reviewer_agente: Codex3") {
		t.Fatalf("reviewer no reconciliado en notas: %s", tarea.Notas)
	}
	if strings.Contains(tarea.Notas, "supervisor_agente: Codex1") {
		t.Fatalf("supervisor stale sigue presente: %s", tarea.Notas)
	}
	if !strings.Contains(tarea.Notas, "handoff Codex1→Codex4: stale") {
		t.Fatalf("la reconciliacion no deberia perder anotaciones posteriores: %s", tarea.Notas)
	}
}

func TestReconciliarEstadoAutonomiaInsertaBloquePersistenteSiFalta(t *testing.T) {
	prepararDBTemporal(t)
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "demo-finish-app-policy-missing",
		Nombre:  "Demo Finish App Policy Missing",
		RutaAbs: t.TempDir(),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("UpsertProyecto: %v", err)
	}
	if _, err := UpsertProyectoAutonomia(&ProyectoAutonomia{
		ProyectoID:        proyectoID,
		Enabled:           true,
		SupervisorAgente:  "Codex2",
		ReviewerAgente:    "Codex3",
		MaxWorkers:        3,
		AutoCreateTasks:   true,
		ReserveSupervisor: true,
		ReserveReviewer:   true,
		EstadoAutonomia:   AutonomiaProyectoActiva,
	}); err != nil {
		t.Fatalf("UpsertProyectoAutonomia: %v", err)
	}
	tareaID, err := CrearTarea(&Tarea{
		Titulo:      "Cerrar app completa",
		Descripcion: "Finish app sin bloque persistente",
		ProyectoID:  &proyectoID,
		Prioridad:   PrioridadAlta,
		CreadoPor:   "orquesta",
		Notas: `autonomia:finish_app
nota previa`,
	})
	if err != nil {
		t.Fatalf("CrearTarea: %v", err)
	}

	if err := ReconciliarEstadoAutonomia(); err != nil {
		t.Fatalf("ReconciliarEstadoAutonomia: %v", err)
	}

	tarea, err := GetTarea(tareaID)
	if err != nil {
		t.Fatalf("GetTarea: %v", err)
	}
	if tarea == nil {
		t.Fatalf("tarea nil")
	}
	if !strings.Contains(tarea.Notas, "autonomia_persistente_v1:") {
		t.Fatalf("deberia insertar bloque persistente: %s", tarea.Notas)
	}
	if !strings.Contains(tarea.Notas, "supervisor_agente: Codex2") || !strings.Contains(tarea.Notas, "reviewer_agente: Codex3") {
		t.Fatalf("bloque persistente incompleto: %s", tarea.Notas)
	}
}

func TestReconciliarEstadoAutonomiaCompactaFinishAppLibreDuplicada(t *testing.T) {
	prepararDBTemporal(t)
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "demo-finish-app-duplicate-free",
		Nombre:  "Demo Finish App Duplicate Free",
		RutaAbs: t.TempDir(),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("UpsertProyecto: %v", err)
	}
	activaID, err := CrearTarea(&Tarea{
		Titulo:      "Cerrar Orquesta al 100%",
		Descripcion: "finish_app activa",
		ProyectoID:  &proyectoID,
		Prioridad:   PrioridadAlta,
		CreadoPor:   "orquesta",
		Notas:       "autonomia:finish_app",
	})
	if err != nil {
		t.Fatalf("CrearTarea activa: %v", err)
	}
	if err := TomarTarea(activaID, "Codex1"); err != nil {
		t.Fatalf("TomarTarea activa: %v", err)
	}
	if err := IniciarTarea(activaID, "Codex1"); err != nil {
		t.Fatalf("IniciarTarea activa: %v", err)
	}
	libreID, err := CrearTarea(&Tarea{
		Titulo:      "Cerrar Orquesta al 100%",
		Descripcion: "finish_app libre duplicada",
		ProyectoID:  &proyectoID,
		Prioridad:   PrioridadAlta,
		CreadoPor:   "orquesta",
		Notas:       "autonomia:finish_app",
	})
	if err != nil {
		t.Fatalf("CrearTarea libre: %v", err)
	}

	if err := ReconciliarEstadoAutonomia(); err != nil {
		t.Fatalf("ReconciliarEstadoAutonomia: %v", err)
	}

	libre, err := GetTarea(libreID)
	if err != nil {
		t.Fatalf("GetTarea libre: %v", err)
	}
	if libre == nil || libre.Estado != EstadoBacklog {
		t.Fatalf("la finish_app libre duplicada deberia pasar a backlog: %+v", libre)
	}
	if !strings.Contains(libre.Notas, "compactada contra tarea #"+fmt.Sprintf("%d", activaID)+" activa") {
		t.Fatalf("nota de compactacion inesperada: %s", libre.Notas)
	}
}
