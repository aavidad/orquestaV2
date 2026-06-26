package orquestaweb

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"
)

type DirectorStatsClientV0 interface {
	ConsultarDirectorStats(
		ctx context.Context,
		query WebDirectorStatsQueryV0,
	) (WebDirectorStatsViewModelV0, error)
}

type ConsultarDirectorStatsClientV0 = DirectorStatsClientV0

type RESTDirectorStatsClientV0 struct {
	BaseURL    string
	Endpoint   string
	Timeout    time.Duration
	HTTPClient *http.Client
}

type RESTConsultarDirectorStatsClientV0 = RESTDirectorStatsClientV0

type WebDirectorStatsClientErrorV0 struct {
	Code       string
	StatusCode int
}

func (err WebDirectorStatsClientErrorV0) Error() string {
	return err.Code
}

func NewRESTDirectorStatsClientV0(baseURL string, timeout time.Duration) *RESTDirectorStatsClientV0 {
	return &RESTDirectorStatsClientV0{
		BaseURL:    normalizeWebRESTBaseURLStringV0(baseURL),
		Endpoint:   WebDirectorStatsInboundEndpointV0,
		Timeout:    timeout,
		HTTPClient: newWebLoopbackHTTPClientV0(timeout),
	}
}

func NewRESTConsultarDirectorStatsClientV0(
	baseURL string,
	timeout time.Duration,
) *RESTConsultarDirectorStatsClientV0 {
	return NewRESTDirectorStatsClientV0(baseURL, timeout)
}

func (client *RESTDirectorStatsClientV0) ConsultarDirectorStats(
	ctx context.Context,
	query WebDirectorStatsQueryV0,
) (WebDirectorStatsViewModelV0, error) {
	ctx = webContextOrBackgroundV0(ctx)
	query = normalizeDirectorStatsQueryV0(query)
	if query.RunRef == "" && query.ExternalJobRef == "" {
		return WebDirectorStatsViewModelV0{}, directorStatsClientErrorV0(WebDirectorStatsErrRunRefRequeridoV0, 0)
	}
	if client.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, client.Timeout)
		defer cancel()
	}
	payload, err := json.Marshal(query)
	if err != nil {
		return WebDirectorStatsViewModelV0{}, directorStatsClientErrorV0(WebDirectorStatsErrRespuestaInvalidaV0, 0)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, client.url(), bytes.NewReader(payload))
	if err != nil {
		return WebDirectorStatsViewModelV0{}, directorStatsClientErrorV0(WebDirectorStatsErrTransporteV0, 0)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set(WebDirectorStatsCorrelationHeaderV0, firstDirectorStatsNonEmptyV0(query.CorrelationID, query.RequestID))

	resp, err := client.httpClient().Do(req)
	if err != nil {
		return WebDirectorStatsViewModelV0{}, directorStatsClientErrorV0(WebDirectorStatsErrTransporteV0, 0)
	}
	defer resp.Body.Close()
	return decodeDirectorStatsResponseV0(resp, query)
}

func decodeDirectorStatsResponseV0(
	resp *http.Response,
	query WebDirectorStatsQueryV0,
) (WebDirectorStatsViewModelV0, error) {
	if (resp.StatusCode < 200 || resp.StatusCode > 299) && resp.StatusCode != http.StatusBadRequest {
		discardWebHTTPResponseBodyV0(resp)
		return WebDirectorStatsViewModelV0{}, directorStatsClientErrorV0(WebDirectorStatsErrTransporteV0, resp.StatusCode)
	}
	var envelope directorStatsEnvelopeV0
	if !decodeWebHTTPJSONResponseV0(resp, &envelope) {
		return WebDirectorStatsViewModelV0{}, directorStatsClientErrorV0(WebDirectorStatsErrRespuestaInvalidaV0, resp.StatusCode)
	}
	result := envelope.ResultV0()
	if resp.StatusCode == http.StatusBadRequest && result.Estado != WebDirectorStatsInboundEstadoErrorV0 {
		return WebDirectorStatsViewModelV0{}, directorStatsClientErrorV0(WebDirectorStatsErrTransporteV0, resp.StatusCode)
	}
	if result.Estado == "" && result.Stats == nil && result.ExternalJob == nil && len(result.Errores) == 0 {
		return WebDirectorStatsViewModelV0{}, directorStatsClientErrorV0(WebDirectorStatsErrRespuestaInvalidaV0, resp.StatusCode)
	}
	return NewWebDirectorStatsPanelV0(query.Locale, result), nil
}

