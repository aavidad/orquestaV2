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
	if !boolEnvOrDefaultV0(envServerAutoprogrammingPromotionGuardianEnabledV0, false) {
		return serverAutoprogrammingPromotionGuardianV0{}
	}
	command := envOrDefaultV0(
		envServerAutoprogrammingPromotionGuardianCommandV0,
		defaultAutoprogrammingPromotionGuardianCommandV0,
	)
	guardian := serverAutoprogrammingPromotionGuardianV0{
		Enabled:            true,
		Command:            command,
		RunnerEnvAllowlist: autoprogrammingPromotionGuardianRunnerEnvAllowlistFromEnvV0(),
		StateDir:           absDirEnvOrDefaultV0(envServerAutoprogrammingPromotionGuardianStateDirV0, filepath.Join(config.StateDir, "guardian")),
		CurrentBin:         strings.TrimSpace(os.Getenv(envServerAutoprogrammingPromotionGuardianCurrentBinV0)),
		CandidateBin:       strings.TrimSpace(os.Getenv(envServerAutoprogrammingPromotionGuardianCandidateBinV0)),
		LastGoodBin:        strings.TrimSpace(os.Getenv(envServerAutoprogrammingPromotionGuardianLastGoodBinV0)),
		ArtifactRoot:       strings.TrimSpace(os.Getenv(envServerAutoprogrammingPromotionGuardianArtifactRootV0)),
		BuildCommand:       strings.TrimSpace(os.Getenv(envServerAutoprogrammingPromotionGuardianBuildCommandV0)),
		TestCommands:       csvEnvOrDefaultV0(envServerAutoprogrammingPromotionGuardianTestCommandsV0, nil),
		HealthTimeout:      strings.TrimSpace(os.Getenv(envServerAutoprogrammingPromotionGuardianHealthTimeoutV0)),
		CommandTimeout:     strings.TrimSpace(os.Getenv(envServerAutoprogrammingPromotionGuardianCommandTimeoutV0)),
		ArtifactMaxBytes:   strings.TrimSpace(os.Getenv(envServerAutoprogrammingPromotionGuardianArtifactMaxBytesV0)),
		RepairCommand:      strings.TrimSpace(os.Getenv(envServerAutoprogrammingPromotionGuardianRepairCommandV0)),
		RepairCodex:        boolEnvOrDefaultV0(envServerAutoprogrammingPromotionGuardianRepairCodexV0, false),
		RepairCodexWriteSet: csvEnvOrDefaultV0(
			envServerAutoprogrammingPromotionGuardianRepairCodexWriteSetV0,
			nil,
		),
		RepairCodexRequiredTests: csvEnvOrDefaultV0(
			envServerAutoprogrammingPromotionGuardianRepairCodexRequiredTestsV0,
			nil,
		),
		RepairCodexWorktreeRef:     strings.TrimSpace(os.Getenv(envServerAutoprogrammingPromotionGuardianRepairCodexWorktreeRefV0)),
		RepairCodexBranchRef:       strings.TrimSpace(os.Getenv(envServerAutoprogrammingPromotionGuardianRepairCodexBranchRefV0)),
		RepairCodexRunRef:          strings.TrimSpace(os.Getenv(envServerAutoprogrammingPromotionGuardianRepairCodexRunRefV0)),
		RepairCodexPromotionRef:    strings.TrimSpace(os.Getenv(envServerAutoprogrammingPromotionGuardianRepairCodexPromotionRefV0)),
		RepairCodexSandbox:         strings.TrimSpace(os.Getenv(envServerAutoprogrammingPromotionGuardianRepairCodexSandboxV0)),
		RepairCodexReasoning:       strings.TrimSpace(os.Getenv(envServerAutoprogrammingPromotionGuardianRepairCodexReasoningV0)),
		RepairCodexRuntimeDir:      strings.TrimSpace(os.Getenv(envServerAutoprogrammingPromotionGuardianRepairCodexRuntimeDirV0)),
		RepairCodexAllowBroad:      boolEnvOrDefaultV0(envServerAutoprogrammingPromotionGuardianRepairCodexAllowBroadV0, false),
		RepairCodexSandboxEvidence: strings.TrimSpace(os.Getenv(envServerAutoprogrammingPromotionGuardianRepairCodexSandboxEvidenceV0)),
		CommandEffectEvidenceRefs: csvEnvOrDefaultV0(
			envServerAutoprogrammingPromotionGuardianCommandEffectEvidenceRefsV0,
			nil,
		),
		SkipHealthEvidenceRefs: csvEnvOrDefaultV0(
			envServerAutoprogrammingPromotionGuardianSkipHealthEvidenceRefsV0,
			nil,
		),
	}
	guardian.Runner = shellAutoprogrammingPromotionGuardianRunnerV0{
		Command:      command,
		EnvAllowlist: append([]string(nil), guardian.RunnerEnvAllowlist...),
	}
	return guardian
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
