package orquestaruntimecodexdelivery

import (
	"context"
	"os"
	"testing"

	orquestaagentprocessregistry "orquesta/modulos/orquesta-agent-process-registry"
	orquestaagentprocessregistrymemory "orquesta/modulos/orquesta-agent-process-registry-memory"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

type failingProgressProcessRegistryV0 struct{}

func (failingProgressProcessRegistryV0) RecordAgentProcessV0(
	context.Context,
	orquestaagentprocessregistry.AgentProcessRegistryRecordV0,
) error {
	return nil
}

func (failingProgressProcessRegistryV0) ResolveAgentProcessV0(
	context.Context,
	string,
	string,
) (orquestaagentprocessregistry.AgentProcessRegistryRecordV0, error) {
	return orquestaagentprocessregistry.AgentProcessRegistryRecordV0{}, orquestaagentprocessregistry.ErrorV0{
		Code:    orquestaagentprocessregistry.ErrAgentProcessRegistryInvalidV0,
		Field:   "agent_process_registry",
		Message: "agent_process conflict",
	}
}

func TestCodexProgressObservationSourceV0OmiteSiACKYaEstaListo(t *testing.T) {
	spec, ackPath := codexProgressSpecAndAckPathForTestV0(t)
	source := codexProgressSourceForTestV0(t, spec, ackPath)
	if err := os.WriteFile(ackPath, codexProgressACKBytesForTestV0(t, spec), 0o600); err != nil {
		t.Fatalf("write ack: %v", err)
	}

	got, err := source.BuildAgentProgressObservationsV0(
		context.Background(),
		codexProgressRequestForTestV0(spec),
	)
	if err != nil {
		t.Fatalf("BuildAgentProgressObservationsV0: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("observations=%+v", got)
	}
}

func TestCodexProgressObservationSourceV0OmiteRegistroDeProcesoFaltante(t *testing.T) {
	spec, ackPath := codexProgressSpecAndAckPathForTestV0(t)
	store := NewInMemoryCodexReceiptDescriptorStoreV0(codexProgressDescriptorForTestV0(spec, ackPath))
	source := CodexProgressObservationSourceV0{
		Store:           store,
		ProcessRegistry: orquestaagentprocessregistrymemory.NewInMemoryAgentProcessRegistryV0(),
		State:           NewInMemoryCodexProgressStateStoreV0(),
		Policy:          orquestaruntime.AgentProgressHeartbeatPolicyV0{StalledAfterNoProgressTicks: 1},
	}

	got, err := source.BuildAgentProgressObservationsV0(
		context.Background(),
		codexProgressRequestForTestV0(spec),
	)
	if err != nil || len(got) != 0 {
		t.Fatalf("observations=%+v err=%v", got, err)
	}
}

func TestCodexProgressObservationSourceV0PropagaRegistroInvalido(t *testing.T) {
	spec, ackPath := codexProgressSpecAndAckPathForTestV0(t)
	source := CodexProgressObservationSourceV0{
		Store:           NewInMemoryCodexReceiptDescriptorStoreV0(codexProgressDescriptorForTestV0(spec, ackPath)),
		ProcessRegistry: failingProgressProcessRegistryV0{},
		State:           NewInMemoryCodexProgressStateStoreV0(),
		Policy:          orquestaruntime.AgentProgressHeartbeatPolicyV0{StalledAfterNoProgressTicks: 1},
	}

	_, err := source.BuildAgentProgressObservationsV0(
		context.Background(),
		codexProgressRequestForTestV0(spec),
	)
	if err == nil {
		t.Fatalf("esperaba error por registro invalido")
	}
}

func TestCodexProgressObservationSourceV0AlimentaCandidateProvider(t *testing.T) {
	spec, ackPath := codexProgressSpecAndAckPathForTestV0(t)
	source := codexProgressSourceForTestV0(t, spec, ackPath)
	source.SnapshotSource = stoppedSnapshotSourceForProgressTestV0{}
	request := orquestacionnucleoapp.SchedulerCandidateRequestV0{
		Run:           codexProgressRunForTestV0(spec),
		OccurredAt:    "2026-05-09T12:30:00Z",
		CorrelationID: "corr-progress-provider-001",
	}
	provider := orquestacionnucleoapp.ProgressSupervisionCandidateProviderV0{
		ProgressSource: source,
		RequestedBy:    "orquesta",
	}

	if _, err := provider.BuildSchedulerCandidatesV0(context.Background(), request); err != nil {
		t.Fatalf("first BuildSchedulerCandidatesV0: %v", err)
	}
	got, err := provider.BuildSchedulerCandidatesV0(context.Background(), request)
	if err != nil {
		t.Fatalf("second BuildSchedulerCandidatesV0: %v", err)
	}
	if len(got.ProgressSupervisionCandidates) != 1 {
		t.Fatalf("progress candidates=%+v", got.ProgressSupervisionCandidates)
	}
	candidate := got.ProgressSupervisionCandidates[0]
	if candidate.SupervisionInput.Report.Status != orquestaruntime.AgentStoppedV0 ||
		!candidate.SupervisionInput.Report.DecisionRequired ||
		candidate.SupervisionInput.AssessmentRef == "" {
		t.Fatalf("candidate=%+v", candidate)
	}
}
