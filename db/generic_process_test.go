package db

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTomarMuestraGenericProcessProcesoActual(t *testing.T) {
	sample, err := TomarMuestraGenericProcess(&RuntimeInstance{
		PID: int64Ptr(t, int64(os.Getpid())),
	})
	if err != nil {
		t.Fatalf("tomar muestra: %v", err)
	}
	if sample == nil {
		t.Fatalf("sample nil")
	}
	if sample.Source != "generic_process" {
		t.Fatalf("source inesperado: %s", sample.Source)
	}
	if strings.TrimSpace(sample.LogicalState) == "" {
		t.Fatalf("logical_state vacio")
	}
	if sample.ThreadCount <= 0 {
		t.Fatalf("thread_count no valido: %d", sample.ThreadCount)
	}
	if sample.SampleJSON == "" || !strings.Contains(sample.SampleJSON, "\"pid\"") {
		t.Fatalf("sample_json inesperado: %s", sample.SampleJSON)
	}
}

func TestRegistrarMuestraGenericProcessPersisteYActualizaRuntime(t *testing.T) {
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
		ExternalSessionID:  "sess-generic-process",
		ResumenContinuidad: "generic process",
		Branch:             "main",
		PID:                int64Ptr(t, int64(os.Getpid())),
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}

	runtime, err := GetRuntimeBySesionID(sesion.ID)
	if err != nil {
		t.Fatalf("get runtime: %v", err)
	}
	if runtime == nil {
		t.Fatalf("runtime nil")
	}

	sample, err := RegistrarMuestraGenericProcess(runtime.ID, runtime)
	if err != nil {
		t.Fatalf("registrar muestra generic process: %v", err)
	}
	if sample == nil {
		t.Fatalf("sample nil")
	}
	if sample.Source != "generic_process" {
		t.Fatalf("source inesperado: %s", sample.Source)
	}

	runtime, err = GetRuntime(runtime.ID)
	if err != nil {
		t.Fatalf("get runtime actualizado: %v", err)
	}
	if runtime.PID == nil {
		t.Fatalf("pid no actualizado")
	}
	if runtime.ThreadCount <= 0 {
		t.Fatalf("thread_count no actualizado: %d", runtime.ThreadCount)
	}
	if strings.TrimSpace(runtime.LogicalState) == "" {
		t.Fatalf("logical_state no actualizado")
	}

	muestras, err := ListarMuestrasRuntime(runtime.ID, 10)
	if err != nil {
		t.Fatalf("listar muestras: %v", err)
	}
	if len(muestras) < 2 {
		t.Fatalf("esperaba al menos 2 muestras, got=%d", len(muestras))
	}
	if muestras[0].Source != "generic_process" {
		t.Fatalf("muestra más reciente inesperada: %+v", muestras[0])
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(sample.SampleJSON), &payload); err != nil {
		t.Fatalf("json muestra: %v", err)
	}
	if _, ok := payload["pid"]; !ok {
		t.Fatalf("json sin pid: %v", payload)
	}
}

func TestTomarMuestraGenericProcessSinPIDFalla(t *testing.T) {
	if _, err := TomarMuestraGenericProcess(&RuntimeInstance{}); err == nil {
		t.Fatalf("esperaba error sin PID real")
	}
}

func int64Ptr(t *testing.T, v int64) *int64 {
	t.Helper()
	return &v
}
