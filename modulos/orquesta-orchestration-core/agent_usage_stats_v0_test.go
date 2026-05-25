package orquestacionnucleoapp

import (
	"context"
	"errors"
	"testing"
)

func TestBuildDirectorRunStatsWithTelemetryPortsV0IncluyeUsoDeAgente(t *testing.T) {
	run := mustActiveProgrammingRunV0(t, "run-agent-usage-stats-001")
	agentRef := "agent-ref-usage-stats-001"
	run.Agents = []string{agentRef}
	run.StartedAgents = []string{agentRef}

	stats := BuildDirectorRunStatsWithTelemetryPortsV0(
		context.Background(),
		run,
		nil,
		nil,
		staticAgentUsageStatsSourceV0{Observations: []AgentUsageStatsObservationV0{{
			AgentRequestID:   agentRef,
			RuntimeKind:      "codex",
			ConnectorRef:     "connector-ref-codex-001",
			ProfileRef:       "profile-ref-premium-001",
			CapacityLevel:    "xhigh",
			QuotaStatus:      DirectorAgentUsageQuotaAvailableV0,
			QuotaRemaining:   123,
			QuotaLimit:       200,
			PromptTokens:     1000,
			CompletionTokens: 500,
			TotalTokens:      1500,
			EvidenceRefs:     []string{"usage-evidence-ref-001"},
		}}},
		DirectorProgressSourceRequestV0{
			CorrelationID:     "corr-agent-usage-stats-001",
			IncludeAgentUsage: true,
		},
	)

	agent := findDirectorAgentStatsForTestV0(t, stats, agentRef)
	if agent.Usage == nil ||
		agent.Usage.QuotaRemaining != 123 ||
		agent.Usage.TotalTokens != 1500 {
		t.Fatalf("usage=%+v", agent.Usage)
	}
	if stats.UsageSummary == nil ||
		stats.UsageSummary.AgentsObserved != 1 ||
		stats.UsageSummary.QuotaStatus != DirectorAgentUsageQuotaAvailableV0 ||
		stats.UsageSummary.TotalTokens != 1500 {
		t.Fatalf("usage_summary=%+v", stats.UsageSummary)
	}
}

func TestBuildDirectorRunStatsWithTelemetryPortsV0ReportaErrorDeUso(t *testing.T) {
	run := mustActiveProgrammingRunV0(t, "run-agent-usage-stats-error-001")

	stats := BuildDirectorRunStatsWithTelemetryPortsV0(
		context.Background(),
		run,
		nil,
		nil,
		staticAgentUsageStatsSourceV0{Err: errors.New("usage unavailable")},
		DirectorProgressSourceRequestV0{
			CorrelationID:     "corr-agent-usage-stats-error-001",
			IncludeAgentUsage: true,
		},
	)

	if len(stats.Progress.Issues) != 1 ||
		stats.Progress.Issues[0].Code != "agent_usage_source_error" {
		t.Fatalf("issues=%+v", stats.Progress.Issues)
	}
}

func TestBuildDirectorRunStatsWithTelemetryPortsV0NoConsultaUsoSinOptIn(t *testing.T) {
	run := mustActiveProgrammingRunV0(t, "run-agent-usage-stats-no-opt-in-001")
	usageSource := &countingAgentUsageStatsSourceV0{}

	stats := BuildDirectorRunStatsWithTelemetryPortsV0(
		context.Background(),
		run,
		nil,
		nil,
		usageSource,
		DirectorProgressSourceRequestV0{CorrelationID: "corr-agent-usage-stats-no-opt-in-001"},
	)

	if usageSource.Calls != 0 ||
		stats.UsageSummary != nil ||
		len(stats.Progress.Issues) != 0 {
		t.Fatalf("calls=%d stats=%+v", usageSource.Calls, stats)
	}
}

func TestBuildDirectorRunStatsWithTelemetryPortsV0ReportaUsoPedidoSinFuente(t *testing.T) {
	run := mustActiveProgrammingRunV0(t, "run-agent-usage-stats-missing-source-001")

	stats := BuildDirectorRunStatsWithTelemetryPortsV0(
		context.Background(),
		run,
		nil,
		nil,
		nil,
		DirectorProgressSourceRequestV0{
			CorrelationID:     "corr-agent-usage-stats-missing-source-001",
			IncludeAgentUsage: true,
		},
	)

	if len(stats.Progress.Issues) != 1 ||
		stats.Progress.Issues[0].Code != "agent_usage_source_not_configured" ||
		stats.UsageSummary != nil {
		t.Fatalf("stats=%+v", stats)
	}
}

func TestApplyDirectorAgentUsageStatsV0IgnoraUsoFueraDelRun(t *testing.T) {
	run := mustActiveProgrammingRunV0(t, "run-agent-usage-stats-scope-001")
	run.Agents = []string{"agent-ref-usage-in-scope-001"}
	run.StartedAgents = []string{"agent-ref-usage-in-scope-001"}
	stats := BuildDirectorRunStatsV0(run)

	ApplyDirectorAgentUsageStatsV0(&stats, []AgentUsageStatsObservationV0{
		{
			AgentRequestID: "agent-ref-usage-in-scope-001",
			QuotaStatus:    DirectorAgentUsageQuotaAvailableV0,
			TotalTokens:    100,
			EvidenceRefs:   []string{"usage-evidence-ref-in-scope"},
		},
		{
			AgentRequestID: "agent-ref-usage-out-of-scope-001",
			QuotaStatus:    DirectorAgentUsageQuotaExhaustedV0,
			TotalTokens:    9000,
			EvidenceRefs:   []string{"usage-evidence-ref-out-of-scope"},
		},
	})

	agent := findDirectorAgentStatsForTestV0(t, stats, "agent-ref-usage-in-scope-001")
	if agent.Usage == nil || agent.Usage.TotalTokens != 100 {
		t.Fatalf("agent usage=%+v", agent.Usage)
	}
	if stats.UsageSummary == nil ||
		stats.UsageSummary.AgentsObserved != 1 ||
		stats.UsageSummary.TotalTokens != 100 ||
		stats.UsageSummary.QuotaStatus != DirectorAgentUsageQuotaAvailableV0 {
		t.Fatalf("usage_summary=%+v", stats.UsageSummary)
	}
}

type staticAgentUsageStatsSourceV0 struct {
	Observations []AgentUsageStatsObservationV0
	Err          error
}

func (source staticAgentUsageStatsSourceV0) BuildAgentUsageStatsV0(
	context.Context,
	AgentUsageStatsRequestV0,
) ([]AgentUsageStatsObservationV0, error) {
	if source.Err != nil {
		return nil, source.Err
	}
	return source.Observations, nil
}

type countingAgentUsageStatsSourceV0 struct {
	Calls int
}

func (source *countingAgentUsageStatsSourceV0) BuildAgentUsageStatsV0(
	context.Context,
	AgentUsageStatsRequestV0,
) ([]AgentUsageStatsObservationV0, error) {
	source.Calls++
	return nil, nil
}
