package selfwatchdog

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"
)

func TestSelfWatchdogSustainedCPUWithoutProgressPublishesEvidenceAndStopsOnlyOwnComposition(t *testing.T) {
	base := time.Unix(1_800_000_000, 0).UTC()
	telemetry := &fakeTelemetry{samples: []TelemetrySample{
		telemetryAt(base),
		telemetryAt(base.Add(time.Minute)),
		telemetryAt(base.Add(2 * time.Minute)),
	}}
	progress := &fakeProgress{snapshots: []ProgressSnapshot{
		progressAt(base),
		progressAt(base.Add(time.Minute)),
		progressAt(base.Add(2 * time.Minute)),
	}}
	evidence := newFakeEvidence()
	shutdown := newFakeShutdown()
	state := &fakeCheckpointStore{}
	watchdog := mustWatchdog(t, testPolicy(), Identity{
		OwnerRef: "owner:orquesta", InstanceRef: "process:one", FencingToken: 1,
	}, Ports{
		Telemetry: telemetry, Progress: progress, Evidence: evidence, Shutdown: shutdown, State: state,
	})

	for index, expected := range []Decision{
		DecisionSustainedWindowPending,
		DecisionSustainedWindowPending,
		DecisionShutdownAdmitted,
	} {
		outcome, err := watchdog.Tick(context.Background())
		if err != nil {
			t.Fatalf("tick %d: %v", index, err)
		}
		if outcome.Decision != expected {
			t.Fatalf("tick %d decision = %s, want %s", index, outcome.Decision, expected)
		}
	}
	outcome, err := watchdog.Tick(context.Background())
	if err != nil || outcome.Decision != DecisionShutdownAlreadyAdmitted {
		t.Fatalf("idempotent tick = %+v/%v", outcome, err)
	}
	if telemetry.calls != 3 || progress.calls != 3 {
		t.Fatalf("completed incident resampled ports: telemetry=%d progress=%d", telemetry.calls, progress.calls)
	}
	if len(evidence.unique) != 1 || evidence.attempts != 1 || len(shutdown.unique) != 1 || shutdown.attempts != 1 {
		t.Fatalf("effects evidence=%d/%d shutdown=%d/%d",
			len(evidence.unique), evidence.attempts, len(shutdown.unique), shutdown.attempts)
	}
	var published Evidence
	for _, value := range evidence.unique {
		published = value
	}
	if published.Code != evidenceCode || published.NextActionCode == "" ||
		published.OwnerRef != "owner:orquesta" || published.InstanceRef != "process:one" ||
		published.HighCPUStartedAt != base || published.ObservedAt != base.Add(2*time.Minute) ||
		published.CPUPercent != 90 || published.CPUHighPercent != 80 ||
		published.SustainedFor != 2*time.Minute || published.NoProgressFor != 3*time.Minute {
		t.Fatalf("evidence is not actionable: %+v", published)
	}
	var request ShutdownAdmissionRequest
	for _, value := range shutdown.unique {
		request = value
	}
	if request.OwnerRef != published.OwnerRef || request.InstanceRef != published.InstanceRef ||
		request.Mode != ShutdownCooperative || request.ReasonCode != published.Code ||
		request.FencingToken != 1 || request.EvidenceRef == "" ||
		request.IdempotencyKey != published.IdempotencyKey {
		t.Fatalf("shutdown escaped evidence/ownership scope: %+v", request)
	}
	if reflect.TypeOf(request).NumField() != 8 {
		t.Fatal("shutdown request gained an unreviewed selector")
	}
}

