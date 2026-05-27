package orquestaautoprogramming

import "fmt"

func autoprogrammingCompletePartitionPlanV0(
	request AutoprogrammingRequestV0,
	plan AutoprogrammingPartitionPlanV0,
	groups []AutoprogrammingTaskGroupV0,
) AutoprogrammingPartitionPlanV0 {
	plan = autoprogrammingApplyLiveWorkDepsV0(request, plan, groups)
	for i, group := range groups {
		deps := plan.DependsOnByArea[group.Area]
		blocked := plan.BlockedByArea[group.Area]
		status := "ready"
		if len(blocked) > 0 {
			status = "postponed"
		} else if len(deps) > 0 {
			status = "sequenced"
		}
		plan.Steps = append(plan.Steps, AutoprogrammingPartitionStepV0{
			StepRef:              fmt.Sprintf("partition-step-%02d", i+1),
			Area:                 group.Area,
			TaskRef:              autoprogrammingProgrammableTaskRefV0(request.RequestRef, i),
			SourceTaskRefs:       append([]string(nil), group.TaskRefs...),
			WriteSet:             append([]string(nil), plan.WriteSetByArea[group.Area]...),
			DependsOn:            append([]string(nil), deps...),
			BlockedByLiveWorkRef: append([]string(nil), blocked...),
			Status:               status,
		})
	}
	return plan
}

func autoprogrammingApplyLiveWorkDepsV0(
	request AutoprogrammingRequestV0,
	plan AutoprogrammingPartitionPlanV0,
	groups []AutoprogrammingTaskGroupV0,
) AutoprogrammingPartitionPlanV0 {
	for _, live := range request.LiveWorks {
		if !autoprogrammingLiveWorkIsActiveV0(live) {
			continue
		}
		liveRef := autoprogrammingLiveWorkDependencyRefV0(live)
		for _, group := range groups {
			if !autoprogrammingWriteSetsOverlapV0(plan.WriteSetByArea[group.Area], live.WriteSet) {
				continue
			}
			plan.DependsOnByArea[group.Area] = appendUniqueStringV0(plan.DependsOnByArea[group.Area], liveRef)
			plan.BlockedByArea[group.Area] = appendUniqueStringV0(plan.BlockedByArea[group.Area], liveRef)
		}
	}
	return plan
}
