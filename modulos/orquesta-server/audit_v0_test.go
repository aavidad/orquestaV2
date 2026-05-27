package orquestaserver

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	orquestarunsupervisor "orquesta/modulos/orquesta-run-supervisor"
)

func TestFileAuditSinkV0AppendJSONLCompletoV0(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit", DefaultAuditFileV0)
	sink, err := NewFileAuditSinkV0(path)
	if err != nil {
		t.Fatalf("NewFileAuditSinkV0: %v", err)
	}
	if err := sink.AppendAuditEventV0(context.Background(), AuditEventV0{
		Event:      "supervisor_tick_result",
		OccurredAt: "2026-05-24T10:00:00Z",
		Status:     "ok",
		Payload: map[string]interface{}{
			"run_ref": "run-ref-001",
			"diagnostics": []map[string]interface{}{{
				"kind":                 "outbox_batch_dispatch",
				"status":               "no_pending",
				"pending_outbox_count": float64(2),
			}},
		},
	}); err != nil {
		t.Fatalf("append audit: %v", err)
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open audit: %v", err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		t.Fatalf("sin linea audit")
	}
	var event AuditEventV0
	if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
		t.Fatalf("json audit: %v", err)
	}
	if event.SchemaVersion != AuditSchemaVersionV0 ||
		event.Event != "supervisor_tick_result" ||
		event.Payload["run_ref"] != "run-ref-001" {
		t.Fatalf("event=%+v", event)
	}
	assertServerDurableFilePolicyV0(t, filepath.Dir(path), filepath.Base(path))
}

func TestRuntimeV0SupervisorEscribeAuditoriaJSONLV0(t *testing.T) {
	stateDir := t.TempDir()
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir:     stateDir,
		TickInterval: time.Hour,
		SupervisorCommand: orquestarunsupervisor.RunSupervisorCommandV0{
			QueueRef: "global",
			MaxTicks: 1,
		},
	}, RuntimeDepsV0{
		Supervisor: &fakeSupervisorV0{},
		StateStore: &memoryStateStoreV0{},
		Clock:      fixedClockV0{now: time.Date(2026, 5, 24, 10, 0, 0, 0, time.UTC)},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}
	runtime.runSupervisorTickV0(context.Background())
	events := readAuditEventsForTestV0(t, AuditPathV0(runtime.config))
	if len(events) < 2 {
		t.Fatalf("events=%+v", events)
	}
	if events[0].Event != "supervisor_tick_start" ||
		events[1].Event != "supervisor_tick_result" ||
		events[1].Status != "ok" {
		t.Fatalf("events=%+v", events)
	}
	if events[1].Payload["result_summary"] == nil ||
		events[1].Payload["result"] != nil ||
		events[1].Payload["command"] != nil {
		t.Fatalf("payload audit supervisor no compacto: %+v", events[1].Payload)
	}
}

func TestRuntimeV0HandlerAuditaPeticionesHTTPV0(t *testing.T) {
	stateDir := t.TempDir()
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir: stateDir,
	}, RuntimeDepsV0{
		StateStore: &memoryStateStoreV0{},
		Clock:      fixedClockV0{now: time.Date(2026, 5, 24, 10, 0, 0, 0, time.UTC)},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz?debug=1", nil)
	runtime.HandlerV0().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	events := readAuditEventsForTestV0(t, AuditPathV0(runtime.config))
	last := events[len(events)-1]
	if last.Event != "http_request" ||
		last.Status != "200" ||
		last.Payload["path"] != "/healthz" ||
		!auditPayloadContainsQueryKeyForTestV0(last.Payload["query_keys"], "debug") ||
		last.Payload["raw_query"] != nil {
		t.Fatalf("last=%+v", last)
	}
}

func TestRuntimeV0HandlerAuditaQueryPublicaAcotadaYRedactadaV0(t *testing.T) {
	stateDir := t.TempDir()
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir: stateDir,
	}, RuntimeDepsV0{
		StateStore: &memoryStateStoreV0{},
		Clock:      fixedClockV0{now: time.Date(2026, 5, 24, 10, 0, 0, 0, time.UTC)},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz?token=secret&raw_query=x&debug=1", nil)
	runtime.HandlerV0().ServeHTTP(rec, req)

	events := readAuditEventsForTestV0(t, AuditPathV0(runtime.config))
	last := events[len(events)-1]
	encoded, _ := json.Marshal(last.Payload)
	body := string(encoded)
	if last.Payload["raw_query"] != nil ||
		strings.Contains(body, "secret") ||
		strings.Contains(body, "raw_query") ||
		!strings.Contains(body, "sensitive_query_key_redacted") ||
		!strings.Contains(body, "debug") {
		t.Fatalf("payload query sin redaccion: %s", body)
	}
}

