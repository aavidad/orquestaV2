package orquestaautoprogramming

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"unicode"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

type AutoprogrammingProgrammableWorkResultV0 struct {
	Accepted bool                              `json:"accepted"`
	Work     AutoprogrammingProgrammableWorkV0 `json:"work"`
	Issues   []AutoprogrammingRequestIssueV0   `json:"issues,omitempty"`
}

type AutoprogrammingProgrammableWorkV0 struct {
	RequestRef  string                                `json:"request_ref"`
	ProjectRef  string                                `json:"project_ref"`
	WorktreeRef string                                `json:"worktree_ref"`
	BranchRef   string                                `json:"branch_ref"`
	Groups      []AutoprogrammingProgrammableGroupV0  `json:"groups"`
	Profiles    []orquestacoreworkflow.WorkProfileV0  `json:"profiles"`
	Tasks       []orquestacoreworkflow.WorkflowTaskV0 `json:"tasks"`
}

type AutoprogrammingProgrammableGroupV0 struct {
	Area          string                              `json:"area"`
	TaskRefs      []string                            `json:"task_refs"`
	WriteSet      []string                            `json:"write_set"`
	RequiredTests []string                            `json:"required_tests"`
	Profile       orquestacoreworkflow.WorkProfileV0  `json:"profile"`
	Task          orquestacoreworkflow.WorkflowTaskV0 `json:"task"`
}

func BuildAutoprogrammingProgrammableWorkV0(
	request AutoprogrammingRequestV0,
) AutoprogrammingProgrammableWorkResultV0 {
	validation := ValidateAutoprogrammingRequestV0(request)
	if !validation.Accepted {
		return AutoprogrammingProgrammableWorkResultV0{
			Accepted: false,
			Work:     autoprogrammingProgrammableWorkSkeletonV0(request),
			Issues:   append([]AutoprogrammingRequestIssueV0(nil), validation.Issues...),
		}
	}

	writeSetByArea, partitionIssues := autoprogrammingPartitionWriteSetByAreaV0(
		validation.WriteSet,
		validation.Groups,
	)
	if len(partitionIssues) > 0 {
		return AutoprogrammingProgrammableWorkResultV0{
			Accepted: false,
			Work:     autoprogrammingProgrammableWorkSkeletonV0(request),
			Issues:   partitionIssues,
		}
	}

	work := autoprogrammingProgrammableWorkSkeletonV0(request)
	for i, group := range validation.Groups {
		profile, task, issue := autoprogrammingWorkflowTaskForGroupV0(
			request,
			group,
			writeSetByArea[group.Area],
			validation.RequiredTests,
			i,
		)
		if issue.Code != "" {
			return AutoprogrammingProgrammableWorkResultV0{
				Accepted: false,
				Work:     work,
				Issues:   []AutoprogrammingRequestIssueV0{issue},
			}
		}
		workGroup := AutoprogrammingProgrammableGroupV0{
			Area:          group.Area,
			TaskRefs:      append([]string(nil), group.TaskRefs...),
			WriteSet:      append([]string(nil), task.WriteSet...),
			RequiredTests: append([]string(nil), task.RequiredTests...),
			Profile:       profile,
			Task:          task,
		}
		work.Groups = append(work.Groups, workGroup)
		work.Profiles = append(work.Profiles, profile)
		work.Tasks = append(work.Tasks, task)
	}

	return AutoprogrammingProgrammableWorkResultV0{
		Accepted: true,
		Work:     work,
	}
}

func autoprogrammingProgrammableWorkSkeletonV0(
	request AutoprogrammingRequestV0,
) AutoprogrammingProgrammableWorkV0 {
	return AutoprogrammingProgrammableWorkV0{
		RequestRef:  strings.TrimSpace(request.RequestRef),
		ProjectRef:  strings.TrimSpace(request.ProjectRef),
		WorktreeRef: strings.TrimSpace(request.WorktreeRef),
		BranchRef:   strings.TrimSpace(request.BranchRef),
	}
}

func autoprogrammingWorkflowTaskForGroupV0(
	request AutoprogrammingRequestV0,
	group AutoprogrammingTaskGroupV0,
	writeSet []string,
	requiredTests []string,
	groupIndex int,
) (orquestacoreworkflow.WorkProfileV0, orquestacoreworkflow.WorkflowTaskV0, AutoprogrammingRequestIssueV0) {
	taskRef := autoprogrammingProgrammableTaskRefV0(request.RequestRef, groupIndex)
	profile, err := orquestacoreworkflow.NewWorkProfileV0(orquestacoreworkflow.WorkProfileV0{
		SchemaVersion:      orquestacoreworkflow.WorkProfileSchemaVersionV0,
		ProfileRef:         "profile-" + taskRef,
		ProfileKind:        orquestacoreworkflow.WorkProfileImplementationV0,
		TaskRef:            taskRef,
		RunRef:             strings.TrimSpace(request.RequestRef),
		Title:              fmt.Sprintf("Cambio acotado %02d", groupIndex+1),
		Objective:          "Resolver refs de tarea asignadas dentro del write-set validado.",
		Summary:            "Autoprogramacion acotada con worktree aislada y rama opaca preservada.",
		ScopeRefs:          append([]string(nil), writeSet...),
		RequiredTests:      append([]string(nil), requiredTests...),
		AcceptanceCriteria: autoprogrammingAcceptanceCriteriaForGroupV0(group),
		FunctionContractRefs: []orquestacoreworkflow.WorkflowFunctionContractRefV0{
			{FunctionName: "ValidateAutoprogrammingRequestV0"},
			{FunctionName: "BuildAutoprogrammingProgrammableWorkV0"},
		},
	})
	if err != nil {
		return orquestacoreworkflow.WorkProfileV0{}, orquestacoreworkflow.WorkflowTaskV0{},
			autoprogrammingRequestIssueV0("work_profile_invalid", "work_profile", err.Error())
	}

	task, err := orquestacoreworkflow.WorkflowTaskFromWorkProfileV0(profile)
	if err != nil {
		return orquestacoreworkflow.WorkProfileV0{}, orquestacoreworkflow.WorkflowTaskV0{},
			autoprogrammingRequestIssueV0("workflow_task_invalid", "workflow_task", err.Error())
	}
	task.ContextRefs = autoprogrammingProgrammableContextRefsV0(request)
	task, err = orquestacoreworkflow.NewWorkflowTaskV0(task)
	if err != nil {
		return orquestacoreworkflow.WorkProfileV0{}, orquestacoreworkflow.WorkflowTaskV0{},
			autoprogrammingRequestIssueV0("workflow_task_invalid", "workflow_task", err.Error())
	}
	return profile, task, AutoprogrammingRequestIssueV0{}
}

