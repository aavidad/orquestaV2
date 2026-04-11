/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package runtimeagente

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPrepareCLISinResumeNativoDevuelvePromptContinuidad(t *testing.T) {
	plan, err := DefaultRegistry().Prepare(LaunchRequest{
		Agente:       "codex1",
		ProyectoSlug: "autofirmav2",
		ProyectoRuta: "/tmp/autofirmav2",
		Conector: ConnectorConfig{
			Slug:         "codex-cli",
			Transporte:   "cli",
			Comando:      "codex",
			ArgsJSON:     `["chat"]`,
			EnvJSON:      `{"A":"1"}`,
			MetadataJSON: `{"familia":"openai","reanudable":true}`,
		},
		Resume: ResumeContext{
			ExternalSessionID:  "sess-1",
			ResumenContinuidad: "seguir con la firma",
			Branch:             "feature/x",
		},
	})
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if plan.Modo != "resume" {
		t.Fatalf("modo inesperado: %s", plan.Modo)
	}
	if plan.NativeResume {
		t.Fatalf("no debía activar resume nativo")
	}
	if !strings.Contains(plan.ContinuityPrompt, "seguir con la firma") {
		t.Fatalf("prompt de continuidad inesperado: %s", plan.ContinuityPrompt)
	}
}

func TestPrepareCLISinResumeNativoCompactaResumePayloadEnPrompt(t *testing.T) {
	plan, err := DefaultRegistry().Prepare(LaunchRequest{
		Agente:       "Codex2",
		ProyectoSlug: "orquestador",
		ProyectoRuta: "/tmp/orquestador",
		Conector: ConnectorConfig{
			Slug:       "codex-cli",
			Transporte: "cli",
			Comando:    "codex",
		},
		Resume: ResumeContext{
			ResumenContinuidad: "seguir con control plane",
			ResumePayloadJSON: `{
				"bootstrap_prompt":"Bootstrap de Orquesta para Codex2.",
				"checkpoint":{"id":9,"kind":"stop"},
				"mailbox":[{"id":1},{"id":2}],
				"project_context":{"proyecto":{"slug":"orquestador"}},
				"governance_catalog":{"hash":"abc123"},
				"persist":"ok"
			}`,
		},
	})
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if strings.Contains(plan.ContinuityPrompt, "Bootstrap de Orquesta para Codex2.") || strings.Contains(plan.ContinuityPrompt, `"checkpoint"`) {
		t.Fatalf("el prompt de continuidad no deberia incrustar el JSON bruto: %s", plan.ContinuityPrompt)
	}
	for _, token := range []string{"checkpoint#9(stop)", "mailbox=2", "project_context", "governance_catalog", "persist"} {
		if !strings.Contains(plan.ContinuityPrompt, token) {
			t.Fatalf("falta resumen de resume payload %s en: %s", token, plan.ContinuityPrompt)
		}
	}
}

func TestPrepareCLICodexCompactaContinuityPromptParaRuntimeNoInteractivo(t *testing.T) {
	plan, err := DefaultRegistry().Prepare(LaunchRequest{
		Agente:       "Codex7",
		ProyectoSlug: "orquestador",
		ProyectoRuta: "/tmp/orquestador",
		Conector: ConnectorConfig{
			Slug:       "codex-cli",
			Transporte: "cli",
			Comando:    "codex",
		},
		Resume: ResumeContext{
			ResumenContinuidad: "texto largo que no deberia pasar completo al prompt de lanzamiento",
			Branch:             "orq-orquestador-codex7",
			ResumePayloadJSON:  `{"checkpoint":{"id":33155,"kind":"stop"},"mailbox":[{"id":1}],"project_context":{},"governance_catalog":{}}`,
		},
	})
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if strings.Contains(plan.ContinuityPrompt, "texto largo que no deberia pasar completo") {
		t.Fatalf("continuity prompt demasiado verboso: %s", plan.ContinuityPrompt)
	}
	for _, token := range []string{"Retoma el trabajo actual desde Orquesta.", "Proyecto: orquestador.", "Rama: orq-orquestador-codex7.", "checkpoint#33155", "mailbox=1"} {
		if !strings.Contains(plan.ContinuityPrompt, token) {
			t.Fatalf("falta %q en continuity prompt: %s", token, plan.ContinuityPrompt)
		}
	}
	for _, token := range []string{
		"Ignora cualquier conversación vieja que no coincida con la tarea activa o el mailbox actual.",
		"No abras frentes nuevos ni reescribas código fuera del alcance inmediato.",
	} {
		if !strings.Contains(plan.ContinuityPrompt, token) {
			t.Fatalf("falta %q en continuity prompt endurecido: %s", token, plan.ContinuityPrompt)
		}
	}
}

