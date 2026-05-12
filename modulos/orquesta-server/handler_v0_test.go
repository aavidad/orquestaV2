package orquestaserver

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHandlerV0ExponeHealthStatusYDelegaV0(t *testing.T) {
	tracker := NewStatusTrackerV0(ConfigV0{Addr: "127.0.0.1:8787"}, time.Now().UTC())
	tracker.MarkServingV0("127.0.0.1:8787", time.Now().UTC())
	handler := NewHandlerV0(HandlerConfigV0{
		Tracker: tracker,
		AppHandler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("app"))
		}),
	})

	assertServerPathV0(t, handler, "/healthz", `"ok"`)
	assertServerPathV0(t, handler, "/api/status", `"running"`)
	assertServerPathV0(t, handler, "/nueva-app", "app")
}

func assertServerPathV0(t *testing.T, handler http.Handler, path string, want string) {
	t.Helper()
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("%s status=%d body=%s", path, rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), want) {
		t.Fatalf("%s body=%s want contiene %s", path, rec.Body.String(), want)
	}
}
