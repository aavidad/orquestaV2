package orquestaappcodexstack

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	orquestaagentprocessregistrymemory "orquesta/modulos/orquesta-agent-process-registry-memory"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
	orquestaservershutdown "orquesta/modulos/orquesta-server-shutdown"
)

func TestStackShutdownCheckpointV0SolicitaAckSiHayAgenteEnVuelo(t *testing.T) {
	fixture := newStackShutdownCheckpointFixtureV0(t)

	result, err := fixture.preparer.PrepareAgentShutdownV0(context.Background(), fixture.command)
	if err != nil {
		t.Fatalf("PrepareAgentShutdownV0: %v", err)
	}
	if result.CheckpointRecorded || len(result.PendingAgentRefs) != 1 ||
		result.PendingAgentRefs[0] != fixture.agentRef {
		t.Fatalf("result=%+v", result)
	}
	if _, err := os.Stat(filepath.Join(fixture.runtimeDir, orquestaruntimecodex.CodexShutdownRequestFileNameV0)); err != nil {
		t.Fatalf("shutdown request no escrito: %v", err)
	}
}

func TestStackShutdownCheckpointV0RegistraCuandoTodosLosAgentesResponden(t *testing.T) {
	fixture := newStackShutdownCheckpointFixtureV0(t)
	request := stackShutdownRequestForAgentV0(fixture.command, fixture.descriptor)
	writeStackShutdownAckForTestV0(t, fixture.runtimeDir, request)

	result, err := fixture.preparer.PrepareAgentShutdownV0(context.Background(), fixture.command)
	if err != nil {
		t.Fatalf("PrepareAgentShutdownV0: %v", err)
	}
	if !result.CheckpointRecorded ||
		result.CheckpointRef != "checkpoint-ref-shutdown-"+safeStackShutdownRefPartV0(fixture.runRef) ||
		len(result.PendingAgentRefs) != 0 {
		t.Fatalf("result=%+v", result)
	}
}

func TestStackShutdownCheckpointV0NoConsumeAckDeIntentoAnterior(t *testing.T) {
	fixture := newStackShutdownCheckpointFixtureV0(t)
	firstRequest := stackShutdownRequestForAgentV0(fixture.command, fixture.descriptor)
	writeStackShutdownAckForTestV0(t, fixture.runtimeDir, firstRequest)
	fixture.command.CorrelationID = "corr-shutdown-stack-002"
	fixture.command.Reason = "shutdown controlado por segunda mutacion"

	result, err := fixture.preparer.PrepareAgentShutdownV0(context.Background(), fixture.command)
	if err != nil {
		t.Fatalf("PrepareAgentShutdownV0: %v", err)
	}
	if result.CheckpointRecorded || len(result.PendingAgentRefs) != 1 ||
		result.PendingAgentRefs[0] != fixture.agentRef {
		t.Fatalf("result=%+v", result)
	}
	if !hasStackShutdownEvidenceForTestV0(result.EvidenceRefs, "shutdown_checkpoint_attempt_mismatch") {
		t.Fatalf("evidence sin mismatch de intento: %+v", result.EvidenceRefs)
	}
}

func TestStackShutdownCheckpointV0RegistraSiAgenteEnVueloNoTieneProcesoVivo(t *testing.T) {
	fixture := newStackShutdownCheckpointFixtureV0(t)
	fixture.preparer.Config.Codex.SnapshotSource = stackShutdownSnapshotSourceForTestV0{}

	result, err := fixture.preparer.PrepareAgentShutdownV0(context.Background(), fixture.command)
	if err != nil {
		t.Fatalf("PrepareAgentShutdownV0: %v", err)
	}
	if !result.CheckpointRecorded || len(result.PendingAgentRefs) != 0 {
		t.Fatalf("result=%+v", result)
	}
	if _, err := os.Stat(filepath.Join(fixture.runtimeDir, orquestaruntimecodex.CodexShutdownRequestFileNameV0)); !os.IsNotExist(err) {
		t.Fatalf("shutdown request no debe escribirse para agente stale: %v", err)
	}
	if !hasStackShutdownEvidenceForTestV0(result.EvidenceRefs, "shutdown-agent-stale-no-process") {
		t.Fatalf("evidence sin stale no-process: %+v", result.EvidenceRefs)
	}
}

