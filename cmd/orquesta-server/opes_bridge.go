package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	orquestaopesbridge "orquesta/modulos/orquesta-opes-bridge"
	orquestaopesconnector "orquesta/modulos/orquesta-opes-connector"
)

type opesDrainSummaryV0 struct {
	OPESBaseURL     string                   `json:"opes_base_url"`
	OrquestaBaseURL string                   `json:"orquesta_base_url"`
	Limit           int                      `json:"limit"`
	JobType         string                   `json:"job_type,omitempty"`
	DryRun          bool                     `json:"dry_run,omitempty"`
	Seen            int                      `json:"seen"`
	Submitted       int                      `json:"submitted"`
	Skipped         int                      `json:"skipped"`
	Results         []opesDrainJobResultV0   `json:"results"`
	Errors          []opesDrainPublicErrorV0 `json:"errors,omitempty"`
}

type opesDrainJobResultV0 struct {
	JobRef    string `json:"job_ref"`
	WorkKind  string `json:"work_kind"`
	RunRef    string `json:"run_ref,omitempty"`
	ChangeRef string `json:"change_ref,omitempty"`
	Status    string `json:"status"`
}

type opesDrainPublicErrorV0 struct {
	JobRef string `json:"job_ref,omitempty"`
	Code   string `json:"code"`
}

func opesDrainOnceCommandV0(stdout io.Writer, stderr io.Writer) int {
	config, err := opesDrainConfigFromEnvV0()
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "opes-drain-once: %v\n", err)
		return 2
	}
	if !config.DryRun && strings.TrimSpace(os.Getenv("ORQUESTA_OPES_BRIDGE_CONFIRM")) != "1" {
		_, _ = fmt.Fprintln(stderr, "opes-drain-once: exporta ORQUESTA_OPES_BRIDGE_CONFIRM=1 para crear runs")
		return 2
	}
	summary, err := runOPESDrainOnceV0(context.Background(), config)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "opes-drain-once: %v\n", err)
		return 1
	}
	_ = json.NewEncoder(stdout).Encode(summary)
	if len(summary.Errors) > 0 {
		return 1
	}
	return 0
}

type opesDrainConfigV0 struct {
	OPESBaseURL     string
	OrquestaBaseURL string
	Limit           int
	JobType         string
	DryRun          bool
	HTTPTimeout     time.Duration
	RunConfig       orquestaopesbridge.JobRunConfigV0
}

func opesDrainConfigFromEnvV0() (opesDrainConfigV0, error) {
	opesBaseURL := firstNonEmptyEnvV0("ORQUESTA_OPES_BASE_URL", "OPES_BASE_URL")
	if opesBaseURL == "" {
		return opesDrainConfigV0{}, fmt.Errorf("ORQUESTA_OPES_BASE_URL requerido")
	}
	dryRun := strings.TrimSpace(os.Getenv("ORQUESTA_OPES_BRIDGE_DRY_RUN")) == "1"
	orquestaBaseURL, err := orquestaBaseURLFromEnvOrStateV0(dryRun)
	if err != nil {
		return opesDrainConfigV0{}, err
	}
	return opesDrainConfigV0{
		OPESBaseURL:     strings.TrimRight(opesBaseURL, "/"),
		OrquestaBaseURL: strings.TrimRight(orquestaBaseURL, "/"),
		Limit:           intEnvOrDefaultV0("ORQUESTA_OPES_BRIDGE_LIMIT", 3),
		JobType:         strings.TrimSpace(os.Getenv("ORQUESTA_OPES_BRIDGE_JOB_TYPE")),
		DryRun:          dryRun,
		HTTPTimeout:     time.Duration(intEnvOrDefaultV0("ORQUESTA_OPES_BRIDGE_TIMEOUT_SECONDS", 30)) * time.Second,
		RunConfig: orquestaopesbridge.JobRunConfigV0{
			PriorityScore: intEnvOrDefaultV0("ORQUESTA_OPES_BRIDGE_PRIORITY", 70),
			RequestedBy:   "orquesta-opes-bridge",
		},
	}, nil
}

