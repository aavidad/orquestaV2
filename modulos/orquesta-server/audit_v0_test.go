package orquestaserver

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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
	if events[1].Payload["result"] == nil {
		t.Fatalf("sin result auditado: %+v", events[1])
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
		last.Payload["raw_query"] != "debug=1" {
		t.Fatalf("last=%+v", last)
	}
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
