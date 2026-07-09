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

func operatorTaskTerminalNotifierFromProjectConfigFileV0(
	config serverProjectConfigFileV0,
) operatorTaskTerminalNotifierV0 {
	targetRef := telegramOperatorNotificationTargetFromProjectConfigFileV0(config)
	sender := telegramBotAPISenderFromProjectConfigFileV0(config)
	if strings.TrimSpace(targetRef) == "" || sender == nil {
		return operatorTaskTerminalNotifierV0{}
	}
	return newOperatorTaskTerminalNotifierV0(targetRef, hermesTelegramNotifierV0{Sender: sender})
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

func (notifier operatorTaskTerminalNotifierV0) NotifyProviderIssueV0(
	ctx context.Context,
	reasonCode string,
	runRefs []string,
	evidenceRefs []string,
) (notifications.MessageV0, bool, error) {
	reasonCode = strings.TrimSpace(reasonCode)
	if reasonCode == "" {
		return notifications.MessageV0{}, false, nil
	}
	runRef := ""
	for _, value := range runRefs {
		if strings.TrimSpace(value) != "" {
			runRef = strings.TrimSpace(value)
			break
		}
	}
	return notifications.NotifyTerminalEventV0(ctx, notifier.Service, notifications.TerminalEventV0{
		EventRef:     "provider:" + reasonCode + ":" + runRef,
		Kind:         notifications.EventKindProviderV0,
		SubjectRef:   reasonCode,
		RunRef:       runRef,
		Status:       notifications.StatusBlockedV0,
		Summary:      providerIssueOperatorSummaryV0(reasonCode),
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

func providerIssueOperatorSummaryV0(reasonCode string) string {
	switch strings.TrimSpace(reasonCode) {
	case "codex_app_server_provider_unauthorized", "codex_app_server_auth_missing":
		return "proveedor no autenticado; accion: ejecutar hermes auth y hermes model, o restaurar auth del CODEX_HOME aislado"
	case "codex_app_server_goal_provider_limited", "provider_usage_limit_retry_after":
		return "proveedor sin cuota/capacidad; accion: esperar cuota y reanudar de forma idempotente"
	case idleSelfImprovementProviderAuthBlockedReasonV0:
		return "proveedor pausado; accion: restaurar auth/cuota, confirmar disponibilidad y reanudar"
	default:
		return "proveedor no disponible; accion: revisar auth/cuota y reanudar cuando el probe pase"
	}
}
