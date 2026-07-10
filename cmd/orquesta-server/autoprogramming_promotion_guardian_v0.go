package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestaserver "orquesta/modulos/orquesta-server"
)

const defaultAutoprogrammingPromotionGuardianCommandV0 = "go run ./cmd/orquesta-guardian check-promote"

func init() {
	for key, label := range map[string]string{
		envServerAutoprogrammingPromotionGuardianEnabledV0:                    "Guardian activo",
		envServerAutoprogrammingPromotionGuardianCommandV0:                    "Comando guardian",
		envServerAutoprogrammingPromotionGuardianRunnerEnvAllowlistV0:         "Allowlist de entorno guardian",
		envServerAutoprogrammingPromotionGuardianStateDirV0:                   "Estado guardian",
		envServerAutoprogrammingPromotionGuardianCurrentBinV0:                 "Binario actual guardian",
		envServerAutoprogrammingPromotionGuardianCandidateBinV0:               "Binario candidato guardian",
		envServerAutoprogrammingPromotionGuardianLastGoodBinV0:                "Ultimo binario valido guardian",
		envServerAutoprogrammingPromotionGuardianArtifactRootV0:               "Artefactos guardian",
		envServerAutoprogrammingPromotionGuardianBuildCommandV0:               "Build guardian",
		envServerAutoprogrammingPromotionGuardianTestCommandsV0:               "Tests guardian",
		envServerAutoprogrammingPromotionGuardianArtifactMaxBytesV0:           "Maximo artefacto guardian",
		envServerAutoprogrammingPromotionGuardianRepairCommandV0:              "Comando reparacion guardian",
		envServerAutoprogrammingPromotionGuardianRepairCodexV0:                "Reparacion Codex guardian",
		envServerAutoprogrammingPromotionGuardianRepairCodexWriteSetV0:        "Write set reparacion guardian",
		envServerAutoprogrammingPromotionGuardianRepairCodexRequiredTestsV0:   "Tests reparacion guardian",
		envServerAutoprogrammingPromotionGuardianRepairCodexWorktreeRefV0:     "Worktree reparacion guardian",
		envServerAutoprogrammingPromotionGuardianRepairCodexBranchRefV0:       "Rama reparacion guardian",
		envServerAutoprogrammingPromotionGuardianRepairCodexRunRefV0:          "Run reparacion guardian",
		envServerAutoprogrammingPromotionGuardianRepairCodexPromotionRefV0:    "Promocion reparacion guardian",
		envServerAutoprogrammingPromotionGuardianRepairCodexSandboxV0:         "Sandbox reparacion guardian",
		envServerAutoprogrammingPromotionGuardianRepairCodexReasoningV0:       "Razonamiento reparacion guardian",
		envServerAutoprogrammingPromotionGuardianRepairCodexRuntimeDirV0:      "Runtime reparacion guardian",
		envServerAutoprogrammingPromotionGuardianRepairCodexAllowBroadV0:      "Sandbox amplio reparacion guardian",
		envServerAutoprogrammingPromotionGuardianRepairCodexSandboxEvidenceV0: "Evidencia sandbox guardian",
		envServerAutoprogrammingPromotionGuardianCommandEffectEvidenceRefsV0:  "Evidencia de efecto guardian",
		envServerAutoprogrammingPromotionGuardianSkipHealthEvidenceRefsV0:     "Evidencia omitir salud guardian",
	} {
		serverEffectiveEnvRegistryV0[key] = serverEnvSettingMetadataV0{
			Scope:       "autoprogramming_promotion_guardian",
			Label:       label,
			Description: "Parametro tipado del guardian de promocion; solo se proyecta al proceso hijo mediante allowlist.",
		}
	}
}

