//go:build linux

package bubblewrap

import (
	"bytes"
	"context"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/sys/unix"

	"orquesta/internal/goal"
	"orquesta/internal/identity"
	"orquesta/internal/ports"
)

func TestIdentityRejectsRoot(t *testing.T) {
	for name, identity := range map[string]processIdentity{
		"root_uid":    {euid: 0, egid: 1000},
		"root_gid":    {euid: 1000, egid: 0},
		"root_both":   {euid: 0, egid: 0},
		"unknown_uid": {euid: -1, egid: 1000},
		"unknown_gid": {euid: 1000, egid: -1},
	} {
		t.Run(name, func(t *testing.T) {
			adapter, err := newAdapter(Config{}, identity)
			if adapter != nil || ErrorCode(err) != CodeIdentityUnsafe {
				t.Fatalf("adapter=%v code=%q err=%v", adapter, ErrorCode(err), err)
			}
		})
	}
	if adapter, err := newAdapter(Config{}, processIdentity{euid: 1000, egid: 1000}); adapter != nil || ErrorCode(err) != CodeConfigInvalid {
		t.Fatalf("non-root seam bypassed config validation: adapter=%v code=%q err=%v", adapter, ErrorCode(err), err)
	}
}

func TestCanonicalPassFailResults(t *testing.T) {
	policy := PolicyIdentity{PolicyRef, testDigest("policy")}
	run := testRun(t, policy)
	adapter := testAdapter(t, &testSnapshotSource{}, policy)
	for _, exit := range []int{0, 7} {
		t.Run(fmt.Sprint(exit), func(t *testing.T) {
			want := ports.TestAttestationPassed
			if exit != 0 {
				want = ports.TestAttestationFailed
			}
			outcomes := []ports.RequiredTestOutcome{{RequiredTestRef: run.Request.RequiredTests[0].Ref(), ExitCode: exit, OutputDigest: testDigest("output")}}
			result, err := adapter.result(run.Request, testNow.Add(time.Second), testNow.Add(2*time.Second), want, outcomes)
			if err != nil || result.Verdict != want || ports.ValidateTestAttestationResult(run.Request, result) != nil {
				t.Fatalf("verdict=%s want=%s", result.Verdict, want)
			}
		})
	}
}

func TestSnapshotRejectsMutationTrailingDataAndBudget(t *testing.T) {
	policy := PolicyIdentity{PolicyRef, testDigest("policy")}
	run := testRun(t, policy)
	valid := testStream(t, run.Request, "internal/main.go", []byte("package main\n"))
	for name, content := range map[string][]byte{
		"mutated": func() []byte {
			value := append([]byte(nil), valid...)
			value[len(value)-sha256.Size-2] ^= 1
			return value
		}(),
		"trailing": append(append([]byte(nil), valid...), 0),
	} {
		t.Run(name, func(t *testing.T) {
			snapshot, err := readSnapshotStream(bytes.NewReader(content), run.Request, 1<<20)
			if snapshot != nil {
				_ = snapshot.Close()
			}
			if ErrorCode(err) != CodeSnapshotInvalid {
				t.Fatalf("code=%q err=%v", ErrorCode(err), err)
			}
		})
	}
	flat := testStream(t, run.Request, "main.go", []byte("package main\n"))
	if snapshot, err := readSnapshotStream(bytes.NewReader(flat), run.Request, int64(len(flat)-1)); snapshot != nil || ErrorCode(err) != CodeSnapshotLimit {
		t.Fatalf("budget snapshot=%v code=%q err=%v", snapshot, ErrorCode(err), err)
	}
}

func TestAttestClosesSnapshotStreamReturnedWithError(t *testing.T) {
	policy := PolicyIdentity{PolicyRef, testDigest("policy")}
	run := testRun(t, policy)
	source := &testSnapshotSource{
		openErr: errors.New("snapshot.open_failed"), closeErr: errors.New("snapshot.close_failed"),
	}
	adapter := testAdapter(t, source, policy)
	result, err := adapter.Attest(context.Background(), run)
	if result.SubjectDigest != "" || result.ReceiptRef != "" || ErrorCode(err) != CodeCleanupFailed ||
		source.opens.Load() != 1 || source.closes.Load() != 1 {
		t.Fatalf("result=%+v code=%q opens=%d closes=%d err=%v", result, ErrorCode(err),
			source.opens.Load(), source.closes.Load(), err)
	}
}

