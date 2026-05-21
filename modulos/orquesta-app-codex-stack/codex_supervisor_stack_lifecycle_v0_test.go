package orquestaappcodexstack

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func TestCodexSupervisorStackLifecycleV0SupervisaRunExistenteSinCanalParaleloV0(t *testing.T) {
	ctx := context.Background()
	runtime := newPendingAckCodexStackRuntimeV0()
	stack := mustBuildCodexStackForTestV0(t, runtime)
	director := postDirectorAPIV0(t, stack)
	if runtime.launchCountV0() != len(director.StartedAgents) {
		t.Fatalf("launches iniciales=%d director=%+v", runtime.launchCountV0(), director)
	}

	lifecycle := CodexSupervisorStackLifecycleV0{
		Stack:  stack,
		RunRef: director.RunRef,
		DrainRequest: DrainRunRequestV0{
			CorrelationID:        "corr-codex-supervisor-stack-lifecycle-001",
			OccurredAt:           "2026-05-18T12:00:00Z",
			MaxBursts:            2,
			MaxStepsPerBurst:     4,
			MaxDispatchesPerWait: 2,
			MaxCommands:          8,
			MaxOutboxPerCycle:    4,
			MaxDecisionCycles:    1,
			MaxExternalWaits:     1,
		},
	}

	result, err := SuperviseCodexV0(ctx, CodexSupervisorDepsV0{AgentLifecycle: lifecycle}, CodexSupervisorCommandV0{
		MaxTicks: 2,
	})
	if err != nil {
		t.Fatalf("SuperviseCodexV0: %v result=%+v", err, result)
	}
	if result.StopReason != CodexSupervisorStopMaxTicksV0 || result.Ticks != 2 {
		t.Fatalf("result=%+v", result)
	}
	if len(result.History) != 2 ||
		result.History[0].Action != "launch" ||
		result.History[1].Action != "continue" {
		t.Fatalf("history=%+v", result.History)
	}
	if result.Last.SessionRef != director.RunRef ||
		result.Last.Status != CodexSupervisorRuntimeRunningV0 ||
		!codexStackRefsContainPartV0(result.Last.EvidenceRefs, "evidence-ref-codex-supervisor-stack-drain") {
		t.Fatalf("last=%+v director=%+v", result.Last, director)
	}
	if runtime.launchCountV0() != len(director.StartedAgents) {
		t.Fatalf("el supervisor relanzo agentes: launches=%d director=%+v", runtime.launchCountV0(), director)
	}
}

func TestCodexSupervisorStackLifecycleV0UsaSupervisorGlobalExistenteV0(t *testing.T) {
	ctx := context.Background()
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	low := postDirectorAPIWithNameV0(t, stack, "app-codex-supervisor-low", "Agenda Supervisor Low")
	high := postDirectorAPIWithNameV0(t, stack, "app-codex-supervisor-high", "Agenda Supervisor High")
	setStackRunPriorityForTestV0(t, stack, low.RunRef, low.AppSpec.Slug, 10)
	setStackRunPriorityForTestV0(t, stack, high.RunRef, high.AppSpec.Slug, 90)

	lifecycle := CodexSupervisorStackLifecycleV0{
		Stack:             stack,
		SupervisorCommand: supervisorCommandForStackTestV0(2),
	}

	snapshot, err := lifecycle.LaunchV0(ctx)
	if err != nil {
		t.Fatalf("LaunchV0: %v snapshot=%+v", err, snapshot)
	}
	if snapshot.Status == CodexSupervisorRuntimeFailedV0 ||
		!codexStackRefsContainPartV0(snapshot.EvidenceRefs, "evidence-ref-codex-supervisor-stack-global") {
		t.Fatalf("snapshot=%+v", snapshot)
	}
	if snapshot.SessionRef != low.RunRef {
		t.Fatalf("session=%s want last execution %s snapshot=%+v", snapshot.SessionRef, low.RunRef, snapshot)
	}
}

