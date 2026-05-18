package orquestaappcodexstack

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
)

func TestCodexStackV0OPESExternalWorkRecuperaEntregaSinACKConArtefactoValido(t *testing.T) {
	runtime := newDomainWorkMissingACKRuntimeV0()
	domainWork := &fakeCodexStackDomainWorkExecutorV0{}
	stack := mustBuildCodexStackWithDomainWorkForTestV0(t, runtime, domainWork)
	result := postExternalWorkRunStackV0(t, stack)

	for attempt := 1; attempt <= 6; attempt++ {
		if _, err := stack.DrainRunV0(context.Background(), DrainRunRequestV0{
			RunRef:               result.RunRef,
			CorrelationID:        "corr-domain-work-recovery-drain",
			MaxBursts:            16,
			MaxStepsPerBurst:     8,
			MaxDispatchesPerWait: 8,
			MaxExternalWaits:     2,
		}); err != nil {
			t.Fatalf("DrainRunV0 intento %d: %v (%#v)", attempt, err, err)
		}
		if _, ok := findDomainWorkSubmitForTestV0(domainWork.inputs); ok {
			break
		}
	}

	submit, ok := findDomainWorkSubmitForTestV0(domainWork.inputs)
	if !ok {
		run, _ := stack.Stores.RunStore.LoadRunV0(context.Background(), result.RunRef)
		t.Fatalf("submit_artifact no invocado tras recovery: run=%+v inputs=%+v", run, domainWork.inputs)
	}
	if submit.Action != orquestamcp.MCPDomainWorkActionSubmitArtifactV0 ||
		submit.ArtifactSubmission.JobRef != "job-ref-001" ||
		!domainWorkFieldValueForTestV0(submit.ArtifactSubmission.PayloadFields, "body", "entrega fake para revision") {
		t.Fatalf("submit=%+v", submit)
	}
}

type domainWorkMissingACKRuntimeV0 struct {
	*fakeCodexStackRuntimeV0
	writeAckFailureMessage bool
	returnStopped          bool
}

func newDomainWorkMissingACKRuntimeV0() *domainWorkMissingACKRuntimeV0 {
	return &domainWorkMissingACKRuntimeV0{
		fakeCodexStackRuntimeV0: newFakeCodexStackRuntimeV0(),
		writeAckFailureMessage:  true,
		returnStopped:           true,
	}
}

func newDomainWorkRunningMissingACKRuntimeV0() *domainWorkMissingACKRuntimeV0 {
	return &domainWorkMissingACKRuntimeV0{
		fakeCodexStackRuntimeV0: newFakeCodexStackRuntimeV0(),
	}
}

func (runtime *domainWorkMissingACKRuntimeV0) LaunchV0(
	ctx context.Context,
	req orquestaruntime.ProcessRuntimeLaunchRequestV0,
) (orquestaruntime.ProcessRuntimeSnapshotV0, error) {
	runtimeDir := filepath.Dir(req.CommandPath)
	packet, err := codexStackPacketFromRuntimeDirForTestV0(runtimeDir)
	if err != nil {
		return orquestaruntime.ProcessRuntimeSnapshotV0{}, err
	}
	if packet.TargetModule != "orquesta-app-stack-programacion" {
		return runtime.fakeCodexStackRuntimeV0.LaunchV0(ctx, req)
	}
	if _, err := runtime.writeDeliveryFilesV0(req.WorkingDir, packet.Task.WriteSet); err != nil {
		return orquestaruntime.ProcessRuntimeSnapshotV0{}, err
	}
	if runtime.writeAckFailureMessage {
		if err := os.WriteFile(
			filepath.Join(runtimeDir, orquestaruntimecodex.CodexLastMessageFileNameV0),
			[]byte("ACK "+packet.DeliveryRefs.AckRef+" failed"),
			0o600,
		); err != nil {
			return orquestaruntime.ProcessRuntimeSnapshotV0{}, err
		}
	}
	snapshot, err := runtime.launchSnapshotWithoutACKV0("process-ref-app-stack-domain-noack-")
	if err != nil {
		return snapshot, err
	}
	if !runtime.returnStopped {
		return snapshot, nil
	}
	runtime.mu.Lock()
	snapshot.Status = orquestaruntime.ProcessRuntimeStoppedV0
	runtime.snapshots[snapshot.ProcessRef] = snapshot
	runtime.mu.Unlock()
	return snapshot, nil
}

