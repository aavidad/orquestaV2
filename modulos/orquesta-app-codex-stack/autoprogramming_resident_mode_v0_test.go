package orquestaappcodexstack

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestamcp "orquesta/modulos/orquesta-mcp"
)

func TestAutoprogrammingResidentModeV0NormalizaBacklogDurableV0(t *testing.T) {
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	executor := NewCodexStackRunSupervisorExecutorV0(&stack)

	result, err := executor.Execute(context.Background(), orquestamcp.MCPRunSupervisorToolInputV0{
		RequestID:     "request-autoprogramming-resident-001",
		CorrelationID: "corr-autoprogramming-resident-001",
		ResidentMode:  true,
		MaxTicks:      1,
	})
	if err != nil {
		t.Fatalf("Execute resident: %v", err)
	}
	if result.Estado != orquestamcp.MCPRunSupervisorEstadoOKV0 ||
		result.StopReason == "" ||
		!codexStackStringInSetForTestV0(result.EvidenceRefs, "evidence-ref-autoprogramming-resident-mode") {
		t.Fatalf("result=%+v", result)
	}
}

func TestAutoprogrammingResidentModeV0TomaRunPreparadaDeColaV0(t *testing.T) {
	runtime := newFakeCodexStackRuntimeV0()
	stack := mustBuildCodexStackForTestV0(t, runtime)
	prepared := postAutoprogrammingPrepareRunStackV0(t, stack, orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0{
		RequestID:              "request-autoprogramming-resident-queue-001",
		CorrelationID:          "corr-autoprogramming-resident-queue-001",
		AutoprogrammingRequest: autoprogrammingBridgeRequestForTestV0(),
	})

	result := postRunSupervisorStackV0(t, stack, orquestamcp.MCPRunSupervisorToolInputV0{
		RequestID:     "request-autoprogramming-resident-supervise-001",
		CorrelationID: "corr-autoprogramming-resident-queue-001",
		ResidentMode:  true,
		MaxTicks:      1,
	})
	if result.Estado != orquestamcp.MCPRunSupervisorEstadoOKV0 ||
		result.RunRef != prepared.RunRef ||
		runtime.launchCountV0() != 1 ||
		!codexStackStringInSetForTestV0(result.EvidenceRefs, "evidence-ref-autoprogramming-resident-backlog-queue") {
		t.Fatalf("result=%+v prepared=%+v launches=%d", result, prepared, runtime.launchCountV0())
	}
}

func TestAutoprogrammingResidentModeV0TomaAutomejoraAutoPreparadaCuandoEstaParadoV0(t *testing.T) {
	runtime := newFakeCodexStackRuntimeV0()
	stack := mustBuildCodexStackForTestV0(t, runtime)
	body := bytes.NewBuffer(nil)
	if err := json.NewEncoder(body).Encode(orquestamcp.MCPAutoprogrammingSelfImprovementToolInputV0{
		RequestID:      "request-autoprogramming-resident-self-improvement-queue-001",
		CorrelationID:  "corr-autoprogramming-resident-self-improvement-queue-001",
		AutoPrepareRun: true,
		Proposal: orquestaautoprogramming.AutoprogrammingSelfImprovementProposalV0{
			ProjectRef:        "project-ref-autoprogramming-resident-self-improvement",
			WorktreeRef:       "worktree-ref-autoprogramming-resident-self-improvement",
			WorktreeIsolated:  true,
			BranchRef:         "branch-ref-autoprogramming-resident-self-improvement",
			ObservedBy:        "orquesta-autoprogramming-resident",
			FailureSummary:    "Mejora reutilizable detectada mientras el residente esta sin trabajo principal.",
			SuggestedArea:     "app-codex-stack",
			SuggestedWriteSet: []string{"modulos/orquesta-app-codex-stack/autoprogramming_resident_mode_v0.go"},
			RequiredTests:     []string{"go test -count=1 ./modulos/orquesta-app-codex-stack -run TestAutoprogrammingResidentModeV0"},
		},
	}); err != nil {
		t.Fatalf("encode self-improvement: %v", err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v0/autoprogramming/self-improvement", body)
	req.Header.Set("Content-Type", "application/json")
	stack.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("self-improvement status=%d body=%s", rec.Code, rec.Body.String())
	}
	var self orquestamcp.MCPAutoprogrammingSelfImprovementToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&self); err != nil {
		t.Fatalf("decode self-improvement: %v", err)
	}
	if self.PreparedRun == nil || !self.PreparedRun.Accepted || self.PreparedRun.RunRef == "" {
		t.Fatalf("self-improvement result=%+v", self)
	}

	result := postRunSupervisorStackV0(t, stack, orquestamcp.MCPRunSupervisorToolInputV0{
		RequestID:      "request-autoprogramming-resident-idle-picks-self-improvement-001",
		CorrelationID:  "corr-autoprogramming-resident-self-improvement-queue-001",
		ResidentMode:   true,
		MaxTicks:       1,
		MaxRunsPerTick: 1,
	})
	if result.Estado != orquestamcp.MCPRunSupervisorEstadoOKV0 ||
		result.RunRef != self.PreparedRun.RunRef ||
		runtime.launchCountV0() != 1 ||
		!codexStackStringInSetForTestV0(result.EvidenceRefs, "evidence-ref-autoprogramming-resident-backlog-queue") {
		t.Fatalf("result=%+v self=%+v launches=%d", result, self, runtime.launchCountV0())
	}
}

