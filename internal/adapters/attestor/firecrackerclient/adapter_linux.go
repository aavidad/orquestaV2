//go:build linux

package firecrackerclient

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"sync"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/ports"
	launcherprotocol "orquesta/internal/testattestorprotocol/launcher"
	"orquesta/internal/testattestorprotocol/rawdrive"
)

type Adapter struct {
	mu               sync.Mutex
	config           Config
	policy           PolicyIdentity
	snapshotIdentity launcherprotocol.Identity
	launcherIdentity launcherprotocol.Identity
	slots            chan struct{}
	life             context.Context
	cancel           context.CancelFunc
	active           int
	drained          chan struct{}
	closed           bool
	close            sync.Once
	closeErr         error
}

func New(config Config) (*Adapter, error) {
	return newAdapter(config, os.Geteuid(), os.Getegid())
}

func newAdapter(config Config, effectiveUID, effectiveGID int) (*Adapter, error) {
	// Rejecting root identifiers prevents an accidental root deployment. It is
	// defense in depth, not proof of an empty capability set; service
	// composition remains responsible for capability and unit confinement.
	if effectiveUID <= 0 || effectiveGID <= 0 {
		return nil, contractError(codeIsolationIdentity)
	}
	if err := validateConfig(config); err != nil {
		return nil, err
	}
	snapshot := snapshotIdentity(config.SnapshotSource)
	launcher := config.Launcher.Identity()
	if !validDependencyIdentity(snapshot) ||
		!validDependencyIdentity(launcher) {
		return nil, contractError(codeConfigInvalid)
	}
	life, cancel := context.WithCancel(context.Background())
	drained := make(chan struct{})
	close(drained)
	adapter := &Adapter{
		config:           config,
		policy:           configuredPolicyIdentity(config, snapshot, launcher, effectiveUID, effectiveGID),
		snapshotIdentity: snapshot,
		launcherIdentity: launcher,
		slots:            make(chan struct{}, config.Limits.MaxConcurrentRuns),
		life:             life,
		cancel:           cancel,
		drained:          drained,
	}
	return adapter, nil
}

func (adapter *Adapter) PolicyIdentity() PolicyIdentity {
	adapter.mu.Lock()
	defer adapter.mu.Unlock()
	if adapter.closed {
		return PolicyIdentity{}
	}
	return adapter.policy
}