func autoprogrammingPartitionWriteSetByAreaV0(
	writeSet []string,
	groups []AutoprogrammingTaskGroupV0,
) (map[string][]string, []AutoprogrammingRequestIssueV0) {
	out := map[string][]string{}
	if len(groups) == 1 {
		out[groups[0].Area] = append([]string(nil), writeSet...)
		return out, nil
	}

	var issues []AutoprogrammingRequestIssueV0
	seenPath := map[string]struct{}{}
	for _, path := range writeSet {
		matches := autoprogrammingWriteSetMatchingAreasV0(path, groups)
		switch len(matches) {
		case 0:
			issues = append(issues, autoprogrammingRequestIssueV0(
				"write_set_unassigned",
				"write_set",
				"ruta sin area unica: "+path,
			))
		case 1:
			if _, ok := seenPath[path]; ok {
				continue
			}
			seenPath[path] = struct{}{}
			out[matches[0]] = append(out[matches[0]], path)
		default:
			issues = append(issues, autoprogrammingRequestIssueV0(
				"write_set_overlap",
				"write_set",
				"ruta coincide con varias areas: "+path,
			))
		}
	}
	for _, group := range groups {
		if len(out[group.Area]) == 0 {
			issues = append(issues, autoprogrammingRequestIssueV0(
				"write_set_group_empty",
				"write_set",
				"area sin write-set: "+group.Area,
			))
		}
	}
	if len(issues) > 0 {
		return nil, issues
	}
	return out, nil
}

func autoprogrammingWriteSetMatchingAreasV0(
	path string,
	groups []AutoprogrammingTaskGroupV0,
) []string {
	normalizedPath := normalizeAutoprogrammingPathForMatchV0(path)
	pathTokens := autoprogrammingNormalizedPathTokensV0(normalizedPath)
	var matches []string
	for _, group := range groups {
		area := normalizeAutoprogrammingPathForMatchV0(group.Area)
		if autoprogrammingPathTokensContainAreaV0(pathTokens, autoprogrammingNormalizedPathTokensV0(area)) {
			matches = append(matches, group.Area)
		}
	}
	return matches
}

func autoprogrammingNormalizedPathTokensV0(value string) []string {
	return compactStringsV0(strings.Split(value, "-"))
}

func autoprogrammingPathTokensContainAreaV0(
	pathTokens []string,
	areaTokens []string,
) bool {
	if len(pathTokens) == 0 || len(areaTokens) == 0 || len(areaTokens) > len(pathTokens) {
		return false
	}
	for start := 0; start <= len(pathTokens)-len(areaTokens); start++ {
		matched := true
		for offset := range areaTokens {
			if pathTokens[start+offset] != areaTokens[offset] {
				matched = false
				break
			}
		}
		if matched {
			return true
		}
	}
	return false
}

func normalizeAutoprogrammingPathForMatchV0(value string) string {
	return strings.Trim(strings.Map(func(r rune) rune {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			return unicode.ToLower(r)
		default:
			return '-'
		}
	}, value), "-")
}

func autoprogrammingProgrammableTaskRefV0(requestRef string, groupIndex int) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(requestRef)))
	return fmt.Sprintf("task-autoprogramming-%s-g%02d", hex.EncodeToString(sum[:])[:12], groupIndex+1)
}

func autoprogrammingProgrammableContextRefsV0(
	request AutoprogrammingRequestV0,
) []string {
	return []string{
		"request_ref:" + strings.TrimSpace(request.RequestRef),
		"project_ref:" + strings.TrimSpace(request.ProjectRef),
		"worktree_ref:" + strings.TrimSpace(request.WorktreeRef),
		"branch_ref:" + strings.TrimSpace(request.BranchRef),
	}
}

func autoprogrammingAcceptanceCriteriaForGroupV0(
	group AutoprogrammingTaskGroupV0,
) []string {
	return []string{
		"Cambios limitados al write-set asignado al grupo.",
		"ACK declara pruebas requeridas ejecutadas y resultado.",
		"Refs de tarea origen preservadas: " + strings.Join(group.TaskRefs, ","),
	}
}
