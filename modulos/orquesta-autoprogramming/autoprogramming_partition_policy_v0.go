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
	for _, path := range writeSet {
		matches := autoprogrammingWriteSetMatchingAreasV0(path, groups, request.AreaAliases)
		switch len(matches) {
		case 0:
			for _, group := range groups {
				plan.WriteSetByArea[group.Area] = appendUniqueStringV0(plan.WriteSetByArea[group.Area], path)
			}
			plan.Repairs = append(plan.Repairs, AutoprogrammingPartitionRepairV0{
				Code:    "write_set_unassigned_shared",
				Path:    path,
				Areas:   autoprogrammingPartitionAreasV0(groups),
				Message: "ruta sin area deducible por nombre; se conserva como write-set compartido para que el Director repare si hace falta",
			})
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
			for _, path := range writeSet {
				plan.WriteSetByArea[group.Area] = appendUniqueStringV0(plan.WriteSetByArea[group.Area], path)
			}
			plan.Repairs = append(plan.Repairs, AutoprogrammingPartitionRepairV0{
				Code:          "write_set_group_empty_shared",
				Areas:         []string{group.Area},
				SuggestedArea: group.Area,
				Message:       "area sin write-set propio; se conserva write-set compartido para que el Director repare si hace falta",
			})
		}
	}
	plan = autoprogrammingApplySequencedPathDepsV0(request, plan, groups, sequenced)
	plan = autoprogrammingApplyDeclaredTaskWriteSetAndDepsV0(request, plan, groups)
	plan = autoprogrammingApplyDeclaredWriteSetOverlapDepsV0(request, plan, groups)
	return autoprogrammingCompletePartitionPlanV0(request, plan, groups), nil
}

// autoprogrammingApplyDeclaredTaskWriteSetAndDepsV0 deja que las tareas declaren
// explicitamente su write-set y sus dependencias (depends_on por task_ref). Si una
// tarea del grupo trae WriteSet/DependsOn declarados, ganan sobre la inferencia
// automatica por area. Es aditivo: tareas sin declaracion conservan lo inferido.
func autoprogrammingApplyDeclaredTaskWriteSetAndDepsV0(
	request AutoprogrammingRequestV0,
	plan AutoprogrammingPartitionPlanV0,
	groups []AutoprogrammingTaskGroupV0,
) AutoprogrammingPartitionPlanV0 {
	sourceToWorkflowTask := autoprogrammingSourceTaskRefMapV0(request, groups)
	for _, group := range groups {
		declaredWriteSet := make([]string, 0)
		declaredDeps := make([]string, 0)
		currentTaskRef := sourceToWorkflowTask["area:"+group.Area]
		for _, task := range group.Tasks {
			for _, path := range task.WriteSet {
				declaredWriteSet = appendUniqueStringV0(declaredWriteSet, path)
			}
			for _, dep := range task.DependsOn {
				dependencyRef := autoprogrammingDeclaredDependencyRefV0(sourceToWorkflowTask, dep)
				if dependencyRef == currentTaskRef {
					continue
				}
				declaredDeps = appendUniqueStringV0(declaredDeps, dependencyRef)
			}
		}
		if len(declaredWriteSet) > 0 {
			plan.WriteSetByArea[group.Area] = declaredWriteSet
		}
		if len(declaredDeps) > 0 {
			plan.DependsOnByArea[group.Area] = declaredDeps
		}
	}
	return plan
}

func autoprogrammingApplyDeclaredWriteSetOverlapDepsV0(
	request AutoprogrammingRequestV0,
	plan AutoprogrammingPartitionPlanV0,
	groups []AutoprogrammingTaskGroupV0,
) AutoprogrammingPartitionPlanV0 {
	declaredByArea := map[string]bool{}
	groupIndex := map[string]int{}
	for i, group := range groups {
		groupIndex[group.Area] = i
		for _, task := range group.Tasks {
			if len(task.WriteSet) > 0 {
				declaredByArea[group.Area] = true
				break
			}
		}
	}
	for i := range groups {
		for j := i + 1; j < len(groups); j++ {
			left := groups[i]
			right := groups[j]
			if !declaredByArea[left.Area] && !declaredByArea[right.Area] {
				continue
			}
			if !autoprogrammingWriteSetsOverlapV0(plan.WriteSetByArea[left.Area], plan.WriteSetByArea[right.Area]) {
				continue
			}
			dependencyRef := autoprogrammingProgrammableTaskRefV0(request.RequestRef, groupIndex[left.Area])
			before := len(plan.DependsOnByArea[right.Area])
			plan.DependsOnByArea[right.Area] = appendUniqueStringV0(plan.DependsOnByArea[right.Area], dependencyRef)
			if len(plan.DependsOnByArea[right.Area]) == before {
				continue
			}
			plan.Repairs = append(plan.Repairs, AutoprogrammingPartitionRepairV0{
				Code:    "declared_write_set_overlap_sequenced",
				Areas:   []string{left.Area, right.Area},
				Message: "write-set declarado compartido secuenciado para evitar ejecucion paralela sobre el mismo alcance",
			})
		}
	}
	return plan
}

func autoprogrammingSourceTaskRefMapV0(
	request AutoprogrammingRequestV0,
	groups []AutoprogrammingTaskGroupV0,
) map[string]string {
	refs := map[string]string{}
	for i, group := range groups {
		taskRef := autoprogrammingProgrammableTaskRefV0(request.RequestRef, i)
		refs["area:"+group.Area] = taskRef
		for _, sourceRef := range group.TaskRefs {
			refs[sourceRef] = taskRef
		}
		for _, task := range group.Tasks {
			refs[task.TaskRef] = taskRef
		}
	}
	return refs
}

func autoprogrammingDeclaredDependencyRefV0(
	sourceToWorkflowTask map[string]string,
	dependency string,
) string {
	if mapped := sourceToWorkflowTask[dependency]; mapped != "" {
		return mapped
	}
	return dependency
}

func autoprogrammingPartitionAreasV0(groups []AutoprogrammingTaskGroupV0) []string {
	areas := make([]string, 0, len(groups))
	for _, group := range groups {
		areas = appendUniqueStringV0(areas, group.Area)
	}
	return areas
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