func (adapter *Adapter) Attest(
	ctx context.Context,
	run ports.TestAttestationRun,
) (result ports.TestAttestationResult, resultErr error) {
	if err := adapter.begin(); err != nil {
		return result, err
	}
	defer adapter.end()
	ctx, cancel := context.WithTimeout(ctx, adapter.config.Limits.Timeout)
	stop := context.AfterFunc(adapter.life, cancel)
	defer func() {
		stop()
		cancel()
	}()
	if err := validateRun(run, adapter.policy); err != nil {
		return result, err
	}
	select {
	case adapter.slots <- struct{}{}:
		defer func() { <-adapter.slots }()
	case <-ctx.Done():
		return result, translateFailure(ctx.Err())
	}
	started := adapter.config.Now().UTC()
	if started.Before(run.Request.RequestedAt) {
		return result, contractError(codeClockInvalid)
	}
	nonce, err := randomNonce()
	if err != nil {
		return result, err
	}
	outputBytes, err := outputDriveBytes(run)
	if err != nil {
		return result, err
	}
	stream, err := adapter.config.SnapshotSource.OpenSnapshotStream(ctx, run.Snapshot)
	if err != nil || stream == nil {
		if stream != nil {
			if closeErr := stream.Close(); closeErr != nil {
				return result, contractError(codeCleanupFailed)
			}
		}
		if ctx.Err() != nil {
			return result, translateFailure(ctx.Err())
		}
		if errors.Is(err, context.DeadlineExceeded) ||
			errors.Is(err, context.Canceled) {
			return result, translateFailure(err)
		}
		return result, contractError(codeSnapshotInvalid)
	}
	snapshotStream := &onceReadCloser{source: stream}
	stopSnapshotClose := context.AfterFunc(ctx, func() {
		_ = snapshotStream.Close()
	})
	input, inputDigest, inputBytes, buildErr := buildInputDrive(
		ctx,
		toRawInput(run, nonce, adapter.config.Limits.MaxOutputBytes),
		snapshotStream,
		adapter.config.Limits.MaxSubjectBytes,
	)
	stopSnapshotClose()
	closeStreamErr := snapshotStream.Close()
	if buildErr != nil {
		if closeStreamErr != nil {
			return result, errors.Join(contractError(codeCleanupFailed), buildErr)
		}
		return result, buildErr
	}
	if closeStreamErr != nil {
		_ = closeWithContract(input)
		return result, contractError(codeCleanupFailed)
	}
	deadline, ok := ctx.Deadline()
	if !ok {
		_ = closeWithContract(input)
		return result, contractError(codeExecutionTimeout)
	}
	remaining := time.Until(deadline)
	if remaining <= 0 {
		_ = closeWithContract(input)
		return result, contractError(codeExecutionTimeout)
	}
	if err := adapter.revalidateDependencyIdentities(); err != nil {
		return result, errors.Join(err, closeWithContract(input))
	}
	launchResult, launchErr := adapter.config.Launcher.Launch(
		ctx,
		launcherprotocol.LaunchRequest{
			Nonce:                  nonce,
			InputDigest:            inputDigest,
			SubjectDigest:          run.Request.SubjectDigest,
			PolicyDigest:           run.Request.Subject.PolicyDigest,
			Timeout:                remaining,
			GuestMemoryMiB:         adapter.config.Limits.GuestMemoryMiB,
			MemoryMaxBytes:         adapter.config.Limits.MemoryMaxBytes,
			PIDsMax:                adapter.config.Limits.PIDsMax,
			CPUQuotaMicros:         adapter.config.Limits.CPUQuotaMicros,
			CPUPeriodMicros:        launcherprotocol.CPUPeriodMicrosV0,
			OutputDriveBytes:       outputBytes,
			MaxCapturedOutputBytes: adapter.config.Limits.MaxOutputBytes,
		},
		input,
		inputBytes,
	)
	identityErr := adapter.revalidateDependencyIdentities()
	inputCloseErr := closeWithContract(input)
	if identityErr != nil {
		var outputCloseErr error
		if launchResult.Output != nil {
			outputCloseErr = closeWithContract(launchResult.Output)
		}
		return result, errors.Join(identityErr, inputCloseErr, outputCloseErr)
	}
	if launchErr != nil {
		var outputCloseErr error
		if launchResult.Output != nil {
			outputCloseErr = closeWithContract(launchResult.Output)
		}
		translated := translateFailure(launchErr)
		return result, errors.Join(translated, inputCloseErr, outputCloseErr)
	}
	if inputCloseErr != nil {
		var outputCloseErr error
		if launchResult.Output != nil {
			outputCloseErr = closeWithContract(launchResult.Output)
		}
		return result, errors.Join(inputCloseErr, outputCloseErr)
	}
	output, outputErr := adapter.consumeOutput(
		ctx,
		run,
		nonce,
		outputBytes,
		launchResult,
	)
	outputCloseErr := closeWithContract(launchResult.Output)
	if outputErr != nil {
		if outputCloseErr != nil {
			return result, errors.Join(outputErr, outputCloseErr)
		}
		return result, outputErr
	}
	if outputCloseErr != nil {
		return result, outputCloseErr
	}
	if output.ErrorCode != "" {
		code, known := rawCauseTranslations[output.ErrorCode]
		if !known || !stringsHasGuestPrefix(output.ErrorCode) {
			return result, contractError(codeOutputInvalid)
		}
		return result, contractError(code)
	}
	finished := adapter.config.Now().UTC()
	return buildResult(run.Request, adapter.policy, started, finished, output)
}

func (adapter *Adapter) consumeOutput(
	ctx context.Context,
	run ports.TestAttestationRun,
	nonce string,
	driveBytes uint64,
	result launcherprotocol.ClientResult,
) (rawdrive.Output, error) {
	response := result.Response
	if response.Nonce != nonce || response.Code != "ok" ||
		!validDigest(response.OutputDigest) ||
		response.AssetDigest != adapter.config.ExpectedAssetDigest ||
		result.Output == nil {
		if response.AssetDigest != "" &&
			response.AssetDigest != adapter.config.ExpectedAssetDigest {
			return rawdrive.Output{}, contractError(codeAssetMismatch)
		}
		return rawdrive.Output{}, contractError(codeOutputInvalid)
	}
	output, err := readOutputDrive(ctx, result.Output, driveBytes, response.OutputDigest)
	if err != nil {
		return rawdrive.Output{}, err
	}
	if output.SubjectDigest != run.Request.SubjectDigest ||
		output.RunNonce != nonce {
		return rawdrive.Output{}, contractError(codeOutputInvalid)
	}
	return output, nil
}

