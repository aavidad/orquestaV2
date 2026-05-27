package orquestaweb

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"time"

	orquestamcp "orquesta/modulos/orquesta-mcp"
	publicidentity "orquesta/modulos/orquesta-server/publicidentity"
)

const (
	AppChangeEndpointV0        = "/api/v0/apps/change"
	AppChangeCorrelationV0     = publicidentity.PublicCorrelationHeaderV0
	AppChangeIdempotencyKeyV0  = publicidentity.PublicIdempotencyKeyHeaderV0
	WebAppChangeErrTransportV0 = "app_change_error_transporte"
	WebAppChangeErrResponseV0  = "app_change_respuesta_invalida"
)

type AppChangeClientV0 interface {
	RequestAppChange(context.Context, WebAppChangeFormV0) (WebAppChangeViewModelV0, error)
}

type RESTAppChangeClientV0 struct {
	BaseURL    string
	Endpoint   string
	Timeout    time.Duration
	HTTPClient *http.Client
}

func NewRESTAppChangeClientV0(baseURL string, timeout time.Duration) *RESTAppChangeClientV0 {
	return &RESTAppChangeClientV0{
		BaseURL:    normalizeWebRESTBaseURLStringV0(baseURL),
		Endpoint:   AppChangeEndpointV0,
		Timeout:    timeout,
		HTTPClient: newWebLoopbackHTTPClientV0(timeout),
	}
}

func (client *RESTAppChangeClientV0) RequestAppChange(
	ctx context.Context,
	form WebAppChangeFormV0,
) (WebAppChangeViewModelV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	input := form.ToMCPRequestAppChangeInputV0()
	if input.RequestID == "" {
		input.RequestID = newWebRequestIDV0()
		input.AppChangeRequest.RequestID = input.RequestID
	}
	if input.CorrelationID == "" {
		input.CorrelationID = input.RequestID
		input.AppChangeRequest.CorrelationID = input.RequestID
	}
	if client.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, client.Timeout)
		defer cancel()
	}
	payload, err := json.Marshal(input)
	if err != nil {
		return WebAppChangeViewModelV0{}, webNuevaAppClientErrorV0(WebAppChangeErrResponseV0, 0)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, client.url(), bytes.NewReader(payload))
	if err != nil {
		return WebAppChangeViewModelV0{}, webNuevaAppClientErrorV0(WebAppChangeErrTransportV0, 0)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set(AppChangeCorrelationV0, input.CorrelationID)

	resp, err := client.httpClient().Do(req)
	if err != nil {
		return WebAppChangeViewModelV0{}, webNuevaAppClientErrorV0(WebAppChangeErrTransportV0, 0)
	}
	defer resp.Body.Close()
	return decodeAppChangeResponseV0(resp)
}

func decodeAppChangeResponseV0(resp *http.Response) (WebAppChangeViewModelV0, error) {
	if (resp.StatusCode < 200 || resp.StatusCode > 299) && resp.StatusCode != http.StatusBadRequest {
		discardWebHTTPResponseBodyV0(resp)
		return WebAppChangeViewModelV0{}, webNuevaAppClientErrorV0(WebAppChangeErrTransportV0, resp.StatusCode)
	}
	var result orquestamcp.MCPRequestAppChangeToolResultV0
	if !decodeWebHTTPJSONResponseV0(resp, &result) {
		return WebAppChangeViewModelV0{}, webNuevaAppClientErrorV0(WebAppChangeErrResponseV0, resp.StatusCode)
	}
	if result.Estado == "" {
		return WebAppChangeViewModelV0{}, webNuevaAppClientErrorV0(WebAppChangeErrResponseV0, resp.StatusCode)
	}
	return NewWebAppChangeViewModelV0(result), nil
}

func (client *RESTAppChangeClientV0) httpClient() *http.Client {
	return webHTTPClientWithRedirectPolicyV0(client.HTTPClient, client.Timeout, client.BaseURL)
}

func (client *RESTAppChangeClientV0) url() string {
	endpoint := client.Endpoint
	if endpoint == "" {
		endpoint = AppChangeEndpointV0
	}
	return webRESTEndpointURLV0(client.BaseURL, endpoint, AppChangeEndpointV0)
}
