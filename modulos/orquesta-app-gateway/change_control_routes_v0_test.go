package orquestaappgateway

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestAppChangeAPIDelegaEnMCPRequestChangeSinCmdDBRuntimeV0(t *testing.T) {
	executor := &recordingRequestAppChangeExecutorV0{}
	handler := NewHTTPHandlerV0(ConfigV0{
		RequestAppChange: executor,
		Timeout:          time.Second,
	})
	body := strings.NewReader(`{
		"app_change_request":{
			"run_ref":"run-app-gateway-change-001",
			"change_ref":"change-app-gateway-web-001",
			"user_intent":"Cambiar la web para mostrar vista semanal"
		}
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v0/apps/agenda-equipo/changes", body)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if executor.Input.AppChangeRequest.AppRef != "agenda-equipo" ||
		executor.Input.AppChangeRequest.ChangeRef != "change-app-gateway-web-001" {
		t.Fatalf("input=%+v", executor.Input.AppChangeRequest)
	}
}

func TestRunControlYRunQueueAPIDeleganEnMCPPortsV0(t *testing.T) {
	control := &recordingRunControlExecutorV0{}
	queue := &recordingRunQueuePriorityExecutorV0{}
	handler := NewHTTPHandlerV0(ConfigV0{
		RunControl:       control,
		RunQueuePriority: queue,
		Timeout:          time.Second,
	})

	controlRec := httptest.NewRecorder()
	controlReq := httptest.NewRequest(http.MethodPost, "/api/v0/runs/control", strings.NewReader(`{
		"action":"pause",
		"run_ref":"run-app-gateway-control-001"
	}`))
	controlReq.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(controlRec, controlReq)

	if controlRec.Code != http.StatusOK || control.Input.Action != "pause" {
		t.Fatalf("control status=%d input=%+v body=%s", controlRec.Code, control.Input, controlRec.Body.String())
	}

	queueRec := httptest.NewRecorder()
	queueReq := httptest.NewRequest(http.MethodPost, "/api/v0/runs/queue/priority", strings.NewReader(`{
		"action":"set_priority",
		"run_ref":"run-app-gateway-control-001",
		"priority_score":75
	}`))
	queueReq.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(queueRec, queueReq)

	if queueRec.Code != http.StatusOK ||
		queue.Input.Action != "set_priority" ||
		queue.Input.PriorityScore != 75 {
		t.Fatalf("queue status=%d input=%+v body=%s", queueRec.Code, queue.Input, queueRec.Body.String())
	}
}

func TestRunQueuePageDelegaEnAPIInternaSinCmdDBRuntimeV0(t *testing.T) {
	queue := &recordingRunQueuePriorityExecutorV0{}
	handler := NewHTTPHandlerV0(ConfigV0{
		RunQueuePriority: queue,
		Timeout:          time.Second,
	})
	req := httptest.NewRequest(http.MethodGet, "/run-queue?action=rank&queue_ref=global&limit=5", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if queue.Input.Action != "rank" ||
		queue.Input.QueueRef != "global" ||
		queue.Input.Limit != 5 {
		t.Fatalf("input=%+v", queue.Input)
	}
}

func TestRunControlPageDelegaEnAPIInternaSinCmdDBRuntimeV0(t *testing.T) {
	control := &recordingRunControlExecutorV0{}
	handler := NewHTTPHandlerV0(ConfigV0{
		RunControl: control,
		Timeout:    time.Second,
	})
	values := url.Values{}
	values.Set("action", "stop")
	values.Set("run_ref", "run-app-gateway-control-page-001")
	values.Set("forced", "true")
	req := httptest.NewRequest(http.MethodPost, "/run-control", strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if control.Input.Action != "stop" ||
		control.Input.RunRef != "run-app-gateway-control-page-001" ||
		!control.Input.Forced {
		t.Fatalf("input=%+v", control.Input)
	}
}

func TestAppChangePageDelegaEnAPIInternaSinCmdDBRuntimeV0(t *testing.T) {
	executor := &recordingRequestAppChangeExecutorV0{}
	handler := NewHTTPHandlerV0(ConfigV0{
		RequestAppChange: executor,
		Timeout:          time.Second,
	})
	values := url.Values{}
	values.Set("locale", "es")
	values.Set("run_ref", "run-app-gateway-change-page-001")
	values.Set("app_ref", "agenda-equipo")
	values.Set("change_ref", "change-app-gateway-page-001")
	values.Set("user_intent", "Cambiar la web para vista semanal")
	req := httptest.NewRequest(http.MethodPost, "/app-change", strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if executor.Input.AppChangeRequest.ChangeRef != "change-app-gateway-page-001" ||
		!strings.Contains(rec.Body.String(), "question-ref-app-gateway-change-001") {
		t.Fatalf("input=%+v body=%s", executor.Input, rec.Body.String())
	}
}
