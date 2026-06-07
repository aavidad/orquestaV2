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

type opesExternalWorkRunRecoveryV0 struct {
	Status      string
	EvidenceRef string
}

func recoverStoppedOPESExternalWorkRunV0(
	ctx context.Context,
	client *http.Client,
	baseURL string,
	runRef string,
	jobRef string,
	correlationID string,
) (opesExternalWorkRunRecoveryV0, error) {
	runRef = strings.TrimSpace(runRef)
	jobRef = strings.TrimSpace(jobRef)
	if runRef == "" {
		return opesExternalWorkRunRecoveryV0{}, fmt.Errorf("run_ref_required")
	}
	recoverRef := opesBridgeCompactRunPartV0(firstNonEmptyEnvlessV0(jobRef, runRef))
	resume, err := postOPESBridgeRecoveryEffectV0(ctx, client, baseURL, "/api/v0/runs/control", map[string]any{
		"request_id":      "req-opes-bridge-resume-" + recoverRef,
		"correlation_id":  strings.TrimSpace(correlationID),
		"action":          "resume",
		"run_ref":         runRef,
		"requested_by":    "orquesta-opes-bridge",
		"reason":          "recover_pending_opes_job_stopped_run",
		"idempotency_key": "idem-opes-bridge-resume-" + recoverRef,
		"evidence_refs": []string{
			"evidence-ref-opes-pending-job-recovery",
			jobRef,
		},
	})
	if err != nil {
		return opesExternalWorkRunRecoveryV0{}, err
	}
	ready, err := postOPESBridgeRecoveryEffectV0(ctx, client, baseURL, "/api/v0/runs/queue/priority", map[string]any{
		"request_id":      "req-opes-bridge-ready-" + recoverRef,
		"correlation_id":  strings.TrimSpace(correlationID),
		"action":          "set_priority",
		"queue_ref":       "global",
		"run_ref":         runRef,
		"app_ref":         "opes",
		"status":          "ready",
		"priority_score":  90,
		"requested_by":    "orquesta-opes-bridge",
		"reason":          "recover_pending_opes_job_stopped_run",
		"idempotency_key": "idem-opes-bridge-ready-" + recoverRef,
		"evidence_refs": []string{
			"evidence-ref-opes-pending-job-recovery",
			jobRef,
		},
	})
	if err != nil {
		return opesExternalWorkRunRecoveryV0{}, err
	}
	return opesExternalWorkRunRecoveryV0{
		Status:      firstNonEmptyEnvlessV0(ready.Status, resume.Status, "ready"),
		EvidenceRef: firstNonEmptyEnvlessV0(ready.EvidenceRef, resume.EvidenceRef),
	}, nil
}

func postOPESBridgeRecoveryEffectV0(
	ctx context.Context,
	client *http.Client,
	baseURL string,
	path string,
	payload map[string]any,
) (opesExternalWorkRunRecoveryV0, error) {
	if client == nil {
		return opesExternalWorkRunRecoveryV0{}, fmt.Errorf("client_required")
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return opesExternalWorkRunRecoveryV0{}, fmt.Errorf("request_marshal_error")
	}
	target, err := commandRESTEndpointURLV0(baseURL, path)
	if err != nil {
		return opesExternalWorkRunRecoveryV0{}, fmt.Errorf("request_build_error")
	}
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, target, bytes.NewReader(body))
	if err != nil {
		return opesExternalWorkRunRecoveryV0{}, fmt.Errorf("request_build_error")
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	response, err := client.Do(httpRequest)
	if err != nil {
		return opesExternalWorkRunRecoveryV0{}, errors.New(commandEffectHTTPErrorCodeV0(ctx, err))
	}
	defer response.Body.Close()
	responseBody, err := readCommandHTTPResponseBodyV0(response, "request")
	if err != nil {
		return opesExternalWorkRunRecoveryV0{}, err
	}
	var decoded struct {
		Estado       string   `json:"estado"`
		Status       string   `json:"status"`
		EvidenceRefs []string `json:"evidence_refs"`
		Updated      struct {
			Status       string   `json:"status"`
			EvidenceRefs []string `json:"evidence_refs"`
		} `json:"updated"`
	}
	if err := json.Unmarshal(responseBody, &decoded); err != nil {
		return opesExternalWorkRunRecoveryV0{}, fmt.Errorf("response_decode_error")
	}
	if decoded.Estado == "error" {
		return opesExternalWorkRunRecoveryV0{}, fmt.Errorf("response_invalid")
	}
	return opesExternalWorkRunRecoveryV0{
		Status: firstNonEmptyEnvlessV0(
			decoded.Updated.Status,
			decoded.Status,
			decoded.Estado,
		),
		EvidenceRef: firstNonEmptyEnvlessV0(
			append(decoded.Updated.EvidenceRefs, decoded.EvidenceRefs...)...,
		),
	}, nil
}