func TestPrepareCLIOllamaNoReanudaContextoNoReanudable(t *testing.T) {
	plan, err := DefaultRegistry().Prepare(LaunchRequest{
		Agente:       "OllamaLocal",
		ProyectoSlug: "orquestador",
		ProyectoRuta: "/tmp/orquestador",
		Conector: ConnectorConfig{
			Slug:         "ollama-cli",
			Transporte:   "cli",
			Comando:      "ollama",
			MetadataJSON: `{"reanudable":false}`,
		},
		Resume: ResumeContext{
			ResumenContinuidad: "texto largo que no deberia pasar completo al prompt de continuidad",
			Branch:             "orq-orquestador-ollama",
			ResumePayloadJSON:  `{"checkpoint":{"id":33155,"kind":"stop"},"mailbox":[{"id":1}],"project_context":{},"governance_catalog":{}}`,
			CWD:                "/tmp/orquestador/.orquesta-worktrees/orq-ollama",
		},
	})
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if plan.Modo != "launch" {
		t.Fatalf("ollama no deberia entrar en resume: %+v", plan)
	}
	if plan.NativeResume {
		t.Fatalf("ollama no deberia activar resume nativo: %+v", plan)
	}
	if strings.TrimSpace(plan.ContinuityPrompt) != "" {
		t.Fatalf("ollama no deberia heredar continuity prompt: %q", plan.ContinuityPrompt)
	}
	if plan.WorkingDir != "/tmp/orquestador/.orquesta-worktrees/orq-ollama" {
		t.Fatalf("deberia conservar cwd de trabajo aunque no reanude: %s", plan.WorkingDir)
	}
}

func TestSanitizarResumeParaConectorNoBorraPayloadPersistido(t *testing.T) {
	resume := SanitizarResumeParaConector(ConnectorConfig{
		Slug:         "ollama-cli",
		Transporte:   "cli",
		Comando:      "ollama",
		MetadataJSON: `{"reanudable":false}`,
	}, ResumeContext{
		ExternalSessionID:  "sess-1",
		ResumenContinuidad: "seguir luego",
		ResumePayloadJSON:  `{"perfil_ejecucion":{"modelo":"gemma4:26b"}}`,
	})
	if resume.ExternalSessionID != "" || resume.ResumenContinuidad != "" {
		t.Fatalf("el conector no reanudable debe limpiar continuidad de proveedor: %+v", resume)
	}
	if !strings.Contains(resume.ResumePayloadJSON, "gemma4:26b") {
		t.Fatalf("el payload persistido no deberia perderse: %s", resume.ResumePayloadJSON)
	}
}

func TestPrepareRemotoSoloUsaResumeConSesionExternaReal(t *testing.T) {
	plan, err := DefaultRegistry().Prepare(LaunchRequest{
		Agente:       "CodexRemote",
		ProyectoSlug: "orquestador",
		ProyectoRuta: "/tmp/orquestador",
		Conector: ConnectorConfig{
			Slug:         "codex-remote",
			Transporte:   "api",
			Comando:      "http://127.0.0.1:8080",
			MetadataJSON: `{"launch_path":"/launch","resume_path":"/resume"}`,
		},
		Resume: ResumeContext{
			ResumePayloadJSON:  `{"checkpoint":{"id":7}}`,
			ResumenContinuidad: "seguir desde checkpoint",
		},
	})
	if err != nil {
		t.Fatalf("prepare remoto: %v", err)
	}
	if plan.Modo != "launch" {
		t.Fatalf("el remoto no debe usar /resume sin external_session_id: %+v", plan)
	}
	if strings.TrimSpace(plan.ContinuityPrompt) != "" {
		t.Fatalf("el remoto no deberia fabricar continuity prompt sin sesión externa: %q", plan.ContinuityPrompt)
	}
}

