package orquestaappcodexstack

import (
	"os"
	"path/filepath"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
)

func reviewReworkPlanRequestForTestV0(
	alreadyReplanned bool,
) orquestacionnucleoapp.ReviewReworkReplanPlanRequestV0 {
	run := orquestacoreworkflow.OrchestrationRunV0{
		RunID:        "run-ref-review-rework-stack-001",
		CurrentPhase: orquestacoreworkflow.OrchestrationPhaseRevisionV0,
		Tasks:        []string{"task-ref-first", "task-ref-target"},
		ReworkRequests: []string{
			"rework-request-ref-target#review_result:review-result-ref-target" +
				"#review_request:review-request-ref-target#delivery:delivery-ref-target",
		},
		ReviewResults: []string{
			"review-result-ref-target#review_result:changes_requested" +
				"#review_request:review-request-ref-target#delivery:delivery-ref-target",
		},
	}
	if alreadyReplanned {
		run.ReplanDecisions = []string{
			"replan-ref-target#source:rework-request-ref-target#task:task-ref-target" +
				"#action:retry_task#followups:capacity-ref-target+agent-ref-target",
		}
	}
	return orquestacionnucleoapp.ReviewReworkReplanPlanRequestV0{
		Run:           run,
		OccurredAt:    "2026-05-10T10:00:00Z",
		CorrelationID: "corr-review-rework-stack-001",
		EvidenceRefs:  []string{"evidence-ref-request-review-rework"},
	}
}

func reviewReworkDescriptorForTestV0(
	deliveryRef string,
	taskRef string,
) orquestaruntimecodexdelivery.CodexReceiptDescriptorV0 {
	return orquestaruntimecodexdelivery.CodexReceiptDescriptorV0{
		DescriptorRef: "receipt-ref-" + deliveryRef,
		RunID:         "run-ref-review-rework-stack-001",
		AgentRef:      "agent-ref-" + taskRef,
		AckPath:       "ack-ref-test.json",
		Spec: orquestaruntime.ExternalAgentLaunchSpecV0{
			RequestID: "agent-ref-" + taskRef,
			AgentPacket: orquestaruntime.AgentStartPacketV0{
				DeliveryRefs: orquestaruntime.AgentStartDeliveryRefsV0{
					AckRef: deliveryRef,
				},
				Task: orquestaruntime.AgentStartTaskV0{
					TaskRef: taskRef,
				},
			},
		},
	}
}

func reviewReworkPlanHasEvidenceForTestV0(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func writeReviewReworkAckForTestV0(t interface {
	Helper()
	Fatalf(string, ...any)
	TempDir() string
}, agentRef string, taskRef string, status string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), orquestaruntimecodex.CodexAgentAckFileNameV0)
	data := []byte(`{
		"schema_version":"codex_agent_ack.v0",
		"request_id":"` + agentRef + `",
		"correlation_id":"corr-review-rework-ack-test",
		"ack_ref":"delivery-ref-target",
		"target_module":"orquesta-app-codex-stack",
		"task_ref":"` + taskRef + `",
		"status":"` + status + `"
	}`)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write ack: %v", err)
	}
	return path
}
