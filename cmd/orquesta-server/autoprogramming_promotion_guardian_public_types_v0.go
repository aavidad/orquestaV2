package main

import "encoding/json"

type autoprogrammingPromotionGuardianPublicResultV0 struct {
	SchemaVersion     string                                              `json:"schema_version"`
	Status            string                                              `json:"status"`
	Phase             string                                              `json:"phase,omitempty"`
	Promoted          bool                                                `json:"promoted,omitempty"`
	Restored          bool                                                `json:"restored,omitempty"`
	RepairStarted     bool                                                `json:"repair_started,omitempty"`
	ManifestRef       string                                              `json:"manifest_ref,omitempty"`
	RepairPacketRef   string                                              `json:"repair_packet_ref,omitempty"`
	RepairLaunchRef   string                                              `json:"repair_launch_packet_ref,omitempty"`
	RepairAttemptRef  string                                              `json:"repair_attempt_ref,omitempty"`
	FailurePacketHash string                                              `json:"failure_packet_hash,omitempty"`
	RepairBlocked     bool                                                `json:"repair_blocked,omitempty"`
	RepairBlockReason string                                              `json:"repair_block_reason,omitempty"`
	RedactionLevel    string                                              `json:"redaction_level"`
	Freshness         string                                              `json:"freshness"`
	ReasonCodes       []string                                            `json:"reason_codes,omitempty"`
	ConfigEffective   *autoprogrammingPromotionGuardianPublicConfigV0     `json:"config_effective,omitempty"`
	PathPolicy        json.RawMessage                                     `json:"path_policy,omitempty"`
	Counters          autoprogrammingPromotionGuardianPublicCountersV0    `json:"counters"`
	Commands          []autoprogrammingPromotionGuardianPublicCommandV0   `json:"commands,omitempty"`
	EvidenceRefs      []string                                            `json:"evidence_refs,omitempty"`
	Lease             json.RawMessage                                     `json:"lease,omitempty"`
	RetentionStatus   json.RawMessage                                     `json:"retention_status,omitempty"`
	Message           string                                              `json:"message,omitempty"`
	Shutdown          *autoprogrammingPromotionGuardianPublicShutdownV0   `json:"shutdown,omitempty"`
	LocalDiagnostics  []autoprogrammingPromotionGuardianLocalDiagnosticV0 `json:"local_diagnostics,omitempty"`
}

type autoprogrammingPromotionGuardianPublicConfigV0 struct {
	Promote                   bool     `json:"promote"`
	SkipHealth                bool     `json:"skip_health"`
	RepairCodex               bool     `json:"repair_codex"`
	RepairCodexSandbox        string   `json:"repair_codex_sandbox,omitempty"`
	RepairCodexEffort         string   `json:"repair_codex_effort,omitempty"`
	RepairMaxAttempts         int      `json:"repair_max_attempts,omitempty"`
	RepairRetryEvidenceRefs   []string `json:"repair_retry_evidence_refs,omitempty"`
	ForceAfterTimeout         bool     `json:"force_after_timeout"`
	ShutdownForced            bool     `json:"shutdown_forced"`
	HealthTimeoutMS           int64    `json:"health_timeout_ms"`
	CommandTimeoutMS          int64    `json:"command_timeout_ms"`
	ShutdownTimeoutMS         int64    `json:"shutdown_timeout_ms"`
	CommandOutputMaxBytes     int64    `json:"command_output_max_bytes"`
	ArtifactMaxBytes          int64    `json:"artifact_max_bytes"`
	ShutdownQueueLimit        int      `json:"shutdown_queue_limit"`
	EnvAllowlist              []string `json:"env_allowlist,omitempty"`
	CommandEffectPolicyRef    string   `json:"command_effect_policy_ref,omitempty"`
	CommandEffectEvidenceRefs []string `json:"command_effect_evidence_refs,omitempty"`
	SkipHealthEvidenceRefs    []string `json:"skip_health_evidence_refs,omitempty"`
}

