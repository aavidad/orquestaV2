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

func TestResolverHandleYRuntimeParaOrdenPrefierenTMUXCanonicoSobreLegacy(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("CodexResolver", "programador"); err != nil {
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

	runtimeLegacyID, err := RegistrarRuntimeInstance(&RuntimeInstance{
		Agente:       "CodexResolver",
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
		"rendered_command": "codex-perfil CodexResolver",
	})
	legacyHandleID := mustInsertID(t, `INSERT INTO runtime_handles (
		agente, proyecto_id, runtime_id, transporte, handle_kind, handle_ref, estado, metadata_json, last_seen_at
	) VALUES (?,?,?,?,?,?, 'activo', ?, ?)`,
		"CodexResolver", proyectoID, runtimeLegacyID, "cli", "process", strconv.Itoa(os.Getpid()), string(legacyMetaJSON), time.Now().UTC())

	tmuxDir := filepath.Join(tmp, "tmux-worker")
	if err := os.MkdirAll(tmuxDir, 0o755); err != nil {
		t.Fatalf("mkdir tmux: %v", err)
	}
	manifestPath := filepath.Join(tmuxDir, "manifest.json")
	statusPath := filepath.Join(tmuxDir, "status.json")
	heartbeatPath := filepath.Join(tmuxDir, "heartbeat.json")
	if err := os.WriteFile(manifestPath, []byte(`{"version":1,"agent":"CodexResolver","driver":"tmux_cli_session","transport":"tmux","tmux_session":"orq-codexresolver-1","tmux_pane_id":"%17","status_path":"`+statusPath+`","heartbeat_path":"`+heartbeatPath+`"}`+"\n"), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(statusPath, []byte(`{"state":"running","updated_at":"`+now+`","alive":true,"child_pid":`+strconv.Itoa(os.Getpid())+`}`+"\n"), 0o600); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, []byte(`{"alive":true,"heartbeat_at":"`+now+`","started_at":"`+now+`","child_pid":`+strconv.Itoa(os.Getpid())+`}`+"\n"), 0o600); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}

	runtimeTMUXID, err := RegistrarRuntimeInstance(&RuntimeInstance{
		Agente:       "CodexResolver",
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
		"tmux_session":          "orq-codexresolver-1",
		"tmux_pane_id":          "%17",
		"worker_manifest_path":  manifestPath,
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
		"rendered_command":      "codex-perfil CodexResolver",
	})
	tmuxHandleID := mustInsertID(t, `INSERT INTO runtime_handles (
		agente, proyecto_id, runtime_id, transporte, handle_kind, handle_ref, estado, metadata_json, last_seen_at
	) VALUES (?,?,?,?,?,?, 'activo', ?, ?)`,
		"CodexResolver", proyectoID, runtimeTMUXID, "tmux", "session", "orq-codexresolver-1", string(tmuxMetaJSON), time.Now().UTC())

	orderID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:     "CodexResolver",
		ProyectoID: &proyectoID,
		RuntimeID:  &runtimeLegacyID,
		HandleID:   &legacyHandleID,
		Tipo:       "pause",
	})
	if err != nil {
		t.Fatalf("encolar runtime order: %v", err)
	}
	order, err := GetRuntimeOrder(orderID)
	if err != nil {
		t.Fatalf("get runtime order: %v", err)
	}

	handle, err := resolverHandleParaOrden(order)
	if err != nil {
		t.Fatalf("resolver handle: %v", err)
	}
	if handle == nil || handle.ID != tmuxHandleID {
		t.Fatalf("deberia resolver al handle tmux canónico: %+v", handle)
	}

	runtime, err := resolverRuntimeParaOrden(order)
	if err != nil {
		t.Fatalf("resolver runtime: %v", err)
	}
	if runtime == nil || runtime.ID != runtimeTMUXID {
		t.Fatalf("deberia resolver al runtime tmux canónico: %+v", runtime)
	}
}

