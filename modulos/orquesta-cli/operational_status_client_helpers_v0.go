package orquestacli

import (
	"errors"
	"net/http"
	"strings"
	"time"

	orquestaobservability "orquesta/modulos/orquesta-observability"
)

func clientErrorOperationalEnvelopeV0(inv CliInvocationContextV0, err error, start time.Time) CliOutputEnvelopeV0 {
	var cliErr CliClientErrorV0
	if errors.As(err, &cliErr) {
		return cliSingleOperationalErrorEnvelopeV0(inv, cliErr.Code, cliErr.Field, cliErr.Detail, cliErr.StatusCode, cliErr.Retryable, start)
	}
	return cliSingleOperationalErrorEnvelopeV0(inv, CliErrConfiguracionInvalidaV0, "server_url", "server_url_invalida", 0, false, start)
}

func transportOperationalEnvelopeV0(inv CliInvocationContextV0, err error, start time.Time) CliOutputEnvelopeV0 {
	detail := "request_failed"
	if isTimeoutErrorV0(err) {
		detail = "timeout"
	}
	return cliSingleOperationalErrorEnvelopeV0(inv, CliErrErrorTransporteV0, "transporte", detail, 0, true, start)
}

func operationalValidationErrorEnvelopeV0(inv CliInvocationContextV0, err error, start time.Time) CliOutputEnvelopeV0 {
	return operationalValidationErrorEnvelopeV0WithStatus(inv, err, 0, start)
}

func operationalValidationErrorEnvelopeV0WithStatus(inv CliInvocationContextV0, err error, statusCode int, start time.Time) CliOutputEnvelopeV0 {
	var validationErr orquestaobservability.OperationalStatusValidationErrorV0
	if !errors.As(err, &validationErr) || len(validationErr.Issues) == 0 {
		return cliSingleOperationalErrorEnvelopeV0(inv, CliErrRespuestaInvalidaV0, "body", "json_invalido", statusCode, false, start)
	}
	return NewCliOutputErrorEnvelopeV0(
		inv,
		CliContractOperationalStatusV0,
		CliContractVersionOperationalV0,
		operationalIssuesToCliErrorsV0(sanitizeOperationalIssuesForCliV0(validationErr.Issues)),
		cliMetaV0(start, statusCode, false),
	)
}

func cliSingleOperationalErrorEnvelopeV0(inv CliInvocationContextV0, code, field, detail string, statusCode int, retryable bool, start time.Time) CliOutputEnvelopeV0 {
	return NewCliOutputErrorEnvelopeV0(
		inv,
		CliContractOperationalStatusV0,
		CliContractVersionOperationalV0,
		[]CliPublicErrorV0{NewCliPublicErrorV0(code, field, detail)},
		cliMetaV0(start, statusCode, retryable),
	)
}

func endpointOperationalV0(client *OperationalStatusCliClientV0) string {
	if client != nil && strings.TrimSpace(client.Endpoint) != "" {
		return client.Endpoint
	}
	return OperationalStatusCliEndpointV0
}

func httpClientOperationalV0(client *OperationalStatusCliClientV0, timeout time.Duration) *http.Client {
	if client == nil {
		return httpClientFromConfigV0(cliRESTClientConfigV0{}, timeout)
	}
	return httpClientFromConfigV0(cliRESTClientConfigV0{HTTPClient: client.HTTPClient}, timeout)
}

func clientTimeoutOperationalV0(client *OperationalStatusCliClientV0) time.Duration {
	if client == nil {
		return clientTimeoutFromConfigV0(cliRESTClientConfigV0{})
	}
	return clientTimeoutFromConfigV0(cliRESTClientConfigV0{Timeout: client.Timeout})
}

func isKnownOperationalIssueCliV0(code string) bool {
	switch strings.TrimSpace(code) {
	case orquestaobservability.ErrOperationalStatusQueryInvalidaV0,
		orquestaobservability.ErrConsumidorNoAutorizadoV0,
		orquestaobservability.ErrScopeNoSoportadoV0,
		orquestaobservability.ErrReferenciaNoOpacaV0,
		orquestaobservability.ErrConsultaDemasiadoAmpliaV0,
		orquestaobservability.ErrProyeccionNoDisponibleV0,
		orquestaobservability.ErrDiagnosticoNoDisponibleV0,
		orquestaobservability.ErrFrescuraNoGarantizadaV0,
		orquestaobservability.ErrSecretoDetectadoV0,
		orquestaobservability.ErrTranscriptNoPermitidoV0:
		return true
	default:
		return false
	}
}
