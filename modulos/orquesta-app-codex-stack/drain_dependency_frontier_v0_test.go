package orquestaappcodexstack

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	orquestadirectoragentfilesource "orquesta/modulos/orquesta-director-agent-file-source"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
)

func TestDrainRunV0TrasEntregaBootstrapLanzaFronteraDependiente(t *testing.T) {
	runtime := newBatchDecisionPendingProgrammingAckRuntimeV0()
	stack := mustBuildCodexStackForTestV0(t, runtime)
	ctx := context.Background()

	director := postDirectorAPIV0(t, stack)
	if _, err := stack.DrainRunV0(ctx, DrainRunRequestV0{
		RunRef:               director.RunRef,
		CorrelationID:        "corr-stack-frontier-bootstrap-001",
		MaxBursts:            16,
		MaxStepsPerBurst:     12,
		MaxDispatchesPerWait: 8,
		MaxCommands:          20,
		MaxOutboxPerCycle:    8,
		MaxExternalWaits:     1,
	}); err != nil {
		t.Fatalf("DrainRunV0 decisiones: %v", err)
	}
	bootstrap := mustCodexStackDescriptorByTaskRefV0(
		t,
		stack,
		"task-ref-stack-agenda-bootstrap",
	)
	if err := writeCodexStackAckForDescriptorV0(t, bootstrap); err != nil {
		t.Fatalf("write bootstrap ack: %v", err)
	}

	if _, err := stack.DrainRunV0(ctx, DrainRunRequestV0{
		RunRef:               director.RunRef,
		CorrelationID:        "corr-stack-frontier-bootstrap-002",
		MaxBursts:            16,
		MaxStepsPerBurst:     12,
		MaxDispatchesPerWait: 8,
		MaxCommands:          20,
		MaxOutboxPerCycle:    8,
		MaxExternalWaits:     1,
	}); err != nil {
		t.Fatalf("DrainRunV0 frontera dependiente: %v", err)
	}
	run := mustLoadCodexStackRunForTestV0(t, stack, director.RunRef)
	if !codexStackStringInSetForTestV0(run.DeliveredTasks, "task-ref-stack-agenda-bootstrap") {
		t.Fatalf("delivered_tasks=%v", run.DeliveredTasks)
	}
	for _, taskRef := range []string{
		"task-ref-stack-agenda-domain",
		"task-ref-stack-agenda-http",
		"task-ref-stack-agenda-web",
		"task-ref-stack-agenda-docs",
	} {
		agentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(taskRef)
		if !codexStackStringInSetForTestV0(run.StartedAgents, agentRef) {
			t.Fatalf("started_agents=%v missing=%s delivered=%v", run.StartedAgents, agentRef, run.DeliveredTasks)
		}
	}
}

type batchDecisionPendingProgrammingAckRuntimeV0 struct {
	*fakeCodexStackRuntimeV0
}

func newBatchDecisionPendingProgrammingAckRuntimeV0() *batchDecisionPendingProgrammingAckRuntimeV0 {
	return &batchDecisionPendingProgrammingAckRuntimeV0{
		fakeCodexStackRuntimeV0: newFakeCodexStackRuntimeV0(),
	}
}

func (runtime *batchDecisionPendingProgrammingAckRuntimeV0) LaunchV0(
	ctx context.Context,
	req orquestaruntime.ProcessRuntimeLaunchRequestV0,
) (orquestaruntime.ProcessRuntimeSnapshotV0, error) {
	snapshot, err := runtime.fakeCodexStackRuntimeV0.LaunchV0(ctx, req)
	if err != nil {
		return snapshot, err
	}
	runtimeDir := filepath.Dir(req.CommandPath)
	packet, err := codexStackPacketFromRuntimeDirForTestV0(runtimeDir)
	if err != nil {
		return snapshot, err
	}
	if packet.TargetModule == "orquesta-app-stack-director" {
		return snapshot, writeBatchDirectorDecisionsForRuntimeDirV0(runtimeDir, packet)
	}
	if packet.TargetModule == "orquesta-app-stack-programacion" {
		return snapshot, os.Remove(filepath.Join(runtimeDir, orquestaruntimecodex.CodexAgentAckFileNameV0))
	}
	return snapshot, nil
}

func writeBatchDirectorDecisionsForRuntimeDirV0(
	runtimeDir string,
	packet orquestaruntime.AgentStartPacketV0,
) error {
	runRef := codexStackObjectiveValueForTestV0(packet.Task.Objective, "RunID para decisiones:")
	brainstormRef := codexStackObjectiveValueForTestV0(packet.Task.Objective, "BrainstormRef inicial:")
	data, err := json.Marshal(orquestadirectoragentfilesource.DirectorAgentDecisionFileEnvelopeV0{
		SchemaVersion: orquestadirectoragentfilesource.DirectorAgentDecisionFileSchemaVersionV0,
		Decisions:     codexStackBatchDirectorDecisionsForTestV0(runRef, brainstormRef),
	})
	if err != nil {
		return err
	}
	return os.WriteFile(
		filepath.Join(runtimeDir, orquestaruntimecodex.CodexDirectorDecisionsFileNameV0),
		data,
		0o600,
	)
}

func mustCodexStackDescriptorByTaskRefV0(
	t *testing.T,
	stack StackV0,
	taskRef string,
) orquestaruntimecodexdelivery.CodexReceiptDescriptorV0 {
	t.Helper()
	descriptors := codexStackDescriptorsForTestV0(t, stack)
	for _, descriptor := range descriptors {
		if descriptor.Spec.AgentPacket.Task.TaskRef == taskRef {
			return descriptor
		}
	}
	t.Fatalf("descriptor no encontrado task_ref=%s descriptors=%v", taskRef, codexStackRealSmokeDescriptorAgentsV0(descriptors))
	return orquestaruntimecodexdelivery.CodexReceiptDescriptorV0{}
}

func writeCodexStackAckForDescriptorV0(
	t *testing.T,
	descriptor orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
) error {
	t.Helper()
	packet := descriptor.Spec.AgentPacket
	data, err := json.Marshal(map[string]any{
		"schema_version": "codex_agent_ack.v0",
		"request_id":     packet.RequestID,
		"correlation_id": packet.CorrelationID,
		"ack_ref":        packet.DeliveryRefs.AckRef,
		"target_module":  packet.TargetModule,
		"task_ref":       packet.Task.TaskRef,
		"status":         "completed",
		"files":          packet.Task.WriteSet,
		"tests":          packet.Task.RequiredTests,
		"test_receipts":  codexStackRequiredTestReceiptsV0(packet.Task.RequiredTests),
		"notes":          codexStackFakeAckNotesV0(packet, "ack de prueba para desbloqueo de frontera"),
	})
	if err != nil {
		return err
	}
	return os.WriteFile(descriptor.AckPath, data, 0o600)
}
