package orquestadirectorcycle

import (
	"encoding/json"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorcycleoutbox "orquesta/modulos/orquesta-director-cycle-outbox"
	orquestadirectorscheduler "orquesta/modulos/orquesta-director-scheduler"
	orquestapersistence "orquesta/modulos/orquesta-persistence"
)

func cycleStepCapacityCandidateV0() *orquestadirectorscheduler.SchedulerCapacityCommandCandidateV0 {
	return &orquestadirectorscheduler.SchedulerCapacityCommandCandidateV0{
		CommandMeta: cycleStepCommandMetaV0("cmd-cycle-step-capacity", "capacity"),
		Payload: orquestacoreworkflow.RequestCapacityCommandPayloadV0{
			CapacityRequestID:          "capacity-ref-cycle-step-001",
			PhaseID:                    string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			TaskRef:                    "task-ref-cycle-step-001",
			ReasonCode:                 "programacion_siguiente_paso",
			Summary:                    "Capacidad para siguiente tarea.",
			MinimumRecommendedCapacity: orquestacoreworkflow.OrchestrationCapacityMediumV0,
			EvidenceRefs:               []string{"evidence-ref-cycle-step-capacity-001"},
		},
	}
}

func cycleStepCapacityOutboxMessageV0(
	t *testing.T,
	runRef string,
	messageID string,
) orquestacoreworkflow.OutboxMessageV0 {
	t.Helper()
	payload, err := json.Marshal(orquestacoreworkflow.CapacityDecisionRequestV0{
		CapacityRequestID:          "capacity-ref-cycle-step-pending-001",
		RunID:                      runRef,
		PhaseID:                    string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
		TaskRef:                    "task-ref-cycle-step-001",
		ReasonCode:                 "programacion_siguiente_paso",
		Summary:                    "Decision compacta para siguiente paso.",
		MinimumRecommendedCapacity: orquestacoreworkflow.OrchestrationCapacityMediumV0,
		EvidenceRefs:               []string{"evidence-ref-cycle-step-pending-001"},
	})
	if err != nil {
		t.Fatalf("payload: %v", err)
	}
	message := orquestacoreworkflow.OutboxMessageV0{
		MessageID:      messageID,
		MessageType:    orquestacoreworkflow.OutboxMessageRequestCapacityDecisionV0,
		RunID:          runRef,
		IdempotencyKey: "idem-" + messageID,
		CorrelationID:  "corr-cycle-step-001",
		TargetPort:     orquestacoreworkflow.OutboxTargetCapacityV0,
		PayloadVersion: orquestacoreworkflow.OutboxPayloadVersionV0,
		Payload:        payload,
	}
	if err := orquestacoreworkflow.ValidateOutboxMessageV0(message); err != nil {
		t.Fatalf("outbox message: %v", err)
	}
	return message
}

func cycleStepStartCommandV0(t *testing.T) orquestacoreworkflow.OrchestrationCommandV0 {
	t.Helper()
	command, err := orquestacoreworkflow.NewStartRunCommandV0(
		cycleStepCommandMetaV0("cmd-cycle-step-start", "start"),
		orquestacoreworkflow.StartRunCommandPayloadV0{
			ProjectRef: "project-ref-cycle-step-001",
			AppSpecRef: "appspec-ref-cycle-step-001",
		},
	)
	if err != nil {
		t.Fatalf("start command: %v", err)
	}
	return command
}

func cycleStepOpenProgramacionCommandV0(t *testing.T) orquestacoreworkflow.OrchestrationCommandV0 {
	t.Helper()
	command, err := orquestacoreworkflow.NewOpenPhaseCommandV0(
		cycleStepCommandMetaV0("cmd-cycle-step-open-programacion", "open-programacion"),
		orquestacoreworkflow.OpenPhaseCommandPayloadV0{
			PhaseID: string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			Reason:  "Preparar programacion",
		},
	)
	if err != nil {
		t.Fatalf("open command: %v", err)
	}
	return command
}

func cycleStepCommandMetaV0(commandID string, suffix string) orquestacoreworkflow.OrchestrationCommandMetaV0 {
	return orquestacoreworkflow.OrchestrationCommandMetaV0{
		CommandID:      commandID,
		RunID:          cycleStepRunRefV0,
		IdempotencyKey: "idem-cycle-step-" + suffix,
		CorrelationID:  "corr-cycle-step-001",
		RequestedBy:    "director-cycle-step-test",
		OccurredAt:     "2026-05-06T13:00:00Z",
	}
}

func cycleStepLedgerIssuesV0(
	issues []orquestapersistence.OutboxLedgerIssueV0,
) []orquestadirectorcycleoutbox.DirectorCycleOutboxIssueV0 {
	if len(issues) == 0 {
		return nil
	}
	out := make([]orquestadirectorcycleoutbox.DirectorCycleOutboxIssueV0, 0, len(issues))
	for _, issue := range issues {
		out = append(out, orquestadirectorcycleoutbox.DirectorCycleOutboxIssueV0{
			Code:    issue.Code,
			Field:   issue.Field,
			Message: issue.Message,
		})
	}
	return out
}
