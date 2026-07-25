//go:build linux

// Package firecrackerattestorworkload runs the unprivileged half of the physical
// Firecracker concurrency smoke. Privileged launch remains behind the launcher
// Unix socket and the production TestAttestor contracts.
package firecrackerattestorworkload

import (
	"context"
	"crypto/sha256"
	"encoding/base32"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"orquesta/internal/adapters/attestor/firecrackerclient"
	"orquesta/internal/adapters/attestor/firecrackerlauncher"
	"orquesta/internal/adapters/workspace/gitlocal"
	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/identity"
	"orquesta/internal/ports"
	launcherprotocol "orquesta/internal/testattestorprotocol/launcher"
)

const (
	codeUnavailable       = "firecracker_physical_e2e.unavailable"
	codeConfigInvalid     = "firecracker_physical_e2e.config_invalid"
	codeWorkRootUnsafe    = "firecracker_physical_e2e.work_root_unsafe"
	codeRepositoryFailed  = "firecracker_physical_e2e.repository_failed"
	codeCausalityInvalid  = "firecracker_physical_e2e.causality_invalid"
	codeAttestationFailed = "firecracker_physical_e2e.attestation_failed"
	codeCleanupFailed     = "firecracker_physical_e2e.cleanup_failed"

	sessionDirectory = "physical-e2e-session"
	repositoryDir    = "repository"
	gitLocalDir      = "gitlocal"
	targetRef        = "refs/heads/main"

	maxConcurrentAttestations = 256
	maxTestSleep              = 30 * time.Second
)

var logicalTime = time.Date(2026, time.July, 25, 12, 0, 0, 0, time.UTC)

// Config contains only explicit, non-secret inputs. WorkRoot must already
// exist as an empty private directory owned by the current unprivileged user.
type Config struct {
	SocketPath          string
	ExpectedAssetDigest string
	WorkRoot            string
	GitCommand          string
	TestSleep           time.Duration
	Limits              firecrackerclient.Limits
	ConcurrentRuns      int
}

// Summary contains no filesystem paths, launcher diagnostics, nonces,
// timestamps, or captured test output. RunID exposes only the launcher's
// canonical physical identifier derived from a fresh nonce.
type Summary struct {
	Attempted    int                  `json:"attempted"`
	Passed       int                  `json:"passed"`
	PolicyDigest string               `json:"policy_digest"`
	AttestorRef  string               `json:"attestor_ref"`
	Attestations []AttestationSummary `json:"attestations"`
}

type AttestationSummary struct {
	SubjectDigest          string `json:"subject_digest"`
	WorkspaceBindingDigest string `json:"workspace_binding_digest"`
	ChangeSetDigest        string `json:"change_set_digest"`
	ReceiptRef             string `json:"receipt_ref"`
	RunID                  string `json:"run_id"`
}

type Error struct {
	Code  string
	cause error
}

func (err *Error) Error() string {
	if err == nil {
		return ""
	}
	return err.Code
}

func (err *Error) Unwrap() error {
	if err == nil {
		return nil
	}
	return err.cause
}

func ErrorCode(err error) string {
	var workerError *Error
	if errors.As(err, &workerError) {
		return workerError.Code
	}
	return ""
}

