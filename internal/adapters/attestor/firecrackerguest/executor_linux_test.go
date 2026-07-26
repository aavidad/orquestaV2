//go:build linux

package firecrackerguest

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func TestMain(testingMain *testing.M) {
	if exitCode, handled := DispatchPID1Supervisor(os.Args); handled {
		os.Exit(exitCode)
	}
	os.Exit(testingMain.Run())
}

func TestGoJSONCaseDetectionPassFailAndNoTests(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    uint64
	}{
		{
			"passing test",
			"{\"Action\":\"run\",\"Test\":\"TestA\"}\n{\"Action\":\"pass\",\"Test\":\"TestA\"}\n",
			1,
		},
		{
			"failing test still executed",
			"{\"Action\":\"run\",\"Test\":\"TestA\"}\n{\"Action\":\"fail\",\"Test\":\"TestA\"}\n",
			1,
		},
		{
			"mixed packages include one executed test",
			"{\"Action\":\"start\",\"Package\":\"example/empty\"}\n" +
				"{\"Action\":\"output\",\"Package\":\"example/empty\",\"Output\":\"? example/empty [no test files]\\n\"}\n" +
				"{\"Action\":\"skip\",\"Package\":\"example/empty\"}\n" +
				"{\"Action\":\"run\",\"Package\":\"example/tested\",\"Test\":\"TestA\"}\n" +
				"{\"Action\":\"pass\",\"Package\":\"example/tested\",\"Test\":\"TestA\"}\n",
			1,
		},
		{
			"output phrase is not execution evidence",
			"{\"Action\":\"output\",\"Package\":\"example\",\"Test\":\"TestPrinter\"," +
				"\"Output\":\"user text: [no tests to run]\\n\"}\n" +
				"{\"Action\":\"pass\",\"Package\":\"example\"}\n",
			0,
		},
		{
			"skipped test executed",
			"{\"Action\":\"run\",\"Package\":\"example\",\"Test\":\"TestOptional\"}\n" +
				"{\"Action\":\"skip\",\"Package\":\"example\",\"Test\":\"TestOptional\"}\n",
			1,
		},
		{
			"package without test files",
			"{\"Action\":\"start\",\"Package\":\"example\"}\n" +
				"{\"Action\":\"output\",\"Package\":\"example\",\"Output\":\"? example [no test files]\\n\"}\n" +
				"{\"Action\":\"skip\",\"Package\":\"example\"}\n",
			0,
		},
		{
			"selector matches no tests",
			"{\"Action\":\"start\",\"Package\":\"example\"}\n" +
				"{\"Action\":\"output\",\"Package\":\"example\",\"Output\":\"testing: warning: no tests to run\\n\"}\n" +
				"{\"Action\":\"pass\",\"Package\":\"example\"}\n",
			0,
		},
		{
			"TestMain without m Run",
			"{\"Action\":\"output\",\"Package\":\"example\",\"Test\":\"TestMain\",\"Output\":\"setup\\n\"}\n" +
				"{\"Action\":\"pass\",\"Package\":\"example\"}\n",
			0,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			file := writeGuestTempFile(t, test.content)
			count, err := countGoTestCases(file)
			if err != nil || count != test.want {
				t.Fatalf("count=%d want=%d err=%v", count, test.want, err)
			}
		})
	}
	file := writeGuestTempFile(t, "not-json\n")
	if _, err := countGoTestCases(file); err == nil {
		t.Fatal("malformed go JSON accepted")
	}
}

func TestCapturedOutputDigestMatchesTestAttestorContract(t *testing.T) {
	stdout := writeGuestTempFile(t, "stdout\n")
	stderr := writeGuestTempFile(t, "stderr\n")
	got, err := capturedOutputDigest(stdout, stderr)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.New()
	for _, value := range []string{"orquesta.test-output.v1", "stdout\n", "stderr\n"} {
		var length [8]byte
		binary.BigEndian.PutUint64(length[:], uint64(len(value)))
		_, _ = digest.Write(length[:])
		_, _ = digest.Write([]byte(value))
	}
	want := hex.EncodeToString(digest.Sum(nil))
	if got != want {
		t.Fatalf("digest=%q want=%q", got, want)
	}
}

