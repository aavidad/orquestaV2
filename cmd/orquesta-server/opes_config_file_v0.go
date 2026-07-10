package main

import "strings"

const (
	defaultOPESTimeoutSecondsV0     = 30
	defaultOPESDefaultMaxAttemptsV0 = 1
)

type serverOPESConfigSnapshotV0 struct {
	ProjectWorkDir     string
	TimeoutSeconds     int
	DefaultMaxAttempts int
}

func serverOPESConfigSnapshotFromProjectConfigFileV0(config serverProjectConfigFileV0) serverOPESConfigSnapshotV0 {
	return serverOPESConfigSnapshotV0{
		ProjectWorkDir: stringProjectConfigOrEnvOrDefaultV0(
			envOPESProjectWorkDirV0,
			config.OPES.ProjectWorkDir,
			"",
		),
		TimeoutSeconds: intProjectConfigOrEnvOrDefaultV0(
			envOPESTimeoutSecondsV0,
			config.OPES.TimeoutSeconds,
			defaultOPESTimeoutSecondsV0,
		),
		DefaultMaxAttempts: intProjectConfigOrEnvOrDefaultV0(
			envOPESDefaultMaxAttemptsV0,
			config.OPES.DefaultMaxAttempts,
			defaultOPESDefaultMaxAttemptsV0,
		),
	}
}

func serverOPESConfigSnapshotFromEnvV0() serverOPESConfigSnapshotV0 {
	return serverOPESConfigSnapshotFromProjectConfigFileV0(opesProjectConfigFromEnvBestEffortV0())
}

func (snapshot serverOPESConfigSnapshotV0) HasProjectWorkDir() bool {
	return strings.TrimSpace(snapshot.ProjectWorkDir) != ""
}

type serverProjectConfigOPESV0 struct {
	BaseURL            *string `json:"base_url,omitempty"`
	ProjectWorkDir     *string `json:"project_workdir,omitempty"`
	TemporalConfirm    *bool   `json:"temporal_confirm,omitempty"`
	TimeoutSeconds     *int    `json:"timeout_seconds,omitempty"`
	DefaultMaxAttempts *int    `json:"default_max_attempts,omitempty"`
}

type serverProjectConfigOPESBridgeV0 struct {
	Enabled                      *bool                                     `json:"enabled,omitempty"`
	Confirm                      *bool                                     `json:"confirm,omitempty"`
	DryRun                       *bool                                     `json:"dry_run,omitempty"`
	JobType                      *string                                   `json:"job_type,omitempty"`
	JobRef                       *string                                   `json:"job_ref,omitempty"`
	ProgramID                    *string                                   `json:"program_id,omitempty"`
	TopicID                      *string                                   `json:"topic_id,omitempty"`
	CorrelationID                *string                                   `json:"correlation_id,omitempty"`
	JobTypeSequence              *[]string                                 `json:"job_type_sequence,omitempty"`
	Limit                        *int                                      `json:"limit,omitempty"`
	TimeoutSeconds               *int                                      `json:"timeout_seconds,omitempty"`
	Priority                     *int                                      `json:"priority,omitempty"`
	IntervalSeconds              *int                                      `json:"interval_seconds,omitempty"`
	InitialDelaySeconds          *int                                      `json:"initial_delay_seconds,omitempty"`
	MaxTicks                     *int                                      `json:"max_ticks,omitempty"`
	SuperviseSubmitted           *bool                                     `json:"supervise_submitted,omitempty"`
	WaitResidentSeconds          *int                                      `json:"wait_resident_seconds,omitempty"`
	WaitResidentIntervalMS       *int                                      `json:"wait_resident_interval_ms,omitempty"`
	AllowUnfiltered              *bool                                     `json:"allow_unfiltered,omitempty"`
	InputLedgerDisabled          *bool                                     `json:"input_ledger_disabled,omitempty"`
	InputLedgerPath              *string                                   `json:"input_ledger_path,omitempty"`
	ProductiveConfirm            *bool                                     `json:"productive_confirm,omitempty"`
	DestinationEvidenceRef       *string                                   `json:"destination_evidence_ref,omitempty"`
	RequireRuntimeCompatibility  *bool                                     `json:"require_runtime_compatibility,omitempty"`
	RequiredRuntimeBinarySHA256  *string                                   `json:"required_runtime_binary_sha256,omitempty"`
	RequiredRuntimeBuildRef      *string                                   `json:"required_runtime_build_ref,omitempty"`
	RequiredRuntimeCommitRef     *string                                   `json:"required_runtime_commit_ref,omitempty"`
	SpeechSynthesis              serverProjectConfigOPESBridgeCapabilityV0 `json:"speech_synthesis,omitempty"`
	SpeechSynthesisNoProgress    *int                                      `json:"speech_synthesis_no_progress_timeout_seconds,omitempty"`
	SpeechSynthesisToolWorkDir   *string                                   `json:"speech_synthesis_tool_workdir,omitempty"`
	SpeechSynthesisToolCommand   *string                                   `json:"speech_synthesis_tool_command,omitempty"`
	SpeechSynthesisToolPreflight *int                                      `json:"speech_synthesis_tool_preflight_timeout_seconds,omitempty"`
	SpeechSynthesisProgressReady *string                                   `json:"speech_synthesis_progress_heartbeat_ready,omitempty"`
	SpeechSynthesisTimeoutReady  *string                                   `json:"speech_synthesis_provider_timeout_ready,omitempty"`
	RemoteQA                     serverProjectConfigOPESBridgeCapabilityV0 `json:"remote_qa,omitempty"`
	RemoteQAAuthStateReady       *string                                   `json:"remote_qa_auth_state_ready,omitempty"`
}