func TestSelfWatchdogEveryTypedOperationalCauseSuppressesShutdown(t *testing.T) {
	tests := map[string]func(*ProgressSnapshot){
		"Goal":                         func(snapshot *ProgressSnapshot) { snapshot.ActiveGoals = 1 },
		"test":                         func(snapshot *ProgressSnapshot) { snapshot.ActiveTests = 1 },
		"Firecracker":                  func(snapshot *ProgressSnapshot) { snapshot.ActiveFirecracker = 1 },
		"active query":                 func(snapshot *ProgressSnapshot) { snapshot.ActiveQueries = 1 },
		"shutdown already in progress": func(snapshot *ProgressSnapshot) { snapshot.ShutdownInProgress = true },
	}
	for name, setCause := range tests {
		t.Run(name, func(t *testing.T) {
			base := time.Unix(1_800_000_100, 0).UTC()
			snapshot := progressAt(base)
			setCause(&snapshot)
			evidence := newFakeEvidence()
			shutdown := newFakeShutdown()
			watchdog := mustWatchdog(t, testPolicy(), Identity{
				OwnerRef: "owner:orquesta", InstanceRef: "process:cause", FencingToken: 1,
			}, Ports{
				Telemetry: &fakeTelemetry{samples: []TelemetrySample{telemetryAt(base)}},
				Progress:  &fakeProgress{snapshots: []ProgressSnapshot{snapshot}},
				Evidence:  evidence, Shutdown: shutdown, State: &fakeCheckpointStore{},
			})
			outcome, err := watchdog.Tick(context.Background())
			if err != nil || outcome.Decision != DecisionLiveCause {
				t.Fatalf("Tick() = %+v/%v", outcome, err)
			}
			if evidence.attempts != 0 || shutdown.attempts != 0 {
				t.Fatal("live operational cause emitted an effect")
			}
		})
	}
}

func TestSelfWatchdogRecentDurableProgressSuppressesAndLowCPUResetsSustainedWindow(t *testing.T) {
	base := time.Unix(1_800_000_200, 0).UTC()
	recent := progressAt(base)
	recent.LastDurableProgressAt = base.Add(-time.Minute)
	recent.DurableRevision = "revision:goal:42"
	telemetry := &fakeTelemetry{samples: []TelemetrySample{
		telemetryAt(base),
		telemetryAt(base.Add(10 * time.Minute)),
		telemetryAt(base.Add(20 * time.Minute)),
	}}
	telemetry.samples[1].CPUPercent = 10
	progress := &fakeProgress{snapshots: []ProgressSnapshot{
		recent,
		progressAt(base.Add(10 * time.Minute)),
		progressAt(base.Add(20 * time.Minute)),
	}}
	watchdog := mustWatchdog(t, testPolicy(), Identity{
		OwnerRef: "owner:orquesta", InstanceRef: "process:progress", FencingToken: 1,
	}, Ports{
		Telemetry: telemetry, Progress: progress,
		Evidence: newFakeEvidence(), Shutdown: newFakeShutdown(), State: &fakeCheckpointStore{},
	})
	for index, expected := range []Decision{
		DecisionRecentDurableProgress,
		DecisionBelowThreshold,
		DecisionSustainedWindowPending,
	} {
		outcome, err := watchdog.Tick(context.Background())
		if err != nil || outcome.Decision != expected {
			t.Fatalf("tick %d = %+v/%v, want %s", index, outcome, err, expected)
		}
	}
}

