package orquestaopesconnector

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

func TestRESTClientV0CreateDomainWorkJobCreaJobExterno(t *testing.T) {
	var received opesCreateJobRequestV0
	client := newRESTClientForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	client := newRESTClientForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	client := newRESTClientForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

func TestRESTClientV0SubmitDomainWorkArtifactEnviaDocumentPlan(t *testing.T) {
	var received opesArtifactRequestV0
	client := newRESTClientForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/jobs/job-ref-plan-temario-001/artifacts" || r.Method != http.MethodPost {
			t.Fatalf("request inesperada %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":              "artifact-receipt-ref-plan-001",
			"artifact_id":     "artifact-receipt-ref-plan-001",
			"job_id":          "job-ref-plan-temario-001",
			"correlation_id":  "corr-plan-temario-001",
			"idempotency_key": "idem-plan-temario-delivery-001",
			"external_refs":   map[string]string{"delivery_ref": "delivery-ref-plan-temario-001"},
		})
	}))

	result, err := client.SubmitDomainWorkArtifactV0(
		context.Background(),
		opesDocumentPlanArtifactSubmissionForTestV0(),
	)

	if err != nil {
		t.Fatalf("SubmitDomainWorkArtifactV0: %v", err)
	}
	if result.Status != orquestadomainwork.DomainWorkStatusAcceptedV0 ||
		result.JobRef != "job-ref-plan-temario-001" ||
		result.ReceiptRef != "artifact-receipt-ref-plan-001" {
		t.Fatalf("result=%+v", result)
	}
	sections, ok := received.PayloadJSON["sections"].([]any)
	if !ok || len(sections) != 1 {
		t.Fatalf("sections=%#v", received.PayloadJSON["sections"])
	}
	deliverables, ok := received.PayloadJSON["deliverables"].([]any)
	if !ok || len(deliverables) != 1 {
		t.Fatalf("deliverables=%#v", received.PayloadJSON["deliverables"])
	}
	if received.ArtifactType != orquestadomainwork.DomainDocumentPlanArtifactTypeV0 ||
		received.IdempotencyKey != "idem-plan-temario-delivery-001" ||
		!received.CompleteJob ||
		received.ExternalRefs["delivery_ref"] != "delivery-ref-plan-temario-001" ||
		received.PayloadJSON["schema_version"] != orquestadomainwork.DomainDocumentPlanSchemaV0 ||
		received.PayloadJSON["work_kind"] != orquestadomainwork.DomainWorkKindPlanSyllabusV0 ||
		received.PayloadJSON["plan_ref"] != "plan-ref-operadores-001" {
		t.Fatalf("payload=%+v", received)
	}
}
