package orquestaautoprogramming

import (
	"strings"
	"unicode"
)

type AutoprogrammingTaskGroupCandidateV0 struct {
	TaskRef            string                             `json:"task_ref"`
	Area               string                             `json:"area"`
	Title              string                             `json:"title,omitempty"`
	Objective          string                             `json:"objective,omitempty"`
	Context            []string                           `json:"context,omitempty"`
	ContextRefs        []string                           `json:"context_refs,omitempty"`
	AcceptanceCriteria []string                           `json:"acceptance_criteria,omitempty"`
	AcceptanceChecks   []AutoprogrammingAcceptanceCheckV0 `json:"acceptance_checks,omitempty"`
	RequiredTests      []string                           `json:"required_tests,omitempty"`
	CompactRules       []string                           `json:"compact_rules,omitempty"`
	SkillRefs          []string                           `json:"skill_refs,omitempty"`
	// WriteSet y DependsOn permiten declarar por tarea el alcance de escritura y
	// las dependencias explicitas (refs de otras task_ref). Si se declaran, ganan
	// sobre la inferencia automatica por area/solapamiento. Vacios = comportamiento
	// historico (inferencia por area). Aditivo y retrocompatible.
	WriteSet  []string `json:"write_set,omitempty"`
	DependsOn []string `json:"depends_on,omitempty"`
}

type AutoprogrammingAcceptanceCheckV0 struct {
	CriterionRef string `json:"criterion_ref"`
	Description  string `json:"description,omitempty"`
	Command      string `json:"command"`
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
	acceptanceCheckRefsByArea := map[string]map[string]struct{}{}

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
		normalizedTask, err := normalizeAutoprogrammingTaskCandidateV0(task, taskRef, area)
		if err != nil {
			return nil, err
		}
		if _, ok := seenTaskRefs[taskRef]; ok {
			continue
		}
		seenTaskRefs[taskRef] = struct{}{}

		groupIndex, ok := groupIndexByArea[area]
		if !ok {
			groupIndex = len(groups)
			groupIndexByArea[area] = groupIndex
			groups = append(groups, AutoprogrammingTaskGroupV0{Area: area})
			acceptanceCheckRefsByArea[area] = map[string]struct{}{}
		}
		for _, check := range normalizedTask.AcceptanceChecks {
			if _, ok := acceptanceCheckRefsByArea[area][check.CriterionRef]; ok {
				return nil, errorV0(
					ErrAutoprogrammingInvalidoV0,
					"acceptance_checks.criterion_ref",
					"criterion_ref duplicado en grupo",
				)
			}
			acceptanceCheckRefsByArea[area][check.CriterionRef] = struct{}{}
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
) (AutoprogrammingTaskGroupCandidateV0, error) {
	checks, err := normalizeAutoprogrammingAcceptanceChecksV0(task.AcceptanceChecks)
	if err != nil {
		return AutoprogrammingTaskGroupCandidateV0{}, err
	}
	return AutoprogrammingTaskGroupCandidateV0{
		TaskRef:            taskRef,
		Area:               area,
		Title:              strings.TrimSpace(task.Title),
		Objective:          strings.TrimSpace(task.Objective),
		Context:            compactStringsV0(task.Context),
		ContextRefs:        compactStringsV0(task.ContextRefs),
		AcceptanceCriteria: compactStringsV0(task.AcceptanceCriteria),
		AcceptanceChecks:   checks,
		RequiredTests:      compactStringsV0(task.RequiredTests),
		CompactRules:       compactStringsV0(task.CompactRules),
		SkillRefs:          compactStringsV0(task.SkillRefs),
		WriteSet:           compactStringsV0(task.WriteSet),
		DependsOn:          compactStringsV0(task.DependsOn),
	}, nil
}

func normalizeAutoprogrammingAcceptanceChecksV0(
	checks []AutoprogrammingAcceptanceCheckV0,
) ([]AutoprogrammingAcceptanceCheckV0, error) {
	out := make([]AutoprogrammingAcceptanceCheckV0, 0, len(checks))
	seenCriterionRefs := map[string]struct{}{}
	for _, check := range checks {
		check.CriterionRef = strings.TrimSpace(check.CriterionRef)
		check.Description = strings.TrimSpace(check.Description)
		check.Command = strings.TrimSpace(check.Command)
		if check.CriterionRef == "" {
			return nil, errorV0(ErrAutoprogrammingInvalidoV0, "acceptance_checks.criterion_ref", "criterion_ref requerido")
		}
		if check.Command == "" {
			return nil, errorV0(ErrAutoprogrammingInvalidoV0, "acceptance_checks.command", "command requerido")
		}
		if _, ok := seenCriterionRefs[check.CriterionRef]; ok {
			return nil, errorV0(ErrAutoprogrammingInvalidoV0, "acceptance_checks.criterion_ref", "criterion_ref duplicado en tarea")
		}
		seenCriterionRefs[check.CriterionRef] = struct{}{}
		out = append(out, check)
	}
	if out == nil {
		return []AutoprogrammingAcceptanceCheckV0{}, nil
	}
	return out, nil
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