func TestAutoprogrammingResidentModeV0SigueBacklogTrasEjecucionDoneV0(t *testing.T) {
	snapshot := autoprogrammingResidentBacklogSnapshotV0(CodexSupervisorRuntimeSnapshotV0{
		Status: CodexSupervisorRuntimeDoneV0,
		EvidenceRefs: []string{
			"evidence-ref-codex-supervisor-stack-global",
			"max_executions",
			"quiescent",
		},
	})
	if snapshot.Status != CodexSupervisorRuntimeRunningV0 ||
		!codexStackStringInSetForTestV0(snapshot.EvidenceRefs, autoprogrammingResidentBacklogContinuesEvidenceV0) {
		t.Fatalf("snapshot=%+v", snapshot)
	}
	drained := autoprogrammingResidentBacklogSnapshotV0(CodexSupervisorRuntimeSnapshotV0{
		Status: CodexSupervisorRuntimeDoneV0,
		EvidenceRefs: []string{
			"evidence-ref-codex-supervisor-stack-global",
			"no_execution",
		},
	})
	if drained.Status != CodexSupervisorRuntimeDoneV0 {
		t.Fatalf("drained=%+v", drained)
	}
}

func TestAutoprogrammingResidentModeV0BloqueoCreaReparacionDurableV0(t *testing.T) {
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	prepared := postAutoprogrammingPrepareRunStackV0(t, stack, orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0{
		RequestID:              "request-autoprogramming-resident-source-001",
		CorrelationID:          "corr-autoprogramming-resident-selfrepair-001",
		AutoprogrammingRequest: autoprogrammingBridgeRequestForTestV0(),
	})
	result := orquestamcp.MCPRunSupervisorToolResultV0{
		Estado:  orquestamcp.MCPRunSupervisorEstadoOKV0,
		RunRef:  prepared.RunRef,
		Last:    orquestamcp.MCPRunSupervisorSnapshotV0{Status: string(CodexSupervisorRuntimeStoppedV0)},
		Errores: []orquestamcp.MCPValidationIssueV0{},
		EvidenceRefs: []string{
			"evidence-ref-codex-supervisor-stack-drain-blocked",
			"quality-gate-ref-required-tests-evidence-missing",
		},
	}

	result = maybePrepareAutoprogrammingResidentSelfRepairV0(
		context.Background(),
		orquestamcp.MCPRunSupervisorToolInputV0{
			RequestID:     "request-autoprogramming-resident-selfrepair-001",
			CorrelationID: "corr-autoprogramming-resident-selfrepair-001",
			ResidentMode:  true,
		},
		stack,
		CodexSupervisorResultV0{Last: CodexSupervisorRuntimeSnapshotV0{Status: CodexSupervisorRuntimeStoppedV0}},
		result,
	)

	if len(result.RepairRunRefs) != 1 ||
		!codexStackStringInSetForTestV0(result.EvidenceRefs, autoprogrammingResidentRepairEvidenceV0) ||
		!codexStackStringInSetForTestV0(result.NextActions, "continue_resident_backlog_after_repair_followup") {
		t.Fatalf("result=%+v", result)
	}
	run, err := stack.Ports.RunStore.LoadRunV0(context.Background(), result.RepairRunRefs[0])
	if err != nil {
		t.Fatalf("LoadRunV0 repair: %v", err)
	}
	tasks, err := stack.Ports.DirectorTaskStore.LoadWorkflowTasksV0(context.Background(), run.RunID, run.Tasks)
	if err != nil {
		t.Fatalf("LoadWorkflowTasksV0 repair: %v", err)
	}
	if len(tasks) != 1 ||
		tasks[0].WriteSet[0] != "modulos/orquesta-app-codex-stack/autoprogramming_bridge_v0.go" ||
		tasks[0].RequiredTests[0] != "go test -count=1 ./modulos/orquesta-app-codex-stack -run TestPrepareAutoprogrammingRunV0" {
		t.Fatalf("repair run=%+v tasks=%+v", run, tasks)
	}
}

