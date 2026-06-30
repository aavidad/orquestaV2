package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestaopesbridge "orquesta/modulos/orquesta-opes-bridge"
	orquestaopesconnector "orquesta/modulos/orquesta-opes-connector"
)

type opesDrainSummaryV0 struct {
	OPESBaseURL      string                       `json:"opes_base_url"`
	OrquestaBaseURL  string                       `json:"orquesta_base_url"`
	Limit            int                          `json:"limit"`
	JobType          string                       `json:"job_type,omitempty"`
	JobTypeSequence  []string                     `json:"job_type_sequence,omitempty"`
	SelectedJobType  string                       `json:"selected_job_type,omitempty"`
	TransportJobType string                       `json:"transport_job_type,omitempty"`
	EmptyJobTypes    []string                     `json:"empty_job_types,omitempty"`
	JobRef           string                       `json:"job_ref,omitempty"`
	ProgramID        string                       `json:"program_id,omitempty"`
	TopicID          string                       `json:"topic_id,omitempty"`
	CorrelationID    string                       `json:"correlation_id,omitempty"`
	DryRun           bool                         `json:"dry_run,omitempty"`
	Seen             int                          `json:"seen"`
	Submitted        int                          `json:"submitted"`
	AlreadySubmitted int                          `json:"already_submitted,omitempty"`
	Claimed          int                          `json:"claimed,omitempty"`
	Skipped          int                          `json:"skipped"`
	Results          []opesDrainJobResultV0       `json:"results"`
	Errors           []opesDrainPublicErrorV0     `json:"errors,omitempty"`
	Destination      opesDrainDestinationPolicyV0 `json:"-"`
}

type opesDrainJobResultV0 struct {
	JobRef                 string         `json:"job_ref"`
	WorkKind               string         `json:"work_kind"`
	TransportJobType       string         `json:"transport_job_type,omitempty"`
	ContextBlocks          int            `json:"context_blocks,omitempty"`
	RunRef                 string         `json:"run_ref,omitempty"`
	ChangeRef              string         `json:"change_ref,omitempty"`
	RoutePolicy            string         `json:"route_policy,omitempty"`
	DirectorExecutionMode  string         `json:"director_execution_mode,omitempty"`
	GoalRef                string         `json:"goal_ref,omitempty"`
	ExternalGoalRef        string         `json:"external_goal_ref,omitempty"`
	NextActions            []string       `json:"next_actions,omitempty"`
	CurrentPhase           string         `json:"current_phase,omitempty"`
	OperationalReason      string         `json:"operational_reason,omitempty"`
	AudioCounters          map[string]int `json:"audio_counters,omitempty"`
	Status                 string         `json:"status"`
	SupervisionStatus      string         `json:"supervision_status,omitempty"`
	SupervisionStopReason  string         `json:"supervision_stop_reason,omitempty"`
	SupervisionProcessRef  string         `json:"supervision_process_ref,omitempty"`
	SupervisionEvidenceRef string         `json:"supervision_evidence_ref,omitempty"`
	GoalArtifactRefs       []string       `json:"goal_artifact_refs,omitempty"`
	GoalDomainReceiptRefs  []string       `json:"goal_domain_receipt_refs,omitempty"`
	GoalEvidenceRefs       []string       `json:"goal_evidence_refs,omitempty"`
	RunRecoveryStatus      string         `json:"run_recovery_status,omitempty"`
	RunRecoveryEvidenceRef string         `json:"run_recovery_evidence_ref,omitempty"`
}

type opesDrainPublicErrorV0 struct {
	JobRef string `json:"job_ref,omitempty"`
	Code   string `json:"code"`
	Reason string `json:"reason,omitempty"`
}

