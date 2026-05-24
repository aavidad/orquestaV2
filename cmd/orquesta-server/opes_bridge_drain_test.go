package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
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
	opesServer := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	orquestaServer := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	opesServer := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	orquestaServer := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	opesServer := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	orquestaServer := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	opesServer := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

	orquestaServer := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	opesServer := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	orquestaServer := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

func TestRunOPESDrainOnceV0SecuenciaDerivadosOPESHastaAssembleTopicV0(t *testing.T) {
	sequence := []string{
		"draft_content_block",
		"generate_visual_asset",
		"review_legal",
		"review_pedagogical",
		"review_quality",
		"validate_topic",
		"assemble_topic",
	}
	expectedArtifactByType := map[string]string{
		"draft_content_block":   "content_block",
		"generate_visual_asset": "visual_asset",
		"review_legal":          "block_revision",
		"review_pedagogical":    "block_revision",
		"review_quality":        "block_revision",
		"validate_topic":        "block_revision",
		"assemble_topic":        "assembled_topic",
	}
	activeStage := 0
	queries := []string{}
	submittedByType := map[string]int{}

	opesServer := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/jobs" || r.Method != http.MethodGet {
			t.Fatalf("opes request inesperada %s %s", r.Method, r.URL.Path)
		}
		jobType := r.URL.Query().Get("job_type")
		queries = append(queries, jobType)
		if activeStage >= len(sequence) || jobType != sequence[activeStage] {
			_ = json.NewEncoder(w).Encode([]map[string]any{})
			return
		}
		_ = json.NewEncoder(w).Encode([]map[string]any{{
			"id":              "job-ref-" + strings.ReplaceAll(jobType, "_", "-") + "-001",
			"type":            jobType,
			"status":          "pending",
			"execution_mode":  "external",
			"payload_json":    opesDerivedPayloadForDrainTestV0(jobType),
			"correlation_id":  "corr-ref-" + strings.ReplaceAll(jobType, "_", "-") + "-001",
			"idempotency_key": "idem-ref-" + strings.ReplaceAll(jobType, "_", "-") + "-001",
			"requested_by":    "opes",
		}})
	}))
	defer opesServer.Close()

	type postedRun struct {
		JobRef       string
		WorkKind     string
		ArtifactType string
		WriteSet     string
	}
	posts := []postedRun{}
	orquestaServer := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v0/external-work/run" || r.Method != http.MethodPost {
			t.Fatalf("orquesta request inesperada %s %s", r.Method, r.URL.Path)
		}
		var envelope struct {
			ExternalWorkRunRequest orquestaexternalworkrun.StartExternalWorkRunRequestV0 `json:"external_work_run_request"`
		}
		if err := json.NewDecoder(r.Body).Decode(&envelope); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		work := envelope.ExternalWorkRunRequest.AppChangeRequest.ExternalWork
		if work == nil {
			t.Fatalf("external_work nil: %+v", envelope.ExternalWorkRunRequest)
		}
		artifactType := domainWorkFieldStringForDrainTestV0(work.InputFields, "expected_artifact_type")
		posts = append(posts, postedRun{
			JobRef:       work.JobRef,
			WorkKind:     work.WorkKind,
			ArtifactType: artifactType,
			WriteSet:     envelope.ExternalWorkRunRequest.AppChangeRequest.AllowedWriteSet[0],
		})
		submittedByType[work.WorkKind]++
		_ = json.NewEncoder(w).Encode(map[string]string{
			"run_ref": "run-ref-" + work.JobRef,
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
		JobTypeSequence: sequence,
		HTTPTimeout:     time.Second,
		RunConfig: orquestaopesbridge.JobRunConfigV0{
			PriorityScore: 70,
			RequestedBy:   "test",
		},
		InputLedger: ledger,
	}

	for index, jobType := range sequence {
		activeStage = index
		first, err := runOPESDrainOnceV0(context.Background(), config)
		if err != nil {
			t.Fatalf("drain %s: %v", jobType, err)
		}
		second, err := runOPESDrainOnceV0(context.Background(), config)
		if err != nil {
			t.Fatalf("drain replay %s: %v", jobType, err)
		}
		if first.SelectedJobType != jobType ||
			first.Submitted != 1 ||
			second.SelectedJobType != jobType ||
			second.Submitted != 0 ||
			second.AlreadySubmitted != 1 ||
			second.Results[0].Status != "already_submitted" {
			t.Fatalf("jobType=%s first=%+v second=%+v", jobType, first, second)
		}
		if submittedByType[jobType] != 1 {
			t.Fatalf("jobType=%s submissions=%d", jobType, submittedByType[jobType])
		}
	}

	if len(posts) != len(sequence) {
		t.Fatalf("posts=%+v", posts)
	}
	for index, post := range posts {
		jobType := sequence[index]
		if post.WorkKind != jobType ||
			post.ArtifactType != expectedArtifactByType[jobType] ||
			post.WriteSet != "external/opes/"+jobType+"/"+post.JobRef {
			t.Fatalf("index=%d post=%+v expected=%s", index, post, expectedArtifactByType[jobType])
		}
	}
	last := posts[len(posts)-1]
	if last.WorkKind != "assemble_topic" || last.ArtifactType != "assembled_topic" {
		t.Fatalf("assemble no entrego assembled_topic: posts=%+v queries=%v", posts, queries)
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

func domainWorkFieldStringForDrainTestV0(
	fields []orquestadomainwork.DomainWorkFieldV0,
	name string,
) string {
	for _, field := range fields {
		if field.Name == name {
			return field.Value
		}
	}
	return ""
}

func opesDerivedPayloadForDrainTestV0(jobType string) string {
	switch jobType {
	case "draft_content_block":
		return `{"program_id":"program-ref-operadores-001","topic_id":"topic-ref-operadores-001","section_ref":"section-ref-001","title":"Bloque operadores"}`
	case "generate_visual_asset":
		return `{"program_id":"program-ref-operadores-001","topic_id":"topic-ref-operadores-001","visual_ref":"visual-ref-001","objective":"Diagrama de precedencia"}`
	case "assemble_topic":
		return `{"program_id":"program-ref-operadores-001","topic_id":"topic-ref-operadores-001","document_plan_artifact_id":"artifact-plan-operadores-001","content_block_artifact_refs":["artifact-block-001"],"visual_asset_artifact_refs":["artifact-visual-001"],"review_artifact_refs":["artifact-review-001"],"validation_artifact_ref":"artifact-validation-001"}`
	default:
		return `{"program_id":"program-ref-operadores-001","topic_id":"topic-ref-operadores-001","document_plan_artifact_id":"artifact-plan-operadores-001","scope":"tema completo"}`
	}
}

func TestSmokeOPESDerivativesRESTWrapperFakeServerV0(t *testing.T) {
	requireLocalTCPForTestV0(t)
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 no disponible")
	}
	repoRoot := filepath.Clean("../..")
	cmd := exec.Command("bash", "scripts/smoke_opes_derivatives_rest.sh")
	cmd.Dir = repoRoot
	cmd.Env = cleanOPESSmokeEnvForDrainTestV0(os.Environ())
	cmd.Env = append(cmd.Env,
		"ORQUESTA_OPES_DERIVATIVES_FAKE_SERVER=1",
		"ORQUESTA_OPES_BRIDGE_LIMIT=1",
		"SMOKE_ID=test-derivatives-rest-fake",
		"SMOKE_OUT_DIR="+filepath.Join(t.TempDir(), "out"),
	)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("script err=%v stdout=%s stderr=%s", err, stdout.String(), stderr.String())
	}
	output := stdout.String()
	if !strings.Contains(output, `"dry_run":true`) ||
		!strings.Contains(output, `"selected_job_type":"assemble_topic"`) ||
		!strings.Contains(output, `"empty_job_types":["draft_content_block","generate_visual_asset","review_legal","review_pedagogical","review_quality","validate_topic"]`) ||
		!strings.Contains(output, `"job_ref":"job-ref-fake-assemble-topic-001"`) ||
		!strings.Contains(output, `"status":"dry_run"`) {
		t.Fatalf("stdout=%s stderr=%s", output, stderr.String())
	}
}

