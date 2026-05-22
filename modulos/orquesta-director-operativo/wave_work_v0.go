package orquestadirectoroperativo

import "fmt"

func BuildOperationalDirectorWaveWorkV0(
	plan OperationalDirectorPlanV0,
) OperationalDirectorWaveWorkV0 {
	work := OperationalDirectorWaveWorkV0{
		PlanRef:           plan.PlanRef,
		RequestRef:        plan.RequestRef,
		RunRef:            plan.RunRef,
		ProjectRef:        plan.ProjectRef,
		Mode:              plan.Mode,
		Status:            plan.Status,
		ReadyToLaunch:     plan.Status == OperationalDirectorPlanReadyV0,
		MaxParallelItems:  plan.MaxParallelAgents,
		RecursiveChildren: plan.RecursiveDelegation,
	}
	if len(plan.Steps) == 0 {
		work.Issues = append(work.Issues, issueV0("plan_steps_missing", "steps", "plan sin pasos operativos"))
		return work
	}

	stepsByID := make(map[string]OperationalDirectorStepV0, len(plan.Steps))
	for _, step := range plan.Steps {
		if step.StepID == "" {
			work.Issues = append(work.Issues, issueV0("step_id_missing", "steps.step_id", "paso sin step_id"))
			continue
		}
		if _, exists := stepsByID[step.StepID]; exists {
			work.Issues = append(work.Issues, issueV0("step_id_duplicate", "steps.step_id", "step_id duplicado"))
			continue
		}
		stepsByID[step.StepID] = step
	}
	if len(work.Issues) > 0 {
		return work
	}
	if issues := validateOperationalDirectorPlanShapeV0(plan, stepsByID); len(issues) > 0 {
		work.Issues = append(work.Issues, issues...)
		work.ReadyToLaunch = false
		return work
	}

	done := make(map[string]bool, len(plan.Steps))
	assigned := make(map[string]bool, len(plan.Steps))
	stepWaveIDs := make(map[string]string, len(plan.Steps))
	for len(assigned) < len(plan.Steps) {
		var ready []OperationalDirectorStepV0
		for _, step := range plan.Steps {
			if assigned[step.StepID] || !operationalDirectorStepDepsDoneV0(step, stepsByID, done) {
				continue
			}
			ready = append(ready, step)
		}
		if len(ready) == 0 {
			work.Issues = append(work.Issues, issueV0("step_dependency_cycle", "steps.depends_on", "dependencias de pasos no resolubles"))
			return work
		}

		wave := OperationalDirectorWaveV0{
			WaveID:    fmt.Sprintf("wave-%02d", len(work.Waves)+1),
			Index:     len(work.Waves) + 1,
			DependsOn: operationalDirectorWaveDependsOnV0(ready, stepWaveIDs),
			Items:     make([]OperationalDirectorWorkItemV0, 0, len(ready)),
		}
		for _, step := range ready {
			item := operationalDirectorWorkItemFromStepV0(step)
			wave.Items = append(wave.Items, item)
			assigned[step.StepID] = true
		}
		for _, step := range ready {
			done[step.StepID] = true
			stepWaveIDs[step.StepID] = wave.WaveID
		}
		work.Waves = append(work.Waves, wave)
	}
	return work
}

func validateOperationalDirectorPlanShapeV0(
	plan OperationalDirectorPlanV0,
	stepsByID map[string]OperationalDirectorStepV0,
) []OperationalDirectorIssueV0 {
	var issues []OperationalDirectorIssueV0
	for _, step := range plan.Steps {
		if step.ParentStepID != "" {
			if _, exists := stepsByID[step.ParentStepID]; !exists {
				issues = append(issues, issueV0("parent_step_missing", "steps.parent_step_id", "parent_step_id no existe"))
			}
		}
		for _, childStepID := range step.ChildStepIDs {
			if _, exists := stepsByID[childStepID]; !exists {
				issues = append(issues, issueV0("child_step_missing", "steps.child_step_ids", "child_step_id no existe"))
			}
		}
		if step.DelegationDepth < 0 {
			issues = append(issues, issueV0("delegation_depth_invalid", "steps.delegation_depth", "delegation_depth invalida"))
		}
		if step.DelegationDepth > 0 && plan.MaxDelegationDepth > 0 && step.DelegationDepth > plan.MaxDelegationDepth {
			issues = append(issues, issueV0("delegation_depth_exceeds_budget", "steps.delegation_depth", "delegation_depth excede el presupuesto del plan"))
		}
		if step.MaxChildAgents < 0 {
			issues = append(issues, issueV0("max_child_agents_invalid", "steps.max_child_agents", "max_child_agents invalido"))
		}
		if step.MaxChildAgents > 0 && plan.MaxSubagentsPerAgent > 0 && step.MaxChildAgents > plan.MaxSubagentsPerAgent {
			issues = append(issues, issueV0("max_child_agents_exceeds_budget", "steps.max_child_agents", "max_child_agents excede el presupuesto del plan"))
		}
	}
	if plan.Status == OperationalDirectorPlanReadyV0 {
		issues = append(issues, validateOperationalDirectorReadyPlanStepsV0(plan)...)
	}
	if plan.Status == OperationalDirectorPlanNeedsContextV0 {
		if operationalDirectorPlanHasStepKindV0(plan, OperationalDirectorStepLaunchSubagentsV0) {
			issues = append(issues, issueV0("needs_context_launch_forbidden", "steps.kind", "un plan needs_context no puede lanzar subagentes"))
		}
		if !operationalDirectorPlanHasStepKindV0(plan, OperationalDirectorStepRequestDomainContextV0) {
			issues = append(issues, issueV0("request_domain_context_missing", "steps.kind", "falta pedir contexto de dominio"))
		}
	}
	return issues
}

