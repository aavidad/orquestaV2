package orquestaappcodexstack

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
)

func TestCodexStackRuntimeUsageMetricsSourceV0IgnoraLogsRuntimeSinReporteRedactado(t *testing.T) {
	ctx := context.Background()
	runtime := newPendingAckCodexStackRuntimeV0()
	stack := mustBuildCodexStackForTestV0(t, runtime)
	director := postDirectorAPIV0(t, stack)
	run := mustLoadCodexStackRunForTestV0(t, stack, director.RunRef)
	descriptors, err := stack.Stores.ReceiptStore.ListCodexReceiptDescriptorsV0(
		ctx,
		orquestaruntimecodexdelivery.CodexReceiptDescriptorRequestV0{
			RunID:         director.RunRef,
			StartedAgents: director.StartedAgents,
		},
	)
	if err != nil || len(descriptors) == 0 {
		t.Fatalf("descriptors=%d err=%v", len(descriptors), err)
	}
	agentRef := strings.TrimSpace(descriptors[0].AgentRef)
	runtimeDir := filepath.Dir(descriptors[0].AckPath)
	if err := os.WriteFile(
		filepath.Join(runtimeDir, orquestaruntimecodex.CodexStderrFileNameV0),
		[]byte("quota status: limited\nquota remaining: 42\nquota limit: 100\n"+
			"prompt tokens: 1000\ncompletion tokens: 250\ntokens used: 1,250\n"+
			"cost micros: 700\naccess_token=abc123\n"),
		0o600,
	); err != nil {
		t.Fatalf("write usage log: %v", err)
	}
	source := CodexStackAgentUsageSourceV0{
		Store: stack.Stores.ReceiptStore,
		UsageMetrics: CodexStackRuntimeUsageMetricsSourceV0{
			Store: stack.Stores.ReceiptStore,
		},
	}

	stats := orquestacionnucleoapp.BuildDirectorRunStatsWithTelemetryPortsV0(
		ctx,
		run,
		nil,
		nil,
		source,
		orquestacionnucleoapp.DirectorProgressSourceRequestV0{IncludeAgentUsage: true},
	)

	agent := findStackDirectorAgentStatsForTestV0(t, stats, agentRef)
	if agent.Usage == nil ||
		agent.Usage.QuotaStatus != orquestacionnucleoapp.DirectorAgentUsageQuotaUnknownV0 ||
		agent.Usage.TotalTokens != 0 ||
		!containsStringV0(agent.Usage.EvidenceRefs, "quota_observed_unavailable") ||
		!containsStringV0(agent.Usage.EvidenceRefs, "usage-accounting-report-missing") {
		t.Fatalf("usage=%+v", agent.Usage)
	}
}

