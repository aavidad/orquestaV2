package bootstrap

import (
	"context"
	"errors"
	"reflect"
	"strconv"
	"sync"
	"testing"
	"time"

	"orquesta/internal/application"
)

func TestSchedulerBoundsLaunchAdmissionsWithoutExpandingLogicalDemand(t *testing.T) {
	const demand = 500
	for _, limit := range []int{1, 5, 10, 16, 20} {
		limit := limit
		t.Run("limite_"+strconv.Itoa(limit), func(t *testing.T) {
			actions := make([]application.ActionKind, demand)
			for index := range actions {
				actions[index] = application.ActionLaunchAgent
			}
			queue := newSchedulerActionQueue(actions...)
			started := make(chan struct{}, limit)
			var mu sync.Mutex
			active, maximum := 0, 0
			process := func(ctx context.Context, claim application.ActionClaim) (application.ProcessResult, error) {
				mu.Lock()
				active++
				if active > maximum {
					maximum = active
				}
				mu.Unlock()
				defer func() {
					mu.Lock()
					active--
					mu.Unlock()
				}()
				select {
				case started <- struct{}{}:
				case <-ctx.Done():
				}
				<-ctx.Done()
				return application.ProcessResult{Processed: true, Action: claim.Action.Kind}, ctx.Err()
			}

			ctx, cancel := context.WithCancel(context.Background())
			t.Cleanup(cancel)
			done := runSchedulerForTest(ctx, scheduler{
				workerRef: "worker:elastic-500", pollInterval: time.Hour,
				maxConcurrentLaunches:  int64(limit),
				claimNextForTesting:    queue.claim,
				processClaimForTesting: process,
			})
			for range limit {
				awaitSchedulerSignal(t, started, "admisión launch")
			}
			for call := 0; call <= limit; call++ {
				selection := awaitSchedulerSignal(t, queue.calls, "reclamo")
				wantExcluded := call == limit
				if selection.ExcludeLaunch != wantExcluded {
					t.Fatalf("selección %d ExcludeLaunch=%t, esperado %t",
						call+1, selection.ExcludeLaunch, wantExcluded)
				}
			}
			assertNoSchedulerSignal(t, queue.calls, "giro activo con capacidad saturada")

			remaining, claimed := queue.stats()
			mu.Lock()
			gotActive, gotMaximum := active, maximum
			mu.Unlock()
			if remaining != demand-limit || claimed != limit ||
				gotActive != limit || gotMaximum != limit {
				t.Fatalf("demanda restante=%d reclamos=%d activos=%d máximo=%d",
					remaining, claimed, gotActive, gotMaximum)
			}

			cancel()
			awaitSchedulerSignal(t, done, "parada del despachador")
			mu.Lock()
			defer mu.Unlock()
			if active != 0 {
				t.Fatalf("admisiones activas tras parada=%d", active)
			}
		})
	}
}

func TestSchedulerPassesTheExactFencedClaimToConcurrentProcessing(t *testing.T) {
	want := application.ActionClaim{
		Action: application.ActionRecord{
			Ref: "action:exact", Kind: application.ActionLaunchAgent,
			ControlRef: "control:exact", ExpectedTargetOID: "oid:exact",
			AvailableAt: time.Unix(1_700_000_000, 123).UTC(),
		},
		Token: "claim:exact", WorkerRef: "worker:exact",
		DeliveryAttempt: 7, Fence: 11,
	}
	var mu sync.Mutex
	claimed := false
	received := make(chan application.ActionClaim, 1)
	claimNext := func(
		ctx context.Context,
		_ string,
		_ application.ActionClaimSelection,
	) (application.ActionClaim, bool, error) {
		mu.Lock()
		defer mu.Unlock()
		if claimed {
			return application.ActionClaim{}, false, nil
		}
		claimed = true
		return want, true, nil
	}
	process := func(ctx context.Context, claim application.ActionClaim) (application.ProcessResult, error) {
		select {
		case received <- claim:
		case <-ctx.Done():
		}
		<-ctx.Done()
		return application.ProcessResult{Processed: true, Action: claim.Action.Kind}, ctx.Err()
	}

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	done := runSchedulerForTest(ctx, scheduler{
		workerRef: "worker:dispatcher", pollInterval: time.Hour, maxConcurrentLaunches: 1,
		claimNextForTesting: claimNext, processClaimForTesting: process,
	})
	if got := awaitSchedulerSignal(t, received, "claim exacto"); !reflect.DeepEqual(got, want) {
		t.Fatalf("claim procesado=%+v, esperado %+v", got, want)
	}
	cancel()
	awaitSchedulerSignal(t, done, "parada del despachador")
}