type serverAutoprogrammingPromotionGuardianV0 struct {
	Enabled                    bool
	Command                    string
	RunnerEnvAllowlist         []string
	StateDir                   string
	CurrentBin                 string
	CandidateBin               string
	LastGoodBin                string
	ArtifactRoot               string
	BuildCommand               string
	TestCommands               []string
	HealthTimeout              string
	CommandTimeout             string
	ArtifactMaxBytes           string
	RepairCommand              string
	RepairCodex                bool
	RepairCodexWriteSet        []string
	RepairCodexRequiredTests   []string
	RepairCodexWorktreeRef     string
	RepairCodexBranchRef       string
	RepairCodexRunRef          string
	RepairCodexPromotionRef    string
	RepairCodexSandbox         string
	RepairCodexReasoning       string
	RepairCodexRuntimeDir      string
	RepairCodexAllowBroad      bool
	RepairCodexSandboxEvidence string
	CommandEffectEvidenceRefs  []string
	SkipHealthEvidenceRefs     []string
	Runner                     serverAutoprogrammingPromotionGuardianRunnerV0
}

type serverAutoprogrammingPromotionGuardianRequestV0 struct {
	ProjectDir                 string
	StateDir                   string
	CurrentBin                 string
	CandidateBin               string
	LastGoodBin                string
	ArtifactRoot               string
	BuildCommand               string
	TestCommands               []string
	HealthTimeout              string
	CommandTimeout             string
	ArtifactMaxBytes           string
	RepairCommand              string
	RepairCodex                bool
	RepairCodexWriteSet        []string
	RepairCodexRequiredTests   []string
	RepairCodexWorktreeRef     string
	RepairCodexBranchRef       string
	RepairCodexRunRef          string
	RepairCodexPromotionRef    string
	RepairCodexSandbox         string
	RepairCodexReasoning       string
	RepairCodexRuntimeDir      string
	RepairCodexAllowBroad      bool
	RepairCodexSandboxEvidence string
	CommandEffectEvidenceRefs  []string
	SkipHealthEvidenceRefs     []string
	PromotionRef               string
	RunRef                     string
	ProjectRef                 string
	AppRef                     string
	RepoRef                    string
	WorktreeRef                string
	BranchRef                  string
	EvidenceRefs               []string
}

type serverAutoprogrammingPromotionGuardianResultV0 struct {
	Status             string
	Phase              string
	Promote            bool
	Promoted           bool
	Restored           bool
	VerifiedOnly       bool
	BreakglassPromoted bool
	ManifestRef        string
	RepairPacketRef    string
	RepairAttemptRef   string
	FailurePacketHash  string
	RepairBlocked      bool
	RepairBlockReason  string
	ReasonCodes        []string
	ProjectRef         string
	AppRef             string
	RepoRef            string
	PromotionRef       string
	RunRef             string
	WorktreeRef        string
	BranchRef          string
	CandidateHash      string
	CandidateSizeBytes int64
	EvidenceRefs       []string
}

type serverAutoprogrammingPromotionGuardianRunnerV0 interface {
	CheckAutoprogrammingPromotionGuardianV0(
		context.Context,
		serverAutoprogrammingPromotionGuardianRequestV0,
	) (serverAutoprogrammingPromotionGuardianResultV0, error)
}

type shellAutoprogrammingPromotionGuardianRunnerV0 struct {
	Command      string
	EnvAllowlist []string
}

