package firecrackerattestor

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"time"
)

type Supervisor struct {
	config    Config
	unit      UnitPort
	workload  WorkloadPort
	monitor   MonitorPort
	publisher PublisherPort
	clock     Clock
}

const (
	failureCodePreflight          = "firecracker_attestor_e2e.preflight_failed"
	failureCodeStart              = "firecracker_attestor_e2e.start_failed"
	failureCodeReadiness          = "firecracker_attestor_e2e.readiness_failed"
	failureCodeReadinessTimeout   = "firecracker_attestor_e2e.readiness_timeout"
	failureCodeUnitIdentity       = "firecracker_attestor_e2e.unit_identity_invalid"
	failureCodeInitialQuiescence  = "firecracker_attestor_e2e.initial_quiescence_failed"
	failureCodePhaseOne           = "firecracker_attestor_e2e.phase_one_failed"
	failureCodePhaseSixteen       = "firecracker_attestor_e2e.phase_sixteen_failed"
	failureCodeQuiescence         = "firecracker_attestor_e2e.quiescence_failed"
	failureCodeStop               = "firecracker_attestor_e2e.stop_failed"
	failureCodePostStop           = "firecracker_attestor_e2e.post_stop_quiescence_failed"
	failureCodeFailureCleanup     = "firecracker_attestor_e2e.failure_cleanup_failed"
	failureCodePublish            = "firecracker_attestor_e2e.publish_failed"
	failureCodeSupervisorFallback = "firecracker_attestor_e2e.failed"
)

type supervisorFailure struct {
	stage string
	code  string
	cause error
}

func (failure *supervisorFailure) Error() string { return failure.code }
func (failure *supervisorFailure) Unwrap() error { return failure.cause }

func newSupervisorFailure(stage, code string, cause error) error {
	return &supervisorFailure{stage: stage, code: code, cause: cause}
}

// FailureDiagnostic returns only fixed machine values. Wrapped OS errors and
// paths remain available to internal callers through errors.Unwrap, but never
// cross the command's public diagnostic boundary.
func FailureDiagnostic(err error) (stage, code string) {
	var failure *supervisorFailure
	if errors.As(err, &failure) {
		return failure.stage, failure.code
	}
	return "supervisor", failureCodeSupervisorFallback
}

func New(
	config Config,
	unit UnitPort,
	workload WorkloadPort,
	monitor MonitorPort,
	publisher PublisherPort,
	clock Clock,
) (*Supervisor, error) {
	if err := validateSupervisorConfig(config); err != nil ||
		unit == nil || workload == nil || monitor == nil || publisher == nil || clock == nil {
		return nil, ErrInvalid
	}
	return &Supervisor{
		config: config, unit: unit, workload: workload,
		monitor: monitor, publisher: publisher, clock: clock,
	}, nil
}

