package main

import (
	"os"
	"strings"
	"time"

	orquestaserver "orquesta/modulos/orquesta-server"
)

func serverIdleSelfImprovementAfterSecondsFromProjectConfigFileV0(config serverProjectConfigFileV0) int {
	if strings.TrimSpace(os.Getenv(envServerIdleSelfImprovementAfterV0)) != "" ||
		strings.TrimSpace(os.Getenv(envServerIdleSelfImprovementAfterLegacyV0)) != "" {
		return idleSelfImprovementAfterSecondsFromEnvV0()
	}
	if config.ServerIdle.AfterSeconds != nil && *config.ServerIdle.AfterSeconds >= 0 {
		return *config.ServerIdle.AfterSeconds
	}
	if config.ServerIdleLegacy.AfterSeconds != nil && *config.ServerIdleLegacy.AfterSeconds >= 0 {
		return *config.ServerIdleLegacy.AfterSeconds
	}
	return int(orquestaserver.DefaultIdleSelfImprovementAfterV0 / time.Second)
}

func serverIdleSelfImprovementDisabledFromProjectConfigFileV0(config serverProjectConfigFileV0, fallback bool) bool {
	return boolProjectConfigOrEnvOrDefaultV0(
		envServerIdleSelfImprovementDisabledV0,
		firstBoolPointerV0(config.ServerIdle.Disabled, config.ServerIdleLegacy.Disabled),
		fallback,
	)
}

func serverIdleSelfImprovementProjectWorkDirFromProjectConfigFileV0(config serverProjectConfigFileV0, fallback string) string {
	return absDirProjectConfigOrEnvOrDefaultV0(
		envServerIdleSelfImprovementProjectWorkDirV0,
		firstStringPointerV0(config.ServerIdle.ProjectWorkDir, config.ServerIdleLegacy.ProjectWorkDir),
		fallback,
	)
}

func serverIdleSelfImprovementStringFromProjectConfigFileV0(
	config serverProjectConfigFileV0,
	key string,
	fallback string,
) string {
	switch key {
	case envServerIdleSelfImprovementProjectRefV0:
		return stringProjectConfigOrEnvOrDefaultV0(key, firstStringPointerV0(config.ServerIdle.ProjectRef, config.ServerIdleLegacy.ProjectRef), fallback)
	case envServerIdleSelfImprovementWorktreeRefV0:
		return stringProjectConfigOrEnvOrDefaultV0(key, firstStringPointerV0(config.ServerIdle.WorktreeRef, config.ServerIdleLegacy.WorktreeRef), fallback)
	case envServerIdleSelfImprovementBranchRefV0:
		return stringProjectConfigOrEnvOrDefaultV0(key, firstStringPointerV0(config.ServerIdle.BranchRef, config.ServerIdleLegacy.BranchRef), fallback)
	case envServerIdleSelfImprovementAreaV0:
		return stringProjectConfigOrEnvOrDefaultV0(key, firstStringPointerV0(config.ServerIdle.Area, config.ServerIdleLegacy.Area), fallback)
	default:
		return envOrDefaultV0(key, fallback)
	}
}

func serverIdleSelfImprovementStringSliceFromProjectConfigFileV0(
	config serverProjectConfigFileV0,
	key string,
	fallback []string,
) []string {
	switch key {
	case envServerIdleSelfImprovementWriteSetV0:
		return stringSliceProjectConfigOrEnvOrDefaultV0(key, firstStringSlicePointerV0(config.ServerIdle.WriteSet, config.ServerIdleLegacy.WriteSet), fallback)
	case envServerIdleSelfImprovementRequiredTestsV0:
		return stringSliceProjectConfigOrEnvOrDefaultV0(key, firstStringSlicePointerV0(config.ServerIdle.RequiredTests, config.ServerIdleLegacy.RequiredTests), fallback)
	case envServerIdleSelfImprovementContextRefsV0:
		return stringSliceProjectConfigOrEnvOrDefaultV0(key, firstStringSlicePointerV0(config.ServerIdle.ContextRefs, config.ServerIdleLegacy.ContextRefs), fallback)
	case envServerIdleSelfImprovementEvidenceRefsV0:
		return stringSliceProjectConfigOrEnvOrDefaultV0(key, firstStringSlicePointerV0(config.ServerIdle.EvidenceRefs, config.ServerIdleLegacy.EvidenceRefs), fallback)
	case envServerIdleSelfImprovementAcceptanceV0:
		return stringSliceProjectConfigOrEnvOrDefaultV0(key, firstStringSlicePointerV0(config.ServerIdle.Acceptance, config.ServerIdleLegacy.Acceptance), fallback)
	case envServerIdleSelfImprovementCompactRulesV0:
		return stringSliceProjectConfigOrEnvOrDefaultV0(key, firstStringSlicePointerV0(config.ServerIdle.CompactRules, config.ServerIdleLegacy.CompactRules), fallback)
	default:
		return csvEnvOrDefaultV0(key, fallback)
	}
}

