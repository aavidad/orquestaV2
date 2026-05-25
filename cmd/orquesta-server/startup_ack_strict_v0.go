package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
)

func startupRunHasStrictCompletedACKV0(runtimeDir string, runRef string) bool {
	if strings.TrimSpace(runtimeDir) == "" || strings.TrimSpace(runRef) == "" {
		return false
	}
	root := filepath.Join(runtimeDir, runRef)
	entries, err := os.ReadDir(root)
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if !entry.IsDir() || !startupAgentHasStrictCompletedACKV0(filepath.Join(root, entry.Name())) {
			continue
		}
		return true
	}
	return false
}

func startupAgentHasStrictCompletedACKV0(agentDir string) bool {
	packet, ok := startupReadAgentPacketV0(filepath.Join(agentDir, "agent_packet.json"))
	if !ok {
		return false
	}
	spec := orquestaruntime.ExternalAgentLaunchSpecV0{
		SchemaVersion: orquestaruntime.ExternalAgentLaunchSpecSchemaVersionV0,
		RequestID:     packet.RequestID,
		CorrelationID: packet.CorrelationID,
		AgentPacket:   packet,
	}
	ack, issues := orquestaruntimecodex.ReadAndValidateStrictCompletedCodexAgentAckFileV0(
		filepath.Join(agentDir, "agent_ack.json"),
		spec,
	)
	return len(issues) == 0 && strings.TrimSpace(ack.Status) == "completed"
}

func startupReadAgentPacketV0(path string) (orquestaruntime.AgentStartPacketV0, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return orquestaruntime.AgentStartPacketV0{}, false
	}
	var packet orquestaruntime.AgentStartPacketV0
	if err := json.Unmarshal(data, &packet); err != nil {
		return orquestaruntime.AgentStartPacketV0{}, false
	}
	return packet, startupAgentPacketHasStrictACKRefsV0(packet)
}

func startupAgentPacketHasStrictACKRefsV0(packet orquestaruntime.AgentStartPacketV0) bool {
	return packet.SchemaVersion == orquestaruntime.AgentStartPacketSchemaVersionV0 &&
		strings.TrimSpace(packet.RequestID) != "" &&
		strings.TrimSpace(packet.CorrelationID) != "" &&
		strings.TrimSpace(packet.TargetModule) != "" &&
		strings.TrimSpace(packet.Task.TaskRef) != "" &&
		strings.TrimSpace(packet.DeliveryRefs.AckRef) != ""
}
