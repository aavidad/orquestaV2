package orquestadirectoragentworkflow

import (
	"encoding/json"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoragent "orquesta/modulos/orquesta-director-agent"
)

func TestBuildDirectorAgentWorkflowCommandV0TraduceRevision(t *testing.T) {
	tests := []struct {
		name     string
		request  DirectorAgentWorkflowCommandRequestV0
		wantType string
	}{
		{
			name:     "request",
			request:  reviewWorkflowRequestV0(reviewRequestDecisionV0()),
			wantType: orquestacoreworkflow.OrchestrationCommandRequestReviewV0,
		},
		{
			name:     "record",
			request:  reviewWorkflowRequestV0(reviewResultDecisionV0()),
			wantType: orquestacoreworkflow.OrchestrationCommandRecordReviewResultV0,
		},
		{
			name:     "accept",
			request:  reviewWorkflowRequestV0(acceptReviewDecisionV0()),
			wantType: orquestacoreworkflow.OrchestrationCommandAcceptReviewV0,
		},
	}
	for _, tt := range tests {
		command, issues := BuildDirectorAgentWorkflowCommandV0(tt.request)
		if len(issues) != 0 {
			t.Fatalf("%s issues inesperados: %+v", tt.name, issues)
		}
		if command.CommandType != tt.wantType {
			t.Fatalf("%s command_type=%s want %s", tt.name, command.CommandType, tt.wantType)
		}
		if !json.Valid(command.Payload) {
			t.Fatalf("%s payload invalido: %s", tt.name, string(command.Payload))
		}
	}
}

func reviewWorkflowRequestV0(
	decision orquestadirectoragent.DirectorAgentDecisionV0,
) DirectorAgentWorkflowCommandRequestV0 {
	return DirectorAgentWorkflowCommandRequestV0{
		Decision:      decision,
		OccurredAt:    "2026-05-09T23:20:00Z",
		CorrelationID: "corr-review-workflow-001",
		RequestedBy:   "orquesta-director-agent-workflow-test",
	}
}

func reviewRequestDecisionV0() orquestadirectoragent.DirectorAgentDecisionV0 {
	return orquestadirectoragent.DirectorAgentDecisionV0{
		SchemaVersion: orquestadirectoragent.DirectorAgentDecisionSchemaVersionV0,
		DecisionRef:   "director-decision-review-request-001",
		RunID:         "run-ref-001",
		PhaseID:       "revision",
		CommandType:   orquestadirectoragent.DirectorAgentCommandRequestReviewV0,
		CommandRef:    "command-ref-review-request-001",
		Summary:       "Solicitar revision de entrega.",
		EvidenceRefs:  []string{"evidence-ref-review-request-001"},
		RequestReview: &orquestadirectoragent.DirectorAgentRequestReviewCommandV0{
			ReviewRequestID: "review-request-ref-001",
			PhaseID:         "revision",
			DeliveryRef:     "delivery-ref-001",
			Summary:         "Revisar entrega compacta.",
			EvidenceRefs:    []string{"evidence-ref-review-request-002"},
		},
	}
}

func reviewResultDecisionV0() orquestadirectoragent.DirectorAgentDecisionV0 {
	return orquestadirectoragent.DirectorAgentDecisionV0{
		SchemaVersion: orquestadirectoragent.DirectorAgentDecisionSchemaVersionV0,
		DecisionRef:   "director-decision-review-result-001",
		RunID:         "run-ref-001",
		PhaseID:       "revision",
		CommandType:   orquestadirectoragent.DirectorAgentCommandRecordReviewResultV0,
		CommandRef:    "command-ref-review-result-001",
		Summary:       "Registrar revision aceptada.",
		EvidenceRefs:  []string{"evidence-ref-review-result-001"},
		RecordReviewResult: &orquestadirectoragent.DirectorAgentReviewResultCommandV0{
			ReviewResultRef: "review-result-ref-001",
			ReviewRequestID: "review-request-ref-001",
			DeliveryRef:     "delivery-ref-001",
			Status:          "accepted",
			Summary:         "Entrega revisada sin cambios bloqueantes.",
			EvidenceRefs:    []string{"evidence-ref-review-result-002"},
			QualityGateRef:  "quality-gate-ref-001",
		},
	}
}

func acceptReviewDecisionV0() orquestadirectoragent.DirectorAgentDecisionV0 {
	return orquestadirectoragent.DirectorAgentDecisionV0{
		SchemaVersion: orquestadirectoragent.DirectorAgentDecisionSchemaVersionV0,
		DecisionRef:   "director-decision-accept-review-001",
		RunID:         "run-ref-001",
		PhaseID:       "revision",
		CommandType:   orquestadirectoragent.DirectorAgentCommandAcceptReviewV0,
		CommandRef:    "command-ref-accept-review-001",
		Summary:       "Aceptar revision.",
		EvidenceRefs:  []string{"evidence-ref-accept-review-001"},
		AcceptReview: &orquestadirectoragent.DirectorAgentAcceptReviewCommandV0{
			AcceptedReviewRef: "accepted-review-ref-001",
			PhaseID:           "revision",
			ReviewRequestID:   "review-request-ref-001",
			DeliveryRef:       "delivery-ref-001",
			Summary:           "Revision aceptada.",
			EvidenceRefs:      []string{"evidence-ref-accept-review-002"},
		},
	}
}
