package orquestacli

import (
	"errors"
	"net/http"
	"strings"
	"time"

	orquestafactory "orquesta/modulos/orquesta-factory"
)

func clientErrorEnvelopeV0(inv CliInvocationContextV0, err error, start time.Time) CliOutputEnvelopeV0 {
	var cliErr CliClientErrorV0
	if errors.As(err, &cliErr) {
		return cliSingleErrorEnvelopeV0(inv, cliErr.Code, cliErr.Field, cliErr.Detail, cliErr.StatusCode, cliErr.Retryable, start)
	}
	return cliSingleErrorEnvelopeV0(inv, CliErrConfiguracionInvalidaV0, "server_url", "server_url_invalida", 0, false, start)
}

func transportErrorEnvelopeV0(inv CliInvocationContextV0, err error, start time.Time) CliOutputEnvelopeV0 {
	detail := "request_failed"
	if isTimeoutErrorV0(err) {
		detail = "timeout"
	}
	return cliSingleErrorEnvelopeV0(inv, CliErrErrorTransporteV0, "transporte", detail, 0, true, start)
}

func cliSingleErrorEnvelopeV0(inv CliInvocationContextV0, code, field, detail string, statusCode int, retryable bool, start time.Time) CliOutputEnvelopeV0 {
	return NewCliOutputErrorEnvelopeV0(
		inv,
		CliContractSolicitarNuevaAppV0,
		CliContractVersionSolicitarAppV0,
		[]CliPublicErrorV0{NewCliPublicErrorV0(code, field, detail)},
		cliMetaV0(start, statusCode, retryable),
	)
}

func endpointV0(client *SolicitarNuevaAppCliClientV0) string {
	if client != nil && strings.TrimSpace(client.Endpoint) != "" {
		return client.Endpoint
	}
	return SolicitarNuevaAppCliEndpointV0
}

func httpClientV0(client *SolicitarNuevaAppCliClientV0, timeout time.Duration) *http.Client {
	if client == nil {
		return httpClientFromConfigV0(cliRESTClientConfigV0{}, timeout)
	}
	return httpClientFromConfigV0(cliRESTClientConfigV0{HTTPClient: client.HTTPClient}, timeout)
}

func clientTimeoutV0(client *SolicitarNuevaAppCliClientV0) time.Duration {
	if client == nil {
		return clientTimeoutFromConfigV0(cliRESTClientConfigV0{})
	}
	return clientTimeoutFromConfigV0(cliRESTClientConfigV0{Timeout: client.Timeout})
}

func isKnownFactoryIssueCliV0(code string) bool {
	switch strings.TrimSpace(code) {
	case orquestafactory.ErrAppSpecInvalida,
		orquestafactory.ErrOpcionIncompatible,
		orquestafactory.ErrTargetNoSoportado,
		orquestafactory.ErrIdiomaInvalido,
		orquestafactory.ErrConectorRequeridoNoDisponible:
		return true
	default:
		return false
	}
}
