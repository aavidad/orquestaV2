package orquestaserver

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	orquestaobservability "orquesta/modulos/orquesta-observability"
)

func TestWriteServerJSONResponseV0EncodeFailAntesDeHeaderSeguroV0(t *testing.T) {
	tracker := NewStatusTrackerV0(ConfigV0{}, time.Date(2026, 5, 26, 10, 0, 0, 0, time.UTC))
	handler := handlerV0{config: HandlerConfigV0{Tracker: tracker}}
	rec := httptest.NewRecorder()

	handler.writeJSONV0(rec, http.StatusOK, map[string]any{
		"token": "secret-token",
		"bad":   func() {},
	})

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, serverResponseEncodeFailedCodeV0) ||
		strings.Contains(body, "secret-token") ||
		strings.Contains(body, "token") {
		t.Fatalf("body inseguro=%q", body)
	}
	state := tracker.SnapshotV0()
	if state.ResponseWriteLastCode != serverResponseEncodeFailedCodeV0 ||
		state.ResponseWriteLastStage != "encode_before_header" ||
		len(state.RecentErrors) == 0 ||
		state.RecentErrors[0].Message != serverResponseEncodeFailedCodeV0 {
		t.Fatalf("state=%+v", state)
	}
}

func TestAuditHTTPHandlerV0RegistraWriteFailBeforeHeaderV0(t *testing.T) {
	sink := &memoryAuditSinkV0{}
	runtime := newRuntimeForResponseWriteTestV0(t, sink, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("domain-ok"))
	}))

	runtime.HandlerV0().ServeHTTP(newFailingResponseWriterV0(), httptest.NewRequest(http.MethodGet, "/delegated", nil))

	assertResponseWriteFailureObservedV0(t, runtime.StateV0(), sink, "body_before_header")
}

func TestAuditHTTPHandlerV0RegistraWriteFailAfterHeaderV0(t *testing.T) {
	sink := &memoryAuditSinkV0{}
	runtime := newRuntimeForResponseWriteTestV0(t, sink, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte("domain-ok"))
	}))

	runtime.HandlerV0().ServeHTTP(newFailingResponseWriterV0(), httptest.NewRequest(http.MethodGet, "/delegated", nil))

	assertResponseWriteFailureObservedV0(t, runtime.StateV0(), sink, "body_after_header")
}

func TestHandlerV0ResponseWriteOKLimpiaFalloVigenteV0(t *testing.T) {
	tracker := NewStatusTrackerV0(ConfigV0{}, time.Date(2026, 5, 26, 10, 0, 0, 0, time.UTC))
	handler := handlerV0{config: HandlerConfigV0{Tracker: tracker}}
	tracker.MarkResponseWriteFailedV0("body_after_header", time.Date(2026, 5, 26, 10, 1, 0, 0, time.UTC))

	handler.writeJSONV0(httptest.NewRecorder(), http.StatusOK, map[string]string{"status": "ok"})

	state := tracker.SnapshotV0()
	if state.ResponseWriteFailures != 1 ||
		state.ResponseWriteLastFailedAt == "" ||
		state.ResponseWriteLastCode != "" ||
		state.ResponseWriteLastStage != "" {
		t.Fatalf("state=%+v", state)
	}
	if blockers := residentOperationalBlockersV0(state); diagnosticoContainsBlockerForResponseWriteTestV0(blockers) {
		t.Fatalf("response_write blocker no recuperado: %+v", blockers)
	}
}

func TestAuditHTTPHandlerV0ResponseWriteOKPersisteRecuperacionV0(t *testing.T) {
	sink := &memoryAuditSinkV0{}
	store := &memoryStateStoreV0{}
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir: t.TempDir(),
	}, RuntimeDepsV0{
		AppHandler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("domain-ok"))
		}),
		StateStore: store,
		AuditSink:  sink,
		Clock:      fixedClockV0{now: time.Date(2026, 5, 26, 10, 0, 0, 0, time.UTC)},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}
	runtime.tracker.MarkResponseWriteFailedV0("body_after_header", time.Date(2026, 5, 26, 10, 1, 0, 0, time.UTC))

	runtime.HandlerV0().ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/delegated", nil))

	state := runtime.StateV0()
	if state.ResponseWriteFailures != 1 ||
		state.ResponseWriteLastCode != "" ||
		state.ResponseWriteLastStage != "" ||
		store.last.ResponseWriteLastCode != "" {
		t.Fatalf("state=%+v stored=%+v", state, store.last)
	}
}

func newRuntimeForResponseWriteTestV0(t *testing.T, sink *memoryAuditSinkV0, app http.Handler) *RuntimeV0 {
	t.Helper()
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir: t.TempDir(),
	}, RuntimeDepsV0{
		AppHandler: app,
		StateStore: &memoryStateStoreV0{},
		AuditSink:  sink,
		Clock:      fixedClockV0{now: time.Date(2026, 5, 26, 10, 0, 0, 0, time.UTC)},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}
	return runtime
}

func assertResponseWriteFailureObservedV0(
	t *testing.T,
	state StateV0,
	sink *memoryAuditSinkV0,
	stage string,
) {
	t.Helper()
	if state.ResponseWriteFailures != 1 ||
		state.ResponseWriteLastCode != serverResponseWriteFailedCodeV0 ||
		state.ResponseWriteLastStage != stage {
		t.Fatalf("state=%+v", state)
	}
	if len(sink.events) == 0 {
		t.Fatalf("sin audit events")
	}
	last := sink.events[len(sink.events)-1]
	if last.Event != "http_request" ||
		last.Status != serverResponseWriteFailedCodeV0 ||
		last.Payload["response_status"] != serverResponseWriteFailedCodeV0 {
		t.Fatalf("audit=%+v", last)
	}
	responseWrite, ok := last.Payload["response_write"].(map[string]interface{})
	if !ok ||
		responseWrite["status"] != "failed" ||
		responseWrite["code"] != serverResponseWriteFailedCodeV0 ||
		responseWrite["stage"] != stage {
		t.Fatalf("response_write=%+v", last.Payload["response_write"])
	}
}

type failingResponseWriterV0 struct {
	header http.Header
}

func newFailingResponseWriterV0() *failingResponseWriterV0 {
	return &failingResponseWriterV0{header: http.Header{}}
}

func (writer *failingResponseWriterV0) Header() http.Header {
	return writer.header
}

func (writer *failingResponseWriterV0) WriteHeader(statusCode int) {}

func (writer *failingResponseWriterV0) Write(_ []byte) (int, error) {
	return 0, errors.New("write failed")
}

func diagnosticoContainsBlockerForResponseWriteTestV0(blockers []orquestaobservability.DiagnosticoBloqueoV0) bool {
	for _, blocker := range blockers {
		if blocker.BlockerRef == "blocker-ref-server-response-write" {
			return true
		}
	}
	return false
}
