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
			ModelAlias:       "gpt-5.5",
			CapacityLevel:    "xhigh",
			ReasoningEffort:  "xhigh",
			QuotaStatus:      DirectorAgentUsageQuotaAvailableV0,
			QuotaRemaining:   123,
			QuotaLimit:       200,
			PromptTokens:     1000,
			CompletionTokens: 500,
			TotalTokens:      1500,
			EvidenceRefs:     []string{"usage-evidence-ref-001"},
		}}},
		DirectorProgressSourceRequestV0{CorrelationID: "corr-agent-usage-stats-001"},
	)

	agent := findDirectorAgentStatsForTestV0(t, stats, agentRef)
	if agent.Usage == nil ||
		agent.Usage.ModelAlias != "gpt-5.5" ||
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
		DirectorProgressSourceRequestV0{CorrelationID: "corr-agent-usage-stats-error-001"},
	)

	if len(stats.Progress.Issues) != 1 ||
		stats.Progress.Issues[0].Code != "agent_usage_source_error" {
		t.Fatalf("issues=%+v", stats.Progress.Issues)
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
