package orquestagoal

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"unicode"
)

func NormalizeGoalWorkSpecV0(spec GoalWorkSpecV0) GoalWorkSpecV0 {
	spec = cloneGoalWorkSpecSlicesV0(spec)
	spec.SchemaVersion = GoalWorkSpecSchemaV0
	spec.GoalRef = strings.TrimSpace(spec.GoalRef)
	spec.RequestRef = strings.TrimSpace(spec.RequestRef)
	spec.RunRef = strings.TrimSpace(spec.RunRef)
	spec.ProjectRef = strings.TrimSpace(spec.ProjectRef)
	spec.DomainRef = strings.TrimSpace(spec.DomainRef)
	spec.WorkKind = strings.TrimSpace(spec.WorkKind)
	spec.WorkProfileKind = strings.TrimSpace(spec.WorkProfileKind)
	spec.Objective = strings.TrimSpace(spec.Objective)
	spec.DirectorKind = strings.TrimSpace(spec.DirectorKind)
	if spec.DirectorKind == "" {
		spec.DirectorKind = GoalDirectorKindRuntimeGoalV0
	}
	for i := range spec.ContextRefs {
		spec.ContextRefs[i].Kind = strings.TrimSpace(spec.ContextRefs[i].Kind)
		spec.ContextRefs[i].Ref = strings.TrimSpace(spec.ContextRefs[i].Ref)
		spec.ContextRefs[i].Purpose = strings.TrimSpace(spec.ContextRefs[i].Purpose)
	}
	for i := range spec.RuleRefs {
		spec.RuleRefs[i].Kind = strings.TrimSpace(spec.RuleRefs[i].Kind)
		spec.RuleRefs[i].Ref = strings.TrimSpace(spec.RuleRefs[i].Ref)
		spec.RuleRefs[i].Enforcement = strings.TrimSpace(spec.RuleRefs[i].Enforcement)
		if spec.RuleRefs[i].Enforcement == "" {
			spec.RuleRefs[i].Enforcement = GoalRuleEnforcementAdvisoryV0
		}
	}
	for i := range spec.SkillRefs {
		spec.SkillRefs[i] = strings.TrimSpace(spec.SkillRefs[i])
	}
	for i := range spec.WriteSet {
		spec.WriteSet[i].Path = filepath.ToSlash(strings.TrimSpace(spec.WriteSet[i].Path))
		spec.WriteSet[i].Purpose = strings.TrimSpace(spec.WriteSet[i].Purpose)
	}
	for i := range spec.RequiredTests {
		spec.RequiredTests[i].TestRef = strings.TrimSpace(spec.RequiredTests[i].TestRef)
		spec.RequiredTests[i].CommandRef = strings.TrimSpace(spec.RequiredTests[i].CommandRef)
		spec.RequiredTests[i].Command = strings.TrimSpace(spec.RequiredTests[i].Command)
	}
	for i := range spec.ArtifactContracts {
		spec.ArtifactContracts[i].ArtifactRef = strings.TrimSpace(spec.ArtifactContracts[i].ArtifactRef)
		spec.ArtifactContracts[i].ArtifactType = strings.TrimSpace(spec.ArtifactContracts[i].ArtifactType)
	}
	for i := range spec.EvidenceRefs {
		spec.EvidenceRefs[i] = strings.TrimSpace(spec.EvidenceRefs[i])
	}
	for i := range spec.ClosurePolicy.RequiredEvidenceRefs {
		spec.ClosurePolicy.RequiredEvidenceRefs[i] = strings.TrimSpace(spec.ClosurePolicy.RequiredEvidenceRefs[i])
	}
	return spec
}

