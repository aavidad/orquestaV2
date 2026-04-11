package db

import (
	"os"
	"path/filepath"
	"testing"
)

func abrirDBTemporalRuntimeObservabilidad(t *testing.T) {
	t.Helper()

	resetRuntimeTranscriptHotIdleState()
	t.Cleanup(resetRuntimeTranscriptHotIdleState)

	if DB != nil {
		Close()
	}

	prev := os.Getenv("ORQUESTA_DB")
	ruta := filepath.Join(t.TempDir(), "orquesta.db")
	if err := os.Setenv("ORQUESTA_DB", ruta); err != nil {
		t.Fatalf("setenv ORQUESTA_DB: %v", err)
	}
	t.Cleanup(func() {
		Close()
		if prev == "" {
			_ = os.Unsetenv("ORQUESTA_DB")
			return
		}
		_ = os.Setenv("ORQUESTA_DB", prev)
	})

	if err := Open(); err != nil {
		t.Fatalf("Open: %v", err)
	}
}

func TestRegistrarRuntimeInstanceYUltimaActividad(t *testing.T) {
	abrirDBTemporalRuntimeObservabilidad(t)

	id1, err := RegistrarRuntimeInstance(&RuntimeInstance{
		Agente:            "codex1",
		Provider:          "openai",
		Connector:         "codex-cli",
		ExternalSessionID: "ext-a",
		LogicalState:      "disponible",
		ProcessState:      "vivo",
		ChildCount:        0,
		ThreadCount:       1,
		Model:             "gpt-5",
		TaskProfile:       "implementacion",
		CWD:               "/tmp/a",
		Branch:            "main",
	})
	if err != nil {
		t.Fatalf("RegistrarRuntimeInstance A: %v", err)
	}

	if _, err := RegistrarRuntimeEvent(&RuntimeEvent{
		RuntimeID:   id1,
		Kind:        "heartbeat",
		Level:       "info",
		Message:     "activo",
		PayloadJSON: "{}",
	}); err != nil {
		t.Fatalf("RegistrarRuntimeEvent A: %v", err)
	}

	id2, err := RegistrarRuntimeInstance(&RuntimeInstance{
		Agente:            "codex2",
		Provider:          "openai",
		Connector:         "codex-cli",
		ExternalSessionID: "ext-b",
		LogicalState:      "pensando",
		ProcessState:      "vivo",
		ChildCount:        1,
		ThreadCount:       2,
		Model:             "gpt-5",
		TaskProfile:       "arquitectura",
		CWD:               "/tmp/b",
		Branch:            "main",
	})
	if err != nil {
		t.Fatalf("RegistrarRuntimeInstance B: %v", err)
	}

	sampleID, err := RegistrarRuntimeTelemetrySample(&RuntimeTelemetrySample{
		RuntimeID:    id2,
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
		t.Fatalf("RegistrarRuntimeTelemetrySample B: %v", err)
	}
	if sampleID <= 0 {
		t.Fatalf("sampleID invalido: %d", sampleID)
	}

	sample, err := UltimaMuestraRuntime(id2)
	if err != nil {
		t.Fatalf("UltimaMuestraRuntime: %v", err)
	}
	if sample == nil || sample.ID != sampleID {
		t.Fatalf("ultima muestra inesperada: %+v", sample)
	}
	if sample.Source != "generic_process" || sample.LogicalState != "pensando" {
		t.Fatalf("muestra inesperada: %+v", sample)
	}

	runtimes, err := ListarRuntimesConUltimaActividad()
	if err != nil {
		t.Fatalf("ListarRuntimesConUltimaActividad: %v", err)
	}
	if len(runtimes) != 2 {
		t.Fatalf("se esperaban 2 runtimes, got %d", len(runtimes))
	}
	if runtimes[0].Agente != "codex2" {
		t.Fatalf("se esperaba codex2 como runtime más reciente, got %+v", runtimes[0])
	}
	if runtimes[0].UltimaActividadAt == nil {
		t.Fatalf("ultima actividad no informada: %+v", runtimes[0])
	}
}

func TestRegistrarRuntimeInstanceActualizaPorID(t *testing.T) {
	abrirDBTemporalRuntimeObservabilidad(t)

	id, err := RegistrarRuntimeInstance(&RuntimeInstance{
		Agente:            "codex1",
		Provider:          "openai",
		Connector:         "codex-cli",
		ExternalSessionID: "ext-reemplazo",
		LogicalState:      "arrancando",
		ProcessState:      "desconocido",
		ChildCount:        0,
		ThreadCount:       1,
		Model:             "gpt-5",
		TaskProfile:       "implementacion",
		CWD:               "/tmp/original",
		Branch:            "main",
	})
	if err != nil {
		t.Fatalf("RegistrarRuntimeInstance inicial: %v", err)
	}

	id2, err := RegistrarRuntimeInstance(&RuntimeInstance{
		ID:                id,
		Agente:            "codex1",
		Provider:          "openai",
		Connector:         "codex-cli",
		ExternalSessionID: "ext-actualizado",
		LogicalState:      "pensando",
		ProcessState:      "vivo",
		ChildCount:        2,
		ThreadCount:       4,
		Model:             "gpt-5",
		TaskProfile:       "arquitectura",
		CWD:               "/tmp/actualizado",
		Branch:            "feature/test",
	})
	if err != nil {
		t.Fatalf("RegistrarRuntimeInstance actualización: %v", err)
	}
	if id2 != id {
		t.Fatalf("la actualización deberia conservar el ID: %d vs %d", id, id2)
	}

	runtimes, err := ListarRuntimesConUltimaActividad()
	if err != nil {
		t.Fatalf("ListarRuntimesConUltimaActividad: %v", err)
	}
	if len(runtimes) != 1 {
		t.Fatalf("se esperaba 1 runtime, got %d", len(runtimes))
	}
	if runtimes[0].ExternalSessionID != "ext-actualizado" {
		t.Fatalf("no se actualizo la instancia: %+v", runtimes[0])
	}
	if runtimes[0].ChildCount != 2 || runtimes[0].ThreadCount != 4 {
		t.Fatalf("no se actualizo la telemetria basica: %+v", runtimes[0])
	}
}
