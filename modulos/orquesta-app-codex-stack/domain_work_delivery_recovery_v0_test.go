package orquestaappcodexstack

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
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
	result := postExternalWorkRunStackLegacyV0(t, stack)

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

func TestRecoverDomainWorkAckV0RecuperaArtefactoValidoSinSenalDeLog(t *testing.T) {
	fixture := newDomainWorkRecoveryAckFixtureForTestV0(t, true)

	ack, ok := recoverDomainWorkAckV0(fixture.descriptor, fixture.task, fixture.record)

	if !ok ||
		ack.AckRef != fixture.descriptor.Spec.AgentPacket.DeliveryRefs.AckRef ||
		!stringInSetV0([]string(ack.Files), fixture.fileRef) ||
		!stringInSetV0([]string(ack.Notes), "domain_work_ack_recovered_from_valid_artifact") {
		t.Fatalf("ack=%+v ok=%v fixture=%+v", ack, ok, fixture)
	}
}

func TestRecoverDomainWorkAckV0NoRecuperaSinContratoOArtefacto(t *testing.T) {
	fixture := newDomainWorkRecoveryAckFixtureForTestV0(t, true)
	taskWithoutContract := fixture.task
	taskWithoutContract.FunctionContractRefs = nil
	if ack, ok := recoverDomainWorkAckV0(fixture.descriptor, taskWithoutContract, fixture.record); ok {
		t.Fatalf("recupero sin contrato: %+v", ack)
	}

	fixture = newDomainWorkRecoveryAckFixtureForTestV0(t, false)
	if ack, ok := recoverDomainWorkAckV0(fixture.descriptor, fixture.task, fixture.record); ok {
		t.Fatalf("recupero sin artefacto: %+v", ack)
	}
}

func TestCodexStackV0RunGlobalTickRecuperaDomainWorkParadoAntesFiltroControl(t *testing.T) {
	runtime := newDomainWorkRunningMissingACKRuntimeV0()
	domainWork := &fakeCodexStackDomainWorkExecutorV0{}
	stack := mustBuildCodexStackWithDomainWorkForTestV0(t, runtime, domainWork)
	result := postExternalWorkRunStackLegacyV0(t, stack)

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
		return
	}
	markDomainWorkRunStoppedForTestV0(t, stack, result.RunRef)
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

