package orquestaruntimecodexdelivery

import (
	"errors"
	"strings"

	orquestaagentprocessregistry "orquesta/modulos/orquesta-agent-process-registry"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func (source CodexProgressObservationSourceV0) snapshotV0(
	record orquestacionnucleoapp.AgentProcessRegistryRecordV0,
) (orquestaruntime.ProcessRuntimeSnapshotV0, error) {
	if source.SnapshotSource != nil {
		snapshot, err := source.SnapshotSource.SnapshotV0(record.ProcessRef)
		if err == nil {
			return snapshot, nil
		}
		if codexProgressSnapshotMissingV0(err) {
			return stoppedCodexProgressSnapshotV0(record), nil
		}
		return orquestaruntime.ProcessRuntimeSnapshotV0{}, err
	}
	return orquestaruntime.ProcessRuntimeSnapshotV0{
		SchemaVersion: orquestaruntime.ProcessRuntimeConnectorVersionV0,
		ProcessRef:    strings.TrimSpace(record.ProcessRef),
		SessionRef:    strings.TrimSpace(record.SessionRef),
		LaunchRef:     strings.TrimSpace(record.LaunchRef),
		Status:        orquestaruntime.ProcessRuntimeRunningV0,
	}, nil
}

func codexProgressSnapshotMissingV0(err error) bool {
	var runtimeErr orquestaruntime.ProcessRuntimeErrorV0
	return errors.As(err, &runtimeErr) &&
		runtimeErr.Code == orquestaruntime.ProcessRuntimeNoEncontradoV0
}

func codexProgressAgentProcessMissingV0(err error) bool {
	var registryErr orquestaagentprocessregistry.ErrorV0
	if errors.As(err, &registryErr) {
		return registryErr.Code == orquestaagentprocessregistry.ErrAgentProcessRegistryNotFoundV0
	}
	var coreErr orquestacionnucleoapp.ErrorV0
	if !errors.As(err, &coreErr) {
		return false
	}
	return coreErr.Code == orquestacionnucleoapp.ErrNucleoOrquestacionStoreV0 &&
		strings.TrimSpace(coreErr.Field) == "agent_process_registry" &&
		strings.Contains(strings.ToLower(coreErr.Message), "no encontrado")
}

func stoppedCodexProgressSnapshotV0(
	record orquestacionnucleoapp.AgentProcessRegistryRecordV0,
) orquestaruntime.ProcessRuntimeSnapshotV0 {
	return orquestaruntime.ProcessRuntimeSnapshotV0{
		SchemaVersion: orquestaruntime.ProcessRuntimeConnectorVersionV0,
		ProcessRef:    strings.TrimSpace(record.ProcessRef),
		SessionRef:    strings.TrimSpace(record.SessionRef),
		LaunchRef:     strings.TrimSpace(record.LaunchRef),
		Status:        orquestaruntime.ProcessRuntimeStoppedV0,
	}
}
