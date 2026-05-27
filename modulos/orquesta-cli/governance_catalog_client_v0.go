package orquestacli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	orquestagovernance "orquesta/modulos/orquesta-governance"
)

const (
	GovernanceCatalogCliEndpointV0          = "/api/v0/governance/catalog/query"
	GovernanceCatalogCliCorrelationHeaderV0 = CliCorrelationHeaderV0
)

type GovernanceCatalogCliReaderV0 struct {
	BaseURL    string
	Endpoint   string
	Timeout    time.Duration
	HTTPClient *http.Client
}

type governanceCatalogQueryHTTPRequestV0 struct {
	RequestID     string                              `json:"request_id,omitempty"`
	CorrelationID string                              `json:"correlation_id,omitempty"`
	Module        string                              `json:"module,omitempty"`
	Role          string                              `json:"role,omitempty"`
	Phase         string                              `json:"phase,omitempty"`
	Tags          []string                            `json:"tags,omitempty"`
	Filters       governanceCatalogQueryHTTPFiltersV0 `json:"filters"`
}

type governanceCatalogQueryHTTPFiltersV0 struct {
	Module string   `json:"module,omitempty"`
	Role   string   `json:"role,omitempty"`
	Phase  string   `json:"phase,omitempty"`
	Tags   []string `json:"tags,omitempty"`
}

type governanceCatalogHTTPErrorV0 struct {
	RequestID     string                     `json:"request_id,omitempty"`
	CorrelationID string                     `json:"correlation_id,omitempty"`
	Errors        []governanceCatalogIssueV0 `json:"errors"`
}

type governanceCatalogIssueV0 struct {
	Code  string `json:"code"`
	Field string `json:"field,omitempty"`
}

func NewGovernanceCatalogCliReaderV0(serverURL string, timeout time.Duration) (*GovernanceCatalogCliReaderV0, error) {
	config, err := newCLIRESTClientConfigV0(serverURL, timeout, GovernanceCatalogCliEndpointV0)
	if err != nil {
		return nil, err
	}
	return &GovernanceCatalogCliReaderV0{
		BaseURL:    config.BaseURL,
		Endpoint:   config.Endpoint,
		Timeout:    config.Timeout,
		HTTPClient: newCLILoopbackHTTPClientV0(config.Timeout),
	}, nil
}

