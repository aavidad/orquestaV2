package orquestaruntimecodex

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
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

func TestReadCodexDeliveryObservationFileV0AceptaACKLegacySinTestReceipts(t *testing.T) {
	spec := codexNeutralSpecForDeliveryObservationTestV0()
	spec.AgentPacket.Policies = nil
	ack := codexNeutralAckForDeliveryObservationTestV0(spec)
	ack.TestReceipts = nil
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

func TestReadCodexDeliveryObservationFileV0RechazaACKEstrictoSinTestReceipts(t *testing.T) {
	spec := codexNeutralSpecForDeliveryObservationTestV0()
	spec.AgentPacket.Policies = append(spec.AgentPacket.Policies, "ack_terminal_strict")
	ack := codexNeutralAckForDeliveryObservationTestV0(spec)
	ack.TestReceipts = nil
	data, err := json.Marshal(ack)
	if err != nil {
		t.Fatalf("marshal ack: %v", err)
	}
	path := filepath.Join(t.TempDir(), CodexAgentAckFileNameV0)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write ack: %v", err)
	}

	_, issues := ReadCodexDeliveryObservationFileV0(path, spec)

	requireCodexIssueEvidenceV0(t, issues, "missing_required_test_receipt")
}

func TestReadCodexDeliveryObservationFileV0NoAceptaTestReceiptsInvalidosComoLegacy(t *testing.T) {
	spec := codexNeutralSpecForDeliveryObservationTestV0()
	spec.AgentPacket.Policies = append(spec.AgentPacket.Policies, "ack_terminal_strict")
	ack := codexNeutralAckForDeliveryObservationTestV0(spec)
	ack.TestReceipts = []CodexRequiredTestReceiptV0{{
		SchemaVersion: CodexRequiredTestReceiptSchemaVersionV0,
		Command:       "otro test",
		Status:        "passed",
	}}
	data, err := json.Marshal(ack)
	if err != nil {
		t.Fatalf("marshal ack: %v", err)
	}
	path := filepath.Join(t.TempDir(), CodexAgentAckFileNameV0)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write ack: %v", err)
	}

	_, issues := ReadCodexDeliveryObservationFileV0(path, spec)

	requireCodexIssueEvidenceV0(t, issues, "required_test_receipt_mismatch")
}

func TestReadCodexDeliveryObservationFileV0RechazaACKMinimoHidratableV0(t *testing.T) {
	spec := codexNeutralSpecForDeliveryObservationTestV0()
	spec.AgentPacket.Policies = append(spec.AgentPacket.Policies, "ack_terminal_strict")
	spec.AgentPacket.Task.RequiredTests = nil
	path := filepath.Join(t.TempDir(), CodexAgentAckFileNameV0)
	if err := os.WriteFile(
		path,
		[]byte(`{"schema_version":"codex_agent_ack.v0","status":"completed"}`),
		0o600,
	); err != nil {
		t.Fatalf("write ack: %v", err)
	}

	_, issues := ReadCodexDeliveryObservationFileV0(path, spec)

	requireCodexIssueV0(t, issues, CodexConnectorAckInvalidV0)
}

func TestReadCodexDeliveryObservationFileV0AceptaACKMinimoSinPolicyStrictV0(t *testing.T) {
	spec := codexNeutralSpecForDeliveryObservationTestV0()
	spec.AgentPacket.Task.RequiredTests = nil
	spec.AgentPacket.Policies = nil
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
	if observation.DeliveryRef != spec.AgentPacket.DeliveryRefs.AckRef {
		t.Fatalf("observation no correlacionada: %+v", observation)
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
	if len(observation.EvidenceRefs) != 4 {
		t.Fatalf("evidence_refs=%v", observation.EvidenceRefs)
	}
	if codexDeliveryObservationUnsafeForCoreV0(observation) {
		t.Fatalf("observation contiene detalle prohibido: %+v", observation)
	}
}

func TestBuildCodexDeliveryObservationV0ConservaRailPendienteCompacto(t *testing.T) {
	enableCodexRailsModeEnforcedForTestV0(t)
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
	if evidenceContainsCodexDeliveryObservationTestV0(
		observation.EvidenceRefs,
		CodexAgentAckPendingRailEvidenceRefV0,
	) {
		t.Fatalf("evidence_refs no deben incluir rail pendiente: %v", observation.EvidenceRefs)
	}
}

func TestBuildCodexDeliveryObservationV0ConservaRailsPendientesEnCamposACK(t *testing.T) {
	enableCodexRailsModeEnforcedForTestV0(t)
	spec := codexNeutralSpecForDeliveryObservationTestV0()
	ack := codexNeutralAckForDeliveryObservationTestV0(spec)
	ack.Files = EvidenceListV0{"README.md", "docs/prompt-policy.md"}
	ack.Tests = EvidenceListV0{"go test ./...", "completion policy redacted"}

	observation, issues := BuildCodexDeliveryObservationV0(ack, spec)
	if len(issues) > 0 {
		t.Fatalf("issues=%+v", issues)
	}
	if evidenceContainsCodexDeliveryObservationTestV0(
		observation.EvidenceRefs,
		CodexAgentAckPendingRailEvidenceRefV0,
	) {
		t.Fatalf("evidence_refs no deben incluir rail pendiente: %v", observation.EvidenceRefs)
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
	enableCodexRailsModeEnforcedForTestV0(t)
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
	if codexDeliveryObservationHasPendingRailV0(observation) {
		t.Fatalf("rail pendiente no debe detectarse observation=%+v", observation)
	}
}

func TestBuildCodexDeliveryObservationV0MarcaValorSensibleComoUnsafe(t *testing.T) {
	enableCodexRailsModeEnforcedForTestV0(t)
	t.Setenv("ORQUESTA_DETAIL_PROHIBITED_RAILS", "on")
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
	enableCodexRailsModeEnforcedForTestV0(t)
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
	if codexDeliveryObservationHasPendingRailV0(observation) {
		t.Fatalf("rail blando no debe conservarse como pendiente: %+v", observation)
	}
}
