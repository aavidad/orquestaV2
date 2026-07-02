package orquestaautoprogramming

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestagoal "orquesta/modulos/orquesta-goal"
)

type AutoprogrammingProgrammableWorkResultV0 struct {
	Accepted bool                              `json:"accepted"`
	Work     AutoprogrammingProgrammableWorkV0 `json:"work"`
	Issues   []AutoprogrammingRequestIssueV0   `json:"issues,omitempty"`
}

type AutoprogrammingProgrammableWorkV0 struct {
	RequestRef    string                                       `json:"request_ref"`
	ProjectRef    string                                       `json:"project_ref"`
	WorktreeRef   string                                       `json:"worktree_ref"`
	BranchRef     string                                       `json:"branch_ref"`
	GoalMigration AutoprogrammingGoalMigrationClassificationV0 `json:"goal_migration"`
	Partition     AutoprogrammingPartitionPlanV0               `json:"partition,omitempty"`
	Groups        []AutoprogrammingProgrammableGroupV0         `json:"groups"`
	GoalSpecs     []orquestagoal.GoalWorkSpecV0                `json:"goal_specs,omitempty"`
	Profiles      []orquestacoreworkflow.WorkProfileV0         `json:"profiles"`
	Tasks         []orquestacoreworkflow.WorkflowTaskV0        `json:"tasks"`
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

const (
	AutoprogrammingDefaultMaxDelegationDepthV0   = 1
	AutoprogrammingDefaultMaxSubagentsPerAgentV0 = 6
	AutoprogrammingDefaultMaxRecursiveAgentsV0   = AutoprogrammingRequestDefaultMaxTaskRefsV0 * AutoprogrammingDefaultMaxSubagentsPerAgentV0
)

type autoprogrammingDelegationLimitsV0 struct {
	maxDelegationDepth   int
	maxSubagentsPerAgent int
	maxRecursiveAgents   int
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
	goalSpecs, goalIssues := BuildAutoprogrammingGoalWorkSpecsV0(work)
	if len(goalIssues) > 0 {
		return AutoprogrammingProgrammableWorkResultV0{
			Accepted: false,
			Work:     work,
			Issues:   goalIssues,
		}
	}
	work.GoalSpecs = goalSpecs
	work = autoprogrammingGoalReadyWithoutLegacyWorkflowSurfaceV0(work)

	return AutoprogrammingProgrammableWorkResultV0{
		Accepted: true,
		Work:     work,
	}
}

func autoprogrammingGoalReadyWithoutLegacyWorkflowSurfaceV0(
	work AutoprogrammingProgrammableWorkV0,
) AutoprogrammingProgrammableWorkV0 {
	if strings.TrimSpace(work.GoalMigration.Status) != AutoprogrammingGoalMigrationGoalReadyV0 {
		return work
	}
	work.Profiles = nil
	work.Tasks = nil
	for index := range work.Groups {
		work.Groups[index].Profile = orquestacoreworkflow.WorkProfileV0{}
		work.Groups[index].Task = orquestacoreworkflow.WorkflowTaskV0{}
	}
	return work
}

func autoprogrammingProgrammableWorkSkeletonV0(
	request AutoprogrammingRequestV0,
) AutoprogrammingProgrammableWorkV0 {
	return AutoprogrammingProgrammableWorkV0{
		RequestRef:    strings.TrimSpace(request.RequestRef),
		ProjectRef:    strings.TrimSpace(request.ProjectRef),
		WorktreeRef:   strings.TrimSpace(request.WorktreeRef),
		BranchRef:     strings.TrimSpace(request.BranchRef),
		GoalMigration: ClassifyAutoprogrammingGoalMigrationV0(request),
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
	limits := autoprogrammingDelegationLimitsForRequestV0(request)
	profile, err := orquestacoreworkflow.NewWorkProfileV0(orquestacoreworkflow.WorkProfileV0{
		SchemaVersion:        orquestacoreworkflow.WorkProfileSchemaVersionV0,
		ProfileRef:           "profile-" + taskRef,
		ProfileKind:          orquestacoreworkflow.WorkProfileImplementationV0,
		TaskRef:              taskRef,
		RunRef:               strings.TrimSpace(request.RequestRef),
		Title:                autoprogrammingTitleForGroupV0(group, groupIndex),
		Objective:            autoprogrammingObjectiveForGroupV0(group),
		Summary:              autoprogrammingSummaryForGroupV0(group),
		ScopeRefs:            append([]string(nil), writeSet...),
		RequiredTests:        autoprogrammingRequiredTestsForGroupV0(requiredTests, group),
		AcceptanceCriteria:   autoprogrammingAcceptanceCriteriaForGroupV0(group),
		SkillRefs:            autoprogrammingSkillRefsForGroupV0(group),
		MaxDelegationDepth:   limits.maxDelegationDepth,
		MaxChildAgents:       limits.maxSubagentsPerAgent,
		MaxSubagentsPerAgent: limits.maxSubagentsPerAgent,
		MaxRecursiveAgents:   limits.maxRecursiveAgents,
		FunctionContractRefs: append([]orquestacoreworkflow.WorkflowFunctionContractRefV0{
			{FunctionName: "ValidateAutoprogrammingRequestV0"},
			{FunctionName: "BuildAutoprogrammingProgrammableWorkV0"},
		}, autoprogrammingWorkflowFunctionContractRefsForGroupV0(group)...),
		DependsOn: dependsOn,
	})
	if err != nil {
		return orquestacoreworkflow.WorkProfileV0{}, orquestacoreworkflow.WorkflowTaskV0{},
			autoprogrammingWorkProfileIssueV0(err)
	}

	task, err := orquestacoreworkflow.WorkflowTaskFromWorkProfileV0(profile)
	if err != nil {
		return orquestacoreworkflow.WorkProfileV0{}, orquestacoreworkflow.WorkflowTaskV0{},
			autoprogrammingWorkflowTaskIssueV0(err)
	}
	task.ContextRefs = autoprogrammingContextRefsForGroupV0(request, group)
	task, issue := EnsureAutoprogrammingWorkflowTaskAcceptedByCoreV0(task)
	if issue.Code != "" {
		return orquestacoreworkflow.WorkProfileV0{}, orquestacoreworkflow.WorkflowTaskV0{}, issue
	}
	task, err = orquestacoreworkflow.NewWorkflowTaskV0(task)
	if err != nil {
		return orquestacoreworkflow.WorkProfileV0{}, orquestacoreworkflow.WorkflowTaskV0{},
			autoprogrammingWorkflowTaskIssueV0(err)
	}
	return profile, task, AutoprogrammingRequestIssueV0{}
}

func autoprogrammingDelegationLimitsForRequestV0(
	request AutoprogrammingRequestV0,
) autoprogrammingDelegationLimitsV0 {
	maxSubagents := AutoprogrammingDefaultMaxSubagentsPerAgentV0
	if request.MaxSubagentsPerAgent > 0 {
		maxSubagents = request.MaxSubagentsPerAgent
	}
	limits := autoprogrammingDelegationLimitsV0{
		maxDelegationDepth:   AutoprogrammingDefaultMaxDelegationDepthV0,
		maxSubagentsPerAgent: maxSubagents,
		maxRecursiveAgents:   autoprogrammingDefaultMaxRecursiveAgentsForRequestV0(request, maxSubagents),
	}
	if request.MaxDelegationDepth > 0 {
		limits.maxDelegationDepth = request.MaxDelegationDepth
	}
	if request.MaxRecursiveAgents > 0 {
		limits.maxRecursiveAgents = request.MaxRecursiveAgents
	}
	return limits
}

func autoprogrammingDefaultMaxRecursiveAgentsForRequestV0(
	request AutoprogrammingRequestV0,
	maxSubagentsPerAgent int,
) int {
	maxAgents := autoprogrammingRequestMaxTaskRefsV0(request) * maxSubagentsPerAgent
	if maxAgents <= 0 {
		return AutoprogrammingDefaultMaxRecursiveAgentsV0
	}
	if maxAgents > AutoprogrammingRequestMaxRecursiveAgentsV0 {
		return AutoprogrammingRequestMaxRecursiveAgentsV0
	}
	return maxAgents
}

func autoprogrammingWorkProfileIssueV0(err error) AutoprogrammingRequestIssueV0 {
	var publicErr orquestacoreworkflow.WorkProfileErrorV0
	if errors.As(err, &publicErr) {
		return autoprogrammingRequestIssueV0(
			"work_profile_invalid",
			"work_profile."+strings.TrimSpace(publicErr.Field),
			publicErr.Code,
		)
	}
	return autoprogrammingRequestIssueV0("work_profile_invalid", "work_profile", err.Error())
}

func autoprogrammingWorkflowTaskIssueV0(err error) AutoprogrammingRequestIssueV0 {
	var publicErr orquestacoreworkflow.WorkflowTaskErrorV0
	if errors.As(err, &publicErr) {
		return autoprogrammingRequestIssueV0(
			"workflow_task_invalid",
			"workflow_task."+strings.TrimSpace(publicErr.Field),
			publicErr.Code,
		)
	}
	return autoprogrammingRequestIssueV0("workflow_task_invalid", "workflow_task", err.Error())
}

func autoprogrammingProgrammableTaskRefV0(requestRef string, groupIndex int) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(requestRef)))
	return fmt.Sprintf("task-autoprogramming-%s-g%02d", hex.EncodeToString(sum[:])[:12], groupIndex+1)
}

func autoprogrammingProgrammableContextRefsV0(
	request AutoprogrammingRequestV0,
) []string {
	refs := []string{
		"request_ref:" + strings.TrimSpace(request.RequestRef),
		"project_ref:" + strings.TrimSpace(request.ProjectRef),
		"worktree_ref:" + strings.TrimSpace(request.WorktreeRef),
		"branch_ref:" + strings.TrimSpace(request.BranchRef),
	}
	if request.BacklogScan.Epoch != "" {
		refs = append(refs, "backlog_scan_epoch:"+request.BacklogScan.Epoch)
	}
	for _, ref := range request.BacklogScan.ReservationRefs {
		refs = append(refs, "backlog_scan_reservation_ref:"+ref)
	}
	for _, doc := range request.BacklogScan.Documents {
		refs = append(refs, "backlog_scan_doc:"+doc.Path+
			":line:"+fmt.Sprint(doc.StartLine)+":sha256:"+doc.SHA256)
	}
	return refs
}
