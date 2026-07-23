package codex

import (
	"errors"
	"strings"
	"testing"

	"orquesta/internal/ports"
)

type testPromptRenderer struct {
	render func(AgentPrompt) (string, error)
}

func (renderer testPromptRenderer) RenderAgentPrompt(prompt AgentPrompt) (string, error) {
	if renderer.render != nil {
		return renderer.render(prompt)
	}
	return "Project:\n" + prompt.ProjectRef + "\n" +
		"Goal:\n" + prompt.GoalRef + "\n" +
		"WorkItem:\n" + prompt.WorkItemRef + "\n" +
		"Execution:\n" + prompt.ExecutionRef + "\n" +
		"Plan generation:\n" + prompt.PlanGeneration + "\n" +
		"AppSpec generation:\n" + prompt.AppSpecGeneration + "\n" +
		"Test objective:\n" + prompt.Objective + "\n" +
		"Phase:\n" + prompt.PhaseRef + "\n" +
		"Phase key:\n" + prompt.PhaseKey + "\n" +
		"Phase template:\n" + prompt.PhaseTemplateRef + "\n" +
		"Inputs:\n" + prompt.PhaseInputRefs + "\n" +
		"Criteria:\n" + prompt.PhaseCriterionRefs + "\n" +
		"Role:\n" + prompt.RoleKey + "\n" +
		"Skills:\n" + prompt.SkillRefs + "\n" +
		"Tools:\n" + prompt.ToolRefs + "\n" +
		"Capabilities:\n" + prompt.CapabilityRefs + "\n" +
		"Writes:\n" + prompt.WriteSet + "\n" +
		"Output:\n" + prompt.OutputContract + "\n" +
		"Media:\n" + prompt.ArtifactMediaType + "\n" +
		"Return only the JSON object required by the supplied schema.\n", nil
}

func TestAdapterUsesInjectedTypedPromptRenderer(t *testing.T) {
	request := testRequest(t, "typed-prompt", "helper:success", 1024)
	var captured AgentPrompt
	config := testConfig(t)
	config.PromptRenderer = testPromptRenderer{render: func(prompt AgentPrompt) (string, error) {
		captured = prompt
		return "rendered", nil
	}}
	adapter := openTestAdapter(t, config)
	rendered, err := adapter.renderAgentPrompt(request)
	if err != nil || rendered != "rendered" {
		t.Fatalf("renderAgentPrompt() = %q, %v", rendered, err)
	}
	for _, value := range []string{
		captured.ProjectRef, captured.GoalRef, captured.WorkItemRef, captured.ExecutionRef,
		captured.PlanGeneration, captured.AppSpecGeneration, captured.Objective,
		captured.PhaseRef, captured.PhaseKey, captured.PhaseTemplateRef,
		captured.PhaseInputRefs, captured.PhaseCriterionRefs, captured.RoleKey,
		captured.SkillRefs, captured.ToolRefs, captured.CapabilityRefs, captured.WriteSet,
		captured.OutputContract, captured.ArtifactMediaType,
	} {
		if value == "" {
			t.Fatalf("typed prompt omitted request metadata: %+v", captured)
		}
	}
	if strings.Contains(rendered, request.SpecHash) {
		t.Fatal("spec hash leaked into rendered prompt")
	}

	request.SessionRef, _ = ports.NewExecutionSessionRef("execution-session:must-not-leak")
	config.PromptRenderer = testPromptRenderer{}
	adapter = openTestAdapter(t, config)
	rendered, err = adapter.renderAgentPrompt(request)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{
		request.SpecHash, request.SessionRef.String(), helperSessionBearer,
	} {
		if strings.Contains(rendered, forbidden) {
			t.Fatalf("private/non-prompt value %q leaked into prompt", forbidden)
		}
	}
}

func TestAdapterRejectsMissingOrFailingPromptRenderer(t *testing.T) {
	config := testConfig(t)
	config.PromptRenderer = nil
	if adapter, err := New(config); adapter != nil || ErrorCode(err) != CodePromptRendererInvalid {
		t.Fatalf("missing renderer adapter=%v error=%v", adapter, err)
	}

	config = testConfig(t)
	config.PromptRenderer = testPromptRenderer{render: func(AgentPrompt) (string, error) {
		return "", errors.New("render failed")
	}}
	adapter := openTestAdapter(t, config)
	request := testRequest(t, "prompt-failed", "helper:success", 1024)
	if _, err := adapter.Launch(t.Context(), request); err != nil {
		t.Fatalf("Launch() must durably admit before render: %v", err)
	}
	observation := awaitTerminal(t, adapter, request.ExecutionRef)
	if observation.ErrorCode != CodePromptRenderFailed {
		t.Fatalf("prompt failure observation=%+v", observation)
	}
}
