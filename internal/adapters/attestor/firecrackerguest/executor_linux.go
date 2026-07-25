//go:build linux

package firecrackerguest

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

const (
	nobodyID            = 65534
	maxGoJSONEventBytes = 1 << 20
	postKillWaitLimit   = 2 * time.Second
)

type Execution struct {
	GoBinary                 string
	Workspace                string
	WorkingDirectory         string
	Arguments                []string
	Environment              []string
	OutputDirectory          string
	MaxOutputBytes           uint64
	UID, GID                 uint32
	ClearSupplementaryGroups bool
}

type ExecutionResult struct {
	ExitCode     uint8
	OutputDigest string
	TestCases    uint64
	OutputLimit  bool
	TimedOut     bool
}

type Executor interface {
	Execute(context.Context, Execution) (ExecutionResult, error)
}

type OSExecutor struct {
	processAttributes func(Execution) *syscall.SysProcAttr
	commandFactory    func(Execution) *exec.Cmd
	dropBoundingSet   func() error
}

func (executor OSExecutor) Execute(ctx context.Context, execution Execution) (result ExecutionResult, resultErr error) {
	if execution.MaxOutputBytes == 0 || execution.MaxOutputBytes > intMax64() {
		return result, guestError(CodeExecutionFailed, nil)
	}
	stdout, err := os.CreateTemp(execution.OutputDirectory, "stdout-")
	if err != nil {
		return result, guestError(CodeExecutionFailed, err)
	}
	var stderr *os.File
	cleanupOutputs := func() error {
		var cleanupErr error
		for _, output := range []*os.File{stdout, stderr} {
			if output == nil {
				continue
			}
			cleanupErr = errors.Join(cleanupErr, output.Close())
			if err := os.Remove(output.Name()); err != nil && !os.IsNotExist(err) {
				cleanupErr = errors.Join(cleanupErr, err)
			}
		}
		return cleanupErr
	}
	cleanupSynchronous := true
	defer func() {
		if cleanupSynchronous {
			cleanupErr := cleanupOutputs()
			if cleanupErr == nil {
				return
			}
			result = ExecutionResult{}
			resultErr = errors.Join(guestError(CodeCleanupFailed, cleanupErr), resultErr)
		}
	}()
	stderr, err = os.CreateTemp(execution.OutputDirectory, "stderr-")
	if err != nil {
		return result, guestError(CodeExecutionFailed, err)
	}
	outputs := newCappedOutputs(int64(execution.MaxOutputBytes), stdout, stderr)
	command := pid1SupervisorCommand(execution)
	if executor.commandFactory != nil {
		command = executor.commandFactory(execution)
	}
	command.Dir, command.Env = execution.WorkingDirectory, append([]string(nil), execution.Environment...)
	command.Stdout, command.Stderr = outputs.stdoutWriter(), outputs.stderrWriter()
	command.WaitDelay = postKillWaitLimit
	command.SysProcAttr = isolatedProcessAttributes(execution)
	if executor.processAttributes != nil {
		command.SysProcAttr = executor.processAttributes(execution)
	}
	type processStartResult struct {
		dropBoundingSetErr error
		noNewPrivilegesErr error
		startErr           error
	}
	started := make(chan processStartResult, 1)
	done := make(chan error, 1)
	go func() {
		runtime.LockOSThread()
		dropBoundingSet := dropCapabilityBoundingSet
		if executor.dropBoundingSet != nil {
			dropBoundingSet = executor.dropBoundingSet
		}
		startResult := processStartResult{dropBoundingSetErr: dropBoundingSet()}
		if startResult.dropBoundingSetErr == nil {
			startResult.noNewPrivilegesErr = unix.Prctl(unix.PR_SET_NO_NEW_PRIVS, 1, 0, 0, 0)
		}
		if startResult.dropBoundingSetErr == nil && startResult.noNewPrivilegesErr == nil {
			startResult.startErr = command.Start()
		}
		started <- startResult
		if startResult.dropBoundingSetErr == nil &&
			startResult.noNewPrivilegesErr == nil &&
			startResult.startErr == nil {
			done <- command.Wait()
		}
		// Deliberately return while locked. The runtime must discard this
		// permanently no_new_privs OS thread instead of returning it to the pool.
	}()
	startResult := <-started
	if startResult.dropBoundingSetErr != nil {
		return result, guestError(CodeExecutionFailed, startResult.dropBoundingSetErr)
	}
	if startResult.noNewPrivilegesErr != nil {
		return result, guestError(CodeExecutionFailed, startResult.noNewPrivilegesErr)
	}
	if startResult.startErr != nil {
		return result, guestError(CodeExecutionFailed, startResult.startErr)
	}
	var waitErr error
	select {
	case waitErr = <-done:
		if outputs.exceeded() {
			result.OutputLimit = true
		}
	case <-outputs.overflow:
		result.OutputLimit = true
		_ = killProcessGroup(command.Process.Pid)
		var reaped bool
		waitErr, reaped = waitForKilledProcess(done, postKillWaitLimit)
		if !reaped {
			cleanupSynchronous = false
			go func() {
				<-done
				_ = cleanupOutputs()
			}()
			return result, nil
		}
	case <-ctx.Done():
		result.TimedOut = true
		_ = killProcessGroup(command.Process.Pid)
		var reaped bool
		waitErr, reaped = waitForKilledProcess(done, postKillWaitLimit)
		if !reaped {
			cleanupSynchronous = false
			go func() {
				<-done
				_ = cleanupOutputs()
			}()
			return result, nil
		}
	}
	if err := errors.Join(stdout.Sync(), stderr.Sync()); err != nil {
		return ExecutionResult{}, guestError(CodeExecutionFailed, err)
	}
	digest, err := capturedOutputDigest(stdout, stderr)
	if err != nil {
		return ExecutionResult{}, guestError(CodeExecutionFailed, err)
	}
	result.OutputDigest = digest
	if result.OutputLimit || result.TimedOut {
		return result, nil
	}
	result.ExitCode = processExitCode(waitErr)
	testCases, err := countGoTestCases(stdout)
	if err != nil {
		return ExecutionResult{}, guestError(CodeExecutionFailed, err)
	}
	result.TestCases = testCases
	return result, nil
}

