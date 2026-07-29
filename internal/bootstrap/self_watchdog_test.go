package bootstrap

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"orquesta/internal/adapters/system/selfwatchdog"
	"orquesta/internal/config"
)

func TestSelfWatchdogCompositionKeepsDisabledCompositionDependencyFree(t *testing.T) {
	loop, err := ComposeSelfWatchdog(config.SelfWatchdogPolicy{}, selfwatchdog.Identity{}, SelfWatchdogDependencies{})
	if err != nil {
		t.Fatalf("disabled composition: %v", err)
	}
	if err := loop.Start(context.Background()); err != nil {
		t.Fatalf("disabled start: %v", err)
	}
	if err := loop.Stop(context.Background()); err != nil {
		t.Fatalf("disabled stop: %v", err)
	}
}

func TestSelfWatchdogCompositionRequiresEveryBoundary(t *testing.T) {
	_, err := ComposeSelfWatchdog(bootstrapWatchdogPolicy(), selfwatchdog.Identity{
		OwnerRef: "owner:orquesta", InstanceRef: "process:bootstrap", FencingToken: 1,
	}, SelfWatchdogDependencies{})
	if !selfwatchdog.HasErrorCode(err, selfwatchdog.ErrorInvalidComposition) {
		t.Fatalf("missing dependencies error = %v", err)
	}
}

func TestSelfWatchdogLoopStartsAndStopsIdempotentlyWithoutResidualGoroutine(t *testing.T) {
	ports := newBootstrapWatchdogPorts()
	loop, err := ComposeSelfWatchdog(bootstrapWatchdogPolicy(), selfwatchdog.Identity{
		OwnerRef: "owner:orquesta", InstanceRef: "process:bootstrap", FencingToken: 1,
	}, ports.dependencies())
	if err != nil {
		t.Fatalf("compose: %v", err)
	}
	parent, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := loop.Start(parent); err != nil {
		t.Fatalf("start: %v", err)
	}
	if err := loop.Start(parent); err != nil {
		t.Fatalf("idempotent start: %v", err)
	}
	select {
	case err := <-loop.Errors():
		t.Fatalf("loop reported error: %v", err)
	case <-ports.sampled:
	case <-time.After(time.Second):
		t.Fatal("watchdog loop did not tick")
	}
	stopCtx, stopCancel := context.WithTimeout(context.Background(), time.Second)
	defer stopCancel()
	if err := loop.Stop(stopCtx); err != nil {
		t.Fatalf("stop: %v", err)
	}
	if err := loop.Stop(stopCtx); err != nil {
		t.Fatalf("idempotent stop: %v", err)
	}
	if err := loop.Start(parent); err == nil {
		t.Fatal("stopped one-shot loop reported a false restart")
	}
	select {
	case <-loop.done:
	default:
		t.Fatal("owned watchdog goroutine remains live")
	}
}

func TestSelfWatchdogLoopManualTickUsesFakeTelemetry(t *testing.T) {
	ports := newBootstrapWatchdogPorts()
	loop, err := ComposeSelfWatchdog(bootstrapWatchdogPolicy(), selfwatchdog.Identity{
		OwnerRef: "owner:orquesta", InstanceRef: "process:manual", FencingToken: 1,
	}, ports.dependencies())
	if err != nil {
		t.Fatalf("compose: %v", err)
	}
	outcome, err := loop.Tick(context.Background())
	if err != nil || outcome.Decision != selfwatchdog.DecisionSustainedWindowPending {
		t.Fatalf("manual tick = %+v/%v", outcome, err)
	}
}

func bootstrapWatchdogPolicy() config.SelfWatchdogPolicy {
	return config.SelfWatchdogPolicy{
		Enabled:             true,
		ObservationInterval: time.Millisecond,
		CPUHighPercent:      80,
		SustainedFor:        2 * time.Millisecond,
		NoProgressFor:       3 * time.Millisecond,
	}
}

type bootstrapWatchdogPorts struct {
	mu         sync.Mutex
	now        time.Time
	revision   uint64
	checkpoint selfwatchdog.Checkpoint
	found      bool
	sampled    chan struct{}
}

