package orquestaappcodexstack

import (
	orquestaappchangedirectorsource "orquesta/modulos/orquesta-app-change-director-source"
	orquestadirectoragentfilesource "orquesta/modulos/orquesta-director-agent-file-source"
	orquestadirectoragentworkflow "orquesta/modulos/orquesta-director-agent-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
	orquestaruntimeworktree "orquesta/modulos/orquesta-runtime-worktree"
)

func deliverySourceV0(config ConfigV0) orquestacionnucleoapp.AgentDeliveryObservationProviderPortV0 {
	base := orquestaruntimecodexdelivery.CodexDeliveryObservationSourceV0{
		Store:            config.Stores.ReceiptStore,
		WorktreeVerifier: codexStackWorktreeVerifierV0(config),
	}
	recovery := domainWorkRecoveryDeliverySourceV0{
		Stores:         config.Stores,
		DomainWork:     config.DomainWork,
		DomainDelivery: config.DomainDelivery,
	}
	if !recovery.readyV0() {
		return base
	}
	return compositeAgentDeliveryObservationSourceV0{
		Sources: []orquestacionnucleoapp.AgentDeliveryObservationProviderPortV0{
			base,
			recovery,
		},
	}
}

func codexStackWorktreeVerifierV0(
	config ConfigV0,
) orquestaruntimecodexdelivery.CodexReceiptWorktreeVerifierPortV0 {
	if config.ReviewGate.LineBudgetSnapshotStore == nil {
		return nil
	}
	return orquestaruntimecodexdelivery.CodexReceiptWorktreeVerifierV0{
		SnapshotStore:  config.ReviewGate.LineBudgetSnapshotStore,
		IgnorePrefixes: codexStackWorktreeIgnorePrefixesV0(),
	}
}

func codexStackWorktreeIgnorePrefixesV0() []string {
	prefixes := []string{".git"}
	return append(prefixes, orquestaruntimeworktree.DefaultWorktreeControlIgnorePrefixesV0()...)
}

func reviewGateSourceV0(config ConfigV0) orquestacionnucleoapp.ReviewGateObservationProviderPortV0 {
	base := orquestaruntimecodexdelivery.CodexReviewGateObservationSourceV0{
		Store:              config.Stores.ReceiptStore,
		FileEvidenceResult: codexStackReviewGateFileEvidenceV0(config.ReviewGate),
		MaxLinesPerFile:    config.ReviewGate.MaxLinesPerFile,
		FailureStatus:      config.ReviewGate.FailureStatus,
	}
	return codexStackReviewGateRepairSourceV0{
		Store: config.Stores.ReceiptStore,
		Inner: base,
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
		Store:        config.Stores.ReceiptStore,
		UsageMetrics: config.Codex.UsageMetrics,
	}
}

func externalJobStatsSourceV0(config ConfigV0) CodexStackExternalJobStatsSourceV0 {
	return CodexStackExternalJobStatsSourceV0{
		RunStore:       config.Stores.RunStore,
		AppChangeStore: config.Stores.AppChangeStore,
		ReceiptStore:   config.Stores.ReceiptStore,
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
		AppChangeStore:       config.Stores.AppChangeStore,
		DomainRequiredPolicy: config.DomainTests.Policy,
		Sources: []orquestadirectoragentworkflow.DirectorAgentDecisionSourcePortV0{
			directorDecisionFileSourceV0(config),
			orquestaappchangedirectorsource.AppChangeDirectorDecisionSourceV0{
				Store: config.Stores.AppChangeStore,
			},
			AutoprogrammingDirectorDecisionSourceV0{
				TaskStore: config.Stores.TaskStore,
			},
		},
	}
}

func directorDecisionFileSourceV0(config ConfigV0) orquestadirectoragentfilesource.DirectorAgentDecisionFileSourceV0 {
	return orquestadirectoragentfilesource.DirectorAgentDecisionFileSourceV0{
		DescriptorProvider: orquestaruntimecodexdelivery.CodexReceiptDirectorDecisionFileDescriptorProviderV0{
			Store: config.Stores.ReceiptStore,
		},
		Reader:              orquestadirectoragentfilesource.OSDirectorAgentDecisionFileReaderV0{},
		ConsumptionRecorder: directorDecisionFileConsumptionRecorderV0(config.Stores.ReceiptStore),
		IgnoreInvalidFiles:  true,
	}
}

func directorDecisionFileConsumptionRecorderV0(
	store CodexReceiptStorePortV0,
) orquestadirectoragentfilesource.DirectorAgentDecisionFileConsumptionRecorderPortV0 {
	recorder, _ := store.(orquestadirectoragentfilesource.DirectorAgentDecisionFileConsumptionRecorderPortV0)
	return recorder
}