func pid1SupervisorCommand(execution Execution) *exec.Cmd {
	arguments := make([]string, 0, len(execution.Environment)+len(execution.Arguments)+3)
	arguments = append(
		arguments,
		privatePID1SupervisorArgument,
		strconv.Itoa(len(execution.Environment)),
	)
	arguments = append(arguments, execution.Environment...)
	arguments = append(arguments, execution.GoBinary)
	arguments = append(arguments, execution.Arguments...)
	return exec.Command("/proc/self/exe", arguments...)
}

const maximumCapabilityBoundingSetBit = 63

func dropCapabilityBoundingSet() error {
	for capability := 0; capability <= maximumCapabilityBoundingSetBit; capability++ {
		err := unix.Prctl(unix.PR_CAPBSET_DROP, uintptr(capability), 0, 0, 0)
		if err != nil && !errors.Is(err, unix.EINVAL) {
			return err
		}
		value, readErr := unix.PrctlRetInt(unix.PR_CAPBSET_READ, uintptr(capability), 0, 0, 0)
		if errors.Is(readErr, unix.EINVAL) {
			continue
		}
		if readErr != nil {
			return readErr
		}
		if value != 0 {
			return unix.EPERM
		}
	}
	_, err := unix.PrctlRetInt(
		unix.PR_CAPBSET_READ, uintptr(maximumCapabilityBoundingSetBit+1), 0, 0, 0,
	)
	if !errors.Is(err, unix.EINVAL) {
		if err == nil {
			return unix.EOVERFLOW
		}
		return err
	}
	return nil
}

func waitForKilledProcess(done <-chan error, limit time.Duration) (error, bool) {
	timer := time.NewTimer(limit)
	defer timer.Stop()
	select {
	case err := <-done:
		return err, true
	case <-timer.C:
		return nil, false
	}
}

func isolatedProcessAttributes(execution Execution) *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		Setpgid:    true,
		Pdeathsig:  syscall.SIGKILL,
		Cloneflags: unix.CLONE_NEWPID,
		Credential: &syscall.Credential{
			Uid: execution.UID, Gid: execution.GID,
			Groups:      supplementaryGroups(execution),
			NoSetGroups: !execution.ClearSupplementaryGroups,
		},
	}
}

func supplementaryGroups(execution Execution) []uint32 {
	if execution.ClearSupplementaryGroups {
		return []uint32{}
	}
	return nil
}

func killProcessGroup(pid int) error {
	if pid <= 0 {
		return unix.EINVAL
	}
	return unix.Kill(-pid, unix.SIGKILL)
}

