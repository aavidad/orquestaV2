package orquestaruntimeclaude

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	orquestagoal "orquesta/modulos/orquesta-goal"
)

func TestClaudeGoalBackendV0LaunchMaterializaSpecYPromptDurableV0(t *testing.T) {
	root := t.TempDir()
	backend := ClaudeGoalBackendV0{
		ProjectWorkDir: filepath.Join(root, "project"),
		RuntimeWorkDir: filepath.Join(root, "runtime"),
	}
	spec := claudeGoalSpecForTestV0()

	receipt, err := backend.LaunchGoalWorkV0(context.Background(), spec)
	if err != nil {
		t.Fatalf("LaunchGoalWorkV0: %v", err)
	}
	if receipt.Status != orquestagoal.GoalStatusRunningV0 ||
		receipt.GoalRef != spec.GoalRef ||
		!strings.HasPrefix(receipt.ExternalGoalRef, "claude-goal-") {
		t.Fatalf("receipt inesperado: %+v", receipt)
	}
	if !containsClaudeGoalStringV0(receipt.EvidenceRefs, ClaudeGoalEvidenceLaunchedV0) {
		t.Fatalf("receipt sin evidencia de launch: %+v", receipt.EvidenceRefs)
	}
	specPath := filepath.Join(backend.RuntimeWorkDir, claudeGoalSpecFileNameV0(spec.GoalRef))
	promptPath := filepath.Join(backend.RuntimeWorkDir, claudeGoalPromptFileNameV0(spec.GoalRef))
	if _, err := os.Stat(specPath); err != nil {
		t.Fatalf("spec no materializado: %v", err)
	}
	prompt := mustReadClaudeGoalFileForTestV0(t, promptPath)
	for _, want := range []string{
		"backend goal-first Claude",
		"RESULTADO DURABLE NEUTRAL",
		"orquesta_goal_result_v0.json",
		"schema_version orquesta_goal_result.v0",
		"materialized_artifacts",
		"docs",
		"required-test-ref-claude-goal",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt sin %q:\n%s", want, prompt)
		}
	}
}

func TestClaudeGoalBackendV0ObserveLeeResultadoDurableDelWriteSetV0(t *testing.T) {
	root := t.TempDir()
	backend := ClaudeGoalBackendV0{
		ProjectWorkDir: filepath.Join(root, "project"),
		RuntimeWorkDir: filepath.Join(root, "runtime"),
	}
	spec := claudeGoalSpecForTestV0()
	if _, err := backend.LaunchGoalWorkV0(context.Background(), spec); err != nil {
		t.Fatalf("LaunchGoalWorkV0: %v", err)
	}
	resultPath := filepath.Join(backend.ProjectWorkDir, "docs", ClaudeGoalResultFileNameV0)
	if err := os.MkdirAll(filepath.Dir(resultPath), 0o700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	result := orquestagoal.GoalWorkResultV0{
		SchemaVersion: orquestagoal.GoalWorkResultSchemaV0,
		Status:        orquestagoal.GoalStatusCompleteV0,
		GoalRef:       spec.GoalRef,
		Summary:       "claude goal complete",
		ArtifactPaths: []string{"docs/orquesta_goal_result_v0.json"},
		MaterializedArtifacts: []orquestagoal.GoalMaterializedArtifactV0{{
			ArtifactRef:  "artifact-ref-claude-goal",
			Path:         "docs/orquesta_goal_result_v0.json",
			ArtifactType: "goal_result",
			Status:       orquestagoal.GoalMaterializedArtifactStatusValidV0,
			EvidenceRefs: []string{"evidence-ref-claude-goal-result"},
		}},
		Checklist: orquestagoal.GoalWorkChecklistV0{
			ExpectedRefs:  []string{"claude_goal_result"},
			CompletedRefs: []string{"claude_goal_result"},
		},
		RequiredTestResults: []orquestagoal.GoalRequiredTestResultV0{{
			TestRef:      "required-test-ref-claude-goal",
			Status:       "passed",
			EvidenceRefs: []string{"evidence-ref-claude-goal-test"},
		}},
		EvidenceRefs: []string{"evidence-ref-claude-goal-result"},
	}
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if err := os.WriteFile(resultPath, data, 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	observed, err := backend.ObserveGoalWorkV0(context.Background(), orquestagoal.GoalObservationRequestV0{
		GoalRef: spec.GoalRef,
	})
	if err != nil {
		t.Fatalf("ObserveGoalWorkV0: %v", err)
	}
	if observed.Status != orquestagoal.GoalStatusCompleteV0 ||
		observed.GoalRef != spec.GoalRef ||
		!containsClaudeGoalStringV0(observed.EvidenceRefs, ClaudeGoalEvidenceResultReadV0) {
		t.Fatalf("observed inesperado: %+v", observed)
	}
}

func TestClaudeGoalBackendV0ObserveSinResultadoDevuelveRunningV0(t *testing.T) {
	root := t.TempDir()
	backend := ClaudeGoalBackendV0{
		ProjectWorkDir: filepath.Join(root, "project"),
		RuntimeWorkDir: filepath.Join(root, "runtime"),
	}
	spec := claudeGoalSpecForTestV0()
	if _, err := backend.LaunchGoalWorkV0(context.Background(), spec); err != nil {
		t.Fatalf("LaunchGoalWorkV0: %v", err)
	}

	observed, err := backend.ObserveGoalWorkV0(context.Background(), orquestagoal.GoalObservationRequestV0{
		GoalRef: spec.GoalRef,
	})
	if err != nil {
		t.Fatalf("ObserveGoalWorkV0: %v", err)
	}
	if observed.Status != orquestagoal.GoalStatusRunningV0 ||
		!containsClaudeGoalStringV0(observed.EvidenceRefs, ClaudeGoalEvidenceResultPendingV0) {
		t.Fatalf("observed inesperado: %+v", observed)
	}
}

func claudeGoalSpecForTestV0() orquestagoal.GoalWorkSpecV0 {
	return orquestagoal.GoalWorkSpecV0{
		SchemaVersion: orquestagoal.GoalWorkSpecSchemaV0,
		GoalRef:       "goal-ref-claude-runtime-001",
		RequestRef:    "request-ref-claude-runtime-001",
		RunRef:        "run-ref-claude-runtime-001",
		Objective:     "Producir resultado durable neutral desde Claude.",
		DirectorKind:  orquestagoal.GoalDirectorKindRuntimeGoalV0,
		WriteSet: []orquestagoal.GoalWriteScopeV0{{
			Path:    "docs",
			Purpose: "resultado durable",
		}},
		RequiredTests: []orquestagoal.GoalRequiredTestV0{{
			TestRef: "required-test-ref-claude-goal",
			Command: "go test ./...",
		}},
	}
}

func mustReadClaudeGoalFileForTestV0(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", path, err)
	}
	return string(data)
}

func containsClaudeGoalStringV0(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
