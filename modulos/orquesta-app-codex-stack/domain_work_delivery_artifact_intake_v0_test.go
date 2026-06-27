package orquestaappcodexstack

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
)

func TestDomainWorkDeliveryArtifactIntakeV0AceptaArtefactoGrandeSinRailDeTamano(t *testing.T) {
	projectDir := t.TempDir()
	writeDomainWorkArtifactIntakeTestFileV0(
		t,
		projectDir,
		"external/opes/generate_question_bank",
		[]byte(`{"artifact_type":"question_bank","payload_json":{"body":"`+strings.Repeat("x", 1024*1024)+`"}}`),
	)

	submission, ok, err := buildDomainWorkArtifactIntakeSubmissionForTestV0(
		projectDir,
		"external/opes/generate_question_bank",
		"generate_question_bank",
	)
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	if submission.ArtifactType != orquestadomainwork.DomainWorkArtifactTypeQuestionBankV0 {
		t.Fatalf("submission=%+v", submission)
	}
}

func TestDomainWorkDeliveryArtifactIntakeV0NoBloqueaBinarioLoConservaComoRef(t *testing.T) {
	projectDir := t.TempDir()
	writeDomainWorkArtifactIntakeTestFileV0(
		t,
		projectDir,
		"external/opes/generate_visual_asset",
		[]byte{0xff, 0x00, 0x01, 0x02},
	)

	submission, ok, err := buildDomainWorkArtifactIntakeSubmissionForTestV0(
		projectDir,
		"external/opes/generate_visual_asset",
		"generate_visual_asset",
	)
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	if !domainWorkFieldValueForTestV0(submission.PayloadFields, "file_ref", "external/opes/generate_visual_asset") {
		t.Fatalf("payload_fields=%+v", submission.PayloadFields)
	}
}

func TestDomainWorkDeliveryArtifactIntakeV0SaneaCamposSensiblesSinBloquear(t *testing.T) {
	projectDir := t.TempDir()
	writeDomainWorkArtifactIntakeTestFileV0(
		t,
		projectDir,
		"external/opes/draft_content_block",
		[]byte(`{"title":"Bloque","body":"Contenido publico.","access_token":"sk-secret-local"}`),
	)

	submission, ok, err := buildDomainWorkArtifactIntakeSubmissionForTestV0(
		projectDir,
		"external/opes/draft_content_block",
		"draft_content_block",
	)
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	if submission.ArtifactType != orquestadomainwork.DomainWorkArtifactTypeContentBlockV0 ||
		!domainWorkFieldValueForTestV0(submission.PayloadFields, "access_token", "<redacted-sensitive-field>") ||
		!domainWorkFieldValueForTestV0(submission.PayloadFields, "body", "Contenido publico.") {
		t.Fatalf("submission=%+v", submission)
	}
}

func TestDomainWorkDeliveryArtifactIntakeV0SaneaTokenSKLargoSinBloquearTaskRef(t *testing.T) {
	projectDir := t.TempDir()
	writeDomainWorkArtifactIntakeTestFileV0(
		t,
		projectDir,
		"reviews/review_codex_orquesta_real.md",
		[]byte("Referencia de tarea: task-ref-app-change-appchange-001\nToken literal: sk-abcdefghijklmnopqrstuvwxyz0123456789\n\nRevision publica."),
	)

	submission, ok, err := buildDomainWorkArtifactIntakeSubmissionForTestV0(
		projectDir,
		"reviews/review_codex_orquesta_real.md",
		"review_codex",
	)
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	if submission.ArtifactType != "agent_review_report" ||
		!domainWorkFieldValueForTestV0(submission.PayloadFields, "body", "Referencia de tarea: task-ref-app-change-appchange-001\nToken literal: <secret-token-redacted>\n\nRevision publica.") {
		t.Fatalf("submission=%+v", submission)
	}
}

func TestDomainWorkDeliveryArtifactIntakeV0AceptaJSONEstructuradoSeguro(t *testing.T) {
	projectDir := t.TempDir()
	writeDomainWorkArtifactIntakeTestFileV0(
		t,
		projectDir,
		"external/opes/summarize_topic",
		[]byte(`{"markdown":"Resumen publico.","source_refs":["fuente-ref-001"]}`),
	)

	submission, ok, err := buildDomainWorkArtifactIntakeSubmissionForTestV0(
		projectDir,
		"external/opes/summarize_topic",
		"summarize_topic",
	)
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	if !domainWorkFieldValueForTestV0(submission.PayloadFields, "markdown", "Resumen publico.") ||
		!domainWorkFieldValuesForTestV0(submission.PayloadFields, "source_refs", []string{"fuente-ref-001"}) {
		t.Fatalf("payload_fields=%+v", submission.PayloadFields)
	}
}

