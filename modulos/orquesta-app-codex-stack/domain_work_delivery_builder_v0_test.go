package orquestaappcodexstack

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
)

func TestDefaultDomainWorkArtifactSubmissionBuilderV0NoFiltraPathsComoPayloadRefs(t *testing.T) {
	projectDir := t.TempDir()
	bodyPath := filepath.Join(projectDir, "external", "opes", "draft_content_block")
	if err := os.MkdirAll(filepath.Dir(bodyPath), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(bodyPath, []byte("# Bloque OPES\n\nContenido."), 0o600); err != nil {
		t.Fatalf("write body: %v", err)
	}

	submission, ok, err := (defaultDomainWorkArtifactSubmissionBuilderV0{}).
		BuildDomainWorkArtifactSubmissionV0(
			context.Background(),
			DomainWorkArtifactSubmissionBuildInputV0{
				Run: orquestacoreworkflow.OrchestrationRunV0{RunID: "run-ref-001"},
				Task: orquestacoreworkflow.WorkflowTaskV0{
					TaskID: "task-ref-001",
					Title:  "Redactar bloque documental OPES",
				},
				Record: orquestaappchange.AppChangeRecordV0{
					Request: orquestaappchange.AppChangeRequestV0{
						CorrelationID: "corr-ref-001",
						ChangeRef:     "change-ref-001",
						ExternalWork: &orquestaappchange.AppChangeExternalWorkV0{
							ProjectRef: "opes",
							JobRef:     "job-ref-001",
							WorkKind:   "draft_content_block",
							InputFields: []orquestadomainwork.DomainWorkFieldV0{
								{Name: "topic_id", Value: "topic-ref-001"},
							},
						},
					},
				},
				Descriptor: orquestaruntimecodexdelivery.CodexReceiptDescriptorV0{
					DescriptorRef:  "descriptor-ref-001",
					ProjectWorkDir: projectDir,
					Spec: orquestaruntime.ExternalAgentLaunchSpecV0{
						AgentPacket: orquestaruntime.AgentStartPacketV0{
							DeliveryRefs: orquestaruntime.AgentStartDeliveryRefsV0{
								MailboxRef:   "mailbox-ref-001",
								ReadinessRef: "readiness-ref-001",
							},
						},
					},
				},
				Ack: orquestaruntimecodex.CodexAgentAckV0{
					Files: []string{"external/opes/draft_content_block"},
				},
				Observation: orquestacionnucleoapp.AgentDeliveryObservationV0{
					DeliveryRef:  "ack-ref-001",
					AgentRef:     "agent-ref-001",
					Summary:      "Entrega validada.",
					EvidenceRefs: []string{"ack-ref-001"},
				},
			},
		)

	if err != nil || !ok {
		t.Fatalf("BuildDomainWorkArtifactSubmissionV0 ok=%v err=%v", ok, err)
	}
	if len(submission.PayloadRefs) != 0 {
		t.Fatalf("payload_refs no debe filtrar rutas internas: %+v", submission.PayloadRefs)
	}
	if !domainWorkFieldValueForTestV0(submission.PayloadFields, "body", "# Bloque OPES\n\nContenido.") {
		t.Fatalf("payload_fields=%+v", submission.PayloadFields)
	}
	if issues := orquestadomainwork.ValidateDomainWorkArtifactSubmissionV0(submission); len(issues) != 0 {
		t.Fatalf("submission invalida: %+v", issues)
	}
}

func TestDefaultDomainWorkArtifactSubmissionBuilderV0EntregaTopicSummaryOPES(t *testing.T) {
	projectDir := t.TempDir()
	bodyPath := filepath.Join(projectDir, "external", "opes", "summarize_topic")
	if err := os.MkdirAll(filepath.Dir(bodyPath), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	payload := `{
		"markdown":"Resumen contrastado del tema.",
		"coverage":["prevencion","factores de riesgo"],
		"key_concepts":["prevencion primaria","epidemiologia"],
		"cross_topic_links":["tema-88"],
		"source_refs":["fuente-ref-001"]
	}`
	if err := os.WriteFile(bodyPath, []byte(payload), 0o600); err != nil {
		t.Fatalf("write body: %v", err)
	}

	submission, ok, err := (defaultDomainWorkArtifactSubmissionBuilderV0{}).
		BuildDomainWorkArtifactSubmissionV0(
			context.Background(),
			DomainWorkArtifactSubmissionBuildInputV0{
				Run:  orquestacoreworkflow.OrchestrationRunV0{RunID: "run-ref-summary-001"},
				Task: orquestacoreworkflow.WorkflowTaskV0{TaskID: "task-ref-summary-001", Title: "Resumir tema OPES"},
				Record: orquestaappchange.AppChangeRecordV0{
					Request: orquestaappchange.AppChangeRequestV0{
						CorrelationID: "corr-ref-summary-001",
						ChangeRef:     "change-ref-summary-001",
						ExternalWork: &orquestaappchange.AppChangeExternalWorkV0{
							ProjectRef: "opes",
							JobRef:     "job-ref-summary-001",
							WorkKind:   "summarize_topic",
							InputFields: []orquestadomainwork.DomainWorkFieldV0{
								{Name: "topic_id", Value: "topic-ref-001"},
							},
						},
					},
				},
				Descriptor: orquestaruntimecodexdelivery.CodexReceiptDescriptorV0{
					DescriptorRef:  "descriptor-ref-summary-001",
					ProjectWorkDir: projectDir,
				},
				Ack: orquestaruntimecodex.CodexAgentAckV0{
					Files: []string{"external/opes/summarize_topic"},
				},
				Observation: orquestacionnucleoapp.AgentDeliveryObservationV0{
					DeliveryRef:  "ack-ref-summary-001",
					AgentRef:     "agent-ref-summary-001",
					Summary:      "Resumen validado.",
					EvidenceRefs: []string{"ack-ref-summary-001"},
				},
			},
		)

	if err != nil || !ok {
		t.Fatalf("BuildDomainWorkArtifactSubmissionV0 ok=%v err=%v", ok, err)
	}
	if submission.ArtifactType != "topic_summary" ||
		!domainWorkFieldValueForTestV0(submission.PayloadFields, "markdown", "Resumen contrastado del tema.") ||
		!domainWorkFieldValuesForTestV0(submission.PayloadFields, "coverage", []string{"prevencion", "factores de riesgo"}) ||
		domainWorkFieldValueForTestV0(submission.PayloadFields, "body", "Resumen contrastado del tema.") {
		t.Fatalf("submission=%+v", submission)
	}
	if issues := orquestadomainwork.ValidateDomainWorkArtifactSubmissionV0(submission); len(issues) != 0 {
		t.Fatalf("submission invalida: %+v", issues)
	}
}

func TestDefaultDomainWorkArtifactSubmissionBuilderV0EntregaExpansionPackageOPES(t *testing.T) {
	projectDir := t.TempDir()
	bodyPath := filepath.Join(projectDir, "external", "opes", "expand_topic_from_summary")
	if err := os.MkdirAll(filepath.Dir(bodyPath), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	payload := `{
		"topic_id":"topic-ref-001",
		"language_code":"es",
		"chapters":[
			{"title":"Capitulo 1","order":1,"blocks":[
				{"block_type":"doctrine","title":"Bloque 1","markdown":"Contenido ampliado.","source_refs":["fuente-ref-001"]}
			]}
		]
	}`
	if err := os.WriteFile(bodyPath, []byte(payload), 0o600); err != nil {
		t.Fatalf("write body: %v", err)
	}

	submission, ok, err := (defaultDomainWorkArtifactSubmissionBuilderV0{}).
		BuildDomainWorkArtifactSubmissionV0(
			context.Background(),
			DomainWorkArtifactSubmissionBuildInputV0{
				Run:  orquestacoreworkflow.OrchestrationRunV0{RunID: "run-ref-expansion-001"},
				Task: orquestacoreworkflow.WorkflowTaskV0{TaskID: "task-ref-expansion-001", Title: "Ampliar tema OPES"},
				Record: orquestaappchange.AppChangeRecordV0{
					Request: orquestaappchange.AppChangeRequestV0{
						CorrelationID: "corr-ref-expansion-001",
						ChangeRef:     "change-ref-expansion-001",
						ExternalWork: &orquestaappchange.AppChangeExternalWorkV0{
							ProjectRef: "opes",
							JobRef:     "job-ref-expansion-001",
							WorkKind:   "expand_topic_from_summary",
						},
					},
				},
				Descriptor: orquestaruntimecodexdelivery.CodexReceiptDescriptorV0{
					DescriptorRef:  "descriptor-ref-expansion-001",
					ProjectWorkDir: projectDir,
				},
				Ack: orquestaruntimecodex.CodexAgentAckV0{
					Files: []string{"external/opes/expand_topic_from_summary"},
				},
				Observation: orquestacionnucleoapp.AgentDeliveryObservationV0{
					DeliveryRef:  "ack-ref-expansion-001",
					AgentRef:     "agent-ref-expansion-001",
					Summary:      "Expansion validada.",
					EvidenceRefs: []string{"ack-ref-expansion-001"},
				},
			},
		)

	if err != nil || !ok {
		t.Fatalf("BuildDomainWorkArtifactSubmissionV0 ok=%v err=%v", ok, err)
	}
	if submission.ArtifactType != "topic_expansion_package" ||
		!domainWorkFieldValueForTestV0(submission.PayloadFields, "topic_id", "topic-ref-001") ||
		!domainWorkFieldHasJSONForTestV0(submission.PayloadFields, "chapters") {
		t.Fatalf("submission=%+v", submission)
	}
	if issues := orquestadomainwork.ValidateDomainWorkArtifactSubmissionV0(submission); len(issues) != 0 {
		t.Fatalf("submission invalida: %+v", issues)
	}
}

func domainWorkFieldValuesForTestV0(
	fields []orquestadomainwork.DomainWorkFieldV0,
	name string,
	values []string,
) bool {
	for _, field := range fields {
		if field.Name != name || len(field.Values) != len(values) {
			continue
		}
		matches := true
		for index := range values {
			if field.Values[index] != values[index] {
				matches = false
				break
			}
		}
		if matches {
			return true
		}
	}
	return false
}

func domainWorkFieldHasJSONForTestV0(
	fields []orquestadomainwork.DomainWorkFieldV0,
	name string,
) bool {
	for _, field := range fields {
		if field.Name == name && len(field.ValueJSON) > 0 && json.Valid(field.ValueJSON) {
			return true
		}
	}
	return false
}

func TestDefaultDomainWorkArtifactSubmissionBuilderV0EntregaVisualAssetOPES(t *testing.T) {
	projectDir := t.TempDir()
	bodyPath := filepath.Join(projectDir, "external", "opes", "generate_visual_asset")
	if err := os.MkdirAll(filepath.Dir(bodyPath), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	svg := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 800 420" role="img" aria-label="Red en estrella"></svg>`
	if err := os.WriteFile(bodyPath, []byte(svg), 0o600); err != nil {
		t.Fatalf("write body: %v", err)
	}

	submission, ok, err := (defaultDomainWorkArtifactSubmissionBuilderV0{}).
		BuildDomainWorkArtifactSubmissionV0(
			context.Background(),
			DomainWorkArtifactSubmissionBuildInputV0{
				Run:  orquestacoreworkflow.OrchestrationRunV0{RunID: "run-ref-visual-001"},
				Task: orquestacoreworkflow.WorkflowTaskV0{TaskID: "task-ref-visual-001", Title: "Generar recurso visual OPES"},
				Record: orquestaappchange.AppChangeRecordV0{
					Request: orquestaappchange.AppChangeRequestV0{
						CorrelationID: "corr-ref-visual-001",
						ChangeRef:     "change-ref-visual-001",
						ExternalWork: &orquestaappchange.AppChangeExternalWorkV0{
							ProjectRef: "opes",
							JobRef:     "job-ref-visual-001",
							WorkKind:   "generate_visual_asset",
							InputFields: []orquestadomainwork.DomainWorkFieldV0{
								{Name: "topic_id", Value: "topic-ref-001"},
								{Name: "chapter_id", Value: "chapter-ref-001"},
								{Name: "asset_type", Value: "vignette"},
								{Name: "format", Value: "svg"},
								{Name: "title", Value: "Red en estrella"},
								{Name: "caption", Value: "Topologia con nodo central."},
								{Name: "alt_text", Value: "Switch central conectado a equipos cliente."},
								{Name: "placement", Value: "after_block"},
								{Name: "language_code", Value: "es"},
							},
						},
					},
				},
				Descriptor: orquestaruntimecodexdelivery.CodexReceiptDescriptorV0{
					DescriptorRef:  "descriptor-ref-visual-001",
					ProjectWorkDir: projectDir,
				},
				Ack: orquestaruntimecodex.CodexAgentAckV0{
					Files: []string{"external/opes/generate_visual_asset"},
				},
				Observation: orquestacionnucleoapp.AgentDeliveryObservationV0{
					DeliveryRef:  "ack-ref-visual-001",
					AgentRef:     "agent-ref-visual-001",
					Summary:      "Visual validado.",
					EvidenceRefs: []string{"ack-ref-visual-001"},
				},
			},
		)

	if err != nil || !ok {
		t.Fatalf("BuildDomainWorkArtifactSubmissionV0 ok=%v err=%v", ok, err)
	}
	if submission.ArtifactType != "visual_asset" ||
		!domainWorkFieldValueForTestV0(submission.PayloadFields, "content_type", "image/svg+xml") ||
		!domainWorkFieldValueForTestV0(submission.PayloadFields, "body", svg) ||
		!domainWorkFieldValueForTestV0(submission.PayloadFields, "title", "Red en estrella") ||
		!domainWorkFieldValueForTestV0(submission.PayloadFields, "caption", "Topologia con nodo central.") ||
		!domainWorkFieldValueForTestV0(submission.PayloadFields, "alt_text", "Switch central conectado a equipos cliente.") {
		t.Fatalf("submission=%+v", submission)
	}
	if issues := orquestadomainwork.ValidateDomainWorkArtifactSubmissionV0(submission); len(issues) != 0 {
		t.Fatalf("submission invalida: %+v", issues)
	}
}
