package orquestamcp

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

func TestMCPDomainWorkHTTPHandlerV0PostDelegaYPropagaCorrelacion(t *testing.T) {
	creator := &fakeMCPDomainWorkCreatorV0{
		job: orquestadomainwork.DomainWorkJobV0{
			SchemaVersion: orquestadomainwork.DomainWorkJobSchemaV0,
			Status:        orquestadomainwork.DomainWorkStatusAcceptedV0,
			JobRef:        "job-domain-http-001",
			CorrelationID: "corr-domain-http-001",
		},
	}
	body, err := json.Marshal(MCPDomainWorkToolInputV0{
		RequestID:     "req-domain-http-001",
		CorrelationID: "corr-domain-http-001",
		Action:        MCPDomainWorkActionCreateJobV0,
	})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	request := httptest.NewRequest(http.MethodPost, MCPDomainWorkHTTPPathV0, bytes.NewReader(body))
	response := httptest.NewRecorder()

	NewMCPDomainWorkHTTPHandlerV0(MCPDomainWorkToolExecutorV0{JobCreator: creator}).ServeHTTP(response, request)

	if response.Code != http.StatusOK ||
		response.Header().Get("X-Correlation-ID") != "corr-domain-http-001" {
		t.Fatalf("http code=%d headers=%v body=%s", response.Code, response.Header(), response.Body.String())
	}
	var result MCPDomainWorkToolResultV0
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Estado != MCPDomainWorkEstadoOKV0 ||
		result.Job == nil ||
		result.Job.JobRef != "job-domain-http-001" {
		t.Fatalf("result=%+v", result)
	}
}