func TestStackShutdownCheckpointV0MantieneEsperaSiProcesoSigueVivo(t *testing.T) {
	fixture := newStackShutdownCheckpointFixtureV0(t)
	processRef := "process-ref-shutdown-stack-001"
	sessionRef := "session-ref-shutdown-stack-001"
	launchRef := "launch-ref-shutdown-stack-001"
	recordStackShutdownProcessForTestV0(t, fixture, processRef, sessionRef, launchRef)
	fixture.preparer.Config.Codex.SnapshotSource = stackShutdownSnapshotSourceForTestV0{
		snapshots: map[string]orquestaruntime.ProcessRuntimeSnapshotV0{
			processRef: {
				SchemaVersion: orquestaruntime.ProcessRuntimeConnectorVersionV0,
				ProcessRef:    processRef,
				SessionRef:    sessionRef,
				LaunchRef:     launchRef,
				Status:        orquestaruntime.ProcessRuntimeRunningV0,
			},
		},
	}

	result, err := fixture.preparer.PrepareAgentShutdownV0(context.Background(), fixture.command)
	if err != nil {
		t.Fatalf("PrepareAgentShutdownV0: %v", err)
	}
	if result.CheckpointRecorded || len(result.PendingAgentRefs) != 1 ||
		result.PendingAgentRefs[0] != fixture.agentRef {
		t.Fatalf("result=%+v", result)
	}
}

func TestStackShutdownStatsV0ExponeLivenessDeAgentesObsoletos(t *testing.T) {
	fixture := newStackShutdownCheckpointFixtureV0(t)
	fixture.preparer.Config.Codex.SnapshotSource = stackShutdownSnapshotSourceForTestV0{}

	stats, err := stackShutdownStatsReaderV0{
		Config: fixture.preparer.Config,
	}.ReadRunShutdownStatsV0(context.Background(), orquestaservershutdown.RunShutdownStatsRequestV0{
		RunRef: fixture.runRef,
	})
	if err != nil {
		t.Fatalf("ReadRunShutdownStatsV0: %v", err)
	}
	if !stats.ProcessLivenessObserved ||
		stats.AgentsInFlight != 1 ||
		stats.AgentsRunningLive != 0 ||
		stats.AgentsRunningStale != 1 ||
		stats.AgentsLost != 0 {
		t.Fatalf("stats=%+v", stats)
	}
}

type stackShutdownCheckpointFixtureDataV0 struct {
	runRef     string
	agentRef   string
	runtimeDir string
	descriptor orquestaruntimecodexdelivery.CodexReceiptDescriptorV0
	preparer   stackShutdownCheckpointPreparerV0
	command    orquestaservershutdown.PrepareAgentShutdownCommandV0
}

func newStackShutdownCheckpointFixtureV0(t *testing.T) stackShutdownCheckpointFixtureDataV0 {
	t.Helper()
	runRef := "run-ref-shutdown-stack-001"
	agentRef := "agent-ref-shutdown-stack-001"
	runtimeDir := t.TempDir()
	descriptor := orquestaruntimecodexdelivery.CodexReceiptDescriptorV0{
		DescriptorRef:  "descriptor-ref-shutdown-stack-001",
		RunID:          runRef,
		AgentRef:       agentRef,
		AckPath:        filepath.Join(runtimeDir, orquestaruntimecodex.CodexAgentAckFileNameV0),
		ProjectWorkDir: t.TempDir(),
		Spec: orquestaruntime.ExternalAgentLaunchSpecV0{
			SchemaVersion: orquestaruntime.ExternalAgentLaunchSpecSchemaVersionV0,
			RequestID:     agentRef,
			CorrelationID: "corr-shutdown-stack-001",
			RuntimeKind:   "cli",
			AgentPacket: orquestaruntime.AgentStartPacketV0{
				RequestID:     agentRef,
				CorrelationID: "corr-shutdown-stack-001",
				CapacityLevel: "medium",
				DeliveryRefs: orquestaruntime.AgentStartDeliveryRefsV0{
					AckRef: "ack-ref-shutdown-stack-001",
				},
			},
		},
	}
	receiptStore := orquestaruntimecodexdelivery.NewInMemoryCodexReceiptDescriptorStoreV0(descriptor)
	return stackShutdownCheckpointFixtureDataV0{
		runRef:     runRef,
		agentRef:   agentRef,
		runtimeDir: runtimeDir,
		descriptor: descriptor,
		preparer: stackShutdownCheckpointPreparerV0{Config: ConfigV0{
			Stores: StoresV0{
				RunStore:        orquestacionnucleoapp.NewInMemoryRunStoreV0(stackShutdownRunForTestV0(runRef, agentRef)),
				ReceiptStore:    receiptStore,
				ProgressState:   orquestaruntimecodexdelivery.NewInMemoryCodexProgressStateStoreV0(),
				ProcessRegistry: orquestaagentprocessregistrymemory.NewInMemoryAgentProcessRegistryV0(),
			},
		}},
		command: orquestaservershutdown.PrepareAgentShutdownCommandV0{
			RunRef:        runRef,
			CorrelationID: "corr-shutdown-stack-001",
			RequestedBy:   "test",
			Reason:        "shutdown controlado",
			EvidenceRefs:  []string{"evidence-ref-shutdown-stack"},
		},
	}
}