func TestSandboxArgumentsDenyAmbientAuthority(t *testing.T) {
	policy := PolicyIdentity{PolicyRef, testDigest("policy")}
	run := testRun(t, policy)
	snapshot, err := readSnapshotStream(bytes.NewReader(testStream(t, run.Request, "internal/main.go", []byte("package main\n"))), run.Request, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	defer snapshot.Close()
	files := testPinnedFiles(t)
	adapter := testAdapter(t, &testSnapshotSource{}, policy)
	adapter.inputs = files
	command, output, arguments, err := adapter.command(snapshot, run.Request.RequiredTests[0])
	if err != nil {
		t.Fatal(err)
	}
	defer arguments.Close()
	args := strings.Join(readArgumentFile(t, arguments.File), "\x00")
	for _, required := range []string{"--unshare-all\x00--unshare-user\x00--disable-userns\x00--assert-userns-disabled", "--clearenv", "--cap-drop\x00ALL", "--uid\x0065534", "--gid\x0065534", "--ro-bind-fd\x004\x00/toolchain", "--ro-bind-data\x005\x00/toolchain/bin/go", "--remount-ro\x00/subject", "--size"} {
		if !strings.Contains(args, required) {
			t.Errorf("missing %q in %q", required, args)
		}
	}
	if got := command.Args[len(command.Args)-4:]; !slices.Equal(got,
		[]string{"--", goCommand, "test", "./..."}) {
		t.Fatalf("test argv=%q", got)
	}
	for _, forbidden := range []string{"--share-net", "--bind\x00/", "/bin/sh", "bash", "\x00-c\x00"} {
		if strings.Contains(args, forbidden) {
			t.Errorf("ambient authority %q in %q", forbidden, args)
		}
	}
	if command.Path != "/proc/self/fd/3" || command.Env == nil || len(command.Env) != 0 || command.SysProcAttr == nil || !command.SysProcAttr.Setpgid || output.remaining != adapter.config.Limits.MaxOutputBytes {
		t.Fatalf("unsafe command path=%q env=%v attrs=%+v output=%d", command.Path, command.Env, command.SysProcAttr, output.remaining)
	}
}

func TestSandboxArgumentsBindPinnedToolchainDirectoryFD(t *testing.T) {
	arguments := sandboxPreambleArguments(t)
	if !strings.Contains(arguments, "--ro-bind-fd\x004\x00/toolchain") ||
		!strings.Contains(arguments, "--ro-bind-data\x005\x00/toolchain/bin/go") ||
		strings.Contains(arguments, "--ro-bind\x00/proc/self/fd") {
		t.Fatalf("toolchain is not descriptor-bound: %q", arguments)
	}
}

func TestSandboxArgumentsDisableFurtherUserNamespaces(t *testing.T) {
	arguments := sandboxPreambleArguments(t)
	if !strings.Contains(arguments, "--disable-userns\x00--assert-userns-disabled") {
		t.Fatalf("nested user namespaces remain available: %q", arguments)
	}
}

func TestSandboxParentExecutesBubblewrapWithTrulyEmptyEnvironment(t *testing.T) {
	policy := PolicyIdentity{PolicyRef, testDigest("policy")}
	run := testRun(t, policy)
	snapshot, err := readSnapshotStream(bytes.NewReader(
		testStream(t, run.Request, "internal/main.go", []byte("package main\n")),
	), run.Request, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	defer snapshot.Close()
	adapter := testAdapter(t, &testSnapshotSource{}, policy)
	adapter.inputs = testPinnedFiles(t)
	command, _, arguments, err := adapter.command(snapshot, run.Request.RequiredTests[0])
	if err != nil {
		t.Fatal(err)
	}
	defer arguments.Close()
	if command.Env == nil || len(command.Env) != 0 {
		t.Fatalf("sandbox parent inherited environment: %#v", command.Env)
	}
}

func TestCommandCleanupClosesAuxiliaryAndPreservesSnapshotDescriptors(t *testing.T) {
	policy := PolicyIdentity{PolicyRef, testDigest("policy")}
	run := testRun(t, policy)
	snapshot, err := readSnapshotStream(bytes.NewReader(
		testStream(t, run.Request, "internal/main.go", []byte("package main\n")),
	), run.Request, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	defer snapshot.Close()
	adapter := testAdapter(t, &testSnapshotSource{}, policy)
	adapter.inputs = testPinnedFiles(t)
	_, _, arguments, err := adapter.command(snapshot, run.Request.RequiredTests[0])
	if err != nil {
		t.Fatal(err)
	}
	auxiliaryFD := arguments.auxiliary[0].Fd()
	snapshotFD := snapshot.entries[0].file.Fd()
	if err := arguments.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := unix.FcntlInt(auxiliaryFD, unix.F_GETFD, 0); !errors.Is(err, unix.EBADF) {
		t.Fatalf("auxiliary descriptor remained open: fd=%d err=%v", auxiliaryFD, err)
	}
	if _, err := unix.FcntlInt(snapshotFD, unix.F_GETFD, 0); err != nil {
		t.Fatalf("snapshot descriptor closed with command inputs: fd=%d err=%v", snapshotFD, err)
	}
}

func TestOutputBudgetIsSharedAndFailClosed(t *testing.T) {
	outputs := &cappedOutputs{remaining: 5, overflow: make(chan struct{})}
	stdout, stderr := cappedWriter{outputs, &outputs.stdout}, cappedWriter{outputs, &outputs.stderr}
	_, _ = stdout.Write([]byte("abc"))
	_, _ = stderr.Write([]byte("def"))
	select {
	case <-outputs.overflow:
	default:
		t.Fatal("combined output overflow not signalled")
	}
	if outputs.stdout.Len()+outputs.stderr.Len() != 5 || exitCode(nil) != 0 {
		t.Fatal("output budget or success exit corrupted")
	}
}

func TestOutputDigestIsDeterministicAcrossConcurrentStreams(t *testing.T) {
	build := func(stderrFirst bool) string {
		outputs := &cappedOutputs{remaining: 1024, overflow: make(chan struct{})}
		stdout, stderr := cappedWriter{outputs, &outputs.stdout}, cappedWriter{outputs, &outputs.stderr}
		if stderrFirst {
			_, _ = stderr.Write([]byte("stderr"))
			_, _ = stdout.Write([]byte("stdout"))
		} else {
			_, _ = stdout.Write([]byte("stdout"))
			_, _ = stderr.Write([]byte("stderr"))
		}
		return outputs.digest()
	}
	if build(false) != build(true) {
		t.Fatal("output digest depends on stdout/stderr scheduling")
	}
}

func TestSnapshotDescriptorBudgetIsLinearAndFailsBeforeExecution(t *testing.T) {
	for _, test := range []struct{ soft, open, concurrent, subject, want int64 }{
		{256, 0, 2, 20, 20}, {256, 0, 4, 40, 32}, {80, 1, 1, 10, 0},
	} {
		if got := snapshotFileLimit(test.soft, test.open, test.concurrent, test.subject); got != test.want {
			t.Fatalf("snapshotFileLimit(%+v)=%d, want %d", test, got, test.want)
		}
	}
}

func TestSealedArgumentsEnforceConfiguredBudgetIncrementally(t *testing.T) {
	arguments, err := newSealedArguments(8)
	if err != nil {
		t.Fatal(err)
	}
	defer arguments.Close()
	if err := arguments.add("abc", "def"); err != nil || arguments.written != 8 {
		t.Fatalf("bounded arguments write=%d err=%v", arguments.written, err)
	}
	if err := arguments.add("x"); ErrorCode(err) != CodeSnapshotLimit || arguments.written != 8 {
		t.Fatalf("overflow write=%d code=%q err=%v", arguments.written, ErrorCode(err), err)
	}
	info, err := arguments.Stat()
	if err != nil || info.Size() != 8 {
		t.Fatalf("argument memfd size=%d err=%v", info.Size(), err)
	}
}

func TestTrustedToolchainRejectsMutableDescendantAndBindsTreeDigest(t *testing.T) {
	root := t.TempDir()
	if err := os.Chmod(root, 0o700); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(root, "bin")
	if err := os.Mkdir(bin, 0o700); err != nil {
		t.Fatal(err)
	}
	tool := filepath.Join(bin, "go")
	if err := os.WriteFile(tool, []byte("version one"), 0o555); err != nil {
		t.Fatal(err)
	}
	bubblewrap := filepath.Join(t.TempDir(), "bwrap")
	if err := os.WriteFile(bubblewrap, []byte("#!/bin/sh\nexit 0\n"), 0o500); err != nil {
		t.Fatal(err)
	}
	inputs, err := openPinnedInputs(Config{BubblewrapCommand: bubblewrap, ToolchainRoot: root}, uint32(os.Geteuid()))
	if err != nil {
		t.Fatal(err)
	}
	defer inputs.Close()
	first := inputs.treeDigest
	if err := inputs.validateToolchain(); err != nil {
		t.Fatalf("unchanged toolchain rejected: %v", err)
	}
	if err := os.Chmod(tool, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tool, []byte("version two"), 0o555); err != nil {
		t.Fatal(err)
	}
	directory, err := os.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	second, identityErr := trustedTreeIdentity(directory, uint32(os.Geteuid()))
	_ = directory.Close()
	if identityErr != nil || first == second {
		t.Fatalf("tree identity first=%q second=%q err=%v", first, second, identityErr)
	}
	if err := inputs.validateToolchain(); ErrorCode(err) != CodeToolchainUnsafe {
		t.Fatalf("captured toolchain drift code=%q err=%v", ErrorCode(err), err)
	}
	if err := os.Chmod(tool, 0o775); err != nil {
		t.Fatal(err)
	}
	directory, err = os.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	_, identityErr = trustedTreeIdentity(directory, uint32(os.Geteuid()))
	_ = directory.Close()
	if identityErr == nil {
		t.Fatal("group-writable toolchain descendant accepted")
	}
}

func TestPreflightExecutesPinnedGoThroughSandboxPath(t *testing.T) {
	policy := PolicyIdentity{PolicyRef, testDigest("policy")}
	adapter := testAdapter(t, &testSnapshotSource{}, policy)
	adapter.inputs = testPinnedFiles(t)
	session := &fakeCgroupSession{}
	controller := &fakeCgroupController{session: session}
	session.controller = controller
	adapter.cgroups = controller
	if err := adapter.preflight(); err != nil || session.closes.Load() != 1 || controller.active.Load() != 0 {
		t.Fatalf("preflight err=%v closes=%d active=%d", err, session.closes.Load(), controller.active.Load())
	}
}

func TestRunTestClosesCommandDescriptorsAfterSuccess(t *testing.T) {
	policy := PolicyIdentity{PolicyRef, testDigest("policy")}
	adapter := testAdapter(t, &testSnapshotSource{}, policy)
	adapter.inputs = testPinnedFiles(t)
	session := &fakeCgroupSession{}
	controller := &fakeCgroupController{session: session}
	session.controller = controller
	adapter.cgroups = controller
	before := openDescriptorCount(t)
	for range 8 {
		if err := adapter.preflight(); err != nil {
			t.Fatal(err)
		}
	}
	if after := openDescriptorCount(t); after != before {
		t.Fatalf("command descriptor leak after success: before=%d after=%d", before, after)
	}
}

func TestRunTestClosesCommandDescriptorsWhenStartFails(t *testing.T) {
	policy := PolicyIdentity{PolicyRef, testDigest("policy")}
	run := testRun(t, policy)
	adapter := testAdapter(t, &testSnapshotSource{}, policy)
	adapter.inputs = testPinnedFilesWithBubblewrap(t, []byte("not-an-executable"))
	session := &fakeCgroupSession{}
	controller := &fakeCgroupController{session: session}
	session.controller = controller
	adapter.cgroups = controller
	before := openDescriptorCount(t)
	for range 8 {
		outcome, err := adapter.runTest(context.Background(), &sandboxSnapshot{}, run.Request.RequiredTests[0])
		if outcome != (ports.RequiredTestOutcome{}) || ErrorCode(err) != CodeExecutionFailed {
			t.Fatalf("outcome=%+v code=%q err=%v", outcome, ErrorCode(err), err)
		}
	}
	if after := openDescriptorCount(t); after != before {
		t.Fatalf("command descriptor leak after start failure: before=%d after=%d", before, after)
	}
}

func TestAttestClosesSnapshotMemfdsWhenStreamCloseFails(t *testing.T) {
	policy := PolicyIdentity{PolicyRef, testDigest("policy")}
	run := testRun(t, policy)
	source := &testSnapshotSource{closeErr: errors.New("snapshot.close_failed")}
	source.content = testStream(t, run.Request, "internal/main.go", []byte("package main\n"))
	adapter := testAdapter(t, source, policy)
	before, err := os.ReadDir("/proc/self/fd")
	if err != nil {
		t.Fatal(err)
	}
	for range 8 {
		if _, err := adapter.Attest(context.Background(), run); ErrorCode(err) != CodeCleanupFailed {
			t.Fatalf("stream close code=%q err=%v", ErrorCode(err), err)
		}
	}
	after, err := os.ReadDir("/proc/self/fd")
	if err != nil || len(after) != len(before) {
		t.Fatalf("snapshot descriptors before=%d after=%d err=%v", len(before), len(after), err)
	}
}

func TestSnapshotParentDiscoveryIsLinearAndDepthBounded(t *testing.T) {
	directories := map[string]struct{}{}
	if err := addParentDirectories(directories, "a/b/c/first.go"); err != nil {
		t.Fatal(err)
	}
	if err := addParentDirectories(directories, "a/b/c/second.go"); err != nil || len(directories) != 3 {
		t.Fatalf("shared parent discovery directories=%v err=%v", directories, err)
	}
	if err := addParentDirectories(map[string]struct{}{}, strings.Repeat("a/", maxSnapshotDepth+1)+"file"); ErrorCode(err) != CodeSnapshotLimit {
		t.Fatalf("deep parent code=%q err=%v", ErrorCode(err), err)
	}
}

func TestPinnedExecutablesSurvivePathSwapAndAreSealed(t *testing.T) {
	root := t.TempDir()
	command := filepath.Join(root, "bwrap")
	toolchain := filepath.Join(root, "go")
	if err := os.MkdirAll(filepath.Join(toolchain, "bin"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(command, []byte("trusted-bwrap"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(toolchain, "bin/go"), []byte("trusted-go"), 0o755); err != nil {
		t.Fatal(err)
	}
	inputs, err := openPinnedInputs(Config{BubblewrapCommand: command, ToolchainRoot: toolchain}, uint32(os.Geteuid()))
	if err != nil {
		t.Fatal(err)
	}
	defer inputs.Close()
	if err := os.Rename(command, command+".old"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(command, []byte("hostile"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := inputs.bubblewrap.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	content, _ := io.ReadAll(inputs.bubblewrap)
	seals, sealErr := unix.FcntlInt(inputs.goBin.Fd(), unix.F_GET_SEALS, 0)
	if string(content) != "trusted-bwrap" || sealErr != nil || seals&unix.F_SEAL_WRITE == 0 || inputs.identity == "" {
		t.Fatalf("pin changed content=%q go-seals=%x err=%v", content, seals, sealErr)
	}
}

func TestSealedExecutableReopensWithIndependentOffsets(t *testing.T) {
	path := filepath.Join(t.TempDir(), "go")
	content := []byte("trusted-go-binary")
	if err := os.WriteFile(path, content, 0o500); err != nil {
		t.Fatal(err)
	}
	source, metadata, err := openPinned(path, uint32(os.Geteuid()), false, CodeToolchainUnsafe)
	if err != nil {
		t.Fatal(err)
	}
	sealed, _, err := sealExecutable(source, metadata, "orquesta-go-offset-test")
	_ = source.Close()
	if err != nil {
		t.Fatal(err)
	}
	defer sealed.Close()
	first, err := reopenSealedExecutable(sealed)
	if err != nil {
		t.Fatal(err)
	}
	defer first.Close()
	second, err := reopenSealedExecutable(sealed)
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	for name, file := range map[string]*os.File{"first": first, "second": second} {
		got := make([]byte, len(content))
		if _, err := io.ReadFull(file, got); err != nil || !bytes.Equal(got, content) {
			t.Fatalf("%s content=%q err=%v", name, got, err)
		}
	}
}

func TestCloseIsConcurrentAndIdempotent(t *testing.T) {
	policy := PolicyIdentity{PolicyRef, testDigest("policy")}
	adapter := testAdapter(t, &testSnapshotSource{}, policy)
	adapter.inputs = testPinnedFiles(t)
	adapter.cgroups = &cgroupRoot{file: testFile(t, "cgroup")}
	closes := make(chan error, 2)
	go func() { closes <- adapter.Close() }()
	go func() { closes <- adapter.Close() }()
	if err := <-closes; err != nil {
		t.Fatal(err)
	}
	if err := <-closes; err != nil {
		t.Fatal(err)
	}
	if adapter.PolicyIdentity() != (PolicyIdentity{}) {
		t.Fatal("closed policy remained available")
	}
}

func TestCgroupParsersFailClosed(t *testing.T) {
	if period, err := cpuPeriod("max 100000"); err != nil || period != 100000 {
		t.Fatalf("period=%d err=%v", period, err)
	}
	for _, invalid := range []string{"", "max", "max 0", "max nope extra"} {
		if _, err := cpuPeriod(invalid); err == nil {
			t.Fatalf("accepted %q", invalid)
		}
	}
	positive, err := eventPositive("oom 1\noom_kill 0", "oom", "oom_kill")
	zero, zeroErr := eventPositive("max 0", "max")
	if err != nil || zeroErr != nil || !positive || zero {
		t.Fatal("cgroup event/control parser corrupted")
	}
}

type testSnapshotSource struct {
	content  []byte
	opens    atomic.Int64
	closes   atomic.Int64
	openErr  error
	closeErr error
}

func (source *testSnapshotSource) OpenSnapshotStream(context.Context, ports.SnapshotVerificationRequest) (io.ReadCloser, error) {
	source.opens.Add(1)
	return &testSnapshotStream{Reader: bytes.NewReader(source.content), source: source}, source.openErr
}
func (*testSnapshotSource) SnapshotIdentity() (string, string) {
	return "snapshot:test", testDigest("source")
}

func testAdapter(t *testing.T, source SnapshotStreamSource, policy PolicyIdentity) *Adapter {
	t.Helper()
	life, cancel := context.WithCancel(context.Background())
	ticks := atomic.Int64{}
	return &Adapter{config: Config{SnapshotSource: source, Now: func() time.Time { return testNow.Add(time.Duration(ticks.Add(1)) * time.Second) }, Limits: Limits{Timeout: time.Minute, CleanupTimeout: time.Second, MaxOutputBytes: 1024, MaxSubjectBytes: 1 << 20, MaxConcurrentRuns: 1, MemoryMaxBytes: 1 << 20, PIDsMax: 16, CPUQuotaMicros: 1000}}, policy: policy, source: [2]string{"snapshot:test", testDigest("source")}, slots: make(chan struct{}, 1), life: life, cancel: cancel, snapshotFileLimit: 128}
}

var testNow = time.Date(2026, 7, 22, 12, 0, 0, 0, time.UTC)

func testRun(t *testing.T, policy PolicyIdentity) ports.TestAttestationRun {
	t.Helper()
	goalRef, _ := goal.NewGoalRef("goal:test")
	item, _ := goal.NewWorkItemRef("work-item:test")
	execution, _ := goal.NewExecutionRef("execution:test")
	workspace, _ := ports.NewExecutionWorkspaceRef("workspace:test")
	change, _ := ports.NewChangeSetRef("change:test")
	repository, _ := identity.NewRepositoryRef("repository:test")
	ref, _ := goal.NewRequiredTestRef("required-test:go")
	tool, _ := goal.NewToolRef(goToolRef)
	spec, _ := goal.NewRequiredTestSpec(goal.RequiredTestSpecInput{Ref: ref, ToolRef: tool, Arguments: []string{"test", "./..."}, WorkingDirectory: "."})
	writes := []string{"internal"}
	subject := ports.TestSubject{GoalRef: goalRef, WorkItemRef: item, ExecutionRef: execution, ExecutionAttempt: 1, PlanGeneration: 1, WorkItemGeneration: 1, AppSpecGeneration: 1, AppSpecHash: testDigest("app"), WorkspaceRef: workspace, WorkspaceBindingDigest: testDigest("workspace"), ChangeSetRef: change, ChangeSetDigest: testDigest("change"), RepositoryRef: repository, ObjectFormat: ports.GitObjectFormatSHA1, BaseOID: strings.Repeat("1", 40), ParentOID: strings.Repeat("2", 40), HeadOID: strings.Repeat("3", 40), TreeOID: strings.Repeat("4", 40), DiffDigest: testDigest("diff"), WriteSetDigest: ports.WorkspaceWriteSetDigest(writes), RequiredTestsDigest: goal.RequiredTestsDigest([]goal.RequiredTestSpec{spec}), PolicyRef: policy.Ref, PolicyDigest: policy.Digest}
	request := ports.TestAttestationRequest{Subject: subject, RequiredTests: []goal.RequiredTestSpec{spec}, IdempotencyKey: "attest:test", RequestedAt: testNow}
	request.SubjectDigest = ports.TestSubjectDigest(subject)
	snapshot := ports.SnapshotVerificationRequest{Subject: subject, SubjectDigest: request.SubjectDigest, ChangedPaths: []string{"internal/main.go"}, WriteSet: writes}
	return ports.TestAttestationRun{Request: request, Snapshot: snapshot}
}

type testSnapshotStream struct {
	*bytes.Reader
	source *testSnapshotSource
}

func (stream *testSnapshotStream) Close() error {
	stream.source.closes.Add(1)
	return stream.source.closeErr
}

type testSnapshotEntry struct {
	mode, name string
	content    []byte
}

func testStream(t *testing.T, request ports.TestAttestationRequest, name string, content []byte) []byte {
	return testStreamEntries(t, request, []testSnapshotEntry{{mode: "100644", name: name, content: content}})
}

func testStreamEntries(t *testing.T, request ports.TestAttestationRequest, entries []testSnapshotEntry) []byte {
	t.Helper()
	var output bytes.Buffer
	digest := sha256.New()
	writer := io.MultiWriter(&output, digest)
	_, _ = writer.Write(snapshotStreamMagic)
	for _, field := range []string{string(request.Subject.ObjectFormat), request.SubjectDigest, request.Subject.HeadOID, request.Subject.TreeOID} {
		testField(writer, field)
	}
	for _, entry := range entries {
		object := sha1.New()
		_, _ = fmt.Fprintf(object, "blob %d%c", len(entry.content), byte(0))
		_, _ = object.Write(entry.content)
		_, _ = writer.Write([]byte{snapshotEntryTag})
		for _, field := range []string{entry.mode, entry.name, hex.EncodeToString(object.Sum(nil))} {
			testField(writer, field)
		}
		var size [8]byte
		binary.BigEndian.PutUint64(size[:], uint64(len(entry.content)))
		_, _ = writer.Write(size[:])
		_, _ = writer.Write(entry.content)
	}
	_ = output.WriteByte(snapshotEndTag)
	_, _ = output.Write(digest.Sum(nil))
	return output.Bytes()
}
func testField(writer io.Writer, value string) {
	var size [4]byte
	binary.BigEndian.PutUint32(size[:], uint32(len(value)))
	_, _ = writer.Write(size[:])
	_, _ = io.WriteString(writer, value)
}
func testDigest(value string) string {
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])
}
func testFile(t *testing.T, value string) *os.File {
	t.Helper()
	file, err := os.CreateTemp(t.TempDir(), value)
	if err != nil {
		t.Fatal(err)
	}
	return file
}
func testPinnedFiles(t *testing.T) *pinnedInputs {
	t.Helper()
	return testPinnedFilesWithBubblewrap(t, []byte("#!/bin/sh\nexit 0\n"))
}

func testPinnedFilesWithBubblewrap(t *testing.T, bubblewrapContent []byte) *pinnedInputs {
	t.Helper()
	root := t.TempDir()
	if err := os.Chmod(root, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "bin"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "bin", "go"), []byte("test-go"), 0o500); err != nil {
		t.Fatal(err)
	}
	bubblewrapPath := filepath.Join(t.TempDir(), "bwrap")
	if err := os.WriteFile(bubblewrapPath, bubblewrapContent, 0o500); err != nil {
		t.Fatal(err)
	}
	inputs, err := openPinnedInputs(Config{BubblewrapCommand: bubblewrapPath, ToolchainRoot: root}, uint32(os.Geteuid()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := inputs.Close(); err != nil && !errors.Is(err, os.ErrClosed) {
			t.Error(err)
		}
	})
	return inputs
}

func openDescriptorCount(t *testing.T) int {
	t.Helper()
	entries, err := os.ReadDir("/proc/self/fd")
	if err != nil {
		t.Fatal(err)
	}
	return len(entries)
}

func readArgumentFile(t *testing.T, file *os.File) []string {
	t.Helper()
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	content, err := io.ReadAll(file)
	if err != nil || len(content) == 0 || content[len(content)-1] != 0 {
		t.Fatalf("argument file invalid: bytes=%d err=%v", len(content), err)
	}
	parts := strings.Split(string(content[:len(content)-1]), "\x00")
	return parts
}

func sandboxPreambleArguments(t *testing.T) string {
	t.Helper()
	arguments, err := newSealedArguments(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer arguments.Close()
	if err := appendSandboxPreamble(arguments); err != nil {
		t.Fatal(err)
	}
	return strings.Join(readArgumentFile(t, arguments.File), "\x00")
}
