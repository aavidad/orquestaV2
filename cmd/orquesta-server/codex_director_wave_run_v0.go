package main

import (
	"context"
	"fmt"
	"path/filepath"

	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
)

func runCodexLaunchDirectorWaveV0(
	ctx context.Context,
	config codexDirectorWaveConfigV0,
) (codexDirectorWaveSummaryV0, error) {
	request := orquestadirectoroperativo.OperationalDirectorRequestV0{
		RequestRef:               config.RequestRef,
		RunRef:                   config.RunRef,
		ProjectRef:               config.ProjectRef,
		Objective:                config.Objective,
		Mode:                     orquestadirectoroperativo.OperationalDirectorModeProgrammingV0,
		DomainRefs:               append([]string(nil), config.DomainRefs...),
		WorktreeRef:              config.WorktreeRef,
		WorktreeIsolated:         true,
		BranchRef:                config.BranchRef,
		WriteSet:                 append([]string(nil), config.WriteSet...),
		RequiredTests:            append([]string(nil), config.RequiredTests...),
		MaxParallelAgents:        config.Wave.Agents,
		AllowRecursiveDelegation: config.AllowRecursiveDelegation,
		MaxDelegationDepth:       config.MaxDelegationDepth,
		MaxSubagentsPerAgent:     config.MaxSubagentsPerAgent,
		MaxRecursiveAgents:       config.RecursiveAgentBudget,
	}
	result := orquestadirectoroperativo.BuildOperationalDirectorPlanV0(request)
	summary := codexDirectorWaveSummaryV0{
		SchemaVersion: codexDirectorWaveSummarySchemaVersionV0,
		Request:       request,
		Plan:          result.Plan,
		GuardOptIn:    codexDirectorGuardOptInSummaryFromConfigV0(config),
		Issues:        append([]orquestadirectoroperativo.OperationalDirectorIssueV0(nil), result.Issues...),
	}
	if guardIssues := codexDirectorStrictGuardIssuesV0(config); len(guardIssues) > 0 {
		summary.Issues = append(summary.Issues, guardIssues...)
		return summary, nil
	}
	isolation, isolationIssues := codexDirectorPrepareWorktreeIsolationV0(ctx, config)
	summary.WorktreeIsolation = isolation
	if len(isolationIssues) > 0 {
		summary.Issues = append(summary.Issues, isolationIssues...)
		return summary, nil
	}
	if !result.Accepted || !result.ReadyToLaunch || result.Blocked {
		return summary, nil
	}
	budget := orquestadirectoroperativo.BuildOperationalDirectorAgentBudgetV0(result.Plan)
	summary.AgentBudget = budget
	if budget.Exceeded {
		summary.Issues = append(summary.Issues, orquestadirectoroperativo.OperationalDirectorIssueV0{
			Code:    "recursive_agent_budget_exceeded",
			Field:   "recursive_agent_budget",
			Message: fmt.Sprintf("planned agents %d exceed budget %d", budget.PlannedAgents, budget.MaxAgents),
		})
		return summary, nil
	}
	return runCodexLaunchDirectorWaveReadyPlanV0(ctx, config, result.Plan, summary)
}

func runCodexLaunchDirectorWaveReadyPlanV0(
	ctx context.Context,
	config codexDirectorWaveConfigV0,
	plan orquestadirectoroperativo.OperationalDirectorPlanV0,
	summary codexDirectorWaveSummaryV0,
) (codexDirectorWaveSummaryV0, error) {
	work := orquestadirectoroperativo.BuildOperationalDirectorWaveWorkV0(plan)
	summary.WaveWork = work
	if len(work.Issues) > 0 {
		summary.Issues = append(summary.Issues, work.Issues...)
		return summary, nil
	}
	launchConfig := config.Wave
	launchConfig.Agents = plan.MaxParallelAgents
	launchConfig.Prompt = plan.Objective
	launchConfig.AgentPrompts = codexDirectorAgentPromptsV0(plan, work, config.DomainContextBlocks, config)
	launch, err := runCodexLaunchWaveV0(ctx, launchConfig)
	if err != nil {
		return summary, err
	}
	summary.Launch = launch
	childLaunches, err := runCodexLaunchDirectorChildWavesV0(ctx, config, plan, work, launch)
	if err != nil {
		return summary, err
	}
	summary.ChildLaunches = childLaunches
	return summary, nil
}

