package orquestaserver

import (
	"strings"

	orquestagoal "orquesta/modulos/orquesta-goal"
)

const ServerPublicIdleSelfImprovementGoalSchemaV0 = "orquesta_server_idle_self_improvement_goal_public.v0"

type ServerPublicIdleSelfImprovementGoalStateV0 struct {
	SchemaVersion               string   `json:"schema_version"`
	Active                      bool     `json:"active"`
	GoalRef                     string   `json:"goal_ref,omitempty"`
	ExternalGoalRef             string   `json:"external_goal_ref,omitempty"`
	GoalRefs                    []string `json:"goal_refs,omitempty"`
	RequestRef                  string   `json:"request_ref,omitempty"`
	RunRef                      string   `json:"run_ref,omitempty"`
	ProjectRef                  string   `json:"project_ref,omitempty"`
	DomainRef                   string   `json:"domain_ref,omitempty"`
	WorkKind                    string   `json:"work_kind,omitempty"`
	WorkProfileKind             string   `json:"work_profile_kind,omitempty"`
	DirectorKind                string   `json:"director_kind,omitempty"`
	OperationalStatus           string   `json:"operational_status,omitempty"`
	OperationalReasonCode       string   `json:"operational_reason_code,omitempty"`
	SpecPresent                 bool     `json:"spec_present,omitempty"`
	SpecContextRefCount         int      `json:"spec_context_ref_count,omitempty"`
	SpecRuleRefCount            int      `json:"spec_rule_ref_count,omitempty"`
	SpecSkillRefCount           int      `json:"spec_skill_ref_count,omitempty"`
	SpecWriteSetCount           int      `json:"spec_write_set_count,omitempty"`
	SpecRequiredTestCount       int      `json:"spec_required_test_count,omitempty"`
	SpecArtifactContractCount   int      `json:"spec_artifact_contract_count,omitempty"`
	SpecEvidenceRefCount        int      `json:"spec_evidence_ref_count,omitempty"`
	ReceiptPresent              bool     `json:"receipt_present,omitempty"`
	ReceiptStatus               string   `json:"receipt_status,omitempty"`
	ReceiptEvidenceRefCount     int      `json:"receipt_evidence_ref_count,omitempty"`
	ReceiptIssueCount           int      `json:"receipt_issue_count,omitempty"`
	ResultPresent               bool     `json:"result_present,omitempty"`
	ResultStatus                string   `json:"result_status,omitempty"`
	ResultArtifactRefCount      int      `json:"result_artifact_ref_count,omitempty"`
	ResultRequiredTestCount     int      `json:"result_required_test_count,omitempty"`
	ResultRequiredTestPassed    int      `json:"result_required_test_passed,omitempty"`
	ResultDomainReceiptRefCount int      `json:"result_domain_receipt_ref_count,omitempty"`
	ResultEvidenceRefCount      int      `json:"result_evidence_ref_count,omitempty"`
	ResultIssueCount            int      `json:"result_issue_count,omitempty"`
	ClosurePresent              bool     `json:"closure_present,omitempty"`
	ClosureStatus               string   `json:"closure_status,omitempty"`
	ClosureAccepted             bool     `json:"closure_accepted,omitempty"`
	ClosureNeedsRework          bool     `json:"closure_needs_rework,omitempty"`
	ClosureEvidenceRefCount     int      `json:"closure_evidence_ref_count,omitempty"`
	ClosureIssueCount           int      `json:"closure_issue_count,omitempty"`
}

