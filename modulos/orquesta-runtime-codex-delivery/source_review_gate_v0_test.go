package orquestaruntimecodexdelivery

import (
	"context"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
	orquestacionnucleoapp "orquesta/orquestacionnucleoapp"
)

func TestCodexReviewGateObservationSourceV0AceptaACKValido(t *testing.T) {
	spec := codexDeliverySpecForTestV0()
	ack := codexDeliveryAckForTestV0(spec)
	path := writeCodexDeliveryAckForTestV0(t, spec, ack)
	store := &staticCodexReceiptStoreV0{
		Descriptors: []CodexReceiptDescriptorV0{{
			DescriptorRef: "receipt-ref-001",
			RunID:         "run-ref-001",
			AgentRef:      spec.RequestID,
			Spec:          spec,
			AckPath:       path,
		}},
	}

	observations, err := (CodexReviewGateObservationSourceV0{Store: store}).
		BuildReviewGateObservationsV0(context.Background(), codexReviewGateRequestForTestV0(spec, nil))
	if err != nil {
		t.Fatalf("BuildReviewGateObservationsV0: %v", err)
	}
	if len(observations) != 1 {
		t.Fatalf("observations=%d", len(observations))
	}
	got := observations[0]
	if got.Status != orquestacoreworkflow.ReviewResultStatusAcceptedV0 {
		t.Fatalf("status=%q", got.Status)
	}
	if got.AcceptedReviewRef == "" || got.DeliveryRef != ack.AckRef {
		t.Fatalf("observacion aceptada incompleta: %+v", got)
	}
	if got.PhaseID != string(orquestacoreworkflow.OrchestrationPhaseRevisionV0) {
		t.Fatalf("phase_id=%q", got.PhaseID)
	}
}

func TestCodexReviewGateObservationSourceV0ListaDescriptorYaEntregado(t *testing.T) {
	spec := codexDeliverySpecForTestV0()
	ack := codexDeliveryAckForTestV0(spec)
	path := writeCodexDeliveryAckForTestV0(t, spec, ack)
	store := NewInMemoryCodexReceiptDescriptorStoreV0(CodexReceiptDescriptorV0{
		DescriptorRef: "receipt-ref-001",
		RunID:         "run-ref-001",
		AgentRef:      spec.RequestID,
		Spec:          spec,
		AckPath:       path,
	})

	observations, err := (CodexReviewGateObservationSourceV0{Store: store}).
		BuildReviewGateObservationsV0(context.Background(), codexReviewGateRequestForTestV0(spec, nil))
	if err != nil {
		t.Fatalf("BuildReviewGateObservationsV0: %v", err)
	}
	if len(observations) != 1 {
		t.Fatalf("observations=%d", len(observations))
	}
}

func TestCodexReviewGateObservationSourceV0RequiereRevisionPorFicheroGrande(t *testing.T) {
	spec := codexDeliverySpecForTestV0()
	ack := codexDeliveryAckForTestV0(spec)
	path := writeCodexDeliveryAckForTestV0(t, spec, ack)
	store := &staticCodexReceiptStoreV0{
		Descriptors: []CodexReceiptDescriptorV0{{
			DescriptorRef: "receipt-ref-001",
			RunID:         "run-ref-001",
			AgentRef:      spec.RequestID,
			Spec:          spec,
			AckPath:       path,
		}},
	}
	fileEvidence := staticCodexReviewGateFileEvidenceV0{
		Files: []orquestacionnucleoapp.AutoprogrammingReviewGateFileV0{{
			Path:      "README.md",
			LineCount: 301,
		}},
	}

	observations, err := (CodexReviewGateObservationSourceV0{
		Store:        store,
		FileEvidence: fileEvidence,
	}).BuildReviewGateObservationsV0(context.Background(), codexReviewGateRequestForTestV0(spec, nil))
	if err != nil {
		t.Fatalf("BuildReviewGateObservationsV0: %v", err)
	}
	assertCodexReviewGateRejectedV0(t, observations, "gate-issue:file_too_large")
}

func TestCodexReviewGateObservationSourceV0RequiereRevisionPorTestObligatorioAusente(t *testing.T) {
	spec := codexDeliverySpecForTestV0()
	ack := codexDeliveryAckForTestV0(spec)
	ack.Tests = nil
	path := writeCodexDeliveryAckForTestV0(t, spec, ack)
	store := &staticCodexReceiptStoreV0{
		Descriptors: []CodexReceiptDescriptorV0{{
			DescriptorRef: "receipt-ref-001",
			RunID:         "run-ref-001",
			AgentRef:      spec.RequestID,
			Spec:          spec,
			AckPath:       path,
		}},
	}

	observations, err := (CodexReviewGateObservationSourceV0{Store: store}).
		BuildReviewGateObservationsV0(context.Background(), codexReviewGateRequestForTestV0(spec, nil))
	if err != nil {
		t.Fatalf("BuildReviewGateObservationsV0: %v", err)
	}
	assertCodexReviewGateRejectedV0(t, observations, "gate-issue:required_test_missing")
}

type staticCodexReviewGateFileEvidenceV0 struct {
	Files []orquestacionnucleoapp.AutoprogrammingReviewGateFileV0
}

func (provider staticCodexReviewGateFileEvidenceV0) BuildCodexReviewGateFilesV0(
	_ context.Context,
	_ CodexReceiptDescriptorV0,
	_ orquestaruntimecodex.CodexAgentAckV0,
) ([]orquestacionnucleoapp.AutoprogrammingReviewGateFileV0, error) {
	return append([]orquestacionnucleoapp.AutoprogrammingReviewGateFileV0(nil), provider.Files...), nil
}

func codexReviewGateRequestForTestV0(
	spec orquestaruntime.ExternalAgentLaunchSpecV0,
	reviewResults []string,
) orquestacionnucleoapp.ReviewGateObservationRequestV0 {
	return orquestacionnucleoapp.ReviewGateObservationRequestV0{
		Run: orquestacoreworkflow.OrchestrationRunV0{
			RunID:         "run-ref-001",
			CurrentPhase:  orquestacoreworkflow.OrchestrationPhaseRevisionV0,
			Agents:        []string{spec.RequestID},
			StartedAgents: []string{spec.RequestID},
			Deliveries:    []string{spec.AgentPacket.DeliveryRefs.AckRef},
			ReviewResults: reviewResults,
		},
		CorrelationID: "corr-review-gate-001",
		EvidenceRefs:  []string{"evidence-ref-review-gate-001"},
	}
}

func assertCodexReviewGateRejectedV0(
	t *testing.T,
	observations []orquestacionnucleoapp.ReviewGateObservationV0,
	wantEvidence string,
) {
	t.Helper()
	if len(observations) != 1 {
		t.Fatalf("observations=%d", len(observations))
	}
	got := observations[0]
	if got.Status == orquestacoreworkflow.ReviewResultStatusAcceptedV0 {
		t.Fatalf("status aceptado inesperado: %+v", got)
	}
	if got.AcceptedReviewRef != "" || got.QualityGateRef == "" {
		t.Fatalf("observacion de fallo incompleta: %+v", got)
	}
	if !stringInCodexDeliverySetV0(got.EvidenceRefs, wantEvidence) {
		t.Fatalf("evidence_refs=%v, want %q", got.EvidenceRefs, wantEvidence)
	}
}