func TestPrepareCLIConHintsActivaResumeNativo(t *testing.T) {
	plan, err := DefaultRegistry().Prepare(LaunchRequest{
		Agente:       "claude1",
		ProyectoSlug: "conta",
		ProyectoRuta: "/tmp/conta",
		Conector: ConnectorConfig{
			Slug:         "claude-code",
			Transporte:   "cli",
			Comando:      "claude",
			MetadataJSON: `{"resume_subcommand":"resume","cwd_flag":"--cwd","session_id_flag":"--session"}`,
		},
		Resume: ResumeContext{
			ExternalSessionID: "abc-123",
			CWD:               "/tmp/conta",
		},
	})
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if !plan.NativeResume {
		t.Fatalf("debía activar resume nativo")
	}
	rendered := RenderCommand(plan)
	if !strings.Contains(rendered, "'resume'") || !strings.Contains(rendered, "'--session'") || !strings.Contains(rendered, "'abc-123'") {
		t.Fatalf("comando inesperado: %s", rendered)
	}
}

func TestPrepareCLINormalizaReasoningLegacyDeCodexCLI(t *testing.T) {
	plan, err := DefaultRegistry().Prepare(LaunchRequest{
		Agente:       "Codex1",
		ProyectoSlug: "orquestador",
		ProyectoRuta: "/tmp/orquestador",
		Modelo:       "gpt-5.4",
		Razonamiento: "xhigh",
		PerfilTarea:  "orquestacion",
		Conector: ConnectorConfig{
			Slug:         "codex-cli",
			Transporte:   "cli",
			Comando:      "codex",
			MetadataJSON: `{"model_flag":"--model","reasoning_flag":"--reasoning-effort"}`,
		},
	})
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if plan.Modelo != "gpt-5.4" || plan.Razonamiento != "xhigh" || plan.PerfilTarea != "orquestacion" {
		t.Fatalf("perfil inesperado: %+v", plan)
	}
	rendered := RenderCommand(plan)
	if !strings.Contains(rendered, "'--model'") || !strings.Contains(rendered, "'gpt-5.4'") {
		t.Fatalf("comando sin modelo esperado: %s", rendered)
	}
	if strings.Contains(rendered, "'--reasoning-effort'") {
		t.Fatalf("comando no deberia usar flag legacy de razonamiento: %s", rendered)
	}
	if !strings.Contains(rendered, "'-c'") || !strings.Contains(rendered, `'model_reasoning_effort="xhigh"'`) {
		t.Fatalf("comando inesperado: %s", rendered)
	}
}

func TestPrepareCLIPropagaRazonamientoPorConfigKey(t *testing.T) {
	plan, err := DefaultRegistry().Prepare(LaunchRequest{
		Agente:       "Codex1",
		ProyectoSlug: "orquestador",
		ProyectoRuta: "/tmp/orquestador",
		Modelo:       "gpt-5.4",
		Razonamiento: "xhigh",
		PerfilTarea:  "implementacion",
		Conector: ConnectorConfig{
			Slug:         "codex-cli",
			Transporte:   "cli",
			Comando:      "codex",
			MetadataJSON: `{"model_flag":"--model","reasoning_config_key":"model_reasoning_effort"}`,
		},
	})
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	rendered := RenderCommand(plan)
	if !strings.Contains(rendered, "'--model'") || !strings.Contains(rendered, "'gpt-5.4'") {
		t.Fatalf("comando sin modelo esperado: %s", rendered)
	}
	if !strings.Contains(rendered, "'-c'") || !strings.Contains(rendered, `'model_reasoning_effort="xhigh"'`) {
		t.Fatalf("comando sin override de razonamiento esperado: %s", rendered)
	}
}