func cloneGoalWorkSpecSlicesV0(spec GoalWorkSpecV0) GoalWorkSpecV0 {
	spec.ContextRefs = append([]GoalContextRefV0(nil), spec.ContextRefs...)
	spec.RuleRefs = append([]GoalRuleRefV0(nil), spec.RuleRefs...)
	spec.SkillRefs = append([]string(nil), spec.SkillRefs...)
	spec.WriteSet = append([]GoalWriteScopeV0(nil), spec.WriteSet...)
	spec.RequiredTests = append([]GoalRequiredTestV0(nil), spec.RequiredTests...)
	for i := range spec.RequiredTests {
		spec.RequiredTests[i].AcceptanceCriteria = append([]string(nil), spec.RequiredTests[i].AcceptanceCriteria...)
		spec.RequiredTests[i].AcceptanceCriteriaRefs = append([]string(nil), spec.RequiredTests[i].AcceptanceCriteriaRefs...)
		spec.RequiredTests[i].EvidenceRefs = append([]string(nil), spec.RequiredTests[i].EvidenceRefs...)
	}
	spec.AcceptanceCriteria = append([]string(nil), spec.AcceptanceCriteria...)
	spec.ArtifactContracts = append([]GoalArtifactContractV0(nil), spec.ArtifactContracts...)
	for i := range spec.ArtifactContracts {
		spec.ArtifactContracts[i].EvidenceRefs = append([]string(nil), spec.ArtifactContracts[i].EvidenceRefs...)
	}
	spec.EvidenceRefs = append([]string(nil), spec.EvidenceRefs...)
	spec.ClosurePolicy.RequiredEvidenceRefs = append([]string(nil), spec.ClosurePolicy.RequiredEvidenceRefs...)
	return spec
}

func ValidateGoalWorkSpecV0(spec GoalWorkSpecV0) []GoalWorkIssueV0 {
	spec = NormalizeGoalWorkSpecV0(spec)
	var issues []GoalWorkIssueV0
	validateGoalWorkSpecLimitsV0(&issues, spec)
	if spec.GoalRef == "" {
		issues = append(issues, GoalWorkIssueV0{Code: ErrGoalRefRequiredV0, Field: "goal_ref"})
	} else if !validGoalRefTokenV0(spec.GoalRef) {
		issues = append(issues, GoalWorkIssueV0{Code: ErrGoalRefInvalidV0, Field: "goal_ref"})
	}
	if spec.Objective == "" {
		issues = append(issues, GoalWorkIssueV0{Code: ErrGoalObjectiveRequiredV0, Field: "objective"})
	}
	if spec.DirectorKind == "" {
		issues = append(issues, GoalWorkIssueV0{Code: ErrGoalDirectorRequiredV0, Field: "director_kind"})
	} else if spec.DirectorKind != GoalDirectorKindRuntimeGoalV0 && spec.DirectorKind != GoalDirectorKindCodexGoalV0 {
		issues = append(issues, GoalWorkIssueV0{Code: ErrGoalDirectorInvalidV0, Field: "director_kind"})
	}
	if len(spec.WriteSet) == 0 {
		issues = append(issues, GoalWorkIssueV0{Code: ErrGoalWriteSetRequiredV0, Field: "write_set"})
	}
	for _, scope := range spec.WriteSet {
		if !validGoalWriteScopePathV0(scope.Path) {
			issues = append(issues, GoalWorkIssueV0{Code: ErrGoalWriteSetInvalidV0, Field: "write_set.path"})
		}
	}
	validateGoalRefsV0(&issues, "request_ref", spec.RequestRef)
	validateGoalRefsV0(&issues, "run_ref", spec.RunRef)
	validateGoalRefsV0(&issues, "project_ref", spec.ProjectRef)
	validateGoalRefsV0(&issues, "domain_ref", spec.DomainRef)
	for _, ctx := range spec.ContextRefs {
		validateRequiredGoalRefV0(&issues, "context_refs.ref", ctx.Ref)
	}
	for _, rule := range spec.RuleRefs {
		validateRequiredGoalRefV0(&issues, "rule_refs.ref", rule.Ref)
		if rule.Enforcement != GoalRuleEnforcementAdvisoryV0 && rule.Enforcement != GoalRuleEnforcementHardV0 {
			issues = append(issues, GoalWorkIssueV0{Code: ErrGoalRuleInvalidV0, Field: "rule_refs.enforcement"})
		}
	}
	for _, skillRef := range spec.SkillRefs {
		validateRequiredGoalRefV0(&issues, "skill_refs", skillRef)
	}
	for _, test := range spec.RequiredTests {
		validateRequiredGoalRefV0(&issues, "required_tests.test_ref", test.TestRef)
		validateGoalRefsV0(&issues, "required_tests.command_ref", test.CommandRef)
	}
	for _, artifact := range spec.ArtifactContracts {
		validateRequiredGoalRefV0(&issues, "artifact_contracts.artifact_ref", artifact.ArtifactRef)
		if artifact.ArtifactType == "" {
			issues = append(issues, GoalWorkIssueV0{Code: ErrGoalRefFieldInvalidV0, Field: "artifact_contracts.artifact_type"})
		}
	}
	for _, evidenceRef := range spec.EvidenceRefs {
		validateRequiredGoalRefV0(&issues, "evidence_refs", evidenceRef)
	}
	for _, evidenceRef := range spec.ClosurePolicy.RequiredEvidenceRefs {
		validateRequiredGoalRefV0(&issues, "closure_policy.required_evidence_refs", evidenceRef)
	}
	return issues
}

