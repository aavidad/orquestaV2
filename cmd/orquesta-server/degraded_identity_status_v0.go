package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestaserver "orquesta/modulos/orquesta-server"
	orquestaservershutdown "orquesta/modulos/orquesta-server-shutdown"
)

const (
	serverDegradedIdentityStatusV0         = "degraded_identity"
	serverPrepareRunDegradedIdentityV0     = "autoprogramming_prepare_run_degraded_identity"
	serverWorkLaunchDegradedIdentityV0     = "server_work_launch_degraded_identity"
	serverIdentityEvidenceValidV0          = "evidence-ref-server-worktree-identity-valid"
	serverIdentityEvidenceDegradedPrefixV0 = "evidence-ref-server-worktree-identity-"
	serverDegradedIdentityShutdownPathV0   = "/api/v0/server/shutdown"
)

type serverDegradedIdentityShutdownRequestV0 struct {
	RequestedBy    string `json:"requested_by"`
	IdempotencyKey string `json:"idempotency_key"`
}

func newDegradedIdentityHTTPHandlerV0(_ orquestaserver.ConfigV0, reason string) http.Handler {
	return newDegradedIdentityHTTPHandlerWithShutdownV0(reason, nil)
}

func newDegradedIdentityHTTPHandlerWithShutdownV0(reason string, requestShutdown func()) http.Handler {
	reason = publicServerWorktreeIdentityReasonV0(reason)
	status := map[string]any{
		"schema_version":       "orquesta_server_state.v0",
		"status":               serverDegradedIdentityStatusV0,
		"availability_status":  serverDegradedIdentityStatusV0,
		"availability_reason":  reason,
		"startup_status":       serverDegradedIdentityStatusV0,
		"startup_ready":        false,
		"runtime_identity_ref": "server-runtime-identity-ref-degraded",
		"worktree_identity": map[string]any{
			"status":        serverDegradedIdentityStatusV0,
			"reason":        reason,
			"evidence_refs": []string{serverIdentityEvidenceDegradedPrefixV0 + reason},
		},
	}
	var shutdownOnce sync.Once
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health", "/healthz":
			writeDegradedIdentityJSONV0(w, http.StatusOK, map[string]string{"status": "ok"})
		case orquestaserver.ServerStatusEndpointV0, orquestaserver.ServerStatusLegacyEndpointV0:
			writeDegradedIdentityJSONV0(w, http.StatusOK, status)
		case orquestaserver.ServerReadinessEndpointV0:
			writeDegradedIdentityJSONV0(w, http.StatusServiceUnavailable, map[string]any{
				"schema_version": orquestaserver.ServerReadinessSchemaVersionV0,
				"ready":          false, "status": serverDegradedIdentityStatusV0,
				"startup_ready": false, "startup_status": serverDegradedIdentityStatusV0,
				"evidence_refs": []string{serverIdentityEvidenceDegradedPrefixV0 + reason},
			})
		case serverDegradedIdentityShutdownPathV0:
			if r.Method != http.MethodPost {
				writeDegradedIdentityWorkBlockedV0(w, r.URL.Path, reason)
				return
			}
			if code, field := validateDegradedIdentityShutdownRequestV0(r); code != "" {
				writeDegradedIdentityShutdownRejectedV0(w, reason, code, field)
				return
			}
			writeDegradedIdentityJSONV0(w, http.StatusOK, map[string]any{
				"estado":         "ok",
				"status":         "ready",
				"shutdown_ready": true,
				"exit_pending":   true,
				"evidence_refs":  []string{serverIdentityEvidenceDegradedPrefixV0 + reason},
			})
			shutdownOnce.Do(func() {
				if requestShutdown != nil {
					requestShutdown()
					return
				}
				shutdownDegradedIdentityHTTPServerV0(r)
			})
		default:
			writeDegradedIdentityWorkBlockedV0(w, r.URL.Path, reason)
		}
	})
}

