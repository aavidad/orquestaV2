package db

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestIngestarRuntimeTranscriptHandleClasificaYGeneraEventos(t *testing.T) {
	abrirDBTemporalRuntimeObservabilidad(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(t.TempDir(), "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(t.TempDir(), "orquestador"),
		Herramienta: "codex-cli",
		Branch:      "main",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	runtime, err := GetRuntimeBySesionID(sesion.ID)
	if err != nil || runtime == nil {
		t.Fatalf("runtime: %+v err=%v", runtime, err)
	}
	handle, err := GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}

	logPath := filepath.Join(t.TempDir(), "codex.log")
	logData := strings.Join([]string{
		"Script started on 2026-03-27",
		"¿me dejas hacer este cambio?",
		"Quedo a la espera de tu confirmación",
		"",
	}, "\n")
	if err := os.WriteFile(logPath, []byte(logData), 0o600); err != nil {
		t.Fatalf("write log: %v", err)
	}
	metaJSON, _ := json.Marshal(map[string]any{"log_path": logPath})
	if _, err := DB.Exec(`UPDATE runtime_handles SET metadata_json=? WHERE id=?`, string(metaJSON), handle.ID); err != nil {
		t.Fatalf("update handle metadata: %v", err)
	}

	n, err := IngestarRuntimeTranscriptHandle(handle.ID)
	if err != nil {
		t.Fatalf("ingestar transcript: %v", err)
	}
	if n != 3 {
		t.Fatalf("esperaba 3 líneas ingeridas, got=%d", n)
	}
	agente := "Codex1"
	items, err := ListarRuntimeTranscript(FiltroRuntimeTranscript{Agente: &agente, Limit: 10})
	if err != nil {
		t.Fatalf("listar transcript: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("transcript inesperado: %+v", items)
	}
	var approval, waiting bool
	for _, item := range items {
		if item == nil {
			continue
		}
		switch item.Classification {
		case "approval_request":
			approval = true
		case "waiting_human":
			waiting = true
		}
	}
	if !approval || !waiting {
		t.Fatalf("clasificación inesperada: %+v", items)
	}
	var count int
	if err := DB.QueryRow(`SELECT COUNT(*) FROM runtime_events WHERE runtime_id=? AND kind IN ('approval_request','waiting_human')`, runtime.ID).Scan(&count); err != nil {
		t.Fatalf("count runtime_events: %v", err)
	}
	if count != 2 {
		t.Fatalf("runtime_events derivados inesperados: %d", count)
	}

	n, err = IngestarRuntimeTranscriptHandle(handle.ID)
	if err != nil {
		t.Fatalf("reingestar transcript: %v", err)
	}
	if n != 0 {
		t.Fatalf("no debería reingestar líneas ya procesadas, got=%d", n)
	}
}

func TestRegistrarRuntimeTranscriptInputPersisteLinea(t *testing.T) {
	abrirDBTemporalRuntimeObservabilidad(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(t.TempDir(), "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(t.TempDir(), "orquestador"),
		Herramienta: "codex-cli",
		Branch:      "main",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	runtime, _ := GetRuntimeBySesionID(sesion.ID)
	handle, _ := GetRuntimeHandleBySesionID(sesion.ID)
	if err := RegistrarRuntimeTranscriptInput(handle, runtime, "continúa automáticamente"); err != nil {
		t.Fatalf("registrar transcript input: %v", err)
	}
	agente := "Codex1"
	items, err := ListarRuntimeTranscript(FiltroRuntimeTranscript{Agente: &agente, Limit: 10})
	if err != nil {
		t.Fatalf("listar transcript: %v", err)
	}
	if len(items) != 1 || items[0].Stream != "stdin" || items[0].Text != "continúa automáticamente" {
		t.Fatalf("entrada transcript inesperada: %+v", items)
	}
}

func TestIngestarRuntimeTranscriptHandleNoClasificaLineaSistemaScript(t *testing.T) {
	abrirDBTemporalRuntimeObservabilidad(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(t.TempDir(), "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(t.TempDir(), "orquestador"),
		Herramienta: "codex-cli",
		Branch:      "main",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	runtime, _ := GetRuntimeBySesionID(sesion.ID)
	handle, _ := GetRuntimeHandleBySesionID(sesion.ID)

	logPath := filepath.Join(t.TempDir(), "system.log")
	logData := strings.Join([]string{
		`Script started on 2026-03-31 13:18:00+02:00 [COMMAND="'/tmp/codex-perfiles/bin/codex-perfil' 'Codex1' 'exec'"]`,
		"",
	}, "\n")
	if err := os.WriteFile(logPath, []byte(logData), 0o600); err != nil {
		t.Fatalf("write log: %v", err)
	}
	metaJSON, _ := json.Marshal(map[string]any{"log_path": logPath})
	if _, err := DB.Exec(`UPDATE runtime_handles SET metadata_json=? WHERE id=?`, string(metaJSON), handle.ID); err != nil {
		t.Fatalf("update handle metadata: %v", err)
	}

	n, err := IngestarRuntimeTranscriptHandle(handle.ID)
	if err != nil {
		t.Fatalf("ingestar transcript: %v", err)
	}
	if n != 1 {
		t.Fatalf("esperaba 1 linea de sistema ingerida, got=%d", n)
	}

	agente := "Codex1"
	items, err := ListarRuntimeTranscript(FiltroRuntimeTranscript{Agente: &agente, Limit: 10})
	if err != nil {
		t.Fatalf("listar transcript: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("transcript inesperado: %+v", items)
	}
	if items[0].Stream != "system" || items[0].Classification != "" {
		t.Fatalf("la linea de sistema no deberia clasificarse como fallo: %+v", items[0])
	}
	var count int
	if err := DB.QueryRow(`SELECT COUNT(*) FROM runtime_events WHERE runtime_id=?`, runtime.ID).Scan(&count); err != nil {
		t.Fatalf("count runtime_events: %v", err)
	}
	if count != 0 {
		t.Fatalf("no deberia generar runtime_events para el banner de script, got=%d", count)
	}
}

func TestIngestarRuntimeTranscriptActivosFiltraHandlesSinLogPath(t *testing.T) {
	abrirDBTemporalRuntimeObservabilidad(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente 1: %v", err)
	}
	if err := RegistrarAgente("Codex2", "programador"); err != nil {
		t.Fatalf("registrar agente 2: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(t.TempDir(), "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}

	sesion1, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(t.TempDir(), "orquestador1"),
		Herramienta: "codex-cli",
		Branch:      "main",
	})
	if err != nil {
		t.Fatalf("iniciar sesion 1: %v", err)
	}
	handle1, err := GetRuntimeHandleBySesionID(sesion1.ID)
	if err != nil || handle1 == nil {
		t.Fatalf("handle 1: %+v err=%v", handle1, err)
	}

	sesion2, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Codex2",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(t.TempDir(), "orquestador2"),
		Herramienta: "codex-cli",
		Branch:      "main",
	})
	if err != nil {
		t.Fatalf("iniciar sesion 2: %v", err)
	}
	handle2, err := GetRuntimeHandleBySesionID(sesion2.ID)
	if err != nil || handle2 == nil {
		t.Fatalf("handle 2: %+v err=%v", handle2, err)
	}

	logPath := filepath.Join(t.TempDir(), "codex1.log")
	if err := os.WriteFile(logPath, []byte("Necesito tu aprobación\n"), 0o600); err != nil {
		t.Fatalf("write log: %v", err)
	}
	metaJSON, _ := json.Marshal(map[string]any{"log_path": logPath})
	if _, err := DB.Exec(`UPDATE runtime_handles SET metadata_json=? WHERE id=?`, string(metaJSON), handle1.ID); err != nil {
		t.Fatalf("update handle1 metadata: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_handles SET metadata_json='{}' WHERE id=?`, handle2.ID); err != nil {
		t.Fatalf("update handle2 metadata: %v", err)
	}

	n, err := IngestarRuntimeTranscriptActivos()
	if err != nil {
		t.Fatalf("ingestar transcript activos: %v", err)
	}
	if n != 1 {
		t.Fatalf("solo deberia ingerir el handle con log_path, got=%d", n)
	}
}

func TestIngestarRuntimeTranscriptActivosDifiereHandleFallidoEstable(t *testing.T) {
	abrirDBTemporalRuntimeObservabilidad(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(t.TempDir(), "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(t.TempDir(), "orquestador"),
		Herramienta: "codex-cli",
		Branch:      "main",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}

	logPath := filepath.Join(t.TempDir(), "fallido.log")
	if err := os.WriteFile(logPath, []byte("thread 'main' panicked at src/ui.rs:1:1\n"), 0o600); err != nil {
		t.Fatalf("write log: %v", err)
	}
	metaJSON, _ := json.Marshal(map[string]any{"log_path": logPath})
	if _, err := DB.Exec(`UPDATE runtime_handles SET estado='fallido', metadata_json=? WHERE id=?`, string(metaJSON), handle.ID); err != nil {
		t.Fatalf("update handle metadata: %v", err)
	}

	n, err := IngestarRuntimeTranscriptActivos()
	if err != nil {
		t.Fatalf("primer ingestado transcript: %v", err)
	}
	if n != 1 {
		t.Fatalf("esperaba 1 línea ingerida en el handle fallido, got=%d", n)
	}
	n, err = IngestarRuntimeTranscriptActivos()
	if err != nil {
		t.Fatalf("segundo ingestado transcript: %v", err)
	}
	if n != 0 {
		t.Fatalf("no debería reingestar el handle fallido estable, got=%d", n)
	}

	blockedDir := filepath.Join(t.TempDir(), "blocked")
	if err := os.Mkdir(blockedDir, 0o700); err != nil {
		t.Fatalf("mkdir blocked: %v", err)
	}
	if err := os.Chmod(blockedDir, 0o000); err != nil {
		t.Fatalf("chmod blocked: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(blockedDir, 0o700)
	})

	blockedMetaJSON, _ := json.Marshal(map[string]any{
		"log_path": filepath.Join(blockedDir, "denied.log"),
	})
	if _, err := DB.Exec(`UPDATE runtime_handles SET metadata_json=? WHERE id=?`, string(blockedMetaJSON), handle.ID); err != nil {
		t.Fatalf("update blocked handle metadata: %v", err)
	}

	n, err = IngestarRuntimeTranscriptActivos()
	if err != nil {
		t.Fatalf("el cooldown en memoria debería evitar re-sondear el handle fallido estable: %v", err)
	}
	if n != 0 {
		t.Fatalf("el handle fallido estable no debería aportar trabajo mientras sigue en cooldown, got=%d", n)
	}

	resetRuntimeTranscriptHotIdleState()
	if _, err := IngestarRuntimeTranscriptActivos(); err == nil {
		t.Fatalf("sin cooldown en memoria el log_path inaccesible debería volver a aflorar error")
	}
}

func TestListarRuntimeHandlesParaPresupuestoFiltraSoloOperativosConMetadata(t *testing.T) {
	abrirDBTemporalRuntimeObservabilidad(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente 1: %v", err)
	}
	if err := RegistrarAgente("Codex2", "programador"); err != nil {
		t.Fatalf("registrar agente 2: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(t.TempDir(), "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}

	sesion1, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(t.TempDir(), "orquestador1"),
		Herramienta: "codex-cli",
		Branch:      "main",
	})
	if err != nil {
		t.Fatalf("iniciar sesion 1: %v", err)
	}
	handle1, err := GetRuntimeHandleBySesionID(sesion1.ID)
	if err != nil || handle1 == nil {
		t.Fatalf("handle 1: %+v err=%v", handle1, err)
	}
	if _, err := DB.Exec(`UPDATE runtime_handles SET metadata_json='{\"working_dir\":\"/tmp/codex1\"}' WHERE id=?`, handle1.ID); err != nil {
		t.Fatalf("update handle1 metadata: %v", err)
	}

	sesion2, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Codex2",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(t.TempDir(), "orquestador2"),
		Herramienta: "codex-cli",
		Branch:      "main",
	})
	if err != nil {
		t.Fatalf("iniciar sesion 2: %v", err)
	}
	handle2, err := GetRuntimeHandleBySesionID(sesion2.ID)
	if err != nil || handle2 == nil {
		t.Fatalf("handle 2: %+v err=%v", handle2, err)
	}
	if _, err := DB.Exec(`UPDATE runtime_handles SET metadata_json='' WHERE id=?`, handle2.ID); err != nil {
		t.Fatalf("update handle2 metadata: %v", err)
	}

	if _, err := DB.Exec(`UPDATE runtime_handles SET estado='cerrado' WHERE id=?`, handle2.ID); err != nil {
		t.Fatalf("cerrar handle2: %v", err)
	}

	handles, err := ListarRuntimeHandlesParaPresupuesto()
	if err != nil {
		t.Fatalf("listar handles para presupuesto: %v", err)
	}
	if len(handles) != 1 || handles[0].ID != handle1.ID {
		t.Fatalf("handles para presupuesto inesperados: %+v", handles)
	}
}

func TestListarRuntimeHandlesParaPresupuestoPrefiereCanonicoTMUXSobreLegacy(t *testing.T) {
	abrirDBTemporalRuntimeObservabilidad(t)

	if err := RegistrarAgente("CodexBudget", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(t.TempDir(), "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}

	pid := int64(os.Getpid())
	runtimeLegacyID, err := RegistrarRuntimeInstance(&RuntimeInstance{
		Agente:       "CodexBudget",
		ProyectoID:   &proyectoID,
		LogicalState: "esperando_io",
		ProcessState: "running",
		PID:          &pid,
	})
	if err != nil {
		t.Fatalf("crear runtime legacy: %v", err)
	}
	legacyMetaJSON, _ := json.Marshal(map[string]any{
		"driver":           "process_pty_cli",
		"working_dir":      filepath.Join(t.TempDir(), "legacy"),
		"rendered_command": "codex-perfil CodexBudget --model gpt-5.4",
	})
	if _, err := DB.Exec(`INSERT INTO runtime_handles (
		agente, proyecto_id, runtime_id, transporte, handle_kind, handle_ref, estado, metadata_json, last_seen_at
	) VALUES (?,?,?,?,?,?, 'activo', ?, ?)`,
		"CodexBudget", proyectoID, runtimeLegacyID, "cli", "process", strconv.Itoa(os.Getpid()), string(legacyMetaJSON), time.Now().UTC()); err != nil {
		t.Fatalf("insert handle legacy: %v", err)
	}

	runDir := filepath.Join(t.TempDir(), "worker")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir worker: %v", err)
	}
	manifestPath := filepath.Join(runDir, "manifest.json")
	statusPath := filepath.Join(runDir, "status.json")
	heartbeatPath := filepath.Join(runDir, "heartbeat.json")
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if err := os.WriteFile(manifestPath, []byte(`{"version":1,"agent":"CodexBudget","driver":"tmux_cli_session","transport":"tmux","tmux_session":"orq-codexbudget-1","tmux_pane_id":"%17","status_path":"`+statusPath+`","heartbeat_path":"`+heartbeatPath+`"}`+"\n"), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(statusPath, []byte(`{"state":"running","updated_at":"`+now+`","alive":true,"child_pid":`+strconv.Itoa(os.Getpid())+`}`+"\n"), 0o600); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, []byte(`{"alive":true,"heartbeat_at":"`+now+`","started_at":"`+now+`","child_pid":`+strconv.Itoa(os.Getpid())+`}`+"\n"), 0o600); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}

	runtimeTMUXID, err := RegistrarRuntimeInstance(&RuntimeInstance{
		Agente:       "CodexBudget",
		ProyectoID:   &proyectoID,
		LogicalState: "esperando_io",
		ProcessState: "running",
		PID:          &pid,
	})
	if err != nil {
		t.Fatalf("crear runtime tmux: %v", err)
	}
	tmuxMetaJSON, _ := json.Marshal(map[string]any{
		"driver":                "tmux_cli_session",
		"tmux_session":          "orq-codexbudget-1",
		"tmux_pane_id":          "%17",
		"worker_manifest_path":  manifestPath,
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
		"working_dir":           filepath.Join(runDir, "cwd"),
	})
	res, err := DB.Exec(`INSERT INTO runtime_handles (
		agente, proyecto_id, runtime_id, transporte, handle_kind, handle_ref, estado, metadata_json, last_seen_at
	) VALUES (?,?,?,?,?,?, 'activo', ?, ?)`,
		"CodexBudget", proyectoID, runtimeTMUXID, "tmux", "process", "orq-codexbudget-1/%17", string(tmuxMetaJSON), time.Now().UTC())
	if err != nil {
		t.Fatalf("insert handle tmux: %v", err)
	}
	tmuxHandleID, _ := res.LastInsertId()

	handles, err := ListarRuntimeHandlesParaPresupuesto()
	if err != nil {
		t.Fatalf("listar handles para presupuesto: %v", err)
	}
	if len(handles) != 1 || handles[0].ID != tmuxHandleID {
		t.Fatalf("deberia devolver solo el handle tmux canonico: %+v", handles)
	}
}

func TestIngestarRuntimeTranscriptHandleNoClasificaRuidoOSCSpinner(t *testing.T) {
	abrirDBTemporalRuntimeObservabilidad(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(t.TempDir(), "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(t.TempDir(), "orquestador"),
		Herramienta: "codex-cli",
		Branch:      "main",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	runtime, _ := GetRuntimeBySesionID(sesion.ID)
	handle, _ := GetRuntimeHandleBySesionID(sesion.ID)

	logPath := filepath.Join(t.TempDir(), "spinner.log")
	logData := strings.Join([]string{
		"\x1b]0;⠋ orquesta-codex1\x1b\\\x1b]0;⠙ orquesta-codex1\x1b\\\x1b]0;⠹ orquesta-codex1\x1b\\",
		"",
	}, "\n")
	if err := os.WriteFile(logPath, []byte(logData), 0o600); err != nil {
		t.Fatalf("write log: %v", err)
	}
	metaJSON, _ := json.Marshal(map[string]any{"log_path": logPath})
	if _, err := DB.Exec(`UPDATE runtime_handles SET metadata_json=? WHERE id=?`, string(metaJSON), handle.ID); err != nil {
		t.Fatalf("update handle metadata: %v", err)
	}

	n, err := IngestarRuntimeTranscriptHandle(handle.ID)
	if err != nil {
		t.Fatalf("ingestar transcript: %v", err)
	}
	if n != 0 {
		t.Fatalf("el ruido OSC del spinner deberia descartarse por completo, got=%d", n)
	}

	agente := "Codex1"
	items, err := ListarRuntimeTranscript(FiltroRuntimeTranscript{Agente: &agente, Limit: 10})
	if err != nil {
		t.Fatalf("listar transcript: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("transcript inesperado: %+v", items)
	}
	var count int
	if err := DB.QueryRow(`SELECT COUNT(*) FROM runtime_events WHERE runtime_id=?`, runtime.ID).Scan(&count); err != nil {
		t.Fatalf("count runtime_events: %v", err)
	}
	if count != 0 {
		t.Fatalf("no deberia generar runtime_events para ruido OSC del spinner, got=%d", count)
	}
}

func TestIngestarRuntimeTranscriptHandleDescartaFragmentosPTYConControlChars(t *testing.T) {
	abrirDBTemporalRuntimeObservabilidad(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(t.TempDir(), "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(t.TempDir(), "orquestador"),
		Herramienta: "codex-cli",
		Branch:      "main",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	runtime, _ := GetRuntimeBySesionID(sesion.ID)
	handle, _ := GetRuntimeHandleBySesionID(sesion.ID)

	logPath := filepath.Join(t.TempDir(), "pty-fragments.log")
	logData := strings.Join([]string{
		"\x07\x08\t\tr",
		"\x07\x08\t\t\"",
		"",
	}, "\n")
	if err := os.WriteFile(logPath, []byte(logData), 0o600); err != nil {
		t.Fatalf("write log: %v", err)
	}
	metaJSON, _ := json.Marshal(map[string]any{"log_path": logPath})
	if _, err := DB.Exec(`UPDATE runtime_handles SET metadata_json=? WHERE id=?`, string(metaJSON), handle.ID); err != nil {
		t.Fatalf("update handle metadata: %v", err)
	}

	n, err := IngestarRuntimeTranscriptHandle(handle.ID)
	if err != nil {
		t.Fatalf("ingestar transcript: %v", err)
	}
	if n != 0 {
		t.Fatalf("los fragmentos PTY con control chars deberian descartarse, got=%d", n)
	}

	agente := "Codex1"
	items, err := ListarRuntimeTranscript(FiltroRuntimeTranscript{Agente: &agente, Limit: 10})
	if err != nil {
		t.Fatalf("listar transcript: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("transcript inesperado: %+v", items)
	}
	var count int
	if err := DB.QueryRow(`SELECT COUNT(*) FROM runtime_events WHERE runtime_id=?`, runtime.ID).Scan(&count); err != nil {
		t.Fatalf("count runtime_events: %v", err)
	}
	if count != 0 {
		t.Fatalf("no deberia generar runtime_events para fragmentos PTY, got=%d", count)
	}
}

func TestListarRuntimeTranscriptFiltraPorTextoLibre(t *testing.T) {
	abrirDBTemporalRuntimeObservabilidad(t)

	if err := RegistrarAgente("Codex2", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "demo",
		Nombre:  "Demo",
		RutaAbs: filepath.Join(t.TempDir(), "demo"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Codex2",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(t.TempDir(), "demo"),
		Herramienta: "codex-cli",
		Branch:      "main",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	runtime, _ := GetRuntimeBySesionID(sesion.ID)

	for _, item := range []*RuntimeTranscriptEntry{
		{
			RuntimeID:      runtime.ID,
			Agente:         "Codex2",
			ProyectoID:     &proyectoID,
			Stream:         "pty_out",
			Text:           "He preparado el refactor del router",
			NormalizedText: "he preparado el refactor del router",
			Classification: "ready_for_review",
		},
		{
			RuntimeID:      runtime.ID,
			Agente:         "Codex2",
			ProyectoID:     &proyectoID,
			Stream:         "pty_out",
			Text:           "Sigo con los tests de integración",
			NormalizedText: "sigo con los tests de integracion",
		},
	} {
		if _, err := RegistrarRuntimeTranscript(item); err != nil {
			t.Fatalf("registrar transcript: %v", err)
		}
	}

	agente := "Codex2"
	query := "refactor router"
	items, err := ListarRuntimeTranscript(FiltroRuntimeTranscript{
		Agente: &agente,
		Query:  &query,
		Limit:  10,
	})
	if err != nil {
		t.Fatalf("listar transcript filtrado: %v", err)
	}
	if len(items) != 1 || !strings.Contains(strings.ToLower(items[0].Text), "refactor") {
		t.Fatalf("resultado filtrado inesperado: %+v", items)
	}
}

func TestIngestarRuntimeTranscriptHandlePersisteByteOffset(t *testing.T) {
	abrirDBTemporalRuntimeObservabilidad(t)

	if err := RegistrarAgente("Codex3", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "trace-demo",
		Nombre:  "Trace Demo",
		RutaAbs: filepath.Join(t.TempDir(), "trace-demo"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Codex3",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(t.TempDir(), "trace-demo"),
		Herramienta: "codex-cli",
		Branch:      "main",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}

	logPath := filepath.Join(t.TempDir(), "offsets.log")
	if err := os.WriteFile(logPath, []byte("uno\ndos\n"), 0o600); err != nil {
		t.Fatalf("write log: %v", err)
	}
	metaJSON, _ := json.Marshal(map[string]any{"log_path": logPath})
	if _, err := DB.Exec(`UPDATE runtime_handles SET metadata_json=? WHERE id=?`, string(metaJSON), handle.ID); err != nil {
		t.Fatalf("update handle metadata: %v", err)
	}

	n, err := IngestarRuntimeTranscriptHandle(handle.ID)
	if err != nil {
		t.Fatalf("ingestar transcript: %v", err)
	}
	if n != 2 {
		t.Fatalf("esperaba 2 líneas ingeridas, got=%d", n)
	}

	agente := "Codex3"
	items, err := ListarRuntimeTranscript(FiltroRuntimeTranscript{Agente: &agente, Limit: 10})
	if err != nil {
		t.Fatalf("listar transcript: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("transcript inesperado: %+v", items)
	}
	offsets := map[string]int64{}
	for _, item := range items {
		if item == nil || item.ByteOffset == nil {
			t.Fatalf("byte_offset ausente: %+v", items)
		}
		offsets[item.Text] = *item.ByteOffset
	}
	if offsets["uno"] != 0 {
		t.Fatalf("offset de 'uno' inesperado: %d", offsets["uno"])
	}
	if offsets["dos"] != 4 {
		t.Fatalf("offset de 'dos' inesperado: %d", offsets["dos"])
	}
}

func TestClasificarTextoTranscriptReconoceReviewYReplan(t *testing.T) {
	cases := []struct {
		raw  string
		want string
	}{
		{raw: "Está listo para revisión final", want: "ready_for_review"},
		{raw: "ready for review after the last fix", want: "ready_for_review"},
		{raw: "Review aprobada, LGTM", want: "review_approved"},
		{raw: "Changes requested after review", want: "review_changes_requested"},
		{raw: "Review blocked pending credentials", want: "review_blocked"},
		{raw: "¿Qué hago ahora? no tengo claro el siguiente paso", want: "needs_replan"},
		{raw: "what should i do next after this task?", want: "needs_replan"},
		{raw: "thread 'main' panicked at src/ui.rs:1:1", want: "runtime_panic"},
		{raw: "connection reset by peer", want: "runtime_crash"},
		{raw: "can i continue refactoring without waiting?", want: "approval_request"},
		{raw: "can i continue refactoring without waiting", want: ""},
	}
	for _, tc := range cases {
		if got := clasificarTextoTranscript(normalizarTextoTranscript(tc.raw)); got != tc.want {
			t.Fatalf("clasificacion inesperada para %q: got=%q want=%q", tc.raw, got, tc.want)
		}
	}
}

func TestIngestarRuntimeTranscriptHandleFiltraRuidoYClasificaPanic(t *testing.T) {
	abrirDBTemporalRuntimeObservabilidad(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(t.TempDir(), "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(t.TempDir(), "orquestador"),
		Herramienta: "codex-cli",
		Branch:      "main",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	runtime, _ := GetRuntimeBySesionID(sesion.ID)
	handle, _ := GetRuntimeHandleBySesionID(sesion.ID)

	logPath := filepath.Join(t.TempDir(), "panic.log")
	logData := strings.Join([]string{
		"│",
		"⠋",
		"thread 'main' panicked at src/ui.rs:1:1",
		"",
	}, "\n")
	if err := os.WriteFile(logPath, []byte(logData), 0o600); err != nil {
		t.Fatalf("write log: %v", err)
	}
	metaJSON, _ := json.Marshal(map[string]any{"log_path": logPath})
	if _, err := DB.Exec(`UPDATE runtime_handles SET metadata_json=? WHERE id=?`, string(metaJSON), handle.ID); err != nil {
		t.Fatalf("update handle metadata: %v", err)
	}

	n, err := IngestarRuntimeTranscriptHandle(handle.ID)
	if err != nil {
		t.Fatalf("ingestar transcript: %v", err)
	}
	if n != 1 {
		t.Fatalf("debería ingerir solo la línea semántica, got=%d", n)
	}

	agente := "Codex1"
	items, err := ListarRuntimeTranscript(FiltroRuntimeTranscript{Agente: &agente, Limit: 10})
	if err != nil {
		t.Fatalf("listar transcript: %v", err)
	}
	if len(items) != 1 || items[0].Classification != "runtime_panic" {
		t.Fatalf("clasificación inesperada: %+v", items)
	}
	var level string
	if err := DB.QueryRow(`SELECT level FROM runtime_events WHERE runtime_id=? AND kind='runtime_panic' ORDER BY id DESC LIMIT 1`, runtime.ID).Scan(&level); err != nil {
		t.Fatalf("runtime_event panic: %v", err)
	}
	if level != "critical" {
		t.Fatalf("level inesperado para runtime_panic: %s", level)
	}
}

func TestIngestarRuntimeTranscriptHandleRecortaPreambuloScriptEnRuntimePanic(t *testing.T) {
	prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(t.TempDir(), "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(t.TempDir(), "orquestador"),
		Herramienta: "codex-cli",
		Branch:      "main",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	runtime, _ := GetRuntimeBySesionID(sesion.ID)
	handle, _ := GetRuntimeHandleBySesionID(sesion.ID)

	logPath := filepath.Join(t.TempDir(), "script-panic.log")
	logData := strings.Join([]string{
		`Script started on 2026-03-31 16:48:56+02:00 [COMMAND="'/tmp/codex-perfiles/bin/codex-perfil' 'Codex1' '-C' '/tmp/orquesta' 'Bootstrap enorme ...'"] The application panicked (crashed).`,
		"",
	}, "\n")
	if err := os.WriteFile(logPath, []byte(logData), 0o600); err != nil {
		t.Fatalf("write log: %v", err)
	}
	metaJSON, _ := json.Marshal(map[string]any{"log_path": logPath})
	if _, err := DB.Exec(`UPDATE runtime_handles SET metadata_json=? WHERE id=?`, string(metaJSON), handle.ID); err != nil {
		t.Fatalf("update handle metadata: %v", err)
	}

	n, err := IngestarRuntimeTranscriptHandle(handle.ID)
	if err != nil {
		t.Fatalf("ingestar transcript: %v", err)
	}
	if n != 1 {
		t.Fatalf("debería ingerir solo una línea útil, got=%d", n)
	}

	agente := "Codex1"
	items, err := ListarRuntimeTranscript(FiltroRuntimeTranscript{Agente: &agente, Limit: 10})
	if err != nil {
		t.Fatalf("listar transcript: %v", err)
	}
	if len(items) != 1 || items[0].Classification != "runtime_panic" {
		t.Fatalf("clasificación inesperada: %+v", items)
	}
	if strings.Contains(items[0].Text, "Script started on ") {
		t.Fatalf("el preámbulo de script no debería persistirse en el panic: %+v", items[0])
	}
	if !strings.Contains(items[0].Text, "The application panicked (crashed).") {
		t.Fatalf("debería conservar el texto útil del panic: %+v", items[0])
	}
	var message string
	if err := DB.QueryRow(`SELECT message FROM runtime_events WHERE runtime_id=? AND kind='runtime_panic' ORDER BY id DESC LIMIT 1`, runtime.ID).Scan(&message); err != nil {
		t.Fatalf("runtime_event panic: %v", err)
	}
	if strings.Contains(message, "Script started on ") {
		t.Fatalf("el evento runtime_panic no debería arrastrar el comando de script: %s", message)
	}
}

func TestIngestarRuntimeTranscriptHandleLimpiaCSIPrivadoEnRuntimePanic(t *testing.T) {
	prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(t.TempDir(), "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(t.TempDir(), "orquestador"),
		Herramienta: "codex-cli",
		Branch:      "main",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	runtime, _ := GetRuntimeBySesionID(sesion.ID)
	handle, _ := GetRuntimeHandleBySesionID(sesion.ID)

	logPath := filepath.Join(t.TempDir(), "csi-panic.log")
	logData := strings.Join([]string{
		".\x1b[<1uThe application panicked (crashed).",
		"",
	}, "\n")
	if err := os.WriteFile(logPath, []byte(logData), 0o600); err != nil {
		t.Fatalf("write log: %v", err)
	}
	metaJSON, _ := json.Marshal(map[string]any{"log_path": logPath})
	if _, err := DB.Exec(`UPDATE runtime_handles SET metadata_json=? WHERE id=?`, string(metaJSON), handle.ID); err != nil {
		t.Fatalf("update handle metadata: %v", err)
	}

	n, err := IngestarRuntimeTranscriptHandle(handle.ID)
	if err != nil {
		t.Fatalf("ingestar transcript: %v", err)
	}
	if n != 1 {
		t.Fatalf("debería ingerir una línea útil, got=%d", n)
	}

	agente := "Codex1"
	items, err := ListarRuntimeTranscript(FiltroRuntimeTranscript{Agente: &agente, Limit: 10})
	if err != nil {
		t.Fatalf("listar transcript: %v", err)
	}
	if len(items) != 1 || items[0].Classification != "runtime_panic" {
		t.Fatalf("clasificación inesperada: %+v", items)
	}
	if strings.Contains(items[0].Text, "\x1b[<1u") {
		t.Fatalf("la secuencia CSI privada no debería persistirse: %+v", items[0])
	}
	if strings.Contains(items[0].Text, "[<1u") {
		t.Fatalf("la secuencia CSI privada no debería quedar visible: %+v", items[0])
	}
	var message string
	if err := DB.QueryRow(`SELECT message FROM runtime_events WHERE runtime_id=? AND kind='runtime_panic' ORDER BY id DESC LIMIT 1`, runtime.ID).Scan(&message); err != nil {
		t.Fatalf("runtime_event panic: %v", err)
	}
	if strings.Contains(message, "[<1u") {
		t.Fatalf("el runtime_event no debería arrastrar el CSI privado: %s", message)
	}
}

func TestSanitizarTextoObservabilidadRuntimeLimpiaPrefijoPuntualAntesDelPanic(t *testing.T) {
	got := SanitizarTextoObservabilidadRuntime(".\x1b[<1uThe application panicked (crashed).")
	if got != "The application panicked (crashed)." {
		t.Fatalf("texto saneado inesperado: %q", got)
	}
}

func TestSanitizarTextoObservabilidadRuntimeLimpiaPrefijosHistoricosCortos(t *testing.T) {
	cases := []string{
		"sThe application panicked (crashed).",
		"[>7uThe application panicked (crashed).",
	}
	for _, raw := range cases {
		got := SanitizarTextoObservabilidadRuntime(raw)
		if got != "The application panicked (crashed)." {
			t.Fatalf("texto saneado inesperado para %q: %q", raw, got)
		}
	}
}

func TestIngestarRuntimeTranscriptHandleCompactaPendingTranscript(t *testing.T) {
	abrirDBTemporalRuntimeObservabilidad(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(t.TempDir(), "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(t.TempDir(), "orquestador"),
		Herramienta: "codex-cli",
		Branch:      "main",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, _ := GetRuntimeHandleBySesionID(sesion.ID)

	logPath := filepath.Join(t.TempDir(), "pending.log")
	logData := "\x1b]0;spinner\x1b\\\x07\x08\t\tpartial pending"
	if err := os.WriteFile(logPath, []byte(logData), 0o600); err != nil {
		t.Fatalf("write log: %v", err)
	}
	metaJSON, _ := json.Marshal(map[string]any{"log_path": logPath})
	if _, err := DB.Exec(`UPDATE runtime_handles SET metadata_json=? WHERE id=?`, string(metaJSON), handle.ID); err != nil {
		t.Fatalf("update handle metadata: %v", err)
	}

	n, err := IngestarRuntimeTranscriptHandle(handle.ID)
	if err != nil {
		t.Fatalf("ingestar transcript: %v", err)
	}
	if n != 0 {
		t.Fatalf("no deberia ingerir líneas completas, got=%d", n)
	}

	handle, err = GetRuntimeHandle(handle.ID)
	if err != nil || handle == nil {
		t.Fatalf("recargar handle: %+v err=%v", handle, err)
	}
	meta := map[string]any{}
	if err := json.Unmarshal([]byte(handle.MetadataJSON), &meta); err != nil {
		t.Fatalf("metadata invalida: %v", err)
	}
	got, _ := meta["transcript_log_pending"].(string)
	if got != "partial pending" {
		t.Fatalf("pending compactado inesperado: %q", got)
	}
}

func TestIngestarRuntimeTranscriptHandleDescartaBannerCodex(t *testing.T) {
	abrirDBTemporalRuntimeObservabilidad(t)

	if err := RegistrarAgente("Codex2", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(t.TempDir(), "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Codex2",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(t.TempDir(), "orquestador"),
		Herramienta: "codex-cli",
		Branch:      "main",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	runtime, _ := GetRuntimeBySesionID(sesion.ID)
	handle, _ := GetRuntimeHandleBySesionID(sesion.ID)

	logPath := filepath.Join(t.TempDir(), "codex-banner.log")
	logData := strings.Join([]string{
		"Perfil activo: Codex2",
		"CODEX_HOME: /tmp/codex",
		"Credenciales: /tmp/auth.json",
		"Consejo: si es el primer arranque, ejecuta codex-perfil Codex2 login",
		"https://github.com/openai/codex/releases/latest",
		"https://chatgpt.com/codex/settings/usage",
		"(https://chatgpt.com/explore/pro),",
		"  (http://localhost:8080).",
		"",
	}, "\n")
	if err := os.WriteFile(logPath, []byte(logData), 0o600); err != nil {
		t.Fatalf("write log: %v", err)
	}
	metaJSON, _ := json.Marshal(map[string]any{"log_path": logPath})
	if _, err := DB.Exec(`UPDATE runtime_handles SET metadata_json=? WHERE id=?`, string(metaJSON), handle.ID); err != nil {
		t.Fatalf("update handle metadata: %v", err)
	}

	n, err := IngestarRuntimeTranscriptHandle(handle.ID)
	if err != nil {
		t.Fatalf("ingestar transcript: %v", err)
	}
	if n != 0 {
		t.Fatalf("el banner de Codex deberia descartarse, got=%d", n)
	}

	agente := "Codex2"
	items, err := ListarRuntimeTranscript(FiltroRuntimeTranscript{Agente: &agente, Limit: 10})
	if err != nil {
		t.Fatalf("listar transcript: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("transcript inesperado: %+v", items)
	}
	var count int
	if err := DB.QueryRow(`SELECT COUNT(*) FROM runtime_events WHERE runtime_id=?`, runtime.ID).Scan(&count); err != nil {
		t.Fatalf("count runtime_events: %v", err)
	}
	if count != 0 {
		t.Fatalf("no deberia generar runtime_events para el banner de Codex, got=%d", count)
	}
}

func TestIngestarRuntimeTranscriptHandleDescartaRuidoUIYConservaSenal(t *testing.T) {
	abrirDBTemporalRuntimeObservabilidad(t)

	if err := RegistrarAgente("Codex3", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(t.TempDir(), "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Codex3",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(t.TempDir(), "orquestador"),
		Herramienta: "codex-cli",
		Branch:      "main",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, _ := GetRuntimeHandleBySesionID(sesion.ID)

	logPath := filepath.Join(t.TempDir(), "codex-ui-noise.log")
	logData := strings.Join([]string{
		"────────────────────────────────────────────────────────────────────────────────◦working(1m 00s • esc to interrupt)›use /skills to list available skills gpt-5.4 xhigh · /home/alberto/trabajo/orquesta",
		"• waited for background terminal",
		"• ran go test ./db -run 'Test(ListarWorktreesCoordCanonizaRelativasYOcultaActivasInexistentes)'",
		"",
	}, "\n")
	if err := os.WriteFile(logPath, []byte(logData), 0o600); err != nil {
		t.Fatalf("write log: %v", err)
	}
	metaJSON, _ := json.Marshal(map[string]any{"log_path": logPath})
	if _, err := DB.Exec(`UPDATE runtime_handles SET metadata_json=? WHERE id=?`, string(metaJSON), handle.ID); err != nil {
		t.Fatalf("update handle metadata: %v", err)
	}

	n, err := IngestarRuntimeTranscriptHandle(handle.ID)
	if err != nil {
		t.Fatalf("ingestar transcript: %v", err)
	}
	if n != 1 {
		t.Fatalf("deberia conservar solo la señal útil, got=%d", n)
	}

	agente := "Codex3"
	items, err := ListarRuntimeTranscript(FiltroRuntimeTranscript{Agente: &agente, Limit: 10})
	if err != nil {
		t.Fatalf("listar transcript: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("transcript inesperado: %+v", items)
	}
	if got := items[0].NormalizedText; !strings.Contains(got, "ran go test ./db") {
		t.Fatalf("deberia conservar la linea útil de ejecución, got=%q", got)
	}
}

func TestPurgarRuntimeTranscriptRuidoHistoricoBorraSoloChromeUI(t *testing.T) {
	abrirDBTemporalRuntimeObservabilidad(t)

	if err := RegistrarAgente("Codex4", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(t.TempDir(), "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Codex4",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(t.TempDir(), "orquestador"),
		Herramienta: "codex-cli",
		Branch:      "main",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	runtime, err := GetRuntimeBySesionID(sesion.ID)
	if err != nil || runtime == nil {
		t.Fatalf("runtime: %+v err=%v", runtime, err)
	}
	handle, err := GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	handleID := handle.ID
	old := time.Now().UTC().Add(-20 * time.Minute)

	noiseID, err := RegistrarRuntimeTranscript(&RuntimeTranscriptEntry{
		RuntimeID:      runtime.ID,
		HandleID:       &handleID,
		Agente:         "Codex4",
		ProyectoID:     &proyectoID,
		Stream:         "pty_out",
		Text:           "• waited for background terminal",
		NormalizedText: normalizarTextoTranscript("• waited for background terminal"),
	})
	if err != nil {
		t.Fatalf("registrar ruido: %v", err)
	}
	keepID, err := RegistrarRuntimeTranscript(&RuntimeTranscriptEntry{
		RuntimeID:      runtime.ID,
		HandleID:       &handleID,
		Agente:         "Codex4",
		ProyectoID:     &proyectoID,
		Stream:         "pty_out",
		Text:           "• ran go test ./db -run TestFoo",
		NormalizedText: normalizarTextoTranscript("• ran go test ./db -run TestFoo"),
	})
	if err != nil {
		t.Fatalf("registrar señal util: %v", err)
	}
	signalID, err := RegistrarRuntimeTranscript(&RuntimeTranscriptEntry{
		RuntimeID:      runtime.ID,
		HandleID:       &handleID,
		Agente:         "Codex4",
		ProyectoID:     &proyectoID,
		Stream:         "pty_out",
		Text:           "¿me dejas seguir con el cambio?",
		NormalizedText: normalizarTextoTranscript("¿me dejas seguir con el cambio?"),
		Classification: "approval_request",
	})
	if err != nil {
		t.Fatalf("registrar signal: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_transcript SET created_at=? WHERE id IN (?,?,?)`, old, noiseID, keepID, signalID); err != nil {
		t.Fatalf("envejecer transcript: %v", err)
	}

	n, err := PurgarRuntimeTranscriptRuidoHistorico()
	if err != nil {
		t.Fatalf("purgar ruido historico: %v", err)
	}
	if n != 1 {
		t.Fatalf("deberia borrar solo una entrada de ruido, got=%d", n)
	}

	var count int
	if err := DB.QueryRow(`SELECT COUNT(*) FROM runtime_transcript WHERE id=?`, noiseID).Scan(&count); err != nil {
		t.Fatalf("count ruido: %v", err)
	}
	if count != 0 {
		t.Fatalf("la entrada de ruido deberia desaparecer")
	}
	if err := DB.QueryRow(`SELECT COUNT(*) FROM runtime_transcript WHERE id IN (?,?)`, keepID, signalID).Scan(&count); err != nil {
		t.Fatalf("count restantes: %v", err)
	}
	if count != 2 {
		t.Fatalf("deberia conservar señal útil y clasificación, got=%d", count)
	}
}