func validateGoalWorkSpecLimitsV0(issues *[]GoalWorkIssueV0, spec GoalWorkSpecV0) {
	if encoded, err := json.Marshal(spec); err == nil && len(encoded) > GoalWorkSpecMaxProjectedJSONBytesV0 {
		*issues = append(*issues, GoalWorkIssueV0{Code: ErrGoalSpecLimitExceededV0, Field: "goal_work_spec"})
	}

	validateGoalStringLimitV0(issues, "goal_ref", spec.GoalRef, GoalWorkSpecMaxStringBytesV0)
	validateGoalStringLimitV0(issues, "request_ref", spec.RequestRef, GoalWorkSpecMaxStringBytesV0)
	validateGoalStringLimitV0(issues, "run_ref", spec.RunRef, GoalWorkSpecMaxStringBytesV0)
	validateGoalStringLimitV0(issues, "project_ref", spec.ProjectRef, GoalWorkSpecMaxStringBytesV0)
	validateGoalStringLimitV0(issues, "domain_ref", spec.DomainRef, GoalWorkSpecMaxStringBytesV0)
	validateGoalStringLimitV0(issues, "work_kind", spec.WorkKind, GoalWorkSpecMaxStringBytesV0)
	validateGoalStringLimitV0(issues, "work_profile_kind", spec.WorkProfileKind, GoalWorkSpecMaxStringBytesV0)
	validateGoalStringLimitV0(issues, "objective", spec.Objective, GoalWorkSpecMaxStringBytesV0)
	validateGoalStringLimitV0(issues, "director_kind", spec.DirectorKind, GoalWorkSpecMaxStringBytesV0)

	validateGoalListLimitV0(issues, "context_refs", len(spec.ContextRefs))
	for _, ctx := range spec.ContextRefs {
		validateGoalStringLimitV0(issues, "context_refs.kind", ctx.Kind, GoalWorkSpecMaxStringBytesV0)
		validateGoalStringLimitV0(issues, "context_refs.ref", ctx.Ref, GoalWorkSpecMaxStringBytesV0)
		validateGoalStringLimitV0(issues, "context_refs.purpose", ctx.Purpose, GoalWorkSpecMaxStringBytesV0)
	}

	validateGoalListLimitV0(issues, "rule_refs", len(spec.RuleRefs))
	for _, rule := range spec.RuleRefs {
		validateGoalStringLimitV0(issues, "rule_refs.kind", rule.Kind, GoalWorkSpecMaxStringBytesV0)
		validateGoalStringLimitV0(issues, "rule_refs.ref", rule.Ref, GoalWorkSpecMaxStringBytesV0)
		validateGoalStringLimitV0(issues, "rule_refs.enforcement", rule.Enforcement, GoalWorkSpecMaxStringBytesV0)
	}

	validateGoalListLimitV0(issues, "skill_refs", len(spec.SkillRefs))
	for _, skillRef := range spec.SkillRefs {
		validateGoalStringLimitV0(issues, "skill_refs", skillRef, GoalWorkSpecMaxStringBytesV0)
	}

	validateGoalListLimitV0(issues, "write_set", len(spec.WriteSet))
	for _, scope := range spec.WriteSet {
		validateGoalStringLimitV0(issues, "write_set.path", scope.Path, GoalWorkSpecMaxStringBytesV0)
		validateGoalStringLimitV0(issues, "write_set.purpose", scope.Purpose, GoalWorkSpecMaxStringBytesV0)
	}

	validateGoalListLimitV0(issues, "required_tests", len(spec.RequiredTests))
	for _, test := range spec.RequiredTests {
		validateGoalStringLimitV0(issues, "required_tests.test_ref", test.TestRef, GoalWorkSpecMaxStringBytesV0)
		validateGoalStringLimitV0(issues, "required_tests.command_ref", test.CommandRef, GoalWorkSpecMaxStringBytesV0)
		validateGoalStringLimitV0(issues, "required_tests.command", test.Command, GoalWorkSpecMaxCommandBytesV0)
		validateGoalStringListLimitV0(issues, "required_tests.acceptance_criteria", test.AcceptanceCriteria)
		validateGoalStringListLimitV0(issues, "required_tests.acceptance_criteria_refs", test.AcceptanceCriteriaRefs)
		validateGoalStringListLimitV0(issues, "required_tests.evidence_refs", test.EvidenceRefs)
	}

	validateGoalStringListLimitV0(issues, "acceptance_criteria", spec.AcceptanceCriteria)

	validateGoalListLimitV0(issues, "artifact_contracts", len(spec.ArtifactContracts))
	for _, artifact := range spec.ArtifactContracts {
		validateGoalStringLimitV0(issues, "artifact_contracts.artifact_ref", artifact.ArtifactRef, GoalWorkSpecMaxStringBytesV0)
		validateGoalStringLimitV0(issues, "artifact_contracts.artifact_type", artifact.ArtifactType, GoalWorkSpecMaxStringBytesV0)
		validateGoalStringListLimitV0(issues, "artifact_contracts.evidence_refs", artifact.EvidenceRefs)
	}

	validateGoalStringListLimitV0(issues, "evidence_refs", spec.EvidenceRefs)
	validateGoalStringListLimitV0(issues, "closure_policy.required_evidence_refs", spec.ClosurePolicy.RequiredEvidenceRefs)
}

