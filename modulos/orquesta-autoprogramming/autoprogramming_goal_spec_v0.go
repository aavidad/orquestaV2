package orquestaautoprogramming

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	orquestagoal "orquesta/modulos/orquesta-goal"
)

const (
	AutoprogrammingGoalWorkKindV0           = "autoprogramming"
	autoprogrammingGoalSpecIssueV0          = "goal_work_spec_invalid"
	autoprogrammingWorktreeBaselinePrefixV0 = "worktree_baseline_ref:"
)

func BuildAutoprogrammingGoalWorkSpecsV0(
	work AutoprogrammingProgrammableWorkV0,
) ([]orquestagoal.GoalWorkSpecV0, []AutoprogrammingRequestIssueV0) {
	if work.GoalMigration.Status != AutoprogrammingGoalMigrationGoalReadyV0 {
		return []orquestagoal.GoalWorkSpecV0{}, nil
	}
	specs := make([]orquestagoal.GoalWorkSpecV0, 0, len(work.Groups))
	for index, group := range work.Groups {
		spec := autoprogrammingGoalWorkSpecForGroupV0(work, group, index)
		if issues := orquestagoal.ValidateGoalWorkSpecV0(spec); len(issues) > 0 {
			return []orquestagoal.GoalWorkSpecV0{}, []AutoprogrammingRequestIssueV0{
				autoprogrammingGoalSpecIssueFromGoalIssuesV0(index, issues),
			}
		}
		specs = append(specs, spec)
	}
	if specs == nil {
		return []orquestagoal.GoalWorkSpecV0{}, nil
	}
	return specs, nil
}

func autoprogrammingGoalWorkSpecForGroupV0(
	work AutoprogrammingProgrammableWorkV0,
	group AutoprogrammingProgrammableGroupV0,
	index int,
) orquestagoal.GoalWorkSpecV0 {
	task := group.Task
	writeSet := autoprogrammingGoalWriteSetV0(group.WriteSet)
	requiredTests := autoprogrammingGoalRequiredTestsV0(task.RequiredTests, task.TaskID, group.AcceptanceChecks)
	return orquestagoal.NormalizeGoalWorkSpecV0(orquestagoal.GoalWorkSpecV0{
		GoalRef:                   "goal-ref-" + task.TaskID,
		RequestRef:                work.RequestRef,
		ProjectRef:                work.ProjectRef,
		WorkKind:                  AutoprogrammingGoalWorkKindV0,
		WorkProfileKind:           string(task.WorkProfileKind),
		Objective:                 firstAutoprogrammingGoalTextV0(group.Profile.Objective, task.Title, task.Summary),
		DirectorKind:              orquestagoal.GoalDirectorKindCodexGoalV0,
		ContextRefs:               autoprogrammingGoalContextRefsV0(work, group),
		RuleRefs:                  autoprogrammingGoalRuleRefsV0(group),
		SkillRefs:                 append([]string(nil), task.SkillRefs...),
		WriteSet:                  writeSet,
		DestructiveAuthorizations: append([]orquestagoal.GoalDestructiveChangeAuthorizationV0(nil), group.DestructiveAuthorizations...),
		WriteSetSHA256:            orquestagoal.GoalWriteSetSHA256V0(writeSet),
		RequiredTests:             requiredTests,
		AcceptanceCriteria:        append([]string(nil), task.AcceptanceCriteria...),
		EvidenceRefs:              autoprogrammingGoalEvidenceRefsV0(work, task.TaskID, index),
		Budget: orquestagoal.GoalBudgetV0{
			MaxSubgoals: task.MaxRecursiveAgents,
		},
		ClosurePolicy: autoprogrammingGoalClosurePolicyV0(requiredTests, group.AcceptanceChecks),
		ReworkPolicy: orquestagoal.GoalReworkPolicyV0{
			PreferNewGoal:     true,
			MaxReworkGoals:    1,
			PreserveArtifacts: true,
		},
	})
}

