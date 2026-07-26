//go:build linux

package firecrackerguest

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"golang.org/x/sys/unix"

	firecracker "orquesta/internal/testattestorprotocol/rawdrive"
)

func TestRunDriveRoundTripPassAndFailurePreservesOrder(t *testing.T) {
	input := validGuestInput()
	snapshot := guestSourceSnapshot(t, input.SubjectDigest)
	executor := &fakeExecutor{results: []ExecutionResult{
		{ExitCode: 0, OutputDigest: guestTestDigest("pass"), TestCases: 1},
		{ExitCode: 1, OutputDigest: guestTestDigest("fail"), TestCases: 1},
	}}
	output, prepared, runErr := runGuest(t, input, snapshot, executor)
	if runErr != nil {
		t.Fatal(runErr)
	}
	if output.SubjectDigest != input.SubjectDigest || output.RunNonce != input.RunNonce ||
		output.Verdict != firecracker.VerdictFailed || output.ErrorCode != "" ||
		len(output.Outcomes) != 2 ||
		output.Outcomes[0].RequiredTestRef != input.RequiredTests[0].Ref ||
		output.Outcomes[1].RequiredTestRef != input.RequiredTests[1].Ref ||
		output.Outcomes[0].ExitCode != 0 || output.Outcomes[1].ExitCode != 1 {
		t.Fatalf("output=%+v", output)
	}
	if len(executor.executions) != 2 {
		t.Fatalf("executions=%d", len(executor.executions))
	}
	if executor.executions[0].OutputDirectory == executor.executions[1].OutputDirectory {
		t.Fatal("required tests shared mutable scratch")
	}
	assertGuestExecution(t, executor.executions[0], input.MaxOutputBytes)
	if len(prepared) != 12 {
		t.Fatalf("prepared paths=%v", prepared)
	}
}

func TestRunPreservesFailuresAndRejectsSuccessfulZeroTests(t *testing.T) {
	tests := []struct {
		name        string
		result      ExecutionResult
		wantExit    uint8
		wantVerdict firecracker.Verdict
	}{
		{
			name: "executed test passes",
			result: ExecutionResult{
				ExitCode: 0, TestCases: 1, OutputDigest: guestTestDigest("pass"),
			},
			wantExit: 0, wantVerdict: firecracker.VerdictPassed,
		},
		{
			name: "successful process without tests fails",
			result: ExecutionResult{
				ExitCode: 0, TestCases: 0, OutputDigest: guestTestDigest("empty"),
			},
			wantExit: NoTestsExitCode, wantVerdict: firecracker.VerdictFailed,
		},
		{
			name: "real process failure is preserved",
			result: ExecutionResult{
				ExitCode: 7, TestCases: 0, OutputDigest: guestTestDigest("failed"),
			},
			wantExit: 7, wantVerdict: firecracker.VerdictFailed,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := validGuestInput()
			input.RequiredTests = input.RequiredTests[:1]
			output, _, runErr := runGuest(
				t,
				input,
				guestSourceSnapshot(t, input.SubjectDigest),
				&fakeExecutor{results: []ExecutionResult{test.result}},
			)
			if runErr != nil {
				t.Fatal(runErr)
			}
			if output.Verdict != test.wantVerdict || len(output.Outcomes) != 1 ||
				output.Outcomes[0].ExitCode != test.wantExit {
				t.Fatalf("output=%+v want verdict/exit=%d/%d",
					output, test.wantVerdict, test.wantExit)
			}
		})
	}
}

func TestRunUsesCleanWritableScratchForEveryRequiredTest(t *testing.T) {
	input := validGuestInput()
	executor := &fakeExecutor{
		results: []ExecutionResult{
			{OutputDigest: guestTestDigest("first"), TestCases: 1},
			{OutputDigest: guestTestDigest("second"), TestCases: 1},
		},
		inspect: func(index int, execution Execution) error {
			poison := filepath.Join(execution.OutputDirectory, "poison")
			if index == 0 {
				return os.WriteFile(poison, []byte("poison"), 0o600)
			}
			if _, err := os.Stat(poison); !os.IsNotExist(err) {
				return errors.New("second required test observed first writable scratch")
			}
			return nil
		},
	}
	output, _, runErr := runGuest(t, input, guestSourceSnapshot(t, input.SubjectDigest), executor)
	if runErr != nil || output.Verdict != firecracker.VerdictPassed {
		t.Fatalf("output=%+v err=%v", output, runErr)
	}
	if executor.executions[0].OutputDirectory == executor.executions[1].OutputDirectory {
		t.Fatal("required tests reused writable scratch")
	}
}

