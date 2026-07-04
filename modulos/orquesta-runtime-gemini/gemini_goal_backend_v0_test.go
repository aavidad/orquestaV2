package orquestaruntimegemini

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	orquestagoal "orquesta/modulos/orquesta-goal"
)

func TestGeminiGoalBackendV0LaunchMaterializaSpecYPromptDurableV0(t *testing.T) {
	root := t.TempDir()
	backend := GeminiGoalBackendV0{
		ProjectWorkDir: filepath.Join(root, "project"),
		RuntimeWorkDir: filepath.Join(root, "runtime"),
	}
	spec := geminiGoalSpecForTestV0()
	spec.AcceptanceCriteria = []string{"criterio-ref-gemini-goal-prompt"}
	spec.ArtifactContracts = []orquestagoal.GoalArtifactContractV0{{
		ArtifactRef:  "artifact-ref-gemini-goal-source",
		ArtifactType: "source_tree",
		Required:     true,
	}}
	spec.ClosurePolicy.RequiredEvidenceRefs = []string{"evidence-ref-gemini-goal-required"}

	receipt, err := backend.LaunchGoalWorkV0(context.Background(), spec)
	if err != nil {
		t.Fatalf("LaunchGoalWorkV0: %v", err)
	}
	if receipt.Status != orquestagoal.GoalStatusRunningV0 ||
		receipt.GoalRef != spec.GoalRef ||
		!strings.HasPrefix(receipt.ExternalGoalRef, "gemini-goal-") {
		t.Fatalf("receipt inesperado: %+v", receipt)
	}
	if !containsGeminiGoalStringV0(receipt.EvidenceRefs, GeminiGoalEvidenceLaunchedV0) {
		t.Fatalf("receipt sin evidencia de launch: %+v", receipt.EvidenceRefs)
	}
	specPath := filepath.Join(backend.RuntimeWorkDir, geminiGoalSpecFileNameV0(spec.GoalRef))
	promptPath := filepath.Join(backend.RuntimeWorkDir, geminiGoalPromptFileNameV0(spec.GoalRef))
	if _, err := os.Stat(specPath); err != nil {
		t.Fatalf("spec no materializado: %v", err)
	}
	prompt := mustReadGeminiGoalFileForTestV0(t, promptPath)
	for _, want := range []string{
		"backend goal-first Gemini",
		"RESULTADO DURABLE NEUTRAL",
		"orquesta_goal_result_v0.json",
		"schema_version orquesta_goal_result.v0",
		"materialized_artifacts",
		"arrays de strings",
		"no uses objetos",
		"criterio-ref-gemini-goal-prompt",
		"artifact-ref-gemini-goal-source",
		"evidence-ref-gemini-goal-required",
		"docs",
		"required-test-ref-gemini-goal",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt sin %q:\n%s", want, prompt)
		}
	}
}

