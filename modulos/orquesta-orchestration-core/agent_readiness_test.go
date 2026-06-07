package orquestacionnucleoapp

import (
	"context"
	"testing"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func TestProbeAgentReadinessV0UsesFakePort(t *testing.T) {
	probe := &fakeAgentReadinessProbeV0{
		result: AgentReadinessProbeResultV0{
			Status:       AgentReadinessReadyV0,
			EvidenceRefs: []string{"readiness-evidence-ref-001"},
		},
	}
	request := validAgentReadinessProbeRequestV0()

	result, err := ProbeAgentReadinessV0(context.Background(), probe, request)
	if err != nil {
		t.Fatalf("probe readiness: %v", err)
	}

	if !probe.called {
		t.Fatal("probe no invocado")
	}
	if probe.request.ReadinessRef != request.ReadinessRef {
		t.Fatalf("request=%+v", probe.request)
	}
	if result.Status != AgentReadinessReadyV0 ||
		!containsNucleoRefV0(result.EvidenceRefs, "readiness-evidence-ref-001") {
		t.Fatalf("result=%+v", result)
	}
}

func TestExternalProcessAgentLauncherV0WaitsForReadinessBeforeReturning(t *testing.T) {
	inbound := externalProcessLauncherInboundV0("run-nucleo-readiness-001", "agent-ref-readiness-001")
	probe := &fakeAgentReadinessProbeV0{
		result: AgentReadinessProbeResultV0{
			Status:       AgentReadinessReadyV0,
			EvidenceRefs: []string{"readiness-evidence-ref-launcher-001"},
		},
	}
	runtime := fakeReadinessExternalProcessRuntimeV0{
		snapshot: fakeReadinessProcessSnapshotV0(),
	}
	launcher := ExternalProcessAgentLauncherV0{
		SpecResolver: externalAgentLaunchSpecResolverForTestV0(
			externalProcessRuntimeRequestForTestV0(t, "exit"),
		),
		Runtime:         runtime,
		ProcessStopper:  runtime,
		Readiness:       probe,
		ProcessRegistry: NewInMemoryAgentProcessRegistryV0(),
	}

	result, err := launcher.LaunchAgentV0(context.Background(), inbound)
	if err != nil {
		t.Fatalf("launch agent: %v", err)
	}

	if !probe.called {
		t.Fatal("readiness probe no invocado")
	}
	if probe.request.ProcessRef != runtime.snapshot.ProcessRef ||
		probe.request.SessionRef != runtime.snapshot.SessionRef ||
		probe.request.LaunchRef != runtime.snapshot.LaunchRef {
		t.Fatalf("readiness request=%+v snapshot=%+v", probe.request, runtime.snapshot)
	}
	if !containsNucleoRefV0(result.EvidenceRefs, "readiness-evidence-ref-launcher-001") {
		t.Fatalf("evidence refs sin readiness probe: %+v", result.EvidenceRefs)
	}
}

func TestExternalProcessAgentLauncherV0BlocksWhenReadinessNotReady(t *testing.T) {
	inbound := externalProcessLauncherInboundV0("run-nucleo-readiness-block-001", "agent-ref-readiness-block-001")
	launcher := ExternalProcessAgentLauncherV0{
		SpecResolver: externalAgentLaunchSpecResolverForTestV0(
			externalProcessRuntimeRequestForTestV0(t, "exit"),
		),
		Runtime:         fakeReadinessExternalProcessRuntimeV0{snapshot: fakeReadinessProcessSnapshotV0()},
		ProcessStopper:  fakeReadinessExternalProcessRuntimeV0{snapshot: fakeReadinessProcessSnapshotV0()},
		ProcessRegistry: NewInMemoryAgentProcessRegistryV0(),
		Readiness: &fakeAgentReadinessProbeV0{
			result: AgentReadinessProbeResultV0{Status: AgentReadinessNotReadyV0},
		},
	}

	_, err := launcher.LaunchAgentV0(context.Background(), inbound)

	assertNucleoErrorV0(t, err, ErrNucleoOrquestacionInvalidoV0, "readiness_probe.status")
}

func TestProbeAgentReadinessV0RejectsNonOpaqueRefs(t *testing.T) {
	request := validAgentReadinessProbeRequestV0()
	request.ProcessRef = "/home/alberto/proc"

	_, err := ProbeAgentReadinessV0(context.Background(), &fakeAgentReadinessProbeV0{}, request)

	assertNucleoErrorV0(t, err, ErrNucleoOrquestacionInvalidoV0, "agent_readiness.process_ref")
}

func TestProbeAgentReadinessV0PermiteVocabularioOperativoOpaco(t *testing.T) {
	request := validAgentReadinessProbeRequestV0()
	request.ProcessRef = "process-ref-oauth-token-budget-provider-model-001"
	request.SessionRef = "session-ref-secrets-policy-runtime-codex-001"
	request.EvidenceRefs = []string{
		"readiness-evidence-ref-token-budget-001",
		"readiness-evidence-ref-secrets-policy-001",
	}
	probe := &fakeAgentReadinessProbeV0{
		result: AgentReadinessProbeResultV0{
			Status:       AgentReadinessReadyV0,
			EvidenceRefs: request.EvidenceRefs,
		},
	}

	result, err := ProbeAgentReadinessV0(context.Background(), probe, request)
	if err != nil {
		t.Fatalf("probe readiness con vocabulario operativo opaco: %v", err)
	}
	if result.Status != AgentReadinessReadyV0 ||
		!containsNucleoRefV0(result.EvidenceRefs, "readiness-evidence-ref-secrets-policy-001") {
		t.Fatalf("result=%+v", result)
	}
}

func TestProbeAgentReadinessV0AceptaValorSensibleEnRefOpaca(t *testing.T) {
	enableOrchestrationCoreRailsModeEnforcedForTestV0(t)
	request := validAgentReadinessProbeRequestV0()
	request.EvidenceRefs = []string{"client-secret:valor"}
	probe := &fakeAgentReadinessProbeV0{
		result: AgentReadinessProbeResultV0{
			Status:       AgentReadinessReadyV0,
			EvidenceRefs: request.EvidenceRefs,
		},
	}

	result, err := ProbeAgentReadinessV0(context.Background(), probe, request)

	if err != nil {
		t.Fatalf("probe readiness no debe bloquear por contenido: %v", err)
	}
	if !containsNucleoRefV0(result.EvidenceRefs, "client-secret:valor") {
		t.Fatalf("result no conserva evidence refs: %+v", result)
	}
}

func TestProbeAgentReadinessV0AceptaRefsInternasAnidadas(t *testing.T) {
	request := validAgentReadinessProbeRequestV0()
	request.AgentRequestID = "agent-ref-assessment-assessment-ref-agent-progress-report-ref-agent-ref-assessment-assessment-ref-agent-progress-report-ref-agent-ref-task-autoprogramming-d6f0b05f2e4d-g01-000088-000016"
	request.ProcessRef = "process-ref-assessment-assessment-ref-agent-progress-report-ref-agent-ref-assessment-assessment-ref-agent-progress-report-ref-agent-ref-task-autoprogramming-d6f0b05f2e4d-g01-000088-000016"
	request.EvidenceRefs = []string{
		"readiness-evidence-ref-assessment-ref-agent-progress-report-ref-agent-ref-assessment-assessment-ref-agent-progress-report-ref-agent-ref-task-autoprogramming-d6f0b05f2e4d-g01-000088-000016",
	}
	probe := &fakeAgentReadinessProbeV0{
		result: AgentReadinessProbeResultV0{
			Status:       AgentReadinessReadyV0,
			EvidenceRefs: request.EvidenceRefs,
		},
	}

	result, err := ProbeAgentReadinessV0(context.Background(), probe, request)
	if err != nil {
		t.Fatalf("probe readiness con refs anidadas: %v", err)
	}
	if result.Status != AgentReadinessReadyV0 {
		t.Fatalf("result=%+v", result)
	}
}

func validAgentReadinessProbeRequestV0() AgentReadinessProbeRequestV0 {
	return AgentReadinessProbeRequestV0{
		RunID:          "run-readiness-001",
		AgentRequestID: "agent-readiness-001",
		ProcessRef:     "process-ref-readiness-001",
		SessionRef:     "session-ref-readiness-001",
		LaunchRef:      "launch-ref-readiness-001",
		ReadinessRef:   "readiness-ref-readiness-001",
		EvidenceRefs:   []string{"evidence-ref-readiness-001"},
	}
}

func fakeReadinessProcessSnapshotV0() orquestaruntime.ProcessRuntimeSnapshotV0 {
	return orquestaruntime.ProcessRuntimeSnapshotV0{
		SchemaVersion: orquestaruntime.ProcessRuntimeConnectorVersionV0,
		ProcessRef:    "process-ref-readiness-launcher-001",
		SessionRef:    "session-ref-readiness-launcher-001",
		LaunchRef:     "launch-ref-readiness-launcher-001",
		Status:        orquestaruntime.ProcessRuntimeRunningV0,
	}
}

type fakeAgentReadinessProbeV0 struct {
	result  AgentReadinessProbeResultV0
	request AgentReadinessProbeRequestV0
	called  bool
}

func (probe *fakeAgentReadinessProbeV0) AwaitAgentReadinessV0(
	_ context.Context,
	request AgentReadinessProbeRequestV0,
) (AgentReadinessProbeResultV0, error) {
	probe.called = true
	probe.request = request
	return probe.result, nil
}

type fakeReadinessExternalProcessRuntimeV0 struct {
	snapshot orquestaruntime.ProcessRuntimeSnapshotV0
}

func (runtime fakeReadinessExternalProcessRuntimeV0) LaunchV0(
	context.Context,
	orquestaruntime.ProcessRuntimeLaunchRequestV0,
) (orquestaruntime.ProcessRuntimeSnapshotV0, error) {
	return runtime.snapshot, nil
}

func (runtime fakeReadinessExternalProcessRuntimeV0) StopV0(
	context.Context,
	string,
) (orquestaruntime.ProcessRuntimeSnapshotV0, error) {
	snapshot := runtime.snapshot
	snapshot.Status = orquestaruntime.ProcessRuntimeStoppedV0
	snapshot.StopRef = "stop-ref-readiness-launcher-001"
	return snapshot, nil
}
