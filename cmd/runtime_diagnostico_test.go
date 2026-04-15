package cmd

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"orquesta/db"
)

func TestRuntimeDiagnosticoUsaAPI(t *testing.T) {
	now := time.Date(2026, 3, 24, 20, 0, 0, 0, time.UTC)
	sesionID := int64(21)
	runtime := &db.RuntimeInstance{
		ID:              7,
		Agente:          "Codex1",
		ProyectoSlug:    "orquestador",
		Provider:        "openai",
		Connector:       "codex-cli",
		LogicalState:    "vivo",
		Branch:          "main",
		LastHeartbeatAt: &now,
	}
	handle := &db.RuntimeHandle{
		ID:         9,
		Agente:     "Codex1",
		Transporte: "stdio",
		HandleKind: "session",
		HandleRef:  "sess-runtime-api",
		Estado:     "vivo",
		LastSeenAt: &now,
		SesionID:   &sesionID,
	}
	order := &db.RuntimeOrder{
		ID:        11,
		Agente:    "Codex1",
		Tipo:      "checkpoint",
		Estado:    "pendiente",
		CreatedAt: now,
	}
	msg := &db.RuntimeMailboxMessage{
		ID:         13,
		FromAgente: "Supervisor",
		ToAgente:   "Codex1",
		Kind:       "nudge",
		Estado:     "pendiente",
		CreatedAt:  now,
	}
	cp := &db.RuntimeCheckpoint{
		ID:             17,
		Agente:         "Codex1",
		CheckpointKind: "manual",
		Branch:         "main",
		Source:         "api-test",
		CreatedAt:      now,
		Resumen:        "checkpoint api",
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/runtimes/tree", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(apiRuntimeTreeResponse{
			Runtimes: []*apiRuntimeTreeNode{{Runtime: runtime}},
		})
	})
	mux.HandleFunc("/api/runtime-handles", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(apiRuntimeHandlesResponse{Handles: []*db.RuntimeHandle{handle}})
	})
	mux.HandleFunc("/api/runtime-orders", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(apiRuntimeOrdersResponse{Orders: []*db.RuntimeOrder{order}})
	})
	mux.HandleFunc("/api/runtime-mailbox", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(apiRuntimeMailboxResponse{Mailbox: []*db.RuntimeMailboxMessage{msg}})
	})
	mux.HandleFunc("/api/runtime-checkpoints", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(apiRuntimeCheckpointsResponse{Checkpoints: []*db.RuntimeCheckpoint{cp}})
	})

	srv := newTestHTTPServerOrSkip(t, mux)
	defer srv.Close()

	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", srv.URL)()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "")()

	if err := runtimeDiagnosticoCmd.Flags().Set("agente", "Codex1"); err != nil {
		t.Fatalf("set agente: %v", err)
	}
	if err := runtimeDiagnosticoCmd.Flags().Set("proyecto", "orquestador"); err != nil {
		t.Fatalf("set proyecto: %v", err)
	}
	if err := runtimeDiagnosticoCmd.Flags().Set("limit", "2"); err != nil {
		t.Fatalf("set limit: %v", err)
	}
	t.Cleanup(func() {
		_ = runtimeDiagnosticoCmd.Flags().Set("agente", "")
		_ = runtimeDiagnosticoCmd.Flags().Set("proyecto", "")
		_ = runtimeDiagnosticoCmd.Flags().Set("limit", "5")
	})

	out := capturarStdout(t, func() {
		if err := runtimeDiagnosticoCmd.RunE(runtimeDiagnosticoCmd, nil); err != nil {
			t.Fatalf("run runtime diagnostico api: %v", err)
		}
	})

	for _, token := range []string{
		"Agente:     Codex1",
		"Fuente:     api",
		"openai",
		"sess-runtime-api",
		"checkpoint api",
		"Mailbox pendiente",
	} {
		if !strings.Contains(out, token) {
			t.Fatalf("salida diagnostico api sin %q:\n%s", token, out)
		}
	}
}

