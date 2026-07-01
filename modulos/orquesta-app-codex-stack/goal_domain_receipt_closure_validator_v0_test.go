package orquestaappcodexstack

import (
	"context"
	"testing"

	orquestagoal "orquesta/modulos/orquesta-goal"
)

func TestGoalDomainReceiptClosureValidatorV0HidrataRequiredTestDominioSinEvidenciaV0(t *testing.T) {
	ctx := context.Background()
	ledger := NewInMemoryDomainWorkArtifactSubmissionLedgerV0()
	spec := goalDomainReceiptRequiredTestSpecForTestV0("test-ref-domain-required-001", "", "")
	record := goalDomainReceiptAcceptedRecordForTestV0(spec, "receipt-ref-required-test-domain-001")
	if err := ledger.RecordDomainWorkArtifactSubmissionV0(ctx, record); err != nil {
		t.Fatalf("RecordDomainWorkArtifactSubmissionV0: %v", err)
	}
	result := goalDomainReceiptRequiredTestResultForTestV0(spec, record.ReceiptRef, []orquestagoal.GoalRequiredTestResultV0{{
		TestRef: "test-ref-domain-required-001",
		Status:  "passed",
	}})

	hydrated := (domainWorkGoalReceiptClosureValidatorV0{Ledger: ledger}).
		goalResultWithAcceptedDomainRequiredTestResultsV0(ctx, spec, result)

	if len(hydrated.RequiredTestResults) != 1 ||
		len(hydrated.RequiredTestResults[0].EvidenceRefs) == 0 ||
		!codexStackStringInSetV0(hydrated.RequiredTestResults[0].EvidenceRefs, record.ReceiptRef) ||
		!codexStackStringInSetV0(hydrated.RequiredTestResults[0].EvidenceRefs, "evidence-ref-required-test-ledger-001") {
		t.Fatalf("required_test_results sin evidencia durable: %+v", hydrated.RequiredTestResults)
	}
}

func TestGoalDomainReceiptClosureValidatorV0BloqueaRequiredTestComandoPassedSinEvidenciaV0(t *testing.T) {
	ctx := context.Background()
	ledger := NewInMemoryDomainWorkArtifactSubmissionLedgerV0()
	spec := goalDomainReceiptRequiredTestSpecForTestV0("test-ref-command-required-001", "go test ./...", "")
	record := goalDomainReceiptAcceptedRecordForTestV0(spec, "receipt-ref-required-test-command-001")
	if err := ledger.RecordDomainWorkArtifactSubmissionV0(ctx, record); err != nil {
		t.Fatalf("RecordDomainWorkArtifactSubmissionV0: %v", err)
	}
	result := goalDomainReceiptRequiredTestResultForTestV0(spec, record.ReceiptRef, []orquestagoal.GoalRequiredTestResultV0{{
		TestRef: "test-ref-command-required-001",
		Status:  "passed",
	}})

	closure, err := (domainWorkGoalReceiptClosureValidatorV0{Ledger: ledger}).ValidateGoalWorkClosureV0(ctx, spec, result)
	if err != nil {
		t.Fatalf("ValidateGoalWorkClosureV0: %v", err)
	}
	if closure.Accepted ||
		closure.Status != orquestagoal.GoalStatusBlockedV0 ||
		!goalDomainReceiptClosureHasIssueForTestV0(closure, goalDomainReceiptRequiredTestEvidenceMissingIssueV0, goalDomainReceiptRequiredTestEvidenceMissingFieldV0) {
		t.Fatalf("closure no bloqueo required_test_results sin evidence_refs: %+v", closure)
	}
}

func goalDomainReceiptRequiredTestSpecForTestV0(
	testRef string,
	command string,
	commandRef string,
) orquestagoal.GoalWorkSpecV0 {
	return orquestagoal.NormalizeGoalWorkSpecV0(orquestagoal.GoalWorkSpecV0{
		GoalRef:   "goal-ref-required-test-evidence-001",
		RunRef:    "run-ref-required-test-evidence-001",
		Objective: "Validar evidencia durable de required tests.",
		WriteSet:  []orquestagoal.GoalWriteScopeV0{{Path: "external/opes/tema_012/trabajo"}},
		RequiredTests: []orquestagoal.GoalRequiredTestV0{{
			TestRef:    testRef,
			Command:    command,
			CommandRef: commandRef,
		}},
		ArtifactContracts: []orquestagoal.GoalArtifactContractV0{{
			ArtifactRef:  "artifact-ref-required-test-evidence-001",
			ArtifactType: "document_plan",
			Required:     true,
		}},
		ClosurePolicy: orquestagoal.GoalClosurePolicyV0{
			RequireArtifacts:     true,
			RequireDomainReceipt: true,
			RequireRequiredTests: true,
		},
	})
}

func goalDomainReceiptAcceptedRecordForTestV0(
	spec orquestagoal.GoalWorkSpecV0,
	receiptRef string,
) DomainWorkArtifactSubmissionRecordV0 {
	return DomainWorkArtifactSubmissionRecordV0{
		IdempotencyKey: "idempotency-key-" + receiptRef,
		Status:         DomainWorkArtifactSubmissionStatusAcceptedV0,
		RunRef:         spec.RunRef,
		DomainRef:      "opes",
		JobRef:         "job-ref-required-test-evidence-001",
		ArtifactRef:    spec.ArtifactContracts[0].ArtifactRef,
		ArtifactType:   spec.ArtifactContracts[0].ArtifactType,
		CompleteJob:    true,
		ReceiptRef:     receiptRef,
		EvidenceRefs:   []string{"evidence-ref-required-test-ledger-001"},
	}
}

func goalDomainReceiptRequiredTestResultForTestV0(
	spec orquestagoal.GoalWorkSpecV0,
	receiptRef string,
	requiredTestResults []orquestagoal.GoalRequiredTestResultV0,
) orquestagoal.GoalWorkResultV0 {
	return orquestagoal.NormalizeGoalWorkResultV0(orquestagoal.GoalWorkResultV0{
		GoalRef:             spec.GoalRef,
		Status:              orquestagoal.GoalStatusCompleteV0,
		ArtifactRefs:        []string{spec.ArtifactContracts[0].ArtifactRef},
		DomainReceiptRefs:   []string{receiptRef},
		EvidenceRefs:        []string{"evidence-ref-required-test-result-001"},
		RequiredTestResults: requiredTestResults,
	})
}

func goalDomainReceiptClosureHasIssueForTestV0(
	closure orquestagoal.GoalClosureValidationV0,
	code string,
	field string,
) bool {
	for _, issue := range closure.Issues {
		if issue.Code == code && issue.Field == field {
			return true
		}
	}
	return false
}
