package orquestaserver

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestagoal "orquesta/modulos/orquesta-goal"
)

const idleSelfImprovementGoalObjectiveMaxRunesV0 = 4000

func (runtime *RuntimeV0) launchIdleSelfImprovementGoalsV0(
	ctx context.Context,
	launcher IdleSelfImprovementGoalLauncherPortV0,
	requests []IdleSelfImprovementRequestV0,
) {
	failed := false
	failures := make([]IdleSelfImprovementResultV0, 0, len(requests))
	for _, request := range requests {
		spec := runtime.idleSelfImprovementGoalWorkSpecV0(request)
		start, err := orquestagoal.StartGoalWorkV0(ctx, orquestagoal.GoalWorkStartRequestV0{
			RunRef: spec.RunRef,
			Spec:   spec,
			EvidenceRefs: []string{
				"evidence-ref-idle-self-improvement-goal-state-v0",
			},
			// Snapshot, attestation and identity are consumed by the stack's
			// observation/closure path; launch only needs the spec binder.
		}, orquestagoal.GoalWorkLifecyclePortsV0{
			Launcher:               launcher,
			StateStore:             runtime.goalStateStore,
			RequiredTestSpecBinder: runtime.goalRequiredTestSpecBinder,
		})
		result := idleSelfImprovementGoalLaunchResultV0(request, spec, start.State, start.Receipt, err)
		if err != nil || !result.Accepted {
			failed = true
			failures = append(failures, result)
			continue
		}
		if failed {
			result.NextActions = compactConfigStringsV0(append(result.NextActions,
				"repair_failed_goal_launches_and_retry",
			))
		}
		runtime.persistStateTransitionV0(
			ctx,
			runtime.tracker.MarkIdleSelfImprovementPreparedV0(result, runtime.clock.Now()),
			"idle_self_improvement_goal_launched",
		)
	}
	if failed {
		runtime.markIdleSelfImprovementPrepareFailedV0(ctx, failures)
	}
}

func (runtime *RuntimeV0) idleSelfImprovementGoalWorkSpecV0(
	request IdleSelfImprovementRequestV0,
) orquestagoal.GoalWorkSpecV0 {
	writeSet := make([]orquestagoal.GoalWriteScopeV0, 0, len(request.WriteSet))
	for _, path := range compactConfigStringsV0(request.WriteSet) {
		writeSet = append(writeSet, orquestagoal.GoalWriteScopeV0{Path: path})
	}
	tests, acceptanceCriterionRefs, invalidAcceptanceChecks := idleSelfImprovementGoalRequiredTestsV0(request)
	contextRefs := make([]orquestagoal.GoalContextRefV0, 0, len(request.ContextRefs))
	for _, ref := range compactConfigStringsV0(request.ContextRefs) {
		contextRefs = append(contextRefs, orquestagoal.GoalContextRefV0{Kind: "context_ref", Ref: ref})
	}
	skillRefs := compactConfigStringsV0(append(
		append([]string(nil), runtime.config.IdleSelfImprovementSkillRefs...),
		request.SkillRefs...,
	))
	for _, ref := range compactConfigStringsV0(request.SkillRefs) {
		contextRefs = append(contextRefs, orquestagoal.GoalContextRefV0{
			Kind:    "skill_ref",
			Ref:     ref,
			Purpose: "Habilidad curada casada por clase de tarea.",
		})
	}
	rules := make([]orquestagoal.GoalRuleRefV0, 0, len(request.CompactRules))
	for _, ref := range compactConfigStringsV0(request.CompactRules) {
		rules = append(rules, orquestagoal.GoalRuleRefV0{
			Kind:        "compact_rule",
			Ref:         ref,
			Enforcement: orquestagoal.GoalRuleEnforcementAdvisoryV0,
		})
	}
	closurePolicy := orquestagoal.GoalClosurePolicyV0{
		RequireRequiredTests:                      len(tests) > 0,
		RequireIndependentRequiredTestAttestation: len(acceptanceCriterionRefs) > 0 || invalidAcceptanceChecks,
		RequiredAcceptanceCriteriaRefs:            acceptanceCriterionRefs,
	}
	if invalidAcceptanceChecks {
		// StartGoalWorkV0 validates this sentinel before launch, so an incomplete
		// typed contract cannot silently fall back to advisory acceptance text.
		tests = append(tests, orquestagoal.GoalRequiredTestV0{
			TestRef: "required-test-ref-invalid-acceptance-check",
		})
	}
	spec := orquestagoal.GoalWorkSpecV0{
		GoalRef:            idleSelfImprovementGoalRefV0(request),
		RequestRef:         request.RequestRef,
		RunRef:             idleSelfImprovementGoalRunRefV0(request),
		ProjectRef:         request.ProjectRef,
		DomainRef:          "domain-ref-autoprogramming",
		WorkKind:           "idle_self_improvement",
		WorkProfileKind:    "implementation",
		Objective:          idleSelfImprovementGoalObjectiveV0(request),
		DirectorKind:       orquestagoal.GoalDirectorKindCodexGoalV0,
		ContextRefs:        contextRefs,
		RuleRefs:           rules,
		SkillRefs:          skillRefs,
		WriteSet:           writeSet,
		RequiredTests:      tests,
		AcceptanceCriteria: append([]string(nil), request.AcceptanceCriteria...),
		EvidenceRefs:       append([]string(nil), request.EvidenceRefs...),
		ClosurePolicy:      closurePolicy,
		ReworkPolicy:       orquestagoal.GoalReworkPolicyV0{PreferNewGoal: true, MaxReworkGoals: 1, PreserveArtifacts: true},
	}
	if closurePolicy.RequireIndependentRequiredTestAttestation {
		spec.WriteSetSHA256 = orquestagoal.GoalWriteSetSHA256V0(writeSet)
	}
	spec = idleSelfImprovementGoalSpecWithFrozenRequiredTestsV0(spec, request)
	return orquestagoal.NormalizeGoalWorkSpecV0(spec)
}