func TestCombinedOutputLimitCountsStdoutAndStderr(t *testing.T) {
	stdout := writeGuestTempFile(t, "")
	stderr := writeGuestTempFile(t, "")
	outputs := newCappedOutputs(5, stdout, stderr)
	if _, err := outputs.stdoutWriter().Write([]byte("abc")); err != nil {
		t.Fatal(err)
	}
	if _, err := outputs.stderrWriter().Write([]byte("def")); err != nil {
		t.Fatal(err)
	}
	if !outputs.exceeded() {
		t.Fatal("combined output limit not observed")
	}
	if value, _ := os.ReadFile(stdout.Name()); string(value) != "abc" {
		t.Fatalf("stdout=%q", value)
	}
	if value, _ := os.ReadFile(stderr.Name()); string(value) != "de" {
		t.Fatalf("stderr=%q", value)
	}
}

func TestDriveSizeRegularFile(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "drive-")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if err := file.Truncate(4096); err != nil {
		t.Fatal(err)
	}
	if size, err := driveSize(file); err != nil || size != 4096 {
		t.Fatalf("size=%d err=%v", size, err)
	}
}

func TestExecutionUsesPIDNamespaceAndNobodyWithoutGroups(t *testing.T) {
	attributes := isolatedProcessAttributes(Execution{
		UID: nobodyID, GID: nobodyID, ClearSupplementaryGroups: true,
	})
	if attributes.Cloneflags&unix.CLONE_NEWPID == 0 ||
		attributes.Credential == nil ||
		attributes.Credential.Uid != nobodyID ||
		attributes.Credential.Gid != nobodyID ||
		attributes.Credential.NoSetGroups ||
		attributes.Credential.Groups == nil ||
		len(attributes.Credential.Groups) != 0 {
		t.Fatalf("attributes=%+v credential=%+v", attributes, attributes.Credential)
	}
	if normalizedExitCode(int(NoTestsExitCode)) == NoTestsExitCode {
		t.Fatal("reserved no-tests exit accepted from real process")
	}
}

func TestPostKillWaitIsBoundedAndStillConsumesReap(t *testing.T) {
	done := make(chan error, 1)
	start := time.Now()
	if _, reaped := waitForKilledProcess(done, 20*time.Millisecond); reaped {
		t.Fatal("unreported process was treated as reaped")
	}
	if elapsed := time.Since(start); elapsed < 15*time.Millisecond || elapsed > time.Second {
		t.Fatalf("post-kill bound elapsed=%s", elapsed)
	}
	reaped := make(chan struct{})
	go func() {
		<-done
		close(reaped)
	}()
	done <- errors.New("killed")
	select {
	case <-reaped:
	case <-time.After(time.Second):
		t.Fatal("asynchronous reaper did not consume terminal wait")
	}
}

