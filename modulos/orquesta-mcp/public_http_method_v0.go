package orquestamcp

import (
	"net/http"
	"strings"
)

func handleMCPPublicHTTPOptionsV0(w http.ResponseWriter, r *http.Request, allowed ...string) bool {
	if r.Method != http.MethodOptions {
		return false
	}
	setMCPPublicHTTPAllowV0(w, allowed...)
	w.WriteHeader(http.StatusNoContent)
	return true
}

func setMCPPublicHTTPAllowV0(w http.ResponseWriter, allowed ...string) {
	w.Header().Set("Allow", mcpPublicHTTPAllowHeaderV0(allowed...))
}

func mcpPublicHTTPAllowHeaderV0(allowed ...string) string {
	methods := make([]string, 0, len(allowed)+1)
	seen := map[string]struct{}{}
	for _, method := range allowed {
		method = strings.TrimSpace(method)
		if method == "" {
			continue
		}
		if _, ok := seen[method]; ok {
			continue
		}
		seen[method] = struct{}{}
		methods = append(methods, method)
	}
	if _, ok := seen[http.MethodOptions]; !ok {
		methods = append(methods, http.MethodOptions)
	}
	return strings.Join(methods, ", ")
}