type serverProjectConfigOPESBridgeCapabilityV0 struct {
	Capability         *string `json:"capability,omitempty"`
	CapabilityRef      *string `json:"capability_ref,omitempty"`
	EvidenceRefs       *string `json:"evidence_refs,omitempty"`
	Reason             *string `json:"reason,omitempty"`
	NetworkReady       *string `json:"network_ready,omitempty"`
	ToolPathReady      *string `json:"tool_path_ready,omitempty"`
	ProviderQuotaReady *string `json:"provider_quota_ready,omitempty"`
}

type serverProjectConfigOPESRegistryFinalPkgV0 struct {
	Enabled          *bool   `json:"enabled,omitempty"`
	Confirm          *bool   `json:"confirm,omitempty"`
	DryRun           *bool   `json:"dry_run,omitempty"`
	RegistryPath     *string `json:"registry_path,omitempty"`
	CourseID         *string `json:"course_id,omitempty"`
	CourseRoot       *string `json:"course_root,omitempty"`
	TemplateRunRef   *string `json:"template_run_ref,omitempty"`
	TemplateTopicID  *string `json:"template_topic_id,omitempty"`
	BatchSize        *int    `json:"batch_size,omitempty"`
	MaxInFlight      *int    `json:"max_in_flight,omitempty"`
	IntervalSeconds  *int    `json:"interval_seconds,omitempty"`
	MaxTicks         *int    `json:"max_ticks,omitempty"`
	QueueRef         *string `json:"queue_ref,omitempty"`
	ReconcileEnabled *bool   `json:"reconcile_enabled,omitempty"`
	ReconcileLimit   *int    `json:"reconcile_limit,omitempty"`
}

type serverProjectConfigOPESTopicRegistryV0 struct {
	Enabled  *bool   `json:"enabled,omitempty"`
	ToolPath *string `json:"tool_path,omitempty"`
	AgentID  *string `json:"agent_id,omitempty"`
	Force    *bool   `json:"force,omitempty"`
}

func opesProjectWorkDirFromProjectConfigFileV0(config serverProjectConfigFileV0) string {
	return serverOPESConfigSnapshotFromProjectConfigFileV0(config).ProjectWorkDir
}