func TestOSExecutorDedicatedNoNewPrivilegesThreadDoesNotContaminateCaller(t *testing.T) {
	kind, mode, _, helper := guestReexecState()
	if helper && kind != "nnp-caller" {
		return
	}
	if mode == "probe" {
		_, _ = os.Stdout.WriteString(
			"{\"Action\":\"run\",\"Test\":\"TestNoThreadContamination\"}\n" +
				"{\"Action\":\"pass\",\"Test\":\"TestNoThreadContamination\"}\n",
		)
		os.Exit(0)
	}

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	before, err := unix.PrctlRetInt(unix.PR_GET_NO_NEW_PRIVS, 0, 0, 0, 0)
	if err != nil || before != 0 {
		t.Fatalf("caller precondition no_new_privs=%d err=%v", before, err)
	}
	scratch := t.TempDir()
	executor := OSExecutor{
		processAttributes: func(Execution) *syscall.SysProcAttr {
			return &syscall.SysProcAttr{Setpgid: true}
		},
		commandFactory: func(execution Execution) *exec.Cmd {
			return exec.Command(execution.GoBinary, execution.Arguments...)
		},
		dropBoundingSet: func() error { return nil },
	}
	result, err := executor.Execute(context.Background(), Execution{
		GoBinary: os.Args[0], Workspace: scratch, WorkingDirectory: scratch,
		Arguments: reexecArguments(
			"TestOSExecutorDedicatedNoNewPrivilegesThreadDoesNotContaminateCaller",
			"nnp-caller", "probe", "",
		),
		Environment: nil, OutputDirectory: scratch, MaxOutputBytes: 1 << 20,
	})
	if err != nil || result.ExitCode != 0 || result.TestCases != 1 {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	after, err := unix.PrctlRetInt(unix.PR_GET_NO_NEW_PRIVS, 0, 0, 0, 0)
	if err != nil || after != before {
		t.Fatalf("caller contaminated before=%d after=%d err=%v", before, after, err)
	}
}

func TestOSExecutorRealNobodyHasNoSupplementaryGroups(t *testing.T) {
	kind, mode, statusPath, helper := guestReexecState()
	if helper && kind != "identity" {
		return
	}
	switch mode {
	case "probe":
		status, err := os.ReadFile("/proc/self/status")
		if err != nil || os.WriteFile(statusPath, status, 0o600) != nil {
			os.Exit(96)
		}
		_, _ = os.Stdout.WriteString(
			"{\"Action\":\"run\",\"Test\":\"TestIdentity\"}\n" +
				"{\"Action\":\"pass\",\"Test\":\"TestIdentity\"}\n",
		)
		os.Exit(0)
	case "driver", "driver-root":
		scratch := filepath.Dir(statusPath)
		executor := OSExecutor{}
		if mode == "driver" {
			executor.processAttributes = func(execution Execution) *syscall.SysProcAttr {
				attributes := isolatedProcessAttributes(execution)
				attributes.Cloneflags |= unix.CLONE_NEWUSER
				attributes.UidMappings = []syscall.SysProcIDMap{
					{ContainerID: nobodyID, HostID: 0, Size: 1},
				}
				attributes.GidMappings = []syscall.SysProcIDMap{
					{ContainerID: nobodyID, HostID: 0, Size: 1},
				}
				attributes.GidMappingsEnableSetgroups = true
				return attributes
			}
		}
		result, err := executor.Execute(context.Background(), Execution{
			GoBinary: os.Args[0], Workspace: scratch, WorkingDirectory: scratch,
			Arguments: reexecArguments(
				"TestOSExecutorRealNobodyHasNoSupplementaryGroups", "identity", "probe", statusPath,
			),
			Environment:     nil,
			OutputDirectory: scratch, MaxOutputBytes: 1 << 20,
			UID: nobodyID, GID: nobodyID, ClearSupplementaryGroups: true,
		})
		if err != nil || result.ExitCode != 0 || result.TestCases != 1 {
			var guestErr *Error
			_ = errors.As(err, &guestErr)
			var cause error
			if guestErr != nil {
				cause = guestErr.Cause
			}
			t.Fatalf("result=%+v err=%v cause=%v", result, err, cause)
		}
		return
	}

	scratch, err := os.MkdirTemp("", "orquesta-guest-identity-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := cleanupPrivateTree(
			context.Background(), scratch, maximumCleanupRecords(),
		); err != nil {
			t.Error(err)
		}
	})
	statusPath = filepath.Join(scratch, "status")
	if err := os.WriteFile(statusPath, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(scratch, 0o711); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(statusPath, 0o666); err != nil {
		t.Fatal(err)
	}
	mode = "driver-root"
	command := exec.Command(
		os.Args[0],
		reexecArguments(
			"TestOSExecutorRealNobodyHasNoSupplementaryGroups", "identity", mode, statusPath,
		)...,
	)
	if os.Geteuid() != 0 {
		mode = "driver"
		command.Args = append(
			[]string{os.Args[0]},
			reexecArguments(
				"TestOSExecutorRealNobodyHasNoSupplementaryGroups", "identity", mode, statusPath,
			)...,
		)
		command.SysProcAttr = &syscall.SysProcAttr{
			Cloneflags: unix.CLONE_NEWUSER,
			UidMappings: []syscall.SysProcIDMap{
				{ContainerID: 0, HostID: os.Getuid(), Size: 1},
			},
			GidMappings: []syscall.SysProcIDMap{
				{ContainerID: 0, HostID: os.Getgid(), Size: 1},
			},
			GidMappingsEnableSetgroups: false,
		}
	}
	output, err := command.CombinedOutput()
	if err != nil {
		if errors.Is(err, syscall.EPERM) || strings.Contains(string(output), "operation not permitted") {
			t.Skipf("kernel denies test-owned user namespace: %v: %s", err, output)
		}
		t.Fatalf("driver: %v\n%s", err, output)
	}
	status, err := os.ReadFile(statusPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"Uid:\t65534\t65534\t65534\t65534",
		"Gid:\t65534\t65534\t65534\t65534",
		"CapInh:\t0000000000000000",
		"CapPrm:\t0000000000000000",
		"CapEff:\t0000000000000000",
		"CapBnd:\t0000000000000000",
		"CapAmb:\t0000000000000000",
		"NoNewPrivs:\t1",
	} {
		if !statusHasExactLine(status, want) {
			t.Fatalf("missing %q in /proc/self/status:\n%s", want, status)
		}
	}
	if !statusHasEmptyField(status, "Groups:") {
		t.Fatalf("supplementary groups not empty in /proc/self/status:\n%s", status)
	}
}

