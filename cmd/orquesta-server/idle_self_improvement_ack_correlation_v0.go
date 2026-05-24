package main

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
	orquestaserver "orquesta/modulos/orquesta-server"
)

const (
	idleSelfImprovementAckStructuredEvidenceRefV0 = "evidence-ref-autoprogramming-backlog-ack-structured"
	idleSelfImprovementAckAmbiguousEvidenceRefV0  = "evidence-ref-autoprogramming-backlog-ack-ambiguous"
)

func (planner idleSelfImprovementBacklogPlannerV0) completedBacklogRequestRefsFromRuntimeV0(
	projectDir string,
) (map[string]bool, []string, []orquestaserver.BacklogScanCollisionV0) {
	runtimeDir := filepath.Join(projectDir, ".orquesta-runtime")
	completed := map[string]bool{}
	evidenceRefs := []string{}
	collisions := []orquestaserver.BacklogScanCollisionV0{}
	_ = filepath.WalkDir(runtimeDir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry == nil || entry.IsDir() || entry.Name() != "agent_ack.json" {
			return nil
		}
		runRef := backlogRunRefFromRuntimeACKPathV0(runtimeDir, path)
		if runRef == "" {
			return nil
		}
		agentRef := backlogAgentRefFromRuntimeACKPathV0(runtimeDir, path)
		ok, stale := planner.structuredACKCompletedAndCurrentV0(path, runRef, agentRef)
		if ok {
			completed[idleSelfImprovementNormalizeQueuedRequestRefV0(runRef)] = true
			evidenceRefs = append(evidenceRefs, idleSelfImprovementAckStructuredEvidenceRefV0)
			return nil
		}
		if stale {
			collisions = append(collisions, orquestaserver.BacklogScanCollisionV0{
				Code:       "backlog_docs_changed_after_plan",
				RequestRef: idleSelfImprovementNormalizeQueuedRequestRefV0(runRef),
				Message:    "ACK completado sobre foto documental obsoleta; requiere rebase/merge",
				EvidenceRefs: []string{
					"evidence-ref-autoprogramming-backlog-doc-merge-pending",
				},
			})
			evidenceRefs = append(evidenceRefs, "evidence-ref-autoprogramming-backlog-doc-merge-pending")
			return nil
		}
		evidenceRefs = append(evidenceRefs, idleSelfImprovementAckAmbiguousEvidenceRefV0)
		return nil
	})
	return completed, compactServerStackStringsV0(evidenceRefs), compactBacklogScanCollisionsV0(collisions)
}

func (planner idleSelfImprovementBacklogPlannerV0) structuredACKCompletedAndCurrentV0(
	ackPath string,
	runRef string,
	agentRef string,
) (bool, bool) {
	packet, ok := idleSelfImprovementReadAgentPacketBesideACKV0(ackPath)
	if !ok || !idleSelfImprovementPacketMatchesBacklogRuntimeV0(packet, runRef, agentRef) {
		return false, false
	}
	data, err := os.ReadFile(ackPath)
	if err != nil {
		return false, false
	}
	spec := orquestaruntime.ExternalAgentLaunchSpecV0{
		SchemaVersion: orquestaruntime.ExternalAgentLaunchSpecSchemaVersionV0,
		RequestID:     packet.RequestID,
		CorrelationID: packet.CorrelationID,
		AgentPacket:   packet,
	}
	ack, issues := orquestaruntimecodex.ValidateStrictCompletedCodexAgentAckBytesForSpecV0(data, spec)
	if len(issues) != 0 || strings.TrimSpace(ack.Status) != "completed" {
		return false, false
	}
	if !planner.packetBacklogDocsCurrentV0(packet) {
		return false, true
	}
	return true, false
}

func idleSelfImprovementReadAgentPacketBesideACKV0(
	ackPath string,
) (orquestaruntime.AgentStartPacketV0, bool) {
	data, err := os.ReadFile(filepath.Join(filepath.Dir(ackPath), "agent_packet.json"))
	if err != nil {
		return orquestaruntime.AgentStartPacketV0{}, false
	}
	var packet orquestaruntime.AgentStartPacketV0
	if err := json.Unmarshal(data, &packet); err != nil {
		return orquestaruntime.AgentStartPacketV0{}, false
	}
	return packet, idleSelfImprovementPacketHasACKFieldsV0(packet)
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

func backlogAgentRefFromRuntimeACKPathV0(runtimeDir string, ackPath string) string {
	rel, err := filepath.Rel(runtimeDir, ackPath)
	if err != nil {
		return ""
	}
	parts := strings.Split(filepath.ToSlash(rel), "/")
	if len(parts) < 3 {
		return ""
	}
	return strings.TrimSpace(parts[1])
}