func validateGoalStringListLimitV0(issues *[]GoalWorkIssueV0, field string, values []string) {
	validateGoalListLimitV0(issues, field, len(values))
	for _, value := range values {
		validateGoalStringLimitV0(issues, field, value, GoalWorkSpecMaxStringBytesV0)
	}
}

func validateGoalListLimitV0(issues *[]GoalWorkIssueV0, field string, length int) {
	if length > GoalWorkSpecMaxListItemsV0 {
		*issues = append(*issues, GoalWorkIssueV0{Code: ErrGoalSpecLimitExceededV0, Field: field})
	}
}

func validateGoalStringLimitV0(issues *[]GoalWorkIssueV0, field, value string, maxBytes int) {
	if len(value) > maxBytes {
		*issues = append(*issues, GoalWorkIssueV0{Code: ErrGoalSpecLimitExceededV0, Field: field})
	}
}

func ValidateGoalWorkClosureV0(spec GoalWorkSpecV0, result GoalWorkResultV0) GoalClosureValidationV0 {
	spec = NormalizeGoalWorkSpecV0(spec)
	result = NormalizeGoalWorkResultV0(result)
	if issues := ValidateGoalWorkSpecV0(spec); len(issues) > 0 {
		return GoalClosureValidationV0{
			Status:      GoalStatusBlockedV0,
			NeedsRework: true,
			Issues:      issues,
		}
	}
	if issues := ValidateGoalWorkResultV0(result); len(issues) > 0 {
		return GoalClosureValidationV0{
			Status:      GoalStatusBlockedV0,
			NeedsRework: true,
			Issues:      issues,
		}
	}
	if result.GoalRef == "" || result.GoalRef != spec.GoalRef {
		return GoalClosureValidationV0{
			Status:      GoalStatusBlockedV0,
			NeedsRework: true,
			Issues:      []GoalWorkIssueV0{{Code: ErrGoalClosureInvalidV0, Field: "goal_ref"}},
		}
	}
	if result.Status != GoalStatusCompleteV0 {
		return GoalClosureValidationV0{
			Status:      GoalStatusBlockedV0,
			NeedsRework: result.Status == GoalStatusBlockedV0 || result.Status == GoalStatusInvalidV0,
			Issues:      []GoalWorkIssueV0{{Code: ErrGoalClosureInvalidV0, Field: "status"}},
		}
	}
	missing := missingGoalEvidenceRefsV0(spec.ClosurePolicy.RequiredEvidenceRefs, result.EvidenceRefs)
	if len(missing) > 0 {
		issues := make([]GoalWorkIssueV0, 0, len(missing))
		for range missing {
			issues = append(issues, GoalWorkIssueV0{Code: ErrGoalClosureInvalidV0, Field: "evidence_refs"})
		}
		return GoalClosureValidationV0{Status: GoalStatusBlockedV0, NeedsRework: true, Issues: issues}
	}
	if spec.ClosurePolicy.RequireRequiredTests && !allRequiredGoalTestsPassedV0(spec.RequiredTests, result.RequiredTestResults) {
		return GoalClosureValidationV0{
			Status:      GoalStatusBlockedV0,
			NeedsRework: true,
			Issues:      []GoalWorkIssueV0{{Code: ErrGoalClosureInvalidV0, Field: "required_tests"}},
		}
	}
	if spec.ClosurePolicy.RequireArtifacts && !allRequiredGoalArtifactsPresentV0(spec.ArtifactContracts, result.ArtifactRefs) {
		return GoalClosureValidationV0{
			Status:      GoalStatusBlockedV0,
			NeedsRework: true,
			Issues:      []GoalWorkIssueV0{{Code: ErrGoalClosureInvalidV0, Field: "artifact_refs"}},
		}
	}
	if spec.ClosurePolicy.RequireDomainReceipt && len(result.DomainReceiptRefs) == 0 {
		return GoalClosureValidationV0{
			Status:      GoalStatusBlockedV0,
			NeedsRework: true,
			Issues:      []GoalWorkIssueV0{{Code: ErrGoalClosureInvalidV0, Field: "domain_receipt_refs"}},
		}
	}
	return GoalClosureValidationV0{
		Status:       GoalStatusAcceptedV0,
		Accepted:     true,
		EvidenceRefs: append([]string(nil), result.EvidenceRefs...),
	}
}

