package orquestaautoprogramming

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

const (
	AutoprogrammingSelfImprovementDefaultPriorityScoreV0 = 10
	AutoprogrammingSelfImprovementDefaultAreaV0          = "automejora"
)

type AutoprogrammingSelfImprovementProposalV0 struct {
	RequestRef         string   `json:"request_ref,omitempty"`
	ProjectRef         string   `json:"project_ref"`
	WorktreeRef        string   `json:"worktree_ref,omitempty"`
	WorktreeIsolated   bool     `json:"worktree_isolated,omitempty"`
	BranchRef          string   `json:"branch_ref,omitempty"`
	ObservedBy         string   `json:"observed_by,omitempty"`
	SourceRunRef       string   `json:"source_run_ref,omitempty"`
	SourceTaskRef      string   `json:"source_task_ref,omitempty"`
	FailureKind        string   `json:"failure_kind,omitempty"`
	FailureSummary     string   `json:"failure_summary"`
	SuggestedArea      string   `json:"suggested_area,omitempty"`
	SuggestedWriteSet  []string `json:"suggested_write_set,omitempty"`
	RequiredTests      []string `json:"required_tests,omitempty"`
	AcceptanceCriteria []string `json:"acceptance_criteria,omitempty"`
	CompactRules       []string `json:"compact_rules,omitempty"`
	ContextRefs        []string `json:"context_refs,omitempty"`
	EvidenceRefs       []string `json:"evidence_refs,omitempty"`
	PriorityScore      int      `json:"priority_score,omitempty"`
}

type AutoprogrammingSelfImprovementResultV0 struct {
	Accepted      bool                            `json:"accepted"`
	Request       AutoprogrammingRequestV0        `json:"request,omitempty"`
	PriorityScore int                             `json:"priority_score"`
	Background    bool                            `json:"background"`
	NextActions   []string                        `json:"next_actions,omitempty"`
	Issues        []AutoprogrammingRequestIssueV0 `json:"issues,omitempty"`
}

func BuildAutoprogrammingSelfImprovementRequestV0(
	proposal AutoprogrammingSelfImprovementProposalV0,
) AutoprogrammingSelfImprovementResultV0 {
	proposal = normalizeAutoprogrammingSelfImprovementProposalV0(proposal)
	result := AutoprogrammingSelfImprovementResultV0{
		PriorityScore: autoprogrammingSelfImprovementPriorityScoreV0(proposal.PriorityScore),
		Background:    true,
	}
	issues := autoprogrammingSelfImprovementIssuesV0(proposal)
	if len(issues) > 0 {
		result.Issues = issues
		result.NextActions = autoprogrammingSelfImprovementNextActionsV0(issues)
		return result
	}
	request := autoprogrammingSelfImprovementRequestV0(proposal)
	validation := ValidateAutoprogrammingRequestV0(request)
	if !validation.Accepted {
		result.Issues = append([]AutoprogrammingRequestIssueV0(nil), validation.Issues...)
		result.NextActions = autoprogrammingSelfImprovementNextActionsV0(result.Issues)
		return result
	}
	result.Accepted = true
	result.Request = request
	result.NextActions = []string{"prepare_run_with_low_priority", "do_not_block_primary_work"}
	return result
}

func normalizeAutoprogrammingSelfImprovementProposalV0(
	proposal AutoprogrammingSelfImprovementProposalV0,
) AutoprogrammingSelfImprovementProposalV0 {
	proposal.RequestRef = strings.TrimSpace(proposal.RequestRef)
	proposal.ProjectRef = strings.TrimSpace(proposal.ProjectRef)
	proposal.WorktreeRef = strings.TrimSpace(proposal.WorktreeRef)
	proposal.BranchRef = strings.TrimSpace(proposal.BranchRef)
	proposal.ObservedBy = strings.TrimSpace(proposal.ObservedBy)
	proposal.SourceRunRef = strings.TrimSpace(proposal.SourceRunRef)
	proposal.SourceTaskRef = strings.TrimSpace(proposal.SourceTaskRef)
	proposal.FailureKind = normalizeAutoprogrammingTaskAreaV0(proposal.FailureKind)
	proposal.FailureSummary = strings.TrimSpace(proposal.FailureSummary)
	proposal.SuggestedArea = normalizeAutoprogrammingTaskAreaV0(proposal.SuggestedArea)
	proposal.SuggestedWriteSet = compactStringsV0(proposal.SuggestedWriteSet)
	proposal.RequiredTests = compactStringsV0(proposal.RequiredTests)
	proposal.AcceptanceCriteria = compactStringsV0(proposal.AcceptanceCriteria)
	proposal.CompactRules = compactStringsV0(proposal.CompactRules)
	proposal.ContextRefs = compactStringsV0(proposal.ContextRefs)
	proposal.EvidenceRefs = compactStringsV0(proposal.EvidenceRefs)
	if proposal.RequestRef == "" {
		proposal.RequestRef = "request-ref-self-improvement-" + autoprogrammingSelfImprovementHashV0(
			proposal.ProjectRef+"|"+proposal.SourceRunRef+"|"+proposal.SourceTaskRef+"|"+proposal.FailureSummary,
		)
	}
	return proposal
}

