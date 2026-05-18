package orquestacionnucleoapp

import "strings"

const (
	autonomousQualityKindAgentV0       = "agent"
	autonomousQualityKindDeliveryV0    = "delivery"
	autonomousQualityKindQualityGateV0 = "quality_gate"

	autonomousQualityReviewStatusSeparatorV0   = "#review_result:"
	autonomousQualityReviewRequestSeparatorV0  = "#review_request:"
	autonomousQualityReviewDeliverySeparatorV0 = "#delivery:"
	autonomousQualityGateDecisionSeparatorV0   = "#decision:"
	autonomousQualityGateSubjectSeparatorV0    = "#subject:"
)

func buildAutonomousQualityRecommendationsV0(
	stats DirectorRunStatsV0,
) []AutonomousQualityRecommendationV0 {
	var recommendations []AutonomousQualityRecommendationV0
	recommendations = append(recommendations, qualityGateRecommendationsV0(stats.Refs.QualityGates)...)
	recommendations = append(recommendations, reviewResultRecommendationsV0(stats.Refs.ReviewResults)...)
	recommendations = append(recommendations, agentQualityRecommendationsV0(stats)...)
	return compactAutonomousQualityRecommendationsV0(recommendations)
}

func qualityGateRecommendationsV0(refs []string) []AutonomousQualityRecommendationV0 {
	out := make([]AutonomousQualityRecommendationV0, 0, len(refs))
	for _, ref := range compactStringsV0(refs) {
		recommendation, ok := qualityGateRecommendationV0(ref)
		if ok {
			out = append(out, recommendation)
		}
	}
	return out
}

func qualityGateRecommendationV0(ref string) (AutonomousQualityRecommendationV0, bool) {
	gateRef, tail, ok := strings.Cut(ref, autonomousQualityGateDecisionSeparatorV0)
	if !ok {
		return AutonomousQualityRecommendationV0{}, false
	}
	decision, subjectRef, ok := strings.Cut(tail, autonomousQualityGateSubjectSeparatorV0)
	if !ok {
		return AutonomousQualityRecommendationV0{}, false
	}
	action, reason, ok := actionForQualityGateDecisionV0(decision)
	if !ok {
		return AutonomousQualityRecommendationV0{}, false
	}
	return AutonomousQualityRecommendationV0{
		SubjectRef:        strings.TrimSpace(subjectRef),
		SubjectKind:       autonomousQualityKindQualityGateV0,
		SignalRef:         strings.TrimSpace(gateRef),
		ReasonCode:        reason,
		RecommendedAction: action,
		Summary:           "Quality gate requiere decision antes de aceptar entrega.",
		EvidenceRefs:      compactStringsV0([]string{strings.TrimSpace(ref)}),
	}, true
}

func actionForQualityGateDecisionV0(decision string) (string, string, bool) {
	switch strings.TrimSpace(decision) {
	case "rework_required":
		return AutonomousQualityActionReplanV0, "quality_gate_rework_required", true
	case "blocked":
		return AutonomousQualityActionAskDirectorV0, "quality_gate_blocked", true
	case "ask_director":
		return AutonomousQualityActionAskDirectorV0, "quality_gate_ask_director", true
	default:
		return "", "", false
	}
}

func reviewResultRecommendationsV0(refs []string) []AutonomousQualityRecommendationV0 {
	out := make([]AutonomousQualityRecommendationV0, 0, len(refs))
	for _, ref := range compactStringsV0(refs) {
		recommendation, ok := reviewResultRecommendationV0(ref)
		if ok {
			out = append(out, recommendation)
		}
	}
	return out
}

func reviewResultRecommendationV0(ref string) (AutonomousQualityRecommendationV0, bool) {
	resultRef, tail, ok := strings.Cut(ref, autonomousQualityReviewStatusSeparatorV0)
	if !ok {
		return AutonomousQualityRecommendationV0{}, false
	}
	status, tail, ok := strings.Cut(tail, autonomousQualityReviewRequestSeparatorV0)
	if !ok || !reviewStatusRequiresReplanV0(status) {
		return AutonomousQualityRecommendationV0{}, false
	}
	_, deliveryRef, ok := strings.Cut(tail, autonomousQualityReviewDeliverySeparatorV0)
	if !ok {
		return AutonomousQualityRecommendationV0{}, false
	}
	return AutonomousQualityRecommendationV0{
		SubjectRef:        strings.TrimSpace(deliveryRef),
		SubjectKind:       autonomousQualityKindDeliveryV0,
		SignalRef:         strings.TrimSpace(resultRef),
		ReasonCode:        "review_" + strings.TrimSpace(status),
		RecommendedAction: AutonomousQualityActionReplanV0,
		Summary:           "Revision no aceptada; replan compacto requerido.",
		EvidenceRefs:      compactStringsV0([]string{strings.TrimSpace(ref)}),
	}, true
}

func reviewStatusRequiresReplanV0(status string) bool {
	status = strings.TrimSpace(status)
	return status == "changes_requested" || status == "rejected"
}

