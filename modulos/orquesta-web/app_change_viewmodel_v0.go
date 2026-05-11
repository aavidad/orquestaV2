package orquestaweb

import (
	"strings"

	orquestamcp "orquesta/modulos/orquesta-mcp"
)

const (
	WebAppChangeEstadoInicialV0  = "initial"
	WebAppChangeEstadoAceptadoV0 = "accepted"
	WebAppChangeEstadoErrorV0    = "error"
)

type WebAppChangeViewModelV0 struct {
	Estado              string                `json:"estado"`
	RequestID           string                `json:"request_id,omitempty"`
	CorrelationID       string                `json:"correlation_id,omitempty"`
	RunRef              string                `json:"run_ref,omitempty"`
	AppRef              string                `json:"app_ref,omitempty"`
	ChangeRef           string                `json:"change_ref,omitempty"`
	DirectorQuestionRef string                `json:"director_question_ref,omitempty"`
	ErroresPublicos     []MCPAppChangeIssueV0 `json:"errores_publicos,omitempty"`
}

type MCPAppChangeIssueV0 = orquestamcp.MCPAppChangeIssueV0

func InitialWebAppChangeViewModelV0() WebAppChangeViewModelV0 {
	return WebAppChangeViewModelV0{Estado: WebAppChangeEstadoInicialV0}
}

func NewWebAppChangeViewModelV0(result orquestamcp.MCPRequestAppChangeToolResultV0) WebAppChangeViewModelV0 {
	estado := WebAppChangeEstadoAceptadoV0
	if strings.TrimSpace(result.Estado) != orquestamcp.MCPRequestAppChangeEstadoOKV0 {
		estado = WebAppChangeEstadoErrorV0
	}
	return WebAppChangeViewModelV0{
		Estado:              estado,
		RequestID:           strings.TrimSpace(result.RequestID),
		CorrelationID:       strings.TrimSpace(result.CorrelationID),
		RunRef:              strings.TrimSpace(result.RunRef),
		AppRef:              strings.TrimSpace(result.AppRef),
		ChangeRef:           strings.TrimSpace(result.ChangeRef),
		DirectorQuestionRef: strings.TrimSpace(result.DirectorQuestionRef),
		ErroresPublicos:     append([]MCPAppChangeIssueV0(nil), result.Errores...),
	}
}