func NewServerPublicIdleSelfImprovementGoalStateV0(state StateV0) *ServerPublicIdleSelfImprovementGoalStateV0 {
	projection := ServerPublicIdleSelfImprovementGoalStateV0{
		SchemaVersion: ServerPublicIdleSelfImprovementGoalSchemaV0,
	}
	goalRefs := []string{}
	if message := state.IdleSelfImprovementOperationalMessage; message != nil {
		projection.OperationalStatus = strings.TrimSpace(message.Status)
		projection.OperationalReasonCode = strings.TrimSpace(message.ReasonCode)
		goalRefs = append(goalRefs, message.GoalRefs...)
	}
	if state.IdleSelfImprovementGoalSpec != nil {
		spec := orquestagoal.NormalizeGoalWorkSpecV0(*state.IdleSelfImprovementGoalSpec)
		projection.SpecPresent = true
		projection.GoalRef = firstNonEmptyConfigStringV0(projection.GoalRef, spec.GoalRef)
		projection.RequestRef = strings.TrimSpace(spec.RequestRef)
		projection.RunRef = strings.TrimSpace(spec.RunRef)
		projection.ProjectRef = strings.TrimSpace(spec.ProjectRef)
		projection.DomainRef = strings.TrimSpace(spec.DomainRef)
		projection.WorkKind = strings.TrimSpace(spec.WorkKind)
		projection.WorkProfileKind = strings.TrimSpace(spec.WorkProfileKind)
		projection.DirectorKind = strings.TrimSpace(spec.DirectorKind)
		projection.SpecContextRefCount = len(spec.ContextRefs)
		projection.SpecRuleRefCount = len(spec.RuleRefs)
		projection.SpecSkillRefCount = len(compactConfigStringsV0(spec.SkillRefs))
		projection.SpecWriteSetCount = len(spec.WriteSet)
		projection.SpecRequiredTestCount = len(spec.RequiredTests)
		projection.SpecArtifactContractCount = len(spec.ArtifactContracts)
		projection.SpecEvidenceRefCount = len(compactConfigStringsV0(spec.EvidenceRefs))
		goalRefs = append(goalRefs, spec.GoalRef)
	}
	if state.IdleSelfImprovementGoalReceipt != nil {
		receipt := copyGoalLaunchReceiptForServerStateV0(*state.IdleSelfImprovementGoalReceipt)
		projection.ReceiptPresent = true
		projection.GoalRef = firstNonEmptyConfigStringV0(projection.GoalRef, receipt.GoalRef)
		projection.ExternalGoalRef = firstNonEmptyConfigStringV0(projection.ExternalGoalRef, receipt.ExternalGoalRef)
		projection.ReceiptStatus = strings.TrimSpace(receipt.Status)
		projection.ReceiptEvidenceRefCount = len(compactConfigStringsV0(receipt.EvidenceRefs))
		projection.ReceiptIssueCount = len(receipt.Issues)
		goalRefs = append(goalRefs, receipt.GoalRef, receipt.ExternalGoalRef)
	}
	if state.IdleSelfImprovementGoalResult != nil {
		result := orquestagoal.NormalizeGoalWorkResultV0(*state.IdleSelfImprovementGoalResult)
		projection.ResultPresent = true
		projection.GoalRef = firstNonEmptyConfigStringV0(projection.GoalRef, result.GoalRef)
		projection.ExternalGoalRef = firstNonEmptyConfigStringV0(projection.ExternalGoalRef, result.ExternalGoalRef)
		projection.ResultStatus = strings.TrimSpace(result.Status)
		projection.ResultArtifactRefCount = len(compactConfigStringsV0(result.ArtifactRefs))
		projection.ResultRequiredTestCount = len(result.RequiredTestResults)
		projection.ResultRequiredTestPassed = countAcceptedGoalTestResultsForPublicStatusV0(result.RequiredTestResults)
		projection.ResultDomainReceiptRefCount = len(compactConfigStringsV0(result.DomainReceiptRefs))
		projection.ResultEvidenceRefCount = len(compactConfigStringsV0(result.EvidenceRefs))
		projection.ResultIssueCount = len(result.Issues)
		goalRefs = append(goalRefs, result.GoalRef, result.ExternalGoalRef)
	}
	if state.IdleSelfImprovementGoalClosure != nil {
		closure := *state.IdleSelfImprovementGoalClosure
		projection.ClosurePresent = true
		projection.ClosureStatus = strings.TrimSpace(closure.Status)
		projection.ClosureAccepted = closure.Accepted
		projection.ClosureNeedsRework = closure.NeedsRework
		projection.ClosureEvidenceRefCount = len(compactConfigStringsV0(closure.EvidenceRefs))
		projection.ClosureIssueCount = len(closure.Issues)
	}
	projection.GoalRefs = compactConfigStringsV0(goalRefs)
	projection.GoalRef = firstNonEmptyConfigStringV0(projection.GoalRef, firstPublicGoalRefV0(projection.GoalRefs))
	projection.ExternalGoalRef = firstNonEmptyConfigStringV0(projection.ExternalGoalRef, firstPublicExternalGoalRefV0(projection.GoalRefs, projection.GoalRef))
	if !serverPublicGoalHasIdentityOrDurableStateV0(projection) {
		return nil
	}
	projection.Active = serverPublicGoalIsOperationallyActiveV0(projection)
	return &projection
}