func TestRuntimePrincipalAgenteProyectoOmiteRuntimeSinSesionOperativaNiHandleCanonica(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("CodexOrphan", "programador"); err != nil {
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
		"CodexOrphan", proyectoID, 0, "cerrada", "codex-cli", "test-host", time.Now().UTC().Add(-2*time.Hour))

	runtimeID, err := RegistrarRuntimeInstance(&RuntimeInstance{
		Agente:       "CodexOrphan",
		ProyectoID:   &proyectoID,
		SesionID:     &sesionID,
		LogicalState: "disponible",
		ProcessState: "desconocido",
	})
	if err != nil {
		t.Fatalf("crear runtime orphan: %v", err)
	}
	now := time.Now().UTC()
	if _, err := DB.Exec(`UPDATE runtime_instances
		SET last_event_at=?, last_heartbeat_at=?, updated_at=?
		WHERE id=?`, now, now, now, runtimeID); err != nil {
		t.Fatalf("actualizar runtime orphan: %v", err)
	}

	runtime, err := runtimePrincipalAgenteProyecto("CodexOrphan", &proyectoID)
	if err != nil {
		t.Fatalf("runtimePrincipalAgenteProyecto: %v", err)
	}
	if runtime != nil {
		t.Fatalf("no deberia elegir una runtime sin sesion operativa ni handle canónica: %+v", runtime)
	}
}