func autoprogrammingSelfImprovementIssuesV0(
	proposal AutoprogrammingSelfImprovementProposalV0,
) []AutoprogrammingRequestIssueV0 {
	var issues []AutoprogrammingRequestIssueV0
	for _, item := range []struct {
		field string
		value string
	}{
		{"project_ref", proposal.ProjectRef},
		{"failure_summary", proposal.FailureSummary},
		{"worktree_ref", proposal.WorktreeRef},
		{"branch_ref", proposal.BranchRef},
	} {
		if item.value == "" {
			issues = append(issues, autoprogrammingRequestIssueV0(
				item.field+"_missing",
				item.field,
				item.field+" requerido para preparar automejora",
			))
		}
	}
	if !proposal.WorktreeIsolated {
		issues = append(issues, autoprogrammingRequestIssueV0(
			"worktree_not_isolated",
			"worktree_isolated",
			"automejora en segundo plano requiere worktree aislada",
		))
	}
	if len(proposal.SuggestedWriteSet) == 0 {
		issues = append(issues, autoprogrammingRequestIssueV0(
			"write_set_missing",
			"suggested_write_set",
			"write-set propio requerido para no pisar el trabajo principal",
		))
	}
	if len(proposal.RequiredTests) == 0 {
		issues = append(issues, autoprogrammingRequestIssueV0(
			"required_tests_missing",
			"required_tests",
			"tests requeridos para cerrar automejora",
		))
	}
	return issues
}

func autoprogrammingSelfImprovementRequestV0(
	proposal AutoprogrammingSelfImprovementProposalV0,
) AutoprogrammingRequestV0 {
	taskRef := "task-ref-self-improvement-" + autoprogrammingSelfImprovementHashV0(proposal.RequestRef)
	area := proposal.SuggestedArea
	if area == "" {
		area = AutoprogrammingSelfImprovementDefaultAreaV0
	}
	return AutoprogrammingRequestV0{
		RequestRef:       proposal.RequestRef,
		ProjectRef:       proposal.ProjectRef,
		WorktreeRef:      proposal.WorktreeRef,
		WorktreeIsolated: true,
		BranchRef:        proposal.BranchRef,
		Tasks: []AutoprogrammingTaskGroupCandidateV0{{
			TaskRef:            taskRef,
			Area:               area,
			Title:              "Automejora en segundo plano",
			Objective:          "Corregir de forma general un patron detectado: " + proposal.FailureSummary,
			Context:            autoprogrammingSelfImprovementContextV0(proposal),
			ContextRefs:        autoprogrammingSelfImprovementContextRefsV0(proposal),
			AcceptanceCriteria: append([]string(nil), proposal.AcceptanceCriteria...),
			RequiredTests:      append([]string(nil), proposal.RequiredTests...),
			CompactRules:       autoprogrammingSelfImprovementRulesV0(proposal),
		}},
		WriteSet:           append([]string(nil), proposal.SuggestedWriteSet...),
		RequiredTests:      append([]string(nil), proposal.RequiredTests...),
		MaxTaskRefs:        1,
		MaxAreas:           1,
		MaxWriteSetEntries: positiveAutoprogrammingSelfImprovementLimitV0(len(proposal.SuggestedWriteSet)),
	}
}

func autoprogrammingSelfImprovementContextV0(
	proposal AutoprogrammingSelfImprovementProposalV0,
) []string {
	var context []string
	for _, item := range []struct {
		key   string
		value string
	}{
		{"observed_by", proposal.ObservedBy},
		{"source_run_ref", proposal.SourceRunRef},
		{"source_task_ref", proposal.SourceTaskRef},
		{"failure_kind", proposal.FailureKind},
		{"failure_summary", proposal.FailureSummary},
	} {
		if strings.TrimSpace(item.value) == "" {
			continue
		}
		context = append(context, item.key+":"+strings.TrimSpace(item.value))
	}
	return compactStringsV0(context)
}

func autoprogrammingSelfImprovementContextRefsV0(
	proposal AutoprogrammingSelfImprovementProposalV0,
) []string {
	refs := append([]string(nil), proposal.ContextRefs...)
	for _, ref := range proposal.EvidenceRefs {
		refs = append(refs, "evidence_ref:"+ref)
	}
	if proposal.SourceRunRef != "" {
		refs = append(refs, "source_run_ref:"+proposal.SourceRunRef)
	}
	if proposal.SourceTaskRef != "" {
		refs = append(refs, "source_task_ref:"+proposal.SourceTaskRef)
	}
	return compactStringsV0(refs)
}

func autoprogrammingSelfImprovementRulesV0(
	proposal AutoprogrammingSelfImprovementProposalV0,
) []string {
	return compactStringsV0(append([]string{
		"trabajo secundario: no bloquear ni mezclar con el trabajo principal",
		"usar evidencia del fallo y corregir la causa general si es posible",
	}, proposal.CompactRules...))
}

func autoprogrammingSelfImprovementPriorityScoreV0(value int) int {
	if value > 0 {
		return value
	}
	return AutoprogrammingSelfImprovementDefaultPriorityScoreV0
}

func autoprogrammingSelfImprovementNextActionsV0(
	issues []AutoprogrammingRequestIssueV0,
) []string {
	actions := []string{"preserve_failure_evidence", "do_not_discard_primary_work"}
	for _, issue := range issues {
		switch strings.TrimSpace(issue.Field) {
		case "worktree_ref", "branch_ref", "worktree_isolated":
			actions = append(actions, "complete_isolated_worktree_before_prepare_run")
		case "suggested_write_set":
			actions = append(actions, "ask_director_to_study_write_set")
		case "required_tests":
			actions = append(actions, "define_required_tests_before_prepare_run")
		}
	}
	return compactStringsV0(actions)
}

func autoprogrammingSelfImprovementHashV0(value string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(value)))
	return hex.EncodeToString(sum[:])[:12]
}

func positiveAutoprogrammingSelfImprovementLimitV0(value int) int {
	if value > 0 {
		return value
	}
	return 1
}
