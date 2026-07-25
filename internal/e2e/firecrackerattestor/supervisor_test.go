package firecrackerattestor

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

const testDigest = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func TestSupervisorPublishesOnlyAfterExactPhysicalSixteenAndStop(t *testing.T) {
	unit := &fakeUnit{}
	workload := &fakeWorkload{policy: testDigest}
	monitor := &fakeMonitor{}
	publisher := &fakePublisher{}
	supervisor, err := New(
		validSupervisorConfig(), unit, workload, monitor, publisher,
		fakeClock{time.Unix(1_700_000_000, 0)},
	)
	if err != nil {
		t.Fatal(err)
	}
	publication, err := supervisor.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if publication.EvidenceSHA256 != testDigest ||
		unit.startCount != 1 || unit.stopCount != 1 ||
		workload.calls != 2 || !publisher.called {
		t.Fatalf(
			"unexpected result publication=%+v unit=%+v workload=%+v publisher=%+v",
			publication, unit, workload, publisher,
		)
	}
	if monitor.waitCount != 3 {
		t.Fatalf("expected initial, pre-stop and post-stop quiescence; got %d", monitor.waitCount)
	}
	if publisher.evidence.PhaseOne.HighWaterRuns != 1 ||
		publisher.evidence.PhaseSixteen.HighWaterRuns != ExpectedConcurrentRuns ||
		len(publisher.evidence.Attestations) != 17 ||
		!publisher.evidence.Cleanup.UnitStopped {
		t.Fatalf("incomplete evidence: %+v", publisher.evidence)
	}
	if publisher.receipt.PhysicalMicroVMCount != 16 ||
		publisher.receipt.ConcurrentHighWater != 16 ||
		!publisher.receipt.APIAbsent ||
		!publisher.receipt.VsockAbsent ||
		!publisher.receipt.SerialAbsent {
		t.Fatalf("incomplete receipt: %+v", publisher.receipt)
	}
}

func TestSupervisorRejectsWorkloadSocketAndAssetDriftBeforeStart(t *testing.T) {
	tests := map[string]func(*Config){
		"socket": func(config *Config) {
			config.WorkloadPayload = []byte(
				`{"socket_path":"/run/orquesta/foreign.sock","expected_asset_digest":"` +
					testDigest + `"}`,
			)
		},
		"asset": func(config *Config) {
			config.WorkloadPayload = []byte(
				`{"socket_path":"/run/orquesta/firecracker.sock","expected_asset_digest":"` +
					strings.Repeat("b", 64) + `"}`,
			)
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			config := validSupervisorConfig()
			mutate(&config)
			unit := &fakeUnit{}
			if _, err := New(
				config, unit, &fakeWorkload{}, &fakeMonitor{}, &fakePublisher{},
				fakeClock{time.Unix(1_700_000_000, 0)},
			); err == nil {
				t.Fatal("workload/candidate drift accepted")
			}
			if unit.startCount != 0 {
				t.Fatal("unit started before binding validation")
			}
		})
	}
}

func TestSupervisorStopsAndDoesNotPublishOnEveryGateFailure(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*fakeUnit, *fakeWorkload, *fakeMonitor)
	}{
		{
			name: "phase_peak_missing",
			mutate: func(_ *fakeUnit, _ *fakeWorkload, monitor *fakeMonitor) {
				monitor.badPeak = true
			},
		},
		{
			name: "unit_identity_changes",
			mutate: func(_ *fakeUnit, _ *fakeWorkload, monitor *fakeMonitor) {
				monitor.unitUnstable = true
			},
		},
		{
			name: "unit_fragment_path_drift",
			mutate: func(unit *fakeUnit, _ *fakeWorkload, _ *fakeMonitor) {
				unit.fragmentPath = "/etc/systemd/system/foreign.service"
			},
		},
		{
			name: "api_present",
			mutate: func(_ *fakeUnit, _ *fakeWorkload, monitor *fakeMonitor) {
				monitor.apiPresent = true
			},
		},
		{
			name: "invalid_attestation",
			mutate: func(_ *fakeUnit, workload *fakeWorkload, _ *fakeMonitor) {
				workload.invalid = true
			},
		},
		{
			name: "residual_process",
			mutate: func(_ *fakeUnit, _ *fakeWorkload, monitor *fakeMonitor) {
				monitor.residual = true
			},
		},
		{
			name: "stop_fails",
			mutate: func(unit *fakeUnit, _ *fakeWorkload, _ *fakeMonitor) {
				unit.stopErr = errors.New("stop failed")
			},
		},
		{
			name: "socket_remains_after_stop",
			mutate: func(_ *fakeUnit, _ *fakeWorkload, monitor *fakeMonitor) {
				monitor.socketResidual = true
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			unit := &fakeUnit{}
			workload := &fakeWorkload{policy: testDigest}
			monitor := &fakeMonitor{}
			publisher := &fakePublisher{}
			test.mutate(unit, workload, monitor)
			supervisor, err := New(
				validSupervisorConfig(), unit, workload, monitor, publisher,
				fakeClock{time.Unix(1_700_000_000, 0)},
			)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := supervisor.Run(context.Background()); err == nil {
				t.Fatal("unsafe run passed")
			}
			if unit.stopCount == 0 || publisher.called {
				t.Fatalf("cleanup/publication mismatch: unit=%+v publisher=%+v", unit, publisher)
			}
		})
	}
}

