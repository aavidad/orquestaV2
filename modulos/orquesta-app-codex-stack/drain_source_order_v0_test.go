package orquestaappcodexstack

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
)

func TestDrainRunV0RegistraACKDirectorAntesDeConsumirDecisionFile(t *testing.T) {
	runtime := newDecisionBeforeDirectorACKCodexStackRuntimeV0()
	stack := mustBuildCodexStackForTestV0(t, runtime)
	stack.Ports.ExternalWaiter = nil

	director := postDirectorAPIV0(t, stack)
	if runtime.launchCountV0() != 4 {
		t.Fatalf("launches iniciales=%d want=4", runtime.launchCountV0())
	}
	if _, err := stack.DrainRunV0(context.Background(), DrainRunRequestV0{
		RunRef:        director.RunRef,
		CorrelationID: "corr-stack-source-order-before-ack",
	}); err != nil {
		t.Fatalf("DrainRunV0 antes de ACK director: %v", err)
	}
	run := mustLoadCodexStackRunForTestV0(t, stack, director.RunRef)
	directorDescriptor := codexStackDirectorDescriptorForOrderTestV0(t, stack)
	directorAckRef := directorDescriptor.Spec.AgentPacket.DeliveryRefs.AckRef
	if run.CurrentPhase == orquestacoreworkflow.OrchestrationPhaseProgramacionV0 {
		t.Fatalf("decision consumida antes de ACK director: phase=%s artifacts=%v",
			run.CurrentPhase,
			run.PhaseArtifacts,
		)
	}
	if codexStackProjectionContainsPartForOrderTestV0(run.PhaseArtifacts, directorAckRef) {
		t.Fatalf("ACK director ya reflejado antes de escribirlo: artifacts=%v", run.PhaseArtifacts)
	}
	if runtime.launchCountV0() != 4 {
		t.Fatalf("programacion arrancada antes de ACK director: launches=%d", runtime.launchCountV0())
	}

	eventSink := codexStackEventSinkForOrderTestV0(t, stack)
	eventsBeforeDirectorACK := len(eventSink.EventsV0())
	if err := writeCodexStackAckForDescriptorV0(t, directorDescriptor); err != nil {
		t.Fatalf("write ACK director: %v", err)
	}
	if _, err := stack.DrainRunV0(context.Background(), DrainRunRequestV0{
		RunRef:        director.RunRef,
		CorrelationID: "corr-stack-source-order-after-ack",
	}); err != nil {
		t.Fatalf("DrainRunV0 tras ACK director: %v", err)
	}
	run = mustLoadCodexStackRunForTestV0(t, stack, director.RunRef)
	if !codexStackProjectionContainsPartForOrderTestV0(run.PhaseArtifacts, directorAckRef) {
		t.Fatalf("ACK director no reflejado: artifacts=%v missing=%s",
			run.PhaseArtifacts,
			directorAckRef,
		)
	}
	if run.CurrentPhase != orquestacoreworkflow.OrchestrationPhaseProgramacionV0 {
		t.Fatalf("current_phase=%s", run.CurrentPhase)
	}
	if runtime.launchCountV0() < 5 {
		t.Fatalf("programacion no arrancada tras ACK director: launches=%d", runtime.launchCountV0())
	}
	codexStackAssertDirectorArtifactBeforePhaseOpenV0(
		t,
		eventSink.EventsV0()[eventsBeforeDirectorACK:],
		directorAckRef,
	)
}

type decisionBeforeDirectorACKCodexStackRuntimeV0 struct {
	*fakeCodexStackRuntimeV0
}

func newDecisionBeforeDirectorACKCodexStackRuntimeV0() *decisionBeforeDirectorACKCodexStackRuntimeV0 {
	return &decisionBeforeDirectorACKCodexStackRuntimeV0{
		fakeCodexStackRuntimeV0: newFakeCodexStackRuntimeV0(),
	}
}

