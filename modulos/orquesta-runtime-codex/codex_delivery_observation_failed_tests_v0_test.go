package orquestaruntimecodex

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestReadCodexDeliveryObservationFileV0ConservaEntregaConTestsFallidosComoRailDeRevision(t *testing.T) {
	spec := codexNeutralSpecForDeliveryObservationTestV0()
	ack := codexNeutralAckForDeliveryObservationTestV0(spec)
	ack.TestReceipts[0].Status = "failed"
	ack.TestReceipts[0].ExitCode = codexDeliveryObservationIntPtrForTestV0(1)
	ack.TestReceipts[0].EvidenceRefs = []string{"required-test-receipt-ref-failed"}
	data, err := json.Marshal(ack)
	if err != nil {
		t.Fatalf("marshal ack: %v", err)
	}
	path := filepath.Join(t.TempDir(), CodexAgentAckFileNameV0)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write ack: %v", err)
	}

	observation, issues := ReadCodexDeliveryObservationFileV0(path, spec)

	if len(issues) != 0 {
		t.Fatalf("issues=%+v, want delivery observation with review rail", issues)
	}
	if observation.DeliveryRef != spec.AgentPacket.DeliveryRefs.AckRef {
		t.Fatalf("delivery_ref=%q want %q", observation.DeliveryRef, spec.AgentPacket.DeliveryRefs.AckRef)
	}
	if !codexDeliveryObservationRefsContainV0(observation.EvidenceRefs, "gate-issue:failed_test_evidence") {
		t.Fatalf("evidence_refs=%v, want failed test rail", observation.EvidenceRefs)
	}
	if !codexDeliveryObservationRefsContainV0(observation.EvidenceRefs, "required-test-receipt-ref-failed") {
		t.Fatalf("evidence_refs=%v, want failed test receipt ref", observation.EvidenceRefs)
	}
}

func codexDeliveryObservationRefsContainV0(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
