package orquestamcp

import (
	"encoding/json"
	"strings"

	orquestaappdirectorintake "orquesta/modulos/orquesta-app-director-intake"
)

func (input *MCPHumanDirectorWorkReviewPlanToolInputV0) UnmarshalJSON(raw []byte) error {
	type alias MCPHumanDirectorWorkReviewPlanToolInputV0
	var envelope struct {
		alias
		WorkIntake json.RawMessage `json:"work_intake"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return err
	}
	*input = MCPHumanDirectorWorkReviewPlanToolInputV0(envelope.alias)
	if len(envelope.WorkIntake) == 0 || strings.TrimSpace(string(envelope.WorkIntake)) == "null" {
		return nil
	}
	workIntake, err := decodeMCPHumanDirectorWorkIntakeV0(envelope.WorkIntake)
	if err != nil {
		return err
	}
	input.WorkIntake = workIntake
	return nil
}

func decodeMCPHumanDirectorWorkIntakeV0(
	raw json.RawMessage,
) (orquestaappdirectorintake.HumanDirectorWorkIntakeRequestV0, error) {
	trimmed := strings.TrimSpace(string(raw))
	if strings.HasPrefix(trimmed, "\"") {
		var text string
		if err := json.Unmarshal(raw, &text); err != nil {
			return orquestaappdirectorintake.HumanDirectorWorkIntakeRequestV0{}, err
		}
		text = strings.TrimSpace(text)
		if text == "" {
			return orquestaappdirectorintake.HumanDirectorWorkIntakeRequestV0{}, nil
		}
		return orquestaappdirectorintake.HumanDirectorWorkIntakeRequestV0{
			Request: orquestaappdirectorintake.HumanDirectorWorkRequestV0{
				Title:     text,
				Objective: text,
			},
		}, nil
	}
	var request orquestaappdirectorintake.HumanDirectorWorkIntakeRequestV0
	if err := json.Unmarshal(raw, &request); err != nil {
		return orquestaappdirectorintake.HumanDirectorWorkIntakeRequestV0{}, err
	}
	return request, nil
}

func humanDirectorWorkIntakeFromMCPV0(
	input MCPHumanDirectorWorkReviewPlanToolInputV0,
) orquestaappdirectorintake.HumanDirectorWorkIntakeRequestV0 {
	request := input.WorkIntake
	if strings.TrimSpace(request.SchemaVersion) == "" {
		request.SchemaVersion = orquestaappdirectorintake.HumanDirectorWorkIntakeSchemaVersionV0
	}
	if strings.TrimSpace(request.RequestRef) == "" {
		request.RequestRef = firstNonEmptyMCPV0(input.RequestID, input.CorrelationID)
	}
	if strings.TrimSpace(request.CorrelationID) == "" {
		request.CorrelationID = firstNonEmptyMCPV0(input.CorrelationID, input.RequestID, request.RequestRef)
	}
	if strings.TrimSpace(request.ProjectRef) == "" {
		request.ProjectRef = "project-ref-" + mcpHumanDirectorSafeRefPartV0(firstNonEmptyMCPV0(request.RequestRef, input.RequestID, input.CorrelationID, "human-work"))
	}
	return request
}

func mcpHumanDirectorSafeRefPartV0(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var builder strings.Builder
	lastDash := false
	for _, r := range value {
		allowed := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if allowed {
			builder.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash && builder.Len() > 0 {
			builder.WriteRune('-')
			lastDash = true
		}
	}
	out := strings.Trim(builder.String(), "-")
	if out == "" {
		return "human-work"
	}
	return out
}
