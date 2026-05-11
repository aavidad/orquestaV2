package orquestaruntimecodexdelivery

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestacionnucleoapp "orquesta/orquestacionnucleoapp"
)

func mcpFormDirectorTeamSmokeDescriptorsV0(
	t *testing.T,
	store *InMemoryCodexReceiptDescriptorStoreV0,
	runRef string,
	startedAgents []string,
) []CodexReceiptDescriptorV0 {
	t.Helper()
	descriptors, err := store.ListCodexReceiptDescriptorsV0(context.Background(), CodexReceiptDescriptorRequestV0{
		RunID:         runRef,
		StartedAgents: startedAgents,
	})
	if err != nil {
		t.Fatalf("ListCodexReceiptDescriptorsV0: %v", err)
	}
	if len(descriptors) != len(startedAgents) {
		t.Fatalf("descriptors=%d started=%d %+v", len(descriptors), len(startedAgents), descriptors)
	}
	return descriptors
}

func mcpFormDirectorTeamSmokeRegisterArtifactsV0(
	t *testing.T,
	ctx context.Context,
	store *orquestacionnucleoapp.InMemoryRunStoreV0,
	sink *orquestacionnucleoapp.InMemoryEventSinkV0,
	ledger *orquestacionnucleoapp.InMemoryOutboxLedgerV0,
	receiptStore *InMemoryCodexReceiptDescriptorStoreV0,
	runRef string,
) orquestacionnucleoapp.ProgressiveLoopResultV0 {
	t.Helper()
	service := orquestacionnucleoapp.ServiceV0{
		RunStore:  store,
		EventSink: sink,
		CandidateProvider: orquestacionnucleoapp.DeliveryCandidateProviderV0{
			DeliverySource: CodexDeliveryObservationSourceV0{Store: receiptStore},
			RequestedBy:    "orquesta-form-director-team",
		},
		OutboxLedger:      ledger,
		MaxCommands:       8,
		MaxOutboxPerCycle: 4,
	}
	result, err := service.RunProgressiveLoopV0(ctx, orquestacionnucleoapp.ProgressiveLoopRequestV0{
		RunRef:               runRef,
		OccurredAt:           "2026-05-09T23:55:00Z",
		MaxBursts:            6,
		MaxStepsPerBurst:     6,
		MaxDispatchesPerWait: 1,
		CorrelationID:        "corr-form-director-team-artifact",
		EvidenceRefs:         []string{"evidence-ref-form-director-team-artifact"},
	})
	if err != nil {
		t.Fatalf("register team artifacts: %v", err)
	}
	if result.Status != orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0 {
		t.Fatalf("artifact loop status=%s result=%+v", result.Status, result)
	}
	return result
}

func mcpFormDirectorTeamSmokeVerifyDocsV0(t *testing.T, projectDir string) {
	t.Helper()
	for _, path := range []string{
		"docs/arquitectura.md",
		"docs/plan_microtareas.md",
		"docs/web.md",
		"docs/api.md",
		"docs/persistencia.md",
	} {
		data, err := os.ReadFile(filepath.Join(projectDir, path))
		if err != nil {
			t.Fatalf("doc missing %s: %v", path, err)
		}
		if len(strings.TrimSpace(string(data))) < 120 {
			t.Fatalf("doc demasiado pequeno %s", path)
		}
	}
}

func mcpFormDirectorTeamSmokeStopProcessesV0(
	t *testing.T,
	processRuntime *orquestaruntime.ProcessRuntimeConnectorV0,
	registry *orquestacionnucleoapp.InMemoryAgentProcessRegistryV0,
	runRef string,
	agents []string,
) {
	t.Helper()
	for _, agentRef := range agents {
		codexRealSmokeStopProcessForTestV0(t, processRuntime, registry, runRef, agentRef)
	}
}
