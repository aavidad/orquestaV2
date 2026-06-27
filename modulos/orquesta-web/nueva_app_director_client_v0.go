package orquestaweb

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	orquestafactory "orquesta/modulos/orquesta-factory"
	orquestagoal "orquesta/modulos/orquesta-goal"
)

const (
	ArrancarDirectorAppEndpointV0    = "/api/v0/apps/director"
	PreviewDirectorAppEndpointV0     = "/api/v0/apps/director/preview"
	ArrancarDirectorAppCorrelationV0 = "X-Correlation-ID"
	ArrancarDirectorAppEstadoOKV0    = "ok"
	ArrancarDirectorAppEstadoErrorV0 = "error"
)

type ArrancarDirectorAppClientV0 interface {
	ArrancarDirectorApp(
		ctx context.Context,
		form WebNuevaAppFormV0,
	) (WebNuevaAppViewModelV0, error)
}

type PreviewDirectorAppClientV0 interface {
	PreviewDirectorApp(
		ctx context.Context,
		form WebNuevaAppFormV0,
	) (WebNuevaAppViewModelV0, error)
}

type RESTArrancarDirectorAppClientV0 struct {
	BaseURL    string
	Endpoint   string
	Timeout    time.Duration
	HTTPClient *http.Client
	Limits     WebArrancarDirectorAppLimitsV0
}

type RESTPreviewDirectorAppClientV0 struct {
	BaseURL    string
	Endpoint   string
	Timeout    time.Duration
	HTTPClient *http.Client
	Limits     WebArrancarDirectorAppLimitsV0
}

type WebArrancarDirectorAppLimitsV0 struct {
	MaxBursts            int `json:"max_bursts,omitempty"`
	MaxStepsPerBurst     int `json:"max_steps_per_burst,omitempty"`
	MaxDispatchesPerWait int `json:"max_dispatches_per_wait,omitempty"`
	MaxCommands          int `json:"max_commands,omitempty"`
	MaxOutboxPerCycle    int `json:"max_outbox_per_cycle,omitempty"`
	MaxExternalWaits     int `json:"max_external_waits,omitempty"`
}

func NewRESTArrancarDirectorAppClientV0(
	baseURL string,
	timeout time.Duration,
) *RESTArrancarDirectorAppClientV0 {
	return &RESTArrancarDirectorAppClientV0{
		BaseURL:    normalizeWebRESTBaseURLStringV0(baseURL),
		Endpoint:   ArrancarDirectorAppEndpointV0,
		Timeout:    timeout,
		HTTPClient: newWebLoopbackHTTPClientV0(timeout),
	}
}

func NewRESTPreviewDirectorAppClientV0(
	baseURL string,
	timeout time.Duration,
) *RESTPreviewDirectorAppClientV0 {
	return &RESTPreviewDirectorAppClientV0{
		BaseURL:    normalizeWebRESTBaseURLStringV0(baseURL),
		Endpoint:   PreviewDirectorAppEndpointV0,
		Timeout:    timeout,
		HTTPClient: newWebLoopbackHTTPClientV0(timeout),
	}
}

func (client *RESTArrancarDirectorAppClientV0) ArrancarDirectorApp(
	ctx context.Context,
	form WebNuevaAppFormV0,
) (WebNuevaAppViewModelV0, error) {
	ctx = webContextOrBackgroundV0(ctx)
	request := form.ToAppSpecRequestV0()
	if strings.TrimSpace(request.RequestID) == "" {
		request.RequestID = newWebRequestIDV0()
		form.RequestID = request.RequestID
	}
	if client.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, client.Timeout)
		defer cancel()
	}
	payload, err := json.Marshal(client.payloadV0(form, request))
	if err != nil {
		return WebNuevaAppViewModelV0{}, webNuevaAppClientErrorV0(WebNuevaAppErrRespuestaInvalidaV0, 0)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, client.url(), bytes.NewReader(payload))
	if err != nil {
		return WebNuevaAppViewModelV0{}, webNuevaAppClientErrorV0(WebNuevaAppErrTransporteV0, 0)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set(ArrancarDirectorAppCorrelationV0, request.RequestID)

	resp, err := client.httpClient().Do(httpReq)
	if err != nil {
		return WebNuevaAppViewModelV0{}, webNuevaAppClientErrorV0(WebNuevaAppErrTransporteV0, 0)
	}
	defer resp.Body.Close()
	return decodeArrancarDirectorAppResponseV0(resp, form)
}

