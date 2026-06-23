package orquestaserver

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
)

const serverResidentDirectorControlRoutePathV0 = "/api/v0/resident-director/control"

type serverResidentDirectorControlRequestV0 struct {
	Action string `json:"action,omitempty"`
	Reason string `json:"reason,omitempty"`
}

type serverResidentDirectorControlProjectionV0 struct {
	Estado          string `json:"estado"`
	Status          string `json:"status"`
	Paused          bool   `json:"paused"`
	TickActive      bool   `json:"tick_active,omitempty"`
	TickPending     bool   `json:"tick_pending,omitempty"`
	ResidentEnabled bool   `json:"resident_enabled"`
}

func (runtime *RuntimeV0) residentDirectorControlHTTPHandlerV0(next http.Handler) http.Handler {
	if next == nil {
		next = http.NotFoundHandler()
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r == nil || r.URL == nil || r.URL.Path != serverResidentDirectorControlRoutePathV0 {
			next.ServeHTTP(w, r)
			return
		}
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, "method_not_allowed", http.StatusMethodNotAllowed)
			return
		}
		request, code := decodeResidentDirectorControlRequestV0(w, r)
		if code != "" {
			writeJSONResponseV0(w, http.StatusBadRequest, map[string]string{"estado": "error", "error": code})
			return
		}
		switch strings.ToLower(strings.TrimSpace(request.Action)) {
		case "pause":
			runtime.PauseResidentDirectorV0(r.Context(), request.Reason)
			writeJSONResponseV0(w, http.StatusOK, runtime.residentDirectorControlProjectionV0("paused"))
		case "resume":
			runtime.ResumeResidentDirectorV0(r.Context(), request.Reason)
			writeJSONResponseV0(w, http.StatusOK, runtime.residentDirectorControlProjectionV0("resumed"))
		case "", "status":
			writeJSONResponseV0(w, http.StatusOK, runtime.residentDirectorControlProjectionV0(""))
		default:
			writeJSONResponseV0(w, http.StatusBadRequest, map[string]string{"estado": "error", "error": "accion_no_soportada"})
		}
	})
}

func decodeResidentDirectorControlRequestV0(
	w http.ResponseWriter,
	r *http.Request,
) (serverResidentDirectorControlRequestV0, string) {
	var request serverResidentDirectorControlRequestV0
	if strings.TrimSpace(r.URL.Query().Get("action")) != "" {
		request.Action = r.URL.Query().Get("action")
	}
	if strings.TrimSpace(r.URL.Query().Get("reason")) != "" {
		request.Reason = r.URL.Query().Get("reason")
	}
	if r.Body == nil || r.Body == http.NoBody {
		return request, ""
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, NormalizeHTTPResourceLimitsV0(HTTPResourceLimitsV0{}).ControlBodyBytes))
	if err := decoder.Decode(&request); err != nil {
		if errors.Is(err, io.EOF) {
			return request, ""
		}
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			return request, "request_body_too_large"
		}
		return request, "request_body_invalido"
	}
	var extra json.RawMessage
	if err := decoder.Decode(&extra); err != io.EOF {
		return request, "request_body_trailing_data"
	}
	return request, ""
}

func (runtime *RuntimeV0) PauseResidentDirectorV0(ctx context.Context, reason string) StateV0 {
	if runtime == nil || runtime.tracker == nil {
		return StateV0{}
	}
	atomic.StoreInt32(&runtime.residentDirectorPaused, 1)
	atomic.StoreInt32(&runtime.residentDirectorTickPending, 0)
	runtime.auditEventV0(ctx, "resident_director_control", "paused", "", map[string]interface{}{"reason": strings.TrimSpace(reason)})
	state := runtime.tracker.MarkResidentDirectorPausedV0(reason, runtime.clock.Now())
	runtime.persistStateTransitionV0(
		ctx,
		state,
		"resident_director_paused",
	)
	return runtime.tracker.SnapshotV0()
}

func (runtime *RuntimeV0) ResumeResidentDirectorV0(ctx context.Context, reason string) StateV0 {
	if runtime == nil || runtime.tracker == nil {
		return StateV0{}
	}
	atomic.StoreInt32(&runtime.residentDirectorPaused, 0)
	runtime.auditEventV0(ctx, "resident_director_control", "resumed", "", map[string]interface{}{"reason": strings.TrimSpace(reason)})
	state := runtime.tracker.MarkResidentDirectorResumedV0(reason, runtime.clock.Now())
	runtime.persistStateTransitionV0(
		ctx,
		state,
		"resident_director_resumed",
	)
	return runtime.tracker.SnapshotV0()
}

func (runtime *RuntimeV0) residentDirectorPausedV0() bool {
	return runtime != nil && atomic.LoadInt32(&runtime.residentDirectorPaused) == 1
}

func (runtime *RuntimeV0) residentDirectorControlProjectionV0(
	status string,
) serverResidentDirectorControlProjectionV0 {
	state := StateV0{}
	if runtime != nil && runtime.tracker != nil {
		state = runtime.tracker.SnapshotV0()
	}
	paused := runtime.residentDirectorPausedV0()
	effectiveStatus := strings.TrimSpace(status)
	if effectiveStatus == "" {
		effectiveStatus = strings.TrimSpace(state.ResidentDirectorStatus)
	}
	if runtime == nil || !runtime.config.ResidentDirectorEnabled {
		effectiveStatus = "disabled"
		paused = false
	} else if paused {
		effectiveStatus = "paused"
	}
	if effectiveStatus == "" {
		effectiveStatus = "idle"
	}
	return serverResidentDirectorControlProjectionV0{
		Estado:          "ok",
		Status:          effectiveStatus,
		Paused:          paused,
		TickActive:      state.ResidentDirectorTickActive,
		TickPending:     runtime != nil && atomic.LoadInt32(&runtime.residentDirectorTickPending) == 1,
		ResidentEnabled: runtime != nil && runtime.config.ResidentDirectorEnabled,
	}
}
