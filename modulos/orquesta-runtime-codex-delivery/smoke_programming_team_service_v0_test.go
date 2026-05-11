package orquestaruntimecodexdelivery

import (
	"time"

	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestadirectoragentfilesource "orquesta/modulos/orquesta-director-agent-file-source"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimeworktree "orquesta/modulos/orquesta-runtime-worktree"
	orquestacionnucleoapp "orquesta/orquestacionnucleoapp"
)

func programmingTeamDirectorStartRequestV0(
	runRef string,
	cfg codexRealSmokeConfigV0,
) orquestaappdirectorservice.StartAppDirectorRequestV0 {
	return orquestaappdirectorservice.StartAppDirectorRequestV0{
		RunRef:               runRef,
		ProjectRef:           "project-ref-programming-team-real-001",
		OccurredAt:           "2026-05-10T00:00:00Z",
		CorrelationID:        "corr-programming-team-001",
		RequestedBy:          "orquesta-programming-team-smoke",
		AppSpecRequest:       mcpFormDirectorSmokeFormV0().ToAppSpecRequestV0(),
		MaxBursts:            10,
		MaxStepsPerBurst:     8,
		MaxDispatchesPerWait: 8,
		MaxCommands:          18,
		MaxOutboxPerCycle:    8,
		MaxDecisionCycles:    4,
		MaxExternalWaits:     programmingTeamMaxExternalWaitsV0(cfg.Timeout, 5*time.Second),
	}
}

func programmingTeamDecisionSourceV0(
	store *InMemoryCodexReceiptDescriptorStoreV0,
) orquestadirectoragentfilesource.DirectorAgentDecisionFileSourceV0 {
	return orquestadirectoragentfilesource.DirectorAgentDecisionFileSourceV0{
		DescriptorProvider: CodexReceiptDirectorDecisionFileDescriptorProviderV0{Store: store},
		Reader:             orquestadirectoragentfilesource.OSDirectorAgentDecisionFileReaderV0{},
	}
}

func programmingTeamDeliverySourceV0(
	receiptStore *InMemoryCodexReceiptDescriptorStoreV0,
	worktreeStore orquestaruntimeworktree.WorktreeSnapshotStorePortV0,
) CodexDeliveryObservationSourceV0 {
	return CodexDeliveryObservationSourceV0{
		Store: receiptStore,
		WorktreeVerifier: CodexReceiptWorktreeVerifierV0{
			SnapshotStore:  worktreeStore,
			IgnorePrefixes: []string{".orquesta-codex-runtime"},
			Mode:           CodexReceiptWorktreeAckFilesV0,
		},
	}
}

func programmingTeamProgressSourceV0(
	receiptStore *InMemoryCodexReceiptDescriptorStoreV0,
	processRegistry orquestacionnucleoapp.AgentProcessRegistryPortV0,
	snapshotSource CodexProcessSnapshotSourcePortV0,
) CodexProgressObservationSourceV0 {
	return CodexProgressObservationSourceV0{
		Store:                      receiptStore,
		ProcessRegistry:            processRegistry,
		SnapshotSource:             snapshotSource,
		State:                      NewInMemoryCodexProgressStateStoreV0(),
		MinUnchangedSampleInterval: 5 * time.Second,
		Policy: orquestaruntime.AgentProgressHeartbeatPolicyV0{
			StalledAfterNoProgressTicks: 60,
			LoopAfterRepeatedActions:    36,
		},
	}
}

func programmingTeamMaxExternalWaitsV0(timeout time.Duration, interval time.Duration) int {
	if interval <= 0 {
		interval = time.Second
	}
	maxWaits := int(timeout / interval)
	if maxWaits < 1 {
		return 1
	}
	return maxWaits
}
