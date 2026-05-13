package orquestaautoprogramming

import (
	"errors"
	"strings"
)

const (
	AutoprogrammingRequestDefaultMaxTaskRefsV0        = 3
	AutoprogrammingRequestDefaultMaxAreasV0           = 2
	AutoprogrammingRequestDefaultMaxWriteSetEntriesV0 = 5
)

type AutoprogrammingRequestV0 struct {
	RequestRef       string                                `json:"request_ref"`
	ProjectRef       string                                `json:"project_ref"`
	WorktreeRef      string                                `json:"worktree_ref"`
	WorktreeIsolated bool                                  `json:"worktree_isolated"`
	BranchRef        string                                `json:"branch_ref"`
	Tasks            []AutoprogrammingTaskGroupCandidateV0 `json:"tasks"`
	WriteSet         []string                              `json:"write_set"`
	RequiredTests    []string                              `json:"required_tests"`

	MaxTaskRefs        int `json:"max_task_refs,omitempty"`
	MaxAreas           int `json:"max_areas,omitempty"`
	MaxWriteSetEntries int `json:"max_write_set_entries,omitempty"`
}

type AutoprogrammingRequestValidationResultV0 struct {
	Accepted      bool
	Groups        []AutoprogrammingTaskGroupV0
	WriteSet      []string
	RequiredTests []string
	Issues        []AutoprogrammingRequestIssueV0
}

type AutoprogrammingRequestIssueV0 struct {
	Code    string
	Field   string
	Message string
}

func ValidateAutoprogrammingRequestV0(
	request AutoprogrammingRequestV0,
) AutoprogrammingRequestValidationResultV0 {
	var issues []AutoprogrammingRequestIssueV0
	issues = append(issues, autoprogrammingRequestRequiredRefIssuesV0(request)...)

	groups, groupIssues := autoprogrammingRequestGroupsV0(request.Tasks)
	issues = append(issues, groupIssues...)
	issues = append(issues, autoprogrammingRequestScopeIssuesV0(request, groups)...)

	writeSet := compactStringsV0(request.WriteSet)
	issues = append(issues, autoprogrammingRequestWriteSetIssuesV0(request, writeSet)...)

	requiredTests := compactStringsV0(request.RequiredTests)
	if len(requiredTests) == 0 {
		issues = append(issues, autoprogrammingRequestIssueV0(
			"required_tests_missing",
			"required_tests",
			"tests obligatorios requeridos",
		))
	}

	return AutoprogrammingRequestValidationResultV0{
		Accepted:      len(issues) == 0,
		Groups:        groups,
		WriteSet:      writeSet,
		RequiredTests: requiredTests,
		Issues:        issues,
	}
}

func autoprogrammingRequestRequiredRefIssuesV0(
	request AutoprogrammingRequestV0,
) []AutoprogrammingRequestIssueV0 {
	required := []struct {
		field string
		value string
	}{
		{field: "request_ref", value: request.RequestRef},
		{field: "project_ref", value: request.ProjectRef},
		{field: "worktree_ref", value: request.WorktreeRef},
		{field: "branch_ref", value: request.BranchRef},
	}
	var issues []AutoprogrammingRequestIssueV0
	for _, item := range required {
		if strings.TrimSpace(item.value) == "" {
			issues = append(issues, autoprogrammingRequestIssueV0(
				item.field+"_missing",
				item.field,
				item.field+" requerido",
			))
		}
	}
	if !request.WorktreeIsolated {
		issues = append(issues, autoprogrammingRequestIssueV0(
			"worktree_not_isolated",
			"worktree_isolated",
			"worktree aislada requerida",
		))
	}
	return issues
}

func autoprogrammingRequestGroupsV0(
	tasks []AutoprogrammingTaskGroupCandidateV0,
) ([]AutoprogrammingTaskGroupV0, []AutoprogrammingRequestIssueV0) {
	if len(tasks) == 0 {
		return nil, []AutoprogrammingRequestIssueV0{autoprogrammingRequestIssueV0(
			"tasks_missing",
			"tasks",
			"tareas requeridas",
		)}
	}
	groups, err := GroupAutoprogrammingTasksByAreaV0(tasks)
	if err == nil {
		return groups, nil
	}
	return nil, []AutoprogrammingRequestIssueV0{autoprogrammingRequestIssueFromErrorV0(err)}
}

