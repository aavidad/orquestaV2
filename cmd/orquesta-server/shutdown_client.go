package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
	orquestaserver "orquesta/modulos/orquesta-server"
	publicidentity "orquesta/modulos/orquesta-server/publicidentity"
)

type serverShutdownClientResultV0 struct {
	Estado                  string                             `json:"estado"`
	Status                  string                             `json:"status"`
	ShutdownReady           bool                               `json:"shutdown_ready"`
	RunsRequested           int                                `json:"runs_requested"`
	RunsStopped             int                                `json:"runs_stopped"`
	AgentsInFlight          int                                `json:"agents_in_flight"`
	CheckpointsPending      int                                `json:"checkpoints_pending"`
	CheckpointAgentsPending int                                `json:"checkpoint_agents_pending"`
	AsyncWorkActive         int                                `json:"shutdown_async_work_active,omitempty"`
	ActiveWorkCount         int                                `json:"active_work_count,omitempty"`
	ActiveWorkRefs          []string                           `json:"active_work_refs,omitempty"`
	ActiveWorks             []serverShutdownClientActiveWorkV0 `json:"active_works,omitempty"`
}

type serverShutdownClientActiveWorkV0 struct {
	Kind            string `json:"kind,omitempty"`
	RunRef          string `json:"run_ref,omitempty"`
	WorkRef         string `json:"work_ref,omitempty"`
	ExternalWorkRef string `json:"external_work_ref,omitempty"`
	Status          string `json:"status,omitempty"`
}

type serverShutdownClientOptionsV0 struct {
	Forced bool
	Reason string
}

const (
	defaultShutdownClientWaitTimeoutV0 = 120 * time.Second
	defaultShutdownClientWaitPollV0    = 500 * time.Millisecond
)

func requestServerShutdownV0(addr string, options serverShutdownClientOptionsV0) error {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return fmt.Errorf("server_addr_vacio")
	}
	reason := strings.TrimSpace(options.Reason)
	if reason == "" {
		reason = "apagado cooperativo solicitado por CLI al Director"
	}
	requestID := "request-orquesta-server-stop"
	correlationID := "corr-orquesta-server-stop"
	idempotencyKey := "idem-orquesta-server-stop"
	result, err := postServerShutdownRequestV0(addr, options, requestID, correlationID, idempotencyKey, reason)
	if err != nil {
		return err
	}
	if result.Estado != "ok" {
		return fmt.Errorf("shutdown_status_%s", result.Status)
	}
	if shutdownClientResultReadyForSignalV0(result) {
		return nil
	}
	return waitServerShutdownReadyV0(
		addr,
		options,
		result,
		serverShutdownClientRequestIdentityV0{
			requestID:      requestID,
			correlationID:  correlationID,
			idempotencyKey: idempotencyKey,
			reason:         reason,
		},
		defaultShutdownClientWaitTimeoutV0,
		defaultShutdownClientWaitPollV0,
	)
}

type serverShutdownClientRequestIdentityV0 struct {
	requestID      string
	correlationID  string
	idempotencyKey string
	reason         string
}

func postServerShutdownRequestV0(
	addr string,
	options serverShutdownClientOptionsV0,
	requestID string,
	correlationID string,
	idempotencyKey string,
	reason string,
) (serverShutdownClientResultV0, error) {
	body := bytes.NewBuffer(nil)
	if err := json.NewEncoder(body).Encode(map[string]any{
		"request_id":            requestID,
		"correlation_id":        correlationID,
		"forced":                options.Forced,
		"cleanup_goal_backends": true,
		"requested_by":          "orquesta-director",
		"reason":                reason,
		"idempotency_key":       idempotencyKey,
		"evidence_refs": []string{
			orquestaruncontrol.RunControlEvidenceAutoResumeAllowedV0,
			"evidence-ref-orquesta-server-cli-stop-auto-resume-safe",
		},
	}); err != nil {
		return serverShutdownClientResultV0{}, fmt.Errorf("shutdown_request_encode")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	baseURL, err := commandRESTBaseURLFromAddrV0(addr)
	if err != nil {
		return serverShutdownClientResultV0{}, fmt.Errorf("shutdown_request_build")
	}
	target, err := commandRESTEndpointURLV0(baseURL, "/api/v0/server/shutdown")
	if err != nil {
		return serverShutdownClientResultV0{}, fmt.Errorf("shutdown_request_build")
	}
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		target,
		body,
	)
	if err != nil {
		return serverShutdownClientResultV0{}, fmt.Errorf("shutdown_request_build")
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set(publicidentity.PublicCorrelationHeaderV0, correlationID)
	request.Header.Set(publicidentity.PublicIdempotencyKeyHeaderV0, idempotencyKey)
	client := commandHTTPClientWithRedirectPolicyV0(120*time.Second, baseURL)
	response, err := client.Do(request)
	if err != nil {
		return serverShutdownClientResultV0{}, errors.New(shutdownHTTPErrorCodeV0(ctx, err))
	}
	defer response.Body.Close()
	responseBody, err := readCommandHTTPResponseBodyV0(response, "shutdown")
	if err != nil {
		return serverShutdownClientResultV0{}, err
	}
	var result serverShutdownClientResultV0
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return serverShutdownClientResultV0{}, fmt.Errorf("shutdown_response_invalid")
	}
	result = normalizeServerShutdownClientResultV0(result)
	return result, nil
}

