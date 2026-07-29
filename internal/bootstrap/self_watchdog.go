package bootstrap

import (
	"context"
	"errors"
	"sync"
	"time"

	"orquesta/internal/adapters/system/selfwatchdog"
	"orquesta/internal/config"
)

type SelfWatchdogDependencies struct {
	Telemetry selfwatchdog.TelemetryPort
	Progress  selfwatchdog.ProgressPort
	Evidence  selfwatchdog.EvidencePort
	Shutdown  selfwatchdog.ShutdownPort
	State     selfwatchdog.CheckpointPort
}

// SelfWatchdogLoop owns only its timer goroutine. Runtime lifecycle and process
// ownership remain behind the injected shutdown port.
type SelfWatchdogLoop struct {
	disabled bool
	interval time.Duration
	watchdog *selfwatchdog.Watchdog

	mu      sync.Mutex
	started bool
	stopped bool
	cancel  context.CancelFunc
	done    chan struct{}
	errors  chan error
}

func ComposeSelfWatchdog(
	policy config.SelfWatchdogPolicy,
	identity selfwatchdog.Identity,
	dependencies SelfWatchdogDependencies,
) (*SelfWatchdogLoop, error) {
	if err := policy.Validate(); err != nil {
		return nil, err
	}
	if !policy.Enabled {
		errors := make(chan error)
		close(errors)
		return &SelfWatchdogLoop{disabled: true, errors: errors}, nil
	}
	watchdog, err := selfwatchdog.New(selfwatchdog.Policy{
		ObservationInterval: policy.ObservationInterval,
		CPUHighPercent:      policy.CPUHighPercent,
		SustainedFor:        policy.SustainedFor,
		NoProgressFor:       policy.NoProgressFor,
	}, identity, selfwatchdog.Ports{
		Telemetry: dependencies.Telemetry,
		Progress:  dependencies.Progress,
		Evidence:  dependencies.Evidence,
		Shutdown:  dependencies.Shutdown,
		State:     dependencies.State,
	})
	if err != nil {
		return nil, err
	}
	return &SelfWatchdogLoop{
		interval: policy.ObservationInterval,
		watchdog: watchdog,
		errors:   make(chan error, 1),
	}, nil
}

// Tick is the deterministic integration seam used by startup/restart checks.
func (loop *SelfWatchdogLoop) Tick(ctx context.Context) (selfwatchdog.Outcome, error) {
	if loop == nil || ctx == nil {
		return selfwatchdog.Outcome{}, errors.New("bootstrap.self_watchdog_context_required")
	}
	if loop.disabled {
		return selfwatchdog.Outcome{}, nil
	}
	return loop.watchdog.Tick(ctx)
}

// Start is idempotent and starts at most one observation loop.
func (loop *SelfWatchdogLoop) Start(parent context.Context) error {
	if loop == nil || parent == nil {
		return errors.New("bootstrap.self_watchdog_context_required")
	}
	if loop.disabled {
		return nil
	}
	loop.mu.Lock()
	defer loop.mu.Unlock()
	if loop.started {
		if loop.stopped {
			return errors.New("bootstrap.self_watchdog_stopped")
		}
		return nil
	}
	ctx, cancel := context.WithCancel(parent)
	loop.started = true
	loop.cancel = cancel
	loop.done = make(chan struct{})
	go loop.run(ctx, loop.done)
	return nil
}

// Errors exposes bounded diagnostics without allowing an observer to block the
// timer goroutine. The channel closes when the loop exits.
func (loop *SelfWatchdogLoop) Errors() <-chan error {
	if loop == nil {
		errors := make(chan error)
		close(errors)
		return errors
	}
	return loop.errors
}

// Stop is idempotent and waits until the owned timer goroutine has exited.
func (loop *SelfWatchdogLoop) Stop(ctx context.Context) error {
	if loop == nil {
		return nil
	}
	if ctx == nil {
		return errors.New("bootstrap.self_watchdog_context_required")
	}
	if loop.disabled {
		return nil
	}
	loop.mu.Lock()
	if !loop.started {
		loop.mu.Unlock()
		return nil
	}
	cancel := loop.cancel
	done := loop.done
	loop.mu.Unlock()
	cancel()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		return nil
	}
}

func (loop *SelfWatchdogLoop) run(ctx context.Context, done chan<- struct{}) {
	defer func() {
		loop.mu.Lock()
		loop.stopped = true
		loop.mu.Unlock()
		close(loop.errors)
		close(done)
	}()
	ticker := time.NewTicker(loop.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := loop.watchdog.Tick(ctx); err != nil && !errors.Is(err, context.Canceled) {
				select {
				case loop.errors <- err:
				default:
				}
			}
		}
	}
}