func TestPrepareCLIExpandePlaceholdersDeConector(t *testing.T) {
	plan, err := DefaultRegistry().Prepare(LaunchRequest{
		Agente:       "Codex4",
		Rol:          "programador",
		ProyectoSlug: "orquestador",
		ProyectoRuta: "/tmp/orquestador",
		Modelo:       "gpt-5.4",
		Razonamiento: "high",
		PerfilTarea:  "implementacion",
		Conector: ConnectorConfig{
			Slug:       "codex-cli",
			Transporte: "cli",
			Comando:    "/tmp/codex-perfil",
			ArgsJSON:   `["{{agent}}","--modo","{{task_profile}}"]`,
			EnvJSON:    `{"ORQUESTA_AGENTE":"{{agent}}","ORQUESTA_PROYECTO":"{{project_slug}}","ORQUESTA_MODELO":"{{model}}","ORQUESTA_BIN":"{{orquesta_executable}}","PATH":"{{orquesta_bin_dir}}:{{host_path}}"}`,
		},
		Resume: ResumeContext{
			Branch: "feature/control-plane",
		},
	})
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if plan.Comando != "/tmp/codex-perfil" {
		t.Fatalf("comando inesperado: %s", plan.Comando)
	}
	if len(plan.Args) != 7 || plan.Args[0] != "Codex4" || plan.Args[2] != "implementacion" {
		t.Fatalf("args inesperados: %+v", plan.Args)
	}
	if plan.Args[3] != "--sandbox" || plan.Args[4] != "danger-full-access" || plan.Args[5] != "--ask-for-approval" || plan.Args[6] != "never" {
		t.Fatalf("faltan flags operativas por defecto: %+v", plan.Args)
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable: %v", err)
	}
	if plan.Env["ORQUESTA_AGENTE"] != "Codex4" || plan.Env["ORQUESTA_PROYECTO"] != "orquestador" || plan.Env["ORQUESTA_MODELO"] != "gpt-5.4" {
		t.Fatalf("env inesperado: %+v", plan.Env)
	}
	if plan.Env["ORQUESTA_BIN"] != exe {
		t.Fatalf("orquesta bin inesperado: %+v", plan.Env)
	}
	if !strings.Contains(plan.Env["PATH"], os.Getenv("PATH")) || !strings.HasPrefix(plan.Env["PATH"], filepath.Dir(exe)+":") {
		t.Fatalf("PATH expandido inesperado: %+v", plan.Env)
	}
}

func TestPrepareCLIResuelveComandoRelativoContraProyecto(t *testing.T) {
	plan, err := DefaultRegistry().Prepare(LaunchRequest{
		Agente:       "Codex8",
		ProyectoSlug: "orquestador",
		ProyectoRuta: "/srv/orquesta",
		Conector: ConnectorConfig{
			Slug:       "codex-cli",
			Transporte: "cli",
			Comando:    "../codex-perfiles/bin/codex-perfil",
			ArgsJSON:   `["{{agent}}"]`,
		},
	})
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if got, want := plan.Comando, "/srv/codex-perfiles/bin/codex-perfil"; got != want {
		t.Fatalf("comando relativo inesperado: got=%s want=%s", got, want)
	}
}

func TestPrepareCLIOllamaUsaModeloPosicional(t *testing.T) {
	plan, err := DefaultRegistry().Prepare(LaunchRequest{
		Agente:       "Ollama1",
		ProyectoSlug: "orquestador",
		ProyectoRuta: "/tmp/orquestador",
		Conector: ConnectorConfig{
			Slug:         "ollama-cli",
			Transporte:   "cli",
			Comando:      "ollama",
			ArgsJSON:     `["run"]`,
			MetadataJSON: `{"default_model":"qwen2.5-coder:7b","default_reasoning_effort":"medium","model_positional":true,"launch_prompt_transport":"post_start"}`,
		},
	})
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if plan.Modelo != "qwen2.5-coder:7b" {
		t.Fatalf("modelo inesperado: %+v", plan)
	}
	rendered := RenderCommand(plan)
	if !strings.Contains(rendered, "'run'") || !strings.Contains(rendered, "'qwen2.5-coder:7b'") {
		t.Fatalf("comando ollama inesperado: %s", rendered)
	}
}

func TestAplicarDefaultsConectorNoPisaPoliticaResuelta(t *testing.T) {
	perfil, modelo, razonamiento, err := AplicarDefaultsConector(
		ConnectorConfig{
			Slug:         "ollama-cli",
			Transporte:   "cli",
			Comando:      "ollama",
			MetadataJSON: `{"default_model":"qwen2.5-coder:7b","default_reasoning_effort":"medium","default_task_profile":"implementacion"}`,
		},
		"",
		"",
		"",
		"correccion",
		"gemma4:26b",
		"high",
	)
	if err != nil {
		t.Fatalf("AplicarDefaultsConector: %v", err)
	}
	if perfil != "correccion" || modelo != "gemma4:26b" || razonamiento != "high" {
		t.Fatalf("los defaults del conector no deberian pisar una politica ya resuelta: perfil=%q modelo=%q razonamiento=%q", perfil, modelo, razonamiento)
	}
}

