package orquestaruntimecodexdelivery

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimeworktree "orquesta/modulos/orquesta-runtime-worktree"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func programmingTeamServiceV0(
	store orquestacionnucleoapp.RunStorePortV0,
	sink orquestacionnucleoapp.EventSinkPortV0,
	ledger *orquestacionnucleoapp.InMemoryOutboxLedgerV0,
	receiptStore *InMemoryCodexReceiptDescriptorStoreV0,
	taskStore orquestacionnucleoapp.WorkflowTaskStorePortV0,
	processRegistry orquestacionnucleoapp.AgentProcessRegistryPortV0,
	snapshotSource CodexProcessSnapshotSourcePortV0,
	worktreeStore orquestaruntimeworktree.WorktreeSnapshotStorePortV0,
) orquestacionnucleoapp.ServiceV0 {
	worktreeIgnore := []string{".orquesta-codex-runtime"}
	deliveryProvider := orquestacionnucleoapp.DeliveryCandidateProviderV0{
		Base: orquestacionnucleoapp.WorkflowTaskCandidateProviderV0{
			TaskStore:       taskStore,
			RequestedBy:     "orquesta-programming-team-smoke",
			DefaultCapacity: orquestacoreworkflow.OrchestrationCapacityHighV0,
		},
		DeliverySource: CodexDeliveryObservationSourceV0{
			Store: receiptStore,
			WorktreeVerifier: CodexReceiptWorktreeVerifierV0{
				SnapshotStore:  worktreeStore,
				IgnorePrefixes: worktreeIgnore,
			},
		},
		RequestedBy: "orquesta-programming-team-smoke",
	}
	return orquestacionnucleoapp.ServiceV0{
		RunStore:  store,
		EventSink: sink,
		CandidateProvider: orquestacionnucleoapp.ProgressSupervisionCandidateProviderV0{
			Base: deliveryProvider,
			ProgressSource: CodexProgressObservationSourceV0{
				Store:           receiptStore,
				ProcessRegistry: processRegistry,
				SnapshotSource:  snapshotSource,
				State:           NewInMemoryCodexProgressStateStoreV0(),
				Policy: orquestaruntime.AgentProgressHeartbeatPolicyV0{
					StalledAfterNoProgressTicks: 12,
					LoopAfterRepeatedActions:    36,
				},
			},
			RequestedBy: "orquesta-programming-team-smoke",
		},
		OutboxLedger:      ledger,
		MaxCommands:       12,
		MaxOutboxPerCycle: 6,
	}
}

func programmingTeamBatchDispatchersV0(
	store orquestacionnucleoapp.RunStorePortV0,
	sink orquestacionnucleoapp.EventSinkPortV0,
	ledger *orquestacionnucleoapp.InMemoryOutboxLedgerV0,
	receiptStore *InMemoryCodexReceiptDescriptorStoreV0,
	processRegistry orquestacionnucleoapp.AgentProcessRegistryPortV0,
	processRuntime *orquestaruntime.ProcessRuntimeConnectorV0,
	cfg codexRealSmokeConfigV0,
	tasks []programmingTeamTaskV0,
	worktreeStore orquestaruntimeworktree.WorktreeSnapshotStorePortV0,
) []orquestacionnucleoapp.OutboxBatchDispatcherBindingV0 {
	worktreeIgnore := []string{".orquesta-codex-runtime"}
	specResolver := CodexReceiptRecordingSpecResolverV0{
		Inner: programmingTeamCodexSpecResolverV0{
			Config: cfg,
			Tasks:  tasks,
		},
		Recorder: receiptStore,
		AckPathResolver: AgentScopedCodexReceiptAckPathResolverV0{
			BaseDir:        cfg.RuntimeWorkDir,
			ProjectWorkDir: cfg.ProjectWorkDir,
		},
		WorktreeBaselineRecorder: CodexReceiptWorktreeBaselineRecorderV0{
			SnapshotStore:  worktreeStore,
			IgnorePrefixes: worktreeIgnore,
		},
	}
	return []orquestacionnucleoapp.OutboxBatchDispatcherBindingV0{{
		TargetPort: orquestacoreworkflow.OutboxTargetAgentLauncherV0,
		MaxReady:   2,
		Reader:     ledger,
		Claimer:    ledger,
		Executor: orquestacionnucleoapp.ExternalProcessAgentBatchExecutorV0{
			RunStore:        store,
			EventSink:       sink,
			SpecResolver:    specResolver,
			Runtime:         processRuntime,
			ProcessStopper:  processRuntime,
			ProcessRegistry: processRegistry,
			MaxConcurrency:  2,
			OccurredAt:      "2026-05-10T00:10:00Z",
			RequestedBy:     "orquesta-programming-team-smoke",
			EvidenceRefs:    []string{"evidence-ref-programming-team-agent"},
		},
		Acker: ledger,
	}}
}

func programmingTeamRunForTestV0(
	t *testing.T,
	runRef string,
	tasks []programmingTeamTaskV0,
) orquestacoreworkflow.OrchestrationRunV0 {
	t.Helper()
	run := codexDeliveryLoopRunForTestV0(t, runRef, tasks[0].TaskRef)
	for _, task := range tasks[1:] {
		run.Tasks = append(run.Tasks, task.TaskRef)
	}
	if issues := orquestacoreworkflow.ValidateOrchestrationRunV0(run); len(issues) > 0 {
		t.Fatalf("run programming team invalido: %+v", issues)
	}
	return run
}