func TestCodexStackRunSupervisorAPIV0EmpujaRunExistenteSinRelanzarAgentes(t *testing.T) {
	runtime := newPendingAckCodexStackRuntimeV0()
	stack := mustBuildCodexStackForTestV0(t, runtime)
	director := postDirectorAPIV0(t, stack)
	body := bytes.NewBuffer(nil)
	if err := json.NewEncoder(body).Encode(orquestamcp.MCPRunSupervisorToolInputV0{
		RequestID:     "request-ref-run-supervisor-api-001",
		CorrelationID: "corr-run-supervisor-api-001",
		RunRef:        director.RunRef,
		MaxTicks:      2,
		MaxBursts:     2,
	}); err != nil {
		t.Fatalf("encode: %v", err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v0/runs/supervise", body)
	req.Header.Set("Content-Type", "application/json")

	stack.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result orquestamcp.MCPRunSupervisorToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Estado != orquestamcp.MCPRunSupervisorEstadoOKV0 ||
		result.RunRef != director.RunRef ||
		result.Last.SessionRef != director.RunRef ||
		result.Last.Status != string(CodexSupervisorRuntimeRunningV0) ||
		!codexStackRefsContainPartV0(result.Last.EvidenceRefs, "evidence-ref-codex-supervisor-stack-drain") {
		t.Fatalf("result=%+v director=%+v", result, director)
	}
	if runtime.launchCountV0() != len(director.StartedAgents) {
		t.Fatalf("el endpoint relanzo agentes: launches=%d director=%+v", runtime.launchCountV0(), director)
	}
}

func TestCodexStackRunSupervisorAPIV0ConsumeDecisionFileYArrancaProgramacion(t *testing.T) {
	runtime := newFakeCodexStackRuntimeV0()
	stack := mustBuildCodexStackForTestV0(t, runtime)
	director := postDirectorAPIV0(t, stack)
	if runtime.launchCountV0() != 4 {
		t.Fatalf("launches iniciales=%d want=4", runtime.launchCountV0())
	}
	codexStackWriteDelayedDirectorDecisionsForTestV0(t, stack, director.RunRef)

	body := bytes.NewBuffer(nil)
	if err := json.NewEncoder(body).Encode(orquestamcp.MCPRunSupervisorToolInputV0{
		RequestID:            "request-ref-run-supervisor-api-decisions-001",
		CorrelationID:        "corr-run-supervisor-api-decisions-001",
		RunRef:               director.RunRef,
		MaxTicks:             4,
		MaxBursts:            16,
		MaxStepsPerBurst:     8,
		MaxDispatchesPerWait: 8,
		MaxCommands:          20,
		MaxOutboxPerCycle:    8,
		MaxDecisionCycles:    4,
		MaxExternalWaits:     4,
	}); err != nil {
		t.Fatalf("encode: %v", err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v0/runs/supervise", body)
	req.Header.Set("Content-Type", "application/json")

	stack.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result orquestamcp.MCPRunSupervisorToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	run := mustLoadCodexStackRunForTestV0(t, stack, director.RunRef)
	taskRef := "task-ref-stack-agenda-001"
	agentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(taskRef)
	if result.Estado != orquestamcp.MCPRunSupervisorEstadoOKV0 ||
		result.RunRef != director.RunRef ||
		run.CurrentPhase != orquestacoreworkflow.OrchestrationPhaseProgramacionV0 ||
		!codexStackStringInSetForTestV0(run.Tasks, taskRef) ||
		!codexStackStringInSetForTestV0(run.StartedAgents, agentRef) ||
		runtime.launchCountV0() < 5 {
		t.Fatalf("result=%+v run_phase=%s tasks=%v started=%v launches=%d",
			result,
			run.CurrentPhase,
			run.Tasks,
			run.StartedAgents,
			runtime.launchCountV0(),
		)
	}
}

func TestCodexStackRunSupervisorAPIV0ColaGlobalConsumeDecisionFileYArrancaProgramacion(t *testing.T) {
	runtime := newFakeCodexStackRuntimeV0()
	stack := mustBuildCodexStackForTestV0(t, runtime)
	director := postDirectorAPIV0(t, stack)
	if runtime.launchCountV0() != 4 {
		t.Fatalf("launches iniciales=%d want=4", runtime.launchCountV0())
	}
	codexStackWriteDelayedDirectorDecisionsForTestV0(t, stack, director.RunRef)

	body := bytes.NewBuffer(nil)
	if err := json.NewEncoder(body).Encode(orquestamcp.MCPRunSupervisorToolInputV0{
		RequestID:            "request-ref-run-supervisor-api-queue-decisions-001",
		CorrelationID:        "corr-run-supervisor-api-queue-decisions-001",
		QueueRef:             DefaultRunQueueRefV0,
		MaxTicks:             4,
		MaxRunsPerTick:       1,
		MaxExecutions:        4,
		MaxBursts:            16,
		MaxStepsPerBurst:     8,
		MaxDispatchesPerWait: 8,
		MaxCommands:          20,
		MaxOutboxPerCycle:    8,
		MaxDecisionCycles:    4,
		MaxExternalWaits:     4,
		AllowRepeatedRuns:    true,
	}); err != nil {
		t.Fatalf("encode: %v", err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v0/runs/supervise", body)
	req.Header.Set("Content-Type", "application/json")

	stack.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result orquestamcp.MCPRunSupervisorToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	run := mustLoadCodexStackRunForTestV0(t, stack, director.RunRef)
	taskRef := "task-ref-stack-agenda-001"
	agentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(taskRef)
	if result.Estado != orquestamcp.MCPRunSupervisorEstadoOKV0 ||
		result.RunRef != director.RunRef ||
		run.CurrentPhase != orquestacoreworkflow.OrchestrationPhaseProgramacionV0 ||
		!codexStackStringInSetForTestV0(run.Tasks, taskRef) ||
		!codexStackStringInSetForTestV0(run.StartedAgents, agentRef) ||
		runtime.launchCountV0() < 5 {
		t.Fatalf("result=%+v run_phase=%s tasks=%v started=%v launches=%d",
			result,
			run.CurrentPhase,
			run.Tasks,
			run.StartedAgents,
			runtime.launchCountV0(),
		)
	}
}

func TestCodexStackRunSupervisorAPIV0ColaGlobalConsumeDecisionFileAparecidoTrasACK(t *testing.T) {
	runtime := newPendingAckCodexStackRuntimeV0()
	stack := mustBuildCodexStackForTestV0(t, runtime)
	director := postDirectorAPIV0(t, stack)
	if runtime.launchCountV0() != 4 {
		t.Fatalf("launches iniciales=%d want=4", runtime.launchCountV0())
	}
	for _, descriptor := range codexStackDescriptorsForTestV0(t, stack) {
		if err := writeCodexStackMinimalAckForDescriptorV0(descriptor.AckPath); err != nil {
			t.Fatalf("write ACK %s: %v", descriptor.AgentRef, err)
		}
	}

	first := postRunSupervisorQueueStackV0(t, stack, "late-decision-first")
	run := mustLoadCodexStackRunForTestV0(t, stack, director.RunRef)
	if first.Estado != orquestamcp.MCPRunSupervisorEstadoOKV0 ||
		len(run.PhaseArtifacts) < len(director.StartedAgents) ||
		len(run.Tasks) != 0 {
		t.Fatalf("first=%+v artifacts=%v tasks=%v", first, run.PhaseArtifacts, run.Tasks)
	}

	codexStackWriteDelayedDirectorDecisionsForTestV0(t, stack, director.RunRef)
	second := postRunSupervisorQueueStackV0(t, stack, "late-decision-second")
	run = mustLoadCodexStackRunForTestV0(t, stack, director.RunRef)
	taskRef := "task-ref-stack-agenda-001"
	agentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(taskRef)
	if second.Estado != orquestamcp.MCPRunSupervisorEstadoOKV0 ||
		second.RunRef != director.RunRef ||
		run.CurrentPhase != orquestacoreworkflow.OrchestrationPhaseProgramacionV0 ||
		!codexStackStringInSetForTestV0(run.Tasks, taskRef) ||
		!codexStackStringInSetForTestV0(run.StartedAgents, agentRef) ||
		runtime.launchCountV0() < 5 {
		t.Fatalf("second=%+v run_phase=%s tasks=%v started=%v launches=%d",
			second,
			run.CurrentPhase,
			run.Tasks,
			run.StartedAgents,
			runtime.launchCountV0(),
		)
	}
}

func postRunSupervisorQueueStackV0(
	t *testing.T,
	stack StackV0,
	ref string,
) orquestamcp.MCPRunSupervisorToolResultV0 {
	t.Helper()
	body := bytes.NewBuffer(nil)
	if err := json.NewEncoder(body).Encode(orquestamcp.MCPRunSupervisorToolInputV0{
		RequestID:            "request-ref-run-supervisor-api-" + ref,
		CorrelationID:        "corr-run-supervisor-api-" + ref,
		QueueRef:             DefaultRunQueueRefV0,
		MaxTicks:             4,
		MaxRunsPerTick:       1,
		MaxExecutions:        4,
		MaxBursts:            16,
		MaxStepsPerBurst:     8,
		MaxDispatchesPerWait: 8,
		MaxCommands:          20,
		MaxOutboxPerCycle:    8,
		MaxDecisionCycles:    4,
		MaxExternalWaits:     4,
		AllowRepeatedRuns:    true,
	}); err != nil {
		t.Fatalf("encode: %v", err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v0/runs/supervise", body)
	req.Header.Set("Content-Type", "application/json")

	stack.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result orquestamcp.MCPRunSupervisorToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return result
}

func writeCodexStackMinimalAckForDescriptorV0(path string) error {
	return os.WriteFile(
		path,
		[]byte(`{"schema_version":"codex_agent_ack.v0","status":"completed"}`),
		0o600,
	)
}

func TestCodexSupervisorRuntimeStateFromLoopV0MapeaEstadosDelNucleoV0(t *testing.T) {
	tests := []struct {
		name      string
		status    orquestacionnucleoapp.ProgressiveLoopStatusV0
		runStatus orquestacoreworkflow.OrchestrationRunStatusV0
		want      CodexSupervisorRuntimeStateV0
	}{
		{
			name:   "espera_externa_sigue_vivo",
			status: orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0,
			want:   CodexSupervisorRuntimeRunningV0,
		},
		{
			name:   "quiescente_termina",
			status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
			want:   CodexSupervisorRuntimeDoneV0,
		},
		{
			name:   "necesita_director_se_puede_reempujar",
			status: orquestacionnucleoapp.ProgressiveLoopStatusNeedsDirectorV0,
			want:   CodexSupervisorRuntimeStoppedV0,
		},
		{
			name:   "error_falla",
			status: orquestacionnucleoapp.ProgressiveLoopStatusStopErrorV0,
			want:   CodexSupervisorRuntimeFailedV0,
		},
		{
			name:      "run_cerrada_termina",
			status:    orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0,
			runStatus: orquestacoreworkflow.OrchestrationRunStatusClosedV0,
			want:      CodexSupervisorRuntimeDoneV0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := codexSupervisorRuntimeStateFromLoopV0(tt.status, tt.runStatus); got != tt.want {
				t.Fatalf("got=%s want=%s", got, tt.want)
			}
		})
	}
}