func TestRuntimeDiagnosticoRequiereServidor(t *testing.T) {
	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", "")()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "1")()
	defer cambiarEnv(t, "ORQUESTA_REQUIRE_SERVER", "")()

	resetCommandFlags(runtimeDiagnosticoCmd)
	_ = runtimeDiagnosticoCmd.Flags().Set("agente", "Codex1")
	err := runtimeDiagnosticoCmd.RunE(runtimeDiagnosticoCmd, nil)
	if err == nil {
		t.Fatalf("se esperaba error sin servidor")
	}
	if !strings.Contains(err.Error(), "requiere el servidor de Orquesta activo") {
		t.Fatalf("error inesperado: %v", err)
	}
}

func TestRuntimeDiagnosticoRecortaRuntimesYHandlesSegunLimit(t *testing.T) {
	data := &runtimeDiagnosticoData{
		Agente: "CodexX",
		Fuente: "api",
		Runtimes: []runtimeRow{
			{ID: 1, Agente: "CodexX", Estado: "cerrado"},
			{ID: 2, Agente: "CodexX", Estado: "fallido"},
			{ID: 3, Agente: "CodexX", Estado: "degradado"},
			{ID: 4, Agente: "CodexX", Estado: "esperando_io"},
		},
		Handles: []*db.RuntimeHandle{
			{ID: 10, Agente: "CodexX", Estado: "fallido"},
			{ID: 11, Agente: "CodexX", Estado: "activo"},
			{ID: 12, Agente: "CodexX", Estado: "fallido"},
		},
		Orders: []*db.RuntimeOrder{},
	}

	out := capturarStdout(t, func() {
		renderRuntimeDiagnostico(data, 2)
	})

	if strings.Contains(out, "1     cerrado") || strings.Contains(out, "10    CodexX       cli") {
		t.Fatalf("diagnostico no deberia mostrar terminales viejos cuando el limit ya lo ocupan vivos:\n%s", out)
	}
	for _, token := range []string{"3     degradado", "4     esperando_…", "11    CodexX"} {
		if !strings.Contains(out, token) {
			t.Fatalf("diagnostico recortado sin %q:\n%s", token, out)
		}
	}
}

func TestRuntimeDiagnosticoAlineaRuntimeConWorkerActivo(t *testing.T) {
	runtimes := []runtimeRow{
		{ID: 1198, Agente: "Gemini1", Estado: "esperando_io"},
		{ID: 1199, Agente: "Otro", Estado: "esperando_io"},
	}
	workers := []runtimeDiagnosticoWorkerRow{
		{HandleID: 1194, Agent: "Gemini1", State: "running", Alive: true},
	}

	got := runtimeDiagnosticoAlinearRuntimesConWorkers("Gemini1", runtimes, workers)
	if got[0].Estado != "activo" {
		t.Fatalf("runtime del agente deberia promocionarse a activo, got=%q", got[0].Estado)
	}
	if got[1].Estado != "esperando_io" {
		t.Fatalf("runtime ajeno no deberia cambiar, got=%q", got[1].Estado)
	}
}

func TestRuntimeDiagnosticoNoPromueveRuntimeTerminalConWorkerActivo(t *testing.T) {
	runtimes := []runtimeRow{
		{ID: 1200, Agente: "Gemini1", Estado: "fallido"},
	}
	workers := []runtimeDiagnosticoWorkerRow{
		{HandleID: 1194, Agent: "Gemini1", State: "running", Alive: true},
	}

	got := runtimeDiagnosticoAlinearRuntimesConWorkers("Gemini1", runtimes, workers)
	if got[0].Estado != "fallido" {
		t.Fatalf("runtime terminal no deberia maquillarse, got=%q", got[0].Estado)
	}
}

