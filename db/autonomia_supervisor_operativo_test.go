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