type DefaultGoalWorkClosureValidatorV0 struct{}

func (DefaultGoalWorkClosureValidatorV0) ValidateGoalWorkClosureV0(_ context.Context, spec GoalWorkSpecV0, result GoalWorkResultV0) (GoalClosureValidationV0, error) {
	return ValidateGoalWorkClosureV0(spec, result), nil
}

func NormalizeGoalWorkResultV0(result GoalWorkResultV0) GoalWorkResultV0 {
	result.SchemaVersion = GoalWorkResultSchemaV0
	result.Status = strings.TrimSpace(result.Status)
	result.GoalRef = strings.TrimSpace(result.GoalRef)
	result.ExternalGoalRef = strings.TrimSpace(result.ExternalGoalRef)
	result.Summary = strings.TrimSpace(result.Summary)
	for i := range result.ArtifactRefs {
		result.ArtifactRefs[i] = strings.TrimSpace(result.ArtifactRefs[i])
	}
	for i := range result.RequiredTestResults {
		result.RequiredTestResults[i].TestRef = strings.TrimSpace(result.RequiredTestResults[i].TestRef)
		result.RequiredTestResults[i].Status = strings.TrimSpace(result.RequiredTestResults[i].Status)
		for j := range result.RequiredTestResults[i].EvidenceRefs {
			result.RequiredTestResults[i].EvidenceRefs[j] = strings.TrimSpace(result.RequiredTestResults[i].EvidenceRefs[j])
		}
	}
	for i := range result.DomainReceiptRefs {
		result.DomainReceiptRefs[i] = strings.TrimSpace(result.DomainReceiptRefs[i])
	}
	for i := range result.EvidenceRefs {
		result.EvidenceRefs[i] = strings.TrimSpace(result.EvidenceRefs[i])
	}
	return result
}