func validateDegradedIdentityShutdownRequestV0(request *http.Request) (string, string) {
	if request == nil || request.Body == nil {
		return orquestamcp.MCPPublicErrBodyInvalidV0, "body"
	}
	var input serverDegradedIdentityShutdownRequestV0
	decoder := json.NewDecoder(request.Body)
	if err := decoder.Decode(&input); err != nil {
		return orquestamcp.MCPPublicErrBodyInvalidV0, "body"
	}
	var extra json.RawMessage
	if err := decoder.Decode(&extra); err != io.EOF {
		return orquestamcp.MCPPublicErrBodyTrailingDataV0, "body"
	}
	if !orquestaservershutdown.ServerShutdownRequesterAuthorizedV0(input.RequestedBy) {
		return orquestaservershutdown.ServerShutdownStatusRequesterDeniedV0, "requested_by"
	}
	if strings.TrimSpace(input.IdempotencyKey) == "" {
		return orquestamcp.MCPPublicMutationIssueIdempotencyKeyRequiredV0, "idempotency_key"
	}
	return "", ""
}

func writeDegradedIdentityShutdownRejectedV0(w http.ResponseWriter, reason string, code string, field string) {
	writeDegradedIdentityJSONV0(w, http.StatusBadRequest, struct {
		Estado          string                             `json:"estado"`
		Accepted        bool                               `json:"accepted"`
		ErroresPublicos []orquestamcp.MCPValidationIssueV0 `json:"errores_publicos"`
		EvidenceRefs    []string                           `json:"evidence_refs,omitempty"`
	}{
		Estado:   "error",
		Accepted: false,
		ErroresPublicos: []orquestamcp.MCPValidationIssueV0{{
			Code: strings.TrimSpace(code), Field: strings.TrimSpace(field), Message: strings.TrimSpace(code),
		}},
		EvidenceRefs: []string{serverIdentityEvidenceDegradedPrefixV0 + reason},
	})
}

func shutdownDegradedIdentityHTTPServerV0(request *http.Request) {
	if request == nil {
		return
	}
	server, ok := request.Context().Value(http.ServerContextKey).(*http.Server)
	if !ok || server == nil {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			_ = server.Close()
		}
	}()
}

func publicServerWorktreeIdentityReasonV0(reason string) string {
	switch strings.TrimSpace(reason) {
	case serverWorktreeIdentityMissingV0,
		serverWorktreeIdentityInvalidV0,
		serverWorktreeIdentitySymlinkV0,
		serverWorktreeIdentityRetiredV0,
		serverWorktreeIdentityStaleV0,
		serverWorktreeIdentityNotAlignedV0,
		serverWorktreeIdentityRemoteV0,
		serverWorktreeIdentityRefV0,
		serverWorktreeIdentityBinaryV0,
		serverWorktreeIdentityBuildCommitV0,
		serverWorktreeIdentityBuildModifiedV0:
		return strings.TrimSpace(reason)
	default:
		return serverWorktreeIdentityInvalidV0
	}
}

func writeDegradedIdentityWorkBlockedV0(w http.ResponseWriter, path string, reason string) {
	code := serverWorkLaunchDegradedIdentityV0
	field := "server_identity"
	if path == orquestamcp.MCPAutoprogrammingPrepareRunHTTPPathV0 {
		code = serverPrepareRunDegradedIdentityV0
		field = "project_work_dir"
	}
	writeDegradedIdentityJSONV0(w, http.StatusServiceUnavailable, struct {
		Estado          string                             `json:"estado"`
		Accepted        bool                               `json:"accepted"`
		ErroresPublicos []orquestamcp.MCPValidationIssueV0 `json:"errores_publicos"`
		EvidenceRefs    []string                           `json:"evidence_refs,omitempty"`
	}{
		Estado:   "error",
		Accepted: false,
		ErroresPublicos: []orquestamcp.MCPValidationIssueV0{{
			Code: code, Field: field, Message: code,
		}},
		EvidenceRefs: []string{serverIdentityEvidenceDegradedPrefixV0 + reason},
	})
}

func writeDegradedIdentityJSONV0(w http.ResponseWriter, code int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(value)
}
