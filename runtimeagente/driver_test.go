/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package runtimeagente

import (
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