func TestRunFailsClosedWhenProcessCleanupFails(t *testing.T) {
	input := validGuestInput()
	input.RequiredTests = input.RequiredTests[:1]
	snapshot := guestSourceSnapshot(t, input.SubjectDigest)
	inputDrive := encodeGuestInput(t, input, snapshot)
	outputDrive := newGuestMemoryDrive(4 * firecracker.SectorSize)
	root := t.TempDir()
	err := Run(context.Background(), RunRequest{
		InputDrive: bytes.NewReader(inputDrive), InputDriveBytes: uint64(len(inputDrive)),
		OutputDrive: outputDrive, OutputDriveBytes: uint64(len(outputDrive.data)),
		ScratchRoot: root, GoBinary: "/toolchain/bin/go",
		Executor: &fakeExecutor{results: []ExecutionResult{{
			ExitCode: 0, OutputDigest: guestTestDigest("pass"), TestCases: 1,
		}}},
		NetworkCheck:    func() error { return nil },
		ScratchCheck:    func(string) error { return nil },
		CapacityCheck:   func(string, uint64) error { return nil },
		PrepareIdentity: func([]string) error { return nil },
		CleanupTree: func(ctx context.Context, path string, maximum uint64) error {
			cleanupErr := cleanupPrivateTree(ctx, path, maximum)
			if strings.HasPrefix(filepath.Base(path), "process-") {
				return errors.Join(errors.New("injected cleanup failure"), cleanupErr)
			}
			return cleanupErr
		},
	})
	output := decodeGuestOutput(t, outputDrive.data)
	if ErrorCode(err) != CodeCleanupFailed || output.ErrorCode != CodeCleanupFailed ||
		output.Verdict != 0 || len(output.Outcomes) != 0 {
		t.Fatalf("code=%q output=%+v err=%v", ErrorCode(err), output, err)
	}
}

func TestRunWritesStableErrorsForOutputLimitTimeoutUnsupportedToolAndSnapshot(t *testing.T) {
	tests := []struct {
		name     string
		mutate   func(*firecracker.Input, *[]byte)
		executor *fakeExecutor
		want     string
	}{
		{
			name: "output limit",
			executor: &fakeExecutor{results: []ExecutionResult{
				{OutputDigest: guestTestDigest("limited"), OutputLimit: true},
			}},
			want: CodeOutputLimit,
		},
		{
			name: "timeout",
			executor: &fakeExecutor{results: []ExecutionResult{
				{OutputDigest: guestTestDigest("timeout"), TimedOut: true},
			}},
			want: CodeExecutionTimeout,
		},
		{
			name: "tool unsupported",
			mutate: func(input *firecracker.Input, _ *[]byte) {
				input.RequiredTests = input.RequiredTests[:1]
				input.RequiredTests[0].ToolRef = "tool:shell"
			},
			executor: &fakeExecutor{},
			want:     CodeToolUnsupported,
		},
		{
			name: "snapshot checksum",
			mutate: func(input *firecracker.Input, snapshot *[]byte) {
				input.RequiredTests = input.RequiredTests[:1]
				(*snapshot)[len(*snapshot)-1] ^= 1
			},
			executor: &fakeExecutor{},
			want:     CodeSnapshotInvalid,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := validGuestInput()
			input.RequiredTests = input.RequiredTests[:1]
			snapshot := guestSourceSnapshot(t, input.SubjectDigest)
			if test.mutate != nil {
				test.mutate(&input, &snapshot)
			}
			output, _, _ := runGuest(t, input, snapshot, test.executor)
			if output.SubjectDigest != input.SubjectDigest || output.RunNonce != input.RunNonce ||
				output.ErrorCode != test.want || output.Verdict != 0 || len(output.Outcomes) != 0 {
				t.Fatalf("output=%+v want=%q", output, test.want)
			}
		})
	}
}

