package orquestacionnucleoapp

import (
	"context"
	"strconv"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

type HeuristicAutonomousDirectorPolicyV0 struct{}

var _ AutonomousDirectorPolicyPortV0 = HeuristicAutonomousDirectorPolicyV0{}

func (HeuristicAutonomousDirectorPolicyV0) DecideAutonomousDirectorV0(
	ctx context.Context,
	input AutonomousDirectorDecisionInputV0,
) (AutonomousDirectorDecisionV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return AutonomousDirectorDecisionV0{}, err
	}
	limits := normalizeAutonomousDirectorLimitsV0(input.Limits)
	stats := autonomousDirectorStatsForLoopV0(input.Run, input.Stats)
	workload := autonomousWorkloadFromStatsV0(input.Run, stats)
	teamSize := boundedAutonomousTeamSizeV0(
		autonomousTeamSizeForStatsV0(
			autonomousBaseTeamSizeV0(input.Run, workload),
			stats,
		),
		limits.MaxTeamSize,
	)
	parallel := autonomousParallelForStatsV0(teamSize, limits.MaxParallelAgents, stats)
	return AutonomousDirectorDecisionV0{
		TeamSize:             teamSize,
		MaxParallelAgents:    parallel,
		MaxBursts:            boundedAutonomousLimitV0(teamSize*3, limits.MaxBursts),
		MaxStepsPerBurst:     boundedAutonomousLimitV0(teamSize*2, limits.MaxStepsPerBurst),
		MaxDispatchesPerWait: boundedAutonomousLimitV0(parallel, limits.MaxDispatchesPerWait),
		MaxCommandsPerCycle:  boundedAutonomousLimitV0(teamSize*2, limits.MaxCommandsPerCycle),
		MaxOutboxPerCycle:    boundedAutonomousLimitV0(parallel, limits.MaxOutboxPerCycle),
		RecommendedCapacity:  autonomousCapacityForStatsV0(input.Run.CurrentPhase, workload, stats),
		Summary:              autonomousDirectorSummaryV0(input.Run, stats, teamSize, parallel),
		EvidenceRefs:         compactStringsV0(append(input.Requests.EvidenceRefs, "evidence-ref-autonomous-director-v0")),
	}, nil
}

func normalizeAutonomousDirectorLimitsV0(
	limits AutonomousDirectorLimitsV0,
) AutonomousDirectorLimitsV0 {
	if limits.MaxTeamSize <= 0 {
		limits.MaxTeamSize = 6
	}
	if limits.MaxParallelAgents <= 0 {
		limits.MaxParallelAgents = 4
	}
	if limits.MaxBursts <= 0 {
		limits.MaxBursts = 18
	}
	if limits.MaxStepsPerBurst <= 0 {
		limits.MaxStepsPerBurst = 8
	}
	if limits.MaxDispatchesPerWait <= 0 {
		limits.MaxDispatchesPerWait = 6
	}
	if limits.MaxCommandsPerCycle <= 0 {
		limits.MaxCommandsPerCycle = 12
	}
	if limits.MaxOutboxPerCycle <= 0 {
		limits.MaxOutboxPerCycle = 6
	}
	return limits
}

func autonomousOpenWorkloadV0(run orquestacoreworkflow.OrchestrationRunV0) int {
	if len(run.Tasks) == 0 {
		return 0
	}
	closed := autonomousStringSetV0(run.ClosedTasks)
	open := 0
	for _, task := range run.Tasks {
		if !closed[strings.TrimSpace(task)] {
			open++
		}
	}
	return open
}

func autonomousDirectorStatsForLoopV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	stats DirectorRunStatsV0,
) DirectorRunStatsV0 {
	if strings.TrimSpace(stats.RunRef) != "" ||
		strings.TrimSpace(stats.SchemaVersion) != "" {
		return stats
	}
	return BuildDirectorRunStatsV0(run)
}

func autonomousWorkloadFromStatsV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	stats DirectorRunStatsV0,
) int {
	if stats.Counts.TasksTotal > 0 || stats.Counts.TasksOpen > 0 {
		return stats.Counts.TasksOpen
	}
	return autonomousOpenWorkloadV0(run)
}

func autonomousStringSetV0(values []string) map[string]bool {
	set := make(map[string]bool, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			set[trimmed] = true
		}
	}
	return set
}

func autonomousTeamSizeForStatsV0(
	base int,
	stats DirectorRunStatsV0,
) int {
	if autonomousCriticalProgressSignalsV0(stats) && base < 2 {
		return 2
	}
	return base
}

