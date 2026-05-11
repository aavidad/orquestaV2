package orquestadirectoragentfilesource

import (
	"encoding/json"
	"strings"

	orquestadirectoragent "orquesta/modulos/orquesta-director-agent"
)

func DecodeDirectorAgentDecisionFileV0(
	data []byte,
) ([]orquestadirectoragent.DirectorAgentDecisionV0, error) {
	if len(data) == 0 {
		return nil, DirectorAgentFileSourceIssueV0{Field: "file"}
	}
	var head struct {
		SchemaVersion string `json:"schema_version"`
	}
	if err := json.Unmarshal(data, &head); err == nil {
		switch strings.TrimSpace(head.SchemaVersion) {
		case DirectorAgentDecisionFileSchemaVersionV0:
			return decodeDirectorAgentDecisionEnvelopeV0(data)
		case orquestadirectoragent.DirectorAgentDecisionSchemaVersionV0:
			return decodeSingleDirectorAgentDecisionV0(data)
		case "":
			return decodeDirectorAgentDecisionArrayV0(data)
		default:
			return nil, DirectorAgentFileSourceIssueV0{Field: "schema_version"}
		}
	}
	return decodeDirectorAgentDecisionArrayV0(data)
}

func decodeDirectorAgentDecisionEnvelopeV0(
	data []byte,
) ([]orquestadirectoragent.DirectorAgentDecisionV0, error) {
	var envelope DirectorAgentDecisionFileEnvelopeV0
	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil, DirectorAgentFileSourceIssueV0{Field: "file"}
	}
	if strings.TrimSpace(envelope.SchemaVersion) != DirectorAgentDecisionFileSchemaVersionV0 {
		return nil, DirectorAgentFileSourceIssueV0{Field: "schema_version"}
	}
	if len(envelope.Decisions) == 0 {
		return nil, DirectorAgentFileSourceIssueV0{Field: "decisions"}
	}
	return envelope.Decisions, nil
}

func decodeSingleDirectorAgentDecisionV0(
	data []byte,
) ([]orquestadirectoragent.DirectorAgentDecisionV0, error) {
	var decision orquestadirectoragent.DirectorAgentDecisionV0
	if err := json.Unmarshal(data, &decision); err != nil {
		return nil, DirectorAgentFileSourceIssueV0{Field: "decision"}
	}
	return []orquestadirectoragent.DirectorAgentDecisionV0{decision}, nil
}

func decodeDirectorAgentDecisionArrayV0(
	data []byte,
) ([]orquestadirectoragent.DirectorAgentDecisionV0, error) {
	var decisions []orquestadirectoragent.DirectorAgentDecisionV0
	if err := json.Unmarshal(data, &decisions); err != nil {
		return nil, DirectorAgentFileSourceIssueV0{Field: "file"}
	}
	if len(decisions) == 0 {
		return nil, DirectorAgentFileSourceIssueV0{Field: "decisions"}
	}
	return decisions, nil
}