func TestSupervisorPhaseFailureStopsAndWaitsForPostStopQuiescence(t *testing.T) {
	unit := &fakeUnit{}
	workload := &fakeWorkload{policy: testDigest}
	monitor := &fakeMonitor{badPeak: true}
	publisher := &fakePublisher{}
	supervisor, err := New(
		validSupervisorConfig(), unit, workload, monitor, publisher,
		fakeClock{time.Unix(1_700_000_000, 0)},
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := supervisor.Run(context.Background()); err == nil {
		t.Fatal("invalid phase passed")
	}
	if unit.stopCount != 1 || monitor.waitCount != 2 || publisher.called {
		t.Fatalf(
			"failure cleanup incomplete: unit=%+v monitor=%+v publisher=%+v",
			unit, monitor, publisher,
		)
	}
}

func TestSupervisorBoundsPreflightWithCleanupTimeout(t *testing.T) {
	config := validSupervisorConfig()
	config.CleanupTimeout = 20 * time.Millisecond
	supervisor, err := New(
		config, blockingPreflightUnit{}, &fakeWorkload{policy: testDigest},
		&fakeMonitor{}, &fakePublisher{},
		fakeClock{time.Unix(1_700_000_000, 0)},
	)
	if err != nil {
		t.Fatal(err)
	}
	started := time.Now()
	if _, err := supervisor.Run(context.Background()); err == nil {
		t.Fatal("blocking preflight passed")
	}
	if elapsed := time.Since(started); elapsed > 250*time.Millisecond {
		t.Fatalf("preflight exceeded cleanup budget: %s", elapsed)
	}
}

func TestSupervisorWaitsForExplicitReadinessBeforeQuiescence(t *testing.T) {
	unit := &delayedReadyUnit{
		entered: make(chan struct{}),
		release: make(chan struct{}),
	}
	monitor := &fakeMonitor{}
	workload := &fakeWorkload{policy: testDigest}
	supervisor, err := New(
		validSupervisorConfig(), unit, workload, monitor, &fakePublisher{},
		fakeClock{time.Unix(1_700_000_000, 0)},
	)
	if err != nil {
		t.Fatal(err)
	}
	result := make(chan error, 1)
	go func() {
		_, runErr := supervisor.Run(context.Background())
		result <- runErr
	}()
	<-unit.entered
	if monitor.waitCount != 0 || workload.calls != 0 {
		t.Fatalf(
			"work advanced before readiness: waits=%d phases=%d",
			monitor.waitCount,
			workload.calls,
		)
	}
	close(unit.release)
	if err := <-result; err != nil {
		t.Fatal(err)
	}
}

func TestSupervisorBoundsReadinessAndStopsCandidate(t *testing.T) {
	config := validSupervisorConfig()
	config.CleanupTimeout = 20 * time.Millisecond
	unit := &neverReadyUnit{}
	monitor := &fakeMonitor{}
	publisher := &fakePublisher{}
	supervisor, err := New(
		config, unit, &fakeWorkload{policy: testDigest}, monitor, publisher,
		fakeClock{time.Unix(1_700_000_000, 0)},
	)
	if err != nil {
		t.Fatal(err)
	}
	started := time.Now()
	if _, err := supervisor.Run(context.Background()); err == nil {
		t.Fatal("never-ready candidate passed")
	} else if stage, code := FailureDiagnostic(err); stage != "readiness" ||
		code != failureCodeReadinessTimeout {
		t.Fatalf("diagnostic=%s/%s err=%v", stage, code, err)
	}
	if elapsed := time.Since(started); elapsed > 250*time.Millisecond {
		t.Fatalf("readiness exceeded cleanup budget: %s", elapsed)
	}
	if unit.stopCount != 1 || monitor.waitCount != 1 || publisher.called {
		t.Fatalf(
			"failure cleanup mismatch unit=%+v monitor=%+v publisher=%+v",
			unit,
			monitor,
			publisher,
		)
	}
}

func TestSupervisorStartFailureStillStopsExactCandidate(t *testing.T) {
	unit := &fakeUnit{startErr: errors.New("systemctl returned after partial start")}
	monitor := &fakeMonitor{}
	publisher := &fakePublisher{}
	supervisor, err := New(
		validSupervisorConfig(), unit, &fakeWorkload{policy: testDigest},
		monitor, publisher, fakeClock{time.Unix(1_700_000_000, 0)},
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := supervisor.Run(context.Background()); err == nil {
		t.Fatal("partial start failure passed")
	} else if stage, code := FailureDiagnostic(err); stage != "start" ||
		code != failureCodeStart {
		t.Fatalf("diagnostic=%s/%s err=%v", stage, code, err)
	}
	if unit.stopCount != 1 || monitor.waitCount != 1 || publisher.called {
		t.Fatalf(
			"partial start cleanup mismatch unit=%+v monitor=%+v publisher=%+v",
			unit,
			monitor,
			publisher,
		)
	}
}

func TestSupervisorReadinessDiagnosticDoesNotExposeCause(t *testing.T) {
	const sensitive = "open /srv/private/launcher.sock: permission denied"
	unit := &failedReadyUnit{cause: errors.New(sensitive)}
	supervisor, err := New(
		validSupervisorConfig(), unit, &fakeWorkload{policy: testDigest},
		&fakeMonitor{}, &fakePublisher{},
		fakeClock{time.Unix(1_700_000_000, 0)},
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := supervisor.Run(context.Background()); err == nil {
		t.Fatal("unsafe readiness passed")
	} else {
		stage, code := FailureDiagnostic(err)
		if stage != "readiness" || code != failureCodeReadiness {
			t.Fatalf("diagnostic=%s/%s", stage, code)
		}
		if strings.Contains(err.Error(), sensitive) ||
			strings.Contains(err.Error(), "/srv/") {
			t.Fatalf("public error leaks cause: %q", err.Error())
		}
	}
}

func TestSupervisorCleanupFailureOutranksPrimaryFailureDiagnostic(t *testing.T) {
	unit := &fakeUnit{stopErr: errors.New("stop /srv/private/unit: denied")}
	monitor := &fakeMonitor{badPeak: true}
	supervisor, err := New(
		validSupervisorConfig(), unit, &fakeWorkload{policy: testDigest},
		monitor, &fakePublisher{},
		fakeClock{time.Unix(1_700_000_000, 0)},
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := supervisor.Run(context.Background()); err == nil {
		t.Fatal("cleanup failure passed")
	} else if stage, code := FailureDiagnostic(err); stage != "failure_cleanup" ||
		code != failureCodeFailureCleanup {
		t.Fatalf("diagnostic=%s/%s err=%v", stage, code, err)
	}
}

func TestRunPhaseCancelsWorkloadImmediatelyWhenMonitorFails(t *testing.T) {
	config := validSupervisorConfig()
	config.PhaseTimeout = time.Second
	supervisor := &Supervisor{
		config: config, workload: contextBoundWorkload{},
		monitor: failingCaptureMonitor{},
	}
	started := time.Now()
	if _, _, err := supervisor.runPhase(
		context.Background(), UnitIdentity{}, 1, testDigest,
	); err == nil {
		t.Fatal("monitor failure passed")
	}
	if elapsed := time.Since(started); elapsed > 250*time.Millisecond {
		t.Fatalf("workload was not cancelled promptly: %s", elapsed)
	}
}

func TestValidatePhaseRejectsEveryRequiredPhysicalInvariant(t *testing.T) {
	base := validPhase(16)
	tests := map[string]func(*PhaseEvidence){
		"high_water":             func(value *PhaseEvidence) { value.HighWaterRuns = 15 },
		"run_ids":                func(value *PhaseEvidence) { value.RunIDs[1] = value.RunIDs[0] },
		"pids":                   func(value *PhaseEvidence) { value.FirecrackerPIDs[1] = value.FirecrackerPIDs[0] },
		"limits":                 func(value *PhaseEvidence) { value.LimitsExact = false },
		"swap":                   func(value *PhaseEvidence) { value.MemorySwapMaxZero = false },
		"network":                func(value *PhaseEvidence) { value.NetworkAbsent = false },
		"api":                    func(value *PhaseEvidence) { value.APIAbsent = false },
		"vsock":                  func(value *PhaseEvidence) { value.VsockAbsent = false },
		"serial":                 func(value *PhaseEvidence) { value.SerialAbsent = false },
		"main_pid_or_invocation": func(value *PhaseEvidence) { value.UnitIdentityStable = false },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			value := base
			value.RunIDs = append([]string(nil), base.RunIDs...)
			value.FirecrackerPIDs = append([]int(nil), base.FirecrackerPIDs...)
			mutate(&value)
			if validatePhase(value, 16) == nil {
				t.Fatal("mutant accepted")
			}
		})
	}
}

func TestValidatePhaseOutcomesRequiresExactObservedRunsAndCausalSubjects(t *testing.T) {
	basePhase := validPhase(16)
	baseOutcomes := validOutcomes(16, testDigest)
	tests := map[string]func(*PhaseEvidence, *[]AttestationOutcome){
		"foreign_run": func(_ *PhaseEvidence, outcomes *[]AttestationOutcome) {
			(*outcomes)[0].RunID = validTestRunID(15, 0)
		},
		"missing_run": func(_ *PhaseEvidence, outcomes *[]AttestationOutcome) {
			*outcomes = (*outcomes)[:len(*outcomes)-1]
		},
		"duplicate_run": func(_ *PhaseEvidence, outcomes *[]AttestationOutcome) {
			(*outcomes)[1].RunID = (*outcomes)[0].RunID
		},
		"duplicate_subject": func(_ *PhaseEvidence, outcomes *[]AttestationOutcome) {
			(*outcomes)[1].SubjectDigest = (*outcomes)[0].SubjectDigest
		},
		"duplicate_receipt": func(_ *PhaseEvidence, outcomes *[]AttestationOutcome) {
			(*outcomes)[1].ReceiptRef = (*outcomes)[0].ReceiptRef
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			phase := basePhase
			phase.RunIDs = append([]string(nil), basePhase.RunIDs...)
			outcomes := append([]AttestationOutcome(nil), baseOutcomes...)
			mutate(&phase, &outcomes)
			if _, err := validatePhaseOutcomes(
				phase, outcomes, 16, testDigest,
			); err == nil {
				t.Fatal("causal mutant accepted")
			}
		})
	}
}

func TestValidateOutcomesAllowsContentAddressedReceiptAcrossPhases(t *testing.T) {
	one := validOutcomes(1, testDigest)
	sixteen := validOutcomes(16, testDigest)
	sixteen[0].SubjectDigest = one[0].SubjectDigest
	sixteen[0].ReceiptRef = one[0].ReceiptRef
	sixteen[0].Ref = one[0].Ref
	outcomes := append(append(
		make([]AttestationOutcome, 0, 17),
		one...,
	), sixteen...)
	if err := validateOutcomes(outcomes, testDigest); err != nil {
		t.Fatalf("legitimate cross-phase content-addressed receipt rejected: %v", err)
	}
}

func TestMarshalReceiptV2IncludesEvidenceAndIsolation(t *testing.T) {
	receipt := receiptFor(Evidence{
		Candidate: CandidateIdentity{
			UnitSHA256: testDigest, LauncherSHA256: testDigest,
			PrimitivesUnitSHA256: testDigest,
			ConfigSHA256:         testDigest, SupervisorSHA256: testDigest,
			AssetDigest: testDigest,
		},
		PolicyDigest: testDigest,
		PhaseSixteen: validPhase(16),
		Cleanup:      CleanupEvidence{UnitStopped: true},
	})
	receipt.EvidenceSHA256 = testDigest
	content, err := marshalReceipt(receipt)
	if err != nil {
		t.Fatal(err)
	}
	if lines := strings.Split(strings.TrimSuffix(string(content), "\n"), "\n"); len(lines) != 20 {
		t.Fatalf("receipt lines=%d content=%s", len(lines), content)
	}
	for _, line := range []string{
		"schema=orquesta_firecracker_activation_receipt.v2",
		"evidence_sha256=" + testDigest,
		"policy_digest=" + testDigest,
		"concurrent_high_water=16",
		"api_absent=true",
		"vsock_absent=true",
		"serial_absent=true",
	} {
		if !strings.Contains(string(content), line+"\n") {
			t.Fatalf("missing %q in %s", line, content)
		}
	}
	if strings.Contains(string(content), "supervisor_sha256=") ||
		strings.Contains(string(content), "main_pid_stable=") ||
		strings.Contains(string(content), "invocation_id_stable=") ||
		strings.Contains(string(content), "nonce") ||
		strings.Contains(string(content), "run_id") {
		t.Fatalf("receipt exceeds installer contract: %s", content)
	}
}

func TestSupervisorIndependentInstancesRace(t *testing.T) {
	var wait sync.WaitGroup
	errorsCh := make(chan error, 32)
	for index := 0; index < 32; index++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			supervisor, err := New(
				validSupervisorConfig(), &fakeUnit{},
				&fakeWorkload{policy: testDigest}, &fakeMonitor{},
				&fakePublisher{}, fakeClock{time.Unix(1_700_000_000, 0)},
			)
			if err == nil {
				_, err = supervisor.Run(context.Background())
			}
			errorsCh <- err
		}()
	}
	wait.Wait()
	close(errorsCh)
	for err := range errorsCh {
		if err != nil {
			t.Fatal(err)
		}
	}
}

