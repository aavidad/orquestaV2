package orquestacli

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	orquestaobservability "orquesta/modulos/orquesta-observability"
)

const (
	OperationalStatusCliEndpointV0          = "/api/v0/operational-status/query"
	OperationalStatusCliCorrelationHeaderV0 = CliCorrelationHeaderV0
)

type OperationalStatusCliClientV0 struct {
	BaseURL    string
	Endpoint   string
	Timeout    time.Duration
	HTTPClient *http.Client
}

func NewOperationalStatusCliClientV0(serverURL string, timeout time.Duration) (*OperationalStatusCliClientV0, error) {
	config, err := newCLIRESTClientConfigV0(serverURL, timeout, OperationalStatusCliEndpointV0)
	if err != nil {
		return nil, err
	}
	return &OperationalStatusCliClientV0{
		BaseURL:    config.BaseURL,
		Endpoint:   config.Endpoint,
		Timeout:    config.Timeout,
		HTTPClient: newCLILoopbackHTTPClientV0(config.Timeout),
	}, nil
}

func (client *OperationalStatusCliClientV0) ConsultarEstadoOperativo(ctx context.Context, inv CliInvocationContextV0, query orquestaobservability.OperationalStatusQueryV0) CliOutputEnvelopeV0 {
	start := time.Now()
	inv = NormalizeCliInvocationContextV0(inv)
	if inv.Command == "" {
		inv.Command = CliDefaultCommandOperationalV0
	}
	timeout := effectiveTimeoutV0(inv.Timeout, clientTimeoutOperationalV0(client))
	inv.Timeout = timeout
	if inv.ServerURL == "" && client != nil {
		inv.ServerURL = client.BaseURL
	}

	if errs := ValidateCliInvocationContextV0(inv); len(errs) > 0 {
		return NewCliOutputErrorEnvelopeV0(inv, CliContractOperationalStatusV0, CliContractVersionOperationalV0, errs, cliMetaV0(start, 0, false))
	}
	baseURL, err := normalizeServerURLV0(inv.ServerURL)
	if err != nil {
		return clientErrorOperationalEnvelopeV0(inv, err, start)
	}

	query.SchemaVersion = orquestaobservability.OperationalStatusQuerySchemaVersionV0
	query.RequestID = inv.RequestID
	query.CorrelationID = inv.CorrelationID
	query.Consumer = orquestaobservability.OperationalStatusConsumerV0{
		Module:  "orquesta-cli",
		Channel: orquestaobservability.OperationalStatusConsumerCLIChannelV0,
	}
	if strings.TrimSpace(query.Locale) == "" && strings.TrimSpace(inv.Locale) != "" {
		query.Locale = strings.TrimSpace(inv.Locale)
	}
	if strings.TrimSpace(query.Locale) == "" {
		query.Locale = "es"
	}

	if err := orquestaobservability.ValidateOperationalStatusQueryV0(query); err != nil {
		return operationalValidationErrorEnvelopeV0(inv, err, start)
	}

	payload, err := json.Marshal(query)
	if err != nil {
		return cliSingleOperationalErrorEnvelopeV0(inv, CliErrRespuestaInvalidaV0, "body", "request_no_serializable", 0, false, start)
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	httpReq, err := prepareCLIRESTRequestV0(ctx, baseURL, endpointOperationalV0(client), inv.CorrelationID, payload)
	if err != nil {
		return cliSingleOperationalErrorEnvelopeV0(inv, CliErrErrorTransporteV0, "server_url", "request_http_invalida", 0, true, start)
	}

	resp, err := httpClientOperationalV0(client, timeout).Do(httpReq)
	if err != nil {
		return transportOperationalEnvelopeV0(inv, err, start)
	}
	defer resp.Body.Close()

	return decodeOperationalStatusResponseV0(resp, inv, start)
}

func decodeOperationalStatusResponseV0(resp *http.Response, inv CliInvocationContextV0, start time.Time) CliOutputEnvelopeV0 {
	if resp.StatusCode == http.StatusBadRequest {
		raw, detail := readCLIRESTResponseBodyForCommandV0(resp, inv.Command)
		if detail != "" {
			return cliSingleOperationalErrorEnvelopeV0(inv, CliErrRespuestaInvalidaV0, "body", detail, resp.StatusCode, false, start)
		}
		issues, ok := decodeOperationalStatusIssuesV0(bytes.NewReader(raw))
		if !ok || len(issues) == 0 {
			return cliSingleOperationalErrorEnvelopeV0(inv, CliErrRespuestaInvalidaV0, "errores", "respuesta_400_sin_errores_publicos", resp.StatusCode, false, start)
		}
		return NewCliOutputErrorEnvelopeV0(
			inv,
			CliContractOperationalStatusV0,
			CliContractVersionOperationalV0,
			operationalIssuesToCliErrorsV0(issues),
			cliMetaV0(start, resp.StatusCode, false),
		)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode > 299 {
		return cliSingleOperationalErrorEnvelopeV0(inv, CliErrErrorTransporteV0, "status_code", cliRESTNo2xxDetailForCommandV0(resp, inv.Command), resp.StatusCode, retryableStatusV0(resp.StatusCode), start)
	}

	raw, detail := readCLIRESTResponseBodyForCommandV0(resp, inv.Command)
	if detail != "" {
		return cliSingleOperationalErrorEnvelopeV0(inv, CliErrRespuestaInvalidaV0, "body", detail, resp.StatusCode, false, start)
	}
	diagnostic, err := orquestaobservability.DecodeDiagnosticoCompactoV0(raw)
	if err != nil {
		return operationalValidationErrorEnvelopeV0WithStatus(inv, err, resp.StatusCode, start)
	}

	return NewCliOutputOKEnvelopeV0(
		inv,
		CliContractOperationalStatusV0,
		CliContractVersionOperationalV0,
		diagnostic,
		cliMetaV0(start, resp.StatusCode, false),
	)
}

func decodeOperationalStatusIssuesV0(body interface{ Read([]byte) (int, error) }) ([]orquestaobservability.OperationalStatusValidationIssueV0, bool) {
	var out orquestaobservability.OperationalStatusValidationErrorV0
	if err := json.NewDecoder(body).Decode(&out); err != nil {
		return nil, false
	}
	return sanitizeOperationalIssuesForCliV0(out.Issues), true
}

func sanitizeOperationalIssuesForCliV0(values []orquestaobservability.OperationalStatusValidationIssueV0) []orquestaobservability.OperationalStatusValidationIssueV0 {
	out := make([]orquestaobservability.OperationalStatusValidationIssueV0, 0, len(values))
	for _, value := range values {
		code := strings.TrimSpace(value.Code)
		if !isKnownOperationalIssueCliV0(code) {
			code = orquestaobservability.ErrOperationalStatusQueryInvalidaV0
		}
		out = append(out, orquestaobservability.OperationalStatusValidationIssueV0{
			Code:  code,
			Field: strings.TrimSpace(value.Field),
		})
	}
	if out == nil {
		return []orquestaobservability.OperationalStatusValidationIssueV0{}
	}
	return out
}

func operationalIssuesToCliErrorsV0(values []orquestaobservability.OperationalStatusValidationIssueV0) []CliPublicErrorV0 {
	out := make([]CliPublicErrorV0, 0, len(values))
	for _, value := range values {
		code := strings.TrimSpace(value.Code)
		out = append(out, CliPublicErrorV0{
			Codigo:      code,
			Campo:       strings.TrimSpace(value.Field),
			MensajeI18N: "orquesta_observability.errores." + code,
		})
	}
	if out == nil {
		return []CliPublicErrorV0{}
	}
	return out
}