func NormalizeGoalObservationRequestV0(
	request GoalObservationRequestV0,
) GoalObservationRequestV0 {
	request.GoalRef = strings.TrimSpace(request.GoalRef)
	request.ExternalGoalRef = strings.TrimSpace(request.ExternalGoalRef)
	return request
}

func NormalizeGoalWorkStateV0(state GoalWorkStateV0) GoalWorkStateV0 {
	state.SchemaVersion = GoalWorkStateSchemaV0
	state.RunRef = strings.TrimSpace(state.RunRef)
	state.GoalRef = strings.TrimSpace(state.GoalRef)
	state.ExternalGoalRef = strings.TrimSpace(state.ExternalGoalRef)
	state.Status = strings.TrimSpace(state.Status)
	state.Spec = NormalizeGoalWorkSpecV0(state.Spec)
	state.LaunchReceipt.SchemaVersion = strings.TrimSpace(state.LaunchReceipt.SchemaVersion)
	if state.LaunchReceipt.SchemaVersion == "" {
		state.LaunchReceipt.SchemaVersion = GoalWorkLaunchReceiptSchemaV0
	}
	state.LaunchReceipt.Status = strings.TrimSpace(state.LaunchReceipt.Status)
	state.LaunchReceipt.GoalRef = strings.TrimSpace(state.LaunchReceipt.GoalRef)
	state.LaunchReceipt.ExternalGoalRef = strings.TrimSpace(state.LaunchReceipt.ExternalGoalRef)
	for i := range state.LaunchReceipt.EvidenceRefs {
		state.LaunchReceipt.EvidenceRefs[i] = strings.TrimSpace(state.LaunchReceipt.EvidenceRefs[i])
	}
	if state.LastResult != nil {
		normalized := NormalizeGoalWorkResultV0(*state.LastResult)
		state.LastResult = &normalized
	}
	for i := range state.EvidenceRefs {
		state.EvidenceRefs[i] = strings.TrimSpace(state.EvidenceRefs[i])
	}
	return state
}

func NewGoalWorkStateV0(state GoalWorkStateV0) (GoalWorkStateV0, error) {
	state = NormalizeGoalWorkStateV0(state)
	var issues []GoalWorkIssueV0
	validateRequiredGoalRefV0(&issues, "run_ref", state.RunRef)
	validateRequiredGoalRefV0(&issues, "goal_ref", state.GoalRef)
	validateGoalRefsV0(&issues, "external_goal_ref", state.ExternalGoalRef)
	if !validGoalWorkResultStatusV0(state.Status) {
		issues = append(issues, GoalWorkIssueV0{Code: ErrGoalStatusInvalidV0, Field: "status"})
	}
	if state.Spec.RunRef != "" && state.Spec.RunRef != state.RunRef {
		issues = append(issues, GoalWorkIssueV0{Code: ErrGoalClosureInvalidV0, Field: "spec.run_ref"})
	}
	if state.Spec.GoalRef != "" && state.Spec.GoalRef != state.GoalRef {
		issues = append(issues, GoalWorkIssueV0{Code: ErrGoalClosureInvalidV0, Field: "spec.goal_ref"})
	}
	issues = append(issues, ValidateGoalWorkSpecV0(state.Spec)...)
	if state.LaunchReceipt.GoalRef != "" && state.LaunchReceipt.GoalRef != state.GoalRef {
		issues = append(issues, GoalWorkIssueV0{Code: ErrGoalClosureInvalidV0, Field: "launch_receipt.goal_ref"})
	}
	if state.LastResult != nil {
		if state.LastResult.GoalRef != state.GoalRef {
			issues = append(issues, GoalWorkIssueV0{Code: ErrGoalClosureInvalidV0, Field: "last_result.goal_ref"})
		}
		issues = append(issues, ValidateGoalWorkResultV0(*state.LastResult)...)
	}
	for _, evidenceRef := range state.EvidenceRefs {
		validateRequiredGoalRefV0(&issues, "evidence_refs", evidenceRef)
	}
	if len(issues) > 0 {
		return GoalWorkStateV0{}, GoalWorkStateInvalidErrorV0{Issues: issues}
	}
	return state, nil
}