func TestDomainWorkDeliveryArtifactIntakeV0SeleccionaFicheroPorTipoArtefacto(t *testing.T) {
	projectDir := t.TempDir()
	writeDomainWorkArtifactIntakeTestFileV0(
		t,
		projectDir,
		"external/opes/notas_auxiliares.md",
		[]byte("Contenido auxiliar que no debe ser el artefacto principal."),
	)
	writeDomainWorkArtifactIntakeTestFileV0(
		t,
		projectDir,
		"external/opes/draft_content_block",
		[]byte(`{"title":"Bloque elegido","body":"Contenido elegido."}`),
	)

	intake, err := readDomainWorkDeliveryArtifactV0(
		orquestaruntimecodexdelivery.CodexReceiptDescriptorV0{
			DescriptorRef:  "descriptor-ref-intake-multiple-001",
			ProjectWorkDir: projectDir,
		},
		orquestaruntimecodex.CodexAgentAckV0{
			Files: []string{
				"external/opes/notas_auxiliares.md",
				"external/opes/draft_content_block",
			},
		},
		orquestadomainwork.DomainWorkArtifactTypeContentBlockV0,
	)

	if err != nil {
		t.Fatalf("readDomainWorkDeliveryArtifactV0: %v", err)
	}
	if intake.FileRef != "external/opes/draft_content_block" ||
		!strings.Contains(intake.Body, "Contenido elegido.") {
		t.Fatalf("intake=%+v", intake)
	}
}

func TestDomainWorkDeliveryArtifactIntakeV0ExigeTipoSiHayVariosFicheros(t *testing.T) {
	projectDir := t.TempDir()
	writeDomainWorkArtifactIntakeTestFileV0(
		t,
		projectDir,
		"external/opes/notas_auxiliares.md",
		[]byte("Contenido auxiliar que no debe ser el artefacto principal."),
	)
	writeDomainWorkArtifactIntakeTestFileV0(
		t,
		projectDir,
		"external/opes/otra_nota.md",
		[]byte("Segunda nota auxiliar."),
	)

	_, err := readDomainWorkDeliveryArtifactV0(
		orquestaruntimecodexdelivery.CodexReceiptDescriptorV0{
			DescriptorRef:  "descriptor-ref-intake-multiple-no-match-001",
			ProjectWorkDir: projectDir,
		},
		orquestaruntimecodex.CodexAgentAckV0{
			Files: []string{
				"external/opes/notas_auxiliares.md",
				"external/opes/otra_nota.md",
			},
		},
		orquestadomainwork.DomainWorkArtifactTypeContentBlockV0,
	)

	if err == nil || !strings.Contains(err.Error(), "artifact_type_match_required_for_multiple_files") {
		t.Fatalf("err=%v", err)
	}
}

func writeDomainWorkArtifactIntakeTestFileV0(
	t *testing.T,
	projectDir string,
	rel string,
	content []byte,
) {
	t.Helper()
	path := filepath.Join(projectDir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func buildDomainWorkArtifactIntakeSubmissionForTestV0(
	projectDir string,
	rel string,
	workKind string,
) (orquestadomainwork.DomainWorkArtifactSubmissionV0, bool, error) {
	return (defaultDomainWorkArtifactSubmissionBuilderV0{}).BuildDomainWorkArtifactSubmissionV0(
		context.Background(),
		DomainWorkArtifactSubmissionBuildInputV0{
			Run:  orquestacoreworkflow.OrchestrationRunV0{RunID: "run-ref-intake-001"},
			Task: orquestacoreworkflow.WorkflowTaskV0{TaskID: "task-ref-intake-001", Title: "Entrega dominio"},
			Record: orquestaappchange.AppChangeRecordV0{
				Request: orquestaappchange.AppChangeRequestV0{
					CorrelationID: "corr-ref-intake-001",
					ExternalWork: &orquestaappchange.AppChangeExternalWorkV0{
						ProjectRef: "opes",
						JobRef:     "job-ref-intake-001",
						WorkKind:   workKind,
					},
				},
			},
			Descriptor: orquestaruntimecodexdelivery.CodexReceiptDescriptorV0{
				DescriptorRef:  "descriptor-ref-intake-001",
				ProjectWorkDir: projectDir,
			},
			Ack: orquestaruntimecodex.CodexAgentAckV0{Files: []string{rel}},
			Observation: orquestacionnucleoapp.AgentDeliveryObservationV0{
				DeliveryRef:  "ack-ref-intake-001",
				AgentRef:     "agent-ref-intake-001",
				Summary:      "Entrega domain_work.",
				EvidenceRefs: []string{"ack-ref-intake-001"},
			},
		},
	)
}
