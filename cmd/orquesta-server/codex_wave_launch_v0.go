package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
)

const codexWaveProcessDoneFileNameV0 = orquestaruntimecodex.CodexProcessDoneFileNameV0

func runCodexLaunchWaveV0(
	ctx context.Context,
	config codexWaveConfigV0,
) (codexWaveLaunchSummaryV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	now := time.Now().UTC().Format(time.RFC3339)
	summary := codexWaveLaunchSummaryV0{
		SchemaVersion:  codexWaveSummarySchemaVersionV0,
		WaveRef:        config.WaveRef,
		AgentCount:     config.Agents,
		ProjectWorkDir: config.ProjectWorkDir,
		RuntimeWorkDir: config.RuntimeWorkDir,
		RegistryPath:   codexWaveRegistryPathV0(config.RuntimeWorkDir),
		Sandbox:        config.Sandbox,
		ApprovalPolicy: config.ApprovalPolicy,
		CreatedAt:      now,
		UpdatedAt:      now,
		DryRun:         config.DryRun,
		OperatorInputs: codexWaveOperatorInputReceiptsCopyV0(config.OperatorInputs),
		Agents:         make([]codexWaveAgentSummaryV0, 0, config.Agents),
	}
	if err := codexWaveValidateUnmanagedLaunchPolicyV0(config); err != nil {
		summary.Errors = append(summary.Errors, codexWavePublicErrorV0{
			Code:    err.Error(),
			Field:   "unmanaged_launch",
			Message: "los agentes reales deben arrancar desde el servidor/cola de Orquesta; usa dry-run o un breakglass auditado",
		})
		return summary, nil
	}
	if config.PurgeRuntime {
		report, err := codexWaveApplyRuntimePurgeV0(config)
		summary.PurgeReport = report
		if err != nil {
			return summary, err
		}
	}
	if err := os.MkdirAll(config.RuntimeWorkDir, 0o700); err != nil {
		return codexWaveLaunchSummaryV0{}, err
	}
	for i := 1; i <= config.Agents; i++ {
		codexWaveLaunchOneAgentV0(ctx, config, &summary, i)
	}
	codexWaveRefreshSummaryV0(&summary)
	if err := codexWaveSaveRegistryV0(summary); err != nil {
		return codexWaveLaunchSummaryV0{}, err
	}
	return summary, nil
}

func codexWaveValidateUnmanagedLaunchPolicyV0(config codexWaveConfigV0) error {
	if config.DryRun {
		return nil
	}
	if !config.AllowUnmanagedLaunch {
		return errors.New("unmanaged_launch_blocked")
	}
	if strings.TrimSpace(config.UnmanagedLaunchReason) == "" {
		return errors.New("unmanaged_launch_reason_required")
	}
	if strings.TrimSpace(config.UnmanagedLaunchConfirm) != strings.TrimSpace(config.WaveRef) {
		return errors.New("unmanaged_launch_confirmation_required")
	}
	return nil
}

