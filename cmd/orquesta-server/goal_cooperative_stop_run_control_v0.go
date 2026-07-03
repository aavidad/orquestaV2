package main

import (
	"context"
	"strings"

	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
	orquestaserver "orquesta/modulos/orquesta-server"
)

type serverGoalRunControlStopperV0 struct {
	control orquestaruncontrol.RunControlPortV0
}

func serverGoalCooperativeStopperFromRunControlV0(
	control orquestaruncontrol.RunControlPortV0,
) orquestaserver.GoalCooperativeStopPortV0 {
	if control == nil {
		return nil
	}
	return serverGoalRunControlStopperV0{control: control}
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