func TestDomainWorkRecoveryArtifactFilesV0AceptaDirectorioPermitido(t *testing.T) {
	projectDir := t.TempDir()
	path := filepath.Join(projectDir, "external", "opes", "job-001", "entrega.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte("contenido"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	files, ok := domainWorkRecoveryArtifactFilesV0(projectDir, []string{"external/opes/job-001"})
	if !ok || len(files) != 1 || !strings.HasSuffix(files[0], "entrega.md") {
		t.Fatalf("files=%v ok=%v", files, ok)
	}
}

func TestDomainWorkRecoveryDirectSubmitEligibleV0RechazaAgenteAssessment(t *testing.T) {
	run := orquestacoreworkflow.OrchestrationRunV0{
		StoppedAgents: []string{
			"agent-ref-assessment-assessment-ref-001",
			"agent-ref-task-ref-app-change-001",
		},
	}
	assessment := orquestaruntimecodexdelivery.CodexReceiptDescriptorV0{
		AgentRef: "agent-ref-assessment-assessment-ref-001",
		Spec: orquestaruntime.ExternalAgentLaunchSpecV0{
			AgentPacket: orquestaruntime.AgentStartPacketV0{
				DeliveryRefs: orquestaruntime.AgentStartDeliveryRefsV0{AckRef: "ack-ref-assessment-001"},
			},
		},
	}
	original := assessment
	original.AgentRef = "agent-ref-task-ref-app-change-001"
	original.Spec.AgentPacket.DeliveryRefs.AckRef = "ack-ref-original-001"

	if domainWorkRecoveryDirectSubmitEligibleV0(run, assessment) {
		t.Fatalf("un agente de assessment no debe enviar artefacto de dominio recuperado")
	}
	if !domainWorkRecoveryDirectSubmitEligibleV0(run, original) {
		t.Fatalf("un agente de trabajo externo parado con ack pendiente debe poder recuperarse")
	}
}

func TestDomainWorkRecoveryAckFailureDetectaSandboxEnStderr(t *testing.T) {
	runtimeDir := t.TempDir()
	descriptor := orquestaruntimecodexdelivery.CodexReceiptDescriptorV0{
		AckPath: filepath.Join(runtimeDir, orquestaruntimecodex.CodexAgentAckFileNameV0),
		Spec: orquestaruntime.ExternalAgentLaunchSpecV0{
			AgentPacket: orquestaruntime.AgentStartPacketV0{
				DeliveryRefs: orquestaruntime.AgentStartDeliveryRefsV0{AckRef: "ack-ref-001"},
			},
		},
	}
	if err := os.WriteFile(
		filepath.Join(runtimeDir, orquestaruntimecodex.CodexStderrFileNameV0),
		[]byte("patch rejected: writing outside of the project"),
		0o600,
	); err != nil {
		t.Fatalf("write stderr: %v", err)
	}
	if !domainWorkRecoveryLastMessageConfirmsAckFailureV0(descriptor) {
		t.Fatalf("stderr de sandbox debe habilitar recovery neutral de ACK")
	}
}

func TestDomainWorkRecoveryAckFailureDetectaTurnInterruptedEnStderr(t *testing.T) {
	runtimeDir := t.TempDir()
	descriptor := orquestaruntimecodexdelivery.CodexReceiptDescriptorV0{
		AckPath: filepath.Join(runtimeDir, orquestaruntimecodex.CodexAgentAckFileNameV0),
		Spec: orquestaruntime.ExternalAgentLaunchSpecV0{
			AgentPacket: orquestaruntime.AgentStartPacketV0{
				DeliveryRefs: orquestaruntime.AgentStartDeliveryRefsV0{AckRef: "ack-ref-001"},
			},
		},
	}
	if err := os.WriteFile(
		filepath.Join(runtimeDir, orquestaruntimecodex.CodexStderrFileNameV0),
		[]byte("turn interrupted\ntokens used\n47,498"),
		0o600,
	); err != nil {
		t.Fatalf("write stderr: %v", err)
	}
	if !domainWorkRecoveryLastMessageConfirmsAckFailureV0(descriptor) {
		t.Fatalf("turn interrupted debe habilitar recovery neutral de ACK")
	}
}

func TestCodexStackV0RunGlobalTickRecuperaDomainWorkParadoAntesFiltroControl(t *testing.T) {
	runtime := newDomainWorkRunningMissingACKRuntimeV0()
	domainWork := &fakeCodexStackDomainWorkExecutorV0{}
	stack := mustBuildCodexStackWithDomainWorkForTestV0(t, runtime, domainWork)
	result := postExternalWorkRunStackV0(t, stack)

	if _, err := stack.DrainRunV0(context.Background(), DrainRunRequestV0{
		RunRef:               result.RunRef,
		CorrelationID:        "corr-domain-work-terminal-prep",
		MaxBursts:            16,
		MaxStepsPerBurst:     8,
		MaxDispatchesPerWait: 8,
		MaxExternalWaits:     0,
	}); err != nil {
		t.Fatalf("DrainRunV0: %v (%#v)", err, err)
	}
	if _, ok := findDomainWorkSubmitForTestV0(domainWork.inputs); ok {
		t.Fatalf("submit_artifact no debe ocurrir sin evidencia de fallo de ACK")
	}
	markDomainWorkRunStoppedForTestV0(t, stack, result.RunRef)
	writeDomainWorkAckFailureLastMessageForTestV0(t, stack, result.RunRef)
	if _, err := stack.Stores.RunControl.CompleteRunControlV0(
		context.Background(),
		orquestaruncontrol.CompleteRunControlCommandV0{
			RunRef:       result.RunRef,
			TargetStatus: orquestaruncontrol.RunControlStatusStoppedV0,
			RequestedBy:  "stack-test",
			Reason:       "simula run terminal historico con artefacto sin ACK",
		},
	); err != nil {
		t.Fatalf("CompleteRunControlV0: %v", err)
	}

	if _, err := stack.RunGlobalTickV0(context.Background(), globalTickCommandForTestV0()); err != nil {
		t.Fatalf("RunGlobalTickV0: %v", err)
	}
	if _, ok := findDomainWorkSubmitForTestV0(domainWork.inputs); !ok {
		t.Fatalf("submit_artifact no invocado para run terminal: inputs=%+v", domainWork.inputs)
	}
}

func markDomainWorkRunStoppedForTestV0(t *testing.T, stack StackV0, runRef string) {
	t.Helper()
	descriptors, err := stack.Stores.ReceiptStore.ListCodexReceiptDescriptorsV0(
		context.Background(),
		orquestaruntimecodexdelivery.CodexReceiptDescriptorRequestV0{RunID: runRef},
	)
	if err != nil {
		t.Fatalf("ListCodexReceiptDescriptorsV0: %v", err)
	}
	if len(descriptors) != 1 {
		t.Fatalf("descriptors=%+v", descriptors)
	}
	run, err := stack.Stores.RunStore.LoadRunV0(context.Background(), runRef)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	run.StoppedAgents = append(run.StoppedAgents, descriptors[0].AgentRef)
	run.ConfirmedStoppedAgents = append(run.ConfirmedStoppedAgents, descriptors[0].AgentRef)
	if err := stack.Stores.RunStore.SaveRunV0(context.Background(), run); err != nil {
		t.Fatalf("SaveRunV0: %v", err)
	}
}

func writeDomainWorkAckFailureLastMessageForTestV0(
	t *testing.T,
	stack StackV0,
	runRef string,
) {
	t.Helper()
	descriptors, err := stack.Stores.ReceiptStore.ListCodexReceiptDescriptorsV0(
		context.Background(),
		orquestaruntimecodexdelivery.CodexReceiptDescriptorRequestV0{RunID: runRef},
	)
	if err != nil {
		t.Fatalf("ListCodexReceiptDescriptorsV0: %v", err)
	}
	if len(descriptors) != 1 {
		t.Fatalf("descriptors=%+v", descriptors)
	}
	ackRef := descriptors[0].Spec.AgentPacket.DeliveryRefs.AckRef
	if err := os.WriteFile(
		filepath.Join(filepath.Dir(descriptors[0].AckPath), orquestaruntimecodex.CodexLastMessageFileNameV0),
		[]byte("ACK "+ackRef+" failed"),
		0o600,
	); err != nil {
		t.Fatalf("write last message: %v", err)
	}
}
