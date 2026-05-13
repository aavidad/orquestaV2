package orquestaopesconnector

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

func TestRESTClientV0CreateDomainWorkJobCreaJobExterno(t *testing.T) {
	var received opesCreateJobRequestV0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/jobs" || r.Method != http.MethodPost {
			t.Fatalf("request inesperada %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":              "job-ref-opes-001",
			"status":          "accepted",
			"correlation_id":  "corr-opes-001",
			"idempotency_key": "idem-opes-001",
			"external_refs":   map[string]string{"run_ref": "run-ref-001"},
			"created":         true,
			"job": map[string]any{
				"id":             "job-ref-opes-001",
				"type":           "draft_content_block",
				"status":         "pending",
				"execution_mode": "external",
			},
		})
	}))
	defer server.Close()

	client := NewRESTClientV0(RESTClientConfigV0{BaseURL: server.URL})
	result, err := client.CreateDomainWorkJobV0(context.Background(), opesJobRequestForTestV0())

	if err != nil {
		t.Fatalf("CreateDomainWorkJobV0: %v", err)
	}
	if result.Status != orquestadomainwork.DomainWorkStatusAcceptedV0 ||
		result.JobRef != "job-ref-opes-001" ||
		result.WorkKind != "draft_content_block" {
		t.Fatalf("result=%+v", result)
	}
	if received.JobType != "draft_content_block" ||
		received.CorrelationID != "corr-opes-001" ||
		received.IdempotencyKey != "idem-opes-001" ||
		received.RequestedBy != "orquesta" ||
		received.MaxAttempts != 1 ||
		received.ExternalRefs["run_ref"] != "run-ref-001" ||
		received.Input["program_id"] != "program-ref-001" ||
		received.Input["level"] != "A1/A2" {
		t.Fatalf("payload=%+v", received)
	}
	blockPosition, ok := received.Input["block_position"].(map[string]any)
	if !ok || blockPosition["block_order"].(float64) != 2 {
		t.Fatalf("block_position=%#v", received.Input["block_position"])
	}
	sourceRefs, ok := received.Input["source_refs"].([]any)
	if !ok || len(sourceRefs) != 1 || sourceRefs[0] != "boe-ref-001" {
		t.Fatalf("source_refs=%#v", received.Input["source_refs"])
	}
}

func TestRESTClientV0SubmitDomainWorkArtifactEnviaArtefacto(t *testing.T) {
	var received opesArtifactRequestV0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/jobs/job-ref-opes-001/artifacts" || r.Method != http.MethodPost {
			t.Fatalf("request inesperada %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":              "artifact-receipt-ref-001",
			"artifact_id":     "artifact-receipt-ref-001",
			"job_id":          "job-ref-opes-001",
			"correlation_id":  "corr-opes-001",
			"idempotency_key": "idem-delivery-001",
			"external_refs":   map[string]string{"delivery_ref": "delivery-ref-001"},
			"artifact": map[string]any{
				"id":           "artifact-receipt-ref-001",
				"job_id":       "job-ref-opes-001",
				"type":         "content_block",
				"reference_id": "block-ref-001",
			},
			"job": map[string]any{
				"id":             "job-ref-opes-001",
				"status":         "completed",
				"execution_mode": "external",
			},
			"block": map[string]any{
				"id": "block-ref-001",
			},
		})
	}))
	defer server.Close()

	client := NewRESTClientV0(RESTClientConfigV0{BaseURL: server.URL})
	result, err := client.SubmitDomainWorkArtifactV0(context.Background(), opesArtifactSubmissionForTestV0())

	if err != nil {
		t.Fatalf("SubmitDomainWorkArtifactV0: %v", err)
	}
	if result.Status != orquestadomainwork.DomainWorkStatusAcceptedV0 ||
		result.ReceiptRef != "artifact-receipt-ref-001" ||
		result.JobRef != "job-ref-opes-001" {
		t.Fatalf("result=%+v", result)
	}
	if received.ArtifactType != "content_block" ||
		received.IdempotencyKey != "idem-delivery-001" ||
		!received.CompleteJob ||
		received.ExternalRefs["delivery_ref"] != "delivery-ref-001" ||
		received.PayloadJSON["title"] != "Bloque 1" ||
		received.PayloadJSON["body"] != "Contenido producido por Orquesta." {
		t.Fatalf("payload=%+v", received)
	}
}

func TestRESTClientV0EntradaInvalidaNoLlamaHTTP(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	}))
	defer server.Close()
	request := opesJobRequestForTestV0()
	request.DomainRef = ""

	client := NewRESTClientV0(RESTClientConfigV0{BaseURL: server.URL})
	result, err := client.CreateDomainWorkJobV0(context.Background(), request)

	if err != nil {
		t.Fatalf("CreateDomainWorkJobV0: %v", err)
	}
	if called || result.Status != orquestadomainwork.DomainWorkStatusInvalidV0 {
		t.Fatalf("called=%v result=%+v", called, result)
	}
}

func opesJobRequestForTestV0() orquestadomainwork.DomainWorkJobRequestV0 {
	return orquestadomainwork.NormalizeDomainWorkJobRequestV0(orquestadomainwork.DomainWorkJobRequestV0{
		RequestID:      "req-opes-001",
		CorrelationID:  "corr-opes-001",
		IdempotencyKey: "idem-opes-001",
		RequestedBy:    "orquesta",
		DomainRef:      "opes",
		InterfaceRefs:  []string{"opes-rest-v0"},
		WorkKind:       "draft_content_block",
		Objective:      "Crear bloque editorial.",
		InputFields: []orquestadomainwork.DomainWorkFieldV0{
			{Name: "program_id", Value: "program-ref-001"},
			{Name: "topic_id", Value: "topic-ref-001"},
			{Name: "chapter_id", Value: "chapter-ref-001"},
			{Name: "level", Value: "A1/A2"},
			{Name: "language_code", Value: "es"},
			{Name: "block_position", ValueJSON: []byte(`{"chapter_order":1,"block_order":2}`)},
			{Name: "source_refs", Values: []string{"boe-ref-001"}},
		},
		ExternalRefs: []orquestadomainwork.DomainWorkExternalRefV0{
			{Kind: "run_ref", Ref: "run-ref-001"},
			{Kind: "task_ref", Ref: "task-ref-001"},
		},
	})
}

func opesArtifactSubmissionForTestV0() orquestadomainwork.DomainWorkArtifactSubmissionV0 {
	return orquestadomainwork.NormalizeDomainWorkArtifactSubmissionV0(orquestadomainwork.DomainWorkArtifactSubmissionV0{
		RequestID:      "req-delivery-001",
		CorrelationID:  "corr-opes-001",
		IdempotencyKey: "idem-delivery-001",
		RequestedBy:    "orquesta",
		DomainRef:      "opes",
		JobRef:         "job-ref-opes-001",
		ArtifactRef:    "artifact-ref-001",
		ArtifactType:   "content_block",
		Summary:        "Bloque 1",
		PayloadFields: []orquestadomainwork.DomainWorkFieldV0{
			{Name: "title", Value: "Bloque 1"},
			{Name: "body", Value: "Contenido producido por Orquesta."},
		},
		ExternalRefs: []orquestadomainwork.DomainWorkExternalRefV0{
			{Kind: "delivery_ref", Ref: "delivery-ref-001"},
		},
		CompleteJob: true,
	})
}
