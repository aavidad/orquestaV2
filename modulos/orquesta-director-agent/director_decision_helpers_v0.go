package orquestadirectoragent

import (
	"strings"
)

const (
	maxDirectorAgentStringV0         = 300
	maxDirectorAgentEvidenceRefsV0   = 10
	maxDirectorAgentTextListV0       = 24
	maxDirectorAgentRecursionLimitV0 = 40
)

func normalizeDirectorAgentDecisionV0(decision DirectorAgentDecisionV0) DirectorAgentDecisionV0 {
	decision.SchemaVersion = strings.TrimSpace(decision.SchemaVersion)
	decision.DecisionRef = strings.TrimSpace(decision.DecisionRef)
	decision.RunID = strings.TrimSpace(decision.RunID)
	decision.PhaseID = strings.TrimSpace(decision.PhaseID)
	decision.CommandType = strings.TrimSpace(decision.CommandType)
	decision.CommandRef = strings.TrimSpace(decision.CommandRef)
	decision.Summary = strings.TrimSpace(decision.Summary)
	decision.EvidenceRefs = compactDirectorAgentStringsV0(decision.EvidenceRefs)
	if decision.RequestBrainstorm != nil {
		normalized := normalizeDirectorAgentBrainstormV0(*decision.RequestBrainstorm)
		decision.RequestBrainstorm = &normalized
	}
	if decision.OpenPhase != nil {
		normalized := normalizeDirectorAgentOpenPhaseV0(*decision.OpenPhase)
		decision.OpenPhase = &normalized
	}
	if decision.RequestVote != nil {
		normalized := normalizeDirectorAgentVoteV0(*decision.RequestVote)
		decision.RequestVote = &normalized
	}
	if decision.AcceptDecision != nil {
		normalized := normalizeDirectorAgentAcceptDecisionV0(*decision.AcceptDecision)
		decision.AcceptDecision = &normalized
	}
	if decision.PublishContract != nil {
		normalized := normalizeDirectorAgentPublishContractV0(*decision.PublishContract)
		decision.PublishContract = &normalized
	}
	if decision.CreateMicrotask != nil {
		normalized := normalizeDirectorAgentCreateMicrotaskV0(*decision.CreateMicrotask)
		decision.CreateMicrotask = &normalized
	}
	normalizeDirectorAgentRoutingDecisionV0(&decision)
	if decision.RequestReview != nil {
		normalized := normalizeDirectorAgentRequestReviewV0(*decision.RequestReview)
		decision.RequestReview = &normalized
	}
	if decision.RecordReviewResult != nil {
		normalized := normalizeDirectorAgentReviewResultV0(*decision.RecordReviewResult)
		decision.RecordReviewResult = &normalized
	}
	if decision.AcceptReview != nil {
		normalized := normalizeDirectorAgentAcceptReviewV0(*decision.AcceptReview)
		decision.AcceptReview = &normalized
	}
	if decision.CloseTask != nil {
		normalized := normalizeDirectorAgentCloseTaskV0(*decision.CloseTask)
		decision.CloseTask = &normalized
	}
	if decision.RegisterFinalValidation != nil {
		normalized := normalizeDirectorAgentFinalValidationV0(*decision.RegisterFinalValidation)
		decision.RegisterFinalValidation = &normalized
	}
	if decision.CloseRun != nil {
		normalized := normalizeDirectorAgentCloseRunV0(*decision.CloseRun)
		decision.CloseRun = &normalized
	}
	if decision.ProposePlanTeam != nil {
		normalized := normalizeDirectorAgentPlanTeamCommandV0(*decision.ProposePlanTeam)
		decision.ProposePlanTeam = &normalized
	}
	if decision.AnswerQuestion != nil {
		normalized := normalizeDirectorAgentAnswerQuestionV0(*decision.AnswerQuestion)
		decision.AnswerQuestion = &normalized
	}
	return decision
}

func normalizeDirectorAgentBrainstormV0(
	payload DirectorAgentBrainstormCommandV0,
) DirectorAgentBrainstormCommandV0 {
	return DirectorAgentBrainstormCommandV0{
		BrainstormRequestID:        strings.TrimSpace(payload.BrainstormRequestID),
		PhaseID:                    strings.TrimSpace(payload.PhaseID),
		TopicRef:                   strings.TrimSpace(payload.TopicRef),
		Summary:                    strings.TrimSpace(payload.Summary),
		MinimumRecommendedCapacity: strings.TrimSpace(payload.MinimumRecommendedCapacity),
		EvidenceRefs:               compactDirectorAgentStringsV0(payload.EvidenceRefs),
	}
}

