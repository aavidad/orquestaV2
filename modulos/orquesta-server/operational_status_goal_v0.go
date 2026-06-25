package orquestaserver

import (
	"strings"

	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestaobservability "orquesta/modulos/orquesta-observability"
)

func residentOperationalGoalProjectionV0(state StateV0) *ServerPublicIdleSelfImprovementGoalStateV0 {
	return NewServerPublicIdleSelfImprovementGoalStateV0(state)
}

func residentOperationalGoalProjectionHasDurableStateV0(goal *ServerPublicIdleSelfImprovementGoalStateV0) bool {
	return goal != nil && (goal.SpecPresent || goal.ReceiptPresent || goal.ResultPresent || goal.ClosurePresent)
}

func residentOperationalGoalReferencesV0(state StateV0) []orquestaobservability.DiagnosticoReferenciaV0 {
	goal := residentOperationalGoalProjectionV0(state)
	if goal == nil {
		return nil
	}
	refs := []orquestaobservability.DiagnosticoReferenciaV0{}
	appendRef := func(rel, targetType, targetRef string) {
		targetRef = strings.TrimSpace(targetRef)
		if targetRef == "" || !serverEvidenceRefPatternV0.MatchString(targetRef) {
			return
		}
		refs = append(refs, orquestaobservability.DiagnosticoReferenciaV0{
			Rel:        rel,
			TargetType: targetType,
			TargetRef:  targetRef,
		})
	}
	appendRef("runtime", "runtime", goal.GoalRef)
	appendRef("runtime", "runtime", goal.ExternalGoalRef)
	appendRef("related", "task", goal.RequestRef)
	appendRef("related", "flow", goal.RunRef)
	appendRef("related", "project", goal.ProjectRef)
	return refs
}

func residentOperationalGoalHealthCheckV0(state StateV0) orquestaobservability.DiagnosticoSaludCheckV0 {
	goal := residentOperationalGoalProjectionV0(state)
	if goal == nil {
		return orquestaobservability.DiagnosticoSaludCheckV0{}
	}
	severity, estado := residentOperationalGoalSeverityAndEstadoV0(goal)
	return orquestaobservability.DiagnosticoSaludCheckV0{
		Area:         "runtime",
		Severity:     severity,
		Estado:       estado,
		I18nKey:      "server.health.idle_self_improvement_goal",
		EvidenceRefs: sanitizeServerEvidenceRefsV0(goal.GoalRefs),
	}
}

func residentOperationalGoalFirstCapabilityHealthCheckV0(state StateV0) orquestaobservability.DiagnosticoSaludCheckV0 {
	if !residentOperationalGoalFirstCapabilityMissingV0(state) {
		return orquestaobservability.DiagnosticoSaludCheckV0{}
	}
	return orquestaobservability.DiagnosticoSaludCheckV0{
		Area:     "runtime",
		Severity: "warning",
		Estado:   orquestaobservability.DiagnosticoEstadoDegradedV0,
		I18nKey:  "server.health.idle_self_improvement_goal_first",
	}
}

func residentOperationalGoalBlockerV0(state StateV0) orquestaobservability.DiagnosticoBloqueoV0 {
	goal := residentOperationalGoalProjectionV0(state)
	if goal == nil || !residentOperationalGoalBlockedV0(goal) {
		return orquestaobservability.DiagnosticoBloqueoV0{}
	}
	return orquestaobservability.DiagnosticoBloqueoV0{
		BlockerRef:   "blocker-ref-server-idle-goal",
		Severity:     residentOperationalGoalBlockerSeverityV0(goal),
		OwnerArea:    "runtime",
		Summary:      "goal residente requiere intervencion o rework",
		EvidenceRefs: sanitizeServerEvidenceRefsV0(goal.GoalRefs),
	}
}

func residentOperationalGoalFirstCapabilityBlockerV0(state StateV0) orquestaobservability.DiagnosticoBloqueoV0 {
	if !residentOperationalGoalFirstCapabilityMissingV0(state) {
		return orquestaobservability.DiagnosticoBloqueoV0{}
	}
	return orquestaobservability.DiagnosticoBloqueoV0{
		BlockerRef: "blocker-ref-server-idle-goal-first-capability",
		Severity:   "warning",
		OwnerArea:  "runtime",
		Summary:    "capacidad goal-first no disponible",
	}
}

