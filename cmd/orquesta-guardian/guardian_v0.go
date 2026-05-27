package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
)

const (
	guardianResultSchemaVersionV0             = "orquesta_guardian_result.v0"
	guardianRepairPacketSchemaVersionV0       = "orquesta_guardian_repair_packet.v0"
	guardianRepairPromptSchemaVersionV0       = "orquesta_guardian_repair_prompt.v0"
	guardianRepairLaunchPacketSchemaVersionV0 = "orquesta_guardian_repair_launch_packet.v0"

	guardianStatusPromotedV0            = "candidate_promoted"
	guardianStatusPromotedBreakglassV0  = "candidate_promoted_breakglass"
	guardianStatusCandidateFailedV0     = "candidate_failed"
	guardianStatusManifestIncompleteV0  = "guardian_manifest_incomplete"
	guardianStatusPromotionIncompleteV0 = "promotion_incomplete"
	guardianStatusLastGoodUnverifiedV0  = "last_good_unverified"
	guardianStatusRestoredV0            = "last_good_restored"
	guardianStatusShutdownReadyV0       = "shutdown_ready"
	guardianStatusLeaseBusyV0           = "guardian_promotion_lease_busy"
	guardianStatusLeaseLostV0           = "guardian_promotion_lease_lost"
)

type guardianConfigV0 struct {
	ProjectDir                    string
	StateDir                      string
	CurrentBin                    string
	CandidateBin                  string
	LastGoodBin                   string
	ArtifactRoot                  string
	BuildCommand                  string
	TestCommands                  []string
	RepairCommand                 string
	CommandEffectEvidenceRefs     []string
	SkipHealthEvidenceRefs        []string
	RepairCodex                   bool
	RepairCodexSandbox            string
	RepairCodexEffort             string
	RepairCodexRuntimeDir         string
	RepairCodexWriteSet           []string
	RepairCodexRequiredTests      []string
	RepairCodexWorktreeRef        string
	RepairCodexBranchRef          string
	RepairCodexRunRef             string
	RepairCodexPromotionRef       string
	RepairCodexAllowBroadSandbox  bool
	RepairCodexSandboxEvidenceRef string
	RepairMaxAttempts             int
	RepairRetryEvidenceRefs       []string
	Promote                       bool
	SkipHealth                    bool
	HealthTimeout                 time.Duration
	CommandTimeout                time.Duration
	CommandOutputMaxBytes         int64
	ArtifactMaxBytes              int64
	EnvAllowlist                  []string
	CandidateAddr                 string
	ServerAddr                    string
	ServerPID                     int
	AttemptRef                    string
	PromotionRef                  string
	RunRef                        string
	WorktreeRef                   string
	BranchRef                     string
	ShutdownRef                   string
	LeaseTTL                      time.Duration
	ShutdownNow                   bool
	ShutdownForced                bool
	ForceAfterTimeout             bool
	ShutdownEscalationEvidenceRef string
	ShutdownTimeout               time.Duration
	ShutdownQueueLimit            int
	OccurredAt                    time.Time
	CandidateStateDir             string
	CandidateRuntimeDir           string
	ManifestPath                  string
	RepairPacketPath              string
	RetentionPolicy               guardianRetentionPolicyV0
	PathPolicy                    guardianPathRootPolicyV0
}