func autonomousParallelForStatsV0(
	teamSize int,
	maxParallel int,
	stats DirectorRunStatsV0,
) int {
	parallel := boundedAutonomousTeamSizeV0(teamSize, maxParallel)
	if autonomousCriticalProgressSignalsV0(stats) && parallel > 2 {
		return 2
	}
	if stats.Progress.OverBudgetButActiveAgents > 0 && parallel > 3 {
		return 3
	}
	return parallel
}

func autonomousBaseTeamSizeV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	workload int,
) int {
	switch run.CurrentPhase {
	case orquestacoreworkflow.OrchestrationPhaseBrainstormingArquitecturaV0,
		orquestacoreworkflow.OrchestrationPhaseVotacionYDecisionV0,
		orquestacoreworkflow.OrchestrationPhaseValidacionFinalV0:
		return 3
	case orquestacoreworkflow.OrchestrationPhaseRevisionV0,
		orquestacoreworkflow.OrchestrationPhaseIntegracionV0:
		return 2
	}
	switch {
	case workload >= 8:
		return 6
	case workload >= 4:
		return 4
	case workload >= 2:
		return 2
	default:
		return 1
	}
}

func autonomousCapacityForPhaseV0(
	phase orquestacoreworkflow.OrchestrationPhaseIDV0,
	workload int,
) orquestacoreworkflow.OrchestrationCapacityRecommendationV0 {
	switch phase {
	case orquestacoreworkflow.OrchestrationPhaseVotacionYDecisionV0,
		orquestacoreworkflow.OrchestrationPhaseValidacionFinalV0:
		return orquestacoreworkflow.OrchestrationCapacityXHighV0
	case orquestacoreworkflow.OrchestrationPhaseBrainstormingArquitecturaV0,
		orquestacoreworkflow.OrchestrationPhaseIntegracionV0,
		orquestacoreworkflow.OrchestrationPhaseRevisionV0:
		return orquestacoreworkflow.OrchestrationCapacityHighV0
	}
	if workload >= 4 {
		return orquestacoreworkflow.OrchestrationCapacityHighV0
	}
	return orquestacoreworkflow.OrchestrationCapacityMediumV0
}

func autonomousCapacityForStatsV0(
	phase orquestacoreworkflow.OrchestrationPhaseIDV0,
	workload int,
	stats DirectorRunStatsV0,
) orquestacoreworkflow.OrchestrationCapacityRecommendationV0 {
	if autonomousCriticalProgressSignalsV0(stats) {
		return orquestacoreworkflow.OrchestrationCapacityXHighV0
	}
	if stats.Progress.OverBudgetButActiveAgents > 0 ||
		stats.Progress.SourceStatus == DirectorProgressSourceErrorV0 {
		return orquestacoreworkflow.OrchestrationCapacityHighV0
	}
	return autonomousCapacityForPhaseV0(phase, workload)
}

func autonomousCriticalProgressSignalsV0(stats DirectorRunStatsV0) bool {
	return stats.Progress.StalledAgents > 0 ||
		stats.Progress.LoopDetectedAgents > 0 ||
		stats.Progress.OverBudgetNoActivityAgents > 0
}

func boundedAutonomousTeamSizeV0(value int, max int) int {
	if value < 1 {
		return 1
	}
	if max > 0 && value > max {
		return max
	}
	return value
}

func boundedAutonomousLimitV0(value int, max int) int {
	if value < 1 {
		value = 1
	}
	if max > 0 && value > max {
		return max
	}
	return value
}

func autonomousDirectorSummaryV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	stats DirectorRunStatsV0,
	teamSize int,
	parallel int,
) string {
	phase := strings.TrimSpace(string(run.CurrentPhase))
	if phase == "" {
		phase = "sin_fase"
	}
	return "director autonomo: fase " + phase +
		", equipo " + strconv.Itoa(teamSize) +
		", paralelo " + strconv.Itoa(parallel) +
		autonomousDirectorProgressSummaryV0(stats)
}

func autonomousDirectorProgressSummaryV0(stats DirectorRunStatsV0) string {
	if stats.Progress.StalledAgents == 0 &&
		stats.Progress.LoopDetectedAgents == 0 &&
		stats.Progress.OverBudgetNoActivityAgents == 0 &&
		stats.Progress.OverBudgetButActiveAgents == 0 {
		return ""
	}
	return ", senales progreso stalled=" + strconv.Itoa(stats.Progress.StalledAgents) +
		", loop=" + strconv.Itoa(stats.Progress.LoopDetectedAgents) +
		", over_budget_no_activity=" + strconv.Itoa(stats.Progress.OverBudgetNoActivityAgents) +
		", over_budget_active=" + strconv.Itoa(stats.Progress.OverBudgetButActiveAgents)
}
