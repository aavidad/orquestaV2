package orquestaserver

import (
	"encoding/json"
	"net/http"
	"time"

	orquestaobservability "orquesta/modulos/orquesta-observability"
)

type HandlerConfigV0 struct {
	AppHandler              http.Handler
	Tracker                 *StatusTrackerV0
	OperationalStatusSource orquestaobservability.OperationalStatusQuerySourceV0
}

func NewHandlerV0(config HandlerConfigV0) http.Handler {
	return handlerV0{config: config}
}

type handlerV0 struct {
	config HandlerConfigV0
}

func (handler handlerV0) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/healthz":
		handler.writeJSONV0(w, http.StatusOK, map[string]string{"status": "ok"})
	case ServerReadinessEndpointV0:
		readiness := NewServerReadinessV0(handler.statusV0())
		status := http.StatusOK
		if !readiness.Ready {
			status = http.StatusServiceUnavailable
		}
		handler.writeJSONV0(w, status, readiness)
	case "/api/status", "/api/v0/server/status":
		handler.writeJSONV0(w, http.StatusOK, NewServerPublicStatusV0(handler.statusV0()))
	case "/api/v0/operational-status/query":
		handler.serveOperationalStatusV0(w, r)
	case ServerResourcesEndpointV0:
		handler.writeJSONV0(w, http.StatusOK, NewServerResourcesV0(handler.statusV0(), time.Now().UTC()))
	default:
		if handler.config.AppHandler != nil {
			handler.config.AppHandler.ServeHTTP(w, r)
			return
		}
		http.NotFound(w, r)
	}
}

func (handler handlerV0) serveOperationalStatusV0(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		handler.writeOperationalStatusErrorV0(w, http.StatusMethodNotAllowed, r.Header.Get("X-Correlation-ID"), "method", "metodo_no_permitido")
		return
	}
	source := handler.config.OperationalStatusSource
	if source == nil && handler.config.Tracker != nil {
		source = ResidentOperationalStatusSourceV0{Tracker: handler.config.Tracker}
	}
	if source == nil {
		handler.writeOperationalStatusErrorV0(w, http.StatusServiceUnavailable, r.Header.Get("X-Correlation-ID"), "source", orquestaobservability.ErrProyeccionNoDisponibleV0)
		return
	}
	var query orquestaobservability.OperationalStatusQueryV0
	if err := json.NewDecoder(r.Body).Decode(&query); err != nil {
		handler.writeOperationalStatusErrorV0(w, http.StatusBadRequest, r.Header.Get("X-Correlation-ID"), "body", "request_body_invalido")
		return
	}
	diagnostic, err := source.QueryOperationalStatusV0(query)
	if err != nil {
		handler.writeOperationalStatusValidationErrorV0(w, statusForServerOperationalStatusErrorV0(err), query.CorrelationID, err)
		return
	}
	handler.writeJSONV0(w, http.StatusOK, diagnostic)
}

func (handler handlerV0) statusV0() StateV0 {
	if handler.config.Tracker == nil {
		return StateV0{SchemaVersion: StateSchemaVersionV0, Status: "unknown"}
	}
	return handler.config.Tracker.SnapshotV0()
}

func (handler handlerV0) writeJSONV0(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func (handler handlerV0) writeOperationalStatusErrorV0(
	w http.ResponseWriter,
	status int,
	correlationID string,
	field string,
	code string,
) {
	handler.writeOperationalStatusValidationV0(
		w,
		status,
		correlationID,
		[]orquestaobservability.OperationalStatusValidationIssueV0{{
			Code:  code,
			Field: field,
		}},
	)
}

func (handler handlerV0) writeOperationalStatusValidationErrorV0(
	w http.ResponseWriter,
	status int,
	correlationID string,
	err error,
) {
	validation, ok := err.(orquestaobservability.OperationalStatusValidationErrorV0)
	if !ok || len(validation.Issues) == 0 {
		handler.writeOperationalStatusErrorV0(w, status, correlationID, "source", orquestaobservability.ErrOperationalStatusQueryInvalidaV0)
		return
	}
	handler.writeOperationalStatusValidationV0(w, status, correlationID, validation.Issues)
}

func (handler handlerV0) writeOperationalStatusValidationV0(
	w http.ResponseWriter,
	status int,
	correlationID string,
	issues []orquestaobservability.OperationalStatusValidationIssueV0,
) {
	w.Header().Set("Content-Type", "application/json")
	if correlationID != "" {
		w.Header().Set("X-Correlation-ID", correlationID)
	}
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(orquestaobservability.OperationalStatusValidationErrorV0{Issues: issues})
}

func statusForServerOperationalStatusErrorV0(err error) int {
	if orquestaobservability.HasOperationalStatusIssueV0(err, orquestaobservability.ErrProyeccionNoDisponibleV0) ||
		orquestaobservability.HasOperationalStatusIssueV0(err, orquestaobservability.ErrDiagnosticoNoDisponibleV0) {
		return http.StatusServiceUnavailable
	}
	return http.StatusBadRequest
}