func copyStringIntMapV0(values map[string]int) map[string]int {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]int, len(values))
	for key, value := range values {
		if strings.TrimSpace(key) == "" || value == 0 {
			continue
		}
		out[strings.TrimSpace(key)] = value
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func opesDrainOnceCommandV0(stdout io.Writer, stderr io.Writer) int {
	config, err := opesDrainConfigFromEnvV0()
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "opes-drain-once: %v\n", err)
		return 2
	}
	if !config.DryRun && strings.TrimSpace(os.Getenv(envOPESBridgeConfirmV0)) != "1" {
		_, _ = fmt.Fprintln(stderr, "opes-drain-once: exporta ORQUESTA_OPES_BRIDGE_CONFIRM=1 para crear runs")
		return 2
	}
	if !config.DryRun && !opesBridgeHasSafeFilterV0(config) {
		_, _ = fmt.Fprintln(stderr, "opes-drain-once: exporta ORQUESTA_OPES_BRIDGE_JOB_REF, ORQUESTA_OPES_BRIDGE_PROGRAM_ID, ORQUESTA_OPES_BRIDGE_TOPIC_ID, ORQUESTA_OPES_BRIDGE_CORRELATION_ID u ORQUESTA_OPES_BRIDGE_ALLOW_UNFILTERED con evidencia para crear runs")
		return 2
	}
	ctx, cancel := context.WithTimeout(context.Background(), config.HTTPTimeout)
	defer cancel()
	summary, err := runOPESDrainOnceV0(ctx, config)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "opes-drain-once: %v\n", err)
		return 1
	}
	if err := writeCommandJSONOutputV0(stdout, commandPublicOPESDrainPayloadV0(summary)); err != nil {
		return reportCommandStdioWriteFailureV0(stderr, "opes-drain-once", "stdout", "json_encode", err)
	}
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
		ProgramID:       config.ProgramID,
		TopicID:         config.TopicID,
		CorrelationID:   config.CorrelationID,
		EmptyJobTypes:   emptyJobTypes,
		DryRun:          config.DryRun,
		Destination:     config.Destination,
	}, nil
}

