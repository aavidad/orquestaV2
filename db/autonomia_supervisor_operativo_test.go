package db

import (
	"testing"
	"time"
)

func TestSeleccionarSupervisorAutonomiaOperativoHaceFallbackSiPreferidoNoDisponible(t *testing.T) {
	prepararDBTemporal(t)

	if err := RegistrarAgente("CodexSupervisor", "programador"); err != nil {
		t.Fatalf("registrar supervisor: %v", err)
	}
	if err := RegistrarAgente("CodexWorker", "programador"); err != nil {
		t.Fatalf("registrar worker: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "supervision-fallback",
		Nombre:  "supervision-fallback",
		RutaAbs: t.TempDir(),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := UpsertProyectoAutonomia(&ProyectoAutonomia{
		ProyectoID:        proyectoID,
		Enabled:           true,
		SupervisorAgente:  "CodexSupervisor",
		ReserveSupervisor: true,
		EstadoAutonomia:   AutonomiaProyectoActiva,
	}); err != nil {
		t.Fatalf("upsert autonomia: %v", err)
	}
	if _, err := DB.Exec(`UPDATE agentes SET habilitado=0 WHERE nombre='CodexSupervisor'`); err != nil {
		t.Fatalf("deshabilitar supervisor preferido: %v", err)
	}
	if err := ActivarAsignacion("CodexWorker", proyectoID, "supervision_worker"); err != nil {
		t.Fatalf("activar asignacion worker: %v", err)
	}

	supervisor, err := SeleccionarSupervisorAutonomiaOperativo(proyectoID, "")
	if err != nil {
		t.Fatalf("seleccionar supervisor operativo: %v", err)
	}
	if supervisor == nil || supervisor.Nombre != "CodexWorker" {
		t.Fatalf("fallback inesperado: %+v", supervisor)
	}
}

func TestSeleccionarSupervisorAutonomiaOperativoPriorizaCandidatoActivoEnProyecto(t *testing.T) {
	prepararDBTemporal(t)

	if err := RegistrarAgente("CodexIdle", "programador"); err != nil {
		t.Fatalf("registrar idle: %v", err)
	}
	if err := RegistrarAgente("CodexActivo", "programador"); err != nil {
		t.Fatalf("registrar activo: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "supervision-activa",
		Nombre:  "supervision-activa",
		RutaAbs: t.TempDir(),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := UpsertProyectoAutonomia(&ProyectoAutonomia{
		ProyectoID:        proyectoID,
		Enabled:           true,
		ReserveSupervisor: true,
		EstadoAutonomia:   AutonomiaProyectoActiva,
	}); err != nil {
		t.Fatalf("upsert autonomia: %v", err)
	}
	if err := ActivarAsignacion("CodexIdle", proyectoID, "idle"); err != nil {
		t.Fatalf("activar asignacion idle: %v", err)
	}
	if err := ActivarAsignacion("CodexActivo", proyectoID, "activo"); err != nil {
		t.Fatalf("activar asignacion activo: %v", err)
	}
	if _, err := IniciarSesionContexto(SesionInicio{
		Agente:            "CodexActivo",
		ProyectoID:        &proyectoID,
		CWD:               t.TempDir(),
		Herramienta:       "codex-cli",
		ExternalSessionID: "sess-codex-activo",
	}); err != nil {
		t.Fatalf("iniciar sesion activa: %v", err)
	}

	supervisor, err := SeleccionarSupervisorAutonomiaOperativo(proyectoID, "")
	if err != nil {
		t.Fatalf("seleccionar supervisor operativo: %v", err)
	}
	if supervisor == nil || supervisor.Nombre != "CodexActivo" {
		t.Fatalf("supervisor priorizado inesperado: %+v", supervisor)
	}
}

func TestSeleccionarSupervisorAutonomiaOperativoRespetaExclude(t *testing.T) {
	prepararDBTemporal(t)

	if err := RegistrarAgente("CodexSupervisor", "programador"); err != nil {
		t.Fatalf("registrar supervisor: %v", err)
	}
	if err := RegistrarAgente("CodexWorker", "programador"); err != nil {
		t.Fatalf("registrar worker: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "supervision-exclude",
		Nombre:  "supervision-exclude",
		RutaAbs: t.TempDir(),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := UpsertProyectoAutonomia(&ProyectoAutonomia{
		ProyectoID:        proyectoID,
		Enabled:           true,
		SupervisorAgente:  "CodexSupervisor",
		ReserveSupervisor: true,
		EstadoAutonomia:   AutonomiaProyectoActiva,
	}); err != nil {
		t.Fatalf("upsert autonomia: %v", err)
	}
	if err := ActivarAsignacion("CodexSupervisor", proyectoID, "supervision_supervisor"); err != nil {
		t.Fatalf("activar asignacion supervisor: %v", err)
	}
	if err := ActivarAsignacion("CodexWorker", proyectoID, "supervision_worker"); err != nil {
		t.Fatalf("activar asignacion worker: %v", err)
	}

	supervisor, err := SeleccionarSupervisorAutonomiaOperativo(proyectoID, "CodexSupervisor")
	if err != nil {
		t.Fatalf("seleccionar supervisor operativo excluyendo preferido: %v", err)
	}
	if supervisor == nil || supervisor.Nombre != "CodexWorker" {
		t.Fatalf("exclude no respetado: %+v", supervisor)
	}
}

func TestSeleccionarSupervisorAutonomiaOperativoEvitaCuentaCompartidaOcupada(t *testing.T) {
	prepararDBTemporal(t)

	if err := RegistrarAgente("CodexSupervisor", "programador"); err != nil {
		t.Fatalf("registrar supervisor: %v", err)
	}
	if err := RegistrarAgente("CodexWorker", "programador"); err != nil {
		t.Fatalf("registrar worker: %v", err)
	}
	if err := RegistrarAgente("CodexOccupant", "programador"); err != nil {
		t.Fatalf("registrar occupant: %v", err)
	}
	now := time.Now().UTC()
	if err := UpsertAgenteIdentidadObservada("CodexSupervisor", "shared@example.com", "shared", "test", &now); err != nil {
		t.Fatalf("identidad supervisor: %v", err)
	}
	if err := UpsertAgenteIdentidadObservada("CodexOccupant", "shared@example.com", "shared", "test", &now); err != nil {
		t.Fatalf("identidad occupant: %v", err)
	}
	if err := UpsertAgenteIdentidadObservada("CodexWorker", "worker@example.com", "worker", "test", &now); err != nil {
		t.Fatalf("identidad worker: %v", err)
	}

	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "supervision-cuenta-compartida",
		Nombre:  "supervision-cuenta-compartida",
		RutaAbs: t.TempDir(),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := UpsertProyectoAutonomia(&ProyectoAutonomia{
		ProyectoID:        proyectoID,
		Enabled:           true,
		SupervisorAgente:  "CodexSupervisor",
		ReserveSupervisor: true,
		EstadoAutonomia:   AutonomiaProyectoActiva,
	}); err != nil {
		t.Fatalf("upsert autonomia: %v", err)
	}
	if err := ActivarAsignacion("CodexSupervisor", proyectoID, "supervision_supervisor"); err != nil {
		t.Fatalf("activar supervisor: %v", err)
	}
	if err := ActivarAsignacion("CodexWorker", proyectoID, "supervision_worker"); err != nil {
		t.Fatalf("activar worker: %v", err)
	}
	if _, err := IniciarSesionContexto(SesionInicio{
		Agente:            "CodexOccupant",
		ProyectoID:        &proyectoID,
		CWD:               t.TempDir(),
		Herramienta:       "codex-cli",
		ExternalSessionID: "sess-codex-occupant",
	}); err != nil {
		t.Fatalf("iniciar sesion occupant: %v", err)
	}

	supervisor, err := SeleccionarSupervisorAutonomiaOperativo(proyectoID, "")
	if err != nil {
		t.Fatalf("seleccionar supervisor operativo: %v", err)
	}
	if supervisor == nil || supervisor.Nombre == "CodexSupervisor" {
		t.Fatalf("deberia evitar arrancar otro supervisor sobre la misma cuenta ocupada, got=%+v", supervisor)
	}
}

func TestEsSupervisorAutonomiaOperativoDevuelveSupervisorSeleccionado(t *testing.T) {
	prepararDBTemporal(t)

	if err := RegistrarAgente("CodexIdle", "programador"); err != nil {
		t.Fatalf("registrar idle: %v", err)
	}
	if err := RegistrarAgente("CodexActivo", "programador"); err != nil {
		t.Fatalf("registrar activo: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "supervision-es-supervisor",
		Nombre:  "supervision-es-supervisor",
		RutaAbs: t.TempDir(),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := UpsertProyectoAutonomia(&ProyectoAutonomia{
		ProyectoID:        proyectoID,
		Enabled:           true,
		ReserveSupervisor: true,
		EstadoAutonomia:   AutonomiaProyectoActiva,
	}); err != nil {
		t.Fatalf("upsert autonomia: %v", err)
	}
	if err := ActivarAsignacion("CodexIdle", proyectoID, "idle"); err != nil {
		t.Fatalf("activar asignacion idle: %v", err)
	}
	if err := ActivarAsignacion("CodexActivo", proyectoID, "activo"); err != nil {
		t.Fatalf("activar asignacion activo: %v", err)
	}
	if _, err := IniciarSesionContexto(SesionInicio{
		Agente:            "CodexActivo",
		ProyectoID:        &proyectoID,
		CWD:               t.TempDir(),
		Herramienta:       "codex-cli",
		ExternalSessionID: "sess-codex-es-supervisor",
	}); err != nil {
		t.Fatalf("iniciar sesion activa: %v", err)
	}

	ok, supervisor, err := EsSupervisorAutonomiaOperativo(proyectoID, "CodexActivo")
	if err != nil {
		t.Fatalf("resolver es supervisor: %v", err)
	}
	if !ok {
		t.Fatalf("CodexActivo deberia ser supervisor operativo")
	}
	if supervisor == nil || supervisor.Nombre != "CodexActivo" {
		t.Fatalf("supervisor devuelto inesperado: %+v", supervisor)
	}

	ok, supervisor, err = EsSupervisorAutonomiaOperativo(proyectoID, "CodexIdle")
	if err != nil {
		t.Fatalf("resolver es supervisor para idle: %v", err)
	}
	if ok {
		t.Fatalf("CodexIdle no deberia ser supervisor operativo")
	}
	if supervisor == nil || supervisor.Nombre != "CodexActivo" {
		t.Fatalf("supervisor canónico inesperado: %+v", supervisor)
	}
}

func TestAgenteAutonomiaActivoEnProyectoCuentaHandleOperativoSinSesionActiva(t *testing.T) {
	prepararDBTemporal(t)

	if err := RegistrarAgente("CodexActivo", "programador"); err != nil {
		t.Fatalf("registrar activo: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "supervision-handle-activo",
		Nombre:  "supervision-handle-activo",
		RutaAbs: t.TempDir(),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if err := ActivarAsignacion("CodexActivo", proyectoID, "activo"); err != nil {
		t.Fatalf("activar asignacion activo: %v", err)
	}
	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:            "CodexActivo",
		ProyectoID:        &proyectoID,
		CWD:               t.TempDir(),
		Herramienta:       "codex-cli",
		ExternalSessionID: "sess-codex-handle-operativo",
	})
	if err != nil {
		t.Fatalf("iniciar sesion activa: %v", err)
	}
	if _, err := DB.Exec(`UPDATE sesiones SET activa=0, estado='cerrada' WHERE id=?`, sesion.ID); err != nil {
		t.Fatalf("cerrar sesion conservando handle: %v", err)
	}

	ok, err := agenteAutonomiaActivoEnProyecto("CodexActivo", proyectoID)
	if err != nil {
		t.Fatalf("agenteAutonomiaActivoEnProyecto: %v", err)
	}
	if !ok {
		t.Fatal("deberia contar el handle operativo reciente aunque la sesion ya no este activa")
	}
}

func TestSeleccionarSupervisorAutonomiaOperativoConsideraTrabajoActivoConAsignacionPausada(t *testing.T) {
	prepararDBTemporal(t)

	if err := RegistrarAgente("CodexSupervisor", "programador"); err != nil {
		t.Fatalf("registrar supervisor: %v", err)
	}
	if err := RegistrarAgente("CodexBudget", "programador"); err != nil {
		t.Fatalf("registrar worker: %v", err)
	}
	now := time.Now().UTC()
	if err := UpsertAgenteIdentidadObservada("CodexSupervisor", "shared@example.com", "shared", "test", &now); err != nil {
		t.Fatalf("identidad supervisor: %v", err)
	}
	if err := UpsertAgenteIdentidadObservada("CodexBudget", "shared@example.com", "shared", "test", &now); err != nil {
		t.Fatalf("identidad worker: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "supervision-runtime-vivo",
		Nombre:  "supervision-runtime-vivo",
		RutaAbs: t.TempDir(),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := UpsertProyectoAutonomia(&ProyectoAutonomia{
		ProyectoID:        proyectoID,
		Enabled:           true,
		SupervisorAgente:  "CodexSupervisor",
		ReserveSupervisor: true,
		EstadoAutonomia:   AutonomiaProyectoActiva,
	}); err != nil {
		t.Fatalf("upsert autonomia: %v", err)
	}
	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:            "CodexBudget",
		ProyectoID:        &proyectoID,
		CWD:               t.TempDir(),
		Herramienta:       "codex-cli",
		ExternalSessionID: "sess-codex-budget",
	})
	if err != nil {
		t.Fatalf("iniciar sesion activa: %v", err)
	}
	runtime, err := GetRuntimeBySesionID(sesion.ID)
	if err != nil || runtime == nil {
		t.Fatalf("get runtime: %+v err=%v", runtime, err)
	}
	if _, err := DB.Exec(`UPDATE runtime_instances SET logical_state='esperando_io', process_state='running' WHERE id=?`, runtime.ID); err != nil {
		t.Fatalf("actualizar runtime: %v", err)
	}
	if _, err := DB.Exec(`UPDATE sesiones SET activa=0, estado='pausada' WHERE id=?`, sesion.ID); err != nil {
		t.Fatalf("pausar sesion: %v", err)
	}
	if handle, err := GetRuntimeHandleBySesionID(sesion.ID); err != nil || handle == nil {
		t.Fatalf("get handle: %+v err=%v", handle, err)
	} else if _, err := DB.Exec(`UPDATE runtime_handles SET estado='cerrado' WHERE id=?`, handle.ID); err != nil {
		t.Fatalf("cerrar handle: %v", err)
	}
	tareaID, err := CrearTarea(&Tarea{
		Titulo:      "Frente vivo del worker",
		ProyectoID:  &proyectoID,
		Prioridad:   PrioridadAlta,
		CreadoPor:   "tester",
		Descripcion: "runtime vivo con asignacion pausada",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := TomarTarea(tareaID, "CodexBudget"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := IniciarTarea(tareaID, "CodexBudget"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}

	supervisor, err := SeleccionarSupervisorAutonomiaOperativo(proyectoID, "")
	if err != nil {
		t.Fatalf("seleccionar supervisor operativo: %v", err)
	}
	if supervisor == nil || supervisor.Nombre != "CodexBudget" {
		t.Fatalf("deberia usar el worker realmente vivo como supervisor efectivo: %+v", supervisor)
	}
}
