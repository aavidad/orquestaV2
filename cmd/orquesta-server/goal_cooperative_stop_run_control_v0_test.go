package main

import (
	"context"
	"testing"

	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
	orquestaserver "orquesta/modulos/orquesta-server"
)

func TestServerGoalRunControlStopperV0HighConsumptionUsesConfirmedBackendStopV0(t *testing.T) {
	control := &serverGoalRunControlForTestV0{}
	executor := &serverGoalRunControlExecutorForTestV0{result: orquestamcp.MCPRunControlToolResultV0{
		Estado:                     orquestamcp.MCPRunControlEstadoOKV0,
		Status:                     string(orquestaruncontrol.RunControlStatusStoppedV0),
		FinalStatus:                string(orquestaruncontrol.RunControlStatusStoppedV0),
		GoalControlSignalConfirmed: true,
		EvidenceRefs:               []string{"evidence-ref-backend-stop-confirmed"},
	}}
	stopper := serverGoalCooperativeStopperFromRunControlV0(control, executor)
	request := orquestaserver.GoalCooperativeStopRequestV0{
		RunRef:                      "run-high-consumption-001",
		RequireConfirmedBackendStop: true,
		IdempotencyKey:              "idem-high-consumption-001",
	}

	first, err := stopper.RequestGoalCooperativeStopV0(context.Background(), request)
	if err != nil || !first.Requested || control.stopCalls != 0 || len(executor.inputs) != 1 {
		t.Fatalf("first result=%+v err=%v control=%+v executor=%+v", first, err, control, executor.inputs)
	}
	second, err := stopper.RequestGoalCooperativeStopV0(context.Background(), request)
	if err != nil || !second.Requested || len(executor.inputs) != 2 {
		t.Fatalf("replay result=%+v err=%v executor=%+v", second, err, executor.inputs)
	}
	for _, input := range executor.inputs {
		if input.Action != "stop" || !input.Forced || input.IdempotencyKey != request.IdempotencyKey {
			t.Fatalf("composite input=%+v", input)
		}
	}
}

func TestServerGoalRunControlStopperV0HighConsumptionNoFalseRequestedV0(t *testing.T) {
	control := &serverGoalRunControlForTestV0{}
	executor := &serverGoalRunControlExecutorForTestV0{result: orquestamcp.MCPRunControlToolResultV0{
		Estado:       orquestamcp.MCPRunControlEstadoErrorV0,
		Status:       string(orquestaruncontrol.RunControlStatusStopRequestedV0),
		FinalStatus:  string(orquestaruncontrol.RunControlStatusStopRequestedV0),
		EvidenceRefs: []string{"evidence-ref-backend-stop-residual"},
	}}
	stopper := serverGoalCooperativeStopperFromRunControlV0(control, executor)

	result, err := stopper.RequestGoalCooperativeStopV0(context.Background(), orquestaserver.GoalCooperativeStopRequestV0{
		RunRef:                      "run-high-consumption-residual-001",
		RequireConfirmedBackendStop: true,
	})
	if err != nil || result.Requested || result.Status != string(orquestaruncontrol.RunControlStatusStopRequestedV0) ||
		result.Message != "confirmed backend stop not established" || control.stopCalls != 0 {
		t.Fatalf("result=%+v err=%v control=%+v", result, err, control)
	}
}

func TestServerGoalRunControlStopperV0HighConsumptionLegacyCooperativePathV0(t *testing.T) {
	control := &serverGoalRunControlForTestV0{}
	executor := &serverGoalRunControlExecutorForTestV0{}
	stopper := serverGoalCooperativeStopperFromRunControlV0(control, executor)

	result, err := stopper.RequestGoalCooperativeStopV0(context.Background(), orquestaserver.GoalCooperativeStopRequestV0{
		RunRef: "run-cooperative-001",
	})
	if err != nil || !result.Requested || control.stopCalls != 1 || control.stop.Forced || len(executor.inputs) != 0 {
		t.Fatalf("result=%+v err=%v control=%+v executor=%+v", result, err, control, executor.inputs)
	}
}

type serverGoalRunControlExecutorForTestV0 struct {
	inputs []orquestamcp.MCPRunControlToolInputV0
	result orquestamcp.MCPRunControlToolResultV0
	err    error
}

func (fake *serverGoalRunControlExecutorForTestV0) Execute(
	_ context.Context,
	input orquestamcp.MCPRunControlToolInputV0,
) (orquestamcp.MCPRunControlToolResultV0, error) {
	fake.inputs = append(fake.inputs, input)
	return fake.result, fake.err
}

type serverGoalRunControlForTestV0 struct {
	stopCalls int
	stop      orquestaruncontrol.StopRunCommandV0
}

func (fake *serverGoalRunControlForTestV0) ReadRunControlStateV0(context.Context, orquestaruncontrol.RunControlReadRequestV0) (orquestaruncontrol.RunControlStateV0, error) {
	return orquestaruncontrol.RunControlStateV0{}, nil
}

func (fake *serverGoalRunControlForTestV0) PauseRunV0(context.Context, orquestaruncontrol.PauseRunCommandV0) (orquestaruncontrol.RunControlStateV0, error) {
	return orquestaruncontrol.RunControlStateV0{}, nil
}

func (fake *serverGoalRunControlForTestV0) ResumeRunV0(context.Context, orquestaruncontrol.ResumeRunCommandV0) (orquestaruncontrol.RunControlStateV0, error) {
	return orquestaruncontrol.RunControlStateV0{}, nil
}

func (fake *serverGoalRunControlForTestV0) StopRunV0(_ context.Context, command orquestaruncontrol.StopRunCommandV0) (orquestaruncontrol.RunControlStateV0, error) {
	fake.stopCalls++
	fake.stop = command
	return orquestaruncontrol.RunControlStateV0{RunRef: command.RunRef, Status: orquestaruncontrol.RunControlStatusStopRequestedV0}, nil
}

func (fake *serverGoalRunControlForTestV0) CancelRunV0(context.Context, orquestaruncontrol.CancelRunCommandV0) (orquestaruncontrol.RunControlStateV0, error) {
	return orquestaruncontrol.RunControlStateV0{}, nil
}

func (fake *serverGoalRunControlForTestV0) RecordRunCheckpointV0(context.Context, orquestaruncontrol.RecordRunCheckpointCommandV0) (orquestaruncontrol.RunControlStateV0, error) {
	return orquestaruncontrol.RunControlStateV0{}, nil
}

func (fake *serverGoalRunControlForTestV0) CompleteRunControlV0(context.Context, orquestaruncontrol.CompleteRunControlCommandV0) (orquestaruncontrol.RunControlStateV0, error) {
	return orquestaruncontrol.RunControlStateV0{}, nil
}