// Run constructs ConcurrentRuns distinct immutable Git-backed test subjects
// and submits them after every worker has reached the start barrier.
func Run(ctx context.Context, config Config) (summary Summary, resultErr error) {
	if ctx == nil {
		return Summary{}, failure(codeUnavailable, nil)
	}
	if ctx.Err() != nil {
		return Summary{}, failure(codeUnavailable, context.Cause(ctx))
	}
	rootInfo, err := validateWorkerConfig(config)
	if err != nil {
		return Summary{}, err
	}
	sessionPath := filepath.Join(config.WorkRoot, sessionDirectory)
	if err := os.Mkdir(sessionPath, 0o700); err != nil {
		return Summary{}, failure(codeWorkRootUnsafe, err)
	}
	sessionInfo, err := os.Lstat(sessionPath)
	if err != nil || !safeOwnedPrivateDirectory(sessionInfo) {
		return Summary{}, failure(codeWorkRootUnsafe, err)
	}

	var gitAdapter *gitlocal.Adapter
	var attestor *firecrackerclient.Adapter
	defer func() {
		var cleanupErr error
		if attestor != nil {
			cleanupErr = errors.Join(cleanupErr, attestor.Close())
		}
		if gitAdapter != nil {
			cleanupErr = errors.Join(cleanupErr, gitAdapter.Close())
		}
		cleanupErr = errors.Join(
			cleanupErr,
			cleanupOwnedSession(config.WorkRoot, rootInfo, sessionPath, sessionInfo),
		)
		if cleanupErr != nil {
			resultErr = failure(codeCleanupFailed, errors.Join(resultErr, cleanupErr))
			summary = Summary{}
		}
	}()

	repositoryPath := filepath.Join(sessionPath, repositoryDir)
	if err := initializeRepository(ctx, config.GitCommand, sessionPath, repositoryPath); err != nil {
		return Summary{}, err
	}
	common, err := newCausalReferences(0)
	if err != nil {
		return Summary{}, failure(codeCausalityInvalid, err)
	}
	gitAdapter, err = gitlocal.New(gitlocal.Config{
		Root:       filepath.Join(sessionPath, gitLocalDir),
		GitCommand: config.GitCommand,
		Locator: repositoryLocator{
			ref: common.repository,
			binding: gitlocal.LocalRepositoryBinding{
				Path: repositoryPath, TargetRef: targetRef,
			},
		},
		Now:                func() time.Time { return logicalTime },
		MaxSnapshotBytes:   config.Limits.MaxSubjectBytes,
		MaxSnapshotEntries: 128,
	})
	if err != nil {
		return Summary{}, failure(codeRepositoryFailed, err)
	}
	launcher, err := firecrackerlauncher.NewClient(config.SocketPath)
	if err != nil {
		return Summary{}, failure(codeConfigInvalid, err)
	}
	recordingLauncher := newRecordingLauncherClient(launcher)
	attestor, err = firecrackerclient.New(firecrackerclient.Config{
		Launcher:            recordingLauncher,
		SnapshotSource:      gitAdapter,
		Now:                 func() time.Time { return logicalTime },
		ExpectedAssetDigest: config.ExpectedAssetDigest,
		Limits:              config.Limits,
	})
	if err != nil {
		return Summary{}, failure(codeConfigInvalid, err)
	}
	runs := make([]ports.TestAttestationRun, config.ConcurrentRuns)
	for index := range runs {
		refs, refsErr := newCausalReferences(index + 1)
		if refsErr != nil {
			return Summary{}, failure(codeCausalityInvalid, refsErr)
		}
		fixture, fixtureErr := prepareCausalFixture(
			ctx, gitAdapter, refs, config.TestSleep, index+1,
		)
		if fixtureErr != nil {
			return Summary{}, fixtureErr
		}
		runs[index], err = buildAttestationRun(fixture, attestor.PolicyIdentity(), index+1)
		if err != nil {
			return Summary{}, err
		}
	}
	summary, err = executeConcurrentAttestations(ctx, runs, attestor.Attest)
	if err != nil {
		return Summary{}, err
	}
	launches, err := recordingLauncher.launches()
	if err != nil {
		return Summary{}, failure(codeCausalityInvalid, err)
	}
	if err := correlateSummaryLaunches(&summary, launches); err != nil {
		return Summary{}, failure(codeCausalityInvalid, err)
	}
	return summary, nil
}

type launchedRun struct {
	nonce         string
	runID         string
	subjectDigest string
}

// recordingLauncherClient is scoped to one worker invocation, which is one
// physical phase. It retains nonces only in-process and returns only the
// canonical run IDs needed by the privileged supervisor.
type recordingLauncherClient struct {
	delegate launcherprotocol.Client

	mu      sync.Mutex
	records []launchedRun
	err     error
}

func newRecordingLauncherClient(delegate launcherprotocol.Client) *recordingLauncherClient {
	return &recordingLauncherClient{delegate: delegate}
}

func (client *recordingLauncherClient) Identity() launcherprotocol.Identity {
	if client == nil || client.delegate == nil {
		return launcherprotocol.Identity{}
	}
	return client.delegate.Identity()
}