func NormalizeGoalWorkRunMarkerV0(marker GoalWorkRunMarkerV0) GoalWorkRunMarkerV0 {
	marker.SchemaVersion = GoalWorkRunMarkerSchemaV0
	marker.RunRef = strings.TrimSpace(marker.RunRef)
	marker.GoalRef = strings.TrimSpace(marker.GoalRef)
	marker.ExternalGoalRef = strings.TrimSpace(marker.ExternalGoalRef)
	marker.DirectorKind = strings.TrimSpace(marker.DirectorKind)
	if marker.DirectorKind == "" {
		marker.DirectorKind = GoalDirectorKindRuntimeGoalV0
	}
	marker.Status = strings.TrimSpace(marker.Status)
	for i := range marker.EvidenceRefs {
		marker.EvidenceRefs[i] = strings.TrimSpace(marker.EvidenceRefs[i])
	}
	return marker
}

func NewGoalWorkRunMarkerV0(marker GoalWorkRunMarkerV0) (GoalWorkRunMarkerV0, error) {
	marker = NormalizeGoalWorkRunMarkerV0(marker)
	var issues []GoalWorkIssueV0
	validateRequiredGoalRefV0(&issues, "run_ref", marker.RunRef)
	validateGoalRefsV0(&issues, "goal_ref", marker.GoalRef)
	validateGoalRefsV0(&issues, "external_goal_ref", marker.ExternalGoalRef)
	if marker.DirectorKind != GoalDirectorKindRuntimeGoalV0 && marker.DirectorKind != GoalDirectorKindCodexGoalV0 {
		issues = append(issues, GoalWorkIssueV0{Code: ErrGoalDirectorInvalidV0, Field: "director_kind"})
	}
	if marker.Status != "" && !validGoalWorkResultStatusV0(marker.Status) {
		issues = append(issues, GoalWorkIssueV0{Code: ErrGoalStatusInvalidV0, Field: "status"})
	}
	for _, evidenceRef := range marker.EvidenceRefs {
		validateRequiredGoalRefV0(&issues, "evidence_refs", evidenceRef)
	}
	if len(issues) > 0 {
		return GoalWorkRunMarkerV0{}, GoalWorkStateInvalidErrorV0{Issues: issues}
	}
	return marker, nil
}

type GoalWorkStateInvalidErrorV0 struct {
	Issues []GoalWorkIssueV0
}

func (err GoalWorkStateInvalidErrorV0) Error() string {
	if len(err.Issues) == 0 {
		return "goal_work_state_invalid"
	}
	if err.Issues[0].Field == "" {
		return "goal_work_state_invalid:" + err.Issues[0].Code
	}
	return "goal_work_state_invalid:" + err.Issues[0].Field
}

func ValidateGoalObservationRequestV0(
	request GoalObservationRequestV0,
) []GoalWorkIssueV0 {
	request = NormalizeGoalObservationRequestV0(request)
	var issues []GoalWorkIssueV0
	validateRequiredGoalRefV0(&issues, "goal_ref", request.GoalRef)
	validateGoalRefsV0(&issues, "external_goal_ref", request.ExternalGoalRef)
	return issues
}

