//go:build linux

package bubblewrap

import (
	"context"
	"errors"
	"os"
	"sync"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

type Adapter struct {
	mu                sync.Mutex
	config            Config
	inputs            *pinnedInputs
	cgroups           cgroupController
	policy            PolicyIdentity
	source            [2]string
	slots             chan struct{}
	life              context.Context
	cancel            context.CancelFunc
	active            sync.WaitGroup
	closed            bool
	close             sync.Once
	closeErr          error
	snapshotFileLimit int64
}

type processIdentity struct{ euid, egid int }

func New(config Config) (*Adapter, error) {
	return newAdapter(config, processIdentity{euid: os.Geteuid(), egid: os.Getegid()})
}

func newAdapter(config Config, identity processIdentity) (*Adapter, error) {
	if identity.euid <= 0 || identity.egid <= 0 {
		return nil, &Error{Code: CodeIdentityUnsafe}
	}
	if !validConfig(config) {
		return nil, &Error{Code: CodeConfigInvalid}
	}
	ref, digest := config.SnapshotSource.SnapshotIdentity()
	if !validIdentity(ref, digest) {
		return nil, &Error{Code: CodeIdentityUnsafe}
	}
	inputs, err := openPinnedInputs(config, 0)
	if err != nil {
		return nil, err
	}
	envelope, err := currentResourceEnvelope(config.Limits)
	if err != nil {
		_ = inputs.Close()
		return nil, err
	}
	cgroups, err := openCgroupRoot(config.CgroupRoot)
	if err != nil {
		_ = inputs.Close()
		return nil, err
	}
	snapshotFileLimit, err := currentSnapshotFileLimit(config.Limits)
	if err != nil {
		_ = cgroups.Close()
		_ = inputs.Close()
		return nil, err
	}
	life, cancel := context.WithCancel(context.Background())
	adapter := &Adapter{
		config: config, inputs: inputs, cgroups: cgroups, source: [2]string{ref, digest}, life: life, cancel: cancel,
		slots: make(chan struct{}, int(config.Limits.MaxConcurrentRuns)), snapshotFileLimit: snapshotFileLimit,
	}
	adapter.policy = configuredPolicyIdentity(config, inputs.identity, cgroups.identity+":"+envelope, ref, digest, 65534, 65534)
	if err := adapter.preflight(); err != nil {
		if closeErr := adapter.Close(); closeErr != nil {
			return nil, &Error{Code: CodeCleanupFailed}
		}
		return nil, err
	}
	return adapter, nil
}

func (adapter *Adapter) preflight() error {
	ref, err := goal.NewRequiredTestRef("required-test:preflight")
	if err != nil {
		return &Error{Code: CodeConfigInvalid}
	}
	tool, err := goal.NewToolRef(goToolRef)
	if err != nil {
		return &Error{Code: CodeConfigInvalid}
	}
	spec, err := goal.NewRequiredTestSpec(goal.RequiredTestSpecInput{
		Ref: ref, ToolRef: tool, Arguments: []string{"version"}, WorkingDirectory: ".",
	})
	if err != nil {
		return &Error{Code: CodeConfigInvalid}
	}
	ctx, cancel := context.WithTimeout(context.Background(), adapter.config.Limits.Timeout)
	defer cancel()
	outcome, err := adapter.runTest(ctx, &sandboxSnapshot{}, spec)
	if err != nil {
		return err
	}
	if outcome.ExitCode != 0 {
		return &Error{Code: CodeToolchainUnsafe}
	}
	return nil
}

func (adapter *Adapter) PolicyIdentity() PolicyIdentity {
	adapter.mu.Lock()
	defer adapter.mu.Unlock()
	if adapter.closed {
		return PolicyIdentity{}
	}
	return adapter.policy
}

func (adapter *Adapter) Attest(ctx context.Context, run ports.TestAttestationRun) (result ports.TestAttestationResult, resultErr error) {
	if err := adapter.begin(); err != nil {
		return ports.TestAttestationResult{}, err
	}
	defer adapter.active.Done()
	ctx, cancel := context.WithTimeout(ctx, adapter.config.Limits.Timeout)
	stop := context.AfterFunc(adapter.life, cancel)
	defer func() { stop(); cancel() }()
	if err := validateRun(run, adapter.policy); err != nil {
		return ports.TestAttestationResult{}, err
	}
	select {
	case adapter.slots <- struct{}{}:
		defer func() { <-adapter.slots }()
	case <-ctx.Done():
		return ports.TestAttestationResult{}, contextFailure(ctx)
	}
	started := adapter.config.Now().UTC()
	stream, err := adapter.config.SnapshotSource.OpenSnapshotStream(ctx, run.Snapshot)
	if err != nil || stream == nil {
		if stream != nil {
			if stream.Close() != nil {
				return result, &Error{Code: CodeCleanupFailed}
			}
		}
		return result, snapshotInvalid()
	}
	snapshot, readErr := readSnapshotStreamLimited(stream, run.Request, adapter.config.Limits.MaxSubjectBytes, adapter.snapshotFileLimit)
	if snapshot != nil {
		defer func() { resultErr = errors.Join(resultErr, snapshot.Close()) }()
	}
	if stream.Close() != nil {
		return result, &Error{Code: CodeCleanupFailed}
	}
	if readErr != nil {
		return result, readErr
	}
	outcomes := make([]ports.RequiredTestOutcome, 0, len(run.Request.RequiredTests))
	verdict := ports.TestAttestationPassed
	for _, spec := range run.Request.RequiredTests {
		outcome, err := adapter.runTest(ctx, snapshot, spec)
		if err != nil {
			return result, err
		}
		outcomes = append(outcomes, outcome)
		if outcome.ExitCode != 0 {
			verdict = ports.TestAttestationFailed
		}
	}
	return adapter.result(run.Request, started, adapter.config.Now().UTC(), verdict, outcomes)
}

func (adapter *Adapter) result(request ports.TestAttestationRequest, started, finished time.Time, verdict ports.TestAttestationVerdict, tests []ports.RequiredTestOutcome) (ports.TestAttestationResult, error) {
	if started.Before(request.RequestedAt) || finished.Before(started) {
		return ports.TestAttestationResult{}, &Error{Code: CodeClockInvalid}
	}
	manifest, err := ports.BuildTestSubjectManifest(request.Subject)
	if err != nil {
		return ports.TestAttestationResult{}, err
	}
	report, err := ports.BuildTestAttestationReport(ports.TestAttestationReportInput{SubjectDigest: request.SubjectDigest,
		Verdict: verdict, Tests: tests, AttestorRef: AttestorRef, PolicyRef: adapter.policy.Ref, PolicyDigest: adapter.policy.Digest})
	if err != nil {
		return ports.TestAttestationResult{}, err
	}
	receiptRef, err := ports.TestAttestationReceiptRef(report)
	if err != nil {
		return ports.TestAttestationResult{}, err
	}
	return ports.TestAttestationResult{Subject: request.Subject, SubjectDigest: request.SubjectDigest, Verdict: verdict,
		Manifest: manifest, Report: report, Tests: tests, AttestorRef: AttestorRef,
		ReceiptRef: receiptRef, PolicyRef: adapter.policy.Ref,
		PolicyDigest: adapter.policy.Digest, StartedAt: started, FinishedAt: finished}, nil
}

func (adapter *Adapter) begin() error {
	adapter.mu.Lock()
	defer adapter.mu.Unlock()
	if adapter.closed {
		return &Error{Code: CodeUnavailable}
	}
	ref, digest := adapter.config.SnapshotSource.SnapshotIdentity()
	if ref != adapter.source[0] || digest != adapter.source[1] {
		return &Error{Code: CodeIdentityUnsafe}
	}
	adapter.active.Add(1)
	return nil
}

func (adapter *Adapter) Close() error {
	adapter.close.Do(func() {
		adapter.mu.Lock()
		adapter.closed = true
		adapter.cancel()
		adapter.mu.Unlock()
		adapter.active.Wait()
		adapter.closeErr = errors.Join(adapter.cgroups.Close(), adapter.inputs.Close())
	})
	return adapter.closeErr
}
