package orquestaappplanner

type AppPlanProgressV0 struct {
	TotalUnits        int      `json:"total_units"`
	DeliveredUnits    int      `json:"delivered_units"`
	Complete          bool     `json:"complete"`
	DeliveredTaskRefs []string `json:"delivered_task_refs,omitempty"`
	ReadyTaskRefs     []string `json:"ready_task_refs,omitempty"`
	BlockedTaskRefs   []string `json:"blocked_task_refs,omitempty"`
	PendingTaskRefs   []string `json:"pending_task_refs,omitempty"`
}

func EvaluateAppPlanProgressV0(
	plan AppMicrotaskPlanV0,
	deliveries []string,
) (AppPlanProgressV0, error) {
	if err := validateAppMicrotaskPlanV0(plan); err != nil {
		return AppPlanProgressV0{}, err
	}
	delivered := appPlanDeliverySetV0(deliveries)
	progress := AppPlanProgressV0{TotalUnits: len(plan.Units)}
	for _, unit := range plan.Units {
		if delivered[unit.DeliveryRef] {
			progress.DeliveredUnits++
			progress.DeliveredTaskRefs = append(progress.DeliveredTaskRefs, unit.TaskRef)
			continue
		}
		progress.PendingTaskRefs = append(progress.PendingTaskRefs, unit.TaskRef)
		if unitDependenciesDeliveredV0(unit, deliveries) {
			progress.ReadyTaskRefs = append(progress.ReadyTaskRefs, unit.TaskRef)
		} else {
			progress.BlockedTaskRefs = append(progress.BlockedTaskRefs, unit.TaskRef)
		}
	}
	progress.Complete = progress.DeliveredUnits == progress.TotalUnits
	return progress, nil
}

func appPlanDeliverySetV0(deliveries []string) map[string]bool {
	out := map[string]bool{}
	for _, ref := range compactAppPlannerStringsV0(deliveries) {
		out[ref] = true
	}
	return out
}
