package cmd

import (
	"path/filepath"
	"testing"
	"time"

	"orquesta/db"
	"orquesta/supervisionapp"
)

func TestResolverSupervisorAutonomiaOperativoReutilizaAgenteActivoAunqueCuentaCompartidaEsteOcupada(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("CodexSupervisor", "programador"); err != nil {
		t.Fatalf("registrar supervisor: %v", err)
	}
	if err := db.RegistrarAgente("CodexWorker", "programador"); err != nil {
		t.Fatalf("registrar worker: %v", err)
	}
	now := time.Now().UTC()
	if err := db.UpsertAgenteIdentidadObservada("CodexSupervisor", "shared@example.com", "shared", "test", &now); err != nil {
		t.Fatalf("identidad supervisor: %v", err)
	}
	if err := db.UpsertAgenteIdentidadObservada("CodexWorker", "shared@example.com", "shared", "test", &now); err != nil {
		t.Fatalf("identidad worker: %v", err)
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
	if _, err := supervisionService.UpsertProjectPolicy("orquestador", supervisionapp.PolicyInput{
		Enabled:              true,
		ObjetivoGeneral:      "terminar app",
		DefinitionOfDoneJSON: `{"done":true}`,
		SupervisorAgente:     "CodexSupervisor",
		ReserveSupervisor:    true,
		AutoCreateTasks:      true,
	}); err != nil {
		t.Fatalf("upsert policy: %v", err)
	}
	if err := db.ActivarAsignacion("CodexWorker", proyectoID, "worker_activo"); err != nil {
		t.Fatalf("activar worker: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:            "CodexWorker",
		ProyectoID:        &proyectoID,
		CWD:               filepath.Join(tmp, "orquestador"),
		Herramienta:       "codex-cli",
		ExternalSessionID: "sess-worker",
	}); err != nil {
		t.Fatalf("iniciar sesion worker: %v", err)
	}

	supervisor, activado, err := resolverSupervisorAutonomiaOperativo(proyectoID, true, "CodexSupervisor", "supervision_automatica", "")
	if err != nil {
		t.Fatalf("resolver supervisor: %v", err)
	}
	if supervisor == nil || supervisor.Nombre != "CodexWorker" {
		t.Fatalf("supervisor efectivo inesperado: %+v", supervisor)
	}
	if activado {
		t.Fatalf("no deberia intentar activar otro runtime si ya hay un codex activo reutilizable")
	}
}

func TestResolverSupervisorAutonomiaOperativoReutilizaRuntimeVivoSinHandleAunqueCuentaCompartidaEsteOcupada(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("CodexSupervisor", "programador"); err != nil {
		t.Fatalf("registrar supervisor: %v", err)
	}
	if err := db.RegistrarAgente("CodexBudget", "programador"); err != nil {
		t.Fatalf("registrar worker: %v", err)
	}
	if err := db.RegistrarAgente("CodexOccupant", "programador"); err != nil {
		t.Fatalf("registrar occupant: %v", err)
	}
	now := time.Now().UTC()
	if err := db.UpsertAgenteIdentidadObservada("CodexSupervisor", "shared@example.com", "shared", "test", &now); err != nil {
		t.Fatalf("identidad supervisor: %v", err)
	}
	if err := db.UpsertAgenteIdentidadObservada("CodexBudget", "shared@example.com", "shared", "test", &now); err != nil {
		t.Fatalf("identidad worker: %v", err)
	}
	if err := db.UpsertAgenteIdentidadObservada("CodexOccupant", "shared@example.com", "shared", "test", &now); err != nil {
		t.Fatalf("identidad occupant: %v", err)
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
	otroProyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "ocupado",
		Nombre:  "ocupado",
		RutaAbs: filepath.Join(tmp, "ocupado"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto ocupado: %v", err)
	}
	if _, err := supervisionService.UpsertProjectPolicy("orquestador", supervisionapp.PolicyInput{
		Enabled:              true,
		ObjetivoGeneral:      "terminar app",
		DefinitionOfDoneJSON: `{"done":true}`,
		SupervisorAgente:     "CodexSupervisor",
		ReserveSupervisor:    true,
		AutoCreateTasks:      true,
	}); err != nil {
		t.Fatalf("upsert policy: %v", err)
	}
	if err := db.ActivarAsignacion("CodexBudget", proyectoID, "worker_activo"); err != nil {
		t.Fatalf("activar worker: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:            "CodexBudget",
		ProyectoID:        &proyectoID,
		CWD:               filepath.Join(tmp, "orquestador"),
		Herramienta:       "codex-cli",
		ExternalSessionID: "sess-budget-runtime",
	})
	if err != nil {
		t.Fatalf("iniciar sesion budget: %v", err)
	}
	runtime, err := db.GetRuntimeBySesionID(sesion.ID)
	if err != nil || runtime == nil {
		t.Fatalf("get runtime: %+v err=%v", runtime, err)
	}
	if _, err := db.DB.Exec(`UPDATE runtime_instances SET logical_state='esperando_io', process_state='running' WHERE id=?`, runtime.ID); err != nil {
		t.Fatalf("actualizar runtime: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE sesiones SET activa=0, estado='pausada' WHERE id=?`, sesion.ID); err != nil {
		t.Fatalf("pausar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("get handle: %+v err=%v", handle, err)
	}
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET estado='cerrado' WHERE id=?`, handle.ID); err != nil {
		t.Fatalf("cerrar handle: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:            "CodexOccupant",
		ProyectoID:        &otroProyectoID,
		CWD:               filepath.Join(tmp, "ocupado"),
		Herramienta:       "codex-cli",
		ExternalSessionID: "sess-occupant-active",
	}); err != nil {
		t.Fatalf("iniciar sesion occupant: %v", err)
	}

	supervisor, activado, err := resolverSupervisorAutonomiaOperativo(proyectoID, true, "CodexSupervisor", "supervision_automatica", "")
	if err != nil {
		t.Fatalf("resolver supervisor: %v", err)
	}
	if supervisor == nil || supervisor.Nombre != "CodexBudget" {
		t.Fatalf("supervisor efectivo inesperado: %+v", supervisor)
	}
	if activado {
		t.Fatalf("no deberia reactivar si ya existe un runtime vivo reutilizable en el proyecto")
	}
}

func TestResolverSupervisorAutonomiaOperativoRearrancaSiSoloQuedaSesionActivaConRuntimeFallido(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("CodexSupervisor", "programador"); err != nil {
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
	if _, err := supervisionService.UpsertProjectPolicy("orquestador", supervisionapp.PolicyInput{
		Enabled:              true,
		ObjetivoGeneral:      "terminar app",
		DefinitionOfDoneJSON: `{"done":true}`,
		SupervisorAgente:     "CodexSupervisor",
		ReserveSupervisor:    true,
		AutoCreateTasks:      true,
	}); err != nil {
		t.Fatalf("upsert policy: %v", err)
	}
	if err := db.ActivarAsignacion("CodexSupervisor", proyectoID, "supervision_automatica"); err != nil {
		t.Fatalf("activar supervisor: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:            "CodexSupervisor",
		ProyectoID:        &proyectoID,
		CWD:               filepath.Join(tmp, "orquestador"),
		Herramienta:       "codex-cli",
		ExternalSessionID: "sess-supervisor-fallido",
	})
	if err != nil {
		t.Fatalf("iniciar sesion supervisor: %v", err)
	}
	runtime, err := db.GetRuntimeBySesionID(sesion.ID)
	if err != nil || runtime == nil {
		t.Fatalf("get runtime: %+v err=%v", runtime, err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("get handle: %+v err=%v", handle, err)
	}
	if _, err := db.DB.Exec(`UPDATE runtime_instances SET logical_state='fallido', process_state='exited' WHERE id=?`, runtime.ID); err != nil {
		t.Fatalf("fallar runtime: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET estado='fallido' WHERE id=?`, handle.ID); err != nil {
		t.Fatalf("fallar handle: %v", err)
	}

	supervisor, activado, err := resolverSupervisorAutonomiaOperativo(proyectoID, true, "CodexSupervisor", "supervision_automatica", "")
	if err != nil {
		t.Fatalf("resolver supervisor: %v", err)
	}
	if supervisor == nil || supervisor.Nombre != "CodexSupervisor" {
		t.Fatalf("supervisor inesperado: %+v", supervisor)
	}
	if !activado {
		t.Fatalf("deberia reactivar cuando solo queda sesion activa con runtime fallido")
	}
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: ptrString("CodexSupervisor"), ProyectoID: &proyectoID})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	startCount := 0
	for _, order := range orders {
		if order != nil && order.Tipo == "start" && order.Estado == "pendiente" {
			startCount++
		}
	}
	if startCount == 0 {
		t.Fatalf("deberia encolar start automatico, orders=%+v", orders)
	}
}