func (client *RESTPreviewDirectorAppClientV0) PreviewDirectorApp(
	ctx context.Context,
	form WebNuevaAppFormV0,
) (WebNuevaAppViewModelV0, error) {
	ctx = webContextOrBackgroundV0(ctx)
	request := form.ToAppSpecRequestV0()
	if strings.TrimSpace(request.RequestID) == "" {
		request.RequestID = newWebRequestIDV0()
		form.RequestID = request.RequestID
	}
	if client.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, client.Timeout)
		defer cancel()
	}
	payload, err := json.Marshal(arrancarDirectorAppPayloadV0(form, request, client.Limits))
	if err != nil {
		return WebNuevaAppViewModelV0{}, webNuevaAppClientErrorV0(WebNuevaAppErrRespuestaInvalidaV0, 0)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, client.url(), bytes.NewReader(payload))
	if err != nil {
		return WebNuevaAppViewModelV0{}, webNuevaAppClientErrorV0(WebNuevaAppErrTransporteV0, 0)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set(ArrancarDirectorAppCorrelationV0, request.RequestID)

	resp, err := client.httpClient().Do(httpReq)
	if err != nil {
		return WebNuevaAppViewModelV0{}, webNuevaAppClientErrorV0(WebNuevaAppErrTransporteV0, 0)
	}
	defer resp.Body.Close()
	return decodePreviewDirectorAppResponseV0(resp, form)
}

func (client *RESTArrancarDirectorAppClientV0) payloadV0(
	form WebNuevaAppFormV0,
	request orquestafactory.AppSpecRequestV0,
) arrancarDirectorAppRequestEnvelopeV0 {
	return arrancarDirectorAppPayloadV0(form, request, client.Limits)
}

func arrancarDirectorAppPayloadV0(
	form WebNuevaAppFormV0,
	request orquestafactory.AppSpecRequestV0,
	limits WebArrancarDirectorAppLimitsV0,
) arrancarDirectorAppRequestEnvelopeV0 {
	return arrancarDirectorAppRequestEnvelopeV0{
		RequestID:             request.RequestID,
		CorrelationID:         request.RequestID,
		DirectorExecutionMode: strings.TrimSpace(form.DirectorExecutionMode),
		AppSpecRequest:        request,
		MaxBursts:             limits.MaxBursts,
		MaxStepsPerBurst:      limits.MaxStepsPerBurst,
		MaxDispatchesPerWait:  limits.MaxDispatchesPerWait,
		MaxCommands:           limits.MaxCommands,
		MaxOutboxPerCycle:     limits.MaxOutboxPerCycle,
		MaxExternalWaits:      limits.MaxExternalWaits,
	}
}

func decodeArrancarDirectorAppResponseV0(
	resp *http.Response,
	form WebNuevaAppFormV0,
) (WebNuevaAppViewModelV0, error) {
	var result WebArrancarDirectorAppResultV0
	if !decodeWebHTTPJSONResponseV0(resp, &result) {
		if resp.StatusCode < 200 || resp.StatusCode > 299 {
			return WebNuevaAppViewModelV0{}, webNuevaAppClientErrorV0(WebNuevaAppErrTransporteV0, resp.StatusCode)
		}
		return WebNuevaAppViewModelV0{}, webNuevaAppClientErrorV0(WebNuevaAppErrRespuestaInvalidaV0, resp.StatusCode)
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		if strings.TrimSpace(result.Estado) == ArrancarDirectorAppEstadoErrorV0 {
			return NewWebNuevaAppDirectorErrorViewModelV0(form, result), nil
		}
		return WebNuevaAppViewModelV0{}, webNuevaAppClientErrorV0(WebNuevaAppErrTransporteV0, resp.StatusCode)
	}
	switch strings.TrimSpace(result.Estado) {
	case ArrancarDirectorAppEstadoOKV0:
		if strings.TrimSpace(result.RunRef) == "" {
			return WebNuevaAppViewModelV0{}, webNuevaAppClientErrorV0(WebNuevaAppErrRespuestaInvalidaV0, resp.StatusCode)
		}
		return NewWebNuevaAppDirectorViewModelV0(form, result), nil
	case ArrancarDirectorAppEstadoErrorV0:
		return NewWebNuevaAppDirectorErrorViewModelV0(form, result), nil
	default:
		return WebNuevaAppViewModelV0{}, webNuevaAppClientErrorV0(WebNuevaAppErrRespuestaInvalidaV0, resp.StatusCode)
	}
}

