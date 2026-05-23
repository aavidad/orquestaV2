package orquestacli

import (
	"errors"
	"net/http"
	"strings"
	"time"

	orquestamcp "orquesta/modulos/orquesta-mcp"
)

func autoprogClientErrorEnvelopeV0(inv CliInvocationContextV0, err error, start time.Time) CliOutputEnvelopeV0 {
	var cliErr CliClientErrorV0
	if errors.As(err, &cliErr) {
		return autoprogSingleErrorEnvelopeV0(inv, cliErr.Code, cliErr.Field, cliErr.Detail, cliErr.StatusCode, cliErr.Retryable, start)
	}
	return autoprogSingleErrorEnvelopeV0(inv, CliErrConfiguracionInvalidaV0, "server_url", "server_url_invalida", 0, false, start)
}

func autoprogTransportEnvelopeV0(inv CliInvocationContextV0, err error, start time.Time) CliOutputEnvelopeV0 {
	detail := "request_failed"
	if isTimeoutErrorV0(err) {
		detail = "timeout"
	}
	return autoprogSingleErrorEnvelopeV0(inv, CliErrErrorTransporteV0, "transporte", detail, 0, true, start)
}

func autoprogSingleErrorEnvelopeV0(inv CliInvocationContextV0, code, field, detail string, statusCode int, retryable bool, start time.Time) CliOutputEnvelopeV0 {
	return NewCliOutputErrorEnvelopeV0(
		inv,
		CliContractAutoprogrammingV0,
		CliContractVersionAutoprogrammingV0,
		[]CliPublicErrorV0{NewCliPublicErrorV0(code, field, detail)},
		cliMetaV0(start, statusCode, retryable),
	)
}

func httpClientAutoprogrammingV0(client *AutoprogrammingCliClientV0, timeout time.Duration) *http.Client {
	if client == nil {
		return httpClientFromConfigV0(cliRESTClientConfigV0{}, timeout)
	}
	return httpClientFromConfigV0(cliRESTClientConfigV0{HTTPClient: client.HTTPClient}, timeout)
}

func clientTimeoutAutoprogrammingV0(client *AutoprogrammingCliClientV0) time.Duration {
	if client == nil {
		return clientTimeoutFromConfigV0(cliRESTClientConfigV0{})
	}
	return clientTimeoutFromConfigV0(cliRESTClientConfigV0{Timeout: client.Timeout})
}

func autoprogIssuesToCliV0(issues []orquestamcp.MCPValidationIssueV0) []CliPublicErrorV0 {
	if len(issues) == 0 {
		return []CliPublicErrorV0{NewCliPublicErrorV0(CliErrRespuestaInvalidaV0, "errores_publicos", "autoprogramming_error_sin_errores_publicos")}
	}
	out := make([]CliPublicErrorV0, 0, len(issues))
	for _, issue := range issues {
		code := strings.TrimSpace(issue.Code)
		if code == "" {
			code = CliErrRespuestaInvalidaV0
		}
		out = append(out, CliPublicErrorV0{
			Codigo:      code,
			Campo:       strings.TrimSpace(issue.Field),
			MensajeI18N: CliDefaultErrorMessageNamespaceV0 + code,
			Detalle:     strings.TrimSpace(issue.Message),
		})
	}
	return out
}

func autoprogEstadoAndIssuesV0(target any) (string, []orquestamcp.MCPValidationIssueV0) {
	switch result := target.(type) {
	case *orquestamcp.MCPAutoprogrammingPrepareRunToolResultV0:
		return strings.TrimSpace(result.Estado), result.Errores
	case *orquestamcp.MCPRunQueuePriorityToolResultV0:
		return strings.TrimSpace(result.Estado), result.Errores
	case *orquestamcp.MCPAutoprogrammingStatusToolResultV0:
		return strings.TrimSpace(result.Estado), result.Errores
	case *orquestamcp.MCPDirectorStatsToolResultV0:
		return strings.TrimSpace(result.Estado), result.Errores
	case *orquestamcp.MCPRunSupervisorToolResultV0:
		return strings.TrimSpace(result.Estado), result.Errores
	default:
		return "", nil
	}
}

func firstNonEmptyCliStringV0(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
