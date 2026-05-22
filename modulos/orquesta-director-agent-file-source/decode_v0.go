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
	data = directorAgentDecisionJSONWithoutTrailingCommasV0(data)
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
	var envelope struct {
		SchemaVersion string            `json:"schema_version"`
		Decisions     []json.RawMessage `json:"decisions"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil, DirectorAgentFileSourceIssueV0{Field: "file"}
	}
	if strings.TrimSpace(envelope.SchemaVersion) != DirectorAgentDecisionFileSchemaVersionV0 {
		return nil, DirectorAgentFileSourceIssueV0{Field: "schema_version"}
	}
	if len(envelope.Decisions) == 0 {
		return nil, DirectorAgentFileSourceIssueV0{Field: "decisions"}
	}
	return decodeDirectorAgentDecisionRawListV0(envelope.Decisions)
}

func decodeSingleDirectorAgentDecisionV0(
	data []byte,
) ([]orquestadirectoragent.DirectorAgentDecisionV0, error) {
	decision, err := decodeDirectorAgentDecisionRawV0(data)
	if err != nil {
		return nil, DirectorAgentFileSourceIssueV0{Field: "decision"}
	}
	return []orquestadirectoragent.DirectorAgentDecisionV0{decision}, nil
}

func decodeDirectorAgentDecisionArrayV0(
	data []byte,
) ([]orquestadirectoragent.DirectorAgentDecisionV0, error) {
	var rawDecisions []json.RawMessage
	if err := json.Unmarshal(data, &rawDecisions); err != nil {
		return nil, DirectorAgentFileSourceIssueV0{Field: "file"}
	}
	if len(rawDecisions) == 0 {
		return nil, DirectorAgentFileSourceIssueV0{Field: "decisions"}
	}
	decisions, err := decodeDirectorAgentDecisionRawListV0(rawDecisions)
	if err != nil {
		return nil, err
	}
	return decisions, nil
}

func decodeDirectorAgentDecisionRawListV0(
	rawDecisions []json.RawMessage,
) ([]orquestadirectoragent.DirectorAgentDecisionV0, error) {
	decisions := make([]orquestadirectoragent.DirectorAgentDecisionV0, 0, len(rawDecisions))
	for _, raw := range rawDecisions {
		decision, err := decodeDirectorAgentDecisionRawV0(raw)
		if err != nil {
			return nil, DirectorAgentFileSourceIssueV0{Field: "decision"}
		}
		decisions = append(decisions, decision)
	}
	return decisions, nil
}

func decodeDirectorAgentDecisionRawV0(
	raw json.RawMessage,
) (orquestadirectoragent.DirectorAgentDecisionV0, error) {
	var decision orquestadirectoragent.DirectorAgentDecisionV0
	if err := json.Unmarshal(raw, &decision); err != nil {
		return orquestadirectoragent.DirectorAgentDecisionV0{}, err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return orquestadirectoragent.DirectorAgentDecisionV0{}, err
	}
	return hydrateDirectorAgentDecisionFlatPayloadV0(decision, fields), nil
}

func hydrateDirectorAgentDecisionFlatPayloadV0(
	decision orquestadirectoragent.DirectorAgentDecisionV0,
	fields map[string]json.RawMessage,
) orquestadirectoragent.DirectorAgentDecisionV0 {
	switch strings.TrimSpace(decision.CommandType) {
	case orquestadirectoragent.DirectorAgentCommandRequestVoteV0:
		if decision.RequestVote == nil && fields["request_vote"] == nil {
			var payload orquestadirectoragent.DirectorAgentVoteCommandV0
			if json.Unmarshal(mustDirectorAgentDecisionFlatPayloadRawV0(fields), &payload) == nil {
				decision.RequestVote = &payload
			}
		}
	case orquestadirectoragent.DirectorAgentCommandAcceptDecisionV0:
		if decision.AcceptDecision == nil && fields["accept_decision"] == nil {
			var payload orquestadirectoragent.DirectorAgentAcceptDecisionCommandV0
			if json.Unmarshal(mustDirectorAgentDecisionFlatPayloadRawV0(fields), &payload) == nil {
				decision.AcceptDecision = &payload
			}
		}
	case orquestadirectoragent.DirectorAgentCommandPublishContractV0:
		if decision.PublishContract == nil && fields["publish_function_contract"] == nil {
			var payload orquestadirectoragent.DirectorAgentPublishContractCommandV0
			if json.Unmarshal(mustDirectorAgentDecisionFlatPayloadRawV0(fields), &payload) == nil {
				decision.PublishContract = &payload
			}
		}
	}
	return decision
}

func mustDirectorAgentDecisionFlatPayloadRawV0(
	fields map[string]json.RawMessage,
) []byte {
	data, err := json.Marshal(fields)
	if err != nil {
		return nil
	}
	return data
}

func directorAgentDecisionJSONWithoutTrailingCommasV0(data []byte) []byte {
	out := make([]byte, 0, len(data))
	inString := false
	escaped := false
	for i, b := range data {
		if inString {
			out = append(out, b)
			if escaped {
				escaped = false
				continue
			}
			if b == '\\' {
				escaped = true
				continue
			}
			if b == '"' {
				inString = false
			}
			continue
		}
		if b == '"' {
			inString = true
			out = append(out, b)
			continue
		}
		if b == ',' {
			next := i + 1
			for next < len(data) && isDirectorAgentDecisionJSONWhitespaceV0(data[next]) {
				next++
			}
			if next < len(data) && (data[next] == '}' || data[next] == ']') {
				continue
			}
		}
		out = append(out, b)
	}
	return out
}

func isDirectorAgentDecisionJSONWhitespaceV0(b byte) bool {
	return b == ' ' || b == '\n' || b == '\r' || b == '\t'
}
