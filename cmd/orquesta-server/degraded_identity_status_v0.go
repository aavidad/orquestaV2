package main

import (
	"encoding/json"
	"net/http"
	"strings"

	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestaserver "orquesta/modulos/orquesta-server"
)

const (
	serverDegradedIdentityStatusV0         = "degraded_identity"
	serverPrepareRunDegradedIdentityV0     = "autoprogramming_prepare_run_degraded_identity"
	serverWorkLaunchDegradedIdentityV0     = "server_work_launch_degraded_identity"
	serverIdentityEvidenceValidV0          = "evidence-ref-server-worktree-identity-valid"
	serverIdentityEvidenceDegradedPrefixV0 = "evidence-ref-server-worktree-identity-"
)

func newDegradedIdentityHTTPHandlerV0(_ orquestaserver.ConfigV0, reason string) http.Handler {
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
		default:
			writeDegradedIdentityWorkBlockedV0(w, r.URL.Path, reason)
		}
	})
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
