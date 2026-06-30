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

func TestMCPDomainWorkStatusHTTPHandlerV0GetProyectaFachadaEstable(t *testing.T) {
	executor := &fakeMCPQueueGlobalStatusExecutorV0{
		result: MCPAutoprogrammingStatusToolResultV0{
			Estado:   MCPAutoprogrammingStatusEstadoOKV0,
			QueueRef: "global",
			Queue: &MCPRunQueuePriorityToolResultV0{
				Estado:   MCPRunQueuePriorityEstadoOKV0,
				QueueRef: "global",
				Ranked: []MCPRunQueueRankedCandidateCompactV0{{
					Rank:         1,
					RunRef:       "run-ref-domain-status-001",
					AppRef:       "app-ref-domain-status-opes",
					Status:       "running",
					EvidenceRefs: []string{"evidence-ref-domain-status-queue"},
				}},
			},
			QueueHealth: &MCPAutoprogrammingQueueHealthV0{
				RunningLive: 1,
				Blocked:     1,
			},
			Operator: &MCPAutoprogrammingOperatorV0{
				ActiveRuns: []MCPAutoprogrammingActiveRunV0{{
					RunRef: "run-ref-domain-status-001",
					AppRef: "app-ref-domain-status-opes",
					Status: "running",
				}},
				SafeActions: []MCPAutoprogrammingSafeActionV0{{
					Action:   "review",
					RunRef:   "run-ref-domain-status-001",
					Endpoint: MCPAutoprogrammingObserveGoalHTTPPathV0,
				}},
			},
			StaleRunning: []MCPAutoprogrammingActionableRunV0{{
				Code:              mcpAutoprogrammingActionGoalFirstBlockedV0,
				RunRef:            "run-ref-domain-status-001",
				AppRef:            "app-ref-domain-status-opes",
				RecommendedAction: "review_replan_goal_first",
				CurrentPhase:      "qa",
				DomainCounters:    map[string]int{"audio_pending": 2},
				EvidenceRefs:      []string{"evidence-ref-domain-status-blocked"},
			}},
		},
	}
	req := httptest.NewRequest(
		http.MethodGet,
		MCPDomainWorkStatusHTTPPathV0+"?request_id=req-domain-status-001&project=opes&course_slug=integracion-social&run_ref=run-ref-domain-status-001&queue_limit=7",
		nil,
	)
	req.Header.Set("X-Correlation-ID", "corr-domain-status-001")
	rec := httptest.NewRecorder()

	NewMCPDomainWorkStatusHTTPHandlerV0(executor).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK ||
		rec.Header().Get("X-Correlation-ID") != "corr-domain-status-001" {
		t.Fatalf("status=%d headers=%v body=%s", rec.Code, rec.Header(), rec.Body.String())
	}
	if executor.input.RunRef != "run-ref-domain-status-001" ||
		executor.input.QueueLimit != 7 ||
		!bool(executor.input.IncludeAgentProgress) ||
		!bool(executor.input.IncludeAgentUsage) {
		t.Fatalf("input=%+v", executor.input)
	}
	var result MCPDomainWorkStatusResultV0
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.SchemaVersion != MCPDomainWorkStatusSchemaVersionV0 ||
		result.Estado != MCPAutoprogrammingStatusEstadoOKV0 ||
		result.Filters.Project != "opes" ||
		result.Filters.CourseSlug != "integracion-social" ||
		result.Summary.Status != "blocked" ||
		result.Summary.Running != 1 ||
		result.Summary.Blocked != 1 ||
		result.Summary.WillFinishAlone ||
		len(result.Items) != 1 ||
		result.Items[0].Status != "blocked" ||
		result.Items[0].CurrentPhase != "qa" ||
		result.Items[0].DomainCounters["audio_pending"] != 2 ||
		len(result.SafeActions) != 1 {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPDomainWorkStatusHTTPHandlerV0FiltroContextualSinRefDiagnostica(t *testing.T) {
	executor := &fakeMCPQueueGlobalStatusExecutorV0{}
	req := httptest.NewRequest(http.MethodGet, MCPDomainWorkStatusHTTPPathV0+"?project=opes&course_slug=curso", nil)
	rec := httptest.NewRecorder()

	NewMCPDomainWorkStatusHTTPHandlerV0(executor).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result MCPDomainWorkStatusResultV0
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !hasMCPQueueGlobalStatusDiagnosticCodeForTestV0(result.Diagnostics, "domain_work_status_context_filter_without_ref") {
		t.Fatalf("diagnostics=%+v", result.Diagnostics)
	}
}