func autoprogrammingRequestScopeIssuesV0(
	request AutoprogrammingRequestV0,
	groups []AutoprogrammingTaskGroupV0,
) []AutoprogrammingRequestIssueV0 {
	var issues []AutoprogrammingRequestIssueV0
	if len(groups) > autoprogrammingRequestMaxAreasV0(request) {
		issues = append(issues, autoprogrammingRequestIssueV0(
			"scope_areas_too_large",
			"tasks.area",
			"demasiadas areas para autoprogramacion",
		))
	}
	if len(autoprogrammingRequestTaskRefsV0(groups)) > autoprogrammingRequestMaxTaskRefsV0(request) {
		issues = append(issues, autoprogrammingRequestIssueV0(
			"scope_tasks_too_large",
			"tasks.task_ref",
			"demasiadas tareas para autoprogramacion",
		))
	}
	return issues
}

func autoprogrammingRequestWriteSetIssuesV0(
	request AutoprogrammingRequestV0,
	writeSet []string,
) []AutoprogrammingRequestIssueV0 {
	if len(writeSet) == 0 {
		return []AutoprogrammingRequestIssueV0{autoprogrammingRequestIssueV0(
			"write_set_missing",
			"write_set",
			"write-set requerido",
		)}
	}
	var issues []AutoprogrammingRequestIssueV0
	if len(writeSet) > autoprogrammingRequestMaxWriteSetEntriesV0(request) {
		issues = append(issues, autoprogrammingRequestIssueV0(
			"scope_write_set_too_large",
			"write_set",
			"write-set demasiado amplio",
		))
	}
	for _, path := range writeSet {
		if autoprogrammingRequestWriteSetPathAllowedV0(path) {
			continue
		}
		issues = append(issues, autoprogrammingRequestIssueV0(
			"write_set_path_invalid",
			"write_set",
			"ruta de write-set no permitida: "+path,
		))
	}
	return issues
}

func autoprogrammingRequestIssueFromErrorV0(err error) AutoprogrammingRequestIssueV0 {
	var publicErr ErrorV0
	if errors.As(err, &publicErr) {
		return autoprogrammingRequestIssueV0(
			"task_group_invalid",
			publicErr.Field,
			publicErr.Message,
		)
	}
	return autoprogrammingRequestIssueV0("task_group_invalid", "tasks", err.Error())
}

func autoprogrammingRequestTaskRefsV0(groups []AutoprogrammingTaskGroupV0) []string {
	var refs []string
	for _, group := range groups {
		refs = append(refs, group.TaskRefs...)
	}
	return compactStringsV0(refs)
}

func autoprogrammingRequestWriteSetPathAllowedV0(path string) bool {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" || strings.HasPrefix(trimmed, "/") || strings.HasPrefix(trimmed, "\\") {
		return false
	}
	if len(trimmed) >= 2 && trimmed[1] == ':' {
		return false
	}
	segments := strings.Split(strings.ReplaceAll(trimmed, "\\", "/"), "/")
	for _, segment := range segments {
		if segment == "" || segment == "." || segment == ".." {
			return false
		}
	}
	return true
}

func autoprogrammingRequestMaxTaskRefsV0(request AutoprogrammingRequestV0) int {
	if request.MaxTaskRefs > 0 {
		return request.MaxTaskRefs
	}
	return AutoprogrammingRequestDefaultMaxTaskRefsV0
}

func autoprogrammingRequestMaxAreasV0(request AutoprogrammingRequestV0) int {
	if request.MaxAreas > 0 {
		return request.MaxAreas
	}
	return AutoprogrammingRequestDefaultMaxAreasV0
}

func autoprogrammingRequestMaxWriteSetEntriesV0(request AutoprogrammingRequestV0) int {
	if request.MaxWriteSetEntries > 0 {
		return request.MaxWriteSetEntries
	}
	return AutoprogrammingRequestDefaultMaxWriteSetEntriesV0
}

func autoprogrammingRequestIssueV0(
	code string,
	field string,
	message string,
) AutoprogrammingRequestIssueV0 {
	return AutoprogrammingRequestIssueV0{
		Code:    code,
		Field:   field,
		Message: message,
	}
}
