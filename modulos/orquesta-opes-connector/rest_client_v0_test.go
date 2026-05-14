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

func TestRESTClientV0SubmitDomainWorkArtifactEnviaVisualAsset(t *testing.T) {
	var received opesArtifactRequestV0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/jobs/job-ref-visual-001/artifacts" || r.Method != http.MethodPost {
			t.Fatalf("request inesperada %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":              "artifact-receipt-ref-visual-001",
			"artifact_id":     "artifact-receipt-ref-visual-001",
			"job_id":          "job-ref-visual-001",
			"correlation_id":  "corr-visual-001",
			"idempotency_key": "idem-visual-delivery-001",
			"external_refs":   map[string]string{"delivery_ref": "delivery-ref-visual-001"},
		})
	}))
	defer server.Close()

	client := NewRESTClientV0(RESTClientConfigV0{BaseURL: server.URL})
	result, err := client.SubmitDomainWorkArtifactV0(context.Background(), opesVisualArtifactSubmissionForTestV0())

	if err != nil {
		t.Fatalf("SubmitDomainWorkArtifactV0: %v", err)
	}
	if result.Status != orquestadomainwork.DomainWorkStatusAcceptedV0 ||
		result.JobRef != "job-ref-visual-001" {
		t.Fatalf("result=%+v", result)
	}
	if received.ArtifactType != "visual_asset" ||
		received.PayloadJSON["format"] != "svg" ||
		received.PayloadJSON["caption"] != "Topologia con nodo central." ||
		received.PayloadJSON["alt_text"] != "Switch central conectado a equipos cliente." ||
		received.PayloadJSON["body"] == "" {
		t.Fatalf("payload=%+v", received)
	}
}

func TestRESTClientV0ListExternalJobsConsultaColaPublica(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/jobs" || r.Method != http.MethodGet {
			t.Fatalf("request inesperada %s %s", r.Method, r.URL.Path)
		}
		if r.URL.Query().Get("execution_mode") != "external" ||
			r.URL.Query().Get("status") != "pending" ||
			r.URL.Query().Get("job_type") != "summarize_topic" ||
			r.URL.Query().Get("limit") != "3" {
			t.Fatalf("query=%s", r.URL.RawQuery)
		}
		_ = json.NewEncoder(w).Encode([]map[string]any{{
			"id":             "job-ref-summary-001",
			"type":           "summarize_topic",
			"status":         "pending",
			"execution_mode": "external",
			"payload_json":   `{"topic_id":"topic-ref-001"}`,
			"requested_by":   "opes",
		}})
	}))
	defer server.Close()

	client := NewRESTClientV0(RESTClientConfigV0{BaseURL: server.URL})
	jobs, err := client.ListExternalJobsV0(context.Background(), ExternalJobQueryV0{
		ExecutionMode: "external",
		Status:        "pending",
		JobType:       "summarize_topic",
		Limit:         3,
	})

	if err != nil {
		t.Fatalf("ListExternalJobsV0: %v", err)
	}
	if len(jobs) != 1 ||
		jobs[0].ID != "job-ref-summary-001" ||
		jobs[0].Type != "summarize_topic" ||
		jobs[0].PayloadJSON != `{"topic_id":"topic-ref-001"}` {
		t.Fatalf("jobs=%+v", jobs)
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

func opesVisualArtifactSubmissionForTestV0() orquestadomainwork.DomainWorkArtifactSubmissionV0 {
	return orquestadomainwork.NormalizeDomainWorkArtifactSubmissionV0(orquestadomainwork.DomainWorkArtifactSubmissionV0{
		RequestID:      "req-visual-delivery-001",
		CorrelationID:  "corr-visual-001",
		IdempotencyKey: "idem-visual-delivery-001",
		RequestedBy:    "orquesta",
		DomainRef:      "opes",
		JobRef:         "job-ref-visual-001",
		ArtifactRef:    "artifact-ref-visual-001",
		ArtifactType:   "visual_asset",
		Summary:        "Visual de red en estrella",
		PayloadFields: []orquestadomainwork.DomainWorkFieldV0{
			{Name: "topic_id", Value: "topic-ref-001"},
			{Name: "chapter_id", Value: "chapter-ref-001"},
			{Name: "asset_type", Value: "vignette"},
			{Name: "format", Value: "svg"},
			{Name: "title", Value: "Red en estrella"},
			{Name: "caption", Value: "Topologia con nodo central."},
			{Name: "alt_text", Value: "Switch central conectado a equipos cliente."},
			{Name: "body", Value: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 800 420"></svg>`},
		},
		ExternalRefs: []orquestadomainwork.DomainWorkExternalRefV0{
			{Kind: "delivery_ref", Ref: "delivery-ref-visual-001"},
		},
		CompleteJob: true,
	})
}