func statusHasExactLine(status []byte, want string) bool {
	for _, line := range strings.Split(strings.TrimSuffix(string(status), "\n"), "\n") {
		if line == want {
			return true
		}
	}
	return false
}

func statusHasEmptyField(status []byte, field string) bool {
	for _, line := range strings.Split(string(status), "\n") {
		if strings.HasPrefix(line, field) {
			return strings.TrimSpace(strings.TrimPrefix(line, field)) == ""
		}
	}
	return false
}

func TestPIDNamespaceKillsEscapedSetsidChild(t *testing.T) {
	kind, mode, marker, helper := guestReexecState()
	if helper && kind != "pid-namespace" {
		return
	}
	switch mode {
	case "init":
		child := exec.Command(
			os.Args[0],
			reexecArguments(
				"TestPIDNamespaceKillsEscapedSetsidChild", "pid-namespace", "child", marker,
			)...,
		)
		child.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
		if err := child.Start(); err != nil {
			os.Exit(91)
		}
		ready := marker + ".ready"
		deadline := time.Now().Add(3 * time.Second)
		for time.Now().Before(deadline) {
			for {
				var status unix.WaitStatus
				pid, err := unix.Wait4(-1, &status, unix.WNOHANG, nil)
				if pid <= 0 || err != nil {
					break
				}
			}
			_, _ = os.Stat(ready)
			time.Sleep(5 * time.Millisecond)
		}
		os.Exit(93)
	case "child":
		grandchild := exec.Command(
			os.Args[0],
			reexecArguments(
				"TestPIDNamespaceKillsEscapedSetsidChild", "pid-namespace", "escaped", marker,
			)...,
		)
		grandchild.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
		if err := grandchild.Start(); err != nil {
			os.Exit(94)
		}
		if err := grandchild.Process.Release(); err != nil {
			os.Exit(97)
		}
		os.Exit(0)
	case "escaped":
		if os.WriteFile(marker+".ready", []byte("ready"), 0o600) != nil {
			os.Exit(95)
		}
		time.Sleep(300 * time.Millisecond)
		if os.WriteFile(marker, []byte("escaped"), 0o600) != nil {
			os.Exit(92)
		}
		os.Exit(0)
	}
	marker = filepath.Join(t.TempDir(), "escaped-marker")
	command := exec.Command(
		os.Args[0],
		reexecArguments(
			"TestPIDNamespaceKillsEscapedSetsidChild", "pid-namespace", "init", marker,
		)...,
	)
	uid, gid := os.Getuid(), os.Getgid()
	command.SysProcAttr = &syscall.SysProcAttr{Cloneflags: unix.CLONE_NEWPID}
	if uid != 0 {
		command.SysProcAttr.Cloneflags |= unix.CLONE_NEWUSER
		command.SysProcAttr.UidMappings = []syscall.SysProcIDMap{{ContainerID: 0, HostID: uid, Size: 1}}
		command.SysProcAttr.GidMappings = []syscall.SysProcIDMap{{ContainerID: 0, HostID: gid, Size: 1}}
		command.SysProcAttr.GidMappingsEnableSetgroups = false
	}
	if err := command.Start(); err != nil {
		if errors.Is(err, syscall.EPERM) {
			t.Skip("kernel denies test-owned PID namespace")
		}
		t.Fatal(err)
	}
	ready := marker + ".ready"
	readyDeadline := time.Now().Add(2 * time.Second)
	for {
		if _, err := os.Stat(ready); err == nil {
			break
		}
		if time.Now().After(readyDeadline) {
			_ = command.Process.Kill()
			_ = command.Wait()
			t.Fatal("PID namespace helper did not become ready before deadline")
		}
		time.Sleep(5 * time.Millisecond)
	}
	if err := command.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	waited := make(chan error, 1)
	go func() { waited <- command.Wait() }()
	select {
	case err := <-waited:
		var exitErr *exec.ExitError
		var waitStatus syscall.WaitStatus
		isExitError := errors.As(err, &exitErr)
		if isExitError && exitErr.ProcessState != nil {
			waitStatus, _ = exitErr.ProcessState.Sys().(syscall.WaitStatus)
		}
		if !isExitError || exitErr.ProcessState == nil ||
			!waitStatus.Signaled() || waitStatus.Signal() != syscall.SIGKILL {
			t.Fatalf("PID namespace init wait=%v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("PID namespace init was not reaped before deadline")
	}
	time.Sleep(500 * time.Millisecond)
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("setsid child escaped PID namespace: %v", err)
	}
}

func TestOSExecutorFailsClosedBeforeStartWhenBoundingSetCannotBeDropped(t *testing.T) {
	kind, mode, marker, helper := guestReexecState()
	if helper && kind != "bounding-failure" {
		return
	}
	if mode == "probe" {
		_ = os.WriteFile(marker, []byte("started"), 0o600)
		os.Exit(0)
	}
	scratch := t.TempDir()
	marker = filepath.Join(scratch, "started")
	executor := OSExecutor{
		commandFactory: func(execution Execution) *exec.Cmd {
			return exec.Command(
				execution.GoBinary,
				reexecArguments(
					"TestOSExecutorFailsClosedBeforeStartWhenBoundingSetCannotBeDropped",
					"bounding-failure", "probe", marker,
				)...,
			)
		},
		dropBoundingSet: func() error { return unix.EPERM },
	}
	_, err := executor.Execute(context.Background(), Execution{
		GoBinary: os.Args[0], Workspace: scratch, WorkingDirectory: scratch,
		Arguments:       []string{"-test.run=^$"},
		OutputDirectory: scratch, MaxOutputBytes: 1 << 20,
	})
	_, markerErr := os.Stat(marker)
	if ErrorCode(err) != CodeExecutionFailed || !os.IsNotExist(markerErr) {
		t.Fatalf("code=%q marker_err=%v err=%v", ErrorCode(err), markerErr, err)
	}
}

func TestOSExecutorRealCapabilityBoundingSetIsEmpty(t *testing.T) {
	kind, mode, statusPath, helper := guestReexecState()
	if helper && kind != "capability-bounding" {
		return
	}
	switch mode {
	case "probe":
		status, err := os.ReadFile("/proc/self/status")
		if err != nil || os.WriteFile(statusPath, status, 0o600) != nil {
			os.Exit(94)
		}
		_, _ = os.Stdout.WriteString(
			"{\"Action\":\"run\",\"Test\":\"TestCapabilityBounding\"}\n" +
				"{\"Action\":\"pass\",\"Test\":\"TestCapabilityBounding\"}\n",
		)
		os.Exit(0)
	case "driver":
		runtime.LockOSThread()
		if err := dropCapabilityBoundingSet(); err != nil {
			t.Fatalf("drop capability bounding set: %v", err)
		}
		if err := unix.Prctl(unix.PR_SET_NO_NEW_PRIVS, 1, 0, 0, 0); err != nil {
			t.Fatalf("set no new privileges: %v", err)
		}
		status, err := os.ReadFile("/proc/self/status")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(statusPath, status, 0o600); err != nil {
			t.Fatal(err)
		}
		return
	}

	scratch := t.TempDir()
	statusPath = filepath.Join(scratch, "status")
	if err := os.WriteFile(statusPath, nil, 0o666); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(scratch, 0o777); err != nil {
		t.Fatal(err)
	}
	command := exec.Command(
		os.Args[0],
		reexecArguments(
			"TestOSExecutorRealCapabilityBoundingSetIsEmpty",
			"capability-bounding", "driver", statusPath,
		)...,
	)
	command.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: unix.CLONE_NEWUSER,
		UidMappings: []syscall.SysProcIDMap{
			{ContainerID: 0, HostID: os.Getuid(), Size: 1},
		},
		GidMappings: []syscall.SysProcIDMap{
			{ContainerID: 0, HostID: os.Getgid(), Size: 1},
		},
		GidMappingsEnableSetgroups: false,
	}
	output, err := command.CombinedOutput()
	if err != nil {
		if errors.Is(err, syscall.EPERM) ||
			strings.Contains(string(output), "operation not permitted") {
			t.Skipf("kernel denies test-owned user namespace: %v: %s", err, output)
		}
		t.Fatalf("capability driver: %v\n%s", err, output)
	}
	status, err := os.ReadFile(statusPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"CapBnd:\t0000000000000000",
		"NoNewPrivs:\t1",
	} {
		if !statusHasExactLine(status, want) {
			t.Fatalf("missing %q in /proc/self/status:\n%s", want, status)
		}
	}
}

