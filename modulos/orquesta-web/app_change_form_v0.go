package orquestaweb

import (
	"strings"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
	orquestamcp "orquesta/modulos/orquesta-mcp"
)

type WebAppChangeFormV0 struct {
	RequestID          string   `json:"request_id,omitempty"`
	CorrelationID      string   `json:"correlation_id,omitempty"`
	Locale             string   `json:"locale,omitempty"`
	RunRef             string   `json:"run_ref"`
	AppRef             string   `json:"app_ref,omitempty"`
	ChangeRef          string   `json:"change_ref"`
	UserIntent         string   `json:"user_intent"`
	TargetArea         string   `json:"target_area,omitempty"`
	CurrentStateRefs   []string `json:"current_state_refs,omitempty"`
	AcceptanceCriteria []string `json:"acceptance_criteria,omitempty"`
	Constraints        []string `json:"constraints,omitempty"`
	AllowedWriteSet    []string `json:"allowed_write_set,omitempty"`
}

func (form WebAppChangeFormV0) ToMCPRequestAppChangeInputV0() orquestamcp.MCPRequestAppChangeToolInputV0 {
	request := orquestaappchange.AppChangeRequestV0{
		SchemaVersion:      orquestaappchange.AppChangeRequestSchemaV0,
		RequestID:          strings.TrimSpace(form.RequestID),
		CorrelationID:      strings.TrimSpace(form.CorrelationID),
		RunRef:             strings.TrimSpace(form.RunRef),
		AppRef:             strings.TrimSpace(form.AppRef),
		ChangeRef:          strings.TrimSpace(form.ChangeRef),
		Locale:             strings.TrimSpace(form.Locale),
		UserIntent:         strings.TrimSpace(form.UserIntent),
		TargetArea:         strings.TrimSpace(form.TargetArea),
		CurrentStateRefs:   compactStringsV0(form.CurrentStateRefs),
		AcceptanceCriteria: compactStringsV0(form.AcceptanceCriteria),
		Constraints:        compactStringsV0(form.Constraints),
		AllowedWriteSet:    compactStringsV0(form.AllowedWriteSet),
	}
	return orquestamcp.MCPRequestAppChangeToolInputV0{
		RequestID:        request.RequestID,
		CorrelationID:    request.CorrelationID,
		AppChangeRequest: request,
	}
}
