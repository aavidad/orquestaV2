package orquestaappcodexstack

import (
	"context"
	"testing"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
	orquestaappchangedirectorsource "orquesta/modulos/orquesta-app-change-director-source"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoragent "orquesta/modulos/orquesta-director-agent"
	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func TestCompositeDirectorDecisionSourceV0UsaPoliticaDomainWorkRequiredTests(t *testing.T) {
	record := codexStackDomainWorkPolicyRecordForTestV0("policy")
	store := orquestaappchange.NewInMemoryAppChangeStoreV0(record)
	taskRef := orquestaappchangedirectorsource.AppChangeTaskRefV0(record.Request.ChangeRef)
	decision := orquestadirectoragent.DirectorAgentDecisionV0{
		CommandType: orquestadirectoragent.DirectorAgentCommandCreateMicrotaskV0,
		CreateMicrotask: &orquestadirectoragent.DirectorAgentCreateMicrotaskCommandV0{
			Task: orquestadirectoragent.DirectorAgentMicrotaskV0{
				TaskID:        taskRef,
				RequiredTests: []string{"validar contrato externo de dominio"},
				FunctionContractRefs: []orquestadirectoragent.DirectorAgentFunctionContractRefV0{{
					FunctionName: "ApplyExternalDomainWorkV0",
				}},
			},
		},
	}
	source := compositeDirectorDecisionSourceV0{
		AppChangeStore:       store,
		DomainRequiredPolicy: orquestadomainwork.DeclaredDomainWorkRequiredTestPolicyV0{},
	}

	got, err := source.normalizeDomainWorkRequiredTestsV0(
		context.Background(),
		orquestacoreworkflow.OrchestrationRunV0{RunID: record.Request.RunRef},
		[]orquestadirectoragent.DirectorAgentDecisionV0{decision},
	)

	if err != nil {
		t.Fatalf("normalizeDomainWorkRequiredTestsV0: %v", err)
	}
	tests := got[0].CreateMicrotask.Task.RequiredTests
	if len(tests) != 1 || tests[0] != "domain-test-ref-policy" ||
		codexStackStringInSetForTestV0(tests, "validar contrato externo de dominio") {
		t.Fatalf("required_tests=%+v", tests)
	}
}

func TestDomainWorkRequiredTestRunnerV0DistingueLatenciaRechazoYAceptacion(t *testing.T) {
	ctx := context.Background()
	record := codexStackDomainWorkPolicyRecordForTestV0("runner")
	store := orquestaappchange.NewInMemoryAppChangeStoreV0(record)
	ledger := NewInMemoryDomainWorkArtifactSubmissionLedgerV0()
	evidence := orquestacionnucleoapp.NewInMemoryRequiredTestEvidenceStoreV0()
	runner := DomainWorkRequiredTestRunnerV0{
		Policy:           orquestadomainwork.DeclaredDomainWorkRequiredTestPolicyV0{},
		AppChangeStore:   store,
		SubmissionLedger: ledger,
		EvidenceWriter:   evidence,
	}
	request := codexStackDomainWorkRequiredTestRequestForTestV0(record, "latency")

	result, err := runner.RunRequiredTestsV0(ctx, request)
	if err != nil || len(result.EvidenceRefs) != 0 {
		t.Fatalf("latencia debe quedar sin evidencia: result=%+v err=%v", result, err)
	}

	rejected := codexStackDomainWorkRequiredTestRequestForTestV0(record, "rejected")
	if err := ledger.RecordDomainWorkArtifactSubmissionV0(ctx, DomainWorkArtifactSubmissionRecordV0{
		IdempotencyKey: "idem-rejected",
		Status:         DomainWorkArtifactSubmissionStatusRejectedV0,
		RunRef:         rejected.RunRef,
		TaskRef:        rejected.TaskRef,
		DeliveryRef:    rejected.DeliveryRef,
		EvidenceRefs:   []string{"artifact-ref-rejected"},
		IssueRefs:      []string{"domain-rejected"},
	}); err != nil {
		t.Fatalf("record rejected: %v", err)
	}
	result, err = runner.RunRequiredTestsV0(ctx, rejected)
	if err != nil || len(result.FailedEvidenceRefs) != 1 {
		t.Fatalf("rechazo debe generar failed evidence: result=%+v err=%v", result, err)
	}

	accepted := codexStackDomainWorkRequiredTestRequestForTestV0(record, "accepted")
	if err := ledger.RecordDomainWorkArtifactSubmissionV0(ctx, DomainWorkArtifactSubmissionRecordV0{
		IdempotencyKey: "idem-accepted",
		Status:         DomainWorkArtifactSubmissionStatusAcceptedV0,
		RunRef:         accepted.RunRef,
		TaskRef:        accepted.TaskRef,
		DeliveryRef:    accepted.DeliveryRef,
		ReceiptRef:     "receipt-ref-accepted",
		EvidenceRefs:   []string{"artifact-ref-accepted"},
	}); err != nil {
		t.Fatalf("record accepted: %v", err)
	}
	result, err = runner.RunRequiredTestsV0(ctx, accepted)
	if err != nil || len(result.PassedEvidenceRefs) != 1 {
		t.Fatalf("aceptacion debe generar passed evidence: result=%+v err=%v", result, err)
	}
}

func codexStackDomainWorkPolicyRecordForTestV0(
	suffix string,
) orquestaappchange.AppChangeRecordV0 {
	return orquestaappchange.AppChangeRecordV0{
		Request: orquestaappchange.PrepareAppChangeRequestV0(orquestaappchange.AppChangeRequestV0{
			RequestID:     "req-domain-policy-" + suffix,
			CorrelationID: "corr-domain-policy-" + suffix,
			RunRef:        "run-domain-policy-" + suffix,
			ChangeRef:     "change-domain-policy-" + suffix,
			UserIntent:    "Resolver trabajo de dominio externo.",
			ExternalWork: &orquestaappchange.AppChangeExternalWorkV0{
				ProjectRef: "domain-ref-policy",
				JobRef:     "job-ref-policy",
				WorkKind:   "compose_external_summary",
				RequiredTests: []orquestadomainwork.DomainWorkRequiredTestV0{{
					TestRef:                "domain-test-ref-policy",
					AcceptanceCriteriaRefs: []string{"criteria-ref-policy"},
					EvidenceRefs:           []string{"artifact-ref-policy"},
				}},
			},
		}),
	}
}

func codexStackDomainWorkRequiredTestRequestForTestV0(
	record orquestaappchange.AppChangeRecordV0,
	suffix string,
) orquestacionnucleoapp.RequiredTestExecutionRequestV0 {
	return orquestacionnucleoapp.RequiredTestExecutionRequestV0{
		RunRef:            record.Request.RunRef,
		TaskRef:           orquestaappchangedirectorsource.AppChangeTaskRefV0(record.Request.ChangeRef),
		TestCommands:      []string{"domain-test-ref-policy"},
		DeliveryRef:       "delivery-ref-" + suffix,
		ReviewRequestID:   "review-request-ref-" + suffix,
		ReviewResultRef:   "review-result-ref-" + suffix,
		AcceptedReviewRef: "accepted-review-ref-" + suffix,
		OccurredAt:        "2026-05-24T12:00:00Z",
		CorrelationID:     record.Request.CorrelationID,
		EvidenceRefs:      []string{"review-evidence-ref-" + suffix},
	}
}