func idleSelfImprovementGoalRequiredTestsV0(
	request IdleSelfImprovementRequestV0,
) ([]orquestagoal.GoalRequiredTestV0, []string, bool) {
	tests := make([]orquestagoal.GoalRequiredTestV0, 0, len(request.RequiredTests)+len(request.AcceptanceChecks))
	byCommand := make(map[string]int, len(request.RequiredTests)+len(request.AcceptanceChecks))
	addTest := func(command string) int {
		command = strings.TrimSpace(command)
		if index, ok := byCommand[command]; ok {
			return index
		}
		index := len(tests)
		byCommand[command] = index
		tests = append(tests, orquestagoal.GoalRequiredTestV0{
			TestRef:    "required-test-ref-" + idleSelfImprovementHashV0(command),
			CommandRef: "command-ref-" + idleSelfImprovementHashV0(command),
			Command:    command,
		})
		return index
	}
	for _, command := range compactConfigStringsV0(request.RequiredTests) {
		addTest(command)
	}

	criterionRefs := make([]string, 0, len(request.AcceptanceChecks))
	seenCriterionRefs := make(map[string]bool, len(request.AcceptanceChecks))
	invalid := false
	for _, check := range request.AcceptanceChecks {
		criterionRef := strings.TrimSpace(check.CriterionRef)
		command := strings.TrimSpace(check.Command)
		if criterionRef == "" || command == "" || seenCriterionRefs[criterionRef] {
			invalid = true
			continue
		}
		seenCriterionRefs[criterionRef] = true
		index := addTest(command)
		tests[index].AcceptanceCriteriaRefs = append(tests[index].AcceptanceCriteriaRefs, criterionRef)
		if description := strings.TrimSpace(check.Description); description != "" {
			tests[index].AcceptanceCriteria = append(tests[index].AcceptanceCriteria, description)
		}
		criterionRefs = append(criterionRefs, criterionRef)
	}
	if len(criterionRefs) == 0 {
		return tests, nil, invalid
	}
	for index := range tests {
		tests[index] = orquestagoal.FreezeGoalRequiredTestV0(tests[index])
	}
	return tests, criterionRefs, invalid
}

func idleSelfImprovementGoalRunRefV0(request IdleSelfImprovementRequestV0) string {
	if ref := strings.TrimSpace(request.RequestRef); ref != "" {
		return ref
	}
	if ref := strings.TrimSpace(request.ActiveAttemptRef); ref != "" {
		return ref
	}
	return "run-ref-idle-self-improvement-" + idleSelfImprovementHashV0(request.FailureSummary)
}

