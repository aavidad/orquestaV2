package orquestaappcodexstack

import (
	"time"

	orquestaappchangedirectorsource "orquesta/modulos/orquesta-app-change-director-source"
	orquestacoreleases "orquesta/modulos/orquesta-core-leases"
	orquestadirectoragentfilesource "orquesta/modulos/orquesta-director-agent-file-source"
	orquestadirectoragentworkflow "orquesta/modulos/orquesta-director-agent-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
	orquestaruntimeworktree "orquesta/modulos/orquesta-runtime-worktree"
)

func deliverySourceV0(config ConfigV0) orquestacionnucleoapp.AgentDeliveryObservationProviderPortV0 {
	base := orquestaruntimecodexdelivery.CodexDeliveryObservationSourceV0{
		Store:                                 config.Stores.ReceiptStore,
		WorktreeVerifier:                      codexStackWorktreeVerifierV0(config),
		PromoteMaterializedArtifactWithoutAck: config.PromoteMaterializedArtifactWithoutAck,
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
		SnapshotStore:      config.ReviewGate.LineBudgetSnapshotStore,
		IgnorePrefixes:     codexStackWorktreeIgnorePrefixesV0(),
		SnapshotReadBudget: config.ReviewGate.SnapshotReadBudget,
	}
}

func codexStackWorktreeIgnorePrefixesV0() []string {
	prefixes := []string{
		".git",
		"bin",
		"orquesta_state",
		"runtime_orquesta",
	}
	return append(prefixes, orquestaruntimeworktree.DefaultWorktreeControlIgnorePrefixesV0()...)
}

func reviewGateSourceV0(config ConfigV0) orquestacionnucleoapp.ReviewGateObservationProviderPortV0 {
	policy := codexStackReviewGatePolicyFromConfigV0(config.ReviewGate)
	base := orquestaruntimecodexdelivery.CodexReviewGateObservationSourceV0{
		Store:              config.Stores.ReceiptStore,
		FileEvidenceResult: codexStackReviewGateFileEvidenceV0(config.ReviewGate),
		MaxLinesPerFile:    policy.MaxGoFileLinesV0(),
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

func progressAndLeaseSourcesV0(
	config ConfigV0,
) (orquestacionnucleoapp.AgentProgressObservationProviderPortV0, orquestacionnucleoapp.AgentLeaseAssessmentProviderPortV0) {
	progress := progressSourceV0(config)
	leasePolicy, ok := codexStackLeasePolicyV0(config)
	if !ok {
		return progress, nil
	}
	bridge := &orquestacionnucleoapp.AgentProgressLeaseBridgeV0{
		ProgressSource: progress,
		PolicySource: orquestacionnucleoapp.StaticAgentProgressLeasePolicyProviderV0{
			Policy:       leasePolicy,
			EvidenceRefs: []string{"evidence-ref-codex-stack-progress-lease-policy"},
		},
	}
	return bridge, bridge
}

func codexStackLeasePolicyV0(
	config ConfigV0,
) (orquestacoreleases.AgentLeasePolicyV0, bool) {
	if config.Codex.ProgressBudget.MaxExpected <= 0 && config.Codex.ProgressBudget.NoActivityLimit <= 0 {
		return orquestacoreleases.AgentLeasePolicyV0{}, false
	}
	noActivitySeconds := codexStackDurationSecondsV0(config.Codex.ProgressBudget.NoActivityLimit)
	maxExpectedSeconds := codexStackDurationSecondsV0(config.Codex.ProgressBudget.MaxExpected)
	if noActivitySeconds <= 0 {
		noActivitySeconds = int64(codexStackProgressPolicyV0(config.Codex.ProgressPolicy).StalledAfterNoProgressTicks)
	}
	if maxExpectedSeconds <= 0 || maxExpectedSeconds < noActivitySeconds {
		maxExpectedSeconds = noActivitySeconds * 2
	}
	return orquestacoreleases.AgentLeasePolicyV0{
		LeasePolicyRef:          "lease-policy-ref-codex-progress-budget-v0",
		LaunchTimeoutSeconds:    codexStackLeaseSecondsV0(noActivitySeconds),
		HeartbeatTimeoutSeconds: codexStackLeaseSecondsV0(noActivitySeconds),
		TotalTimeoutSeconds:     codexStackLeaseSecondsV0(maxExpectedSeconds),
		TimeoutAction:           orquestacoreleases.AgentLeaseTimeoutStopAgentV0,
		EvidenceRefs:            []string{"evidence-ref-codex-stack-progress-budget-v0"},
	}, true
}

func codexStackDurationSecondsV0(value time.Duration) int64 {
	if value <= 0 {
		return 0
	}
	return int64(value / time.Second)
}

func codexStackLeaseSecondsV0(value int64) int {
	if value < 1 {
		return 1
	}
	if value > 86400 {
		return 86400
	}
	return int(value)
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
		TaskStore:      config.Stores.TaskStore,
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
		Budget:               config.DirectorDecisionBudget,
		PlanBootstrapSource: ProjectPlanBootstrapDirectorDecisionSourceV0{
			ProjectWorkDir: config.Codex.ProjectWorkDir,
		},
		Sources: []orquestadirectoragentworkflow.DirectorAgentDecisionSourcePortV0{
			directorDecisionFileSourceV0(config),
			orquestaappchangedirectorsource.AppChangeDirectorDecisionSourceV0{
				Store: config.Stores.AppChangeStore,
			},
			ExternalJobIntegrationDecisionSourceV0{
				AppChangeStore: config.Stores.AppChangeStore,
				TaskStore:      config.Stores.TaskStore,
			},
			ProjectBacklogDirectorDecisionSourceV0{
				ProjectWorkDir: config.Codex.ProjectWorkDir,
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
		BatchBudget:         config.DirectorDecisionBudget,
		IgnoreInvalidFiles:  true,
	}
}

func directorDecisionFileConsumptionRecorderV0(
	store CodexReceiptStorePortV0,
) orquestadirectoragentfilesource.DirectorAgentDecisionFileConsumptionRecorderPortV0 {
	recorder, _ := store.(orquestadirectoragentfilesource.DirectorAgentDecisionFileConsumptionRecorderPortV0)
	return recorder
}