func (supervisor *Supervisor) Run(ctx context.Context) (publication Publication, resultErr error) {
	if supervisor == nil || ctx == nil {
		return Publication{}, ErrInvalid
	}
	started := supervisor.clock.Now().UTC()
	preflightContext, cancelPreflight := context.WithTimeout(
		ctx, supervisor.config.CleanupTimeout,
	)
	candidate, err := supervisor.unit.Preflight(preflightContext, supervisor.config)
	cancelPreflight()
	if err != nil {
		return Publication{}, newSupervisorFailure("preflight", failureCodePreflight, err)
	}
	stopped := false
	defer func() {
		if !stopped {
			stopContext, cancel := context.WithTimeout(
				context.Background(), supervisor.config.CleanupTimeout,
			)
			stopErr := supervisor.unit.Stop(
				stopContext, supervisor.config.Candidate.UnitName,
			)
			cancel()
			cleanupContext, cancelCleanup := context.WithTimeout(
				context.Background(), supervisor.config.CleanupTimeout,
			)
			cleanup, cleanupErr := supervisor.monitor.WaitQuiescent(
				cleanupContext,
				UnitIdentity{
					UnitName:     supervisor.config.Candidate.UnitName,
					FragmentPath: supervisor.config.Candidate.UnitPath,
					Loaded:       true,
				},
			)
			cancelCleanup()
			if cleanupErr != nil || !cleanAfterStop(cleanup) {
				cleanupErr = newSupervisorFailure(
					"failure_cleanup",
					failureCodeFailureCleanup,
					errors.Join(cleanupErr, ErrInvalid),
				)
			}
			if stopErr != nil {
				stopErr = newSupervisorFailure(
					"failure_cleanup", failureCodeFailureCleanup, stopErr,
				)
			}
			// Residual privileged state outranks the primary gate failure in
			// the public diagnostic while retaining both causes internally.
			resultErr = errors.Join(stopErr, cleanupErr, resultErr)
		}
	}()
	startContext, cancelStart := context.WithTimeout(
		ctx, supervisor.config.CleanupTimeout,
	)
	unit, err := supervisor.unit.Start(
		startContext, supervisor.config.Candidate.UnitName,
	)
	cancelStart()
	if err != nil {
		return Publication{}, newSupervisorFailure("start", failureCodeStart, err)
	}
	if err := validateActiveUnit(
		unit,
		supervisor.config.Candidate.UnitName,
		supervisor.config.Candidate.UnitPath,
	); err != nil {
		return Publication{}, newSupervisorFailure(
			"start_identity", failureCodeUnitIdentity, err,
		)
	}
	readinessContext, cancelReadiness := context.WithTimeout(
		ctx, supervisor.config.CleanupTimeout,
	)
	unit, err = supervisor.unit.WaitReady(
		readinessContext, supervisor.config, unit,
	)
	cancelReadiness()
	if err != nil {
		code := failureCodeReadiness
		if errors.Is(err, errReadinessTimeout) {
			code = failureCodeReadinessTimeout
		}
		return Publication{}, newSupervisorFailure("readiness", code, err)
	}
	if err := validateActiveUnit(
		unit,
		supervisor.config.Candidate.UnitName,
		supervisor.config.Candidate.UnitPath,
	); err != nil {
		return Publication{}, newSupervisorFailure(
			"readiness", failureCodeUnitIdentity, err,
		)
	}
	initialContext, cancelInitial := context.WithTimeout(ctx, supervisor.config.CleanupTimeout)
	initial, err := supervisor.monitor.WaitQuiescent(initialContext, unit)
	cancelInitial()
	if err != nil || !cleanBeforeStop(initial) {
		return Publication{}, newSupervisorFailure(
			"initial_quiescence",
			failureCodeInitialQuiescence,
			errors.Join(err, ErrInvalid),
		)
	}

	phaseOne, oneOutcomes, err := supervisor.runPhase(
		ctx, unit, 1, supervisor.config.PolicyDigest,
	)
	if err != nil {
		return Publication{}, newSupervisorFailure("phase_one", failureCodePhaseOne, err)
	}
	if err := validatePhase(phaseOne, 1); err != nil {
		return Publication{}, newSupervisorFailure("phase_one", failureCodePhaseOne, err)
	}
	policyDigest, err := validatePhaseOutcomes(
		phaseOne, oneOutcomes, 1, supervisor.config.PolicyDigest,
	)
	if err != nil {
		return Publication{}, newSupervisorFailure("phase_one", failureCodePhaseOne, err)
	}
	phaseSixteen, sixteenOutcomes, err := supervisor.runPhase(
		ctx, unit, ExpectedConcurrentRuns, policyDigest,
	)
	if err != nil {
		return Publication{}, newSupervisorFailure(
			"phase_sixteen", failureCodePhaseSixteen, err,
		)
	}
	if err := validatePhase(phaseSixteen, ExpectedConcurrentRuns); err != nil {
		return Publication{}, newSupervisorFailure(
			"phase_sixteen", failureCodePhaseSixteen, err,
		)
	}
	if _, err := validatePhaseOutcomes(
		phaseSixteen,
		sixteenOutcomes,
		ExpectedConcurrentRuns,
		policyDigest,
	); err != nil {
		return Publication{}, newSupervisorFailure(
			"phase_sixteen", failureCodePhaseSixteen, err,
		)
	}
	allOutcomes := append(append(
		make([]AttestationOutcome, 0, 1+ExpectedConcurrentRuns),
		oneOutcomes...,
	), sixteenOutcomes...)
	if err := validateOutcomes(allOutcomes, policyDigest); err != nil {
		return Publication{}, newSupervisorFailure(
			"attestations", failureCodePhaseSixteen, err,
		)
	}
	cleanupContext, cancelCleanup := context.WithTimeout(ctx, supervisor.config.CleanupTimeout)
	cleanup, err := supervisor.monitor.WaitQuiescent(cleanupContext, unit)
	cancelCleanup()
	if err != nil || !cleanBeforeStop(cleanup) {
		return Publication{}, newSupervisorFailure(
			"quiescence", failureCodeQuiescence, errors.Join(err, ErrInvalid),
		)
	}
	stopContext, cancelStop := context.WithTimeout(
		ctx, supervisor.config.CleanupTimeout,
	)
	err = supervisor.unit.Stop(stopContext, supervisor.config.Candidate.UnitName)
	cancelStop()
	if err != nil {
		return Publication{}, newSupervisorFailure("stop", failureCodeStop, err)
	}
	postStopContext, cancelPostStop := context.WithTimeout(
		ctx, supervisor.config.CleanupTimeout,
	)
	cleanup, err = supervisor.monitor.WaitQuiescent(postStopContext, UnitIdentity{
		UnitName:     supervisor.config.Candidate.UnitName,
		FragmentPath: unit.FragmentPath, Loaded: true,
	})
	cancelPostStop()
	if err != nil || !cleanAfterStop(cleanup) {
		return Publication{}, newSupervisorFailure(
			"post_stop_quiescence",
			failureCodePostStop,
			errors.Join(err, ErrInvalid),
		)
	}
	stopped = true

	evidence := Evidence{
		Schema: EvidenceSchema, Suite: Suite, Status: "passed",
		StartedAt:  started.Format(time.RFC3339Nano),
		FinishedAt: supervisor.clock.Now().UTC().Format(time.RFC3339Nano),
		Candidate:  candidate, PolicyDigest: policyDigest,
		Unit: unit, PhaseOne: phaseOne, PhaseSixteen: phaseSixteen,
		Attestations: allOutcomes, Cleanup: cleanup,
	}
	sort.Slice(evidence.Attestations, func(i, j int) bool {
		if evidence.Attestations[i].Ref == evidence.Attestations[j].Ref {
			return evidence.Attestations[i].RunID < evidence.Attestations[j].RunID
		}
		return evidence.Attestations[i].Ref < evidence.Attestations[j].Ref
	})
	receipt := receiptFor(evidence)
	publication, err = supervisor.publisher.Publish(evidence, receipt)
	if err != nil {
		return Publication{}, newSupervisorFailure("publish", failureCodePublish, err)
	}
	return publication, nil
}

