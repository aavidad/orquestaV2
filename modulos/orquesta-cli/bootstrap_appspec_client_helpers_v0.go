package orquestacli

import (
	"errors"
	"net/http"
	"strings"
	"time"

	orquestadirector "orquesta/modulos/orquesta-director"
)

func newBootstrapAppSpecLegacyClientConfigV0(serverURL string, timeout time.Duration) (cliRESTClientConfigV0, error) {
	serverURL = strings.TrimSpace(serverURL)
	baseURL := ""
	if serverURL != "" {
		normalized, err := normalizeServerURLV0(serverURL)
		if err != nil {
			return cliRESTClientConfigV0{}, err
		}
		baseURL = normalized
	}
	return cliRESTClientConfigV0{
		BaseURL:  baseURL,
		Endpoint: "",
		Timeout:  effectiveTimeoutV0(timeout, 0),
	}, nil
}

func bootstrapAppSpecQuarantineEnvelopeV0(inv CliInvocationContextV0, start time.Time) CliOutputEnvelopeV0 {
	return NewCliOutputErrorEnvelopeV0(
		inv,
		BootstrapAppSpecCliContractV0,
		BootstrapAppSpecCliContractVersionV0,
		[]CliPublicErrorV0{
			NewCliPublicErrorV0(CliErrContratoNoConfiguradoV0, "route_policy", BootstrapAppSpecCliQuarantineDetailV0),
		},
		cliMetaV0(start, 0, false),
	)
}

func bootstrapAppSpecQuarantinedCommandV0() orquestadirector.BootstrapProyectoDesdeAppSpecCommandV0 {
	return orquestadirector.BootstrapProyectoDesdeAppSpecCommandV0{}
}

func clientErrorBootstrapAppSpecEnvelopeV0(inv CliInvocationContextV0, err error, start time.Time) CliOutputEnvelopeV0 {
	var cliErr CliClientErrorV0
	if errors.As(err, &cliErr) {
		return cliSingleBootstrapAppSpecErrorEnvelopeV0(inv, cliErr.Code, cliErr.Field, cliErr.Detail, cliErr.StatusCode, cliErr.Retryable, start)
	}
	return cliSingleBootstrapAppSpecErrorEnvelopeV0(inv, CliErrConfiguracionInvalidaV0, "server_url", "server_url_invalida", 0, false, start)
}

func transportBootstrapAppSpecEnvelopeV0(inv CliInvocationContextV0, err error, start time.Time) CliOutputEnvelopeV0 {
	detail := "request_failed"
	if isTimeoutErrorV0(err) {
		detail = "timeout"
	}
	return cliSingleBootstrapAppSpecErrorEnvelopeV0(inv, CliErrErrorTransporteV0, "transporte", detail, 0, true, start)
}

func cliSingleBootstrapAppSpecErrorEnvelopeV0(inv CliInvocationContextV0, code, field, detail string, statusCode int, retryable bool, start time.Time) CliOutputEnvelopeV0 {
	return NewCliOutputErrorEnvelopeV0(
		inv,
		BootstrapAppSpecCliContractV0,
		BootstrapAppSpecCliContractVersionV0,
		[]CliPublicErrorV0{NewCliPublicErrorV0(code, field, detail)},
		cliMetaV0(start, statusCode, retryable),
	)
}

func endpointBootstrapAppSpecV0(client *BootstrapAppSpecCliClientV0) string {
	if client != nil && strings.TrimSpace(client.Endpoint) != "" {
		return client.Endpoint
	}
	return BootstrapAppSpecCliLegacyEndpointV0
}

func httpClientBootstrapAppSpecV0(client *BootstrapAppSpecCliClientV0, timeout time.Duration) *http.Client {
	if client == nil {
		return httpClientFromConfigV0(cliRESTClientConfigV0{}, timeout)
	}
	return httpClientFromConfigV0(cliRESTClientConfigV0{HTTPClient: client.HTTPClient}, timeout)
}

func clientTimeoutBootstrapAppSpecV0(client *BootstrapAppSpecCliClientV0) time.Duration {
	if client == nil {
		return clientTimeoutFromConfigV0(cliRESTClientConfigV0{})
	}
	return clientTimeoutFromConfigV0(cliRESTClientConfigV0{Timeout: client.Timeout})
}

func validateBootstrapAppSpecInvocationV0(inv CliInvocationContextV0) []CliPublicErrorV0 {
	errs := ValidateCliInvocationContextV0(inv)
	if inv.DryRun {
		errs = append(errs, NewCliPublicErrorV0(CliErrOpcionInvalidaV0, "dry_run", "BootstrapProyectoDesdeAppSpec_v0_no_declara_dry_run"))
	}
	return errs
}

func bootstrapAppSpecIssueToCliErrorV0(code, field, detail string) CliPublicErrorV0 {
	code = strings.TrimSpace(code)
	if code == "" || !isKnownBootstrapAppSpecIssueCliV0(code) {
		code = orquestadirector.ErrDirectorBootstrapInvalidoV0
	}
	return CliPublicErrorV0{
		Codigo:      code,
		Campo:       strings.TrimSpace(field),
		MensajeI18N: "orquesta_director.errores." + code,
		Detalle:     strings.TrimSpace(detail),
	}
}

func isKnownBootstrapAppSpecIssueCliV0(code string) bool {
	switch strings.TrimSpace(code) {
	case orquestadirector.ErrDirectorBootstrapInvalidoV0,
		"comando_invalido",
		"idempotency_key_requerida",
		"transicion_invalida",
		"payload_invalido",
		"detalle_prohibido":
		return true
	default:
		return false
	}
}

func firstNonEmptyBootstrapAppSpecV0(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