func TestAuditQueryKeysV0NoParseaRawQueryExcesivaV0(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/healthz?q="+strings.Repeat("x", auditQueryRawMaxBytesV0+1), nil)

	keys := auditQueryKeysV0(req)

	if len(keys) != 1 || keys[0] != "query_too_large" {
		t.Fatalf("keys=%+v", keys)
	}
}

func TestRuntimeV0AuditEventNoSilenciaFalloDeSinkV0(t *testing.T) {
	store := &memoryStateStoreV0{}
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir:      t.TempDir(),
		AuditDisabled: true,
	}, RuntimeDepsV0{
		StateStore: store,
		AuditSink:  failingAuditSinkV0{},
		Clock:      fixedClockV0{now: time.Date(2026, 5, 24, 10, 0, 0, 0, time.UTC)},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}

	runtime.auditEventV0(context.Background(), "supervisor_tick_result", "ok", "", map[string]interface{}{"run_ref": "run-ref-001"})

	if store.last.AuditStatus != "degraded" ||
		store.last.AuditFailures != 1 ||
		store.last.AuditLastCode != auditWriteFailureCodeV0 ||
		store.last.AuditLastEvent != "supervisor_tick_result" ||
		store.last.AuditLastSeverity != "warning" {
		t.Fatalf("fallo audit no visible en proyeccion compacta: %+v", store.last)
	}
	if len(store.last.RecentErrors) != 1 ||
		store.last.RecentErrors[0].Code != auditWriteFailureCodeV0 ||
		store.last.RecentErrors[0].Message != auditWriteFailureCodeV0 {
		t.Fatalf("recent_errors no registra audit failure: %+v", store.last.RecentErrors)
	}
	body, _ := json.Marshal(store.last)
	for _, forbidden := range []string{"sink roto", "runtime-secret", "state.json", "HOME", "token"} {
		if strings.Contains(string(body), forbidden) {
			t.Fatalf("audit failure filtra detalle privado %q: %s", forbidden, string(body))
		}
	}
	readiness := NewServerReadinessV0(store.last)
	if readiness.Ready || readiness.LivenessStatus != "ok" {
		t.Fatalf("readiness/liveness inesperado: %+v", readiness)
	}
}

func TestRuntimeV0AuditFailureNoRecursivoSiStateStoreFallaV0(t *testing.T) {
	store := &failingStateStoreV0{err: errors.New("write failed at /tmp/runtime-secret/state.json")}
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir:      t.TempDir(),
		AuditDisabled: true,
	}, RuntimeDepsV0{
		StateStore: store,
		AuditSink:  failingAuditSinkV0{},
		Clock:      fixedClockV0{now: time.Date(2026, 5, 24, 10, 0, 0, 0, time.UTC)},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}

	runtime.auditEventV0(context.Background(), "http_request?token=raw", "500", "", nil)

	state := runtime.StateV0()
	if state.AuditFailures != 1 ||
		state.AuditLastEvent != "http_request" ||
		state.StatePersistStatus != "degraded" ||
		state.StatePersistLastTransition != "audit_write_failed" {
		t.Fatalf("estado degradado=%+v", state)
	}
	body, _ := json.Marshal(state)
	for _, forbidden := range []string{"runtime-secret", "state.json", "sink roto", "token=raw", "tokenraw"} {
		if strings.Contains(string(body), forbidden) {
			t.Fatalf("estado filtra detalle privado %q: %s", forbidden, string(body))
		}
	}
}

type failingAuditSinkV0 struct{}

func (failingAuditSinkV0) AppendAuditEventV0(context.Context, AuditEventV0) error {
	return errors.New("audit sink roto")
}

func auditPayloadContainsQueryKeyForTestV0(value any, want string) bool {
	keys, ok := value.([]interface{})
	if !ok {
		return false
	}
	for _, key := range keys {
		if key == want {
			return true
		}
	}
	return false
}

func readAuditEventsForTestV0(t *testing.T, path string) []AuditEventV0 {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open audit: %v", err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	var events []AuditEventV0
	for scanner.Scan() {
		var event AuditEventV0
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			t.Fatalf("json audit: %v", err)
		}
		events = append(events, event)
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan audit: %v", err)
	}
	return events
}