func TestCodexStackV0RunGlobalTickRecuperaDomainWorkPerdidoConArtefactoValidoV0(t *testing.T) {
	runtime := newDomainWorkRunningMissingACKRuntimeV0()
	domainWork := &fakeCodexStackDomainWorkExecutorV0{}
	stack := mustBuildCodexStackWithDomainWorkForTestV0(t, runtime, domainWork)
	result := postExternalWorkRunStackLegacyV0(t, stack)

	if _, err := stack.DrainRunV0(context.Background(), DrainRunRequestV0{
		RunRef:               result.RunRef,
		CorrelationID:        "corr-domain-work-lost-prep",
		MaxBursts:            16,
		MaxStepsPerBurst:     8,
		MaxDispatchesPerWait: 8,
		MaxExternalWaits:     0,
	}); err != nil {
		t.Fatalf("DrainRunV0: %v (%#v)", err, err)
	}
	if _, ok := findDomainWorkSubmitForTestV0(domainWork.inputs); ok {
		return
	}
	markDomainWorkRunLostForTestV0(t, stack, result.RunRef)
	if _, err := stack.Stores.RunControl.CompleteRunControlV0(
		context.Background(),
		orquestaruncontrol.CompleteRunControlCommandV0{
			RunRef:       result.RunRef,
			TargetStatus: orquestaruncontrol.RunControlStatusStoppedV0,
			RequestedBy:  "stack-test",
			Reason:       "simula run terminal historico con artefacto valido y agente perdido",
		},
	); err != nil {
		t.Fatalf("CompleteRunControlV0: %v", err)
	}

	if _, err := stack.RunGlobalTickV0(context.Background(), globalTickCommandForTestV0()); err != nil {
		t.Fatalf("RunGlobalTickV0: %v", err)
	}
	if _, ok := findDomainWorkSubmitForTestV0(domainWork.inputs); !ok {
		t.Fatalf("submit_artifact no invocado para agente perdido: inputs=%+v", domainWork.inputs)
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

func markDomainWorkRunLostForTestV0(t *testing.T, stack StackV0, runRef string) {
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
	run.LostAgents = append(run.LostAgents, descriptors[0].AgentRef)
	if err := stack.Stores.RunStore.SaveRunV0(context.Background(), run); err != nil {
		t.Fatalf("SaveRunV0: %v", err)
	}
}

type domainWorkRecoveryAckFixtureV0 struct {
	descriptor orquestaruntimecodexdelivery.CodexReceiptDescriptorV0
	task       orquestacoreworkflow.WorkflowTaskV0
	record     orquestaappchange.AppChangeRecordV0
	fileRef    string
}

func newDomainWorkRecoveryAckFixtureForTestV0(
	t *testing.T,
	withArtifact bool,
) domainWorkRecoveryAckFixtureV0 {
	t.Helper()
	projectDir := t.TempDir()
	runtimeDir := t.TempDir()
	runRef := "run-ref-domain-work-recovery-ack-001"
	taskRef := "task-ref-domain-work-recovery-ack-001"
	fileRef := "external/opes/job-001/entrega.md"
	if withArtifact {
		path := filepath.Join(projectDir, filepath.FromSlash(fileRef))
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatalf("mkdir artifact: %v", err)
		}
		if err := os.WriteFile(path, []byte("entrega fake para revision"), 0o600); err != nil {
			t.Fatalf("write artifact: %v", err)
		}
	}
	spec := orquestaruntime.ExternalAgentLaunchSpecV0{
		RequestID:     "agent-ref-domain-work-recovery-ack-001",
		CorrelationID: "corr-domain-work-recovery-ack-001",
		AgentPacket: orquestaruntime.AgentStartPacketV0{
			SchemaVersion: orquestaruntime.AgentStartPacketSchemaVersionV0,
			RequestID:     "agent-ref-domain-work-recovery-ack-001",
			CorrelationID: "corr-domain-work-recovery-ack-001",
			TargetModule:  "orquesta-app-stack-programacion",
			Phase:         string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			Task: orquestaruntime.AgentStartTaskV0{
				TaskRef:  taskRef,
				WriteSet: []string{"external/opes/job-001"},
			},
			DeliveryRefs: orquestaruntime.AgentStartDeliveryRefsV0{
				AckRef: "ack-ref-domain-work-recovery-ack-001",
			},
		},
	}
	return domainWorkRecoveryAckFixtureV0{
		descriptor: orquestaruntimecodexdelivery.CodexReceiptDescriptorV0{
			DescriptorRef:  "descriptor-ref-domain-work-recovery-ack-001",
			RunID:          runRef,
			AgentRef:       "agent-ref-domain-work-recovery-ack-001",
			AckPath:        filepath.Join(runtimeDir, orquestaruntimecodex.CodexAgentAckFileNameV0),
			ProjectWorkDir: projectDir,
			Spec:           spec,
		},
		task: orquestacoreworkflow.WorkflowTaskV0{
			TaskID: taskRef,
			RunID:  runRef,
			FunctionContractRefs: []orquestacoreworkflow.WorkflowFunctionContractRefV0{{
				FunctionName: "ApplyExternalDomainWorkV0",
			}},
		},
		record: orquestaappchange.AppChangeRecordV0{
			Request: orquestaappchange.AppChangeRequestV0{
				RunRef: runRef,
				ExternalWork: &orquestaappchange.AppChangeExternalWorkV0{
					ProjectRef: "opes",
					JobRef:     "job-ref-domain-work-recovery-ack-001",
					WorkKind:   "draft_content_block",
				},
			},
		},
		fileRef: fileRef,
	}
}