func TestRunRejectsNetworkBeforeExecutionAndCommitsError(t *testing.T) {
	input := validGuestInput()
	input.RequiredTests = input.RequiredTests[:1]
	snapshot := guestSourceSnapshot(t, input.SubjectDigest)
	inputDrive := encodeGuestInput(t, input, snapshot)
	outputDrive := newGuestMemoryDrive(4 * firecracker.SectorSize)
	executor := &fakeExecutor{}
	root := t.TempDir()
	err := Run(context.Background(), RunRequest{
		InputDrive: bytes.NewReader(inputDrive), InputDriveBytes: uint64(len(inputDrive)),
		OutputDrive: outputDrive, OutputDriveBytes: uint64(len(outputDrive.data)),
		ScratchRoot: root, GoBinary: "/toolchain/bin/go", Executor: executor,
		NetworkCheck:    func() error { return errors.New("eth0 exists") },
		ScratchCheck:    func(string) error { return nil },
		CapacityCheck:   func(string, uint64) error { return nil },
		PrepareIdentity: func([]string) error { return nil },
		CleanupTree:     cleanupPrivateTree,
	})
	if ErrorCode(err) != CodeNetworkAvailable || len(executor.executions) != 0 {
		t.Fatalf("code=%q executions=%d err=%v", ErrorCode(err), len(executor.executions), err)
	}
	output := decodeGuestOutput(t, outputDrive.data)
	if output.ErrorCode != CodeNetworkAvailable {
		t.Fatalf("output=%+v", output)
	}
}

func TestRunCapacityPreflightFailsBeforeReadingInput(t *testing.T) {
	input := validGuestInput()
	inputDrive := encodeGuestInput(t, input, guestSourceSnapshot(t, input.SubjectDigest))
	counted := &countingReader{reader: bytes.NewReader(inputDrive)}
	root := t.TempDir()
	capacityCalls := 0
	err := Run(context.Background(), RunRequest{
		InputDrive: counted, InputDriveBytes: uint64(len(inputDrive)),
		OutputDrive:      newGuestMemoryDrive(4 * firecracker.SectorSize),
		OutputDriveBytes: 4 * firecracker.SectorSize,
		ScratchRoot:      root, GoBinary: "/toolchain/bin/go", Executor: &fakeExecutor{},
		NetworkCheck: func() error { return nil },
		ScratchCheck: func(string) error { return nil },
		CapacityCheck: func(_ string, required uint64) error {
			capacityCalls++
			want, ok := checkedScratchBytes(uint64(len(inputDrive)), scratchFixedReserveBytes)
			if !ok || required != want {
				t.Fatalf("required=%d want=%d", required, want)
			}
			return errors.New("insufficient tmpfs")
		},
		PrepareIdentity: func([]string) error { return nil },
		CleanupTree:     cleanupPrivateTree,
	})
	if ErrorCode(err) != CodeScratchCapacity || capacityCalls != 1 || counted.reads != 0 {
		t.Fatalf("code=%q capacity_calls=%d reads=%d err=%v", ErrorCode(err), capacityCalls, counted.reads, err)
	}
	entries, readErr := os.ReadDir(root)
	if readErr != nil || len(entries) != 0 {
		t.Fatalf("scratch entries=%v err=%v", entries, readErr)
	}
}

func TestRunCapacityOverflowFailsBeforeReadingInput(t *testing.T) {
	counted := &countingReader{reader: bytes.NewReader(nil)}
	err := Run(context.Background(), RunRequest{
		InputDrive: counted, InputDriveBytes: ^uint64(0),
		OutputDrive:      newGuestMemoryDrive(4 * firecracker.SectorSize),
		OutputDriveBytes: 4 * firecracker.SectorSize,
		ScratchRoot:      t.TempDir(), GoBinary: "/toolchain/bin/go", Executor: &fakeExecutor{},
		NetworkCheck:    func() error { return nil },
		ScratchCheck:    func(string) error { return nil },
		CapacityCheck:   func(string, uint64) error { t.Fatal("capacity called after overflow"); return nil },
		PrepareIdentity: func([]string) error { return nil },
		CleanupTree:     cleanupPrivateTree,
	})
	if ErrorCode(err) != CodeScratchCapacity || counted.reads != 0 {
		t.Fatalf("code=%q reads=%d err=%v", ErrorCode(err), counted.reads, err)
	}
}

