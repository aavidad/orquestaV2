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
