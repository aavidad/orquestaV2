package main

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
	orquestaserver "orquesta/modulos/orquesta-server"
)

const (
	idleSelfImprovementAckStructuredEvidenceRefV0 = "evidence-ref-autoprogramming-backlog-ack-structured"
	idleSelfImprovementAckAmbiguousEvidenceRefV0  = "evidence-ref-autoprogramming-backlog-ack-ambiguous"
	idleSelfImprovementAckScanBudgetEvidenceRefV0 = "evidence-ref-autoprogramming-backlog-ack-scan-budget-exhausted"

	idleSelfImprovementAckScanMaxRuntimeDirsV0      = 256
	idleSelfImprovementAckScanMaxRunEntriesV0       = 128
	idleSelfImprovementAckScanMaxAgentDirsPerRunV0  = 64
	idleSelfImprovementAckScanMaxControlFileBytesV0 = 128 * 1024
	idleSelfImprovementAckScanMaxDurationV0         = 2 * time.Second
)

func (planner idleSelfImprovementBacklogPlannerV0) completedBacklogRequestRefsFromRuntimeV0(
	projectDir string,
) (map[string]bool, []string, []orquestaserver.BacklogScanCollisionV0) {
	runtimeDir := filepath.Join(projectDir, ".orquesta-runtime")
	completed := map[string]bool{}
	evidenceRefs := []string{}
	collisions := []orquestaserver.BacklogScanCollisionV0{}
	scan := idleSelfImprovementACKRuntimeScanV0{
		runtimeDir: runtimeDir,
		deadline:   time.Now().Add(idleSelfImprovementAckScanMaxDurationV0),
	}
	for _, candidate := range scan.runtimeACKCandidatesV0() {
		ok, stale, ambiguous := planner.structuredACKCompletedAndCurrentV0(candidate.ackPath, candidate.runRef, candidate.agentRef)
		if ok {
			completed[idleSelfImprovementNormalizeQueuedRequestRefV0(candidate.runRef)] = true
			evidenceRefs = append(evidenceRefs, idleSelfImprovementAckStructuredEvidenceRefV0)
			continue
		}
		if stale {
			collisions = append(collisions, idleSelfImprovementBacklogACKDocStaleCollisionV0(candidate.runRef))
			evidenceRefs = append(evidenceRefs, "evidence-ref-autoprogramming-backlog-doc-merge-pending")
			continue
		}
		if ambiguous {
			collisions = append(collisions, idleSelfImprovementBacklogACKAmbiguousCollisionV0(candidate.runRef))
		}
		evidenceRefs = append(evidenceRefs, idleSelfImprovementAckAmbiguousEvidenceRefV0)
	}
	for _, runRef := range compactServerStackStringsV0(scan.ambiguousRunRefs) {
		collisions = append(collisions, idleSelfImprovementBacklogACKAmbiguousCollisionV0(runRef))
		evidenceRefs = append(evidenceRefs, idleSelfImprovementAckAmbiguousEvidenceRefV0)
	}
	if scan.exhausted {
		collisions = append(collisions, idleSelfImprovementBacklogACKScanBudgetCollisionV0())
		evidenceRefs = append(evidenceRefs, idleSelfImprovementAckScanBudgetEvidenceRefV0)
	}
	return completed, compactServerStackStringsV0(evidenceRefs), compactBacklogScanCollisionsV0(collisions)
}

func (planner idleSelfImprovementBacklogPlannerV0) structuredACKCompletedAndCurrentV0(
	ackPath string,
	runRef string,
	agentRef string,
) (bool, bool, bool) {
	packet, ok := idleSelfImprovementReadAgentPacketBesideACKV0(ackPath)
	if !ok || !idleSelfImprovementPacketMatchesBacklogRuntimeV0(packet, runRef, agentRef) {
		return false, false, false
	}
	data, oversized, err := idleSelfImprovementReadBoundedControlFileV0(ackPath)
	if err != nil || oversized {
		return false, false, true
	}
	spec := orquestaruntime.ExternalAgentLaunchSpecV0{
		SchemaVersion: orquestaruntime.ExternalAgentLaunchSpecSchemaVersionV0,
		RequestID:     packet.RequestID,
		CorrelationID: packet.CorrelationID,
		AgentPacket:   packet,
	}
	ack, issues := orquestaruntimecodex.ValidateStrictCompletedCodexAgentAckBytesForSpecV0(data, spec)
	if len(issues) != 0 || strings.TrimSpace(ack.Status) != "completed" {
		return false, false, false
	}
	if !planner.packetBacklogDocsCurrentV0(packet) {
		return false, true, false
	}
	return true, false, false
}

func idleSelfImprovementReadAgentPacketBesideACKV0(
	ackPath string,
) (orquestaruntime.AgentStartPacketV0, bool) {
	data, oversized, err := idleSelfImprovementReadBoundedControlFileV0(filepath.Join(filepath.Dir(ackPath), "agent_packet.json"))
	if err != nil || oversized {
		return orquestaruntime.AgentStartPacketV0{}, false
	}
	var packet orquestaruntime.AgentStartPacketV0
	if err := json.Unmarshal(data, &packet); err != nil {
		return orquestaruntime.AgentStartPacketV0{}, false
	}
	return packet, idleSelfImprovementPacketHasACKFieldsV0(packet)
}

func idleSelfImprovementReadBoundedControlFileV0(path string) ([]byte, bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, false, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, idleSelfImprovementAckScanMaxControlFileBytesV0+1))
	if err != nil {
		return nil, false, err
	}
	if len(data) > idleSelfImprovementAckScanMaxControlFileBytesV0 {
		return nil, true, nil
	}
	return data, false, nil
}

func idleSelfImprovementPacketHasACKFieldsV0(packet orquestaruntime.AgentStartPacketV0) bool {
	return packet.SchemaVersion == orquestaruntime.AgentStartPacketSchemaVersionV0 &&
		strings.TrimSpace(packet.RequestID) != "" &&
		strings.TrimSpace(packet.CorrelationID) != "" &&
		strings.TrimSpace(packet.TargetModule) != "" &&
		strings.TrimSpace(packet.Task.TaskRef) != "" &&
		strings.TrimSpace(packet.DeliveryRefs.AckRef) != ""
}

func idleSelfImprovementPacketMatchesBacklogRuntimeV0(
	packet orquestaruntime.AgentStartPacketV0,
	runRef string,
	agentRef string,
) bool {
	normalizedRunRef := idleSelfImprovementNormalizeQueuedRequestRefV0(runRef)
	return normalizedRunRef != "" &&
		strings.Contains(packet.CorrelationID, normalizedRunRef) &&
		(agentRef == "" || packet.RequestID == agentRef)
}