func validSupervisorConfig() Config {
	return Config{
		Candidate: Candidate{
			UnitName: "orquesta-firecracker-attestor-" + testDigest + ".service",
			UnitPath: "/etc/systemd/system/orquesta-firecracker-attestor-" +
				testDigest + ".service",
			UnitSHA256:         testDigest,
			PrimitivesUnitPath: "/primitives", PrimitivesUnitSHA256: testDigest,
			LauncherPath: "/launcher", LauncherSHA256: testDigest,
			LauncherSocketPath: "/run/orquesta/firecracker.sock",
			ConfigPath:         "/config", ConfigSHA256: testDigest,
			SupervisorPath: "/supervisor", SupervisorSHA256: testDigest,
			RuntimeRoot: "/runtime", CgroupRoot: "/cgroup",
			ParentCgroup: "parent", NetNSPath: "/netns",
			AssetDigest: testDigest,
		},
		PolicyDigest: testDigest,
		EvidencePath: "/evidence", ReceiptPath: "/receipt",
		PhaseTimeout: time.Second, CleanupTimeout: time.Second,
		StableFor: time.Millisecond, PollInterval: time.Millisecond,
		ChildUID: 1000, ChildGID: 1000,
		WorkloadPayload: []byte(
			`{"socket_path":"/run/orquesta/firecracker.sock","expected_asset_digest":"` +
				testDigest + `"}`,
		),
	}
}

