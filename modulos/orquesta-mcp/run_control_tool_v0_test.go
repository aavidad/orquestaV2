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
		!result.Forced {
		t.Fatalf("result=%+v", result)
	}
	if port.stop.RunRef != "run-ref-001" ||
		port.stop.RequestedBy != "director" ||
		port.stop.IdempotencyKey != "idem-001" ||
		len(port.stop.EvidenceRefs) != 1 {
		t.Fatalf("command=%+v", port.stop)
	}
}

func TestMCPRunControlExecutorV0ValidaAction(t *testing.T) {
	result, err := NewMCPRunControlToolExecutorV0(&fakeMCPRunControlPortV0{}).Execute(
		context.Background(),
		MCPRunControlToolInputV0{Action: "restart", RunRef: "run-ref-001"},
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

type fakeMCPRunControlPortV0 struct {
	pause  orquestaruncontrol.PauseRunCommandV0
	resume orquestaruncontrol.ResumeRunCommandV0
	stop   orquestaruncontrol.StopRunCommandV0
	cancel orquestaruncontrol.CancelRunCommandV0
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
