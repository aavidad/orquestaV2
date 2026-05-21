package orquestaappcodexstack

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	orquestaagentprocessregistrymemory "orquesta/modulos/orquesta-agent-process-registry-memory"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
)

func codexStackRealSmokeDescriptorsV0(
	t *testing.T,
	store *orquestaruntimecodexdelivery.InMemoryCodexReceiptDescriptorStoreV0,
) []orquestaruntimecodexdelivery.CodexReceiptDescriptorV0 {
	t.Helper()
	descriptors, err := store.ListCodexReceiptDescriptorsV0(
		context.Background(),
		orquestaruntimecodexdelivery.CodexReceiptDescriptorRequestV0{},
	)
	if err != nil {
		t.Fatalf("ListCodexReceiptDescriptorsV0: %v", err)
	}
	return descriptors
}

func codexStackRealSmokeWaitForAckPathV0(
	ctx context.Context,
	ackPath string,
	spec orquestaruntime.ExternalAgentLaunchSpecV0,
) error {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		_, issues := orquestaruntimecodex.ReadCodexDeliveryObservationFileV0(ackPath, spec)
		if len(issues) == 0 {
			return nil
		}
		if !codexStackRealSmokeAckNotReadyV0(issues[0]) {
			return fmt.Errorf("%s:%s:%s", issues[0].Code, issues[0].Field, strings.Join(issues[0].Evidence, ","))
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func codexStackRealSmokeAckNotReadyV0(issue orquestaruntime.ExternalAgentConnectorErrorV0) bool {
	if issue.Code != orquestaruntime.ExternalAgentConnectorErrorCodeV0(orquestaruntimecodex.CodexConnectorAckInvalidV0) ||
		strings.TrimSpace(issue.Field) != orquestaruntimecodex.CodexAgentAckFileNameV0 ||
		!issue.Retryable {
		return false
	}
	return codexStackRealSmokeContainsProjectionPartV0(issue.Evidence, "ack_not_ready")
}

func codexStackRealSmokeStopProcessV0(
	t *testing.T,
	processRuntime *orquestaruntime.ProcessRuntimeConnectorV0,
	registry *orquestaagentprocessregistrymemory.InMemoryAgentProcessRegistryV0,
	runRef string,
	agentRef string,
) {
	t.Helper()
	record, err := registry.ResolveAgentProcessV0(context.Background(), runRef, agentRef)
	if err != nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, _ = processRuntime.StopV0(ctx, record.ProcessRef)
}

func codexStackRealSmokeStopAllProcessesV0(
	t *testing.T,
	processRuntime *orquestaruntime.ProcessRuntimeConnectorV0,
	registry *orquestaagentprocessregistrymemory.InMemoryAgentProcessRegistryV0,
	store *orquestaruntimecodexdelivery.InMemoryCodexReceiptDescriptorStoreV0,
) {
	t.Helper()
	for _, descriptor := range codexStackRealSmokeDescriptorsV0(t, store) {
		codexStackRealSmokeStopProcessV0(t, processRuntime, registry, descriptor.RunID, descriptor.AgentRef)
	}
}

func codexStackRealSmokeVerifyDocsV0(t *testing.T, projectDir string) {
	t.Helper()
	for _, path := range []string{"docs/arquitectura.md", "docs/plan_tareas.md"} {
		codexStackRealSmokeVerifyProjectFileV0(t, projectDir, path)
	}
}

func codexStackRealSmokeVerifyDescriptorWriteSetsV0(
	t *testing.T,
	projectDir string,
	descriptors []orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
) {
	t.Helper()
	seen := map[string]bool{}
	for _, descriptor := range descriptors {
		for _, path := range descriptor.Spec.AgentPacket.Task.WriteSet {
			if seen[path] {
				continue
			}
			seen[path] = true
			codexStackRealSmokeVerifyProjectFileV0(t, projectDir, path)
		}
	}
}

func codexStackRealSmokeDeliveryObservationsV0(
	t *testing.T,
	ctx context.Context,
	stack StackV0,
	run orquestacoreworkflow.OrchestrationRunV0,
) []orquestacionnucleoapp.AgentDeliveryObservationV0 {
	t.Helper()
	observations, err := stack.Ports.DeliverySource.BuildAgentDeliveryObservationsV0(
		ctx,
		orquestacionnucleoapp.AgentDeliveryObservationRequestV0{
			Run:           run,
			OccurredAt:    "2026-05-10T12:00:00Z",
			CorrelationID: "corr-app-stack-real-diagnostics",
		},
	)
	if err != nil {
		t.Fatalf("BuildAgentDeliveryObservationsV0: %v", err)
	}
	return observations
}

func codexStackRealSmokeDescriptorAgentsV0(
	descriptors []orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
) []string {
	refs := make([]string, 0, len(descriptors))
	for _, descriptor := range descriptors {
		refs = append(refs, descriptor.AgentRef)
	}
	return compactStringsV0(refs)
}

func codexStackRealSmokeHasProgrammingDescriptorV0(
	descriptors []orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
) bool {
	return len(codexStackRealSmokeProgrammingReceiptDescriptorsV0(descriptors)) > 0
}

func codexStackRealSmokeProgrammingDescriptorsV0(
	descriptors []orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
) []string {
	programming := codexStackRealSmokeProgrammingReceiptDescriptorsV0(descriptors)
	refs := make([]string, 0, len(programming))
	for _, descriptor := range programming {
		refs = append(refs, descriptor.AgentRef)
	}
	return compactStringsV0(refs)
}

func codexStackRealSmokeProgrammingReceiptDescriptorsV0(
	descriptors []orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
) []orquestaruntimecodexdelivery.CodexReceiptDescriptorV0 {
	programming := make([]orquestaruntimecodexdelivery.CodexReceiptDescriptorV0, 0, len(descriptors))
	for _, descriptor := range descriptors {
		if descriptor.Spec.AgentPacket.Phase == string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0) {
			programming = append(programming, descriptor)
		}
	}
	return programming
}

func codexStackRealSmokeWaitForDescriptorsAckV0(
	ctx context.Context,
	descriptors []orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
) error {
	for _, descriptor := range descriptors {
		if err := codexStackRealSmokeWaitForAckPathV0(
			ctx,
			descriptor.AckPath,
			descriptor.Spec,
		); err != nil {
			return fmt.Errorf("%s: %w", descriptor.AgentRef, err)
		}
	}
	return nil
}

func codexStackRealSmokeAllProgrammingDeliveriesRegisteredV0(
	deliveries []string,
	descriptors []orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
) bool {
	tasks := map[string]bool{}
	deliveredTasks := map[string]bool{}
	for _, descriptor := range descriptors {
		taskRef := codexStackRealSmokeDescriptorTaskRefV0(descriptor)
		if taskRef == "" {
			return false
		}
		tasks[taskRef] = true
		ackRef := descriptor.Spec.AgentPacket.DeliveryRefs.AckRef
		if codexStackRealSmokeContainsProjectionPartV0(deliveries, ackRef) {
			deliveredTasks[taskRef] = true
		}
	}
	for taskRef := range tasks {
		if !deliveredTasks[taskRef] {
			return false
		}
	}
	return true
}

func TestCodexStackRealSmokeAllProgrammingDeliveriesRegisteredV0AceptaRecoveryPorTaskRef(t *testing.T) {
	descriptors := []orquestaruntimecodexdelivery.CodexReceiptDescriptorV0{
		codexStackRealSmokeDescriptorForTaskRefV0("agent-ref-original-001", "task-ref-programming-001", "ack-ref-original-001"),
		codexStackRealSmokeDescriptorForTaskRefV0("agent-ref-assessment-001", "task-ref-programming-001", "ack-ref-assessment-001"),
	}

	if !codexStackRealSmokeAllProgrammingDeliveriesRegisteredV0(
		[]string{"ack-ref-assessment-001"},
		descriptors,
	) {
		t.Fatalf("recovery por task_ref no aceptado")
	}
}

func TestCodexStackRealSmokeAllProgrammingDeliveriesRegisteredV0RequiereCadaTask(t *testing.T) {
	descriptors := []orquestaruntimecodexdelivery.CodexReceiptDescriptorV0{
		codexStackRealSmokeDescriptorForTaskRefV0("agent-ref-original-001", "task-ref-programming-001", "ack-ref-original-001"),
		codexStackRealSmokeDescriptorForTaskRefV0("agent-ref-original-002", "task-ref-programming-002", "ack-ref-original-002"),
	}

	if codexStackRealSmokeAllProgrammingDeliveriesRegisteredV0(
		[]string{"ack-ref-original-001"},
		descriptors,
	) {
		t.Fatalf("acepta lote con task sin entrega")
	}
}

func codexStackRealSmokeDescriptorForTaskRefV0(
	agentRef string,
	taskRef string,
	ackRef string,
) orquestaruntimecodexdelivery.CodexReceiptDescriptorV0 {
	return orquestaruntimecodexdelivery.CodexReceiptDescriptorV0{
		AgentRef: agentRef,
		Spec: orquestaruntime.ExternalAgentLaunchSpecV0{
			AgentPacket: orquestaruntime.AgentStartPacketV0{
				WorkOrderRef: taskRef,
				Task: orquestaruntime.AgentStartTaskV0{
					TaskRef: taskRef,
				},
				DeliveryRefs: orquestaruntime.AgentStartDeliveryRefsV0{
					AckRef: ackRef,
				},
			},
		},
	}
}

func codexStackRealSmokeDescriptorTaskRefV0(
	descriptor orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
) string {
	for _, value := range []string{
		descriptor.Spec.AgentPacket.Task.TaskRef,
		descriptor.Spec.AgentPacket.WorkOrderRef,
	} {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func codexStackRealSmokeAllDescriptorsStartedV0(
	started []string,
	descriptors []orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
) bool {
	startedSet := map[string]bool{}
	for _, ref := range compactStringsV0(started) {
		startedSet[ref] = true
	}
	if len(startedSet) < len(descriptors) {
		return false
	}
	for _, descriptor := range descriptors {
		if !startedSet[descriptor.AgentRef] {
			return false
		}
	}
	return true
}

func codexStackRealSmokeAllDescriptorsRequestedAndStartedV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	descriptors []orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
) bool {
	return codexStackRealSmokeAllDescriptorsStartedV0(run.Agents, descriptors) &&
		codexStackRealSmokeAllDescriptorsStartedV0(run.StartedAgents, descriptors)
}

func codexStackRealSmokeHasEventV0(
	events []orquestacoreworkflow.OrchestrationEventV0,
	eventType string,
) bool {
	return codexStackRealSmokeEventCountV0(events, eventType) > 0
}

func codexStackRealSmokeEventCountV0(
	events []orquestacoreworkflow.OrchestrationEventV0,
	eventType string,
) int {
	count := 0
	for _, event := range events {
		if event.EventType == eventType {
			count++
		}
	}
	return count
}

func codexStackRealSmokeContainsProjectionPartV0(values []string, want string) bool {
	want = strings.TrimSpace(want)
	for _, value := range values {
		if strings.Contains(value, want) {
			return true
		}
	}
	return false
}
