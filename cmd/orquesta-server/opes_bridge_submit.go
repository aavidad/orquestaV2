package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

func submitOPESExternalWorkRunV0(
	ctx context.Context,
	client *http.Client,
	baseURL string,
	request any,
) (string, error) {
	body, err := json.Marshal(map[string]any{"external_work_run_request": request})
	if err != nil {
		return "", fmt.Errorf("request_marshal_error")
	}
	target, err := commandRESTEndpointURLV0(baseURL, "/api/v0/external-work/run")
	if err != nil {
		return "", fmt.Errorf("request_build_error")
	}
	httpRequest, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		target,
		bytes.NewReader(body),
	)
	if err != nil {
		return "", fmt.Errorf("request_build_error")
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	response, err := client.Do(httpRequest)
	if err != nil {
		return "", errors.New(commandEffectHTTPErrorCodeV0(ctx, err))
	}
	defer response.Body.Close()
	responseBody, err := readCommandHTTPResponseBodyV0(response, "request")
	if err != nil {
		return "", err
	}
	var decoded struct {
		RunRef string `json:"run_ref"`
		Estado string `json:"estado"`
	}
	if err := json.Unmarshal(responseBody, &decoded); err != nil {
		return "", fmt.Errorf("response_decode_error")
	}
	if strings.TrimSpace(decoded.RunRef) == "" || decoded.Estado == "error" {
		return "", fmt.Errorf("response_invalid")
	}
	return decoded.RunRef, nil
}

type opesExternalWorkRunSupervisionV0 struct {
	Status      string
	StopReason  string
	ProcessRef  string
	EvidenceRef string
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
		Estado     string `json:"estado"`
		StopReason string `json:"stop_reason"`
		Last       struct {
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
		Status:      firstNonEmptyEnvlessV0(decoded.Last.Status, decoded.Estado),
		StopReason:  strings.TrimSpace(decoded.StopReason),
		ProcessRef:  strings.TrimSpace(decoded.Last.ProcessRef),
		EvidenceRef: firstNonEmptyEnvlessV0(decoded.Last.EvidenceRefs...),
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
