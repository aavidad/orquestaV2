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
		StartupRevision: StartupRevisionSummaryV0{
			RevisionRef:     "revision-ref-orquesta-startup-20260526t120000z",
			QueueRemoved:    2,
			ControlRemoved:  1,
			RuntimeArchived: 1,
			RetentionDays:   14,
		},
	}, time.Now().UTC())
	handler := NewHandlerV0(HandlerConfigV0{
		Tracker: tracker,
		RouteManifest: []ServerRouteResourceV0{{
			Ref:             "route-ref-test-domain-work-v0",
			Pattern:         "/api/v0/domain-work",
			Kind:            "exact",
			Owner:           "test",
			Methods:         []string{http.MethodPost},
			SecurityProfile: "control_plane_mutation",
			Mounted:         true,
		}},
		AppHandler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("app"))
		}),
	})

	assertServerPathV0(t, handler, "/health", `"ok"`)
	assertServerPathV0(t, handler, "/healthz", `"ok"`)
	assertServerPathV0(t, handler, ServerReadinessEndpointV0, `"ready":true`)
	assertServerPathV0(t, handler, ServerReadinessEndpointV0, `"revision_ref":"revision-ref-orquesta-startup-20260526t120000z"`)
	assertServerPathV0(t, handler, ServerStatusEndpointV0, `"running"`)
	assertServerPathV0(t, handler, ServerStatusEndpointV0, `"startup_revision"`)
	assertServerPathV0(t, handler, ServerStatusEndpointV0, `"daemon_epoch_ref"`)
	assertServerPathV0(t, handler, ServerStatusEndpointV0, `"effective_config"`)
	assertLegacyServerStatusAliasV0(t, handler)
	assertServerPathV0(t, handler, ServerResourcesEndpointV0, ServerResourcesSchemaVersionV0)
	assertServerPathV0(t, handler, ServerResourcesEndpointV0, ServerRouteManifestSchemaVersionV0)
	assertServerPathV0(t, handler, ServerResourcesEndpointV0, `"/api/v0/domain-work"`)
	assertServerPathV0(t, handler, ServerResourcesEndpointV0, `"security_profile":"control_plane_mutation"`)
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

func TestHandlerV0ReadinessExponeStartupBlockersV0(t *testing.T) {
	tracker := NewStatusTrackerV0(ConfigV0{Addr: "127.0.0.1:8787"}, time.Now().UTC())
	tracker.MarkStartupBlockedV0(StartupCheckResultV0{
		Status: "startup_dirty_runs_detected",
		Ready:  false,
		Blockers: []StartupBlockerV0{{
			RunRef:       "run-startup-blocker-001",
			AppRef:       "app-temporal",
			QueueStatus:  "ready",
			UpdatedAt:    "2026-06-30T16:50:00Z",
			AgeSeconds:   600,
			Active:       true,
			ProcessState: "active_or_in_flight",
			Action:       "inspect_live_run_or_force_stop_explicit",
			EvidenceRefs: []string{"evidence-ref-orquesta-startup-dirty-runs"},
		}},
	}, time.Date(2026, 6, 30, 17, 0, 0, 0, time.UTC))
	handler := NewHandlerV0(HandlerConfigV0{Tracker: tracker})

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, ServerReadinessEndpointV0, nil))

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var readiness ServerReadinessV0
	if err := json.Unmarshal(rec.Body.Bytes(), &readiness); err != nil {
		t.Fatalf("json=%v body=%s", err, rec.Body.String())
	}
	if readiness.Ready ||
		readiness.StartupReady ||
		readiness.StartupStatus != "startup_dirty_runs_detected" ||
		len(readiness.StartupBlockers) != 1 {
		t.Fatalf("readiness=%+v", readiness)
	}
	blocker := readiness.StartupBlockers[0]
	if blocker.RunRef != "run-startup-blocker-001" ||
		blocker.AgeSeconds != 600 ||
		blocker.ProcessState != "active_or_in_flight" ||
		blocker.Action != "inspect_live_run_or_force_stop_explicit" ||
		!containsServerStringForTestV0(blocker.EvidenceRefs, "evidence-ref-orquesta-startup-dirty-runs") {
		t.Fatalf("blocker=%+v", blocker)
	}
}

