package orquestaautoprogramming

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

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
	Partition   AutoprogrammingPartitionPlanV0        `json:"partition,omitempty"`
	Groups      []AutoprogrammingProgrammableGroupV0  `json:"groups"`
	Profiles    []orquestacoreworkflow.WorkProfileV0  `json:"profiles"`
	Tasks       []orquestacoreworkflow.WorkflowTaskV0 `json:"tasks"`
}

type AutoprogrammingProgrammableGroupV0 struct {
	Area          string                              `json:"area"`
	TaskRefs      []string                            `json:"task_refs"`
	WriteSet      []string                            `json:"write_set"`
	RequiredTests []string                            `json:"required_tests"`
	DependsOn     []string                            `json:"depends_on,omitempty"`
	BlockedBy     []string                            `json:"blocked_by,omitempty"`
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

	partition, partitionIssues := autoprogrammingPartitionWriteSetByAreaV0(
		request,
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
	work.Partition = partition
	for i, group := range validation.Groups {
		profile, task, issue := autoprogrammingWorkflowTaskForGroupV0(
			request,
			group,
			partition.WriteSetByArea[group.Area],
			partition.DependsOnByArea[group.Area],
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
			DependsOn:     append([]string(nil), task.DependsOn...),
			BlockedBy:     append([]string(nil), partition.BlockedByArea[group.Area]...),
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
	dependsOn []string,
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
		Title:              autoprogrammingTitleForGroupV0(group, groupIndex),
		Objective:          autoprogrammingObjectiveForGroupV0(group),
		Summary:            autoprogrammingSummaryForGroupV0(group),
		ScopeRefs:          append([]string(nil), writeSet...),
		RequiredTests:      autoprogrammingRequiredTestsForGroupV0(requiredTests, group),
		AcceptanceCriteria: autoprogrammingAcceptanceCriteriaForGroupV0(group),
		FunctionContractRefs: []orquestacoreworkflow.WorkflowFunctionContractRefV0{
			{FunctionName: "ValidateAutoprogrammingRequestV0"},
			{FunctionName: "BuildAutoprogrammingProgrammableWorkV0"},
		},
		DependsOn: dependsOn,
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
	task.ContextRefs = autoprogrammingContextRefsForGroupV0(request, group)
	task, err = orquestacoreworkflow.NewWorkflowTaskV0(task)
	if err != nil {
		return orquestacoreworkflow.WorkProfileV0{}, orquestacoreworkflow.WorkflowTaskV0{},
			autoprogrammingRequestIssueV0("workflow_task_invalid", "workflow_task", err.Error())
	}
	return profile, task, AutoprogrammingRequestIssueV0{}
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
