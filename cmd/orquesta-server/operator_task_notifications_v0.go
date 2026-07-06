package main

import (
	"context"
	"strings"

	orquestagoal "orquesta/modulos/orquesta-goal"
	notifications "orquesta/modulos/orquesta-operator-notifications"
	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
)

type operatorTaskTerminalNotifierV0 struct {
	Service notifications.ServiceV0
}

func newOperatorTaskTerminalNotifierV0(targetRef string, sender notifications.SendPortV0) operatorTaskTerminalNotifierV0 {
	return operatorTaskTerminalNotifierV0{
		Service: notifications.ServiceV0{
			TargetRef: strings.TrimSpace(targetRef),
			Sender:    sender,
			Dedupe:    notifications.NewMemoryDedupeStoreV0(),
		},
	}
}

func (notifier operatorTaskTerminalNotifierV0) NotifyGoalStateV0(
	ctx context.Context,
	state orquestagoal.GoalWorkStateV0,
) (notifications.MessageV0, bool, error) {
	summary := ""
	evidenceRefs := state.EvidenceRefs
	if state.LastResult != nil {
		summary = state.LastResult.Summary
		evidenceRefs = append(evidenceRefs, state.LastResult.EvidenceRefs...)
	}
	return notifications.NotifyTerminalEventV0(ctx, notifier.Service, notifications.TerminalEventV0{
		Kind:         notifications.EventKindGoalV0,
		SubjectRef:   firstNonEmptyOperatorNotificationV0(state.GoalRef, state.RunRef),
		RunRef:       state.RunRef,
		GoalRef:      state.GoalRef,
		Status:       state.Status,
		Summary:      summary,
		EvidenceRefs: evidenceRefs,
	})
}

func (notifier operatorTaskTerminalNotifierV0) NotifyRunControlStateV0(
	ctx context.Context,
	state orquestaruncontrol.RunControlStateV0,
) (notifications.MessageV0, bool, error) {
	return notifications.NotifyTerminalEventV0(ctx, notifier.Service, notifications.TerminalEventV0{
		Kind:         notifications.EventKindRunV0,
		SubjectRef:   state.RunRef,
		RunRef:       state.RunRef,
		Status:       string(state.Status),
		Summary:      state.Meta.Reason,
		EvidenceRefs: state.EvidenceRefs,
	})
}

func (notifier operatorTaskTerminalNotifierV0) NotifyTaskTerminalV0(
	ctx context.Context,
	taskRef string,
	status string,
	summary string,
	evidenceRefs []string,
) (notifications.MessageV0, bool, error) {
	return notifications.NotifyTerminalEventV0(ctx, notifier.Service, notifications.TerminalEventV0{
		Kind:         notifications.EventKindTaskV0,
		SubjectRef:   taskRef,
		Status:       status,
		Summary:      summary,
		EvidenceRefs: evidenceRefs,
	})
}

func firstNonEmptyOperatorNotificationV0(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
