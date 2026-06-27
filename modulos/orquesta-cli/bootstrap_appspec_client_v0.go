package orquestacli

import (
	"context"
	"net/http"
	"time"

	orquestadirector "orquesta/modulos/orquesta-director"
)

const (
	BootstrapAppSpecCliLegacyEndpointV0     = "/api/v0/director/bootstrap/appspec"
	BootstrapAppSpecCliPreferredEndpointV0  = "/api/v0/apps/director"
	BootstrapAppSpecCliCorrelationHeaderV0  = CliCorrelationHeaderV0
	BootstrapAppSpecCliContractV0           = "BootstrapProyectoDesdeAppSpec"
	BootstrapAppSpecCliContractVersionV0    = "v0"
	BootstrapAppSpecCliDefaultCommandV0     = "app spec bootstrap"
	BootstrapAppSpecCliQuarantineDetailV0   = "bootstrap_appspec_legacy_route_en_cuarentena_use_api_v0_apps_director"
	BootstrapAppSpecCliQuarantineEvidenceV0 = "T86_bootstrap_appspec_legacy_route_quarantine"
)

type BootstrapAppSpecCliClientV0 struct {
	BaseURL    string
	Endpoint   string
	Timeout    time.Duration
	HTTPClient *http.Client
}

func NewBootstrapAppSpecCliClientV0(serverURL string, timeout time.Duration) (*BootstrapAppSpecCliClientV0, error) {
	config, err := newBootstrapAppSpecLegacyClientConfigV0(serverURL, timeout)
	if err != nil {
		return nil, err
	}
	return &BootstrapAppSpecCliClientV0{
		BaseURL:    config.BaseURL,
		Endpoint:   config.Endpoint,
		Timeout:    config.Timeout,
		HTTPClient: newCLILoopbackHTTPClientV0(config.Timeout),
	}, nil
}

func (client *BootstrapAppSpecCliClientV0) BootstrapProyectoDesdeAppSpec(ctx context.Context, inv CliInvocationContextV0, cmd orquestadirector.BootstrapProyectoDesdeAppSpecCommandV0) CliOutputEnvelopeV0 {
	_ = ctx
	_ = cmd
	start := time.Now()
	inv = NormalizeCliInvocationContextV0(inv)
	if inv.Command == "" {
		inv.Command = BootstrapAppSpecCliDefaultCommandV0
	}
	timeout := effectiveTimeoutV0(inv.Timeout, clientTimeoutBootstrapAppSpecV0(client))
	inv.Timeout = timeout
	if inv.ServerURL == "" && client != nil {
		inv.ServerURL = client.BaseURL
	}

	if errs := validateBootstrapAppSpecInvocationV0(inv); len(errs) > 0 {
		return NewCliOutputErrorEnvelopeV0(inv, BootstrapAppSpecCliContractV0, BootstrapAppSpecCliContractVersionV0, errs, cliMetaV0(start, 0, false))
	}
	return bootstrapAppSpecQuarantineEnvelopeV0(inv, start)
}