type guardianResultV0 struct {
	SchemaVersion     string                      `json:"schema_version"`
	Status            string                      `json:"status"`
	Phase             string                      `json:"phase,omitempty"`
	Promoted          bool                        `json:"promoted,omitempty"`
	Restored          bool                        `json:"restored,omitempty"`
	RepairStarted     bool                        `json:"repair_started,omitempty"`
	ManifestPath      string                      `json:"manifest_path,omitempty"`
	RepairPacketPath  string                      `json:"repair_packet_path,omitempty"`
	RepairLaunchPath  string                      `json:"repair_launch_packet_path,omitempty"`
	RepairAttemptRef  string                      `json:"repair_attempt_ref,omitempty"`
	FailurePacketHash string                      `json:"failure_packet_hash,omitempty"`
	RepairBlocked     bool                        `json:"repair_blocked,omitempty"`
	RepairBlockReason string                      `json:"repair_block_reason,omitempty"`
	ProjectDir        string                      `json:"project_dir,omitempty"`
	CurrentBin        string                      `json:"current_bin,omitempty"`
	CandidateBin      string                      `json:"candidate_bin,omitempty"`
	LastGoodBin       string                      `json:"last_good_bin,omitempty"`
	Commands          []guardianCommandResultV0   `json:"commands,omitempty"`
	ArtifactManifest  *guardianArtifactManifestV0 `json:"artifact_manifest,omitempty"`
	PathPolicy        guardianPathRootPolicyV0    `json:"path_policy,omitempty"`
	Lease             *guardianLeaseReceiptV0     `json:"lease,omitempty"`
	EvidenceRefs      []string                    `json:"evidence_refs,omitempty"`
	AttemptRef        string                      `json:"attempt_ref,omitempty"`
	PromotionRef      string                      `json:"promotion_ref,omitempty"`
	RunRef            string                      `json:"run_ref,omitempty"`
	WorktreeRef       string                      `json:"worktree_ref,omitempty"`
	BranchRef         string                      `json:"branch_ref,omitempty"`
	ShutdownRef       string                      `json:"shutdown_ref,omitempty"`
	RetentionStatus   guardianRetentionStatusV0   `json:"retention_status,omitempty"`
	Message           string                      `json:"message,omitempty"`
	Shutdown          *guardianShutdownResultV0   `json:"shutdown,omitempty"`
}