func serverIdleSelfImprovementGoalFirstFromProjectConfigFileV0(projectConfig serverProjectConfigFileV0) bool {
	if strings.TrimSpace(os.Getenv(envServerIdleSelfImprovementGoalFirstV0)) != "" {
		return boolEnvOrDefaultV0(envServerIdleSelfImprovementGoalFirstV0, false)
	}
	if value := firstBoolPointerV0(projectConfig.ServerIdle.GoalFirstEnabled, projectConfig.ServerIdleLegacy.GoalFirstEnabled); value != nil {
		return *value
	}
	return serverGoalBackendOperationalFromProjectConfigFileV0(projectConfig)
}

func serverIdleSelfImprovementFrozenTestsFromProjectConfigFileV0(config serverProjectConfigFileV0) bool {
	return boolProjectConfigOrEnvOrDefaultV0(
		envServerIdleSelfImprovementFrozenTestsV0,
		firstBoolPointerV0(config.ServerIdle.FrozenTestsEnabled, config.ServerIdleLegacy.FrozenTestsEnabled),
		false,
	)
}

func serverIdleSelfImprovementIntFromProjectConfigFileV0(
	config serverProjectConfigFileV0,
	key string,
	fallback int,
) int {
	switch key {
	case envServerIdleSelfImprovementPriorityScoreV0:
		return intProjectConfigOrEnvOrDefaultV0(key, firstIntPointerV0(config.ServerIdle.PriorityScore, config.ServerIdleLegacy.PriorityScore), fallback)
	case envServerIdleSelfImprovementMaxRequestsV0:
		return intProjectConfigOrEnvOrDefaultV0(key, firstIntPointerV0(config.ServerIdle.MaxRequests, config.ServerIdleLegacy.MaxRequests), fallback)
	case envServerIdleSelfImprovementTargetQueueV0:
		return intProjectConfigOrEnvOrDefaultV0(key, firstIntPointerV0(config.ServerIdle.TargetQueue, config.ServerIdleLegacy.TargetQueue), fallback)
	case envServerIdleSelfImprovementDailyGoalBudgetV0:
		return intProjectConfigOrEnvOrDefaultV0(key, firstIntPointerV0(config.ServerIdle.DailyGoalBudget, config.ServerIdleLegacy.DailyGoalBudget), fallback)
	case envServerIdleSelfImprovementDailyContextBudgetBytesV0:
		return intProjectConfigOrEnvOrDefaultV0(key, firstIntPointerV0(config.ServerIdle.DailyContextBudgetBytes, config.ServerIdleLegacy.DailyContextBudgetBytes), fallback)
	default:
		return intEnvOrDefaultV0(key, fallback)
	}
}