func TestSchedulerKeepsStopAndObserveMovingWhileLaunchesAreSaturated(t *testing.T) {
	queue := newSchedulerActionQueue(
		application.ActionLaunchAgent,
		application.ActionLaunchAgent,
		application.ActionStopAgent,
		application.ActionObserveAgent,
	)
	launchStarted := make(chan struct{}, 2)
	launchesReady := make(chan struct{})
	stopStarted := make(chan struct{}, 1)
	observeStarted := make(chan struct{}, 1)
	releaseStop := make(chan struct{})
	var mu sync.Mutex
	activeLaunches, activeNonLaunch, maximumNonLaunch := 0, 0, 0
	process := func(ctx context.Context, claim application.ActionClaim) (application.ProcessResult, error) {
		result := application.ProcessResult{Processed: true, Action: claim.Action.Kind}
		if claim.Action.Kind == application.ActionLaunchAgent {
			mu.Lock()
			activeLaunches++
			if activeLaunches == 2 {
				close(launchesReady)
			}
			mu.Unlock()
			select {
			case launchStarted <- struct{}{}:
			case <-ctx.Done():
			}
			<-ctx.Done()
			mu.Lock()
			activeLaunches--
			mu.Unlock()
			return result, ctx.Err()
		}

		mu.Lock()
		activeNonLaunch++
		if activeNonLaunch > maximumNonLaunch {
			maximumNonLaunch = activeNonLaunch
		}
		mu.Unlock()
		defer func() {
			mu.Lock()
			activeNonLaunch--
			mu.Unlock()
		}()
		select {
		case <-launchesReady:
		case <-ctx.Done():
			return result, ctx.Err()
		}
		switch claim.Action.Kind {
		case application.ActionStopAgent:
			select {
			case stopStarted <- struct{}{}:
			case <-ctx.Done():
				return result, ctx.Err()
			}
			select {
			case <-ctx.Done():
				return result, ctx.Err()
			case <-releaseStop:
			}
		case application.ActionObserveAgent:
			select {
			case observeStarted <- struct{}{}:
			case <-ctx.Done():
				return result, ctx.Err()
			}
		}
		return result, nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	done := runSchedulerForTest(ctx, scheduler{
		workerRef: "worker:control-progress", pollInterval: time.Hour, maxConcurrentLaunches: 2,
		claimNextForTesting: queue.claim, processClaimForTesting: process,
	})
	for range 2 {
		awaitSchedulerSignal(t, launchStarted, "launch saturado")
	}
	awaitSchedulerSignal(t, stopStarted, "parada prioritaria")
	assertNoSchedulerSignal(t, observeStarted, "solapamiento de acciones no launch")
	close(releaseStop)
	awaitSchedulerSignal(t, observeStarted, "observación con launch saturado")

	cancel()
	awaitSchedulerSignal(t, done, "parada del despachador")
	mu.Lock()
	defer mu.Unlock()
	if activeLaunches != 0 || activeNonLaunch != 0 || maximumNonLaunch != 1 {
		t.Fatalf("launch activos=%d no launch activos=%d máximo no launch=%d",
			activeLaunches, activeNonLaunch, maximumNonLaunch)
	}
}

func TestSchedulerReportsAsynchronousLaunchError(t *testing.T) {
	queue := newSchedulerActionQueue(application.ActionLaunchAgent)
	want := errors.New("test.launch_failed")
	reported := make(chan error, 1)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	done := runSchedulerForTest(ctx, scheduler{
		workerRef: "worker:report", pollInterval: time.Hour, maxConcurrentLaunches: 1,
		claimNextForTesting: queue.claim,
		processClaimForTesting: func(
			context.Context,
			application.ActionClaim,
		) (application.ProcessResult, error) {
			return application.ProcessResult{Processed: true, Action: application.ActionLaunchAgent}, want
		},
		report: func(err error) { reported <- err },
	})
	if got := awaitSchedulerSignal(t, reported, "informe de error asíncrono"); !errors.Is(got, want) {
		t.Fatalf("error informado=%v, esperado %v", got, want)
	}
	cancel()
	awaitSchedulerSignal(t, done, "parada del despachador")
}

func TestSchedulerCancellationWaitsForOwnedLaunchCleanupWithoutReportingNoise(t *testing.T) {
	queue := newSchedulerActionQueue(application.ActionLaunchAgent)
	launchStarted := make(chan struct{}, 1)
	cancellationSeen := make(chan struct{})
	allowCleanup := make(chan struct{})
	var allowCleanupOnce sync.Once
	releaseCleanup := func() { allowCleanupOnce.Do(func() { close(allowCleanup) }) }
	reported := make(chan error, 1)
	process := func(ctx context.Context, claim application.ActionClaim) (application.ProcessResult, error) {
		select {
		case launchStarted <- struct{}{}:
		case <-ctx.Done():
		}
		<-ctx.Done()
		close(cancellationSeen)
		<-allowCleanup
		return application.ProcessResult{Processed: true, Action: claim.Action.Kind}, ctx.Err()
	}

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	t.Cleanup(releaseCleanup)
	done := runSchedulerForTest(ctx, scheduler{
		workerRef: "worker:clean-shutdown", pollInterval: time.Hour, maxConcurrentLaunches: 1,
		claimNextForTesting: queue.claim, processClaimForTesting: process,
		report: func(err error) { reported <- err },
	})
	awaitSchedulerSignal(t, launchStarted, "inicio de launch")
	cancel()
	awaitSchedulerSignal(t, cancellationSeen, "cancelación de launch")
	assertNoSchedulerSignal(t, done, "retorno antes de limpiar launch")
	releaseCleanup()
	awaitSchedulerSignal(t, done, "limpieza de launch")
	select {
	case err := <-reported:
		t.Fatalf("la cancelación produjo ruido: %v", err)
	default:
	}
}

type schedulerActionQueue struct {
	mu      sync.Mutex
	actions []application.ActionKind
	claimed int
	calls   chan application.ActionClaimSelection
}

func newSchedulerActionQueue(actions ...application.ActionKind) *schedulerActionQueue {
	return &schedulerActionQueue{
		actions: append([]application.ActionKind(nil), actions...),
		calls:   make(chan application.ActionClaimSelection, len(actions)+16),
	}
}

func (queue *schedulerActionQueue) claim(
	ctx context.Context,
	_ string,
	selection application.ActionClaimSelection,
) (application.ActionClaim, bool, error) {
	select {
	case queue.calls <- selection:
	case <-ctx.Done():
		return application.ActionClaim{}, false, ctx.Err()
	}
	queue.mu.Lock()
	defer queue.mu.Unlock()
	for index, kind := range queue.actions {
		if selection.ExcludeLaunch && kind == application.ActionLaunchAgent {
			continue
		}
		queue.actions = append(queue.actions[:index], queue.actions[index+1:]...)
		queue.claimed++
		return application.ActionClaim{Action: application.ActionRecord{Kind: kind}}, true, nil
	}
	return application.ActionClaim{}, false, nil
}

func (queue *schedulerActionQueue) stats() (remaining int, claimed int) {
	queue.mu.Lock()
	defer queue.mu.Unlock()
	return len(queue.actions), queue.claimed
}

func runSchedulerForTest(ctx context.Context, candidate scheduler) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		candidate.run(ctx)
	}()
	return done
}

func awaitSchedulerSignal[T any](t *testing.T, channel <-chan T, subject string) T {
	t.Helper()
	select {
	case value := <-channel:
		return value
	case <-time.After(time.Second):
		t.Fatalf("tiempo agotado esperando %s", subject)
		var zero T
		return zero
	}
}

func assertNoSchedulerSignal[T any](t *testing.T, channel <-chan T, subject string) {
	t.Helper()
	select {
	case <-channel:
		t.Fatalf("se observó %s", subject)
	case <-time.After(25 * time.Millisecond):
	}
}
