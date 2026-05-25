package orquestaweb

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestamcp "orquesta/modulos/orquesta-mcp"
)

type AutoprogrammingPrepareRunClientV0 interface {
	PrepararAutoprogrammingRun(context.Context, WebAutoprogrammingPrepareRunCommandV0) (WebAutoprogrammingPrepareRunViewModelV0, error)
}

type AutoprogrammingStatusClientV0 interface {
	ConsultarAutoprogrammingStatus(context.Context, WebAutoprogrammingStatusQueryV0) (WebAutoprogrammingStatusViewModelV0, error)
}

type RESTAutoprogrammingPrepareRunClientV0 struct {
	BaseURL    string
	Endpoint   string
	Timeout    time.Duration
	HTTPClient *http.Client
}

type WebAutoprogrammingPrepareRunClientErrorV0 struct {
	Code       string
	StatusCode int
}

func (err WebAutoprogrammingPrepareRunClientErrorV0) Error() string {
	return err.Code
}

func NewRESTAutoprogrammingPrepareRunClientV0(baseURL string, timeout time.Duration) *RESTAutoprogrammingPrepareRunClientV0 {
	return &RESTAutoprogrammingPrepareRunClientV0{
		BaseURL:  strings.TrimRight(baseURL, "/"),
		Endpoint: WebAutoprogrammingPrepareRunInboundEndpointV0,
		Timeout:  timeout,
		HTTPClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (client *RESTAutoprogrammingPrepareRunClientV0) PrepararAutoprogrammingRun(ctx context.Context, command WebAutoprogrammingPrepareRunCommandV0) (WebAutoprogrammingPrepareRunViewModelV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	command = normalizeAutoprogrammingPrepareRunCommandV0(command)
	if client.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, client.Timeout)
		defer cancel()
	}
	resp, err := client.postJSONV0(
		ctx,
		client.url(),
		autoprogrammingPrepareRunToolInputV0(command),
		command.CorrelationID,
		WebAutoprogrammingPrepareRunErrRespuestaInvalidaV0,
		WebAutoprogrammingPrepareRunErrTransporteV0,
	)
	if err != nil {
		return WebAutoprogrammingPrepareRunViewModelV0{}, err
	}
	defer resp.Body.Close()
	return decodeAutoprogrammingPrepareRunResponseV0(resp, command)
}

func normalizeAutoprogrammingPrepareRunCommandV0(command WebAutoprogrammingPrepareRunCommandV0) WebAutoprogrammingPrepareRunCommandV0 {
	command.RequestID = trimV0(command.RequestID)
	if command.RequestID == "" {
		command.RequestID = newWebRequestIDV0()
	}
	command.CorrelationID = firstDirectorStatsNonEmptyV0(command.CorrelationID, command.RequestID)
	command.Locale = normalizeDirectorStatsLocaleV0(command.Locale)
	command.OccurredAt = trimV0(command.OccurredAt)
	command.RequestedBy = trimV0(command.RequestedBy)
	command.RequestRef = firstDirectorStatsNonEmptyV0(command.RequestRef, command.RequestID)
	command.ProjectRef = trimV0(command.ProjectRef)
	command.WorktreeRef = trimV0(command.WorktreeRef)
	command.BranchRef = trimV0(command.BranchRef)
	command.WriteSet = compactStringsV0(command.WriteSet)
	command.RequiredTests = compactStringsV0(command.RequiredTests)
	command.Tasks = compactAutoprogrammingTasksV0(command.Tasks)
	return command
}