func idleSelfImprovementGoalRefV0(request IdleSelfImprovementRequestV0) string {
	base := strings.TrimSpace(request.RequestRef)
	base = strings.TrimPrefix(base, "request-ref-")
	if base == "" {
		base = idleSelfImprovementHashV0(request.FailureSummary)
	}
	return "goal-ref-" + base
}

func idleSelfImprovementGoalObjectiveV0(request IdleSelfImprovementRequestV0) string {
	var b strings.Builder
	b.WriteString(strings.TrimSpace(request.FailureSummary))
	if area := strings.TrimSpace(request.SuggestedArea); area != "" {
		b.WriteString("\nArea: ")
		b.WriteString(area)
	}
	for _, criterion := range compactConfigStringsV0(request.AcceptanceCriteria) {
		b.WriteString("\nCriterio: ")
		b.WriteString(criterion)
	}
	switch request.FrozenRequiredTestsPhase {
	case orquestaautoprogramming.FrozenRequiredTestsDefinerPhaseV0:
		b.WriteString("\nCriterio: definir solo tests requeridos nuevos en ficheros _test.go y no implementar el cambio.")
	case orquestaautoprogramming.FrozenRequiredTestsImplementerPhaseV0:
		b.WriteString("\nCriterio: no modificar tests congelados; el cierre rechaza frozen_tests_modified.")
	}
	return compactIdleSelfImprovementGoalObjectiveV0(b.String())
}

func compactIdleSelfImprovementGoalObjectiveV0(objective string) string {
	objective = strings.TrimSpace(objective)
	if len([]rune(objective)) <= idleSelfImprovementGoalObjectiveMaxRunesV0 {
		return objective
	}
	sum := sha256.Sum256([]byte(objective))
	suffix := fmt.Sprintf(
		"\n\n[objective_compacted original_sha256=%x original_bytes=%d full_context_in_goal_spec_refs]",
		sum[:],
		len([]byte(objective)),
	)
	limit := idleSelfImprovementGoalObjectiveMaxRunesV0 - len([]rune(suffix))
	if limit < 1 {
		return strings.TrimSpace(string([]rune(suffix)[:idleSelfImprovementGoalObjectiveMaxRunesV0]))
	}
	return strings.TrimSpace(string([]rune(objective)[:limit])) + suffix
}

func idleSelfImprovementGoalLaunchResultV0(
	request IdleSelfImprovementRequestV0,
	spec orquestagoal.GoalWorkSpecV0,
	state orquestagoal.GoalWorkStateV0,
	receipt orquestagoal.GoalLaunchReceiptV0,
	err error,
) IdleSelfImprovementResultV0 {
	result := IdleSelfImprovementResultV0{
		RequestRef:      request.RequestRef,
		RunRef:          firstNonEmptyConfigStringV0(state.RunRef, spec.RunRef),
		GoalRef:         spec.GoalRef,
		ExternalGoalRef: firstNonEmptyConfigStringV0(state.ExternalGoalRef, receipt.ExternalGoalRef),
		Status:          firstNonEmptyConfigStringV0(receipt.Status, orquestagoal.GoalStatusAcceptedV0),
		Message:         "goal_first_launched",
		EvidenceRefs:    append([]string(nil), receipt.EvidenceRefs...),
		NextActions:     []string{"observe_goal_ref=" + spec.GoalRef},
		GoalSpec:        spec,
		GoalReceipt:     receipt,
	}
	evidenceRefs := append([]string(nil), result.EvidenceRefs...)
	evidenceRefs = append(evidenceRefs, request.EvidenceRefs...)
	evidenceRefs = append(evidenceRefs, "evidence-ref-idle-self-improvement-goal-first-launched")
	result.EvidenceRefs = compactConfigStringsV0(evidenceRefs)
	if err != nil {
		result.Accepted = false
		result.Status = "error"
		result.Message = err.Error()
		return idleSelfImprovementPrepareFailureFromResultV0(request, result)
	}
	result.Accepted = receipt.Status == "" ||
		receipt.Status == orquestagoal.GoalStatusAcceptedV0 ||
		receipt.Status == orquestagoal.GoalStatusRunningV0
	if !result.Accepted && result.Message == "goal_first_launched" {
		result.Message = "goal_first_launch_rejected"
	}
	return result
}

func firstNonEmptyConfigStringV0(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