func TestHandlerV0ReadinessBloqueaExternalWorkGoalFirstSinBackendV0(t *testing.T) {
	tracker := NewStatusTrackerV0(ConfigV0{
		Addr: "127.0.0.1:8787",
		EffectiveConfig: ServerEffectiveConfigV0{
			Diagnostics: []ServerDiagnosticV0{{
				Code:    "external_work_goal_backend_required",
				Scope:   "external_work",
				Message: "external_work goal-first no ejecutable sin ORQUESTA_CODEX_GOAL_BACKEND",
				EvidenceRefs: []string{
					"evidence-ref-server-external-work-goal-backend-required",
				},
			}},
		},
	}, time.Now().UTC())
	tracker.MarkServingV0("127.0.0.1:8787", time.Now().UTC())
	tracker.MarkStartupReadyV0(StartupCheckResultV0{
		Status: StartupCheckStatusReadyV0,
		Ready:  true,
	}, time.Now().UTC())
	handler := NewHandlerV0(HandlerConfigV0{Tracker: tracker})

	health := httptest.NewRecorder()
	handler.ServeHTTP(health, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if health.Code != http.StatusOK {
		t.Fatalf("healthz status=%d body=%s", health.Code, health.Body.String())
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, ServerReadinessEndpointV0, nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("readiness status=%d body=%s", rec.Code, rec.Body.String())
	}
	var readiness ServerReadinessV0
	if err := json.Unmarshal(rec.Body.Bytes(), &readiness); err != nil {
		t.Fatalf("json=%v body=%s", err, rec.Body.String())
	}
	if readiness.Ready ||
		!readiness.StartupReady ||
		readiness.LivenessStatus != "ok" ||
		readiness.StartupStatus != "startup_degraded_external_work_goal_backend_required" ||
		len(readiness.Diagnostics) != 1 ||
		readiness.Diagnostics[0].Code != "external_work_goal_backend_required" ||
		!containsServerStringForTestV0(readiness.EvidenceRefs, "evidence-ref-server-external-work-goal-backend-required") {
		t.Fatalf("readiness=%+v", readiness)
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

func assertLegacyServerStatusAliasV0(t *testing.T, handler http.Handler) {
	t.Helper()
	versioned := httptest.NewRecorder()
	handler.ServeHTTP(versioned, httptest.NewRequest(http.MethodGet, ServerStatusEndpointV0, nil))
	legacy := httptest.NewRecorder()
	handler.ServeHTTP(legacy, httptest.NewRequest(http.MethodGet, ServerStatusLegacyEndpointV0, nil))
	if legacy.Code != http.StatusOK {
		t.Fatalf("legacy status=%d body=%s", legacy.Code, legacy.Body.String())
	}
	if strings.TrimSpace(legacy.Body.String()) != strings.TrimSpace(versioned.Body.String()) {
		t.Fatalf("legacy body difiere de versionada\nlegacy=%s\nversioned=%s", legacy.Body.String(), versioned.Body.String())
	}
	if got := legacy.Header().Get(ServerStatusCanonicalHeaderV0); got != ServerStatusEndpointV0 {
		t.Fatalf("canonical header=%q", got)
	}
	if got := legacy.Header().Get(ServerStatusCompatibilityHeaderV0); got != ServerStatusCompatibilityLegacyV0 {
		t.Fatalf("compat header=%q", got)
	}
	if got := legacy.Header().Get(ServerStatusOwnerHeaderV0); got != ServerStatusOwnerServerV0 {
		t.Fatalf("owner header=%q", got)
	}
	if got := legacy.Header().Get(ServerStatusSunsetHeaderV0); got != ServerStatusSunsetNoNewUseV0 {
		t.Fatalf("sunset header=%q", got)
	}
	if got := legacy.Header().Get("Deprecation"); got != "true" {
		t.Fatalf("deprecation=%q", got)
	}
	if !strings.Contains(legacy.Header().Get("Link"), ServerStatusEndpointV0) ||
		!strings.Contains(legacy.Header().Get("Warning"), ServerStatusEndpointV0) {
		t.Fatalf("headers legacy insuficientes: link=%q warning=%q", legacy.Header().Get("Link"), legacy.Header().Get("Warning"))
	}
}