func TestPrepareCLICodexSimpleUsaWrapperCanonicoYPerfil(t *testing.T) {
	tmp := t.TempDir()
	projectPath := filepath.Join(tmp, "orquesta")
	wrapperPath := filepath.Join(tmp, "codex-perfiles", "bin", "codex-perfil")
	if err := os.MkdirAll(filepath.Dir(wrapperPath), 0o755); err != nil {
		t.Fatalf("mkdir wrapper: %v", err)
	}
	if err := os.WriteFile(wrapperPath, []byte("#!/usr/bin/env bash\n"), 0o755); err != nil {
		t.Fatalf("write wrapper: %v", err)
	}
	plan, err := DefaultRegistry().Prepare(LaunchRequest{
		Agente:       "Codex8",
		ProyectoSlug: "orquestador",
		ProyectoRuta: projectPath,
		Conector: ConnectorConfig{
			Slug:       "codex-cli",
			Transporte: "cli",
			Comando:    "codex",
		},
	})
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if got, want := plan.Comando, wrapperPath; got != want {
		t.Fatalf("comando canonico inesperado: got=%s want=%s", got, want)
	}
	if len(plan.Args) != 5 {
		t.Fatalf("args inesperados: %+v", plan.Args)
	}
	if plan.Args[0] != "Codex8" {
		t.Fatalf("el wrapper codex debe recibir el perfil como primer argumento: %+v", plan.Args)
	}
}

func TestApplyLaunchPromptMetadataEmbebePromptInicial(t *testing.T) {
	plan := &LaunchPlan{
		Comando:          "codex",
		Args:             []string{"Codex2", "-C", "/tmp/demo"},
		BootstrapPrompt:  "Bootstrap de Orquesta para Codex2.",
		ContinuityPrompt: "Retoma el contexto del proyecto demo.",
		Env:              map[string]string{},
	}

	if err := ApplyLaunchPromptMetadata(plan, `{"launch_prompt_positional":true}`); err != nil {
		t.Fatalf("apply launch prompt metadata: %v", err)
	}
	if !plan.LaunchPromptEmbedded {
		t.Fatalf("el plan deberia marcar el prompt inicial como embebido: %+v", plan)
	}
	if got := plan.Args[len(plan.Args)-1]; got != "Bootstrap de Orquesta para Codex2.\n\nRetoma el contexto del proyecto demo." {
		t.Fatalf("prompt final inesperado: %q", got)
	}

	rendered := RenderCommand(plan)
	if !strings.Contains(rendered, "'Bootstrap de Orquesta para Codex2.\n\nRetoma el contexto del proyecto demo.'") {
		t.Fatalf("comando renderizado sin prompt inicial citado: %s", rendered)
	}
}

func TestApplyLaunchPromptMetadataPostStartNoEmbebePromptInicial(t *testing.T) {
	plan := &LaunchPlan{
		Comando:          "codex",
		Args:             []string{"Codex2", "-C", "/tmp/demo"},
		BootstrapPrompt:  "Bootstrap de Orquesta para Codex2.",
		ContinuityPrompt: "Retoma el contexto del proyecto demo.",
	}

	if err := ApplyLaunchPromptMetadata(plan, `{"launch_prompt_transport":"post_start","launch_prompt_delay_ms":1000}`); err != nil {
		t.Fatalf("apply launch prompt metadata: %v", err)
	}
	if plan.LaunchPromptEmbedded {
		t.Fatalf("el plan no deberia marcar el prompt inicial como embebido: %+v", plan)
	}
	if plan.LaunchPromptMode != "post_start" {
		t.Fatalf("modo de prompt inesperado: %+v", plan)
	}
	if plan.LaunchPromptDelayMS != 1000 {
		t.Fatalf("delay de prompt inesperado: %+v", plan)
	}
	if len(plan.Args) != 3 {
		t.Fatalf("args inesperados tras post_start: %+v", plan.Args)
	}
}

func TestApplyLaunchPromptMetadataExcesoTamanoCaeAPostStart(t *testing.T) {
	plan := &LaunchPlan{
		Comando:         "codex",
		Args:            []string{"Codex2", "-C", "/tmp/demo"},
		BootstrapPrompt: strings.Repeat("a", 32),
	}

	if err := ApplyLaunchPromptMetadata(plan, `{"launch_prompt_positional":true,"launch_prompt_max_bytes":8}`); err != nil {
		t.Fatalf("apply launch prompt metadata: %v", err)
	}
	if plan.LaunchPromptEmbedded {
		t.Fatalf("el plan no deberia embeber un prompt sobredimensionado: %+v", plan)
	}
	if plan.LaunchPromptMode != "post_start" {
		t.Fatalf("modo de prompt inesperado: %+v", plan)
	}
	if len(plan.Args) != 3 {
		t.Fatalf("args inesperados tras fallback por tamano: %+v", plan.Args)
	}
}

