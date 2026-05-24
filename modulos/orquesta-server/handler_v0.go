package orquestaserver

import (
	"encoding/json"
	"net/http"
	"time"
)

type HandlerConfigV0 struct {
	AppHandler http.Handler
	Tracker    *StatusTrackerV0
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
	case "/api/status", "/api/v0/server/status":
		handler.writeJSONV0(w, http.StatusOK, handler.statusV0())
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