func validPhase(count uint32) PhaseEvidence {
	phase := PhaseEvidence{
		RequestedRuns: count, HighWaterRuns: count, Samples: 1,
		LimitsExact: true, MemorySwapMaxZero: true, NetworkAbsent: true,
		APIAbsent: true, VsockAbsent: true, SerialAbsent: true,
		UnitIdentityStable: true,
	}
	for index := uint32(0); index < count; index++ {
		phase.RunIDs = append(phase.RunIDs, validTestRunID(count, index))
		phase.FirecrackerPIDs = append(phase.FirecrackerPIDs, 100+int(index))
	}
	return phase
}

func validOutcomes(count uint32, policy string) []AttestationOutcome {
	outcomes := make([]AttestationOutcome, 0, count)
	for index := uint32(0); index < count; index++ {
		runID := validTestRunID(count, index)
		receiptRef := fmt.Sprintf("receipt:%d:%d", count, index)
		outcomes = append(outcomes, AttestationOutcome{
			Ref: receiptRef, RunID: runID,
			SubjectDigest: fmt.Sprintf("%064x", index+1),
			ReceiptRef:    receiptRef,
			PolicyDigest:  policy, Valid: true,
		})
	}
	return outcomes
}

func validTestRunID(count, index uint32) string {
	const alphabet = "abcdefghijklmnopqrstuvwxyz234567"
	return "orq-" + strings.Repeat("a", 50) +
		string(alphabet[count%uint32(len(alphabet))]) +
		string(alphabet[index%uint32(len(alphabet))])
}

