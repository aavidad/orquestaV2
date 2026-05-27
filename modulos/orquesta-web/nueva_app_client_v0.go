package orquestaweb

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	orquestafactory "orquesta/modulos/orquesta-factory"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

const (
	SolicitarNuevaAppEndpointV0       = "/api/v0/apps/spec"
	SolicitarNuevaAppCorrelationV0    = "X-Correlation-ID"
	WebNuevaAppErrTransporteV0        = webPublicErrTransportV0
	WebNuevaAppErrRespuestaInvalidaV0 = webPublicErrResponseInvalidV0
)

var webRequestIDGeneratorV0 = orquestaruntime.NewSystemRefGeneratorV0()

type SolicitarNuevaAppClientV0 interface {
	SolicitarNuevaApp(ctx context.Context, form WebNuevaAppFormV0) (WebNuevaAppViewModelV0, error)
}

type RESTSolicitarNuevaAppClientV0 struct {
	BaseURL    string
	Endpoint   string
	Timeout    time.Duration
	HTTPClient *http.Client
}

type WebNuevaAppClientErrorV0 struct {
	Code       string
	StatusCode int
}

func (err WebNuevaAppClientErrorV0) Error() string {
	return err.Code
}

func NewRESTSolicitarNuevaAppClientV0(baseURL string, timeout time.Duration) *RESTSolicitarNuevaAppClientV0 {
	return &RESTSolicitarNuevaAppClientV0{
		BaseURL:    normalizeWebRESTBaseURLStringV0(baseURL),
		Endpoint:   SolicitarNuevaAppEndpointV0,
		Timeout:    timeout,
		HTTPClient: newWebLoopbackHTTPClientV0(timeout),
	}
}

func (client *RESTSolicitarNuevaAppClientV0) SolicitarNuevaApp(ctx context.Context, form WebNuevaAppFormV0) (WebNuevaAppViewModelV0, error) {
	ctx = webContextOrBackgroundV0(ctx)
	req := form.ToAppSpecRequestV0()
	if strings.TrimSpace(req.RequestID) == "" {
		req.RequestID = newWebRequestIDV0()
	}
	correlationID := req.RequestID
	if client.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, client.Timeout)
		defer cancel()
	}

	payload, err := json.Marshal(req)
	if err != nil {
		return WebNuevaAppViewModelV0{}, webNuevaAppClientErrorV0(WebNuevaAppErrRespuestaInvalidaV0, 0)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, client.url(), bytes.NewReader(payload))
	if err != nil {
		return WebNuevaAppViewModelV0{}, webNuevaAppClientErrorV0(WebNuevaAppErrTransporteV0, 0)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set(SolicitarNuevaAppCorrelationV0, correlationID)

	resp, err := client.httpClient().Do(httpReq)
	if err != nil {
		return WebNuevaAppViewModelV0{}, webNuevaAppClientErrorV0(WebNuevaAppErrTransporteV0, 0)
	}
	defer resp.Body.Close()

	return client.decodeResponse(resp, req)
}

func (client *RESTSolicitarNuevaAppClientV0) decodeResponse(resp *http.Response, req orquestafactory.AppSpecRequestV0) (WebNuevaAppViewModelV0, error) {
	if resp.StatusCode == http.StatusBadRequest {
		var issues []orquestafactory.ValidationIssue
		if body, ok := readWebHTTPResponseBodyV0(resp); ok {
			issues = decodeValidationIssuesV0(bytes.NewReader(body))
		}
		if len(issues) == 0 {
			issues = []orquestafactory.ValidationIssue{publicValidationIssueV0(orquestafactory.ErrAppSpecInvalida, "", "")}
		}
		return NewWebNuevaAppErrorViewModelV0(req.RequestID, req.Locale, issues), nil
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return WebNuevaAppViewModelV0{}, webNuevaAppClientErrorV0(WebNuevaAppErrTransporteV0, resp.StatusCode)
	}

	var out solicitarNuevaAppSuccessEnvelopeV0
	if !decodeWebHTTPJSONResponseV0(resp, &out) {
		return WebNuevaAppViewModelV0{}, webNuevaAppClientErrorV0(WebNuevaAppErrRespuestaInvalidaV0, resp.StatusCode)
	}
	spec := out.AppSpecValue()
	backlog := out.BacklogValue()
	if strings.TrimSpace(spec.SchemaVersion) == "" || strings.TrimSpace(backlog.SchemaVersion) == "" {
		return WebNuevaAppViewModelV0{}, webNuevaAppClientErrorV0(WebNuevaAppErrRespuestaInvalidaV0, resp.StatusCode)
	}
	return NewWebNuevaAppViewModelV0(spec, backlog), nil
}