func autoprogrammingGoalContextRefsV0(
	work AutoprogrammingProgrammableWorkV0,
	group AutoprogrammingProgrammableGroupV0,
) []orquestagoal.GoalContextRefV0 {
	var refs []orquestagoal.GoalContextRefV0
	for _, ref := range []struct {
		kind string
		ref  string
	}{
		{kind: "request", ref: work.RequestRef},
		{kind: "project", ref: work.ProjectRef},
		{kind: "worktree", ref: work.WorktreeRef},
		{kind: "branch", ref: work.BranchRef},
	} {
		if strings.TrimSpace(ref.ref) != "" {
			refs = append(refs, orquestagoal.GoalContextRefV0{Kind: ref.kind, Ref: ref.ref, Required: true})
		}
	}
	for _, ref := range group.Task.ContextRefs {
		ref = strings.TrimSpace(ref)
		if ref == "" {
			continue
		}
		if baselineRef, ok := strings.CutPrefix(ref, autoprogrammingWorktreeBaselinePrefixV0); ok {
			baselineRef = strings.TrimSpace(baselineRef)
			if baselineRef != "" {
				refs = append(refs, orquestagoal.GoalContextRefV0{
					Kind:     "worktree_baseline",
					Ref:      baselineRef,
					Purpose:  "Baseline congelado del worktree antes de lanzar el Goal.",
					Required: true,
				})
			}
			continue
		}
		refs = append(refs, orquestagoal.GoalContextRefV0{
			Kind:    "workflow_task_context",
			Ref:     ref,
			Purpose: "Contexto compacto heredado de WorkflowTaskV0.",
		})
	}
	return refs
}

func autoprogrammingGoalRuleRefsV0(
	group AutoprogrammingProgrammableGroupV0,
) []orquestagoal.GoalRuleRefV0 {
	refs := []orquestagoal.GoalRuleRefV0{
		{Kind: "repo_instructions", Ref: "AGENTS.md", Enforcement: orquestagoal.GoalRuleEnforcementHardV0},
		{Kind: "goal_first_cut", Ref: "docs/orquesta_goal_first_codex_2026-06-25.md", Enforcement: orquestagoal.GoalRuleEnforcementAdvisoryV0},
	}
	refs = append(refs, autoprogrammingGoalContractRuleRefsForGroupV0(group)...)
	return refs
}

func autoprogrammingGoalWriteSetV0(paths []string) []orquestagoal.GoalWriteScopeV0 {
	out := make([]orquestagoal.GoalWriteScopeV0, 0, len(paths))
	for _, path := range paths {
		if strings.TrimSpace(path) != "" {
			out = append(out, orquestagoal.GoalWriteScopeV0{
				Path:    path,
				Purpose: "Write-set acotado de autoprogramacion.",
			})
		}
	}
	if out == nil {
		return []orquestagoal.GoalWriteScopeV0{}
	}
	return out
}

func autoprogrammingGoalRequiredTestsV0(
	commands []string,
	taskID string,
	checks []AutoprogrammingAcceptanceCheckV0,
) []orquestagoal.GoalRequiredTestV0 {
	out := make([]orquestagoal.GoalRequiredTestV0, 0, len(commands))
	for index, command := range commands {
		command = strings.TrimSpace(command)
		if command == "" {
			continue
		}
		out = append(out, orquestagoal.FreezeGoalRequiredTestV0(orquestagoal.GoalRequiredTestV0{
			TestRef:    "test-ref-" + taskID + "-" + shortAutoprogrammingGoalHashV0(fmt.Sprintf("%02d:%s", index+1, command)),
			CommandRef: "command-ref-" + taskID + "-" + shortAutoprogrammingGoalHashV0(command),
			Command:    command,
		}))
	}
	byCommand := make(map[string]int, len(out))
	for index, test := range out {
		byCommand[test.Command] = index
	}
	for _, check := range checks {
		command := strings.TrimSpace(check.Command)
		if index, ok := byCommand[command]; ok {
			out[index] = autoprogrammingGoalRequiredTestWithAcceptanceCheckV0(out[index], check)
			continue
		}
		out = append(out, autoprogrammingGoalRequiredTestWithAcceptanceCheckV0(orquestagoal.GoalRequiredTestV0{
			TestRef:    "test-ref-" + taskID + "-" + shortAutoprogrammingGoalHashV0(fmt.Sprintf("%02d:%s", len(out)+1, command)),
			CommandRef: "command-ref-" + taskID + "-" + shortAutoprogrammingGoalHashV0(command),
			Command:    command,
		}, check))
		byCommand[command] = len(out) - 1
	}
	if out == nil {
		return []orquestagoal.GoalRequiredTestV0{}
	}
	return out
}

