package orquestacionnucleoapp

import (
	"errors"

	orquestaagentprocessregistry "orquesta/modulos/orquesta-agent-process-registry"
)

func NormalizeAgentProcessRegistryRecordV0(
	record AgentProcessRecordV0,
) AgentProcessRecordV0 {
	return orquestaagentprocessregistry.NormalizeAgentProcessRegistryRecordV0(record)
}

func ValidateAgentProcessRegistryRecordV0(
	record AgentProcessRecordV0,
) error {
	return mapAgentProcessRegistryValidationErrorV0(
		orquestaagentprocessregistry.ValidateAgentProcessRegistryRecordV0(record),
	)
}

func NormalizeAgentProcessRegistryLookupV0(
	runID string,
	agentRequestID string,
) (string, string) {
	return orquestaagentprocessregistry.NormalizeAgentProcessRegistryLookupV0(
		runID,
		agentRequestID,
	)
}

func ValidateAgentProcessRegistryLookupV0(
	runID string,
	agentRequestID string,
) error {
	return mapAgentProcessRegistryValidationErrorV0(
		orquestaagentprocessregistry.ValidateAgentProcessRegistryLookupV0(runID, agentRequestID),
	)
}

func NormalizeAgentProcessRegistryListFilterV0(
	filter AgentProcessRegistryListFilterV0,
) AgentProcessRegistryListFilterV0 {
	return orquestaagentprocessregistry.NormalizeAgentProcessRegistryListFilterV0(filter)
}

func ValidateAgentProcessRegistryListFilterV0(
	filter AgentProcessRegistryListFilterV0,
) error {
	return mapAgentProcessRegistryValidationErrorV0(
		orquestaagentprocessregistry.ValidateAgentProcessRegistryListFilterV0(filter),
	)
}

func mapAgentProcessRegistryValidationErrorV0(err error) error {
	if err == nil {
		return nil
	}
	var registryErr orquestaagentprocessregistry.ErrorV0
	if errors.As(err, &registryErr) {
		return errorV0(
			ErrNucleoOrquestacionInvalidoV0,
			registryErr.Field,
			registryErr.Message,
		)
	}
	return err
}
