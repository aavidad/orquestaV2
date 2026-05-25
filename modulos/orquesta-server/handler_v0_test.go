package orquestaserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHandlerV0ExponeHealthStatusYDelegaV0(t *testing.T) {
	tracker := NewStatusTrackerV0(ConfigV0{
		Addr: "127.0.0.1:8787",
		EffectiveConfig: ServerEffectiveConfigV0{
			Settings: []ServerConfigSettingV0{{
				Key:      "ORQUESTA_SERVER_MAX_RUNS_PER_TICK",
				Value:    "10",
				Editable: true,
			}},
		},
	}, time.Now().UTC())
	tracker.MarkServingV0("127.0.0.1:8787", time.Now().UTC())
	tracker.MarkStartupReadyV0(StartupCheckResultV0{
		Status:       StartupCheckStatusReadyV0,
		Ready:        true,
		Message:      "startup_ready_public",
		EvidenceRefs: []string{"evidence-ref-startup-ready"},
	}, time.Now().UTC())
	handler := NewHandlerV0(HandlerConfigV0{
		Tracker: tracker,
		AppHandler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("app"))
		}),
	})

	assertServerPathV0(t, handler, "/healthz", `"ok"`)
	assertServerPathV0(t, handler, ServerReadinessEndpointV0, `"ready":true`)
	assertServerPathV0(t, handler, "/api/status", `"running"`)
	assertServerPathV0(t, handler, "/api/v0/server/status", `"effective_config"`)
	assertServerPathV0(t, handler, ServerResourcesEndpointV0, ServerResourcesSchemaVersionV0)
	assertServerPathV0(t, handler, "/nueva-app", "app")
}

func TestHandlerV0ReadinessNoExponePathsNiRuntimeDirsV0(t *testing.T) {
	tracker := NewStatusTrackerV0(ConfigV0{
		Addr:           "127.0.0.1:8787",
		ProjectWorkDir: "/tmp/orquesta-project-secret",
		RuntimeWorkDir: "/tmp/orquesta-runtime-secret",
	}, time.Now().UTC())
	tracker.MarkStartupBlockedV0(StartupCheckResultV0{
		Status:       "startup_waiting_cleanup",
		Ready:        false,
		Message:      "cleanup pending in /tmp/orquesta-runtime-secret",
		EvidenceRefs: []string{"evidence-ref-cleanup-pending"},
	}, time.Now().UTC())
	handler := NewHandlerV0(HandlerConfigV0{Tracker: tracker})

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, ServerReadinessEndpointV0, nil))

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, forbidden := range []string{"orquesta-project-secret", "orquesta-runtime-secret", "project_work_dir", "runtime_work_dir"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("readiness filtra %q en %s", forbidden, body)
		}
	}
	var readiness ServerReadinessV0
	if err := json.Unmarshal(rec.Body.Bytes(), &readiness); err != nil {
		t.Fatalf("json=%v body=%s", err, body)
	}
	if readiness.Ready || readiness.StartupReady || readiness.StartupStatus != "startup_waiting_cleanup" {
		t.Fatalf("readiness=%+v", readiness)
	}
	if readiness.StartupMessage != "startup_message_redacted" {
		t.Fatalf("startup_message=%q", readiness.StartupMessage)
	}
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
