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
)

const (
	ArrancarDirectorAppEndpointV0    = "/api/v0/apps/director"
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

type RESTArrancarDirectorAppClientV0 struct {
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
		BaseURL:  strings.TrimRight(baseURL, "/"),
		Endpoint: ArrancarDirectorAppEndpointV0,
		Timeout:  timeout,
		HTTPClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (client *RESTArrancarDirectorAppClientV0) ArrancarDirectorApp(
	ctx context.Context,
	form WebNuevaAppFormV0,
) (WebNuevaAppViewModelV0, error) {
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
	payload, err := json.Marshal(client.payloadV0(request))
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

func (client *RESTArrancarDirectorAppClientV0) payloadV0(
	request orquestafactory.AppSpecRequestV0,
) arrancarDirectorAppRequestEnvelopeV0 {
	return arrancarDirectorAppRequestEnvelopeV0{
		RequestID:            request.RequestID,
		CorrelationID:        request.RequestID,
		AppSpecRequest:       request,
		MaxBursts:            client.Limits.MaxBursts,
		MaxStepsPerBurst:     client.Limits.MaxStepsPerBurst,
		MaxDispatchesPerWait: client.Limits.MaxDispatchesPerWait,
		MaxCommands:          client.Limits.MaxCommands,
		MaxOutboxPerCycle:    client.Limits.MaxOutboxPerCycle,
		MaxExternalWaits:     client.Limits.MaxExternalWaits,
	}
}

func decodeArrancarDirectorAppResponseV0(
	resp *http.Response,
	form WebNuevaAppFormV0,
) (WebNuevaAppViewModelV0, error) {
	if (resp.StatusCode < 200 || resp.StatusCode > 299) && resp.StatusCode != http.StatusBadRequest {
		return WebNuevaAppViewModelV0{}, webNuevaAppClientErrorV0(WebNuevaAppErrTransporteV0, resp.StatusCode)
	}
	var result WebArrancarDirectorAppResultV0
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
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

func (client *RESTArrancarDirectorAppClientV0) httpClient() *http.Client {
	if client.HTTPClient != nil {
		return client.HTTPClient
	}
	return &http.Client{Timeout: client.Timeout}
}

func (client *RESTArrancarDirectorAppClientV0) url() string {
	endpoint := client.Endpoint
	if endpoint == "" {
		endpoint = ArrancarDirectorAppEndpointV0
	}
	return strings.TrimRight(client.BaseURL, "/") + "/" + strings.TrimLeft(endpoint, "/")
}

func IsWebArrancarDirectorClientErrorCodeV0(err error, code string) bool {
	var clientErr WebNuevaAppClientErrorV0
	if errors.As(err, &clientErr) {
		return clientErr.Code == code
	}
	return false
}

type arrancarDirectorAppRequestEnvelopeV0 struct {
	RequestID            string                           `json:"request_id,omitempty"`
	CorrelationID        string                           `json:"correlation_id,omitempty"`
	AppSpecRequest       orquestafactory.AppSpecRequestV0 `json:"app_spec_request"`
	MaxBursts            int                              `json:"max_bursts,omitempty"`
	MaxStepsPerBurst     int                              `json:"max_steps_per_burst,omitempty"`
	MaxDispatchesPerWait int                              `json:"max_dispatches_per_wait,omitempty"`
	MaxCommands          int                              `json:"max_commands,omitempty"`
	MaxOutboxPerCycle    int                              `json:"max_outbox_per_cycle,omitempty"`
	MaxExternalWaits     int                              `json:"max_external_waits,omitempty"`
}
