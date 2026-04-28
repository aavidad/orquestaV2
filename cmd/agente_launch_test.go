package cmd

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"orquesta/db"
	"orquesta/runtimeagente"
)

func TestResolveAgenteLaunchMode(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().Bool("tabs", false, "")
	cmd.Flags().Bool("ventanas", false, "")

	mode, err := resolveAgenteLaunchMode(cmd)
	if err != nil {
		t.Fatalf("resolveAgenteLaunchMode default: %v", err)
	}
	if mode != "windows" {
		t.Fatalf("mode default = %q, want windows", mode)
	}
}

func TestAgenteLauncherBootstrapInputSoloParaPostStart(t *testing.T) {
	plan := &runtimeagente.LaunchPlan{
		BootstrapPrompt:  "Bootstrap de Orquesta",
		LaunchPromptMode: "post_start",
	}
	if got := agenteLauncherBootstrapInput(plan); got != "Bootstrap de Orquesta\n\n" {
		t.Fatalf("bootstrap input inesperado: %q", got)
	}
	plan.LaunchPromptMode = "embedded"
	if got := agenteLauncherBootstrapInput(plan); got != "" {
		t.Fatalf("no deberia inyectar bootstrap fuera de post_start: %q", got)
	}
}

func TestAgentePlanDebeUsarArranqueCanonicoParaOllamaCLI(t *testing.T) {
	plan := &runtimeagente.LaunchPlan{
		Comando: "ollama",
		Args:    []string{"run", "qwen2.5-coder:7b"},
	}
	if !agentePlanDebeUsarArranqueCanonico(plan) {
		t.Fatal("deberia usar arranque canónico para ollama cli")
	}
}

func TestAgenteArranqueCanonicoPuedeReutilizarRuntime(t *testing.T) {
	if !agenteArranqueCanonicoPuedeReutilizarRuntime(assertErr("el agente Gemma1 ya tiene un runtime handle activo")) {
		t.Fatalf("deberia reutilizar runtime cuando el handle ya esta activo")
	}
	if agenteArranqueCanonicoPuedeReutilizarRuntime(assertErr("agente no encontrado")) {
		t.Fatalf("no deberia reutilizar runtime para errores ajenos")
	}
}

func TestEsperarArranqueCanonicoAgenteDespiertaYDetectaHandleActivo(t *testing.T) {
	now := time.Date(2026, 4, 22, 10, 0, 0, 0, time.UTC)
	runtime := &db.RuntimeInstance{
		ID:              168,
		Agente:          "QwenCoder1",
		ProyectoSlug:    "orquestador",
		Provider:        "ollama",
		Connector:       "ollama-cli",
		LogicalState:    "activo",
		LastHeartbeatAt: &now,
	}
	handle := &db.RuntimeHandle{
		ID:           268,
		Agente:       "QwenCoder1",
		Transporte:   "tmux",
		HandleKind:   "session",
		HandleRef:    "orq-qwen/%4",
		Estado:       "activo",
		LastSeenAt:   &now,
		MetadataJSON: `{"driver":"tmux_cli_session","tmux_session":"orq-qwencoder1-123","can_send_input":true}`,
	}
	var wakeCalls, processCalls int
	mux := http.NewServeMux()
	mux.HandleFunc("/api/runtime/wake", func(w http.ResponseWriter, r *http.Request) {
		wakeCalls++
		_ = json.NewEncoder(w).Encode(apiRuntimeWakeResponse{Orders: true})
	})
	mux.HandleFunc("/api/runtime/process-orders", func(w http.ResponseWriter, r *http.Request) {
		processCalls++
		_ = json.NewEncoder(w).Encode(apiRuntimeProcessOrdersResponse{OK: true, Count: 1})
	})
	mux.HandleFunc("/api/runtimes/tree", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(apiRuntimeTreeResponse{
			Runtimes: []*apiRuntimeTreeNode{{Runtime: runtime}},
		})
	})
	mux.HandleFunc("/api/runtime-handles", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(apiRuntimeHandlesResponse{Handles: []*db.RuntimeHandle{handle}})
	})
	mux.HandleFunc("/api/runtime-orders", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(apiRuntimeOrdersResponse{Orders: []*db.RuntimeOrder{}})
	})
	mux.HandleFunc("/api/runtime-mailbox", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(apiRuntimeMailboxResponse{Mailbox: []*db.RuntimeMailboxMessage{}})
	})
	mux.HandleFunc("/api/runtime-checkpoints", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(apiRuntimeCheckpointsResponse{Checkpoints: []*db.RuntimeCheckpoint{}})
	})

	srv := newTestHTTPServerOrSkip(t, mux)
	defer srv.Close()
	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", srv.URL)()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "")()

	data, err := esperarArranqueCanonicoAgente("QwenCoder1", "orquestador", 1500*time.Millisecond)
	if err != nil {
		t.Fatalf("esperarArranqueCanonicoAgente: %v", err)
	}
	if data == nil || len(data.Handles) != 1 || data.Handles[0].ID != 268 {
		t.Fatalf("diagnóstico inesperado: %+v", data)
	}
	if wakeCalls == 0 || processCalls == 0 {
		t.Fatalf("deberia llamar wake y process-orders, wake=%d process=%d", wakeCalls, processCalls)
	}
}

