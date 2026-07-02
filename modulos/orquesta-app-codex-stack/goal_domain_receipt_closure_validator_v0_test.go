package orquestaappcodexstack

import (
	"context"
	"testing"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
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

func TestGoalDomainReceiptClosureValidatorV0BloqueaOPESFinalConQAGenericoSinStrictEditorialV0(t *testing.T) {
	ctx := context.Background()
	ledger := NewInMemoryDomainWorkArtifactSubmissionLedgerV0()
	spec := orquestagoal.NormalizeGoalWorkSpecV0(orquestagoal.GoalWorkSpecV0{
		GoalRef:    "goal-ref-opes-final-strict-editorial-001",
		RunRef:     "run-ref-opes-final-strict-editorial-001",
		DomainRef:  "opes",
		ProjectRef: "opes",
		WorkKind:   "finalize_temario_package",
		Objective:  "Cerrar paquete final OPES.",
		WriteSet:   []orquestagoal.GoalWriteScopeV0{{Path: "external/opes/final"}},
		ArtifactContracts: []orquestagoal.GoalArtifactContractV0{{
			ArtifactRef:  "artifact-ref-opes-final-strict-editorial-001",
			ArtifactType: orquestadomainwork.DomainWorkArtifactTypeFinalDomainPackageV0,
			Required:     true,
		}},
		ClosurePolicy: orquestagoal.GoalClosurePolicyV0{
			RequireArtifacts:     true,
			RequireDomainReceipt: true,
		},
	})
	record := DomainWorkArtifactSubmissionRecordV0{
		IdempotencyKey: "idempotency-key-opes-final-strict-editorial-001",
		Status:         DomainWorkArtifactSubmissionStatusAcceptedV0,
		RunRef:         spec.RunRef,
		DomainRef:      "opes",
		JobRef:         "job-ref-opes-final-strict-editorial-001",
		ArtifactRef:    spec.ArtifactContracts[0].ArtifactRef,
		ArtifactType:   spec.ArtifactContracts[0].ArtifactType,
		CompleteJob:    true,
		ReceiptRef:     "receipt-ref-opes-final-strict-editorial-001",
		PayloadFields: []orquestadomainwork.DomainWorkFieldV0{
			{Name: "source_work_kind", Value: "finalize_temario_package"},
			{Name: "manifest_cierre", ValueJSON: []byte(`{
				"schema_version":"opes_final_package_evidence_manifest.v0",
				"package_ref":"package-ref-opes-final-strict-editorial-001",
				"manifest_ref":"manifest-cierre-ref-opes-final-strict-editorial-001",
				"checksum_refs":["checksum-ref-opes-final-strict-editorial-001"],
				"validation_report_ref":"validation-report-ref-opes-final-strict-editorial-001",
				"review_matrix_ref":"review-matrix-ref-opes-final-strict-editorial-001",
				"required_evidence_refs":{
					"html":"opes-final-evidence:html:strict-editorial",
					"rag":"opes-final-evidence:rag:strict-editorial",
					"audio":"opes-final-evidence:audio:strict-editorial",
					"tests":"opes-final-evidence:tests:strict-editorial",
					"tutor":"opes-final-evidence:tests:strict-editorial-tutor",
					"visual":"opes-final-evidence:visual:strict-editorial",
					"qa":"opes-final-evidence:qa:strict-editorial"
				}
			}`)},
		},
		EvidenceRefs: []string{"opes-final-evidence:qa:strict-editorial"},
	}
	if err := ledger.RecordDomainWorkArtifactSubmissionV0(ctx, record); err != nil {
		t.Fatalf("RecordDomainWorkArtifactSubmissionV0: %v", err)
	}
	result := orquestagoal.NormalizeGoalWorkResultV0(orquestagoal.GoalWorkResultV0{
		GoalRef:           spec.GoalRef,
		Status:            orquestagoal.GoalStatusCompleteV0,
		ArtifactRefs:      []string{spec.ArtifactContracts[0].ArtifactRef},
		DomainReceiptRefs: []string{record.ReceiptRef},
		EvidenceRefs:      []string{"evidence-ref-opes-final-strict-editorial-001"},
	})

	closure, err := (domainWorkGoalReceiptClosureValidatorV0{Ledger: ledger}).ValidateGoalWorkClosureV0(ctx, spec, result)
	if err != nil {
		t.Fatalf("ValidateGoalWorkClosureV0: %v", err)
	}
	if closure.Accepted ||
		closure.Status != orquestagoal.GoalStatusBlockedV0 ||
		!goalDomainReceiptClosureHasIssueForTestV0(closure, goalDomainReceiptOPESFinalPackageStrictEditorialIssueV0, goalDomainReceiptOPESFinalPackageQAPassesFieldV0) {
		t.Fatalf("closure no bloqueo QA estricta ausente: %+v", closure)
	}
}

func TestGoalDomainReceiptClosureValidatorV0BloqueaOPESFinalSinTopicQualityContractRefsV0(t *testing.T) {
	ctx := context.Background()
	ledger := NewInMemoryDomainWorkArtifactSubmissionLedgerV0()
	spec := orquestagoal.NormalizeGoalWorkSpecV0(orquestagoal.GoalWorkSpecV0{
		GoalRef:    "goal-ref-opes-final-topic-quality-001",
		RunRef:     "run-ref-opes-final-topic-quality-001",
		DomainRef:  "opes",
		ProjectRef: "opes",
		WorkKind:   "finalize_temario_package",
		Objective:  "Cerrar paquete final OPES.",
		WriteSet:   []orquestagoal.GoalWriteScopeV0{{Path: "external/opes/final"}},
		ArtifactContracts: []orquestagoal.GoalArtifactContractV0{{
			ArtifactRef:  "artifact-ref-opes-final-topic-quality-001",
			ArtifactType: orquestadomainwork.DomainWorkArtifactTypeFinalDomainPackageV0,
			Required:     true,
		}},
		ClosurePolicy: orquestagoal.GoalClosurePolicyV0{
			RequireArtifacts:     true,
			RequireDomainReceipt: true,
		},
	})
	record := DomainWorkArtifactSubmissionRecordV0{
		IdempotencyKey: "idempotency-key-opes-final-topic-quality-001",
		Status:         DomainWorkArtifactSubmissionStatusAcceptedV0,
		RunRef:         spec.RunRef,
		DomainRef:      "opes",
		JobRef:         "job-ref-opes-final-topic-quality-001",
		ArtifactRef:    spec.ArtifactContracts[0].ArtifactRef,
		ArtifactType:   spec.ArtifactContracts[0].ArtifactType,
		CompleteJob:    true,
		ReceiptRef:     "receipt-ref-opes-final-topic-quality-001",
		PayloadFields: []orquestadomainwork.DomainWorkFieldV0{
			{Name: "source_work_kind", Value: "finalize_temario_package"},
			{Name: "manifest_cierre", ValueJSON: []byte(`{
				"schema_version":"opes_final_package_evidence_manifest.v0",
				"package_ref":"package-ref-opes-final-topic-quality-001",
				"manifest_ref":"manifest-cierre-ref-opes-final-topic-quality-001",
				"checksum_refs":["checksum-ref-opes-final-topic-quality-001"],
				"validation_report_ref":"validation-report-ref-opes-final-topic-quality-001",
				"review_matrix_ref":"review-matrix-ref-opes-final-topic-quality-001",
				"qa_passes":{
					"extension_pass":true,
					"official_text_qa_pass":true,
					"strict_editorial_qa_pass":true,
				"question_bank_publicable":true,
				"tutor_assets_publicable":true
				},
				"qa_report_refs":{
					"extension":"09_validacion/informe_extension_temario.json",
					"official_text":"09_validacion/informe_texto_publico_sin_notas_autor.json",
					"strict_editorial":"09_validacion/informe_texto_publico_sin_andamiaje_interno.json",
				"question_bank_publicable":"09_validacion/informe_question_bank_publicable.json",
				"tutor_assets_publicable":"09_validacion/informe_tutor_assets_publicable.json"
				},
				"required_evidence_refs":{
					"html":"opes-final-evidence:html:topic-quality",
					"rag":"opes-final-evidence:rag:topic-quality",
					"audio":"opes-final-evidence:audio:topic-quality",
					"tests":"opes-final-evidence:tests:topic-quality",
					"tutor":"opes-final-evidence:tests:topic-quality-tutor",
					"visual":"opes-final-evidence:visual:topic-quality",
					"qa":"opes-final-evidence:qa:topic-quality"
				}
			}`)},
		},
		EvidenceRefs: []string{
			"opes-extension-minima-passed",
			"opes-common-master-not-applicable",
			"opes-final-evidence:qa:topic-quality",
		},
	}
	if err := ledger.RecordDomainWorkArtifactSubmissionV0(ctx, record); err != nil {
		t.Fatalf("RecordDomainWorkArtifactSubmissionV0: %v", err)
	}
	result := orquestagoal.NormalizeGoalWorkResultV0(orquestagoal.GoalWorkResultV0{
		GoalRef:           spec.GoalRef,
		Status:            orquestagoal.GoalStatusCompleteV0,
		ArtifactRefs:      []string{spec.ArtifactContracts[0].ArtifactRef},
		DomainReceiptRefs: []string{record.ReceiptRef},
		EvidenceRefs:      []string{"evidence-ref-opes-final-topic-quality-001"},
	})

	closure, err := (domainWorkGoalReceiptClosureValidatorV0{Ledger: ledger}).ValidateGoalWorkClosureV0(ctx, spec, result)
	if err != nil {
		t.Fatalf("ValidateGoalWorkClosureV0: %v", err)
	}
	if closure.Accepted ||
		closure.Status != orquestagoal.GoalStatusBlockedV0 ||
		!goalDomainReceiptClosureHasIssueForTestV0(
			closure,
			codexStackOPESFinalPackageTopicQualityMissingIssueV0,
			goalDomainReceiptOPESFinalPackageTopicQualityFieldV0,
		) {
		t.Fatalf("closure no bloqueo topic_quality_contract_result_refs ausente: %+v", closure)
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
