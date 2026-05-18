package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	orquestaopesbridge "orquesta/modulos/orquesta-opes-bridge"
	orquestaopesconnector "orquesta/modulos/orquesta-opes-connector"
)

type opesDrainSummaryV0 struct {
	OPESBaseURL      string                   `json:"opes_base_url"`
	OrquestaBaseURL  string                   `json:"orquesta_base_url"`
	Limit            int                      `json:"limit"`
	JobType          string                   `json:"job_type,omitempty"`
	JobTypeSequence  []string                 `json:"job_type_sequence,omitempty"`
	SelectedJobType  string                   `json:"selected_job_type,omitempty"`
	EmptyJobTypes    []string                 `json:"empty_job_types,omitempty"`
	JobRef           string                   `json:"job_ref,omitempty"`
	DryRun           bool                     `json:"dry_run,omitempty"`
	Seen             int                      `json:"seen"`
	Submitted        int                      `json:"submitted"`
	AlreadySubmitted int                      `json:"already_submitted,omitempty"`
	Skipped          int                      `json:"skipped"`
	Results          []opesDrainJobResultV0   `json:"results"`
	Errors           []opesDrainPublicErrorV0 `json:"errors,omitempty"`
}

type opesDrainJobResultV0 struct {
	JobRef        string `json:"job_ref"`
	WorkKind      string `json:"work_kind"`
	ContextBlocks int    `json:"context_blocks,omitempty"`
	RunRef        string `json:"run_ref,omitempty"`
	ChangeRef     string `json:"change_ref,omitempty"`
	Status        string `json:"status"`
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
	if !config.DryRun && !opesBridgeHasSafeFilterV0(config) {
		_, _ = fmt.Fprintln(stderr, "opes-drain-once: exporta ORQUESTA_OPES_BRIDGE_JOB_TYPE, ORQUESTA_OPES_BRIDGE_JOB_REF u ORQUESTA_OPES_BRIDGE_JOB_TYPE_SEQUENCE para crear runs")
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

func runOPESDrainOnceV0(
	ctx context.Context,
	config opesDrainConfigV0,
) (opesDrainSummaryV0, error) {
	if len(config.JobTypeSequence) > 0 {
		return runOPESDrainSequenceOnceV0(ctx, config)
	}
	return runOPESDrainSingleOnceV0(ctx, config)
}

func runOPESDrainSequenceOnceV0(
	ctx context.Context,
	config opesDrainConfigV0,
) (opesDrainSummaryV0, error) {
	sequence := append([]string(nil), config.JobTypeSequence...)
	emptyJobTypes := make([]string, 0, len(sequence))
	for _, jobType := range sequence {
		stageConfig := config
		stageConfig.JobType = jobType
		stageConfig.JobTypeSequence = nil
		stageConfig.JobRef = ""
		summary, err := runOPESDrainSingleOnceV0(ctx, stageConfig)
		summary.JobTypeSequence = sequence
		summary.SelectedJobType = jobType
		summary.EmptyJobTypes = append([]string(nil), emptyJobTypes...)
		if err != nil {
			return summary, err
		}
		if summary.Seen > 0 {
			return summary, nil
		}
		emptyJobTypes = append(emptyJobTypes, jobType)
	}
	return opesDrainSummaryV0{
		OPESBaseURL:     config.OPESBaseURL,
		OrquestaBaseURL: config.OrquestaBaseURL,
		Limit:           config.Limit,
		JobTypeSequence: sequence,
		EmptyJobTypes:   emptyJobTypes,
		DryRun:          config.DryRun,
	}, nil
}

func runOPESDrainSingleOnceV0(
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
		JobRef:        config.JobRef,
		Limit:         opesBridgeScanLimitV0(config),
	})
	if err != nil {
		return opesDrainSummaryV0{}, err
	}
	summary := opesDrainSummaryV0{
		OPESBaseURL:     config.OPESBaseURL,
		OrquestaBaseURL: config.OrquestaBaseURL,
		Limit:           config.Limit,
		JobType:         config.JobType,
		JobRef:          config.JobRef,
		DryRun:          config.DryRun,
		Seen:            len(jobs),
	}
	for _, job := range jobs {
		if !config.DryRun && summary.Submitted >= config.Limit {
			break
		}
		result := opesDrainJobResultV0{JobRef: job.ID, WorkKind: job.Type}
		ledgerEntry, alreadySubmitted, err := opesBridgeSubmittedLedgerEntryV0(
			ctx,
			config.InputLedger,
			job,
		)
		if err != nil {
			summary.Skipped++
			result.Status = "ledger_error"
			summary.Results = append(summary.Results, result)
			summary.Errors = append(summary.Errors, opesDrainPublicErrorV0{JobRef: job.ID, Code: err.Error()})
			continue
		}
		if alreadySubmitted {
			summary.AlreadySubmitted++
			result.Status = "already_submitted"
			result.RunRef = ledgerEntry.RunRef
			result.ChangeRef = ledgerEntry.ChangeRef
			summary.Results = append(summary.Results, result)
			continue
		}
		jobContext, err := opesJobContextV0(ctx, client, job)
		if err != nil {
			summary.Skipped++
			result.Status = "context_error"
			summary.Results = append(summary.Results, result)
			summary.Errors = append(summary.Errors, opesDrainPublicErrorV0{JobRef: job.ID, Code: err.Error()})
			continue
		}
		result.ContextBlocks = len(jobContext.TopicBlocks)
		request, ok := orquestaopesbridge.BuildExternalWorkRunRequestWithContextV0(job, config.RunConfig, jobContext)
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
		if err := opesBridgeRecordSubmittedV0(ctx, config.InputLedger, job, runRef, result.ChangeRef); err != nil {
			result.Status = "submitted_ledger_error"
			summary.Errors = append(summary.Errors, opesDrainPublicErrorV0{JobRef: job.ID, Code: err.Error()})
		}
		summary.Submitted++
		summary.Results = append(summary.Results, result)
	}
	return summary, nil
}

