package orquestaappgateway

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	orquestaobservability "orquesta/modulos/orquesta-observability"
	orquestaweb "orquesta/modulos/orquesta-web"
)

func TestAppGatewayObservaWriteHTMLFallidoSinQueryNiPayloadV0(t *testing.T) {
	observer := orquestaobservability.NewInMemoryWebHTMLRenderObserverV0()
	handler := NewHTTPHandlerV0(ConfigV0{Timeout: time.Second, WebHTMLRenderObserver: observer})
	writer := &failingGatewayHTMLWriterV0{header: http.Header{}}
	req := httptest.NewRequest(http.MethodGet, "/ops?token=secret", nil)

	handler.ServeHTTP(writer, req)

	observations, counters := observer.SnapshotV0()
	if len(observations) != 1 ||
		observations[0].ReasonCode != orquestaobservability.WebHTMLResponseWriteFailedReasonV0 ||
		observations[0].Stage != orquestaobservability.WebHTMLRenderStageWriteV0 ||
		observations[0].RouteRef != "route-ref-web-ops" ||
		strings.Contains(observations[0].RouteRef, "token") ||
		counters["web_response_write_failed_total"] != 1 {
		t.Fatalf("observations=%+v counters=%+v", observations, counters)
	}
}

func TestAppGatewayObservaRenderHTMLFallidoDesdeHeaderPublicoV0(t *testing.T) {
	observer := orquestaobservability.NewInMemoryWebHTMLRenderObserverV0()
	handler := ObserveWebHTMLRenderErrorsV0(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set(orquestaweb.WebHTMLErrorCodeHeaderV0, orquestaweb.WebHTMLRenderFailedV0)
		w.Header().Set(orquestaweb.WebHTMLLocaleHeaderV0, "es")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("<html></html>"))
	}), observer)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/nueva-app?raw=form", nil))

	observations, counters := observer.SnapshotV0()
	if len(observations) != 1 ||
		observations[0].ReasonCode != orquestaobservability.WebHTMLRenderFailedReasonV0 ||
		observations[0].Stage != orquestaobservability.WebHTMLRenderStageRenderV0 ||
		observations[0].StatusCode != http.StatusInternalServerError ||
		observations[0].Locale != "es" ||
		counters["web_html_render_failed_total"] != 1 {
		t.Fatalf("observations=%+v counters=%+v", observations, counters)
	}
}

type failingGatewayHTMLWriterV0 struct {
	header http.Header
	status int
}

func (writer *failingGatewayHTMLWriterV0) Header() http.Header {
	return writer.header
}

func (writer *failingGatewayHTMLWriterV0) WriteHeader(status int) {
	writer.status = status
}

func (writer *failingGatewayHTMLWriterV0) Write(_ []byte) (int, error) {
	return 0, errors.New("write failed")
}
