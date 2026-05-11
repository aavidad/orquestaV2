package orquestadirectorcandidates

import "strings"

func validatePlanInputV0(input CompactBacklogPlanCandidatesInputV0) error {
	fields := []requiredFieldV0{
		{"run_ref", input.RunRef},
		{"phase_id", input.PhaseID},
	}
	for _, field := range fields {
		if strings.TrimSpace(field.value) == "" {
			return candidateErrorV0(field.name)
		}
	}
	if len(input.WorkItems) == 0 {
		return candidateErrorV0("work_items")
	}
	return validatePlanWorkItemsV0(input.WorkItems)
}

func validatePlanWorkItemsV0(items []CompactBacklogPlanWorkItemV0) error {
	for _, item := range items {
		fields := []requiredFieldV0{
			{"work_items.candidate_ref", item.CandidateRef},
			{"work_items.task_ref", item.TaskRef},
		}
		for _, field := range fields {
			if strings.TrimSpace(field.value) == "" {
				return candidateErrorV0(field.name)
			}
		}
	}
	return nil
}
