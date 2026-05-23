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
	PrepararAutoprogrammingRun(
		context.Context,
		WebAutoprogrammingPrepareRunCommandV0,
	) (WebAutoprogrammingPrepareRunViewModelV0, error)
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

func NewRESTAutoprogrammingPrepareRunClientV0(
	baseURL string,
	timeout time.Duration,
) *RESTAutoprogrammingPrepareRunClientV0 {
	return &RESTAutoprogrammingPrepareRunClientV0{
		BaseURL:  strings.TrimRight(baseURL, "/"),
		Endpoint: WebAutoprogrammingPrepareRunInboundEndpointV0,
		Timeout:  timeout,
		HTTPClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (client *RESTAutoprogrammingPrepareRunClientV0) PrepararAutoprogrammingRun(
	ctx context.Context,
	command WebAutoprogrammingPrepareRunCommandV0,
) (WebAutoprogrammingPrepareRunViewModelV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	command = normalizeAutoprogrammingPrepareRunCommandV0(command)
	if client.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, client.Timeout)
		defer cancel()
	}
	payload, err := json.Marshal(autoprogrammingPrepareRunToolInputV0(command))
	if err != nil {
		return WebAutoprogrammingPrepareRunViewModelV0{}, autoprogrammingPrepareRunClientErrorV0(
			WebAutoprogrammingPrepareRunErrRespuestaInvalidaV0,
			0,
		)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, client.url(), bytes.NewReader(payload))
	if err != nil {
		return WebAutoprogrammingPrepareRunViewModelV0{}, autoprogrammingPrepareRunClientErrorV0(
			WebAutoprogrammingPrepareRunErrTransporteV0,
			0,
		)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set(AppChangeCorrelationV0, command.CorrelationID)

	resp, err := client.httpClient().Do(req)
	if err != nil {
		return WebAutoprogrammingPrepareRunViewModelV0{}, autoprogrammingPrepareRunClientErrorV0(
			WebAutoprogrammingPrepareRunErrTransporteV0,
			0,
		)
	}
	defer resp.Body.Close()
	return decodeAutoprogrammingPrepareRunResponseV0(resp, command)
}

func normalizeAutoprogrammingPrepareRunCommandV0(
	command WebAutoprogrammingPrepareRunCommandV0,
) WebAutoprogrammingPrepareRunCommandV0 {
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

func autoprogrammingPrepareRunToolInputV0(
	command WebAutoprogrammingPrepareRunCommandV0,
) orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0 {
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
	}
}

func decodeAutoprogrammingPrepareRunResponseV0(
	resp *http.Response,
	command WebAutoprogrammingPrepareRunCommandV0,
) (WebAutoprogrammingPrepareRunViewModelV0, error) {
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

func compactAutoprogrammingTasksV0(
	values []orquestaautoprogramming.AutoprogrammingTaskGroupCandidateV0,
) []orquestaautoprogramming.AutoprogrammingTaskGroupCandidateV0 {
	out := make([]orquestaautoprogramming.AutoprogrammingTaskGroupCandidateV0, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		taskRef := trimV0(value.TaskRef)
		area := trimV0(value.Area)
		if taskRef == "" || area == "" {
			continue
		}
		key := taskRef + "\x00" + area
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, orquestaautoprogramming.AutoprogrammingTaskGroupCandidateV0{
			TaskRef:            taskRef,
			Area:               area,
			Title:              trimV0(value.Title),
			Objective:          trimV0(value.Objective),
			Context:            compactStringsV0(value.Context),
			ContextRefs:        compactStringsV0(value.ContextRefs),
			AcceptanceCriteria: compactStringsV0(value.AcceptanceCriteria),
			RequiredTests:      compactStringsV0(value.RequiredTests),
			CompactRules:       compactStringsV0(value.CompactRules),
		})
	}
	return out
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
