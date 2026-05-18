package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestaexternalworkrun "orquesta/modulos/orquesta-external-work-run"
	orquestaopesbridge "orquesta/modulos/orquesta-opes-bridge"
	orquestaopesconnector "orquesta/modulos/orquesta-opes-connector"
)

func TestRunOPESDrainOnceV0LedgerEvitaReenviarJobPendienteV0(t *testing.T) {
	opesServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/jobs" || r.Method != http.MethodGet {
			t.Fatalf("opes request inesperada %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode([]map[string]any{{
			"id":             "job-ref-expansion-001",
			"type":           "expand_topic_from_summary",
			"status":         "pending",
			"execution_mode": "external",
			"payload_json":   `{"topic_id":"topic-ref-001","summary_payload_json":{"markdown":"Resumen"}}`,
			"requested_by":   "opes",
		}})
	}))
	defer opesServer.Close()

	posts := 0
	orquestaServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v0/external-work/run" || r.Method != http.MethodPost {
			t.Fatalf("orquesta request inesperada %s %s", r.Method, r.URL.Path)
		}
		posts++
		_ = json.NewEncoder(w).Encode(map[string]string{
			"run_ref": "run-ref-expansion-001",
			"estado":  "accepted",
		})
	}))
	defer orquestaServer.Close()

	ledger, err := newFileExternalBridgeInputLedgerV0(
		filepath.Join(t.TempDir(), "external-bridge-input-ledger.json"),
	)
	if err != nil {
		t.Fatalf("ledger: %v", err)
	}
	config := opesDrainConfigV0{
		OPESBaseURL:     opesServer.URL,
		OrquestaBaseURL: orquestaServer.URL,
		Limit:           1,
		HTTPTimeout:     time.Second,
		RunConfig: orquestaopesbridge.JobRunConfigV0{
			PriorityScore: 70,
			RequestedBy:   "test",
		},
		InputLedger: ledger,
	}

	first, err := runOPESDrainOnceV0(context.Background(), config)
	if err != nil {
		t.Fatalf("first drain: %v", err)
	}
	second, err := runOPESDrainOnceV0(context.Background(), config)
	if err != nil {
		t.Fatalf("second drain: %v", err)
	}

	if posts != 1 {
		t.Fatalf("posts=%d first=%+v second=%+v", posts, first, second)
	}
	if first.Submitted != 1 || second.AlreadySubmitted != 1 ||
		second.Results[0].Status != "already_submitted" ||
		second.Results[0].RunRef != "run-ref-expansion-001" {
		t.Fatalf("first=%+v second=%+v", first, second)
	}
}