func opesBridgeScanLimitV0(config opesDrainConfigV0) int {
	limit := config.Limit
	if limit < 1 {
		limit = 1
	}
	if config.DryRun || strings.TrimSpace(config.JobRef) != "" {
		return limit
	}
	scanLimit := limit * 10
	if scanLimit < limit {
		return limit
	}
	if scanLimit > 100 {
		return 100
	}
	return scanLimit
}

func opesJobContextV0(
	ctx context.Context,
	client orquestaopesconnector.RESTClientV0,
	job orquestaopesconnector.ExternalJobV0,
) (orquestaopesbridge.JobContextV0, error) {
	if strings.TrimSpace(job.Type) != "summarize_topic" {
		return orquestaopesbridge.JobContextV0{}, nil
	}
	topicID := opesJobPayloadStringV0(job.PayloadJSON, "topic_id")
	if topicID == "" {
		return orquestaopesbridge.JobContextV0{}, fmt.Errorf("topic_id_required")
	}
	blocks, err := client.ListTopicBlocksV0(ctx, topicID)
	if err != nil {
		return orquestaopesbridge.JobContextV0{}, fmt.Errorf("topic_blocks_fetch_error")
	}
	if len(blocks) == 0 {
		return orquestaopesbridge.JobContextV0{}, fmt.Errorf("topic_blocks_empty")
	}
	return orquestaopesbridge.JobContextV0{TopicBlocks: blocks}, nil
}

func opesJobPayloadStringV0(payload string, key string) string {
	var values map[string]json.RawMessage
	if err := json.Unmarshal([]byte(strings.TrimSpace(payload)), &values); err != nil {
		return ""
	}
	var value string
	if raw := values[key]; len(raw) > 0 && json.Unmarshal(raw, &value) == nil {
		return strings.TrimSpace(value)
	}
	return ""
}