func runCodexLaunchDirectorChildWavesV0(
	ctx context.Context,
	config codexDirectorWaveConfigV0,
	plan orquestadirectoroperativo.OperationalDirectorPlanV0,
	work orquestadirectoroperativo.OperationalDirectorWaveWorkV0,
	parentLaunch codexWaveLaunchSummaryV0,
) ([]codexDirectorChildWaveSummaryV0, error) {
	if !plan.RecursiveDelegation || plan.MaxSubagentsPerAgent <= 0 {
		return nil, nil
	}
	return runCodexLaunchDirectorChildWavesAtDepthV0(ctx, config, plan, work, parentLaunch, 1)
}

func runCodexLaunchDirectorChildWavesAtDepthV0(
	ctx context.Context,
	config codexDirectorWaveConfigV0,
	plan orquestadirectoroperativo.OperationalDirectorPlanV0,
	work orquestadirectoroperativo.OperationalDirectorWaveWorkV0,
	parentLaunch codexWaveLaunchSummaryV0,
	depth int,
) ([]codexDirectorChildWaveSummaryV0, error) {
	if depth <= 0 || depth > plan.MaxDelegationDepth {
		return nil, nil
	}
	out := make([]codexDirectorChildWaveSummaryV0, 0, len(parentLaunch.Agents))
	for index := range parentLaunch.Agents {
		child, err := runCodexLaunchDirectorChildWaveV0(ctx, config, plan, work, parentLaunch, index, depth)
		if err != nil {
			return out, err
		}
		out = append(out, child)
	}
	return out, nil
}

func runCodexLaunchDirectorChildWaveV0(
	ctx context.Context,
	config codexDirectorWaveConfigV0,
	plan orquestadirectoroperativo.OperationalDirectorPlanV0,
	work orquestadirectoroperativo.OperationalDirectorWaveWorkV0,
	parentLaunch codexWaveLaunchSummaryV0,
	parentIndexZeroBased int,
	depth int,
) (codexDirectorChildWaveSummaryV0, error) {
	parentIndex := parentIndexZeroBased + 1
	parent := parentLaunch.Agents[parentIndexZeroBased]
	childWaveRef := fmt.Sprintf("%s-depth-%02d-parent-%02d-children", parentLaunch.WaveRef, depth, parentIndex)
	childConfig := config.Wave
	childConfig.WaveRef = childWaveRef
	childConfig.Agents = plan.MaxSubagentsPerAgent
	childConfig.RuntimeWorkDir = filepath.Join(config.Wave.RuntimeWorkDir, "children", childWaveRef)
	childConfig.PurgeRuntime = false
	childConfig.PurgeReportOnly = false
	childConfig.PurgeConfirm = ""
	childConfig.Prompt = plan.Objective
	childConfig.AgentPrompts = codexDirectorChildAgentPromptsV0(
		plan, work, config.DomainContextBlocks, config, parent.AgentRef,
		parentIndex, len(parentLaunch.Agents), childConfig.Agents, depth,
	)
	launch, err := runCodexLaunchWaveV0(ctx, childConfig)
	if err != nil {
		return codexDirectorChildWaveSummaryV0{}, err
	}
	childLaunches, err := runCodexLaunchDirectorChildWavesAtDepthV0(ctx, config, plan, work, launch, depth+1)
	if err != nil {
		return codexDirectorChildWaveSummaryV0{}, err
	}
	return codexDirectorChildWaveSummaryV0{
		ParentAgentRef:            parent.AgentRef,
		ParentWaveRef:             parentLaunch.WaveRef,
		ParentIndex:               parentIndex,
		DelegationDepth:           depth,
		MaxDelegationDepth:        plan.MaxDelegationDepth,
		MaxSubagentsPerAgent:      plan.MaxSubagentsPerAgent,
		SubtreeAgentBudget:        codexDirectorSubtreeAgentBudgetV0(launch, childLaunches),
		ReviewRequiredBeforeClose: true,
		Launch:                    launch,
		ChildLaunches:             childLaunches,
	}, nil
}

func codexDirectorSubtreeAgentBudgetV0(
	launch codexWaveLaunchSummaryV0,
	childLaunches []codexDirectorChildWaveSummaryV0,
) int {
	total := len(launch.Agents)
	for _, child := range childLaunches {
		total += child.SubtreeAgentBudget
	}
	return total
}
