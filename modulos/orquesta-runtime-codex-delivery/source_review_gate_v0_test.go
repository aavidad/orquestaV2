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

func TestCodexReviewGateObservationSourceV0WaitAgentRefsFiltraScope(t *testing.T) {
	firstSpec := codexDeliverySpecWithRefsForTestV0("agent-ref-scope-001", "task-ref-scope-001", "ack-ref-scope-001")
	secondSpec := codexDeliverySpecWithRefsForTestV0("agent-ref-outside-001", "task-ref-outside-001", "ack-ref-outside-001")
	store := NewInMemoryCodexReceiptDescriptorStoreV0(
		CodexReceiptDescriptorV0{
			DescriptorRef: "receipt-ref-scope-001",
			RunID:         "run-ref-001",
			AgentRef:      firstSpec.RequestID,
			Spec:          firstSpec,
			AckPath:       writeCodexDeliveryAckForTestV0(t, firstSpec, codexDeliveryAckForTestV0(firstSpec)),
		},
		CodexReceiptDescriptorV0{
			DescriptorRef: "receipt-ref-outside-001",
			RunID:         "run-ref-001",
			AgentRef:      secondSpec.RequestID,
			Spec:          secondSpec,
			AckPath:       writeCodexDeliveryAckForTestV0(t, secondSpec, codexDeliveryAckForTestV0(secondSpec)),
		},
	)
	request := codexReviewGateRequestForTestV0(firstSpec, nil)
	request.Run.Agents = []string{firstSpec.RequestID, secondSpec.RequestID}
	request.Run.StartedAgents = []string{firstSpec.RequestID, secondSpec.RequestID}
	request.Run.Deliveries = []string{
		firstSpec.AgentPacket.DeliveryRefs.AckRef,
		secondSpec.AgentPacket.DeliveryRefs.AckRef,
	}
	request.WaitAgentRefs = []string{firstSpec.RequestID}

	observations, err := (CodexReviewGateObservationSourceV0{Store: store}).
		BuildReviewGateObservationsV0(context.Background(), request)
	if err != nil {
		t.Fatalf("BuildReviewGateObservationsV0: %v", err)
	}
	if len(observations) != 1 {
		t.Fatalf("observations=%+v", observations)
	}
	if observations[0].DeliveryRef != firstSpec.AgentPacket.DeliveryRefs.AckRef {
		t.Fatalf("delivery fuera de scope: %+v", observations[0])
	}
}

func TestCodexReviewGateObservationSourceV0ContinuaTrasRequestReviewPendiente(t *testing.T) {
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
	request := codexReviewGateRequestForTestV0(spec, nil)
	request.Run.Reviews = []string{codexReviewGateReviewRequestIDV0(ack.AckRef)}

	observations, err := (CodexReviewGateObservationSourceV0{Store: store}).
		BuildReviewGateObservationsV0(context.Background(), request)
	if err != nil {
		t.Fatalf("BuildReviewGateObservationsV0: %v", err)
	}
	if len(observations) != 1 {
		t.Fatalf("observations=%d", len(observations))
	}
}

func TestCodexReviewGateObservationSourceV0RevisaEntregaDeAgenteYaCerrado(t *testing.T) {
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
	request := codexReviewGateRequestForTestV0(spec, nil)
	request.Run.StoppedAgents = []string{spec.RequestID}
	request.Run.ConfirmedStoppedAgents = []string{spec.RequestID}

	observations, err := (CodexReviewGateObservationSourceV0{Store: store}).
		BuildReviewGateObservationsV0(context.Background(), request)
	if err != nil {
		t.Fatalf("BuildReviewGateObservationsV0: %v", err)
	}
	if len(observations) != 1 {
		t.Fatalf("review gate debe revisar entregas cerradas: observations=%d", len(observations))
	}
}

func TestCodexReviewGateObservationSourceV0OmiteTrasReworkSolicitado(t *testing.T) {
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
	fileEvidence := staticCodexReviewGateFileEvidenceV0{
		Files: []orquestaautoprogramming.AutoprogrammingReviewGateFileV0{{
			Path:      "README.md",
			LineCount: 301,
		}},
	}
	reviewResultRef := codexReviewGateReviewResultRefV0(ack.AckRef)
	request := codexReviewGateRequestForTestV0(spec, []string{
		reviewResultRef + "#review_result:changes_requested#review_request:" +
			codexReviewGateReviewRequestIDV0(ack.AckRef) + "#delivery:" + ack.AckRef,
	})
	request.Run.ReworkRequests = []string{
		"rework-request-ref-" + reviewResultRef + "#review_result:" + reviewResultRef +
			"#review_request:" + codexReviewGateReviewRequestIDV0(ack.AckRef) +
			"#delivery:" + ack.AckRef,
	}

	observations, err := (CodexReviewGateObservationSourceV0{
		Store:        store,
		FileEvidence: fileEvidence,
	}).BuildReviewGateObservationsV0(context.Background(), request)
	if err != nil {
		t.Fatalf("BuildReviewGateObservationsV0: %v", err)
	}
	if len(observations) != 0 {
		t.Fatalf("observations=%+v", observations)
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
		Files: []orquestaautoprogramming.AutoprogrammingReviewGateFileV0{{
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
