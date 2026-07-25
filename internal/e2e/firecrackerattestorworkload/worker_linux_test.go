//go:build linux

package firecrackerattestorworkload

import (
	"context"
	"encoding/hex"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"orquesta/internal/adapters/attestor/firecrackerclient"
	"orquesta/internal/adapters/attestor/firecrackerguest"
	"orquesta/internal/adapters/workspace/gitlocal"
	"orquesta/internal/goal"
	"orquesta/internal/identity"
	"orquesta/internal/ports"
	launcherprotocol "orquesta/internal/testattestorprotocol/launcher"
)

func TestWorkerConfigRequiresEmptyPrivateRootAndBoundedConcurrency(t *testing.T) {
	root := t.TempDir()
	if err := os.Chmod(root, 0o700); err != nil {
		t.Fatal(err)
	}
	config := Config{
		SocketPath:          "/run/orquesta/firecracker.sock",
		ExpectedAssetDigest: strings.Repeat("a", 64),
		WorkRoot:            root, GitCommand: "/usr/bin/git",
		TestSleep: 100 * time.Millisecond, ConcurrentRuns: 16,
		Limits: validWorkerLimits(16),
	}
	if _, err := validateWorkerConfig(config); err != nil {
		t.Fatal(err)
	}
	config.ConcurrentRuns = 17
	if _, err := validateWorkerConfig(config); ErrorCode(err) != codeConfigInvalid {
		t.Fatalf("concurrency error=%v", err)
	}
	config.ConcurrentRuns = 16
	if err := os.WriteFile(filepath.Join(root, "foreign"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := validateWorkerConfig(config); ErrorCode(err) != codeWorkRootUnsafe {
		t.Fatalf("non-empty root error=%v", err)
	}
}

func TestPrepareCausalFixtureUsesPublicGitLocalContracts(t *testing.T) {
	root := t.TempDir()
	if err := os.Chmod(root, 0o700); err != nil {
		t.Fatal(err)
	}
	session := filepath.Join(root, sessionDirectory)
	if err := os.Mkdir(session, 0o700); err != nil {
		t.Fatal(err)
	}
	repository := filepath.Join(session, repositoryDir)
	if err := initializeRepository(context.Background(), "/usr/bin/git", session, repository); err != nil {
		t.Fatal(err)
	}
	refs, err := newCausalReferences(0)
	requireNoError(t, err)
	adapter, err := gitLocalFixtureAdapter(session, repository, refs.repository)
	requireNoError(t, err)
	defer func() {
		if err := adapter.Close(); err != nil {
			t.Error(err)
		}
	}()
	seenSubjects := make(map[string]struct{}, 3)
	seenChanges := make(map[string]struct{}, 3)
	for index := 1; index <= 3; index++ {
		indexedRefs, refsErr := newCausalReferences(index)
		requireNoError(t, refsErr)
		fixture, fixtureErr := prepareCausalFixture(
			context.Background(), adapter, indexedRefs, 100*time.Millisecond, index,
		)
		requireNoError(t, fixtureErr)
		run, runErr := buildAttestationRun(fixture, firecrackerclient.PolicyIdentity{
			Ref: "policy:firecracker:test", Digest: digestString("policy"),
		}, index)
		requireNoError(t, runErr)
		if err := ports.ValidateTestAttestationRequest(run.Request); err != nil {
			t.Fatal(err)
		}
		stream, streamErr := adapter.OpenSnapshotStream(
			context.Background(),
			run.Snapshot,
		)
		requireNoError(t, streamErr)
		identity, validateErr := firecrackerguest.ValidateSnapshot(
			context.Background(),
			stream,
			run.Request.SubjectDigest,
		)
		closeErr := stream.Close()
		requireNoError(t, validateErr)
		requireNoError(t, closeErr)
		if identity.SubjectDigest != run.Request.SubjectDigest ||
			identity.HeadOID != run.Request.Subject.HeadOID ||
			identity.TreeOID != run.Request.Subject.TreeOID {
			t.Fatalf("snapshot identity=%+v subject=%+v", identity, run.Request.Subject)
		}
		if run.Request.Subject.WorkspaceBindingDigest != fixture.binding.Digest() ||
			run.Request.Subject.ChangeSetDigest != fixture.change.Digest() {
			t.Fatal("causal digests differ")
		}
		if _, duplicate := seenSubjects[run.Request.SubjectDigest]; duplicate {
			t.Fatal("duplicate subject")
		}
		if _, duplicate := seenChanges[fixture.change.Digest()]; duplicate {
			t.Fatal("duplicate change")
		}
		seenSubjects[run.Request.SubjectDigest] = struct{}{}
		seenChanges[fixture.change.Digest()] = struct{}{}
	}
}

func TestExecuteConcurrentAttestationsUsesOneBarrierAndValidatesAllResults(t *testing.T) {
	const count = 16
	runs := make([]ports.TestAttestationRun, count)
	results := make(map[string]ports.TestAttestationResult, count)
	for index := range runs {
		runs[index] = validWorkerRun(t, index+1)
		results[runs[index].Request.SubjectDigest] = validWorkerResult(t, runs[index])
	}
	allEntered := make(chan struct{})
	var closeEntered sync.Once
	var calls atomic.Int32
	var active atomic.Int32
	var maximum atomic.Int32
	attest := func(
		ctx context.Context,
		run ports.TestAttestationRun,
	) (ports.TestAttestationResult, error) {
		calls.Add(1)
		current := active.Add(1)
		defer active.Add(-1)
		for {
			observed := maximum.Load()
			if current <= observed || maximum.CompareAndSwap(observed, current) {
				break
			}
		}
		if current == count {
			closeEntered.Do(func() { close(allEntered) })
		}
		select {
		case <-allEntered:
			return results[run.Request.SubjectDigest], nil
		case <-ctx.Done():
			return ports.TestAttestationResult{}, ctx.Err()
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	summary, err := executeConcurrentAttestations(ctx, runs, attest)
	if err != nil {
		t.Fatal(err)
	}
	if calls.Load() != count || maximum.Load() != count ||
		summary.Attempted != count || summary.Passed != count ||
		summary.PolicyDigest != runs[0].Request.Subject.PolicyDigest ||
		summary.AttestorRef != results[runs[0].Request.SubjectDigest].AttestorRef ||
		len(summary.Attestations) != count {
		t.Fatalf("calls=%d maximum=%d summary=%+v", calls.Load(), maximum.Load(), summary)
	}
	seenSubjects := make(map[string]struct{}, count)
	seenReceipts := make(map[string]struct{}, count)
	for index, attestation := range summary.Attestations {
		if attestation.SubjectDigest != runs[index].Request.SubjectDigest ||
			attestation.WorkspaceBindingDigest != runs[index].Request.Subject.WorkspaceBindingDigest ||
			attestation.ChangeSetDigest != runs[index].Request.Subject.ChangeSetDigest {
			t.Fatalf("attestation[%d]=%+v", index, attestation)
		}
		if _, duplicate := seenSubjects[attestation.SubjectDigest]; duplicate {
			t.Fatal("duplicate subject")
		}
		if _, duplicate := seenReceipts[attestation.ReceiptRef]; duplicate {
			t.Fatal("duplicate receipt")
		}
		seenSubjects[attestation.SubjectDigest] = struct{}{}
		seenReceipts[attestation.ReceiptRef] = struct{}{}
	}
}

func TestExecuteConcurrentAttestationsFailsClosedOnInvalidResult(t *testing.T) {
	run := validWorkerRun(t, 1)
	result := validWorkerResult(t, run)
	result.SubjectDigest = digestString("tampered")
	_, err := executeConcurrentAttestations(
		context.Background(),
		[]ports.TestAttestationRun{run},
		func(context.Context, ports.TestAttestationRun) (ports.TestAttestationResult, error) {
			return result, nil
		},
	)
	if ErrorCode(err) != codeAttestationFailed {
		t.Fatalf("error=%v", err)
	}
}

func TestExecuteConcurrentAttestationsRejectsDuplicateSubjectsAndReceipts(t *testing.T) {
	run := validWorkerRun(t, 1)
	result := validWorkerResult(t, run)
	_, err := executeConcurrentAttestations(
		context.Background(),
		[]ports.TestAttestationRun{run, run},
		func(context.Context, ports.TestAttestationRun) (ports.TestAttestationResult, error) {
			return result, nil
		},
	)
	if ErrorCode(err) != codeCausalityInvalid {
		t.Fatalf("error=%v", err)
	}
}

func TestRecordingLauncherDerivesCanonicalRunIDsConcurrently(t *testing.T) {
	const count = 16
	delegate := &recordingDelegate{}
	client := newRecordingLauncherClient(delegate)
	errorsCh := make(chan error, count)
	var wait sync.WaitGroup
	for index := 0; index < count; index++ {
		wait.Add(1)
		go func(index int) {
			defer wait.Done()
			var nonce [32]byte
			nonce[len(nonce)-1] = byte(index + 1)
			_, err := client.Launch(
				context.Background(),
				launcherprotocol.LaunchRequest{
					Nonce:         hex.EncodeToString(nonce[:]),
					SubjectDigest: digestString("subject:" + strconv.Itoa(index)),
				},
				nil,
				0,
			)
			errorsCh <- err
		}(index)
	}
	wait.Wait()
	close(errorsCh)
	for err := range errorsCh {
		if err != nil {
			t.Fatal(err)
		}
	}
	launches, err := client.launches()
	if err != nil {
		t.Fatal(err)
	}
	if len(launches) != count || delegate.calls.Load() != count {
		t.Fatalf("launches=%d delegate_calls=%d", len(launches), delegate.calls.Load())
	}
	seen := make(map[string]struct{}, count)
	for _, launch := range launches {
		expected, err := runIDFromNonce(launch.nonce)
		if err != nil || launch.runID != expected {
			t.Fatalf("launch=%+v error=%v", launch, err)
		}
		if _, duplicate := seen[launch.runID]; duplicate {
			t.Fatal("duplicate derived run ID")
		}
		seen[launch.runID] = struct{}{}
	}
}

func TestRunIDFromNonceRejectsInvalidAndMatchesLauncherVector(t *testing.T) {
	if runID, err := runIDFromNonce(strings.Repeat("0", 64)); err != nil ||
		runID != "orq-"+strings.Repeat("a", 52) {
		t.Fatalf("zero vector run_id=%q error=%v", runID, err)
	}
	for _, nonce := range []string{
		"", strings.Repeat("0", 63), strings.Repeat("A", 64),
		strings.Repeat("g", 64),
	} {
		if _, err := runIDFromNonce(nonce); err == nil {
			t.Fatalf("invalid nonce accepted: %q", nonce)
		}
	}
	delegate := &recordingDelegate{}
	client := newRecordingLauncherClient(delegate)
	if _, err := client.Launch(
		context.Background(),
		launcherprotocol.LaunchRequest{
			Nonce: strings.Repeat("A", 64), SubjectDigest: digestString("subject"),
		},
		nil,
		0,
	); err == nil {
		t.Fatal("invalid nonce reached delegate")
	}
	launches, err := client.launches()
	if err != nil || len(launches) != 0 || delegate.calls.Load() != 0 {
		t.Fatalf(
			"invalid launch recorded: launches=%+v calls=%d error=%v",
			launches, delegate.calls.Load(), err,
		)
	}
}

func TestRecordingLauncherRejectsDuplicateNonceRunAndSubject(t *testing.T) {
	tests := map[string]struct {
		secondNonce   string
		secondSubject string
	}{
		"nonce_and_run": {
			secondNonce:   strings.Repeat("0", 64),
			secondSubject: digestString("subject:2"),
		},
		"subject": {
			secondNonce:   strings.Repeat("1", 64),
			secondSubject: digestString("subject:1"),
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			delegate := &recordingDelegate{}
			client := newRecordingLauncherClient(delegate)
			if _, err := client.Launch(
				context.Background(),
				launcherprotocol.LaunchRequest{
					Nonce:         strings.Repeat("0", 64),
					SubjectDigest: digestString("subject:1"),
				},
				nil,
				0,
			); err != nil {
				t.Fatal(err)
			}
			if _, err := client.Launch(
				context.Background(),
				launcherprotocol.LaunchRequest{
					Nonce:         test.secondNonce,
					SubjectDigest: test.secondSubject,
				},
				nil,
				0,
			); err == nil {
				t.Fatal("duplicate launch accepted")
			}
			if delegate.calls.Load() != 1 {
				t.Fatalf("duplicate reached delegate: calls=%d", delegate.calls.Load())
			}
		})
	}
}

func TestPhysicalTestSourceUsesConfiguredDurationWithoutRuntimeEnvironment(t *testing.T) {
	source := string(physicalTestSource(125 * time.Millisecond))
	if !strings.Contains(source, "125000000 * time.Nanosecond") ||
		strings.Contains(source, "Getenv") {
		t.Fatalf("source=%q", source)
	}
}

type recordingDelegate struct {
	calls atomic.Int32
}

func (*recordingDelegate) Identity() launcherprotocol.Identity {
	return launcherprotocol.Identity{Ref: "launcher:test", Digest: digestString("launcher")}
}

func (delegate *recordingDelegate) Launch(
	context.Context,
	launcherprotocol.LaunchRequest,
	*os.File,
	int64,
) (launcherprotocol.ClientResult, error) {
	delegate.calls.Add(1)
	return launcherprotocol.ClientResult{}, nil
}

func validWorkerLimits(concurrent int) firecrackerclient.Limits {
	return firecrackerclient.Limits{
		Timeout: 30 * time.Second, CleanupTimeout: 5 * time.Second,
		MaxOutputBytes: 4 << 20, MaxSubjectBytes: 32 << 20,
		MaxConcurrentRuns: concurrent, GuestMemoryMiB: 512,
		MemoryMaxBytes: 768 << 20, PIDsMax: 256, CPUQuotaMicros: 200_000,
	}
}

func gitLocalFixtureAdapter(
	session string,
	repository string,
	repositoryRef identity.RepositoryRef,
) (*gitlocal.Adapter, error) {
	return gitlocal.New(gitlocal.Config{
		Root: filepath.Join(session, gitLocalDir), GitCommand: "/usr/bin/git",
		Locator: repositoryLocator{
			ref: repositoryRef,
			binding: gitlocal.LocalRepositoryBinding{
				Path: repository, TargetRef: targetRef,
			},
		},
		Now:              func() time.Time { return logicalTime },
		MaxSnapshotBytes: 32 << 20, MaxSnapshotEntries: 128,
	})
}

func validWorkerRun(t *testing.T, index int) ports.TestAttestationRun {
	t.Helper()
	suffix := ":run-" + strconv.Itoa(index)
	goalRef, err := goal.NewGoalRef("goal:test")
	requireNoError(t, err)
	workItemRef, err := goal.NewWorkItemRef("work-item:test" + suffix)
	requireNoError(t, err)
	executionRef, err := goal.NewExecutionRef("execution:test" + suffix)
	requireNoError(t, err)
	workspaceRef, err := ports.NewExecutionWorkspaceRef("workspace:test" + suffix)
	requireNoError(t, err)
	changeRef, err := ports.NewChangeSetRef("change:test" + suffix)
	requireNoError(t, err)
	repositoryRef, err := identity.NewRepositoryRef("repository:test")
	requireNoError(t, err)
	requiredRef, err := goal.NewRequiredTestRef("required:test" + suffix)
	requireNoError(t, err)
	toolRef, err := goal.NewToolRef("tool:go")
	requireNoError(t, err)
	required, err := goal.NewRequiredTestSpec(goal.RequiredTestSpecInput{
		Ref: requiredRef, ToolRef: toolRef,
		Arguments:        []string{"test", "-count=1", "./..."},
		WorkingDirectory: ".",
	})
	requireNoError(t, err)
	writeSet := []string{"physical_e2e_" + strconv.Itoa(index) + "_test.go"}
	subject := ports.TestSubject{
		GoalRef: goalRef, WorkItemRef: workItemRef, ExecutionRef: executionRef,
		ExecutionAttempt: 1, PlanGeneration: 1, WorkItemGeneration: 1,
		AppSpecGeneration: 1, AppSpecHash: digestString("app-spec"),
		WorkspaceRef: workspaceRef, WorkspaceBindingDigest: digestString("workspace" + suffix),
		ChangeSetRef: changeRef, ChangeSetDigest: digestString("change" + suffix),
		RepositoryRef: repositoryRef, ObjectFormat: ports.GitObjectFormatSHA1,
		BaseOID: strings.Repeat("1", 40), ParentOID: strings.Repeat("2", 40),
		HeadOID: strings.Repeat("3", 40), TreeOID: strings.Repeat("4", 40),
		DiffDigest:          digestString("diff" + suffix),
		WriteSetDigest:      ports.WorkspaceWriteSetDigest(writeSet),
		RequiredTestsDigest: goal.RequiredTestsDigest([]goal.RequiredTestSpec{required}),
		PolicyRef:           "policy:firecracker:test", PolicyDigest: digestString("policy"),
	}
	request := ports.TestAttestationRequest{
		Subject: subject, SubjectDigest: ports.TestSubjectDigest(subject),
		RequiredTests:  []goal.RequiredTestSpec{required},
		IdempotencyKey: "attest:test" + suffix, RequestedAt: logicalTime.Add(-time.Minute),
	}
	return ports.TestAttestationRun{
		Request: request,
		Snapshot: ports.SnapshotVerificationRequest{
			Subject: subject, SubjectDigest: request.SubjectDigest,
			ChangedPaths: writeSet, WriteSet: writeSet,
		},
	}
}

func validWorkerResult(
	t *testing.T,
	run ports.TestAttestationRun,
) ports.TestAttestationResult {
	t.Helper()
	outcomes := []ports.RequiredTestOutcome{{
		RequiredTestRef: run.Request.RequiredTests[0].Ref(),
		ExitCode:        0, OutputDigest: digestString("test-output"),
	}}
	manifest, err := ports.BuildTestSubjectManifest(run.Request.Subject)
	requireNoError(t, err)
	report, err := ports.BuildTestAttestationReport(ports.TestAttestationReportInput{
		SubjectDigest: run.Request.SubjectDigest,
		Verdict:       ports.TestAttestationPassed, Tests: outcomes,
		AttestorRef:  "attestor:firecracker:test",
		PolicyRef:    run.Request.Subject.PolicyRef,
		PolicyDigest: run.Request.Subject.PolicyDigest,
	})
	requireNoError(t, err)
	receipt, err := ports.TestAttestationReceiptRef(report)
	requireNoError(t, err)
	return ports.TestAttestationResult{
		Subject: run.Request.Subject, SubjectDigest: run.Request.SubjectDigest,
		Verdict: ports.TestAttestationPassed, Manifest: manifest, Report: report,
		Tests: outcomes, AttestorRef: "attestor:firecracker:test",
		ReceiptRef: receipt, PolicyRef: run.Request.Subject.PolicyRef,
		PolicyDigest: run.Request.Subject.PolicyDigest,
		StartedAt:    logicalTime, FinishedAt: logicalTime,
	}
}

func requireNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