func TestNormalizeConnectorMetadataCodexPreservaPromptEmbebido(t *testing.T) {
	normalized := normalizeConnectorMetadata(ConnectorConfig{
		Slug:    "codex-cli",
		Comando: "codex",
	}, map[string]any{
		"launch_prompt_positional": true,
	})
	if got := stringMetadata(normalized, "launch_prompt_transport"); got != "" {
		t.Fatalf("codex no deberia forzar post_start cuando el conector ya soporta prompt embebido: %q", got)
	}
	if !boolMetadata(normalized, "launch_prompt_positional") {
		t.Fatalf("codex deberia preservar launch_prompt_positional: %+v", normalized)
	}
	if !hasMetadataKey(normalized, "can_send_input") || boolMetadata(normalized, "can_send_input") {
		t.Fatalf("codex deberia seguir con can_send_input=false por defecto: %+v", normalized)
	}
	if !hasMetadataKey(normalized, "can_run_slash_commands") || boolMetadata(normalized, "can_run_slash_commands") {
		t.Fatalf("codex no deberia anunciar slash commands por defecto en el plan canonico: %+v", normalized)
	}
}

func TestPrepareRemoteGeneraContratoMinimoDeAdaptador(t *testing.T) {
	plan, err := DefaultRegistry().Prepare(LaunchRequest{
		Agente:       "Codex1",
		ProyectoSlug: "orquestador",
		ProyectoRuta: "/tmp/orquestador",
		Conector: ConnectorConfig{
			Slug:         "codex-remote",
			Transporte:   "api",
			Comando:      "https://runtime.example.test",
			MetadataJSON: `{"resume_path":"/sessions/resume","status_path":"/sessions/status","input_path":"/sessions/input","auth_header":"Authorization","auth_token_env":"ORQUESTA_REMOTE_TOKEN","auth_mode":"oauth","preserve_external_session_on_stop":true,"requires_human_reauth":true,"timeout_ms":2500,"control_retry_count":4,"control_retry_backoff_ms":150,"control_retry_statuses":[408,429,503]}`,
		},
	})
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if strings.TrimSpace(plan.RemoteConfigJSON) == "" {
		t.Fatalf("faltaba remote_config_json: %+v", plan)
	}
	cfg := map[string]any{}
	if err := json.Unmarshal([]byte(plan.RemoteConfigJSON), &cfg); err != nil {
		t.Fatalf("parse remote config: %v", err)
	}
	if cfg["endpoint"] != "https://runtime.example.test" {
		t.Fatalf("endpoint inesperado: %+v", cfg)
	}
	if cfg["resume_path"] != "/sessions/resume" || cfg["status_path"] != "/sessions/status" || cfg["input_path"] != "/sessions/input" {
		t.Fatalf("paths inesperados: %+v", cfg)
	}
	if cfg["auth_mode"] != "oauth" || cfg["preserve_external_session_on_stop"] != true || cfg["requires_human_reauth"] != true {
		t.Fatalf("auth/persistencia inesperadas: %+v", cfg)
	}
	if cfg["timeout_ms"] != float64(2500) {
		t.Fatalf("timeout inesperado: %+v", cfg)
	}
	if cfg["control_retry_count"] != float64(4) || cfg["control_retry_backoff_ms"] != float64(150) {
		t.Fatalf("retries inesperados: %+v", cfg)
	}
}

func TestPrepareRemotePermiteDesactivarCapacidadesPorMetadata(t *testing.T) {
	plan, err := DefaultRegistry().Prepare(LaunchRequest{
		Agente:       "Codex1",
		ProyectoSlug: "orquestador",
		ProyectoRuta: "/tmp/orquestador",
		Conector: ConnectorConfig{
			Slug:         "codex-remote",
			Transporte:   "api",
			Comando:      "https://runtime.example.test",
			MetadataJSON: `{"pause_path":"","input_path":"","can_pause":false,"can_send_input":false,"can_stop":false}`,
		},
	})
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	cfg := map[string]any{}
	if err := json.Unmarshal([]byte(plan.RemoteConfigJSON), &cfg); err != nil {
		t.Fatalf("parse remote config: %v", err)
	}
	if cfg["pause_path"] != "" || cfg["input_path"] != "" {
		t.Fatalf("paths remotos inesperados: %+v", cfg)
	}
	if cfg["can_pause"] != false || cfg["can_send_input"] != false || cfg["can_stop"] != false {
		t.Fatalf("capacidades remotas inesperadas: %+v", cfg)
	}
}

