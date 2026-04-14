package db

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

func TestRegistrarMuestraGenericProcessPrefiereEstadoTMUXReady(t *testing.T) {
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

	workdir := filepath.Join(tmp, "orquestador")
	if err := os.MkdirAll(workdir, 0o755); err != nil {
		t.Fatalf("mkdir workdir: %v", err)
	}
	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:             "Codex1",
		ProyectoID:         &proyectoID,
		CWD:                workdir,
		Herramienta:        "codex-cli",
		ExternalSessionID:  "sess-generic-process-tmux",
		ResumenContinuidad: "generic process tmux",
		Branch:             "main",
		PID:                int64Ptr(t, int64(os.Getpid())),
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}

	handle, err := GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("get handle: %+v err=%v", handle, err)
	}
	statusPath := filepath.Join(tmp, "worker-generic-status.json")
	heartbeatPath := filepath.Join(tmp, "worker-generic-heartbeat.json")
	now := time.Now().UTC()
	statusRaw, _ := json.Marshal(map[string]any{
		"state":      "ready",
		"alive":      true,
		"updated_at": now.Format(time.RFC3339Nano),
		"ready_at":   now.Format(time.RFC3339Nano),
	})
	heartbeatRaw, _ := json.Marshal(map[string]any{
		"alive":        true,
		"heartbeat_at": now.Format(time.RFC3339Nano),
		"ready_at":     now.Format(time.RFC3339Nano),
	})
	if err := os.WriteFile(statusPath, append(statusRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, append(heartbeatRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}
	metaJSON, _ := json.Marshal(map[string]any{
		"driver":                "tmux_cli_session",
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
	})
	if _, err := DB.Exec(`
		UPDATE runtime_handles
		SET transporte='tmux',
		    handle_kind='session',
		    metadata_json=?
		WHERE id=?`, string(metaJSON), handle.ID); err != nil {
		t.Fatalf("update handle tmux: %v", err)
	}

	runtimeInst, err := GetRuntimeBySesionID(sesion.ID)
	if err != nil {
		t.Fatalf("get runtime: %v", err)
	}
	if runtimeInst == nil {
		t.Fatal("runtime nil")
	}

	sample, err := RegistrarMuestraGenericProcess(runtimeInst.ID, runtimeInst)
	if err != nil {
		t.Fatalf("registrar muestra generic process tmux: %v", err)
	}
	if sample == nil {
		t.Fatal("sample nil")
	}
	if sample.LogicalState != "activo" {
		t.Fatalf("logical_state de muestra inesperado: %s", sample.LogicalState)
	}

	runtimeInst, err = GetRuntime(runtimeInst.ID)
	if err != nil || runtimeInst == nil {
		t.Fatalf("get runtime actualizado: %+v err=%v", runtimeInst, err)
	}
	if runtimeInst.LogicalState != "activo" {
		t.Fatalf("logical_state runtime inesperado: %s", runtimeInst.LogicalState)
	}
	if runtimeInst.ProcessState != "running" {
		t.Fatalf("process_state runtime inesperado: %s", runtimeInst.ProcessState)
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