func (client *recordingLauncherClient) Launch(
	ctx context.Context,
	request launcherprotocol.LaunchRequest,
	input *os.File,
	maxInputBytes int64,
) (launcherprotocol.ClientResult, error) {
	if client == nil || client.delegate == nil {
		return launcherprotocol.ClientResult{}, failure(codeCausalityInvalid, nil)
	}
	runID, err := runIDFromNonce(request.Nonce)
	if err != nil || !validDigest(request.SubjectDigest) {
		return launcherprotocol.ClientResult{}, failure(codeCausalityInvalid, err)
	}
	client.mu.Lock()
	if client.err != nil {
		err := client.err
		client.mu.Unlock()
		return launcherprotocol.ClientResult{}, failure(codeCausalityInvalid, err)
	}
	for _, existing := range client.records {
		if existing.nonce == request.Nonce ||
			existing.runID == runID ||
			existing.subjectDigest == request.SubjectDigest {
			duplicateErr := errors.New("duplicate physical launch identity")
			client.err = duplicateErr
			client.mu.Unlock()
			return launcherprotocol.ClientResult{}, failure(codeCausalityInvalid, duplicateErr)
		}
	}
	client.records = append(client.records, launchedRun{
		nonce: request.Nonce, runID: runID, subjectDigest: request.SubjectDigest,
	})
	client.mu.Unlock()
	return client.delegate.Launch(ctx, request, input, maxInputBytes)
}

func (client *recordingLauncherClient) launches() ([]launchedRun, error) {
	if client == nil {
		return nil, errors.New("nil recording launcher")
	}
	client.mu.Lock()
	defer client.mu.Unlock()
	return append([]launchedRun(nil), client.records...), client.err
}

func runIDFromNonce(nonce string) (string, error) {
	if len(nonce) != sha256.Size*2 ||
		strings.Trim(nonce, "0123456789abcdef") != "" {
		return "", errors.New("invalid launcher nonce")
	}
	raw, err := hex.DecodeString(nonce)
	if err != nil || len(raw) != sha256.Size {
		return "", errors.New("invalid launcher nonce")
	}
	encoded := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(raw)
	return "orq-" + strings.ToLower(encoded), nil
}

func correlateSummaryLaunches(summary *Summary, launches []launchedRun) error {
	if summary == nil || len(launches) != summary.Attempted ||
		len(summary.Attestations) != summary.Attempted {
		return errors.New("physical launch count mismatch")
	}
	bySubject := make(map[string]launchedRun, len(launches))
	runIDs := make(map[string]struct{}, len(launches))
	for _, launch := range launches {
		expectedRunID, err := runIDFromNonce(launch.nonce)
		if err != nil || expectedRunID != launch.runID || !validDigest(launch.subjectDigest) {
			return errors.New("invalid physical launch identity")
		}
		if _, duplicate := bySubject[launch.subjectDigest]; duplicate {
			return errors.New("duplicate physical launch subject")
		}
		if _, duplicate := runIDs[launch.runID]; duplicate {
			return errors.New("duplicate physical launch run")
		}
		bySubject[launch.subjectDigest] = launch
		runIDs[launch.runID] = struct{}{}
	}
	for index := range summary.Attestations {
		attestation := &summary.Attestations[index]
		launch, ok := bySubject[attestation.SubjectDigest]
		if !ok || attestation.ReceiptRef == "" {
			return errors.New("attestation has no physical launch")
		}
		attestation.RunID = launch.runID
		delete(bySubject, attestation.SubjectDigest)
	}
	if len(bySubject) != 0 {
		return errors.New("physical launch has no attestation")
	}
	return nil
}

type causalReferences struct {
	principal  identity.PrincipalRef
	actor      goal.ActorRef
	project    goal.ProjectRef
	repository identity.RepositoryRef
	goal       goal.GoalRef
	workItem   goal.WorkItemRef
	execution  goal.ExecutionRef
	workspace  ports.ExecutionWorkspaceRef
	change     ports.ChangeSetRef
	required   goal.RequiredTestRef
	tool       goal.ToolRef
}

