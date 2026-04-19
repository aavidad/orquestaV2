package db

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"orquesta/coordinacion"
	"orquesta/internal/controlruntime"
	"orquesta/runtimeagente"
)

func mustGetwdRuntimeHandleTest(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	return wd
}

func mustExecutableRuntimeHandleTest(t *testing.T) string {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("executable: %v", err)
	}
	return exe
}

func TestGetRuntimeHandleActivoAgenteProyectoIgnoraFantasmaMasReciente(t *testing.T) {
	tmp := prepararDBTemporal(t)

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

	runtimeVivoID, err := RegistrarRuntimeInstance(&RuntimeInstance{
		Agente:       "Codex1",
		ProyectoID:   &proyectoID,
		LogicalState: "esperando_io",
		ProcessState: "running",
		PID:          int64PtrTest(int64(os.Getpid())),
	})
	if err != nil {
		t.Fatalf("crear runtime vivo: %v", err)
	}
	handleVivoID := mustInsertID(t, `INSERT INTO runtime_handles (
		agente, proyecto_id, runtime_id, transporte, handle_kind, handle_ref, estado, last_seen_at
	) VALUES (?,?,?,?,?,?, 'activo', ?)`,
		"Codex1", proyectoID, runtimeVivoID, "cli", "process", strconv.Itoa(os.Getpid()), time.Now().UTC().Add(-5*time.Minute))

	pidFantasma := int64(99999991)
	runtimeFantasmaID, err := RegistrarRuntimeInstance(&RuntimeInstance{
		Agente:       "Codex1",
		ProyectoID:   &proyectoID,
		LogicalState: "esperando_io",
		ProcessState: "running",
		PID:          &pidFantasma,
	})
	if err != nil {
		t.Fatalf("crear runtime fantasma: %v", err)
	}
	handleFantasmaID := mustInsertID(t, `INSERT INTO runtime_handles (
		agente, proyecto_id, runtime_id, transporte, handle_kind, handle_ref, estado, last_seen_at
	) VALUES (?,?,?,?,?,?, 'activo', ?)`,
		"Codex1", proyectoID, runtimeFantasmaID, "cli", "process", strconv.FormatInt(pidFantasma, 10), time.Now().UTC())

	handle, err := GetRuntimeHandleActivoAgenteProyecto("Codex1", &proyectoID)
	if err != nil {
		t.Fatalf("get handle activo: %v", err)
	}
	if handle == nil || handle.ID != handleVivoID || handle.Estado != "activo" {
		t.Fatalf("deberia recuperar el handle vivo y no el fantasma reciente: %+v", handle)
	}

	fantasma, err := GetRuntimeHandle(handleFantasmaID)
	if err != nil {
		t.Fatalf("get handle fantasma: %v", err)
	}
	if fantasma == nil || fantasma.Estado != "fallido" {
		t.Fatalf("el handle fantasma reciente deberia quedar fallido: %+v", fantasma)
	}
}

