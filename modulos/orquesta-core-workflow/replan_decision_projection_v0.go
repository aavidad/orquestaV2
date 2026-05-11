package orquestacoreworkflow

import "strings"

const (
	replanDecisionProjectionSourceSeparatorV0    = "#source:"
	replanDecisionProjectionTaskSeparatorV0      = "#task:"
	replanDecisionProjectionActionSeparatorV0    = "#action:"
	replanDecisionProjectionFollowupsSeparatorV0 = "#followups:"
	replanDecisionProjectionFollowupsJoinerV0    = "+"
)

type replanDecisionProjectionPartsV0 struct {
	ReplanRef      string
	SourceRef      string
	TaskRef        string
	AcceptedAction ReplanDecisionActionV0
	FollowupRefs   []string
}

func replanDecisionProjectionRefV0(payload ReplanDecisionRecordedPayloadV0) string {
	normalized := normalizeReplanDecisionRecordedPayloadV0(payload)
	return normalized.ReplanRef +
		replanDecisionProjectionSourceSeparatorV0 + normalized.SourceRef +
		replanDecisionProjectionTaskSeparatorV0 + normalized.TaskRef +
		replanDecisionProjectionActionSeparatorV0 + string(normalized.AcceptedAction) +
		replanDecisionProjectionFollowupsSeparatorV0 + strings.Join(normalized.FollowupRefs, replanDecisionProjectionFollowupsJoinerV0)
}

func replanDecisionProjectionPartsFromRefV0(ref string) (replanDecisionProjectionPartsV0, bool) {
	replanRef, tail, ok := strings.Cut(strings.TrimSpace(ref), replanDecisionProjectionSourceSeparatorV0)
	if !ok {
		return replanDecisionProjectionPartsV0{}, false
	}
	sourceRef, tail, ok := strings.Cut(tail, replanDecisionProjectionTaskSeparatorV0)
	if !ok {
		return replanDecisionProjectionPartsV0{}, false
	}
	taskRef, tail, ok := strings.Cut(tail, replanDecisionProjectionActionSeparatorV0)
	if !ok {
		return replanDecisionProjectionPartsV0{}, false
	}
	action, followups, ok := strings.Cut(tail, replanDecisionProjectionFollowupsSeparatorV0)
	if !ok {
		return replanDecisionProjectionPartsV0{}, false
	}
	parts := replanDecisionProjectionPartsV0{
		ReplanRef:      strings.TrimSpace(replanRef),
		SourceRef:      strings.TrimSpace(sourceRef),
		TaskRef:        strings.TrimSpace(taskRef),
		AcceptedAction: ReplanDecisionActionV0(strings.TrimSpace(action)),
		FollowupRefs:   replanDecisionProjectionFollowupsV0(followups),
	}
	return parts, parts.ReplanRef != "" && parts.SourceRef != "" && parts.TaskRef != "" &&
		isSupportedReplanDecisionActionV0(parts.AcceptedAction) && len(parts.FollowupRefs) > 0
}

func replanDecisionProjectionFollowupsV0(joined string) []string {
	raw := strings.Split(strings.TrimSpace(joined), replanDecisionProjectionFollowupsJoinerV0)
	result := make([]string, 0, len(raw))
	for _, ref := range raw {
		if compact := strings.TrimSpace(ref); compact != "" {
			result = append(result, compact)
		}
	}
	return result
}

func replanDecisionProjectionForRefV0(current OrchestrationRunV0, replanRef string) (string, bool) {
	replanRef = strings.TrimSpace(replanRef)
	for _, existing := range current.ReplanDecisions {
		parts, ok := replanDecisionProjectionPartsFromRefV0(existing)
		if ok && parts.ReplanRef == replanRef {
			return strings.TrimSpace(existing), true
		}
	}
	return "", false
}

func replanDecisionMatchesPayloadV0(current OrchestrationRunV0, payload RecordReplanDecisionCommandPayloadV0) (bool, bool) {
	existing, ok := replanDecisionProjectionForRefV0(current, payload.ReplanRef)
	if !ok {
		return false, false
	}
	expected := replanDecisionProjectionRefV0(replanDecisionRecordedPayloadFromCommandV0(payload))
	return existing == expected, existing != expected
}

func replanDecisionEventMatchesPayloadV0(current OrchestrationRunV0, payload ReplanDecisionRecordedPayloadV0) (bool, bool) {
	existing, ok := replanDecisionProjectionForRefV0(current, payload.ReplanRef)
	if !ok {
		return false, false
	}
	expected := replanDecisionProjectionRefV0(payload)
	return existing == expected, existing != expected
}

func replanDecisionRefsInvalidV0(run OrchestrationRunV0) bool {
	seen := map[string]bool{}
	for _, projection := range run.ReplanDecisions {
		parts, ok := replanDecisionProjectionPartsFromRefV0(projection)
		if !ok || seen[parts.ReplanRef] {
			return true
		}
		if !microtaskAlreadyReflectedV0(run, parts.TaskRef) ||
			!replanDecisionSourceAlreadyReflectedV0(run, parts.SourceRef) {
			return true
		}
		seen[parts.ReplanRef] = true
	}
	return false
}

func replanDecisionSourceAlreadyReflectedV0(run OrchestrationRunV0, sourceRef string) bool {
	return reworkRequestAlreadyReflectedV0(run, sourceRef) ||
		agentFailedAlreadyReflectedV0(run, sourceRef) ||
		agentAssessmentAlreadyReflectedV0(run, sourceRef) ||
		qualityGateBlockedAlreadyReflectedV0(run, sourceRef)
}

func replanDecisionProjectionFieldUnsafeV0(payload RecordReplanDecisionCommandPayloadV0) bool {
	values := []string{payload.ReplanRef, payload.RunRef, payload.TaskRef, payload.SourceRef, string(payload.AcceptedAction)}
	values = append(values, payload.FollowupRefs...)
	for _, value := range values {
		if replanDecisionProjectionContainsSeparatorV0(value) {
			return true
		}
	}
	return false
}

func replanDecisionProjectionContainsSeparatorV0(value string) bool {
	return strings.Contains(value, replanDecisionProjectionSourceSeparatorV0) ||
		strings.Contains(value, replanDecisionProjectionTaskSeparatorV0) ||
		strings.Contains(value, replanDecisionProjectionActionSeparatorV0) ||
		strings.Contains(value, replanDecisionProjectionFollowupsSeparatorV0) ||
		strings.Contains(value, replanDecisionProjectionFollowupsJoinerV0)
}