func (supervisor *Supervisor) runPhase(
	ctx context.Context,
	unit UnitIdentity,
	count uint32,
	policyDigest string,
) (PhaseEvidence, []AttestationOutcome, error) {
	phaseContext, cancel := context.WithTimeout(ctx, supervisor.config.PhaseTimeout)
	defer cancel()
	run, err := supervisor.workload.StartPhase(phaseContext, WorkloadRequest{
		Count: count, GuestMemoryMiB: ExpectedGuestMemoryMiB,
		MemoryMaxBytes: ExpectedMemoryMaxBytes, PIDsMax: ExpectedPIDsMax,
		CPUQuotaMicros:  ExpectedCPUQuotaMicros,
		CPUPeriodMicros: ExpectedCPUPeriodMicros,
		PolicyDigest:    policyDigest,
	})
	if err != nil || run == nil {
		return PhaseEvidence{}, nil, errors.Join(err, ErrInvalid)
	}
	phase, captureErr := supervisor.monitor.CapturePhase(phaseContext, unit, run, count)
	if captureErr != nil {
		cancel()
	}
	outcomes, resultErr := run.Result()
	closeErr := run.Close()
	return phase, outcomes, errors.Join(captureErr, resultErr, closeErr)
}

func receiptFor(evidence Evidence) Receipt {
	phase := evidence.PhaseSixteen
	return Receipt{
		Schema: ReceiptSchema, Status: "passed",
		ConfigSHA256:         evidence.Candidate.ConfigSHA256,
		UnitSHA256:           evidence.Candidate.UnitSHA256,
		PrimitivesUnitSHA256: evidence.Candidate.PrimitivesUnitSHA256,
		LauncherSHA256:       evidence.Candidate.LauncherSHA256,
		AssetDigest:          evidence.Candidate.AssetDigest,
		PolicyDigest:         evidence.PolicyDigest, E2ESuite: Suite,
		MaxConcurrentRuns:    ExpectedConcurrentRuns,
		PhysicalMicroVMCount: uint32(len(phase.RunIDs)),
		ConcurrentHighWater:  phase.HighWaterRuns,
		AllAttestationsValid: true,
		ZeroResidualRuns: evidence.Cleanup.ResidualRuns == 0 &&
			evidence.Cleanup.ResidualCgroups == 0 &&
			evidence.Cleanup.ResidualProcesses == 0,
		NetworkAbsent:     phase.NetworkAbsent,
		MemorySwapMaxZero: phase.MemorySwapMaxZero,
		APIAbsent:         phase.APIAbsent, VsockAbsent: phase.VsockAbsent,
		SerialAbsent: phase.SerialAbsent,
	}
}

