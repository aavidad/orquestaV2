package orquestaruntimecodexdelivery

import (
	"context"
	"fmt"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntimeworktree "orquesta/modulos/orquesta-runtime-worktree"
)

func programmingTeamAppendSourceProbeDiagnosticsV0(
	ctx context.Context,
	out *strings.Builder,
	receiptStore *InMemoryCodexReceiptDescriptorStoreV0,
	processRegistry orquestacionnucleoapp.AgentProcessRegistryPortV0,
	snapshotSource CodexProcessSnapshotSourcePortV0,
	worktreeStore orquestaruntimeworktree.WorktreeSnapshotStorePortV0,
	run orquestacoreworkflow.OrchestrationRunV0,
) {
	deliverySource := programmingTeamDeliverySourceV0(receiptStore, worktreeStore)
	deliveries, deliveryErr := deliverySource.BuildAgentDeliveryObservationsV0(
		ctx,
		orquestacionnucleoapp.AgentDeliveryObservationRequestV0{
			Run:           run,
			OccurredAt:    "2026-05-10T00:00:00Z",
			CorrelationID: "corr-programming-team-diagnostics",
		},
	)
	fmt.Fprintf(out, "delivery_probe count=%d refs=%v err=%v\n",
		len(deliveries), programmingTeamDeliveryObservationRefsV0(deliveries), deliveryErr)
	progressSource := programmingTeamProgressSourceV0(receiptStore, processRegistry, snapshotSource)
	progress, progressErr := progressSource.BuildAgentProgressObservationsV0(
		ctx,
		orquestacionnucleoapp.AgentProgressObservationRequestV0{
			Run:           run,
			OccurredAt:    "2026-05-10T00:00:00Z",
			CorrelationID: "corr-programming-team-diagnostics",
		},
	)
	fmt.Fprintf(out, "progress_probe count=%d refs=%v err=%v\n",
		len(progress), programmingTeamProgressObservationRefsV0(progress), progressErr)
}

func programmingTeamDeliveryObservationRefsV0(
	observations []orquestacionnucleoapp.AgentDeliveryObservationV0,
) []string {
	refs := make([]string, 0, len(observations))
	for _, observation := range observations {
		refs = append(refs, observation.AgentRef+":"+observation.DeliveryRef+":"+observation.ArtifactRef)
	}
	return refs
}

func programmingTeamProgressObservationRefsV0(
	observations []orquestacionnucleoapp.AgentProgressObservationV0,
) []string {
	refs := make([]string, 0, len(observations))
	for _, observation := range observations {
		refs = append(refs, observation.Report.AgentRequestID+":"+string(observation.Report.Status))
	}
	return refs
}