func TestPID1SupervisorReapsOrphanStormAndLeavesNoChildren(t *testing.T) {
	const orphanCount = 64
	kind, mode, statePath, helper := guestReexecState()
	if helper && kind != "supervisor-reaper" {
		return
	}
	switch mode {
	case "init":
		result := runPID1Supervisor(
			os.Args[0],
			reexecArguments(
				"TestPID1SupervisorReapsOrphanStormAndLeavesNoChildren",
				"supervisor-reaper", "target", statePath,
			),
			nil,
		)
		value := fmt.Sprintf(
			"exit=%d reaped=%d empty=%t err=%v",
			result.ExitCode, result.Reaped, result.Empty, result.Err,
		)
		if err := os.WriteFile(statePath, []byte(value), 0o600); err != nil {
			os.Exit(90)
		}
		if result.Err != nil || !result.Empty ||
			result.ExitCode != 23 || result.Reaped != orphanCount+1 {
			os.Exit(91)
		}
		return
	case "target":
		for range orphanCount {
			child := exec.Command(
				os.Args[0],
				reexecArguments(
					"TestPID1SupervisorReapsOrphanStormAndLeavesNoChildren",
					"supervisor-reaper", "orphan", statePath,
				)...,
			)
			child.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
			if err := child.Start(); err != nil {
				os.Exit(92)
			}
			if err := child.Process.Release(); err != nil {
				os.Exit(93)
			}
		}
		// Children exit without being waited by this process. This delay makes
		// the supervisor inherit a mix of zombies and live setsid orphans.
		time.Sleep(100 * time.Millisecond)
		os.Exit(23)
	case "orphan":
		os.Exit(0)
	}

	statePath = filepath.Join(t.TempDir(), "supervisor-result")
	command := exec.Command(
		os.Args[0],
		reexecArguments(
			"TestPID1SupervisorReapsOrphanStormAndLeavesNoChildren",
			"supervisor-reaper", "init", statePath,
		)...,
	)
	uid, gid := os.Getuid(), os.Getgid()
	command.SysProcAttr = &syscall.SysProcAttr{Cloneflags: unix.CLONE_NEWPID}
	if uid != 0 {
		command.SysProcAttr.Cloneflags |= unix.CLONE_NEWUSER
		command.SysProcAttr.UidMappings = []syscall.SysProcIDMap{
			{ContainerID: 0, HostID: uid, Size: 1},
		}
		command.SysProcAttr.GidMappings = []syscall.SysProcIDMap{
			{ContainerID: 0, HostID: gid, Size: 1},
		}
		command.SysProcAttr.GidMappingsEnableSetgroups = false
	}
	output, err := command.CombinedOutput()
	if err != nil {
		if errors.Is(err, syscall.EPERM) ||
			strings.Contains(string(output), "operation not permitted") {
			t.Skipf("kernel denies test-owned PID namespace: %v: %s", err, output)
		}
		t.Fatalf("supervisor driver: %v\n%s", err, output)
	}
	value, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatal(err)
	}
	want := fmt.Sprintf("exit=23 reaped=%d empty=true err=<nil>", orphanCount+1)
	if string(value) != want {
		t.Fatalf("supervisor result=%q want=%q", value, want)
	}
}

func reexecArguments(testName, kind, mode, state string) []string {
	return []string{
		"-test.run=^" + testName + "$",
		"--",
		"orquesta-guest-reexec-v1",
		kind,
		mode,
		state,
	}
}

func guestReexecState() (kind, mode, state string, ok bool) {
	for index, argument := range os.Args {
		if argument == "--" && len(os.Args) == index+5 &&
			os.Args[index+1] == "orquesta-guest-reexec-v1" {
			return os.Args[index+2], os.Args[index+3], os.Args[index+4], true
		}
	}
	return "", "", "", false
}

func writeGuestTempFile(t *testing.T, value string) *os.File {
	t.Helper()
	file, err := os.CreateTemp(t.TempDir(), "output-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = file.Close() })
	if _, err := bytes.NewBufferString(value).WriteTo(file); err != nil {
		t.Fatal(err)
	}
	return file
}
