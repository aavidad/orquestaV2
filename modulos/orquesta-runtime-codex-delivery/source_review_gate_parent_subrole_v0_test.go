package orquestaruntimecodexdelivery

import (
	"context"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
)

func TestCodexReviewGateIssuesEvaluableV0ParentSubroleCollisionEsObservable(t *testing.T) {
	if codexReviewGateIssuesEvaluableV0([]orquestaruntime.ExternalAgentConnectorErrorV0{{
		Code:     orquestaruntime.ExternalAgentConnectorErrorCodeV0(orquestaruntimecodex.CodexConnectorAckCorrelationV0),
		Field:    "agent_ack",
		Evidence: []string{orquestaruntimecodex.CodexAgentAckInvalidParentSubroleCollisionEvidenceV0},
	}}) != true {
		t.Fatalf("parent/subrole collision debe llegar al review gate como issue observable")
	}
	if codexReviewGateIssuesEvaluableV0([]orquestaruntime.ExternalAgentConnectorErrorV0{{
		Code:     orquestaruntime.ExternalAgentConnectorErrorCodeV0(orquestaruntimecodex.CodexConnectorAckCorrelationV0),
		Field:    "agent_ack",
		Evidence: []string{"correlation_mismatch"},
	}}) {
		t.Fatalf("correlacion generica sigue siendo causalidad rota dura")
	}
}

func TestCodexReviewGateObservationSourceV0ParentAckConTaskRefSubroleNoCierra(t *testing.T) {
	spec := codexDeliverySpecWithRefsForTestV0(
		"agent-ref-opes-parent-001",
		"task-ref-opes-parent-001",
		"ack-ref-opes-parent-001",
	)
	ack := codexDeliveryAckForTestV0(spec)
	ack.TaskRef = spec.AgentPacket.Task.TaskRef + "-subrole-tests-tutor"
	path := writeCodexDeliveryAckForTestV0(t, spec, ack)
	store := NewInMemoryCodexReceiptDescriptorStoreV0(CodexReceiptDescriptorV0{
		DescriptorRef: "receipt-ref-parent-subrole-collision-001",
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
		t.Fatalf("observations=%+v", observations)
	}
	got := observations[0]
	if got.Status == orquestacoreworkflow.ReviewResultStatusAcceptedV0 ||
		got.AcceptedReviewRef != "" {
		t.Fatalf("parent/subrole collision no debe cerrar como aceptada: %+v", got)
	}
	if !stringInCodexDeliverySetV0(got.EvidenceRefs, "gate-issue:"+orquestaruntimecodex.CodexAgentAckInvalidParentSubroleCollisionEvidenceV0) {
		t.Fatalf("evidence_refs=%v", got.EvidenceRefs)
	}
}
