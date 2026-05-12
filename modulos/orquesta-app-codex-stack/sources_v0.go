package orquestaappcodexstack

import (
	orquestaappchangedirectorsource "orquesta/modulos/orquesta-app-change-director-source"
	orquestadirectoragentfilesource "orquesta/modulos/orquesta-director-agent-file-source"
	orquestadirectoragentworkflow "orquesta/modulos/orquesta-director-agent-workflow"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
)

func deliverySourceV0(config ConfigV0) orquestaruntimecodexdelivery.CodexDeliveryObservationSourceV0 {
	return orquestaruntimecodexdelivery.CodexDeliveryObservationSourceV0{
		Store: config.Stores.ReceiptStore,
	}
}

func reviewGateSourceV0(config ConfigV0) orquestaruntimecodexdelivery.CodexReviewGateObservationSourceV0 {
	return orquestaruntimecodexdelivery.CodexReviewGateObservationSourceV0{
		Store:              config.Stores.ReceiptStore,
		FileEvidenceResult: config.ReviewGate.FileEvidence,
		MaxLinesPerFile:    config.ReviewGate.MaxLinesPerFile,
		FailureStatus:      config.ReviewGate.FailureStatus,
	}
}

func reviewReworkReplanSourceV0(config ConfigV0) ReviewReworkReplanSourceV0 {
	return ReviewReworkReplanSourceV0{
		Store:    config.Stores.ReceiptStore,
		Capacity: config.Capacity,
	}
}

func assessmentReplanSourceV0(config ConfigV0) AssessmentReplanSourceV0 {
	return AssessmentReplanSourceV0{
		Store:    config.Stores.ReceiptStore,
		Capacity: config.Capacity,
	}
}

func progressSourceV0(config ConfigV0) orquestaruntimecodexdelivery.CodexProgressObservationSourceV0 {
	return orquestaruntimecodexdelivery.CodexProgressObservationSourceV0{
		Store:                      config.Stores.ReceiptStore,
		ProcessRegistry:            config.Stores.ProcessRegistry,
		SnapshotSource:             config.Codex.SnapshotSource,
		State:                      config.Stores.ProgressState,
		Policy:                     codexStackProgressPolicyV0(config.Codex.ProgressPolicy),
		BudgetPolicy:               config.Codex.ProgressBudget,
		MinUnchangedSampleInterval: config.Codex.WaitInterval,
	}
}

func statsProgressSourceV0(config ConfigV0) orquestaruntimecodexdelivery.CodexProgressObservationSourceV0 {
	source := progressSourceV0(config)
	source.EmitProgressing = true
	return source
}

func agentUsageSourceV0(config ConfigV0) CodexStackAgentUsageSourceV0 {
	return CodexStackAgentUsageSourceV0{
		Store:           config.Stores.ReceiptStore,
		ModelAlias:      config.Codex.Model,
		ReasoningEffort: string(config.Capacity.ReasoningEffort),
		UsageMetrics:    config.Codex.UsageMetrics,
	}
}

func codexStackProgressPolicyV0(
	policy orquestaruntime.AgentProgressHeartbeatPolicyV0,
) orquestaruntime.AgentProgressHeartbeatPolicyV0 {
	if policy.StalledAfterNoProgressTicks <= 0 {
		policy.StalledAfterNoProgressTicks = 30
	}
	if policy.LoopAfterRepeatedActions <= 0 {
		policy.LoopAfterRepeatedActions = 180
	}
	if policy.LoopAfterRepeatedActions < policy.StalledAfterNoProgressTicks {
		policy.LoopAfterRepeatedActions = policy.StalledAfterNoProgressTicks
	}
	return policy
}

func directorDecisionSourceV0(config ConfigV0) orquestadirectoragentworkflow.DirectorAgentDecisionSourcePortV0 {
	return compositeDirectorDecisionSourceV0{
		Sources: []orquestadirectoragentworkflow.DirectorAgentDecisionSourcePortV0{
			directorDecisionFileSourceV0(config),
			orquestaappchangedirectorsource.AppChangeDirectorDecisionSourceV0{
				Store: config.Stores.AppChangeStore,
			},
		},
	}
}

func directorDecisionFileSourceV0(config ConfigV0) orquestadirectoragentfilesource.DirectorAgentDecisionFileSourceV0 {
	return orquestadirectoragentfilesource.DirectorAgentDecisionFileSourceV0{
		DescriptorProvider: orquestaruntimecodexdelivery.CodexReceiptDirectorDecisionFileDescriptorProviderV0{
			Store: config.Stores.ReceiptStore,
		},
		Reader: orquestadirectoragentfilesource.OSDirectorAgentDecisionFileReaderV0{},
	}
}
