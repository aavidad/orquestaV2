package orquestaserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"
)

const serverHandoffRoutePathV0 = "/api/v0/server/handoff"

type serverHandoffHTTPProjectionV0 struct {
	Estado              string `json:"estado"`
	Status              string `json:"status"`
	HandoffReady        bool   `json:"handoff_ready"`
	AgentsPreserved     bool   `json:"agents_preserved"`
	NextStartupModeHint string `json:"next_startup_mode_hint,omitempty"`
}

func (runtime *RuntimeV0) handoffHTTPHandlerV0(next http.Handler) http.Handler {
	if next == nil {
		next = http.NotFoundHandler()
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r == nil || r.URL == nil || r.URL.Path != serverHandoffRoutePathV0 {
			next.ServeHTTP(w, r)
			return
		}
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, "method_not_allowed", http.StatusMethodNotAllowed)
			return
		}
		runtime.requestHandoffV0(r.Context(), "server_handoff_http_request")
		writeJSONResponseV0(w, http.StatusOK, serverHandoffHTTPProjectionV0{
			Estado:              "ok",
			Status:              "handoff_requested",
			HandoffReady:        true,
			AgentsPreserved:     true,
			NextStartupModeHint: "diagnose",
		})
	})
}

func (runtime *RuntimeV0) requestHandoffV0(ctx context.Context, reason string) {
	if runtime == nil {
		return
	}
	runtime.handoffOnce.Do(func() {
		atomic.StoreInt32(&runtime.shutdownInProgress, 1)
		runtime.auditEventV0(ctx, "server_handoff_requested", "active", "", map[string]interface{}{"reason": reason})
		runtime.persistStateTransitionV0(
			ctx,
			runtime.tracker.MarkHandoffRequestedV0(runtime.clock.Now()),
			"handoff_requested",
		)
		close(runtime.handoffRequested)
	})
}

func (runtime *RuntimeV0) handoffRuntimeV0(server *http.Server, cancelRun context.CancelFunc) error {
	shutdownCtx, cancelShutdown := runtime.shutdownContextV0()
	defer cancelShutdown()
	atomic.StoreInt32(&runtime.shutdownInProgress, 1)
	runtime.persistStateTransitionV0(
		shutdownCtx,
		runtime.tracker.MarkRuntimeStoppingV0("handoff_draining_server", runtime.asyncWorkActiveV0(), runtime.clock.Now()),
		"handoff_draining_server",
	)
	if cancelRun != nil {
		cancelRun()
	}
	if err := server.Shutdown(shutdownCtx); err != nil {
		_ = server.Close()
		runtime.withAsyncReceiptContextV0(func(receiptCtx context.Context) {
			runtime.persistStateTransitionV0(
				receiptCtx,
				runtime.tracker.MarkRuntimeStopTimeoutV0("handoff_http_shutdown_timeout", runtime.asyncWorkActiveV0(), runtime.clock.Now()),
				"handoff_stop_timeout",
			)
		})
		return fmt.Errorf("orquesta_server: handoff_timeout")
	}
	if !runtime.waitAsyncWorkV0(shutdownCtx) {
		runtime.withAsyncReceiptContextV0(func(receiptCtx context.Context) {
			runtime.persistStateTransitionV0(
				receiptCtx,
				runtime.tracker.MarkRuntimeStopTimeoutV0("handoff_async_work_timeout", runtime.asyncWorkActiveV0(), runtime.clock.Now()),
				"handoff_stop_timeout",
			)
		})
		return fmt.Errorf("orquesta_server: handoff_async_work_timeout")
	}
	runtime.withAsyncReceiptContextV0(func(receiptCtx context.Context) {
		runtime.persistStateTransitionV0(
			receiptCtx,
			runtime.tracker.MarkHandoffReadyV0(runtime.clock.Now()),
			"handoff_ready",
		)
	})
	return nil
}

func (tracker *StatusTrackerV0) MarkHandoffRequestedV0(now time.Time) StateV0 {
	return tracker.updateV0(func(state *StateV0) {
		state.Status = "handoff_requested"
		state.LastHeartbeatAt = formatTimeV0(now)
		state.LastShutdownAt = formatTimeV0(now)
		state.ShutdownInProgress = true
		state.ShutdownStatus = "handoff_requested"
		state.ShutdownReady = false
		state.SupervisorFrozen = true
		state.SupervisorTickActive = false
		state.GoalObserverTickActive = false
	})
}

func (tracker *StatusTrackerV0) MarkHandoffReadyV0(now time.Time) StateV0 {
	return tracker.updateV0(func(state *StateV0) {
		state.Status = "handoff_ready"
		state.LastHeartbeatAt = formatTimeV0(now)
		state.LastShutdownAt = formatTimeV0(now)
		state.ShutdownInProgress = false
		state.ShutdownStatus = "handoff_ready"
		state.ShutdownReady = true
		state.ShutdownAsyncWorkActive = 0
		state.SupervisorFrozen = false
		state.SupervisorTickActive = false
		state.GoalObserverTickActive = false
	})
}

func writeJSONResponseV0(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
