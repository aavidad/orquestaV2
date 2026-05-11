package orquestacoreworkflow

import (
	"encoding/json"
	"strings"
)

func decodeStartRunCommandPayloadV0(raw json.RawMessage) (StartRunCommandPayloadV0, error) {
	var payload StartRunCommandPayloadV0
	if err := json.Unmarshal(raw, &payload); err != nil {
		return StartRunCommandPayloadV0{}, commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	payload.ProjectRef = strings.TrimSpace(payload.ProjectRef)
	payload.AppSpecRef = strings.TrimSpace(payload.AppSpecRef)
	return payload, nil
}

func decodeOpenPhaseCommandPayloadV0(raw json.RawMessage) (OpenPhaseCommandPayloadV0, error) {
	var payload OpenPhaseCommandPayloadV0
	if err := json.Unmarshal(raw, &payload); err != nil {
		return OpenPhaseCommandPayloadV0{}, commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	payload.PhaseID = strings.TrimSpace(payload.PhaseID)
	payload.Reason = strings.TrimSpace(payload.Reason)
	return payload, nil
}

func decodeBlockRunCommandPayloadV0(raw json.RawMessage) (BlockRunCommandPayloadV0, error) {
	var payload BlockRunCommandPayloadV0
	if err := json.Unmarshal(raw, &payload); err != nil {
		return BlockRunCommandPayloadV0{}, commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	payload.BlockerID = strings.TrimSpace(payload.BlockerID)
	payload.ReasonCode = strings.TrimSpace(payload.ReasonCode)
	payload.Summary = strings.TrimSpace(payload.Summary)
	payload.SourceGroup = strings.TrimSpace(payload.SourceGroup)
	payload.EvidenceRefs = compactStringsV0(payload.EvidenceRefs)
	return payload, nil
}
