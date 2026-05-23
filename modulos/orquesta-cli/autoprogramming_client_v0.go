package orquestacli

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	orquestamcp "orquesta/modulos/orquesta-mcp"
)

const (
	AutoprogrammingPrepareRunCliEndpointV0 = "/api/v0/autoprogramming/prepare-run"
	AutoprogrammingRunQueueCliEndpointV0   = "/api/v0/runs/queue/priority"
	AutoprogrammingRunStatsCliEndpointV0   = "/api/v0/director/stats"
)

type AutoprogrammingCliClientV0 struct {
	BaseURL    string
	Timeout    time.Duration
	HTTPClient *http.Client
}

func NewAutoprogrammingCliClientV0(serverURL string, timeout time.Duration) (*AutoprogrammingCliClientV0, error) {
	config, err := newCLIRESTClientConfigV0(serverURL, timeout, AutoprogrammingPrepareRunCliEndpointV0)
	if err != nil {
		return nil, err
	}
	return &AutoprogrammingCliClientV0{
		BaseURL:    config.BaseURL,
		Timeout:    config.Timeout,
		HTTPClient: &http.Client{Timeout: config.Timeout},
	}, nil
}

func (client *AutoprogrammingCliClientV0) PrepararRun(
	ctx context.Context,
	inv CliInvocationContextV0,
	input orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0,
) CliOutputEnvelopeV0 {
	input.RequestID = firstNonEmptyCliStringV0(input.RequestID, inv.RequestID)
	input.CorrelationID = firstNonEmptyCliStringV0(input.CorrelationID, inv.CorrelationID)
	return client.postV0(ctx, inv, CliDefaultCommandAutoprogPrepareV0, AutoprogrammingPrepareRunCliEndpointV0, input, decodeAutoprogPrepareRunV0)
}

func (client *AutoprogrammingCliClientV0) ConsultarCola(
	ctx context.Context,
	inv CliInvocationContextV0,
	input orquestamcp.MCPRunQueuePriorityToolInputV0,
) CliOutputEnvelopeV0 {
	input.RequestID = firstNonEmptyCliStringV0(input.RequestID, inv.RequestID)
	input.CorrelationID = firstNonEmptyCliStringV0(input.CorrelationID, inv.CorrelationID)
	if strings.TrimSpace(input.Action) == "" {
		input.Action = orquestamcp.MCPRunQueuePriorityActionRankV0
	}
	return client.postV0(ctx, inv, CliDefaultCommandAutoprogQueueV0, AutoprogrammingRunQueueCliEndpointV0, input, decodeAutoprogQueueV0)
}

func (client *AutoprogrammingCliClientV0) ConsultarRun(
	ctx context.Context,
	inv CliInvocationContextV0,
	input orquestamcp.MCPDirectorStatsToolInputV0,
) CliOutputEnvelopeV0 {
	input.RequestID = firstNonEmptyCliStringV0(input.RequestID, inv.RequestID)
	input.CorrelationID = firstNonEmptyCliStringV0(input.CorrelationID, inv.CorrelationID)
	return client.postV0(ctx, inv, CliDefaultCommandAutoprogRunV0, AutoprogrammingRunStatsCliEndpointV0, input, decodeAutoprogRunStatsV0)
}

