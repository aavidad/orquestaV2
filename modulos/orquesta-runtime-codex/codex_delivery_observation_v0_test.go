package orquestaruntimecodex

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func TestReadCodexDeliveryObservationFileV0LeeACKYConstruyeObservacion(t *testing.T) {
	spec := codexNeutralSpecForDeliveryObservationTestV0()
	ack := codexNeutralAckForDeliveryObservationTestV0(spec)
	data, err := json.Marshal(ack)
	if err != nil {
		t.Fatalf("marshal ack: %v", err)
	}
	path := filepath.Join(t.TempDir(), CodexAgentAckFileNameV0)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write ack: %v", err)
	}

	observation, issues := ReadCodexDeliveryObservationFileV0(path, spec)
	if len(issues) > 0 {
		t.Fatalf("issues=%+v", issues)
	}
	if observation.DeliveryRef != spec.AgentPacket.DeliveryRefs.AckRef {
		t.Fatalf("observation=%+v", observation)
	}
}

func TestReadCodexDeliveryObservationFileV0AceptaACKMinimoYCorrelacionaConSpec(t *testing.T) {
	spec := codexNeutralSpecForDeliveryObservationTestV0()
	spec.AgentPacket.Task.RequiredTests = nil
	path := filepath.Join(t.TempDir(), CodexAgentAckFileNameV0)
	if err := os.WriteFile(
		path,
		[]byte(`{"schema_version":"codex_agent_ack.v0","status":"completed"}`),
		0o600,
	); err != nil {
		t.Fatalf("write ack: %v", err)
	}

	observation, issues := ReadCodexDeliveryObservationFileV0(path, spec)

	if len(issues) > 0 {
		t.Fatalf("issues=%+v", issues)
	}
	if observation.DeliveryRef != spec.AgentPacket.DeliveryRefs.AckRef ||
		observation.AgentRef != spec.RequestID ||
		observation.TaskID != spec.AgentPacket.Task.TaskRef ||
		observation.PhaseID != spec.AgentPacket.Phase {
		t.Fatalf("observation no correlacionada: %+v spec=%+v", observation, spec)
	}
}

func TestReadCodexDeliveryObservationFileV0MarcaACKAusenteComoNoListo(t *testing.T) {
	spec := codexNeutralSpecForDeliveryObservationTestV0()
	path := filepath.Join(t.TempDir(), CodexAgentAckFileNameV0)

	_, issues := ReadCodexDeliveryObservationFileV0(path, spec)
	requireCodexIssueV0(t, issues, CodexConnectorAckInvalidV0)
	if !issues[0].Retryable {
		t.Fatalf("issue no retryable: %+v", issues[0])
	}
	if !evidenceContainsCodexDeliveryObservationTestV0(issues[0].Evidence, "ack_not_ready") {
		t.Fatalf("evidence=%v", issues[0].Evidence)
	}
}

func TestBuildCodexDeliveryObservationV0AceptaACKCompletoNeutral(t *testing.T) {
	spec := codexNeutralSpecForDeliveryObservationTestV0()
	ack := codexNeutralAckForDeliveryObservationTestV0(spec)

	observation, issues := BuildCodexDeliveryObservationV0(ack, spec)
	if len(issues) > 0 {
		t.Fatalf("issues=%+v", issues)
	}
	if observation.DeliveryRef != spec.AgentPacket.DeliveryRefs.AckRef ||
		observation.AgentRef != spec.RequestID ||
		observation.TaskID != spec.AgentPacket.Task.TaskRef ||
		observation.PhaseID != spec.AgentPacket.Phase {
		t.Fatalf("observation no correlacionada: %+v", observation)
	}
	if len(observation.EvidenceRefs) != 3 {
		t.Fatalf("evidence_refs=%v", observation.EvidenceRefs)
	}
	if codexDeliveryObservationUnsafeForCoreV0(observation) {
		t.Fatalf("observation contiene detalle prohibido: %+v", observation)
	}
}

func TestBuildCodexDeliveryObservationV0ConservaRailPendienteCompacto(t *testing.T) {
	spec := codexNeutralSpecForDeliveryObservationTestV0()
	ack := codexNeutralAckForDeliveryObservationTestV0(spec)
	ack.Notes = EvidenceListV0{
		"rail pendiente: token/provider/home/prompt solo para matriz externa",
		"rail pendiente: access_token=redacted sin valor real",
	}

	observation, issues := BuildCodexDeliveryObservationV0(ack, spec)
	if len(issues) > 0 {
		t.Fatalf("issues=%+v", issues)
	}
	if !evidenceContainsCodexDeliveryObservationTestV0(
		observation.EvidenceRefs,
		CodexAgentAckPendingRailEvidenceRefV0,
	) {
		t.Fatalf("evidence_refs sin rail pendiente compacto: %v", observation.EvidenceRefs)
	}
}