func processExitCode(err error) uint8 {
	if err == nil {
		return 0
	}
	var exitError *exec.ExitError
	if errors.As(err, &exitError) && exitError.ExitCode() >= 0 && exitError.ExitCode() <= 255 {
		return normalizedExitCode(exitError.ExitCode())
	}
	return 255
}

func normalizedExitCode(code int) uint8 {
	if code < 0 || code > 255 || code == int(NoTestsExitCode) {
		return 255
	}
	return uint8(code)
}

type cappedOutputs struct {
	mu        sync.Mutex
	remaining int64
	stdout    *os.File
	stderr    *os.File
	overflow  chan struct{}
	once      sync.Once
}

type cappedOutputWriter struct {
	outputs *cappedOutputs
	file    *os.File
}

func newCappedOutputs(maximum int64, stdout, stderr *os.File) *cappedOutputs {
	return &cappedOutputs{
		remaining: maximum, stdout: stdout, stderr: stderr, overflow: make(chan struct{}),
	}
}

func (outputs *cappedOutputs) stdoutWriter() io.Writer {
	return cappedOutputWriter{outputs: outputs, file: outputs.stdout}
}

func (outputs *cappedOutputs) stderrWriter() io.Writer {
	return cappedOutputWriter{outputs: outputs, file: outputs.stderr}
}

func (writer cappedOutputWriter) Write(value []byte) (int, error) {
	writer.outputs.mu.Lock()
	defer writer.outputs.mu.Unlock()
	keep := min(int64(len(value)), writer.outputs.remaining)
	if keep > 0 {
		written, err := writer.file.Write(value[:keep])
		if err != nil {
			return 0, err
		}
		if int64(written) != keep {
			return 0, io.ErrShortWrite
		}
		writer.outputs.remaining -= keep
	}
	if int64(len(value)) > keep {
		writer.outputs.once.Do(func() { close(writer.outputs.overflow) })
	}
	return len(value), nil
}

func (outputs *cappedOutputs) exceeded() bool {
	select {
	case <-outputs.overflow:
		return true
	default:
		return false
	}
}

func capturedOutputDigest(stdout, stderr *os.File) (string, error) {
	digest := sha256.New()
	if err := writeDigestField(digest, []byte("orquesta.test-output.v1")); err != nil {
		return "", err
	}
	for _, file := range []*os.File{stdout, stderr} {
		stat, err := file.Stat()
		if err != nil || stat.Size() < 0 {
			return "", err
		}
		var size [8]byte
		binary.BigEndian.PutUint64(size[:], uint64(stat.Size()))
		if _, err := digest.Write(size[:]); err != nil {
			return "", err
		}
		if _, err := file.Seek(0, io.SeekStart); err != nil {
			return "", err
		}
		if _, err := io.CopyN(digest, file, stat.Size()); err != nil {
			return "", err
		}
	}
	return hex.EncodeToString(digest.Sum(nil)), nil
}

func writeDigestField(destination io.Writer, value []byte) error {
	var size [8]byte
	binary.BigEndian.PutUint64(size[:], uint64(len(value)))
	if _, err := destination.Write(size[:]); err != nil {
		return err
	}
	_, err := destination.Write(value)
	return err
}

func countGoTestCases(stdout *os.File) (uint64, error) {
	if _, err := stdout.Seek(0, io.SeekStart); err != nil {
		return 0, err
	}
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 32*1024), maxGoJSONEventBytes)
	var count uint64
	for scanner.Scan() {
		var event struct {
			Action string `json:"Action"`
			Test   string `json:"Test"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			return 0, err
		}
		if event.Action == "run" && event.Test != "" {
			count++
		}
	}
	return count, scanner.Err()
}

func minimalEnvironment(scratch string) []string {
	return []string{
		"CGO_ENABLED=0",
		"GOCACHE=" + filepath.Join(scratch, "gocache"),
		"GOENV=off",
		"GONOSUMDB=*",
		"GOPATH=" + filepath.Join(scratch, "gopath"),
		"GOPROXY=off",
		"GOROOT=/toolchain",
		"GOSUMDB=off",
		"GOTOOLCHAIN=local",
		"GOTMPDIR=" + filepath.Join(scratch, "gotmp"),
		"GOVCS=off",
		"HOME=" + filepath.Join(scratch, "home"),
		"PATH=/toolchain/bin",
		"TMPDIR=" + filepath.Join(scratch, "tmp"),
	}
}

func intMax64() uint64 { return uint64(^uint64(0) >> 1) }
