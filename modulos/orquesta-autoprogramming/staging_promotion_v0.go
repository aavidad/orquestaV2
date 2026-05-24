package orquestaautoprogramming

import (
	"context"
	"strings"
)

const (
	AutoprogrammingStagingPromotionSchemaVersionV0 = "autoprogramming_staging_promotion.v0"

	AutoprogrammingStagingPromotionStatusReadyV0   = "ready_to_promote"
	AutoprogrammingStagingPromotionStatusPendingV0 = "promotion_pending"
	AutoprogrammingStagingPromotionStatusBlockedV0 = "promotion_blocked"

	AutoprogrammingStagingEffectPromotedV0    = "promoted"
	AutoprogrammingStagingEffectCleanV0       = "clean"
	AutoprogrammingStagingEffectPendingV0     = "pending"
	AutoprogrammingStagingEffectBlockedV0     = "blocked"
	AutoprogrammingStagingEffectArchivedV0    = "archived"
	AutoprogrammingStagingEffectPendingPushV0 = "pending_push"
)

type AutoprogrammingRequiredTestEvidenceV0 struct {
	EvidenceRef string `json:"evidence_ref"`
	TaskRef     string `json:"task_ref,omitempty"`
	TestCommand string `json:"test_command"`
	Status      string `json:"status"`
}

type AutoprogrammingStagingPromotionRequestV0 struct {
	RequestRef           string                                  `json:"request_ref,omitempty"`
	RunRef               string                                  `json:"run_ref"`
	ProjectRef           string                                  `json:"project_ref"`
	WorktreeRef          string                                  `json:"worktree_ref"`
	BranchRef            string                                  `json:"branch_ref"`
	RunClosed            bool                                    `json:"run_closed"`
	WriteSet             []string                                `json:"write_set"`
	RequiredTests        []string                                `json:"required_tests"`
	ClosedTaskRefs       []string                                `json:"closed_task_refs,omitempty"`
	AcceptedReviewRefs   []string                                `json:"accepted_review_refs,omitempty"`
	RequiredTestEvidence []AutoprogrammingRequiredTestEvidenceV0 `json:"required_test_evidence,omitempty"`
	LiveWorks            []AutoprogrammingLiveWorkV0             `json:"live_works,omitempty"`
	EvidenceRefs         []string                                `json:"evidence_refs,omitempty"`
}

type AutoprogrammingStagingPromotionDecisionV0 struct {
	Status           string                                   `json:"status"`
	Ready            bool                                     `json:"ready"`
	PromotionRef     string                                   `json:"promotion_ref,omitempty"`
	ArchiveRef       string                                   `json:"archive_ref,omitempty"`
	PromotionCommand AutoprogrammingStagingPromotionCommandV0 `json:"promotion_command,omitempty"`
	CleanupCommand   AutoprogrammingStagingCleanupCommandV0   `json:"cleanup_command,omitempty"`
	PendingRefs      []string                                 `json:"pending_refs,omitempty"`
	NextActions      []string                                 `json:"next_actions,omitempty"`
	EvidenceRefs     []string                                 `json:"evidence_refs,omitempty"`
	Issues           []AutoprogrammingRequestIssueV0          `json:"issues,omitempty"`
}

type AutoprogrammingStagingPromotionCommandV0 struct {
	PromotionRef string   `json:"promotion_ref"`
	RequestRef   string   `json:"request_ref,omitempty"`
	RunRef       string   `json:"run_ref"`
	ProjectRef   string   `json:"project_ref"`
	WorktreeRef  string   `json:"worktree_ref"`
	BranchRef    string   `json:"branch_ref"`
	WriteSet     []string `json:"write_set"`
	EvidenceRefs []string `json:"evidence_refs,omitempty"`
}

type AutoprogrammingStagingCleanupCommandV0 struct {
	ArchiveRef   string   `json:"archive_ref"`
	PromotionRef string   `json:"promotion_ref"`
	RequestRef   string   `json:"request_ref,omitempty"`
	RunRef       string   `json:"run_ref"`
	ProjectRef   string   `json:"project_ref"`
	WorktreeRef  string   `json:"worktree_ref"`
	BranchRef    string   `json:"branch_ref"`
	WriteSet     []string `json:"write_set"`
	EvidenceRefs []string `json:"evidence_refs,omitempty"`
}

type AutoprogrammingStagingEffectResultV0 struct {
	SchemaVersion  string                          `json:"schema_version"`
	Status         string                          `json:"status"`
	PromotionRef   string                          `json:"promotion_ref,omitempty"`
	ArchiveRef     string                          `json:"archive_ref,omitempty"`
	RunRef         string                          `json:"run_ref,omitempty"`
	ProjectRef     string                          `json:"project_ref,omitempty"`
	WorktreeRef    string                          `json:"worktree_ref,omitempty"`
	BranchRef      string                          `json:"branch_ref,omitempty"`
	ChangedPaths   []string                        `json:"changed_paths,omitempty"`
	CommitRef      string                          `json:"commit_ref,omitempty"`
	CommitShortRef string                          `json:"commit_short_ref,omitempty"`
	Retryable      bool                            `json:"retryable,omitempty"`
	EvidenceRefs   []string                        `json:"evidence_refs,omitempty"`
	Issues         []AutoprogrammingRequestIssueV0 `json:"issues,omitempty"`
}