func TestRuntimeHandlesRelevantesPriorizaHandleActivoMasReciente(t *testing.T) {
	now := time.Now().UTC()
	oldSeen := now.Add(-2 * time.Minute)
	newSeen := now.Add(-10 * time.Second)

	handles := runtimeHandlesRelevantes([]*db.RuntimeHandle{
		{ID: 1151, Agente: "Codex1", Estado: "fallido", LastSeenAt: &oldSeen},
		{ID: 1152, Agente: "Codex1", Estado: "activo", LastSeenAt: &newSeen},
	}, 5)
	if len(handles) != 2 {
		t.Fatalf("handles inesperados: %+v", handles)
	}
	if handles[0] == nil || handles[0].ID != 1152 {
		t.Fatalf("el handle activo y mas reciente deberia ir primero: %+v", handles)
	}
	if handles[1] == nil || handles[1].ID != 1151 {
		t.Fatalf("el handle viejo/fallido deberia quedar detras: %+v", handles)
	}
}

func TestRuntimeDiagnosticoMuestraWorkerEstructurado(t *testing.T) {
	tmp := t.TempDir()
	now := time.Now().UTC()
	manifestPath := filepath.Join(tmp, "manifest.json")
	statusPath := filepath.Join(tmp, "status.json")
	heartbeatPath := filepath.Join(tmp, "heartbeat.json")

	writeJSON := func(path string, payload any) {
		t.Helper()
		data, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal %s: %v", path, err)
		}
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
	writeJSON(manifestPath, map[string]any{
		"agent":                 "Codex8",
		"driver":                "tmux_cli_session",
		"transport":             "tmux",
		"tmux_session":          "orq-codex8-093000",
		"tmux_pane_id":          "%3",
		"child_pid":             4242,
		"external_session_id":   "sess-rt-1",
		"mailbox_delivery_mode": "session_resume",
	})
	writeJSON(statusPath, map[string]any{
		"state":                 "running",
		"updated_at":            now.Format(time.RFC3339),
		"alive":                 true,
		"external_session_id":   "sess-rt-1",
		"mailbox_delivery_mode": "session_resume",
	})
	writeJSON(heartbeatPath, map[string]any{
		"alive":               true,
		"heartbeat_at":        now.Add(5 * time.Second).Format(time.RFC3339),
		"external_session_id": "sess-rt-1",
	})
	meta, err := json.Marshal(map[string]any{
		"worker_manifest_path":  manifestPath,
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
	})
	if err != nil {
		t.Fatalf("marshal meta: %v", err)
	}

	data := &runtimeDiagnosticoData{
		Agente: "Codex8",
		Fuente: "api",
		Handles: []*db.RuntimeHandle{
			{ID: 547, Agente: "Codex8", Estado: "activo", MetadataJSON: string(meta)},
		},
		Workers: runtimeWorkerRows([]*db.RuntimeHandle{
			{ID: 547, Agente: "Codex8", Estado: "activo", MetadataJSON: string(meta)},
		}, 5),
	}

	out := capturarStdout(t, func() {
		renderRuntimeDiagnostico(data, 5)
	})
	for _, token := range []string{"Worker estructurado", "running", "sess-rt-1", "session_resume", "tmux_cli_se…", "orq-codex8-093000/%3"} {
		if !strings.Contains(out, token) {
			t.Fatalf("salida sin %q:\n%s", token, out)
		}
	}
}