func normalizeDirectorAgentOpenPhaseV0(
	payload DirectorAgentOpenPhaseCommandV0,
) DirectorAgentOpenPhaseCommandV0 {
	return DirectorAgentOpenPhaseCommandV0{
		PhaseID: strings.TrimSpace(payload.PhaseID),
		Reason:  strings.TrimSpace(payload.Reason),
	}
}

func normalizeDirectorAgentVoteV0(
	payload DirectorAgentVoteCommandV0,
) DirectorAgentVoteCommandV0 {
	return DirectorAgentVoteCommandV0{
		VoteRequestID:              strings.TrimSpace(payload.VoteRequestID),
		PhaseID:                    strings.TrimSpace(payload.PhaseID),
		DecisionTopicRef:           strings.TrimSpace(payload.DecisionTopicRef),
		BrainstormRef:              strings.TrimSpace(payload.BrainstormRef),
		Summary:                    strings.TrimSpace(payload.Summary),
		MinimumRecommendedCapacity: strings.TrimSpace(payload.MinimumRecommendedCapacity),
		EvidenceRefs:               compactDirectorAgentStringsV0(payload.EvidenceRefs),
	}
}

func normalizeDirectorAgentAcceptDecisionV0(
	payload DirectorAgentAcceptDecisionCommandV0,
) DirectorAgentAcceptDecisionCommandV0 {
	return DirectorAgentAcceptDecisionCommandV0{
		DecisionRef:       strings.TrimSpace(payload.DecisionRef),
		PhaseID:           strings.TrimSpace(payload.PhaseID),
		VoteRef:           strings.TrimSpace(payload.VoteRef),
		AcceptedOptionRef: strings.TrimSpace(payload.AcceptedOptionRef),
		Summary:           strings.TrimSpace(payload.Summary),
		EvidenceRefs:      compactDirectorAgentStringsV0(payload.EvidenceRefs),
	}
}

func normalizeDirectorAgentPublishContractV0(
	payload DirectorAgentPublishContractCommandV0,
) DirectorAgentPublishContractCommandV0 {
	return DirectorAgentPublishContractCommandV0{
		ContractRef:   strings.TrimSpace(payload.ContractRef),
		PhaseID:       strings.TrimSpace(payload.PhaseID),
		DecisionRef:   strings.TrimSpace(payload.DecisionRef),
		Summary:       strings.TrimSpace(payload.Summary),
		FunctionNames: compactDirectorAgentStringsV0(payload.FunctionNames),
		EvidenceRefs:  compactDirectorAgentStringsV0(payload.EvidenceRefs),
	}
}

func normalizeDirectorAgentCreateMicrotaskV0(
	payload DirectorAgentCreateMicrotaskCommandV0,
) DirectorAgentCreateMicrotaskCommandV0 {
	return DirectorAgentCreateMicrotaskCommandV0{
		Task: normalizeDirectorAgentMicrotaskV0(payload.Task),
	}
}

func normalizeDirectorAgentRequestReviewV0(
	payload DirectorAgentRequestReviewCommandV0,
) DirectorAgentRequestReviewCommandV0 {
	return DirectorAgentRequestReviewCommandV0{
		ReviewRequestID: strings.TrimSpace(payload.ReviewRequestID),
		PhaseID:         strings.TrimSpace(payload.PhaseID),
		DeliveryRef:     strings.TrimSpace(payload.DeliveryRef),
		Summary:         strings.TrimSpace(payload.Summary),
		EvidenceRefs:    compactDirectorAgentStringsV0(payload.EvidenceRefs),
	}
}

func normalizeDirectorAgentReviewResultV0(
	payload DirectorAgentReviewResultCommandV0,
) DirectorAgentReviewResultCommandV0 {
	return DirectorAgentReviewResultCommandV0{
		ReviewResultRef: strings.TrimSpace(payload.ReviewResultRef),
		ReviewRequestID: strings.TrimSpace(payload.ReviewRequestID),
		DeliveryRef:     strings.TrimSpace(payload.DeliveryRef),
		Status:          strings.TrimSpace(payload.Status),
		Summary:         strings.TrimSpace(payload.Summary),
		EvidenceRefs:    compactDirectorAgentStringsV0(payload.EvidenceRefs),
		QualityGateRef:  strings.TrimSpace(payload.QualityGateRef),
	}
}