func validateSupervisorConfig(config Config) error {
	candidate := config.Candidate
	if len(config.WorkloadPayload) == 0 ||
		len(config.WorkloadPayload) > 64<<10 ||
		!json.Valid(config.WorkloadPayload) {
		return ErrInvalid
	}
	var workloadBinding struct {
		SocketPath          string `json:"socket_path"`
		ExpectedAssetDigest string `json:"expected_asset_digest"`
	}
	bindingErr := json.Unmarshal(config.WorkloadPayload, &workloadBinding)
	if config.PolicyDigest != "" && !validDigest(config.PolicyDigest) ||
		config.PhaseTimeout <= 0 || config.CleanupTimeout <= 0 ||
		config.StableFor <= 0 || config.PollInterval <= 0 ||
		config.PollInterval > config.StableFor ||
		config.ChildUID == 0 || config.ChildGID == 0 ||
		!validDigest(candidate.UnitSHA256) ||
		!validDigest(candidate.PrimitivesUnitSHA256) ||
		!validDigest(candidate.LauncherSHA256) ||
		!validDigest(candidate.ConfigSHA256) ||
		!validDigest(candidate.SupervisorSHA256) ||
		!validDigest(candidate.AssetDigest) ||
		bindingErr != nil ||
		workloadBinding.SocketPath != candidate.LauncherSocketPath ||
		workloadBinding.ExpectedAssetDigest != candidate.AssetDigest {
		return ErrInvalid
	}
	for _, value := range []string{
		candidate.UnitName, candidate.UnitPath, candidate.PrimitivesUnitPath,
		candidate.LauncherPath, candidate.LauncherSocketPath,
		candidate.ConfigPath, candidate.SupervisorPath, candidate.RuntimeRoot,
		candidate.CgroupRoot, candidate.ParentCgroup, candidate.NetNSPath,
		config.EvidencePath, config.ReceiptPath,
	} {
		if value == "" || strings.ContainsRune(value, 0) {
			return ErrInvalid
		}
	}
	return nil
}

func validateActiveUnit(unit UnitIdentity, expectedName, expectedPath string) error {
	if unit.UnitName != expectedName || unit.MainPID <= 1 ||
		!validInvocationID(unit.InvocationID) || !unit.Active ||
		!unit.Loaded || unit.NeedDaemonReload ||
		unit.FragmentPath != expectedPath {
		return errors.New("firecracker_attestor_e2e.unit_identity_invalid")
	}
	return nil
}

func validatePhase(phase PhaseEvidence, expected uint32) error {
	if phase.RequestedRuns != expected || phase.HighWaterRuns != expected ||
		uint32(len(phase.RunIDs)) != expected ||
		uint32(len(phase.FirecrackerPIDs)) != expected ||
		phase.Samples == 0 || !phase.LimitsExact ||
		!phase.MemorySwapMaxZero || !phase.NetworkAbsent ||
		!phase.APIAbsent || !phase.VsockAbsent || !phase.SerialAbsent ||
		!phase.UnitIdentityStable ||
		!uniqueStrings(phase.RunIDs) || !uniquePositiveInts(phase.FirecrackerPIDs) {
		return errors.New("firecracker_attestor_e2e.phase_invalid")
	}
	for _, runID := range phase.RunIDs {
		if !validCorrelatedRunID(runID) {
			return errors.New("firecracker_attestor_e2e.phase_invalid")
		}
	}
	return nil
}

func validateOutcomes(outcomes []AttestationOutcome, policyDigest string) error {
	if len(outcomes) != int(1+ExpectedConcurrentRuns) {
		return errors.New("firecracker_attestor_e2e.attestations_invalid")
	}
	runIDs := make([]string, 0, len(outcomes))
	for _, outcome := range outcomes {
		if outcome.Ref == "" || outcome.Ref != outcome.ReceiptRef ||
			!validCorrelatedRunID(outcome.RunID) ||
			!validDigest(outcome.SubjectDigest) ||
			outcome.ReceiptRef == "" ||
			!outcome.Valid || outcome.PolicyDigest != policyDigest {
			return errors.New("firecracker_attestor_e2e.attestations_invalid")
		}
		runIDs = append(runIDs, outcome.RunID)
	}
	if !uniqueStrings(runIDs) {
		return errors.New("firecracker_attestor_e2e.attestations_invalid")
	}
	return nil
}