func runOPESDrainSingleOnceV0(
	ctx context.Context,
	config opesDrainConfigV0,
) (opesDrainSummaryV0, error) {
	orquestaHTTPClient := commandHTTPClientWithRedirectPolicyV0(config.HTTPTimeout, config.OrquestaBaseURL)
	httpClient := commandOPESTemporalHTTPClientV0(config.HTTPTimeout)
	client := orquestaopesconnector.NewRESTClientV0(orquestaopesconnector.RESTClientConfigV0{
		BaseURL:    config.OPESBaseURL,
		HTTPClient: httpClient,
	})
	jobs, transportJobType, err := opesBridgeListExternalJobsForWorkKindV0(ctx, client, config)
	if err != nil {
		return opesDrainSummaryV0{}, err
	}
	summary := opesDrainSummaryV0{
		OPESBaseURL:     config.OPESBaseURL,
		OrquestaBaseURL: config.OrquestaBaseURL,
		Limit:           config.Limit,
		JobType:         config.JobType,
		TransportJobType: opesBridgePublicTransportJobTypeV0(
			config.JobType,
			transportJobType,
		),
		JobRef:        config.JobRef,
		ProgramID:     config.ProgramID,
		TopicID:       config.TopicID,
		CorrelationID: config.CorrelationID,
		DryRun:        config.DryRun,
		Seen:          len(jobs),
		Destination:   config.Destination,
	}
	for _, job := range jobs {
		if !config.DryRun && summary.Submitted >= config.Limit {
			break
		}
		workKind := orquestaopesbridge.EffectiveWorkKindForExternalJobV0(job)
		result := opesDrainJobResultV0{
			JobRef:           job.ID,
			WorkKind:         workKind,
			TransportJobType: opesBridgePublicTransportJobTypeV0(workKind, job.Type),
		}
		if opesBridgeSkipRecordedInputV0(
			ctx,
			config.InputLedger,
			job,
			&summary,
			&result,
		) {
			opesBridgeSuperviseSubmittedRunV0(ctx, &orquestaHTTPClient, config, &summary, &result)
			if len(summary.Results) > 0 {
				summary.Results[len(summary.Results)-1] = result
			}
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
		if evaluation, blocked := opesBridgeDomainWorkflowPreconditionBlockedV0(request.AppChangeRequest); blocked {
			summary.Skipped++
			result.Status = "phase_precondition_missing"
			result.CurrentPhase = evaluation.CurrentPhase
			result.OperationalReason = evaluation.OperationalReason
			result.NextActions = append([]string(nil), evaluation.NextActions...)
			result.AudioCounters = copyStringIntMapV0(evaluation.AudioCounters)
			summary.Results = append(summary.Results, result)
			summary.Errors = append(summary.Errors, opesDrainPublicErrorV0{
				JobRef: job.ID,
				Code:   orquestaopesbridge.OPESAudioWorkflowPreconditionMissingErrorV0,
				Reason: evaluation.OperationalReason,
			})
			continue
		}
		if evaluation, blocked := opesBridgeExternalCapabilityBlockedV0(
			request.AppChangeRequest,
			config.ExternalCapabilities,
		); blocked {
			summary.Skipped++
			result.Status = "external_capability_missing"
			result.OperationalReason = evaluation.OperationalReason
			result.NextActions = opesBridgeExternalCapabilityNextActionsV0(evaluation)
			summary.Results = append(summary.Results, result)
			summary.Errors = append(summary.Errors, opesDrainPublicErrorV0{
				JobRef: job.ID,
				Code:   opesBridgeExternalCapabilityErrorCodeV0(evaluation),
				Reason: evaluation.OperationalReason,
			})
			continue
		}
		if config.DryRun {
			result.Status = "dry_run"
			summary.Results = append(summary.Results, result)
			continue
		}
		request, claimedSkip := opesBridgeClaimRunRequestV0(
			ctx,
			config.InputLedger,
			job,
			request,
			&summary,
			&result,
		)
		if claimedSkip {
			opesBridgeSuperviseSubmittedRunV0(ctx, &orquestaHTTPClient, config, &summary, &result)
			if len(summary.Results) > 0 {
				summary.Results[len(summary.Results)-1] = result
			}
			continue
		}
		submitResult, err := submitOPESExternalWorkRunResultV0(ctx, &orquestaHTTPClient, config.OrquestaBaseURL, request)
		if err != nil {
			errorCode := err.Error()
			if recordErr := opesBridgeRecordSubmitFailedV0(ctx, config.InputLedger, job, result, errorCode); recordErr != nil {
				appendOPESDrainErrorV0(&summary, job.ID, externalBridgeClaimFailedCodeV0)
			}
			summary.Skipped++
			result.Status = "submit_error"
			summary.Results = append(summary.Results, result)
			summary.Errors = append(summary.Errors, opesDrainPublicErrorV0{JobRef: job.ID, Code: errorCode})
			continue
		}
		opesBridgeApplySubmitResultV0(&result, submitResult)
		result.Status = "submitted"
		if err := opesBridgeRecordSubmittedV0(
			ctx,
			config.InputLedger,
			job,
			result.RunRef,
			result.ChangeRef,
			opesBridgeRunMetadataFromDrainResultV0(result),
		); err != nil {
			result.Status = "recovery_required"
			appendOPESDrainErrorV0(&summary, job.ID, externalBridgeRecoveryRequiredCodeV0)
		}
		opesBridgeSuperviseSubmittedRunV0(ctx, &orquestaHTTPClient, config, &summary, &result)
		summary.Submitted++
		summary.Results = append(summary.Results, result)
	}
	return summary, nil
}

func opesBridgeListExternalJobsForWorkKindV0(
	ctx context.Context,
	client orquestaopesconnector.RESTClientV0,
	config opesDrainConfigV0,
) ([]orquestaopesconnector.ExternalJobV0, string, error) {
	requestedWorkKind := strings.TrimSpace(config.JobType)
	queryJobTypes := opesBridgeQueryJobTypesForWorkKindV0(requestedWorkKind)
	if len(queryJobTypes) == 0 {
		queryJobTypes = []string{""}
	}
	for _, queryJobType := range queryJobTypes {
		jobs, err := client.ListExternalJobsV0(ctx, orquestaopesconnector.ExternalJobQueryV0{
			ExecutionMode: "external",
			Status:        "pending",
			JobType:       queryJobType,
			JobRef:        config.JobRef,
			ProgramID:     config.ProgramID,
			TopicID:       config.TopicID,
			CorrelationID: config.CorrelationID,
			Limit:         opesBridgeScanLimitV0(config),
		})
		if err != nil {
			return nil, "", err
		}
		jobs = opesBridgeFilterExternalJobsByWorkKindV0(jobs, requestedWorkKind)
		if len(jobs) > 0 {
			return jobs, queryJobType, nil
		}
	}
	return []orquestaopesconnector.ExternalJobV0{}, "", nil
}

func opesBridgeQueryJobTypesForWorkKindV0(workKind string) []string {
	workKind = strings.TrimSpace(workKind)
	if workKind == "" {
		return []string{}
	}
	transportJobType := orquestaopesbridge.OPESBridgeTransportJobTypeForWorkKindV0(workKind)
	out := make([]string, 0, 2)
	for _, candidate := range []string{workKind, transportJobType} {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" || opesBridgeStringInSliceV0(out, candidate) {
			continue
		}
		out = append(out, candidate)
	}
	return out
}