func TestRuntimeHandlePuedeRepresentarActivoCanonicoOmiteTMUXFueraDeWorktreeCanonica(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex9", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	rutaBase := filepath.Join(tmp, "orquestador")
	rutaWorktree := filepath.Join(rutaBase, ".orquesta-worktrees", "orquestador-codex9")
	if err := os.MkdirAll(rutaWorktree, 0o755); err != nil {
		t.Fatalf("mkdir worktree: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: rutaBase,
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := (CoordinationWorktreeSQLRepository{}).Create(&coordinacion.Worktree{
		ProjectID: proyectoID,
		Agent:     "Codex9",
		Name:      "orquestador-codex9",
		Path:      rutaWorktree,
		Branch:    "orq-orquestador-codex9",
		BaseRef:   "HEAD",
		State:     coordinacion.WorktreeActive,
		Reason:    "test",
	}); err != nil {
		t.Fatalf("crear worktree activa: %v", err)
	}

	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:            "Codex9",
		ProyectoID:        &proyectoID,
		CWD:               rutaWorktree,
		Herramienta:       "codex-cli",
		ExternalSessionID: "sess-codex9",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	runtime, err := GetRuntimeBySesionID(sesion.ID)
	if err != nil || runtime == nil {
		t.Fatalf("runtime: %+v err=%v", runtime, err)
	}

	runDir := filepath.Join(tmp, "worker-codex9")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir worker: %v", err)
	}
	manifestPath := filepath.Join(runDir, "manifest.json")
	statusPath := filepath.Join(runDir, "status.json")
	heartbeatPath := filepath.Join(runDir, "heartbeat.json")
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if err := os.WriteFile(manifestPath, []byte(`{"version":1,"agent":"Codex9","driver":"tmux_cli_session","transport":"tmux","tmux_session":"orq-codex9-root","tmux_pane_id":"%12","status_path":"`+statusPath+`","heartbeat_path":"`+heartbeatPath+`","working_dir":"`+rutaWorktree+`"}`+"\n"), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(statusPath, []byte(`{"state":"running","updated_at":"`+now+`","alive":true,"child_pid":`+strconv.Itoa(os.Getpid())+`,"working_dir":"`+rutaWorktree+`","current_path":"`+rutaBase+`"}`+"\n"), 0o600); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, []byte(`{"alive":true,"heartbeat_at":"`+now+`","started_at":"`+now+`","child_pid":`+strconv.Itoa(os.Getpid())+`}`+"\n"), 0o600); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}

	metaJSON, _ := json.Marshal(map[string]any{
		"driver":                "tmux_cli_session",
		"tmux_session":          "orq-codex9-root",
		"tmux_pane_id":          "%12",
		"working_dir":           rutaWorktree,
		"cwd":                   rutaWorktree,
		"external_session_id":   "sess-codex9",
		"worker_manifest_path":  manifestPath,
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
	})
	capsJSON, _ := json.Marshal(map[string]any{
		"mailbox_delivery_mode": runtimeagente.MailboxDeliverySessionResume,
		"can_send_input":        false,
	})
	if _, err := DB.Exec(`UPDATE runtime_handles SET transporte='tmux', handle_kind='session', handle_ref='orq-codex9-root/%12', estado='activo', metadata_json=?, capabilities_json=?, last_seen_at=CURRENT_TIMESTAMP WHERE id=?`, string(metaJSON), string(capsJSON), handle.ID); err != nil {
		t.Fatalf("update handle: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_instances SET cwd=?, logical_state='esperando_io', process_state='running', last_heartbeat_at=CURRENT_TIMESTAMP, last_event_at=CURRENT_TIMESTAMP WHERE id=?`, rutaWorktree, runtime.ID); err != nil {
		t.Fatalf("update runtime: %v", err)
	}
	handle, err = GetRuntimeHandle(handle.ID)
	if err != nil || handle == nil {
		t.Fatalf("refresh handle: %+v err=%v", handle, err)
	}

	if runtimeHandlePuedeRepresentarActivoCanonico(handle, time.Now().UTC()) {
		t.Fatal("tmux fuera de la worktree canonica no deberia representar activo canonico")
	}
}

func TestRuntimeHandleTMUXCurrentPathMismatchDetectaCurrentPathBorrado(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("CodexDeletedPath", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	rutaBase := filepath.Join(tmp, "orquestador")
	rutaWorktree := filepath.Join(rutaBase, ".orquesta-worktrees", "orquestador-codexdeleted")
	if err := os.MkdirAll(rutaWorktree, 0o755); err != nil {
		t.Fatalf("mkdir worktree: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: rutaBase,
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := (CoordinationWorktreeSQLRepository{}).Create(&coordinacion.Worktree{
		ProjectID: proyectoID,
		Agent:     "CodexDeletedPath",
		Name:      "orquestador-codexdeleted",
		Path:      rutaWorktree,
		Branch:    "orq-orquestador-codexdeleted",
		BaseRef:   "HEAD",
		State:     coordinacion.WorktreeActive,
		Reason:    "test",
	}); err != nil {
		t.Fatalf("crear worktree activa: %v", err)
	}

	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:            "CodexDeletedPath",
		ProyectoID:        &proyectoID,
		CWD:               rutaWorktree,
		Herramienta:       "codex-cli",
		ExternalSessionID: "sess-codexdeleted",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}

	runDir := filepath.Join(tmp, "worker-codexdeleted")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir worker: %v", err)
	}
	manifestPath := filepath.Join(runDir, "manifest.json")
	statusPath := filepath.Join(runDir, "status.json")
	heartbeatPath := filepath.Join(runDir, "heartbeat.json")
	deletedCurrent := filepath.Join(tmp, "deleted-current-path")
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if err := os.WriteFile(manifestPath, []byte(`{"version":1,"agent":"CodexDeletedPath","driver":"tmux_cli_session","transport":"tmux","tmux_session":"orq-codexdeleted","tmux_pane_id":"%44","status_path":"`+statusPath+`","heartbeat_path":"`+heartbeatPath+`","working_dir":"`+rutaWorktree+`"}`+"\n"), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(statusPath, []byte(`{"state":"running","updated_at":"`+now+`","alive":true,"child_pid":`+strconv.Itoa(os.Getpid())+`,"working_dir":"`+rutaWorktree+`","current_path":"`+deletedCurrent+`"}`+"\n"), 0o600); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, []byte(`{"alive":true,"heartbeat_at":"`+now+`","started_at":"`+now+`","child_pid":`+strconv.Itoa(os.Getpid())+`}`+"\n"), 0o600); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}

	metaJSON, _ := json.Marshal(map[string]any{
		"driver":                "tmux_cli_session",
		"tmux_session":          "orq-codexdeleted",
		"tmux_pane_id":          "%44",
		"working_dir":           rutaWorktree,
		"cwd":                   rutaWorktree,
		"external_session_id":   "sess-codexdeleted",
		"worker_manifest_path":  manifestPath,
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
	})
	capsJSON, _ := json.Marshal(map[string]any{
		"mailbox_delivery_mode": runtimeagente.MailboxDeliverySessionResume,
		"can_send_input":        false,
	})
	if _, err := DB.Exec(`UPDATE runtime_handles SET transporte='tmux', handle_kind='session', handle_ref='orq-codexdeleted/%44', estado='activo', metadata_json=?, capabilities_json=?, last_seen_at=CURRENT_TIMESTAMP WHERE id=?`, string(metaJSON), string(capsJSON), handle.ID); err != nil {
		t.Fatalf("update handle: %v", err)
	}
	handle, err = GetRuntimeHandle(handle.ID)
	if err != nil || handle == nil {
		t.Fatalf("refresh handle: %+v err=%v", handle, err)
	}

	if !runtimeHandleTMUXCurrentPathMismatch(handle) {
		t.Fatal("current_path borrado deberia marcar mismatch tmux")
	}
}

func TestRuntimeHandleCanonicoRecienteConFallbackOmiteCLISessionSinContinuidad(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("CodexCLIStale", "programador"); err != nil {
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

	sesionID := mustInsertID(t, `INSERT INTO sesiones (
		agente, proyecto_id, activa, estado, herramienta, host, heartbeat_at
	) VALUES (?,?,?,?,?,?,?)`,
		"CodexCLIStale", proyectoID, 0, "cerrada", "codex-cli", "test-host", time.Now().UTC().Add(-2*time.Hour))

	runtimeID, err := RegistrarRuntimeInstance(&RuntimeInstance{
		Agente:       "CodexCLIStale",
		ProyectoID:   &proyectoID,
		SesionID:     &sesionID,
		LogicalState: "disponible",
		ProcessState: "desconocido",
	})
	if err != nil {
		t.Fatalf("crear runtime stale: %v", err)
	}
	metaJSON, _ := json.Marshal(map[string]any{
		"herramienta": "codex-cli",
		"cwd":         filepath.Join(tmp, "stale"),
		"working_dir": filepath.Join(tmp, "stale"),
	})
	mustInsertID(t, `INSERT INTO runtime_handles (
		agente, proyecto_id, runtime_id, sesion_id, transporte, handle_kind, handle_ref, estado, metadata_json, last_seen_at
	) VALUES (?,?,?,?,?,?,?,?,?,?)`,
		"CodexCLIStale", proyectoID, runtimeID, sesionID, "cli", "session", jsonNumber(sesionID), "activo", string(metaJSON), time.Now().UTC())

	runtimeHandleHotReset()
	handle, err := runtimeHandleCanonicoRecienteConFallback("CodexCLIStale", &proyectoID)
	if err != nil {
		t.Fatalf("runtimeHandleCanonicoRecienteConFallback: %v", err)
	}
	if handle != nil {
		t.Fatalf("el cli/session sin continuidad no deberia contar como canónico: %+v", handle)
	}
}

func TestListarRuntimeHandlesActivosOperativosRecientesFiltraStaleYDeduplica(t *testing.T) {
	tmp := prepararDBTemporal(t)

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

	oldSeen := time.Now().UTC().Add(-15 * time.Minute)
	newSeen := time.Now().UTC().Add(-30 * time.Second)
	if _, err := DB.Exec(`INSERT INTO runtime_handles (
		agente, proyecto_id, transporte, handle_kind, handle_ref, estado, last_seen_at
	) VALUES (?,?,?,?,?,?,?)`, "Codex1", proyectoID, "cli", "process", "old", "activo", oldSeen); err != nil {
		t.Fatalf("insert handle stale: %v", err)
	}
	if _, err := DB.Exec(`INSERT INTO runtime_handles (
		agente, proyecto_id, transporte, handle_kind, handle_ref, estado, last_seen_at
	) VALUES (?,?,?,?,?,?,?)`, "Codex1", proyectoID, "cli", "process", "new", "activo", newSeen); err != nil {
		t.Fatalf("insert handle fresh: %v", err)
	}

	handles, err := ListarRuntimeHandlesActivosOperativosRecientes()
	if err != nil {
		t.Fatalf("listar handles operativos recientes: %v", err)
	}
	key := runtimeHandleCacheKey("Codex1", &proyectoID)
	handle := handles[key]
	if handle == nil {
		t.Fatalf("deberia devolver un handle operativo para %s", key)
	}
	if handle.HandleRef != "new" {
		t.Fatalf("deberia priorizar el handle reciente, got=%q", handle.HandleRef)
	}
}

func TestGetRuntimeHandleActivoAgenteProyectoPrefiereWorkerEstructuradoFrescoYCierraLegacy(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex8", "programador"); err != nil {
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

	pid := int64(os.Getpid())
	runtimeLegacyID, err := RegistrarRuntimeInstance(&RuntimeInstance{
		Agente:       "Codex8",
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
		"working_dir":      filepath.Join(tmp, "legacy"),
		"rendered_command": "codex-perfil Codex8",
	})
	legacyHandleID := mustInsertID(t, `INSERT INTO runtime_handles (
		agente, proyecto_id, runtime_id, transporte, handle_kind, handle_ref, estado, metadata_json, last_seen_at
	) VALUES (?,?,?,?,?,?, 'activo', ?, ?)`,
		"Codex8", proyectoID, runtimeLegacyID, "cli", "process", strconv.Itoa(os.Getpid()), string(legacyMetaJSON), time.Now().UTC())

	runDir := filepath.Join(tmp, "worker")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir worker: %v", err)
	}
	manifestPath := filepath.Join(runDir, "manifest.json")
	statusPath := filepath.Join(runDir, "status.json")
	heartbeatPath := filepath.Join(runDir, "heartbeat.json")
	if err := os.WriteFile(manifestPath, []byte(`{"version":1,"agent":"Codex8","driver":"tmux_cli_session","transport":"tmux","tmux_session":"orq-codex8-001353","tmux_pane_id":"%7","status_path":"`+statusPath+`","heartbeat_path":"`+heartbeatPath+`"}`+"\n"), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if err := os.WriteFile(statusPath, []byte(`{"state":"running","updated_at":"`+now+`","alive":true,"child_pid":`+strconv.Itoa(os.Getpid())+`}`+"\n"), 0o600); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, []byte(`{"alive":true,"heartbeat_at":"`+now+`","started_at":"`+now+`","child_pid":`+strconv.Itoa(os.Getpid())+`}`+"\n"), 0o600); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}

	runtimeWorkerID, err := RegistrarRuntimeInstance(&RuntimeInstance{
		Agente:       "Codex8",
		ProyectoID:   &proyectoID,
		LogicalState: "esperando_io",
		ProcessState: "running",
		PID:          &pid,
	})
	if err != nil {
		t.Fatalf("crear runtime worker: %v", err)
	}
	workerMetaJSON, _ := json.Marshal(map[string]any{
		"driver":                "tmux_cli_session",
		"tmux_session":          "orq-codex8-001353",
		"tmux_pane_id":          "%7",
		"worker_manifest_path":  manifestPath,
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
		"working_dir":           filepath.Join(tmp, "worker"),
		"rendered_command":      "codex-perfil Codex8",
	})
	workerHandleID := mustInsertID(t, `INSERT INTO runtime_handles (
		agente, proyecto_id, runtime_id, transporte, handle_kind, handle_ref, estado, metadata_json, last_seen_at
	) VALUES (?,?,?,?,?,?, 'activo', ?, ?)`,
		"Codex8", proyectoID, runtimeWorkerID, "cli", "process", strconv.Itoa(os.Getpid()), string(workerMetaJSON), time.Now().UTC())

	handle, err := GetRuntimeHandleActivoAgenteProyecto("Codex8", &proyectoID)
	if err != nil {
		t.Fatalf("get handle activo preferente: %v", err)
	}
	if handle == nil || handle.ID != workerHandleID {
		t.Fatalf("deberia devolver el worker estructurado fresco: %+v", handle)
	}

	legacyHandle, err := GetRuntimeHandle(legacyHandleID)
	if err != nil {
		t.Fatalf("get handle legacy: %v", err)
	}
	if legacyHandle == nil || (legacyHandle.Estado != "cerrado" && legacyHandle.Estado != "fallido") {
		t.Fatalf("el handle legacy no deberia seguir operativo: %+v", legacyHandle)
	}
}