func autoprogrammingPromotionGuardianFromEnvV0(
	config orquestaserver.ConfigV0,
) serverAutoprogrammingPromotionGuardianV0 {
	projectConfig := projectConfigFromServerConfigBestEffortV0(config)
	guardianConfig := projectConfig.Autoprogramming.Promotion.Guardian
	if !autoprogrammingPromotionGuardianEnabledFromProjectConfigV0(guardianConfig) {
		return serverAutoprogrammingPromotionGuardianV0{}
	}
	command := stringProjectConfigOrEnvOrDefaultV0(
		envServerAutoprogrammingPromotionGuardianCommandV0,
		guardianConfig.Command,
		defaultAutoprogrammingPromotionGuardianCommandV0,
	)
	guardian := serverAutoprogrammingPromotionGuardianV0{
		Enabled: true,
		Command: command,
		RunnerEnvAllowlist: autoprogrammingPromotionGuardianStringsFromProjectConfigV0(
			envServerAutoprogrammingPromotionGuardianRunnerEnvAllowlistV0,
			guardianConfig.RunnerEnvAllowlist,
			defaultAutoprogrammingPromotionGuardianRunnerEnvAllowlistV0(),
		),
		StateDir: absDirProjectConfigOrEnvOrDefaultV0(
			envServerAutoprogrammingPromotionGuardianStateDirV0,
			guardianConfig.StateDir,
			filepath.Join(config.StateDir, "guardian"),
		),
		CurrentBin: stringProjectConfigOrEnvOrDefaultV0(
			envServerAutoprogrammingPromotionGuardianCurrentBinV0, guardianConfig.CurrentBin, "",
		),
		CandidateBin: stringProjectConfigOrEnvOrDefaultV0(
			envServerAutoprogrammingPromotionGuardianCandidateBinV0, guardianConfig.CandidateBin, "",
		),
		LastGoodBin: stringProjectConfigOrEnvOrDefaultV0(
			envServerAutoprogrammingPromotionGuardianLastGoodBinV0, guardianConfig.LastGoodBin, "",
		),
		ArtifactRoot: stringProjectConfigOrEnvOrDefaultV0(
			envServerAutoprogrammingPromotionGuardianArtifactRootV0, guardianConfig.ArtifactRoot, "",
		),
		BuildCommand: stringProjectConfigOrEnvOrDefaultV0(
			envServerAutoprogrammingPromotionGuardianBuildCommandV0, guardianConfig.BuildCommand, "",
		),
		TestCommands: autoprogrammingPromotionGuardianStringsFromProjectConfigV0(
			envServerAutoprogrammingPromotionGuardianTestCommandsV0, guardianConfig.TestCommands, nil,
		),
		HealthTimeout:  stringProjectConfigFileOrDefaultV0(guardianConfig.HealthTimeout, ""),
		CommandTimeout: stringProjectConfigFileOrDefaultV0(guardianConfig.CommandTimeout, ""),
		ArtifactMaxBytes: stringProjectConfigOrEnvOrDefaultV0(
			envServerAutoprogrammingPromotionGuardianArtifactMaxBytesV0, guardianConfig.ArtifactMaxBytes, "",
		),
		RepairCommand: stringProjectConfigOrEnvOrDefaultV0(
			envServerAutoprogrammingPromotionGuardianRepairCommandV0, guardianConfig.RepairCommand, "",
		),
		RepairCodex: boolProjectConfigOrEnvOrDefaultV0(
			envServerAutoprogrammingPromotionGuardianRepairCodexV0, guardianConfig.RepairCodex, false,
		),
		RepairCodexWriteSet: autoprogrammingPromotionGuardianStringsFromProjectConfigV0(
			envServerAutoprogrammingPromotionGuardianRepairCodexWriteSetV0,
			guardianConfig.RepairCodexWriteSet,
			nil,
		),
		RepairCodexRequiredTests: autoprogrammingPromotionGuardianStringsFromProjectConfigV0(
			envServerAutoprogrammingPromotionGuardianRepairCodexRequiredTestsV0,
			guardianConfig.RepairCodexRequiredTests,
			nil,
		),
		RepairCodexWorktreeRef:     stringProjectConfigOrEnvOrDefaultV0(envServerAutoprogrammingPromotionGuardianRepairCodexWorktreeRefV0, guardianConfig.RepairCodexWorktreeRef, ""),
		RepairCodexBranchRef:       stringProjectConfigOrEnvOrDefaultV0(envServerAutoprogrammingPromotionGuardianRepairCodexBranchRefV0, guardianConfig.RepairCodexBranchRef, ""),
		RepairCodexRunRef:          stringProjectConfigOrEnvOrDefaultV0(envServerAutoprogrammingPromotionGuardianRepairCodexRunRefV0, guardianConfig.RepairCodexRunRef, ""),
		RepairCodexPromotionRef:    stringProjectConfigOrEnvOrDefaultV0(envServerAutoprogrammingPromotionGuardianRepairCodexPromotionRefV0, guardianConfig.RepairCodexPromotionRef, ""),
		RepairCodexSandbox:         stringProjectConfigOrEnvOrDefaultV0(envServerAutoprogrammingPromotionGuardianRepairCodexSandboxV0, guardianConfig.RepairCodexSandbox, ""),
		RepairCodexReasoning:       stringProjectConfigOrEnvOrDefaultV0(envServerAutoprogrammingPromotionGuardianRepairCodexReasoningV0, guardianConfig.RepairCodexReasoning, ""),
		RepairCodexRuntimeDir:      stringProjectConfigOrEnvOrDefaultV0(envServerAutoprogrammingPromotionGuardianRepairCodexRuntimeDirV0, guardianConfig.RepairCodexRuntimeDir, ""),
		RepairCodexAllowBroad:      boolProjectConfigOrEnvOrDefaultV0(envServerAutoprogrammingPromotionGuardianRepairCodexAllowBroadV0, guardianConfig.RepairCodexAllowBroad, false),
		RepairCodexSandboxEvidence: stringProjectConfigOrEnvOrDefaultV0(envServerAutoprogrammingPromotionGuardianRepairCodexSandboxEvidenceV0, guardianConfig.RepairCodexSandboxEvidence, ""),
		CommandEffectEvidenceRefs: autoprogrammingPromotionGuardianStringsFromProjectConfigV0(
			envServerAutoprogrammingPromotionGuardianCommandEffectEvidenceRefsV0,
			guardianConfig.CommandEffectEvidenceRefs,
			nil,
		),
		SkipHealthEvidenceRefs: autoprogrammingPromotionGuardianStringsFromProjectConfigV0(
			envServerAutoprogrammingPromotionGuardianSkipHealthEvidenceRefsV0,
			guardianConfig.SkipHealthEvidenceRefs,
			nil,
		),
	}
	guardian.Runner = shellAutoprogrammingPromotionGuardianRunnerV0{
		Command:      command,
		EnvAllowlist: append([]string(nil), guardian.RunnerEnvAllowlist...),
	}
	return guardian
}