type fakeUnit struct {
	startCount   int
	stopCount    int
	stopped      bool
	startErr     error
	stopErr      error
	fragmentPath string
}

func (unit *fakeUnit) Preflight(context.Context, Config) (CandidateIdentity, error) {
	return CandidateIdentity{
		UnitSHA256: testDigest, LauncherSHA256: testDigest,
		PrimitivesUnitSHA256: testDigest,
		ConfigSHA256:         testDigest, SupervisorSHA256: testDigest,
		AssetDigest: testDigest,
	}, nil
}

func (unit *fakeUnit) Start(_ context.Context, name string) (UnitIdentity, error) {
	unit.startCount++
	if unit.startErr != nil {
		return UnitIdentity{}, unit.startErr
	}
	return UnitIdentity{
		UnitName: name, MainPID: 42,
		InvocationID: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", Active: true,
		FragmentPath: unit.path(name), Loaded: true,
	}, nil
}

func (unit *fakeUnit) WaitReady(
	ctx context.Context,
	_ Config,
	baseline UnitIdentity,
) (UnitIdentity, error) {
	return unit.Observe(ctx, baseline.UnitName)
}

func (unit *fakeUnit) Observe(_ context.Context, name string) (UnitIdentity, error) {
	if unit.stopped {
		return UnitIdentity{
			UnitName: name, FragmentPath: unit.path(name),
			Loaded: true,
		}, nil
	}
	return UnitIdentity{
		UnitName: name, MainPID: 42,
		InvocationID: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", Active: true,
		FragmentPath: unit.path(name), Loaded: true,
	}, nil
}

