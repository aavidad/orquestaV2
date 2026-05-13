package orquestaautoprogramming

import (
	"strings"
	"unicode"
)

type AutoprogrammingTaskGroupCandidateV0 struct {
	TaskRef string `json:"task_ref"`
	Area    string `json:"area"`
}

type AutoprogrammingTaskGroupV0 struct {
	Area     string   `json:"area"`
	TaskRefs []string `json:"task_refs"`
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

		groupIndex, ok := groupIndexByArea[area]
		if !ok {
			groupIndex = len(groups)
			groupIndexByArea[area] = groupIndex
			groups = append(groups, AutoprogrammingTaskGroupV0{Area: area})
		}
		groups[groupIndex].TaskRefs = append(groups[groupIndex].TaskRefs, taskRef)
	}

	return groups, nil
}

func normalizeAutoprogrammingTaskAreaV0(area string) string {
	parts := strings.FieldsFunc(strings.ToLower(strings.TrimSpace(area)), func(r rune) bool {
		return unicode.IsSpace(r) || r == '_' || r == '-'
	})
	return strings.Join(parts, "-")
}