func newCausalReferences(index int) (causalReferences, error) {
	var refs causalReferences
	var err error
	suffix := ""
	if index > 0 {
		suffix = ":run-" + strconv.Itoa(index)
	}
	refs.principal, err = identity.NewPrincipalRef("principal:firecracker-physical-e2e")
	if err != nil {
		return causalReferences{}, err
	}
	refs.actor, err = goal.NewActorRef("actor:firecracker-physical-e2e")
	if err != nil {
		return causalReferences{}, err
	}
	refs.project, err = goal.NewProjectRef("project:firecracker-physical-e2e")
	if err != nil {
		return causalReferences{}, err
	}
	refs.repository, err = identity.NewRepositoryRef("repository:firecracker-physical-e2e")
	if err != nil {
		return causalReferences{}, err
	}
	refs.goal, err = goal.NewGoalRef("goal:firecracker-physical-e2e")
	if err != nil {
		return causalReferences{}, err
	}
	refs.workItem, err = goal.NewWorkItemRef("work-item:firecracker-physical-e2e" + suffix)
	if err != nil {
		return causalReferences{}, err
	}
	refs.execution, err = goal.NewExecutionRef("execution:firecracker-physical-e2e" + suffix)
	if err != nil {
		return causalReferences{}, err
	}
	refs.workspace, err = ports.NewExecutionWorkspaceRef("workspace:firecracker-physical-e2e" + suffix)
	if err != nil {
		return causalReferences{}, err
	}
	refs.change, err = ports.NewChangeSetRef("change-set:firecracker-physical-e2e" + suffix)
	if err != nil {
		return causalReferences{}, err
	}
	refs.required, err = goal.NewRequiredTestRef("required-test:firecracker-physical-e2e" + suffix)
	if err != nil {
		return causalReferences{}, err
	}
	refs.tool, err = goal.NewToolRef("tool:go")
	return refs, err
}

type repositoryLocator struct {
	ref     identity.RepositoryRef
	binding gitlocal.LocalRepositoryBinding
}

func (locator repositoryLocator) LocateLocalRepository(
	_ context.Context,
	ref identity.RepositoryRef,
) (gitlocal.LocalRepositoryBinding, error) {
	if ref != locator.ref {
		return gitlocal.LocalRepositoryBinding{}, failure(codeRepositoryFailed, nil)
	}
	return locator.binding, nil
}

type causalFixture struct {
	binding  application.WorkspaceBinding
	change   application.ChangeSet
	required goal.RequiredTestSpec
}