func (unit *fakeUnit) path(name string) string {
	if unit.fragmentPath != "" {
		return unit.fragmentPath
	}
	return "/etc/systemd/system/" + name
}

func (unit *fakeUnit) Stop(context.Context, string) error {
	unit.stopCount++
	if unit.stopErr == nil {
		unit.stopped = true
	}
	return unit.stopErr
}

type blockingPreflightUnit struct{}

func (blockingPreflightUnit) Preflight(
	ctx context.Context,
	_ Config,
) (CandidateIdentity, error) {
	<-ctx.Done()
	return CandidateIdentity{}, ctx.Err()
}

func (blockingPreflightUnit) Start(context.Context, string) (UnitIdentity, error) {
	return UnitIdentity{}, errors.New("unexpected start")
}

func (blockingPreflightUnit) WaitReady(
	context.Context,
	Config,
	UnitIdentity,
) (UnitIdentity, error) {
	return UnitIdentity{}, errors.New("unexpected readiness")
}

func (blockingPreflightUnit) Observe(context.Context, string) (UnitIdentity, error) {
	return UnitIdentity{}, errors.New("unexpected observe")
}

func (blockingPreflightUnit) Stop(context.Context, string) error {
	return errors.New("unexpected stop")
}

type delayedReadyUnit struct {
	fakeUnit
	entered chan struct{}
	release chan struct{}
}

func (unit *delayedReadyUnit) WaitReady(
	ctx context.Context,
	_ Config,
	baseline UnitIdentity,
) (UnitIdentity, error) {
	close(unit.entered)
	select {
	case <-ctx.Done():
		return UnitIdentity{}, errReadinessTimeout
	case <-unit.release:
		return unit.Observe(ctx, baseline.UnitName)
	}
}

type neverReadyUnit struct{ fakeUnit }