type AutoprogrammingStagingPromotionPortV0 interface {
	PromoteAutoprogrammingStagingV0(
		context.Context,
		AutoprogrammingStagingPromotionCommandV0,
	) (AutoprogrammingStagingEffectResultV0, error)
	ArchiveAutoprogrammingStagingV0(
		context.Context,
		AutoprogrammingStagingCleanupCommandV0,
	) (AutoprogrammingStagingEffectResultV0, error)
}

func EvaluateAutoprogrammingStagingPromotionV0(
	request AutoprogrammingStagingPromotionRequestV0,
) AutoprogrammingStagingPromotionDecisionV0 {
	request = normalizeAutoprogrammingStagingPromotionRequestV0(request)
	if issues := autoprogrammingStagingPromotionStaticIssuesV0(request); len(issues) > 0 {
		return autoprogrammingStagingPromotionBlockedV0(request, issues)
	}
	if !request.RunClosed {
		return autoprogrammingStagingPromotionPendingV0(request, nil, "wait_for_causal_closure")
	}
	if issues := autoprogrammingStagingPromotionClosureIssuesV0(request); len(issues) > 0 {
		return autoprogrammingStagingPromotionBlockedV0(request, issues)
	}
	if pending := autoprogrammingStagingPromotionOverlapRefsV0(request); len(pending) > 0 {
		return autoprogrammingStagingPromotionPendingV0(request, pending, "wait_for_overlapping_live_work")
	}
	return autoprogrammingStagingPromotionReadyV0(request)
}

func normalizeAutoprogrammingStagingPromotionRequestV0(
	request AutoprogrammingStagingPromotionRequestV0,
) AutoprogrammingStagingPromotionRequestV0 {
	request.RequestRef = strings.TrimSpace(request.RequestRef)
	request.RunRef = strings.TrimSpace(request.RunRef)
	request.ProjectRef = strings.TrimSpace(request.ProjectRef)
	request.WorktreeRef = strings.TrimSpace(request.WorktreeRef)
	request.BranchRef = strings.TrimSpace(request.BranchRef)
	request.WriteSet = compactStringsV0(request.WriteSet)
	request.RequiredTests = compactStringsV0(request.RequiredTests)
	request.ClosedTaskRefs = compactStringsV0(request.ClosedTaskRefs)
	request.AcceptedReviewRefs = compactStringsV0(request.AcceptedReviewRefs)
	request.EvidenceRefs = compactStringsV0(request.EvidenceRefs)
	return request
}