func serverIdleProjectConfigHasValueForEnvKeyV0(config serverProjectConfigFileV0, key string) bool {
	switch key {
	case envServerIdleSelfImprovementAfterV0:
		return configIntPointerNonNegativeV0(firstIntPointerV0(config.ServerIdle.AfterSeconds, config.ServerIdleLegacy.AfterSeconds))
	case envServerIdleSelfImprovementDisabledV0:
		return firstBoolPointerV0(config.ServerIdle.Disabled, config.ServerIdleLegacy.Disabled) != nil
	case envServerIdleSelfImprovementProjectWorkDirV0:
		return configStringPointerHasValueV0(firstStringPointerV0(config.ServerIdle.ProjectWorkDir, config.ServerIdleLegacy.ProjectWorkDir))
	case envServerIdleSelfImprovementProjectRefV0:
		return configStringPointerHasValueV0(firstStringPointerV0(config.ServerIdle.ProjectRef, config.ServerIdleLegacy.ProjectRef))
	case envServerIdleSelfImprovementWorktreeRefV0:
		return configStringPointerHasValueV0(firstStringPointerV0(config.ServerIdle.WorktreeRef, config.ServerIdleLegacy.WorktreeRef))
	case envServerIdleSelfImprovementBranchRefV0:
		return configStringPointerHasValueV0(firstStringPointerV0(config.ServerIdle.BranchRef, config.ServerIdleLegacy.BranchRef))
	case envServerIdleSelfImprovementAreaV0:
		return configStringPointerHasValueV0(firstStringPointerV0(config.ServerIdle.Area, config.ServerIdleLegacy.Area))
	case envServerIdleSelfImprovementWriteSetV0:
		return configStringSlicePointerHasValueV0(firstStringSlicePointerV0(config.ServerIdle.WriteSet, config.ServerIdleLegacy.WriteSet))
	case envServerIdleSelfImprovementRequiredTestsV0:
		return configStringSlicePointerHasValueV0(firstStringSlicePointerV0(config.ServerIdle.RequiredTests, config.ServerIdleLegacy.RequiredTests))
	case envServerIdleSelfImprovementContextRefsV0:
		return configStringSlicePointerHasValueV0(firstStringSlicePointerV0(config.ServerIdle.ContextRefs, config.ServerIdleLegacy.ContextRefs))
	case envServerIdleSelfImprovementEvidenceRefsV0:
		return configStringSlicePointerHasValueV0(firstStringSlicePointerV0(config.ServerIdle.EvidenceRefs, config.ServerIdleLegacy.EvidenceRefs))
	case envServerIdleSelfImprovementAcceptanceV0:
		return configStringSlicePointerHasValueV0(firstStringSlicePointerV0(config.ServerIdle.Acceptance, config.ServerIdleLegacy.Acceptance))
	case envServerIdleSelfImprovementGoalFirstV0:
		return firstBoolPointerV0(config.ServerIdle.GoalFirstEnabled, config.ServerIdleLegacy.GoalFirstEnabled) != nil
	case envServerIdleSelfImprovementFrozenTestsV0:
		return firstBoolPointerV0(config.ServerIdle.FrozenTestsEnabled, config.ServerIdleLegacy.FrozenTestsEnabled) != nil
	case envServerIdleSelfImprovementCompactRulesV0:
		return configStringSlicePointerHasValueV0(firstStringSlicePointerV0(config.ServerIdle.CompactRules, config.ServerIdleLegacy.CompactRules))
	case envServerIdleSelfImprovementPriorityScoreV0:
		return configIntPointerPositiveV0(firstIntPointerV0(config.ServerIdle.PriorityScore, config.ServerIdleLegacy.PriorityScore))
	case envServerIdleSelfImprovementMaxRequestsV0:
		return configIntPointerPositiveV0(firstIntPointerV0(config.ServerIdle.MaxRequests, config.ServerIdleLegacy.MaxRequests))
	case envServerIdleSelfImprovementTargetQueueV0:
		return configIntPointerPositiveV0(firstIntPointerV0(config.ServerIdle.TargetQueue, config.ServerIdleLegacy.TargetQueue))
	case envServerIdleSelfImprovementDailyGoalBudgetV0:
		return configIntPointerPositiveV0(firstIntPointerV0(config.ServerIdle.DailyGoalBudget, config.ServerIdleLegacy.DailyGoalBudget))
	case envServerIdleSelfImprovementDailyContextBudgetBytesV0:
		return configIntPointerPositiveV0(firstIntPointerV0(config.ServerIdle.DailyContextBudgetBytes, config.ServerIdleLegacy.DailyContextBudgetBytes))
	default:
		return false
	}
}

func serverIdleEnvKeysV0() []string {
	return []string{
		envServerIdleSelfImprovementAfterV0,
		envServerIdleSelfImprovementDisabledV0,
		envServerIdleSelfImprovementProjectWorkDirV0,
		envServerIdleSelfImprovementProjectRefV0,
		envServerIdleSelfImprovementWorktreeRefV0,
		envServerIdleSelfImprovementBranchRefV0,
		envServerIdleSelfImprovementAreaV0,
		envServerIdleSelfImprovementWriteSetV0,
		envServerIdleSelfImprovementRequiredTestsV0,
		envServerIdleSelfImprovementContextRefsV0,
		envServerIdleSelfImprovementEvidenceRefsV0,
		envServerIdleSelfImprovementAcceptanceV0,
		envServerIdleSelfImprovementGoalFirstV0,
		envServerIdleSelfImprovementFrozenTestsV0,
		envServerIdleSelfImprovementCompactRulesV0,
		envServerIdleSelfImprovementPriorityScoreV0,
		envServerIdleSelfImprovementMaxRequestsV0,
		envServerIdleSelfImprovementTargetQueueV0,
		envServerIdleSelfImprovementDailyGoalBudgetV0,
		envServerIdleSelfImprovementDailyContextBudgetBytesV0,
	}
}

func firstStringPointerV0(values ...*string) *string {
	for _, value := range values {
		if configStringPointerHasValueV0(value) {
			return value
		}
	}
	return nil
}

func firstBoolPointerV0(values ...*bool) *bool {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

func firstIntPointerV0(values ...*int) *int {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

func firstStringSlicePointerV0(values ...*[]string) *[]string {
	for _, value := range values {
		if configStringSlicePointerHasValueV0(value) {
			return value
		}
	}
	return nil
}

func configIntPointerNonNegativeV0(value *int) bool {
	return value != nil && *value >= 0
}
