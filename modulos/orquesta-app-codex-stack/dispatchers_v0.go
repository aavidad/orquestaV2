package orquestaappcodexstack

import (
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaoutboxdispatch "orquesta/modulos/orquesta-outbox-dispatch"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
	orquestacionnucleoapp "orquesta/orquestacionnucleoapp"
)

func capacityDispatcherV0(config ConfigV0) orquestacionnucleoapp.OutboxDispatcherBindingV0 {
	return orquestacionnucleoapp.OutboxDispatcherBindingV0{
		TargetPort:  orquestacoreworkflow.OutboxTargetCapacityV0,
		MessageType: orquestacoreworkflow.OutboxMessageRequestCapacityDecisionV0,
		Reader:      config.Stores.OutboxLedger,
		Claimer:     config.Stores.OutboxLedger,
		Executor: orquestacionnucleoapp.CapacityDecisionExecutorV0{
			RunStore:        config.Stores.RunStore,
			EventSink:       config.Stores.EventSink,
			Tier:            config.Capacity.Tier,
			ReasoningEffort: config.Capacity.ReasoningEffort,
			OccurredAt:      config.Capacity.OccurredAt,
			RequestedBy:     config.Capacity.RequestedBy,
			Summary:         config.Capacity.Summary,
			EvidenceRefs:    append([]string(nil), config.Capacity.EvidenceRefs...),
		},
		Acker: config.Stores.OutboxLedger,
	}
}

func stopperDispatcherV0(config ConfigV0) orquestacionnucleoapp.OutboxDispatcherBindingV0 {
	return orquestacionnucleoapp.OutboxDispatcherBindingV0{
		TargetPort:  orquestacoreworkflow.OutboxTargetAgentLauncherV0,
		MessageType: orquestacoreworkflow.OutboxMessageStopRuntimeAgentV0,
		Reader:      config.Stores.OutboxLedger,
		Claimer:     config.Stores.OutboxLedger,
		Executor: orquestacionnucleoapp.AgentStopperExecutorV0{
			RunStore:  config.Stores.RunStore,
			EventSink: config.Stores.EventSink,
			Stopper: orquestacionnucleoapp.ProcessAgentStopperV0{
				Registry: config.Stores.ProcessRegistry,
				Runtime:  config.Codex.ProcessStopper,
			},
			ObservedAt:   config.Capacity.OccurredAt,
			RequestedBy:  config.Capacity.RequestedBy,
			Summary:      "Parada controlada por supervision exterior.",
			EvidenceRefs: []string{"evidence-ref-app-stack-stopper"},
		},
		Acker: config.Stores.OutboxLedger,
	}
}

func directorQuestionDispatcherV0(config ConfigV0) orquestacionnucleoapp.OutboxDispatcherBindingV0 {
	return orquestacionnucleoapp.OutboxDispatcherBindingV0{
		TargetPort:  orquestacoreworkflow.OutboxTargetDirectorV0,
		MessageType: orquestacoreworkflow.OutboxMessageSendDirectorQuestionV0,
		Reader:      config.Stores.OutboxLedger,
		Claimer:     config.Stores.OutboxLedger,
		Executor:    directorQuestionAckExecutorV0{},
		Acker:       config.Stores.OutboxLedger,
	}
}

type directorQuestionAckExecutorV0 struct{}

func (directorQuestionAckExecutorV0) ExecuteOutboxDispatchV0(
	intent orquestaoutboxdispatch.DispatchIntentV0,
) (orquestaoutboxdispatch.OutboxDispatchExecutionResultV0, error) {
	return orquestaoutboxdispatch.OutboxDispatchExecutionResultV0{
		DispatchRef:  "dispatch-ref-" + intent.MessageID,
		EvidenceRefs: []string{"evidence-ref-app-stack-director-question"},
	}, nil
}

func agentBatchDispatcherV0(config ConfigV0) orquestacionnucleoapp.OutboxBatchDispatcherBindingV0 {
	return orquestacionnucleoapp.OutboxBatchDispatcherBindingV0{
		TargetPort:  orquestacoreworkflow.OutboxTargetAgentLauncherV0,
		MessageType: orquestacoreworkflow.OutboxMessageLaunchRuntimeAgentV0,
		MaxReady:    config.Codex.MaxBatchReady,
		Reader:      config.Stores.OutboxLedger,
		Claimer:     config.Stores.OutboxLedger,
		Executor: orquestacionnucleoapp.ExternalProcessAgentBatchExecutorV0{
			RunStore:        config.Stores.RunStore,
			EventSink:       config.Stores.EventSink,
			SpecResolver:    recordingSpecResolverV0(config),
			Runtime:         config.Codex.Runtime,
			ProcessStopper:  config.Codex.ProcessStopper,
			ProcessRegistry: config.Stores.ProcessRegistry,
			MaxConcurrency:  config.Codex.MaxConcurrency,
			OccurredAt:      config.Capacity.OccurredAt,
			RequestedBy:     config.Capacity.RequestedBy,
			EvidenceRefs:    []string{"evidence-ref-app-stack-agent"},
		},
		Acker: config.Stores.OutboxLedger,
	}
}

func recordingSpecResolverV0(
	config ConfigV0,
) orquestaruntimecodexdelivery.CodexReceiptRecordingSpecResolverV0 {
	return orquestaruntimecodexdelivery.CodexReceiptRecordingSpecResolverV0{
		Inner: CodexLaunchSpecResolverV0{
			Config:    config.Codex,
			TaskStore: config.Stores.TaskStore,
		},
		Recorder: config.Stores.ReceiptStore,
		AckPathResolver: orquestaruntimecodexdelivery.AgentScopedCodexReceiptAckPathResolverV0{
			BaseDir:        config.Codex.RuntimeWorkDir,
			ProjectWorkDir: config.Codex.ProjectWorkDir,
		},
	}
}
