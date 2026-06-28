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
	publicidentity "orquesta/modulos/orquesta-server/publicidentity"
)

type RunQueueClientV0 interface {
	ConsultarRunQueue(context.Context, WebRunQueueQueryV0) (WebRunQueueViewModelV0, error)
}

type RESTRunQueueClientV0 struct {
	BaseURL    string
	Endpoint   string
	Timeout    time.Duration
	HTTPClient *http.Client
}

type WebRunQueueClientErrorV0 struct {
	Code       string
	StatusCode int
}

func (err WebRunQueueClientErrorV0) Error() string {
	return err.Code
}

func NewRESTRunQueueClientV0(baseURL string, timeout time.Duration) *RESTRunQueueClientV0 {
	return &RESTRunQueueClientV0{
		BaseURL:    normalizeWebRESTBaseURLStringV0(baseURL),
		Endpoint:   WebRunQueueInboundEndpointV0,
		Timeout:    timeout,
		HTTPClient: newWebLoopbackHTTPClientV0(timeout),
	}
}

func (client *RESTRunQueueClientV0) ConsultarRunQueue(
	ctx context.Context,
	query WebRunQueueQueryV0,
) (WebRunQueueViewModelV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	query = normalizeRunQueueQueryV0(query)
	if client.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, client.Timeout)
		defer cancel()
	}
	payload, err := json.Marshal(runQueueToolInputV0(query))
	if err != nil {
		return WebRunQueueViewModelV0{}, runQueueClientErrorV0(WebRunQueueErrRespuestaInvalidaV0, 0)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, client.url(), bytes.NewReader(payload))
	if err != nil {
		return WebRunQueueViewModelV0{}, runQueueClientErrorV0(WebRunQueueErrTransporteV0, 0)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set(AppChangeCorrelationV0, query.CorrelationID)
	if strings.TrimSpace(query.IdempotencyKey) != "" {
		req.Header.Set(AppChangeIdempotencyKeyV0, strings.TrimSpace(query.IdempotencyKey))
	}

	resp, err := client.httpClient().Do(req)
	if err != nil {
		return WebRunQueueViewModelV0{}, runQueueClientErrorV0(WebRunQueueErrTransporteV0, 0)
	}
	defer resp.Body.Close()
	return decodeRunQueueResponseV0(resp, query)
}

func normalizeRunQueueQueryV0(query WebRunQueueQueryV0) WebRunQueueQueryV0 {
	query.RequestID = trimV0(query.RequestID)
	if query.RequestID == "" {
		query.RequestID = newWebRequestIDV0()
	}
	query.CorrelationID = firstDirectorStatsNonEmptyV0(query.CorrelationID, query.RequestID)
	mutating := strings.ToLower(trimV0(query.Action)) == WebRunQueueActionSetV0
	identity := publicidentity.NormalizePublicIdentityV0(publicidentity.PublicIdentityInputV0{
		RequestID:      query.RequestID,
		CorrelationID:  query.CorrelationID,
		IdempotencyKey: query.IdempotencyKey,
		Mutating:       mutating,
	})
	query.CorrelationID = identity.CorrelationID
	query.IdempotencyKey = identity.IdempotencyKey
	query.Locale = normalizeDirectorStatsLocaleV0(query.Locale)
	query.Action = strings.ToLower(trimV0(query.Action))
	if query.Action == "" {
		query.Action = WebRunQueueActionRankV0
	}
	query.QueueRef = trimV0(query.QueueRef)
	query.AppRefs = compactStringsV0(query.AppRefs)
	query.RunRef = trimV0(query.RunRef)
	query.AppRef = trimV0(query.AppRef)
	query.Status = trimV0(query.Status)
	query.RequestedBy = trimV0(query.RequestedBy)
	query.Reason = trimV0(query.Reason)
	query.OccurredAt = trimV0(query.OccurredAt)
	return query
}

func runQueueToolInputV0(query WebRunQueueQueryV0) orquestamcp.MCPRunQueuePriorityToolInputV0 {
	return orquestamcp.MCPRunQueuePriorityToolInputV0{
		RequestID:      query.RequestID,
		CorrelationID:  query.CorrelationID,
		Action:         query.Action,
		QueueRef:       query.QueueRef,
		AppRefs:        query.AppRefs,
		RunRef:         query.RunRef,
		AppRef:         query.AppRef,
		Status:         query.Status,
		PriorityScore:  query.PriorityScore,
		RequestedBy:    query.RequestedBy,
		Reason:         query.Reason,
		IdempotencyKey: query.IdempotencyKey,
		Limit:          query.Limit,
		OccurredAt:     query.OccurredAt,
	}
}

func decodeRunQueueResponseV0(
	resp *http.Response,
	query WebRunQueueQueryV0,
) (WebRunQueueViewModelV0, error) {
	var result orquestamcp.MCPRunQueuePriorityToolResultV0
	if !decodeWebHTTPJSONResponseV0(resp, &result) {
		if resp.StatusCode < 200 || resp.StatusCode > 299 {
			return WebRunQueueViewModelV0{}, runQueueClientErrorV0(WebRunQueueErrTransporteV0, resp.StatusCode)
		}
		return WebRunQueueViewModelV0{}, runQueueClientErrorV0(WebRunQueueErrRespuestaInvalidaV0, resp.StatusCode)
	}
	if result.Estado == "" {
		return WebRunQueueViewModelV0{}, runQueueClientErrorV0(WebRunQueueErrRespuestaInvalidaV0, resp.StatusCode)
	}
	if (resp.StatusCode < 200 || resp.StatusCode > 299) && result.Estado != WebRunQueueEstadoErrorV0 {
		return WebRunQueueViewModelV0{}, runQueueClientErrorV0(WebRunQueueErrTransporteV0, resp.StatusCode)
	}
	return NewWebRunQueueViewModelV0(query.Locale, result), nil
}

func (client *RESTRunQueueClientV0) httpClient() *http.Client {
	return webHTTPClientWithRedirectPolicyV0(client.HTTPClient, client.Timeout, client.BaseURL)
}

func (client *RESTRunQueueClientV0) url() string {
	endpoint := client.Endpoint
	if endpoint == "" {
		endpoint = WebRunQueueInboundEndpointV0
	}
	return webRESTEndpointURLV0(client.BaseURL, endpoint, WebRunQueueInboundEndpointV0)
}

func runQueueClientErrorV0(code string, statusCode int) WebRunQueueClientErrorV0 {
	return WebRunQueueClientErrorV0{Code: code, StatusCode: statusCode}
}

func IsWebRunQueueClientErrorCodeV0(err error, code string) bool {
	var clientErr WebRunQueueClientErrorV0
	if errors.As(err, &clientErr) {
		return clientErr.Code == code
	}
	return false
}
