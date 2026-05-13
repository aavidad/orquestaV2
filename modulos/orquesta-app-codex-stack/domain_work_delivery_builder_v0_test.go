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