func autoprogrammingPromotionGuardianEnabledFromProjectConfigV0(
	guardian serverProjectConfigAutoprogrammingPromotionGuardianV0,
) bool {
	if _, ok := os.LookupEnv(envServerAutoprogrammingPromotionGuardianEnabledV0); ok {
		return boolEnvOrDefaultV0(envServerAutoprogrammingPromotionGuardianEnabledV0, false)
	}
	return guardian.Enabled != nil && *guardian.Enabled
}

func autoprogrammingPromotionGuardianStringsFromProjectConfigV0(
	key string,
	values []string,
	fallback []string,
) []string {
	if raw := strings.TrimSpace(os.Getenv(key)); raw != "" {
		return csvEnvOrDefaultV0(key, fallback)
	}
	if len(values) == 0 {
		return append([]string(nil), fallback...)
	}
	return compactStringsV0(values)
}

func autoprogrammingPromotionGuardianRequestV0(
	port serverAutoprogrammingPromotionPortV0,
	command orquestaautoprogramming.AutoprogrammingStagingPromotionCommandV0,
) serverAutoprogrammingPromotionGuardianRequestV0 {
	return serverAutoprogrammingPromotionGuardianRequestV0{
		ProjectDir:       port.ProjectWorkDir,
		StateDir:         port.Guardian.StateDir,
		CurrentBin:       port.Guardian.CurrentBin,
		CandidateBin:     port.Guardian.CandidateBin,
		LastGoodBin:      port.Guardian.LastGoodBin,
		ArtifactRoot:     port.Guardian.ArtifactRoot,
		BuildCommand:     port.Guardian.BuildCommand,
		TestCommands:     append([]string(nil), port.Guardian.TestCommands...),
		HealthTimeout:    port.Guardian.HealthTimeout,
		CommandTimeout:   port.Guardian.CommandTimeout,
		ArtifactMaxBytes: port.Guardian.ArtifactMaxBytes,
		RepairCommand:    port.Guardian.RepairCommand,
		RepairCodex:      port.Guardian.RepairCodex,
		RepairCodexWriteSet: append(
			append([]string(nil), port.Guardian.RepairCodexWriteSet...),
			command.WriteSet...,
		),
		RepairCodexRequiredTests:   append([]string(nil), port.Guardian.RepairCodexRequiredTests...),
		RepairCodexWorktreeRef:     firstNonEmptyV0(port.Guardian.RepairCodexWorktreeRef, command.WorktreeRef),
		RepairCodexBranchRef:       firstNonEmptyV0(port.Guardian.RepairCodexBranchRef, command.BranchRef),
		RepairCodexRunRef:          firstNonEmptyV0(port.Guardian.RepairCodexRunRef, command.RunRef),
		RepairCodexPromotionRef:    firstNonEmptyV0(port.Guardian.RepairCodexPromotionRef, command.PromotionRef),
		RepairCodexSandbox:         port.Guardian.RepairCodexSandbox,
		RepairCodexReasoning:       port.Guardian.RepairCodexReasoning,
		RepairCodexRuntimeDir:      port.Guardian.RepairCodexRuntimeDir,
		RepairCodexAllowBroad:      port.Guardian.RepairCodexAllowBroad,
		RepairCodexSandboxEvidence: port.Guardian.RepairCodexSandboxEvidence,
		CommandEffectEvidenceRefs: append(
			append([]string(nil), port.Guardian.CommandEffectEvidenceRefs...),
			command.EvidenceRefs...,
		),
		SkipHealthEvidenceRefs: append([]string(nil), port.Guardian.SkipHealthEvidenceRefs...),
		PromotionRef:           command.PromotionRef,
		RunRef:                 command.RunRef,
		ProjectRef:             command.ProjectRef,
		AppRef:                 port.AppRef,
		RepoRef:                port.RepoRef,
		WorktreeRef:            command.WorktreeRef,
		BranchRef:              command.BranchRef,
		EvidenceRefs:           append([]string(nil), command.EvidenceRefs...),
	}
}