func opesBridgeStringInSliceV0(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}

func opesBridgeFilterExternalJobsByWorkKindV0(
	jobs []orquestaopesconnector.ExternalJobV0,
	requestedWorkKind string,
) []orquestaopesconnector.ExternalJobV0 {
	requestedWorkKind = strings.TrimSpace(requestedWorkKind)
	if requestedWorkKind == "" {
		return jobs
	}
	requestedWorkKind = orquestaopesbridge.EffectiveWorkKindForExternalJobV0(
		orquestaopesconnector.ExternalJobV0{Type: requestedWorkKind},
	)
	out := make([]orquestaopesconnector.ExternalJobV0, 0, len(jobs))
	for _, job := range jobs {
		if orquestaopesbridge.EffectiveWorkKindForExternalJobV0(job) != requestedWorkKind {
			continue
		}
		out = append(out, job)
	}
	if out == nil {
		return []orquestaopesconnector.ExternalJobV0{}
	}
	return out
}

func opesBridgePublicTransportJobTypeV0(workKind string, transportJobType string) string {
	workKind = strings.TrimSpace(workKind)
	transportJobType = strings.TrimSpace(transportJobType)
	if workKind == "" || transportJobType == "" || workKind == transportJobType {
		return ""
	}
	return transportJobType
}

func opesBridgeScanLimitV0(config opesDrainConfigV0) int {
	limit := config.Limit
	if limit < 1 {
		limit = 1
	}
	if strings.TrimSpace(config.JobRef) != "" {
		return limit
	}
	scanLimit := limit * 100
	if scanLimit < 100 {
		return 100
	}
	if scanLimit > 500 {
		return 500
	}
	return scanLimit
}

func opesJobContextV0(
	ctx context.Context,
	client orquestaopesconnector.RESTClientV0,
	job orquestaopesconnector.ExternalJobV0,
) (orquestaopesbridge.JobContextV0, error) {
	workKind := orquestaopesbridge.EffectiveWorkKindForExternalJobV0(job)
	if workKind == "generate_question_bank" && opesQuestionBankPayloadHasPublicSourceContextV0(job.PayloadJSON) {
		return orquestaopesbridge.JobContextV0{}, nil
	}
	if workKind != "summarize_topic" && workKind != "generate_question_bank" {
		return orquestaopesbridge.JobContextV0{}, nil
	}
	topicID := opesJobPayloadStringV0(job.PayloadJSON, "topic_id")
	if topicID == "" {
		if workKind == "generate_question_bank" {
			return orquestaopesbridge.JobContextV0{}, fmt.Errorf("question_bank_source_context_required")
		}
		return orquestaopesbridge.JobContextV0{}, fmt.Errorf("topic_id_required")
	}
	blocks, err := client.ListTopicBlocksV0(ctx, topicID)
	if err != nil {
		return orquestaopesbridge.JobContextV0{}, fmt.Errorf("topic_blocks_fetch_error")
	}
	if len(blocks) == 0 {
		if workKind == "generate_question_bank" {
			return orquestaopesbridge.JobContextV0{}, fmt.Errorf("question_bank_source_context_required")
		}
		return orquestaopesbridge.JobContextV0{}, fmt.Errorf("topic_blocks_empty")
	}
	return orquestaopesbridge.JobContextV0{TopicBlocks: blocks}, nil
}