func TestPrepareCLIPropagaCanSendInputDesdeMetadata(t *testing.T) {
	plan, err := DefaultRegistry().Prepare(LaunchRequest{
		Agente:       "Codex1",
		ProyectoSlug: "orquestador",
		ProyectoRuta: "/tmp/orquestador",
		Conector: ConnectorConfig{
			Slug:         "codex-cli",
			Transporte:   "cli",
			Comando:      "codex",
			MetadataJSON: `{"can_send_input":false}`,
		},
	})
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if plan.CanSendInput == nil || *plan.CanSendInput {
		t.Fatalf("el plan CLI deberia propagar can_send_input=false desde metadata: %+v", plan)
	}
	if plan.MailboxDeliveryMode != MailboxDeliveryBootstrapOnly {
		t.Fatalf("modo mailbox inesperado: %+v", plan)
	}
}

func TestPrepareCLICodexDesactivaSendInputInteractivoPorDefecto(t *testing.T) {
	plan, err := DefaultRegistry().Prepare(LaunchRequest{
		Agente:       "Codex1",
		ProyectoSlug: "orquestador",
		ProyectoRuta: "/tmp/orquestador",
		Conector: ConnectorConfig{
			Slug:       "codex-cli",
			Transporte: "cli",
			Comando:    "codex",
		},
	})
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if plan.CanSendInput == nil || *plan.CanSendInput {
		t.Fatalf("codex-cli deberia desactivar input interactivo por defecto: %+v", plan)
	}
	if plan.MailboxDeliveryMode != MailboxDeliveryBootstrapOnly {
		t.Fatalf("codex-cli deberia dejar mailbox en bootstrap_only: %+v", plan)
	}
	rendered := RenderCommand(plan)
	for _, token := range []string{"'--sandbox' 'danger-full-access'", "'--ask-for-approval' 'never'"} {
		if !strings.Contains(rendered, token) {
			t.Fatalf("codex-cli deberia arrancar con permisos reales por defecto; falta %s en %s", token, rendered)
		}
	}
}

func TestPrepareCLICodexResumeMantieneBootstrapOnlyAunqueTengaSesionExterna(t *testing.T) {
	plan, err := DefaultRegistry().Prepare(LaunchRequest{
		Agente:       "Codex7",
		ProyectoSlug: "orquestador",
		ProyectoRuta: "/tmp/orquestador",
		Conector: ConnectorConfig{
			Slug:       "codex-cli",
			Transporte: "cli",
			Comando:    "codex",
		},
		Resume: ResumeContext{
			ExternalSessionID: "sess-codex-7",
		},
	})
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if plan.Modo != "resume" {
		t.Fatalf("modo inesperado: %+v", plan)
	}
	if plan.CanSendInput == nil || *plan.CanSendInput {
		t.Fatalf("codex-cli deberia seguir sin input interactivo en resume: %+v", plan)
	}
	if plan.MailboxDeliveryMode != MailboxDeliveryBootstrapOnly {
		t.Fatalf("codex-cli resume deberia seguir en bootstrap_only: %+v", plan)
	}
}

func TestPrepareCLICodexRespetaSandboxYApprovalExplicitos(t *testing.T) {
	plan, err := DefaultRegistry().Prepare(LaunchRequest{
		Agente:       "Codex1",
		ProyectoSlug: "orquestador",
		ProyectoRuta: "/tmp/orquestador",
		Conector: ConnectorConfig{
			Slug:       "codex-cli",
			Transporte: "cli",
			Comando:    "codex",
			ArgsJSON:   `["--sandbox","workspace-write","--ask-for-approval","on-request"]`,
		},
	})
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	rendered := RenderCommand(plan)
	if strings.Count(rendered, "'--sandbox'") != 1 || strings.Count(rendered, "'--ask-for-approval'") != 1 {
		t.Fatalf("no deberia duplicar sandbox/approval cuando ya vienen definidos: %s", rendered)
	}
	if !strings.Contains(rendered, "'workspace-write'") || !strings.Contains(rendered, "'on-request'") {
		t.Fatalf("deberia respetar sandbox/approval explicitos: %s", rendered)
	}
}
