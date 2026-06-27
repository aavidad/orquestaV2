package orquestaserver

import (
	"context"
	"strings"

	orquestagoal "orquesta/modulos/orquesta-goal"
)

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
		}, orquestagoal.GoalWorkLifecyclePortsV0{
			Launcher:   launcher,
			StateStore: runtime.goalStateStore,
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
	tests := make([]orquestagoal.GoalRequiredTestV0, 0, len(request.RequiredTests))
	for _, command := range compactConfigStringsV0(request.RequiredTests) {
		tests = append(tests, orquestagoal.GoalRequiredTestV0{
			TestRef: "required-test-ref-" + idleSelfImprovementHashV0(command),
			Command: command,
		})
	}
	contextRefs := make([]orquestagoal.GoalContextRefV0, 0, len(request.ContextRefs))
	for _, ref := range compactConfigStringsV0(request.ContextRefs) {
		contextRefs = append(contextRefs, orquestagoal.GoalContextRefV0{Kind: "context_ref", Ref: ref})
	}
	rules := make([]orquestagoal.GoalRuleRefV0, 0, len(request.CompactRules))
	for _, ref := range compactConfigStringsV0(request.CompactRules) {
		rules = append(rules, orquestagoal.GoalRuleRefV0{
			Kind:        "compact_rule",
			Ref:         ref,
			Enforcement: orquestagoal.GoalRuleEnforcementAdvisoryV0,
		})
	}
	return orquestagoal.NormalizeGoalWorkSpecV0(orquestagoal.GoalWorkSpecV0{
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
		SkillRefs:          append([]string(nil), runtime.config.IdleSelfImprovementSkillRefs...),
		WriteSet:           writeSet,
		RequiredTests:      tests,
		AcceptanceCriteria: append([]string(nil), request.AcceptanceCriteria...),
		EvidenceRefs:       append([]string(nil), request.EvidenceRefs...),
		ClosurePolicy:      orquestagoal.GoalClosurePolicyV0{RequireRequiredTests: len(tests) > 0},
		ReworkPolicy:       orquestagoal.GoalReworkPolicyV0{PreferNewGoal: true, MaxReworkGoals: 1, PreserveArtifacts: true},
	})
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
	return b.String()
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
