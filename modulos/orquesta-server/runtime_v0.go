package orquestaserver

import (
	"context"
	"errors"
	"net"
	"net/http"
	"sync"

	orquestagoal "orquesta/modulos/orquesta-goal"
)

type RuntimeDepsV0 struct {
	AppHandler       http.Handler
	Supervisor       SupervisorPortV0
	ResidentDirector ResidentDirectorPortV0
	GoalStateStore   orquestagoal.GoalWorkStateStorePortV0
	StateStore       StateStorePortV0
	AuditSink        AuditSinkPortV0
	StartupCheck     StartupCheckPortV0
	SelfWatchdog     SelfWatchdogObservationPortV0
	Clock            ClockPortV0
}

type RuntimeV0 struct {
	config                      ConfigV0
	appHandler                  http.Handler
	supervisor                  SupervisorPortV0
	residentDirector            ResidentDirectorPortV0
	goalStateStore              orquestagoal.GoalWorkStateStorePortV0
	asyncWork                   runtimeAsyncWorkGroupV0
	supervisorTickActive        int32
	supervisorTickPending       int32
	supervisorWakeups           chan SupervisorWakeupV0
	residentDirectorPaused      int32
	residentDirectorTickActive  int32
	residentDirectorTickPending int32
	residentDirectorWakeups     chan ResidentDirectorWakeupV0
	shutdownInProgress          int32
	stateStore                  StateStorePortV0
	auditSink                   AuditSinkPortV0
	startupCheck                StartupCheckPortV0
	selfWatchdog                SelfWatchdogObservationPortV0
	clock                       ClockPortV0
	tracker                     *StatusTrackerV0
	handoffRequested            chan struct{}
	handoffOnce                 sync.Once
}

func NewRuntimeV0(config ConfigV0, deps RuntimeDepsV0) (*RuntimeV0, error) {
	config = NormalizeConfigV0(config)
	if err := ValidateConfigV0(config); err != nil {
		return nil, err
	}
	if deps.Clock == nil {
		deps.Clock = SystemClockV0{}
	}
	if deps.StateStore == nil {
		store, err := NewFileStateStoreV0(StatePathV0(config))
		if err != nil {
			return nil, err
		}
		deps.StateStore = store
	}
	if deps.AuditSink == nil && !config.AuditDisabled {
		sink, err := NewFileAuditSinkV0(AuditPathV0(config))
		if err != nil {
			return nil, err
		}
		deps.AuditSink = sink
	}
	tracker := NewStatusTrackerV0(config, deps.Clock.Now())
	if restored, ok := restoreStatusTrackerFromStoreV0(context.Background(), config, deps.StateStore, deps.Clock.Now()); ok {
		tracker = restored
	}
	return &RuntimeV0{
		config:            config,
		appHandler:        deps.AppHandler,
		supervisor:        deps.Supervisor,
		residentDirector:  deps.ResidentDirector,
		goalStateStore:    deps.GoalStateStore,
		stateStore:        deps.StateStore,
		auditSink:         deps.AuditSink,
		startupCheck:      deps.StartupCheck,
		selfWatchdog:      deps.SelfWatchdog,
		clock:             deps.Clock,
		tracker:           tracker,
		supervisorWakeups: make(chan SupervisorWakeupV0, 1),
		residentDirectorWakeups: make(
			chan ResidentDirectorWakeupV0,
			1,
		),
		handoffRequested: make(chan struct{}),
	}, nil
}

func (runtime *RuntimeV0) HandlerV0() http.Handler {
	handler := NewHandlerV0(HandlerConfigV0{
		AppHandler: runtime.serverLifecycleHTTPHandlerV0(runtime.appHandler),
		Tracker:    runtime.tracker,
	})
	return runtime.auditHTTPHandlerV0(runtime.controlPlaneGuardHTTPHandlerV0(handler))
}

func (runtime *RuntimeV0) RunV0(ctx context.Context) error {
	return runtime.RunWithShutdownCauseV0(ctx, nil)
}