func TestAutoprogrammingResidentModeV0BloqueoDeReparacionNoCreaSegundaReparacionV0(t *testing.T) {
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	primary := postAutoprogrammingPrepareRunStackV0(t, stack, orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0{
		RequestID:              "request-autoprogramming-resident-primary-loop-001",
		CorrelationID:          "corr-autoprogramming-resident-loopguard-001",
		AutoprogrammingRequest: autoprogrammingBridgeRequestForTestV0(),
	})
	primaryResult := orquestamcp.MCPRunSupervisorToolResultV0{
		Estado:       orquestamcp.MCPRunSupervisorEstadoOKV0,
		RunRef:       primary.RunRef,
		Last:         orquestamcp.MCPRunSupervisorSnapshotV0{Status: string(CodexSupervisorRuntimeStoppedV0)},
		EvidenceRefs: []string{"quality-gate-ref-required-tests-evidence-missing"},
	}
	primaryResult = maybePrepareAutoprogrammingResidentSelfRepairV0(
		context.Background(),
		orquestamcp.MCPRunSupervisorToolInputV0{
			RequestID:     "request-autoprogramming-resident-primary-repair-001",
			CorrelationID: "corr-autoprogramming-resident-loopguard-001",
			ResidentMode:  true,
		},
		stack,
		CodexSupervisorResultV0{Last: CodexSupervisorRuntimeSnapshotV0{Status: CodexSupervisorRuntimeStoppedV0}},
		primaryResult,
	)
	if len(primaryResult.RepairRunRefs) != 1 {
		t.Fatalf("primary_result=%+v", primaryResult)
	}

	repairResult := maybePrepareAutoprogrammingResidentSelfRepairV0(
		context.Background(),
		orquestamcp.MCPRunSupervisorToolInputV0{
			RequestID:     "request-autoprogramming-resident-repair-loop-001",
			CorrelationID: "corr-autoprogramming-resident-loopguard-001",
			ResidentMode:  true,
		},
		stack,
		CodexSupervisorResultV0{Last: CodexSupervisorRuntimeSnapshotV0{Status: CodexSupervisorRuntimeStoppedV0}},
		orquestamcp.MCPRunSupervisorToolResultV0{
			Estado:       orquestamcp.MCPRunSupervisorEstadoOKV0,
			RunRef:       primaryResult.RepairRunRefs[0],
			Last:         orquestamcp.MCPRunSupervisorSnapshotV0{Status: string(CodexSupervisorRuntimeStoppedV0)},
			EvidenceRefs: []string{"quality-gate-ref-required-tests-evidence-missing"},
		},
	)

	if len(repairResult.RepairRunRefs) != 0 ||
		!codexStackStringInSetForTestV0(repairResult.EvidenceRefs, autoprogrammingResidentLoopGuardEvidenceV0) ||
		!codexStackStringInSetForTestV0(repairResult.NextActions, "ask_human_review_or_director_before_new_repair") {
		t.Fatalf("repair_result=%+v", repairResult)
	}
}

func TestAutoprogrammingResidentModeV0SinBloqueoNoCreaReparacionV0(t *testing.T) {
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	result := maybePrepareAutoprogrammingResidentSelfRepairV0(
		context.Background(),
		orquestamcp.MCPRunSupervisorToolInputV0{ResidentMode: true},
		stack,
		CodexSupervisorResultV0{Last: CodexSupervisorRuntimeSnapshotV0{Status: CodexSupervisorRuntimeDoneV0}},
		orquestamcp.MCPRunSupervisorToolResultV0{
			Estado:       orquestamcp.MCPRunSupervisorEstadoOKV0,
			RunRef:       "run-ref-done",
			Errores:      []orquestamcp.MCPValidationIssueV0{},
			EvidenceRefs: []string{"evidence-ref-autoprogramming-resident-mode"},
		},
	)
	if len(result.RepairRunRefs) != 0 ||
		!codexStackStringInSetForTestV0(result.EvidenceRefs, autoprogrammingResidentNoActionV0) {
		t.Fatalf("result=%+v", result)
	}
}

func TestNormalizeAutoprogrammingResidentInputV0DespachaLoteSinEsperarAgentesLargos(t *testing.T) {
	input := normalizeAutoprogrammingResidentInputV0(
		orquestamcp.MCPRunSupervisorToolInputV0{
			ResidentMode: true,
		},
		StackV0{},
	)

	if input.MaxRunsPerTick != 10 {
		t.Fatalf("max_runs_per_tick=%d want 10", input.MaxRunsPerTick)
	}
	if input.MaxExternalWaits != 1 {
		t.Fatalf("max_external_waits=%d want 1", input.MaxExternalWaits)
	}
	if input.MaxDispatchesPerWait != 10 {
		t.Fatalf("max_dispatches_per_wait=%d want 10", input.MaxDispatchesPerWait)
	}
	if input.MaxOutboxPerCycle != 10 {
		t.Fatalf("max_outbox_per_cycle=%d want 10", input.MaxOutboxPerCycle)
	}
	if !input.AllowRepeatedRuns {
		t.Fatalf("allow_repeated_runs=false want true")
	}
}

func TestNormalizeAutoprogrammingResidentInputV0RespetaCapacidadExplicita(t *testing.T) {
	input := normalizeAutoprogrammingResidentInputV0(
		orquestamcp.MCPRunSupervisorToolInputV0{
			ResidentMode:     true,
			MaxRunsPerTick:   6,
			MaxExternalWaits: 3,
		},
		StackV0{},
	)

	if input.MaxRunsPerTick != 6 {
		t.Fatalf("max_runs_per_tick=%d want 6", input.MaxRunsPerTick)
	}
	if input.MaxExternalWaits != 3 {
		t.Fatalf("max_external_waits=%d want 3", input.MaxExternalWaits)
	}
}
