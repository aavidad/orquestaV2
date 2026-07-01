package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	guardianCandidateLivenessNotReadyV0  = "candidate_liveness_not_ready"
	guardianCandidateReadinessNotReadyV0 = "candidate_readiness_not_ready"
	guardianServerReadinessSchemaV0      = "orquesta_server_readiness.v0"
)

type guardianCandidateReadinessV0 struct {
	SchemaVersion string   `json:"schema_version"`
	Ready         bool     `json:"ready"`
	Status        string   `json:"status"`
	StartupReady  bool     `json:"startup_ready"`
	StartupStatus string   `json:"startup_status"`
	EvidenceRefs  []string `json:"evidence_refs"`
}

type guardianCandidateReadinessErrorV0 struct {
	code string
	text string
}

func (err guardianCandidateReadinessErrorV0) Error() string {
	if strings.TrimSpace(err.text) == "" {
		return err.code
	}
	return err.code + ": " + err.text
}

func waitGuardianCandidateReadinessV0(
	ctx context.Context,
	baseURL string,
	timeout time.Duration,
) (guardianCandidateReadinessV0, error) {
	deadline := time.Now().Add(timeout)
	client := newGuardianCandidateReadinessHTTPClientV0()
	var last error
	for time.Now().Before(deadline) {
		if err := ctx.Err(); err != nil {
			return guardianCandidateReadinessV0{}, err
		}
		if err := requestGuardianCandidateLivenessV0(client, baseURL); err != nil {
			last = err
			time.Sleep(200 * time.Millisecond)
			continue
		}
		readiness, err := requestGuardianCandidateReadinessV0(client, baseURL)
		if err == nil && readiness.Ready {
			return readiness, nil
		}
		last = err
		time.Sleep(200 * time.Millisecond)
	}
	if last != nil {
		return guardianCandidateReadinessV0{}, last
	}
	return guardianCandidateReadinessV0{}, guardianCandidateReadinessErrorV0{code: guardianCandidateReadinessNotReadyV0, text: "timeout"}
}

var newGuardianCandidateReadinessHTTPClientV0 = func() http.Client {
	return http.Client{Timeout: 500 * time.Millisecond}
}

func requestGuardianCandidateLivenessV0(client http.Client, baseURL string) error {
	response, err := client.Get(strings.TrimRight(baseURL, "/") + "/healthz")
	if err != nil {
		return guardianCandidateReadinessErrorV0{code: guardianCandidateLivenessNotReadyV0, text: "request_failed"}
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return guardianCandidateReadinessErrorV0{code: guardianCandidateLivenessNotReadyV0, text: fmt.Sprintf("status=%d", response.StatusCode)}
	}
	return nil
}

func requestGuardianCandidateReadinessV0(
	client http.Client,
	baseURL string,
) (guardianCandidateReadinessV0, error) {
	response, err := client.Get(strings.TrimRight(baseURL, "/") + "/api/v0/server/readiness")
	if err != nil {
		return guardianCandidateReadinessV0{}, guardianCandidateReadinessErrorV0{code: guardianCandidateReadinessNotReadyV0, text: "request_failed"}
	}
	defer response.Body.Close()
	var readiness guardianCandidateReadinessV0
	decoder := json.NewDecoder(io.LimitReader(response.Body, 32*1024))
	if err := decoder.Decode(&readiness); err != nil {
		return guardianCandidateReadinessV0{}, guardianCandidateReadinessErrorV0{code: guardianCandidateReadinessNotReadyV0, text: "json_decode_failed"}
	}
	if response.StatusCode != http.StatusOK || !guardianCandidateReadinessCompatibleV0(readiness) {
		return readiness, guardianCandidateReadinessErrorV0{
			code: guardianCandidateReadinessNotReadyV0,
			text: "status=" + compactGuardianReadinessStatusV0(response.StatusCode, readiness),
		}
	}
	return readiness, nil
}

func guardianCandidateReadinessCompatibleV0(readiness guardianCandidateReadinessV0) bool {
	return readiness.SchemaVersion == guardianServerReadinessSchemaV0 &&
		readiness.Ready &&
		readiness.StartupReady &&
		strings.TrimSpace(readiness.Status) == "running"
}

func compactGuardianReadinessStatusV0(statusCode int, readiness guardianCandidateReadinessV0) string {
	parts := []string{
		fmt.Sprintf("http_%d", statusCode),
		"status_" + sanitizeFilenamePartV0(readiness.Status),
		"startup_" + sanitizeFilenamePartV0(readiness.StartupStatus),
	}
	if readiness.StartupReady {
		parts = append(parts, "startup_ready")
	}
	return strings.Join(parts, ",")
}

func guardianCandidateReadinessReasonCodeV0(err error) string {
	if typed, ok := err.(guardianCandidateReadinessErrorV0); ok {
		return typed.code
	}
	return guardianCandidateReadinessNotReadyV0
}
