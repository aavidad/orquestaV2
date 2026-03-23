package localrpc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type Executor func(context.Context, *ExecRequest) (*ExecResponse, error)

type Server struct {
	State    State
	Executor Executor
}

func NewMux(server *Server) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc(HealthPath, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, http.StatusOK, HealthResponse{
			OK:        true,
			Addr:      server.State.Addr,
			PID:       server.State.PID,
			Kind:      server.State.Kind,
			ScopeID:   server.State.ScopeID,
			DBPath:    server.State.DBPath,
			StartedAt: server.State.StartedAt,
			Version:   server.State.Version,
		})
	})
	mux.HandleFunc(ExecPath, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if server != nil && server.State.Token != "" && r.Header.Get(HeaderAuthToken) != server.State.Token {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if server == nil || server.Executor == nil {
			http.Error(w, "executor no configurado", http.StatusServiceUnavailable)
			return
		}

		var req ExecRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("json invalido: %v", err), http.StatusBadRequest)
			return
		}
		res, err := server.Executor(r.Context(), &req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, res)
	})
	return mux
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
