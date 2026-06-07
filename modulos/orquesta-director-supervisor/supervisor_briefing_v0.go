package orquestadirectorsupervisor

func BuildDirectorSupervisorBriefingV0(
	input DirectorSupervisorBriefingInputV0,
) (DirectorSupervisorBriefingV0, error) {
	input = normalizeDirectorSupervisorBriefingInputV0(input)
	if err := validateDirectorSupervisorBriefingInputV0(input); err != nil {
		return DirectorSupervisorBriefingV0{}, err
	}
	decision := input.Decision
	action := supervisorRecommendedActionForDecisionV0(decision)
	briefing := DirectorSupervisorBriefingV0{
		SchemaVersion:              DirectorSupervisorBriefingSchemaV0,
		RunRef:                     decision.RunRef,
		ObjectiveRef:               input.ObjectiveRef,
		DecisionAction:             decision.Action,
		AutonomousRecommendation:   decision.AutonomousRecommendation,
		ReasonCode:                 decision.ReasonCode,
		ActionQueue:                []DirectorSupervisorRecommendedActionV0{action},
		Timeline:                   []DirectorSupervisorTimelineEventV0{supervisorTimelineEventForActionV0(decision, action)},
		PendingOutboxRefs:          compactSupervisorStringsV0(decision.PendingOutboxRefs),
		WaitingReasons:             supervisorWaitingReasonStringsV0(decision.WaitingReasons),
		BlockedRefs:                compactSupervisorStringsV0(decision.BlockedRefs),
		ContextRefs:                compactSupervisorStringsV0(input.ContextRefs),
		EvidenceRefs:               compactSupervisorStringsV0(decision.EvidenceRefs),
		StopProjectionPublicReason: decision.StopProjection.PublicReason,
	}
	briefing.NextAction = &briefing.ActionQueue[0]
	return briefing, nil
}

func supervisorRecommendedActionForDecisionV0(
	decision DirectorSupervisorDecisionV0,
) DirectorSupervisorRecommendedActionV0 {
	kind, priority, safe, requiresDirector := supervisorActionKindV0(decision.Action)
	return DirectorSupervisorRecommendedActionV0{
		ActionRef:        supervisorActionRefV0(decision, kind),
		Kind:             kind,
		RunRef:           decision.RunRef,
		ReasonCode:       decision.ReasonCode,
		Priority:         priority,
		SafeToApply:      safe,
		RequiresDirector: requiresDirector,
		SourceAction:     decision.Action,
		TargetRefs:       supervisorActionTargetRefsV0(decision),
		EvidenceRefs:     compactSupervisorStringsV0(decision.EvidenceRefs),
	}
}

func supervisorTimelineEventForActionV0(
	decision DirectorSupervisorDecisionV0,
	action DirectorSupervisorRecommendedActionV0,
) DirectorSupervisorTimelineEventV0 {
	return DirectorSupervisorTimelineEventV0{
		EventRef:     action.ActionRef + ":event",
		Kind:         action.Kind,
		RunRef:       decision.RunRef,
		StepNumber:   decision.StepNumber,
		ReasonCode:   decision.ReasonCode,
		SourceAction: decision.Action,
		TargetRefs:   append(action.TargetRefs[:0:0], action.TargetRefs...),
		EvidenceRefs: append(action.EvidenceRefs[:0:0], action.EvidenceRefs...),
	}
}
