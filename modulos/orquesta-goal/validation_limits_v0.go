package orquestagoal

import "encoding/json"

func validateGoalWorkSpecLimitsV0(issues *[]GoalWorkIssueV0, spec GoalWorkSpecV0) {
	if encoded, err := json.Marshal(spec); err == nil && len(encoded) > GoalWorkSpecMaxProjectedJSONBytesV0 {
		*issues = append(*issues, GoalWorkIssueV0{Code: ErrGoalSpecLimitExceededV0, Field: "goal_work_spec"})
	}

	validateGoalStringLimitV0(issues, "goal_ref", spec.GoalRef, GoalWorkSpecMaxStringBytesV0)
	validateGoalStringLimitV0(issues, "request_ref", spec.RequestRef, GoalWorkSpecMaxStringBytesV0)
	validateGoalStringLimitV0(issues, "run_ref", spec.RunRef, GoalWorkSpecMaxStringBytesV0)
	validateGoalStringLimitV0(issues, "implementer_agent_ref", spec.ImplementerAgentRef, GoalWorkSpecMaxStringBytesV0)
	validateGoalStringLimitV0(issues, "implementer_credential_ref", spec.ImplementerCredentialRef, GoalWorkSpecMaxStringBytesV0)
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
	validateGoalStringLimitV0(issues, "closure_policy.required_attestor_trust_policy_ref", spec.ClosurePolicy.RequiredAttestorTrustPolicyRef, GoalWorkSpecMaxStringBytesV0)
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