func prepareCausalFixture(
	ctx context.Context,
	adapter *gitlocal.Adapter,
	refs causalReferences,
	testSleep time.Duration,
	index int,
) (causalFixture, error) {
	suffix := strconv.Itoa(index)
	writeSet := []string{"physical_e2e_" + suffix + "_test.go"}
	appSpecHash := digestString("orquesta.firecracker-physical-e2e.app-spec.v1")
	prepare := ports.WorkspacePrepareRequest{
		WorkspaceRef: refs.workspace, PrincipalRef: refs.principal,
		ActorRef: refs.actor, ProjectRef: refs.project, RepositoryRef: refs.repository,
		GoalRef: refs.goal, WorkItemRef: refs.workItem, ExecutionRef: refs.execution,
		ExecutionAttempt: 1, PlanGeneration: 1, AppSpecGeneration: 1,
		AppSpecHash: appSpecHash, WriteSet: writeSet,
		WriteSetDigest: ports.WorkspaceWriteSetDigest(writeSet), TargetRef: targetRef,
		IntentRef:   "intent:firecracker-physical-e2e-workspace:" + suffix,
		AttemptRef:  "attempt:firecracker-physical-e2e-workspace:" + suffix,
		ActionFence: 1, IdempotencyKey: "prepare:firecracker-physical-e2e:" + suffix,
		PreparedAt: logicalTime.Add(-3 * time.Minute),
	}
	prepared, err := adapter.Prepare(ctx, prepare)
	if err != nil {
		return causalFixture{}, failure(codeRepositoryFailed, err)
	}
	workspacePath, err := adapter.ResolveExecutionWorkspace(ctx, refs.workspace)
	if err != nil {
		return causalFixture{}, failure(codeRepositoryFailed, err)
	}
	if err := writeExclusiveFile(
		filepath.Join(workspacePath, writeSet[0]),
		physicalTestSource(testSleep),
	); err != nil {
		return causalFixture{}, failure(codeRepositoryFailed, err)
	}
	commitRequest := ports.CommitRequest{
		ChangeSetRef: refs.change, WorkspaceRef: refs.workspace,
		PrincipalRef: refs.principal, ActorRef: refs.actor, ProjectRef: refs.project,
		RepositoryRef: refs.repository, GoalRef: refs.goal, WorkItemRef: refs.workItem,
		ExecutionRef: refs.execution, ExecutionAttempt: 1, PlanGeneration: 1,
		AppSpecGeneration: 1, AppSpecHash: appSpecHash,
		BaseOID: prepared.BaseOID, ObjectFormat: prepared.ObjectFormat,
		WriteSet: writeSet, WriteSetDigest: prepare.WriteSetDigest,
		IntentRef:   "intent:firecracker-physical-e2e-change:" + suffix,
		AttemptRef:  "attempt:firecracker-physical-e2e-change:" + suffix,
		ActionFence: 2, IdempotencyKey: "commit:firecracker-physical-e2e:" + suffix,
		CommittedAt: logicalTime.Add(-2 * time.Minute),
	}
	committed, err := adapter.Commit(ctx, commitRequest)
	if err != nil {
		return causalFixture{}, failure(codeRepositoryFailed, err)
	}
	binding := application.WorkspaceBinding{
		Ref: prepared.WorkspaceRef, PrincipalRef: refs.principal,
		ActorRef: refs.actor, ProjectRef: refs.project, RepositoryRef: refs.repository,
		GoalRef: refs.goal, WorkItemRef: refs.workItem, ExecutionRef: refs.execution,
		ExecutionAttempt: 1, PlanGeneration: 1, AppSpecGeneration: 1,
		SpecHash: appSpecHash, WriteSet: writeSet, WriteSetDigest: prepared.WriteSetDigest,
		TargetRef: prepared.TargetRef, BaseOID: prepared.BaseOID,
		ObjectFormat: prepared.ObjectFormat, AdapterRef: prepared.AdapterRef,
		EffectIntentRef: prepare.IntentRef, EffectAttemptRef: prepare.AttemptRef,
		EffectFence: prepare.ActionFence,
		ReceiptRef:  "effect-receipt:" + prepare.IntentRef,
		PreparedAt:  prepared.PreparedAt,
	}
	change := application.ChangeSet{
		Ref: committed.ChangeSetRef, WorkspaceRef: committed.WorkspaceRef,
		PrincipalRef: refs.principal, ActorRef: refs.actor, ProjectRef: refs.project,
		RepositoryRef: refs.repository, GoalRef: refs.goal, WorkItemRef: refs.workItem,
		ExecutionRef: refs.execution, ExecutionAttempt: 1, PlanGeneration: 1,
		AppSpecGeneration: 1, SpecHash: appSpecHash,
		BaseOID: committed.BaseOID, ParentOID: committed.ParentOID,
		HeadOID: committed.HeadOID, TreeOID: committed.TreeOID,
		ObjectFormat: committed.ObjectFormat, DiffDigest: committed.DiffDigest,
		ChangedPaths: committed.ChangedPaths, WriteSet: writeSet,
		WriteSetDigest: committed.WriteSetDigest, ParentChangeRef: committed.ParentChangeRef,
		EffectIntentRef: commitRequest.IntentRef, EffectAttemptRef: commitRequest.AttemptRef,
		EffectFence: commitRequest.ActionFence, IdempotencyKey: commitRequest.IdempotencyKey,
		AdapterRef:  committed.AdapterRef,
		ReceiptRef:  "effect-receipt:" + commitRequest.IntentRef,
		CommittedAt: committed.CommittedAt,
	}
	required, err := goal.NewRequiredTestSpec(goal.RequiredTestSpecInput{
		Ref: refs.required, ToolRef: refs.tool,
		Arguments:        []string{"test", "-count=1", "./..."},
		WorkingDirectory: ".",
	})
	if err != nil {
		return causalFixture{}, failure(codeCausalityInvalid, err)
	}
	if err := application.ValidateWorkspaceBinding(binding); err != nil {
		return causalFixture{}, failure(codeCausalityInvalid, err)
	}
	if err := application.ValidateChangeSet(change); err != nil {
		return causalFixture{}, failure(codeCausalityInvalid, err)
	}
	return causalFixture{binding: binding, change: change, required: required}, nil
}

