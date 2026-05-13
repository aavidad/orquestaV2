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
