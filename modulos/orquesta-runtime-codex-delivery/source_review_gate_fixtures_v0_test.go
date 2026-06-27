package orquestaruntimecodexdelivery

import (
	"context"
	"testing"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
)

type staticCodexReviewGateFileEvidenceV0 struct {
	Files []orquestaautoprogramming.AutoprogrammingReviewGateFileV0
}

func (provider staticCodexReviewGateFileEvidenceV0) BuildCodexReviewGateFilesV0(
	_ context.Context,
	_ CodexReceiptDescriptorV0,
	_ orquestaruntimecodex.CodexAgentAckV0,
) ([]orquestaautoprogramming.AutoprogrammingReviewGateFileV0, error) {
	return append([]orquestaautoprogramming.AutoprogrammingReviewGateFileV0(nil), provider.Files...), nil
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

func assertCodexReviewGateAcceptedWithEvidenceV0(
	t *testing.T,
	observations []orquestacionnucleoapp.ReviewGateObservationV0,
	wantEvidence string,
) {
	t.Helper()
	if len(observations) != 1 {
		t.Fatalf("observations=%d", len(observations))
	}
	got := observations[0]
	if got.Status != orquestacoreworkflow.ReviewResultStatusAcceptedV0 ||
		got.AcceptedReviewRef == "" || got.QualityGateRef == "" {
		t.Fatalf("observacion aceptada incompleta: %+v", got)
	}
	if !stringInCodexDeliverySetV0(got.EvidenceRefs, wantEvidence) {
		t.Fatalf("evidence_refs=%v, want %q", got.EvidenceRefs, wantEvidence)
	}
}

func codexReviewGateObservationByDeliveryForTestV0(
	observations []orquestacionnucleoapp.ReviewGateObservationV0,
	deliveryRef string,
) orquestacionnucleoapp.ReviewGateObservationV0 {
	for _, observation := range observations {
		if observation.DeliveryRef == deliveryRef {
			return observation
		}
	}
	return orquestacionnucleoapp.ReviewGateObservationV0{}
}
