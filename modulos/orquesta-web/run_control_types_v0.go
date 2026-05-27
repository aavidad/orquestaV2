package orquestaweb

import (
	"strings"

	orquestamcp "orquesta/modulos/orquesta-mcp"
	publicidentity "orquesta/modulos/orquesta-server/publicidentity"
)

const (
	WebRunControlPageEndpointV0    = "/run-control"
	WebRunControlInboundEndpointV0 = "/api/v0/runs/control"

	WebRunControlEstadoOKV0     = "ok"
	WebRunControlEstadoErrorV0  = "error"
	WebRunControlActionPauseV0  = "pause"
	WebRunControlActionResumeV0 = "resume"
	WebRunControlActionStopV0   = "stop"
	WebRunControlActionCancelV0 = "cancel"

	WebRunControlErrTransporteV0        = "run_control_error_transporte"
	WebRunControlErrRespuestaInvalidaV0 = "run_control_respuesta_invalida"
)

type WebRunControlCommandV0 struct {
	RequestID      string   `json:"request_id,omitempty"`
	CorrelationID  string   `json:"correlation_id,omitempty"`
	Locale         string   `json:"locale,omitempty"`
	Action         string   `json:"action"`
	RunRef         string   `json:"run_ref"`
	RequestedBy    string   `json:"requested_by,omitempty"`
	Reason         string   `json:"reason,omitempty"`
	Forced         bool     `json:"forced,omitempty"`
	IdempotencyKey string   `json:"idempotency_key,omitempty"`
	EvidenceRefs   []string `json:"evidence_refs,omitempty"`
}

type WebRunControlViewModelV0 struct {
	SchemaVersion      string                       `json:"schema_version"`
	Locale             string                       `json:"locale,omitempty"`
	Estado             string                       `json:"estado"`
	Action             string                       `json:"action,omitempty"`
	RunRef             string                       `json:"run_ref,omitempty"`
	Status             string                       `json:"status,omitempty"`
	CheckpointRecorded bool                         `json:"checkpoint_recorded,omitempty"`
	Forced             bool                         `json:"forced,omitempty"`
	EvidenceRefs       []string                     `json:"evidence_refs,omitempty"`
	ErroresPublicos    []WebRunControlPublicIssueV0 `json:"errores_publicos,omitempty"`
}

type WebRunControlPublicIssueV0 struct {
	Code    string `json:"code"`
	Field   string `json:"field,omitempty"`
	Message string `json:"message,omitempty"`
}

func NewWebRunControlViewModelV0(
	locale string,
	result orquestamcp.MCPRunControlToolResultV0,
) WebRunControlViewModelV0 {
	return WebRunControlViewModelV0{
		SchemaVersion:      "web_run_control.v0",
		Locale:             normalizeDirectorStatsLocaleV0(locale),
		Estado:             runControlEstadoV0(result.Estado),
		Action:             trimV0(result.Action),
		RunRef:             trimV0(result.RunRef),
		Status:             trimV0(result.Status),
		CheckpointRecorded: result.CheckpointRecorded,
		Forced:             result.Forced,
		EvidenceRefs:       compactStringsV0(result.EvidenceRefs),
		ErroresPublicos:    webRunControlIssuesV0(result.Errores),
	}
}

func NewWebRunControlErrorViewModelV0(locale string, code string) WebRunControlViewModelV0 {
	return WebRunControlViewModelV0{
		SchemaVersion: "web_run_control.v0",
		Locale:        normalizeDirectorStatsLocaleV0(locale),
		Estado:        WebRunControlEstadoErrorV0,
		ErroresPublicos: []WebRunControlPublicIssueV0{{
			Code:    trimV0(code),
			Message: trimV0(code),
		}},
	}
}

func normalizeRunControlCommandV0(command WebRunControlCommandV0) WebRunControlCommandV0 {
	command.RequestID = trimV0(command.RequestID)
	if command.RequestID == "" {
		command.RequestID = newWebRequestIDV0()
	}
	command.CorrelationID = firstDirectorStatsNonEmptyV0(command.CorrelationID, command.RequestID)
	identity := publicidentity.NormalizePublicIdentityV0(publicidentity.PublicIdentityInputV0{
		RequestID:      command.RequestID,
		CorrelationID:  command.CorrelationID,
		IdempotencyKey: command.IdempotencyKey,
		Mutating:       true,
	})
	command.CorrelationID = identity.CorrelationID
	command.IdempotencyKey = identity.IdempotencyKey
	command.Locale = normalizeDirectorStatsLocaleV0(command.Locale)
	command.Action = strings.ToLower(trimV0(command.Action))
	command.RunRef = trimV0(command.RunRef)
	command.RequestedBy = trimV0(command.RequestedBy)
	command.Reason = trimV0(command.Reason)
	command.EvidenceRefs = compactStringsV0(command.EvidenceRefs)
	return command
}

func runControlToolInputV0(command WebRunControlCommandV0) orquestamcp.MCPRunControlToolInputV0 {
	return orquestamcp.MCPRunControlToolInputV0{
		RequestID:      command.RequestID,
		CorrelationID:  command.CorrelationID,
		Action:         command.Action,
		RunRef:         command.RunRef,
		RequestedBy:    command.RequestedBy,
		Reason:         command.Reason,
		Forced:         command.Forced,
		IdempotencyKey: command.IdempotencyKey,
		EvidenceRefs:   command.EvidenceRefs,
	}
}

func runControlEstadoV0(value string) string {
	if trimV0(value) == WebRunControlEstadoOKV0 {
		return WebRunControlEstadoOKV0
	}
	return WebRunControlEstadoErrorV0
}

func webRunControlIssuesV0(values []orquestamcp.MCPValidationIssueV0) []WebRunControlPublicIssueV0 {
	out := make([]WebRunControlPublicIssueV0, 0, len(values))
	for _, value := range values {
		out = append(out, WebRunControlPublicIssueV0{
			Code:    trimV0(value.Code),
			Field:   trimV0(value.Field),
			Message: trimV0(value.Message),
		})
	}
	return out
}
