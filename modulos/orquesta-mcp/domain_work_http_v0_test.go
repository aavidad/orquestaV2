package orquestamcp

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestadomainworkmemory "orquesta/modulos/orquesta-domain-work-memory"
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

func TestMCPDomainWorkStatusHTTPHandlerV0ConservaRepairReceiptGoalFirst(t *testing.T) {
	executor := &fakeMCPQueueGlobalStatusExecutorV0{
		result: MCPAutoprogrammingStatusToolResultV0{
			Estado:   MCPAutoprogrammingStatusEstadoOKV0,
			QueueRef: "global",
			QueueHealth: &MCPAutoprogrammingQueueHealthV0{
				Blocked: 1,
			},
			StaleRunning: []MCPAutoprogrammingActionableRunV0{{
				Code:              mcpAutoprogrammingActionGoalFirstBlockedV0,
				RunRef:            "run-ref-domain-status-repair-receipt-001",
				AppRef:            "app-ref-domain-status-opes",
				RecommendedAction: MCPGoalFirstRepairReceiptActionV0,
				EvidenceRefs:      []string{"evidence-ref-domain-status-repair-receipt"},
			}},
		},
	}
	req := httptest.NewRequest(
		http.MethodGet,
		MCPDomainWorkStatusHTTPPathV0+"?project=opes&course_slug=integracion-social&run_ref=run-ref-domain-status-repair-receipt-001",
		nil,
	)
	rec := httptest.NewRecorder()

	NewMCPDomainWorkStatusHTTPHandlerV0(executor).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result MCPDomainWorkStatusResultV0
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Summary.Status != "blocked" ||
		!result.Summary.NeedsAction ||
		len(result.Items) != 1 ||
		result.Items[0].Status != "blocked" ||
		!result.Items[0].NeedsAction ||
		result.Items[0].RecommendedAction != MCPGoalFirstRepairReceiptActionV0 {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPDomainWorkStatusHTTPHandlerV0ConservaOutputSaneadoGoalFirst(t *testing.T) {
	result := domainWorkStatusResultForActionableRunTestV0(
		t,
		mcpAutoprogrammingActionThreadOutputSanitizedV0,
		mcpQueueGlobalStatusActionReplanNarrowContextV0,
		"evidence-ref-domain-status-thread-output-sanitized",
	)

	if result.Summary.Status != "blocked" ||
		!result.Summary.NeedsAction ||
		len(result.Items) != 1 ||
		result.Items[0].Status != "blocked" ||
		!result.Items[0].NeedsAction ||
		result.Items[0].RecommendedAction != mcpQueueGlobalStatusActionReplanNarrowContextV0 {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPDomainWorkStatusHTTPHandlerV0ConservaWriteSetRequiresWorkspaceWrite(t *testing.T) {
	result := domainWorkStatusResultForActionableRunTestV0(
		t,
		mcpAutoprogrammingActionWriteSetRequiresWorkspaceWriteV0,
		mcpQueueGlobalStatusActionConfigureWorkspaceWriteSandboxV0,
		"evidence-ref-domain-status-write-set-workspace-write",
	)

	if result.Summary.Status != "blocked" ||
		!result.Summary.NeedsAction ||
		len(result.Items) != 1 ||
		result.Items[0].Status != "blocked" ||
		!result.Items[0].NeedsAction ||
		result.Items[0].RecommendedAction != mcpQueueGlobalStatusActionConfigureWorkspaceWriteSandboxV0 {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPDomainWorkStatusHTTPHandlerV0ConservaWriteSetGuardContract(t *testing.T) {
	result := domainWorkStatusResultForActionableRunTestV0(
		t,
		mcpAutoprogrammingActionWriteSetGuardAllowedWriteSetMismatchV0,
		mcpQueueGlobalStatusActionRepairGoalWriteSetContractV0,
		"evidence-ref-domain-status-write-set-guard-mismatch",
	)

	if result.Summary.Status != "blocked" ||
		!result.Summary.NeedsAction ||
		len(result.Items) != 1 ||
		result.Items[0].Status != "blocked" ||
		!result.Items[0].NeedsAction ||
		result.Items[0].RecommendedAction != mcpQueueGlobalStatusActionRepairGoalWriteSetContractV0 {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPDomainWorkStatusHTTPHandlerV0NormalizaSenalesGoalFirstRecuperablesComoBloqueadas(t *testing.T) {
	cases := []struct {
		name              string
		code              string
		recommendedAction string
	}{
		{
			name:              "qa_failed_public_text",
			code:              mcpAutoprogrammingActionQAFailedPublicTextV0,
			recommendedAction: MCPGoalFirstReworkPublicTextActionV0,
		},
		{
			name:              "partial_artifacts_written",
			code:              mcpAutoprogrammingActionPartialArtifactsWrittenV0,
			recommendedAction: MCPGoalFirstReviewPartialArtifactsActionV0,
		},
		{
			name:              "artifact_paths_omitted_materialized",
			code:              mcpAutoprogrammingActionArtifactPathsOmittedV0,
			recommendedAction: MCPGoalFirstRepairReceiptActionV0,
		},
		{
			name:              "out_of_scope_materialized_artifacts",
			code:              mcpAutoprogrammingActionOutOfScopeMaterializedArtifactsV0,
			recommendedAction: MCPGoalFirstReworkWriteSetViolationActionV0,
		},
		{
			name:              "phase0_complete_non_publishable",
			code:              mcpAutoprogrammingActionPhase0CompleteNonPublishableV0,
			recommendedAction: MCPGoalFirstContinueFromPhase0ActionV0,
		},
		{
			name:              "required_test_evidence_missing",
			code:              mcpAutoprogrammingActionRequiredTestEvidenceMissingV0,
			recommendedAction: MCPGoalFirstRepairReceiptActionV0,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := domainWorkStatusResultForActionableRunTestV0(
				t,
				tc.code,
				tc.recommendedAction,
				"evidence-ref-domain-status-"+tc.name,
			)

			if result.Summary.Status != "blocked" ||
				!result.Summary.NeedsAction ||
				len(result.Items) != 1 ||
				result.Items[0].Status != "blocked" ||
				!result.Items[0].NeedsAction ||
				result.Items[0].RecommendedAction != tc.recommendedAction {
				t.Fatalf("result=%+v", result)
			}
		})
	}
}

func domainWorkStatusResultForActionableRunTestV0(
	t *testing.T,
	code string,
	recommendedAction string,
	evidenceRef string,
) MCPDomainWorkStatusResultV0 {
	t.Helper()
	executor := &fakeMCPQueueGlobalStatusExecutorV0{
		result: MCPAutoprogrammingStatusToolResultV0{
			Estado:   MCPAutoprogrammingStatusEstadoOKV0,
			QueueRef: "global",
			QueueHealth: &MCPAutoprogrammingQueueHealthV0{
				Blocked: 1,
			},
			StaleRunning: []MCPAutoprogrammingActionableRunV0{{
				Code:              code,
				RunRef:            "run-ref-domain-status-actionable-001",
				AppRef:            "app-ref-domain-status-opes",
				RecommendedAction: recommendedAction,
				EvidenceRefs:      []string{evidenceRef},
			}},
		},
	}
	req := httptest.NewRequest(
		http.MethodGet,
		MCPDomainWorkStatusHTTPPathV0+"?project=opes&course_slug=integracion-social&run_ref=run-ref-domain-status-actionable-001",
		nil,
	)
	rec := httptest.NewRecorder()

	NewMCPDomainWorkStatusHTTPHandlerV0(executor).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result MCPDomainWorkStatusResultV0
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return result
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

func TestMCPDomainWorkStatusHTTPHandlerV0ObservaRecordsDirectosConColaVacia(t *testing.T) {
	creator := orquestadomainworkmemory.NewInMemoryDomainWorkJobCreatorV0()
	job, err := creator.CreateDomainWorkJobV0(context.Background(), orquestadomainwork.DomainWorkJobRequestV0{
		SchemaVersion:  orquestadomainwork.DomainWorkJobRequestSchemaV0,
		RequestID:      "request-ref-domain-record-status-001",
		CorrelationID:  "corr-domain-record-status-001",
		IdempotencyKey: "idem-domain-record-status-001",
		RequestedBy:    orquestadomainwork.DomainWorkDefaultRequestedByV0,
		DomainRef:      "opes",
		WorkKind:       "generate_audio_asset",
		Objective:      "crear audio observable aunque no aparezca en cola",
		InputFields: []orquestadomainwork.DomainWorkFieldV0{
			{Name: "course_id", Value: "integracion-social"},
		},
		ExternalRefs: []orquestadomainwork.DomainWorkExternalRefV0{
			{Kind: "run_ref", Ref: "run-ref-domain-record-status-001"},
			{Kind: "app_ref", Ref: "app-ref-opes"},
		},
		EvidenceRefs: []string{"evidence-ref-domain-record-request"},
	})
	if err != nil {
		t.Fatalf("CreateDomainWorkJobV0: %v", err)
	}
	executor := &fakeMCPQueueGlobalStatusExecutorV0{
		result: MCPAutoprogrammingStatusToolResultV0{
			Estado:   MCPAutoprogrammingStatusEstadoOKV0,
			QueueRef: "global",
		},
	}
	req := httptest.NewRequest(
		http.MethodGet,
		MCPDomainWorkStatusHTTPPathV0+"?project=opes&course_slug=integracion-social&queue_limit=5",
		nil,
	)
	rec := httptest.NewRecorder()

	NewMCPDomainWorkStatusHTTPHandlerWithRecordsV0(executor, creator).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result MCPDomainWorkStatusResultV0
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Summary.Status != "queued" ||
		result.Summary.Queued != 1 ||
		result.Summary.Completed != 0 ||
		len(result.Items) != 1 ||
		result.Items[0].JobRef != job.JobRef ||
		result.Items[0].Status != "queued" ||
		result.Items[0].WorkKind != "generate_audio_asset" ||
		result.Items[0].RunRef != "run-ref-domain-record-status-001" {
		t.Fatalf("result=%+v", result)
	}
}
