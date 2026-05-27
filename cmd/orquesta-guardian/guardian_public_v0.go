package main

import "strings"

const guardianPublicRedactionLevelV0 = "refs_only"

type guardianPublicResultV0 struct {
	SchemaVersion     string                          `json:"schema_version"`
	Status            string                          `json:"status"`
	Phase             string                          `json:"phase,omitempty"`
	Promoted          bool                            `json:"promoted,omitempty"`
	Restored          bool                            `json:"restored,omitempty"`
	RepairStarted     bool                            `json:"repair_started,omitempty"`
	ManifestRef       string                          `json:"manifest_ref,omitempty"`
	RepairPacketRef   string                          `json:"repair_packet_ref,omitempty"`
	RepairLaunchRef   string                          `json:"repair_launch_packet_ref,omitempty"`
	RepairAttemptRef  string                          `json:"repair_attempt_ref,omitempty"`
	FailurePacketHash string                          `json:"failure_packet_hash,omitempty"`
	RepairBlocked     bool                            `json:"repair_blocked,omitempty"`
	RepairBlockReason string                          `json:"repair_block_reason,omitempty"`
	RedactionLevel    string                          `json:"redaction_level"`
	Freshness         string                          `json:"freshness"`
	ReasonCodes       []string                        `json:"reason_codes,omitempty"`
	ConfigEffective   *guardianPublicConfigV0         `json:"config_effective,omitempty"`
	PathPolicy        guardianPathRootPolicyV0        `json:"path_policy,omitempty"`
	Counters          guardianPublicCountersV0        `json:"counters"`
	Commands          []guardianPublicCommandResultV0 `json:"commands,omitempty"`
	EvidenceRefs      []string                        `json:"evidence_refs,omitempty"`
	Lease             *guardianLeaseReceiptV0         `json:"lease,omitempty"`
	RetentionStatus   guardianRetentionStatusV0       `json:"retention_status,omitempty"`
	Message           string                          `json:"message,omitempty"`
	Shutdown          *guardianShutdownResultV0       `json:"shutdown,omitempty"`
	LocalDiagnostics  []guardianLocalDiagnosticRefV0  `json:"local_diagnostics,omitempty"`
}

type guardianPublicCountersV0 struct {
	CommandCount int `json:"command_count"`
	FailedCount  int `json:"failed_count"`
	EvidenceRefs int `json:"evidence_refs"`
}

type guardianPublicConfigV0 struct {
	Promote                   bool     `json:"promote"`
	SkipHealth                bool     `json:"skip_health"`
	RepairCodex               bool     `json:"repair_codex"`
	RepairCodexSandbox        string   `json:"repair_codex_sandbox,omitempty"`
	RepairCodexEffort         string   `json:"repair_codex_effort,omitempty"`
	RepairMaxAttempts         int      `json:"repair_max_attempts"`
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
	CommandEffectPolicyRef    string   `json:"command_effect_policy_ref"`
	CommandEffectEvidenceRefs []string `json:"command_effect_evidence_refs,omitempty"`
	SkipHealthEvidenceRefs    []string `json:"skip_health_evidence_refs,omitempty"`
}

type guardianPublicCommandResultV0 struct {
	Phase                   string                            `json:"phase"`
	CommandRef              string                            `json:"command_ref"`
	CommandProfile          string                            `json:"command_profile"`
	TemplateRef             string                            `json:"template_ref"`
	TemplateStatus          string                            `json:"template_status,omitempty"`
	TemplateEvidenceRefs    []string                          `json:"template_evidence_refs,omitempty"`
	ExitCode                int                               `json:"exit_code"`
	DurationMS              int64                             `json:"duration_ms"`
	OutputRef               string                            `json:"output_ref,omitempty"`
	OutputBytes             int64                             `json:"output_bytes,omitempty"`
	OutputTruncated         bool                              `json:"output_truncated,omitempty"`
	OutputReasonCode        string                            `json:"output_reason_code,omitempty"`
	Error                   string                            `json:"error,omitempty"`
	ReasonCode              string                            `json:"reason_code,omitempty"`
	ProcessPolicy           *guardianCandidateProcessPolicyV0 `json:"process_policy,omitempty"`
	StopReceipt             *guardianCandidateStopReceiptV0   `json:"stop_receipt,omitempty"`
	EffectProfile           string                            `json:"effect_profile,omitempty"`
	EffectPolicyRef         string                            `json:"effect_policy_ref,omitempty"`
	EffectAuthorization     string                            `json:"effect_authorization,omitempty"`
	ExternalEffects         []string                          `json:"external_effects,omitempty"`
	EffectEvidenceRefs      []string                          `json:"effect_evidence_refs,omitempty"`
	CommandExpansionChecked bool                              `json:"command_expansion_checked,omitempty"`
}

