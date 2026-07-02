package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	orquestaserver "orquesta/modulos/orquesta-server"
)

func submitOPESExternalWorkRunV0(
	ctx context.Context,
	client *http.Client,
	baseURL string,
	request any,
) (string, error) {
	result, err := submitOPESExternalWorkRunResultV0(ctx, client, baseURL, request)
	if err != nil {
		return "", err
	}
	return result.RunRef, nil
}

type opesExternalWorkRunSubmitResultV0 struct {
	RunRef                string
	RoutePolicy           string
	DirectorExecutionMode string
	GoalRef               string
	ExternalGoalRef       string
	NextActions           []string
}

func submitOPESExternalWorkRunResultV0(
	ctx context.Context,
	client *http.Client,
	baseURL string,
	request any,
) (opesExternalWorkRunSubmitResultV0, error) {
	body, err := json.Marshal(map[string]any{"external_work_run_request": request})
	if err != nil {
		return opesExternalWorkRunSubmitResultV0{}, fmt.Errorf("request_marshal_error")
	}
	target, err := commandRESTEndpointURLV0(baseURL, "/api/v0/external-work/run")
	if err != nil {
		return opesExternalWorkRunSubmitResultV0{}, fmt.Errorf("request_build_error")
	}
	httpRequest, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		target,
		bytes.NewReader(body),
	)
	if err != nil {
		return opesExternalWorkRunSubmitResultV0{}, fmt.Errorf("request_build_error")
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	response, err := client.Do(httpRequest)
	if err != nil {
		return opesExternalWorkRunSubmitResultV0{}, errors.New(opesBridgeOrquestaSubmitErrorCodeV0(ctx, err))
	}
	defer response.Body.Close()
	responseBody, err := readOPESExternalWorkRunSubmitResponseBodyV0(response)
	if err != nil {
		return opesExternalWorkRunSubmitResultV0{}, err
	}
	var decoded struct {
		RunRef                string   `json:"run_ref"`
		Estado                string   `json:"estado"`
		RoutePolicy           string   `json:"route_policy"`
		DirectorExecutionMode string   `json:"director_execution_mode"`
		GoalRef               string   `json:"goal_ref"`
		ExternalGoalRef       string   `json:"external_goal_ref"`
		NextActions           []string `json:"next_actions"`
	}
	if err := json.Unmarshal(responseBody, &decoded); err != nil {
		return opesExternalWorkRunSubmitResultV0{}, fmt.Errorf("response_decode_error")
	}
	if strings.TrimSpace(decoded.RunRef) == "" || decoded.Estado == "error" {
		return opesExternalWorkRunSubmitResultV0{}, fmt.Errorf("response_invalid")
	}
	return opesExternalWorkRunSubmitResultV0{
		RunRef:                strings.TrimSpace(decoded.RunRef),
		RoutePolicy:           strings.TrimSpace(decoded.RoutePolicy),
		DirectorExecutionMode: strings.TrimSpace(decoded.DirectorExecutionMode),
		GoalRef:               strings.TrimSpace(decoded.GoalRef),
		ExternalGoalRef:       strings.TrimSpace(decoded.ExternalGoalRef),
		NextActions:           compactStringsV0(decoded.NextActions),
	}, nil
}

func opesBridgeOrquestaSubmitErrorCodeV0(ctx context.Context, err error) string {
	code := commandEffectHTTPErrorCodeV0(ctx, err)
	switch code {
	case "request_http_error":
		return "orquesta_unreachable"
	case "effect_timeout":
		return "orquesta_unreachable_timeout"
	case "effect_cancelled":
		return "orquesta_unreachable_cancelled"
	default:
		return code
	}
}