func autoprogrammingPrepareRunToolInputV0(command WebAutoprogrammingPrepareRunCommandV0) orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0 {
	return orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0{
		RequestID:     command.RequestID,
		CorrelationID: command.CorrelationID,
		OccurredAt:    command.OccurredAt,
		RequestedBy:   command.RequestedBy,
		AutoprogrammingRequest: orquestaautoprogramming.AutoprogrammingRequestV0{
			RequestRef:         command.RequestRef,
			ProjectRef:         command.ProjectRef,
			WorktreeRef:        command.WorktreeRef,
			WorktreeIsolated:   true,
			BranchRef:          command.BranchRef,
			Tasks:              command.Tasks,
			WriteSet:           command.WriteSet,
			RequiredTests:      command.RequiredTests,
			MaxTaskRefs:        command.MaxTaskRefs,
			MaxAreas:           command.MaxAreas,
			MaxWriteSetEntries: command.MaxWriteSetEntries,
		},
		MaxBursts:            command.MaxBursts,
		MaxStepsPerBurst:     command.MaxStepsPerBurst,
		MaxDispatchesPerWait: command.MaxDispatchesPerWait,
		MaxCommands:          command.MaxCommands,
		MaxOutboxPerCycle:    command.MaxOutboxPerCycle,
		PriorityScore:        command.PriorityScore,
	}
}

func decodeAutoprogrammingPrepareRunResponseV0(resp *http.Response, command WebAutoprogrammingPrepareRunCommandV0) (WebAutoprogrammingPrepareRunViewModelV0, error) {
	var result orquestamcp.MCPAutoprogrammingPrepareRunToolResultV0
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return WebAutoprogrammingPrepareRunViewModelV0{}, autoprogrammingPrepareRunClientErrorV0(
			WebAutoprogrammingPrepareRunErrRespuestaInvalidaV0,
			resp.StatusCode,
		)
	}
	if result.Estado == "" {
		return WebAutoprogrammingPrepareRunViewModelV0{}, autoprogrammingPrepareRunClientErrorV0(
			WebAutoprogrammingPrepareRunErrRespuestaInvalidaV0,
			resp.StatusCode,
		)
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		if result.Estado == WebAutoprogrammingPrepareRunEstadoErrorV0 {
			return NewWebAutoprogrammingPrepareRunViewModelV0(command.Locale, result), nil
		}
		return WebAutoprogrammingPrepareRunViewModelV0{}, autoprogrammingPrepareRunClientErrorV0(
			WebAutoprogrammingPrepareRunErrTransporteV0,
			resp.StatusCode,
		)
	}
	return NewWebAutoprogrammingPrepareRunViewModelV0(command.Locale, result), nil
}

func (client *RESTAutoprogrammingPrepareRunClientV0) ConsultarAutoprogrammingStatus(ctx context.Context, query WebAutoprogrammingStatusQueryV0) (WebAutoprogrammingStatusViewModelV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	query = normalizeAutoprogrammingStatusQueryV0(query)
	if client.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, client.Timeout)
		defer cancel()
	}
	resp, err := client.postJSONV0(
		ctx,
		client.endpointURLV0(WebAutoprogrammingStatusInboundEndpointV0),
		autoprogrammingStatusToolInputV0(query),
		query.CorrelationID,
		WebAutoprogrammingStatusErrRespuestaInvalidaV0,
		WebAutoprogrammingStatusErrTransporteV0,
	)
	if err != nil {
		return WebAutoprogrammingStatusViewModelV0{}, err
	}
	defer resp.Body.Close()
	return decodeAutoprogrammingStatusResponseV0(resp, query)
}

func normalizeAutoprogrammingStatusQueryV0(query WebAutoprogrammingStatusQueryV0) WebAutoprogrammingStatusQueryV0 {
	query.RequestID = trimV0(query.RequestID)
	if query.RequestID == "" {
		query.RequestID = newWebRequestIDV0()
	}
	query.CorrelationID = firstDirectorStatsNonEmptyV0(query.CorrelationID, query.RequestID)
	query.Locale = normalizeDirectorStatsLocaleV0(query.Locale)
	query.OccurredAt = trimV0(query.OccurredAt)
	query.RunRef = trimV0(query.RunRef)
	query.AppRef = trimV0(query.AppRef)
	query.ExternalJobRef = trimV0(query.ExternalJobRef)
	query.QueueRef = trimV0(query.QueueRef)
	query.AppRefs = compactStringsV0(query.AppRefs)
	if !query.IncludeProcessRefs && !query.IncludeAgentProgress && !query.IncludeAgentUsage {
		query.IncludeAgentProgress = true
	}
	return query
}

