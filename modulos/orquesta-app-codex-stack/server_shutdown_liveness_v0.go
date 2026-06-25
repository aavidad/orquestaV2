package orquestaappcodexstack

import (
	"context"
	"strings"

	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

type stackShutdownAgentLivenessV0 struct {
	Observed     bool
	LiveRefs     []string
	StaleRefs    []string
	LostRefs     []string
	EvidenceRefs []string
}

func buildStackShutdownAgentLivenessV0(
	ctx context.Context,
	config ConfigV0,
	stats orquestacionnucleoapp.DirectorRunStatsV0,
) stackShutdownAgentLivenessV0 {
	if ctx == nil {
		ctx = context.Background()
	}
	liveness := stackShutdownAgentLivenessV0{}
	if config.Stores.ProcessRegistry == nil || config.Codex.SnapshotSource == nil {
		liveness.LiveRefs = stackShutdownInFlightAgentRefsV0(stats)
		return liveness
	}
	liveness.Observed = true
	for _, agent := range stats.Agents {
		if err := ctx.Err(); err != nil {
			return normalizeStackShutdownAgentLivenessV0(stackShutdownAgentLivenessV0{
				LiveRefs:     stackShutdownInFlightAgentRefsV0(stats),
				EvidenceRefs: []string{"shutdown-agent-liveness-context-done"},
			})
		}
		if !agent.InFlight {
			continue
		}
		liveness = classifyStackShutdownAgentLivenessV0(config, liveness, agent)
	}
	return normalizeStackShutdownAgentLivenessV0(liveness)
}

func classifyStackShutdownAgentLivenessV0(
	config ConfigV0,
	liveness stackShutdownAgentLivenessV0,
	agent orquestacionnucleoapp.DirectorAgentStatsV0,
) stackShutdownAgentLivenessV0 {
	agentRef := strings.TrimSpace(agent.AgentRequestID)
	if agent.Process == nil || strings.TrimSpace(agent.Process.ProcessRef) == "" {
		liveness.StaleRefs = append(liveness.StaleRefs, agentRef)
		liveness.EvidenceRefs = append(liveness.EvidenceRefs,
			"shutdown-agent-stale-no-process-"+safeStackShutdownRefPartV0(agentRef))
		return liveness
	}
	record := orquestacionnucleoapp.AgentProcessRegistryRecordV0{
		AgentRequestID: agentRef,
		ProcessRef:     strings.TrimSpace(agent.Process.ProcessRef),
		SessionRef:     strings.TrimSpace(agent.Process.SessionRef),
		LaunchRef:      strings.TrimSpace(agent.Process.LaunchRef),
	}
	snapshot, err := config.Codex.SnapshotSource.SnapshotV0(record.ProcessRef)
	if err != nil {
		if codexStackProcessRuntimeMissingV0(err) {
			liveness.LostRefs = append(liveness.LostRefs, agentRef)
			liveness.EvidenceRefs = append(liveness.EvidenceRefs,
				"shutdown-agent-lost-runtime-missing-"+safeStackShutdownRefPartV0(agentRef))
			return liveness
		}
		liveness.LiveRefs = append(liveness.LiveRefs, agentRef)
		liveness.EvidenceRefs = append(liveness.EvidenceRefs,
			"shutdown-agent-live-runtime-error-"+safeStackShutdownRefPartV0(agentRef))
		return liveness
	}
	if !runControlRegisteredProcessMatchesSnapshotV0(record, snapshot) {
		liveness.LiveRefs = append(liveness.LiveRefs, agentRef)
		liveness.EvidenceRefs = append(liveness.EvidenceRefs,
			"shutdown-agent-live-identity-mismatch-"+safeStackShutdownRefPartV0(agentRef))
		return liveness
	}
	switch snapshot.Status {
	case orquestaruntime.ProcessRuntimeRunningV0, orquestaruntime.ProcessRuntimeStoppingV0:
		liveness.LiveRefs = append(liveness.LiveRefs, agentRef)
	case orquestaruntime.ProcessRuntimeStoppedV0:
		liveness.StaleRefs = append(liveness.StaleRefs, agentRef)
		liveness.EvidenceRefs = append(liveness.EvidenceRefs,
			"shutdown-agent-stale-runtime-stopped-"+safeStackShutdownRefPartV0(agentRef))
	default:
		liveness.LiveRefs = append(liveness.LiveRefs, agentRef)
		liveness.EvidenceRefs = append(liveness.EvidenceRefs,
			"shutdown-agent-live-runtime-unknown-"+safeStackShutdownRefPartV0(agentRef))
	}
	return liveness
}

func normalizeStackShutdownAgentLivenessV0(
	liveness stackShutdownAgentLivenessV0,
) stackShutdownAgentLivenessV0 {
	liveness.LiveRefs = compactStringsV0(liveness.LiveRefs)
	liveness.StaleRefs = compactStringsV0(liveness.StaleRefs)
	liveness.LostRefs = compactStringsV0(liveness.LostRefs)
	liveness.EvidenceRefs = compactStringsV0(liveness.EvidenceRefs)
	return liveness
}