func opesQuestionBankPayloadHasPublicSourceContextV0(payload string) bool {
	return opesJobPayloadHasAnyValueV0(payload, "topic_title", "title", "planned_title") &&
		opesJobPayloadHasAnyValueV0(payload,
			"source_lesson_markdown",
			"source_text",
			"topic_text",
			"syllabus_full",
			"topic_blocks",
		) &&
		opesJobPayloadHasAnyValueV0(payload,
			"official_epigraph_text",
			"official_topic_text",
			"official_text",
			"topic_official_text",
			"section_plan",
			"sections",
			"document_plan",
		)
}

func opesJobPayloadHasAnyValueV0(payload string, keys ...string) bool {
	var values map[string]json.RawMessage
	if err := json.Unmarshal([]byte(strings.TrimSpace(payload)), &values); err != nil {
		return false
	}
	for _, key := range keys {
		raw := values[key]
		if opesJobPayloadRawHasValueV0(raw) {
			return true
		}
	}
	return false
}

func opesJobPayloadRawHasValueV0(raw json.RawMessage) bool {
	raw = json.RawMessage(strings.TrimSpace(string(raw)))
	if len(raw) == 0 || string(raw) == "null" || string(raw) == `""` || string(raw) == "[]" || string(raw) == "{}" {
		return false
	}
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return strings.TrimSpace(text) != ""
	}
	return json.Valid(raw)
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

func opesBridgeExternalCapabilityBlockedV0(
	request orquestaappchange.AppChangeRequestV0,
	capabilities []orquestadomainwork.DomainWorkExternalCapabilityV0,
) (orquestadomainwork.DomainWorkExternalCapabilityEvaluationV0, bool) {
	domainRequest, ok := orquestaappchange.DomainWorkJobRequestFromAppChangeV0(request)
	if !ok {
		return orquestadomainwork.DomainWorkExternalCapabilityEvaluationV0{}, false
	}
	evaluation := orquestadomainwork.EvaluateDomainWorkExternalCapabilitiesV0(
		domainRequest,
		capabilities,
	)
	return evaluation, !evaluation.Ready && len(evaluation.MissingRequirements) > 0
}

func opesBridgeDomainWorkflowPreconditionBlockedV0(
	request orquestaappchange.AppChangeRequestV0,
) (orquestaopesbridge.OPESAudioWorkflowPreconditionEvaluationV0, bool) {
	domainRequest, ok := orquestaappchange.DomainWorkJobRequestFromAppChangeV0(request)
	if !ok {
		return orquestaopesbridge.OPESAudioWorkflowPreconditionEvaluationV0{}, false
	}
	evaluation := orquestaopesbridge.EvaluateOPESAudioWorkflowPreconditionsV0(domainRequest)
	return evaluation, evaluation.Applies && !evaluation.Ready
}

func opesBridgeExternalCapabilityErrorCodeV0(
	evaluation orquestadomainwork.DomainWorkExternalCapabilityEvaluationV0,
) string {
	if len(evaluation.Issues) > 0 && strings.TrimSpace(evaluation.Issues[0].Code) != "" {
		return strings.TrimSpace(evaluation.Issues[0].Code)
	}
	return orquestadomainwork.ErrDomainWorkExternalCapabilityMissingV0
}

func opesBridgeExternalCapabilityNextActionsV0(
	evaluation orquestadomainwork.DomainWorkExternalCapabilityEvaluationV0,
) []string {
	for _, requirement := range evaluation.MissingRequirements {
		if requirement.Kind == orquestadomainwork.DomainWorkExternalCapabilityKindSpeechSynthesisV0 {
			return []string{
				"declare_speech_synthesis_capability",
				"declare_speech_synthesis_progress_heartbeat",
				"declare_speech_synthesis_provider_timeout",
				"configure_or_skip_audio_asset_generation",
			}
		}
	}
	return []string{"declare_required_external_capability"}
}