func newBootstrapWatchdogPorts() *bootstrapWatchdogPorts {
	return &bootstrapWatchdogPorts{
		now:     time.Unix(1_800_001_000, 0).UTC(),
		sampled: make(chan struct{}, 1),
	}
}

func (ports *bootstrapWatchdogPorts) dependencies() SelfWatchdogDependencies {
	return SelfWatchdogDependencies{
		Telemetry: ports,
		Progress:  ports,
		Evidence:  ports,
		Shutdown:  ports,
		State:     ports,
	}
}

func (ports *bootstrapWatchdogPorts) Sample(context.Context) (selfwatchdog.TelemetrySample, error) {
	ports.mu.Lock()
	defer ports.mu.Unlock()
	ports.now = ports.now.Add(time.Millisecond)
	select {
	case ports.sampled <- struct{}{}:
	default:
	}
	return selfwatchdog.TelemetrySample{
		ObservedAt: ports.now,
		CPUPercent: 90,
		Uptime:     time.Hour,
	}, nil
}

func (ports *bootstrapWatchdogPorts) ObserveProgress(context.Context) (selfwatchdog.ProgressSnapshot, error) {
	ports.mu.Lock()
	defer ports.mu.Unlock()
	return selfwatchdog.ProgressSnapshot{ObservedAt: ports.now}, nil
}

func (ports *bootstrapWatchdogPorts) PublishEvidence(
	_ context.Context,
	evidence selfwatchdog.Evidence,
) (selfwatchdog.EvidenceReceipt, error) {
	return selfwatchdog.EvidenceReceipt{
		OwnerRef:       evidence.OwnerRef,
		InstanceRef:    evidence.InstanceRef,
		FencingToken:   evidence.FencingToken,
		EvidenceRef:    "evidence:bootstrap",
		IdempotencyKey: evidence.IdempotencyKey,
	}, nil
}

func (ports *bootstrapWatchdogPorts) AdmitOwnShutdown(
	_ context.Context,
	request selfwatchdog.ShutdownAdmissionRequest,
) (selfwatchdog.ShutdownAdmissionReceipt, error) {
	return selfwatchdog.ShutdownAdmissionReceipt{
		OwnerRef:       request.OwnerRef,
		InstanceRef:    request.InstanceRef,
		FencingToken:   request.FencingToken,
		Mode:           request.Mode,
		IdempotencyKey: request.IdempotencyKey,
	}, nil
}

func (ports *bootstrapWatchdogPorts) LoadCheckpoint(
	context.Context,
	string,
) (selfwatchdog.CheckpointSnapshot, error) {
	ports.mu.Lock()
	defer ports.mu.Unlock()
	return selfwatchdog.CheckpointSnapshot{
		Found:      ports.found,
		Revision:   ports.revision,
		Checkpoint: ports.checkpoint,
	}, nil
}

func (ports *bootstrapWatchdogPorts) StoreCheckpoint(
	_ context.Context,
	write selfwatchdog.CheckpointWrite,
) (selfwatchdog.CheckpointReceipt, error) {
	ports.mu.Lock()
	defer ports.mu.Unlock()
	if write.ExpectedRevision != ports.revision {
		return selfwatchdog.CheckpointReceipt{}, errors.New("bootstrap.test_revision_conflict")
	}
	if ports.found &&
		(write.Checkpoint.FencingToken < ports.checkpoint.FencingToken ||
			(write.Checkpoint.FencingToken == ports.checkpoint.FencingToken &&
				write.Checkpoint.InstanceRef != ports.checkpoint.InstanceRef)) {
		return selfwatchdog.CheckpointReceipt{}, errors.New("bootstrap.test_fence_stale")
	}
	previous := ports.revision
	ports.revision++
	ports.checkpoint = write.Checkpoint
	ports.found = true
	return selfwatchdog.CheckpointReceipt{
		OwnerRef:         write.OwnerRef,
		InstanceRef:      write.Checkpoint.InstanceRef,
		FencingToken:     write.Checkpoint.FencingToken,
		PreviousRevision: previous,
		Revision:         ports.revision,
	}, nil
}