func autoprogrammingGoalRequiredTestWithAcceptanceCheckV0(
	test orquestagoal.GoalRequiredTestV0,
	check AutoprogrammingAcceptanceCheckV0,
) orquestagoal.GoalRequiredTestV0 {
	if description := strings.TrimSpace(check.Description); description != "" {
		test.AcceptanceCriteria = compactStringsV0(append(test.AcceptanceCriteria, description))
	}
	test.AcceptanceCriteriaRefs = compactStringsV0(append(test.AcceptanceCriteriaRefs, check.CriterionRef))
	return orquestagoal.FreezeGoalRequiredTestV0(test)
}

func autoprogrammingGoalClosurePolicyV0(
	requiredTests []orquestagoal.GoalRequiredTestV0,
	checks []AutoprogrammingAcceptanceCheckV0,
) orquestagoal.GoalClosurePolicyV0 {
	criterionRefs := make([]string, 0, len(checks))
	for _, check := range checks {
		criterionRefs = append(criterionRefs, check.CriterionRef)
	}
	return orquestagoal.GoalClosurePolicyV0{
		RequireRequiredTests:                      len(requiredTests) > 0,
		RequireIndependentRequiredTestAttestation: len(requiredTests) > 0,
		RequiredAcceptanceCriteriaRefs:            compactStringsV0(criterionRefs),
	}
}

func autoprogrammingGoalEvidenceRefsV0(
	work AutoprogrammingProgrammableWorkV0,
	taskID string,
	index int,
) []string {
	return compactStringsV0([]string{
		"evidence-ref-goal-migration-" + work.GoalMigration.Status,
		"evidence-ref-autoprogramming-goal-spec-" + taskID,
		fmt.Sprintf("evidence-ref-autoprogramming-goal-group-%02d", index+1),
	})
}

func autoprogrammingGoalSpecIssueFromGoalIssuesV0(
	index int,
	issues []orquestagoal.GoalWorkIssueV0,
) AutoprogrammingRequestIssueV0 {
	field := fmt.Sprintf("goal_specs.%d", index)
	if len(issues) > 0 && strings.TrimSpace(issues[0].Field) != "" {
		field += "." + strings.TrimSpace(issues[0].Field)
	}
	message := autoprogrammingGoalSpecIssueSummaryV0(issues)
	return autoprogrammingRequestIssueV0(autoprogrammingGoalSpecIssueV0, field, message)
}

func autoprogrammingGoalSpecIssueSummaryV0(issues []orquestagoal.GoalWorkIssueV0) string {
	values := make([]string, 0, len(issues))
	for _, issue := range issues {
		if strings.TrimSpace(issue.Code) != "" {
			values = append(values, strings.TrimSpace(issue.Code))
		}
	}
	if len(values) == 0 {
		return autoprogrammingGoalSpecIssueV0
	}
	return strings.Join(values, ",")
}

func firstAutoprogrammingGoalTextV0(values ...string) string {
	for _, value := range values {
		if text := strings.TrimSpace(value); text != "" {
			return text
		}
	}
	return "Autoprogramacion acotada."
}

func shortAutoprogrammingGoalHashV0(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])[:12]
}