func TestSelfWatchdogPendingIncidentIsCanceledWhenAGoalBecomesActive(t *testing.T) {
	base := time.Unix(1_800_000_250, 0).UTC()
	activeGoal := progressAt(base.Add(3 * time.Minute))
	activeGoal.ActiveGoals = 1
	evidence := newFakeEvidence()
	evidence.failAttempt = 1
	shutdown := newFakeShutdown()
	state := &fakeCheckpointStore{}
	ports := Ports{
		Telemetry: &fakeTelemetry{samples: []TelemetrySample{
			telemetryAt(base),
			telemetryAt(base.Add(2 * time.Minute)),
			telemetryAt(base.Add(3 * time.Minute)),
		}},
		Progress: &fakeProgress{snapshots: []ProgressSnapshot{
			progressAt(base),
			progressAt(base.Add(2 * time.Minute)),
			activeGoal,
		}},
		Evidence: evidence, Shutdown: shutdown, State: state,
	}
	identity := Identity{OwnerRef: "owner:orquesta", InstanceRef: "process:late-goal", FencingToken: 1}
	watchdog := mustWatchdog(t, testPolicy(), identity, ports)
	if _, err := watchdog.Tick(context.Background()); err != nil {
		t.Fatalf("prime high CPU: %v", err)
	}
	if _, err := watchdog.Tick(context.Background()); !errors.Is(err, errFakeEvidence) {
		t.Fatalf("persisted incident error = %v", err)
	}

	restarted := mustWatchdog(t, testPolicy(), identity, ports)
	outcome, err := restarted.Tick(context.Background())
	if err != nil || outcome.Decision != DecisionLiveCause {
		t.Fatalf("active Goal did not veto pending shutdown: %+v/%v", outcome, err)
	}
	if evidence.attempts != 1 || shutdown.attempts != 0 ||
		state.checkpoint.Incident.IdempotencyKey != "" || !state.checkpoint.HighCPUSince.IsZero() {
		t.Fatalf("pending incident was not canceled safely: evidence=%d shutdown=%d checkpoint=%+v",
			evidence.attempts, shutdown.attempts, state.checkpoint)
	}
}

func TestSelfWatchdogRestartDoesNotInheritHighCPUWindowFromPreviousProcessInstance(t *testing.T) {
	base := time.Unix(1_800_000_300, 0).UTC()
	state := &fakeCheckpointStore{}
	evidence := newFakeEvidence()
	shutdown := newFakeShutdown()
	first := mustWatchdog(t, testPolicy(), Identity{
		OwnerRef: "owner:orquesta", InstanceRef: "process:first", FencingToken: 1,
	}, Ports{
		Telemetry: &fakeTelemetry{samples: []TelemetrySample{telemetryAt(base)}},
		Progress:  &fakeProgress{snapshots: []ProgressSnapshot{progressAt(base)}},
		Evidence:  evidence, Shutdown: shutdown, State: state,
	})
	outcome, err := first.Tick(context.Background())
	if err != nil || outcome.Decision != DecisionSustainedWindowPending {
		t.Fatalf("first process tick = %+v/%v", outcome, err)
	}

	restartedAt := base.Add(30 * time.Minute)
	second := mustWatchdog(t, testPolicy(), Identity{
		OwnerRef: "owner:orquesta", InstanceRef: "process:second", FencingToken: 2,
	}, Ports{
		Telemetry: &fakeTelemetry{samples: []TelemetrySample{
			telemetryAt(restartedAt),
			telemetryAt(restartedAt.Add(2 * time.Minute)),
		}},
		Progress: &fakeProgress{snapshots: []ProgressSnapshot{
			progressAt(restartedAt),
			progressAt(restartedAt.Add(2 * time.Minute)),
		}},
		Evidence: evidence, Shutdown: shutdown, State: state,
	})
	outcome, err = second.Tick(context.Background())
	if err != nil || outcome.Decision != DecisionSustainedWindowPending {
		t.Fatalf("restart inherited stale window: %+v/%v", outcome, err)
	}
	outcome, err = second.Tick(context.Background())
	if err != nil || outcome.Decision != DecisionShutdownAdmitted {
		t.Fatalf("new process did not build its own window: %+v/%v", outcome, err)
	}
	if len(shutdown.unique) != 1 {
		t.Fatalf("shutdown count = %d", len(shutdown.unique))
	}
	for _, request := range shutdown.unique {
		if request.InstanceRef != "process:second" {
			t.Fatalf("restart stopped foreign instance: %+v", request)
		}
	}
}

