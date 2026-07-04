package orquestaappcodexstack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestadomainworkmemory "orquesta/modulos/orquesta-domain-work-memory"
	orquestaexternalworkrun "orquesta/modulos/orquesta-external-work-run"
	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestaopesbridge "orquesta/modulos/orquesta-opes-bridge"
	orquestaopesconnector "orquesta/modulos/orquesta-opes-connector"
	orquestaopesdirector "orquesta/modulos/orquesta-opes-director"
)

func TestCodexStackV0ExternalWorkGoalFirstCierraSecuenciaOPESDerivadosConReceiptsLedgerV0(t *testing.T) {
	stack, observer, launcher, domainWork := buildExternalWorkGoalFirstDomainDeliveryStackWithExecutorForTestV0(t)
	sequence := orquestaopesbridge.OPESFullTemarioJobTypeSequenceV0()
	if len(sequence) != 25 || sequence[0] != "plan_temario" {
		t.Fatalf("secuencia OPES inesperada=%+v", sequence)
	}

	derivatives := sequence[1:]
	if len(derivatives) != 24 || derivatives[len(derivatives)-1] != "finalize_temario_package" {
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
			observer.result.EvidenceRefs = append(observer.result.EvidenceRefs, "evidence-ref-goal-materialized-checkpoint-detected")

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
			if !domainWorkFieldValueForTestV0(submission.PayloadFields, "director_execution_mode", "goal_first") ||
				!domainWorkFieldValueForTestV0(submission.PayloadFields, "goal_first_status", "complete") ||
				!domainWorkFieldValueForTestV0(submission.PayloadFields, "external_goal_ref", started.ExternalGoalRef) ||
				!domainWorkFieldValuesForTestV0(submission.PayloadFields, "goal_first_checkpoint_refs", []string{"evidence-ref-goal-materialized-checkpoint-detected"}) ||
				!domainWorkFieldContainsValuesForTestV0(
					submission.PayloadFields,
					"orquesta_goal_result_refs",
					[]string{
						spec.GoalRef,
						started.ExternalGoalRef,
						contract.ArtifactRef,
						"evidence-ref-external-work-goal-first-required-test",
						"evidence-ref-external-work-goal-spec-v0",
						"evidence-ref-goal-materialized-checkpoint-detected",
					},
				) {
				t.Fatalf("payload lifecycle goal-first ausente: fields=%+v", submission.PayloadFields)
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

func TestCodexStackV0OPESGoalFirstLifecycleAsientaDerivadosYCierraRegistroFinalConTopicQualityV0(t *testing.T) {
	stack, observer, launcher, domainWork := buildExternalWorkGoalFirstDomainDeliveryStackWithExecutorForTestV0(t)
	creator := orquestadomainworkmemory.NewInMemoryDomainWorkJobCreatorV0()

	textSubmission := runOPESGoalFirstSequenceWorkAndSubmitForTestV0(
		t,
		stack,
		observer,
		launcher,
		domainWork,
		"draft_content_block",
		"job-ref-opes-lc-t",
		opesGoalFirstLifecycleTextArtifactJSONForTestV0(),
	)
	textResult := produceOPESCausalJobsFromStackSubmissionsForTestV0(t, creator, textSubmission)
	textUpdate := opesDirectorRequestedWorkKindForStackTestV0(t, textResult.RequestedJobs, "update_topic_registry")
	if !domainWorkFieldValueForTestV0(textUpdate.InputFields, "registry_action", "update") ||
		!domainWorkFieldValueForTestV0(textUpdate.InputFields, "proposed_status", "texto_asentado_pendiente_derivados") ||
		!domainWorkFieldValueForTestV0(textUpdate.InputFields, "operational_status", "waiting") ||
		!domainWorkFieldValueForTestV0(textUpdate.InputFields, "settlement_status", "settled_text") ||
		!domainWorkFieldValueForTestV0(textUpdate.InputFields, "settlement_reason", "topic_quality_contract_passed") ||
		!domainWorkFieldContainsValuesForTestV0(textUpdate.InputFields, "next_required_work_kinds", []string{
			"generate_visual_asset",
			"generate_question_bank",
			"generate_html_site",
			"finalize_temario_package",
		}) {
		t.Fatalf("registro texto no asienta y deriva: %+v", textUpdate)
	}

	finalWithTopicQuality := runOPESGoalFirstSequenceWorkAndSubmitForTestV0(
		t,
		stack,
		observer,
		launcher,
		domainWork,
		"finalize_temario_package",
		"job-ref-opes-lc-f",
		opesGoalFirstLifecycleFinalArtifactJSONForTestV0(true),
	)
	finalResult := produceOPESCausalJobsFromStackSubmissionsForTestV0(t, creator, finalWithTopicQuality)
	finalUpdate := opesDirectorRequestedWorkKindForStackTestV0(t, finalResult.RequestedJobs, "update_topic_registry")
	if !domainWorkFieldValueForTestV0(finalUpdate.InputFields, "registry_action", "release") ||
		!domainWorkFieldValueForTestV0(finalUpdate.InputFields, "proposed_status", "paquete_final_local_verificable") ||
		!domainWorkFieldValueForTestV0(finalUpdate.InputFields, "operational_status", "complete") ||
		!domainWorkFieldValueForTestV0(finalUpdate.InputFields, "settlement_status", "settled_final") ||
		!domainWorkFieldValueForTestV0(finalUpdate.InputFields, "settlement_reason", "final_package_closure_evidence_complete") ||
		!domainWorkFieldContainsValuesForTestV0(finalUpdate.InputFields, "settled_refs", []string{"topic-quality-contract-result-ref-lifecycle-final-001"}) ||
		domainWorkFieldContainsValuesForTestV0(finalUpdate.InputFields, "pending_refs", []string{"final-package-manifest-closure-evidence-required"}) {
		t.Fatalf("registro final con topic quality debe liberar: %+v", finalUpdate)
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

func runOPESGoalFirstSequenceWorkAndSubmitForTestV0(
	t *testing.T,
	stack StackV0,
	observer *goalFirstQueueObserverForTestV0,
	launcher *goalFirstQueueLauncherForTestV0,
	domainWork *fakeCodexStackDomainWorkExecutorV0,
	workKind string,
	jobRef string,
	artifactJSON string,
) orquestadomainwork.DomainWorkArtifactSubmissionV0 {
	t.Helper()
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

	writeGoalFirstDomainWorkArtifactRawForTestV0(t, stack.Codex.ProjectWorkDir, spec, contract.ArtifactType, artifactJSON)
	observer.result = externalWorkGoalFirstCompleteResultForTestV0(spec, started.ExternalGoalRef)
	observer.result.EvidenceRefs = append(observer.result.EvidenceRefs, "evidence-ref-goal-materialized-checkpoint-detected")

	result, err := stack.ObserveAppDirectorGoalV0(
		context.Background(),
		orquestaappdirectorservice.ObserveAppDirectorGoalRequestV0{
			RunRef:        started.RunRef,
			CorrelationID: "corr-" + jobRef + "-observe",
			RequestedBy:   "orquesta-app-codex-stack-test",
		},
	)
	if err != nil {
		if len(domainWork.inputs) > 0 {
			last := domainWork.inputs[len(domainWork.inputs)-1].ArtifactSubmission
			t.Fatalf("ObserveAppDirectorGoalV0: %v last_submission=%+v validation_issues=%+v", err, last, orquestadomainwork.ValidateDomainWorkArtifactSubmissionV0(last))
		}
		t.Fatalf("ObserveAppDirectorGoalV0: %v", err)
	}
	if result.Run.Status != orquestacoreworkflow.OrchestrationRunStatusClosedV0 ||
		!result.Closure.Accepted ||
		!codexStackStringInSetForTestV0(result.Closure.EvidenceRefs, goalDomainReceiptLedgerAcceptedEvidenceRefV0) {
		t.Fatalf("goal-first OPES no cerro por ledger: result=%+v", result)
	}

	submit := domainWork.inputs[len(domainWork.inputs)-1]
	if submit.Action != orquestamcp.MCPDomainWorkActionSubmitArtifactV0 ||
		submit.ArtifactSubmission.JobRef != jobRef ||
		submit.ArtifactSubmission.ArtifactType != expectedArtifactType ||
		!submit.ArtifactSubmission.CompleteJob {
		t.Fatalf("submit_artifact inesperado=%+v expected=%s", submit, expectedArtifactType)
	}
	if !domainWorkFieldValueForTestV0(submit.ArtifactSubmission.PayloadFields, "director_execution_mode", "goal_first") ||
		!domainWorkFieldValuesForTestV0(
			submit.ArtifactSubmission.PayloadFields,
			"goal_first_checkpoint_refs",
			[]string{"evidence-ref-goal-materialized-checkpoint-detected"},
		) {
		t.Fatalf("payload lifecycle goal-first ausente: %+v", submit.ArtifactSubmission.PayloadFields)
	}
	return submit.ArtifactSubmission
}

func writeGoalFirstDomainWorkArtifactRawForTestV0(
	t *testing.T,
	projectDir string,
	spec orquestagoal.GoalWorkSpecV0,
	artifactType string,
	body string,
) {
	t.Helper()
	if len(spec.WriteSet) == 0 {
		t.Fatalf("spec sin write-set=%+v", spec)
	}
	dir := filepath.Join(projectDir, filepath.FromSlash(spec.WriteSet[0].Path))
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, artifactType+".json"), []byte(body), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

func opesGoalFirstLifecycleTextArtifactJSONForTestV0() string {
	return `{
		"artifact_type":"content_block",
		"payload_json":{
			"course_id":"course-ref-opes-lifecycle-001",
			"topic_id":"tema-001",
			"source_work_kind":"draft_content_block",
			"status":"ready",
			"canonical_word_count":"12000",
			"topic_quality_status":"passed",
			"topic_quality_evidence_refs":["topic-quality-contract-result-ref-lifecycle-text-001"],
			"topic_text":"Contenido publico final del tema OPES con desarrollo normativo y ejemplos profesionales."
		}
	}`
}

func opesGoalFirstLifecycleFinalArtifactJSONForTestV0(includeTopicQuality bool) string {
	topicQuality := ""
	if includeTopicQuality {
		topicQuality = `"topic_quality_contract_result_refs":{"tema_001":"topic-quality-contract-result-ref-lifecycle-final-001"},`
	}
	return `{
		"artifact_type":"completed_syllabus_package",
		"payload_json":{
			"course_id":"course-ref-opes-lifecycle-001",
			"topic_id":"tema-001",
			"package_ref":"package-ref-lifecycle-final-001",
			"manifest_cierre":{
				"schema_version":"opes_final_package_evidence_manifest.v0",
				"package_ref":"package-ref-lifecycle-final-001",
				"manifest_ref":"manifest-cierre-ref-lifecycle-final-001",
				"checksum_refs":["checksum-ref-lifecycle-final-001"],
				"validation_report_ref":"validation-report-ref-lifecycle-final-001",
				"review_matrix_ref":"review-matrix-ref-lifecycle-final-001",
				` + topicQuality + `
				"qa_passes":{
					"extension_pass":true,
					"official_text_qa_pass":true,
					"strict_editorial_qa_pass":true,
					"question_bank_publicable":true,
					"tutor_assets_publicable":true
				},
				"qa_report_refs":{
					"extension":"09_validacion/informe_extension_temario.json",
					"official_text":[
						"09_validacion/informe_texto_publico_sin_notas_autor.json",
						"09_validacion/informe_texto_publico_sin_metacomentarios_examen.json"
					],
					"strict_editorial":"09_validacion/informe_texto_publico_sin_andamiaje_interno.json",
					"question_bank_publicable":"09_validacion/informe_question_bank_publicable.json",
					"tutor_assets_publicable":"09_validacion/informe_tutor_assets_publicable.json"
				},
				"required_evidence_refs":{
					"html":"opes-final-evidence:html:lifecycle",
					"rag":"opes-final-evidence:rag:lifecycle",
					"audio":"opes-final-evidence:audio:lifecycle",
					"tests":"opes-final-evidence:tests:lifecycle",
					"tutor":"opes-final-evidence:tutor:lifecycle",
					"visual":"opes-final-evidence:visual:lifecycle",
					"qa":"opes-final-evidence:qa:lifecycle"
				}
			}
		}
	}`
}

func produceOPESCausalJobsFromStackSubmissionsForTestV0(
	t *testing.T,
	creator orquestadomainwork.DomainWorkJobRecordStorePortV0,
	submissions ...orquestadomainwork.DomainWorkArtifactSubmissionV0,
) orquestaopesdirector.OPESCausalProducerResultV0 {
	t.Helper()
	records := make([]orquestaopesdirector.OPESCausalArtifactRecordV0, 0, len(submissions))
	for _, submission := range submissions {
		records = append(records, opesCausalRecordFromDomainWorkSubmissionForStackTestV0(submission))
	}
	result, err := orquestaopesdirector.ProduceOPESCausalJobsV0(
		context.Background(),
		orquestaopesdirector.OPESCausalProducerRequestV0{DomainRef: orquestaopesdirector.OPESCausalProducerDefaultDomainRefV0},
		orquestaopesdirector.OPESCausalProducerPortsV0{
			ArtifactSource: opesCausalRecordSourceForStackTestV0{records: records},
			JobCreator:     creator,
			JobRecords:     creator,
		},
	)
	if err != nil {
		t.Fatalf("ProduceOPESCausalJobsV0: %v", err)
	}
	return result
}

type opesCausalRecordSourceForStackTestV0 struct {
	records []orquestaopesdirector.OPESCausalArtifactRecordV0
}

func (source opesCausalRecordSourceForStackTestV0) ListOPESCausalArtifactRecordsV0(
	_ context.Context,
	filter orquestaopesdirector.OPESCausalArtifactRecordFilterV0,
) ([]orquestaopesdirector.OPESCausalArtifactRecordV0, error) {
	var out []orquestaopesdirector.OPESCausalArtifactRecordV0
	for _, record := range source.records {
		if filter.DomainRef != "" && record.DomainRef != filter.DomainRef {
			continue
		}
		if filter.CorrelationID != "" && record.CorrelationID != filter.CorrelationID {
			continue
		}
		out = append(out, record)
	}
	return out, nil
}

func opesCausalRecordFromDomainWorkSubmissionForStackTestV0(
	submission orquestadomainwork.DomainWorkArtifactSubmissionV0,
) orquestaopesdirector.OPESCausalArtifactRecordV0 {
	return orquestaopesdirector.OPESCausalArtifactRecordV0{
		IdempotencyKey: submission.IdempotencyKey,
		Status:         orquestadomainwork.DomainWorkStatusAcceptedV0,
		CorrelationID:  submission.CorrelationID,
		DomainRef:      submission.DomainRef,
		JobRef:         submission.JobRef,
		ArtifactRef:    submission.ArtifactRef,
		ArtifactType:   submission.ArtifactType,
		Summary:        submission.Summary,
		PayloadFields:  submission.PayloadFields,
		PayloadRefs:    submission.PayloadRefs,
		ExternalRefs:   submission.ExternalRefs,
		CompleteJob:    submission.CompleteJob,
		ReceiptRef:     "receipt-ref-" + submission.ArtifactRef,
		EvidenceRefs:   submission.EvidenceRefs,
	}
}

func opesDirectorRequestedWorkKindForStackTestV0(
	t *testing.T,
	requests []orquestadomainwork.DomainWorkJobRequestV0,
	workKind string,
) orquestadomainwork.DomainWorkJobRequestV0 {
	t.Helper()
	request, ok := opesDirectorMaybeRequestedWorkKindForStackTestV0(requests, workKind)
	if !ok {
		t.Fatalf("no se encontro work_kind=%s en %+v", workKind, requests)
	}
	return request
}

func opesDirectorMaybeRequestedWorkKindForStackTestV0(
	requests []orquestadomainwork.DomainWorkJobRequestV0,
	workKind string,
) (orquestadomainwork.DomainWorkJobRequestV0, bool) {
	for _, request := range requests {
		if request.WorkKind == workKind {
			return request, true
		}
	}
	return orquestadomainwork.DomainWorkJobRequestV0{}, false
}

func domainWorkFieldContainsValuesForTestV0(
	fields []orquestadomainwork.DomainWorkFieldV0,
	name string,
	values []string,
) bool {
	for _, want := range values {
		found := false
		for _, field := range fields {
			if field.Name != name {
				continue
			}
			if field.Value == want {
				found = true
				break
			}
			for _, value := range field.Values {
				if value == want {
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
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