func serverPublicGoalHasIdentityOrDurableStateV0(projection ServerPublicIdleSelfImprovementGoalStateV0) bool {
	return strings.TrimSpace(projection.GoalRef) != "" ||
		strings.TrimSpace(projection.ExternalGoalRef) != "" ||
		len(projection.GoalRefs) > 0 ||
		projection.SpecPresent ||
		projection.ReceiptPresent ||
		projection.ResultPresent ||
		projection.ClosurePresent
}

func serverPublicGoalIsOperationallyActiveV0(projection ServerPublicIdleSelfImprovementGoalStateV0) bool {
	reason := strings.TrimSpace(projection.OperationalReasonCode)
	switch reason {
	case "attempt_blocked",
		idleSelfImprovementGoalObserverUnavailableReasonV0,
		idleSelfImprovementGoalObservationErrorReasonV0,
		idleSelfImprovementGoalBlockedReasonV0,
		idleSelfImprovementGoalBackendGoneWithoutResultV0,
		idleSelfImprovementGoalHighConsumptionNoProgressReasonV0,
		idleSelfImprovementGoalInvalidReasonV0,
		idleSelfImprovementGoalCompletePendingClosureV0,
		idleSelfImprovementGoalClosureAcceptedReasonV0:
		return false
	case idleSelfImprovementGoalRunningReasonV0:
		return true
	}
	if projection.ClosureAccepted || projection.ClosureNeedsRework {
		return false
	}
	for _, status := range []string{
		projection.ResultStatus,
		projection.OperationalStatus,
		projection.ReceiptStatus,
	} {
		switch strings.TrimSpace(status) {
		case orquestagoal.GoalStatusRunningV0,
			idleSelfImprovementPreparePendingStatusV0,
			"prepared":
			return true
		case orquestagoal.GoalStatusCompleteV0,
			orquestagoal.GoalStatusAcceptedV0,
			orquestagoal.GoalStatusBlockedV0,
			orquestagoal.GoalStatusInvalidV0,
			"checked",
			"error":
			return false
		}
	}
	return false
}

func countAcceptedGoalTestResultsForPublicStatusV0(results []orquestagoal.GoalRequiredTestResultV0) int {
	count := 0
	for _, result := range results {
		switch strings.TrimSpace(result.Status) {
		case orquestagoal.GoalStatusAcceptedV0, "passed":
			count++
		}
	}
	return count
}

func firstPublicGoalRefV0(refs []string) string {
	for _, ref := range refs {
		ref = strings.TrimSpace(ref)
		if ref != "" && !strings.Contains(ref, "external") {
			return ref
		}
	}
	if len(refs) > 0 {
		return strings.TrimSpace(refs[0])
	}
	return ""
}

func firstPublicExternalGoalRefV0(refs []string, goalRef string) string {
	for _, ref := range refs {
		ref = strings.TrimSpace(ref)
		if ref != "" && ref != goalRef && strings.Contains(ref, "external") {
			return ref
		}
	}
	return ""
}
