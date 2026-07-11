package orquestagoal

import (
	"context"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

var (
	goalWorkIssueAbsolutePathPatternV0 = regexp.MustCompile(`(^|[\s"'=:(])(/[^ \n\r\t"')]+)`)
	goalWorkIssueWindowsPathPatternV0  = regexp.MustCompile(`(?i)(^|[\s"'=:(])([a-z]:\\[^ \n\r\t"')]+)`)
	goalWorkIssueSecretPatternV0       = regexp.MustCompile(`(?i)(bearer\s+)[a-z0-9._\-+/=]{12,}|(sk-[a-z0-9_\-]{8,})|((?:api[_-]?key|token|oauth[_-]?token|access[_-]?token|authorization)=)[^ \n\r\t"')]+`)
)

func NormalizeGoalWorkSpecV0(spec GoalWorkSpecV0) GoalWorkSpecV0 {
	spec = cloneGoalWorkSpecSlicesV0(spec)
	spec.SchemaVersion = GoalWorkSpecSchemaV0
	spec.GoalRef = strings.TrimSpace(spec.GoalRef)
	spec.RequestRef = strings.TrimSpace(spec.RequestRef)
	spec.RunRef = strings.TrimSpace(spec.RunRef)
	spec.ImplementerAgentRef = strings.TrimSpace(spec.ImplementerAgentRef)
	spec.ImplementerCredentialRef = strings.TrimSpace(spec.ImplementerCredentialRef)
	spec.WriteSetSHA256 = strings.ToLower(strings.TrimSpace(spec.WriteSetSHA256))
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
		spec.RequiredTests[i] = normalizeGoalRequiredTestAttestationTestV0(spec.RequiredTests[i])
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
	spec.ClosurePolicy.RequiredAcceptanceCriteriaRefs = normalizeGoalRequiredAcceptanceCriteriaRefsV0(spec.ClosurePolicy.RequiredAcceptanceCriteriaRefs)
	spec.ClosurePolicy.RequiredAttestorTrustPolicyRef = strings.TrimSpace(spec.ClosurePolicy.RequiredAttestorTrustPolicyRef)
	return spec
}

func normalizeGoalRequiredAcceptanceCriteriaRefsV0(refs []string) []string {
	refs = append([]string(nil), refs...)
	seen := make(map[string]bool, len(refs))
	out := make([]string, 0, len(refs))
	for _, ref := range refs {
		ref = strings.TrimSpace(ref)
		if seen[ref] {
			continue
		}
		seen[ref] = true
		out = append(out, ref)
	}
	return out
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
	spec.ClosurePolicy.RequiredAcceptanceCriteriaRefs = append([]string(nil), spec.ClosurePolicy.RequiredAcceptanceCriteriaRefs...)
	spec.ClosurePolicy.RequiredEvidenceRefs = append([]string(nil), spec.ClosurePolicy.RequiredEvidenceRefs...)
	return spec
}

func NormalizeGoalWorkIssueDetailV0(detail string) string {
	detail = strings.TrimSpace(detail)
	if detail == "" {
		return ""
	}
	var builder strings.Builder
	previousSpace := false
	for _, r := range detail {
		if unicode.IsControl(r) || unicode.IsSpace(r) {
			if !previousSpace {
				builder.WriteByte(' ')
				previousSpace = true
			}
			continue
		}
		builder.WriteRune(r)
		previousSpace = false
	}
	detail = strings.TrimSpace(builder.String())
	detail = goalWorkIssueAbsolutePathPatternV0.ReplaceAllString(detail, `${1}<path>`)
	detail = goalWorkIssueWindowsPathPatternV0.ReplaceAllString(detail, `${1}<path>`)
	detail = goalWorkIssueSecretPatternV0.ReplaceAllStringFunc(detail, func(match string) string {
		lower := strings.ToLower(match)
		switch {
		case strings.HasPrefix(lower, "bearer "):
			return "<redacted>"
		case strings.Contains(match, "="):
			return "<redacted>"
		default:
			return "<redacted>"
		}
	})
	if len([]byte(detail)) <= GoalWorkIssueDetailMaxBytesV0 {
		return detail
	}
	return strings.TrimSpace(string([]rune(detail)[:goalIssueDetailRuneLimitV0(detail, GoalWorkIssueDetailMaxBytesV0)]))
}

func NormalizeGoalWorkIssueV0(issue GoalWorkIssueV0) GoalWorkIssueV0 {
	return normalizeGoalWorkIssueV0(issue)
}

func NormalizeGoalContextBudgetV0(metric GoalContextBudgetV0) GoalContextBudgetV0 {
	if metric.ContextBudgetTotalBytes < 0 {
		metric.ContextBudgetTotalBytes = 0
	}
	if metric.StaticPromptBytes < 0 {
		metric.StaticPromptBytes = 0
	}
	if metric.QueriedContextBytes < 0 {
		metric.QueriedContextBytes = 0
	}
	if metric.MaterializedContextBytes < 0 {
		metric.MaterializedContextBytes = 0
	}
	if metric.DynamicContextBytes < 0 {
		metric.DynamicContextBytes = 0
	}
	metric.CodeContextCacheStatus = strings.TrimSpace(metric.CodeContextCacheStatus)
	if metric.DynamicContextBytes == 0 && (metric.QueriedContextBytes > 0 || metric.MaterializedContextBytes > 0) {
		metric.DynamicContextBytes = metric.QueriedContextBytes + metric.MaterializedContextBytes
	}
	if metric.ContextBudgetTotalBytes == 0 && (metric.StaticPromptBytes > 0 || metric.DynamicContextBytes > 0) {
		metric.ContextBudgetTotalBytes = metric.StaticPromptBytes + metric.DynamicContextBytes
	}
	return metric
}

func GoalContextBudgetEmptyV0(metric GoalContextBudgetV0) bool {
	metric = NormalizeGoalContextBudgetV0(metric)
	return metric.ContextBudgetTotalBytes == 0 &&
		metric.StaticPromptBytes == 0 &&
		metric.QueriedContextBytes == 0 &&
		metric.MaterializedContextBytes == 0 &&
		metric.DynamicContextBytes == 0 &&
		metric.CodeContextCacheStatus == ""
}

func MergeGoalContextBudgetV0(base GoalContextBudgetV0, update GoalContextBudgetV0) GoalContextBudgetV0 {
	base = NormalizeGoalContextBudgetV0(base)
	update = NormalizeGoalContextBudgetV0(update)
	if update.ContextBudgetTotalBytes > 0 {
		base.ContextBudgetTotalBytes = update.ContextBudgetTotalBytes
	}
	if update.StaticPromptBytes > 0 {
		base.StaticPromptBytes = update.StaticPromptBytes
	}
	if update.QueriedContextBytes > 0 {
		base.QueriedContextBytes = update.QueriedContextBytes
	}
	if update.MaterializedContextBytes > 0 {
		base.MaterializedContextBytes = update.MaterializedContextBytes
	}
	if update.DynamicContextBytes > 0 {
		base.DynamicContextBytes = update.DynamicContextBytes
	}
	if update.CodeContextCacheStatus != "" {
		base.CodeContextCacheStatus = update.CodeContextCacheStatus
	}
	return NormalizeGoalContextBudgetV0(base)
}

func normalizeGoalWorkIssueV0(issue GoalWorkIssueV0) GoalWorkIssueV0 {
	issue.Code = strings.TrimSpace(issue.Code)
	issue.Field = strings.TrimSpace(issue.Field)
	issue.Detail = NormalizeGoalWorkIssueDetailV0(issue.Detail)
	if issue.Detail == "" {
		if code, detail, ok := splitGoalWorkIssueDetailCarrierV0(issue.Code); ok {
			issue.Code = code
			issue.Detail = NormalizeGoalWorkIssueDetailV0(detail)
		}
	}
	return issue
}

func splitGoalWorkIssueDetailCarrierV0(value string) (string, string, bool) {
	left, right, found := strings.Cut(strings.TrimSpace(value), ":")
	if !found {
		return "", "", false
	}
	left = strings.TrimSpace(left)
	right = strings.TrimSpace(right)
	if left == "" || right == "" || strings.ContainsAny(left, " \n\r\t/\\") {
		return "", "", false
	}
	return left, right, true
}

func goalIssueDetailRuneLimitV0(value string, maxBytes int) int {
	total := 0
	for i, r := range []rune(value) {
		total += len(string(r))
		if total > maxBytes {
			if i == 0 {
				return 0
			}
			return i
		}
	}
	return len([]rune(value))
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
	validateGoalRefsV0(&issues, "implementer_agent_ref", spec.ImplementerAgentRef)
	validateGoalRefsV0(&issues, "implementer_credential_ref", spec.ImplementerCredentialRef)
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
	requiresIndependentRequiredTestAttestation := spec.ClosurePolicy.RequireIndependentRequiredTestAttestation || len(spec.ClosurePolicy.RequiredAcceptanceCriteriaRefs) > 0
	if requiresIndependentRequiredTestAttestation {
		if len(spec.RequiredTests) == 0 {
			issues = append(issues, GoalWorkIssueV0{Code: ErrGoalRequiredTestAttestationMissingV0, Field: "required_tests"})
		}
		if spec.WriteSetSHA256 != GoalWriteSetSHA256V0(spec.WriteSet) {
			issues = append(issues, GoalWorkIssueV0{Code: ErrGoalRequiredTestAttestationMismatchV0, Field: "write_set_sha256"})
		}
		for _, test := range spec.RequiredTests {
			validateFrozenGoalRequiredTestV0(&issues, test, "required_tests")
		}
	}
	if len(spec.ClosurePolicy.RequiredAcceptanceCriteriaRefs) > 0 {
		if !spec.ClosurePolicy.RequireIndependentRequiredTestAttestation {
			issues = append(issues, GoalWorkIssueV0{Code: ErrGoalRequiredTestAttestationMissingV0, Field: "closure_policy.require_independent_required_test_attestation"})
		}
		for _, criterionRef := range spec.ClosurePolicy.RequiredAcceptanceCriteriaRefs {
			validateRequiredGoalRefV0(&issues, "closure_policy.required_acceptance_criteria_refs", criterionRef)
			if !goalRequiredAcceptanceCriterionCoveredByFrozenTestV0(criterionRef, spec.RequiredTests) {
				issues = append(issues, GoalWorkIssueV0{Code: ErrGoalRequiredAcceptanceCriterionAttestationMissingV0, Field: "closure_policy.required_acceptance_criteria_refs"})
			}
		}
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

func goalRequiredAcceptanceCriterionCoveredByFrozenTestV0(criterionRef string, requiredTests []GoalRequiredTestV0) bool {
	for _, test := range requiredTests {
		for _, testCriterionRef := range test.AcceptanceCriteriaRefs {
			if criterionRef == testCriterionRef {
				return true
			}
		}
	}
	return false
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
	closureEvidenceRefs := goalClosureEvidenceRefsV0(result)
	partialIssues := goalPartialArtifactClosureIssuesV0(spec, result)
	if result.Status != GoalStatusCompleteV0 {
		issues := []GoalWorkIssueV0{{Code: ErrGoalClosureInvalidV0, Field: "status"}}
		issues = append(issues, result.Issues...)
		issues = append(issues, partialIssues...)
		return GoalClosureValidationV0{
			Status:       GoalStatusBlockedV0,
			NeedsRework:  result.Status == GoalStatusBlockedV0 || result.Status == GoalStatusInvalidV0 || len(partialIssues) > 0,
			EvidenceRefs: closureEvidenceRefs,
			Issues:       issues,
		}
	}
	if spec.ClosurePolicy.RequireArtifactPaths && len(result.ArtifactPaths) == 0 {
		return GoalClosureValidationV0{
			Status:      GoalStatusBlockedV0,
			NeedsRework: true,
			Issues:      []GoalWorkIssueV0{{Code: ErrGoalClosureInvalidV0, Field: "artifact_paths"}},
		}
	}
	if spec.ClosurePolicy.RequireMaterializedArtifacts && len(result.MaterializedArtifacts) == 0 {
		return GoalClosureValidationV0{
			Status:      GoalStatusBlockedV0,
			NeedsRework: true,
			Issues:      []GoalWorkIssueV0{{Code: ErrGoalClosureInvalidV0, Field: "materialized_artifacts"}},
		}
	}
	if len(partialIssues) > 0 {
		return GoalClosureValidationV0{
			Status:       GoalStatusBlockedV0,
			NeedsRework:  true,
			EvidenceRefs: closureEvidenceRefs,
			Issues:       partialIssues,
		}
	}
	if outOfScope := goalArtifactPathsOutsideWriteSetV0(spec.WriteSet, result.ArtifactPaths); len(outOfScope) > 0 {
		issues := make([]GoalWorkIssueV0, 0, len(outOfScope))
		for range outOfScope {
			issues = append(issues, GoalWorkIssueV0{Code: ErrGoalArtifactPathScopeV0, Field: "artifact_paths"})
		}
		return GoalClosureValidationV0{Status: GoalStatusBlockedV0, NeedsRework: true, Issues: issues}
	}
	if outOfScope := goalArtifactPathsOutsideWriteSetV0(spec.WriteSet, goalMaterializedArtifactPathsForScopeV0(result)); len(outOfScope) > 0 {
		issues := make([]GoalWorkIssueV0, 0, len(outOfScope))
		for range outOfScope {
			issues = append(issues, GoalWorkIssueV0{Code: ErrGoalArtifactPathScopeV0, Field: "materialized_artifacts.path"})
		}
		return GoalClosureValidationV0{Status: GoalStatusBlockedV0, NeedsRework: true, Issues: issues}
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
		EvidenceRefs: closureEvidenceRefs,
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
	result.ContextBudget = NormalizeGoalContextBudgetV0(result.ContextBudget)
	for i := range result.ArtifactRefs {
		result.ArtifactRefs[i] = strings.TrimSpace(result.ArtifactRefs[i])
	}
	for i := range result.ArtifactPaths {
		result.ArtifactPaths[i] = filepath.ToSlash(strings.TrimSpace(result.ArtifactPaths[i]))
	}
	for i := range result.MaterializedArtifacts {
		result.MaterializedArtifacts[i].ArtifactRef = strings.TrimSpace(result.MaterializedArtifacts[i].ArtifactRef)
		result.MaterializedArtifacts[i].Path = filepath.ToSlash(strings.TrimSpace(result.MaterializedArtifacts[i].Path))
		result.MaterializedArtifacts[i].ArtifactType = strings.TrimSpace(result.MaterializedArtifacts[i].ArtifactType)
		result.MaterializedArtifacts[i].Scope = strings.TrimSpace(result.MaterializedArtifacts[i].Scope)
		result.MaterializedArtifacts[i].Status = strings.TrimSpace(result.MaterializedArtifacts[i].Status)
		for j := range result.MaterializedArtifacts[i].EvidenceRefs {
			result.MaterializedArtifacts[i].EvidenceRefs[j] = strings.TrimSpace(result.MaterializedArtifacts[i].EvidenceRefs[j])
		}
		for j := range result.MaterializedArtifacts[i].Issues {
			result.MaterializedArtifacts[i].Issues[j] = normalizeGoalWorkIssueV0(result.MaterializedArtifacts[i].Issues[j])
		}
	}
	for i := range result.Checklist.ExpectedRefs {
		result.Checklist.ExpectedRefs[i] = strings.TrimSpace(result.Checklist.ExpectedRefs[i])
	}
	for i := range result.Checklist.CompletedRefs {
		result.Checklist.CompletedRefs[i] = strings.TrimSpace(result.Checklist.CompletedRefs[i])
	}
	for i := range result.Checklist.MissingRefs {
		result.Checklist.MissingRefs[i] = strings.TrimSpace(result.Checklist.MissingRefs[i])
	}
	for i := range result.Checklist.EvidenceRefs {
		result.Checklist.EvidenceRefs[i] = strings.TrimSpace(result.Checklist.EvidenceRefs[i])
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
	for i := range result.ReworkPlanRefs {
		result.ReworkPlanRefs[i] = strings.TrimSpace(result.ReworkPlanRefs[i])
	}
	for i := range result.EvidenceRefs {
		result.EvidenceRefs[i] = strings.TrimSpace(result.EvidenceRefs[i])
	}
	for i := range result.Issues {
		result.Issues[i] = normalizeGoalWorkIssueV0(result.Issues[i])
	}
	if result.RepairReceipt != nil {
		receipt := *result.RepairReceipt
		receipt.SchemaVersion = GoalWorkResultRepairReceiptSchemaV0
		receipt.OriginalSchemaVersion = strings.TrimSpace(receipt.OriginalSchemaVersion)
		receipt.OriginalRefHash = strings.TrimSpace(receipt.OriginalRefHash)
		for i := range receipt.Transformations {
			receipt.Transformations[i].Field = strings.TrimSpace(receipt.Transformations[i].Field)
			receipt.Transformations[i].Kind = strings.TrimSpace(receipt.Transformations[i].Kind)
		}
		for i := range receipt.EvidenceRefs {
			receipt.EvidenceRefs[i] = strings.TrimSpace(receipt.EvidenceRefs[i])
		}
		result.RepairReceipt = &receipt
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
	state.LaunchReceipt = NormalizeGoalLaunchReceiptV0(state.LaunchReceipt)
	state.ContextBudget = MergeGoalContextBudgetV0(state.ContextBudget, state.LaunchReceipt.ContextBudget)
	if state.LastResult != nil {
		normalized := NormalizeGoalWorkResultV0(*state.LastResult)
		state.ContextBudget = MergeGoalContextBudgetV0(state.ContextBudget, normalized.ContextBudget)
		normalized.ContextBudget = state.ContextBudget
		state.LastResult = &normalized
	}
	if state.LastClosure != nil {
		closure := *state.LastClosure
		closure.EvidenceRefs = compactGoalStringsV0(closure.EvidenceRefs)
		closure.Issues = append([]GoalWorkIssueV0(nil), closure.Issues...)
		for i := range closure.Issues {
			closure.Issues[i] = normalizeGoalWorkIssueV0(closure.Issues[i])
		}
		closure.AttestationVerifications = append([]GoalRequiredTestIdentityVerificationV0(nil), closure.AttestationVerifications...)
		for i := range closure.AttestationVerifications {
			closure.AttestationVerifications[i] = normalizeGoalRequiredTestIdentityVerificationV0(closure.AttestationVerifications[i])
		}
		state.LastClosure = &closure
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
	if marker.Spec != nil {
		normalized := NormalizeGoalWorkSpecV0(*marker.Spec)
		marker.Spec = &normalized
	}
	if marker.LaunchReceipt != nil {
		normalized := NormalizeGoalLaunchReceiptV0(*marker.LaunchReceipt)
		marker.LaunchReceipt = &normalized
		marker.ContextBudget = MergeGoalContextBudgetV0(marker.ContextBudget, normalized.ContextBudget)
	}
	marker.ContextBudget = NormalizeGoalContextBudgetV0(marker.ContextBudget)
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
	if marker.Spec != nil {
		if marker.Spec.RunRef != "" && marker.Spec.RunRef != marker.RunRef {
			issues = append(issues, GoalWorkIssueV0{Code: ErrGoalClosureInvalidV0, Field: "spec.run_ref"})
		}
		if marker.GoalRef != "" && marker.Spec.GoalRef != "" && marker.Spec.GoalRef != marker.GoalRef {
			issues = append(issues, GoalWorkIssueV0{Code: ErrGoalClosureInvalidV0, Field: "spec.goal_ref"})
		}
		issues = append(issues, ValidateGoalWorkSpecV0(*marker.Spec)...)
	}
	if marker.LaunchReceipt != nil {
		if marker.GoalRef != "" && marker.LaunchReceipt.GoalRef != "" && marker.LaunchReceipt.GoalRef != marker.GoalRef {
			issues = append(issues, GoalWorkIssueV0{Code: ErrGoalClosureInvalidV0, Field: "launch_receipt.goal_ref"})
		}
		issues = append(issues, ValidateGoalLaunchReceiptV0(*marker.LaunchReceipt)...)
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
	for _, artifactPath := range result.ArtifactPaths {
		if !validGoalWriteScopePathV0(artifactPath) {
			issues = append(issues, GoalWorkIssueV0{Code: ErrGoalWriteSetInvalidV0, Field: "artifact_paths"})
		}
	}
	for _, artifact := range result.MaterializedArtifacts {
		if artifact.ArtifactRef == "" && artifact.Path == "" {
			issues = append(issues, GoalWorkIssueV0{Code: ErrGoalMaterializedArtifactInvalidV0, Field: "materialized_artifacts"})
		}
		validateGoalRefsV0(&issues, "materialized_artifacts.artifact_ref", artifact.ArtifactRef)
		if artifact.Path != "" && !validGoalWriteScopePathV0(artifact.Path) {
			issues = append(issues, GoalWorkIssueV0{Code: ErrGoalWriteSetInvalidV0, Field: "materialized_artifacts.path"})
		}
		if !validGoalMaterializedArtifactStatusV0(artifact.Status) {
			issues = append(issues, GoalWorkIssueV0{Code: ErrGoalMaterializedArtifactInvalidV0, Field: "materialized_artifacts.status"})
		}
		for _, evidenceRef := range artifact.EvidenceRefs {
			validateRequiredGoalRefV0(&issues, "materialized_artifacts.evidence_refs", evidenceRef)
		}
		for _, issue := range artifact.Issues {
			validateRequiredGoalRefV0(&issues, "materialized_artifacts.issues.code", issue.Code)
		}
	}
	validateGoalChecklistRefsV0(&issues, "checklist.expected_refs", result.Checklist.ExpectedRefs)
	validateGoalChecklistRefsV0(&issues, "checklist.completed_refs", result.Checklist.CompletedRefs)
	validateGoalChecklistRefsV0(&issues, "checklist.missing_refs", result.Checklist.MissingRefs)
	validateGoalChecklistRefsV0(&issues, "checklist.evidence_refs", result.Checklist.EvidenceRefs)
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
	for _, reworkPlanRef := range result.ReworkPlanRefs {
		validateRequiredGoalRefV0(&issues, "rework_plan_refs", reworkPlanRef)
	}
	for _, evidenceRef := range result.EvidenceRefs {
		validateRequiredGoalRefV0(&issues, "evidence_refs", evidenceRef)
	}
	for _, issue := range result.Issues {
		validateRequiredGoalRefV0(&issues, "issues.code", issue.Code)
	}
	if result.RepairReceipt != nil {
		if strings.TrimSpace(result.RepairReceipt.OriginalRefHash) == "" {
			issues = append(issues, GoalWorkIssueV0{Code: ErrGoalResultJSONInvalidV0, Field: "repair_receipt.original_ref_hash"})
		}
		for _, evidenceRef := range result.RepairReceipt.EvidenceRefs {
			validateRequiredGoalRefV0(&issues, "repair_receipt.evidence_refs", evidenceRef)
		}
	}
	return issues
}

func validateGoalChecklistRefsV0(issues *[]GoalWorkIssueV0, field string, refs []string) {
	for _, ref := range refs {
		validateRequiredGoalRefV0(issues, field, ref)
	}
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

func validGoalMaterializedArtifactStatusV0(status string) bool {
	switch strings.TrimSpace(status) {
	case GoalMaterializedArtifactStatusValidV0,
		GoalMaterializedArtifactStatusInvalidV0,
		GoalMaterializedArtifactStatusPartialV0,
		GoalMaterializedArtifactStatusNonPublishableV0:
		return true
	default:
		return false
	}
}
