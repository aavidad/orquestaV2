package orquestacli

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	orquestafactory "orquesta/modulos/orquesta-factory"
)

const (
	SolicitarNuevaAppCliEndpointV0          = "/api/v0/apps/spec"
	SolicitarNuevaAppCliCorrelationHeaderV0 = CliCorrelationHeaderV0
	SolicitarNuevaAppCliSourceV0            = "orquesta-cli"
	SolicitarNuevaAppCliDefaultTimeoutV0    = CliDefaultTimeoutRESTV0
)

type SolicitarNuevaAppCliClientV0 struct {
	BaseURL    string
	Endpoint   string
	Timeout    time.Duration
	HTTPClient *http.Client
}

type SolicitarNuevaAppCliResultV0 struct {
	AppSpec orquestafactory.AppSpecV0                 `json:"app_spec"`
	Backlog orquestafactory.BacklogInicialPropuestoV0 `json:"backlog"`
}

func NewSolicitarNuevaAppCliClientV0(serverURL string, timeout time.Duration) (*SolicitarNuevaAppCliClientV0, error) {
	config, err := newCLIRESTClientConfigV0(serverURL, timeout, SolicitarNuevaAppCliEndpointV0)
	if err != nil {
		return nil, err
	}
	return &SolicitarNuevaAppCliClientV0{
		BaseURL:    config.BaseURL,
		Endpoint:   config.Endpoint,
		Timeout:    config.Timeout,
		HTTPClient: newCLILoopbackHTTPClientV0(config.Timeout),
	}, nil
}

