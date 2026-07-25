//go:build linux

package firecrackerlauncher

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

func buildJailerArguments(
	config Config,
	request LaunchRequest,
	workspace *runWorkspace,
) []string {
	return []string{
		"--id", workspace.id,
		"--exec-file", workspace.firecrackerPath,
		"--uid", strconv.FormatUint(uint64(config.JailUID), 10),
		"--gid", strconv.FormatUint(uint64(config.JailGID), 10),
		"--chroot-base-dir", filepathJoin(workspace.path, "jail"),
		"--netns", config.NetNSPath,
		// Do not pass --new-pid-ns or --daemonize: jailer must exec Firecracker
		// in the process observed by command.Wait.
		"--cgroup-version", "2",
		"--parent-cgroup", config.ParentCgroup,
		"--cgroup", fmt.Sprintf("cpu.max=%d %d", request.CPUQuotaMicros, request.CPUPeriodMicros),
		"--cgroup", "memory.max=" + strconv.FormatUint(request.MemoryMaxBytes, 10),
		"--cgroup", "memory.swap.max=0",
		"--cgroup", "memory.oom.group=1",
		"--cgroup", "pids.max=" + strconv.FormatUint(uint64(request.PIDsMax), 10),
		"--resource-limit", "fsize=" + strconv.FormatUint(request.OutputDriveBytes, 10),
		"--resource-limit", "no-file=" + strconv.Itoa(firecrackerNoFileLimit),
		"--",
		"--no-api",
		"--config-file", "/" + firecrackerConfig,
	}
}

func filepathJoin(parent, child string) string {
	return parent + string(os.PathSeparator) + child
}

type physicalProcessWait struct {
	err    error
	reaped bool
	done   <-chan error
}

func waitPhysicalProcess(
	ctx context.Context,
	command *exec.Cmd,
	cgroups runnerCgroup,
	id string,
	budget time.Duration,
) physicalProcessWait {
	done := make(chan error, 1)
	go func() { done <- command.Wait() }()
	return waitPhysicalProcessDone(ctx, command, cgroups, id, budget, done)
}

func waitPhysicalProcessDone(
	ctx context.Context,
	command *exec.Cmd,
	cgroups runnerCgroup,
	id string,
	budget time.Duration,
	done <-chan error,
) physicalProcessWait {
	select {
	case err := <-done:
		return physicalProcessWait{err: err, reaped: true, done: done}
	case <-ctx.Done():
	}
	deadline := time.Now().Add(budget)
	termErr := signalProcessGroup(command, syscall.SIGTERM)
	termGrace := min(budget/3, 250*time.Millisecond)
	if termGrace <= 0 {
		termGrace = time.Nanosecond
	}
	timer := time.NewTimer(termGrace)
	defer timer.Stop()
	select {
	case err := <-done:
		return physicalProcessWait{
			err: errors.Join(termErr, err), reaped: true, done: done,
		}
	case <-timer.C:
	}
	killErr := errors.Join(
		signalProcessGroup(command, syscall.SIGKILL),
		cgroups.kill(id),
	)
	remaining := time.Until(deadline)
	if remaining <= 0 {
		return physicalProcessWait{
			err: launcherError(CodeCleanupFailed), done: done,
		}
	}
	killWait := time.NewTimer(remaining)
	defer killWait.Stop()
	select {
	case waitErr := <-done:
		return physicalProcessWait{
			err: errors.Join(killErr, waitErr), reaped: true, done: done,
		}
	case <-killWait.C:
		return physicalProcessWait{
			err:  errors.Join(launcherError(CodeCleanupFailed), killErr),
			done: done,
		}
	}
}

func signalProcessGroup(command *exec.Cmd, signal syscall.Signal) error {
	if command == nil || command.Process == nil || command.Process.Pid <= 0 {
		return nil
	}
	err := unix.Kill(-command.Process.Pid, signal)
	if errors.Is(err, unix.ESRCH) {
		return nil
	}
	return err
}

func (run *activePhysicalRun) setCommand(command *exec.Cmd) {
	run.mu.Lock()
	defer run.mu.Unlock()
	run.command = command
}

func (run *activePhysicalRun) clearCommand() {
	run.mu.Lock()
	defer run.mu.Unlock()
	run.command = nil
}

func (run *activePhysicalRun) signal(signal syscall.Signal) error {
	run.cancel()
	run.mu.Lock()
	defer run.mu.Unlock()
	return signalProcessGroup(run.command, signal)
}