func (client *AutoprogrammingCliClientV0) postV0(
	ctx context.Context,
	inv CliInvocationContextV0,
	command string,
	endpoint string,
	input any,
	decode func(*http.Response, CliInvocationContextV0, time.Time) CliOutputEnvelopeV0,
) CliOutputEnvelopeV0 {
	start := time.Now()
	inv = NormalizeCliInvocationContextV0(inv)
	if inv.Command == "" {
		inv.Command = command
	}
	timeout := effectiveTimeoutV0(inv.Timeout, clientTimeoutAutoprogrammingV0(client))
	inv.Timeout = timeout
	if inv.ServerURL == "" && client != nil {
		inv.ServerURL = client.BaseURL
	}
	if errs := ValidateCliInvocationContextV0(inv); len(errs) > 0 {
		return NewCliOutputErrorEnvelopeV0(inv, CliContractAutoprogrammingV0, CliContractVersionAutoprogrammingV0, errs, cliMetaV0(start, 0, false))
	}
	baseURL, err := normalizeServerURLV0(inv.ServerURL)
	if err != nil {
		return autoprogClientErrorEnvelopeV0(inv, err, start)
	}
	payload, err := json.Marshal(input)
	if err != nil {
		return autoprogSingleErrorEnvelopeV0(inv, CliErrRespuestaInvalidaV0, "body", "request_no_serializable", 0, false, start)
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := prepareCLIRESTRequestV0(ctx, baseURL, endpoint, inv.CorrelationID, payload)
	if err != nil {
		return autoprogSingleErrorEnvelopeV0(inv, CliErrErrorTransporteV0, "server_url", "request_http_invalida", 0, true, start)
	}
	resp, err := httpClientAutoprogrammingV0(client, timeout).Do(req)
	if err != nil {
		return autoprogTransportEnvelopeV0(inv, err, start)
	}
	defer resp.Body.Close()
	return decode(resp, inv, start)
}

func decodeAutoprogPrepareRunV0(resp *http.Response, inv CliInvocationContextV0, start time.Time) CliOutputEnvelopeV0 {
	var result orquestamcp.MCPAutoprogrammingPrepareRunToolResultV0
	return decodeAutoprogToolResponseV0(resp, inv, start, &result, result.Estado, result.Errores)
}

func decodeAutoprogQueueV0(resp *http.Response, inv CliInvocationContextV0, start time.Time) CliOutputEnvelopeV0 {
	var result orquestamcp.MCPRunQueuePriorityToolResultV0
	return decodeAutoprogToolResponseV0(resp, inv, start, &result, result.Estado, result.Errores)
}

func decodeAutoprogRunStatsV0(resp *http.Response, inv CliInvocationContextV0, start time.Time) CliOutputEnvelopeV0 {
	var result orquestamcp.MCPDirectorStatsToolResultV0
	return decodeAutoprogToolResponseV0(resp, inv, start, &result, result.Estado, result.Errores)
}

func decodeAutoprogToolResponseV0(
	resp *http.Response,
	inv CliInvocationContextV0,
	start time.Time,
	target any,
	estado string,
	issues []orquestamcp.MCPValidationIssueV0,
) CliOutputEnvelopeV0 {
	if resp.StatusCode < http.StatusOK || resp.StatusCode > 299 {
		if resp.StatusCode != http.StatusBadRequest {
			return autoprogSingleErrorEnvelopeV0(inv, CliErrErrorTransporteV0, "status_code", "status_no_2xx", resp.StatusCode, retryableStatusV0(resp.StatusCode), start)
		}
	}
	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return autoprogSingleErrorEnvelopeV0(inv, CliErrRespuestaInvalidaV0, "body", "json_invalido", resp.StatusCode, false, start)
	}
	estado, issues = autoprogEstadoAndIssuesV0(target)
	if strings.TrimSpace(estado) == "" {
		return autoprogSingleErrorEnvelopeV0(inv, CliErrRespuestaInvalidaV0, "data", "autoprogramming_result_invalido", resp.StatusCode, false, start)
	}
	if resp.StatusCode == http.StatusBadRequest || estado == "error" {
		return NewCliOutputErrorEnvelopeV0(inv, CliContractAutoprogrammingV0, CliContractVersionAutoprogrammingV0, autoprogIssuesToCliV0(issues), cliMetaV0(start, resp.StatusCode, false))
	}
	return NewCliOutputOKEnvelopeV0(inv, CliContractAutoprogrammingV0, CliContractVersionAutoprogrammingV0, target, cliMetaV0(start, resp.StatusCode, false))
}
