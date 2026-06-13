package orquestaappcodexstack

import (
	"context"
	"testing"

	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaoutboxdispatch "orquesta/modulos/orquesta-outbox-dispatch"
)

func TestRecoverMissingLaunchOutboxForRequestedAgentsV0ReconstruyeOutboxTrasPersistParcial(t *testing.T) {
	ctx := context.Background()
	runRef := "run-ref-launch-outbox-recovery-001"
	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	eventSink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	run := codexStackLaunchOutboxRecoveryRunV0(t, ctx, runStore, eventSink, runRef)
	stack := StackV0{
		Stores: StoresV0{
			RunStore:     runStore,
			EventSink:    eventSink,
			OutboxLedger: ledger,
		},
		Ports: stackLaunchOutboxRecoveryPortsV0(runStore, eventSink, ledger),
	}

	recovered, err := stack.recoverMissingLaunchOutboxForRequestedAgentsV0(ctx, run)
	if err != nil {
		t.Fatalf("recoverMissingLaunchOutboxForRequestedAgentsV0: %v", err)
	}
	if !recovered {
		t.Fatalf("recovery no reconstruyo outbox")
	}
	pending, issues := ledger.ListPendingOutboxV0(orquestaoutboxdispatch.PendingOutboxFilterV0{
		RunID:       runRef,
		TargetPort:  orquestacoreworkflow.OutboxTargetAgentLauncherV0,
		MessageType: orquestacoreworkflow.OutboxMessageLaunchRuntimeAgentV0,
	})
	if len(issues) > 0 || len(pending) != 1 {
		t.Fatalf("pending=%+v issues=%+v", pending, issues)
	}
	if pending[0].MessageID != "outbox-launchruntimeagent-idem-agent-launch-outbox-recovery-001" {
		t.Fatalf("message_id=%s", pending[0].MessageID)
	}
}

func codexStackLaunchOutboxRecoveryRunV0(
	t *testing.T,
	ctx context.Context,
	runStore *orquestacionnucleoapp.InMemoryRunStoreV0,
	eventSink *orquestacionnucleoapp.InMemoryEventSinkV0,
	runRef string,
) orquestacoreworkflow.OrchestrationRunV0 {
	t.Helper()
	run := orquestacoreworkflow.OrchestrationRunV0{}
	commands := codexStackLaunchOutboxRecoveryCommandsV0(t, runRef)
	for _, command := range commands {
		result, err := orquestacoreworkflow.HandleCommandV0(run, command)
		if err != nil {
			t.Fatalf("HandleCommandV0 %s: %v", command.CommandType, err)
		}
		for _, event := range result.Events {
			var applyErr error
			run, applyErr = orquestacoreworkflow.ApplyEventV0(run, event)
			if applyErr != nil {
				t.Fatalf("ApplyEventV0 %s: %v", event.EventType, applyErr)
			}
		}
		if len(result.Events) > 0 {
			if err := eventSink.AppendRunEventsV0(ctx, runRef, result.Events); err != nil {
				t.Fatalf("AppendRunEventsV0: %v", err)
			}
		}
	}
	if err := runStore.SaveRunV0(ctx, run); err != nil {
		t.Fatalf("SaveRunV0: %v", err)
	}
	return run
}