func autoprogrammingStatusToolInputV0(query WebAutoprogrammingStatusQueryV0) orquestamcp.MCPAutoprogrammingStatusToolInputV0 {
	return orquestamcp.MCPAutoprogrammingStatusToolInputV0{
		RequestID: query.RequestID, CorrelationID: query.CorrelationID,
		RunRef: query.RunRef, AppRef: query.AppRef, ExternalJobRef: query.ExternalJobRef,
		QueueRef: query.QueueRef, AppRefs: query.AppRefs, QueueLimit: query.QueueLimit,
		OccurredAt: query.OccurredAt, IncludeProcessRefs: orquestamcp.MCPFlexibleBoolV0(query.IncludeProcessRefs),
		IncludeAgentProgress: orquestamcp.MCPFlexibleBoolV0(query.IncludeAgentProgress),
		IncludeAgentUsage:    orquestamcp.MCPFlexibleBoolV0(query.IncludeAgentUsage),
	}
}

func decodeAutoprogrammingStatusResponseV0(resp *http.Response, query WebAutoprogrammingStatusQueryV0) (WebAutoprogrammingStatusViewModelV0, error) {
	var result orquestamcp.MCPAutoprogrammingStatusToolResultV0
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return WebAutoprogrammingStatusViewModelV0{}, autoprogrammingPrepareRunClientErrorV0(
			WebAutoprogrammingStatusErrRespuestaInvalidaV0,
			resp.StatusCode,
		)
	}
	if result.Estado == "" {
		return WebAutoprogrammingStatusViewModelV0{}, autoprogrammingPrepareRunClientErrorV0(
			WebAutoprogrammingStatusErrRespuestaInvalidaV0,
			resp.StatusCode,
		)
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		if result.Estado == WebAutoprogrammingPrepareRunEstadoErrorV0 {
			return NewWebAutoprogrammingStatusViewModelV0(query.Locale, result), nil
		}
		return WebAutoprogrammingStatusViewModelV0{}, autoprogrammingPrepareRunClientErrorV0(
			WebAutoprogrammingStatusErrTransporteV0,
			resp.StatusCode,
		)
	}
	return NewWebAutoprogrammingStatusViewModelV0(query.Locale, result), nil
}

func (client *RESTAutoprogrammingPrepareRunClientV0) postJSONV0(ctx context.Context, url string, payloadValue any, correlationID string, invalidCode string, transportCode string) (*http.Response, error) {
	payload, err := json.Marshal(payloadValue)
	if err != nil {
		return nil, autoprogrammingPrepareRunClientErrorV0(invalidCode, 0)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, autoprogrammingPrepareRunClientErrorV0(transportCode, 0)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set(AppChangeCorrelationV0, correlationID)
	resp, err := client.httpClient().Do(req)
	if err != nil {
		return nil, autoprogrammingPrepareRunClientErrorV0(transportCode, 0)
	}
	return resp, nil
}

func (client *RESTAutoprogrammingPrepareRunClientV0) httpClient() *http.Client {
	if client.HTTPClient != nil {
		return client.HTTPClient
	}
	return &http.Client{Timeout: client.Timeout}
}

func (client *RESTAutoprogrammingPrepareRunClientV0) url() string {
	endpoint := client.Endpoint
	if endpoint == "" {
		endpoint = WebAutoprogrammingPrepareRunInboundEndpointV0
	}
	return client.endpointURLV0(endpoint)
}

func (client *RESTAutoprogrammingPrepareRunClientV0) endpointURLV0(endpoint string) string {
	return strings.TrimRight(client.BaseURL, "/") + "/" + strings.TrimLeft(endpoint, "/")
}

func autoprogrammingPrepareRunClientErrorV0(
	code string,
	statusCode int,
) WebAutoprogrammingPrepareRunClientErrorV0 {
	return WebAutoprogrammingPrepareRunClientErrorV0{Code: code, StatusCode: statusCode}
}

func IsWebAutoprogrammingPrepareRunClientErrorCodeV0(err error, code string) bool {
	var clientErr WebAutoprogrammingPrepareRunClientErrorV0
	if errors.As(err, &clientErr) {
		return clientErr.Code == code
	}
	return false
}