func decodePreviewDirectorAppResponseV0(
	resp *http.Response,
	form WebNuevaAppFormV0,
) (WebNuevaAppViewModelV0, error) {
	var result WebPreviewDirectorAppResultV0
	if !decodeWebHTTPJSONResponseV0(resp, &result) {
		if resp.StatusCode < 200 || resp.StatusCode > 299 {
			return WebNuevaAppViewModelV0{}, webNuevaAppClientErrorV0(WebNuevaAppErrTransporteV0, resp.StatusCode)
		}
		return WebNuevaAppViewModelV0{}, webNuevaAppClientErrorV0(WebNuevaAppErrRespuestaInvalidaV0, resp.StatusCode)
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		if strings.TrimSpace(result.Estado) == ArrancarDirectorAppEstadoErrorV0 {
			return NewWebNuevaAppGoalPreviewErrorViewModelV0(form, result), nil
		}
		return WebNuevaAppViewModelV0{}, webNuevaAppClientErrorV0(WebNuevaAppErrTransporteV0, resp.StatusCode)
	}
	switch strings.TrimSpace(result.Estado) {
	case ArrancarDirectorAppEstadoOKV0:
		if strings.TrimSpace(result.GoalSpec.GoalRef) == "" {
			return WebNuevaAppViewModelV0{}, webNuevaAppClientErrorV0(WebNuevaAppErrRespuestaInvalidaV0, resp.StatusCode)
		}
		return NewWebNuevaAppGoalPreviewViewModelV0(form, result), nil
	case ArrancarDirectorAppEstadoErrorV0:
		return NewWebNuevaAppGoalPreviewErrorViewModelV0(form, result), nil
	default:
		return WebNuevaAppViewModelV0{}, webNuevaAppClientErrorV0(WebNuevaAppErrRespuestaInvalidaV0, resp.StatusCode)
	}
}

func (client *RESTArrancarDirectorAppClientV0) httpClient() *http.Client {
	return webHTTPClientWithRedirectPolicyV0(client.HTTPClient, client.Timeout, client.BaseURL)
}

func (client *RESTPreviewDirectorAppClientV0) httpClient() *http.Client {
	return webHTTPClientWithRedirectPolicyV0(client.HTTPClient, client.Timeout, client.BaseURL)
}

func (client *RESTArrancarDirectorAppClientV0) url() string {
	endpoint := client.Endpoint
	if endpoint == "" {
		endpoint = ArrancarDirectorAppEndpointV0
	}
	return webRESTEndpointURLV0(client.BaseURL, endpoint, ArrancarDirectorAppEndpointV0)
}

func (client *RESTPreviewDirectorAppClientV0) url() string {
	endpoint := client.Endpoint
	if endpoint == "" {
		endpoint = PreviewDirectorAppEndpointV0
	}
	return webRESTEndpointURLV0(client.BaseURL, endpoint, PreviewDirectorAppEndpointV0)
}

func IsWebArrancarDirectorClientErrorCodeV0(err error, code string) bool {
	var clientErr WebNuevaAppClientErrorV0
	if errors.As(err, &clientErr) {
		return clientErr.Code == code
	}
	return false
}

type arrancarDirectorAppRequestEnvelopeV0 struct {
	RequestID             string                           `json:"request_id,omitempty"`
	CorrelationID         string                           `json:"correlation_id,omitempty"`
	DirectorExecutionMode string                           `json:"director_execution_mode,omitempty"`
	AppSpecRequest        orquestafactory.AppSpecRequestV0 `json:"app_spec_request"`
	MaxBursts             int                              `json:"max_bursts,omitempty"`
	MaxStepsPerBurst      int                              `json:"max_steps_per_burst,omitempty"`
	MaxDispatchesPerWait  int                              `json:"max_dispatches_per_wait,omitempty"`
	MaxCommands           int                              `json:"max_commands,omitempty"`
	MaxOutboxPerCycle     int                              `json:"max_outbox_per_cycle,omitempty"`
	MaxExternalWaits      int                              `json:"max_external_waits,omitempty"`
}

type WebPreviewDirectorAppResultV0 struct {
	Estado                string                           `json:"estado"`
	RequestID             string                           `json:"request_id,omitempty"`
	CorrelationID         string                           `json:"correlation_id,omitempty"`
	RunRef                string                           `json:"run_ref,omitempty"`
	DirectorExecutionMode string                           `json:"director_execution_mode,omitempty"`
	GoalSpec              orquestagoal.GoalWorkSpecV0      `json:"goal_spec,omitempty"`
	WriteSet              []string                         `json:"write_set,omitempty"`
	RequiredTests         []string                         `json:"required_tests,omitempty"`
	Estimate              WebNuevaAppGoalPreviewEstimateV0 `json:"estimate,omitempty"`
	GoalSpecIssues        []orquestagoal.GoalWorkIssueV0   `json:"goal_spec_issues,omitempty"`
	Errores               []WebNuevaAppIssueV0             `json:"errores_publicos,omitempty"`
	EvidenceRefs          []string                         `json:"evidence_refs,omitempty"`
}
