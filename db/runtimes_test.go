package db

import (
	"path/filepath"
	"testing"
)

func TestRuntimesSeSincronizanDesdeSesiones(t *testing.T) {
	tmp := prepararDBTemporalInspeccionSesiones(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
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

	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:             "Codex1",
		ProyectoID:         &proyectoID,
		CWD:                filepath.Join(tmp, "orquestador"),
		Herramienta:        "codex-cli",
		ExternalSessionID:  "sess-runtime-001",
		ResumenContinuidad: "runtime test",
		Branch:             "main",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}

	runtime, err := GetRuntimeBySesionID(sesion.ID)
	if err != nil {
		t.Fatalf("get runtime por sesion: %v", err)
	}
	if runtime == nil {
		t.Fatalf("runtime nil")
	}
	if runtime.Agente != "Codex1" || runtime.Provider != "openai" {
		t.Fatalf("runtime inesperado: %+v", runtime)
	}
	if runtime.LogicalState != "disponible" {
		t.Fatalf("logical_state inesperado: %s", runtime.LogicalState)
	}

	pid := int64(4321)
	host := "host-runtime"
	if err := GuardarSesionActiva("Codex1", &proyectoID, SesionUpdate{
		PID:       &pid,
		Host:      &host,
		Heartbeat: true,
	}); err != nil {
		t.Fatalf("guardar sesion: %v", err)
	}

	runtimes, err := ListarRuntimes(FiltroRuntimes{})
	if err != nil {
		t.Fatalf("listar runtimes: %v", err)
	}
	if len(runtimes) != 1 {
		t.Fatalf("esperaba 1 runtime, got=%d", len(runtimes))
	}
	if runtimes[0].PID == nil || *runtimes[0].PID != pid {
		t.Fatalf("pid no sincronizado: %+v", runtimes[0])
	}

	muestras, err := ListarMuestrasRuntime(runtime.ID, 10)
	if err != nil {
		t.Fatalf("listar muestras: %v", err)
	}
	if len(muestras) == 0 {
		t.Fatalf("esperaba al menos una muestra de runtime")
	}

	if err := FinSesion("Codex1"); err != nil {
		t.Fatalf("fin sesion: %v", err)
	}
	runtime, err = GetRuntime(runtime.ID)
	if err != nil {
		t.Fatalf("get runtime final: %v", err)
	}
	if runtime.LogicalState != "cerrado" || runtime.ProcessState != "finalizado" {
		t.Fatalf("runtime no cerrado tras fin de sesion: %+v", runtime)
	}
}
