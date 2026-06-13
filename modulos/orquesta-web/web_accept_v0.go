package orquestaweb

import (
	"net/http"
	"strings"
)

func webRequestWantsHTMLV0(r *http.Request) bool {
	if r == nil {
		return false
	}
	accept := strings.ToLower(strings.TrimSpace(r.Header.Get("Accept")))
	return strings.Contains(accept, "text/html") && !strings.Contains(accept, "application/json")
}