func TestRunOPESDrainOnceV0PlanTemarioOperadoresEnviaExternalWorkDocumentPlanV0(t *testing.T) {
	opesServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/jobs/job-ref-plan-operadores-001" || r.Method != http.MethodGet {
			t.Fatalf("opes request inesperada %s %s", r.Method, r.URL.Path)
		}
		if r.URL.RawQuery != "" {
			t.Fatalf("query=%s", r.URL.RawQuery)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":             "job-ref-plan-operadores-001",
			"type":           "plan_temario",
			"status":         "pending",
			"execution_mode": "external",
			"payload_json": `{
				"program_id":"program-ref-operadores-001",
				"document_kind":"temario_oposicion",
				"language_code":"es",
				"title":"Temario operadores",
				"official_outline":"Operadores, tipos, precedencia, asociatividad y usos.",
				"quality_criteria":["completo","sin placeholders"],
				"target_pages_min":20,
				"target_pages_max":40
			}`,
			"correlation_id":  "corr-job-plan-operadores-001",
			"idempotency_key": "idem-job-plan-operadores-001",
			"requested_by":    "opes",
		})
	}))
	defer opesServer.Close()

	var received orquestaexternalworkrun.StartExternalWorkRunRequestV0
	orquestaServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v0/external-work/run" || r.Method != http.MethodPost {
			t.Fatalf("orquesta request inesperada %s %s", r.Method, r.URL.Path)
		}
		var envelope struct {
			ExternalWorkRunRequest orquestaexternalworkrun.StartExternalWorkRunRequestV0 `json:"external_work_run_request"`
		}
		if err := json.NewDecoder(r.Body).Decode(&envelope); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		received = envelope.ExternalWorkRunRequest
		_ = json.NewEncoder(w).Encode(map[string]string{
			"run_ref": "run-ref-plan-operadores-001",
			"estado":  "ok",
		})
	}))
	defer orquestaServer.Close()

	ledger, err := newFileExternalBridgeInputLedgerV0(
		filepath.Join(t.TempDir(), "external-bridge-input-ledger.json"),
	)
	if err != nil {
		t.Fatalf("ledger: %v", err)
	}
	summary, err := runOPESDrainOnceV0(context.Background(), opesDrainConfigV0{
		OPESBaseURL:     opesServer.URL,
		OrquestaBaseURL: orquestaServer.URL,
		Limit:           1,
		JobType:         "plan_temario",
		JobRef:          "job-ref-plan-operadores-001",
		HTTPTimeout:     time.Second,
		RunConfig: orquestaopesbridge.JobRunConfigV0{
			PriorityScore: 90,
			RequestedBy:   "test",
		},
		InputLedger: ledger,
	})
	if err != nil {
		t.Fatalf("drain: %v", err)
	}

	work := received.AppChangeRequest.ExternalWork
	if work == nil {
		t.Fatalf("external_work nil: summary=%+v received=%+v", summary, received)
	}
	fields := work.InputFields
	if summary.Submitted != 1 ||
		summary.Results[0].Status != "submitted" ||
		summary.Results[0].RunRef != "run-ref-plan-operadores-001" ||
		work == nil ||
		work.JobRef != "job-ref-plan-operadores-001" ||
		work.WorkKind != "plan_temario" ||
		received.AppChangeRequest.AllowedWriteSet[0] != "external/opes/plan_temario/job-ref-plan-operadores-001" ||
		!domainWorkFieldValueForDrainTestV0(fields, "expected_artifact_type", orquestadomainwork.DomainDocumentPlanArtifactTypeV0) ||
		!domainWorkFieldValueForDrainTestV0(fields, "context_budget_profile", "large") ||
		!domainWorkFieldValueForDrainTestV0(fields, "document_kind", "temario_oposicion") {
		t.Fatalf("summary=%+v received=%+v work=%+v", summary, received, work)
	}
}

