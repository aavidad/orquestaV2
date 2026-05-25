package orquestaweb

import (
	"net/http"
	"strings"
)

const WebOpsDashboardPageEndpointV0 = "/ops"

type OpsDashboardWebEndpointV0 struct{}

func NewOpsDashboardWebEndpointV0() OpsDashboardWebEndpointV0 {
	return OpsDashboardWebEndpointV0{}
}

func (endpoint OpsDashboardWebEndpointV0) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "metodo no permitido", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(opsDashboardHTMLV0()))
}

func opsDashboardHTMLV0() string {
	return strings.TrimSpace(strings.Join([]string{
		opsDashboardHTMLChunk0V0,
		opsDashboardHTMLChunk1V0,
		opsDashboardHTMLChunk2V0,
		opsDashboardHTMLChunk3V0,
		opsDashboardHTMLChunk4V0,
		opsDashboardHTMLChunk5V0,
	}, "\n")) + "\n"
}
