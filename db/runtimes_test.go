package db

import (
	"path/filepath"
	"testing"
	"time"
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
	if runtime.LogicalState == "" {
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

func TestObservabilidadRuntimeConUltimaActividad(t *testing.T) {
	tmp := prepararDBTemporalInspeccionSesiones(t)

	for _, agente := range []string{"Codex1", "Codex2"} {
		if err := RegistrarAgente(agente, "programador"); err != nil {
			t.Fatalf("registrar agente %s: %v", agente, err)
		}
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

	sesion1, err := IniciarSesionContexto(SesionInicio{
		Agente:            "Codex1",
		ProyectoID:        &proyectoID,
		CWD:               filepath.Join(tmp, "orquestador"),
		Herramienta:       "codex-cli",
		ExternalSessionID: "sess-obs-1",
		Branch:            "main",
	})
	if err != nil {
		t.Fatalf("iniciar sesion 1: %v", err)
	}
	sesion2, err := IniciarSesionContexto(SesionInicio{
		Agente:            "Codex2",
		ProyectoID:        &proyectoID,
		CWD:               filepath.Join(tmp, "orquestador"),
		Herramienta:       "codex-cli",
		ExternalSessionID: "sess-obs-2",
		Branch:            "main",
	})
	if err != nil {
		t.Fatalf("iniciar sesion 2: %v", err)
	}

	runtime1, err := GetRuntimeBySesionID(sesion1.ID)
	if err != nil || runtime1 == nil {
		t.Fatalf("runtime1: %+v err=%v", runtime1, err)
	}
	runtime2, err := GetRuntimeBySesionID(sesion2.ID)
	if err != nil || runtime2 == nil {
		t.Fatalf("runtime2: %+v err=%v", runtime2, err)
	}

	if _, err := RegistrarRuntimeEvent(&RuntimeEvent{
		RuntimeID:   runtime1.ID,
		Kind:        "heartbeat",
		Level:       "info",
		Message:     "activo",
		PayloadJSON: "{}",
	}); err != nil {
		t.Fatalf("registrar evento runtime1: %v", err)
	}

	sampleID, err := RegistrarRuntimeTelemetrySample(&RuntimeTelemetrySample{
		RuntimeID:    runtime2.ID,
		CPUPct:       12.5,
		MemBytes:     2048,
		RSSBytes:     1024,
		OpenFDs:      7,
		ChildCount:   1,
		ThreadCount:  2,
		LogicalState: "pensando",
		Source:       "generic_process",
		SampleJSON:   sampleJSONConFuente("generic_process"),
	})
	if err != nil {
		t.Fatalf("registrar muestra runtime2: %v", err)
	}
	if sampleID <= 0 {
		t.Fatalf("sampleID invalido: %d", sampleID)
	}

	sample, err := UltimaMuestraRuntime(runtime2.ID)
	if err != nil {
		t.Fatalf("ultima muestra runtime2: %v", err)
	}
	if sample == nil || sample.ID != sampleID {
		t.Fatalf("ultima muestra inesperada: %+v", sample)
	}
	if sample.Source != "generic_process" || sample.LogicalState != "pensando" {
		t.Fatalf("sample inesperada: %+v", sample)
	}
	normalizada, err := NormalizarMuestraRuntime(sample)
	if err != nil {
		t.Fatalf("normalizar muestra runtime2: %v", err)
	}
	if normalizada.Provider != "generic_process" {
		t.Fatalf("muestra normalizada inesperada: %+v", normalizada)
	}

	principal, err := RuntimePrincipalAgente("Codex2")
	if err != nil {
		t.Fatalf("runtime principal: %v", err)
	}
	if principal == nil || principal.ID != runtime2.ID {
		t.Fatalf("runtime principal inesperado: %+v", principal)
	}

	runtimes, err := ListarRuntimesConUltimaActividad()
	if err != nil {
		t.Fatalf("listar runtimes con actividad: %v", err)
	}
	if len(runtimes) != 2 {
		t.Fatalf("se esperaban 2 runtimes, got=%d", len(runtimes))
	}
	if runtimes[0].Agente != "Codex2" {
		t.Fatalf("se esperaba Codex2 como runtime mas reciente, got %+v", runtimes[0])
	}
	if runtimes[0].UltimaActividadAt == nil {
		t.Fatalf("ultima actividad no informada: %+v", runtimes[0])
	}
}

func TestUltimaActividadPrefiereMuestraMasRecienteQueEvento(t *testing.T) {
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
		Agente:            "Codex1",
		ProyectoID:        &proyectoID,
		CWD:               filepath.Join(tmp, "orquestador"),
		Herramienta:       "codex-cli",
		ExternalSessionID: "sess-obs-order",
		Branch:            "main",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}

	runtime, err := GetRuntimeBySesionID(sesion.ID)
	if err != nil || runtime == nil {
		t.Fatalf("runtime: %+v err=%v", runtime, err)
	}

	if _, err := RegistrarRuntimeEvent(&RuntimeEvent{
		RuntimeID:   runtime.ID,
		Kind:        "heartbeat",
		Level:       "info",
		Message:     "activo",
		PayloadJSON: "{}",
	}); err != nil {
		t.Fatalf("registrar evento: %v", err)
	}

	time.Sleep(1100 * time.Millisecond)

	sampleID, err := RegistrarRuntimeTelemetrySample(&RuntimeTelemetrySample{
		RuntimeID:    runtime.ID,
		CPUPct:       8.5,
		MemBytes:     1024,
		RSSBytes:     768,
		OpenFDs:      5,
		ChildCount:   0,
		ThreadCount:  1,
		LogicalState: "pensando",
		Source:       "generic_process",
		SampleJSON:   sampleJSONConFuente("generic_process"),
	})
	if err != nil {
		t.Fatalf("registrar muestra: %v", err)
	}

	sample, err := UltimaMuestraRuntime(runtime.ID)
	if err != nil {
		t.Fatalf("ultima muestra: %v", err)
	}
	if sample == nil || sample.ID != sampleID {
		t.Fatalf("sample inesperada: %+v", sample)
	}

	runtimes, err := ListarRuntimesConUltimaActividad()
	if err != nil {
		t.Fatalf("listar runtimes con actividad: %v", err)
	}
	if len(runtimes) != 1 {
		t.Fatalf("se esperaba 1 runtime, got=%d", len(runtimes))
	}
	if runtimes[0].UltimaActividadAt == nil {
		t.Fatalf("ultima actividad ausente: %+v", runtimes[0])
	}
	if runtimes[0].UltimaActividadAt.Before(sample.CreatedAt) {
		t.Fatalf("ultima actividad no refleja la muestra mas reciente: actividad=%v sample=%v", *runtimes[0].UltimaActividadAt, sample.CreatedAt)
	}
}