func TestSelfWatchdogRestartRequiresStrictlyNewerInstanceFence(t *testing.T) {
	base := time.Unix(1_800_000_325, 0).UTC()
	state := &fakeCheckpointStore{
		found:    true,
		revision: 4,
		checkpoint: Checkpoint{
			SchemaVersion: checkpointSchemaVersion,
			InstanceRef:   "process:current",
			FencingToken:  7,
			HighCPUSince:  base,
		},
	}
	ports := Ports{
		Telemetry: &fakeTelemetry{samples: []TelemetrySample{telemetryAt(base.Add(time.Hour))}},
		Progress:  &fakeProgress{snapshots: []ProgressSnapshot{progressAt(base.Add(time.Hour))}},
		Evidence:  newFakeEvidence(), Shutdown: newFakeShutdown(), State: state,
	}
	stale := mustWatchdog(t, testPolicy(), Identity{
		OwnerRef: "owner:orquesta", InstanceRef: "process:stale", FencingToken: 7,
	}, ports)
	if _, err := stale.Tick(context.Background()); !HasErrorCode(err, ErrorCheckpointInvalid) {
		t.Fatalf("stale fence error = %v", err)
	}
	if state.checkpoint.InstanceRef != "process:current" || state.checkpoint.FencingToken != 7 {
		t.Fatalf("stale instance replaced checkpoint: %+v", state.checkpoint)
	}

	current := mustWatchdog(t, testPolicy(), Identity{
		OwnerRef: "owner:orquesta", InstanceRef: "process:new", FencingToken: 8,
	}, ports)
	outcome, err := current.Tick(context.Background())
	if err != nil || outcome.Decision != DecisionSustainedWindowPending {
		t.Fatalf("new fenced restart = %+v/%v", outcome, err)
	}
	if state.checkpoint.InstanceRef != "process:new" || state.checkpoint.FencingToken != 8 {
		t.Fatalf("new fence did not own checkpoint: %+v", state.checkpoint)
	}
}

func TestSelfWatchdogCheckpointCASRejectsAConcurrentStaleWriter(t *testing.T) {
	state := &fakeCheckpointStore{
		found: true, revision: 1,
		checkpoint: Checkpoint{
			SchemaVersion: checkpointSchemaVersion,
			InstanceRef:   "process:old", FencingToken: 1,
		},
	}
	first := CheckpointWrite{
		OwnerRef: "owner:orquesta", ExpectedRevision: 1,
		Checkpoint: Checkpoint{
			SchemaVersion: checkpointSchemaVersion,
			InstanceRef:   "process:first", FencingToken: 2,
		},
	}
	second := CheckpointWrite{
		OwnerRef: "owner:orquesta", ExpectedRevision: 1,
		Checkpoint: Checkpoint{
			SchemaVersion: checkpointSchemaVersion,
			InstanceRef:   "process:second", FencingToken: 3,
		},
	}
	if _, err := state.StoreCheckpoint(context.Background(), first); err != nil {
		t.Fatalf("first CAS: %v", err)
	}
	if _, err := state.StoreCheckpoint(context.Background(), second); err == nil {
		t.Fatal("concurrent stale expected revision was accepted")
	}
	if state.checkpoint.InstanceRef != "process:first" || state.revision != 2 {
		t.Fatalf("stale writer mutated checkpoint: %+v revision=%d", state.checkpoint, state.revision)
	}
}