func TestSmokeOPESDerivativesRESTWrapperFakeServerRunUntilAssembleV0(t *testing.T) {
	requireLocalTCPForTestV0(t)
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 no disponible")
	}
	if _, err := exec.LookPath("curl"); err != nil {
		t.Skip("curl no disponible")
	}
	repoRoot := filepath.Clean("../..")
	cmd := exec.Command("bash", "scripts/smoke_opes_derivatives_rest.sh")
	cmd.Dir = repoRoot
	cmd.Env = cleanOPESSmokeEnvForDrainTestV0(os.Environ())
	cmd.Env = append(cmd.Env,
		"ORQUESTA_OPES_DERIVATIVES_FAKE_SERVER=1",
		"ORQUESTA_OPES_DERIVATIVES_SMOKE_MODE=run-until-assemble",
		"ORQUESTA_OPES_DERIVATIVES_EXECUTE=1",
		"ORQUESTA_OPES_BRIDGE_LIMIT=1",
		"ORQUESTA_OPES_BRIDGE_MAX_TICKS=10",
		"ORQUESTA_OPES_DERIVATIVES_TICK_SLEEP_SECONDS=0",
		"SMOKE_ID=test-derivatives-rest-run-until-fake",
		"SMOKE_OUT_DIR="+filepath.Join(t.TempDir(), "out"),
	)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("script err=%v stdout=%s stderr=%s", err, stdout.String(), stderr.String())
	}
	output := stdout.String()
	if !strings.Contains(output, `"selected_job_type":"draft_content_block"`) ||
		!strings.Contains(output, `"selected_job_type":"assemble_topic"`) ||
		!strings.Contains(output, `"artifact_type": "assembled_topic"`) ||
		!strings.Contains(output, `run_until_status=assembled`) {
		t.Fatalf("stdout=%s stderr=%s", output, stderr.String())
	}
}

