package orquestaserver

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync/atomic"
)

const serverShutdownRoutePathV0 = "/api/v0/server/shutdown"

type serverShutdownHTTPProjectionV0 struct {
	Estado             string `json:"estado,omitempty"`
	Status             string `json:"status,omitempty"`
	ShutdownReady      bool   `json:"shutdown_ready,omitempty"`
	RunsRequested      int    `json:"runs_requested,omitempty"`
	RunsStopped        int    `json:"runs_stopped,omitempty"`
	AgentsInFlight     int    `json:"agents_in_flight,omitempty"`
	CheckpointsPending int    `json:"checkpoints_pending,omitempty"`
}

func (runtime *RuntimeV0) shutdownFreezeHTTPHandlerV0(next http.Handler) http.Handler {
	if next == nil {
		return nil
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r == nil || r.URL == nil ||
			r.URL.Path != serverShutdownRoutePathV0 ||
			r.Method != http.MethodPost {
			next.ServeHTTP(w, r)
			return
		}
		runtime.freezeSupervisorForShutdownV0(r.Context(), "server_shutdown_http_request")
		capture := &shutdownFreezeResponseCaptureV0{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(capture, r)
		runtime.recordShutdownHTTPResultV0(r.Context(), capture.statusCode, capture.body)
	})
}

func (runtime *RuntimeV0) serverLifecycleHTTPHandlerV0(next http.Handler) http.Handler {
	return runtime.handoffHTTPHandlerV0(runtime.shutdownFreezeHTTPHandlerV0(next))
}

func (runtime *RuntimeV0) freezeSupervisorForShutdownV0(ctx context.Context, reason string) {
	if runtime == nil {
		return
	}
	if atomic.CompareAndSwapInt32(&runtime.shutdownInProgress, 0, 1) {
		runtime.auditEventV0(ctx, "server_shutdown_freeze", "active", "", map[string]interface{}{"reason": reason})
	}
	if runtime.tracker != nil {
		runtime.persistStateTransitionV0(ctx, runtime.tracker.MarkShutdownRequestedV0(runtime.clock.Now()), "shutdown_requested")
	}
}

func (runtime *RuntimeV0) supervisorFrozenForShutdownV0() bool {
	if runtime == nil {
		return false
	}
	return atomic.LoadInt32(&runtime.shutdownInProgress) == 1
}

func (runtime *RuntimeV0) markSupervisorFrozenForShutdownV0(ctx context.Context, reason string) {
	if runtime == nil || runtime.tracker == nil {
		return
	}
	runtime.auditEventV0(ctx, "supervisor_tick_skipped", "skipped", "", map[string]interface{}{"reason": reason})
	runtime.persistStateTransitionV0(ctx, runtime.tracker.MarkSupervisorFrozenV0(reason, runtime.clock.Now()), "supervisor_frozen")
}

func (runtime *RuntimeV0) recordShutdownHTTPResultV0(ctx context.Context, statusCode int, body []byte) {
	if runtime == nil || runtime.tracker == nil {
		return
	}
	projection, keepFrozen := shutdownProjectionFromHTTPV0(statusCode, body)
	if !keepFrozen {
		atomic.StoreInt32(&runtime.shutdownInProgress, 0)
	}
	runtime.persistStateTransitionV0(ctx, runtime.tracker.MarkShutdownResultV0(
		projection,
		keepFrozen,
		runtime.clock.Now(),
	), "shutdown_result")
}

func shutdownProjectionFromHTTPV0(statusCode int, body []byte) (ShutdownProjectionV0, bool) {
	var payload serverShutdownHTTPProjectionV0
	_ = json.Unmarshal(body, &payload)
	projection := ShutdownProjectionV0{
		Status:             firstNonEmptyShutdownFreezeV0(payload.Status, payload.Estado, "http_status"),
		Ready:              payload.ShutdownReady,
		HTTPStatus:         statusCode,
		RunsRequested:      payload.RunsRequested,
		RunsStopped:        payload.RunsStopped,
		AgentsInFlight:     payload.AgentsInFlight,
		CheckpointsPending: payload.CheckpointsPending,
	}
	if statusCode >= http.StatusBadRequest || shutdownFreezeResultIsRejectedV0(payload) {
		return projection, false
	}
	if payload.ShutdownReady {
		return projection, false
	}
	return projection, true
}

func shutdownFreezeResultIsRejectedV0(payload serverShutdownHTTPProjectionV0) bool {
	if strings.TrimSpace(payload.Estado) == "error" {
		return true
	}
	switch strings.TrimSpace(payload.Status) {
	case "requester_not_authorized",
		"queue_reader_required",
		"run_control_reader_required",
		"run_control_writer_required":
		return true
	default:
		return false
	}
}

func firstNonEmptyShutdownFreezeV0(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

type shutdownFreezeResponseCaptureV0 struct {
	http.ResponseWriter
	statusCode int
	body       []byte
}

func (capture *shutdownFreezeResponseCaptureV0) WriteHeader(statusCode int) {
	capture.statusCode = statusCode
	capture.ResponseWriter.WriteHeader(statusCode)
}

func (capture *shutdownFreezeResponseCaptureV0) Write(data []byte) (int, error) {
	if capture.statusCode == 0 {
		capture.statusCode = http.StatusOK
	}
	const maxCapturedShutdownBodyV0 = 1 << 20
	if len(capture.body) < maxCapturedShutdownBodyV0 {
		remaining := maxCapturedShutdownBodyV0 - len(capture.body)
		if len(data) < remaining {
			remaining = len(data)
		}
		capture.body = append(capture.body, data[:remaining]...)
	}
	return capture.ResponseWriter.Write(data)
}