func normalizeDirectorAgentAcceptReviewV0(
	payload DirectorAgentAcceptReviewCommandV0,
) DirectorAgentAcceptReviewCommandV0 {
	return DirectorAgentAcceptReviewCommandV0{
		AcceptedReviewRef: strings.TrimSpace(payload.AcceptedReviewRef),
		PhaseID:           strings.TrimSpace(payload.PhaseID),
		ReviewRequestID:   strings.TrimSpace(payload.ReviewRequestID),
		DeliveryRef:       strings.TrimSpace(payload.DeliveryRef),
		Summary:           strings.TrimSpace(payload.Summary),
		EvidenceRefs:      compactDirectorAgentStringsV0(payload.EvidenceRefs),
	}
}

func normalizeDirectorAgentAnswerQuestionV0(
	payload DirectorAgentAnswerQuestionCommandV0,
) DirectorAgentAnswerQuestionCommandV0 {
	return DirectorAgentAnswerQuestionCommandV0{
		AnswerID:     strings.TrimSpace(payload.AnswerID),
		QuestionID:   strings.TrimSpace(payload.QuestionID),
		Decision:     strings.TrimSpace(payload.Decision),
		Summary:      strings.TrimSpace(payload.Summary),
		EvidenceRefs: compactDirectorAgentStringsV0(payload.EvidenceRefs),
		Unblocks:     payload.Unblocks,
	}
}

func normalizeDirectorAgentMicrotaskV0(task DirectorAgentMicrotaskV0) DirectorAgentMicrotaskV0 {
	return DirectorAgentMicrotaskV0{
		SchemaVersion:        strings.TrimSpace(task.SchemaVersion),
		TaskID:               strings.TrimSpace(task.TaskID),
		RunID:                strings.TrimSpace(task.RunID),
		PhaseID:              strings.TrimSpace(task.PhaseID),
		WorkProfileKind:      strings.TrimSpace(task.WorkProfileKind),
		Title:                strings.TrimSpace(task.Title),
		Summary:              strings.TrimSpace(task.Summary),
		WriteSet:             compactDirectorAgentStringsV0(task.WriteSet),
		AcceptanceCriteria:   compactDirectorAgentStringsV0(task.AcceptanceCriteria),
		RequiredTests:        compactDirectorAgentStringsV0(task.RequiredTests),
		DependsOn:            compactDirectorAgentStringsV0(task.DependsOn),
		ContextRefs:          compactDirectorAgentStringsV0(task.ContextRefs),
		ParentTaskRef:        strings.TrimSpace(task.ParentTaskRef),
		CohortRef:            strings.TrimSpace(task.CohortRef),
		WaveRef:              strings.TrimSpace(task.WaveRef),
		DelegationDepth:      task.DelegationDepth,
		MaxChildAgents:       task.MaxChildAgents,
		ChildTaskRefs:        compactDirectorAgentStringsV0(task.ChildTaskRefs),
		FunctionContractRefs: normalizeDirectorAgentFunctionRefsV0(task.FunctionContractRefs),
	}
}

func normalizeDirectorAgentFunctionRefsV0(
	refs []DirectorAgentFunctionContractRefV0,
) []DirectorAgentFunctionContractRefV0 {
	normalized := make([]DirectorAgentFunctionContractRefV0, 0, len(refs))
	seen := map[DirectorAgentFunctionContractRefV0]struct{}{}
	for _, ref := range refs {
		compact := DirectorAgentFunctionContractRefV0{
			ContractRef:  strings.TrimSpace(ref.ContractRef),
			FunctionName: strings.TrimSpace(ref.FunctionName),
		}
		if compact.ContractRef == "" && compact.FunctionName == "" {
			continue
		}
		if _, ok := seen[compact]; ok {
			continue
		}
		seen[compact] = struct{}{}
		normalized = append(normalized, compact)
	}
	return normalized
}

func compactDirectorAgentStringsV0(values []string) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}