func TestRunOPESDrainOnceV0SecuenciaPasesSaltaTiposSinPendientesV0(t *testing.T) {
	queries := []string{}
	opesServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/jobs" || r.Method != http.MethodGet {
			t.Fatalf("opes request inesperada %s %s", r.Method, r.URL.Path)
		}
		if r.URL.Query().Get("limit") != "10" {
			t.Fatalf("limit=%q", r.URL.Query().Get("limit"))
		}
		jobType := r.URL.Query().Get("job_type")
		queries = append(queries, jobType)
		switch jobType {
		case "draft_content_block":
			_ = json.NewEncoder(w).Encode([]map[string]any{})
		case "generate_visual_asset":
			_ = json.NewEncoder(w).Encode([]map[string]any{{
				"id":             "job-ref-visual-001",
				"type":           "generate_visual_asset",
				"status":         "pending",
				"execution_mode": "external",
				"payload_json":   `{"topic_id":"topic-ref-001","objective":"Crear esquema visual"}`,
				"requested_by":   "opes",
			}})
		default:
			t.Fatalf("job_type inesperado=%q", jobType)
		}
	}))
	defer opesServer.Close()

	posts := 0
	orquestaServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v0/external-work/run" || r.Method != http.MethodPost {
			t.Fatalf("orquesta request inesperada %s %s", r.Method, r.URL.Path)
		}
		posts++
		_ = json.NewEncoder(w).Encode(map[string]string{
			"run_ref": "run-ref-visual-001",
			"estado":  "accepted",
		})
	}))
	defer orquestaServer.Close()

	ledger, err := newFileExternalBridgeInputLedgerV0(
		filepath.Join(t.TempDir(), "external-bridge-input-ledger.json"),
	)
	if err != nil {
		t.Fatalf("ledger: %v", err)
	}
	summary, err := runOPESDrainOnceV0(context.Background(), opesDrainConfigV0{
		OPESBaseURL:     opesServer.URL,
		OrquestaBaseURL: orquestaServer.URL,
		Limit:           1,
		JobTypeSequence: []string{
			"draft_content_block",
			"generate_visual_asset",
			"review_quality",
		},
		HTTPTimeout: time.Second,
		RunConfig: orquestaopesbridge.JobRunConfigV0{
			PriorityScore: 70,
			RequestedBy:   "test",
		},
		InputLedger: ledger,
	})
	if err != nil {
		t.Fatalf("drain: %v", err)
	}

	if posts != 1 ||
		strings.Join(queries, ",") != "draft_content_block,generate_visual_asset" ||
		summary.SelectedJobType != "generate_visual_asset" ||
		summary.JobType != "generate_visual_asset" ||
		strings.Join(summary.EmptyJobTypes, ",") != "draft_content_block" ||
		summary.Submitted != 1 ||
		summary.Results[0].Status != "submitted" {
		t.Fatalf("posts=%d queries=%v summary=%+v", posts, queries, summary)
	}
}

func TestRunOPESDrainOnceV0SecuenciaPasesNoAvanzaSiPrimerTipoYaEnviadoV0(t *testing.T) {
	queries := []string{}
	opesServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/jobs" || r.Method != http.MethodGet {
			t.Fatalf("opes request inesperada %s %s", r.Method, r.URL.Path)
		}
		jobType := r.URL.Query().Get("job_type")
		queries = append(queries, jobType)
		if jobType != "draft_content_block" {
			t.Fatalf("job_type inesperado=%q", jobType)
		}
		_ = json.NewEncoder(w).Encode([]map[string]any{{
			"id":             "job-ref-draft-001",
			"type":           "draft_content_block",
			"status":         "pending",
			"execution_mode": "external",
			"payload_json":   `{"topic_id":"topic-ref-001","title":"Bloque 1"}`,
			"requested_by":   "opes",
		}})
	}))
	defer opesServer.Close()

	orquestaServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("orquesta no debe recibir POST, recibido %s %s", r.Method, r.URL.Path)
	}))
	defer orquestaServer.Close()

	ledger, err := newFileExternalBridgeInputLedgerV0(
		filepath.Join(t.TempDir(), "external-bridge-input-ledger.json"),
	)
	if err != nil {
		t.Fatalf("ledger: %v", err)
	}
	if err := opesBridgeRecordSubmittedV0(
		context.Background(),
		ledger,
		orquestaopesconnector.ExternalJobV0{ID: "job-ref-draft-001"},
		"run-ref-draft-001",
		"change-ref-draft-001",
	); err != nil {
		t.Fatalf("record ledger: %v", err)
	}

	summary, err := runOPESDrainOnceV0(context.Background(), opesDrainConfigV0{
		OPESBaseURL:     opesServer.URL,
		OrquestaBaseURL: orquestaServer.URL,
		Limit:           1,
		JobTypeSequence: []string{
			"draft_content_block",
			"generate_visual_asset",
			"review_quality",
		},
		HTTPTimeout: time.Second,
		RunConfig: orquestaopesbridge.JobRunConfigV0{
			PriorityScore: 70,
			RequestedBy:   "test",
		},
		InputLedger: ledger,
	})
	if err != nil {
		t.Fatalf("drain: %v", err)
	}

	if strings.Join(queries, ",") != "draft_content_block" ||
		summary.SelectedJobType != "draft_content_block" ||
		summary.AlreadySubmitted != 1 ||
		summary.Submitted != 0 ||
		summary.Results[0].Status != "already_submitted" ||
		summary.Results[0].RunRef != "run-ref-draft-001" {
		t.Fatalf("queries=%v summary=%+v", queries, summary)
	}
}