type autoprogrammingPromotionGuardianPublicCountersV0 struct {
	CommandCount int `json:"command_count"`
	FailedCount  int `json:"failed_count"`
	EvidenceRefs int `json:"evidence_refs"`
}

type autoprogrammingPromotionGuardianPublicCommandV0 struct {
	Phase                   string                                           `json:"phase"`
	CommandRef              string                                           `json:"command_ref"`
	CommandProfile          string                                           `json:"command_profile,omitempty"`
	TemplateRef             string                                           `json:"template_ref,omitempty"`
	TemplateStatus          string                                           `json:"template_status,omitempty"`
	TemplateEvidenceRefs    []string                                         `json:"template_evidence_refs,omitempty"`
	ExitCode                int                                              `json:"exit_code"`
	DurationMS              int64                                            `json:"duration_ms"`
	OutputRef               string                                           `json:"output_ref,omitempty"`
	OutputBytes             int64                                            `json:"output_bytes,omitempty"`
	OutputTruncated         bool                                             `json:"output_truncated,omitempty"`
	OutputReasonCode        string                                           `json:"output_reason_code,omitempty"`
	Error                   string                                           `json:"error,omitempty"`
	ReasonCode              string                                           `json:"reason_code,omitempty"`
	ProcessPolicy           *autoprogrammingPromotionGuardianProcessPolicyV0 `json:"process_policy,omitempty"`
	StopReceipt             *autoprogrammingPromotionGuardianStopReceiptV0   `json:"stop_receipt,omitempty"`
	EffectProfile           string                                           `json:"effect_profile,omitempty"`
	EffectPolicyRef         string                                           `json:"effect_policy_ref,omitempty"`
	EffectAuthorization     string                                           `json:"effect_authorization,omitempty"`
	ExternalEffects         []string                                         `json:"external_effects,omitempty"`
	EffectEvidenceRefs      []string                                         `json:"effect_evidence_refs,omitempty"`
	CommandExpansionChecked bool                                             `json:"command_expansion_checked,omitempty"`
}

type autoprogrammingPromotionGuardianProcessPolicyV0 struct {
	Platform       string `json:"platform"`
	Scope          string `json:"scope"`
	TreeStop       bool   `json:"tree_stop"`
	PublicContract string `json:"public_contract"`
}

type autoprogrammingPromotionGuardianStopReceiptV0 struct {
	Scope             string `json:"scope"`
	Status            string `json:"status"`
	ReasonCode        string `json:"reason_code"`
	DeadlineExceeded  bool   `json:"deadline_exceeded,omitempty"`
	Escalated         bool   `json:"escalated,omitempty"`
	TreeStopConfirmed bool   `json:"tree_stop_confirmed,omitempty"`
	Ambiguous         bool   `json:"ambiguous,omitempty"`
}

type autoprogrammingPromotionGuardianPublicShutdownV0 struct {
	Estado                  string `json:"estado,omitempty"`
	Status                  string `json:"status,omitempty"`
	ShutdownReady           bool   `json:"shutdown_ready,omitempty"`
	RunsRequested           int    `json:"runs_requested,omitempty"`
	RunsStopped             int    `json:"runs_stopped,omitempty"`
	AgentsInFlight          int    `json:"agents_in_flight,omitempty"`
	CheckpointsPending      int    `json:"checkpoints_pending,omitempty"`
	CheckpointAgentsPending int    `json:"checkpoint_agents_pending,omitempty"`
	EscalationStatus        string `json:"escalation_status,omitempty"`
	EscalationEvidenceRef   string `json:"escalation_evidence_ref,omitempty"`
	SignalStatus            string `json:"signal_status,omitempty"`
	SignalBlockReason       string `json:"signal_block_reason,omitempty"`
}

type autoprogrammingPromotionGuardianLocalDiagnosticV0 struct {
	Ref             string `json:"ref"`
	Kind            string `json:"kind"`
	Classification  string `json:"classification"`
	RedactionLevel  string `json:"redaction_level"`
	AccessPolicyRef string `json:"access_policy_ref"`
}