func normalizeServerShutdownClientResultV0(result serverShutdownClientResultV0) serverShutdownClientResultV0 {
	activeWorkRefs := compactStringsV0(append(
		append([]string(nil), result.ActiveWorkRefs...),
		serverShutdownClientActiveWorkRefsV0(result.ActiveWorks)...,
	))
	result.ActiveWorkRefs = activeWorkRefs
	if result.ActiveWorkCount <= 0 {
		if len(result.ActiveWorks) > 0 {
			result.ActiveWorkCount = len(result.ActiveWorks)
		} else if len(activeWorkRefs) > 0 {
			result.ActiveWorkCount = len(activeWorkRefs)
		}
	}
	return result
}

func serverShutdownClientActiveWorkRefsV0(works []serverShutdownClientActiveWorkV0) []string {
	out := make([]string, 0, len(works)*4)
	for _, work := range works {
		prefix := "shutdown-active-work"
		if kind := safeShutdownClientRefPartV0(work.Kind); kind != "" {
			prefix += "-" + kind
		}
		for _, ref := range []string{work.RunRef, work.WorkRef, work.ExternalWorkRef, work.Status} {
			if safe := safeShutdownClientRefPartV0(ref); safe != "" {
				out = append(out, prefix+"-"+safe)
			}
		}
	}
	return compactStringsV0(out)
}

func safeShutdownClientRefPartV0(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return ""
	}
	var builder strings.Builder
	lastSep := false
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z',
			r >= '0' && r <= '9':
			builder.WriteRune(r)
			lastSep = false
		default:
			if !lastSep {
				builder.WriteByte('-')
				lastSep = true
			}
		}
	}
	return strings.Trim(builder.String(), "-")
}

func shutdownHTTPErrorCodeV0(ctx context.Context, err error) string {
	switch {
	case commandHTTPRedirectDeniedErrorV0(err):
		return commandHTTPRedirectDeniedV0
	case errors.Is(err, context.DeadlineExceeded), errors.Is(ctx.Err(), context.DeadlineExceeded):
		return "shutdown_timeout"
	case errors.Is(err, context.Canceled), errors.Is(ctx.Err(), context.Canceled):
		return "shutdown_cancelled"
	default:
		return "shutdown_request_failed"
	}
}

func waitServerShutdownReadyV0(
	addr string,
	options serverShutdownClientOptionsV0,
	initial serverShutdownClientResultV0,
	identity serverShutdownClientRequestIdentityV0,
	timeout time.Duration,
	pollInterval time.Duration,
) error {
	if shutdownClientResultReadyForSignalV0(initial) {
		return nil
	}
	if !shutdownClientResultCanWaitV0(initial) {
		return shutdownClientNotReadyErrorV0(initial)
	}
	if timeout <= 0 {
		return shutdownClientNotReadyErrorV0(initial)
	}
	if pollInterval <= 0 {
		pollInterval = defaultShutdownClientWaitPollV0
	}
	deadline := time.Now().Add(timeout)
	last := initial
	for {
		if shutdownClientResultCanWaitV0(last) {
			if refreshed, err := postServerShutdownRequestV0(
				addr,
				options,
				identity.requestID,
				identity.correlationID,
				identity.idempotencyKey,
				identity.reason,
			); err == nil {
				last = refreshed
				if shutdownClientResultReadyForSignalV0(last) {
					return nil
				}
			}
		}
		status, err := getServerPublicStatusForShutdownV0(addr)
		if err == nil {
			if shutdownPublicStatusReadyForSignalV0(status) {
				return nil
			}
			last = serverShutdownClientResultFromStatusV0(status)
		}
		if !time.Now().Before(deadline) {
			return shutdownClientNotReadyErrorV0(last)
		}
		time.Sleep(pollInterval)
	}
}

func getServerPublicStatusForShutdownV0(addr string) (orquestaserver.ServerPublicStatusV0, error) {
	body, err := getStatusBodyV0(addr)
	if err != nil {
		return orquestaserver.ServerPublicStatusV0{}, err
	}
	var status orquestaserver.ServerPublicStatusV0
	if err := json.Unmarshal(body, &status); err != nil {
		return orquestaserver.ServerPublicStatusV0{}, fmt.Errorf("shutdown_status_response_invalid")
	}
	return status, nil
}
