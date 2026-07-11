package main

import (
	"context"
	"strings"

	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
	orquestaserver "orquesta/modulos/orquesta-server"
)

type serverGoalRunControlStopperV0 struct {
	control    orquestaruncontrol.RunControlPortV0
	runControl orquestamcp.MCPTransportRunControlExecutorV0
}

func serverGoalCooperativeStopperFromRunControlV0(
	control orquestaruncontrol.RunControlPortV0,
	runControl orquestamcp.MCPTransportRunControlExecutorV0,
) orquestaserver.GoalCooperativeStopPortV0 {
	if control == nil {
		return nil
	}
	return serverGoalRunControlStopperV0{control: control, runControl: runControl}
}

func (stopper serverGoalRunControlStopperV0) RequestGoalCooperativeStopV0(
	ctx context.Context,
	request orquestaserver.GoalCooperativeStopRequestV0,
) (orquestaserver.GoalCooperativeStopResultV0, error) {
	if stopper.control == nil {
		return orquestaserver.GoalCooperativeStopResultV0{}, nil
	}
	reason := strings.TrimSpace(request.Reason)
	if reason == "" {
		reason = "goal_cooperative_stop_requested"
	}
	if action := strings.TrimSpace(request.RecommendedAction); action != "" {
		reason = reason + ";recommended_action=" + action
	}
	requestedBy := strings.TrimSpace(request.RequestedBy)
	if requestedBy == "" {
		requestedBy = "orquesta-server"
	}
	idempotencyKey := strings.TrimSpace(request.IdempotencyKey)
	if idempotencyKey == "" {
		idempotencyKey = "idem-goal-cooperative-stop-" + serverGoalStopSafeRefPartV0(request.RunRef)
	}
	if request.RequireConfirmedBackendStop {
		return stopper.requestConfirmedBackendStopV0(ctx, request, reason, requestedBy, idempotencyKey)
	}
	state, err := stopper.control.StopRunV0(ctx, orquestaruncontrol.StopRunCommandV0{
		RunRef:         strings.TrimSpace(request.RunRef),
		RequestedBy:    requestedBy,
		Reason:         reason,
		Forced:         false,
		IdempotencyKey: idempotencyKey,
		EvidenceRefs:   request.EvidenceRefs,
	})
	if err != nil {
		return orquestaserver.GoalCooperativeStopResultV0{}, err
	}
	return orquestaserver.GoalCooperativeStopResultV0{
		Requested: true,
		Status:    string(state.Status),
		EvidenceRefs: append(
			append([]string(nil), state.EvidenceRefs...),
			"evidence-ref-goal-cooperative-stop-requested-run-control",
		),
	}, nil
}

func (stopper serverGoalRunControlStopperV0) requestConfirmedBackendStopV0(
	ctx context.Context,
	request orquestaserver.GoalCooperativeStopRequestV0,
	reason string,
	requestedBy string,
	idempotencyKey string,
) (orquestaserver.GoalCooperativeStopResultV0, error) {
	if stopper.runControl == nil {
		return orquestaserver.GoalCooperativeStopResultV0{
			Status:       "backend_stop_unavailable",
			Message:      "confirmed backend stop executor unavailable",
			EvidenceRefs: append([]string(nil), request.EvidenceRefs...),
		}, nil
	}
	result, err := stopper.runControl.Execute(ctx, orquestamcp.MCPRunControlToolInputV0{
		Action:         "stop",
		RunRef:         strings.TrimSpace(request.RunRef),
		RequestedBy:    requestedBy,
		Reason:         reason,
		Forced:         true,
		IdempotencyKey: idempotencyKey,
		EvidenceRefs:   request.EvidenceRefs,
	})
	if err != nil {
		return orquestaserver.GoalCooperativeStopResultV0{}, err
	}
	confirmed := result.Estado == orquestamcp.MCPRunControlEstadoOKV0 &&
		result.GoalControlSignalConfirmed &&
		result.Status == string(orquestaruncontrol.RunControlStatusStoppedV0) &&
		result.FinalStatus == string(orquestaruncontrol.RunControlStatusStoppedV0)
	message := "confirmed backend stop requested"
	if !confirmed {
		message = "confirmed backend stop not established"
	}
	return orquestaserver.GoalCooperativeStopResultV0{
		Requested:    confirmed,
		Status:       result.FinalStatus,
		Message:      message,
		EvidenceRefs: append([]string(nil), result.EvidenceRefs...),
	}, nil
}

func serverGoalStopSafeRefPartV0(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "unknown"
	}
	var builder strings.Builder
	for _, r := range value {
		if r >= 'a' && r <= 'z' ||
			r >= 'A' && r <= 'Z' ||
			r >= '0' && r <= '9' ||
			r == '-' || r == '_' {
			builder.WriteRune(r)
			continue
		}
		builder.WriteByte('-')
	}
	out := strings.Trim(builder.String(), "-")
	if out == "" {
		return "unknown"
	}
	return out
}
