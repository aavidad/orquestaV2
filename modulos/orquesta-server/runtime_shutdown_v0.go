package orquestaserver

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
)

func (runtime *RuntimeV0) shutdownRuntimeV0(
	server *http.Server,
	cancelRun context.CancelFunc,
	cause ShutdownSignalCauseV0,
) error {
	defer runtime.cancelShutdownReadyForceExitV0()
	shutdownCtx, cancelShutdown := runtime.shutdownContextV0()
	defer cancelShutdown()
	atomic.StoreInt32(&runtime.shutdownInProgress, 1)
	initialState := runtime.tracker.MarkRuntimeStoppingV0(
		"async_work_draining",
		runtime.asyncWorkActiveV0(),
		runtime.clock.Now(),
	)
	if strings.TrimSpace(cause.SignalName) != "" {
		initialState = runtime.tracker.MarkRuntimeStoppingBySignalV0(
			cause,
			runtime.asyncWorkActiveV0(),
			runtime.clock.Now(),
		)
		runtime.auditEventV0(shutdownCtx, "server_shutdown_signal", "stopping_by_signal", "", map[string]interface{}{
			"signal":    cause.SignalName,
			"count":     cause.Count,
			"escalated": cause.Escalated,
		})
	}
	runtime.persistStateTransitionV0(
		shutdownCtx,
		initialState,
		"runtime_stopping",
	)
	if cancelRun != nil {
		cancelRun()
	}
	if err := server.Shutdown(shutdownCtx); err != nil {
		_ = server.Close()
		runtime.withAsyncReceiptContextV0(func(receiptCtx context.Context) {
			runtime.persistStateTransitionV0(
				receiptCtx,
				runtime.tracker.MarkRuntimeStopTimeoutV0("http_shutdown_timeout", runtime.asyncWorkActiveV0(), runtime.clock.Now()),
				"runtime_stop_timeout",
			)
		})
		return fmt.Errorf("orquesta_server: shutdown_timeout")
	}
	runtime.persistStateTransitionV0(
		shutdownCtx,
		runtime.runtimeShutdownDrainingStateV0(cause),
		"runtime_async_work_draining",
	)
	if !runtime.waitAsyncWorkV0(shutdownCtx) {
		runtime.withAsyncReceiptContextV0(func(receiptCtx context.Context) {
			runtime.persistStateTransitionV0(
				receiptCtx,
				runtime.tracker.MarkRuntimeStopTimeoutV0("async_work_timeout", runtime.asyncWorkActiveV0(), runtime.clock.Now()),
				"runtime_stop_timeout",
			)
		})
		return fmt.Errorf("orquesta_server: async_work_timeout")
	}
	runtime.runShutdownHooksV0(shutdownCtx)
	runtime.withAsyncReceiptContextV0(func(receiptCtx context.Context) {
		runtime.persistStateTransitionV0(
			receiptCtx,
			runtime.tracker.MarkRuntimeStoppedV0(runtime.clock.Now()),
			"runtime_stopped",
		)
	})
	return nil
}

func (runtime *RuntimeV0) runtimeShutdownDrainingStateV0(cause ShutdownSignalCauseV0) StateV0 {
	if strings.TrimSpace(cause.SignalName) != "" {
		return runtime.tracker.MarkRuntimeStoppingBySignalV0(
			cause,
			runtime.asyncWorkActiveV0(),
			runtime.clock.Now(),
		)
	}
	return runtime.tracker.MarkRuntimeStoppingV0(
		"async_work_draining",
		runtime.asyncWorkActiveV0(),
		runtime.clock.Now(),
	)
}

func (runtime *RuntimeV0) stopRuntimeAfterServeClosedV0(cancelRun context.CancelFunc) error {
	shutdownCtx, cancelShutdown := runtime.shutdownContextV0()
	defer cancelShutdown()
	if cancelRun != nil {
		cancelRun()
	}
	atomic.StoreInt32(&runtime.shutdownInProgress, 1)
	runtime.persistStateTransitionV0(
		shutdownCtx,
		runtime.tracker.MarkRuntimeStoppingV0("async_work_draining", runtime.asyncWorkActiveV0(), runtime.clock.Now()),
		"runtime_async_work_draining",
	)
	if !runtime.waitAsyncWorkV0(shutdownCtx) {
		runtime.withAsyncReceiptContextV0(func(receiptCtx context.Context) {
			runtime.persistStateTransitionV0(
				receiptCtx,
				runtime.tracker.MarkRuntimeStopTimeoutV0("async_work_timeout", runtime.asyncWorkActiveV0(), runtime.clock.Now()),
				"runtime_stop_timeout",
			)
		})
		return fmt.Errorf("orquesta_server: async_work_timeout")
	}
	runtime.runShutdownHooksV0(shutdownCtx)
	runtime.withAsyncReceiptContextV0(func(receiptCtx context.Context) {
		runtime.persistStateTransitionV0(
			receiptCtx,
			runtime.tracker.MarkRuntimeStoppedV0(runtime.clock.Now()),
			"runtime_stopped",
		)
	})
	return nil
}

func (runtime *RuntimeV0) shutdownContextV0() (context.Context, context.CancelFunc) {
	timeout := runtime.config.ShutdownGracePeriod
	if timeout <= 0 {
		timeout = DefaultShutdownGracePeriodV0
	}
	return context.WithTimeout(context.Background(), timeout)
}

func (runtime *RuntimeV0) withAsyncReceiptContextV0(fn func(context.Context)) {
	if fn == nil {
		return
	}
	ctx, cancel := runtime.shutdownContextV0()
	defer cancel()
	fn(ctx)
}

func (runtime *RuntimeV0) runShutdownHooksV0(ctx context.Context) {
	for _, hook := range runtime.shutdownHooks {
		if hook == nil {
			continue
		}
		if err := hook.ShutdownV0(ctx); err != nil {
			runtime.auditEventV0(ctx, "runtime_shutdown_hook", "failed", "", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}
}