func buildResult(
	request ports.TestAttestationRequest,
	policy PolicyIdentity,
	started, finished time.Time,
	output rawdrive.Output,
) (ports.TestAttestationResult, error) {
	if finished.Before(started) {
		return ports.TestAttestationResult{}, contractError(codeClockInvalid)
	}
	verdict := ports.TestAttestationFailed
	if output.Verdict == rawdrive.VerdictPassed {
		verdict = ports.TestAttestationPassed
	}
	tests := make([]ports.RequiredTestOutcome, 0, len(output.Outcomes))
	for _, outcome := range output.Outcomes {
		ref, err := requiredTestRef(request, outcome.RequiredTestRef)
		if err != nil {
			return ports.TestAttestationResult{}, err
		}
		tests = append(tests, ports.RequiredTestOutcome{
			RequiredTestRef: ref,
			ExitCode:        int(outcome.ExitCode),
			OutputDigest:    outcome.OutputDigest,
		})
	}
	if err := ports.ValidateRequiredTestOutcomes(request.RequiredTests, verdict, tests); err != nil {
		return ports.TestAttestationResult{}, err
	}
	manifest, err := ports.BuildTestSubjectManifest(request.Subject)
	if err != nil {
		return ports.TestAttestationResult{}, err
	}
	report, err := ports.BuildTestAttestationReport(ports.TestAttestationReportInput{
		SubjectDigest: request.SubjectDigest,
		Verdict:       verdict,
		Tests:         tests,
		AttestorRef:   AttestorRef,
		PolicyRef:     policy.Ref,
		PolicyDigest:  policy.Digest,
	})
	if err != nil {
		return ports.TestAttestationResult{}, err
	}
	receiptRef, err := ports.TestAttestationReceiptRef(report)
	if err != nil {
		return ports.TestAttestationResult{}, err
	}
	result := ports.TestAttestationResult{
		Subject:       request.Subject,
		SubjectDigest: request.SubjectDigest,
		Verdict:       verdict,
		Manifest:      manifest,
		Report:        report,
		Tests:         tests,
		AttestorRef:   AttestorRef,
		ReceiptRef:    receiptRef,
		PolicyRef:     policy.Ref,
		PolicyDigest:  policy.Digest,
		StartedAt:     started,
		FinishedAt:    finished,
	}
	if err := ports.ValidateTestAttestationResult(request, result); err != nil {
		return ports.TestAttestationResult{}, err
	}
	return result, nil
}

func requiredTestRef(
	request ports.TestAttestationRequest,
	value string,
) (goal.RequiredTestRef, error) {
	for _, spec := range request.RequiredTests {
		if spec.Ref().String() == value {
			return spec.Ref(), nil
		}
	}
	return goal.RequiredTestRef{}, contractError("test_attestor.outcomes_invalid")
}

func randomNonce() (string, error) {
	var value [32]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", contractError(codeUnavailable)
	}
	return hex.EncodeToString(value[:]), nil
}

func stringsHasGuestPrefix(value string) bool {
	const prefix = "test_attestor.firecracker.guest_"
	return len(value) > len(prefix) && value[:len(prefix)] == prefix
}

func (adapter *Adapter) begin() error {
	adapter.mu.Lock()
	if adapter.closed {
		adapter.mu.Unlock()
		return contractError(codeUnavailable)
	}
	if adapter.active == 0 {
		adapter.drained = make(chan struct{})
	}
	adapter.active++
	adapter.mu.Unlock()
	if err := adapter.revalidateDependencyIdentities(); err != nil {
		adapter.end()
		return err
	}
	return nil
}

func (adapter *Adapter) end() {
	adapter.mu.Lock()
	defer adapter.mu.Unlock()
	adapter.active--
	if adapter.active == 0 {
		close(adapter.drained)
	}
}

func (adapter *Adapter) revalidateDependencyIdentities() error {
	if snapshotIdentity(adapter.config.SnapshotSource) != adapter.snapshotIdentity ||
		adapter.config.Launcher.Identity() != adapter.launcherIdentity {
		return contractError(codePolicyMismatch)
	}
	return nil
}

// Close cancels active work and waits only CleanupTimeout. Dependencies are
// contractually cooperative; timeout reports cleanup_failed without creating a
// waiter goroutine or claiming that external work was reaped.
func (adapter *Adapter) Close() error {
	adapter.close.Do(func() {
		adapter.mu.Lock()
		adapter.closed = true
		adapter.cancel()
		drained := adapter.drained
		timeout := adapter.config.Limits.CleanupTimeout
		adapter.mu.Unlock()
		timer := time.NewTimer(timeout)
		defer timer.Stop()
		select {
		case <-drained:
		case <-timer.C:
			adapter.closeErr = contractError(codeCleanupFailed)
		}
	})
	return adapter.closeErr
}

type onceReadCloser struct {
	once   sync.Once
	source io.ReadCloser
	err    error
}

func (stream *onceReadCloser) Read(value []byte) (int, error) {
	return stream.source.Read(value)
}

func (stream *onceReadCloser) Close() error {
	stream.once.Do(func() { stream.err = stream.source.Close() })
	return stream.err
}