type guardianLocalDiagnosticRefV0 struct {
	Ref             string `json:"ref"`
	Kind            string `json:"kind"`
	Classification  string `json:"classification"`
	RedactionLevel  string `json:"redaction_level"`
	AccessPolicyRef string `json:"access_policy_ref"`
}

func publicGuardianResultV0(config guardianConfigV0, result guardianResultV0) guardianPublicResultV0 {
	return guardianPublicResultV0{
		SchemaVersion:     guardianResultSchemaVersionV0,
		Status:            result.Status,
		Phase:             result.Phase,
		Promoted:          result.Promoted,
		Restored:          result.Restored,
		RepairStarted:     result.RepairStarted,
		ManifestRef:       guardianManifestRefV0(config, result),
		RepairPacketRef:   guardianRepairPacketRefV0(config, result),
		RepairLaunchRef:   guardianRepairLaunchRefV0(config, result),
		RepairAttemptRef:  strings.TrimSpace(result.RepairAttemptRef),
		FailurePacketHash: strings.TrimSpace(result.FailurePacketHash),
		RepairBlocked:     result.RepairBlocked,
		RepairBlockReason: strings.TrimSpace(result.RepairBlockReason),
		RedactionLevel:    guardianPublicRedactionLevelV0,
		Freshness:         config.OccurredAt.Format("2006-01-02T15:04:05Z"),
		ReasonCodes:       guardianReasonCodesV0(result),
		ConfigEffective:   publicGuardianConfigV0(config),
		PathPolicy:        config.PathPolicy,
		Counters:          guardianPublicCountersFromCommandsV0(result.Commands, result.EvidenceRefs),
		Commands:          publicGuardianCommandsV0(config, result.Commands),
		EvidenceRefs:      append([]string(nil), result.EvidenceRefs...),
		Lease:             result.Lease,
		RetentionStatus:   result.RetentionStatus,
		Message:           guardianRedactOperationalTextV0(config, result.Message),
		Shutdown:          result.Shutdown,
		LocalDiagnostics:  guardianLocalDiagnosticsV0(config, result),
	}
}

func publicGuardianConfigV0(config guardianConfigV0) *guardianPublicConfigV0 {
	return &guardianPublicConfigV0{
		Promote:                   config.Promote,
		SkipHealth:                config.SkipHealth,
		RepairCodex:               config.RepairCodex,
		RepairCodexSandbox:        strings.TrimSpace(config.RepairCodexSandbox),
		RepairCodexEffort:         strings.TrimSpace(config.RepairCodexEffort),
		RepairMaxAttempts:         config.RepairMaxAttempts,
		RepairRetryEvidenceRefs:   append([]string(nil), config.RepairRetryEvidenceRefs...),
		ForceAfterTimeout:         config.ForceAfterTimeout,
		ShutdownForced:            config.ShutdownForced,
		HealthTimeoutMS:           config.HealthTimeout.Milliseconds(),
		CommandTimeoutMS:          config.CommandTimeout.Milliseconds(),
		ShutdownTimeoutMS:         config.ShutdownTimeout.Milliseconds(),
		CommandOutputMaxBytes:     config.CommandOutputMaxBytes,
		ArtifactMaxBytes:          config.ArtifactMaxBytes,
		ShutdownQueueLimit:        config.ShutdownQueueLimit,
		EnvAllowlist:              append([]string(nil), config.EnvAllowlist...),
		CommandEffectPolicyRef:    guardianCommandEffectPolicyRefV0,
		CommandEffectEvidenceRefs: append([]string(nil), config.CommandEffectEvidenceRefs...),
		SkipHealthEvidenceRefs:    append([]string(nil), config.SkipHealthEvidenceRefs...),
	}
}

