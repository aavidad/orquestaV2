package orquestaappplanner

import "strings"

func FindAppPlanUnitByTaskRefV0(plan AppMicrotaskPlanV0, taskRef string) (AppWorkUnitV0, bool) {
	unit, ok := findAppPlanUnitByTaskRefV0(plan, taskRef)
	return unit, ok
}

func FindAppPlanUnitByCapacityRequestRefV0(plan AppMicrotaskPlanV0, capacityRef string) (AppWorkUnitV0, bool) {
	capacityRef = strings.TrimSpace(capacityRef)
	for _, unit := range plan.Units {
		if capacityRefV0(unit) == capacityRef {
			return unit, true
		}
	}
	return AppWorkUnitV0{}, false
}

func findAppPlanUnitByTaskRefV0(plan AppMicrotaskPlanV0, taskRef string) (AppWorkUnitV0, bool) {
	taskRef = strings.TrimSpace(taskRef)
	for _, unit := range plan.Units {
		if unit.TaskRef == taskRef {
			return unit, true
		}
	}
	return AppWorkUnitV0{}, false
}
