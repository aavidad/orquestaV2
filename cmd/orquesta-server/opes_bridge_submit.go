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
	responseBody, err := readOPESExternalWorkRunSubmitResponseBodyV0(response)
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
	code := fmt.Sprintf("request_http_%d", status)
	if detail := opesExternalWorkRunSubmitPublicErrorDetailV0(body); detail != "" {
		return fmt.Errorf("%s:%s", code, detail)
	}
	return fmt.Errorf("%s", code)
}

func opesExternalWorkRunSubmitPublicErrorDetailV0(body []byte) string {
	if len(body) == 0 || !json.Valid(body) {
		return ""
	}
	var decoded struct {
		ErroresPublicos []struct {
			Code string `json:"code"`
		} `json:"errores_publicos"`
		Issues []struct {
			Code string `json:"code"`
		} `json:"issues"`
		Code string `json:"code"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		return ""
	}
	codes := make([]string, 0, len(decoded.ErroresPublicos)+len(decoded.Issues)+1)
	for _, issue := range decoded.ErroresPublicos {
		codes = append(codes, compactOPESBridgeErrorCodeV0(issue.Code))
	}
	for _, issue := range decoded.Issues {
		codes = append(codes, compactOPESBridgeErrorCodeV0(issue.Code))
	}
	codes = append(codes, compactOPESBridgeErrorCodeV0(decoded.Code))
	return strings.Join(compactOPESBridgeRunPartsV0(codes), ",")
}

func compactOPESBridgeErrorCodeV0(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	var b strings.Builder
	lastSep := false
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z',
			r >= 'A' && r <= 'Z',
			r >= '0' && r <= '9',
			r == '_',
			r == '-',
			r == '.',
			r == ':':
			b.WriteRune(r)
			lastSep = false
		default:
			if !lastSep {
				b.WriteByte('-')
				lastSep = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
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