func TestRunCapacityAccountsForSnapshotWorkspaceCachesAndOutput(t *testing.T) {
	input := validGuestInput()
	input.RequiredTests = input.RequiredTests[:1]
	snapshot := guestSourceSnapshot(t, input.SubjectDigest)
	inputDrive := encodeGuestInput(t, input, snapshot)
	outputDrive := newGuestMemoryDrive(4 * firecracker.SectorSize)
	var required []uint64
	err := Run(context.Background(), RunRequest{
		InputDrive: bytes.NewReader(inputDrive), InputDriveBytes: uint64(len(inputDrive)),
		OutputDrive: outputDrive, OutputDriveBytes: uint64(len(outputDrive.data)),
		ScratchRoot: t.TempDir(), GoBinary: "/toolchain/bin/go",
		Executor: &fakeExecutor{results: []ExecutionResult{{
			OutputDigest: guestTestDigest("pass"), TestCases: 1,
		}}},
		NetworkCheck:    func() error { return nil },
		ScratchCheck:    func(string) error { return nil },
		CapacityCheck:   func(_ string, value uint64) error { required = append(required, value); return nil },
		PrepareIdentity: func([]string) error { return nil },
		CleanupTree:     cleanupPrivateTree,
	})
	want, ok := requiredScratchPeak(uint64(len(snapshot)), input.MaxOutputBytes)
	if err != nil || !ok || len(required) != 2 || required[1] != want {
		t.Fatalf("required=%v want_second=%d err=%v", required, want, err)
	}
}

func TestRequiredScratchPeakModelsMaterializationAndExecutionBoundaries(t *testing.T) {
	tests := []struct {
		name              string
		snapshot, output  uint64
		wantMaterializing bool
	}{
		{
			name:     "execution peak",
			snapshot: 1 << 20, output: 32 << 20,
		},
		{
			name:     "materialization peak",
			snapshot: 512 << 20, output: 1,
			wantMaterializing: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			materialization, materializationOK := checkedScratchBytes(
				test.snapshot, test.snapshot, scratchFixedReserveBytes,
			)
			execution, executionOK := checkedScratchBytes(
				test.snapshot, scratchCacheReserveBytes, test.output, scratchFixedReserveBytes,
			)
			required, ok := requiredScratchPeak(test.snapshot, test.output)
			want := execution
			if test.wantMaterializing {
				want = materialization
			}
			if !materializationOK || !executionOK || !ok || required != want {
				t.Fatalf(
					"materialization=%d execution=%d required=%d want=%d ok=%v",
					materialization, execution, required, want, ok,
				)
			}
			if required-1 >= required {
				t.Fatal("invalid lower capacity boundary")
			}
		})
	}
	if _, ok := requiredScratchPeak(^uint64(0)/2+1, 1); ok {
		t.Fatal("materialization overflow accepted")
	}
	if _, ok := requiredScratchPeak(1, ^uint64(0)); ok {
		t.Fatal("execution overflow accepted")
	}
}

func TestRunScratchPeakAcceptsExactCapacityAndRejectsOneByteLess(t *testing.T) {
	input := validGuestInput()
	input.RequiredTests = input.RequiredTests[:1]
	snapshot := guestSourceSnapshot(t, input.SubjectDigest)
	inputDrive := encodeGuestInput(t, input, snapshot)
	required, ok := requiredScratchPeak(uint64(len(snapshot)), input.MaxOutputBytes)
	if !ok || required == 0 {
		t.Fatalf("required=%d ok=%v", required, ok)
	}
	for _, test := range []struct {
		name      string
		available uint64
		wantCode  string
	}{
		{name: "exact", available: required},
		{name: "one byte less", available: required - 1, wantCode: CodeScratchCapacity},
	} {
		t.Run(test.name, func(t *testing.T) {
			outputDrive := newGuestMemoryDrive(4 * firecracker.SectorSize)
			executor := &fakeExecutor{results: []ExecutionResult{{
				OutputDigest: guestTestDigest("pass"), TestCases: 1,
			}}}
			capacityCalls := 0
			err := Run(context.Background(), RunRequest{
				InputDrive: bytes.NewReader(inputDrive), InputDriveBytes: uint64(len(inputDrive)),
				OutputDrive: outputDrive, OutputDriveBytes: uint64(len(outputDrive.data)),
				ScratchRoot: t.TempDir(), GoBinary: "/toolchain/bin/go", Executor: executor,
				NetworkCheck: func() error { return nil },
				ScratchCheck: func(string) error { return nil },
				CleanupTree:  cleanupPrivateTree,
				PrepareIdentity: func([]string) error {
					return nil
				},
				CapacityCheck: func(_ string, value uint64) error {
					capacityCalls++
					if value > test.available {
						return errors.New("capacity boundary")
					}
					return nil
				},
			})
			if ErrorCode(err) != test.wantCode || capacityCalls != 2 {
				t.Fatalf(
					"code=%q want=%q capacity_calls=%d err=%v",
					ErrorCode(err), test.wantCode, capacityCalls, err,
				)
			}
			output := decodeGuestOutput(t, outputDrive.data)
			if test.wantCode == "" {
				if output.Verdict != firecracker.VerdictPassed || len(executor.executions) != 1 {
					t.Fatalf("output=%+v executions=%d", output, len(executor.executions))
				}
				return
			}
			if output.ErrorCode != test.wantCode || len(executor.executions) != 0 {
				t.Fatalf("output=%+v executions=%d", output, len(executor.executions))
			}
		})
	}
}