func TestRuntimeDiagnosticoSeparaMailboxCubiertaDePendienteReal(t *testing.T) {
	now := time.Now().UTC()
	mailboxPendiente := &db.RuntimeMailboxMessage{
		ID:         41,
		FromAgente: "server",
		ToAgente:   "Codex1",
		Kind:       "autonomia",
		Estado:     "pendiente",
		CreatedAt:  now,
	}
	mailboxReal := &db.RuntimeMailboxMessage{
		ID:         42,
		FromAgente: "server",
		ToAgente:   "Codex1",
		Kind:       "autonomia",
		Estado:     "pendiente",
		CreatedAt:  now,
	}
	orderCubierta := &db.RuntimeOrder{
		ID:            77,
		Agente:        "Codex1",
		Tipo:          "send_instruction",
		Estado:        "completada",
		CreatedAt:     now,
		PayloadJSON:   `{"to_agente":"Codex1","mailbox_id":41,"mailbox_kind":"autonomia"}`,
		ResultadoJSON: `{"mailbox_only":true,"delivery_state":"delivered"}`,
	}

	pendiente, cubierta := runtimeDiagnosticoSepararMailboxPendiente(
		[]*db.RuntimeMailboxMessage{mailboxPendiente, mailboxReal},
		[]*db.RuntimeOrder{orderCubierta},
	)
	if len(pendiente) != 1 || pendiente[0] == nil || pendiente[0].ID != 42 {
		t.Fatalf("mailbox pendiente inesperada: %+v", pendiente)
	}
	if len(cubierta) != 1 || cubierta[0] == nil || cubierta[0].ID != 41 {
		t.Fatalf("mailbox cubierta inesperada: %+v", cubierta)
	}

	data := &runtimeDiagnosticoData{
		Agente:           "Codex1",
		Fuente:           "api",
		Orders:           []*db.RuntimeOrder{orderCubierta},
		MailboxPendiente: pendiente,
		MailboxCubierta:  cubierta,
	}
	out := capturarStdout(t, func() {
		renderRuntimeDiagnostico(data, 5)
	})
	for _, token := range []string{"Mailbox:    1 pendiente(s) · 1 cubierta(s)", "Mailbox cubierta", "41"} {
		if !strings.Contains(out, token) {
			t.Fatalf("salida sin %q:\n%s", token, out)
		}
	}
}

func TestRuntimeDiagnosticoMarcaWorkerStaleSiElHandleYaEsTerminal(t *testing.T) {
	tmp := t.TempDir()
	manifestPath := filepath.Join(tmp, "manifest.json")
	statusPath := filepath.Join(tmp, "status.json")
	heartbeatPath := filepath.Join(tmp, "heartbeat.json")

	writeJSON := func(path string, payload any) {
		t.Helper()
		data, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal %s: %v", path, err)
		}
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
	writeJSON(manifestPath, map[string]any{"agent": "Codex7", "driver": "tmux_cli_session", "transport": "tmux"})
	writeJSON(statusPath, map[string]any{"state": "running", "alive": true})
	writeJSON(heartbeatPath, map[string]any{"alive": true})
	meta, err := json.Marshal(map[string]any{
		"worker_manifest_path":  manifestPath,
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
	})
	if err != nil {
		t.Fatalf("marshal meta: %v", err)
	}

	rows := runtimeWorkerRows([]*db.RuntimeHandle{
		{ID: 568, Agente: "Codex7", Estado: "fallido", MetadataJSON: string(meta)},
	}, 5)
	if len(rows) != 1 {
		t.Fatalf("rows inesperadas: %+v", rows)
	}
	if rows[0].Alive {
		t.Fatalf("worker stale no deberia salir vivo: %+v", rows[0])
	}
	if rows[0].State != "stale" {
		t.Fatalf("state stale esperado: %+v", rows[0])
	}
	if !strings.Contains(rows[0].ExitError, "handle=fallido") {
		t.Fatalf("exit error stale inesperado: %+v", rows[0])
	}
}

func TestRuntimeDiagnosticoMarcaWorkerStalePorHeartbeatAtrasado(t *testing.T) {
	tmp := t.TempDir()
	now := time.Now().UTC()
	manifestPath := filepath.Join(tmp, "manifest.json")
	statusPath := filepath.Join(tmp, "status.json")
	heartbeatPath := filepath.Join(tmp, "heartbeat.json")

	writeJSON := func(path string, payload any) {
		t.Helper()
		data, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal %s: %v", path, err)
		}
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
	writeJSON(manifestPath, map[string]any{
		"agent":        "Codex7",
		"driver":       "tmux_cli_session",
		"transport":    "tmux",
		"tmux_session": "orq-codex7-232657",
		"tmux_pane_id": "%0",
	})
	writeJSON(statusPath, map[string]any{
		"state":      "running",
		"updated_at": now.Add(-2 * time.Minute).Format(time.RFC3339),
		"alive":      true,
	})
	writeJSON(heartbeatPath, map[string]any{
		"alive":        true,
		"heartbeat_at": now.Add(-2 * time.Minute).Format(time.RFC3339),
	})
	meta, err := json.Marshal(map[string]any{
		"worker_manifest_path":  manifestPath,
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
	})
	if err != nil {
		t.Fatalf("marshal meta: %v", err)
	}

	rows := runtimeWorkerRows([]*db.RuntimeHandle{
		{ID: 569, Agente: "Codex7", Estado: "activo", MetadataJSON: string(meta)},
	}, 5)
	if len(rows) != 1 {
		t.Fatalf("rows inesperadas: %+v", rows)
	}
	if rows[0].Alive {
		t.Fatalf("worker con heartbeat atrasado no deberia salir vivo: %+v", rows[0])
	}
	if rows[0].State != "stale" {
		t.Fatalf("state stale esperado: %+v", rows[0])
	}
	if rows[0].SessionRef != "orq-codex7-232657/%0" {
		t.Fatalf("session ref inesperado: %+v", rows[0])
	}
	if rows[0].ExitError != "heartbeat_lag" {
		t.Fatalf("exit error inesperado: %+v", rows[0])
	}
}

