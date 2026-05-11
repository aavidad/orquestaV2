package orquestaappcodexstack

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
	orquestaweb "orquesta/modulos/orquesta-web"
)

func TestCodexStackV0DirectorStatsIncluyeProcesoYProgresoPorPuertos(t *testing.T) {
	runtime := newPendingAckCodexStackRuntimeV0()
	stack := mustBuildCodexStackForTestV0(t, runtime)
	director := postDirectorAPIV0(t, stack)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodGet,
		"/director-stats?run_ref="+url.QueryEscape(director.RunRef)+
			"&include_process_refs=true&include_agent_progress=true&include_agent_usage=true",
		nil,
	)
	stack.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("stats status=%d body=%s", rec.Code, rec.Body.String())
	}
	var page orquestaweb.WebDirectorStatsPageV0
	if err := json.NewDecoder(rec.Body).Decode(&page); err != nil {
		t.Fatalf("decode stats: %v", err)
	}
	if page.ViewModel.Progress.SourceStatus != orquestacionnucleoapp.DirectorProgressSourceLoadedV0 ||
		len(page.ViewModel.Progress.NoSignalAgentRefs) != len(director.StartedAgents) {
		t.Fatalf("progress=%+v director=%+v", page.ViewModel.Progress, director)
	}
	if len(page.ViewModel.Agents) != len(director.StartedAgents) {
		t.Fatalf("agents=%+v director=%+v", page.ViewModel.Agents, director)
	}
	for _, agent := range page.ViewModel.Agents {
		if !agent.CanStop {
			t.Fatalf("agent sin control de parada: %+v", agent)
		}
		if agent.ModelAlias != "gpt-5.5" ||
			agent.QuotaStatus != orquestacionnucleoapp.DirectorAgentUsageQuotaNotConfiguredV0 {
			t.Fatalf("agent sin usage esperado: %+v", agent)
		}
	}
}

func TestCodexStackAgentUsageSourceV0UneMetricasInyectadas(t *testing.T) {
	ctx := context.Background()
	runtime := newPendingAckCodexStackRuntimeV0()
	stack := mustBuildCodexStackForTestV0(t, runtime)
	director := postDirectorAPIV0(t, stack)
	run := mustLoadCodexStackRunForTestV0(t, stack, director.RunRef)
	agentRef := firstStartedDirectorAgentV0(director.StartedAgents)
	if agentRef == "" {
		t.Fatalf("sin agente para usage: %+v", director)
	}
	source := CodexStackAgentUsageSourceV0{
		Store:           stack.Stores.ReceiptStore,
		ModelAlias:      "gpt-5.5",
		ReasoningEffort: "xhigh",
		UsageMetrics: staticCodexStackUsageMetricsSourceV0{
			Metrics: []CodexStackAgentUsageMetricV0{{
				AgentRequestID:   agentRef,
				QuotaStatus:      orquestacionnucleoapp.DirectorAgentUsageQuotaLimitedV0,
				QuotaRemaining:   42,
				QuotaLimit:       100,
				PromptTokens:     1000,
				CompletionTokens: 250,
				TotalTokens:      1250,
				CostMicros:       700,
				EvidenceRefs:     []string{"usage-evidence-ref-stack-001"},
			}},
		},
	}

	stats := orquestacionnucleoapp.BuildDirectorRunStatsWithTelemetryPortsV0(
		ctx,
		run,
		nil,
		nil,
		source,
		orquestacionnucleoapp.DirectorProgressSourceRequestV0{
			CorrelationID: "corr-stack-usage-metrics-001",
		},
	)

	agent := findStackDirectorAgentStatsForTestV0(t, stats, agentRef)
	if agent.Usage == nil ||
		agent.Usage.QuotaStatus != orquestacionnucleoapp.DirectorAgentUsageQuotaLimitedV0 ||
		agent.Usage.TotalTokens != 1250 {
		t.Fatalf("usage=%+v", agent.Usage)
	}
	if stats.UsageSummary == nil ||
		stats.UsageSummary.AgentsObserved != len(director.StartedAgents) ||
		stats.UsageSummary.TotalTokens != 1250 ||
		stats.UsageSummary.QuotaStatus != orquestacionnucleoapp.DirectorAgentUsageQuotaLimitedV0 {
		t.Fatalf("usage_summary=%+v", stats.UsageSummary)
	}
}