func TestEsperarArranqueCanonicoAgenteIgnoraRuntimeDeProyectoAjeno(t *testing.T) {
	now := time.Date(2026, 4, 22, 10, 0, 0, 0, time.UTC)
	runtime := &db.RuntimeInstance{
		ID:              281,
		Agente:          "Gemma1",
		ProyectoSlug:    "api",
		Provider:        "ollama-cli",
		Connector:       "ollama-cli",
		LogicalState:    "esperando_io",
		LastHeartbeatAt: &now,
	}
	handle := &db.RuntimeHandle{
		ID:           381,
		Agente:       "Gemma1",
		ProyectoID:   int64Ptr(6),
		Transporte:   "tmux",
		HandleKind:   "session",
		HandleRef:    "orq-gemma1/%84",
		Estado:       "activo",
		LastSeenAt:   &now,
		MetadataJSON: `{"driver":"tmux_cli_session","tmux_session":"orq-gemma1-140919","can_send_input":true}`,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/runtime/wake", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(apiRuntimeWakeResponse{Orders: true})
	})
	mux.HandleFunc("/api/runtime/process-orders", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(apiRuntimeProcessOrdersResponse{OK: true, Count: 1})
	})
	mux.HandleFunc("/api/runtimes/tree", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(apiRuntimeTreeResponse{
			Runtimes: []*apiRuntimeTreeNode{{Runtime: runtime}},
		})
	})
	mux.HandleFunc("/api/runtime-handles", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(apiRuntimeHandlesResponse{Handles: []*db.RuntimeHandle{handle}})
	})
	mux.HandleFunc("/api/runtime-orders", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(apiRuntimeOrdersResponse{Orders: []*db.RuntimeOrder{}})
	})
	mux.HandleFunc("/api/runtime-mailbox", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(apiRuntimeMailboxResponse{Mailbox: []*db.RuntimeMailboxMessage{}})
	})
	mux.HandleFunc("/api/runtime-checkpoints", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(apiRuntimeCheckpointsResponse{Checkpoints: []*db.RuntimeCheckpoint{}})
	})

	srv := newTestHTTPServerOrSkip(t, mux)
	defer srv.Close()
	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", srv.URL)()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "")()

	data, err := esperarArranqueCanonicoAgente("Gemma1", "orquestador", 1500*time.Millisecond)
	if err == nil {
		t.Fatalf("deberia ignorar runtime vivo de proyecto ajeno: %+v", data)
	}
	if data == nil || len(data.Runtimes) != 1 || data.Runtimes[0].Proyecto != "api" {
		t.Fatalf("diagnóstico inesperado: %+v", data)
	}
}

func assertErr(text string) error {
	return &testErr{text: text}
}

type testErr struct{ text string }

func (e *testErr) Error() string { return e.text }

func TestResolveAgenteLaunchModeConflicto(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().Bool("tabs", false, "")
	cmd.Flags().Bool("ventanas", false, "")
	cmd.Flags().Set("tabs", "true")
	cmd.Flags().Set("ventanas", "true")

	if _, err := resolveAgenteLaunchMode(cmd); err == nil {
		t.Fatalf("se esperaba error por conflicto tabs/ventanas")
	}
}

