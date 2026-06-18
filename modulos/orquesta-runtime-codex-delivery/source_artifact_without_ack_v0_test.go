package orquestaruntimecodexdelivery

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// Recuperacion de trabajo: el agente materializo su write-set pero no escribio
// agent_ack.json. Con la promocion activada, la entrega se recupera del disco con
// un gate-issue de revision humana en vez de perderse y colgar el run.
func TestCodexDeliveryObservationSourceV0PromueveArtefactoSinAckV0(t *testing.T) {
	spec := codexDeliverySpecForTestV0()
	projectDir := t.TempDir()
	// El write-set existe en disco (artefacto materializado), pero NO hay ACK.
	if err := os.WriteFile(filepath.Join(projectDir, "README.md"), []byte("contenido real del agente"), 0o600); err != nil {
		t.Fatalf("materializar write-set: %v", err)
	}
	ackPath := filepath.Join(projectDir, "agent_ack.json") // no existe

	store := &staticCodexReceiptStoreV0{
		Descriptors: []CodexReceiptDescriptorV0{{
			DescriptorRef:  "receipt-ref-no-ack",
			RunID:          "run-ref-001",
			AgentRef:       spec.RequestID,
			Spec:           spec,
			AckPath:        ackPath,
			ProjectWorkDir: projectDir,
		}},
	}

	observations, err := (CodexDeliveryObservationSourceV0{
		Store:                                 store,
		PromoteMaterializedArtifactWithoutAck: true,
	}).BuildAgentDeliveryObservationsV0(context.Background(), codexDeliveryRequestForTestV0(spec, nil))
	if err != nil {
		t.Fatalf("BuildAgentDeliveryObservationsV0: %v", err)
	}
	if len(observations) != 1 {
		t.Fatalf("observations=%d want 1 (entrega promovida)", len(observations))
	}
	got := observations[0]
	if got.DeliveryRef != spec.AgentPacket.DeliveryRefs.AckRef ||
		got.AgentRef != spec.RequestID ||
		got.TaskID != spec.AgentPacket.Task.TaskRef {
		t.Fatalf("observacion promovida no correlada: %+v", got)
	}
	if !stringInCodexDeliverySetV0(got.EvidenceRefs, "gate-issue:artifact_without_ack_requires_review") {
		t.Fatalf("falta gate-issue de revision: %+v", got.EvidenceRefs)
	}
	if observationLeaksCodexDeliveryPathV0(got, ackPath) {
		t.Fatalf("observacion filtra path: %+v", got)
	}
}

// Sin la promocion (defecto), un ACK ausente sigue siendo pending: no se promueve.
func TestCodexDeliveryObservationSourceV0SinPromocionDescartaArtefactoSinAckV0(t *testing.T) {
	spec := codexDeliverySpecForTestV0()
	projectDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectDir, "README.md"), []byte("contenido"), 0o600); err != nil {
		t.Fatalf("materializar: %v", err)
	}
	store := &staticCodexReceiptStoreV0{
		Descriptors: []CodexReceiptDescriptorV0{{
			DescriptorRef:  "receipt-ref-no-ack-off",
			RunID:          "run-ref-001",
			AgentRef:       spec.RequestID,
			Spec:           spec,
			AckPath:        filepath.Join(projectDir, "agent_ack.json"),
			ProjectWorkDir: projectDir,
		}},
	}

	observations, err := (CodexDeliveryObservationSourceV0{Store: store}).
		BuildAgentDeliveryObservationsV0(context.Background(), codexDeliveryRequestForTestV0(spec, nil))
	if err != nil {
		t.Fatalf("BuildAgentDeliveryObservationsV0: %v", err)
	}
	if len(observations) != 0 {
		t.Fatalf("observations=%d want 0 (sin promocion, ACK ausente queda pending)", len(observations))
	}
}

// Sin artefacto materializado, aunque la promocion este activa, no se inventa
// entrega: el ACK ausente sigue pending.
func TestCodexDeliveryObservationSourceV0PromocionNoInventaSinArtefactoV0(t *testing.T) {
	spec := codexDeliverySpecForTestV0()
	projectDir := t.TempDir() // write-set NO materializado
	store := &staticCodexReceiptStoreV0{
		Descriptors: []CodexReceiptDescriptorV0{{
			DescriptorRef:  "receipt-ref-no-ack-empty",
			RunID:          "run-ref-001",
			AgentRef:       spec.RequestID,
			Spec:           spec,
			AckPath:        filepath.Join(projectDir, "agent_ack.json"),
			ProjectWorkDir: projectDir,
		}},
	}

	observations, err := (CodexDeliveryObservationSourceV0{
		Store:                                 store,
		PromoteMaterializedArtifactWithoutAck: true,
	}).BuildAgentDeliveryObservationsV0(context.Background(), codexDeliveryRequestForTestV0(spec, nil))
	if err != nil {
		t.Fatalf("BuildAgentDeliveryObservationsV0: %v", err)
	}
	if len(observations) != 0 {
		t.Fatalf("observations=%d want 0 (sin artefacto no se promueve)", len(observations))
	}
}
