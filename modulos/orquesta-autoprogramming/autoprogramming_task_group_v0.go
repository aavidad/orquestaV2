package orquestaautoprogramming

import (
	"strings"
	"unicode"
)

type AutoprogrammingTaskGroupCandidateV0 struct {
	TaskRef            string   `json:"task_ref"`
	Area               string   `json:"area"`
	Title              string   `json:"title,omitempty"`
	Objective          string   `json:"objective,omitempty"`
	Context            []string `json:"context,omitempty"`
	ContextRefs        []string `json:"context_refs,omitempty"`
	AcceptanceCriteria []string `json:"acceptance_criteria,omitempty"`
	RequiredTests      []string `json:"required_tests,omitempty"`
	CompactRules       []string `json:"compact_rules,omitempty"`
	SkillRefs          []string `json:"skill_refs,omitempty"`
	// WriteSet y DependsOn permiten declarar por tarea el alcance de escritura y
	// las dependencias explicitas (refs de otras task_ref). Si se declaran, ganan
	// sobre la inferencia automatica por area/solapamiento. Vacios = comportamiento
	// historico (inferencia por area). Aditivo y retrocompatible.
	WriteSet  []string `json:"write_set,omitempty"`
	DependsOn []string `json:"depends_on,omitempty"`
}

type AutoprogrammingTaskGroupV0 struct {
	Area     string                                `json:"area"`
	TaskRefs []string                              `json:"task_refs"`
	Tasks    []AutoprogrammingTaskGroupCandidateV0 `json:"tasks,omitempty"`
}

func GroupAutoprogrammingTasksByAreaV0(
	tasks []AutoprogrammingTaskGroupCandidateV0,
) ([]AutoprogrammingTaskGroupV0, error) {
	if len(tasks) == 0 {
		return nil, nil
	}

	groups := make([]AutoprogrammingTaskGroupV0, 0, len(tasks))
	groupIndexByArea := map[string]int{}
	seenTaskRefs := map[string]struct{}{}

	for _, task := range tasks {
		taskRef := strings.TrimSpace(task.TaskRef)
		area := normalizeAutoprogrammingTaskAreaV0(task.Area)
		if taskRef == "" {
			return nil, errorV0(
				ErrAutoprogrammingInvalidoV0,
				"task_ref",
				"task_ref requerido",
			)
		}
		if area == "" {
			return nil, errorV0(
				ErrAutoprogrammingInvalidoV0,
				"area",
				"area requerida",
			)
		}
		if _, ok := seenTaskRefs[taskRef]; ok {
			continue
		}
		seenTaskRefs[taskRef] = struct{}{}
		normalizedTask := normalizeAutoprogrammingTaskCandidateV0(task, taskRef, area)

		groupIndex, ok := groupIndexByArea[area]
		if !ok {
			groupIndex = len(groups)
			groupIndexByArea[area] = groupIndex
			groups = append(groups, AutoprogrammingTaskGroupV0{Area: area})
		}
		groups[groupIndex].TaskRefs = append(groups[groupIndex].TaskRefs, taskRef)
		groups[groupIndex].Tasks = append(groups[groupIndex].Tasks, normalizedTask)
	}

	return groups, nil
}

func normalizeAutoprogrammingTaskCandidateV0(
	task AutoprogrammingTaskGroupCandidateV0,
	taskRef string,
	area string,
) AutoprogrammingTaskGroupCandidateV0 {
	return AutoprogrammingTaskGroupCandidateV0{
		TaskRef:            taskRef,
		Area:               area,
		Title:              strings.TrimSpace(task.Title),
		Objective:          strings.TrimSpace(task.Objective),
		Context:            compactStringsV0(task.Context),
		ContextRefs:        compactStringsV0(task.ContextRefs),
		AcceptanceCriteria: compactStringsV0(task.AcceptanceCriteria),
		RequiredTests:      compactStringsV0(task.RequiredTests),
		CompactRules:       compactStringsV0(task.CompactRules),
		SkillRefs:          compactStringsV0(task.SkillRefs),
		WriteSet:           compactStringsV0(task.WriteSet),
		DependsOn:          compactStringsV0(task.DependsOn),
	}
}

func normalizeAutoprogrammingTaskAreaV0(area string) string {
	parts := strings.FieldsFunc(strings.ToLower(strings.TrimSpace(area)), func(r rune) bool {
		return unicode.IsSpace(r) || r == '_' || r == '-'
	})
	return strings.Join(parts, "-")
}

func autoprogrammingSkillRefsForGroupV0(group AutoprogrammingTaskGroupV0) []string {
	var refs []string
	for _, task := range group.Tasks {
		refs = append(refs, task.SkillRefs...)
	}
	return compactStringsV0(refs)
}