func agentQualityRecommendationsV0(
	stats DirectorRunStatsV0,
) []AutonomousQualityRecommendationV0 {
	out := make([]AutonomousQualityRecommendationV0, 0, len(stats.Agents)+len(stats.Progress.NoSignalAgentRefs))
	for _, agent := range stats.Agents {
		if recommendation, ok := failedAgentQualityRecommendationV0(agent); ok {
			out = append(out, recommendation)
			continue
		}
		if recommendation, ok := lostAgentQualityRecommendationV0(agent); ok {
			out = append(out, recommendation)
			continue
		}
		if recommendation, ok := progressAgentQualityRecommendationV0(agent); ok {
			out = append(out, recommendation)
		}
	}
	for _, agentRef := range compactStringsV0(stats.Progress.NoSignalAgentRefs) {
		out = append(out, AutonomousQualityRecommendationV0{
			SubjectRef:        agentRef,
			SubjectKind:       autonomousQualityKindAgentV0,
			SignalRef:         agentRef,
			ReasonCode:        "agent_no_signal",
			RecommendedAction: AutonomousQualityActionAskDirectorV0,
			Summary:           "Agente en vuelo sin senal compacta; consultar direccion.",
			EvidenceRefs:      []string{agentRef},
		})
	}
	return out
}

func failedAgentQualityRecommendationV0(
	agent DirectorAgentStatsV0,
) (AutonomousQualityRecommendationV0, bool) {
	if !agent.Failed {
		return AutonomousQualityRecommendationV0{}, false
	}
	return AutonomousQualityRecommendationV0{
		SubjectRef:        agent.AgentRequestID,
		SubjectKind:       autonomousQualityKindAgentV0,
		SignalRef:         agent.AgentRequestID,
		ReasonCode:        "agent_failed",
		RecommendedAction: AutonomousQualityActionReplanV0,
		Summary:           "Agente fallido; replan compacto requerido.",
		EvidenceRefs:      []string{agent.AgentRequestID},
	}, true
}

func lostAgentQualityRecommendationV0(
	agent DirectorAgentStatsV0,
) (AutonomousQualityRecommendationV0, bool) {
	if !agent.Lost {
		return AutonomousQualityRecommendationV0{}, false
	}
	return AutonomousQualityRecommendationV0{
		SubjectRef:        agent.AgentRequestID,
		SubjectKind:       autonomousQualityKindAgentV0,
		SignalRef:         agent.AgentRequestID,
		ReasonCode:        "agent_lost",
		RecommendedAction: AutonomousQualityActionReplanV0,
		Summary:           "Agente perdido; replan compacto requerido.",
		EvidenceRefs:      []string{agent.AgentRequestID},
	}, true
}

func progressAgentQualityRecommendationV0(
	agent DirectorAgentStatsV0,
) (AutonomousQualityRecommendationV0, bool) {
	if agent.LastProgress == nil {
		return AutonomousQualityRecommendationV0{}, false
	}
	progress := *agent.LastProgress
	action, reason, ok := actionForAgentProgressV0(agent, progress)
	if !ok {
		return AutonomousQualityRecommendationV0{}, false
	}
	return AutonomousQualityRecommendationV0{
		SubjectRef:        agent.AgentRequestID,
		SubjectKind:       autonomousQualityKindAgentV0,
		SignalRef:         progress.ReportRef,
		ReasonCode:        reason,
		RecommendedAction: action,
		Summary:           progress.Summary,
		EvidenceRefs:      compactStringsV0(append([]string{progress.ReportRef}, progress.EvidenceRefs...)),
	}, true
}

func actionForAgentProgressV0(
	agent DirectorAgentStatsV0,
	progress DirectorAgentProgressV0,
) (string, string, bool) {
	if progress.Classification == DirectorProgressClassificationOverBudgetNoActivityV0 {
		return stoppableAgentActionV0(agent), "over_budget_no_activity", true
	}
	switch progress.Status {
	case DirectorTaskProgressLoopDetectedV0, DirectorTaskProgressStoppedV0:
		return stoppableAgentActionV0(agent), "agent_" + progress.Status, true
	case DirectorTaskProgressStalledV0:
		return AutonomousQualityActionAskDirectorV0, "agent_stalled", true
	default:
		return "", "", false
	}
}

func stoppableAgentActionV0(agent DirectorAgentStatsV0) string {
	if agent.CanStop {
		return AutonomousQualityActionStopAgentV0
	}
	return AutonomousQualityActionAskDirectorV0
}

func compactAutonomousQualityRecommendationsV0(
	recommendations []AutonomousQualityRecommendationV0,
) []AutonomousQualityRecommendationV0 {
	seen := map[string]bool{}
	out := make([]AutonomousQualityRecommendationV0, 0, len(recommendations))
	for _, recommendation := range recommendations {
		recommendation = normalizeAutonomousQualityRecommendationV0(recommendation)
		if recommendation.SubjectRef == "" || recommendation.RecommendedAction == "" {
			continue
		}
		key := recommendation.SubjectKind + "|" + recommendation.SubjectRef + "|" + recommendation.RecommendedAction
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, recommendation)
	}
	return out
}

func normalizeAutonomousQualityRecommendationV0(
	recommendation AutonomousQualityRecommendationV0,
) AutonomousQualityRecommendationV0 {
	recommendation.SubjectRef = strings.TrimSpace(recommendation.SubjectRef)
	recommendation.SubjectKind = strings.TrimSpace(recommendation.SubjectKind)
	recommendation.SignalRef = strings.TrimSpace(recommendation.SignalRef)
	recommendation.ReasonCode = strings.TrimSpace(recommendation.ReasonCode)
	recommendation.RecommendedAction = strings.TrimSpace(recommendation.RecommendedAction)
	recommendation.Summary = strings.TrimSpace(recommendation.Summary)
	recommendation.EvidenceRefs = compactStringsV0(recommendation.EvidenceRefs)
	return recommendation
}