func (client *RESTDirectorStatsClientV0) httpClient() *http.Client {
	return webHTTPClientWithRedirectPolicyV0(client.HTTPClient, client.Timeout, client.BaseURL)
}

func (client *RESTDirectorStatsClientV0) url() string {
	endpoint := client.Endpoint
	if endpoint == "" {
		endpoint = WebDirectorStatsInboundEndpointV0
	}
	return webRESTEndpointURLV0(client.BaseURL, endpoint, WebDirectorStatsInboundEndpointV0)
}

func normalizeDirectorStatsQueryV0(query WebDirectorStatsQueryV0) WebDirectorStatsQueryV0 {
	query.RequestID = trimDirectorStatsV0(query.RequestID)
	if query.RequestID == "" {
		query.RequestID = newWebRequestIDV0()
	}
	query.CorrelationID = firstDirectorStatsNonEmptyV0(query.CorrelationID, query.RequestID)
	query.Locale = normalizeDirectorStatsLocaleV0(query.Locale)
	query.RunRef = trimDirectorStatsV0(query.RunRef)
	query.AppRef = trimDirectorStatsV0(query.AppRef)
	query.ExternalJobRef = trimDirectorStatsV0(query.ExternalJobRef)
	query.OccurredAt = trimDirectorStatsV0(query.OccurredAt)
	if query.RunRef != "" || query.ExternalJobRef != "" {
		query.IncludeProcessRefs = true
		query.IncludeAgentProgress = true
	}
	return query
}

func directorStatsClientErrorV0(code string, statusCode int) WebDirectorStatsClientErrorV0 {
	return WebDirectorStatsClientErrorV0{Code: code, StatusCode: statusCode}
}

func IsWebDirectorStatsClientErrorCodeV0(err error, code string) bool {
	var clientErr WebDirectorStatsClientErrorV0
	if errors.As(err, &clientErr) {
		return clientErr.Code == code
	}
	return false
}

type directorStatsEnvelopeV0 struct {
	Estado        string                          `json:"estado,omitempty"`
	RequestID     string                          `json:"request_id,omitempty"`
	CorrelationID string                          `json:"correlation_id,omitempty"`
	RunRef        string                          `json:"run_ref,omitempty"`
	ExternalJob   *WebDirectorExternalJobV0       `json:"external_job,omitempty"`
	Goal          *WebDirectorGoalStatsContractV0 `json:"goal,omitempty"`
	Stats         *WebDirectorRunStatsContractV0  `json:"stats,omitempty"`
	DirectorStats *WebDirectorRunStatsContractV0  `json:"director_stats,omitempty"`
	Errores       []WebDirectorStatsPublicIssueV0 `json:"errores_publicos,omitempty"`
}

func (envelope directorStatsEnvelopeV0) ResultV0() WebDirectorStatsInboundResultV0 {
	stats := envelope.Stats
	if stats == nil {
		stats = envelope.DirectorStats
	}
	return WebDirectorStatsInboundResultV0{
		Estado:        envelope.Estado,
		RequestID:     envelope.RequestID,
		CorrelationID: envelope.CorrelationID,
		RunRef:        envelope.RunRef,
		ExternalJob:   envelope.ExternalJob,
		Goal:          envelope.Goal,
		Stats:         stats,
		Errores:       envelope.Errores,
	}
}
