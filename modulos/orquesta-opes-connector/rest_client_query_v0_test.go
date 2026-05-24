package orquestaopesconnector

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

func TestRESTClientV0ListExternalJobsConsultaColaPublica(t *testing.T) {
	client := newRESTClientForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

func TestRESTClientV0ListExternalJobsFiltraRespuestaMixta(t *testing.T) {
	client := newRESTClientForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/jobs/job-ref-plan-001" || r.Method != http.MethodGet {
			t.Fatalf("request inesperada %s %s", r.Method, r.URL.Path)
		}
		if r.URL.RawQuery != "" {
			t.Fatalf("query=%s", r.URL.RawQuery)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":             "job-ref-plan-001",
			"type":           "plan_temario",
			"status":         "pending",
			"execution_mode": "external",
		})
	}))

	jobs, err := client.ListExternalJobsV0(context.Background(), ExternalJobQueryV0{
		ExecutionMode: "external",
		Status:        "pending",
		JobType:       "plan_temario",
		JobRef:        "job-ref-plan-001",
		Limit:         1,
	})

	if err != nil {
		t.Fatalf("ListExternalJobsV0: %v", err)
	}
	if len(jobs) != 1 ||
		jobs[0].ID != "job-ref-plan-001" ||
		jobs[0].Type != "plan_temario" ||
		jobs[0].Status != "pending" ||
		jobs[0].ExecutionMode != "external" {
		t.Fatalf("jobs=%+v", jobs)
	}
}

func TestRESTClientV0ListTopicBlocksUsaAPIPublica(t *testing.T) {
	client := newRESTClientForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/topics/topic-ref-001/blocks" || r.Method != http.MethodGet {
			t.Fatalf("request inesperada %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode([]map[string]any{{
			"ID":               "block-ref-001",
			"StableID":         "stable-ref-001",
			"CanonicalTopicID": "topic-ref-001",
			"ChapterID":        "chapter-ref-001",
			"Type":             "technical",
			"Status":           "pendiente_revision",
			"Title":            "Bloque 1",
			"Markdown":         "Contenido del bloque.",
			"LanguageCode":     "es",
			"SourceRefs":       []string{"source-ref-001"},
		}})
	}))

	blocks, err := client.ListTopicBlocksV0(context.Background(), "topic-ref-001")

	if err != nil {
		t.Fatalf("ListTopicBlocksV0: %v", err)
	}
	if len(blocks) != 1 ||
		blocks[0].ID != "block-ref-001" ||
		blocks[0].StableID != "stable-ref-001" ||
		blocks[0].Markdown != "Contenido del bloque." ||
		len(blocks[0].SourceRefs) != 1 {
		t.Fatalf("blocks=%+v", blocks)
	}
}

func TestRESTClientV0EntradaInvalidaNoLlamaHTTP(t *testing.T) {
	called := false
	request := opesJobRequestForTestV0()
	request.DomainRef = ""
	client := newRESTClientForTestV0(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	}))

	result, err := client.CreateDomainWorkJobV0(context.Background(), request)

	if err != nil {
		t.Fatalf("CreateDomainWorkJobV0: %v", err)
	}
	if called || result.Status != orquestadomainwork.DomainWorkStatusInvalidV0 {
		t.Fatalf("called=%v result=%+v", called, result)
	}
}
