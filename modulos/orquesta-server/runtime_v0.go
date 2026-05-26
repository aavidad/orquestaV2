package orquestaserver

import (
	"context"
	"errors"
	"net"
	"net/http"
)

type RuntimeDepsV0 struct {
	AppHandler   http.Handler
	Supervisor   SupervisorPortV0
	StateStore   StateStorePortV0
	AuditSink    AuditSinkPortV0
	StartupCheck StartupCheckPortV0
	Clock        ClockPortV0
}

type RuntimeV0 struct {
	config                ConfigV0
	appHandler            http.Handler
	supervisor            SupervisorPortV0
	supervisorTickActive  int32
	supervisorTickPending int32
	shutdownInProgress    int32
	stateStore            StateStorePortV0
	auditSink             AuditSinkPortV0
	startupCheck          StartupCheckPortV0
	clock                 ClockPortV0
	tracker               *StatusTrackerV0
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
	return &RuntimeV0{
		config:       config,
		appHandler:   deps.AppHandler,
		supervisor:   deps.Supervisor,
		stateStore:   deps.StateStore,
		auditSink:    deps.AuditSink,
		startupCheck: deps.StartupCheck,
		clock:        deps.Clock,
		tracker:      NewStatusTrackerV0(config, deps.Clock.Now()),
	}, nil
}

func (runtime *RuntimeV0) HandlerV0() http.Handler {
	handler := NewHandlerV0(HandlerConfigV0{
		AppHandler: runtime.shutdownFreezeHTTPHandlerV0(runtime.appHandler),
		Tracker:    runtime.tracker,
	})
	return runtime.auditHTTPHandlerV0(runtime.controlPlaneGuardHTTPHandlerV0(handler))
}

func (runtime *RuntimeV0) RunV0(ctx context.Context) error {
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
	server := &http.Server{Handler: runtime.HandlerV0()}
	addr := listener.Addr().String()
	if err := runtime.saveStateV0(ctx, runtime.tracker.MarkServingV0(addr, runtime.clock.Now())); err != nil {
		_ = listener.Close()
		return err
	}

	serverDone := make(chan error, 1)
	go func() { serverDone <- server.Serve(listener) }()
	go runtime.runSupervisorLoopV0(ctx)

	select {
	case <-ctx.Done():
		_ = server.Shutdown(context.Background())
		runtime.persistStateTransitionV0(context.Background(), runtime.tracker.MarkStoppedV0(runtime.clock.Now()), "stopped")
		return nil
	case err := <-serverDone:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		runtime.persistStateTransitionV0(context.Background(), runtime.tracker.MarkErrorV0(err.Error(), runtime.clock.Now()), "server_error")
		return err
	}
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
