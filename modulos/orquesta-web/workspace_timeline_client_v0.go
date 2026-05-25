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
	orquestaobservability "orquesta/modulos/orquesta-observability"
)

const (
	WebWorkspaceTimelineInboundEndpointV0 = orquestamcp.MCPWorkspaceTimelineEndpointV0
	WebWorkspaceTimelineErrTransporteV0   = "workspace_timeline_transporte_error"
	WebWorkspaceTimelineErrRespuestaV0    = "workspace_timeline_respuesta_invalida"
)

type WorkspaceTimelineClientV0 interface {
	ConsultarWorkspaceTimeline(
		context.Context,
		orquestaobservability.WorkspaceTimelineQueryV0,
	) (WebWorkspaceTimelineViewModelV0, error)
}

type RESTWorkspaceTimelineClientV0 struct {
	BaseURL    string
	Endpoint   string
	Timeout    time.Duration
	HTTPClient *http.Client
}

type WebWorkspaceTimelineClientErrorV0 struct {
	Code       string
	StatusCode int
}

func (err WebWorkspaceTimelineClientErrorV0) Error() string {
	return err.Code
}

func NewRESTWorkspaceTimelineClientV0(baseURL string, timeout time.Duration) *RESTWorkspaceTimelineClientV0 {
	return &RESTWorkspaceTimelineClientV0{
		BaseURL:  strings.TrimRight(baseURL, "/"),
		Endpoint: WebWorkspaceTimelineInboundEndpointV0,
		Timeout:  timeout,
		HTTPClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (client *RESTWorkspaceTimelineClientV0) ConsultarWorkspaceTimeline(
	ctx context.Context,
	query orquestaobservability.WorkspaceTimelineQueryV0,
) (WebWorkspaceTimelineViewModelV0, error) {
	if client.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, client.Timeout)
		defer cancel()
	}
	payload, err := json.Marshal(query)
	if err != nil {
		return WebWorkspaceTimelineViewModelV0{}, workspaceTimelineClientErrorV0(WebWorkspaceTimelineErrRespuestaV0, 0)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, client.url(), bytes.NewReader(payload))
	if err != nil {
		return WebWorkspaceTimelineViewModelV0{}, workspaceTimelineClientErrorV0(WebWorkspaceTimelineErrTransporteV0, 0)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Correlation-ID", strings.TrimSpace(query.CorrelationID))
	resp, err := client.httpClient().Do(req)
	if err != nil {
		return WebWorkspaceTimelineViewModelV0{}, workspaceTimelineClientErrorV0(WebWorkspaceTimelineErrTransporteV0, 0)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return WebWorkspaceTimelineViewModelV0{}, workspaceTimelineClientErrorV0(WebWorkspaceTimelineErrTransporteV0, resp.StatusCode)
	}
	var timeline orquestaobservability.WorkspaceTimelineV0
	if err := json.NewDecoder(resp.Body).Decode(&timeline); err != nil {
		return WebWorkspaceTimelineViewModelV0{}, workspaceTimelineClientErrorV0(WebWorkspaceTimelineErrRespuestaV0, resp.StatusCode)
	}
	if err := orquestaobservability.ValidateWorkspaceTimelineV0(timeline); err != nil {
		return WebWorkspaceTimelineViewModelV0{}, workspaceTimelineClientErrorV0(WebWorkspaceTimelineErrRespuestaV0, resp.StatusCode)
	}
	return NewWebWorkspaceTimelineV0(query.Locale, timeline), nil
}

func (client *RESTWorkspaceTimelineClientV0) httpClient() *http.Client {
	if client.HTTPClient != nil {
		return client.HTTPClient
	}
	return &http.Client{Timeout: client.Timeout}
}

func (client *RESTWorkspaceTimelineClientV0) url() string {
	endpoint := client.Endpoint
	if endpoint == "" {
		endpoint = WebWorkspaceTimelineInboundEndpointV0
	}
	return strings.TrimRight(client.BaseURL, "/") + "/" + strings.TrimLeft(endpoint, "/")
}

func workspaceTimelineClientErrorV0(code string, statusCode int) WebWorkspaceTimelineClientErrorV0 {
	return WebWorkspaceTimelineClientErrorV0{Code: code, StatusCode: statusCode}
}

func IsWebWorkspaceTimelineClientErrorCodeV0(err error, code string) bool {
	var clientErr WebWorkspaceTimelineClientErrorV0
	if errors.As(err, &clientErr) {
		return clientErr.Code == code
	}
	return false
}
