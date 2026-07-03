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
	Estado                  string                               `json:"estado,omitempty"`
	Status                  string                               `json:"status,omitempty"`
	ShutdownReady           bool                                 `json:"shutdown_ready,omitempty"`
	RunsRequested           int                                  `json:"runs_requested,omitempty"`
	RunsStopped             int                                  `json:"runs_stopped,omitempty"`
	AgentsInFlight          int                                  `json:"agents_in_flight,omitempty"`
	CheckpointsPending      int                                  `json:"checkpoints_pending,omitempty"`
	CheckpointAgentsPending int                                  `json:"checkpoint_agents_pending,omitempty"`
	AsyncWorkActive         int                                  `json:"shutdown_async_work_active,omitempty"`
	ActiveWorkCount         int                                  `json:"active_work_count,omitempty"`
	ActiveWorkRefs          []string                             `json:"active_work_refs,omitempty"`
	ActiveWorks             []serverShutdownHTTPWorkProjectionV0 `json:"active_works,omitempty"`
}

type serverShutdownHTTPWorkProjectionV0 struct {
	Kind            string `json:"kind,omitempty"`
	RunRef          string `json:"run_ref,omitempty"`
	WorkRef         string `json:"work_ref,omitempty"`
	ExternalWorkRef string `json:"external_work_ref,omitempty"`
	Status          string `json:"status,omitempty"`
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
		runtime.snapshotShutdownActiveWorkForHTTPV0(r.Context())
		capture := &shutdownFreezeResponseCaptureV0{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(capture, r)
		runtime.recordShutdownHTTPResultV0(r.Context(), capture.statusCode, capture.body)
	})
}

func (runtime *RuntimeV0) serverLifecycleHTTPHandlerV0(next http.Handler) http.Handler {
	return runtime.handoffHTTPHandlerV0(runtime.residentDirectorControlHTTPHandlerV0(runtime.shutdownFreezeHTTPHandlerV0(next)))
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
	projection = runtime.shutdownProjectionWithPreviousSnapshotV0(projection, keepFrozen)
	projection = normalizeShutdownStopConfirmationV0(projection)
	if !projection.Ready && shutdownProjectionHasBlockingWorkV0(projection) {
		keepFrozen = true
	}
	if !keepFrozen {
		atomic.StoreInt32(&runtime.shutdownInProgress, 0)
	}
	runtime.persistStateTransitionV0(ctx, runtime.tracker.MarkShutdownResultV0(
		projection,
		keepFrozen,
		runtime.clock.Now(),
	), "shutdown_result")
	if projection.Ready && !keepFrozen {
		runtime.requestShutdownReadyV0()
	}
}

func (runtime *RuntimeV0) shutdownProjectionWithPreviousSnapshotV0(
	projection ShutdownProjectionV0,
	keepFrozen bool,
) ShutdownProjectionV0 {
	if runtime == nil || runtime.tracker == nil || (!keepFrozen && !projection.Ready) {
		return projection
	}
	state := runtime.tracker.SnapshotV0()
	if projection.ActiveWorkCount <= 0 &&
		len(projection.ActiveWorkRefs) == 0 &&
		(state.ShutdownActiveWorkCount > 0 || len(state.ShutdownActiveWorkRefs) > 0) {
		projection.ActiveWorkCount = state.ShutdownActiveWorkCount
		projection.ActiveWorkRefs = compactServerStringsV0(state.ShutdownActiveWorkRefs)
	}
	if projection.AsyncWorkActive <= 0 && state.ShutdownAsyncWorkActive > 0 {
		projection.AsyncWorkActive = state.ShutdownAsyncWorkActive
	}
	if strings.TrimSpace(projection.Status) == "http_status" &&
		(projection.ActiveWorkCount > 0 || len(projection.ActiveWorkRefs) > 0 || projection.AsyncWorkActive > 0) {
		projection.Status = firstNonEmptyShutdownFreezeV0(state.ShutdownStatus, projection.Status)
	}
	return projection
}

func shutdownProjectionHasBlockingWorkV0(projection ShutdownProjectionV0) bool {
	return projection.AgentsInFlight > 0 ||
		projection.CheckpointsPending > 0 ||
		projection.CheckpointAgentsPending > 0 ||
		projection.AsyncWorkActive > 0 ||
		projection.ActiveWorkCount > 0 ||
		len(projection.ActiveWorkRefs) > 0 ||
		(projection.RunsRequested > 0 && projection.RunsStopped < projection.RunsRequested)
}

