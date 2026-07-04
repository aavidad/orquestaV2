package orquestaruntimeclaude

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

func TestClaudeGoalProcessBackendV0LanzaProcesoYObservaResultadoDurableV0(t *testing.T) {
	root := t.TempDir()
	goalRef := "goal-ref-claude-process-complete-001"
	backend := claudeGoalProcessBackendForTestV0(
		t,
		root,
		claudeGoalFakeCommandWritesResultV0(t, root, goalRef),
	)
	spec := claudeGoalProcessSpecForTestV0(goalRef)

	receipt, err := backend.LaunchGoalWorkV0(context.Background(), spec)
	if err != nil {
		t.Fatalf("LaunchGoalWorkV0: %v", err)
	}
	if receipt.Status != orquestagoal.GoalStatusRunningV0 ||
		!containsClaudeGoalStringV0(receipt.EvidenceRefs, ClaudeGoalEvidenceProcessLaunchedV0) {
		t.Fatalf("receipt inesperado: %+v", receipt)
	}
	resultPath := filepath.Join(backend.Control.ProjectWorkDir, "docs", ClaudeGoalResultFileNameV0)
	waitForClaudeGoalProcessTestV0(t, func() bool {
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
		!containsClaudeGoalStringV0(observed.EvidenceRefs, ClaudeGoalEvidenceResultReadV0) {
		t.Fatalf("observed inesperado: %+v", observed)
	}
}

func TestClaudeGoalProcessBackendV0ProcesoSinResultadoBloqueaV0(t *testing.T) {
	root := t.TempDir()
	goalRef := "goal-ref-claude-process-no-result-001"
	backend := claudeGoalProcessBackendForTestV0(
		t,
		root,
		claudeGoalFakeCommandNoResultV0(t, root),
	)
	spec := claudeGoalProcessSpecForTestV0(goalRef)
	if _, err := backend.LaunchGoalWorkV0(context.Background(), spec); err != nil {
		t.Fatalf("LaunchGoalWorkV0: %v", err)
	}
	processRef := backend.loadProcessRefV0(goalRef)
	waitForClaudeGoalProcessTestV0(t, func() bool {
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
		observed.Summary != ErrClaudeGoalProcessStoppedWithoutResultV0 ||
		!containsClaudeGoalStringV0(observed.EvidenceRefs, ClaudeGoalEvidenceProcessStoppedV0) {
		t.Fatalf("observed inesperado: %+v", observed)
	}
}

func TestClaudeGoalProcessBackendV0StopParaProcesoVivoV0(t *testing.T) {
	root := t.TempDir()
	goalRef := "goal-ref-claude-process-stop-001"
	startedPath := filepath.Join(root, "started.txt")
	backend := claudeGoalProcessBackendForTestV0(
		t,
		root,
		claudeGoalFakeCommandLongRunningV0(t, root, startedPath),
	)
	spec := claudeGoalProcessSpecForTestV0(goalRef)
	if _, err := backend.LaunchGoalWorkV0(context.Background(), spec); err != nil {
		t.Fatalf("LaunchGoalWorkV0: %v", err)
	}
	waitForClaudeGoalProcessTestV0(t, func() bool {
		_, err := os.Stat(startedPath)
		return err == nil
	})

	result, err := backend.StopClaudeGoalV0(context.Background(), ClaudeGoalStopRequestV0{
		GoalRef:      goalRef,
		Action:       "stop",
		Reason:       "test_stop",
		Forced:       true,
		EvidenceRefs: []string{"evidence-ref-test-stop"},
	})
	if err != nil {
		t.Fatalf("StopClaudeGoalV0: %v result=%+v", err, result)
	}
	if result.Status != orquestagoal.GoalStatusBlockedV0 ||
		!result.GoalStatusSet ||
		!result.BackendStopped ||
		!containsClaudeGoalStringV0(result.EvidenceRefs, ClaudeGoalEvidenceStopCompletedV0) {
		t.Fatalf("stop result inesperado: %+v", result)
	}
}

func TestClaudeGoalProcessBackendV0AdoptaProcesoPersistidoTrasReinicioYLoParaV0(t *testing.T) {
	root := t.TempDir()
	goalRef := "goal-ref-claude-process-adopt-001"
	startedPath := filepath.Join(root, "started-adopt.txt")
	commandPath := claudeGoalFakeCommandLongRunningV0(t, root, startedPath)
	backend := claudeGoalProcessBackendForTestV0(t, root, commandPath)
	spec := claudeGoalProcessSpecForTestV0(goalRef)
	if _, err := backend.LaunchGoalWorkV0(context.Background(), spec); err != nil {
		t.Fatalf("LaunchGoalWorkV0: %v", err)
	}
	processRef := backend.loadProcessRefV0(goalRef)
	if processRef == "" {
		t.Fatalf("process_ref no persistido en memoria")
	}
	statePath := filepath.Join(backend.Control.RuntimeWorkDir, claudeGoalProcessStateFileNameV0(goalRef))
	if _, err := os.Stat(statePath); err != nil {
		t.Fatalf("estado de proceso no persistido: %v", err)
	}
	waitForClaudeGoalProcessTestV0(t, func() bool {
		_, err := os.Stat(startedPath)
		return err == nil
	})

	restarted := claudeGoalProcessBackendForTestV0(t, root, commandPath)
	if got := restarted.loadProcessRefV0(goalRef); got != "" {
		t.Fatalf("backend reiniciado no debe tener process_ref en memoria: %q", got)
	}
	result, err := restarted.StopClaudeGoalV0(context.Background(), ClaudeGoalStopRequestV0{
		GoalRef:      goalRef,
		Action:       "stop",
		Reason:       "test_restart_adoption",
		Forced:       true,
		EvidenceRefs: []string{"evidence-ref-test-restart-adoption"},
	})
	if err != nil {
		t.Fatalf("StopClaudeGoalV0 reiniciado: %v result=%+v", err, result)
	}
	if result.Status != orquestagoal.GoalStatusBlockedV0 ||
		!result.BackendStopped ||
		!containsClaudeGoalStringV0(result.EvidenceRefs, ClaudeGoalEvidenceProcessAdoptedV0) ||
		!containsClaudeGoalStringV0(result.EvidenceRefs, ClaudeGoalEvidenceStopCompletedV0) {
		t.Fatalf("stop adoptado inesperado: %+v", result)
	}
	waitForClaudeGoalProcessTestV0(t, func() bool {
		snapshot, err := backend.ProcessRuntime.SnapshotV0(processRef)
		return err == nil && snapshot.Status == orquestaruntime.ProcessRuntimeStoppedV0
	})
}

func claudeGoalProcessBackendForTestV0(
	t *testing.T,
	root string,
	commandPath string,
) *ClaudeGoalProcessBackendV0 {
	t.Helper()
	projectDir := filepath.Join(root, "project")
	runtimeDir := filepath.Join(root, "runtime")
	return &ClaudeGoalProcessBackendV0{
		Control: ClaudeGoalBackendV0{
			ProjectWorkDir: projectDir,
			RuntimeWorkDir: runtimeDir,
		},
		Profile: ClaudeConnectorProfileV0{
			SchemaVersion:  ClaudeConnectorProfileSchemaVersionV0,
			OptIn:          true,
			CommandPath:    commandPath,
			ProjectWorkDir: projectDir,
			RuntimeWorkDir: runtimeDir,
			PermissionMode: "bypassPermissions",
			OutputFormat:   "text",
		},
		ProcessRuntime: orquestaruntime.NewProcessRuntimeConnectorV0(),
	}
}

func claudeGoalProcessSpecForTestV0(goalRef string) orquestagoal.GoalWorkSpecV0 {
	return orquestagoal.GoalWorkSpecV0{
		SchemaVersion: orquestagoal.GoalWorkSpecSchemaV0,
		GoalRef:       goalRef,
		RequestRef:    "request-ref-" + goalRef,
		RunRef:        "run-ref-" + goalRef,
		Objective:     "Ejecutar Claude goal process fake.",
		DirectorKind:  orquestagoal.GoalDirectorKindRuntimeGoalV0,
		WriteSet: []orquestagoal.GoalWriteScopeV0{{
			Path:    "docs",
			Purpose: "resultado durable",
		}},
		RequiredTests: []orquestagoal.GoalRequiredTestV0{{
			TestRef: "required-test-ref-" + goalRef,
			Command: "fake claude process test",
		}},
	}
}

func claudeGoalFakeCommandWritesResultV0(t *testing.T, root string, goalRef string) string {
	t.Helper()
	result := orquestagoal.GoalWorkResultV0{
		SchemaVersion: orquestagoal.GoalWorkResultSchemaV0,
		Status:        orquestagoal.GoalStatusCompleteV0,
		GoalRef:       goalRef,
		Summary:       "claude process complete",
		ArtifactPaths: []string{"docs/orquesta_goal_result_v0.json"},
		MaterializedArtifacts: []orquestagoal.GoalMaterializedArtifactV0{{
			ArtifactRef:  "artifact-ref-" + goalRef,
			Path:         "docs/orquesta_goal_result_v0.json",
			ArtifactType: "goal_result",
			Status:       orquestagoal.GoalMaterializedArtifactStatusValidV0,
			EvidenceRefs: []string{"evidence-ref-claude-process-result"},
		}},
		Checklist: orquestagoal.GoalWorkChecklistV0{
			ExpectedRefs:  []string{"claude_process_result"},
			CompletedRefs: []string{"claude_process_result"},
		},
		RequiredTestResults: []orquestagoal.GoalRequiredTestResultV0{{
			TestRef:      "required-test-ref-" + goalRef,
			Status:       "passed",
			EvidenceRefs: []string{"evidence-ref-claude-process-test"},
		}},
		EvidenceRefs: []string{"evidence-ref-claude-process-result"},
	}
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Marshal result: %v", err)
	}
	script := "#!/bin/sh\nset -eu\nprompt=$(cat)\ncase \"$prompt\" in\n  *" + goalRef + "*) ;;\n  *) exit 7 ;;\nesac\nmkdir -p docs\ncat > docs/" + ClaudeGoalResultFileNameV0 + " <<'JSON'\n" + string(data) + "\nJSON\n"
	path := filepath.Join(root, "fake-claude-complete")
	if err := os.WriteFile(path, []byte(script), 0o700); err != nil {
		t.Fatalf("write fake claude: %v", err)
	}
	return path
}

func claudeGoalFakeCommandNoResultV0(t *testing.T, root string) string {
	t.Helper()
	path := filepath.Join(root, "fake-claude-no-result")
	script := "#!/bin/sh\nset -eu\ncat >/dev/null\nexit 0\n"
	if err := os.WriteFile(path, []byte(script), 0o700); err != nil {
		t.Fatalf("write fake claude: %v", err)
	}
	return path
}

func claudeGoalFakeCommandLongRunningV0(t *testing.T, root string, startedPath string) string {
	t.Helper()
	path := filepath.Join(root, "fake-claude-long-running")
	script := "#!/bin/sh\nset -eu\ncat >/dev/null\nprintf started > " + shellQuoteClaudeGoalProcessTestV0(startedPath) + "\nsleep 30\n"
	if err := os.WriteFile(path, []byte(script), 0o700); err != nil {
		t.Fatalf("write fake claude long running: %v", err)
	}
	return path
}

func waitForClaudeGoalProcessTestV0(t *testing.T, ready func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if ready() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("timeout esperando proceso Claude goal")
}

func shellQuoteClaudeGoalProcessTestV0(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}
