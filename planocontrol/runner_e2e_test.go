package planocontrol

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"orquesta/db"
)

type dbAutomationServiceTest struct{}

func (dbAutomationServiceTest) CheckReanimaciones() ([]*db.Agente, error) {
	return db.CheckReanimaciones()
}
func (dbAutomationServiceTest) ResetReanimacion(nombre string) error {
	return db.ResetReanimacion(nombre)
}
func (dbAutomationServiceTest) GarantizarSaludAgentes() error { return db.GarantizarSaludAgentes() }
func (dbAutomationServiceTest) PlanificarTareasAutomaticamente() error {
	return db.PlanificarTareasAutomaticamente()
}
func (dbAutomationServiceTest) ProcesarAutonomiaAgentesBatch() (int, error) {
	return 0, nil
}
func (dbAutomationServiceTest) ProcesarSupervisionAutonomaBatch() (int, error) {
	return 0, nil
}
func (dbAutomationServiceTest) ProcesarReviewGatesBatch() (int, error) {
	return 0, nil
}
func (dbAutomationServiceTest) ReconciliarRuntimeHandlesStale() (int, error) {
	return db.ReconciliarRuntimeHandlesStale()
}
func (dbAutomationServiceTest) ReconciliarRuntimeOrdersStale() (int, error) {
	return db.ReconciliarRuntimeOrdersStale()
}
func (dbAutomationServiceTest) ProcesarRuntimeTranscriptBatch() (int, error) {
	return db.IngestarRuntimeTranscriptActivos()
}
func (dbAutomationServiceTest) ProcesarRuntimeOrdersBatch() (int, error) {
	return db.ProcesarRuntimeOrdersBatch()
}
func (dbAutomationServiceTest) ProcesarGitMergesBatch() (int, error) {
	return 0, nil
}
func (dbAutomationServiceTest) ProcesarRefineriaBatch() (int, error) {
	return db.ProcesarRefineriaBatch()
}
func (dbAutomationServiceTest) ProcesarHandoffsBatch() (int, error) {
	return db.ProcesarHandoffsBatch()
}
func (dbAutomationServiceTest) Audit(agente, accion, entidad string, entidadID int64, detalle string) {
	db.Audit(agente, accion, entidad, entidadID, detalle)
}

func prepararDBTemporalRunner(t *testing.T) string {
	t.Helper()

	anteriorDB := os.Getenv("ORQUESTA_DB")
	t.Cleanup(func() {
		db.Close()
		db.DB = nil
		if anteriorDB == "" {
			_ = os.Unsetenv("ORQUESTA_DB")
		} else {
			_ = os.Setenv("ORQUESTA_DB", anteriorDB)
		}
	})

	db.Close()
	db.DB = nil

	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "orquesta-runner-e2e.db")
	if err := os.Setenv("ORQUESTA_DB", dbPath); err != nil {
		t.Fatalf("setenv ORQUESTA_DB: %v", err)
	}
	if err := db.Open(); err != nil {
		t.Fatalf("open db temporal: %v", err)
	}
	return tmp
}