func TestBuildAgenteLanzarPlanSpecUsaScriptsDir(t *testing.T) {
	tmp := t.TempDir()
	scriptsDir := filepath.Join(tmp, "scripts")
	if err := os.MkdirAll(scriptsDir, 0o755); err != nil {
		t.Fatalf("mkdir scripts dir: %v", err)
	}
	scriptPath := filepath.Join(scriptsDir, "launch_agentes.sh")
	if err := os.WriteFile(scriptPath, []byte("#!/usr/bin/env bash\n"), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}
	planPath := filepath.Join(tmp, "plan.plan")
	if err := os.WriteFile(planPath, []byte("Codex1|programador|/tmp/proj|Titulo|alta|mod|nota|codex-cli|codex\n"), 0o644); err != nil {
		t.Fatalf("write plan: %v", err)
	}

	defer cambiarEnv(t, "ORQUESTA_SCRIPTS_DIR", scriptsDir)()
	spec, err := buildAgenteLanzarPlanSpec(planPath, "tmux", "", "", "sesion-demo", false, "tabs", true)
	if err != nil {
		t.Fatalf("buildAgenteLanzarPlanSpec: %v", err)
	}
	if spec.ScriptPath != scriptPath {
		t.Fatalf("scriptPath = %q, want %q", spec.ScriptPath, scriptPath)
	}
	if len(spec.Args) != 3 || spec.Args[0] != "--ejecutar" || spec.Args[1] != "--tabs" {
		t.Fatalf("args inesperados: %+v", spec.Args)
	}
	if spec.Env["ORQUESTA_TERMINAL_BACKEND"] != "tmux" {
		t.Fatalf("backend env inesperado: %+v", spec.Env)
	}
	if spec.Env["ORQUESTA_TMUX_SESSION_NAME"] != "sesion-demo" {
		t.Fatalf("tmux session env inesperado: %+v", spec.Env)
	}
	if spec.Env["ORQUESTA_TMUX_ATTACH"] != "0" {
		t.Fatalf("tmux attach env inesperado: %+v", spec.Env)
	}
}

func TestMergeAgenteLaunchEnvSobrescribe(t *testing.T) {
	out := mergeAgenteLaunchEnv([]string{"A=1", "B=2"}, map[string]string{"B": "3", "C": "4"})
	got := map[string]string{}
	for _, item := range out {
		key, value, ok := strings.Cut(item, "=")
		if !ok {
			t.Fatalf("env sin separador: %q", item)
		}
		got[key] = value
	}
	if got["A"] != "1" || got["B"] != "3" || got["C"] != "4" {
		t.Fatalf("env merge inesperado: %+v", got)
	}
}

func TestCommandNeedsDBAgenteLanzarPlanNo(t *testing.T) {
	if commandNeedsDB([]string{"agente", "lanzar-plan", "plan.plan"}) {
		t.Fatalf("agente lanzar-plan no deberia requerir DB")
	}
}

func TestAgenteLanzarPlanJSONSmoke(t *testing.T) {
	tmp := t.TempDir()
	scriptsDir := filepath.Join(tmp, "scripts")
	if err := os.MkdirAll(scriptsDir, 0o755); err != nil {
		t.Fatalf("mkdir scripts dir: %v", err)
	}
	scriptPath := filepath.Join(scriptsDir, "launch_agentes.sh")
	if err := os.WriteFile(scriptPath, []byte("#!/usr/bin/env bash\n"), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}
	planPath := filepath.Join(tmp, "plan.plan")
	if err := os.WriteFile(planPath, []byte("Codex1|programador|/tmp/proj|Titulo|alta|mod|nota|codex-cli|codex\n"), 0o644); err != nil {
		t.Fatalf("write plan: %v", err)
	}

	defer cambiarEnv(t, "ORQUESTA_SCRIPTS_DIR", scriptsDir)()

	out := capturarStdout(t, func() {
		if err := executeLocalArgs([]string{"agente", "lanzar-plan", planPath, "--backend", "tmux", "--tabs", "--json"}, os.Stdout, os.Stderr); err != nil {
			t.Fatalf("executeLocalArgs agente lanzar-plan: %v", err)
		}
	})

	var spec agenteLanzarPlanOutput
	if err := json.Unmarshal([]byte(out), &spec); err != nil {
		t.Fatalf("parse json: %v\n%s", err, out)
	}
	if spec.ScriptPath != scriptPath {
		t.Fatalf("scriptPath = %q, want %q", spec.ScriptPath, scriptPath)
	}
	if spec.Backend != "tmux" || spec.LaunchMode != "tabs" {
		t.Fatalf("spec inesperado: %+v", spec)
	}
}

