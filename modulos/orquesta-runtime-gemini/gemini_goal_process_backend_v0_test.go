package orquestaruntimegemini

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func TestGeminiGoalProcessBackendV0LanzaProcesoYObservaResultadoDurableV0(t *testing.T) {
	root := t.TempDir()
	goalRef := "goal-ref-gemini-process-complete-001"
	backend := geminiGoalProcessBackendForTestV0(
		t,
		root,
		geminiGoalFakeCommandWritesResultV0(t, root, goalRef),
	)
	spec := geminiGoalProcessSpecForTestV0(goalRef)

	receipt, err := backend.LaunchGoalWorkV0(context.Background(), spec)
	if err != nil {
		t.Fatalf("LaunchGoalWorkV0: %v", err)
	}
	if receipt.Status != orquestagoal.GoalStatusRunningV0 ||
		!containsGeminiGoalStringV0(receipt.EvidenceRefs, GeminiGoalEvidenceProcessLaunchedV0) {
		t.Fatalf("receipt inesperado: %+v", receipt)
	}
	resultPath := filepath.Join(backend.Control.ProjectWorkDir, "docs", GeminiGoalResultFileNameV0)
	waitForGeminiGoalProcessTestV0(t, func() bool {
		_, err := os.Stat(resultPath)
		return err == nil
	})

	observed, err := backend.ObserveGoalWorkV0(context.Background(), orquestagoal.GoalObservationRequestV0{
		GoalRef:         spec.GoalRef,
		ExternalGoalRef: receipt.ExternalGoalRef,
	})
	if err != nil {
		t.Fatalf("ObserveGoalWorkV0: %v", err)
	}
	if observed.Status != orquestagoal.GoalStatusCompleteV0 ||
		!containsGeminiGoalStringV0(observed.EvidenceRefs, GeminiGoalEvidenceResultReadV0) {
		t.Fatalf("observed inesperado: %+v", observed)
	}
}

func TestGeminiGoalProcessBackendV0ProcesoSinResultadoBloqueaV0(t *testing.T) {
	root := t.TempDir()
	goalRef := "goal-ref-gemini-process-no-result-001"
	backend := geminiGoalProcessBackendForTestV0(
		t,
		root,
		geminiGoalFakeCommandNoResultV0(t, root),
	)
	spec := geminiGoalProcessSpecForTestV0(goalRef)
	if _, err := backend.LaunchGoalWorkV0(context.Background(), spec); err != nil {
		t.Fatalf("LaunchGoalWorkV0: %v", err)
	}
	processRef := backend.loadProcessRefV0(goalRef)
	waitForGeminiGoalProcessTestV0(t, func() bool {
		snapshot, err := backend.ProcessRuntime.SnapshotV0(processRef)
		return err == nil && snapshot.Status == orquestaruntime.ProcessRuntimeStoppedV0
	})

	observed, err := backend.ObserveGoalWorkV0(context.Background(), orquestagoal.GoalObservationRequestV0{
		GoalRef: spec.GoalRef,
	})
	if err != nil {
		t.Fatalf("ObserveGoalWorkV0: %v", err)
	}
	if observed.Status != orquestagoal.GoalStatusBlockedV0 ||
		observed.Summary != ErrGeminiGoalProcessStoppedWithoutResultV0 ||
		!containsGeminiGoalStringV0(observed.EvidenceRefs, GeminiGoalEvidenceProcessStoppedV0) {
		t.Fatalf("observed inesperado: %+v", observed)
	}
}

