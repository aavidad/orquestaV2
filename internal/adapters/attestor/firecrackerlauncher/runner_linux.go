//go:build linux

package firecrackerlauncher

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"sync"
	"syscall"
)

const (
	firecrackerNoFileLimit = 256
	kvmAPIVersion          = 12
	kvmGetAPIVersionIOCTL  = 0xAE00
)

type runnerNetNamespace interface {
	revalidate() error
	Close() error
}

type launcherCommandFactory func(string, ...string) *exec.Cmd

type physicalRunner struct {
	config      Config
	assets      *assetSet
	namespace   runnerNetNamespace
	cgroups     runnerCgroup
	runtimeRoot *os.File
	runs        *runsRoot
	command     launcherCommandFactory

	mu               sync.Mutex
	closed           bool
	active           map[*activePhysicalRun]struct{}
	residualErr      error
	wait             sync.WaitGroup
	closeMu          sync.Mutex
	resourceClose    sync.Once
	resourceCloseErr error
}

type activePhysicalRun struct {
	cancel  context.CancelFunc
	mu      sync.Mutex
	command *exec.Cmd
	cgroups runnerCgroup
	id      string
}

type boundedDiagnostic struct {
	mu     sync.Mutex
	buffer []byte
	limit  int
}

func (runner *physicalRunner) Run(
	parent context.Context,
	request LaunchRequest,
	inputDrive *os.File,
	outputDrive *os.File,
) (result RunResult, resultErr error) {
	if runner == nil || parent == nil || runner.command == nil ||
		runner.assets == nil || runner.namespace == nil ||
		runner.cgroups == nil || runner.runs == nil ||
		runner.config.validateRequest(request) != nil {
		return RunResult{}, launcherError(CodeUnavailable)
	}
	if _, err := validateInputDriveDescriptor(
		parent,
		inputDrive,
		runner.config.MaxInputBytes,
		request.InputDigest,
	); err != nil {
		return RunResult{}, err
	}
	if _, err := validateEmptyOutputDescriptor(
		outputDrive,
		request.OutputDriveBytes,
	); err != nil {
		return RunResult{}, err
	}
	runContext, cancel := context.WithCancel(parent)
	id, err := runID(request.Nonce)
	if err != nil {
		cancel()
		return RunResult{}, err
	}
	active := &activePhysicalRun{
		cancel: cancel, cgroups: runner.cgroups, id: id,
	}
	if !runner.register(active) {
		cancel()
		return RunResult{}, launcherError(CodeUnavailable)
	}
	owned := true
	defer func() {
		if owned {
			runner.unregister(active)
		}
	}()
	defer cancel()
	if err := runner.namespace.revalidate(); err != nil {
		return RunResult{}, err
	}
	if err := runner.cgroups.prepare(id); err != nil {
		return RunResult{}, err
	}
	workspace, err := createRunWorkspace(
		runContext,
		runner.runs,
		runner.config,
		runner.assets,
		request,
		inputDrive,
	)
	if err != nil {
		return RunResult{}, err
	}
	processStarted := false
	defer func() {
		if !owned {
			return
		}
		cleanupContext, cleanupCancel := context.WithTimeout(
			context.Background(),
			runner.config.CleanupTimeout,
		)
		defer cleanupCancel()
		cleanupErr := errors.Join(
			runner.cgroups.cleanup(
				cleanupContext,
				id,
				request,
				processStarted,
			),
			workspace.CleanupContext(cleanupContext),
		)
		if cleanupErr != nil {
			runner.mu.Lock()
			runner.residualErr = launcherError(CodeCleanupFailed)
			runner.mu.Unlock()
			result = RunResult{}
			resultErr = launcherError(CodeCleanupFailed)
		}
	}()
	diagnostic := &boundedDiagnostic{limit: int(runner.config.MaxDiagnosticBytes)}
	command := runner.command(
		workspace.jailerPath,
		buildJailerArguments(runner.config, request, workspace)...,
	)
	if command == nil {
		return RunResult{}, launcherError(CodeExecutionFailed)
	}
	command.Env = []string{}
	command.Stdout, command.Stderr = diagnostic, diagnostic
	command.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true, Pdeathsig: syscall.SIGKILL,
	}
	nullInput, err := os.OpenFile("/dev/null", os.O_RDONLY|syscall.O_CLOEXEC, 0)
	if err != nil {
		return RunResult{}, launcherError(CodeExecutionFailed)
	}
	command.Stdin = nullInput
	if err := command.Start(); err != nil {
		_ = nullInput.Close()
		return RunResult{}, launcherError(CodeExecutionFailed)
	}
	processStarted = true
	active.setCommand(command)
	_ = nullInput.Close()
	waitResult := waitPhysicalProcess(
		runContext,
		command,
		runner.cgroups,
		id,
		runner.config.CleanupTimeout,
	)
	if !waitResult.reaped {
		owned = false
		go runner.finishResidual(
			active,
			waitResult.done,
			workspace,
			request,
		)
		return RunResult{}, launcherError(CodeCleanupFailed)
	}
	active.clearCommand()
	if runContext.Err() != nil {
		return RunResult{}, launcherError(contextFailureCode(runContext))
	}
	if waitResult.err != nil {
		return RunResult{}, launcherError(CodeExecutionFailed)
	}
	if err := workspace.copyOutput(outputDrive, request.OutputDriveBytes); err != nil {
		return RunResult{}, err
	}
	return RunResult{
		CapturedOutputBytes: diagnostic.bytes(),
		AssetDigest:         runner.assets.digest,
	}, nil
}