func repairGuardianPacketV0(config guardianConfigV0, result guardianResultV0) guardianRepairPacketV0 {
	commands := publicGuardianCommandsV0(config, result.Commands)
	return guardianRepairPacketV0{
		SchemaVersion:       guardianRepairPacketSchemaVersionV0,
		FailurePhase:        result.Phase,
		Summary:             guardianRedactOperationalTextV0(config, result.Message),
		RedactionLevel:      guardianPublicRedactionLevelV0,
		Freshness:           config.OccurredAt.Format("2006-01-02T15:04:05Z"),
		ReasonCodes:         guardianReasonCodesV0(result),
		RepairAttemptRef:    strings.TrimSpace(result.RepairAttemptRef),
		FailurePacketHash:   strings.TrimSpace(result.FailurePacketHash),
		RepairBudget:        guardianRepairBudgetV0{MaxAgents: 1, MaxAttempts: config.RepairMaxAttempts},
		Counters:            guardianPublicCountersFromCommandsV0(result.Commands, result.EvidenceRefs),
		PathPolicy:          config.PathPolicy,
		RequiredCommandRefs: guardianRequiredCommandRefsV0(commands),
		Commands:            commands,
		EvidenceRefs:        append([]string(nil), result.EvidenceRefs...),
		LocalDiagnostics:    guardianLocalDiagnosticsV0(config, result),
		RetentionStatus:     result.RetentionStatus,
		CreatedAt:           config.OccurredAt.Format("2006-01-02T15:04:05Z"),
	}
}

func publicGuardianCommandsV0(
	config guardianConfigV0,
	commands []guardianCommandResultV0,
) []guardianPublicCommandResultV0 {
	out := make([]guardianPublicCommandResultV0, 0, len(commands))
	for _, command := range commands {
		reasonCode := strings.TrimSpace(command.ReasonCode)
		if reasonCode == "" {
			reasonCode = guardianCommandReasonCodeV0(command)
		}
		identity := guardianCommandIdentityV0(config, command)
		public := guardianPublicCommandResultV0{
			Phase:                   command.Phase,
			CommandRef:              identity.CommandRef,
			CommandProfile:          identity.CommandProfile,
			TemplateRef:             identity.TemplateRef,
			TemplateStatus:          identity.TemplateStatus,
			TemplateEvidenceRefs:    append([]string(nil), identity.TemplateEvidenceRef...),
			ExitCode:                command.ExitCode,
			DurationMS:              command.DurationMS,
			OutputRef:               guardianCommandOutputRefV0(config, command),
			OutputBytes:             command.OutputBytes,
			OutputTruncated:         command.OutputTruncated,
			OutputReasonCode:        command.OutputReasonCode,
			Error:                   guardianPublicErrorV0(command.Error),
			ReasonCode:              reasonCode,
			ProcessPolicy:           command.ProcessPolicy,
			StopReceipt:             command.StopReceipt,
			EffectProfile:           command.EffectProfile,
			EffectPolicyRef:         command.EffectPolicyRef,
			EffectAuthorization:     command.EffectAuthorization,
			ExternalEffects:         append([]string(nil), command.ExternalEffects...),
			EffectEvidenceRefs:      append([]string(nil), command.EffectEvidenceRefs...),
			CommandExpansionChecked: command.CommandExpansionChecked,
		}
		out = append(out, public)
	}
	return out
}

func guardianPublicCountersFromCommandsV0(commands []guardianCommandResultV0, refs []string) guardianPublicCountersV0 {
	counters := guardianPublicCountersV0{CommandCount: len(commands), EvidenceRefs: len(refs)}
	for _, command := range commands {
		if command.ExitCode != 0 {
			counters.FailedCount++
		}
	}
	return counters
}

func guardianRequiredCommandRefsV0(commands []guardianPublicCommandResultV0) []string {
	refs := make([]string, 0, len(commands))
	for _, command := range commands {
		refs = append(refs, command.CommandRef)
	}
	return refs
}