func (runtime *decisionBeforeDirectorACKCodexStackRuntimeV0) LaunchV0(
	ctx context.Context,
	req orquestaruntime.ProcessRuntimeLaunchRequestV0,
) (orquestaruntime.ProcessRuntimeSnapshotV0, error) {
	packet, err := codexStackPacketFromRuntimeDirForTestV0(filepath.Dir(req.CommandPath))
	if err != nil {
		return orquestaruntime.ProcessRuntimeSnapshotV0{}, err
	}
	if packet.TargetModule != "orquesta-app-stack-director" {
		return runtime.fakeCodexStackRuntimeV0.LaunchV0(ctx, req)
	}
	if _, err := runtime.writeDeliveryFilesV0(req.WorkingDir, packet.Task.WriteSet); err != nil {
		return orquestaruntime.ProcessRuntimeSnapshotV0{}, err
	}
	snapshot, err := runtime.launchSnapshotWithoutACKV0("process-ref-app-stack-director-pending-ack-")
	if err != nil {
		return snapshot, err
	}
	decisionRuntime := decisionWritingFakeCodexStackRuntimeV0{
		fakeCodexStackRuntimeV0: runtime.fakeCodexStackRuntimeV0,
	}
	return snapshot, decisionRuntime.writeDirectorDecisionsV0(req)
}

func codexStackDirectorDescriptorForOrderTestV0(
	t *testing.T,
	stack StackV0,
) orquestaruntimecodexdelivery.CodexReceiptDescriptorV0 {
	t.Helper()
	descriptors := codexStackDescriptorsForTestV0(t, stack)
	for _, descriptor := range descriptors {
		if descriptor.Spec.AgentPacket.TargetModule == "orquesta-app-stack-director" {
			return descriptor
		}
	}
	t.Fatalf("descriptor director no encontrado: %v", codexStackRealSmokeDescriptorAgentsV0(descriptors))
	return orquestaruntimecodexdelivery.CodexReceiptDescriptorV0{}
}

func codexStackEventSinkForOrderTestV0(
	t *testing.T,
	stack StackV0,
) *orquestacionnucleoapp.InMemoryEventSinkV0 {
	t.Helper()
	sink, ok := stack.Stores.EventSink.(*orquestacionnucleoapp.InMemoryEventSinkV0)
	if !ok {
		t.Fatalf("event sink inesperado: %T", stack.Stores.EventSink)
	}
	return sink
}

func codexStackProjectionContainsPartForOrderTestV0(values []string, part string) bool {
	part = strings.TrimSpace(part)
	for _, value := range values {
		if strings.Contains(strings.TrimSpace(value), part) {
			return true
		}
	}
	return false
}

func codexStackAssertDirectorArtifactBeforePhaseOpenV0(
	t *testing.T,
	events []orquestacoreworkflow.OrchestrationEventV0,
	directorAckRef string,
) {
	t.Helper()
	artifactIndex := -1
	phaseOpenIndex := -1
	for i, event := range events {
		if event.EventType == orquestacoreworkflow.OrchestrationEventPhaseArtifactRegisteredV0 &&
			strings.Contains(string(event.Payload), directorAckRef) {
			artifactIndex = i
		}
		if event.EventType == orquestacoreworkflow.OrchestrationEventPhaseOpenedV0 && phaseOpenIndex == -1 {
			phaseOpenIndex = i
		}
	}
	if artifactIndex == -1 {
		t.Fatalf("evento PhaseArtifactRegistered director no encontrado en eventos=%+v", events)
	}
	if phaseOpenIndex == -1 {
		t.Fatalf("evento PhaseOpened no encontrado tras ACK director: eventos=%+v", events)
	}
	if artifactIndex > phaseOpenIndex {
		t.Fatalf("orden causal invalido: artifact_index=%d phase_open_index=%d eventos=%+v",
			artifactIndex,
			phaseOpenIndex,
			events,
		)
	}
}