func (runner shellAutoprogrammingPromotionGuardianRunnerV0) CheckAutoprogrammingPromotionGuardianV0(
	ctx context.Context,
	request serverAutoprogrammingPromotionGuardianRequestV0,
) (serverAutoprogrammingPromotionGuardianResultV0, error) {
	command := strings.TrimSpace(runner.Command)
	if command == "" {
		return serverAutoprogrammingPromotionGuardianResultV0{}, fmt.Errorf("guardian_command_required")
	}
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd", "/C", command)
	}
	cmd.Dir = request.ProjectDir
	cmd.Env = autoprogrammingPromotionGuardianRunnerEnvV0(
		autoprogrammingPromotionGuardianRunnerParentEnvV0(),
		request,
		runner.EnvAllowlist,
	)
	stdout := newAutoprogrammingPromotionGuardianOutputBufferV0()
	stderr := newAutoprogrammingPromotionGuardianOutputBufferV0()
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	err := cmd.Run()
	if ctx.Err() != nil {
		return serverAutoprogrammingPromotionGuardianResultV0{
			EvidenceRefs: []string{"evidence-ref-autoprogramming-promotion-guardian-timeout"},
		}, fmt.Errorf("guardian_timeout")
	}
	guarded, parseErr := parseAutoprogrammingPromotionGuardianOutputV0(stdout, stderr)
	guarded = autoprogrammingPromotionGuardianResultWithRequestV0(guarded, request)
	if parseErr != nil {
		if reason := autoprogrammingPromotionGuardianFailureCodeV0(stdout.String() + "\n" + stderr.String()); reason != "" {
			invalid := autoprogrammingPromotionGuardianResultWithRequestV0(serverAutoprogrammingPromotionGuardianResultV0{
				EvidenceRefs: []string{"evidence-ref-autoprogramming-promotion-guardian-config-invalid"},
			}, request)
			return invalid, fmt.Errorf("%s", reason)
		}
		invalid := autoprogrammingPromotionGuardianResultWithRequestV0(serverAutoprogrammingPromotionGuardianResultV0{
			Status:       "guardian_result_invalid",
			EvidenceRefs: []string{"evidence-ref-autoprogramming-promotion-guardian-result-invalid"},
		}, request)
		return invalid, parseErr
	}
	if err != nil {
		if autoprogrammingPromotionGuardianResultAcceptedV0(guarded) {
			return guarded, fmt.Errorf("guardian_exit_nonzero")
		}
		return guarded, fmt.Errorf("%s", autoprogrammingPromotionGuardianPublicErrorV0(guarded))
	}
	if !autoprogrammingPromotionGuardianResultAcceptedV0(guarded) {
		return guarded, fmt.Errorf("%s", autoprogrammingPromotionGuardianPublicErrorV0(guarded))
	}
	return guarded, nil
}