func (runtime *RuntimeV0) RunWithShutdownCauseV0(
	ctx context.Context,
	cause func() ShutdownSignalCauseV0,
) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := runtime.prepareStartupV0(ctx); err != nil {
		return err
	}
	listener, err := net.Listen("tcp", runtime.config.Addr)
	if err != nil {
		return err
	}
	server := runtime.httpServerV0()
	addr := listener.Addr().String()
	if err := runtime.saveStateV0(ctx, runtime.tracker.MarkServingV0(addr, runtime.clock.Now())); err != nil {
		_ = listener.Close()
		return err
	}

	runCtx, cancelRun := context.WithCancel(ctx)
	defer cancelRun()
	serverDone := make(chan error, 1)
	selfWatchdogStop := make(chan SelfWatchdogDecisionV0, 1)
	runtime.runAsyncWorkV0("http_serve", func() { serverDone <- server.Serve(listener) })
	runtime.runAsyncWorkV0("supervisor_loop", func() { runtime.runSupervisorLoopV0(runCtx) })
	if runtime.residentDirector != nil && runtime.config.ResidentDirectorEnabled {
		runtime.runAsyncWorkV0("resident_director_loop", func() {
			runtime.runResidentDirectorLoopV0(runCtx)
		})
	}
	if runtime.selfWatchdog != nil && !runtime.config.SelfWatchdog.Disabled {
		runtime.runAsyncWorkV0("self_watchdog_loop", func() {
			runtime.runSelfWatchdogLoopV0(runCtx, selfWatchdogStop)
		})
	}

	select {
	case <-ctx.Done():
		return runtime.shutdownRuntimeV0(server, cancelRun, shutdownCauseFromFuncV0(cause))
	case <-runtime.handoffRequested:
		return runtime.handoffRuntimeV0(server, cancelRun)
	case decision := <-selfWatchdogStop:
		return runtime.shutdownRuntimeV0(server, cancelRun, selfWatchdogShutdownCauseV0(decision))
	case err := <-serverDone:
		if errors.Is(err, http.ErrServerClosed) {
			return runtime.stopRuntimeAfterServeClosedV0(cancelRun)
		}
		runtime.persistStateTransitionV0(context.Background(), runtime.tracker.MarkErrorV0(err.Error(), runtime.clock.Now()), "server_error")
		return err
	}
}

func shutdownCauseFromFuncV0(cause func() ShutdownSignalCauseV0) ShutdownSignalCauseV0 {
	if cause == nil {
		return ShutdownSignalCauseV0{}
	}
	out := cause()
	if out.SignalName == "" {
		return ShutdownSignalCauseV0{}
	}
	return normalizeShutdownSignalCauseV0(out)
}

func (runtime *RuntimeV0) RecordShutdownSignalV0(cause ShutdownSignalCauseV0) {
	if runtime == nil || runtime.tracker == nil {
		return
	}
	runtime.persistStateTransitionV0(
		context.Background(),
		runtime.tracker.MarkRuntimeStoppingBySignalV0(cause, runtime.asyncWorkActiveV0(), runtime.clock.Now()),
		"runtime_stopping_by_signal",
	)
}

func (runtime *RuntimeV0) StateV0() StateV0 {
	return runtime.tracker.SnapshotV0()
}

func (runtime *RuntimeV0) persistStateV0(ctx context.Context, state StateV0) {
	runtime.persistStateTransitionV0(ctx, state, "state_update")
}

func (runtime *RuntimeV0) persistStateTransitionV0(ctx context.Context, state StateV0, transition string) {
	if err := runtime.saveStateV0(ctx, state); err != nil && runtime.tracker != nil {
		runtime.tracker.MarkStatePersistFailedV0(transition, runtime.clock.Now())
	} else if err == nil && runtime.tracker != nil {
		runtime.tracker.MarkStatePersistConfirmedV0(transition, runtime.clock.Now())
	}
}

func (runtime *RuntimeV0) saveStateV0(ctx context.Context, state StateV0) error {
	if runtime.stateStore == nil {
		return nil
	}
	return runtime.stateStore.SaveServerStateV0(ctx, state)
}