func TestGetRuntimeHandleActivoAgenteProyectoOmiteLegacyTMUXPreferredSinOptIn(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("CodexLegacy", "programador"); err != nil {
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

	pid := int64(os.Getpid())
	runtimeID, err := RegistrarRuntimeInstance(&RuntimeInstance{
		Agente:       "CodexLegacy",
		ProyectoID:   &proyectoID,
		LogicalState: "esperando_io",
		ProcessState: "running",
		PID:          &pid,
	})
	if err != nil {
		t.Fatalf("crear runtime legacy: %v", err)
	}
	metaJSON, _ := json.Marshal(map[string]any{
		"driver":           "process_pty_cli",
		"working_dir":      filepath.Join(tmp, "legacy"),
		"rendered_command": "codex-perfil CodexLegacy --model gpt-5.4",
	})
	if _, err := DB.Exec(`INSERT INTO runtime_handles (
		agente, proyecto_id, runtime_id, transporte, handle_kind, handle_ref, estado, metadata_json, last_seen_at
	) VALUES (?,?,?,?,?,?, 'activo', ?, ?)`,
		"CodexLegacy", proyectoID, runtimeID, "cli", "process", strconv.Itoa(os.Getpid()), string(metaJSON), time.Now().UTC()); err != nil {
		t.Fatalf("insert handle legacy: %v", err)
	}

	handle, err := GetRuntimeHandleActivoAgenteProyecto("CodexLegacy", &proyectoID)
	if err != nil {
		t.Fatalf("get handle activo: %v", err)
	}
	if handle != nil {
		t.Fatalf("el handle legacy tmux-preferred no deberia contar como activo canónico: %+v", handle)
	}
}

func TestGetRuntimeHandleActivoAgenteProyectoOmiteLegacyProcessPTYGenerico(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("ProcessLegacy", "programador"); err != nil {
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

	pid := int64(os.Getpid())
	runtimeID, err := RegistrarRuntimeInstance(&RuntimeInstance{
		Agente:       "ProcessLegacy",
		ProyectoID:   &proyectoID,
		LogicalState: "esperando_io",
		ProcessState: "running",
		PID:          &pid,
	})
	if err != nil {
		t.Fatalf("crear runtime legacy: %v", err)
	}
	metaJSON, _ := json.Marshal(map[string]any{
		"driver":           "process_pty_cli",
		"working_dir":      filepath.Join(tmp, "legacy"),
		"rendered_command": "cat-cli Worker1",
	})
	if _, err := DB.Exec(`INSERT INTO runtime_handles (
		agente, proyecto_id, runtime_id, transporte, handle_kind, handle_ref, estado, metadata_json, last_seen_at
	) VALUES (?,?,?,?,?,?, 'activo', ?, ?)`,
		"ProcessLegacy", proyectoID, runtimeID, "cli", "process", strconv.Itoa(os.Getpid()), string(metaJSON), time.Now().UTC()); err != nil {
		t.Fatalf("insert handle legacy generico: %v", err)
	}

	handle, err := GetRuntimeHandleActivoAgenteProyecto("ProcessLegacy", &proyectoID)
	if err != nil {
		t.Fatalf("get handle activo: %v", err)
	}
	if handle != nil {
		t.Fatalf("ningun process_pty_cli deberia contar como activo canónico: %+v", handle)
	}
}

func TestRuntimeHandleWorkerInfoNoMarcaLegacyProcessPTYComoEstructurado(t *testing.T) {
	now := time.Now().UTC()
	tmp := t.TempDir()
	statusPath := filepath.Join(tmp, "status.json")
	heartbeatPath := filepath.Join(tmp, "heartbeat.json")
	statusRaw, _ := json.Marshal(map[string]any{
		"state":      "running",
		"updated_at": now.Format(time.RFC3339Nano),
		"alive":      true,
	})
	heartbeatRaw, _ := json.Marshal(map[string]any{
		"alive":        true,
		"heartbeat_at": now.Format(time.RFC3339Nano),
	})
	if err := os.WriteFile(statusPath, append(statusRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, append(heartbeatRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}
	metaJSON, _ := json.Marshal(map[string]any{
		"driver":                "process_pty_cli",
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
		"rendered_command":      "codex-perfil Codex7 --model gpt-5.4",
	})
	handle := &RuntimeHandle{
		Agente:       "Codex7",
		Estado:       "activo",
		Transporte:   "cli",
		HandleKind:   "process",
		MetadataJSON: string(metaJSON),
	}

	info := runtimeHandleWorkerInfoFor(handle, now)
	if info.structured {
		t.Fatalf("process_pty_cli legacy no deberia contarse como worker estructurado: %+v", info)
	}
	if !info.fresh {
		t.Fatalf("el snapshot sigue siendo fresco aunque no sea estructurado: %+v", info)
	}
}

func TestRefrescarRuntimeHandleSiSigueVivoNoReviveLegacyTMUXPreferred(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("CodexLegacyRevive", "programador"); err != nil {
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

	pid := int64(os.Getpid())
	runtimeID, err := RegistrarRuntimeInstance(&RuntimeInstance{
		Agente:       "CodexLegacyRevive",
		ProyectoID:   &proyectoID,
		LogicalState: "bloqueado",
		ProcessState: "running",
		PID:          &pid,
	})
	if err != nil {
		t.Fatalf("crear runtime legacy: %v", err)
	}
	metaJSON, _ := json.Marshal(map[string]any{
		"driver":           "process_pty_cli",
		"working_dir":      filepath.Join(tmp, "legacy"),
		"rendered_command": "codex-perfil CodexLegacyRevive --model gpt-5.4",
	})
	handleID := mustInsertID(t, `INSERT INTO runtime_handles (
		agente, proyecto_id, runtime_id, transporte, handle_kind, handle_ref, estado, metadata_json, last_seen_at
	) VALUES (?,?,?,?,?,?, 'fallido', ?, ?)`,
		"CodexLegacyRevive", proyectoID, runtimeID, "cli", "process", strconv.Itoa(os.Getpid()), string(metaJSON), time.Now().UTC())
	handle, err := GetRuntimeHandle(handleID)
	if err != nil || handle == nil {
		t.Fatalf("get handle legacy: %+v err=%v", handle, err)
	}

	revived, err := refrescarRuntimeHandleSiSigueVivo(handle)
	if err != nil {
		t.Fatalf("refrescarRuntimeHandleSiSigueVivo: %v", err)
	}
	if revived != nil {
		t.Fatalf("un handle legacy tmux-preferred no deberia revivirse por PID: %+v", revived)
	}
}

func TestRuntimeHandleSePuedeValidarLocalmenteOmiteLegacyTMUXPreferred(t *testing.T) {
	handle := &RuntimeHandle{
		Transporte: "cli",
		HandleKind: "process",
		Estado:     "activo",
		MetadataJSON: `{
			"driver":"process_pty_cli",
			"rendered_command":"codex-perfil CodexLegacy --model gpt-5.4"
		}`,
	}
	if runtimeHandleSePuedeValidarLocalmente(handle) {
		t.Fatalf("un handle legacy tmux-preferred no deberia validarse localmente: %+v", handle)
	}
}

func TestRuntimeHandleSePuedeValidarLocalmenteOmiteLegacyProcessPTYGenerico(t *testing.T) {
	handle := &RuntimeHandle{
		Transporte: "cli",
		HandleKind: "process",
		Estado:     "activo",
		MetadataJSON: `{
			"driver":"process_pty_cli",
			"rendered_command":"cat-cli Worker1"
		}`,
	}
	if runtimeHandleSePuedeValidarLocalmente(handle) {
		t.Fatalf("ningun process_pty_cli deberia validarse localmente: %+v", handle)
	}
}

func TestValidarRuntimeHandleActivoSinFallbackOmiteLegacyTMUXPreferred(t *testing.T) {
	handle := &RuntimeHandle{
		Agente:     "CodexLegacy",
		Transporte: "cli",
		HandleKind: "process",
		Estado:     "activo",
		MetadataJSON: `{
			"driver":"process_pty_cli",
			"rendered_command":"codex-perfil CodexLegacy --model gpt-5.4"
		}`,
	}
	validado, err := validarRuntimeHandleActivoSinFallback(handle)
	if err != nil {
		t.Fatalf("validarRuntimeHandleActivoSinFallback: %v", err)
	}
	if validado != nil {
		t.Fatalf("un handle legacy tmux-preferred no deberia validarse como activo: %+v", validado)
	}
}

func TestObjetivoProcesoDesdeHandleRuntimeOmitePIDParaTMUXYLegacy(t *testing.T) {
	pid := int64(4242)
	runtime := &RuntimeInstance{PID: &pid}

	generico := objetivoProcesoDesdeHandleRuntime(&RuntimeHandle{
		Transporte: "cli",
		HandleKind: "process",
		MetadataJSON: `{
			"driver":"process_exec",
			"rendered_command":"bash worker.sh"
		}`,
	}, runtime)
	if generico.PID == nil || *generico.PID != pid {
		t.Fatalf("un proceso genérico debería conservar PID: %+v", generico)
	}

	tmux := objetivoProcesoDesdeHandleRuntime(&RuntimeHandle{
		Transporte: "tmux",
		HandleKind: "session",
		MetadataJSON: `{
			"driver":"tmux_cli_session",
			"tmux_session":"orq-codex/%1"
		}`,
	}, runtime)
	if tmux.PID != nil {
		t.Fatalf("un handle tmux no debería heredar PID: %+v", tmux)
	}

	legacy := objetivoProcesoDesdeHandleRuntime(&RuntimeHandle{
		Transporte: "cli",
		HandleKind: "process",
		MetadataJSON: `{
			"driver":"process_pty_cli",
			"rendered_command":"codex-perfil CodexLegacy --model gpt-5.4"
		}`,
	}, runtime)
	if legacy.PID != nil {
		t.Fatalf("un legacy tmux-preferred no debería heredar PID: %+v", legacy)
	}

	legacyGeneric := objetivoProcesoDesdeHandleRuntime(&RuntimeHandle{
		Transporte: "cli",
		HandleKind: "process",
		MetadataJSON: `{
			"driver":"process_pty_cli",
			"rendered_command":"cat-cli Worker1"
		}`,
	}, runtime)
	if legacyGeneric.PID != nil {
		t.Fatalf("ningun process_pty_cli debería heredar PID: %+v", legacyGeneric)
	}

	tmp := t.TempDir()
	manifestPath := filepath.Join(tmp, "manifest.json")
	statusPath := filepath.Join(tmp, "status.json")
	heartbeatPath := filepath.Join(tmp, "heartbeat.json")
	if err := os.WriteFile(manifestPath, []byte(`{"version":1,"agent":"CodexObserved","driver":"tmux_cli_session","transport":"tmux","tmux_session":"orq-codexobserved-1","tmux_pane_id":"%17","status_path":"`+statusPath+`","heartbeat_path":"`+heartbeatPath+`"}`+"\n"), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(statusPath, []byte(`{"state":"running","alive":true}`+"\n"), 0o600); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, []byte(`{"alive":true,"timestamp":"`+time.Now().UTC().Format(time.RFC3339Nano)+`"}`+"\n"), 0o600); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}
	observedTMUX := objetivoProcesoDesdeHandleRuntime(&RuntimeHandle{
		Transporte: "cli",
		HandleKind: "process",
		HandleRef:  "7777",
		MetadataJSON: `{
			"driver":"process_pty_cli",
			"worker_manifest_path":"` + manifestPath + `",
			"worker_status_path":"` + statusPath + `",
			"worker_heartbeat_path":"` + heartbeatPath + `"
		}`,
	}, runtime)
	if observedTMUX.PID != nil {
		t.Fatalf("un worker tmux observado por snapshot no debería heredar PID: %+v", observedTMUX)
	}
	if observedTMUX.HandleKind != "session" {
		t.Fatalf("un worker tmux observado debería normalizar handle_kind=session: %+v", observedTMUX)
	}
	if observedTMUX.HandleRef != "orq-codexobserved-1/%17" {
		t.Fatalf("handle_ref tmux observado inesperado: %+v", observedTMUX)
	}
}

func TestGetRuntimeHandleActivoAgenteProyectoAceptaTMUXFreshSinProbeExterno(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("CodexTmuxFresh", "programador"); err != nil {
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

	runtimeID, err := RegistrarRuntimeInstance(&RuntimeInstance{
		Agente:       "CodexTmuxFresh",
		ProyectoID:   &proyectoID,
		LogicalState: "esperando_io",
		ProcessState: "running",
	})
	if err != nil {
		t.Fatalf("crear runtime worker: %v", err)
	}

	runDir := filepath.Join(tmp, "worker-fresh")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir worker: %v", err)
	}
	manifestPath := filepath.Join(runDir, "manifest.json")
	statusPath := filepath.Join(runDir, "status.json")
	heartbeatPath := filepath.Join(runDir, "heartbeat.json")
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if err := os.WriteFile(manifestPath, []byte(`{"version":1,"agent":"CodexTmuxFresh","driver":"tmux_cli_session","transport":"tmux","tmux_session":"orq-codexfresh-missing","tmux_pane_id":"%9","status_path":"`+statusPath+`","heartbeat_path":"`+heartbeatPath+`"}`+"\n"), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(statusPath, []byte(`{"state":"running","updated_at":"`+now+`","alive":true,"child_pid":43210}`+"\n"), 0o600); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, []byte(`{"alive":true,"heartbeat_at":"`+now+`","started_at":"`+now+`","child_pid":43210}`+"\n"), 0o600); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}
	workerMetaJSON, _ := json.Marshal(map[string]any{
		"driver":                "tmux_cli_session",
		"tmux_session":          "orq-codexfresh-missing",
		"tmux_pane_id":          "%9",
		"worker_manifest_path":  manifestPath,
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
		"working_dir":           filepath.Join(tmp, "worker-fresh"),
		"rendered_command":      "codex-perfil CodexTmuxFresh",
	})
	if _, err := DB.Exec(`INSERT INTO runtime_handles (
		agente, proyecto_id, runtime_id, transporte, handle_kind, handle_ref, estado, metadata_json, last_seen_at
	) VALUES (?,?,?,?,?,?, 'activo', ?, ?)`,
		"CodexTmuxFresh", proyectoID, runtimeID, "tmux", "session", "orq-codexfresh-missing/%9", string(workerMetaJSON), time.Now().UTC()); err != nil {
		t.Fatalf("insert handle worker: %v", err)
	}

	handle, err := GetRuntimeHandleActivoAgenteProyecto("CodexTmuxFresh", &proyectoID)
	if err != nil {
		t.Fatalf("get handle activo: %v", err)
	}
	if handle == nil {
		t.Fatal("deberia aceptar el worker tmux fresco sin probe externo")
	}
	if got := strings.TrimSpace(handle.HandleRef); got != "orq-codexfresh-missing/%9" {
		t.Fatalf("handle inesperado: %+v", handle)
	}
}

func TestGetRuntimeHandleActivoAgenteProyectoRecuperaFallidoDesdeWorkerTMUXFresco(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("CodexTmuxRecover", "programador"); err != nil {
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

	runtimeID, err := RegistrarRuntimeInstance(&RuntimeInstance{
		Agente:       "CodexTmuxRecover",
		ProyectoID:   &proyectoID,
		LogicalState: "fallido",
		ProcessState: "fallido",
	})
	if err != nil {
		t.Fatalf("crear runtime worker: %v", err)
	}

	runDir := filepath.Join(tmp, "worker-recover")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir worker: %v", err)
	}
	manifestPath := filepath.Join(runDir, "manifest.json")
	statusPath := filepath.Join(runDir, "status.json")
	heartbeatPath := filepath.Join(runDir, "heartbeat.json")
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if err := os.WriteFile(manifestPath, []byte(`{"version":1,"agent":"CodexTmuxRecover","driver":"tmux_cli_session","transport":"tmux","tmux_session":"orq-codexrecover-1","tmux_pane_id":"%5","status_path":"`+statusPath+`","heartbeat_path":"`+heartbeatPath+`"}`+"\n"), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(statusPath, []byte(`{"state":"running","updated_at":"`+now+`","alive":true,"child_pid":54321}`+"\n"), 0o600); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, []byte(`{"alive":true,"heartbeat_at":"`+now+`","started_at":"`+now+`","child_pid":54321}`+"\n"), 0o600); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}
	workerMetaJSON, _ := json.Marshal(map[string]any{
		"driver":                "tmux_cli_session",
		"tmux_session":          "orq-codexrecover-1",
		"tmux_pane_id":          "%5",
		"worker_manifest_path":  manifestPath,
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
		"working_dir":           filepath.Join(tmp, "worker-recover"),
		"rendered_command":      "codex-perfil CodexTmuxRecover",
	})
	handleID := mustInsertID(t, `INSERT INTO runtime_handles (
		agente, proyecto_id, runtime_id, transporte, handle_kind, handle_ref, estado, metadata_json, last_seen_at
	) VALUES (?,?,?,?,?,?, 'fallido', ?, ?)`,
		"CodexTmuxRecover", proyectoID, runtimeID, "tmux", "session", "orq-codexrecover-1/%5", string(workerMetaJSON), time.Now().UTC().Add(-10*time.Minute))

	handle, err := GetRuntimeHandleActivoAgenteProyecto("CodexTmuxRecover", &proyectoID)
	if err != nil {
		t.Fatalf("get handle activo: %v", err)
	}
	if handle == nil || handle.ID != handleID || strings.TrimSpace(handle.Estado) != "activo" {
		t.Fatalf("deberia revivir el handle desde worker tmux fresco: %+v", handle)
	}
	runtime, err := GetRuntime(runtimeID)
	if err != nil {
		t.Fatalf("get runtime: %v", err)
	}
	if runtime == nil || strings.TrimSpace(runtime.LogicalState) != "activo" || strings.TrimSpace(runtime.ProcessState) != "running" {
		t.Fatalf("runtime no rehidratado desde worker fresco: %+v", runtime)
	}
}

func TestSincronizarRuntimeHandleSupervisadoCierraLegacySiExisteWorkerTMUXFresco(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex8", "programador"); err != nil {
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

	pid := int64(os.Getpid())
	runtimeLegacyID, err := RegistrarRuntimeInstance(&RuntimeInstance{
		Agente:       "Codex8",
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
		"working_dir":      filepath.Join(tmp, "legacy"),
		"rendered_command": "codex-perfil Codex8",
	})
	legacyHandleID := mustInsertID(t, `INSERT INTO runtime_handles (
		agente, proyecto_id, runtime_id, transporte, handle_kind, handle_ref, estado, metadata_json, last_seen_at
	) VALUES (?,?,?,?,?,?, 'activo', ?, ?)`,
		"Codex8", proyectoID, runtimeLegacyID, "cli", "process", strconv.Itoa(os.Getpid()), string(legacyMetaJSON), time.Now().UTC())

	runDir := filepath.Join(tmp, "worker-sync")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir worker: %v", err)
	}
	manifestPath := filepath.Join(runDir, "manifest.json")
	statusPath := filepath.Join(runDir, "status.json")
	heartbeatPath := filepath.Join(runDir, "heartbeat.json")
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if err := os.WriteFile(manifestPath, []byte(`{"version":1,"agent":"Codex8","driver":"tmux_cli_session","transport":"tmux","tmux_session":"orq-codex8-sync","tmux_pane_id":"%7","status_path":"`+statusPath+`","heartbeat_path":"`+heartbeatPath+`"}`+"\n"), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(statusPath, []byte(`{"state":"running","updated_at":"`+now+`","alive":true,"child_pid":`+strconv.Itoa(os.Getpid())+`}`+"\n"), 0o600); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, []byte(`{"alive":true,"heartbeat_at":"`+now+`","started_at":"`+now+`","child_pid":`+strconv.Itoa(os.Getpid())+`}`+"\n"), 0o600); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}

	runtimeWorkerID, err := RegistrarRuntimeInstance(&RuntimeInstance{
		Agente:       "Codex8",
		ProyectoID:   &proyectoID,
		LogicalState: "esperando_io",
		ProcessState: "running",
		PID:          &pid,
	})
	if err != nil {
		t.Fatalf("crear runtime worker: %v", err)
	}
	workerMetaJSON, _ := json.Marshal(map[string]any{
		"driver":                "tmux_cli_session",
		"tmux_session":          "orq-codex8-sync",
		"tmux_pane_id":          "%7",
		"worker_manifest_path":  manifestPath,
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
		"working_dir":           filepath.Join(tmp, "worker"),
		"rendered_command":      "codex-perfil Codex8",
	})
	if _, err := DB.Exec(`INSERT INTO runtime_handles (
		agente, proyecto_id, runtime_id, transporte, handle_kind, handle_ref, estado, metadata_json, last_seen_at
	) VALUES (?,?,?,?,?,?, 'activo', ?, ?)`,
		"Codex8", proyectoID, runtimeWorkerID, "cli", "process", strconv.Itoa(os.Getpid()), string(workerMetaJSON), time.Now().UTC()); err != nil {
		t.Fatalf("insert handle worker: %v", err)
	}

	legacyHandle, err := GetRuntimeHandle(legacyHandleID)
	if err != nil {
		t.Fatalf("get handle legacy: %v", err)
	}
	legacyRuntime, err := GetRuntime(runtimeLegacyID)
	if err != nil {
		t.Fatalf("get runtime legacy: %v", err)
	}
	refreshed, _, _, err := SincronizarRuntimeHandleSupervisado(legacyHandle, legacyRuntime, "test_sync_superseded")
	if err != nil {
		t.Fatalf("sincronizar handle legacy: %v", err)
	}
	if refreshed == nil || refreshed.Estado != "cerrado" {
		t.Fatalf("el handle legacy deberia quedar cerrado tras la sincronizacion: %+v", refreshed)
	}
}