func (runtime *RuntimeV0) requestShutdownReadyV0() {
	if runtime == nil || runtime.shutdownReadyRequested == nil {
		return
	}
	runtime.shutdownReadyOnce.Do(func() {
		close(runtime.shutdownReadyRequested)
	})
}

func shutdownProjectionFromHTTPV0(statusCode int, body []byte) (ShutdownProjectionV0, bool) {
	var payload serverShutdownHTTPProjectionV0
	_ = json.Unmarshal(body, &payload)
	projection := ShutdownProjectionV0{
		Status:                  firstNonEmptyShutdownFreezeV0(payload.Status, payload.Estado, "http_status"),
		Ready:                   payload.ShutdownReady,
		HTTPStatus:              statusCode,
		RunsRequested:           payload.RunsRequested,
		RunsStopped:             payload.RunsStopped,
		AgentsInFlight:          payload.AgentsInFlight,
		CheckpointsPending:      payload.CheckpointsPending,
		CheckpointAgentsPending: payload.CheckpointAgentsPending,
		AsyncWorkActive:         payload.AsyncWorkActive,
		ActiveWorkCount:         payload.ActiveWorkCount,
		ActiveWorkRefs: compactServerStringsV0(append(
			shutdownProjectionDirectActiveWorkRefsV0(payload.ActiveWorkRefs),
			shutdownProjectionActiveWorkRefsV0(payload.ActiveWorks)...,
		)),
	}
	if projection.ActiveWorkCount <= 0 && len(payload.ActiveWorks) > 0 {
		projection.ActiveWorkCount = len(payload.ActiveWorks)
	}
	projection = normalizeShutdownStopConfirmationV0(projection)
	if statusCode >= http.StatusBadRequest || shutdownFreezeResultIsRejectedV0(payload) {
		projection.Ready = false
		return projection, false
	}
	if payload.ShutdownReady {
		if projection.Status == "stop_pending" {
			projection.Ready = false
			return projection, true
		}
		return projection, projection.Status == "stop_pending"
	}
	return projection, true
}

func shutdownProjectionActiveWorkRefsV0(
	works []serverShutdownHTTPWorkProjectionV0,
) []string {
	out := make([]string, 0, len(works)*4)
	for _, work := range works {
		prefix := "shutdown-active-work"
		if kind := safeShutdownProjectionRefPartV0(work.Kind); kind != "" {
			prefix += "-" + kind
		}
		for _, ref := range []string{work.RunRef, work.WorkRef, work.ExternalWorkRef, work.Status} {
			if safe := safeShutdownProjectionRefPartV0(ref); safe != "" {
				out = append(out, prefix+"-"+safe)
			}
		}
	}
	return compactServerStringsV0(out)
}

func shutdownProjectionDirectActiveWorkRefsV0(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if strings.Contains(value, "/") || strings.Contains(value, `\`) {
			out = append(out, "shutdown-active-work-ref-redacted")
			continue
		}
		if safe := safeShutdownProjectionRefPartV0(value); safe != "" {
			if strings.HasPrefix(safe, "shutdown-active-work-") {
				out = append(out, safe)
				continue
			}
			out = append(out, "shutdown-active-work-"+safe)
		}
	}
	return compactServerStringsV0(out)
}

func safeShutdownProjectionRefPartV0(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return ""
	}
	var builder strings.Builder
	lastSep := false
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z',
			r >= '0' && r <= '9':
			builder.WriteRune(r)
			lastSep = false
		default:
			if !lastSep {
				builder.WriteByte('-')
				lastSep = true
			}
		}
	}
	return strings.Trim(builder.String(), "-")
}

func normalizeShutdownStopConfirmationV0(projection ShutdownProjectionV0) ShutdownProjectionV0 {
	if projection.ActiveWorkCount <= 0 && len(projection.ActiveWorkRefs) > 0 {
		projection.ActiveWorkCount = len(projection.ActiveWorkRefs)
	}
	if !projection.Ready {
		return projection
	}
	if shutdownProjectionHasBlockingWorkV0(projection) {
		projection.Ready = false
		projection.Status = "stop_pending"
	}
	return projection
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
