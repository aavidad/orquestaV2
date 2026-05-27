package orquestamcp

import (
	"context"
	"strings"
	"testing"

	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
)

func TestMCPRunControlDescriptorV0EsAdaptadorFino(t *testing.T) {
	descriptor := MCPRunControlDescriptorV0()
	if descriptor.Name != MCPRunControlToolNameV0 ||
		descriptor.ResourceURI != MCPRunControlResourceURIV0 ||
		descriptor.InputSchema == "" ||
		len(descriptor.Invariantes) == 0 {
		t.Fatalf("descriptor=%+v", descriptor)
	}
	for _, action := range []string{"pause", "resume", "stop", "cancel"} {
		if !strings.Contains(descriptor.InputSchema, action) {
			t.Fatalf("descriptor no declara action %q: %+v", action, descriptor)
		}
	}
}

func TestMCPRunControlExecutorV0DelegaEnPuertoInyectado(t *testing.T) {
	port := &fakeMCPRunControlPortV0{}
	executor := NewMCPRunControlToolExecutorV0(port)

	result, err := executor.Execute(context.Background(), MCPRunControlToolInputV0{
		RequestID:      "req-run-control-001",
		CorrelationID:  "corr-run-control-001",
		Action:         " stop ",
		RunRef:         " run-ref-001 ",
		RequestedBy:    " director ",
		Reason:         " cierre operativo ",
		Forced:         true,
		IdempotencyKey: " idem-001 ",
		EvidenceRefs:   []string{" evidence-1 ", "evidence-1", ""},
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != MCPRunControlEstadoOKV0 ||
		result.Action != "stop" ||
		result.Status != string(orquestaruncontrol.RunControlStatusStopRequestedV0) ||
		result.RunRef != "run-ref-001" ||
		!result.CheckpointRecorded ||
		!result.Forced {
		t.Fatalf("result=%+v", result)
	}
	if port.stop.RunRef != "run-ref-001" ||
		port.stop.RequestedBy != "director" ||
		port.stop.IdempotencyKey != "idem-001" ||
		len(port.stop.EvidenceRefs) != 1 {
		t.Fatalf("command=%+v", port.stop)
	}
	if port.checkpoint.RunRef != "run-ref-001" ||
		port.checkpoint.RequestedBy != "director" ||
		port.checkpoint.IdempotencyKey != "idem-001" ||
		!containsStringMCPTestV0(port.checkpoint.EvidenceRefs, "evidence-ref-mcp-run-control-checkpoint-recorded") {
		t.Fatalf("checkpoint=%+v", port.checkpoint)
	}
}

func TestMCPRunControlExecutorV0ValidaAction(t *testing.T) {
	result, err := NewMCPRunControlToolExecutorV0(&fakeMCPRunControlPortV0{}).Execute(
		context.Background(),
		MCPRunControlToolInputV0{RequestID: "request-ref-run-control-action-001", Action: "restart", RunRef: "run-ref-001"},
	)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != MCPRunControlEstadoErrorV0 ||
		len(result.Errores) != 1 ||
		result.Errores[0].Code != "action_no_soportada" {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPRunControlExecutorV0ResuelveRunPorJobExterno(t *testing.T) {
	port := &fakeMCPRunControlPortV0{}
	executor := MCPRunControlToolExecutorV0{
		Port: port,
		ExternalJobSource: mcpDirectorExternalJobStatsSourceForTestV0{
			Stats: MCPDirectorExternalJobStatsV0{
				AppRef:   "opes",
				JobRef:   "job-ref-opes-001",
				RunRef:   "run-ref-opes-001",
				TaskRef:  "task-ref-opes-001",
				AgentRef: "agent-ref-opes-001",
				Status:   "running",
			},
		},
	}

	result, err := executor.Execute(context.Background(), MCPRunControlToolInputV0{
		RequestID:      "req-run-control-opes-001",
		CorrelationID:  "corr-run-control-opes-001",
		Action:         "pause",
		AppRef:         "opes",
		ExternalJobRef: "job-ref-opes-001",
		RequestedBy:    "opes",
		Reason:         "pausa solicitada desde job OPES",
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != MCPRunControlEstadoOKV0 ||
		result.RunRef != "run-ref-opes-001" ||
		result.Status != string(orquestaruncontrol.RunControlStatusPausedV0) ||
		port.pause.RunRef != "run-ref-opes-001" ||
		!containsStringMCPTestV0(port.pause.EvidenceRefs, "job-ref-opes-001") {
		t.Fatalf("result=%+v pause=%+v", result, port.pause)
	}
}

type fakeMCPRunControlPortV0 struct {
	pause  orquestaruncontrol.PauseRunCommandV0
	resume orquestaruncontrol.ResumeRunCommandV0
	stop   orquestaruncontrol.StopRunCommandV0
	cancel orquestaruncontrol.CancelRunCommandV0
	checkpoint orquestaruncontrol.RecordRunCheckpointCommandV0
}

func (fake *fakeMCPRunControlPortV0) PauseRunV0(
	_ context.Context,
	command orquestaruncontrol.PauseRunCommandV0,
) (orquestaruncontrol.RunControlStateV0, error) {
	fake.pause = command
	return runControlStateForMCPTestV0(command.RunRef, orquestaruncontrol.RunControlStatusPausedV0, false), nil
}

func (fake *fakeMCPRunControlPortV0) ResumeRunV0(
	_ context.Context,
	command orquestaruncontrol.ResumeRunCommandV0,
) (orquestaruncontrol.RunControlStateV0, error) {
	fake.resume = command
	return runControlStateForMCPTestV0(command.RunRef, orquestaruncontrol.RunControlStatusRunningV0, false), nil
}

func (fake *fakeMCPRunControlPortV0) StopRunV0(
	_ context.Context,
	command orquestaruncontrol.StopRunCommandV0,
) (orquestaruncontrol.RunControlStateV0, error) {
	fake.stop = command
	return runControlStateForMCPTestV0(command.RunRef, orquestaruncontrol.RunControlStatusStopRequestedV0, command.Forced), nil
}

func (fake *fakeMCPRunControlPortV0) CancelRunV0(
	_ context.Context,
	command orquestaruncontrol.CancelRunCommandV0,
) (orquestaruncontrol.RunControlStateV0, error) {
	fake.cancel = command
	return runControlStateForMCPTestV0(command.RunRef, orquestaruncontrol.RunControlStatusCancelRequestedV0, command.Forced), nil
}

func (fake *fakeMCPRunControlPortV0) RecordRunCheckpointV0(
	_ context.Context,
	command orquestaruncontrol.RecordRunCheckpointCommandV0,
) (orquestaruncontrol.RunControlStateV0, error) {
	fake.checkpoint = command
	status := orquestaruncontrol.RunControlStatusStopRequestedV0
	forced := fake.stop.Forced
	if fake.cancel.RunRef == command.RunRef && fake.stop.RunRef == "" {
		status = orquestaruncontrol.RunControlStatusCancelRequestedV0
		forced = fake.cancel.Forced
	}
	state := runControlStateForMCPTestV0(command.RunRef, status, forced)
	state.CheckpointRecorded = true
	state.EvidenceRefs = command.EvidenceRefs
	return state, nil
}

func runControlStateForMCPTestV0(
	runRef string,
	status orquestaruncontrol.RunControlStatusV0,
	forced bool,
) orquestaruncontrol.RunControlStateV0 {
	return orquestaruncontrol.RunControlStateV0{
		RunRef:             runRef,
		Status:             status,
		CheckpointRecorded: false,
		Forced:             forced,
		EvidenceRefs:       []string{"evidence-1"},
	}
}