func TestSincronizarRuntimeHandleSupervisadoNormalizaHandleTMUXCanonico(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("CodexTMUXSync", "programador"); err != nil {
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

	now := time.Now().UTC().Format(time.RFC3339Nano)
	pid := int64(os.Getpid())
	runDir := filepath.Join(tmp, "tmux-normalize")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir runDir: %v", err)
	}
	manifestPath := filepath.Join(runDir, "manifest.json")
	statusPath := filepath.Join(runDir, "status.json")
	heartbeatPath := filepath.Join(runDir, "heartbeat.json")
	if err := os.WriteFile(manifestPath, []byte(`{"version":1,"agent":"CodexTMUXSync","driver":"tmux_cli_session","transport":"tmux","tmux_session":"orq-codextmuxsync-1","tmux_pane_id":"%4","status_path":"`+statusPath+`","heartbeat_path":"`+heartbeatPath+`"}`+"\n"), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(statusPath, []byte(`{"state":"running","updated_at":"`+now+`","alive":true,"child_pid":`+strconv.Itoa(os.Getpid())+`}`+"\n"), 0o600); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, []byte(`{"alive":true,"heartbeat_at":"`+now+`","started_at":"`+now+`","child_pid":`+strconv.Itoa(os.Getpid())+`}`+"\n"), 0o600); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}

	runtimeID, err := RegistrarRuntimeInstance(&RuntimeInstance{
		Agente:       "CodexTMUXSync",
		ProyectoID:   &proyectoID,
		LogicalState: "esperando_io",
		ProcessState: "running",
		PID:          &pid,
	})
	if err != nil {
		t.Fatalf("crear runtime: %v", err)
	}
	metaJSON, _ := json.Marshal(map[string]any{
		"driver":                "tmux_cli_session",
		"tmux_session":          "orq-codextmuxsync-1",
		"tmux_pane_id":          "%4",
		"worker_manifest_path":  manifestPath,
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
		"working_dir":           filepath.Join(tmp, "orquestador"),
	})
	handleID := mustInsertID(t, `INSERT INTO runtime_handles (
		agente, proyecto_id, runtime_id, transporte, handle_kind, handle_ref, estado, metadata_json, last_seen_at
	) VALUES (?,?,?,?,?,?, 'activo', ?, ?)`,
		"CodexTMUXSync", proyectoID, runtimeID, "cli", "process", strconv.Itoa(os.Getpid()), string(metaJSON), time.Now().UTC())
	handle, err := GetRuntimeHandle(handleID)
	if err != nil {
		t.Fatalf("get handle inicial: %v", err)
	}

	refreshed, _, _, err := SincronizarRuntimeHandleSupervisado(handle, nil, "test_sync_tmux_normalize")
	if err != nil {
		t.Fatalf("sincronizar handle tmux: %v", err)
	}
	if refreshed == nil {
		t.Fatalf("handle refrescado nil")
	}
	if got := strings.TrimSpace(refreshed.Transporte); got != "tmux" {
		t.Fatalf("transporte tmux esperado, got=%q", got)
	}
	if got := strings.TrimSpace(refreshed.HandleKind); got != "session" {
		t.Fatalf("handle_kind session esperado, got=%q", got)
	}
	if got := strings.TrimSpace(refreshed.HandleRef); got != "orq-codextmuxsync-1/%4" {
		t.Fatalf("handle_ref canonico esperado, got=%q", got)
	}
}

