package main

import (
	"os"
	"strings"
	"time"

	orquestaappcodexstack "orquesta/modulos/orquesta-app-codex-stack"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
	orquestaserver "orquesta/modulos/orquesta-server"
)

func codexRuntimeConfigV0(
	serverConfig orquestaserver.ConfigV0,
	processRuntime *orquestaruntime.ProcessRuntimeConnectorV0,
	usageMetrics ...orquestaappcodexstack.CodexStackAgentUsageMetricsProviderPortV0,
) orquestaappcodexstack.CodexRuntimeConfigV0 {
	var usageSource orquestaappcodexstack.CodexStackAgentUsageMetricsProviderPortV0
	if len(usageMetrics) > 0 {
		usageSource = usageMetrics[0]
	}
	envConfig := codexRuntimeEnvConfigFromEnvV0()
	return orquestaappcodexstack.CodexRuntimeConfigV0{
		CommandPath:              envConfig.CommandPath,
		ProjectWorkDir:           serverConfig.ProjectWorkDir,
		RuntimeWorkDir:           serverConfig.RuntimeWorkDir,
		CodeHomeDir:              envConfig.CodeHomeDir,
		HomeDir:                  envConfig.HomeDir,
		PathEnv:                  envConfig.PathEnv,
		Model:                    envConfig.Model,
		ReasoningEffort:          envConfig.ReasoningEffort,
		Profile:                  envConfig.Profile,
		Sandbox:                  envConfig.Sandbox,
		ApprovalPolicy:           envConfig.ApprovalPolicy,
		DirectorSandbox:          envConfig.DirectorSandbox,
		DirectorApprovalPolicy:   envConfig.DirectorApprovalPolicy,
		InteractiveApprovalOptIn: envConfig.InteractiveApprovalOptIn,
		ExtraArgs:                envConfig.ExtraArgs,
		PromptHints:              codexServerPromptHintsV0(serverConfig),
		Runtime:                  processRuntime,
		ProcessStopper:           processRuntime,
		SnapshotSource:           processRuntime,
		MaxBatchReady:            envConfig.Limits.MaxBatchReady,
		MaxConcurrency:           envConfig.Limits.MaxLiveProcesses,
		WaitInterval:             envConfig.WaitInterval,
		ProgressPolicy:           envConfig.ProgressPolicy,
		ProgressBudget:           envConfig.ProgressBudget,
		UsageMetrics:             usageSource,
	}
}

type codexRuntimeEnvConfigV0 struct {
	CommandPath              string
	CodeHomeDir              string
	HomeDir                  string
	PathEnv                  string
	Model                    string
	ReasoningEffort          string
	Profile                  string
	Sandbox                  string
	ApprovalPolicy           string
	DirectorSandbox          string
	DirectorApprovalPolicy   string
	InteractiveApprovalOptIn bool
	ExtraArgs                []string
	Limits                   codexRuntimeLimitsV0
	WaitInterval             time.Duration
	ProgressPolicy           orquestaruntime.AgentProgressHeartbeatPolicyV0
	ProgressBudget           orquestaruntimecodexdelivery.CodexBudgetActivityPolicyV0
}

func codexRuntimeEnvConfigFromEnvV0() codexRuntimeEnvConfigV0 {
	return codexRuntimeEnvConfigV0{
		CommandPath: codexCommandPathV0(),
		CodeHomeDir: codeHomeDirV0(),
		HomeDir:     homeDirV0(),
		PathEnv:     envOrDefaultV0(envCodexPathV0, os.Getenv("PATH")),
		Model:       strings.TrimSpace(os.Getenv(envCodexModelV0)),
		ReasoningEffort: codexReasoningEffortPolicyV0(envOrDefaultV0(
			envCodexReasoningEffortV0,
			string(orquestacoreworkflow.OrchestrationCapacityMediumV0),
		)),
		Profile:                  strings.TrimSpace(os.Getenv(envCodexProfileV0)),
		Sandbox:                  codexSandboxFromEnvV0(envCodexSandboxV0, "danger-full-access"),
		ApprovalPolicy:           envOrDefaultV0(envCodexApprovalPolicyV0, "never"),
		DirectorSandbox:          codexOptionalSandboxFromEnvV0(envCodexDirectorSandboxV0),
		DirectorApprovalPolicy:   strings.TrimSpace(os.Getenv(envCodexDirectorApprovalPolicyV0)),
		InteractiveApprovalOptIn: boolEnvOrDefaultV0(envCodexAllowInteractiveApprovalV0, false),
		ExtraArgs:                strings.Fields(os.Getenv(envCodexExtraArgsV0)),
		Limits:                   codexRuntimeLimitsFromEnvV0(),
		WaitInterval:             time.Duration(intEnvOrDefaultV0(envCodexWaitIntervalMSV0, defaultCodexWaitIntervalMSV0)) * time.Millisecond,
		ProgressPolicy: orquestaruntime.AgentProgressHeartbeatPolicyV0{
			StalledAfterNoProgressTicks: intEnvOrDefaultV0(envCodexStalledTicksV0, defaultCodexStalledTicksV0),
			LoopAfterRepeatedActions:    intEnvOrDefaultV0(envCodexLoopTicksV0, defaultCodexLoopTicksV0),
		},
		ProgressBudget: orquestaruntimecodexdelivery.CodexBudgetActivityPolicyV0{
			MaxExpected:     time.Duration(intEnvOrDefaultV0(envCodexMaxExpectedSecondsV0, defaultCodexMaxExpectedSecondsV0)) * time.Second,
			NoActivityLimit: time.Duration(intEnvOrDefaultV0(envCodexNoActivitySecondsV0, defaultCodexNoActivitySecondsV0)) * time.Second,
		},
	}
}

type codexRuntimeLimitsV0 struct {
	MaxBatchReady    int
	MaxLiveProcesses int
}

func codexRuntimeLimitsFromEnvV0() codexRuntimeLimitsV0 {
	return codexRuntimeLimitsV0{
		MaxBatchReady:    intEnvOrDefaultV0(envCodexMaxBatchReadyV0, defaultCodexMaxBatchReadyV0),
		MaxLiveProcesses: intEnvOrDefaultV0(envCodexMaxConcurrencyV0, defaultCodexMaxConcurrencyV0),
	}
}
