package orquestadirectoragent

import "strings"

func normalizeDirectorAgentRoutingDecisionV0(decision *DirectorAgentDecisionV0) {
	if decision.AskDirector != nil {
		normalized := normalizeDirectorAgentAskQuestionV0(*decision.AskDirector)
		decision.AskDirector = &normalized
	}
	if decision.AskUser != nil {
		normalized := normalizeDirectorAgentAskQuestionV0(*decision.AskUser)
		decision.AskUser = &normalized
	}
	if decision.RequestCapacity != nil {
		normalized := normalizeDirectorAgentRequestCapacityV0(*decision.RequestCapacity)
		decision.RequestCapacity = &normalized
	}
	if decision.RequestAgent != nil {
		normalized := normalizeDirectorAgentRequestAgentV0(*decision.RequestAgent)
		decision.RequestAgent = &normalized
	}
	if decision.RequestRework != nil {
		normalized := normalizeDirectorAgentRequestReworkV0(*decision.RequestRework)
		decision.RequestRework = &normalized
	}
	if decision.RecordReplanDecision != nil {
		normalized := normalizeDirectorAgentReplanDecisionV0(*decision.RecordReplanDecision)
		decision.RecordReplanDecision = &normalized
	}
}

func normalizeDirectorAgentAskQuestionV0(
	payload DirectorAgentAskQuestionCommandV0,
) DirectorAgentAskQuestionCommandV0 {
	return DirectorAgentAskQuestionCommandV0{
		QuestionID:   strings.TrimSpace(payload.QuestionID),
		SourceGroup:  strings.TrimSpace(payload.SourceGroup),
		TargetGroup:  strings.TrimSpace(payload.TargetGroup),
		Summary:      strings.TrimSpace(payload.Summary),
		Options:      compactDirectorAgentStringsV0(payload.Options),
		EvidenceRefs: compactDirectorAgentStringsV0(payload.EvidenceRefs),
		Blocking:     payload.Blocking,
	}
}

func normalizeDirectorAgentRequestCapacityV0(
	payload DirectorAgentRequestCapacityCommandV0,
) DirectorAgentRequestCapacityCommandV0 {
	return DirectorAgentRequestCapacityCommandV0{
		CapacityRequestID:          strings.TrimSpace(payload.CapacityRequestID),
		PhaseID:                    strings.TrimSpace(payload.PhaseID),
		TaskRef:                    strings.TrimSpace(payload.TaskRef),
		ReasonCode:                 strings.TrimSpace(payload.ReasonCode),
		Summary:                    strings.TrimSpace(payload.Summary),
		MinimumRecommendedCapacity: strings.TrimSpace(payload.MinimumRecommendedCapacity),
		EvidenceRefs:               compactDirectorAgentStringsV0(payload.EvidenceRefs),
	}
}

func normalizeDirectorAgentRequestAgentV0(
	payload DirectorAgentRequestAgentCommandV0,
) DirectorAgentRequestAgentCommandV0 {
	return DirectorAgentRequestAgentCommandV0{
		AgentRequestID:     strings.TrimSpace(payload.AgentRequestID),
		PhaseID:            strings.TrimSpace(payload.PhaseID),
		TaskRef:            strings.TrimSpace(payload.TaskRef),
		CapacityRequestRef: strings.TrimSpace(payload.CapacityRequestRef),
		Role:               strings.TrimSpace(payload.Role),
		Summary:            strings.TrimSpace(payload.Summary),
		EvidenceRefs:       compactDirectorAgentStringsV0(payload.EvidenceRefs),
	}
}

func normalizeDirectorAgentRequestReworkV0(
	payload DirectorAgentRequestReworkCommandV0,
) DirectorAgentRequestReworkCommandV0 {
	return DirectorAgentRequestReworkCommandV0{
		ReworkRequestRef: strings.TrimSpace(payload.ReworkRequestRef),
		PhaseID:          strings.TrimSpace(payload.PhaseID),
		ReviewResultRef:  strings.TrimSpace(payload.ReviewResultRef),
		ReviewRequestID:  strings.TrimSpace(payload.ReviewRequestID),
		DeliveryRef:      strings.TrimSpace(payload.DeliveryRef),
		Summary:          strings.TrimSpace(payload.Summary),
		EvidenceRefs:     compactDirectorAgentStringsV0(payload.EvidenceRefs),
	}
}

func normalizeDirectorAgentReplanDecisionV0(
	payload DirectorAgentReplanDecisionCommandV0,
) DirectorAgentReplanDecisionCommandV0 {
	return DirectorAgentReplanDecisionCommandV0{
		ReplanRef:      strings.TrimSpace(payload.ReplanRef),
		RunRef:         strings.TrimSpace(payload.RunRef),
		TaskRef:        strings.TrimSpace(payload.TaskRef),
		SourceRef:      strings.TrimSpace(payload.SourceRef),
		AcceptedAction: strings.TrimSpace(payload.AcceptedAction),
		FollowupRefs:   compactDirectorAgentStringsV0(payload.FollowupRefs),
		Summary:        strings.TrimSpace(payload.Summary),
		EvidenceRefs:   compactDirectorAgentStringsV0(payload.EvidenceRefs),
	}
}
