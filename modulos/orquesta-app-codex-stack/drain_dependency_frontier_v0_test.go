package orquestaappcodexstack

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	orquestadirectoragentfilesource "orquesta/modulos/orquesta-director-agent-file-source"
	orquestamcp "orquesta/modulos/orquesta-mcp"
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

func TestCodexStackAutoprogrammingPrepareRunAPIV0TickGlobalTrasACKLanzaFronteraDependienteV0(t *testing.T) {
	ctx := context.Background()
	runtime := newPendingAckCodexStackRuntimeV0()
	stack := mustBuildCodexStackForTestV0(t, runtime)
	request := autoprogrammingBridgeRequestForTestV0()
	request.RequestRef = "run-autoprogramming-dependency-frontier-001"
	bootstrapTaskRef := codexStackAutoprogrammingTaskRefForTestV0(request.RequestRef, 0)
	request.Tasks[0].TaskRef = "source-task-ref-autoprogramming-bootstrap"
	request.Tasks[0].Area = "bootstrap"
	request.Tasks[0].WriteSet = []string{"modulos/orquesta-app-codex-stack/bootstrap_frontier_test.go"}
	request.Tasks = append(request.Tasks, request.Tasks[0])
	request.Tasks[1].TaskRef = "source-task-ref-autoprogramming-dependent"
	request.Tasks[1].Area = "dependent"
	request.Tasks[1].WriteSet = []string{"modulos/orquesta-app-codex-stack/dependent_frontier_test.go"}
	request.Tasks[1].DependsOn = []string{bootstrapTaskRef}
	request.WriteSet = append(request.Tasks[0].WriteSet, request.Tasks[1].WriteSet...)

	prepared := postAutoprogrammingPrepareRunStackV0(t, stack, legacyAutoprogrammingPrepareRunInputForStackTestV0(orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0{
		RequestID:              "request-autoprogramming-dependency-frontier-001",
		CorrelationID:          "corr-autoprogramming-dependency-frontier-001",
		OccurredAt:             "2026-06-18T09:00:00Z",
		RequestedBy:            "orquesta-stack-api-test",
		AutoprogrammingRequest: request,
		MaxBursts:              3,
		MaxStepsPerBurst:       4,
		MaxDispatchesPerWait:   3,
		MaxCommands:            8,
		MaxOutboxPerCycle:      5,
	}))
	if !prepared.Accepted || prepared.RunRef == "" || len(prepared.WorkflowTaskRefs) != 2 {
		t.Fatalf("prepared=%+v", prepared)
	}
	bootstrap := prepared.WorkflowTaskRefs[0]
	dependent := prepared.WorkflowTaskRefs[1]
	if bootstrap != bootstrapTaskRef {
		t.Fatalf("bootstrap ref inesperada: got=%s want=%s prepared=%+v", bootstrap, bootstrapTaskRef, prepared)
	}

	first, err := stack.RunGlobalTickV0(ctx, globalTickCommandForTestV0())
	if err != nil {
		t.Fatalf("RunGlobalTickV0 first: %v", err)
	}
	run := mustLoadCodexStackRunForTestV0(t, stack, prepared.RunRef)
	bootstrapAgent := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(bootstrap)
	dependentAgent := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(dependent)
	if runtime.launchCountV0() != 1 ||
		!codexStackStringInSetForTestV0(run.StartedAgents, bootstrapAgent) ||
		codexStackStringInSetForTestV0(run.StartedAgents, dependentAgent) {
		t.Fatalf("first=%+v launches=%d started=%v bootstrap=%s dependent=%s",
			first,
			runtime.launchCountV0(),
			run.StartedAgents,
			bootstrapAgent,
			dependentAgent,
		)
	}
	descriptor := mustCodexStackDescriptorByTaskRefV0(t, stack, bootstrap)
	if err := writeCodexStackAckForDescriptorV0(t, descriptor); err != nil {
		t.Fatalf("write bootstrap ack: %v", err)
	}

	second, err := stack.RunGlobalTickV0(ctx, globalTickCommandForTestV0())
	if err != nil {
		t.Fatalf("RunGlobalTickV0 second: %v", err)
	}
	run = mustLoadCodexStackRunForTestV0(t, stack, prepared.RunRef)
	if !codexStackStringInSetForTestV0(run.DeliveredTasks, bootstrap) ||
		!codexStackStringInSetForTestV0(run.StartedAgents, dependentAgent) ||
		runtime.launchCountV0() != 2 {
		t.Fatalf("second=%+v launches=%d delivered=%v started=%v bootstrap=%s dependent=%s",
			second,
			runtime.launchCountV0(),
			run.DeliveredTasks,
			run.StartedAgents,
			bootstrap,
			dependentAgent,
		)
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

func codexStackAutoprogrammingTaskRefForTestV0(requestRef string, groupIndex int) string {
	sum := sha256.Sum256([]byte(requestRef))
	return fmt.Sprintf("task-autoprogramming-%x-g%02d", sum[:6], groupIndex+1)
}
