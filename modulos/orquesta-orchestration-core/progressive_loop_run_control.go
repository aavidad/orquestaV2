package orquestacionnucleoapp

import (
	"context"
	"errors"

	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
)

type progressiveRunControlGateV0 struct {
	Allowed           bool
	Status            ProgressiveLoopStatusV0
	StopAgentsAllowed bool
	ReasonCode        string
	EvidenceRefs      []string
}

func (service ServiceV0) evaluateProgressiveRunControlGateV0(
	ctx context.Context,
	runRef string,
) (progressiveRunControlGateV0, error) {
	if service.RunControl == nil {
		return progressiveRunControlGateV0{Allowed: true}, nil
	}
	state, err := service.RunControl.ReadRunControlStateV0(
		ctx,
		orquestaruncontrol.RunControlReadRequestV0{RunRef: runRef},
	)
	if err != nil {
		var notFound orquestaruncontrol.RunControlStateNotFoundErrorV0
		if !errors.As(err, &notFound) {
			return progressiveRunControlGateV0{}, err
		}
		state = orquestaruncontrol.DefaultRunControlStateV0(runRef)
	}
	evaluation := orquestaruncontrol.EvaluateRunControlV0(state)
	if evaluation.SchedulingAllowed && evaluation.DispatchAllowed {
		return progressiveRunControlGateV0{Allowed: true}, nil
	}
	return progressiveRunControlGateV0{
		Allowed:           false,
		Status:            progressiveRunControlStatusV0(state),
		StopAgentsAllowed: evaluation.StopAgentsAllowed,
		ReasonCode:        progressiveRunControlReasonCodeV0(state),
		EvidenceRefs:      compactStringsV0(state.EvidenceRefs),
	}, nil
}

func progressiveRunControlStatusV0(
	state orquestaruncontrol.RunControlStateV0,
) ProgressiveLoopStatusV0 {
	switch orquestaruncontrol.NormalizeRunControlStatusV0(state.Status) {
	case orquestaruncontrol.RunControlStatusPausedV0:
		return ProgressiveLoopStatusRunPausedV0
	case orquestaruncontrol.RunControlStatusStopRequestedV0:
		return ProgressiveLoopStatusRunStopRequestedV0
	case orquestaruncontrol.RunControlStatusCancelRequestedV0:
		return ProgressiveLoopStatusRunCanceledV0
	case orquestaruncontrol.RunControlStatusStoppedV0,
		orquestaruncontrol.RunControlStatusCanceledV0:
		return ProgressiveLoopStatusRunTerminalV0
	default:
		return ProgressiveLoopStatusBlockedV0
	}
}

func (service ServiceV0) finishProgressiveRunControlGateV0(
	ctx context.Context,
	request ProgressiveLoopRequestV0,
	result ProgressiveLoopResultV0,
	gate progressiveRunControlGateV0,
) (ProgressiveLoopResultV0, error) {
	result.Status = gate.Status
	if gate.StopAgentsAllowed {
		dispatched, err := service.stopRunControlAgentsV0(ctx, request, gate)
		result.Dispatches = append(result.Dispatches, dispatched.Once...)
		result.BatchDispatches = append(result.BatchDispatches, dispatched.Batch...)
		if err != nil {
			result.Status = ProgressiveLoopStatusStopErrorV0
			return service.finishProgressiveLoopV0(ctx, request, result), err
		}
	}
	return service.finishProgressiveLoopV0(ctx, request, result), nil
}

func progressiveRunControlReasonCodeV0(
	state orquestaruncontrol.RunControlStateV0,
) string {
	switch orquestaruncontrol.NormalizeRunControlStatusV0(state.Status) {
	case orquestaruncontrol.RunControlStatusCancelRequestedV0:
		return "run_cancel_requested"
	case orquestaruncontrol.RunControlStatusStopRequestedV0:
		return "run_stop_requested"
	default:
		return "run_control_blocked"
	}
}
