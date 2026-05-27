package orquestaserver

import (
	"net/http"
	"time"

	orquestaobservability "orquesta/modulos/orquesta-observability"
	publicidentity "orquesta/modulos/orquesta-server/publicidentity"
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
	case ServerStatusEndpointV0:
		handler.writeJSONV0(w, http.StatusOK, NewServerPublicStatusV0(handler.statusV0()))
	case ServerStatusLegacyEndpointV0:
		handler.writeLegacyStatusAliasV0(w, NewServerPublicStatusV0(handler.statusV0()))
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
		handler.writeOperationalStatusErrorV0(w, http.StatusMethodNotAllowed, r.Header.Get(publicidentity.PublicCorrelationHeaderV0), "method", "metodo_no_permitido")
		return
	}
	source := handler.config.OperationalStatusSource
	if source == nil && handler.config.Tracker != nil {
		source = ResidentOperationalStatusSourceV0{Tracker: handler.config.Tracker}
	}
	if source == nil {
		handler.writeOperationalStatusErrorV0(w, http.StatusServiceUnavailable, r.Header.Get(publicidentity.PublicCorrelationHeaderV0), "source", orquestaobservability.ErrProyeccionNoDisponibleV0)
		return
	}
	var query orquestaobservability.OperationalStatusQueryV0
	if code := decodeServerControlJSONV0(w, r, &query); code != "" {
		handler.writeOperationalStatusErrorV0(w, http.StatusBadRequest, r.Header.Get(publicidentity.PublicCorrelationHeaderV0), "body", code)
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
	handler.observeResponseWriteV0(writeServerJSONResponseV0(w, status, value))
}

func (handler handlerV0) writeLegacyStatusAliasV0(w http.ResponseWriter, value ServerPublicStatusV0) {
	w.Header().Set(ServerStatusCanonicalHeaderV0, ServerStatusEndpointV0)
	w.Header().Set(ServerStatusCompatibilityHeaderV0, ServerStatusCompatibilityLegacyV0)
	w.Header().Set(ServerStatusOwnerHeaderV0, ServerStatusOwnerServerV0)
	w.Header().Set(ServerStatusSunsetHeaderV0, ServerStatusSunsetNoNewUseV0)
	w.Header().Set("Deprecation", "true")
	w.Header().Set("Link", "<"+ServerStatusEndpointV0+">; rel=\"canonical\"")
	w.Header().Set("Warning", `299 - "legacy status alias; use /api/v0/server/status"`)
	handler.writeJSONV0(w, http.StatusOK, value)
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
	if correlationID != "" {
		w.Header().Set(publicidentity.PublicCorrelationHeaderV0, correlationID)
	}
	handler.observeResponseWriteV0(writeServerJSONResponseV0(w, status, orquestaobservability.OperationalStatusValidationErrorV0{Issues: issues}))
}

func (handler handlerV0) observeResponseWriteV0(observation serverHTTPResponseObservationV0) {
	if observation.OK || handler.config.Tracker == nil {
		return
	}
	switch observation.Code {
	case serverResponseEncodeFailedCodeV0:
		handler.config.Tracker.MarkResponseEncodeFailedV0(observation.Stage, time.Now().UTC())
	case serverResponseWriteFailedCodeV0:
		handler.config.Tracker.MarkResponseWriteFailedV0(observation.Stage, time.Now().UTC())
	}
}

func statusForServerOperationalStatusErrorV0(err error) int {
	if orquestaobservability.HasOperationalStatusIssueV0(err, orquestaobservability.ErrProyeccionNoDisponibleV0) ||
		orquestaobservability.HasOperationalStatusIssueV0(err, orquestaobservability.ErrDiagnosticoNoDisponibleV0) {
		return http.StatusServiceUnavailable
	}
	return http.StatusBadRequest
}