func autoprogrammingStagingPromotionStaticIssuesV0(
	request AutoprogrammingStagingPromotionRequestV0,
) []AutoprogrammingRequestIssueV0 {
	var issues []AutoprogrammingRequestIssueV0
	for _, item := range []struct{ field, value string }{
		{"run_ref", request.RunRef},
		{"project_ref", request.ProjectRef},
		{"worktree_ref", request.WorktreeRef},
		{"branch_ref", request.BranchRef},
	} {
		if item.value == "" || strings.ContainsAny(item.value, `/\`) {
			issues = append(issues, autoprogrammingRequestIssueV0(item.field+"_invalid", item.field, item.field+" opaco requerido"))
		}
	}
	for _, path := range request.WriteSet {
		if !autoprogrammingRequestWriteSetPathAllowedV0(path) {
			issues = append(issues, autoprogrammingRequestIssueV0("write_set_path_invalid", "write_set", "ruta no permitida: "+path))
		}
	}
	if len(request.WriteSet) == 0 {
		issues = append(issues, autoprogrammingRequestIssueV0("write_set_missing", "write_set", "write-set requerido"))
	}
	if len(request.RequiredTests) == 0 {
		issues = append(issues, autoprogrammingRequestIssueV0("required_tests_missing", "required_tests", "tests requeridos"))
	}
	issues = append(issues, autoprogrammingRequestLiveWorkIssuesV0(request.LiveWorks)...)
	return issues
}

func autoprogrammingStagingPromotionClosureIssuesV0(
	request AutoprogrammingStagingPromotionRequestV0,
) []AutoprogrammingRequestIssueV0 {
	var issues []AutoprogrammingRequestIssueV0
	if len(request.ClosedTaskRefs) == 0 {
		issues = append(issues, autoprogrammingRequestIssueV0("closed_tasks_missing", "closed_task_refs", "cierre causal sin tareas cerradas"))
	}
	if len(request.AcceptedReviewRefs) == 0 {
		issues = append(issues, autoprogrammingRequestIssueV0("accepted_review_missing", "accepted_review_refs", "review aceptada requerida"))
	}
	for _, test := range request.RequiredTests {
		if !autoprogrammingStagingPromotionTestPassedV0(test, request.RequiredTestEvidence) {
			issues = append(issues, autoprogrammingRequestIssueV0("required_test_not_passed", "required_tests", "test requerido sin evidencia passed: "+test))
		}
	}
	return issues
}

func autoprogrammingStagingPromotionTestPassedV0(
	test string,
	evidence []AutoprogrammingRequiredTestEvidenceV0,
) bool {
	for _, item := range evidence {
		if strings.TrimSpace(item.TestCommand) == test && strings.TrimSpace(item.EvidenceRef) != "" &&
			strings.EqualFold(strings.TrimSpace(item.Status), "passed") {
			return true
		}
	}
	return false
}

func autoprogrammingStagingPromotionOverlapRefsV0(
	request AutoprogrammingStagingPromotionRequestV0,
) []string {
	var refs []string
	for _, live := range request.LiveWorks {
		if autoprogrammingLiveWorkIsActiveV0(live) && autoprogrammingWriteSetsOverlapV0(request.WriteSet, live.WriteSet) {
			refs = appendUniqueStringV0(refs, autoprogrammingLiveWorkDependencyRefV0(live))
		}
	}
	return compactStringsV0(refs)
}

func autoprogrammingStagingPromotionReadyV0(
	request AutoprogrammingStagingPromotionRequestV0,
) AutoprogrammingStagingPromotionDecisionV0 {
	promotionRef := "promotion-ref-" + autoprogrammingSelfImprovementHashV0(request.RunRef+"|"+request.WorktreeRef)
	archiveRef := "archive-ref-" + autoprogrammingSelfImprovementHashV0(promotionRef+"|"+request.BranchRef)
	evidence := autoprogrammingStagingPromotionEvidenceRefsV0(request)
	return AutoprogrammingStagingPromotionDecisionV0{
		Status:       AutoprogrammingStagingPromotionStatusReadyV0,
		Ready:        true,
		PromotionRef: promotionRef,
		ArchiveRef:   archiveRef,
		PromotionCommand: AutoprogrammingStagingPromotionCommandV0{
			PromotionRef: promotionRef, RequestRef: request.RequestRef, RunRef: request.RunRef,
			ProjectRef: request.ProjectRef, WorktreeRef: request.WorktreeRef, BranchRef: request.BranchRef,
			WriteSet: request.WriteSet, EvidenceRefs: evidence,
		},
		CleanupCommand: AutoprogrammingStagingCleanupCommandV0{
			ArchiveRef: archiveRef, PromotionRef: promotionRef, RequestRef: request.RequestRef, RunRef: request.RunRef,
			ProjectRef: request.ProjectRef, WorktreeRef: request.WorktreeRef, BranchRef: request.BranchRef,
			WriteSet: request.WriteSet, EvidenceRefs: evidence,
		},
		NextActions:  []string{"promote_staging", "archive_staging_without_delete"},
		EvidenceRefs: evidence,
	}
}

func autoprogrammingStagingPromotionPendingV0(
	request AutoprogrammingStagingPromotionRequestV0,
	pending []string,
	action string,
) AutoprogrammingStagingPromotionDecisionV0 {
	return AutoprogrammingStagingPromotionDecisionV0{
		Status:       AutoprogrammingStagingPromotionStatusPendingV0,
		PendingRefs:  compactStringsV0(pending),
		NextActions:  compactStringsV0([]string{action, "keep_promotion_pending"}),
		EvidenceRefs: autoprogrammingStagingPromotionEvidenceRefsV0(request),
	}
}

func autoprogrammingStagingPromotionBlockedV0(
	request AutoprogrammingStagingPromotionRequestV0,
	issues []AutoprogrammingRequestIssueV0,
) AutoprogrammingStagingPromotionDecisionV0 {
	return AutoprogrammingStagingPromotionDecisionV0{
		Status:       AutoprogrammingStagingPromotionStatusBlockedV0,
		NextActions:  []string{"preserve_staging_evidence", "ask_director_before_promotion"},
		EvidenceRefs: autoprogrammingStagingPromotionEvidenceRefsV0(request),
		Issues:       issues,
	}
}

func autoprogrammingStagingPromotionEvidenceRefsV0(
	request AutoprogrammingStagingPromotionRequestV0,
) []string {
	refs := append([]string{"evidence-ref-autoprogramming-staging-promotion-v0"}, request.EvidenceRefs...)
	refs = append(refs, request.AcceptedReviewRefs...)
	for _, item := range request.RequiredTestEvidence {
		refs = append(refs, item.EvidenceRef)
	}
	return compactStringsV0(refs)
}