func (run *activePhysicalRun) kill() error {
	return errors.Join(
		run.signal(syscall.SIGKILL),
		run.cgroups.kill(run.id),
	)
}

func (runner *physicalRunner) register(run *activePhysicalRun) bool {
	runner.mu.Lock()
	defer runner.mu.Unlock()
	if runner.closed {
		return false
	}
	runner.active[run] = struct{}{}
	runner.wait.Add(1)
	return true
}

func (runner *physicalRunner) unregister(run *activePhysicalRun) {
	runner.mu.Lock()
	delete(runner.active, run)
	closed := runner.closed
	empty := len(runner.active) == 0
	clean := runner.residualErr == nil
	runner.mu.Unlock()
	runner.wait.Done()
	if closed && empty && clean {
		runner.closeResources()
	}
}

func (runner *physicalRunner) finishResidual(
	active *activePhysicalRun,
	done <-chan error,
	workspace *runWorkspace,
	request LaunchRequest,
) {
	<-done
	active.clearCommand()
	cleanupContext, cancel := context.WithTimeout(
		context.Background(),
		runner.config.CleanupTimeout,
	)
	cleanupErr := errors.Join(
		runner.cgroups.cleanup(
			cleanupContext,
			active.id,
			request,
			true,
		),
		workspace.CleanupContext(cleanupContext),
	)
	cancel()
	if cleanupErr != nil {
		runner.mu.Lock()
		runner.residualErr = launcherError(CodeCleanupFailed)
		runner.mu.Unlock()
	}
	runner.unregister(active)
}

func (runner *physicalRunner) Close() error {
	if runner == nil {
		return nil
	}
	runner.closeMu.Lock()
	defer runner.closeMu.Unlock()
	runner.mu.Lock()
	runner.closed = true
	active := make([]*activePhysicalRun, 0, len(runner.active))
	for run := range runner.active {
		active = append(active, run)
	}
	runner.mu.Unlock()
	if len(active) == 0 {
		return runner.finishClose()
	}
	deadline := time.Now().Add(runner.config.CleanupTimeout)
	for _, run := range active {
		_ = run.signal(syscall.SIGTERM)
	}
	done := make(chan struct{})
	go func() {
		runner.wait.Wait()
		close(done)
	}()
	termGrace := min(runner.config.CleanupTimeout/3, 250*time.Millisecond)
	if termGrace <= 0 {
		termGrace = time.Nanosecond
	}
	timer := time.NewTimer(termGrace)
	select {
	case <-done:
		if !timer.Stop() {
			<-timer.C
		}
		return runner.finishClose()
	case <-timer.C:
	}
	for _, run := range active {
		_ = run.kill()
	}
	remaining := time.Until(deadline)
	if remaining <= 0 {
		return launcherError(CodeCleanupFailed)
	}
	killWait := time.NewTimer(remaining)
	defer killWait.Stop()
	select {
	case <-done:
		return runner.finishClose()
	case <-killWait.C:
		return launcherError(CodeCleanupFailed)
	}
}

func (runner *physicalRunner) finishClose() error {
	runner.mu.Lock()
	residualErr := runner.residualErr
	runner.mu.Unlock()
	if residualErr != nil {
		return launcherError(CodeCleanupFailed)
	}
	runner.closeResources()
	return runner.resourceCloseErr
}

func (runner *physicalRunner) closeResources() {
	runner.resourceClose.Do(func() {
		resourceErr := errors.Join(
			runner.runs.Close(),
			runner.runtimeRoot.Close(),
			runner.cgroups.Close(),
			runner.namespace.Close(),
			runner.assets.Close(),
		)
		if resourceErr != nil {
			runner.resourceCloseErr = launcherError(CodeCleanupFailed)
		}
	})
}

func (diagnostic *boundedDiagnostic) Write(payload []byte) (int, error) {
	diagnostic.mu.Lock()
	defer diagnostic.mu.Unlock()
	remaining := diagnostic.limit - len(diagnostic.buffer)
	if remaining > 0 {
		keep := min(remaining, len(payload))
		diagnostic.buffer = append(diagnostic.buffer, payload[:keep]...)
	}
	return len(payload), nil
}

func (diagnostic *boundedDiagnostic) bytes() uint64 {
	diagnostic.mu.Lock()
	defer diagnostic.mu.Unlock()
	return uint64(len(diagnostic.buffer))
}

var _ io.Writer = (*boundedDiagnostic)(nil)