func TestRunnerWatchdogYHandoffConProcesoVivo(t *testing.T) {
	tmp := prepararDBTemporalRunner(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar Codex1: %v", err)
	}
	if err := db.RegistrarAgente("Codex2", "programador"); err != nil {
		t.Fatalf("registrar Codex2: %v", err)
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

	proc := exec.Command("sleep", "30")
	if err := proc.Start(); err != nil {
		t.Fatalf("start sleep: %v", err)
	}
	t.Cleanup(func() {
		if proc.Process == nil {
			return
		}
		_ = proc.Process.Signal(syscall.SIGTERM)
		done := make(chan error, 1)
		go func() { done <- proc.Wait() }()
		select {
		case <-time.After(2 * time.Second):
			_ = proc.Process.Kill()
			<-done
		case <-done:
		}
	})
	if err := proc.Process.Signal(syscall.Signal(0)); err != nil {
		t.Fatalf("proceso vivo no accesible: %v", err)
	}

	pid := int64(proc.Process.Pid)
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:             "Codex1",
		ProyectoID:         &proyectoID,
		CWD:                filepath.Join(tmp, "orquestador", "sesion-codex1-live"),
		Herramienta:        "codex-cli",
		ExternalSessionID:  "sess-codex1-live",
		ResumenContinuidad: "sesion viva para watchdog e2e",
		Branch:             "main",
		PID:                &pid,
	})
	if err != nil {
		t.Fatalf("iniciar sesion viva: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("runtime handle vivo: handle=%+v err=%v", handle, err)
	}
	if handle.Estado != "activo" || handle.HandleKind != "process" {
		t.Fatalf("handle vivo inesperado: %+v", handle)
	}

	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Watchdog con runtime vivo",
		Descripcion: "Cobertura e2e watchdog/handoff",
		Modulo:      "runtime",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "tester",
		ProyectoID:  &proyectoID,
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

	if _, err := db.DB.Exec(`UPDATE config SET valor = '1' WHERE clave = 'pool_handoff_threshold_seconds'`); err != nil {
		t.Fatalf("config threshold handoff: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE config SET valor = '3600' WHERE clave = 'runtime_handle_stale_seconds'`); err != nil {
		t.Fatalf("config threshold handle stale: %v", err)
	}
	staleAt := time.Now().UTC().Add(-10 * time.Minute)
	if _, err := db.DB.Exec(`UPDATE sesiones SET heartbeat_at = ? WHERE id = ?`, staleAt, sesion.ID); err != nil {
		t.Fatalf("envejecer heartbeat sesion: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	runner := &Runner{
		Automation:        dbAutomationServiceTest{},
		ReanimacionCada:   time.Hour,
		SaludCada:         time.Hour,
		PlanificacionCada: time.Hour,
		ControlPlaneCada:  20 * time.Millisecond,
	}
	runner.Start(ctx)

	agenteOrigen := "Codex1"
	agenteDestino := "Codex2"
	estadoPendiente := "pendiente"
	var (
		syncOrder       *db.RuntimeOrder
		nudgeOrder      *db.RuntimeOrder
		handoffOrder    *db.RuntimeOrder
		watchdogInbox   []*db.RuntimeMailboxMessage
		tareaFinal      *db.Tarea
		sawWatchdog     bool
		sawHandoffBatch bool
	)

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		ordersOrigen, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agenteOrigen})
		if err != nil {
			t.Fatalf("listar orders origen: %v", err)
		}
		syncOrder = nil
		nudgeOrder = nil
		for _, order := range ordersOrigen {
			if order == nil {
				continue
			}
			switch order.Tipo {
			case "sync_status":
				syncOrder = order
			case "nudge":
				if strings.Contains(strings.ToLower(order.PayloadJSON), `"kind":"watchdog"`) {
					nudgeOrder = order
				}
			}
		}

		ordersDestino, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agenteDestino, Estado: &estadoPendiente})
		if err != nil {
			t.Fatalf("listar orders destino: %v", err)
		}
		handoffOrder = nil
		for _, order := range ordersDestino {
			if order != nil && order.Tipo == "handoff" {
				handoffOrder = order
				break
			}
		}

		watchdogInbox, err = db.ListarRuntimeMailbox(db.FiltroRuntimeMailbox{
			ToAgente:   &agenteOrigen,
			ProyectoID: &proyectoID,
			Estado:     &estadoPendiente,
		})
		if err != nil {
			t.Fatalf("listar mailbox watchdog: %v", err)
		}

		tareaFinal, err = db.GetTarea(tareaID)
		if err != nil {
			t.Fatalf("get tarea final: %v", err)
		}

		logs, err := db.ListarAuditoria(db.FiltroAuditoria{Limite: 50})
		if err != nil {
			t.Fatalf("listar auditoria: %v", err)
		}
		sawWatchdog = false
		sawHandoffBatch = false
		for _, item := range logs {
			switch item.Accion {
			case "watchdog_runtime_sondeo":
				sawWatchdog = true
			case "handoff_batch":
				sawHandoffBatch = true
			}
		}

		if syncOrder != nil && syncOrder.Estado == "completada" &&
			nudgeOrder != nil && nudgeOrder.Estado == "completada" &&
			handoffOrder != nil &&
			tareaFinal != nil && tareaFinal.Agente != nil && *tareaFinal.Agente == "Codex2" &&
			sawWatchdog && sawHandoffBatch {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	if syncOrder == nil || syncOrder.Estado != "completada" {
		t.Fatalf("sync_status watchdog no completada: %+v", syncOrder)
	}
	if nudgeOrder == nil || nudgeOrder.Estado != "completada" {
		t.Fatalf("nudge watchdog no completada: %+v", nudgeOrder)
	}
	if handoffOrder == nil || handoffOrder.Tipo != "handoff" {
		t.Fatalf("handoff watchdog no creada: %+v", handoffOrder)
	}
	if tareaFinal == nil || tareaFinal.Agente == nil || *tareaFinal.Agente != "Codex2" || tareaFinal.Estado != db.TareaAsignada {
		t.Fatalf("tarea no reasignada por watchdog: %+v", tareaFinal)
	}
	if len(watchdogInbox) != 1 || watchdogInbox[0].Kind != "watchdog" {
		t.Fatalf("mailbox watchdog inesperado: %+v", watchdogInbox)
	}
	if !strings.Contains(strings.ToLower(watchdogInbox[0].PayloadJSON), "heartbeat obsoleto") {
		t.Fatalf("payload watchdog inesperado: %s", watchdogInbox[0].PayloadJSON)
	}

	handle, err = db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil {
		t.Fatalf("get handle origen final: %v", err)
	}
	if handle == nil || handle.Estado != "pausado" {
		t.Fatalf("handle origen no pausado tras handoff watchdog: %+v", handle)
	}
	if err := proc.Process.Signal(syscall.Signal(0)); err != nil {
		t.Fatalf("el proceso vivo debería seguir accesible durante el handoff: %v", err)
	}
	if !sawWatchdog || !sawHandoffBatch {
		t.Fatalf("auditoria watchdog/handoff incompleta: watchdog=%v handoff=%v", sawWatchdog, sawHandoffBatch)
	}
}