func TestGeminiGoalProcessBackendV0StopParaProcesoVivoV0(t *testing.T) {
	root := t.TempDir()
	goalRef := "goal-ref-gemini-process-stop-001"
	startedPath := filepath.Join(root, "started.txt")
	backend := geminiGoalProcessBackendForTestV0(
		t,
		root,
		geminiGoalFakeCommandLongRunningV0(t, root, startedPath),
	)
	spec := geminiGoalProcessSpecForTestV0(goalRef)
	if _, err := backend.LaunchGoalWorkV0(context.Background(), spec); err != nil {
		t.Fatalf("LaunchGoalWorkV0: %v", err)
	}
	waitForGeminiGoalProcessTestV0(t, func() bool {
		_, err := os.Stat(startedPath)
		return err == nil
	})

	result, err := backend.StopGeminiGoalV0(context.Background(), GeminiGoalStopRequestV0{
		GoalRef:      goalRef,
		Action:       "stop",
		Reason:       "test_stop",
		Forced:       true,
		EvidenceRefs: []string{"evidence-ref-test-stop"},
	})
	if err != nil {
		t.Fatalf("StopGeminiGoalV0: %v result=%+v", err, result)
	}
	if result.Status != orquestagoal.GoalStatusBlockedV0 ||
		!result.GoalStatusSet ||
		!result.BackendStopped ||
		!containsGeminiGoalStringV0(result.EvidenceRefs, GeminiGoalEvidenceStopCompletedV0) {
		t.Fatalf("stop result inesperado: %+v", result)
	}
}

func TestGeminiGoalProcessBackendV0AdoptaProcesoPersistidoTrasReinicioYLoParaV0(t *testing.T) {
	root := t.TempDir()
	goalRef := "goal-ref-gemini-process-adopt-001"
	startedPath := filepath.Join(root, "started-adopt.txt")
	commandPath := geminiGoalFakeCommandLongRunningV0(t, root, startedPath)
	backend := geminiGoalProcessBackendForTestV0(t, root, commandPath)
	spec := geminiGoalProcessSpecForTestV0(goalRef)
	if _, err := backend.LaunchGoalWorkV0(context.Background(), spec); err != nil {
		t.Fatalf("LaunchGoalWorkV0: %v", err)
	}
	processRef := backend.loadProcessRefV0(goalRef)
	if processRef == "" {
		t.Fatalf("process_ref no persistido en memoria")
	}
	statePath := filepath.Join(backend.Control.RuntimeWorkDir, geminiGoalProcessStateFileNameV0(goalRef))
	if _, err := os.Stat(statePath); err != nil {
		t.Fatalf("estado de proceso no persistido: %v", err)
	}
	waitForGeminiGoalProcessTestV0(t, func() bool {
		_, err := os.Stat(startedPath)
		return err == nil
	})

	restarted := geminiGoalProcessBackendForTestV0(t, root, commandPath)
	if got := restarted.loadProcessRefV0(goalRef); got != "" {
		t.Fatalf("backend reiniciado no debe tener process_ref en memoria: %q", got)
	}
	result, err := restarted.StopGeminiGoalV0(context.Background(), GeminiGoalStopRequestV0{
		GoalRef:      goalRef,
		Action:       "stop",
		Reason:       "test_restart_adoption",
		Forced:       true,
		EvidenceRefs: []string{"evidence-ref-test-restart-adoption"},
	})
	if err != nil {
		t.Fatalf("StopGeminiGoalV0 reiniciado: %v result=%+v", err, result)
	}
	if result.Status != orquestagoal.GoalStatusBlockedV0 ||
		!result.BackendStopped ||
		!containsGeminiGoalStringV0(result.EvidenceRefs, GeminiGoalEvidenceProcessAdoptedV0) ||
		!containsGeminiGoalStringV0(result.EvidenceRefs, GeminiGoalEvidenceStopCompletedV0) {
		t.Fatalf("stop adoptado inesperado: %+v", result)
	}
	waitForGeminiGoalProcessTestV0(t, func() bool {
		snapshot, err := backend.ProcessRuntime.SnapshotV0(processRef)
		return err == nil && snapshot.Status == orquestaruntime.ProcessRuntimeStoppedV0
	})
}

func geminiGoalProcessBackendForTestV0(
	t *testing.T,
	root string,
	commandPath string,
) *GeminiGoalProcessBackendV0 {
	t.Helper()
	projectDir := filepath.Join(root, "project")
	runtimeDir := filepath.Join(root, "runtime")
	return &GeminiGoalProcessBackendV0{
		Control: GeminiGoalBackendV0{
			ProjectWorkDir: projectDir,
			RuntimeWorkDir: runtimeDir,
		},
		Profile: GeminiConnectorProfileV0{
			SchemaVersion:  GeminiConnectorProfileSchemaVersionV0,
			OptIn:          true,
			CommandPath:    commandPath,
			ProjectWorkDir: projectDir,
			RuntimeWorkDir: runtimeDir,
			ApprovalMode:   "auto_edit",
			OutputFormat:   "text",
		},
		ProcessRuntime: orquestaruntime.NewProcessRuntimeConnectorV0(),
	}
}