func ValidateGoalWorkResultV0(result GoalWorkResultV0) []GoalWorkIssueV0 {
	result = NormalizeGoalWorkResultV0(result)
	var issues []GoalWorkIssueV0
	validateRequiredGoalRefV0(&issues, "goal_ref", result.GoalRef)
	validateGoalRefsV0(&issues, "external_goal_ref", result.ExternalGoalRef)
	if !validGoalWorkResultStatusV0(result.Status) {
		issues = append(issues, GoalWorkIssueV0{Code: ErrGoalStatusInvalidV0, Field: "status"})
	}
	for _, artifactRef := range result.ArtifactRefs {
		validateRequiredGoalRefV0(&issues, "artifact_refs", artifactRef)
	}
	for _, test := range result.RequiredTestResults {
		validateRequiredGoalRefV0(&issues, "required_test_results.test_ref", test.TestRef)
		if strings.TrimSpace(test.Status) == "" {
			issues = append(issues, GoalWorkIssueV0{Code: ErrGoalStatusInvalidV0, Field: "required_test_results.status"})
		}
		for _, evidenceRef := range test.EvidenceRefs {
			validateRequiredGoalRefV0(&issues, "required_test_results.evidence_refs", evidenceRef)
		}
	}
	for _, receiptRef := range result.DomainReceiptRefs {
		validateRequiredGoalRefV0(&issues, "domain_receipt_refs", receiptRef)
	}
	for _, evidenceRef := range result.EvidenceRefs {
		validateRequiredGoalRefV0(&issues, "evidence_refs", evidenceRef)
	}
	return issues
}

func validateRequiredGoalRefV0(issues *[]GoalWorkIssueV0, field, value string) {
	if strings.TrimSpace(value) == "" || !validGoalRefTokenV0(value) {
		*issues = append(*issues, GoalWorkIssueV0{Code: ErrGoalRefFieldInvalidV0, Field: field})
	}
}

func validateGoalRefsV0(issues *[]GoalWorkIssueV0, field, value string) {
	if strings.TrimSpace(value) != "" && !validGoalRefTokenV0(value) {
		*issues = append(*issues, GoalWorkIssueV0{Code: ErrGoalRefFieldInvalidV0, Field: field})
	}
}

func validGoalRefTokenV0(value string) bool {
	if strings.TrimSpace(value) != value || value == "" {
		return false
	}
	if strings.Contains(value, "\n") || strings.Contains(value, "\r") || strings.Contains(value, "\t") {
		return false
	}
	if strings.HasPrefix(value, "/") || strings.Contains(value, `:\`) {
		return false
	}
	for _, r := range value {
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}

func validGoalWriteScopePathV0(path string) bool {
	if strings.TrimSpace(path) != path || path == "" {
		return false
	}
	slashPath := filepath.ToSlash(path)
	if strings.HasPrefix(slashPath, "/") || slashPath == "." {
		return false
	}
	for _, segment := range strings.Split(slashPath, "/") {
		if segment == ".." {
			return false
		}
	}
	if strings.Contains(path, `:\`) {
		return false
	}
	for _, r := range path {
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}

func validGoalWorkResultStatusV0(status string) bool {
	switch strings.TrimSpace(status) {
	case GoalStatusRunningV0,
		GoalStatusCompleteV0,
		GoalStatusBlockedV0,
		GoalStatusInvalidV0:
		return true
	default:
		return false
	}
}

func missingGoalEvidenceRefsV0(required, actual []string) []string {
	if len(required) == 0 {
		return nil
	}
	seen := make(map[string]bool, len(actual))
	for _, ref := range actual {
		seen[ref] = true
	}
	var missing []string
	for _, ref := range required {
		if ref != "" && !seen[ref] {
			missing = append(missing, ref)
		}
	}
	return missing
}

func allRequiredGoalTestsPassedV0(required []GoalRequiredTestV0, results []GoalRequiredTestResultV0) bool {
	if len(required) == 0 {
		return true
	}
	passed := make(map[string]bool, len(results))
	for _, result := range results {
		if result.Status == GoalStatusAcceptedV0 || result.Status == "passed" {
			passed[result.TestRef] = true
		}
	}
	for _, test := range required {
		if !passed[test.TestRef] {
			return false
		}
	}
	return true
}

func allRequiredGoalArtifactsPresentV0(required []GoalArtifactContractV0, actual []string) bool {
	present := make(map[string]bool, len(actual))
	for _, ref := range actual {
		present[ref] = true
	}
	for _, artifact := range required {
		if artifact.Required && !present[artifact.ArtifactRef] {
			return false
		}
	}
	return true
}