func TestCodexStackRuntimeUsageMetricsSourceV0LeeReporteRedactado(t *testing.T) {
	ctx := context.Background()
	runtime := newPendingAckCodexStackRuntimeV0()
	stack := mustBuildCodexStackForTestV0(t, runtime)
	director := postDirectorAPIV0(t, stack)
	run := mustLoadCodexStackRunForTestV0(t, stack, director.RunRef)
	descriptors, err := stack.Stores.ReceiptStore.ListCodexReceiptDescriptorsV0(
		ctx,
		orquestaruntimecodexdelivery.CodexReceiptDescriptorRequestV0{
			RunID:         director.RunRef,
			StartedAgents: director.StartedAgents,
		},
	)
	if err != nil || len(descriptors) == 0 {
		t.Fatalf("descriptors=%d err=%v", len(descriptors), err)
	}
	agentRef := strings.TrimSpace(descriptors[0].AgentRef)
	runtimeDir := filepath.Dir(descriptors[0].AckPath)
	if err := os.WriteFile(
		filepath.Join(runtimeDir, orquestaruntimecodex.CodexStderrFileNameV0),
		[]byte("tokens used: 11\n"),
		0o600,
	); err != nil {
		t.Fatalf("write usage log: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(runtimeDir, orquestaruntimecodex.CodexUsageAccountingFileNameV0),
		[]byte(`{"usage":{"input_tokens":2100,"output_tokens":400},`+
			`"quota":{"status":"available","remaining":88,"limit":100}}`),
		0o600,
	); err != nil {
		t.Fatalf("write usage report: %v", err)
	}
	source := CodexStackAgentUsageSourceV0{
		Store: stack.Stores.ReceiptStore,
		UsageMetrics: CodexStackRuntimeUsageMetricsSourceV0{
			Store: stack.Stores.ReceiptStore,
		},
	}

	stats := orquestacionnucleoapp.BuildDirectorRunStatsWithTelemetryPortsV0(
		ctx,
		run,
		nil,
		nil,
		source,
		orquestacionnucleoapp.DirectorProgressSourceRequestV0{IncludeAgentUsage: true},
	)

	agent := findStackDirectorAgentStatsForTestV0(t, stats, agentRef)
	if agent.Usage == nil ||
		agent.Usage.QuotaStatus != orquestacionnucleoapp.DirectorAgentUsageQuotaAvailableV0 ||
		agent.Usage.QuotaRemaining != 88 ||
		agent.Usage.TotalTokens != 2500 {
		t.Fatalf("usage=%+v", agent.Usage)
	}
}

func TestCodexStackRuntimeUsageMetricsSourceV0UsoSinCuotaRealQuedaUnknown(t *testing.T) {
	ctx := context.Background()
	runtime := newPendingAckCodexStackRuntimeV0()
	stack := mustBuildCodexStackForTestV0(t, runtime)
	director := postDirectorAPIV0(t, stack)
	run := mustLoadCodexStackRunForTestV0(t, stack, director.RunRef)
	descriptors, err := stack.Stores.ReceiptStore.ListCodexReceiptDescriptorsV0(
		ctx,
		orquestaruntimecodexdelivery.CodexReceiptDescriptorRequestV0{
			RunID:         director.RunRef,
			StartedAgents: director.StartedAgents,
		},
	)
	if err != nil || len(descriptors) == 0 {
		t.Fatalf("descriptors=%d err=%v", len(descriptors), err)
	}
	agentRef := strings.TrimSpace(descriptors[0].AgentRef)
	runtimeDir := filepath.Dir(descriptors[0].AckPath)
	if err := os.WriteFile(
		filepath.Join(runtimeDir, orquestaruntimecodex.CodexUsageAccountingFileNameV0),
		[]byte(`{"usage":{"input_tokens":2100,"output_tokens":400}}`),
		0o600,
	); err != nil {
		t.Fatalf("write usage report: %v", err)
	}
	source := CodexStackAgentUsageSourceV0{
		Store: stack.Stores.ReceiptStore,
		UsageMetrics: CodexStackRuntimeUsageMetricsSourceV0{
			Store: stack.Stores.ReceiptStore,
		},
	}

	stats := orquestacionnucleoapp.BuildDirectorRunStatsWithTelemetryPortsV0(
		ctx,
		run,
		nil,
		nil,
		source,
		orquestacionnucleoapp.DirectorProgressSourceRequestV0{IncludeAgentUsage: true},
	)

	agent := findStackDirectorAgentStatsForTestV0(t, stats, agentRef)
	if agent.Usage == nil ||
		agent.Usage.QuotaStatus != orquestacionnucleoapp.DirectorAgentUsageQuotaUnknownV0 ||
		agent.Usage.QuotaRemaining != 0 ||
		agent.Usage.QuotaLimit != 0 ||
		agent.Usage.TotalTokens != 2500 ||
		!containsStringV0(agent.Usage.EvidenceRefs, "quota_observed_unavailable") ||
		!containsStringV0(agent.Usage.EvidenceRefs, "usage-accounting-quota-unknown-no-real-quota") {
		t.Fatalf("usage=%+v", agent.Usage)
	}
}

func TestCodexStackRuntimeUsageMetricsSourceV0ReporteUnavailableNoQuedaNotConfigured(t *testing.T) {
	ctx := context.Background()
	runtime := newPendingAckCodexStackRuntimeV0()
	stack := mustBuildCodexStackForTestV0(t, runtime)
	director := postDirectorAPIV0(t, stack)
	run := mustLoadCodexStackRunForTestV0(t, stack, director.RunRef)
	descriptors, err := stack.Stores.ReceiptStore.ListCodexReceiptDescriptorsV0(
		ctx,
		orquestaruntimecodexdelivery.CodexReceiptDescriptorRequestV0{
			RunID:         director.RunRef,
			StartedAgents: director.StartedAgents,
		},
	)
	if err != nil || len(descriptors) == 0 {
		t.Fatalf("descriptors=%d err=%v", len(descriptors), err)
	}
	agentRef := strings.TrimSpace(descriptors[0].AgentRef)
	runtimeDir := filepath.Dir(descriptors[0].AckPath)
	if err := os.WriteFile(
		filepath.Join(runtimeDir, orquestaruntimecodex.CodexUsageAccountingFileNameV0),
		[]byte(`{"usage":{},"quota":{"status":"unknown","reason":"quota_observed_unavailable"}}`),
		0o600,
	); err != nil {
		t.Fatalf("write usage report: %v", err)
	}
	source := CodexStackAgentUsageSourceV0{
		Store: stack.Stores.ReceiptStore,
		UsageMetrics: CodexStackRuntimeUsageMetricsSourceV0{
			Store: stack.Stores.ReceiptStore,
		},
	}

	stats := orquestacionnucleoapp.BuildDirectorRunStatsWithTelemetryPortsV0(
		ctx,
		run,
		nil,
		nil,
		source,
		orquestacionnucleoapp.DirectorProgressSourceRequestV0{IncludeAgentUsage: true},
	)

	agent := findStackDirectorAgentStatsForTestV0(t, stats, agentRef)
	if agent.Usage == nil ||
		agent.Usage.QuotaStatus != orquestacionnucleoapp.DirectorAgentUsageQuotaUnknownV0 ||
		!containsStringV0(agent.Usage.EvidenceRefs, "quota_observed_unavailable") {
		t.Fatalf("usage=%+v", agent.Usage)
	}
	if stats.UsageSummary == nil ||
		stats.UsageSummary.QuotaStatus != orquestacionnucleoapp.DirectorAgentUsageQuotaUnknownV0 {
		t.Fatalf("usage_summary=%+v", stats.UsageSummary)
	}
}

func TestCodexStackRuntimeUsageMetricsSourceV0RechazaReporteNoRedactado(t *testing.T) {
	ctx := context.Background()
	runtime := newPendingAckCodexStackRuntimeV0()
	stack := mustBuildCodexStackForTestV0(t, runtime)
	director := postDirectorAPIV0(t, stack)
	run := mustLoadCodexStackRunForTestV0(t, stack, director.RunRef)
	descriptors, err := stack.Stores.ReceiptStore.ListCodexReceiptDescriptorsV0(
		ctx,
		orquestaruntimecodexdelivery.CodexReceiptDescriptorRequestV0{
			RunID:         director.RunRef,
			StartedAgents: director.StartedAgents,
		},
	)
	if err != nil || len(descriptors) == 0 {
		t.Fatalf("descriptors=%d err=%v", len(descriptors), err)
	}
	agentRef := strings.TrimSpace(descriptors[0].AgentRef)
	runtimeDir := filepath.Dir(descriptors[0].AckPath)
	if err := os.WriteFile(
		filepath.Join(runtimeDir, orquestaruntimecodex.CodexUsageAccountingFileNameV0),
		[]byte(`{"usage":{"total_tokens":10},"prompt":"no debe viajar"}`),
		0o600,
	); err != nil {
		t.Fatalf("write usage report: %v", err)
	}
	source := CodexStackAgentUsageSourceV0{
		Store: stack.Stores.ReceiptStore,
		UsageMetrics: CodexStackRuntimeUsageMetricsSourceV0{
			Store: stack.Stores.ReceiptStore,
		},
	}

	stats := orquestacionnucleoapp.BuildDirectorRunStatsWithTelemetryPortsV0(
		ctx,
		run,
		nil,
		nil,
		source,
		orquestacionnucleoapp.DirectorProgressSourceRequestV0{IncludeAgentUsage: true},
	)

	agent := findStackDirectorAgentStatsForTestV0(t, stats, agentRef)
	if agent.Usage == nil ||
		agent.Usage.QuotaStatus != orquestacionnucleoapp.DirectorAgentUsageQuotaUnknownV0 ||
		agent.Usage.TotalTokens != 0 ||
		!containsStringV0(agent.Usage.EvidenceRefs, "quota_observed_unavailable") ||
		!containsStringV0(agent.Usage.EvidenceRefs, "usage-accounting-report-rejected-not-redacted") {
		t.Fatalf("usage=%+v", agent.Usage)
	}
}

func containsStringV0(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