func geminiGoalProcessSpecForTestV0(goalRef string) orquestagoal.GoalWorkSpecV0 {
	return orquestagoal.GoalWorkSpecV0{
		SchemaVersion: orquestagoal.GoalWorkSpecSchemaV0,
		GoalRef:       goalRef,
		RequestRef:    "request-ref-" + goalRef,
		RunRef:        "run-ref-" + goalRef,
		Objective:     "Ejecutar Gemini goal process fake.",
		DirectorKind:  orquestagoal.GoalDirectorKindRuntimeGoalV0,
		WriteSet: []orquestagoal.GoalWriteScopeV0{{
			Path:    "docs",
			Purpose: "resultado durable",
		}},
		RequiredTests: []orquestagoal.GoalRequiredTestV0{{
			TestRef: "required-test-ref-" + goalRef,
			Command: "fake gemini process test",
		}},
	}
}

func geminiGoalFakeCommandWritesResultV0(t *testing.T, root string, goalRef string) string {
	t.Helper()
	result := orquestagoal.GoalWorkResultV0{
		SchemaVersion: orquestagoal.GoalWorkResultSchemaV0,
		Status:        orquestagoal.GoalStatusCompleteV0,
		GoalRef:       goalRef,
		Summary:       "gemini process complete",
		ArtifactPaths: []string{"docs/orquesta_goal_result_v0.json"},
		MaterializedArtifacts: []orquestagoal.GoalMaterializedArtifactV0{{
			ArtifactRef:  "artifact-ref-" + goalRef,
			Path:         "docs/orquesta_goal_result_v0.json",
			ArtifactType: "goal_result",
			Status:       orquestagoal.GoalMaterializedArtifactStatusValidV0,
			EvidenceRefs: []string{"evidence-ref-gemini-process-result"},
		}},
		Checklist: orquestagoal.GoalWorkChecklistV0{
			ExpectedRefs:  []string{"gemini_process_result"},
			CompletedRefs: []string{"gemini_process_result"},
		},
		RequiredTestResults: []orquestagoal.GoalRequiredTestResultV0{{
			TestRef:      "required-test-ref-" + goalRef,
			Status:       "passed",
			EvidenceRefs: []string{"evidence-ref-gemini-process-test"},
		}},
		EvidenceRefs: []string{"evidence-ref-gemini-process-result"},
	}
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Marshal result: %v", err)
	}
	script := "#!/bin/sh\nset -eu\nprompt=$(cat)\ncase \"$prompt\" in\n  *" + goalRef + "*) ;;\n  *) exit 7 ;;\nesac\nmkdir -p docs\ncat > docs/" + GeminiGoalResultFileNameV0 + " <<'JSON'\n" + string(data) + "\nJSON\n"
	path := filepath.Join(root, "fake-gemini-complete")
	if err := os.WriteFile(path, []byte(script), 0o700); err != nil {
		t.Fatalf("write fake gemini: %v", err)
	}
	return path
}

func geminiGoalFakeCommandNoResultV0(t *testing.T, root string) string {
	t.Helper()
	path := filepath.Join(root, "fake-gemini-no-result")
	script := "#!/bin/sh\nset -eu\ncat >/dev/null\nexit 0\n"
	if err := os.WriteFile(path, []byte(script), 0o700); err != nil {
		t.Fatalf("write fake gemini: %v", err)
	}
	return path
}

func geminiGoalFakeCommandLongRunningV0(t *testing.T, root string, startedPath string) string {
	t.Helper()
	path := filepath.Join(root, "fake-gemini-long-running")
	script := "#!/bin/sh\nset -eu\ncat >/dev/null\nprintf started > " + shellQuoteGeminiGoalProcessTestV0(startedPath) + "\nsleep 30\n"
	if err := os.WriteFile(path, []byte(script), 0o700); err != nil {
		t.Fatalf("write fake gemini long running: %v", err)
	}
	return path
}

func waitForGeminiGoalProcessTestV0(t *testing.T, ready func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if ready() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("timeout esperando proceso Gemini goal")
}

func shellQuoteGeminiGoalProcessTestV0(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}