func TestSmokeOPESPlanTemarioWrapperFakeServerV0(t *testing.T) {
	requireLocalTCPForTestV0(t)
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 no disponible")
	}
	repoRoot := filepath.Clean("../..")
	cmd := exec.Command("bash", "scripts/smoke_opes_plan_temario_operadores.sh")
	cmd.Dir = repoRoot
	cmd.Env = cleanOPESSmokeEnvForDrainTestV0(os.Environ())
	cmd.Env = append(cmd.Env,
		"ORQUESTA_OPES_PLAN_TEMARIO_FAKE_SERVER=1",
		"SMOKE_ID=test-plan-temario-fake",
		"SMOKE_OUT_DIR="+filepath.Join(t.TempDir(), "out"),
	)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("script err=%v stdout=%s stderr=%s", err, stdout.String(), stderr.String())
	}
	output := stdout.String()
	if !strings.Contains(output, `"dry_run":true`) ||
		!strings.Contains(output, `"job_type":"plan_temario"`) ||
		!strings.Contains(output, `"job_ref":"job-ref-fake-plan-temario-operadores-001"`) ||
		!strings.Contains(output, `"status":"dry_run"`) {
		t.Fatalf("stdout=%s stderr=%s", output, stderr.String())
	}
}

func cleanOPESSmokeEnvForDrainTestV0(env []string) []string {
	out := make([]string, 0, len(env))
	for _, item := range env {
		if strings.HasPrefix(item, "ORQUESTA_OPES") ||
			strings.HasPrefix(item, "OPES_BASE_URL=") ||
			strings.HasPrefix(item, "ORQUESTA_BASE_URL=") ||
			strings.HasPrefix(item, "SMOKE_ID=") ||
			strings.HasPrefix(item, "SMOKE_OUT_DIR=") {
			continue
		}
		out = append(out, item)
	}
	return out
}