func codexStackLaunchOutboxRecoveryCommandsV0(
	t *testing.T,
	runRef string,
) []orquestacoreworkflow.OrchestrationCommandV0 {
	t.Helper()
	commands := make([]orquestacoreworkflow.OrchestrationCommandV0, 0, 5)
	command, err := orquestacoreworkflow.NewStartRunCommandV0(
		launchOutboxRecoveryCommandMetaV0(runRef, "cmd-start-launch-outbox-recovery-001", "idem-start-launch-outbox-recovery-001"),
		orquestacoreworkflow.StartRunCommandPayloadV0{
			ProjectRef: "project-ref-launch-outbox-recovery-001",
			AppSpecRef: "appspec-ref-launch-outbox-recovery-001",
		},
	)
	commands = append(commands, mustLaunchOutboxRecoveryCommandV0(t, command, err))
	command, err = orquestacoreworkflow.NewOpenPhaseCommandV0(
		launchOutboxRecoveryCommandMetaV0(runRef, "cmd-open-programacion-launch-outbox-recovery-001", "idem-open-programacion-launch-outbox-recovery-001"),
		orquestacoreworkflow.OpenPhaseCommandPayloadV0{PhaseID: string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0)},
	)
	commands = append(commands, mustLaunchOutboxRecoveryCommandV0(t, command, err))
	command, err = orquestacoreworkflow.NewRequestCapacityCommandV0(
		launchOutboxRecoveryCommandMetaV0(runRef, "cmd-capacity-launch-outbox-recovery-001", "idem-capacity-launch-outbox-recovery-001"),
		orquestacoreworkflow.RequestCapacityCommandPayloadV0{
			CapacityRequestID:          "capacity-ref-launch-outbox-recovery-001",
			PhaseID:                    string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			TaskRef:                    "task-ref-launch-outbox-recovery-001",
			ReasonCode:                 "work_profile_implementation",
			Summary:                    "Capacidad para implementar tarea acotada.",
			MinimumRecommendedCapacity: orquestacoreworkflow.OrchestrationCapacityHighV0,
		},
	)
	commands = append(commands, mustLaunchOutboxRecoveryCommandV0(t, command, err))
	command, err = orquestacoreworkflow.NewRegisterCapacityDecisionCommandV0(
		launchOutboxRecoveryCommandMetaV0(runRef, "cmd-capacity-decision-launch-outbox-recovery-001", "idem-capacity-decision-launch-outbox-recovery-001"),
		orquestacoreworkflow.RegisterCapacityDecisionCommandPayloadV0{
			CapacityRequestID: "capacity-ref-launch-outbox-recovery-001",
			DecisionRef:       "capacity-decision-launch-outbox-recovery-001",
			Tier:              orquestacoreworkflow.OrchestrationCapacityHighV0,
			ReasoningEffort:   orquestacoreworkflow.OrchestrationCapacityHighV0,
			Summary:           "Capacidad aceptada.",
		},
	)
	commands = append(commands, mustLaunchOutboxRecoveryCommandV0(t, command, err))
	command, err = orquestacoreworkflow.NewRequestAgentCommandV0(
		launchOutboxRecoveryCommandMetaV0(runRef, "cmd-agent-launch-outbox-recovery-001", "idem-agent-launch-outbox-recovery-001"),
		orquestacoreworkflow.RequestAgentCommandPayloadV0{
			AgentRequestID:     "agent-ref-launch-outbox-recovery-001",
			PhaseID:            string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			TaskRef:            "task-ref-launch-outbox-recovery-001",
			CapacityRequestRef: "capacity-ref-launch-outbox-recovery-001",
			Role:               "implementacion",
			Summary:            "Microtarea acotada lista.",
			EvidenceRefs:       []string{"evidence-ref-agent-launch-outbox-recovery-001"},
			SkillRefs:          []string{"skill-ref-orquesta-programacion-integracion-v0"},
		},
	)
	commands = append(commands, mustLaunchOutboxRecoveryCommandV0(t, command, err))
	return commands
}

func stackLaunchOutboxRecoveryPortsV0(
	runStore *orquestacionnucleoapp.InMemoryRunStoreV0,
	eventSink *orquestacionnucleoapp.InMemoryEventSinkV0,
	ledger *orquestacionnucleoapp.InMemoryOutboxLedgerV0,
) orquestaappdirectorservice.StartAppDirectorPortsV0 {
	return orquestaappdirectorservice.StartAppDirectorPortsV0{
		RunStore:     runStore,
		EventSink:    eventSink,
		EventReader:  eventSink,
		OutboxLedger: ledger,
	}
}

func launchOutboxRecoveryCommandMetaV0(
	runRef string,
	commandID string,
	idempotencyKey string,
) orquestacoreworkflow.OrchestrationCommandMetaV0 {
	return orquestacoreworkflow.OrchestrationCommandMetaV0{
		CommandID:      commandID,
		RunID:          runRef,
		IdempotencyKey: idempotencyKey,
		CorrelationID:  "corr-launch-outbox-recovery-001",
		RequestedBy:    "test-launch-outbox-recovery",
		OccurredAt:     "2026-06-13T10:00:00Z",
	}
}

func mustLaunchOutboxRecoveryCommandV0(
	t *testing.T,
	command orquestacoreworkflow.OrchestrationCommandV0,
	err error,
) orquestacoreworkflow.OrchestrationCommandV0 {
	t.Helper()
	if err != nil {
		t.Fatalf("command: %v", err)
	}
	return command
}