func validatePhaseOutcomes(
	phase PhaseEvidence,
	outcomes []AttestationOutcome,
	expectedCount uint32,
	expectedPolicy string,
) (string, error) {
	if uint32(len(outcomes)) != expectedCount {
		return "", errors.New("firecracker_attestor_e2e.attestations_invalid")
	}
	policy := expectedPolicy
	refs := make([]string, 0, len(outcomes))
	runIDs := make([]string, 0, len(outcomes))
	subjects := make([]string, 0, len(outcomes))
	receipts := make([]string, 0, len(outcomes))
	for _, outcome := range outcomes {
		if policy == "" {
			policy = outcome.PolicyDigest
		}
		if outcome.Ref == "" || outcome.Ref != outcome.ReceiptRef ||
			!validCorrelatedRunID(outcome.RunID) ||
			!validDigest(outcome.SubjectDigest) ||
			outcome.ReceiptRef == "" ||
			!outcome.Valid ||
			!validDigest(outcome.PolicyDigest) || outcome.PolicyDigest != policy {
			return "", errors.New("firecracker_attestor_e2e.attestations_invalid")
		}
		refs = append(refs, outcome.Ref)
		runIDs = append(runIDs, outcome.RunID)
		subjects = append(subjects, outcome.SubjectDigest)
		receipts = append(receipts, outcome.ReceiptRef)
	}
	if !uniqueStrings(refs) || !uniqueStrings(runIDs) ||
		!uniqueStrings(subjects) || !uniqueStrings(receipts) ||
		!sameStringSet(phase.RunIDs, runIDs) {
		return "", errors.New("firecracker_attestor_e2e.attestations_invalid")
	}
	return policy, nil
}

func validCorrelatedRunID(value string) bool {
	const prefix = "orq-"
	const encodedNonceLength = 52
	if len(value) != len(prefix)+encodedNonceLength ||
		!strings.HasPrefix(value, prefix) {
		return false
	}
	for _, character := range value[len(prefix):] {
		if character >= 'a' && character <= 'z' ||
			character >= '2' && character <= '7' {
			continue
		}
		return false
	}
	return true
}

func sameStringSet(left, right []string) bool {
	if len(left) != len(right) || !uniqueStrings(left) || !uniqueStrings(right) {
		return false
	}
	expected := make(map[string]struct{}, len(left))
	for _, value := range left {
		expected[value] = struct{}{}
	}
	for _, value := range right {
		if _, ok := expected[value]; !ok {
			return false
		}
	}
	return true
}

func cleanBeforeStop(cleanup CleanupEvidence) bool {
	return cleanup.StableSamples >= 2 && cleanup.ResidualRuns == 0 &&
		cleanup.ResidualCgroups == 0 && cleanup.ResidualProcesses == 0 &&
		!cleanup.UnitStopped
}

func cleanAfterStop(cleanup CleanupEvidence) bool {
	return cleanup.StableSamples >= 2 && cleanup.ResidualRuns == 0 &&
		cleanup.ResidualCgroups == 0 && cleanup.ResidualProcesses == 0 &&
		cleanup.UnitStopped && cleanup.SocketAbsent
}

func validDigest(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, character := range value {
		if character < '0' || character > '9' {
			if character < 'a' || character > 'f' {
				return false
			}
		}
	}
	return true
}

func validInvocationID(value string) bool {
	if len(value) != 32 {
		return false
	}
	for _, character := range value {
		if character < '0' || character > '9' {
			if character < 'a' || character > 'f' {
				return false
			}
		}
	}
	return true
}

func uniqueStrings(values []string) bool {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if value == "" {
			return false
		}
		if _, exists := seen[value]; exists {
			return false
		}
		seen[value] = struct{}{}
	}
	return true
}

func uniquePositiveInts(values []int) bool {
	seen := make(map[int]struct{}, len(values))
	for _, value := range values {
		if value <= 1 {
			return false
		}
		if _, exists := seen[value]; exists {
			return false
		}
		seen[value] = struct{}{}
	}
	return true
}