func TestBuildCodexDeliveryObservationV0ConservaRailsPendientesEnCamposACK(t *testing.T) {
	spec := codexNeutralSpecForDeliveryObservationTestV0()
	ack := codexNeutralAckForDeliveryObservationTestV0(spec)
	ack.Files = EvidenceListV0{"README.md", "docs/prompt-policy.md"}
	ack.Tests = EvidenceListV0{"go test ./...", "completion policy redacted"}

	observation, issues := BuildCodexDeliveryObservationV0(ack, spec)
	if len(issues) > 0 {
		t.Fatalf("issues=%+v", issues)
	}
	if !evidenceContainsCodexDeliveryObservationTestV0(
		observation.EvidenceRefs,
		CodexAgentAckPendingRailEvidenceRefV0,
	) {
		t.Fatalf("evidence_refs sin rail pendiente compacto: %v", observation.EvidenceRefs)
	}
}

func evidenceContainsCodexDeliveryObservationTestV0(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func TestBuildCodexDeliveryObservationV0RechazaACKNoCompletado(t *testing.T) {
	spec := codexNeutralSpecForDeliveryObservationTestV0()
	ack := codexNeutralAckForDeliveryObservationTestV0(spec)
	ack.Status = "failed"

	_, issues := BuildCodexDeliveryObservationV0(ack, spec)
	requireCodexIssueV0(t, issues, CodexConnectorAckInvalidV0)
}

func TestBuildCodexDeliveryObservationV0AceptaRefConRailPendiente(t *testing.T) {
	spec := codexNeutralSpecForDeliveryObservationTestV0()
	spec.RequestID = "agent-codex-001"
	spec.AgentPacket.RequestID = spec.RequestID
	ack := codexNeutralAckForDeliveryObservationTestV0(spec)

	observation, issues := BuildCodexDeliveryObservationV0(ack, spec)
	if len(issues) != 0 {
		t.Fatalf("issues=%+v", issues)
	}
	if codexDeliveryObservationUnsafeForCoreV0(observation) {
		t.Fatalf("rail pendiente corto observation=%+v", observation)
	}
	if !codexDeliveryObservationHasPendingRailV0(observation) {
		t.Fatalf("rail pendiente no detecto observation=%+v", observation)
	}
}

func TestBuildCodexDeliveryObservationV0MarcaSoloValorSensibleComoUnsafe(t *testing.T) {
	observation := CodexDeliveryObservationV0{
		DeliveryRef: "ack-ref-001",
		PhaseID:     "programacion",
		TaskID:      "task-ref-001",
		AgentRef:    "agent-ref-001",
		Summary:     "access_token=valor",
	}

	if !codexDeliveryObservationUnsafeForCoreV0(observation) {
		t.Fatalf("detalle sensible no detectado: %+v", observation)
	}
	if codexDeliveryObservationHasPendingRailV0(observation) {
		t.Fatalf("detalle sensible marcado como rail pendiente: %+v", observation)
	}
}

func TestCodexDeliveryObservationV0ConservaValorRedactadoComoRailPendiente(t *testing.T) {
	observation := CodexDeliveryObservationV0{
		DeliveryRef: "ack-ref-001",
		PhaseID:     "programacion",
		TaskID:      "task-ref-001",
		AgentRef:    "agent-ref-001",
		Summary:     "access_token=redacted sin valor real",
	}

	if codexDeliveryObservationUnsafeForCoreV0(observation) {
		t.Fatalf("rail blando redactado marcado unsafe: %+v", observation)
	}
	if !codexDeliveryObservationHasPendingRailV0(observation) {
		t.Fatalf("rail blando no conservado como pendiente: %+v", observation)
	}
}

func codexNeutralSpecForDeliveryObservationTestV0() orquestaruntime.ExternalAgentLaunchSpecV0 {
	spec := codexSpecForTestV0()
	spec.RequestID = "agent-ref-001"
	spec.CorrelationID = "corr-agent-001"
	spec.AgentPacket.RequestID = spec.RequestID
	spec.AgentPacket.CorrelationID = spec.CorrelationID
	spec.AgentPacket.DeliveryRefs.AckRef = "ack-ref-001"
	spec.AgentPacket.DeliveryRefs.MailboxRef = "mailbox-ref-001"
	spec.AgentPacket.DeliveryRefs.ReadinessRef = "readiness-ref-001"
	spec.AgentPacket.DeliveryRefs.CheckpointRef = ""
	return spec
}

func codexNeutralAckForDeliveryObservationTestV0(
	spec orquestaruntime.ExternalAgentLaunchSpecV0,
) CodexAgentAckV0 {
	return CodexAgentAckV0{
		SchemaVersion: CodexAgentAckSchemaVersionV0,
		RequestID:     spec.RequestID,
		CorrelationID: spec.CorrelationID,
		AckRef:        spec.AgentPacket.DeliveryRefs.AckRef,
		TargetModule:  spec.AgentPacket.TargetModule,
		TaskRef:       spec.AgentPacket.Task.TaskRef,
		Status:        codexAgentAckStatusCompletedV0,
		Files:         EvidenceListV0{"README.md"},
		Tests:         EvidenceListV0{"go test ./..."},
	}
}