func TestRequirePrivateTmpfsExactRootOwnershipAndMode(t *testing.T) {
	if os.Geteuid() != 0 {
		t.Skip("mount probe requires root")
	}
	parent := t.TempDir()
	sameMount := filepath.Join(parent, "same-mount")
	if err := os.Mkdir(sameMount, 0o711); err != nil {
		t.Fatal(err)
	}
	if err := RequirePrivateTmpfs(sameMount); ErrorCode(err) != CodeScratchUnavailable {
		t.Fatalf("parent mount accepted: %v", err)
	}
	for _, test := range []struct {
		name  string
		flags uintptr
		valid bool
	}{
		{"required flags and exec", unix.MS_NODEV | unix.MS_NOSUID, true},
		{"missing nodev", unix.MS_NOSUID, false},
		{"missing nosuid", unix.MS_NODEV, false},
		{"noexec forbidden", unix.MS_NODEV | unix.MS_NOSUID | unix.MS_NOEXEC, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := filepath.Join(parent, strings.ReplaceAll(test.name, " ", "-"))
			if err := os.Mkdir(root, 0o700); err != nil {
				t.Fatal(err)
			}
			if err := unix.Mount("tmpfs", root, "tmpfs", test.flags, "mode=0711,size=4m"); err != nil {
				if errors.Is(err, unix.EPERM) {
					t.Skip("mount namespace denies tmpfs probe")
				}
				t.Fatal(err)
			}
			t.Cleanup(func() {
				_ = unix.Unmount(root, 0)
				if err := cleanupPrivateTree(
					context.Background(), root, maximumCleanupRecords(),
				); err != nil {
					t.Error(err)
				}
			})
			err := RequirePrivateTmpfs(root)
			if !test.valid {
				if ErrorCode(err) != CodeScratchUnavailable {
					t.Fatalf("unsafe mount accepted: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			probe := filepath.Join(root, "exec-probe")
			if err := os.WriteFile(probe, []byte("#!/bin/sh\nexit 0\n"), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := exec.Command(probe).Run(); err != nil {
				t.Fatalf("scratch unexpectedly noexec: %v", err)
			}
			if err := os.Chmod(root, 0o700); err != nil {
				t.Fatal(err)
			}
			if err := RequirePrivateTmpfs(root); ErrorCode(err) != CodeScratchUnavailable {
				t.Fatalf("wrong mode accepted: %v", err)
			}
		})
	}
}

func TestCleanupPrivateTreeIsIterativeContextAwareAndBounded(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "deep")
	if err := os.Mkdir(root, 0o700); err != nil {
		t.Fatal(err)
	}
	current := root
	directories := []string{root}
	for range 300 {
		current = filepath.Join(current, "d")
		if err := os.Mkdir(current, 0o700); err != nil {
			t.Fatal(err)
		}
		directories = append(directories, current)
	}
	if err := os.WriteFile(filepath.Join(current, "leaf"), []byte("leaf"), 0o400); err != nil {
		t.Fatal(err)
	}
	for index := len(directories) - 1; index >= 0; index-- {
		if err := os.Chmod(directories[index], 0o500); err != nil {
			t.Fatal(err)
		}
	}
	if err := cleanupPrivateTree(context.Background(), root, 302); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(root); !os.IsNotExist(err) {
		t.Fatalf("deep tree remains: %v", err)
	}

	bounded := filepath.Join(parent, "bounded")
	if err := os.Mkdir(bounded, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bounded, "file"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := cleanupPrivateTree(context.Background(), bounded, 1); !errors.Is(err, unix.EFBIG) {
		t.Fatalf("limit not enforced: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := cleanupPrivateTree(ctx, bounded, 2); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation not preserved: %v", err)
	}
	if err := cleanupPrivateTree(context.Background(), bounded, 2); err != nil {
		t.Fatal(err)
	}
}

func TestCanonicalGoArgumentsRejectNonTestAndCannotDisableJSON(t *testing.T) {
	required := firecracker.RequiredTest{
		ToolRef: "tool:go", Arguments: []string{"test", "-count=1", "./..."},
	}
	arguments, err := canonicalGoTestArguments(required)
	if err != nil || !reflect.DeepEqual(arguments, []string{"test", "-json", "-count=1", "./..."}) {
		t.Fatalf("arguments=%v err=%v", arguments, err)
	}
	for _, alias := range []string{"-json", "--json", "-json=false", "--json=false", "-json=true", "--json=true"} {
		required.Arguments = []string{"test", alias, "./..."}
		if _, err := canonicalGoTestArguments(required); ErrorCode(err) != CodeToolUnsupported {
			t.Fatalf("alias=%q code=%q", alias, ErrorCode(err))
		}
	}
	required.Arguments = []string{"test", "-args", "--json=false"}
	arguments, err = canonicalGoTestArguments(required)
	if err != nil || !reflect.DeepEqual(arguments, []string{"test", "-json", "-args", "--json=false"}) {
		t.Fatalf("arguments after -args=%v err=%v", arguments, err)
	}
	required.Arguments = []string{"version"}
	if _, err := canonicalGoTestArguments(required); ErrorCode(err) != CodeToolUnsupported {
		t.Fatalf("non-test code=%q", ErrorCode(err))
	}
}

func runGuest(
	t *testing.T,
	input firecracker.Input,
	snapshot []byte,
	executor *fakeExecutor,
) (firecracker.Output, []string, error) {
	t.Helper()
	inputDrive := encodeGuestInput(t, input, snapshot)
	outputDrive := newGuestMemoryDrive(4 * firecracker.SectorSize)
	root := t.TempDir()
	var prepared []string
	err := Run(context.Background(), RunRequest{
		InputDrive: bytes.NewReader(inputDrive), InputDriveBytes: uint64(len(inputDrive)),
		OutputDrive: outputDrive, OutputDriveBytes: uint64(len(outputDrive.data)),
		ScratchRoot: root, GoBinary: "/toolchain/bin/go", Executor: executor,
		NetworkCheck:  func() error { return nil },
		ScratchCheck:  func(string) error { return nil },
		CapacityCheck: func(string, uint64) error { return nil },
		PrepareIdentity: func(paths []string) error {
			prepared = append(prepared, paths...)
			return nil
		},
		CleanupTree: cleanupPrivateTree,
	})
	entries, readErr := os.ReadDir(root)
	if readErr != nil || len(entries) != 0 {
		t.Fatalf("guest scratch not cleaned entries=%v err=%v", entries, readErr)
	}
	return decodeGuestOutput(t, outputDrive.data), prepared, err
}

func validGuestInput() firecracker.Input {
	return firecracker.Input{
		SubjectDigest:  guestTestDigest("subject"),
		RunNonce:       guestTestDigest("nonce"),
		PolicyRef:      "policy:test-attestor:firecracker:v1",
		PolicyDigest:   guestTestDigest("policy"),
		MaxOutputBytes: 64 * 1024,
		RequiredTests: []firecracker.RequiredTest{
			{
				Ref: "required:test-a", ToolRef: "tool:go",
				Arguments:        []string{"test", "-count=1", "-run", "^TestA$", "./..."},
				WorkingDirectory: ".",
			},
			{
				Ref: "required:test-b", ToolRef: "tool:go",
				Arguments:        []string{"test", "-run", "^TestB$", "./..."},
				WorkingDirectory: ".",
			},
		},
	}
}

func guestSourceSnapshot(t *testing.T, subject string) []byte {
	t.Helper()
	return buildSnapshot(t, subject, []testSnapshotEntry{
		{mode: "100644", name: "go.mod", content: []byte("module guesttest\n\ngo 1.25\n")},
		{
			mode: "100644", name: "subject_test.go",
			content: []byte("package guesttest\nimport \"testing\"\nfunc TestA(t *testing.T){}\nfunc TestB(t *testing.T){}\n"),
		},
	}, nil)
}

func encodeGuestInput(t *testing.T, input firecracker.Input, snapshot []byte) []byte {
	t.Helper()
	driveBytes, err := firecracker.InputDriveSize(input, uint64(len(snapshot)))
	if err != nil {
		t.Fatal(err)
	}
	var drive bytes.Buffer
	if err := firecracker.WriteInputDrive(
		&drive, driveBytes, input, bytes.NewReader(snapshot), uint64(len(snapshot)),
	); err != nil {
		t.Fatal(err)
	}
	return drive.Bytes()
}

func decodeGuestOutput(t *testing.T, drive []byte) firecracker.Output {
	t.Helper()
	output, err := firecracker.ReadOutputDrive(bytes.NewReader(drive), uint64(len(drive)))
	if err != nil {
		t.Fatal(err)
	}
	return output
}

type fakeExecutor struct {
	results    []ExecutionResult
	errs       []error
	executions []Execution
	inspect    func(int, Execution) error
}

func (executor *fakeExecutor) Execute(_ context.Context, execution Execution) (ExecutionResult, error) {
	executor.executions = append(executor.executions, execution)
	index := len(executor.executions) - 1
	info, err := os.Stat(filepath.Join(execution.Workspace, "go.mod"))
	if err != nil || info.Mode().Perm() != 0o444 {
		return ExecutionResult{}, errors.New("workspace not materialized read-only")
	}
	if executor.inspect != nil {
		if err := executor.inspect(index, execution); err != nil {
			return ExecutionResult{}, err
		}
	}
	if index < len(executor.errs) && executor.errs[index] != nil {
		return ExecutionResult{}, executor.errs[index]
	}
	if index >= len(executor.results) {
		return ExecutionResult{}, errors.New("missing fake result")
	}
	return executor.results[index], nil
}

func assertGuestExecution(t *testing.T, execution Execution, maxOutput uint64) {
	t.Helper()
	if execution.GoBinary != "/toolchain/bin/go" ||
		execution.UID != nobodyID || execution.GID != nobodyID || !execution.ClearSupplementaryGroups ||
		execution.MaxOutputBytes != maxOutput ||
		!reflect.DeepEqual(execution.Arguments, []string{
			"test", "-json", "-count=1", "-run", "^TestA$", "./...",
		}) {
		t.Fatalf("execution=%+v", execution)
	}
	var scratch string
	for _, value := range execution.Environment {
		if strings.HasPrefix(value, "GOCACHE=") {
			scratch = filepath.Dir(strings.TrimPrefix(value, "GOCACHE="))
		}
	}
	if scratch == "" {
		t.Fatal("GOCACHE missing")
	}
	wantEnv := minimalEnvironment(scratch)
	got, want := append([]string(nil), execution.Environment...), append([]string(nil), wantEnv...)
	sort.Strings(got)
	sort.Strings(want)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("env=%v want=%v", got, want)
	}
	hasGoRoot, hasPath := false, false
	for _, value := range execution.Environment {
		hasGoRoot = hasGoRoot || value == "GOROOT=/toolchain"
		hasPath = hasPath || value == "PATH=/toolchain/bin"
		if stringsContainsSensitiveEnv(value) {
			t.Fatalf("unexpected env=%q", value)
		}
	}
	if !hasGoRoot || !hasPath {
		t.Fatalf("exact toolchain environment missing: %v", execution.Environment)
	}
}

func stringsContainsSensitiveEnv(value string) bool {
	for _, prefix := range []string{"TOKEN=", "SSH_", "AWS_", "HOME=/home/"} {
		if len(value) >= len(prefix) && value[:len(prefix)] == prefix {
			return true
		}
	}
	return false
}

type guestMemoryDrive struct {
	data  []byte
	syncs int
}

type countingReader struct {
	reader io.Reader
	reads  int
}

func (reader *countingReader) Read(value []byte) (int, error) {
	reader.reads++
	return reader.reader.Read(value)
}

func newGuestMemoryDrive(size int) *guestMemoryDrive {
	return &guestMemoryDrive{data: make([]byte, size)}
}

func (drive *guestMemoryDrive) WriteAt(value []byte, offset int64) (int, error) {
	if offset < 0 || offset > int64(len(drive.data)) ||
		int64(len(value)) > int64(len(drive.data))-offset {
		return 0, io.ErrShortWrite
	}
	copy(drive.data[offset:], value)
	return len(value), nil
}

func (drive *guestMemoryDrive) Sync() error {
	drive.syncs++
	return nil
}