func residentOperationalGoalActivityV0(state StateV0) orquestaobservability.DiagnosticoActividadV0 {
	if strings.TrimSpace(state.IdleSelfImprovementCheck) == "" {
		return orquestaobservability.DiagnosticoActividadV0{}
	}
	goal := residentOperationalGoalProjectionV0(state)
	if goal == nil {
		return orquestaobservability.DiagnosticoActividadV0{}
	}
	return orquestaobservability.DiagnosticoActividadV0{
		ActivityRef: "activity-ref-server-idle-goal",
		OccurredAt:  strings.TrimSpace(state.IdleSelfImprovementCheck),
		Area:        "runtime",
		Summary:     "goal residente observado",
		EventRef:    firstNonEmptyServerDiagnosticV0(goal.GoalRef, goal.ExternalGoalRef),
	}
}

func residentOperationalGoalSeverityAndEstadoV0(
	goal *ServerPublicIdleSelfImprovementGoalStateV0,
) (string, string) {
	reason := strings.TrimSpace(goal.OperationalReasonCode)
	status := strings.TrimSpace(firstNonEmptyServerDiagnosticV0(goal.OperationalStatus, goal.ResultStatus, goal.ReceiptStatus))
	switch {
	case reason == idleSelfImprovementGoalClosureAcceptedReasonV0 ||
		goal.ClosureAccepted ||
		status == orquestagoal.GoalStatusAcceptedV0:
		return "info", orquestaobservability.DiagnosticoEstadoOKV0
	case reason == idleSelfImprovementGoalObserverUnavailableReasonV0 ||
		reason == idleSelfImprovementGoalObservationErrorReasonV0 ||
		reason == idleSelfImprovementGoalInvalidReasonV0 ||
		status == orquestagoal.GoalStatusInvalidV0:
		return "error", orquestaobservability.DiagnosticoEstadoBlockedV0
	case reason == idleSelfImprovementGoalBlockedReasonV0 ||
		status == orquestagoal.GoalStatusBlockedV0 ||
		goal.ClosureNeedsRework:
		return "warning", orquestaobservability.DiagnosticoEstadoBlockedV0
	case reason == idleSelfImprovementGoalCompletePendingClosureV0:
		return "warning", orquestaobservability.DiagnosticoEstadoDegradedV0
	case reason == idleSelfImprovementGoalRunningReasonV0 ||
		status == orquestagoal.GoalStatusRunningV0 ||
		status == idleSelfImprovementPreparePendingStatusV0 ||
		reason == "prepared":
		return "info", orquestaobservability.DiagnosticoEstadoOKV0
	default:
		return "info", orquestaobservability.DiagnosticoEstadoUnknownV0
	}
}

func residentOperationalGoalBlockedV0(goal *ServerPublicIdleSelfImprovementGoalStateV0) bool {
	if goal == nil {
		return false
	}
	reason := strings.TrimSpace(goal.OperationalReasonCode)
	status := strings.TrimSpace(firstNonEmptyServerDiagnosticV0(goal.OperationalStatus, goal.ResultStatus, goal.ReceiptStatus))
	return reason == idleSelfImprovementGoalObserverUnavailableReasonV0 ||
		reason == idleSelfImprovementGoalObservationErrorReasonV0 ||
		reason == idleSelfImprovementGoalInvalidReasonV0 ||
		reason == idleSelfImprovementGoalBlockedReasonV0 ||
		status == orquestagoal.GoalStatusInvalidV0 ||
		status == orquestagoal.GoalStatusBlockedV0 ||
		goal.ClosureNeedsRework
}

func residentOperationalGoalBlockerSeverityV0(goal *ServerPublicIdleSelfImprovementGoalStateV0) string {
	reason := strings.TrimSpace(goal.OperationalReasonCode)
	status := strings.TrimSpace(firstNonEmptyServerDiagnosticV0(goal.OperationalStatus, goal.ResultStatus, goal.ReceiptStatus))
	if reason == idleSelfImprovementGoalObserverUnavailableReasonV0 ||
		reason == idleSelfImprovementGoalObservationErrorReasonV0 ||
		reason == idleSelfImprovementGoalInvalidReasonV0 ||
		status == orquestagoal.GoalStatusInvalidV0 {
		return "error"
	}
	return "warning"
}

func residentOperationalGoalFirstCapabilityMissingV0(state StateV0) bool {
	message := state.IdleSelfImprovementOperationalMessage
	if message != nil && strings.TrimSpace(message.ReasonCode) == idleSelfImprovementGoalLauncherUnavailableReasonV0 {
		return true
	}
	return strings.TrimSpace(state.IdleSelfImprovementReason) == idleSelfImprovementGoalLauncherUnavailableReasonV0
}