func codexWaveLaunchOneAgentV0(
	ctx context.Context,
	config codexWaveConfigV0,
	summary *codexWaveLaunchSummaryV0,
	index int,
) {
	agent, err := codexWaveMaterializeAgentV0(config, index)
	if err != nil {
		summary.Errors = append(summary.Errors, codexWavePublicErrorV0{
			AgentRef: fmt.Sprintf("%s-agent-%02d", config.WaveRef, index),
			Code:     "materialize_failed",
			Message:  err.Error(),
		})
		return
	}
	if config.DryRun {
		agent.Status = "dry_run"
		summary.Agents = append(summary.Agents, agent)
		return
	}
	pid, err := codexWaveStartAgentProcessV0(ctx, agent.WrapperPath, config.ProjectWorkDir)
	if err != nil {
		agent.Status = "launch_failed"
		summary.Errors = append(summary.Errors, codexWavePublicErrorV0{
			AgentRef: agent.AgentRef,
			Code:     "launch_failed",
			Message:  err.Error(),
		})
		summary.Agents = append(summary.Agents, agent)
		return
	}
	agent.ProcessRef = fmt.Sprintf("%s-process-%02d", config.WaveRef, index)
	agent.SessionRef = fmt.Sprintf("%s-session-%02d", config.WaveRef, index)
	agent.LaunchRef = fmt.Sprintf("%s-launch-%02d", config.WaveRef, index)
	agent.PID = pid
	agent.StartedAt = time.Now().UTC().Format(time.RFC3339)
	if err := codexWaveAttachProcessProofV0(&agent, config, index); err != nil {
		_ = signalProcessV0(pid)
		agent.Status = "launch_failed"
		summary.Errors = append(summary.Errors, codexWaveProcessProofAttachErrorV0(agent.AgentRef, err))
		summary.Agents = append(summary.Agents, agent)
		return
	}
	agent.Status = "running"
	summary.Agents = append(summary.Agents, agent)
}

func codexWaveStartAgentProcessV0(ctx context.Context, wrapperPath string, projectWorkDir string) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	donePath := codexWaveProcessDonePathV0(wrapperPath)
	_ = os.Remove(donePath)
	cmd := exec.Command(wrapperPath)
	cmd.Dir = projectWorkDir
	cmd.Env = []string{}
	cmd.Stdin = nil
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	configureDetachedProcessV0(cmd)
	if err := cmd.Start(); err != nil {
		return 0, err
	}
	pid := cmd.Process.Pid
	go func() {
		err := cmd.Wait()
		status := []byte("completed\n")
		if err != nil {
			status = []byte("failed\n")
		}
		_ = codexWaveWriteControlFileV0(filepath.Dir(donePath), donePath, codexWaveProcessDoneFileNameV0, "codex_wave_process_done", status, 0o600)
	}()
	return pid, nil
}

func codexWaveProcessDonePathV0(wrapperPath string) string {
	return filepath.Join(filepath.Dir(wrapperPath), codexWaveProcessDoneFileNameV0)
}
func codexWaveMaterializeAgentV0(
	config codexWaveConfigV0,
	index int,
) (codexWaveAgentSummaryV0, error) {
	agentRef := fmt.Sprintf("%s-agent-%02d", config.WaveRef, index)
	agentRuntimeDir := filepath.Join(config.RuntimeWorkDir, fmt.Sprintf("agent-%02d", index))
	if err := os.MkdirAll(agentRuntimeDir, 0o700); err != nil {
		return codexWaveAgentSummaryV0{}, err
	}
	homeDir, codeHomeDir, credentialProjection, err := codexWaveAgentHomesV0(config, agentRuntimeDir)
	if err != nil {
		return codexWaveAgentSummaryV0{}, err
	}
	profile := codexWaveConnectorProfileV0(config, agentRuntimeDir, codeHomeDir, homeDir, index)
	if issues := orquestaruntimecodex.ValidateCodexConnectorProfileV0(profile); len(issues) > 0 {
		return codexWaveAgentSummaryV0{}, fmt.Errorf("codex_profile_invalid:%s", issues[0].Field)
	}
	return codexWaveWriteAgentFilesV0(config, profile, agentRef, agentRuntimeDir, homeDir, codeHomeDir, credentialProjection, index)
}

func codexWaveAgentHomesV0(config codexWaveConfigV0, agentRuntimeDir string) (string, string, *codexWaveCredentialProjectionReceiptV0, error) {
	if config.IsolateHome {
		return codexWaveCopyCodeHomeV0(config.SourceCodeHome, agentRuntimeDir, config.CredentialProjectionPolicy)
	}
	return homeDirV0(), config.SourceCodeHome, nil, nil
}