func TestCodexStackV0ProgressLoopDetectedProtegeDirectorInicial(t *testing.T) {
	ctx := context.Background()
	runtime := newPendingAckCodexStackRuntimeV0()
	stack := mustBuildCodexStackForTestV0(t, runtime)
	director := postDirectorAPIV0(t, stack)
	if len(director.StartedAgents) == 0 {
		t.Fatalf("sin agentes arrancados: %+v", director)
	}
	// El tramo bajo prueba ya tiene agentes lanzados; dejamos solo el stopper del stack.
	stack.Ports.BatchDispatchers = nil
	stack.Ports.ProgressSource = orquestaruntimecodexdelivery.CodexProgressObservationSourceV0{
		Store:           stack.Stores.ReceiptStore,
		ProcessRegistry: stack.Stores.ProcessRegistry,
		SnapshotSource:  runtime,
		State:           stack.Stores.ProgressState,
		Policy: orquestaruntime.AgentProgressHeartbeatPolicyV0{
			StalledAfterNoProgressTicks: 99,
			LoopAfterRepeatedActions:    1,
		},
	}

	drain, err := stack.DrainRunV0(ctx, DrainRunRequestV0{
		RunRef:               director.RunRef,
		OccurredAt:           "2026-05-10T12:30:00Z",
		CorrelationID:        "corr-app-stack-progress-loop-001",
		MaxBursts:            6,
		MaxStepsPerBurst:     4,
		MaxDispatchesPerWait: 8,
		MaxCommands:          20,
		MaxOutboxPerCycle:    8,
		MaxExternalWaits:     3,
	})
	if err != nil {
		t.Fatalf("DrainRunV0: %v status=%s final=%+v", err, drain.Status, drain.Final)
	}
	run, err := stack.Stores.RunStore.LoadRunV0(ctx, director.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	agentRef := firstStartedDirectorAgentV0(director.StartedAgents)
	if agentRef == "" {
		t.Fatalf("run sin director arrancado: run=%+v drain=%+v", run, drain)
	}
	if !codexStackRefsContainPartV0(run.AgentAssessments, "assessment-ref-agent-progress-report-ref-"+agentRef) ||
		!codexStackRefsContainPartV0(run.AgentAssessments, "#action:ask_director") ||
		!codexStackRefsContainPartV0(run.DirectorQuestions, "question-ref-agent-progress-report-ref-"+agentRef) {
		t.Fatalf("run sin assessment/pregunta esperados para %s: %+v", agentRef, run)
	}
	if codexStackHasRefV0(run.StoppedAgents, agentRef) ||
		codexStackHasRefV0(run.ConfirmedStoppedAgents, agentRef) {
		t.Fatalf("director protegido parado: %+v", run)
	}
	record, err := stack.Stores.ProcessRegistry.ResolveAgentProcessV0(ctx, director.RunRef, agentRef)
	if err != nil {
		t.Fatalf("ResolveAgentProcessV0: %v", err)
	}
	snapshot, err := runtime.SnapshotV0(record.ProcessRef)
	if err != nil {
		t.Fatalf("SnapshotV0: %v", err)
	}
	if snapshot.Status != orquestaruntime.ProcessRuntimeRunningV0 {
		t.Fatalf("snapshot=%+v", snapshot)
	}
	sink := stack.Stores.EventSink.(*orquestacionnucleoapp.InMemoryEventSinkV0)
	if !codexStackRealSmokeHasEventV0(sink.EventsV0(), orquestacoreworkflow.OrchestrationEventAgentWorkAssessedV0) ||
		!codexStackRealSmokeHasEventV0(sink.EventsV0(), orquestacoreworkflow.OrchestrationEventDirectorQuestionRaisedV0) {
		t.Fatalf("falta assessment/pregunta: %+v", sink.EventsV0())
	}
	for _, eventType := range []string{
		orquestacoreworkflow.OrchestrationEventAgentStopRequestedV0,
		orquestacoreworkflow.OrchestrationEventAgentStopConfirmedV0,
	} {
		if codexStackRealSmokeHasEventV0(sink.EventsV0(), eventType) {
			t.Fatalf("evento de parada inesperado %s: %+v", eventType, sink.EventsV0())
		}
	}
}

type staticCodexStackUsageMetricsSourceV0 struct {
	Metrics []CodexStackAgentUsageMetricV0
	Err     error
}

func (source staticCodexStackUsageMetricsSourceV0) BuildCodexStackAgentUsageMetricsV0(
	context.Context,
	CodexStackAgentUsageMetricsRequestV0,
) ([]CodexStackAgentUsageMetricV0, error) {
	if source.Err != nil {
		return nil, source.Err
	}
	return append([]CodexStackAgentUsageMetricV0(nil), source.Metrics...), nil
}

func findStackDirectorAgentStatsForTestV0(
	t *testing.T,
	stats orquestacionnucleoapp.DirectorRunStatsV0,
	agentRef string,
) orquestacionnucleoapp.DirectorAgentStatsV0 {
	t.Helper()
	for _, agent := range stats.Agents {
		if agent.AgentRequestID == agentRef {
			return agent
		}
	}
	t.Fatalf("agent stats no encontrado agent=%s stats=%+v", agentRef, stats.Agents)
	return orquestacionnucleoapp.DirectorAgentStatsV0{}
}

type pendingAckCodexStackRuntimeV0 struct {
	*fakeCodexStackRuntimeV0
}

func newPendingAckCodexStackRuntimeV0() *pendingAckCodexStackRuntimeV0 {
	return &pendingAckCodexStackRuntimeV0{
		fakeCodexStackRuntimeV0: newFakeCodexStackRuntimeV0(),
	}
}

func (runtime *pendingAckCodexStackRuntimeV0) LaunchV0(
	_ context.Context,
	_ orquestaruntime.ProcessRuntimeLaunchRequestV0,
) (orquestaruntime.ProcessRuntimeSnapshotV0, error) {
	runtime.mu.Lock()
	defer runtime.mu.Unlock()
	runtime.next++
	ref := strconv.Itoa(runtime.next)
	snapshot := orquestaruntime.ProcessRuntimeSnapshotV0{
		SchemaVersion: orquestaruntime.ProcessRuntimeConnectorVersionV0,
		ProcessRef:    "process-ref-app-stack-pending-" + ref,
		SessionRef:    "session-ref-app-stack-pending-" + ref,
		LaunchRef:     "launch-ref-app-stack-pending-" + ref,
		Status:        orquestaruntime.ProcessRuntimeRunningV0,
	}
	runtime.snapshots[snapshot.ProcessRef] = snapshot
	return snapshot, nil
}

func firstConfirmedStoppedAgentV0(started []string, stopped []string) string {
	for _, agentRef := range compactStringsV0(started) {
		if codexStackHasRefV0(stopped, agentRef) {
			return agentRef
		}
	}
	return ""
}

func firstStartedDirectorAgentV0(started []string) string {
	for _, agentRef := range compactStringsV0(started) {
		if strings.Contains(agentRef, "-director") {
			return agentRef
		}
	}
	return ""
}

func codexStackHasRefV0(values []string, want string) bool {
	for _, value := range compactStringsV0(values) {
		if value == want {
			return true
		}
	}
	return false
}

func codexStackRefsContainPartV0(values []string, want string) bool {
	for _, value := range compactStringsV0(values) {
		if strings.Contains(value, want) {
			return true
		}
	}
	return false
}
