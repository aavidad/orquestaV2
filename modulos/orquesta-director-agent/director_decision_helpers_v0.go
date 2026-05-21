package orquestadirectoragent

import "strings"

const (
	maxDirectorAgentStringV0         = 300
	maxDirectorAgentEvidenceRefsV0   = 10
	maxDirectorAgentRecursionLimitV0 = 40
)

var forbiddenDirectorAgentFragmentsV0 = []string{
	"/home/", "\\home\\", "oauth", "token", "secret", "secreto", "password",
	"credential", "credencial", "api_key", "transcript", "prompt completo",
	"provider", "proveedor", "model", "modelo", "codex", "claude", "gemini",
	"ollama", "vllm", "runtime", "database", "base de datos", "base_de_datos",
	"sqlite", "postgres", "mysql", "mongodb", "dsn",
}

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
		Title:                strings.TrimSpace(task.Title),
		Summary:              strings.TrimSpace(task.Summary),
		WriteSet:             compactDirectorAgentStringsV0(task.WriteSet),
		AcceptanceCriteria:   compactDirectorAgentStringsV0(task.AcceptanceCriteria),
		RequiredTests:        compactDirectorAgentStringsV0(task.RequiredTests),
		DependsOn:            compactDirectorAgentStringsV0(task.DependsOn),
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

func directorAgentHasForbiddenDetailV0(values ...string) bool {
	for _, value := range values {
		lower := strings.ToLower(value)
		for _, fragment := range forbiddenDirectorAgentFragmentsV0 {
			if directorAgentContainsForbiddenFragmentV0(lower, fragment) {
				return true
			}
		}
	}
	return false
}

func directorAgentContainsForbiddenFragmentV0(lowerValue string, fragment string) bool {
	fragment = strings.ToLower(strings.TrimSpace(fragment))
	if fragment == "" {
		return false
	}
	if directorAgentFragmentHasSeparatorV0(fragment) {
		return strings.Contains(lowerValue, fragment)
	}
	start := 0
	for {
		index := strings.Index(lowerValue[start:], fragment)
		if index < 0 {
			return false
		}
		absolute := start + index
		if directorAgentHasTokenBoundaryV0(lowerValue, absolute, absolute+len(fragment)) {
			return true
		}
		start = absolute + len(fragment)
	}
}

func directorAgentFragmentHasSeparatorV0(fragment string) bool {
	for i := 0; i < len(fragment); i++ {
		if !directorAgentIsAsciiLetterOrDigitV0(fragment[i]) {
			return true
		}
	}
	return false
}

func directorAgentHasTokenBoundaryV0(value string, start int, end int) bool {
	before := start == 0 || !directorAgentIsAsciiLetterOrDigitV0(value[start-1])
	after := end >= len(value) || !directorAgentIsAsciiLetterOrDigitV0(value[end])
	return before && after
}

func directorAgentIsAsciiLetterOrDigitV0(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9')
}

func directorAgentRefCompactV0(value string) bool {
	if len(value) < 2 || len(value) > 160 {
		return false
	}
	for i := 0; i < len(value); i++ {
		b := value[i]
		if (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') {
			continue
		}
		if b == '.' || b == '_' || b == ':' || b == '-' {
			continue
		}
		return false
	}
	return true
}