func TestRuntimeDiagnosticoMarcaWorkerPausedSiHandlePausado(t *testing.T) {
	tmp := t.TempDir()
	now := time.Now().UTC()
	manifestPath := filepath.Join(tmp, "manifest.json")
	statusPath := filepath.Join(tmp, "status.json")
	heartbeatPath := filepath.Join(tmp, "heartbeat.json")

	writeJSON := func(path string, payload any) {
		t.Helper()
		data, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal %s: %v", path, err)
		}
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
	writeJSON(manifestPath, map[string]any{
		"agent":        "Codex7",
		"driver":       "tmux_cli_session",
		"transport":    "tmux",
		"tmux_session": "orq-codex7-232657",
		"tmux_pane_id": "%0",
	})
	writeJSON(statusPath, map[string]any{
		"state":      "running",
		"updated_at": now.Format(time.RFC3339),
		"alive":      true,
	})
	writeJSON(heartbeatPath, map[string]any{
		"alive":        true,
		"heartbeat_at": now.Format(time.RFC3339),
	})
	meta, err := json.Marshal(map[string]any{
		"worker_manifest_path":  manifestPath,
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
	})
	if err != nil {
		t.Fatalf("marshal meta: %v", err)
	}

	rows := runtimeWorkerRows([]*db.RuntimeHandle{
		{ID: 569, Agente: "Codex7", Estado: "pausado", MetadataJSON: string(meta)},
	}, 5)
	if len(rows) != 1 {
		t.Fatalf("rows inesperadas: %+v", rows)
	}
	if !rows[0].Alive {
		t.Fatalf("worker pausado no deberia salir muerto: %+v", rows[0])
	}
	if rows[0].State != "paused" {
		t.Fatalf("state paused esperado: %+v", rows[0])
	}
}

func TestRuntimeDiagnosticoIssuesDetectaHeartbeatLagYTMUX(t *testing.T) {
	orig := runtimeDiagnosticoTmuxHasSession
	runtimeDiagnosticoTmuxHasSession = func(sessionName string) bool {
		return strings.TrimSpace(sessionName) == "orq-codex7-live"
	}
	origNow := runtimeDiagnosticoNow
	runtimeDiagnosticoNow = func() time.Time {
		return time.Date(2026, 4, 6, 22, 48, 0, 0, time.UTC)
	}
	t.Cleanup(func() {
		runtimeDiagnosticoTmuxHasSession = orig
		runtimeDiagnosticoNow = origNow
	})

	rows := []runtimeDiagnosticoWorkerRow{
		{
			HandleID:    1,
			Agent:       "Codex7",
			Transport:   "tmux",
			RuntimeRef:  "orq-codex7-old/%0",
			State:       "stale",
			Alive:       false,
			ExitError:   "heartbeat_lag",
			TmuxSession: "orq-codex7-old",
			SessionRef:  "orq-codex7-old/%0",
			HeartbeatAt: nil,
			UpdatedAt:   nil,
			ChildPID:    123,
		},
		{
			HandleID:    2,
			Agent:       "Codex7",
			Transport:   "tmux",
			RuntimeRef:  "orq-codex7-live/%0",
			State:       "paused",
			Alive:       true,
			TmuxSession: "orq-codex7-live",
			SessionRef:  "orq-codex7-live/%0",
		},
		{
			HandleID:    3,
			Agent:       "Codex7",
			Transport:   "tmux",
			RuntimeRef:  "orq-codex7-missing/%0",
			State:       "running",
			Alive:       true,
			TmuxSession: "orq-codex7-missing",
			SessionRef:  "orq-codex7-missing/%0",
		},
	}

	issues := runtimeDiagnosticoIssues(rows, nil, nil, nil)
	if len(issues) != 2 {
		t.Fatalf("issues inesperadas: %+v", issues)
	}
	if issues[0].Code != "heartbeat_lag" {
		t.Fatalf("primer issue inesperado: %+v", issues)
	}
	if issues[1].Code != "tmux_session_missing" {
		t.Fatalf("segundo issue inesperado: %+v", issues)
	}
}