func validateOperationalDirectorReadyPlanStepsV0(
	plan OperationalDirectorPlanV0,
) []OperationalDirectorIssueV0 {
	var issues []OperationalDirectorIssueV0
	for _, required := range []OperationalDirectorStepKindV0{
		OperationalDirectorStepLaunchSubagentsV0,
		OperationalDirectorStepWaitSubagentsV0,
		OperationalDirectorStepReviewDeliveriesV0,
		OperationalDirectorStepReplanOrCloseV0,
	} {
		if !operationalDirectorPlanHasStepKindV0(plan, required) {
			issues = append(issues, issueV0("ready_step_missing", "steps.kind", "falta paso operativo requerido"))
		}
	}
	if plan.Mode == OperationalDirectorModeProgrammingV0 &&
		!operationalDirectorPlanHasStepKindV0(plan, OperationalDirectorStepRunRequiredTestsV0) {
		issues = append(issues, issueV0("run_required_tests_missing", "steps.kind", "programming requiere run_required_tests antes de cierre"))
	}
	if plan.Mode == OperationalDirectorModeDomainWorkV0 {
		if len(compactStringsV0(plan.DomainRefs)) == 0 {
			issues = append(issues, issueV0("domain_refs_missing", "domain_refs", "refs de dominio requeridas"))
		}
		issues = append(issues, validateOperationalDirectorWriteSetV0(plan.WriteSet)...)
	}
	return issues
}

func operationalDirectorPlanHasStepKindV0(
	plan OperationalDirectorPlanV0,
	kind OperationalDirectorStepKindV0,
) bool {
	for _, step := range plan.Steps {
		if step.Kind == kind {
			return true
		}
	}
	return false
}

func operationalDirectorStepDepsDoneV0(
	step OperationalDirectorStepV0,
	stepsByID map[string]OperationalDirectorStepV0,
	done map[string]bool,
) bool {
	for _, dependency := range step.DependsOn {
		if _, exists := stepsByID[dependency]; !exists {
			return false
		}
		if !done[dependency] {
			return false
		}
	}
	return true
}

func operationalDirectorWaveDependsOnV0(
	ready []OperationalDirectorStepV0,
	stepWaveIDs map[string]string,
) []string {
	dependencies := make([]string, 0)
	seen := make(map[string]bool)
	for _, step := range ready {
		for _, dependency := range step.DependsOn {
			waveID := stepWaveIDs[dependency]
			if waveID == "" || seen[waveID] {
				continue
			}
			seen[waveID] = true
			dependencies = append(dependencies, waveID)
		}
	}
	return dependencies
}

func operationalDirectorWorkItemFromStepV0(
	step OperationalDirectorStepV0,
) OperationalDirectorWorkItemV0 {
	return OperationalDirectorWorkItemV0{
		ItemID:             operationalDirectorWorkItemIDForStepV0(step.StepID),
		SourceStepID:       step.StepID,
		Kind:               step.Kind,
		Title:              step.Title,
		WorkProfileKind:    step.WorkProfileKind,
		DependsOn:          operationalDirectorWorkItemIDsForStepsV0(step.DependsOn),
		ParentItemID:       operationalDirectorWorkItemIDForOptionalStepV0(step.ParentStepID),
		DelegationDepth:    step.DelegationDepth,
		MaxChildItems:      step.MaxChildAgents,
		ChildItemIDs:       operationalDirectorWorkItemIDsForStepsV0(step.ChildStepIDs),
		DomainRefs:         append([]string(nil), step.DomainRefs...),
		WriteSet:           append([]string(nil), step.WriteSet...),
		RequiredTests:      append([]string(nil), step.RequiredTests...),
		AcceptanceCriteria: append([]string(nil), step.AcceptanceCriteria...),
		EvidenceRefs:       append([]string(nil), step.EvidenceRefs...),
	}
}

func operationalDirectorWorkItemIDsForStepsV0(stepIDs []string) []string {
	if len(stepIDs) == 0 {
		return nil
	}
	itemIDs := make([]string, 0, len(stepIDs))
	for _, stepID := range stepIDs {
		itemIDs = append(itemIDs, operationalDirectorWorkItemIDForStepV0(stepID))
	}
	return itemIDs
}

func operationalDirectorWorkItemIDForOptionalStepV0(stepID string) string {
	if stepID == "" {
		return ""
	}
	return operationalDirectorWorkItemIDForStepV0(stepID)
}

func operationalDirectorWorkItemIDForStepV0(stepID string) string {
	return "work-item-" + stepID
}