func TestRuntimePrincipalAgenteProyectoAceptaRuntimeConSesionActivaSinHandle(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("CodexRuntimeOnly", "programador"); err != nil {
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
	) VALUES (?,?,?,?,?,?,CURRENT_TIMESTAMP)`,
		"CodexRuntimeOnly", proyectoID, 1, "activa", "codex-cli", "test-host")

	runtimeID, err := RegistrarRuntimeInstance(&RuntimeInstance{
		Agente:       "CodexRuntimeOnly",
		ProyectoID:   &proyectoID,
		SesionID:     &sesionID,
		LogicalState: "esperando_io",
		ProcessState: "running",
	})
	if err != nil {
		t.Fatalf("crear runtime sin handle: %v", err)
	}
	now := time.Now().UTC()
	if _, err := DB.Exec(`UPDATE runtime_instances
		SET last_event_at=?, last_heartbeat_at=?, updated_at=?
		WHERE id=?`, now, now, now, runtimeID); err != nil {
		t.Fatalf("actualizar runtime activo: %v", err)
	}

	runtime, err := runtimePrincipalAgenteProyecto("CodexRuntimeOnly", &proyectoID)
	if err != nil {
		t.Fatalf("runtimePrincipalAgenteProyecto: %v", err)
	}
	if runtime == nil || runtime.ID != runtimeID {
		t.Fatalf("deberia aceptar la runtime con sesion activa aunque no haya handle: %+v", runtime)
	}
}

func TestRuntimeOrderRuntimeOperativoConSesionActivaOmiteHeartbeatStale(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("CodexStartGuard", "programador"); err != nil {
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
		"CodexStartGuard", proyectoID, 1, "activa", "codex-cli", "test-host", time.Now().UTC().Add(-2*time.Hour))

	runtimeID, err := RegistrarRuntimeInstance(&RuntimeInstance{
		Agente:       "CodexStartGuard",
		ProyectoID:   &proyectoID,
		SesionID:     &sesionID,
		LogicalState: "activo",
		ProcessState: "running",
	})
	if err != nil {
		t.Fatalf("crear runtime guard: %v", err)
	}

	order := &RuntimeOrder{Agente: "CodexStartGuard", ProyectoID: &proyectoID, Tipo: "start"}
	runtime, err := GetRuntime(runtimeID)
	if err != nil {
		t.Fatalf("GetRuntime: %v", err)
	}
	if runtimeOrderRuntimeOperativoConSesionActiva(order, runtime) {
		t.Fatal("una sesion activa sin heartbeat operativo no deberia satisfacer start")
	}
}

func TestResolverRuntimeParaOrdenConSoloRuntimeIDPrefiereTMUXCanonico(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("CodexResolver", "programador"); err != nil {
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

	runtimeLegacyID, err := RegistrarRuntimeInstance(&RuntimeInstance{
		Agente:       "CodexResolver",
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
		"rendered_command": "codex-perfil CodexResolver",
	})
	if _, err := DB.Exec(`INSERT INTO runtime_handles (
		agente, proyecto_id, runtime_id, transporte, handle_kind, handle_ref, estado, metadata_json, last_seen_at
	) VALUES (?,?,?,?,?,?, 'activo', ?, ?)`,
		"CodexResolver", proyectoID, runtimeLegacyID, "cli", "process", strconv.Itoa(os.Getpid()), string(legacyMetaJSON), time.Now().UTC()); err != nil {
		t.Fatalf("insert handle legacy: %v", err)
	}

	tmuxDir := filepath.Join(tmp, "tmux-worker-runtime-only")
	if err := os.MkdirAll(tmuxDir, 0o755); err != nil {
		t.Fatalf("mkdir tmux: %v", err)
	}
	manifestPath := filepath.Join(tmuxDir, "manifest.json")
	statusPath := filepath.Join(tmuxDir, "status.json")
	heartbeatPath := filepath.Join(tmuxDir, "heartbeat.json")
	if err := os.WriteFile(manifestPath, []byte(`{"version":1,"agent":"CodexResolver","driver":"tmux_cli_session","transport":"tmux","tmux_session":"orq-codexresolver-rt","tmux_pane_id":"%27","status_path":"`+statusPath+`","heartbeat_path":"`+heartbeatPath+`"}`+"\n"), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(statusPath, []byte(`{"state":"running","updated_at":"`+now+`","alive":true,"child_pid":`+strconv.Itoa(os.Getpid())+`}`+"\n"), 0o600); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, []byte(`{"alive":true,"heartbeat_at":"`+now+`","started_at":"`+now+`","child_pid":`+strconv.Itoa(os.Getpid())+`}`+"\n"), 0o600); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}

	runtimeTMUXID, err := RegistrarRuntimeInstance(&RuntimeInstance{
		Agente:       "CodexResolver",
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
		"tmux_session":          "orq-codexresolver-rt",
		"tmux_pane_id":          "%27",
		"worker_manifest_path":  manifestPath,
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
	})
	if _, err := DB.Exec(`INSERT INTO runtime_handles (
		agente, proyecto_id, runtime_id, transporte, handle_kind, handle_ref, estado, metadata_json, last_seen_at
	) VALUES (?,?,?,?,?,?, 'activo', ?, ?)`,
		"CodexResolver", proyectoID, runtimeTMUXID, "tmux", "session", "orq-codexresolver-rt", string(tmuxMetaJSON), time.Now().UTC()); err != nil {
		t.Fatalf("insert handle tmux: %v", err)
	}

	orderID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:     "CodexResolver",
		ProyectoID: &proyectoID,
		RuntimeID:  &runtimeLegacyID,
		Tipo:       "pause",
	})
	if err != nil {
		t.Fatalf("encolar runtime order: %v", err)
	}
	order, err := GetRuntimeOrder(orderID)
	if err != nil {
		t.Fatalf("get runtime order: %v", err)
	}

	runtime, err := resolverRuntimeParaOrden(order)
	if err != nil {
		t.Fatalf("resolver runtime: %v", err)
	}
	if runtime == nil || runtime.ID != runtimeTMUXID {
		t.Fatalf("deberia resolver al runtime tmux canónico aunque la orden solo tenga runtime_id viejo: %+v", runtime)
	}
}

func TestRuntimePrincipalAgenteProyectoPrefiereRuntimeDelHandleCanonicoReciente(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("CodexResolver", "programador"); err != nil {
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
	now := time.Now().UTC()

	runtimeLegacyID, err := RegistrarRuntimeInstance(&RuntimeInstance{
		Agente:       "CodexResolver",
		ProyectoID:   &proyectoID,
		LogicalState: "esperando_io",
		ProcessState: "running",
		PID:          &pid,
	})
	if err != nil {
		t.Fatalf("crear runtime legacy: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_instances
		SET last_event_at=?, updated_at=?
		WHERE id=?`, now, now, runtimeLegacyID); err != nil {
		t.Fatalf("actualizar runtime legacy: %v", err)
	}
	legacyMetaJSON, _ := json.Marshal(map[string]any{
		"driver":           "process_pty_cli",
		"rendered_command": "codex-perfil CodexResolver",
	})
	if _, err := DB.Exec(`INSERT INTO runtime_handles (
		agente, proyecto_id, runtime_id, transporte, handle_kind, handle_ref, estado, metadata_json, last_seen_at
	) VALUES (?,?,?,?,?,?, 'activo', ?, ?)`,
		"CodexResolver", proyectoID, runtimeLegacyID, "cli", "process", strconv.Itoa(os.Getpid()), string(legacyMetaJSON), now); err != nil {
		t.Fatalf("insert handle legacy: %v", err)
	}

	tmuxDir := filepath.Join(tmp, "tmux-worker-principal")
	if err := os.MkdirAll(tmuxDir, 0o755); err != nil {
		t.Fatalf("mkdir tmux: %v", err)
	}
	manifestPath := filepath.Join(tmuxDir, "manifest.json")
	statusPath := filepath.Join(tmuxDir, "status.json")
	heartbeatPath := filepath.Join(tmuxDir, "heartbeat.json")
	nowText := now.Format(time.RFC3339Nano)
	if err := os.WriteFile(manifestPath, []byte(`{"version":1,"agent":"CodexResolver","driver":"tmux_cli_session","transport":"tmux","tmux_session":"orq-codexresolver-main","tmux_pane_id":"%31","status_path":"`+statusPath+`","heartbeat_path":"`+heartbeatPath+`"}`+"\n"), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(statusPath, []byte(`{"state":"running","updated_at":"`+nowText+`","alive":true,"child_pid":`+strconv.Itoa(os.Getpid())+`}`+"\n"), 0o600); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, []byte(`{"alive":true,"heartbeat_at":"`+nowText+`","started_at":"`+nowText+`","child_pid":`+strconv.Itoa(os.Getpid())+`}`+"\n"), 0o600); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}

	runtimeTMUXID, err := RegistrarRuntimeInstance(&RuntimeInstance{
		Agente:       "CodexResolver",
		ProyectoID:   &proyectoID,
		LogicalState: "esperando_io",
		ProcessState: "running",
		PID:          &pid,
	})
	if err != nil {
		t.Fatalf("crear runtime tmux: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_instances
		SET last_event_at=?, updated_at=?
		WHERE id=?`, now.Add(-time.Hour), now.Add(-time.Hour), runtimeTMUXID); err != nil {
		t.Fatalf("actualizar runtime tmux: %v", err)
	}
	tmuxMetaJSON, _ := json.Marshal(map[string]any{
		"driver":                "tmux_cli_session",
		"tmux_session":          "orq-codexresolver-main",
		"tmux_pane_id":          "%31",
		"worker_manifest_path":  manifestPath,
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
	})
	if _, err := DB.Exec(`INSERT INTO runtime_handles (
		agente, proyecto_id, runtime_id, transporte, handle_kind, handle_ref, estado, metadata_json, last_seen_at
	) VALUES (?,?,?,?,?,?, 'activo', ?, ?)`,
		"CodexResolver", proyectoID, runtimeTMUXID, "tmux", "session", "orq-codexresolver-main", string(tmuxMetaJSON), now); err != nil {
		t.Fatalf("insert handle tmux: %v", err)
	}

	runtime, err := runtimePrincipalAgenteProyecto("CodexResolver", &proyectoID)
	if err != nil {
		t.Fatalf("runtimePrincipalAgenteProyecto: %v", err)
	}
	if runtime == nil || runtime.ID != runtimeTMUXID {
		t.Fatalf("deberia preferir el runtime del handle tmux canónico reciente: %+v", runtime)
	}
}

func TestResolverSesionParaOrdenPrefiereSesionDelTMUXCanonico(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("CodexResolver", "programador"); err != nil {
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

	sesionLegacyID := mustInsertID(t, `INSERT INTO sesiones (
		agente, proyecto_id, activa, estado, herramienta, host, heartbeat_at
	) VALUES (?,?,?,?,?,?,CURRENT_TIMESTAMP)`,
		"CodexResolver", proyectoID, 0, "cerrada", "codex-cli", "test-host")

	sesionTMUXID := mustInsertID(t, `INSERT INTO sesiones (
		agente, proyecto_id, activa, estado, herramienta, host, heartbeat_at
	) VALUES (?,?,?,?,?,?,CURRENT_TIMESTAMP)`,
		"CodexResolver", proyectoID, 1, "activa", "codex-cli", "test-host")

	now := time.Now().UTC()
	nowText := now.Format(time.RFC3339Nano)
	pid := int64(os.Getpid())

	runtimeLegacyID, err := RegistrarRuntimeInstance(&RuntimeInstance{
		Agente:       "CodexResolver",
		ProyectoID:   &proyectoID,
		SesionID:     &sesionLegacyID,
		LogicalState: "esperando_io",
		ProcessState: "running",
		PID:          &pid,
		Model:        "gpt-4.1",
	})
	if err != nil {
		t.Fatalf("crear runtime legacy: %v", err)
	}
	legacyMetaJSON, _ := json.Marshal(map[string]any{
		"driver":           "process_pty_cli",
		"working_dir":      filepath.Join(tmp, "legacy"),
		"rendered_command": "'/tmp/codex-perfiles/bin/codex-perfil' 'CuentaLegacy'",
	})
	legacyHandleID := mustInsertID(t, `INSERT INTO runtime_handles (
		agente, proyecto_id, runtime_id, sesion_id, transporte, handle_kind, handle_ref, estado, metadata_json, last_seen_at
	) VALUES (?,?,?,?,?,?,?, 'activo', ?, ?)`,
		"CodexResolver", proyectoID, runtimeLegacyID, sesionLegacyID, "cli", "process", strconv.Itoa(os.Getpid()), string(legacyMetaJSON), now)

	tmuxDir := filepath.Join(tmp, "tmux-worker-sesion")
	if err := os.MkdirAll(tmuxDir, 0o755); err != nil {
		t.Fatalf("mkdir tmux: %v", err)
	}
	manifestPath := filepath.Join(tmuxDir, "manifest.json")
	statusPath := filepath.Join(tmuxDir, "status.json")
	heartbeatPath := filepath.Join(tmuxDir, "heartbeat.json")
	if err := os.WriteFile(manifestPath, []byte(`{"version":1,"agent":"CodexResolver","driver":"tmux_cli_session","transport":"tmux","tmux_session":"orq-codexresolver-sesion","tmux_pane_id":"%37","status_path":"`+statusPath+`","heartbeat_path":"`+heartbeatPath+`"}`+"\n"), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(statusPath, []byte(`{"state":"running","updated_at":"`+nowText+`","alive":true,"child_pid":`+strconv.Itoa(os.Getpid())+`}`+"\n"), 0o600); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, []byte(`{"alive":true,"heartbeat_at":"`+nowText+`","started_at":"`+nowText+`","child_pid":`+strconv.Itoa(os.Getpid())+`}`+"\n"), 0o600); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}

	runtimeTMUXID, err := RegistrarRuntimeInstance(&RuntimeInstance{
		Agente:       "CodexResolver",
		ProyectoID:   &proyectoID,
		SesionID:     &sesionTMUXID,
		LogicalState: "esperando_io",
		ProcessState: "running",
		PID:          &pid,
		Model:        "gpt-5.4",
	})
	if err != nil {
		t.Fatalf("crear runtime tmux: %v", err)
	}
	tmuxMetaJSON, _ := json.Marshal(map[string]any{
		"driver":                "tmux_cli_session",
		"tmux_session":          "orq-codexresolver-sesion",
		"tmux_pane_id":          "%37",
		"worker_manifest_path":  manifestPath,
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
		"rendered_command":      "'/tmp/codex-perfiles/bin/codex-perfil' 'CuentaTMUX'",
	})
	tmuxHandleID := mustInsertID(t, `INSERT INTO runtime_handles (
		agente, proyecto_id, runtime_id, sesion_id, transporte, handle_kind, handle_ref, estado, metadata_json, last_seen_at
	) VALUES (?,?,?,?,?,?,?, 'activo', ?, ?)`,
		"CodexResolver", proyectoID, runtimeTMUXID, sesionTMUXID, "tmux", "session", "orq-codexresolver-sesion", string(tmuxMetaJSON), now)

	orderID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:     "CodexResolver",
		ProyectoID: &proyectoID,
		RuntimeID:  &runtimeLegacyID,
		HandleID:   &legacyHandleID,
		Tipo:       "pause",
	})
	if err != nil {
		t.Fatalf("encolar runtime order: %v", err)
	}
	order, err := GetRuntimeOrder(orderID)
	if err != nil {
		t.Fatalf("get runtime order: %v", err)
	}

	handle, err := resolverHandleParaOrden(order)
	if err != nil {
		t.Fatalf("resolver handle: %v", err)
	}
	if handle == nil || handle.ID != tmuxHandleID {
		t.Fatalf("deberia resolver el handle tmux canónico antes de la sesión: %+v", handle)
	}

	sesion, err := resolverSesionParaOrden(order)
	if err != nil {
		t.Fatalf("resolver sesion: %v", err)
	}
	if sesion == nil || sesion.ID != sesionTMUXID {
		t.Fatalf("deberia resolver la sesión del runtime tmux canónico: %+v", sesion)
	}

	sesionID, model, err := resolverSesionYModeloRuntimeOrder(order)
	if err != nil {
		t.Fatalf("resolver sesion y modelo: %v", err)
	}
	if sesionID != sesionTMUXID {
		t.Fatalf("deberia resolver el sesion_id del runtime tmux canónico, got=%d", sesionID)
	}
	if model != "gpt-5.4" {
		t.Fatalf("deberia resolver el modelo del runtime tmux canónico, got=%q", model)
	}

	identidad := identidadObservadaDesdeOrdenRuntime(order)
	if got := strings.TrimSpace(stringFromMap(identidad, "account_user", "")); got != "CuentaTMUX" {
		t.Fatalf("deberia observar la identidad del handle tmux canónico, got=%+v", identidad)
	}
}

func TestRuntimeHandleRuntimeHaceFallbackASesionID(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("CodexResolver", "programador"); err != nil {
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
	) VALUES (?,?,?,?,?,?,CURRENT_TIMESTAMP)`,
		"CodexResolver", proyectoID, 1, "activa", "codex-cli", "test-host")

	pid := int64(os.Getpid())
	runtimeID, err := RegistrarRuntimeInstance(&RuntimeInstance{
		Agente:       "CodexResolver",
		ProyectoID:   &proyectoID,
		SesionID:     &sesionID,
		LogicalState: "esperando_io",
		ProcessState: "running",
		PID:          &pid,
		Model:        "gpt-5.4",
	})
	if err != nil {
		t.Fatalf("registrar runtime: %v", err)
	}
	metaJSON, _ := json.Marshal(map[string]any{
		"driver":       "tmux_cli_session",
		"tmux_session": "orq-codexresolver-runtime-helper",
	})
	handleID := mustInsertID(t, `INSERT INTO runtime_handles (
		agente, proyecto_id, sesion_id, transporte, handle_kind, handle_ref, estado, metadata_json, last_seen_at
	) VALUES (?,?,?,?,?,?, 'activo', ?, CURRENT_TIMESTAMP)`,
		"CodexResolver", proyectoID, sesionID, "tmux", "session", "orq-codexresolver-runtime-helper", string(metaJSON))
	handle, err := GetRuntimeHandle(handleID)
	if err != nil {
		t.Fatalf("get handle: %v", err)
	}
	if handle.RuntimeID != nil {
		t.Fatalf("el setup de prueba requiere handle sin runtime_id enlazado: %+v", handle)
	}

	runtime, err := runtimeHandleRuntime(handle)
	if err != nil {
		t.Fatalf("runtimeHandleRuntime: %v", err)
	}
	if runtime == nil || runtime.ID != runtimeID {
		t.Fatalf("deberia hacer fallback a runtime por sesion_id: %+v", runtime)
	}
}