func TestRuntimeDiagnosticoIssuesNoMarcaOrphanSiOtraPaneVivaComparteSesion(t *testing.T) {
	orig := runtimeDiagnosticoTmuxHasSession
	runtimeDiagnosticoTmuxHasSession = func(sessionName string) bool {
		return strings.TrimSpace(sessionName) == "orq-codex8-shared"
	}
	origNow := runtimeDiagnosticoNow
	runtimeDiagnosticoNow = func() time.Time {
		return time.Date(2026, 4, 6, 22, 48, 0, 0, time.UTC)
	}
	t.Cleanup(func() {
		runtimeDiagnosticoTmuxHasSession = orig
		runtimeDiagnosticoNow = origNow
	})

	rows := []runtimeDiagnosticoWorkerRow{
		{
			HandleID:    10,
			Agent:       "Codex8",
			Transport:   "tmux",
			RuntimeRef:  "orq-codex8-shared/%6",
			State:       "stopped",
			Alive:       false,
			TmuxSession: "orq-codex8-shared",
			SessionRef:  "orq-codex8-shared/%6",
		},
		{
			HandleID:    11,
			Agent:       "Codex8",
			Transport:   "tmux",
			RuntimeRef:  "orq-codex8-shared/%7",
			State:       "running",
			Alive:       true,
			TmuxSession: "orq-codex8-shared",
			SessionRef:  "orq-codex8-shared/%7",
		},
	}

	issues := runtimeDiagnosticoIssues(rows, nil, nil, nil)
	if len(issues) != 0 {
		t.Fatalf("no deberia marcar sesion compartida viva como huérfana: %+v", issues)
	}
}

func TestRuntimeDiagnosticoIssuesDetectaSessionResumeBlocked(t *testing.T) {
	origNow := runtimeDiagnosticoNow
	runtimeDiagnosticoNow = func() time.Time {
		return time.Date(2026, 4, 6, 22, 48, 0, 0, time.UTC)
	}
	t.Cleanup(func() { runtimeDiagnosticoNow = origNow })

	rows := []runtimeDiagnosticoWorkerRow{
		{
			HandleID:            645,
			Agent:               "Codex2",
			State:               "running",
			Alive:               true,
			MailboxDeliveryMode: "session_resume",
			UpdatedAt:           diagTimePtr(time.Date(2026, 4, 6, 22, 34, 0, 0, time.UTC)),
		},
	}
	mailbox := []*db.RuntimeMailboxMessage{
		{
			ID:        65040,
			ToAgente:  "Codex2",
			Estado:    "pendiente",
			CreatedAt: time.Date(2026, 4, 6, 22, 34, 56, 0, time.UTC),
		},
	}

	issues := runtimeDiagnosticoIssues(rows, nil, mailbox, nil)
	if len(issues) != 1 {
		t.Fatalf("issues inesperadas: %+v", issues)
	}
	if issues[0].Code != "session_resume_blocked" {
		t.Fatalf("issue inesperado: %+v", issues)
	}
}