func orquestaBaseURLFromEnvOrStateV0(dryRun bool) (string, error) {
	if value := strings.TrimSpace(os.Getenv("ORQUESTA_BASE_URL")); value != "" {
		return value, nil
	}
	if dryRun {
		return "dry-run", nil
	}
	config, err := serverConfigFromEnvV0()
	if err != nil {
		return "", err
	}
	state, err := loadStateV0(config)
	if err != nil {
		return "", fmt.Errorf("ORQUESTA_BASE_URL requerido si no hay estado de servidor")
	}
	if strings.TrimSpace(state.Addr) == "" {
		return "", fmt.Errorf("estado de servidor sin addr")
	}
	return "http://" + strings.TrimSpace(state.Addr), nil
}

func runOPESDrainOnceV0(
	ctx context.Context,
	config opesDrainConfigV0,
) (opesDrainSummaryV0, error) {
	httpClient := &http.Client{Timeout: config.HTTPTimeout}
	client := orquestaopesconnector.NewRESTClientV0(orquestaopesconnector.RESTClientConfigV0{
		BaseURL:    config.OPESBaseURL,
		HTTPClient: httpClient,
	})
	jobs, err := client.ListExternalJobsV0(ctx, orquestaopesconnector.ExternalJobQueryV0{
		ExecutionMode: "external",
		Status:        "pending",
		JobType:       config.JobType,
		Limit:         config.Limit,
	})
	if err != nil {
		return opesDrainSummaryV0{}, err
	}
	summary := opesDrainSummaryV0{
		OPESBaseURL:     config.OPESBaseURL,
		OrquestaBaseURL: config.OrquestaBaseURL,
		Limit:           config.Limit,
		JobType:         config.JobType,
		DryRun:          config.DryRun,
		Seen:            len(jobs),
	}
	for _, job := range jobs {
		result := opesDrainJobResultV0{JobRef: job.ID, WorkKind: job.Type}
		request, ok := orquestaopesbridge.BuildExternalWorkRunRequestV0(job, config.RunConfig)
		if !ok {
			summary.Skipped++
			result.Status = "skipped"
			summary.Results = append(summary.Results, result)
			summary.Errors = append(summary.Errors, opesDrainPublicErrorV0{JobRef: job.ID, Code: "job_no_convertible"})
			continue
		}
		result.ChangeRef = request.AppChangeRequest.ChangeRef
		if config.DryRun {
			result.Status = "dry_run"
			summary.Results = append(summary.Results, result)
			continue
		}
		runRef, err := submitOPESExternalWorkRunV0(ctx, httpClient, config.OrquestaBaseURL, request)
		if err != nil {
			summary.Skipped++
			result.Status = "submit_error"
			summary.Results = append(summary.Results, result)
			summary.Errors = append(summary.Errors, opesDrainPublicErrorV0{JobRef: job.ID, Code: err.Error()})
			continue
		}
		result.RunRef = runRef
		result.Status = "submitted"
		summary.Submitted++
		summary.Results = append(summary.Results, result)
	}
	return summary, nil
}

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
	httpRequest, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		baseURL+"/api/v0/external-work/run",
		bytes.NewReader(body),
	)
	if err != nil {
		return "", fmt.Errorf("request_build_error")
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	response, err := client.Do(httpRequest)
	if err != nil {
		return "", fmt.Errorf("request_http_error")
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		_, _ = io.Copy(io.Discard, response.Body)
		return "", fmt.Errorf("request_http_%d", response.StatusCode)
	}
	var decoded struct {
		RunRef string `json:"run_ref"`
		Estado string `json:"estado"`
	}
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		return "", fmt.Errorf("response_decode_error")
	}
	if strings.TrimSpace(decoded.RunRef) == "" || decoded.Estado == "error" {
		return "", fmt.Errorf("response_invalid")
	}
	return decoded.RunRef, nil
}

func firstNonEmptyEnvV0(keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value
		}
	}
	return ""
}