func checkOPESBridgeRuntimeCompatibilityV0(
	ctx context.Context,
	client *http.Client,
	baseURL string,
	policy opesBridgeRuntimeCompatibilityPolicyV0,
) error {
	if !policy.Required {
		return nil
	}
	if client == nil {
		return fmt.Errorf("orquesta_runtime_compatibility_client_missing")
	}
	target, err := commandRESTEndpointURLV0(baseURL, orquestaserver.ServerReadinessEndpointV0)
	if err != nil {
		return fmt.Errorf("orquesta_runtime_compatibility_request_build_error")
	}
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return fmt.Errorf("orquesta_runtime_compatibility_request_build_error")
	}
	response, err := client.Do(httpRequest)
	if err != nil {
		return fmt.Errorf("orquesta_runtime_compatibility_readiness_unavailable")
	}
	defer response.Body.Close()
	body, err := readOPESExternalWorkRunSubmitResponseBodyV0(response)
	if err != nil {
		return fmt.Errorf("orquesta_runtime_compatibility_readiness_invalid")
	}
	var decoded struct {
		Ready           bool `json:"ready"`
		RuntimeIdentity struct {
			BinarySHA256 string `json:"binary_sha256"`
			BuildRef     string `json:"build_ref"`
			CommitRef    string `json:"commit_ref"`
		} `json:"runtime_identity"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		return fmt.Errorf("orquesta_runtime_compatibility_readiness_invalid")
	}
	if !decoded.Ready {
		return fmt.Errorf("orquesta_runtime_compatibility_not_ready")
	}
	if policy.RequiredBinarySHA256 != "" &&
		!strings.EqualFold(strings.TrimSpace(decoded.RuntimeIdentity.BinarySHA256), strings.TrimSpace(policy.RequiredBinarySHA256)) {
		return fmt.Errorf("orquesta_runtime_compatibility_binary_sha_mismatch")
	}
	if policy.RequiredBuildRef != "" &&
		strings.TrimSpace(decoded.RuntimeIdentity.BuildRef) != strings.TrimSpace(policy.RequiredBuildRef) {
		return fmt.Errorf("orquesta_runtime_compatibility_build_ref_mismatch")
	}
	if policy.RequiredCommitRef != "" &&
		strings.TrimSpace(decoded.RuntimeIdentity.CommitRef) != strings.TrimSpace(policy.RequiredCommitRef) {
		return fmt.Errorf("orquesta_runtime_compatibility_commit_ref_mismatch")
	}
	if policy.RequiredBinarySHA256 == "" && policy.RequiredBuildRef == "" && policy.RequiredCommitRef == "" &&
		strings.TrimSpace(decoded.RuntimeIdentity.BinarySHA256) == "" &&
		strings.TrimSpace(decoded.RuntimeIdentity.BuildRef) == "" &&
		strings.TrimSpace(decoded.RuntimeIdentity.CommitRef) == "" {
		return fmt.Errorf("orquesta_runtime_compatibility_identity_missing")
	}
	return nil
}

func readOPESExternalWorkRunSubmitResponseBodyV0(response *http.Response) ([]byte, error) {
	if response == nil || response.Body == nil {
		return nil, fmt.Errorf("request_response_missing")
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, commandHTTPResponseMaxBytesV0+1))
	if err != nil {
		return nil, fmt.Errorf("request_response_read_error")
	}
	if int64(len(body)) > commandHTTPResponseMaxBytesV0 {
		return nil, fmt.Errorf("request_response_too_large")
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, opesExternalWorkRunSubmitHTTPErrorV0(response.StatusCode, body)
	}
	if !commandHTTPContentTypeIsJSONV0(response.Header.Get("Content-Type")) {
		return nil, fmt.Errorf("request_response_content_type")
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	var payload any
	if err := decoder.Decode(&payload); err != nil {
		return nil, fmt.Errorf("request_response_invalid_json")
	}
	if decoder.Decode(&payload) != io.EOF {
		return nil, fmt.Errorf("request_response_trailing_data")
	}
	return body, nil
}

func opesExternalWorkRunSubmitHTTPErrorV0(status int, body []byte) error {
	return commandHTTPStatusErrorV0("request", status, "application/json", body)
}

type opesExternalWorkRunSupervisionV0 struct {
	Status                string
	StopReason            string
	ProcessRef            string
	EvidenceRef           string
	ArtifactRefs          []string
	DomainReceiptRefs     []string
	EvidenceRefs          []string
	RoutePolicy           string
	DirectorExecutionMode string
	GoalRef               string
	ExternalGoalRef       string
	NextActions           []string
	CurrentPhase          string
	OperationalReason     string
	AudioCounters         map[string]int
}

func superviseOPESExternalWorkRunV0(
	ctx context.Context,
	client *http.Client,
	baseURL string,
	runRef string,
	correlationID string,
) (opesExternalWorkRunSupervisionV0, error) {
	runRef = strings.TrimSpace(runRef)
	if runRef == "" {
		return opesExternalWorkRunSupervisionV0{}, fmt.Errorf("run_ref_required")
	}
	body, err := json.Marshal(map[string]any{
		"request_id":              "req-opes-bridge-supervise-" + opesBridgeCompactRunPartV0(runRef),
		"correlation_id":          strings.TrimSpace(correlationID),
		"director_execution_mode": "legacy_director_loop",
		"run_ref":                 runRef,
		"max_ticks":               4,
		"max_bursts":              16,
		"max_steps_per_burst":     12,
		"max_dispatches_per_wait": 8,
		"max_commands":            32,
		"max_outbox_per_cycle":    16,
		"max_decision_cycles":     8,
		"max_external_waits":      4,
	})
	if err != nil {
		return opesExternalWorkRunSupervisionV0{}, fmt.Errorf("request_marshal_error")
	}
	target, err := commandRESTEndpointURLV0(baseURL, "/api/v0/runs/supervise")
	if err != nil {
		return opesExternalWorkRunSupervisionV0{}, fmt.Errorf("request_build_error")
	}
	httpRequest, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		target,
		bytes.NewReader(body),
	)
	if err != nil {
		return opesExternalWorkRunSupervisionV0{}, fmt.Errorf("request_build_error")
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	response, err := client.Do(httpRequest)
	if err != nil {
		return opesExternalWorkRunSupervisionV0{}, errors.New(commandEffectHTTPErrorCodeV0(ctx, err))
	}
	defer response.Body.Close()
	responseBody, err := readCommandHTTPResponseBodyV0(response, "request")
	if err != nil {
		return opesExternalWorkRunSupervisionV0{}, err
	}
	var decoded struct {
		Estado                string   `json:"estado"`
		StopReason            string   `json:"stop_reason"`
		RoutePolicy           string   `json:"route_policy"`
		DirectorExecutionMode string   `json:"director_execution_mode"`
		GoalRef               string   `json:"goal_ref"`
		ExternalGoalRef       string   `json:"external_goal_ref"`
		NextActions           []string `json:"next_actions"`
		Last                  struct {
			Status       string   `json:"status"`
			ProcessRef   string   `json:"process_ref"`
			EvidenceRefs []string `json:"evidence_refs"`
		} `json:"last"`
	}
	if err := json.Unmarshal(responseBody, &decoded); err != nil {
		return opesExternalWorkRunSupervisionV0{}, fmt.Errorf("response_decode_error")
	}
	if decoded.Estado == "error" {
		return opesExternalWorkRunSupervisionV0{}, fmt.Errorf("response_invalid")
	}
	return opesExternalWorkRunSupervisionV0{
		Status:                firstNonEmptyEnvlessV0(decoded.Last.Status, decoded.Estado),
		StopReason:            strings.TrimSpace(decoded.StopReason),
		ProcessRef:            strings.TrimSpace(decoded.Last.ProcessRef),
		EvidenceRef:           firstNonEmptyEnvlessV0(decoded.Last.EvidenceRefs...),
		EvidenceRefs:          compactStringsV0(decoded.Last.EvidenceRefs),
		RoutePolicy:           strings.TrimSpace(decoded.RoutePolicy),
		DirectorExecutionMode: strings.TrimSpace(decoded.DirectorExecutionMode),
		GoalRef:               strings.TrimSpace(decoded.GoalRef),
		ExternalGoalRef:       strings.TrimSpace(decoded.ExternalGoalRef),
		NextActions:           compactStringsV0(decoded.NextActions),
	}, nil
}

func firstNonEmptyEnvlessV0(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func commandEffectHTTPErrorCodeV0(ctx context.Context, err error) string {
	switch {
	case commandHTTPRedirectDeniedErrorV0(err):
		return commandHTTPRedirectDeniedV0
	case errors.Is(err, context.DeadlineExceeded), errors.Is(ctx.Err(), context.DeadlineExceeded):
		return "effect_timeout"
	case errors.Is(err, context.Canceled), errors.Is(ctx.Err(), context.Canceled):
		return "effect_cancelled"
	default:
		return "request_http_error"
	}
}