func programmingTeamWorkflowTasksV0(
	runRef string,
	tasks []programmingTeamTaskV0,
) []orquestacoreworkflow.WorkflowTaskV0 {
	workflowTasks := make([]orquestacoreworkflow.WorkflowTaskV0, 0, len(tasks))
	for _, task := range tasks {
		workflowTasks = append(workflowTasks, orquestacoreworkflow.WorkflowTaskV0{
			SchemaVersion: orquestacoreworkflow.WorkflowTaskSchemaVersionV0,
			TaskID:        task.TaskRef,
			RunID:         runRef,
			PhaseID:       orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
			Title:         task.Title,
			Summary:       "Microtarea acotada para la app de agenda.",
			WriteSet:      task.WriteSet,
			AcceptanceCriteria: []string{
				"Solo se modifica el write-set declarado.",
				"El resultado queda listo para revision.",
				"Cada fichero queda por debajo de 300 lineas.",
			},
			FunctionContractRefs: []orquestacoreworkflow.WorkflowFunctionContractRefV0{
				{ContractRef: "contract:function:agenda-work:v0", FunctionName: "NewWorkflowTaskV0"},
			},
		})
	}
	return workflowTasks
}

func programmingTeamVerifyProjectV0(
	t *testing.T,
	projectDir string,
) {
	t.Helper()
	for _, path := range []string{
		"internal/api/handler.go",
		"internal/api/handler_test.go",
		"server/main.go",
		"web/index.html",
		"README.md",
	} {
		programmingTeamVerifyProjectFileV0(t, projectDir, path)
	}
	programmingTeamVerifyAPIQualityV0(t, projectDir)
	programmingTeamVerifyWebQualityV0(t, projectDir)
	programmingTeamVerifyDocsQualityV0(t, projectDir)
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "test", "./...")
	cmd.Dir = projectDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go test generated project: %v\n%s", err, string(output))
	}
}

func programmingTeamVerifyProjectFileV0(t *testing.T, projectDir string, path string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(projectDir, path))
	if err != nil {
		t.Fatalf("artifact missing %s: %v", path, err)
	}
	if len(strings.TrimSpace(string(data))) < 80 {
		t.Fatalf("artifact demasiado pequeno %s", path)
	}
	if countLinesV0(string(data)) > 300 {
		t.Fatalf("artifact demasiado grande %s", path)
	}
}

func programmingTeamVerifyAPIQualityV0(t *testing.T, projectDir string) {
	t.Helper()
	handler := programmingTeamReadProjectTextV0(t, projectDir, "internal/api/handler.go")
	testFile := programmingTeamReadProjectTextV0(t, projectDir, "internal/api/handler_test.go")
	mainFile := programmingTeamReadProjectTextV0(t, projectDir, "server/main.go")
	for path, value := range map[string]string{
		"internal/api/handler.go":      handler,
		"internal/api/handler_test.go": testFile,
		"server/main.go":               mainFile,
	} {
		if strings.Contains(strings.ToLower(value), "sqlite") ||
			strings.Contains(strings.ToLower(value), "postgres") {
			t.Fatalf("artifact %s elige persistencia concreta", path)
		}
	}
	if !programmingTeamHandlerLooksRESTV0(handler) {
		t.Fatalf("handler.go no expone API REST reconocible")
	}
	if !programmingTeamGeneratedAPIHasDescriptionV0(handler, testFile) {
		t.Fatalf("API generada no conserva description en codigo y tests")
	}
	if !strings.Contains(testFile, "func Test") {
		t.Fatalf("handler_test.go no contiene tests generados")
	}
	if !strings.Contains(mainFile, "package main") || !strings.Contains(mainFile, "ListenAndServe") {
		t.Fatalf("server/main.go no arranca servidor HTTP")
	}
}

func programmingTeamVerifyWebQualityV0(t *testing.T, projectDir string) {
	t.Helper()
	web := strings.ToLower(programmingTeamReadProjectTextV0(t, projectDir, "web/index.html"))
	for _, want := range []string{"<html", "lang=", "es", "en", "agenda", "event", "description"} {
		if !strings.Contains(web, want) {
			t.Fatalf("web/index.html no contiene %s", want)
		}
	}
}

func programmingTeamHandlerLooksRESTV0(handler string) bool {
	return strings.Contains(handler, "net/http") &&
		strings.Contains(handler, "/health") &&
		strings.Contains(handler, "/events") &&
		programmingTeamHandlerHasMethodV0(handler, "GET", "http.MethodGet") &&
		programmingTeamHandlerHasMethodV0(handler, "POST", "http.MethodPost")
}

func programmingTeamHandlerHasMethodV0(handler string, method string, goConst string) bool {
	return strings.Contains(handler, goConst) ||
		strings.Contains(handler, method+" /")
}

func programmingTeamGeneratedAPIHasDescriptionV0(handler string, testFile string) bool {
	handler = strings.ToLower(handler)
	testFile = strings.ToLower(testFile)
	return strings.Contains(handler, "description") &&
		strings.Contains(handler, "`json:\"description") &&
		strings.Contains(testFile, "description")
}

func programmingTeamVerifyDocsQualityV0(t *testing.T, projectDir string) {
	t.Helper()
	readme := strings.ToLower(programmingTeamReadProjectTextV0(t, projectDir, "README.md"))
	for _, want := range []string{"agenda", "api", "web", "description"} {
		if !strings.Contains(readme, want) {
			t.Fatalf("README.md no documenta %s", want)
		}
	}
}

func programmingTeamReadProjectTextV0(t *testing.T, projectDir string, path string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(projectDir, path))
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

func countLinesV0(value string) int {
	value = strings.TrimRight(value, "\n")
	if value == "" {
		return 0
	}
	return strings.Count(value, "\n") + 1
}