type guardianShutdownResultV0 struct {
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

type guardianCommandResultV0 struct {
	Phase                   string                            `json:"phase"`
	Command                 string                            `json:"command"`
	ExitCode                int                               `json:"exit_code"`
	DurationMS              int64                             `json:"duration_ms"`
	OutputPath              string                            `json:"output_path,omitempty"`
	Error                   string                            `json:"error,omitempty"`
	ReasonCode              string                            `json:"reason_code,omitempty"`
	OutputBytes             int64                             `json:"output_bytes,omitempty"`
	OutputTruncated         bool                              `json:"output_truncated,omitempty"`
	OutputReasonCode        string                            `json:"output_reason_code,omitempty"`
	ProcessPolicy           *guardianCandidateProcessPolicyV0 `json:"process_policy,omitempty"`
	StopReceipt             *guardianCandidateStopReceiptV0   `json:"stop_receipt,omitempty"`
	EffectProfile           string                            `json:"effect_profile,omitempty"`
	EffectPolicyRef         string                            `json:"effect_policy_ref,omitempty"`
	EffectAuthorization     string                            `json:"effect_authorization,omitempty"`
	ExternalEffects         []string                          `json:"external_effects,omitempty"`
	EffectEvidenceRefs      []string                          `json:"effect_evidence_refs,omitempty"`
	CommandExpansionChecked bool                              `json:"command_expansion_checked,omitempty"`
}

type guardianCandidateProcessPolicyV0 struct {
	Platform       string `json:"platform"`
	Scope          string `json:"scope"`
	TreeStop       bool   `json:"tree_stop"`
	PublicContract string `json:"public_contract"`
}

type guardianCandidateStopReceiptV0 struct {
	Scope             string `json:"scope"`
	Status            string `json:"status"`
	ReasonCode        string `json:"reason_code"`
	DeadlineExceeded  bool   `json:"deadline_exceeded,omitempty"`
	Escalated         bool   `json:"escalated,omitempty"`
	TreeStopConfirmed bool   `json:"tree_stop_confirmed,omitempty"`
	Ambiguous         bool   `json:"ambiguous,omitempty"`
}

type guardianRepairPacketV0 struct {
	SchemaVersion       string                          `json:"schema_version"`
	FailurePhase        string                          `json:"failure_phase"`
	Summary             string                          `json:"summary"`
	RedactionLevel      string                          `json:"redaction_level"`
	Freshness           string                          `json:"freshness"`
	ReasonCodes         []string                        `json:"reason_codes,omitempty"`
	RepairAttemptRef    string                          `json:"repair_attempt_ref,omitempty"`
	FailurePacketHash   string                          `json:"failure_packet_hash,omitempty"`
	RepairBudget        guardianRepairBudgetV0          `json:"repair_budget,omitempty"`
	Counters            guardianPublicCountersV0        `json:"counters"`
	PathPolicy          guardianPathRootPolicyV0        `json:"path_policy,omitempty"`
	RequiredCommandRefs []string                        `json:"required_command_refs,omitempty"`
	Commands            []guardianPublicCommandResultV0 `json:"commands,omitempty"`
	EvidenceRefs        []string                        `json:"evidence_refs,omitempty"`
	LocalDiagnostics    []guardianLocalDiagnosticRefV0  `json:"local_diagnostics,omitempty"`
	RetentionStatus     guardianRetentionStatusV0       `json:"retention_status,omitempty"`
	CreatedAt           string                          `json:"created_at"`
}

type guardianRepairLaunchPacketV0 struct {
	SchemaVersion      string                      `json:"schema_version"`
	Reason             string                      `json:"reason"`
	FailurePhase       string                      `json:"failure_phase"`
	Objective          string                      `json:"objective"`
	WriteSet           []string                    `json:"write_set"`
	RequiredTests      []string                    `json:"required_tests"`
	AcceptanceCriteria []string                    `json:"acceptance_criteria"`
	ManifestRef        string                      `json:"manifest_ref,omitempty"`
	RepairPacketRef    string                      `json:"repair_packet_ref"`
	RepairAttemptRef   string                      `json:"repair_attempt_ref"`
	FailurePacketHash  string                      `json:"failure_packet_hash"`
	RunRef             string                      `json:"run_ref,omitempty"`
	PromotionRef       string                      `json:"promotion_ref,omitempty"`
	Budget             guardianRepairBudgetV0      `json:"budget"`
	ACKContract        guardianRepairACKContractV0 `json:"ack_contract"`
	SandboxPolicy      guardianRepairSandboxV0     `json:"sandbox_policy"`
	EvidenceRefs       []string                    `json:"evidence_refs,omitempty"`
	CreatedAt          string                      `json:"created_at"`
}

type guardianRepairBudgetV0 struct {
	MaxAgents   int `json:"max_agents"`
	MaxAttempts int `json:"max_attempts,omitempty"`
}

type guardianRepairACKContractV0 struct {
	SchemaVersion string `json:"schema_version"`
	Required      bool   `json:"required"`
	TerminalFile  string `json:"terminal_file"`
}

type guardianRepairSandboxV0 struct {
	Sandbox                 string `json:"sandbox"`
	BroadSandboxAllowed     bool   `json:"broad_sandbox_allowed,omitempty"`
	BroadSandboxEvidenceRef string `json:"broad_sandbox_evidence_ref,omitempty"`
}

type repeatedStringFlagV0 []string

func (values *repeatedStringFlagV0) String() string {
	return strings.Join(*values, "\n")
}

func (values *repeatedStringFlagV0) Set(value string) error {
	value = strings.TrimSpace(value)
	if value != "" {
		*values = append(*values, value)
	}
	return nil
}

func runMain(args []string, stdout io.Writer, stderr io.Writer) int {
	command := "check-promote"
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		command = args[0]
		args = args[1:]
	}
	switch command {
	case "check-promote":
		config, err := parseGuardianConfigV0(args)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "orquesta-guardian: %s\n", guardianPublicConfigErrorV0(err))
			return 2
		}
		result := runGuardianCheckPromoteV0(context.Background(), config)
		_ = json.NewEncoder(stdout).Encode(publicGuardianResultV0(config, result))
		if !guardianCheckPromoteAcceptedV0(result) {
			return 1
		}
		return 0
	case "restore-last-good":
		config, err := parseGuardianConfigV0(args)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "orquesta-guardian: %s\n", guardianPublicConfigErrorV0(err))
			return 2
		}
		result := restoreLastGoodCommandV0(config)
		_ = json.NewEncoder(stdout).Encode(publicGuardianResultV0(config, result))
		if !result.Restored {
			return 1
		}
		return 0
	case "shutdown-server":
		config, err := parseGuardianShutdownConfigV0(args)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "orquesta-guardian: %s\n", guardianPublicConfigErrorV0(err))
			return 2
		}
		result := runGuardianShutdownServerV0(context.Background(), config)
		_ = json.NewEncoder(stdout).Encode(publicGuardianResultV0(config, result))
		if result.Status != guardianStatusShutdownReadyV0 {
			return 1
		}
		return 0
	default:
		_, _ = fmt.Fprintf(stderr, "comando no soportado: %s\n", command)
		return 2
	}
}

func guardianCheckPromoteAcceptedV0(result guardianResultV0) bool {
	return result.Status == guardianStatusPromotedV0 ||
		result.Status == guardianStatusPromotedBreakglassV0
}