func (client *RESTSolicitarNuevaAppClientV0) httpClient() *http.Client {
	return webHTTPClientWithRedirectPolicyV0(client.HTTPClient, client.Timeout, client.BaseURL)
}

func (client *RESTSolicitarNuevaAppClientV0) url() string {
	endpoint := client.Endpoint
	if endpoint == "" {
		endpoint = SolicitarNuevaAppEndpointV0
	}
	return webRESTEndpointURLV0(client.BaseURL, endpoint, SolicitarNuevaAppEndpointV0)
}

type solicitarNuevaAppSuccessEnvelopeV0 struct {
	AppSpec    orquestafactory.AppSpecV0                 `json:"app_spec"`
	SpecAlias  orquestafactory.AppSpecV0                 `json:"spec"`
	Backlog    orquestafactory.BacklogInicialPropuestoV0 `json:"backlog"`
	BacklogAlt orquestafactory.BacklogInicialPropuestoV0 `json:"backlog_inicial_propuesto"`
}

func (out solicitarNuevaAppSuccessEnvelopeV0) AppSpecValue() orquestafactory.AppSpecV0 {
	if out.AppSpec.SchemaVersion != "" {
		return out.AppSpec
	}
	return out.SpecAlias
}

func (out solicitarNuevaAppSuccessEnvelopeV0) BacklogValue() orquestafactory.BacklogInicialPropuestoV0 {
	if out.Backlog.SchemaVersion != "" {
		return out.Backlog
	}
	return out.BacklogAlt
}

type solicitarNuevaAppErrorEnvelopeV0 struct {
	Errores []orquestafactory.ValidationIssue `json:"errores"`
	Errors  []orquestafactory.ValidationIssue `json:"errors"`
	Issues  []orquestafactory.ValidationIssue `json:"issues"`
	Error   string                            `json:"error"`
	Code    string                            `json:"code"`
}

func decodeValidationIssuesV0(body io.Reader) []orquestafactory.ValidationIssue {
	var out solicitarNuevaAppErrorEnvelopeV0
	if err := json.NewDecoder(body).Decode(&out); err != nil {
		return []orquestafactory.ValidationIssue{publicValidationIssueV0(orquestafactory.ErrAppSpecInvalida, "", "")}
	}
	issues := append(append(out.Errores, out.Errors...), out.Issues...)
	if len(issues) == 0 && isKnownFactoryIssueCodeV0(out.Code) {
		issues = append(issues, orquestafactory.ValidationIssue{Code: out.Code})
	}
	if len(issues) == 0 && isKnownFactoryIssueCodeV0(out.Error) {
		issues = append(issues, orquestafactory.ValidationIssue{Code: out.Error})
	}
	return sanitizeFactoryIssuesV0(issues)
}

func sanitizeFactoryIssuesV0(values []orquestafactory.ValidationIssue) []orquestafactory.ValidationIssue {
	out := make([]orquestafactory.ValidationIssue, 0, len(values))
	for _, value := range values {
		code := strings.TrimSpace(value.Code)
		if !isKnownFactoryIssueCodeV0(code) {
			code = orquestafactory.ErrAppSpecInvalida
		}
		out = append(out, publicValidationIssueV0(code, value.Field, value.Message))
	}
	return out
}

func publicValidationIssueV0(code, field, message string) orquestafactory.ValidationIssue {
	message = strings.TrimSpace(message)
	if message == "" {
		message = "validacion_publica"
	}
	return orquestafactory.ValidationIssue{
		Code:    code,
		Field:   strings.TrimSpace(field),
		Message: message,
	}
}

func isKnownFactoryIssueCodeV0(code string) bool {
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

func webNuevaAppClientErrorV0(code string, statusCode int) WebNuevaAppClientErrorV0 {
	return WebNuevaAppClientErrorV0{Code: code, StatusCode: statusCode}
}

func newWebRequestIDV0() string {
	ref, _ := webRequestIDGeneratorV0.NextRefV0("req-web", "client-mutation")
	return ref
}

func IsWebNuevaAppClientErrorCodeV0(err error, code string) bool {
	var clientErr WebNuevaAppClientErrorV0
	if errors.As(err, &clientErr) {
		return clientErr.Code == code
	}
	return false
}