func buildAttestationRun(
	fixture causalFixture,
	policy firecrackerclient.PolicyIdentity,
	index int,
) (ports.TestAttestationRun, error) {
	requiredTests := []goal.RequiredTestSpec{fixture.required}
	subject := ports.TestSubject{
		GoalRef: fixture.binding.GoalRef, WorkItemRef: fixture.binding.WorkItemRef,
		ExecutionRef:     fixture.binding.ExecutionRef,
		ExecutionAttempt: fixture.binding.ExecutionAttempt,
		PlanGeneration:   fixture.binding.PlanGeneration, WorkItemGeneration: 1,
		AppSpecGeneration: fixture.binding.AppSpecGeneration,
		AppSpecHash:       fixture.binding.SpecHash,
		WorkspaceRef:      fixture.binding.Ref, WorkspaceBindingDigest: fixture.binding.Digest(),
		ChangeSetRef: fixture.change.Ref, ChangeSetDigest: fixture.change.Digest(),
		RepositoryRef: fixture.change.RepositoryRef, ObjectFormat: fixture.change.ObjectFormat,
		BaseOID: fixture.change.BaseOID, ParentOID: fixture.change.ParentOID,
		HeadOID: fixture.change.HeadOID, TreeOID: fixture.change.TreeOID,
		DiffDigest: fixture.change.DiffDigest, WriteSetDigest: fixture.change.WriteSetDigest,
		RequiredTestsDigest: goal.RequiredTestsDigest(requiredTests),
		PolicyRef:           policy.Ref, PolicyDigest: policy.Digest,
	}
	request := ports.TestAttestationRequest{
		Subject: subject, SubjectDigest: ports.TestSubjectDigest(subject),
		RequiredTests:  requiredTests,
		IdempotencyKey: "attest:firecracker-physical-e2e:" + strconv.Itoa(index),
		RequestedAt:    logicalTime.Add(-time.Minute),
	}
	run := ports.TestAttestationRun{
		Request: request,
		Snapshot: ports.SnapshotVerificationRequest{
			Subject: subject, SubjectDigest: request.SubjectDigest,
			ChangedPaths: append([]string(nil), fixture.change.ChangedPaths...),
			WriteSet:     append([]string(nil), fixture.change.WriteSet...),
		},
	}
	if err := application.ValidateTestAttestationRun(run); err != nil {
		return ports.TestAttestationRun{}, failure(codeCausalityInvalid, err)
	}
	return run, nil
}

type attestFunction func(
	context.Context,
	ports.TestAttestationRun,
) (ports.TestAttestationResult, error)

func executeConcurrentAttestations(
	ctx context.Context,
	runs []ports.TestAttestationRun,
	attest attestFunction,
) (Summary, error) {
	if ctx == nil || len(runs) <= 0 || attest == nil {
		return Summary{}, failure(codeCausalityInvalid, nil)
	}
	inputSubjects := make(map[string]struct{}, len(runs))
	inputBindings := make(map[string]struct{}, len(runs))
	inputChanges := make(map[string]struct{}, len(runs))
	for _, run := range runs {
		if application.ValidateTestAttestationRun(run) != nil {
			return Summary{}, failure(codeCausalityInvalid, nil)
		}
		subject := run.Request.Subject
		if _, duplicate := inputSubjects[run.Request.SubjectDigest]; duplicate {
			return Summary{}, failure(codeCausalityInvalid, nil)
		}
		if _, duplicate := inputBindings[subject.WorkspaceBindingDigest]; duplicate {
			return Summary{}, failure(codeCausalityInvalid, nil)
		}
		if _, duplicate := inputChanges[subject.ChangeSetDigest]; duplicate {
			return Summary{}, failure(codeCausalityInvalid, nil)
		}
		inputSubjects[run.Request.SubjectDigest] = struct{}{}
		inputBindings[subject.WorkspaceBindingDigest] = struct{}{}
		inputChanges[subject.ChangeSetDigest] = struct{}{}
	}
	type outcome struct {
		result ports.TestAttestationResult
		err    error
	}
	outcomes := make([]outcome, len(runs))
	start := make(chan struct{})
	var ready sync.WaitGroup
	var finished sync.WaitGroup
	ready.Add(len(runs))
	finished.Add(len(runs))
	for index := range outcomes {
		go func(index int) {
			defer finished.Done()
			ready.Done()
			<-start
			outcomes[index].result, outcomes[index].err = attest(ctx, runs[index])
		}(index)
	}
	ready.Wait()
	close(start)
	finished.Wait()

	summary := Summary{
		Attempted: len(runs), Passed: len(runs),
		PolicyDigest: runs[0].Request.Subject.PolicyDigest,
		Attestations: make([]AttestationSummary, len(runs)),
	}
	receipts := make(map[string]struct{}, len(runs))
	for index, outcome := range outcomes {
		run := runs[index]
		if outcome.err != nil || outcome.result.Verdict != ports.TestAttestationPassed ||
			ports.ValidateTestAttestationResult(run.Request, outcome.result) != nil {
			return Summary{}, failure(codeAttestationFailed, outcome.err)
		}
		if index == 0 {
			summary.AttestorRef = outcome.result.AttestorRef
		} else if outcome.result.PolicyDigest != summary.PolicyDigest ||
			outcome.result.AttestorRef != summary.AttestorRef {
			return Summary{}, failure(codeAttestationFailed, nil)
		}
		if _, duplicate := receipts[outcome.result.ReceiptRef]; duplicate {
			return Summary{}, failure(codeAttestationFailed, nil)
		}
		receipts[outcome.result.ReceiptRef] = struct{}{}
		summary.Attestations[index] = AttestationSummary{
			SubjectDigest:          outcome.result.SubjectDigest,
			WorkspaceBindingDigest: run.Request.Subject.WorkspaceBindingDigest,
			ChangeSetDigest:        run.Request.Subject.ChangeSetDigest,
			ReceiptRef:             outcome.result.ReceiptRef,
		}
	}
	return summary, nil
}

