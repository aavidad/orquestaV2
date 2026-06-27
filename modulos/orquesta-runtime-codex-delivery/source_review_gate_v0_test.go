package orquestaruntimecodexdelivery

import (
	"context"
	"testing"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
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

func TestCodexReviewGateObservationSourceV0ACKStrictSinReceiptNoBloqueaOtros(t *testing.T) {
	incompleteSpec := codexDeliverySpecWithRefsForTestV0(
		"agent-ref-review-missing-receipt-001",
		"task-ref-review-missing-receipt-001",
		"ack-ref-review-missing-receipt-001",
	)
	incompleteSpec.AgentPacket.Policies = append(incompleteSpec.AgentPacket.Policies, "ack_terminal_strict")
	incompleteAck := codexDeliveryAckForTestV0(incompleteSpec)
	incompleteAck.TestReceipts = nil
	completeSpec := codexDeliverySpecWithRefsForTestV0(
		"agent-ref-review-valid-001",
		"task-ref-review-valid-001",
		"ack-ref-review-valid-001",
	)
	completeSpec.AgentPacket.Policies = append(completeSpec.AgentPacket.Policies, "ack_terminal_strict")
	store := NewInMemoryCodexReceiptDescriptorStoreV0(
		CodexReceiptDescriptorV0{
			DescriptorRef: "receipt-ref-review-missing-receipt-001",
			RunID:         "run-ref-001",
			AgentRef:      incompleteSpec.RequestID,
			Spec:          incompleteSpec,
			AckPath:       writeCodexDeliveryAckForTestV0(t, incompleteSpec, incompleteAck),
		},
		CodexReceiptDescriptorV0{
			DescriptorRef: "receipt-ref-review-valid-001",
			RunID:         "run-ref-001",
			AgentRef:      completeSpec.RequestID,
			Spec:          completeSpec,
			AckPath:       writeCodexDeliveryAckForTestV0(t, completeSpec, codexDeliveryAckForTestV0(completeSpec)),
		},
	)
	request := codexReviewGateRequestForTestV0(incompleteSpec, nil)
	request.Run.Agents = []string{incompleteSpec.RequestID, completeSpec.RequestID}
	request.Run.StartedAgents = []string{incompleteSpec.RequestID, completeSpec.RequestID}
	request.Run.Deliveries = []string{
		incompleteSpec.AgentPacket.DeliveryRefs.AckRef,
		completeSpec.AgentPacket.DeliveryRefs.AckRef,
	}

	observations, err := (CodexReviewGateObservationSourceV0{Store: store}).
		BuildReviewGateObservationsV0(context.Background(), request)

	if err != nil {
		t.Fatalf("BuildReviewGateObservationsV0 no debe tumbar el lote por receipt faltante: %v", err)
	}
	if len(observations) != 2 {
		t.Fatalf("observations=%+v", observations)
	}
	incompleteObservation := codexReviewGateObservationByDeliveryForTestV0(
		observations,
		incompleteSpec.AgentPacket.DeliveryRefs.AckRef,
	)
	if incompleteObservation.Status != orquestacoreworkflow.ReviewResultStatusChangesRequestedV0 ||
		!stringInCodexDeliverySetV0(
			incompleteObservation.EvidenceRefs,
			"gate-issue:missing_required_test_receipt",
		) {
		t.Fatalf("review incompleta no quedo como changes_requested con evidencia: %+v", incompleteObservation)
	}
	completeObservation := codexReviewGateObservationByDeliveryForTestV0(
		observations,
		completeSpec.AgentPacket.DeliveryRefs.AckRef,
	)
	if completeObservation.Status != orquestacoreworkflow.ReviewResultStatusAcceptedV0 {
		t.Fatalf("review valida no aceptada: %+v", completeObservation)
	}
}

func TestCodexReviewGateObservationSourceV0AceptaWriteSetRaiz(t *testing.T) {
	spec := codexDeliverySpecForTestV0()
	spec.AgentPacket.Task.WriteSet = []string{"."}
	ack := codexDeliveryAckForTestV0(spec)
	ack.Files = orquestaruntimecodex.EvidenceListV0{"docs/extra.md"}
	path := writeCodexDeliveryAckForTestV0(t, spec, ack)
	store := &staticCodexReceiptStoreV0{
		Descriptors: []CodexReceiptDescriptorV0{{
			DescriptorRef: "receipt-ref-root-001",
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
	if len(observations) != 1 || observations[0].Status != orquestacoreworkflow.ReviewResultStatusAcceptedV0 {
		t.Fatalf("observations=%+v", observations)
	}
}

func TestCodexReviewGateObservationSourceV0AceptaWriteSetGlobstar(t *testing.T) {
	spec := codexDeliverySpecForTestV0()
	spec.AgentPacket.Task.WriteSet = []string{
		"go.mod",
		"internal/domain/**",
		"web/admin/**",
		"docs/**",
	}
	ack := codexDeliveryAckForTestV0(spec)
	ack.Files = orquestaruntimecodex.EvidenceListV0{
		"go.mod",
		"internal/domain/appointment.go",
		"web/admin/app.js",
		"docs/pendientes.md",
	}
	path := writeCodexDeliveryAckForTestV0(t, spec, ack)
	store := &staticCodexReceiptStoreV0{
		Descriptors: []CodexReceiptDescriptorV0{{
			DescriptorRef: "receipt-ref-globstar-001",
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
	if len(observations) != 1 || observations[0].Status != orquestacoreworkflow.ReviewResultStatusAcceptedV0 {
		t.Fatalf("observations=%+v", observations)
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

func TestCodexReviewGateObservationSourceV0ReviewAceptaReemplazoSiWaitScopeQuedoViejo(t *testing.T) {
	oldSpec := codexDeliverySpecWithRefsForTestV0("agent-ref-old-scope-001", "task-ref-replaced-001", "ack-ref-old-scope-001")
	replacementSpec := codexDeliverySpecWithRefsForTestV0("agent-ref-replacement-001", "task-ref-replaced-001", "ack-ref-replacement-001")
	store := NewInMemoryCodexReceiptDescriptorStoreV0(
		CodexReceiptDescriptorV0{
			DescriptorRef: "receipt-ref-old-scope-001",
			RunID:         "run-ref-001",
			AgentRef:      oldSpec.RequestID,
			Spec:          oldSpec,
			AckPath:       writeCodexDeliveryAckForTestV0(t, oldSpec, codexDeliveryAckForTestV0(oldSpec)),
		},
		CodexReceiptDescriptorV0{
			DescriptorRef: "receipt-ref-replacement-001",
			RunID:         "run-ref-001",
			AgentRef:      replacementSpec.RequestID,
			Spec:          replacementSpec,
			AckPath:       writeCodexDeliveryAckForTestV0(t, replacementSpec, codexDeliveryAckForTestV0(replacementSpec)),
		},
	)
	request := codexReviewGateRequestForTestV0(replacementSpec, nil)
	request.Run.Agents = []string{oldSpec.RequestID, replacementSpec.RequestID}
	request.Run.StartedAgents = []string{oldSpec.RequestID, replacementSpec.RequestID}
	request.Run.StoppedAgents = []string{oldSpec.RequestID}
	request.Run.LostAgents = []string{oldSpec.RequestID}
	request.Run.DeliveredAgents = []string{replacementSpec.RequestID}
	request.Run.Deliveries = []string{replacementSpec.AgentPacket.DeliveryRefs.AckRef}
	request.WaitAgentRefs = []string{oldSpec.RequestID}

	observations, err := (CodexReviewGateObservationSourceV0{Store: store}).
		BuildReviewGateObservationsV0(context.Background(), request)
	if err != nil {
		t.Fatalf("BuildReviewGateObservationsV0: %v", err)
	}
	if len(observations) != 1 {
		t.Fatalf("observations=%+v", observations)
	}
	if observations[0].DeliveryRef != replacementSpec.AgentPacket.DeliveryRefs.AckRef {
		t.Fatalf("no reviso entrega de reemplazo: %+v", observations[0])
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

func TestCodexReviewGateObservationSourceV0PermiteAceptacionTrasReworkSolicitado(t *testing.T) {
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
	if len(observations) != 1 ||
		observations[0].Status != orquestacoreworkflow.ReviewResultStatusAcceptedV0 ||
		observations[0].AcceptedReviewRef == "" {
		t.Fatalf("observations=%+v", observations)
	}
}