func TestSincronizarRuntimeHandleSupervisadoNormalizaTMUXDesdeSnapshotEstructurado(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("CodexTMUXSnap", "programador"); err != nil {
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

	now := time.Now().UTC().Format(time.RFC3339Nano)
	runDir := filepath.Join(tmp, "tmux-snapshot-only")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir runDir: %v", err)
	}
	manifestPath := filepath.Join(runDir, "manifest.json")
	statusPath := filepath.Join(runDir, "status.json")
	heartbeatPath := filepath.Join(runDir, "heartbeat.json")
	if err := os.WriteFile(manifestPath, []byte(`{"version":1,"agent":"CodexTMUXSnap","driver":"tmux_cli_session","transport":"tmux","tmux_session":"orq-codextmuxsnap-1","tmux_pane_id":"%8","status_path":"`+statusPath+`","heartbeat_path":"`+heartbeatPath+`"}`+"\n"), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(statusPath, []byte(`{"state":"stopped","updated_at":"`+now+`","alive":false}`+"\n"), 0o600); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, []byte(`{"alive":false,"heartbeat_at":"`+now+`"}`+"\n"), 0o600); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}

	runtimeID, err := RegistrarRuntimeInstance(&RuntimeInstance{
		Agente:       "CodexTMUXSnap",
		ProyectoID:   &proyectoID,
		LogicalState: "degradado",
		ProcessState: "missing",
	})
	if err != nil {
		t.Fatalf("crear runtime: %v", err)
	}
	metaJSON, _ := json.Marshal(map[string]any{
		"worker_manifest_path":  manifestPath,
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
	})
	handleID := mustInsertID(t, `INSERT INTO runtime_handles (
		agente, proyecto_id, runtime_id, transporte, handle_kind, handle_ref, estado, metadata_json, last_seen_at
	) VALUES (?,?,?,?,?,?, 'fallido', ?, ?)`,
		"CodexTMUXSnap", proyectoID, runtimeID, "cli", "process", "99999", string(metaJSON), time.Now().UTC())
	handle, err := GetRuntimeHandle(handleID)
	if err != nil {
		t.Fatalf("get handle inicial: %v", err)
	}

	refreshed, _, _, err := SincronizarRuntimeHandleSupervisado(handle, nil, "test_sync_tmux_snapshot_only")
	if err != nil {
		t.Fatalf("sincronizar handle snapshot-only: %v", err)
	}
	if refreshed == nil {
		t.Fatalf("handle refrescado nil")
	}
	if got := strings.TrimSpace(refreshed.Transporte); got != "tmux" {
		t.Fatalf("transporte tmux esperado, got=%q", got)
	}
	if got := strings.TrimSpace(refreshed.HandleKind); got != "session" {
		t.Fatalf("handle_kind session esperado, got=%q", got)
	}
	if got := strings.TrimSpace(refreshed.HandleRef); got != "orq-codextmuxsnap-1/%8" {
		t.Fatalf("handle_ref canonico esperado, got=%q", got)
	}
}

