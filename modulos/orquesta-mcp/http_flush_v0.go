package orquestamcp

import "net/http"

func flushMCPHTTPResponseV0(w http.ResponseWriter) {
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
}