func TestRunOPESDrainOnceV0EscaneaMasQueLimitYEnviaSiguientesNoEnviadosV0(t *testing.T) {
	opesServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/jobs" || r.Method != http.MethodGet {
			t.Fatalf("opes request inesperada %s %s", r.Method, r.URL.Path)
		}
		if r.URL.Query().Get("limit") != "30" {
			t.Fatalf("limit=%q", r.URL.Query().Get("limit"))
		}
		jobs := []map[string]any{}
		for _, id := range []string{
			"job-ref-draft-001",
			"job-ref-draft-002",
			"job-ref-draft-003",
			"job-ref-draft-004",
			"job-ref-draft-005",
			"job-ref-draft-006",
			"job-ref-draft-007",
		} {
			jobs = append(jobs, map[string]any{
				"id":             id,
				"type":           "draft_content_block",
				"status":         "pending",
				"execution_mode": "external",
				"payload_json":   `{"topic_id":"topic-ref-001","title":"Bloque"}`,
				"requested_by":   "opes",
			})
		}
		_ = json.NewEncoder(w).Encode(jobs)
	}))
	defer opesServer.Close()

	posts := []string{}
	orquestaServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v0/external-work/run" || r.Method != http.MethodPost {
			t.Fatalf("orquesta request inesperada %s %s", r.Method, r.URL.Path)
		}
		var envelope struct {
			ExternalWorkRunRequest orquestaexternalworkrun.StartExternalWorkRunRequestV0 `json:"external_work_run_request"`
		}
		if err := json.NewDecoder(r.Body).Decode(&envelope); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		jobRef := envelope.ExternalWorkRunRequest.AppChangeRequest.ExternalWork.JobRef
		posts = append(posts, jobRef)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"run_ref": "run-ref-" + jobRef,
			"estado":  "accepted",
		})
	}))
	defer orquestaServer.Close()

	ledger, err := newFileExternalBridgeInputLedgerV0(
		filepath.Join(t.TempDir(), "external-bridge-input-ledger.json"),
	)
	if err != nil {
		t.Fatalf("ledger: %v", err)
	}
	for _, id := range []string{"job-ref-draft-001", "job-ref-draft-002", "job-ref-draft-003"} {
		if err := opesBridgeRecordSubmittedV0(
			context.Background(),
			ledger,
			orquestaopesconnector.ExternalJobV0{ID: id},
			"run-ref-"+id,
			"change-ref-"+id,
		); err != nil {
			t.Fatalf("record ledger: %v", err)
		}
	}

	summary, err := runOPESDrainOnceV0(context.Background(), opesDrainConfigV0{
		OPESBaseURL:     opesServer.URL,
		OrquestaBaseURL: orquestaServer.URL,
		Limit:           3,
		JobType:         "draft_content_block",
		HTTPTimeout:     time.Second,
		RunConfig: orquestaopesbridge.JobRunConfigV0{
			PriorityScore: 70,
			RequestedBy:   "test",
		},
		InputLedger: ledger,
	})
	if err != nil {
		t.Fatalf("drain: %v", err)
	}

	if strings.Join(posts, ",") != "job-ref-draft-004,job-ref-draft-005,job-ref-draft-006" ||
		summary.Seen != 7 ||
		summary.AlreadySubmitted != 3 ||
		summary.Submitted != 3 {
		t.Fatalf("posts=%v summary=%+v", posts, summary)
	}
}

func domainWorkFieldValueForDrainTestV0(
	fields []orquestadomainwork.DomainWorkFieldV0,
	name string,
	value string,
) bool {
	for _, field := range fields {
		if field.Name == name && field.Value == value {
			return true
		}
	}
	return false
}
