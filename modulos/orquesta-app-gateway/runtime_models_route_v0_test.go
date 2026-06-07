package orquestaappgateway

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func TestRuntimeModelsAPIDelegaEnPuertoOptInV0(t *testing.T) {
	models := &recordingRuntimeModelsPortV0{}
	handler := NewHTTPHandlerV0(ConfigV0{
		RuntimeModels: models,
		Timeout:       time.Second,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v0/runtime/models", strings.NewReader(`{
		"action":"serve",
		"endpoint_ref":"endpoint-ref-ollama-local-001",
		"model":"qwen2.5:7b"
	}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK ||
		models.serve.EndpointRef != "endpoint-ref-ollama-local-001" ||
		models.serve.Model != "qwen2.5:7b" {
		t.Fatalf("status=%d input=%+v body=%s", rec.Code, models.serve, rec.Body.String())
	}
}

func TestRuntimeModelsAPIQuedaNoConfiguradaSinPuertoV0(t *testing.T) {
	handler := NewHTTPHandlerV0(ConfigV0{Timeout: time.Second})
	req := httptest.NewRequest(http.MethodPost, "/api/v0/runtime/models", strings.NewReader(`{"action":"list"}`))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

type recordingRuntimeModelsPortV0 struct {
	list   orquestaruntime.RuntimeModelListRequestV0
	status orquestaruntime.RuntimeModelListRequestV0
	pull   orquestaruntime.RuntimeModelActionRequestV0
	serve  orquestaruntime.RuntimeModelActionRequestV0
	stop   orquestaruntime.RuntimeModelActionRequestV0
}

func (port *recordingRuntimeModelsPortV0) ListRuntimeModelsV0(
	_ context.Context,
	request orquestaruntime.RuntimeModelListRequestV0,
) (orquestaruntime.RuntimeModelListResultV0, error) {
	port.list = request
	return orquestaruntime.RuntimeModelListResultV0{ProviderRef: "ollama"}, nil
}

func (port *recordingRuntimeModelsPortV0) RuntimeModelStatusV0(
	_ context.Context,
	request orquestaruntime.RuntimeModelListRequestV0,
) (orquestaruntime.RuntimeModelListResultV0, error) {
	port.status = request
	return orquestaruntime.RuntimeModelListResultV0{ProviderRef: "ollama"}, nil
}

func (port *recordingRuntimeModelsPortV0) PullRuntimeModelV0(
	_ context.Context,
	request orquestaruntime.RuntimeModelActionRequestV0,
) (orquestaruntime.RuntimeModelActionResultV0, error) {
	port.pull = request
	return recordingRuntimeModelsActionResultV0(request, "pulled"), nil
}

func (port *recordingRuntimeModelsPortV0) ServeRuntimeModelV0(
	_ context.Context,
	request orquestaruntime.RuntimeModelActionRequestV0,
) (orquestaruntime.RuntimeModelActionResultV0, error) {
	port.serve = request
	return recordingRuntimeModelsActionResultV0(request, "served"), nil
}

func (port *recordingRuntimeModelsPortV0) StopRuntimeModelV0(
	_ context.Context,
	request orquestaruntime.RuntimeModelActionRequestV0,
) (orquestaruntime.RuntimeModelActionResultV0, error) {
	port.stop = request
	return recordingRuntimeModelsActionResultV0(request, "stopped"), nil
}

func recordingRuntimeModelsActionResultV0(
	request orquestaruntime.RuntimeModelActionRequestV0,
	status string,
) orquestaruntime.RuntimeModelActionResultV0 {
	return orquestaruntime.RuntimeModelActionResultV0{
		ProviderRef: "ollama",
		EndpointRef: request.EndpointRef,
		Model:       request.Model,
		Accepted:    true,
		Status:      status,
	}
}
