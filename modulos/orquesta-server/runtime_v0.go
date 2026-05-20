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
	StartupCheck StartupCheckPortV0
	Clock        ClockPortV0
}

type RuntimeV0 struct {
	config       ConfigV0
	appHandler   http.Handler
	supervisor   SupervisorPortV0
	stateStore   StateStorePortV0
	startupCheck StartupCheckPortV0
	clock        ClockPortV0
	tracker      *StatusTrackerV0
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
	return &RuntimeV0{
		config:       config,
		appHandler:   deps.AppHandler,
		supervisor:   deps.Supervisor,
		stateStore:   deps.StateStore,
		startupCheck: deps.StartupCheck,
		clock:        deps.Clock,
		tracker:      NewStatusTrackerV0(config, deps.Clock.Now()),
	}, nil
}

func (runtime *RuntimeV0) HandlerV0() http.Handler {
	return NewHandlerV0(HandlerConfigV0{
		AppHandler: runtime.appHandler,
		Tracker:    runtime.tracker,
	})
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
		runtime.persistStateV0(context.Background(), runtime.tracker.MarkStoppedV0(runtime.clock.Now()))
		return nil
	case err := <-serverDone:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		runtime.persistStateV0(context.Background(), runtime.tracker.MarkErrorV0(err.Error(), runtime.clock.Now()))
		return err
	}
}

func (runtime *RuntimeV0) StateV0() StateV0 {
	return runtime.tracker.SnapshotV0()
}

func (runtime *RuntimeV0) persistStateV0(ctx context.Context, state StateV0) {
	_ = runtime.saveStateV0(ctx, state)
}

func (runtime *RuntimeV0) saveStateV0(ctx context.Context, state StateV0) error {
	if runtime.stateStore == nil {
		return nil
	}
	return runtime.stateStore.SaveServerStateV0(ctx, state)
}
