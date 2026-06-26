package orquestaappcodexstack

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
)

func TestDefaultDomainWorkArtifactSubmissionBuilderV0EntregaDocumentPlanOPES(t *testing.T) {
	projectDir := t.TempDir()
	bodyPath := filepath.Join(projectDir, "external", "opes", "plan_tema")
	if err := os.MkdirAll(filepath.Dir(bodyPath), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(bodyPath, []byte(validDocumentPlanPayloadForTestV0()), 0o600); err != nil {
		t.Fatalf("write body: %v", err)
	}

	submission, ok, err := (defaultDomainWorkArtifactSubmissionBuilderV0{}).
		BuildDomainWorkArtifactSubmissionV0(
			context.Background(),
			DomainWorkArtifactSubmissionBuildInputV0{
				Run:  orquestacoreworkflow.OrchestrationRunV0{RunID: "run-ref-plan-001"},
				Task: orquestacoreworkflow.WorkflowTaskV0{TaskID: "task-ref-plan-001", Title: "Planificar tema OPES"},
				Record: orquestaappchange.AppChangeRecordV0{
					Request: orquestaappchange.AppChangeRequestV0{
						CorrelationID: "corr-ref-plan-001",
						ChangeRef:     "change-ref-plan-001",
						ExternalWork: &orquestaappchange.AppChangeExternalWorkV0{
							ProjectRef: "opes",
							JobRef:     "job-ref-plan-001",
							WorkKind:   "plan_tema",
						},
					},
				},
				Descriptor: orquestaruntimecodexdelivery.CodexReceiptDescriptorV0{
					DescriptorRef:  "descriptor-ref-plan-001",
					ProjectWorkDir: projectDir,
				},
				Ack: orquestaruntimecodex.CodexAgentAckV0{
					Files: []string{"external/opes/plan_tema"},
				},
				Observation: orquestacionnucleoapp.AgentDeliveryObservationV0{
					DeliveryRef:  "ack-ref-plan-001",
					AgentRef:     "agent-ref-plan-001",
					Summary:      "Plan validado.",
					EvidenceRefs: []string{"ack-ref-plan-001"},
				},
			},
		)

	if err != nil || !ok {
		t.Fatalf("BuildDomainWorkArtifactSubmissionV0 ok=%v err=%v", ok, err)
	}
	if submission.ArtifactType != orquestadomainwork.DomainDocumentPlanArtifactTypeV0 ||
		!domainWorkFieldValueForTestV0(submission.PayloadFields, "content_type", "application/json") ||
		!domainWorkFieldValueForTestV0(submission.PayloadFields, "plan_ref", "plan_ref_080") ||
		!domainWorkFieldHasJSONForTestV0(submission.PayloadFields, "sections") ||
		!domainWorkFieldHasJSONForTestV0(submission.PayloadFields, "deliverables") {
		t.Fatalf("submission=%+v", submission)
	}
	if issues := orquestadomainwork.ValidateDomainWorkArtifactSubmissionV0(submission); len(issues) != 0 {
		t.Fatalf("submission invalida: %+v", issues)
	}
}

func TestDefaultDomainWorkArtifactSubmissionBuilderV0MarcaValidacionVaciaComoInvalida(t *testing.T) {
	projectDir := t.TempDir()
	bodyPath := filepath.Join(projectDir, "external", "opes", "validacion_tema")
	if err := os.MkdirAll(filepath.Dir(bodyPath), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	body := `{
		"artifact_type":"block_revision",
		"payload_json":{
			"status":"pass",
			"files_scanned":0,
			"finding_count":0,
			"summary":"Validador sin notas de autor."
		}
	}`
	if err := os.WriteFile(bodyPath, []byte(body), 0o600); err != nil {
		t.Fatalf("write body: %v", err)
	}

	submission, ok, err := (defaultDomainWorkArtifactSubmissionBuilderV0{}).
		BuildDomainWorkArtifactSubmissionV0(
			context.Background(),
			DomainWorkArtifactSubmissionBuildInputV0{
				Run:  orquestacoreworkflow.OrchestrationRunV0{RunID: "run-ref-validation-empty-scan-001"},
				Task: orquestacoreworkflow.WorkflowTaskV0{TaskID: "task-ref-validation-empty-scan-001", Title: "Validar texto publico"},
				Record: orquestaappchange.AppChangeRecordV0{
					Request: orquestaappchange.AppChangeRequestV0{
						CorrelationID: "corr-ref-validation-empty-scan-001",
						ChangeRef:     "change-ref-validation-empty-scan-001",
						ExternalWork: &orquestaappchange.AppChangeExternalWorkV0{
							ProjectRef: "opes",
							JobRef:     "job-ref-validation-empty-scan-001",
							WorkKind:   "validate_topic",
						},
					},
				},
				Descriptor: orquestaruntimecodexdelivery.CodexReceiptDescriptorV0{
					DescriptorRef:  "descriptor-ref-validation-empty-scan-001",
					ProjectWorkDir: projectDir,
				},
				Ack: orquestaruntimecodex.CodexAgentAckV0{
					Files: []string{"external/opes/validacion_tema"},
				},
				Observation: orquestacionnucleoapp.AgentDeliveryObservationV0{
					DeliveryRef:  "ack-ref-validation-empty-scan-001",
					AgentRef:     "agent-ref-validation-empty-scan-001",
					Summary:      "Validador estructurado.",
					EvidenceRefs: []string{"ack-ref-validation-empty-scan-001"},
				},
			},
		)

	if err != nil || !ok {
		t.Fatalf("BuildDomainWorkArtifactSubmissionV0 ok=%v err=%v", ok, err)
	}
	if submission.ArtifactType != orquestadomainwork.DomainWorkArtifactTypeBlockRevisionV0 ||
		submission.CompleteJob ||
		!domainWorkFieldValueForTestV0(submission.PayloadFields, "status", "invalid") ||
		!domainWorkFieldValueForTestV0(submission.PayloadFields, "validation_status", "invalid") ||
		!domainWorkFieldValuesForTestV0(submission.PayloadFields, "validation_issue_refs", []string{"invalid_validation_empty_scan"}) {
		t.Fatalf("submission=%+v", submission)
	}
	if issues := orquestadomainwork.ValidateDomainWorkArtifactSubmissionV0(submission); len(issues) != 0 {
		t.Fatalf("submission invalida: %+v", issues)
	}
}

func validDocumentPlanPayloadForTestV0() string {
	return `{
		"schema_version":"domain_document_plan.v0",
		"plan_ref":"plan_ref_080",
		"domain_ref":"opes",
		"work_kind":"plan_tema",
		"document_kind":"tema_oposicion",
		"scope_ref":"topic_ref_080",
		"language_code":"es",
		"title":"Evaluacion diagnostica en psicologia",
		"objective":"Planificar un tema completo de oposicion con enfoque pedagogico.",
		"estimated_pages_min":45,
		"estimated_pages_max":50,
		"sections":[
			{
				"section_ref":"section_intro",
				"order":1,
				"title":"Marco conceptual",
				"objective":"Delimitar conceptos, autores y teorias.",
				"work_kind":"draft_content_block",
				"target_words_min":1200,
				"acceptance_criteria":["incluir autores relevantes"]
			}
		],
		"visuals":[
			{
				"visual_ref":"visual_diagnostico",
				"visual_type":"vineta_educativa",
				"placement_ref":"section_intro",
				"objective":"Apoyar la comprension del proceso diagnostico.",
				"work_kind":"generate_visual_asset"
			}
		],
		"review_steps":[
			{
				"review_ref":"review_pedagogical",
				"order":1,
				"work_kind":"review_pedagogical",
				"objective":"Verificar claridad pedagogica."
			}
		],
		"deliverables":[
			{
				"deliverable_ref":"deliverable_tema_grande",
				"artifact_type":"topic_expansion_package",
				"title":"Tema grande",
				"required":true
			}
		],
		"quality_criteria":["lectura facil","sin placeholders"]
	}`
}
