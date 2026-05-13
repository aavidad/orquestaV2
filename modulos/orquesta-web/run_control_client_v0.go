package orquestaweb

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	orquestamcp "orquesta/modulos/orquesta-mcp"
)

type RunControlClientV0 interface {
	EnviarRunControl(context.Context, WebRunControlCommandV0) (WebRunControlViewModelV0, error)
}

type RESTRunControlClientV0 struct {
	BaseURL    string
	Endpoint   string
	Timeout    time.Duration
	HTTPClient *http.Client
}

type WebRunControlClientErrorV0 struct {
	Code       string
	StatusCode int
}

func (err WebRunControlClientErrorV0) Error() string {
	return err.Code
}

func NewRESTRunControlClientV0(baseURL string, timeout time.Duration) *RESTRunControlClientV0 {
	return &RESTRunControlClientV0{
		BaseURL:  strings.TrimRight(baseURL, "/"),
		Endpoint: WebRunControlInboundEndpointV0,
		Timeout:  timeout,
		HTTPClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (client *RESTRunControlClientV0) EnviarRunControl(
	ctx context.Context,
	command WebRunControlCommandV0,
) (WebRunControlViewModelV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	command = normalizeRunControlCommandV0(command)
	if client.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, client.Timeout)
		defer cancel()
	}
	payload, err := json.Marshal(runControlToolInputV0(command))
	if err != nil {
		return WebRunControlViewModelV0{}, runControlClientErrorV0(WebRunControlErrRespuestaInvalidaV0, 0)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, client.url(), bytes.NewReader(payload))
	if err != nil {
		return WebRunControlViewModelV0{}, runControlClientErrorV0(WebRunControlErrTransporteV0, 0)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set(AppChangeCorrelationV0, command.CorrelationID)

	resp, err := client.httpClient().Do(req)
	if err != nil {
		return WebRunControlViewModelV0{}, runControlClientErrorV0(WebRunControlErrTransporteV0, 0)
	}
	defer resp.Body.Close()
	return decodeRunControlResponseV0(resp, command)
}

func decodeRunControlResponseV0(
	resp *http.Response,
	command WebRunControlCommandV0,
) (WebRunControlViewModelV0, error) {
	if (resp.StatusCode < 200 || resp.StatusCode > 299) && resp.StatusCode != http.StatusBadRequest {
		return WebRunControlViewModelV0{}, runControlClientErrorV0(WebRunControlErrTransporteV0, resp.StatusCode)
	}
	var result orquestamcp.MCPRunControlToolResultV0
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return WebRunControlViewModelV0{}, runControlClientErrorV0(WebRunControlErrRespuestaInvalidaV0, resp.StatusCode)
	}
	if result.Estado == "" {
		return WebRunControlViewModelV0{}, runControlClientErrorV0(WebRunControlErrRespuestaInvalidaV0, resp.StatusCode)
	}
	if resp.StatusCode == http.StatusBadRequest && result.Estado != WebRunControlEstadoErrorV0 {
		return WebRunControlViewModelV0{}, runControlClientErrorV0(WebRunControlErrTransporteV0, resp.StatusCode)
	}
	return NewWebRunControlViewModelV0(command.Locale, result), nil
}

func (client *RESTRunControlClientV0) httpClient() *http.Client {
	if client.HTTPClient != nil {
		return client.HTTPClient
	}
	return &http.Client{Timeout: client.Timeout}
}

func (client *RESTRunControlClientV0) url() string {
	endpoint := client.Endpoint
	if endpoint == "" {
		endpoint = WebRunControlInboundEndpointV0
	}
	return strings.TrimRight(client.BaseURL, "/") + "/" + strings.TrimLeft(endpoint, "/")
}

func runControlClientErrorV0(code string, statusCode int) WebRunControlClientErrorV0 {
	return WebRunControlClientErrorV0{Code: code, StatusCode: statusCode}
}

func IsWebRunControlClientErrorCodeV0(err error, code string) bool {
	var clientErr WebRunControlClientErrorV0
	if errors.As(err, &clientErr) {
		return clientErr.Code == code
	}
	return false
}
