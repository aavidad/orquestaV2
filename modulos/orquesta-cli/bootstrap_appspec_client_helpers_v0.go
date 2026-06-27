package orquestacli

import (
	"errors"
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

func cliSingleBootstrapAppSpecErrorEnvelopeV0(inv CliInvocationContextV0, code, field, detail string, statusCode int, retryable bool, start time.Time) CliOutputEnvelopeV0 {
	return NewCliOutputErrorEnvelopeV0(
		inv,
		BootstrapAppSpecCliContractV0,
		BootstrapAppSpecCliContractVersionV0,
		[]CliPublicErrorV0{NewCliPublicErrorV0(code, field, detail)},
		cliMetaV0(start, statusCode, retryable),
	)
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
