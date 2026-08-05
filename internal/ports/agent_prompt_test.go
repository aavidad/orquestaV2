package ports

import (
	"crypto/sha256"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestAgentPromptProjectionIsExactAllowlist(t *testing.T) {
	request := validAgentLaunchRequest(t)
	sessionRef, _ := NewExecutionSessionRef("execution-session:private")
	request.SessionRef = sessionRef
	request.SpecHash = strings.Repeat("a", 64)
	request.IdempotencyKey = "idempotency:private"
	request.PhaseInputRefs = []string{"input:first", "input:second"}
	request.WriteSet = []string{"internal/ports", "internal/application"}
	egressPayload := `{"private-egress-marker":"do-not-project"}`
	request.EgressAuthority = AgentLaunchEgressAuthority{
		PolicyRef:        "egress-policy:private-marker",
		PayloadSHA256:    fmt.Sprintf("%x", sha256.Sum256([]byte(egressPayload))),
		CanonicalPayload: egressPayload,
	}
	if err := ValidateAgentLaunchRequest(request); err != nil {
		t.Fatalf("prompt fixture must start from a valid launch request: %v", err)
	}

	prompt := AgentPromptFromLaunchRequest(request)
	want := AgentPrompt{
		ProjectRef: request.ProjectRef.String(), GoalRef: request.GoalRef.String(),
		WorkItemRef: request.WorkItemRef.String(), ExecutionRef: request.ExecutionRef.String(),
		PlanGeneration: "2", AppSpecGeneration: "3", Objective: request.Objective,
		PhaseRef: request.PhaseRef, PhaseKey: request.PhaseKey, PhaseTemplateRef: request.PhaseTemplateRef,
		PhaseInputRefs: "input:first\ninput:second", PhaseCriterionRefs: "criterion:tests-green",
		RoleKey: request.RoleKey, SkillRefs: "skill:go", ToolRefs: "tool:go-test",
		CapabilityRefs: request.CapabilityRefs[0], WriteSet: "internal/ports\ninternal/application",
		OutputContract: request.OutputContract, ArtifactMediaType: request.ArtifactMediaType,
	}
	if !reflect.DeepEqual(prompt, want) {
		t.Fatalf("projection = %+v, want %+v", prompt, want)
	}

	promptType := reflect.TypeOf(prompt)
	for _, forbiddenField := range []string{
		"SpecHash", "IdempotencyKey", "SessionRef", "AccessAuthority", "EffectAuthority", "EgressAuthority",
		"ExecutionWorkspaceRef", "ReferenciaColocacion",
	} {
		if _, exists := promptType.FieldByName(forbiddenField); exists {
			t.Fatalf("private launch field %s entered prompt DTO", forbiddenField)
		}
	}
	encoded := strings.Join([]string{
		prompt.ProjectRef, prompt.GoalRef, prompt.WorkItemRef, prompt.ExecutionRef,
		prompt.PlanGeneration, prompt.AppSpecGeneration, prompt.Objective, prompt.PhaseRef,
		prompt.PhaseKey, prompt.PhaseTemplateRef, prompt.PhaseInputRefs, prompt.PhaseCriterionRefs,
		prompt.RoleKey, prompt.SkillRefs, prompt.ToolRefs, prompt.CapabilityRefs,
		prompt.WriteSet, prompt.OutputContract, prompt.ArtifactMediaType,
	}, "\x00")
	for _, forbidden := range []string{
		request.SpecHash, request.IdempotencyKey, request.SessionRef.String(),
		request.EgressAuthority.PolicyRef, request.EgressAuthority.PayloadSHA256, egressPayload,
	} {
		if strings.Contains(encoded, forbidden) {
			t.Fatalf("private launch field leaked into prompt: %q", forbidden)
		}
	}
}