func TestRuntimeDiagnosticoIssuesNoMarcaProgressStaleSiHaySalidaReciente(t *testing.T) {
	now := time.Date(2026, 4, 13, 14, 30, 0, 0, time.UTC)
	runtimeDiagnosticoNow = func() time.Time { return now }
	t.Cleanup(func() {
		runtimeDiagnosticoNow = func() time.Time { return time.Now().UTC() }
	})

	lastProgress := now.Add(-2 * runtimeDiagnosticoWorkerProgressThreshold)
	lastOutput := now.Add(-5 * time.Minute)
	rows := []runtimeDiagnosticoWorkerRow{{
		HandleID:            852,
		Agent:               "Codex1",
		Alive:               true,
		Driver:              "tmux_cli_session",
		Transport:           "tmux",
		RuntimeRef:          "orq-codex1/%3",
		State:               "ready",
		LastProgressAt:      &lastProgress,
		LastOutputAt:        &lastOutput,
		MailboxDeliveryMode: "session_resume",
		TmuxSession:         "orq-codex1-140134",
	}}
	issues := runtimeDiagnosticoIssues(rows, nil, nil, nil)
	for _, issue := range issues {
		if issue.Code == "worker_progress_stale" {
			t.Fatalf("no deberia marcar worker_progress_stale con salida reciente: %+v", issues)
		}
	}
}

func TestRuntimeDiagnosticoIssuesDetectaResumeOrderPendingYRestartLoop(t *testing.T) {
	origNow := runtimeDiagnosticoNow
	runtimeDiagnosticoNow = func() time.Time {
		return time.Date(2026, 4, 6, 22, 48, 0, 0, time.UTC)
	}
	t.Cleanup(func() { runtimeDiagnosticoNow = origNow })

	rows := []runtimeDiagnosticoWorkerRow{
		{
			HandleID:            646,
			Agent:               "Codex3",
			State:               "running",
			Alive:               true,
			MailboxDeliveryMode: "session_resume",
			UpdatedAt:           diagTimePtr(time.Date(2026, 4, 6, 22, 20, 0, 0, time.UTC)),
		},
	}
	orders := []*db.RuntimeOrder{
		{
			ID:        84350,
			Agente:    "Codex3",
			Tipo:      "resume",
			Estado:    "pendiente",
			CreatedAt: time.Date(2026, 4, 6, 22, 30, 0, 0, time.UTC),
		},
	}
	checkpoints := []*db.RuntimeCheckpoint{
		{
			ID:             35149,
			Agente:         "Codex3",
			CheckpointKind: "stop",
			CreatedAt:      time.Date(2026, 4, 6, 18, 22, 56, 0, time.UTC),
			Resumen:        "worker_atascado",
			Source:         "runtime_order:84350",
		},
		{
			ID:             35154,
			Agente:         "Codex3",
			CheckpointKind: "stop",
			CreatedAt:      time.Date(2026, 4, 6, 22, 34, 56, 0, time.UTC),
			Resumen:        "worker_atascado",
			Source:         "runtime_order:84379",
		},
	}

	issues := runtimeDiagnosticoIssues(rows, orders, nil, checkpoints)
	if len(issues) != 2 {
		t.Fatalf("issues inesperadas: %+v", issues)
	}
	if issues[0].Code != "resume_order_pending" {
		t.Fatalf("primer issue inesperado: %+v", issues)
	}
	if issues[1].Code != "restart_loop" {
		t.Fatalf("segundo issue inesperado: %+v", issues)
	}
}