func codexWaveConnectorProfileV0(
	config codexWaveConfigV0,
	agentRuntimeDir string,
	codeHomeDir string,
	homeDir string,
	index int,
) orquestaruntimecodex.CodexConnectorProfileV0 {
	return orquestaruntimecodex.CodexConnectorProfileV0{
		SchemaVersion:   orquestaruntimecodex.CodexConnectorProfileSchemaVersionV0,
		OptIn:           true,
		CommandPath:     config.CommandPath,
		ProjectWorkDir:  config.ProjectWorkDir,
		RuntimeWorkDir:  agentRuntimeDir,
		CodeHomeDir:     codeHomeDir,
		HomeDir:         homeDir,
		PathEnv:         config.PathEnv,
		Model:           config.Model,
		ReasoningEffort: config.ReasoningEffort,
		Profile:         config.Profile,
		Sandbox:         config.Sandbox,
		ApprovalPolicy:  config.ApprovalPolicy,
		ExtraArgs:       append([]string(nil), config.ExtraArgs...),
		PromptHints: []string{
			"codex-launch-wave",
			fmt.Sprintf("wave=%s", config.WaveRef),
			fmt.Sprintf("agent=%d/%d", index, config.Agents),
		},
	}
}

func codexWaveWriteAgentFilesV0(
	config codexWaveConfigV0,
	profile orquestaruntimecodex.CodexConnectorProfileV0,
	agentRef string,
	agentRuntimeDir string,
	homeDir string,
	codeHomeDir string,
	credentialProjection *codexWaveCredentialProjectionReceiptV0,
	index int,
) (codexWaveAgentSummaryV0, error) {
	promptPath := filepath.Join(agentRuntimeDir, orquestaruntimecodex.CodexAgentPromptFileNameV0)
	wrapperPath := filepath.Join(agentRuntimeDir, orquestaruntimecodex.CodexWrapperFileNameV0)
	if err := codexWaveWriteControlFileV0(agentRuntimeDir, promptPath, orquestaruntimecodex.CodexAgentPromptFileNameV0, "agent_prompt", []byte(codexWaveAgentPromptV0(config, agentRef, index)), 0o600); err != nil {
		return codexWaveAgentSummaryV0{}, err
	}
	if err := codexWaveWriteControlFileV0(agentRuntimeDir, wrapperPath, orquestaruntimecodex.CodexWrapperFileNameV0, "codex_wrapper", []byte(orquestaruntimecodex.BuildCodexWrapperScriptV0(profile)), 0o700); err != nil {
		return codexWaveAgentSummaryV0{}, err
	}
	return codexWaveAgentSummaryV0{
		AgentRef:             agentRef,
		RuntimeWorkDir:       agentRuntimeDir,
		PromptPath:           promptPath,
		WrapperPath:          wrapperPath,
		StdoutPath:           filepath.Join(agentRuntimeDir, orquestaruntimecodex.CodexStdoutFileNameV0),
		StderrPath:           filepath.Join(agentRuntimeDir, orquestaruntimecodex.CodexStderrFileNameV0),
		LastMessagePath:      filepath.Join(agentRuntimeDir, orquestaruntimecodex.CodexLastMessageFileNameV0),
		HomeDir:              homeDir,
		CodeHomeDir:          codeHomeDir,
		Status:               "materialized",
		CredentialProjection: credentialProjection,
	}, nil
}

func codexWaveWriteControlFileV0(
	rootDir string,
	path string,
	fileName string,
	controlKind string,
	data []byte,
	perm os.FileMode,
) error {
	_, err := orquestaruntimecodex.WriteCodexControlFileBytesV0(orquestaruntimecodex.CodexControlFileWriteRequestV0{
		RootDir:     rootDir,
		Path:        path,
		FileName:    fileName,
		ControlKind: controlKind,
		Data:        data,
		Mode:        orquestaruntimecodex.CodexControlFileWriteCreateOrReplaceV0,
		Perm:        perm,
	})
	return err
}