func TestSelfWatchdogSpoofedCheckpointCannotPublishOrRequestShutdown(t *testing.T) {
	base := time.Unix(1_800_000_350, 0).UTC()
	identity := Identity{
		OwnerRef: "owner:orquesta", InstanceRef: "process:spoofed", FencingToken: 1,
	}
	incidentKey := buildIncidentKey(identity, base)
	state := &fakeCheckpointStore{
		found:    true,
		revision: 1,
		checkpoint: Checkpoint{
			SchemaVersion: checkpointSchemaVersion,
			InstanceRef:   "process:spoofed",
			FencingToken:  1,
			HighCPUSince:  base,
			Incident: IncidentCheckpoint{
				IdempotencyKey:   incidentKey,
				HighCPUStartedAt: base,
				TriggeredAt:      base.Add(2 * time.Minute),
				Evidence: Evidence{
					Code:           "content_selected_shutdown",
					NextActionCode: nextActionCode,
					OwnerRef:       "owner:orquesta", InstanceRef: "process:spoofed",
					FencingToken:     1,
					IdempotencyKey:   incidentKey,
					HighCPUStartedAt: base, ObservedAt: base.Add(2 * time.Minute),
					CPUPercent: 90, CPUHighPercent: 80,
					SustainedFor: 2 * time.Minute, NoProgressFor: 3 * time.Minute,
				},
			},
		},
	}
	evidence := newFakeEvidence()
	shutdown := newFakeShutdown()
	watchdog := mustWatchdog(t, testPolicy(), identity, Ports{
		Telemetry: &fakeTelemetry{}, Progress: &fakeProgress{},
		Evidence: evidence, Shutdown: shutdown, State: state,
	})
	_, err := watchdog.Tick(context.Background())
	if !HasErrorCode(err, ErrorCheckpointInvalid) {
		t.Fatalf("spoofed checkpoint error = %v", err)
	}
	if evidence.attempts != 0 || shutdown.attempts != 0 {
		t.Fatal("spoofed checkpoint emitted an effect")
	}
}

func TestSelfWatchdogSpoofedEffectReceiptsAreRejected(t *testing.T) {
	base := time.Unix(1_800_000_375, 0).UTC()
	tests := map[string]struct {
		evidence *fakeEvidence
		shutdown *fakeShutdown
	}{
		"evidence owner": {
			evidence: func() *fakeEvidence {
				fake := newFakeEvidence()
				fake.spoofOwner = true
				return fake
			}(),
			shutdown: newFakeShutdown(),
		},
		"shutdown fence": {
			evidence: newFakeEvidence(),
			shutdown: func() *fakeShutdown {
				fake := newFakeShutdown()
				fake.spoofFence = true
				return fake
			}(),
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			watchdog := mustWatchdog(t, testPolicy(), Identity{
				OwnerRef: "owner:orquesta", InstanceRef: "process:spoof-receipt", FencingToken: 1,
			}, Ports{
				Telemetry: &fakeTelemetry{samples: []TelemetrySample{
					telemetryAt(base), telemetryAt(base.Add(2 * time.Minute)),
				}},
				Progress: &fakeProgress{snapshots: []ProgressSnapshot{
					progressAt(base), progressAt(base.Add(2 * time.Minute)),
				}},
				Evidence: test.evidence, Shutdown: test.shutdown, State: &fakeCheckpointStore{},
			})
			if _, err := watchdog.Tick(context.Background()); err != nil {
				t.Fatalf("prime high CPU: %v", err)
			}
			if _, err := watchdog.Tick(context.Background()); !HasErrorCode(err, ErrorReceiptInvalid) {
				t.Fatalf("spoofed receipt error = %v", err)
			}
		})
	}
}

