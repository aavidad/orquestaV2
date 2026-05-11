package orquestacorereplanner

import (
	"encoding/json"
	"strings"
)

func validateReplanDecisionRequiredFieldsV0(decision ReplanDecisionV0) error {
	fields := []struct {
		name  string
		value string
	}{
		{name: "replan_ref", value: decision.ReplanRef},
		{name: "run_ref", value: decision.RunRef},
		{name: "task_ref", value: decision.TaskRef},
		{name: "source_ref", value: decision.SourceRef},
		{name: "accepted_action", value: string(decision.AcceptedAction)},
		{name: "summary", value: decision.Summary},
	}
	for _, field := range fields {
		if strings.TrimSpace(field.value) == "" {
			return replanDecisionErrorV0(ErrReplanDecisionInvalidaV0, field.name)
		}
	}
	return nil
}

func validateReplanDecisionActionV0(action ReplanRecommendedActionV0) error {
	if !IsSupportedReplanRecommendedActionV0(action) {
		return replanDecisionErrorV0(ErrReplanDecisionActionInvalidaV0, "accepted_action")
	}
	return nil
}

func validateReplanDecisionCollectionsV0(decision ReplanDecisionV0) error {
	if len(decision.FollowupRefs) == 0 {
		return replanDecisionErrorV0(ErrReplanDecisionInvalidaV0, "followup_refs")
	}
	if len(decision.FollowupRefs) > maxReplanProposalEvidenceRefsV0 {
		return replanDecisionErrorV0(ErrReplanDecisionInvalidaV0, "followup_refs")
	}
	if len(decision.EvidenceRefs) > maxReplanProposalEvidenceRefsV0 {
		return replanDecisionErrorV0(ErrReplanDecisionInvalidaV0, "evidence_refs")
	}
	for _, ref := range decision.FollowupRefs {
		if strings.TrimSpace(ref) == "" {
			return replanDecisionErrorV0(ErrReplanDecisionInvalidaV0, "followup_refs")
		}
	}
	for _, ref := range decision.EvidenceRefs {
		if strings.TrimSpace(ref) == "" {
			return replanDecisionErrorV0(ErrReplanDecisionInvalidaV0, "evidence_refs")
		}
	}
	return nil
}

func validateReplanDecisionCompactPayloadV0(decision ReplanDecisionV0) error {
	data, err := json.Marshal(decision)
	if err != nil {
		return replanDecisionErrorV0(ErrReplanDecisionPayloadInvalidoV0, "payload")
	}
	if len(data) == 0 || len(data) > maxReplanProposalPayloadBytesV0 {
		return replanDecisionErrorV0(ErrReplanDecisionPayloadInvalidoV0, "payload")
	}
	return nil
}

func replanDecisionTextFieldsV0(decision ReplanDecisionV0) []string {
	values := []string{
		decision.ReplanRef,
		decision.RunRef,
		decision.TaskRef,
		decision.SourceRef,
		string(decision.AcceptedAction),
		decision.Summary,
	}
	values = append(values, decision.FollowupRefs...)
	return append(values, decision.EvidenceRefs...)
}

func replanDecisionErrorV0(code string, field string) ReplanDecisionErrorV0 {
	return ReplanDecisionErrorV0{Code: code, Field: field}
}
