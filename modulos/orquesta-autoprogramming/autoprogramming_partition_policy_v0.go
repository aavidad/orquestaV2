package orquestaautoprogramming

type AutoprogrammingAreaAliasV0 struct {
	Alias string `json:"alias"`
	Area  string `json:"area"`
}

type AutoprogrammingLiveWorkV0 struct {
	WorkRef  string   `json:"work_ref,omitempty"`
	TaskRef  string   `json:"task_ref,omitempty"`
	AgentRef string   `json:"agent_ref,omitempty"`
	Status   string   `json:"status,omitempty"`
	WriteSet []string `json:"write_set,omitempty"`
}

type AutoprogrammingPartitionPlanV0 struct {
	WriteSetByArea  map[string][]string                `json:"write_set_by_area,omitempty"`
	DependsOnByArea map[string][]string                `json:"depends_on_by_area,omitempty"`
	BlockedByArea   map[string][]string                `json:"blocked_by_area,omitempty"`
	Steps           []AutoprogrammingPartitionStepV0   `json:"steps,omitempty"`
	Repairs         []AutoprogrammingPartitionRepairV0 `json:"repairs,omitempty"`
}

type AutoprogrammingPartitionStepV0 struct {
	StepRef              string   `json:"step_ref"`
	Area                 string   `json:"area"`
	TaskRef              string   `json:"task_ref"`
	SourceTaskRefs       []string `json:"source_task_refs,omitempty"`
	WriteSet             []string `json:"write_set,omitempty"`
	DependsOn            []string `json:"depends_on,omitempty"`
	BlockedByLiveWorkRef []string `json:"blocked_by_live_work_ref,omitempty"`
	Status               string   `json:"status"`
}

type AutoprogrammingPartitionRepairV0 struct {
	Code          string   `json:"code"`
	Path          string   `json:"path,omitempty"`
	Areas         []string `json:"areas,omitempty"`
	SuggestedArea string   `json:"suggested_area,omitempty"`
	Message       string   `json:"message"`
}

func autoprogrammingPartitionWriteSetByAreaV0(
	request AutoprogrammingRequestV0,
	writeSet []string,
	groups []AutoprogrammingTaskGroupV0,
) (AutoprogrammingPartitionPlanV0, []AutoprogrammingRequestIssueV0) {
	plan := AutoprogrammingPartitionPlanV0{
		WriteSetByArea:  map[string][]string{},
		DependsOnByArea: map[string][]string{},
		BlockedByArea:   map[string][]string{},
	}
	if len(groups) == 1 {
		plan.WriteSetByArea[groups[0].Area] = append([]string(nil), writeSet...)
		return autoprogrammingCompletePartitionPlanV0(request, plan, groups), nil
	}

	sequenced := map[string][]string{}
	var issues []AutoprogrammingRequestIssueV0
	for _, path := range writeSet {
		matches := autoprogrammingWriteSetMatchingAreasV0(path, groups, request.AreaAliases)
		switch len(matches) {
		case 0:
			issues = append(issues, autoprogrammingRequestIssueV0(
				"write_set_unassigned",
				"write_set",
				"ruta sin area reparable: "+path,
			))
		case 1:
			plan.WriteSetByArea[matches[0]] = appendUniqueStringV0(plan.WriteSetByArea[matches[0]], path)
		default:
			for _, area := range matches {
				plan.WriteSetByArea[area] = appendUniqueStringV0(plan.WriteSetByArea[area], path)
			}
			sequenced[path] = matches
			plan.Repairs = append(plan.Repairs, AutoprogrammingPartitionRepairV0{
				Code:    "write_set_overlap_sequenced",
				Path:    path,
				Areas:   append([]string(nil), matches...),
				Message: "ruta compartida secuenciada entre areas compatibles",
			})
		}
	}
	for _, group := range groups {
		if len(plan.WriteSetByArea[group.Area]) == 0 {
			issues = append(issues, autoprogrammingRequestIssueV0(
				"write_set_group_empty",
				"write_set",
				"area sin write-set: "+group.Area,
			))
		}
	}
	if len(issues) > 0 {
		return AutoprogrammingPartitionPlanV0{}, issues
	}
	plan = autoprogrammingApplySequencedPathDepsV0(request, plan, groups, sequenced)
	return autoprogrammingCompletePartitionPlanV0(request, plan, groups), nil
}

func autoprogrammingApplySequencedPathDepsV0(
	request AutoprogrammingRequestV0,
	plan AutoprogrammingPartitionPlanV0,
	groups []AutoprogrammingTaskGroupV0,
	sequenced map[string][]string,
) AutoprogrammingPartitionPlanV0 {
	groupIndex := map[string]int{}
	for i, group := range groups {
		groupIndex[group.Area] = i
	}
	for _, areas := range sequenced {
		for i := 1; i < len(areas); i++ {
			prev := autoprogrammingProgrammableTaskRefV0(request.RequestRef, groupIndex[areas[i-1]])
			plan.DependsOnByArea[areas[i]] = appendUniqueStringV0(plan.DependsOnByArea[areas[i]], prev)
		}
	}
	return plan
}