func TestSincronizarRuntimeHandleSupervisadoMarcaTMUXAusenteComoFallido(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("ClaudeTMUXMissing", "programador"); err != nil {
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

	fakeTmux := filepath.Join(tmp, "tmux")
	if err := os.WriteFile(fakeTmux, []byte("#!/usr/bin/env bash\nset -euo pipefail\nexit 1\n"), 0o755); err != nil {
		t.Fatalf("write fake tmux: %v", err)
	}

	pid := int64(os.Getpid())
	runtimeID, err := RegistrarRuntimeInstance(&RuntimeInstance{
		Agente:       "ClaudeTMUXMissing",
		ProyectoID:   &proyectoID,
		LogicalState: "esperando_io",
		ProcessState: "running",
		PID:          &pid,
	})
	if err != nil {
		t.Fatalf("crear runtime: %v", err)
	}
	metaJSON, _ := json.Marshal(map[string]any{
		"driver":                "tmux_cli_session",
		"transport":             "tmux",
		"tmux_command":          fakeTmux,
		"tmux_session":          "orq-claude-missing",
		"working_dir":           mustGetwdRuntimeHandleTest(t),
		"rendered_command":      "claude-code",
		"wrapped_command":       mustExecutableRuntimeHandleTest(t),
		"mailbox_delivery_mode": "interactive",
	})
	handleID := mustInsertID(t, `INSERT INTO runtime_handles (
		agente, proyecto_id, runtime_id, transporte, handle_kind, handle_ref, estado, metadata_json, last_seen_at
	) VALUES (?,?,?,?,?,?, 'activo', ?, ?)`,
		"ClaudeTMUXMissing", proyectoID, runtimeID, "tmux", "session", "orq-claude-missing/%1", string(metaJSON), time.Now().UTC())
	handle, err := GetRuntimeHandle(handleID)
	if err != nil {
		t.Fatalf("get handle inicial: %v", err)
	}

	refreshed, refreshedRuntime, _, err := SincronizarRuntimeHandleSupervisado(handle, nil, "test_sync_tmux_missing")
	if err != nil {
		t.Fatalf("sincronizar handle tmux ausente: %v", err)
	}
	if refreshed == nil || strings.TrimSpace(refreshed.Estado) != "fallido" {
		t.Fatalf("el handle tmux ausente deberia quedar fallido: %+v", refreshed)
	}
	if refreshedRuntime == nil {
		var getErr error
		refreshedRuntime, getErr = GetRuntime(runtimeID)
		if getErr != nil {
			t.Fatalf("get runtime: %v", getErr)
		}
	}
	if refreshedRuntime == nil || strings.TrimSpace(refreshedRuntime.LogicalState) != "degradado" || strings.TrimSpace(refreshedRuntime.ProcessState) != "missing" {
		t.Fatalf("runtime no degradado tras tmux ausente: %+v", refreshedRuntime)
	}
}

func TestGetRuntimeHandleActivoAgenteProyectoPrefiereTMUXSobreProcessPTYFresco(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex4", "programador"); err != nil {
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

	now := time.Now().UTC().Format(time.RFC3339Nano)
	pid := int64(os.Getpid())

	legacyDir := filepath.Join(tmp, "legacy-pty")
	if err := os.MkdirAll(legacyDir, 0o755); err != nil {
		t.Fatalf("mkdir legacy: %v", err)
	}
	legacyManifestPath := filepath.Join(legacyDir, "manifest.json")
	legacyStatusPath := filepath.Join(legacyDir, "status.json")
	legacyHeartbeatPath := filepath.Join(legacyDir, "heartbeat.json")
	if err := os.WriteFile(legacyManifestPath, []byte(`{"version":1,"agent":"Codex4","driver":"process_pty_cli","transport":"cli","status_path":"`+legacyStatusPath+`","heartbeat_path":"`+legacyHeartbeatPath+`"}`+"\n"), 0o600); err != nil {
		t.Fatalf("write legacy manifest: %v", err)
	}
	if err := os.WriteFile(legacyStatusPath, []byte(`{"state":"running","updated_at":"`+now+`","alive":true,"child_pid":`+strconv.Itoa(os.Getpid())+`}`+"\n"), 0o600); err != nil {
		t.Fatalf("write legacy status: %v", err)
	}
	if err := os.WriteFile(legacyHeartbeatPath, []byte(`{"alive":true,"heartbeat_at":"`+now+`","started_at":"`+now+`","child_pid":`+strconv.Itoa(os.Getpid())+`}`+"\n"), 0o600); err != nil {
		t.Fatalf("write legacy heartbeat: %v", err)
	}
	runtimeLegacyID, err := RegistrarRuntimeInstance(&RuntimeInstance{
		Agente:       "Codex4",
		ProyectoID:   &proyectoID,
		LogicalState: "esperando_io",
		ProcessState: "running",
		PID:          &pid,
	})
	if err != nil {
		t.Fatalf("crear runtime legacy: %v", err)
	}
	legacyMetaJSON, _ := json.Marshal(map[string]any{
		"driver":                "process_pty_cli",
		"worker_manifest_path":  legacyManifestPath,
		"worker_status_path":    legacyStatusPath,
		"worker_heartbeat_path": legacyHeartbeatPath,
		"rendered_command":      "codex-perfil Codex4",
	})
	legacyHandleID := mustInsertID(t, `INSERT INTO runtime_handles (
		agente, proyecto_id, runtime_id, transporte, handle_kind, handle_ref, estado, metadata_json, last_seen_at
	) VALUES (?,?,?,?,?,?, 'activo', ?, ?)`,
		"Codex4", proyectoID, runtimeLegacyID, "cli", "process", strconv.Itoa(os.Getpid()), string(legacyMetaJSON), time.Now().UTC())

	tmuxDir := filepath.Join(tmp, "tmux-worker")
	if err := os.MkdirAll(tmuxDir, 0o755); err != nil {
		t.Fatalf("mkdir tmux: %v", err)
	}
	tmuxManifestPath := filepath.Join(tmuxDir, "manifest.json")
	tmuxStatusPath := filepath.Join(tmuxDir, "status.json")
	tmuxHeartbeatPath := filepath.Join(tmuxDir, "heartbeat.json")
	if err := os.WriteFile(tmuxManifestPath, []byte(`{"version":1,"agent":"Codex4","driver":"tmux_cli_session","transport":"tmux","tmux_session":"orq-codex4-063552","tmux_pane_id":"%10","status_path":"`+tmuxStatusPath+`","heartbeat_path":"`+tmuxHeartbeatPath+`"}`+"\n"), 0o600); err != nil {
		t.Fatalf("write tmux manifest: %v", err)
	}
	if err := os.WriteFile(tmuxStatusPath, []byte(`{"state":"running","updated_at":"`+now+`","alive":true,"child_pid":`+strconv.Itoa(os.Getpid())+`}`+"\n"), 0o600); err != nil {
		t.Fatalf("write tmux status: %v", err)
	}
	if err := os.WriteFile(tmuxHeartbeatPath, []byte(`{"alive":true,"heartbeat_at":"`+now+`","started_at":"`+now+`","child_pid":`+strconv.Itoa(os.Getpid())+`}`+"\n"), 0o600); err != nil {
		t.Fatalf("write tmux heartbeat: %v", err)
	}
	runtimeTMUXID, err := RegistrarRuntimeInstance(&RuntimeInstance{
		Agente:       "Codex4",
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
		"tmux_session":          "orq-codex4-063552",
		"tmux_pane_id":          "%10",
		"worker_manifest_path":  tmuxManifestPath,
		"worker_status_path":    tmuxStatusPath,
		"worker_heartbeat_path": tmuxHeartbeatPath,
		"rendered_command":      "codex-perfil Codex4",
	})
	tmuxHandleID := mustInsertID(t, `INSERT INTO runtime_handles (
		agente, proyecto_id, runtime_id, transporte, handle_kind, handle_ref, estado, metadata_json, last_seen_at
	) VALUES (?,?,?,?,?,?, 'activo', ?, ?)`,
		"Codex4", proyectoID, runtimeTMUXID, "cli", "process", strconv.Itoa(os.Getpid()), string(tmuxMetaJSON), time.Now().UTC())

	handle, err := GetRuntimeHandleActivoAgenteProyecto("Codex4", &proyectoID)
	if err != nil {
		t.Fatalf("get handle activo preferente: %v", err)
	}
	if handle == nil || handle.ID != tmuxHandleID {
		t.Fatalf("deberia preferir el worker tmux fresco sobre process_pty_cli: %+v", handle)
	}
	if got := strings.TrimSpace(handle.Transporte); got != "tmux" {
		t.Fatalf("deberia normalizar transporte tmux, got=%q", got)
	}
	if got := strings.TrimSpace(handle.HandleKind); got != "session" {
		t.Fatalf("deberia normalizar handle_kind session, got=%q", got)
	}
	if got := strings.TrimSpace(handle.HandleRef); got != "orq-codex4-063552/%10" {
		t.Fatalf("deberia normalizar handle_ref canonico tmux, got=%q", got)
	}

	legacyHandle, err := GetRuntimeHandle(legacyHandleID)
	if err != nil {
		t.Fatalf("get handle legacy: %v", err)
	}
	if legacyHandle == nil || legacyHandle.Estado != "cerrado" {
		t.Fatalf("el handle process_pty_cli deberia quedar cerrado: %+v", legacyHandle)
	}

	freshTMUX, err := GetRuntimeHandle(tmuxHandleID)
	if err != nil {
		t.Fatalf("get handle tmux: %v", err)
	}
	if freshTMUX == nil {
		t.Fatalf("handle tmux nil tras normalizar")
	}
	if got := strings.TrimSpace(freshTMUX.Transporte); got != "tmux" {
		t.Fatalf("deberia persistir transporte tmux, got=%q", got)
	}
	if got := strings.TrimSpace(freshTMUX.HandleKind); got != "session" {
		t.Fatalf("deberia persistir handle_kind session, got=%q", got)
	}
	if got := strings.TrimSpace(freshTMUX.HandleRef); got != "orq-codex4-063552/%10" {
		t.Fatalf("deberia persistir handle_ref canonico tmux, got=%q", got)
	}
}

func TestNormalizarRuntimeHandleTMUXCanonicoToleraMetadataInvalidaNoTMUX(t *testing.T) {
	prepararDBTemporal(t)

	if err := RegistrarAgente("ProcessBrokenMeta", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	handleID := mustInsertID(t, `INSERT INTO runtime_handles (
		agente, transporte, handle_kind, handle_ref, estado, metadata_json, last_seen_at
	) VALUES (?,?,?,?,?,?,?)`,
		"ProcessBrokenMeta", "cli", "process", "12345", "activo", "{not-json", time.Now().UTC())
	handle, err := GetRuntimeHandle(handleID)
	if err != nil {
		t.Fatalf("get handle: %v", err)
	}

	normalizado, err := normalizarRuntimeHandleTMUXCanonico(handle)
	if err != nil {
		t.Fatalf("normalizar handle no tmux no deberia fallar: %v", err)
	}
	if normalizado == nil {
		t.Fatalf("handle normalizado nil")
	}
	if got := strings.TrimSpace(normalizado.Transporte); got != "cli" {
		t.Fatalf("transporte no deberia cambiar, got=%q", got)
	}
	if got := strings.TrimSpace(normalizado.HandleKind); got != "process" {
		t.Fatalf("handle_kind no deberia cambiar, got=%q", got)
	}
	if got := strings.TrimSpace(normalizado.HandleRef); got != "12345" {
		t.Fatalf("handle_ref no deberia cambiar, got=%q", got)
	}
}

func TestGetRuntimeHandleActivoAgenteProyectoInvalidatesHotCacheTrasActualizarMetadataArranque(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex9", "programador"); err != nil {
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

	now := time.Now().UTC().Format(time.RFC3339Nano)
	pid := int64(os.Getpid())
	runtimeID, err := RegistrarRuntimeInstance(&RuntimeInstance{
		Agente:       "Codex9",
		ProyectoID:   &proyectoID,
		LogicalState: "esperando_io",
		ProcessState: "running",
		PID:          &pid,
	})
	if err != nil {
		t.Fatalf("crear runtime: %v", err)
	}
	tmuxDir := filepath.Join(tmp, "tmux-initial")
	if err := os.MkdirAll(tmuxDir, 0o755); err != nil {
		t.Fatalf("mkdir tmux: %v", err)
	}
	manifestPath := filepath.Join(tmuxDir, "manifest.json")
	statusPath := filepath.Join(tmuxDir, "status.json")
	heartbeatPath := filepath.Join(tmuxDir, "heartbeat.json")
	if err := os.WriteFile(manifestPath, []byte(`{"version":1,"agent":"Codex9","driver":"tmux_cli_session","transport":"tmux","tmux_session":"orq-codex9-cache","tmux_pane_id":"%12","status_path":"`+statusPath+`","heartbeat_path":"`+heartbeatPath+`"}`+"\n"), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(statusPath, []byte(`{"state":"running","updated_at":"`+now+`","alive":true,"child_pid":`+strconv.Itoa(os.Getpid())+`}`+"\n"), 0o600); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, []byte(`{"alive":true,"heartbeat_at":"`+now+`","started_at":"`+now+`","child_pid":`+strconv.Itoa(os.Getpid())+`}`+"\n"), 0o600); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}
	initialMetaJSON, _ := json.Marshal(map[string]any{
		"driver":                "tmux_cli_session",
		"tmux_session":          "orq-codex9-cache",
		"tmux_pane_id":          "%12",
		"worker_manifest_path":  manifestPath,
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
		"rendered_command":      "codex-perfil Codex9",
		"working_dir":           filepath.Join(tmp, "old"),
	})
	handleID := mustInsertID(t, `INSERT INTO runtime_handles (
		agente, proyecto_id, runtime_id, transporte, handle_kind, handle_ref, estado, metadata_json, last_seen_at
	) VALUES (?,?,?,?,?,?, 'activo', ?, ?)`,
		"Codex9", proyectoID, runtimeID, "tmux", "session", "orq-codex9-cache", string(initialMetaJSON), time.Now().UTC())

	handle, err := GetRuntimeHandleActivoAgenteProyecto("Codex9", &proyectoID)
	if err != nil {
		t.Fatalf("get handle activo inicial: %v", err)
	}
	if handle == nil || handle.ID != handleID {
		t.Fatalf("deberia recuperar el handle activo inicial: %+v", handle)
	}

	newWorkingDir := filepath.Join(tmp, "new")
	arranque := &controlruntime.ProcesoArrancado{
		PID:             os.Getpid(),
		HandleKind:      "session",
		HandleRef:       "orq-codex9-cache",
		WorkingDir:      newWorkingDir,
		RenderedCommand: "codex-perfil Codex9",
		MetadataJSON:    `{"driver":"tmux_cli_session","tmux_session":"orq-codex9-cache","tmux_pane_id":"%12"}`,
	}
	if err := actualizarHandleRuntimeArranque(handleID, arranque, nil, nil, runtimeagente.ResumeContext{}); err != nil {
		t.Fatalf("actualizar arranque: %v", err)
	}

	handle, err = GetRuntimeHandleActivoAgenteProyecto("Codex9", &proyectoID)
	if err != nil {
		t.Fatalf("get handle activo tras actualizar metadata: %v", err)
	}
	if handle == nil || handle.ID != handleID {
		t.Fatalf("deberia seguir siendo el mismo handle: %+v", handle)
	}
	meta := mapFromJSON(handle.MetadataJSON)
	if got := strings.TrimSpace(stringFromMap(meta, "working_dir", "")); got != newWorkingDir {
		t.Fatalf("deberia ver working_dir actualizado tras invalidar cache, got=%q want=%q", got, newWorkingDir)
	}
	if got := strings.TrimSpace(handle.HandleKind); got != "session" {
		t.Fatalf("deberia ver handle_kind actualizado tras invalidar cache, got=%q", got)
	}
}

func TestMarcarRuntimeHandleSupersededDetieneSesionTMUX(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("CodexTMUX", "programador"); err != nil {
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

	invocations := filepath.Join(tmp, "tmux.log")
	fakeTmux := filepath.Join(tmp, "tmux")
	script := "#!/usr/bin/env bash\n" +
		"printf '%s\\n' \"$*\" >> " + strconv.Quote(invocations) + "\n" +
		"exit 0\n"
	if err := os.WriteFile(fakeTmux, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake tmux: %v", err)
	}

	pid := int64(os.Getpid())
	runtimeID, err := RegistrarRuntimeInstance(&RuntimeInstance{
		Agente:       "CodexTMUX",
		ProyectoID:   &proyectoID,
		LogicalState: "esperando_io",
		ProcessState: "running",
		PID:          &pid,
	})
	if err != nil {
		t.Fatalf("crear runtime: %v", err)
	}
	metaJSON, _ := json.Marshal(map[string]any{
		"driver":       "tmux_cli_session",
		"tmux_command": fakeTmux,
		"tmux_session": "orq-codextmux-1",
	})
	handleID := mustInsertID(t, `INSERT INTO runtime_handles (
		agente, proyecto_id, runtime_id, transporte, handle_kind, handle_ref, estado, metadata_json, last_seen_at
	) VALUES (?,?,?,?,?,?, 'activo', ?, ?)`,
		"CodexTMUX", proyectoID, runtimeID, "tmux", "session", "orq-codextmux-1", string(metaJSON), time.Now().UTC())

	handle, err := GetRuntimeHandle(handleID)
	if err != nil {
		t.Fatalf("get handle: %v", err)
	}
	if err := marcarRuntimeHandleSuperseded(handle); err != nil {
		t.Fatalf("marcar superseded: %v", err)
	}

	fresh, err := GetRuntimeHandle(handleID)
	if err != nil {
		t.Fatalf("get handle refreshed: %v", err)
	}
	if fresh == nil || fresh.Estado != "cerrado" {
		t.Fatalf("el handle superseded deberia quedar cerrado: %+v", fresh)
	}

	data, err := os.ReadFile(invocations)
	if err != nil {
		t.Fatalf("read tmux log: %v", err)
	}
	if !strings.Contains(string(data), "kill-session -t orq-codextmux-1") {
		t.Fatalf("deberia matar la sesion tmux del handle superseded: %s", string(data))
	}
}

func TestMarcarRuntimeHandleFantasmaDetieneSesionTMUX(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("CodexGhost", "programador"); err != nil {
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

	invocations := filepath.Join(tmp, "tmux.log")
	fakeTmux := filepath.Join(tmp, "tmux")
	script := "#!/usr/bin/env bash\n" +
		"printf '%s\\n' \"$*\" >> " + strconv.Quote(invocations) + "\n" +
		"exit 0\n"
	if err := os.WriteFile(fakeTmux, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake tmux: %v", err)
	}

	pid := int64(os.Getpid())
	runtimeID, err := RegistrarRuntimeInstance(&RuntimeInstance{
		Agente:       "CodexGhost",
		ProyectoID:   &proyectoID,
		LogicalState: "esperando_io",
		ProcessState: "running",
		PID:          &pid,
	})
	if err != nil {
		t.Fatalf("crear runtime: %v", err)
	}
	metaJSON, _ := json.Marshal(map[string]any{
		"driver":       "tmux_cli_session",
		"tmux_command": fakeTmux,
		"tmux_session": "orq-codexghost-1",
	})
	handleID := mustInsertID(t, `INSERT INTO runtime_handles (
		agente, proyecto_id, runtime_id, transporte, handle_kind, handle_ref, estado, metadata_json, last_seen_at
	) VALUES (?,?,?,?,?,?, 'activo', ?, ?)`,
		"CodexGhost", proyectoID, runtimeID, "tmux", "session", "orq-codexghost-1", string(metaJSON), time.Now().UTC())

	handle, err := GetRuntimeHandle(handleID)
	if err != nil {
		t.Fatalf("get handle: %v", err)
	}
	if err := marcarRuntimeHandleFantasma(handle); err != nil {
		t.Fatalf("marcar fantasma: %v", err)
	}

	fresh, err := GetRuntimeHandle(handleID)
	if err != nil {
		t.Fatalf("get handle refreshed: %v", err)
	}
	if fresh == nil || fresh.Estado != "fallido" {
		t.Fatalf("el handle fantasma deberia quedar fallido: %+v", fresh)
	}

	data, err := os.ReadFile(invocations)
	if err != nil {
		t.Fatalf("read tmux log: %v", err)
	}
	if !strings.Contains(string(data), "kill-session -t orq-codexghost-1") {
		t.Fatalf("deberia matar la sesion tmux del handle fantasma: %s", string(data))
	}
}
