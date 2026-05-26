package orquestacli

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	orquestaserver "orquesta/modulos/orquesta-server"
)

const ServerStatusCliEndpointV0 = "/api/v0/server/status"

type ServerStatusCliClientV0 struct {
	BaseURL    string
	Endpoint   string
	Timeout    time.Duration
	HTTPClient *http.Client
}

func NewServerStatusCliClientV0(serverURL string, timeout time.Duration) (*ServerStatusCliClientV0, error) {
	config, err := newCLIRESTClientConfigV0(serverURL, timeout, ServerStatusCliEndpointV0)
	if err != nil {
		return nil, err
	}
	return &ServerStatusCliClientV0{
		BaseURL:    config.BaseURL,
		Endpoint:   config.Endpoint,
		Timeout:    config.Timeout,
		HTTPClient: &http.Client{Timeout: config.Timeout},
	}, nil
}

func (client *ServerStatusCliClientV0) ConsultarEstadoServidor(
	ctx context.Context,
	inv CliInvocationContextV0,
) CliOutputEnvelopeV0 {
	start := time.Now()
	inv = NormalizeCliInvocationContextV0(inv)
	if inv.Command == "" {
		inv.Command = CliDefaultCommandServerStatusV0
	}
	timeout := effectiveTimeoutV0(inv.Timeout, clientTimeoutServerStatusV0(client))
	inv.Timeout = timeout
	if inv.ServerURL == "" && client != nil {
		inv.ServerURL = client.BaseURL
	}
	if errs := ValidateCliInvocationContextV0(inv); len(errs) > 0 {
		return NewCliOutputErrorEnvelopeV0(inv, CliContractServerStatusV0, CliContractVersionServerStatusV0, errs, cliMetaV0(start, 0, false))
	}
	baseURL, err := normalizeServerURLV0(inv.ServerURL)
	if err != nil {
		return serverStatusClientErrorEnvelopeV0(inv, err, start)
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, joinEndpointV0(baseURL, endpointServerStatusV0(client)), nil)
	if err != nil {
		return serverStatusSingleErrorEnvelopeV0(inv, CliErrErrorTransporteV0, "server_url", "request_http_invalida", 0, true, start)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set(CliCorrelationHeaderV0, strings.TrimSpace(inv.CorrelationID))
	resp, err := httpClientServerStatusV0(client, timeout).Do(req)
	if err != nil {
		return serverStatusTransportEnvelopeV0(inv, err, start)
	}
	defer resp.Body.Close()
	return decodeServerStatusResponseV0(resp, inv, start)
}

func decodeServerStatusResponseV0(resp *http.Response, inv CliInvocationContextV0, start time.Time) CliOutputEnvelopeV0 {
	if resp.StatusCode < http.StatusOK || resp.StatusCode > 299 {
		return serverStatusSingleErrorEnvelopeV0(inv, CliErrErrorTransporteV0, "status_code", "status_no_2xx", resp.StatusCode, retryableStatusV0(resp.StatusCode), start)
	}
	var state orquestaserver.ServerPublicStatusV0
	if err := json.NewDecoder(resp.Body).Decode(&state); err != nil {
		return serverStatusSingleErrorEnvelopeV0(inv, CliErrRespuestaInvalidaV0, "body", "json_invalido", resp.StatusCode, false, start)
	}
	if strings.TrimSpace(state.SchemaVersion) == "" || strings.TrimSpace(state.Status) == "" {
		return serverStatusSingleErrorEnvelopeV0(inv, CliErrRespuestaInvalidaV0, "data", "server_status_invalido", resp.StatusCode, false, start)
	}
	return NewCliOutputOKEnvelopeV0(inv, CliContractServerStatusV0, CliContractVersionServerStatusV0, state, cliMetaV0(start, resp.StatusCode, false))
}
