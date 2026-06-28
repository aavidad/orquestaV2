package main

import (
	"encoding/json"
	"os"
	"strings"
	"time"

	orquestaappcodexstack "orquesta/modulos/orquesta-app-codex-stack"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
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
		SkillInstructions:        envConfig.SkillInstructions,
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
	SkillInstructions        []orquestaruntimecodex.CodexSkillInstructionV0
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
		// Default medium: Goal-first y la recuperacion de artefacto-sin-ACK
		// reducen el coste de exigir high por defecto. Las composiciones pueden
		// subirlo con ORQUESTA_CODEX_REASONING_EFFORT cuando el riesgo lo justifique.
		ReasoningEffort: codexReasoningEffortPolicyV0(envOrDefaultV0(
			envCodexReasoningEffortV0,
			string(orquestacoreworkflow.OrchestrationCapacityMediumV0),
		)),
		Profile:                  strings.TrimSpace(os.Getenv(envCodexProfileV0)),
		Sandbox:                  codexSandboxFromEnvV0(envCodexSandboxV0, "workspace-write"),
		ApprovalPolicy:           envOrDefaultV0(envCodexApprovalPolicyV0, "never"),
		DirectorSandbox:          codexOptionalSandboxFromEnvV0(envCodexDirectorSandboxV0),
		DirectorApprovalPolicy:   strings.TrimSpace(os.Getenv(envCodexDirectorApprovalPolicyV0)),
		InteractiveApprovalOptIn: boolEnvOrDefaultV0(envCodexAllowInteractiveApprovalV0, false),
		ExtraArgs:                strings.Fields(os.Getenv(envCodexExtraArgsV0)),
		SkillInstructions:        codexSkillInstructionsFromEnvV0(),
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

func codexSkillInstructionsFromEnvV0() []orquestaruntimecodex.CodexSkillInstructionV0 {
	raw := strings.TrimSpace(os.Getenv(envCodexSkillInstructionsJSONV0))
	if raw == "" {
		return nil
	}
	var instructions []orquestaruntimecodex.CodexSkillInstructionV0
	if err := json.Unmarshal([]byte(raw), &instructions); err != nil {
		return nil
	}
	return compactCodexSkillInstructionsV0(instructions)
}

func compactCodexSkillInstructionsV0(
	instructions []orquestaruntimecodex.CodexSkillInstructionV0,
) []orquestaruntimecodex.CodexSkillInstructionV0 {
	seen := map[string]bool{}
	result := make([]orquestaruntimecodex.CodexSkillInstructionV0, 0, len(instructions))
	for _, instruction := range instructions {
		ref := strings.TrimSpace(instruction.SkillRef)
		text := strings.TrimSpace(instruction.Text)
		key := strings.ToLower(ref)
		if ref == "" || text == "" || seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, orquestaruntimecodex.CodexSkillInstructionV0{
			SkillRef: ref,
			Text:     text,
		})
	}
	return result
}

type codexRuntimeLimitsV0 struct {
	MaxBatchReady    int
	MaxLiveProcesses int
}

func codexRuntimeLimitsFromEnvV0() codexRuntimeLimitsV0 {
	executionMode := codexExecutionModeFromEnvV0()
	return codexRuntimeLimitsV0{
		MaxBatchReady:    codexExecutionModeCapPositiveV0(executionMode, intEnvOrDefaultV0(envCodexMaxBatchReadyV0, defaultCodexMaxBatchReadyV0)),
		MaxLiveProcesses: codexExecutionModeCapPositiveV0(executionMode, intEnvOrDefaultV0(envCodexMaxConcurrencyV0, defaultCodexMaxConcurrencyV0)),
	}
}
