package orquestaappcodexstack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestaexternalworkrun "orquesta/modulos/orquesta-external-work-run"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestaopesbridge "orquesta/modulos/orquesta-opes-bridge"
	orquestaopesconnector "orquesta/modulos/orquesta-opes-connector"
)

func TestCodexStackV0ExternalWorkGoalFirstCierraSecuenciaOPESDerivadosConReceiptsLedgerV0(t *testing.T) {
	stack, observer, launcher, domainWork := buildExternalWorkGoalFirstDomainDeliveryStackWithExecutorForTestV0(t)
	sequence := orquestaopesbridge.OPESFullTemarioJobTypeSequenceV0()
	if len(sequence) != 24 || sequence[0] != "plan_temario" {
		t.Fatalf("secuencia OPES inesperada=%+v", sequence)
	}

	derivatives := sequence[1:]
	if len(derivatives) != 23 || derivatives[len(derivatives)-1] != "finalize_temario_package" {
		t.Fatalf("derivados OPES inesperados=%+v", derivatives)
	}

	for index, workKind := range derivatives {
		workKind := workKind
		t.Run(workKind, func(t *testing.T) {
			jobRef := fmt.Sprintf("job-ref-opes-seq-%02d", index+1)
			request := buildOPESGoalFirstSequenceRunRequestForTestV0(t, workKind, jobRef)
			started := postExternalWorkRunRequestGoalFirstStackV0(t, stack, request)
			spec := launcher.specs[len(launcher.specs)-1]
			contract := spec.ArtifactContracts[0]
			expectedArtifactType := orquestadomainwork.ExpectedDomainWorkArtifactTypeForWorkKindV0(workKind)
			if spec.RunRef != started.RunRef ||
				contract.ArtifactType != expectedArtifactType ||
				!spec.ClosurePolicy.RequireDomainReceipt {
				t.Fatalf("spec/contract inesperado: started=%+v spec=%+v contract=%+v expected=%s", started, spec, contract, expectedArtifactType)
			}

			writeGoalFirstDomainWorkArtifactForTestV0(t, stack.Codex.ProjectWorkDir, spec, contract.ArtifactType)
			observer.result = externalWorkGoalFirstCompleteResultForTestV0(
				spec,
				started.ExternalGoalRef,
			)

			result, err := stack.ObserveAppDirectorGoalV0(
				context.Background(),
				orquestaappdirectorservice.ObserveAppDirectorGoalRequestV0{
					RunRef:        started.RunRef,
					CorrelationID: "corr-" + jobRef + "-observe",
					RequestedBy:   "orquesta-app-codex-stack-test",
				},
			)
			if err != nil {
				t.Fatalf("ObserveAppDirectorGoalV0: %v", err)
			}
			if result.Run.Status != orquestacoreworkflow.OrchestrationRunStatusClosedV0 ||
				!result.Closure.Accepted ||
				!codexStackStringInSetForTestV0(result.Closure.EvidenceRefs, goalDomainReceiptLedgerAcceptedEvidenceRefV0) {
				t.Fatalf("goal-first OPES no cerro por ledger: result=%+v", result)
			}

			submit := domainWork.inputs[len(domainWork.inputs)-1]
			submission := submit.ArtifactSubmission
			if submit.Action != orquestamcp.MCPDomainWorkActionSubmitArtifactV0 ||
				submission.JobRef != jobRef ||
				submission.ArtifactRef != contract.ArtifactRef ||
				submission.ArtifactType != expectedArtifactType ||
				!submission.CompleteJob {
				t.Fatalf("submit_artifact inesperado=%+v contract=%+v", submit, contract)
			}

			records, err := stack.DomainDelivery.Ledger.(DomainWorkArtifactSubmissionRecordReaderPortV0).ListDomainWorkArtifactSubmissionsV0(
				context.Background(),
				DomainWorkArtifactSubmissionRecordFilterV0{
					RunRef: started.RunRef,
					Status: DomainWorkArtifactSubmissionStatusAcceptedV0,
				},
			)
			if err != nil || len(records) != 1 {
				t.Fatalf("records=%+v err=%v", records, err)
			}
			record := records[0]
			if record.RunRef != started.RunRef ||
				record.JobRef != jobRef ||
				record.ArtifactRef != contract.ArtifactRef ||
				record.ArtifactType != expectedArtifactType ||
				record.ReceiptRef == "" ||
				!record.CompleteJob {
				t.Fatalf("record ledger inesperado=%+v contract=%+v", record, contract)
			}
			if !codexStackStringInSetForTestV0(
				result.Closure.EvidenceRefs,
				"domain-work-goal-receipt-derived-"+safeDomainWorkEvidenceRefV0(record.ReceiptRef),
			) {
				t.Fatalf("closure no conserva receipt derivado: record=%+v closure=%+v", record, result.Closure)
			}
		})
	}
}

func buildOPESGoalFirstSequenceRunRequestForTestV0(
	t *testing.T,
	workKind string,
	jobRef string,
) orquestaexternalworkrun.StartExternalWorkRunRequestV0 {
	t.Helper()
	expectedArtifactType := orquestadomainwork.ExpectedDomainWorkArtifactTypeForWorkKindV0(workKind)
	payload := map[string]any{
		"program_id":             "program-ref-opes-sequence-001",
		"course_id":              "course-ref-opes-sequence-001",
		"topic_id":               "tema-001",
		"correlation_id":         "corr-opes-sequence-001",
		"expected_artifact_type": expectedArtifactType,
		"source_refs":            []string{"source-ref-opes-sequence-001"},
		"required_outputs":       []string{expectedArtifactType},
	}
	rawPayload, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Marshal payload: %v", err)
	}
	request, ok := orquestaopesbridge.BuildExternalWorkRunRequestV0(
		orquestaopesconnector.ExternalJobV0{
			ID:            jobRef,
			Type:          workKind,
			CorrelationID: "corr-" + jobRef,
			RequestedBy:   "opes",
			PayloadJSON:   string(rawPayload),
		},
		orquestaopesbridge.JobRunConfigV0{
			PriorityScore: 80,
			RequestedBy:   "opes-sequence-test",
		},
	)
	if !ok {
		t.Fatalf("request OPES no construida para %s", workKind)
	}
	return request
}

func postExternalWorkRunRequestGoalFirstStackV0(
	t *testing.T,
	stack StackV0,
	request orquestaexternalworkrun.StartExternalWorkRunRequestV0,
) orquestamcp.MCPExternalWorkRunToolResultV0 {
	t.Helper()
	body := bytes.NewBuffer(nil)
	if err := json.NewEncoder(body).Encode(orquestamcp.MCPExternalWorkRunToolInputV0{
		RequestID:              request.RequestID,
		CorrelationID:          request.CorrelationID,
		ExternalWorkRunRequest: request,
	}); err != nil {
		t.Fatalf("encode external work run: %v", err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v0/external-work/run", body)
	req.Header.Set("Content-Type", "application/json")
	stack.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("external work run status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result orquestamcp.MCPExternalWorkRunToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode external work run: %v", err)
	}
	if result.Estado != orquestamcp.MCPExternalWorkRunEstadoOKV0 ||
		result.RoutePolicy != orquestamcp.MCPExternalWorkRunRoutePolicyGoalFirstV0 {
		t.Fatalf("external work run result=%+v", result)
	}
	return result
}
