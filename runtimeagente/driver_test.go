/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package runtimeagente

import (
	"encoding/json"
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
	if !strings.Contains(rendered, "resume") || !strings.Contains(rendered, "--session abc-123") {
		t.Fatalf("comando inesperado: %s", rendered)
	}
}

func TestPrepareCLIPropagaModeloYRazonamiento(t *testing.T) {
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
	if !strings.Contains(rendered, "--model gpt-5.4") || !strings.Contains(rendered, "--reasoning-effort xhigh") {
		t.Fatalf("comando inesperado: %s", rendered)
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