func (client *SolicitarNuevaAppCliClientV0) SolicitarNuevaApp(ctx context.Context, inv CliInvocationContextV0, req orquestafactory.AppSpecRequestV0) CliOutputEnvelopeV0 {
	start := time.Now()
	inv = NormalizeCliInvocationContextV0(inv)
	if inv.Command == "" {
		inv.Command = CliDefaultCommandSolicitarAppV0
	}
	timeout := effectiveTimeoutV0(inv.Timeout, clientTimeoutV0(client))
	inv.Timeout = timeout
	if inv.ServerURL == "" && client != nil {
		inv.ServerURL = client.BaseURL
	}

	if errs := validateSolicitarNuevaAppInvocationV0(inv); len(errs) > 0 {
		return NewCliOutputErrorEnvelopeV0(inv, CliContractSolicitarNuevaAppV0, CliContractVersionSolicitarAppV0, errs, cliMetaV0(start, 0, false))
	}
	baseURL, err := normalizeServerURLV0(inv.ServerURL)
	if err != nil {
		return clientErrorEnvelopeV0(inv, err, start)
	}

	req.Source = SolicitarNuevaAppCliSourceV0
	req.RequestID = inv.RequestID
	payload, err := json.Marshal(req)
	if err != nil {
		return cliSingleErrorEnvelopeV0(inv, CliErrRespuestaInvalidaV0, "body", "request_no_serializable", 0, false, start)
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	httpReq, err := prepareCLIRESTRequestV0(ctx, baseURL, endpointV0(client), inv.CorrelationID, payload)
	if err != nil {
		return cliSingleErrorEnvelopeV0(inv, CliErrErrorTransporteV0, "server_url", "request_http_invalida", 0, true, start)
	}

	resp, err := httpClientV0(client, timeout).Do(httpReq)
	if err != nil {
		return transportErrorEnvelopeV0(inv, err, start)
	}
	defer resp.Body.Close()

	return decodeSolicitarNuevaAppResponseV0(resp, inv, start)
}

func decodeSolicitarNuevaAppResponseV0(resp *http.Response, inv CliInvocationContextV0, start time.Time) CliOutputEnvelopeV0 {
	if resp.StatusCode == http.StatusBadRequest {
		raw, detail := readCLIRESTResponseBodyForCommandV0(resp, inv.Command)
		if detail != "" {
			return cliSingleErrorEnvelopeV0(inv, CliErrRespuestaInvalidaV0, "body", detail, resp.StatusCode, false, start)
		}
		issues, ok := decodeSolicitarNuevaAppValidationIssuesV0(bytes.NewReader(raw))
		if !ok || len(issues) == 0 {
			return cliSingleErrorEnvelopeV0(inv, CliErrRespuestaInvalidaV0, "errores", "respuesta_400_sin_errores_publicos", resp.StatusCode, false, start)
		}
		return NewCliOutputErrorEnvelopeV0(
			inv,
			CliContractSolicitarNuevaAppV0,
			CliContractVersionSolicitarAppV0,
			factoryIssuesToCliErrorsV0(issues),
			cliMetaV0(start, resp.StatusCode, false),
		)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode > 299 {
		return cliSingleErrorEnvelopeV0(inv, CliErrErrorTransporteV0, "status_code", cliRESTNo2xxDetailForCommandV0(resp, inv.Command), resp.StatusCode, retryableStatusV0(resp.StatusCode), start)
	}

	var out solicitarNuevaAppCliHTTPResponseV0
	if detail := decodeCLIRESTJSONBodyForCommandV0(resp, inv.Command, &out); detail != "" {
		return cliSingleErrorEnvelopeV0(inv, CliErrRespuestaInvalidaV0, "body", detail, resp.StatusCode, false, start)
	}
	spec := out.AppSpecValue()
	backlog := out.BacklogValue()
	if strings.TrimSpace(spec.SchemaVersion) == "" || strings.TrimSpace(backlog.SchemaVersion) == "" {
		return cliSingleErrorEnvelopeV0(inv, CliErrRespuestaInvalidaV0, "data", "app_spec_y_backlog_requeridos", resp.StatusCode, false, start)
	}

	return NewCliOutputOKEnvelopeV0(
		inv,
		CliContractSolicitarNuevaAppV0,
		CliContractVersionSolicitarAppV0,
		SolicitarNuevaAppCliResultV0{AppSpec: spec, Backlog: backlog},
		cliMetaV0(start, resp.StatusCode, false),
	)
}

type solicitarNuevaAppCliHTTPResponseV0 struct {
	AppSpec    orquestafactory.AppSpecV0                 `json:"app_spec"`
	SpecAlias  orquestafactory.AppSpecV0                 `json:"spec"`
	Backlog    orquestafactory.BacklogInicialPropuestoV0 `json:"backlog"`
	BacklogAlt orquestafactory.BacklogInicialPropuestoV0 `json:"backlog_inicial_propuesto"`
}

func (out solicitarNuevaAppCliHTTPResponseV0) AppSpecValue() orquestafactory.AppSpecV0 {
	if strings.TrimSpace(out.AppSpec.SchemaVersion) != "" {
		return out.AppSpec
	}
	return out.SpecAlias
}

func (out solicitarNuevaAppCliHTTPResponseV0) BacklogValue() orquestafactory.BacklogInicialPropuestoV0 {
	if strings.TrimSpace(out.Backlog.SchemaVersion) != "" {
		return out.Backlog
	}
	return out.BacklogAlt
}

type solicitarNuevaAppCliHTTPErrorV0 struct {
	Errores []orquestafactory.ValidationIssue `json:"errores"`
}

func decodeSolicitarNuevaAppValidationIssuesV0(body io.Reader) ([]orquestafactory.ValidationIssue, bool) {
	var out solicitarNuevaAppCliHTTPErrorV0
	if err := json.NewDecoder(body).Decode(&out); err != nil {
		return nil, false
	}
	return sanitizeFactoryIssuesForCliV0(out.Errores), true
}

func sanitizeFactoryIssuesForCliV0(values []orquestafactory.ValidationIssue) []orquestafactory.ValidationIssue {
	out := make([]orquestafactory.ValidationIssue, 0, len(values))
	for _, value := range values {
		code := strings.TrimSpace(value.Code)
		if !isKnownFactoryIssueCliV0(code) {
			code = orquestafactory.ErrAppSpecInvalida
		}
		out = append(out, orquestafactory.ValidationIssue{
			Code:    code,
			Field:   strings.TrimSpace(value.Field),
			Message: strings.TrimSpace(value.Message),
		})
	}
	if out == nil {
		return []orquestafactory.ValidationIssue{}
	}
	return out
}

func factoryIssuesToCliErrorsV0(values []orquestafactory.ValidationIssue) []CliPublicErrorV0 {
	out := make([]CliPublicErrorV0, 0, len(values))
	for _, value := range values {
		code := strings.TrimSpace(value.Code)
		out = append(out, CliPublicErrorV0{
			Codigo:      code,
			Campo:       strings.TrimSpace(value.Field),
			MensajeI18N: "orquesta_factory.errores." + code,
			Detalle:     strings.TrimSpace(value.Message),
		})
	}
	if out == nil {
		return []CliPublicErrorV0{}
	}
	return out
}

func validateSolicitarNuevaAppInvocationV0(inv CliInvocationContextV0) []CliPublicErrorV0 {
	errs := ValidateCliInvocationContextV0(inv)
	if inv.DryRun {
		errs = append(errs, NewCliPublicErrorV0(CliErrOpcionInvalidaV0, "dry_run", "SolicitarNuevaApp_v0_no_declara_dry_run"))
	}
	return errs
}
