package orquestaweb

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestRunControlWebEndpointV0POSTFormDelegaEnCliente(t *testing.T) {
	client := &recordingRunControlWebClientV0{}
	endpoint := NewRunControlWebEndpointV0(client)
	values := url.Values{}
	values.Set("locale", "es")
	values.Set("action", "cancel")
	values.Set("run_ref", "run-web-control-form-001")
	values.Set("forced", "true")
	values.Add("evidence_refs", "ev-001")
	req := httptest.NewRequest(http.MethodPost, WebRunControlPageEndpointV0, strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	endpoint.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if client.Command.Action != "cancel" ||
		client.Command.RunRef != "run-web-control-form-001" ||
		!client.Command.Forced ||
		len(client.Command.EvidenceRefs) != 1 {
		t.Fatalf("command=%+v", client.Command)
	}
	if !strings.Contains(rec.Body.String(), "cancel_requested") {
		t.Fatalf("body=%s", rec.Body.String())
	}
}

func TestRunControlWebEndpointV0POSTJSONDelegaEnCliente(t *testing.T) {
	client := &recordingRunControlWebClientV0{}
	endpoint := NewRunControlWebEndpointV0(client)
	req := httptest.NewRequest(http.MethodPost, WebRunControlPageEndpointV0, strings.NewReader(`{
		"locale":"es",
		"action":"pause",
		"run_ref":"run-web-control-json-001",
		"requested_by":"operador"
	}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	endpoint.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if client.Command.Action != "pause" ||
		client.Command.RunRef != "run-web-control-json-001" ||
		client.Command.RequestedBy != "operador" {
		t.Fatalf("command=%+v", client.Command)
	}
	var page WebRunControlPageV0
	if err := json.NewDecoder(rec.Body).Decode(&page); err != nil {
		t.Fatalf("decode page: %v", err)
	}
	if page.SchemaVersion != "web_run_control_page.v0" ||
		page.ViewModel.Estado != WebRunControlEstadoOKV0 {
		t.Fatalf("page=%+v", page)
	}
}

func TestRunControlWebEndpointV0RechazaMetodoNoMutador(t *testing.T) {
	endpoint := NewRunControlWebEndpointV0(&recordingRunControlWebClientV0{})
	req := httptest.NewRequest(http.MethodGet, WebRunControlPageEndpointV0, nil)
	rec := httptest.NewRecorder()

	endpoint.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestRunControlWebEndpointV0GETHTMLParaNavegador(t *testing.T) {
	endpoint := NewRunControlWebEndpointV0(&recordingRunControlWebClientV0{})
	req := httptest.NewRequest(http.MethodGet, WebRunControlPageEndpointV0, nil)
	req.Header.Set("Accept", "text/html")
	rec := httptest.NewRecorder()

	endpoint.ServeHTTP(rec, req)

	body := rec.Body.String()
	if rec.Code != http.StatusOK ||
		rec.Header().Get("Content-Type") != WebHTMLContentTypeHeaderV0 ||
		!strings.Contains(body, "run-control-root") ||
		!strings.Contains(body, "postJSON('/run-control'") {
		t.Fatalf("html control invalido status=%d headers=%v body=%s", rec.Code, rec.Header(), body)
	}
}

type recordingRunControlWebClientV0 struct {
	Command WebRunControlCommandV0
}

func (client *recordingRunControlWebClientV0) EnviarRunControl(
	_ context.Context,
	command WebRunControlCommandV0,
) (WebRunControlViewModelV0, error) {
	client.Command = normalizeRunControlCommandV0(command)
	return WebRunControlViewModelV0{
		SchemaVersion: "web_run_control.v0",
		Locale:        client.Command.Locale,
		Estado:        WebRunControlEstadoOKV0,
		Action:        client.Command.Action,
		RunRef:        client.Command.RunRef,
		Status:        client.Command.Action + "_requested",
		Forced:        client.Command.Forced,
	}, nil
}
