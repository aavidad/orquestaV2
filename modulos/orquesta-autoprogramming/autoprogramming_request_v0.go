package orquestaautoprogramming

import (
	"errors"
	"path/filepath"
	"strings"
)

const (
	AutoprogrammingRequestDefaultMaxTaskRefsV0        = 10
	AutoprogrammingRequestDefaultMaxAreasV0           = 10
	AutoprogrammingRequestDefaultMaxWriteSetEntriesV0 = 10
	AutoprogrammingRequestMaxDelegationDepthV0        = 3
	AutoprogrammingRequestMaxSubagentsPerAgentV0      = 6
	AutoprogrammingRequestMaxRecursiveAgentsV0        = 4096
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
	AreaAliases      []AutoprogrammingAreaAliasV0          `json:"area_aliases,omitempty"`
	LiveWorks        []AutoprogrammingLiveWorkV0           `json:"live_works,omitempty"`
	BacklogScan      AutoprogrammingBacklogScanV0          `json:"backlog_scan,omitempty"`

	MaxTaskRefs          int `json:"max_task_refs,omitempty"`
	MaxAreas             int `json:"max_areas,omitempty"`
	MaxWriteSetEntries   int `json:"max_write_set_entries,omitempty"`
	MaxDelegationDepth   int `json:"max_delegation_depth,omitempty"`
	MaxSubagentsPerAgent int `json:"max_subagents_per_agent,omitempty"`
	MaxRecursiveAgents   int `json:"max_recursive_agents,omitempty"`
}

type AutoprogrammingBacklogScanV0 struct {
	Epoch           string                             `json:"epoch,omitempty"`
	Documents       []AutoprogrammingBacklogDocumentV0 `json:"documents,omitempty"`
	ReservationRefs []string                           `json:"reservation_refs,omitempty"`
}

type AutoprogrammingBacklogDocumentV0 struct {
	Path       string `json:"path"`
	StartLine  int    `json:"start_line,omitempty"`
	SHA256     string `json:"sha256"`
	Missing    bool   `json:"missing,omitempty"`
	SectionRef string `json:"section_ref,omitempty"`
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
	issues = append(issues, autoprogrammingRequestLimitContractIssuesV0(request)...)
	issues = append(issues, autoprogrammingRequestScopeIssuesV0(request, groups)...)

	writeSet := compactStringsV0(request.WriteSet)
	issues = append(issues, autoprogrammingRequestWriteSetIssuesV0(request, writeSet)...)
	issues = append(issues, autoprogrammingRequestLiveWorkIssuesV0(request.LiveWorks)...)
	issues = append(issues, autoprogrammingRequestBacklogScanIssuesV0(request.BacklogScan)...)
	issues = append(issues, autoprogrammingRequestDelegationIssuesV0(request)...)

	requiredTests := compactStringsV0(request.RequiredTests)
	if len(requiredTests) == 0 {
		issues = append(issues, autoprogrammingRequestIssueV0(
			"required_tests_missing",
			"required_tests",
			"tests obligatorios requeridos",
		))
	}
	issues = append(issues, autoprogrammingRequestMultiGoalFocalTestIssuesV0(request, groups)...)

	return AutoprogrammingRequestValidationResultV0{
		Accepted:      len(issues) == 0,
		Groups:        groups,
		WriteSet:      writeSet,
		RequiredTests: requiredTests,
		Issues:        issues,
	}
}

func autoprogrammingRequestMultiGoalFocalTestIssuesV0(
	request AutoprogrammingRequestV0,
	groups []AutoprogrammingTaskGroupV0,
) []AutoprogrammingRequestIssueV0 {
	if ClassifyAutoprogrammingGoalMigrationV0(request).Status != AutoprogrammingGoalMigrationGoalReadyV0 ||
		len(groups) <= 1 {
		return nil
	}
	var issues []AutoprogrammingRequestIssueV0
	for _, group := range groups {
		if len(autoprogrammingFocalRequiredTestsForGroupV0(group)) > 0 ||
			len(autoprogrammingAcceptanceChecksForGroupV0(group)) > 0 {
			continue
		}
		issues = append(issues, autoprogrammingRequestIssueV0(
			"goal_focal_required_tests_missing",
			"tasks."+group.Area+".required_tests",
			"cada goal multigrupo requiere tests focales declarados por tarea, acceptance check o guard contractual; required_tests globales pertenecen al batch",
		))
	}
	return issues
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

func autoprogrammingRequestDelegationIssuesV0(
	request AutoprogrammingRequestV0,
) []AutoprogrammingRequestIssueV0 {
	var issues []AutoprogrammingRequestIssueV0
	if request.MaxDelegationDepth < 0 ||
		request.MaxDelegationDepth > AutoprogrammingRequestMaxDelegationDepthV0 {
		issues = append(issues, autoprogrammingRequestIssueV0(
			"delegation_depth_budget_invalid",
			"max_delegation_depth",
			"max_delegation_depth fuera de rango",
		))
	}
	if request.MaxSubagentsPerAgent < 0 ||
		request.MaxSubagentsPerAgent > AutoprogrammingRequestMaxSubagentsPerAgentV0 {
		issues = append(issues, autoprogrammingRequestIssueV0(
			"subagents_per_agent_budget_invalid",
			"max_subagents_per_agent",
			"max_subagents_per_agent fuera de rango",
		))
	}
	if request.MaxRecursiveAgents < 0 ||
		request.MaxRecursiveAgents > AutoprogrammingRequestMaxRecursiveAgentsV0 {
		issues = append(issues, autoprogrammingRequestIssueV0(
			"recursive_agents_budget_invalid",
			"max_recursive_agents",
			"max_recursive_agents fuera de rango",
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
	if trimmed == "" ||
		trimmed == "." ||
		autoprogrammingPathHasURISchemeV0(trimmed) ||
		strings.HasPrefix(trimmed, "~") ||
		strings.Contains(trimmed, "$") ||
		strings.Contains(trimmed, "\\") ||
		strings.ContainsAny(trimmed, "\x00\r\n") ||
		filepath.IsAbs(trimmed) ||
		autoprogrammingPathHasDrivePrefixV0(trimmed) {
		return false
	}
	for _, segment := range strings.Split(trimmed, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return false
		}
	}
	cleaned := filepath.ToSlash(filepath.Clean(trimmed))
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return false
	}
	for _, segment := range strings.Split(cleaned, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return false
		}
	}
	return true
}

func autoprogrammingPathHasDrivePrefixV0(path string) bool {
	return len(path) >= 2 &&
		((path[0] >= 'a' && path[0] <= 'z') || (path[0] >= 'A' && path[0] <= 'Z')) &&
		path[1] == ':'
}

func autoprogrammingPathHasURISchemeV0(path string) bool {
	separator := strings.IndexByte(path, ':')
	if separator <= 0 || !((path[0] >= 'a' && path[0] <= 'z') || (path[0] >= 'A' && path[0] <= 'Z')) {
		return false
	}
	for _, character := range path[1:separator] {
		if !((character >= 'a' && character <= 'z') || (character >= 'A' && character <= 'Z') || (character >= '0' && character <= '9') || character == '+' || character == '-' || character == '.') {
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
