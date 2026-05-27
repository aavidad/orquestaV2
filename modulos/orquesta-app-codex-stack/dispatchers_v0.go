package orquestaappcodexstack

import (
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaoutboxdispatch "orquesta/modulos/orquesta-outbox-dispatch"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
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
			Policy:          capacityDecisionPolicyV0(config.Capacity),
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
		TargetPort:   orquestacoreworkflow.OutboxTargetAgentLauncherV0,
		MessageType:  orquestacoreworkflow.OutboxMessageLaunchRuntimeAgentV0,
		MaxReady:     config.Codex.MaxBatchReady,
		CapacityGate: codexLiveProcessCapacityGateV0(config),
		Reader:       config.Stores.OutboxLedger,
		Claimer:      config.Stores.OutboxLedger,
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

func codexLiveProcessCapacityGateV0(
	config ConfigV0,
) orquestacionnucleoapp.LiveProcessCapacityGatePortV0 {
	registry, ok := config.Stores.ProcessRegistry.(orquestacionnucleoapp.AgentProcessRegistryListPortV0)
	if !ok || registry == nil || config.Codex.SnapshotSource == nil || config.Codex.MaxConcurrency <= 0 {
		return nil
	}
	return &orquestacionnucleoapp.LiveProcessCapacityGateV0{
		Registry:       registry,
		SnapshotSource: config.Codex.SnapshotSource,
		Limit:          config.Codex.MaxConcurrency,
		EvidenceRefs:   []string{"evidence-ref-codex-live-process-capacity"},
	}
}

func recordingSpecResolverV0(
	config ConfigV0,
) orquestaruntimecodexdelivery.CodexReceiptRecordingSpecResolverV0 {
	return orquestaruntimecodexdelivery.CodexReceiptRecordingSpecResolverV0{
		Inner: CodexLaunchSpecResolverV0{
			Config:         config.Codex,
			TaskStore:      config.Stores.TaskStore,
			AppChangeStore: config.Stores.AppChangeStore,
		},
		Recorder: config.Stores.ReceiptStore,
		AckPathResolver: orquestaruntimecodexdelivery.AgentScopedCodexReceiptAckPathResolverV0{
			BaseDir:        config.Codex.RuntimeWorkDir,
			ProjectWorkDir: config.Codex.ProjectWorkDir,
		},
		WorktreeBaselineRecorder: codexStackWorktreeBaselineRecorderV0(config),
	}
}

func codexStackWorktreeBaselineRecorderV0(
	config ConfigV0,
) orquestaruntimecodexdelivery.CodexReceiptWorktreeBaselineRecorderPortV0 {
	if config.ReviewGate.LineBudgetSnapshotStore == nil {
		return nil
	}
	return orquestaruntimecodexdelivery.CodexReceiptWorktreeBaselineRecorderV0{
		SnapshotStore:      config.ReviewGate.LineBudgetSnapshotStore,
		IgnorePrefixes:     codexStackWorktreeIgnorePrefixesV0(),
		SnapshotReadBudget: config.ReviewGate.SnapshotReadBudget,
	}
}