func validateWorkerConfig(config Config) (os.FileInfo, error) {
	if !canonicalAbsolute(config.SocketPath) ||
		!canonicalAbsolute(config.WorkRoot) ||
		!canonicalAbsolute(config.GitCommand) ||
		!validDigest(config.ExpectedAssetDigest) ||
		config.TestSleep <= 0 || config.TestSleep > maxTestSleep ||
		config.Limits.Timeout <= config.TestSleep ||
		config.ConcurrentRuns <= 0 ||
		config.ConcurrentRuns > maxConcurrentAttestations ||
		config.ConcurrentRuns > config.Limits.MaxConcurrentRuns {
		return nil, failure(codeConfigInvalid, nil)
	}
	rootInfo, err := os.Lstat(config.WorkRoot)
	if err != nil || !safeOwnedPrivateDirectory(rootInfo) {
		return nil, failure(codeWorkRootUnsafe, err)
	}
	entries, err := os.ReadDir(config.WorkRoot)
	if err != nil || len(entries) != 0 {
		return nil, failure(codeWorkRootUnsafe, err)
	}
	gitInfo, err := os.Lstat(config.GitCommand)
	if err != nil || !safeRootExecutable(gitInfo) {
		return nil, failure(codeConfigInvalid, err)
	}
	return rootInfo, nil
}

func canonicalAbsolute(value string) bool {
	return value != "" && filepath.IsAbs(value) &&
		filepath.Clean(value) == value && !strings.ContainsRune(value, 0)
}