func (unit *neverReadyUnit) WaitReady(
	ctx context.Context,
	_ Config,
	_ UnitIdentity,
) (UnitIdentity, error) {
	<-ctx.Done()
	return UnitIdentity{}, errReadinessTimeout
}

type failedReadyUnit struct {
	fakeUnit
	cause error
}

func (unit *failedReadyUnit) WaitReady(
	context.Context,
	Config,
	UnitIdentity,
) (UnitIdentity, error) {
	return UnitIdentity{}, unit.cause
}

type fakeWorkload struct {
	policy  string
	calls   int
	invalid bool
}

func (workload *fakeWorkload) StartPhase(
	_ context.Context,
	request WorkloadRequest,
) (WorkloadRun, error) {
	workload.calls++
	done := make(chan struct{})
	close(done)
	outcomes := validOutcomes(request.Count, workload.policy)
	for index := range outcomes {
		outcomes[index].Valid = !workload.invalid
	}
	return &fakeRun{done: done, outcomes: outcomes}, nil
}

type fakeRun struct {
	done     chan struct{}
	outcomes []AttestationOutcome
}

func (run *fakeRun) Done() <-chan struct{} { return run.done }
func (run *fakeRun) Result() ([]AttestationOutcome, error) {
	return append([]AttestationOutcome(nil), run.outcomes...), nil
}
func (run *fakeRun) Close() error { return nil }

type contextBoundWorkload struct{}

func (contextBoundWorkload) StartPhase(
	ctx context.Context,
	_ WorkloadRequest,
) (WorkloadRun, error) {
	return &contextBoundRun{ctx: ctx, done: make(chan struct{})}, nil
}

type contextBoundRun struct {
	ctx  context.Context
	done chan struct{}
}

func (run *contextBoundRun) Done() <-chan struct{} { return run.done }

func (run *contextBoundRun) Result() ([]AttestationOutcome, error) {
	<-run.ctx.Done()
	close(run.done)
	return nil, run.ctx.Err()
}

func (*contextBoundRun) Close() error { return nil }

type failingCaptureMonitor struct{}

func (failingCaptureMonitor) CapturePhase(
	context.Context,
	UnitIdentity,
	WorkloadRun,
	uint32,
) (PhaseEvidence, error) {
	return PhaseEvidence{}, errors.New("capture failed")
}

func (failingCaptureMonitor) WaitQuiescent(
	context.Context,
	UnitIdentity,
) (CleanupEvidence, error) {
	return CleanupEvidence{}, errors.New("unexpected wait")
}

type fakeMonitor struct {
	badPeak        bool
	unitUnstable   bool
	apiPresent     bool
	residual       bool
	socketResidual bool
	waitCount      int
}

func (monitor *fakeMonitor) CapturePhase(
	_ context.Context,
	_ UnitIdentity,
	_ WorkloadRun,
	expected uint32,
) (PhaseEvidence, error) {
	phase := validPhase(expected)
	if monitor.badPeak {
		phase.HighWaterRuns--
	}
	if monitor.unitUnstable {
		phase.UnitIdentityStable = false
	}
	if monitor.apiPresent {
		phase.APIAbsent = false
	}
	return phase, nil
}

func (monitor *fakeMonitor) WaitQuiescent(
	_ context.Context,
	baseline UnitIdentity,
) (CleanupEvidence, error) {
	monitor.waitCount++
	cleanup := CleanupEvidence{
		StableSamples: 2,
		UnitStopped:   !baseline.Active,
		SocketAbsent:  !baseline.Active,
	}
	if monitor.residual {
		cleanup.ResidualProcesses = 1
	}
	if monitor.socketResidual && !baseline.Active {
		cleanup.SocketAbsent = false
	}
	return cleanup, nil
}

type fakePublisher struct {
	called   bool
	evidence Evidence
	receipt  Receipt
}

func (publisher *fakePublisher) Publish(
	evidence Evidence,
	receipt Receipt,
) (Publication, error) {
	publisher.called = true
	publisher.evidence = evidence
	publisher.receipt = receipt
	return Publication{
		EvidenceSHA256: testDigest,
		EvidencePath:   "/evidence", ReceiptPath: "/receipt",
	}, nil
}

type fakeClock struct{ value time.Time }

func (clock fakeClock) Now() time.Time { return clock.value }
