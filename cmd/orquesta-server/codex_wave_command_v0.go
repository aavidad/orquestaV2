package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
)

const codexWaveSummarySchemaVersionV0 = "orquesta_codex_wave_launch.v0"
const codexWaveRegistryFileNameV0 = "codex_wave_registry_v0.json"

type codexWaveLaunchSummaryV0 struct {
	SchemaVersion  string                    `json:"schema_version"`
	WaveRef        string                    `json:"wave_ref"`
	AgentCount     int                       `json:"agent_count"`
	ProjectWorkDir string                    `json:"project_work_dir"`
	RuntimeWorkDir string                    `json:"runtime_work_dir"`
	RegistryPath   string                    `json:"registry_path,omitempty"`
	Sandbox        string                    `json:"sandbox"`
	ApprovalPolicy string                    `json:"approval_policy"`
	CreatedAt      string                    `json:"created_at,omitempty"`
	UpdatedAt      string                    `json:"updated_at,omitempty"`
	DryRun         bool                      `json:"dry_run,omitempty"`
	PurgeReport    *codexWavePurgeReportV0   `json:"purge_report,omitempty"`
	Agents         []codexWaveAgentSummaryV0 `json:"agents"`
	Errors         []codexWavePublicErrorV0  `json:"errors,omitempty"`
}

type codexWaveAgentSummaryV0 struct {
	AgentRef             string                                  `json:"agent_ref"`
	RuntimeWorkDir       string                                  `json:"runtime_work_dir"`
	PromptPath           string                                  `json:"prompt_path"`
	WrapperPath          string                                  `json:"wrapper_path"`
	StdoutPath           string                                  `json:"stdout_path"`
	StderrPath           string                                  `json:"stderr_path"`
	LastMessagePath      string                                  `json:"last_message_path"`
	HomeDir              string                                  `json:"home_dir,omitempty"`
	CodeHomeDir          string                                  `json:"code_home_dir,omitempty"`
	ProcessRef           string                                  `json:"process_ref,omitempty"`
	ProcessProofRef      string                                  `json:"process_proof_ref,omitempty"`
	ProcessProofDigest   string                                  `json:"process_proof_digest,omitempty"`
	SessionRef           string                                  `json:"session_ref,omitempty"`
	LaunchRef            string                                  `json:"launch_ref,omitempty"`
	PID                  int                                     `json:"pid,omitempty"`
	StartedAt            string                                  `json:"started_at,omitempty"`
	StopRequestedAt      string                                  `json:"stop_requested_at,omitempty"`
	StdoutBytes          int64                                   `json:"stdout_bytes,omitempty"`
	StderrBytes          int64                                   `json:"stderr_bytes,omitempty"`
	LastMessageBytes     int64                                   `json:"last_message_bytes,omitempty"`
	Status               string                                  `json:"status"`
	CredentialProjection *codexWaveCredentialProjectionReceiptV0 `json:"credential_projection,omitempty"`
}

type codexWavePublicErrorV0 struct {
	AgentRef string `json:"agent_ref,omitempty"`
	Code     string `json:"code"`
	Field    string `json:"field,omitempty"`
	Message  string `json:"message,omitempty"`
}

type codexWaveConfigV0 struct {
	Agents                     int
	WaveRef                    string
	Prompt                     string
	ProjectWorkDir             string
	RuntimeWorkDir             string
	CommandPath                string
	SourceCodeHome             string
	PathEnv                    string
	Model                      string
	ReasoningEffort            string
	Profile                    string
	Sandbox                    string
	ApprovalPolicy             string
	ExtraArgs                  []string
	IsolateHome                bool
	DryRun                     bool
	PurgeRuntime               bool
	PurgeConfirm               string
	PurgeReportOnly            bool
	AllowUnmanagedLaunch       bool
	UnmanagedLaunchReason      string
	UnmanagedLaunchConfirm     string
	AgentPrompts               []string
	CredentialProjectionPolicy codexWaveCredentialProjectionPolicyV0
}

func codexLaunchWaveCommandV0(args []string, stdout io.Writer, stderr io.Writer) int {
	config, err := codexWaveConfigFromArgsV0(args, stderr)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "codex-launch-wave: %v\n", err)
		return 2
	}
	summary, err := runCodexLaunchWaveV0(context.Background(), config)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "codex-launch-wave: %v\n", err)
		return 1
	}
	_ = json.NewEncoder(stdout).Encode(summary)
	if len(summary.Errors) > 0 {
		return 1
	}
	return 0
}