func TestParseAgenteLanzarPlanEntriesUsaConectorOllamaPorDefecto(t *testing.T) {
	tmp := t.TempDir()
	planPath := filepath.Join(tmp, "plan.plan")
	if err := os.WriteFile(planPath, []byte("Ollama1|programador|orquestador|Titulo|alta|mod|nota\n"), 0o644); err != nil {
		t.Fatalf("write plan: %v", err)
	}

	entries, err := parseAgenteLanzarPlanEntries(planPath)
	if err != nil {
		t.Fatalf("parseAgenteLanzarPlanEntries: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("entries inesperadas: %+v", entries)
	}
	if entries[0].Conector != "ollama-cli" {
		t.Fatalf("conector por defecto inesperado: %+v", entries[0])
	}
}

func TestParseAgenteLanzarPlanEntries(t *testing.T) {
	tmp := t.TempDir()
	planPath := filepath.Join(tmp, "plan.plan")
	plan := strings.Join([]string{
		"# comentario",
		"",
		"Codex1|programador|orquestador|Titulo|alta|mod|nota|codex-cli|codex --x",
		"Codex2|programador|orquestador|Titulo 2|media|mod|nota||",
	}, "\n")
	if err := os.WriteFile(planPath, []byte(plan), 0o644); err != nil {
		t.Fatalf("write plan: %v", err)
	}

	entries, err := parseAgenteLanzarPlanEntries(planPath)
	if err != nil {
		t.Fatalf("parseAgenteLanzarPlanEntries: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("entries len = %d, want 2", len(entries))
	}
	if entries[0].Agente != "Codex1" || entries[0].Proyecto != "orquestador" || entries[0].Conector != "codex-cli" {
		t.Fatalf("entry 0 inesperada: %+v", entries[0])
	}
	if entries[1].Conector != "codex-cli" {
		t.Fatalf("connector default inesperado: %+v", entries[1])
	}
}

func TestAgenteLanzarPlanServerFirstPreviewNoRequiereScripts(t *testing.T) {
	tmp := t.TempDir()
	planPath := filepath.Join(tmp, "plan.plan")
	if err := os.WriteFile(planPath, []byte("Codex1|programador|orquestador|Titulo|alta|mod|nota|codex-cli|codex\n"), 0o644); err != nil {
		t.Fatalf("write plan: %v", err)
	}

	out := capturarStdout(t, func() {
		if err := executeLocalArgs([]string{"agente", "lanzar-plan", planPath, "--server-first"}, os.Stdout, os.Stderr); err != nil {
			t.Fatalf("executeLocalArgs server-first preview: %v", err)
		}
	})
	if !strings.Contains(out, "Preview server-first") || !strings.Contains(out, "agente=Codex1") {
		t.Fatalf("salida preview inesperada: %s", out)
	}
}

func TestEnqueueAgenteLanzarPlanServerFirst(t *testing.T) {
	prev := createRuntimeOrderForLaunch
	defer func() { createRuntimeOrderForLaunch = prev }()
	type request struct {
		Agente   string
		Tipo     string
		Proyecto string
		Payload  string
	}
	var seen []request
	createRuntimeOrderForLaunch = func(agente, tipo, proyecto, payload string) (int64, bool, error) {
		seen = append(seen, request{
			Agente:   agente,
			Tipo:     tipo,
			Proyecto: proyecto,
			Payload:  payload,
		})
		return int64(len(seen)), true, nil
	}

	entries := []agenteLanzarPlanEntry{
		{Agente: "Codex1", Proyecto: "orquestador", Conector: "codex-cli"},
		{Agente: "Codex2", Proyecto: "orquestador", Conector: "codex-cli"},
	}
	ids, err := enqueueAgenteLanzarPlanServerFirst(entries)
	if err != nil {
		t.Fatalf("enqueueAgenteLanzarPlanServerFirst: %v", err)
	}
	if len(ids) != 2 || ids[0] != 1 || ids[1] != 2 {
		t.Fatalf("ids inesperados: %+v", ids)
	}
	if len(seen) != 2 {
		t.Fatalf("requests vistas inesperadas: %d", len(seen))
	}
	if got := seen[0].Agente; got != "Codex1" {
		t.Fatalf("agente request 0 inesperado: %v", got)
	}
	if got := seen[0].Tipo; got != "start" {
		t.Fatalf("tipo request 0 inesperado: %v", got)
	}
	if got := seen[0].Proyecto; got != "orquestador" {
		t.Fatalf("proyecto request 0 inesperado: %v", got)
	}
	if !strings.Contains(seen[0].Payload, `"conector":"codex-cli"`) {
		t.Fatalf("payload request 0 inesperado: %s", seen[0].Payload)
	}
}