func TestRunnerHandoffPreventivoPorPresupuesto(t *testing.T) {
	tmp := prepararDBTemporalRunner(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar Codex1: %v", err)
	}
	if err := db.RegistrarAgente("Codex2", "programador"); err != nil {
		t.Fatalf("registrar Codex2: %v", err)
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
		Agente:             "Codex1",
		ProyectoID:         &proyectoID,
		CWD:                filepath.Join(tmp, "orquestador", "sesion-codex1-budget"),
		Herramienta:        "codex-cli",
		ExternalSessionID:  "sess-codex1-budget",
		ResumenContinuidad: "sesion viva para handoff preventivo por presupuesto",
		Branch:             "main",
	})
	if err != nil {
		t.Fatalf("iniciar sesion viva: %v", err)
	}

	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Handoff preventivo por presupuesto",
		Descripcion: "Cobertura e2e presupuesto/handoff",
		Modulo:      "runtime",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "tester",
		ProyectoID:  &proyectoID,
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

	if _, err := db.DB.Exec(`UPDATE config SET valor = '1800' WHERE clave = 'pool_handoff_threshold_seconds'`); err != nil {
		t.Fatalf("config threshold handoff: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE config SET valor = '300' WHERE clave = 'pool_budget_snapshot_max_age_seconds'`); err != nil {
		t.Fatalf("config freshness presupuesto: %v", err)
	}
	remaining := int64(15)
	if _, err := db.RegistrarPresupuestoSesion(&db.PresupuestoSesion{
		SesionID:         sesion.ID,
		WindowKind:       "5h",
		RemainingSeconds: &remaining,
		BudgetSource:     "manual",
		CheckedAt:        time.Now().UTC(),
	}); err != nil {
		t.Fatalf("registrar presupuesto: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	runner := &Runner{
		Automation:        dbAutomationServiceTest{},
		ReanimacionCada:   time.Hour,
		SaludCada:         time.Hour,
		PlanificacionCada: time.Hour,
		ControlPlaneCada:  20 * time.Millisecond,
	}
	runner.Start(ctx)

	agenteOrigen := "Codex1"
	agenteDestino := "Codex2"
	estadoPendiente := "pendiente"
	var handoffOrder *db.RuntimeOrder

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		ordersDestino, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agenteDestino, Estado: &estadoPendiente})
		if err != nil {
			t.Fatalf("listar orders destino: %v", err)
		}
		handoffOrder = nil
		for _, order := range ordersDestino {
			if order != nil && order.Tipo == "handoff" {
				handoffOrder = order
				break
			}
		}
		if handoffOrder != nil {
			break
		}
		time.Sleep(25 * time.Millisecond)
	}
	if handoffOrder == nil {
		t.Fatalf("no aparecio handoff por presupuesto")
	}

	ordersOrigen, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agenteOrigen, Estado: &estadoPendiente})
	if err != nil {
		t.Fatalf("listar orders origen: %v", err)
	}
	for _, order := range ordersOrigen {
		if order == nil {
			continue
		}
		if order.Tipo == "sync_status" || order.Tipo == "nudge" {
			t.Fatalf("no deberia haber sondeo watchdog en handoff por presupuesto: %+v", ordersOrigen)
		}
	}

	tareaFinal, err := db.GetTarea(tareaID)
	if err != nil {
		t.Fatalf("get tarea final: %v", err)
	}
	if tareaFinal.Agente == nil || *tareaFinal.Agente != "Codex2" || tareaFinal.Estado != db.TareaAsignada {
		t.Fatalf("tarea final inesperada: %+v", tareaFinal)
	}
}