func TestSelfWatchdogEffectRetriesRemainIdempotentAcrossCheckpointFailures(t *testing.T) {
	tests := map[string]int{
		"evidence acknowledgement": 4,
		"shutdown acknowledgement": 5,
	}
	for name, failingStoreCall := range tests {
		t.Run(name, func(t *testing.T) {
			base := time.Unix(1_800_000_400, 0).UTC()
			state := &fakeCheckpointStore{failStoreCall: failingStoreCall}
			evidence := newFakeEvidence()
			shutdown := newFakeShutdown()
			telemetry := &fakeTelemetry{samples: []TelemetrySample{
				telemetryAt(base),
				telemetryAt(base.Add(2 * time.Minute)),
				telemetryAt(base.Add(3 * time.Minute)),
			}}
			progress := &fakeProgress{snapshots: []ProgressSnapshot{
				progressAt(base),
				progressAt(base.Add(2 * time.Minute)),
				progressAt(base.Add(3 * time.Minute)),
			}}
			ports := Ports{
				Telemetry: telemetry, Progress: progress,
				Evidence: evidence, Shutdown: shutdown, State: state,
			}
			identity := Identity{
				OwnerRef: "owner:orquesta", InstanceRef: "process:retry", FencingToken: 1,
			}
			watchdog := mustWatchdog(t, testPolicy(), identity, ports)
			if _, err := watchdog.Tick(context.Background()); err != nil {
				t.Fatalf("prime high CPU: %v", err)
			}
			if _, err := watchdog.Tick(context.Background()); !errors.Is(err, errFakeStore) {
				t.Fatalf("trigger error = %v", err)
			}

			restarted := mustWatchdog(t, testPolicy(), identity, ports)
			outcome, err := restarted.Tick(context.Background())
			if err != nil || outcome.Decision != DecisionShutdownAdmitted {
				t.Fatalf("retry = %+v/%v", outcome, err)
			}
			if len(evidence.unique) != 1 || len(shutdown.unique) != 1 {
				t.Fatalf("duplicate semantic effects evidence=%d shutdown=%d",
					len(evidence.unique), len(shutdown.unique))
			}
			if failingStoreCall == 4 && evidence.attempts != 2 {
				t.Fatalf("evidence attempts = %d, want replay", evidence.attempts)
			}
			if failingStoreCall == 5 && shutdown.attempts != 1 {
				t.Fatalf("shutdown attempts = %d, want one admission after durable intent", shutdown.attempts)
			}
		})
	}
}

func TestSelfWatchdogShutdownAdmissionRetryUsesPersistedIntentAndStableKey(t *testing.T) {
	base := time.Unix(1_800_000_450, 0).UTC()
	shutdown := newFakeShutdown()
	shutdown.failAttempt = 1
	state := &fakeCheckpointStore{}
	telemetry := &fakeTelemetry{samples: []TelemetrySample{
		telemetryAt(base),
		telemetryAt(base.Add(2 * time.Minute)),
		telemetryAt(base.Add(3 * time.Minute)),
	}}
	activeGoal := progressAt(base.Add(3 * time.Minute))
	activeGoal.ActiveGoals = 1
	progress := &fakeProgress{snapshots: []ProgressSnapshot{
		progressAt(base),
		progressAt(base.Add(2 * time.Minute)),
		activeGoal,
	}}
	ports := Ports{
		Telemetry: telemetry,
		Progress:  progress,
		Evidence:  newFakeEvidence(), Shutdown: shutdown, State: state,
	}
	identity := Identity{
		OwnerRef: "owner:orquesta", InstanceRef: "process:admission-retry", FencingToken: 1,
	}
	watchdog := mustWatchdog(t, testPolicy(), identity, ports)
	if _, err := watchdog.Tick(context.Background()); err != nil {
		t.Fatalf("prime high CPU: %v", err)
	}
	if _, err := watchdog.Tick(context.Background()); !errors.Is(err, errFakeShutdown) {
		t.Fatalf("first admission error = %v", err)
	}
	if !state.checkpoint.Incident.ShutdownIntentStored {
		t.Fatal("shutdown intent was not durable before admission")
	}

	restarted := mustWatchdog(t, testPolicy(), identity, ports)
	outcome, err := restarted.Tick(context.Background())
	if err != nil || outcome.Decision != DecisionShutdownAdmitted {
		t.Fatalf("admission retry = %+v/%v", outcome, err)
	}
	if shutdown.attempts != 2 || len(shutdown.unique) != 1 {
		t.Fatalf("admission replay attempts=%d unique=%d", shutdown.attempts, len(shutdown.unique))
	}
	if telemetry.calls != 2 || progress.calls != 2 ||
		state.checkpoint.Incident.IdempotencyKey == "" {
		t.Fatalf("durable intent was reobserved or canceled: telemetry=%d progress=%d checkpoint=%+v",
			telemetry.calls, progress.calls, state.checkpoint)
	}
}