func (client *GovernanceCatalogCliReaderV0) ConsultarCatalogo(ctx context.Context, inv CliInvocationContextV0, query orquestagovernance.GovernanceCatalogQueryV0) CliOutputEnvelopeV0 {
	start := time.Now()
	inv = NormalizeCliInvocationContextV0(inv)
	if inv.Command == "" {
		inv.Command = CliDefaultCommandGovernanceV0
	}
	timeout := effectiveTimeoutV0(inv.Timeout, clientTimeoutGovernanceV0(client))
	inv.Timeout = timeout
	if inv.ServerURL == "" && client != nil {
		inv.ServerURL = client.BaseURL
	}

	if errs := ValidateCliInvocationContextV0(inv); len(errs) > 0 {
		return NewCliOutputErrorEnvelopeV0(inv, CliContractGovernanceCatalogV0, CliContractVersionGovernanceV0, errs, cliMetaV0(start, 0, false))
	}
	baseURL, err := normalizeServerURLV0(inv.ServerURL)
	if err != nil {
		return clientErrorGovernanceEnvelopeV0(inv, err, start)
	}

	payload, err := json.Marshal(governanceCatalogRequestFromQueryV0(inv, query))
	if err != nil {
		return cliSingleGovernanceErrorEnvelopeV0(inv, CliErrRespuestaInvalidaV0, "body", "request_no_serializable", 0, false, start)
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	httpReq, err := prepareCLIRESTRequestV0(ctx, baseURL, endpointGovernanceV0(client), inv.CorrelationID, payload)
	if err != nil {
		return cliSingleGovernanceErrorEnvelopeV0(inv, CliErrErrorTransporteV0, "server_url", "request_http_invalida", 0, true, start)
	}

	resp, err := httpClientGovernanceV0(client, timeout).Do(httpReq)
	if err != nil {
		return transportGovernanceEnvelopeV0(inv, err, start)
	}
	defer resp.Body.Close()

	return decodeGovernanceCatalogResponseV0(resp, inv, start)
}

func governanceCatalogRequestFromQueryV0(inv CliInvocationContextV0, query orquestagovernance.GovernanceCatalogQueryV0) governanceCatalogQueryHTTPRequestV0 {
	tags := governanceCatalogCleanTagsV0(query.Tags)
	return governanceCatalogQueryHTTPRequestV0{
		RequestID:     strings.TrimSpace(inv.RequestID),
		CorrelationID: strings.TrimSpace(inv.CorrelationID),
		Module:        strings.TrimSpace(query.Module),
		Role:          strings.TrimSpace(query.Role),
		Phase:         strings.TrimSpace(query.Phase),
		Tags:          tags,
		Filters: governanceCatalogQueryHTTPFiltersV0{
			Module: strings.TrimSpace(query.Module),
			Role:   strings.TrimSpace(query.Role),
			Phase:  strings.TrimSpace(query.Phase),
			Tags:   tags,
		},
	}
}

func decodeGovernanceCatalogResponseV0(resp *http.Response, inv CliInvocationContextV0, start time.Time) CliOutputEnvelopeV0 {
	if resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusServiceUnavailable {
		raw, detail := readCLIRESTResponseBodyForCommandV0(resp, inv.Command)
		if detail != "" {
			return cliSingleGovernanceErrorEnvelopeV0(inv, CliErrRespuestaInvalidaV0, "body", detail, resp.StatusCode, false, start)
		}
		issues, ok := decodeGovernanceCatalogIssuesV0(bytes.NewReader(raw))
		if !ok || len(issues) == 0 {
			return cliSingleGovernanceErrorEnvelopeV0(inv, CliErrRespuestaInvalidaV0, "errors", "respuesta_error_sin_errores_publicos", resp.StatusCode, false, start)
		}
		return NewCliOutputErrorEnvelopeV0(
			inv,
			CliContractGovernanceCatalogV0,
			CliContractVersionGovernanceV0,
			governanceIssuesToCliErrorsV0(issues),
			cliMetaV0(start, resp.StatusCode, false),
		)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode > 299 {
		return cliSingleGovernanceErrorEnvelopeV0(inv, CliErrErrorTransporteV0, "status_code", cliRESTNo2xxDetailForCommandV0(resp, inv.Command), resp.StatusCode, retryableStatusV0(resp.StatusCode), start)
	}

	raw, detail := readCLIRESTResponseBodyForCommandV0(resp, inv.Command)
	if detail != "" {
		return cliSingleGovernanceErrorEnvelopeV0(inv, CliErrRespuestaInvalidaV0, "body", detail, resp.StatusCode, false, start)
	}

	result, err := decodeGovernanceCatalogResultV0(raw)
	if err != nil {
		return cliSingleGovernanceErrorEnvelopeV0(inv, CliErrRespuestaInvalidaV0, "body", err.Error(), resp.StatusCode, false, start)
	}

	return NewCliOutputOKEnvelopeV0(
		inv,
		CliContractGovernanceCatalogV0,
		CliContractVersionGovernanceV0,
		result,
		cliMetaV0(start, resp.StatusCode, false),
	)
}

func decodeGovernanceCatalogResultV0(raw []byte) (orquestagovernance.GovernanceCatalogQueryResultV0, error) {
	var response orquestagovernance.GovernanceCatalogPublicQueryResponseV0
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&response); err != nil {
		return orquestagovernance.GovernanceCatalogQueryResultV0{}, errors.New("json_invalido")
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return orquestagovernance.GovernanceCatalogQueryResultV0{}, errors.New("json_invalido")
	}
	result := orquestagovernance.GovernanceCatalogQueryResultV0{
		Effective: response.Effective,
		Counters:  response.Counters,
	}
	if result.Counters.Effective < len(result.Effective) ||
		(response.OutputBudget.Status != orquestagovernance.GovernanceCatalogPublicBudgetTruncatedV0 &&
			result.Counters.Effective != len(result.Effective)) {
		return orquestagovernance.GovernanceCatalogQueryResultV0{}, errors.New("effective_count_no_coincide")
	}
	if result.Counters.Proposed < 0 || result.Counters.Quarantine < 0 {
		return orquestagovernance.GovernanceCatalogQueryResultV0{}, errors.New("contadores_invalidos")
	}
	for _, entry := range result.Effective {
		if err := orquestagovernance.ValidateEffectiveGovernanceEntryV0(entry); err != nil {
			return orquestagovernance.GovernanceCatalogQueryResultV0{}, errors.New("effective_invalido")
		}
	}
	return result, nil
}

func decodeGovernanceCatalogIssuesV0(body io.Reader) ([]governanceCatalogIssueV0, bool) {
	var out governanceCatalogHTTPErrorV0
	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&out); err != nil {
		return nil, false
	}
	return sanitizeGovernanceIssuesForCliV0(out.Errors), true
}

func sanitizeGovernanceIssuesForCliV0(values []governanceCatalogIssueV0) []governanceCatalogIssueV0 {
	out := make([]governanceCatalogIssueV0, 0, len(values))
	for _, value := range values {
		code := strings.TrimSpace(value.Code)
		if !isKnownGovernanceIssueCliV0(code) {
			code = string(orquestagovernance.GovernanceCatalogForbiddenActivationErrorV0)
		}
		out = append(out, governanceCatalogIssueV0{
			Code:  code,
			Field: strings.TrimSpace(value.Field),
		})
	}
	if out == nil {
		return []governanceCatalogIssueV0{}
	}
	return out
}

func governanceIssuesToCliErrorsV0(values []governanceCatalogIssueV0) []CliPublicErrorV0 {
	out := make([]CliPublicErrorV0, 0, len(values))
	for _, value := range values {
		code := strings.TrimSpace(value.Code)
		out = append(out, CliPublicErrorV0{
			Codigo:      code,
			Campo:       strings.TrimSpace(value.Field),
			MensajeI18N: "orquesta_governance.errores." + code,
		})
	}
	if out == nil {
		return []CliPublicErrorV0{}
	}
	return out
}