func stackShutdownRunForTestV0(
	runRef string,
	agentRef string,
) orquestacoreworkflow.OrchestrationRunV0 {
	return orquestacoreworkflow.OrchestrationRunV0{
		SchemaVersion: orquestacoreworkflow.OrchestrationRunSchemaVersionV0,
		RunID:         runRef,
		ProjectRef:    "project-ref-shutdown-stack-001",
		AppSpecRef:    "app-spec-ref-shutdown-stack-001",
		Status:        orquestacoreworkflow.OrchestrationRunStatusActiveV0,
		CurrentPhase:  orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		Tasks:         []string{"task-ref-shutdown-stack-001"},
		Agents:        []string{agentRef},
		StartedAgents: []string{agentRef},
	}
}

func recordStackShutdownProcessForTestV0(
	t *testing.T,
	fixture stackShutdownCheckpointFixtureDataV0,
	processRef string,
	sessionRef string,
	launchRef string,
) {
	t.Helper()
	if err := fixture.preparer.Config.Stores.ProcessRegistry.RecordAgentProcessV0(context.Background(), orquestacionnucleoapp.AgentProcessRecordV0{
		RunID:          fixture.runRef,
		AgentRequestID: fixture.agentRef,
		ProcessRef:     processRef,
		SessionRef:     sessionRef,
		LaunchRef:      launchRef,
		ReadinessRef:   "readiness-ref-shutdown-stack-001",
		EvidenceRefs:   []string{"evidence-ref-shutdown-process-record"},
	}); err != nil {
		t.Fatalf("RecordAgentProcessV0: %v", err)
	}
}

type stackShutdownSnapshotSourceForTestV0 struct {
	snapshots map[string]orquestaruntime.ProcessRuntimeSnapshotV0
}

func (source stackShutdownSnapshotSourceForTestV0) SnapshotV0(
	processRef string,
) (orquestaruntime.ProcessRuntimeSnapshotV0, error) {
	if snapshot, ok := source.snapshots[processRef]; ok {
		return snapshot, nil
	}
	return orquestaruntime.ProcessRuntimeSnapshotV0{}, orquestaruntime.ProcessRuntimeErrorV0{
		Code:       orquestaruntime.ProcessRuntimeNoEncontradoV0,
		MessageKey: "orquesta.runtime.process.process_runtime_no_encontrado",
		Field:      "process_ref",
	}
}

func writeStackShutdownAckForTestV0(
	t *testing.T,
	runtimeDir string,
	request orquestaruntimecodex.CodexShutdownRequestV0,
) {
	t.Helper()
	ack := orquestaruntimecodex.CodexShutdownCheckpointAckV0{
		SchemaVersion:      orquestaruntimecodex.CodexShutdownCheckpointAckSchemaVersionV0,
		RunRef:             request.RunRef,
		AgentRef:           request.AgentRef,
		ShutdownAttemptRef: request.ShutdownAttemptRef,
		CheckpointRef:      request.CheckpointRef,
		Status:             orquestaruntimecodex.CodexShutdownCheckpointStatusReadyV0,
		EvidenceRefs:       []string{"evidence-ref-agent-checkpoint"},
	}
	data, err := json.Marshal(ack)
	if err != nil {
		t.Fatalf("marshal ack: %v", err)
	}
	path := filepath.Join(runtimeDir, orquestaruntimecodex.CodexShutdownCheckpointAckFileNameV0)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write ack: %v", err)
	}
}

func hasStackShutdownEvidenceForTestV0(refs []string, want string) bool {
	for _, ref := range refs {
		if strings.Contains(ref, want) {
			return true
		}
	}
	return false
}