func TestGeminiGoalBackendV0ObserveNormalizaEvidenceRefsObjetoRecuperableV0(t *testing.T) {
	root := t.TempDir()
	backend := GeminiGoalBackendV0{
		ProjectWorkDir: filepath.Join(root, "project"),
		RuntimeWorkDir: filepath.Join(root, "runtime"),
	}
	spec := geminiGoalSpecForTestV0()
	if _, err := backend.LaunchGoalWorkV0(context.Background(), spec); err != nil {
		t.Fatalf("LaunchGoalWorkV0: %v", err)
	}
	resultPath := filepath.Join(backend.ProjectWorkDir, "docs", GeminiGoalResultFileNameV0)
	if err := os.MkdirAll(filepath.Dir(resultPath), 0o700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	raw := `{
		"schema_version":"orquesta_goal_result.v0",
		"status":"complete",
		"goal_ref":"goal-ref-gemini-runtime-001",
		"artifact_refs":["artifact-ref-provider-described"],
		"artifact_paths":["docs/orquesta_goal_result_v0.json"],
		"materialized_artifacts":[{
			"artifact_ref":"artifact-ref-provider-described",
			"path":"docs/orquesta_goal_result_v0.json",
			"status":"valid",
			"evidence_refs":[{"ref":"evidence-ref-nested-described","description":"detalle no canonico"}]
		}],
		"required_test_results":[{
			"test_ref":"required-test-ref-gemini-goal",
			"status":"passed",
			"evidence_refs":[{"ref":"evidence-ref-test-described","description":"detalle no canonico"}]
		}],
		"evidence_refs":[{"ref":"evidence-ref-provider-described","description":"detalle no canonico"}]
	}`
	if err := os.WriteFile(resultPath, []byte(raw), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	got, err := backend.ObserveGoalWorkV0(context.Background(), orquestagoal.GoalObservationRequestV0{
		GoalRef: spec.GoalRef,
	})
	if err != nil {
		t.Fatalf("ObserveGoalWorkV0: %v", err)
	}
	if len(got.MaterializedArtifacts) != 1 || len(got.RequiredTestResults) != 1 ||
		got.Status != orquestagoal.GoalStatusCompleteV0 ||
		!containsGeminiGoalStringV0(got.EvidenceRefs, "evidence-ref-provider-described") ||
		!containsGeminiGoalStringV0(got.MaterializedArtifacts[0].EvidenceRefs, "evidence-ref-nested-described") ||
		!containsGeminiGoalStringV0(got.RequiredTestResults[0].EvidenceRefs, "evidence-ref-test-described") {
		t.Fatalf("result no normalizado: %+v", got)
	}
}

func TestGeminiGoalBackendV0ObserveLeeResultadoDurableDelWriteSetV0(t *testing.T) {
	root := t.TempDir()
	backend := GeminiGoalBackendV0{
		ProjectWorkDir: filepath.Join(root, "project"),
		RuntimeWorkDir: filepath.Join(root, "runtime"),
	}
	spec := geminiGoalSpecForTestV0()
	if _, err := backend.LaunchGoalWorkV0(context.Background(), spec); err != nil {
		t.Fatalf("LaunchGoalWorkV0: %v", err)
	}
	resultPath := filepath.Join(backend.ProjectWorkDir, "docs", GeminiGoalResultFileNameV0)
	if err := os.MkdirAll(filepath.Dir(resultPath), 0o700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	result := orquestagoal.GoalWorkResultV0{
		SchemaVersion: orquestagoal.GoalWorkResultSchemaV0,
		Status:        orquestagoal.GoalStatusCompleteV0,
		GoalRef:       spec.GoalRef,
		Summary:       "gemini goal complete",
		ArtifactPaths: []string{"docs/orquesta_goal_result_v0.json"},
		MaterializedArtifacts: []orquestagoal.GoalMaterializedArtifactV0{{
			ArtifactRef:  "artifact-ref-gemini-goal",
			Path:         "docs/orquesta_goal_result_v0.json",
			ArtifactType: "goal_result",
			Status:       orquestagoal.GoalMaterializedArtifactStatusValidV0,
			EvidenceRefs: []string{"evidence-ref-gemini-goal-result"},
		}},
		Checklist: orquestagoal.GoalWorkChecklistV0{
			ExpectedRefs:  []string{"gemini_goal_result"},
			CompletedRefs: []string{"gemini_goal_result"},
		},
		RequiredTestResults: []orquestagoal.GoalRequiredTestResultV0{{
			TestRef:      "required-test-ref-gemini-goal",
			Status:       "passed",
			EvidenceRefs: []string{"evidence-ref-gemini-goal-test"},
		}},
		EvidenceRefs: []string{"evidence-ref-gemini-goal-result"},
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
		!containsGeminiGoalStringV0(observed.EvidenceRefs, GeminiGoalEvidenceResultReadV0) {
		t.Fatalf("observed inesperado: %+v", observed)
	}
}

func TestGeminiGoalBackendV0ObserveSinResultadoDevuelveRunningV0(t *testing.T) {
	root := t.TempDir()
	backend := GeminiGoalBackendV0{
		ProjectWorkDir: filepath.Join(root, "project"),
		RuntimeWorkDir: filepath.Join(root, "runtime"),
	}
	spec := geminiGoalSpecForTestV0()
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
		!containsGeminiGoalStringV0(observed.EvidenceRefs, GeminiGoalEvidenceResultPendingV0) {
		t.Fatalf("observed inesperado: %+v", observed)
	}
}

func geminiGoalSpecForTestV0() orquestagoal.GoalWorkSpecV0 {
	return orquestagoal.GoalWorkSpecV0{
		SchemaVersion: orquestagoal.GoalWorkSpecSchemaV0,
		GoalRef:       "goal-ref-gemini-runtime-001",
		RequestRef:    "request-ref-gemini-runtime-001",
		RunRef:        "run-ref-gemini-runtime-001",
		Objective:     "Producir resultado durable neutral desde Gemini.",
		DirectorKind:  orquestagoal.GoalDirectorKindRuntimeGoalV0,
		WriteSet: []orquestagoal.GoalWriteScopeV0{{
			Path:    "docs",
			Purpose: "resultado durable",
		}},
		RequiredTests: []orquestagoal.GoalRequiredTestV0{{
			TestRef: "required-test-ref-gemini-goal",
			Command: "go test ./...",
		}},
	}
}

func mustReadGeminiGoalFileForTestV0(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", path, err)
	}
	return string(data)
}

func containsGeminiGoalStringV0(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