func TestSelfWatchdogMissingOrStaleObservationsFailSafeWithoutEffects(t *testing.T) {
	base := time.Unix(1_800_000_500, 0).UTC()
	tests := map[string]struct {
		telemetry TelemetrySample
		progress  ProgressSnapshot
		code      ErrorCode
	}{
		"invalid CPU": {
			telemetry: func() TelemetrySample {
				sample := telemetryAt(base)
				sample.CPUPercent = 101
				return sample
			}(),
			progress: progressAt(base),
			code:     ErrorTelemetryInvalid,
		},
		"stale progress": {
			telemetry: telemetryAt(base),
			progress:  progressAt(base.Add(-2 * time.Minute)),
			code:      ErrorProgressInvalid,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			evidence := newFakeEvidence()
			shutdown := newFakeShutdown()
			watchdog := mustWatchdog(t, testPolicy(), Identity{
				OwnerRef: "owner:orquesta", InstanceRef: "process:invalid", FencingToken: 1,
			}, Ports{
				Telemetry: &fakeTelemetry{samples: []TelemetrySample{test.telemetry}},
				Progress:  &fakeProgress{snapshots: []ProgressSnapshot{test.progress}},
				Evidence:  evidence, Shutdown: shutdown, State: &fakeCheckpointStore{},
			})
			_, err := watchdog.Tick(context.Background())
			if !HasErrorCode(err, test.code) {
				t.Fatalf("Tick() error = %v", err)
			}
			if evidence.attempts != 0 || shutdown.attempts != 0 {
				t.Fatal("invalid observation emitted an effect")
			}
		})
	}
}

func testPolicy() Policy {
	return Policy{
		ObservationInterval: time.Minute,
		CPUHighPercent:      80,
		SustainedFor:        2 * time.Minute,
		NoProgressFor:       3 * time.Minute,
	}
}

func telemetryAt(at time.Time) TelemetrySample {
	return TelemetrySample{
		ObservedAt:  at,
		CPUPercent:  90,
		MemoryBytes: 32 << 20,
		Uptime:      10 * time.Minute,
	}
}

func progressAt(at time.Time) ProgressSnapshot {
	return ProgressSnapshot{ObservedAt: at}
}

func mustWatchdog(t *testing.T, policy Policy, identity Identity, ports Ports) *Watchdog {
	t.Helper()
	watchdog, err := New(policy, identity, ports)
	if err != nil {
		t.Fatalf("New(): %v", err)
	}
	return watchdog
}

type fakeTelemetry struct {
	samples []TelemetrySample
	calls   int
}

func (fake *fakeTelemetry) Sample(context.Context) (TelemetrySample, error) {
	if fake.calls >= len(fake.samples) {
		return TelemetrySample{}, errors.New("fake.telemetry_exhausted")
	}
	sample := fake.samples[fake.calls]
	fake.calls++
	return sample, nil
}

type fakeProgress struct {
	snapshots []ProgressSnapshot
	calls     int
}

func (fake *fakeProgress) ObserveProgress(context.Context) (ProgressSnapshot, error) {
	if fake.calls >= len(fake.snapshots) {
		return ProgressSnapshot{}, errors.New("fake.progress_exhausted")
	}
	snapshot := fake.snapshots[fake.calls]
	fake.calls++
	return snapshot, nil
}

type fakeEvidence struct {
	attempts    int
	failAttempt int
	spoofOwner  bool
	unique      map[string]Evidence
}

func newFakeEvidence() *fakeEvidence {
	return &fakeEvidence{unique: make(map[string]Evidence)}
}

