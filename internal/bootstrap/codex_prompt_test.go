package bootstrap

import (
	"context"
	"orquesta/internal/adapters/agent/codex"
	localruntime "orquesta/internal/adapters/system/local"
	"orquesta/internal/i18n"
	"os"
	"strings"
	"testing"
)

func TestCodexRuntimeUsesCatalogOwnedPrompt(t *testing.T) {
	catalog, err := i18n.LoadBundled()
	v22NoError(t, err)
	prompt := fullCodexPromptFixture()
	for _, test := range []struct {
		locale, marker string
	}{
		{locale: "en-GB", marker: "Objective:\nobjective:catalog-owned {goal_ref}"},
		{locale: "fr", marker: "Objetivo:\nobjective:catalog-owned {goal_ref}"},
	} {
		renderer, err := newCatalogCodexPromptRenderer(catalog, test.locale)
		if err != nil {
			t.Fatalf("renderer locale=%s: %v", test.locale, err)
		}
		rendered, err := renderer.RenderAgentPrompt(prompt)
		if err != nil || !strings.Contains(rendered, test.marker) ||
			!strings.Contains(rendered, "project:default") ||
			!strings.Contains(rendered, "goal:catalog") ||
			!strings.Contains(rendered, "work-item:catalog") ||
			!strings.Contains(rendered, "execution:catalog") ||
			!strings.Contains(rendered, "phase-instance:catalog") ||
			!strings.Contains(rendered, "internal/bootstrap") {
			t.Fatalf("locale=%s prompt=%q error=%v", test.locale, rendered, err)
		}
	}
	root, configPath := credentialFactoryFixture(t)
	handle, err := os.OpenFile(configPath, os.O_APPEND|os.O_WRONLY, 0)
	v22NoError(t, err)
	if _, err := handle.WriteString("\n[api]\nlocale = \"en\"\n"); err != nil {
		_ = handle.Close()
		t.Fatal(err)
	}
	if err := handle.Close(); err != nil {
		t.Fatal(err)
	}
	snapshot, err := loadConfigSnapshot(context.Background(), configPath)
	if err != nil {
		t.Fatalf("load config in %s: %v", root, err)
	}
	renderer, err := newCatalogCodexPromptRenderer(catalog, snapshot.APILocale())
	v22NoError(t, err)
	agent, err := productionAgentFactory(snapshot, localruntime.Clock{}, renderer)
	if err != nil {
		t.Fatalf("production factory: %v", err)
	}
	t.Cleanup(func() { _ = agent.Shutdown(context.Background()) })
	launchAndAwaitCredentialArtifact(
		t, agent, "catalog-owned-prompt", "bootstrap-helper:v1", "artifact:credential-v1",
	)
}

func fullCodexPromptFixture() codex.AgentPrompt {
	return codex.AgentPrompt{
		ProjectRef:         "project:default",
		GoalRef:            "goal:catalog",
		WorkItemRef:        "work-item:catalog",
		ExecutionRef:       "execution:catalog",
		PlanGeneration:     "2",
		AppSpecGeneration:  "3",
		Objective:          "objective:catalog-owned {goal_ref}",
		PhaseRef:           "phase-instance:catalog",
		PhaseKey:           "phase:build",
		PhaseTemplateRef:   "phase-template:program",
		PhaseInputRefs:     "input:app-spec",
		PhaseCriterionRefs: "criterion:tests-green",
		RoleKey:            "role:worker",
		SkillRefs:          "skill:go",
		ToolRefs:           "tool:go-test",
		CapabilityRefs:     "capability:patch",
		WriteSet:           "internal/bootstrap",
		OutputContract:     "evidence_bundle",
		ArtifactMediaType:  "text/markdown",
	}
}
