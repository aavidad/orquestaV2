package orquestaserver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestStatusTrackerExternalBridgeLifecycleV0(t *testing.T) {
	tracker := NewStatusTrackerV0(ConfigV0{Addr: "127.0.0.1:8787"}, time.Now().UTC())
	now := time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC)

	state := tracker.MarkExternalBridgeLifecycleV0(ExternalBridgeLifecycleUpdateV0{
		Component:    "opes_bridge_loop",
		Status:       "degraded",
		TickNumber:   3,
		ErrorCode:    "http_503",
		Filters:      []string{"job_type=plan_temario", "job_ref=configured"},
		Counters:     map[string]int{"seen": 2, "submitted": 0, "errors": 1},
		EvidenceRefs: []string{"run-ref-opes-a1-t002-finalpkg-20260612"},
	}, now)

	if state.ExternalBridgeStatus != "degraded" ||
		state.ExternalBridgeLastTickRef != "opes_bridge_loop-tick-ref-3" ||
		state.ExternalBridgeLastError != "http_503" ||
		state.ExternalBridgeErrorTicks != 1 ||
		len(state.ExternalBridgeEvidenceRefs) != 1 ||
		state.ExternalBridgeEvidenceRefs[0] != "run-ref-opes-a1-t002-finalpkg-20260612" {
		t.Fatalf("state=%+v", state)
	}
	public := NewServerPublicStatusV0(state)
	if len(public.ExternalBridgeEvidenceRefs) != 1 ||
		public.ExternalBridgeEvidenceRefs[0] != "run-ref-opes-a1-t002-finalpkg-20260612" {
		t.Fatalf("public=%+v", public)
	}
	body, err := json.Marshal(public)
	if err != nil {
		t.Fatalf("json=%v", err)
	}
	for _, forbidden := range []string{"token", "secret", "private?", "127.0.0.1/private"} {
		if strings.Contains(string(body), forbidden) {
			t.Fatalf("status filtra %q en %s", forbidden, string(body))
		}
	}
}

func TestReadinessDistingueServidorListoYBridgeDegradadoV0(t *testing.T) {
	tracker := NewStatusTrackerV0(ConfigV0{Addr: "127.0.0.1:8787"}, time.Now().UTC())
	tracker.MarkServingV0("127.0.0.1:8787", time.Now().UTC())
	tracker.MarkStartupReadyV0(StartupCheckResultV0{
		Status: StartupCheckStatusReadyV0,
		Ready:  true,
	}, time.Now().UTC())
	tracker.MarkExternalBridgeLifecycleV0(ExternalBridgeLifecycleUpdateV0{
		Component: "opes_bridge_loop",
		Status:    "degraded",
		ErrorCode: "external_bridge_tick_error",
	}, time.Now().UTC())
	handler := NewHandlerV0(HandlerConfigV0{Tracker: tracker})

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, ServerReadinessEndpointV0, nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var readiness ServerReadinessV0
	if err := json.Unmarshal(rec.Body.Bytes(), &readiness); err != nil {
		t.Fatalf("json=%v body=%s", err, rec.Body.String())
	}
	if !readiness.Ready || readiness.ExternalBridgeReady ||
		readiness.ExternalBridgeStatus != "degraded" {
		t.Fatalf("readiness=%+v", readiness)
	}
}

func TestRuntimeExternalBridgeLifecycleAuditaErroresV0(t *testing.T) {
	audit := &memoryAuditSinkV0{}
	store := &bridgeMemoryStateStoreV0{}
	runtime, err := NewRuntimeV0(ConfigV0{
		Addr:     "127.0.0.1:8787",
		StateDir: t.TempDir(),
	}, RuntimeDepsV0{
		StateStore: store,
		AuditSink:  audit,
		Clock:      bridgeFixedClockV0{now: time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC)},
	})
	if err != nil {
		t.Fatalf("runtime=%v", err)
	}

	runtime.MarkExternalBridgeLifecycleV0(context.Background(), ExternalBridgeLifecycleUpdateV0{
		Component:  "opes_bridge_loop",
		Status:     "timeout",
		TickNumber: 1,
		ErrorCode:  "external_bridge_tick_timeout",
		Filters:    []string{"job_ref=configured"},
	})

	if len(audit.events) != 1 || audit.events[0].Event != "external_bridge_tick" ||
		audit.events[0].Error != "external_bridge_tick_timeout" {
		t.Fatalf("audit=%+v", audit.events)
	}
	if store.last.ExternalBridgeStatus != "timeout" ||
		store.last.ExternalBridgeLastError != "external_bridge_tick_timeout" {
		t.Fatalf("state=%+v", store.last)
	}
}

type memoryAuditSinkV0 struct {
	events []AuditEventV0
}

func (sink *memoryAuditSinkV0) AppendAuditEventV0(_ context.Context, event AuditEventV0) error {
	sink.events = append(sink.events, event)
	return nil
}

type bridgeMemoryStateStoreV0 struct {
	last StateV0
}

func (store *bridgeMemoryStateStoreV0) SaveServerStateV0(_ context.Context, state StateV0) error {
	store.last = state
	return nil
}

func (store *bridgeMemoryStateStoreV0) LoadServerStateV0(context.Context) (StateV0, error) {
	return store.last, nil
}

type bridgeFixedClockV0 struct {
	now time.Time
}

func (clock bridgeFixedClockV0) Now() time.Time {
	return clock.now
}