func validDigest(value string) bool {
	if len(value) != sha256.Size*2 || strings.ToLower(value) != value {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func safeOwnedPrivateDirectory(info os.FileInfo) bool {
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && info.IsDir() && info.Mode()&os.ModeSymlink == 0 &&
		info.Mode().Perm()&0o077 == 0 && int(stat.Uid) == os.Geteuid()
}

func safeRootExecutable(info os.FileInfo) bool {
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && info.Mode().IsRegular() && info.Mode()&os.ModeSymlink == 0 &&
		info.Mode().Perm()&0o111 != 0 && info.Mode().Perm()&0o022 == 0 &&
		stat.Uid == 0 && stat.Nlink > 0 && info.Size() > 0
}

func initializeRepository(
	ctx context.Context,
	gitCommand, sessionPath, repositoryPath string,
) error {
	if err := os.Mkdir(repositoryPath, 0o700); err != nil {
		return failure(codeRepositoryFailed, err)
	}
	if err := writeExclusiveFile(
		filepath.Join(repositoryPath, "go.mod"),
		[]byte("module example.invalid/orquesta-firecracker-physical-e2e\n\ngo 1.25.0\n"),
	); err != nil {
		return failure(codeRepositoryFailed, err)
	}
	for _, arguments := range [][]string{
		{"-C", repositoryPath, "init", "-b", "main"},
		{"-C", repositoryPath, "add", "--", "go.mod"},
		{"-C", repositoryPath, "commit", "-m", "base física Firecracker"},
	} {
		if err := runGit(ctx, gitCommand, sessionPath, arguments...); err != nil {
			return failure(codeRepositoryFailed, err)
		}
	}
	if err := os.Chmod(repositoryPath, 0o700); err != nil {
		return failure(codeRepositoryFailed, err)
	}
	if err := os.Chmod(filepath.Join(repositoryPath, ".git"), 0o700); err != nil {
		return failure(codeRepositoryFailed, err)
	}
	return nil
}

func runGit(ctx context.Context, commandPath, home string, arguments ...string) error {
	common := []string{
		"--no-replace-objects",
		"-c", "core.hooksPath=/dev/null",
		"-c", "core.attributesFile=/dev/null",
		"-c", "core.pager=cat",
		"-c", "credential.helper=",
		"-c", "core.fsmonitor=false",
		"-c", "diff.external=",
		"-c", "commit.gpgSign=false",
		"-c", "fetch.ifMissing=false",
		"-c", "user.name=Orquesta",
		"-c", "user.email=orquesta@local",
	}
	command := exec.CommandContext(ctx, commandPath, append(common, arguments...)...)
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	command.Cancel = func() error {
		if command.Process == nil {
			return os.ErrProcessDone
		}
		err := syscall.Kill(-command.Process.Pid, syscall.SIGKILL)
		if errors.Is(err, syscall.ESRCH) {
			return os.ErrProcessDone
		}
		return err
	}
	command.WaitDelay = 2 * time.Second
	stamp := logicalTime.Add(-4 * time.Minute).Format(time.RFC3339)
	command.Env = []string{
		"PATH=/usr/bin:/bin", "HOME=" + home, "LANG=C", "LC_ALL=C",
		"GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null",
		"GIT_CONFIG_SYSTEM=/dev/null", "GIT_TERMINAL_PROMPT=0",
		"GIT_PAGER=cat", "GIT_EDITOR=/bin/false", "GIT_SEQUENCE_EDITOR=/bin/false",
		"GIT_ASKPASS=/bin/false", "GIT_SSH_COMMAND=/bin/false",
		"GIT_EXTERNAL_DIFF=", "GIT_OPTIONAL_LOCKS=0",
		"GIT_AUTHOR_NAME=Orquesta", "GIT_AUTHOR_EMAIL=orquesta@local",
		"GIT_AUTHOR_DATE=" + stamp, "GIT_COMMITTER_NAME=Orquesta",
		"GIT_COMMITTER_EMAIL=orquesta@local", "GIT_COMMITTER_DATE=" + stamp,
	}
	command.Stdout, command.Stderr = io.Discard, io.Discard
	if err := command.Run(); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return err
	}
	return nil
}

func writeExclusiveFile(path string, content []byte) (resultErr error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, file.Close()) }()
	if _, err := file.Write(content); err != nil {
		return err
	}
	return file.Sync()
}

func physicalTestSource(sleep time.Duration) []byte {
	return []byte("package smoke\n\nimport (\n\t\"testing\"\n\t\"time\"\n)\n\n" +
		"func TestFirecrackerPhysicalE2E(t *testing.T) {\n\ttime.Sleep(" +
		strconv.FormatInt(int64(sleep), 10) + " * time.Nanosecond)\n}\n")
}

func cleanupOwnedSession(
	root string,
	rootInfo os.FileInfo,
	session string,
	sessionInfo os.FileInfo,
) error {
	currentRoot, err := os.Lstat(root)
	if err != nil || !os.SameFile(rootInfo, currentRoot) ||
		!safeOwnedPrivateDirectory(currentRoot) {
		return failure(codeCleanupFailed, err)
	}
	currentSession, err := os.Lstat(session)
	if err != nil || !os.SameFile(sessionInfo, currentSession) ||
		!safeOwnedPrivateDirectory(currentSession) ||
		filepath.Dir(session) != root {
		return failure(codeCleanupFailed, err)
	}
	if err := os.RemoveAll(session); err != nil {
		return failure(codeCleanupFailed, err)
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 0 {
		return failure(codeCleanupFailed, err)
	}
	return nil
}

func digestString(value string) string {
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])
}

func failure(code string, cause error) error {
	return &Error{Code: code, cause: cause}
}