func TestRuntimeDiagnosticoOcultaRestartLoopSiWorkerActualEstaSano(t *testing.T) {
	now := time.Date(2026, 4, 6, 22, 40, 0, 0, time.UTC)
	runtimeDiagnosticoNow = func() time.Time { return now }
	defer func() { runtimeDiagnosticoNow = func() time.Time { return time.Now().UTC() } }()

	rows := []runtimeDiagnosticoWorkerRow{
		{
			Agent:       "Gemini1",
			State:       "ready",
			Alive:       true,
			HeartbeatAt: diagTimePtr(now.Add(-10 * time.Second)),
			UpdatedAt:   diagTimePtr(now.Add(-10 * time.Second)),
		},
	}
	checkpoints := []*db.RuntimeCheckpoint{
		{
			ID:             35149,
			Agente:         "Gemini1",
			CheckpointKind: "stop",
			CreatedAt:      now.Add(-4 * time.Hour),
			Resumen:        "worker_atascado",
			Source:         "runtime_order:84350",
		},
		{
			ID:             35154,
			Agente:         "Gemini1",
			CheckpointKind: "stop",
			CreatedAt:      now.Add(-30 * time.Minute),
			Resumen:        "worker_atascado",
			Source:         "runtime_order:84379",
		},
	}

	issues := runtimeDiagnosticoIssues(rows, nil, nil, checkpoints)
	for _, issue := range issues {
		if issue.Code == "restart_loop" {
			t.Fatalf("no deberia alertar restart_loop si el worker actual esta sano: %+v", issues)
		}
	}
}

func TestRuntimeDiagnosticoOcultaRestartLoopSiWorkerActualEstaBloqueadoPorCuota(t *testing.T) {
	now := time.Date(2026, 4, 6, 22, 40, 0, 0, time.UTC)
	runtimeDiagnosticoNow = func() time.Time { return now }
	defer func() { runtimeDiagnosticoNow = func() time.Time { return time.Now().UTC() } }()

	rows := []runtimeDiagnosticoWorkerRow{
		{
			Agent:       "Gemini1",
			State:       "blocked_quota",
			Alive:       true,
			HeartbeatAt: diagTimePtr(now.Add(-10 * time.Second)),
			UpdatedAt:   diagTimePtr(now.Add(-10 * time.Second)),
		},
	}
	checkpoints := []*db.RuntimeCheckpoint{
		{
			ID:             35149,
			Agente:         "Gemini1",
			CheckpointKind: "stop",
			CreatedAt:      now.Add(-4 * time.Hour),
			Resumen:        "worker_atascado",
			Source:         "runtime_order:84350",
		},
		{
			ID:             35154,
			Agente:         "Gemini1",
			CheckpointKind: "stop",
			CreatedAt:      now.Add(-30 * time.Minute),
			Resumen:        "worker_atascado",
			Source:         "runtime_order:84379",
		},
	}

	issues := runtimeDiagnosticoIssues(rows, nil, nil, checkpoints)
	for _, issue := range issues {
		if issue.Code == "restart_loop" {
			t.Fatalf("no deberia alertar restart_loop si el worker actual esta bloqueado por cuota: %+v", issues)
		}
	}
}

func TestRuntimeDiagnosticoOcultaRestartLoopSiElCheckpointRecienteEsPausaPorCuota(t *testing.T) {
	now := time.Date(2026, 4, 6, 22, 40, 0, 0, time.UTC)
	runtimeDiagnosticoNow = func() time.Time { return now }
	defer func() { runtimeDiagnosticoNow = func() time.Time { return time.Now().UTC() } }()

	checkpoints := []*db.RuntimeCheckpoint{
		{
			ID:             35149,
			Agente:         "Gemini1",
			CheckpointKind: "stop",
			CreatedAt:      now.Add(-4 * time.Hour),
			Resumen:        "worker_atascado",
			Source:         "runtime_order:84350",
		},
		{
			ID:             35154,
			Agente:         "Gemini1",
			CheckpointKind: "stop",
			CreatedAt:      now.Add(-30 * time.Minute),
			Resumen:        "worker_atascado",
			Source:         "runtime_order:84379",
		},
		{
			ID:             35155,
			Agente:         "Gemini1",
			CheckpointKind: "pause",
			CreatedAt:      now.Add(-10 * time.Minute),
			Resumen:        "worker bloqueado por cuota",
			Source:         "runtime_order:84380",
		},
	}

	issues := runtimeDiagnosticoIssues(nil, nil, nil, checkpoints)
	for _, issue := range issues {
		if issue.Code == "restart_loop" {
			t.Fatalf("no deberia alertar restart_loop si el ultimo estado estable es pausa por cuota: %+v", issues)
		}
	}
}

func diagTimePtr(ts time.Time) *time.Time {
	return &ts
}