func (fake *fakeEvidence) PublishEvidence(_ context.Context, evidence Evidence) (EvidenceReceipt, error) {
	fake.attempts++
	if fake.attempts == fake.failAttempt {
		return EvidenceReceipt{}, errFakeEvidence
	}
	if previous, found := fake.unique[evidence.IdempotencyKey]; found {
		if !reflect.DeepEqual(previous, evidence) {
			return EvidenceReceipt{}, errors.New("fake.evidence_idempotency_conflict")
		}
	} else {
		fake.unique[evidence.IdempotencyKey] = evidence
	}
	receipt := EvidenceReceipt{
		OwnerRef:       evidence.OwnerRef,
		InstanceRef:    evidence.InstanceRef,
		FencingToken:   evidence.FencingToken,
		EvidenceRef:    "evidence:" + evidence.IdempotencyKey,
		IdempotencyKey: evidence.IdempotencyKey,
	}
	if fake.spoofOwner {
		receipt.OwnerRef = "owner:foreign"
	}
	return receipt, nil
}

var errFakeEvidence = errors.New("fake.evidence_failed")
var errFakeShutdown = errors.New("fake.shutdown_admission_failed")

type fakeShutdown struct {
	attempts    int
	failAttempt int
	spoofFence  bool
	unique      map[string]ShutdownAdmissionRequest
}

func newFakeShutdown() *fakeShutdown {
	return &fakeShutdown{unique: make(map[string]ShutdownAdmissionRequest)}
}

func (fake *fakeShutdown) AdmitOwnShutdown(
	_ context.Context,
	request ShutdownAdmissionRequest,
) (ShutdownAdmissionReceipt, error) {
	fake.attempts++
	if fake.attempts == fake.failAttempt {
		return ShutdownAdmissionReceipt{}, errFakeShutdown
	}
	if previous, found := fake.unique[request.IdempotencyKey]; found {
		if !reflect.DeepEqual(previous, request) {
			return ShutdownAdmissionReceipt{}, errors.New("fake.shutdown_idempotency_conflict")
		}
	} else {
		fake.unique[request.IdempotencyKey] = request
	}
	receipt := ShutdownAdmissionReceipt{
		OwnerRef:       request.OwnerRef,
		InstanceRef:    request.InstanceRef,
		FencingToken:   request.FencingToken,
		Mode:           request.Mode,
		IdempotencyKey: request.IdempotencyKey,
	}
	if fake.spoofFence {
		receipt.FencingToken++
	}
	return receipt, nil
}

var errFakeStore = errors.New("fake.checkpoint_store_failed")

type fakeCheckpointStore struct {
	mu            sync.Mutex
	found         bool
	revision      uint64
	checkpoint    Checkpoint
	storeCalls    int
	failStoreCall int
}

func (fake *fakeCheckpointStore) LoadCheckpoint(
	_ context.Context,
	_ string,
) (CheckpointSnapshot, error) {
	fake.mu.Lock()
	defer fake.mu.Unlock()
	return CheckpointSnapshot{
		Found:      fake.found,
		Revision:   fake.revision,
		Checkpoint: fake.checkpoint,
	}, nil
}

func (fake *fakeCheckpointStore) StoreCheckpoint(
	_ context.Context,
	write CheckpointWrite,
) (CheckpointReceipt, error) {
	fake.mu.Lock()
	defer fake.mu.Unlock()
	fake.storeCalls++
	if fake.storeCalls == fake.failStoreCall {
		return CheckpointReceipt{}, errFakeStore
	}
	if write.ExpectedRevision != fake.revision {
		return CheckpointReceipt{}, errors.New("fake.checkpoint_revision_conflict")
	}
	if fake.found &&
		(write.Checkpoint.FencingToken < fake.checkpoint.FencingToken ||
			(write.Checkpoint.FencingToken == fake.checkpoint.FencingToken &&
				write.Checkpoint.InstanceRef != fake.checkpoint.InstanceRef)) {
		return CheckpointReceipt{}, errors.New("fake.checkpoint_fence_stale")
	}
	previous := fake.revision
	fake.revision++
	fake.found = true
	fake.checkpoint = write.Checkpoint
	return CheckpointReceipt{
		OwnerRef:         write.OwnerRef,
		InstanceRef:      write.Checkpoint.InstanceRef,
		FencingToken:     write.Checkpoint.FencingToken,
		PreviousRevision: previous,
		Revision:         fake.revision,
	}, nil
}
