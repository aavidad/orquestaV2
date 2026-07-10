package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestaserver "orquesta/modulos/orquesta-server"
)

const (
	serverProjectConfigFileNameV0      = "orquesta.config.json"
	serverProjectConfigSchemaVersionV0 = "orquesta_config.v0"
	configSettingSourceConfigFileV0    = "config_file"
	configFileInvalidPublicCodeV0      = "orquesta_config_file_invalid"
	configFileUnsupportedSchemaCodeV0  = "orquesta_config_schema_unsupported"
)

type serverProjectConfigFileV0 struct {
	SchemaVersion        string                                    `json:"schema_version"`
	Server               serverProjectConfigServerV0               `json:"server,omitempty"`
	ServerHTTP           serverProjectConfigServerHTTPV0           `json:"server_http,omitempty"`
	ServerLifecycle      serverProjectConfigServerLifecycleV0      `json:"server_lifecycle,omitempty"`
	ControlPlane         serverProjectConfigControlPlaneV0         `json:"control_plane,omitempty"`
	ServerSupervisor     serverProjectConfigServerSupervisorV0     `json:"server_supervisor,omitempty"`
	ServerIdle           serverProjectConfigServerIdleV0           `json:"server_idle,omitempty"`
	ServerIdleLegacy     serverProjectConfigServerIdleV0           `json:"server_idle_self_improvement,omitempty"`
	ServerResident       serverProjectConfigServerResidentV0       `json:"server_resident_director,omitempty"`
	WorktreeSnapshot     serverProjectConfigWorktreeSnapshotV0     `json:"worktree_snapshot,omitempty"`
	DaemonLogs           serverProjectConfigDaemonLogsV0           `json:"daemon_logs,omitempty"`
	CodexRuntime         serverProjectConfigCodexRuntimeV0         `json:"codex_runtime,omitempty"`
	CodexModelRouting    *serverProjectConfigCodexModelRoutingV0   `json:"codex_model_routing,omitempty"`
	GeminiRuntime        serverProjectConfigGeminiRuntimeV0        `json:"gemini_runtime,omitempty"`
	ClaudeRuntime        serverProjectConfigClaudeRuntimeV0        `json:"claude_runtime,omitempty"`
	ClaudeModelRouting   *serverProjectConfigClaudeModelRoutingV0  `json:"claude_model_routing,omitempty"`
	RequiredTestRunner   serverProjectConfigRequiredTestRunnerV0   `json:"required_test_runner,omitempty"`
	CodexDirector        serverProjectConfigCodexDirectorV0        `json:"codex_director,omitempty"`
	CodexWave            serverProjectConfigCodexWaveV0            `json:"codex_wave,omitempty"`
	GoalBackend          serverProjectConfigGoalBackendV0          `json:"goal_backend,omitempty"`
	CodexUsageAccounting serverProjectConfigCodexUsageAccountingV0 `json:"codex_usage_accounting,omitempty"`
	CodebaseBroker       serverProjectConfigCodebaseBrokerV0       `json:"codebase_broker,omitempty"`
	WizardBot            serverProjectConfigWizardBotV0            `json:"wizard_bot,omitempty"`
	TelegramOperator     serverProjectConfigTelegramOperatorV0     `json:"telegram_operator,omitempty"`
	DomainWork           serverProjectConfigDomainWorkV0           `json:"domain_work,omitempty"`
	OPES                 serverProjectConfigOPESV0                 `json:"opes,omitempty"`
	OPESBridge           serverProjectConfigOPESBridgeV0           `json:"opes_bridge,omitempty"`
	RailsSecurity        serverProjectConfigRailsSecurityV0        `json:"rails_security,omitempty"`
	EgressSanitizer      serverProjectConfigEgressSanitizerV0      `json:"egress_sanitizer,omitempty"`
	OPESRegistryFinalPkg serverProjectConfigOPESRegistryFinalPkgV0 `json:"opes_registry_finalpkg,omitempty"`
	OPESTopicRegistry    serverProjectConfigOPESTopicRegistryV0    `json:"opes_topic_registry,omitempty"`
	Autoprogramming      serverProjectConfigAutoprogrammingV0      `json:"autoprogramming,omitempty"`
}

type serverProjectConfigServerV0 struct {
	Addr          *string `json:"addr,omitempty"`
	StateDir      *string `json:"state_dir,omitempty"`
	AuditFile     *string `json:"audit_file,omitempty"`
	AuditDisabled *bool   `json:"audit_disabled,omitempty"`
}
type serverProjectConfigServerHTTPV0 struct {
	ReadHeaderTimeoutMS *int `json:"read_header_timeout_ms,omitempty"`
	ReadTimeoutMS       *int `json:"read_timeout_ms,omitempty"`
	WriteTimeoutMS      *int `json:"write_timeout_ms,omitempty"`
	IdleTimeoutMS       *int `json:"idle_timeout_ms,omitempty"`
	MaxHeaderBytes      *int `json:"max_header_bytes,omitempty"`
	ControlBodyMaxBytes *int `json:"control_body_max_bytes,omitempty"`
}
type serverProjectConfigServerLifecycleV0 struct {
	ShutdownGraceMS *int `json:"shutdown_grace_ms,omitempty"`
}
type serverProjectConfigControlPlaneV0 struct {
	RemoteAccessOptIn *bool   `json:"remote_access_opt_in,omitempty"`
	Token             *string `json:"token,omitempty"`
	Principal         *string `json:"principal,omitempty"`
	PermissionRef     *string `json:"permission_ref,omitempty"`
	PublicReason      *string `json:"public_reason,omitempty"`
}
type serverProjectConfigServerSupervisorV0 struct {
	MaxRunsPerTick        *int `json:"max_runs_per_tick,omitempty"`
	MaxExecutionsPerTick  *int `json:"max_executions_per_tick,omitempty"`
	QueueLimit            *int `json:"queue_limit,omitempty"`
	DrainMaxDispatches    *int `json:"drain_max_dispatches,omitempty"`
	DrainMaxCommands      *int `json:"drain_max_commands,omitempty"`
	DrainMaxOutbox        *int `json:"drain_max_outbox,omitempty"`
	DrainMaxExternalWaits *int `json:"drain_max_external_waits,omitempty"`
}
type serverProjectConfigServerIdleV0 struct {
	AfterSeconds            *int      `json:"after_seconds,omitempty"`
	Disabled                *bool     `json:"disabled,omitempty"`
	ProjectWorkDir          *string   `json:"project_workdir,omitempty"`
	ProjectRef              *string   `json:"project_ref,omitempty"`
	WorktreeRef             *string   `json:"worktree_ref,omitempty"`
	BranchRef               *string   `json:"branch_ref,omitempty"`
	Area                    *string   `json:"area,omitempty"`
	WriteSet                *[]string `json:"write_set,omitempty"`
	RequiredTests           *[]string `json:"required_tests,omitempty"`
	ContextRefs             *[]string `json:"context_refs,omitempty"`
	EvidenceRefs            *[]string `json:"evidence_refs,omitempty"`
	Acceptance              *[]string `json:"acceptance,omitempty"`
	GoalFirstEnabled        *bool     `json:"goal_first_enabled,omitempty"`
	FrozenTestsEnabled      *bool     `json:"frozen_tests_enabled,omitempty"`
	CompactRules            *[]string `json:"compact_rules,omitempty"`
	PriorityScore           *int      `json:"priority_score,omitempty"`
	MaxRequests             *int      `json:"max_requests,omitempty"`
	TargetQueue             *int      `json:"target_queue,omitempty"`
	DailyGoalBudget         *int      `json:"daily_goal_budget,omitempty"`
	DailyContextBudgetBytes *int      `json:"daily_context_budget_bytes,omitempty"`
}
type serverProjectConfigWorktreeSnapshotV0 struct {
	MaxFiles      *int `json:"max_files,omitempty"`
	MaxFileBytes  *int `json:"max_file_bytes,omitempty"`
	MaxTotalBytes *int `json:"max_total_bytes,omitempty"`
}
type serverProjectConfigDaemonLogsV0 struct {
	MaxBytes        *int    `json:"max_bytes,omitempty"`
	MaxRotatedFiles *int    `json:"max_rotated_files,omitempty"`
	RetentionDays   *int    `json:"retention_days,omitempty"`
	LocalRawEnabled *bool   `json:"local_raw_enabled,omitempty"`
	LocalRawReason  *string `json:"local_raw_reason,omitempty"`
}
type serverProjectConfigCodexRuntimeV0 struct {
	RuntimeWorkDir     *string `json:"runtime_work_dir,omitempty"`
	ExecutionMode      *string `json:"execution_mode,omitempty"`
	ReasoningEffort    *string `json:"reasoning_effort,omitempty"`
	MaxExpectedSeconds *int    `json:"max_expected_seconds,omitempty"`
	MaxBatchReady      *int    `json:"max_batch_ready,omitempty"`
	MaxConcurrency     *int    `json:"max_concurrency,omitempty"`
}

type serverProjectConfigGeminiRuntimeV0 struct {
	Enabled        *bool    `json:"enabled,omitempty"`
	CommandPath    *string  `json:"command_path,omitempty"`
	ProjectWorkDir *string  `json:"project_work_dir,omitempty"`
	RuntimeWorkDir *string  `json:"runtime_work_dir,omitempty"`
	HomeDir        *string  `json:"home_dir,omitempty"`
	Path           *string  `json:"path,omitempty"`
	Model          *string  `json:"model,omitempty"`
	ApprovalMode   *string  `json:"approval_mode,omitempty"`
	OutputFormat   *string  `json:"output_format,omitempty"`
	ExtraArgs      []string `json:"extra_args,omitempty"`
}

type serverProjectConfigRequiredTestRunnerV0 struct {
	Enabled         *bool             `json:"enabled,omitempty"`
	GoCommand       *string           `json:"go_command,omitempty"`
	AllowedCommands map[string]string `json:"allowed_commands,omitempty"`
	OutputDir       *string           `json:"output_dir,omitempty"`
	Environment     map[string]string `json:"environment,omitempty"`
	MaxOutputBytes  *int              `json:"max_output_bytes,omitempty"`
	MaxArtifacts    *int              `json:"max_artifacts,omitempty"`
}

type serverProjectConfigCodexDirectorV0 struct {
	WaveAgents           *int `json:"wave_agents,omitempty"`
	MaxSubagentsPerAgent *int `json:"max_subagents_per_agent,omitempty"`
	RecursiveAgentBudget *int `json:"recursive_agent_budget,omitempty"`
}

type serverProjectConfigGoalBackendV0 struct {
	Kind                          *string `json:"kind,omitempty"`
	PromptLocale                  *string `json:"prompt_locale,omitempty"`
	TimeoutMS                     *int    `json:"timeout_ms,omitempty"`
	PreflightTimeoutMS            *int    `json:"preflight_timeout_ms,omitempty"`
	AllowAppServerProxyDiagnostic *bool   `json:"allow_app_server_proxy_diagnostic,omitempty"`
}

type serverProjectConfigCodexUsageAccountingV0 struct {
	Mode        *string `json:"mode,omitempty"`
	LogMaxBytes *int    `json:"log_max_bytes,omitempty"`
}

type serverProjectConfigCodebaseBrokerV0 struct {
	ProviderKind                *string `json:"provider_kind,omitempty"`
	ExternalIndexerEnabled      *bool   `json:"external_indexer_enabled,omitempty"`
	MaxConcurrent               *int    `json:"max_concurrent,omitempty"`
	TimeoutMS                   *int    `json:"timeout_ms,omitempty"`
	StateDir                    *string `json:"state_dir,omitempty"`
	WatchdogEnabled             *bool   `json:"watchdog_enabled,omitempty"`
	WatchdogStopOrphans         *bool   `json:"watchdog_stop_orphans,omitempty"`
	WatchdogOrphanMinAgeSeconds *int    `json:"watchdog_orphan_min_age_seconds,omitempty"`
	Command                     *string `json:"command,omitempty"`
	ProjectName                 *string `json:"project_name,omitempty"`
}

type serverProjectConfigDomainWorkV0 struct {
	HTTPBaseURL        *string   `json:"http_base_url,omitempty"`
	HTTPDomainRef      *string   `json:"http_domain_ref,omitempty"`
	FileDir            *string   `json:"file_dir,omitempty"`
	FileEnabled        *bool     `json:"file_enabled,omitempty"`
	HTTPCreatePath     *string   `json:"http_create_path,omitempty"`
	HTTPSubmitPath     *string   `json:"http_submit_path,omitempty"`
	HTTPTimeoutSeconds *int      `json:"http_timeout_seconds,omitempty"`
	HTTPEgressMode     *string   `json:"http_egress_mode,omitempty"`
	HTTPAllowedHosts   *[]string `json:"http_allowed_hosts,omitempty"`
	DeliveryLedgerPath *string   `json:"delivery_ledger_path,omitempty"`
}

type serverProjectConfigRailsSecurityV0 struct {
	SecurityMode               *string `json:"security_mode,omitempty"`
	RailsMode                  *string `json:"rails_mode,omitempty"`
	DetailProhibitedRails      *string `json:"detail_prohibited_rails,omitempty"`
	DetailProhibitedRailsScope *string `json:"detail_prohibited_rails_scope,omitempty"`
}

type serverProjectConfigEgressSanitizerV0 struct {
	Enabled      *bool                                       `json:"enabled,omitempty"`
	SanitizerRef *string                                     `json:"sanitizer_ref,omitempty"`
	LocalModel   serverProjectConfigEgressSanitizerModelV0   `json:"local_model,omitempty"`
	Sidecar      serverProjectConfigEgressSanitizerSidecarV0 `json:"sidecar,omitempty"`
}

type serverProjectConfigEgressSanitizerModelV0 struct {
	Enabled     *bool   `json:"enabled,omitempty"`
	ModelRef    *string `json:"model_ref,omitempty"`
	ModelPath   *string `json:"model_path,omitempty"`
	RuntimeRef  *string `json:"runtime_ref,omitempty"`
	EvidenceRef *string `json:"evidence_ref,omitempty"`
}

type serverProjectConfigEgressSanitizerSidecarV0 struct {
	Enabled       *bool   `json:"enabled,omitempty"`
	SidecarRef    *string `json:"sidecar_ref,omitempty"`
	AdapterRef    *string `json:"adapter_ref,omitempty"`
	TransportRef  *string `json:"transport_ref,omitempty"`
	EvidenceRef   *string `json:"evidence_ref,omitempty"`
	Command       *string `json:"command,omitempty"`
	LocalEndpoint *string `json:"local_endpoint,omitempty"`
}

func serverProjectConfigFilePathV0(projectDir string) string {
	return filepath.Join(projectDir, serverProjectConfigFileNameV0)
}

func loadServerProjectConfigFileV0(projectDir string) (serverProjectConfigFileV0, bool, error) {
	projectDir = strings.TrimSpace(projectDir)
	if projectDir == "" {
		return serverProjectConfigFileV0{}, false, nil
	}
	return loadServerProjectConfigPathV0(serverProjectConfigFilePathV0(projectDir))
}

func loadServerProjectConfigPathV0(path string) (serverProjectConfigFileV0, bool, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return serverProjectConfigFileV0{}, false, nil
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return serverProjectConfigFileV0{}, false, fmt.Errorf("%s: path", configFileInvalidPublicCodeV0)
	}
	raw, err := os.ReadFile(abs)
	if errors.Is(err, os.ErrNotExist) {
		return serverProjectConfigFileV0{}, false, nil
	}
	if err != nil {
		return serverProjectConfigFileV0{}, false, fmt.Errorf("%s: read", configFileInvalidPublicCodeV0)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var config serverProjectConfigFileV0
	if err := decoder.Decode(&config); err != nil {
		return serverProjectConfigFileV0{}, false, fmt.Errorf("%s: json", configFileInvalidPublicCodeV0)
	}
	if strings.TrimSpace(config.SchemaVersion) != serverProjectConfigSchemaVersionV0 {
		return serverProjectConfigFileV0{}, false, fmt.Errorf("%s: %s", configFileUnsupportedSchemaCodeV0, serverProjectConfigSchemaVersionV0)
	}
	return config, true, nil
}

func resolveServerProjectConfigV0(
	projectDir string,
	explicitPath string,
) (serverProjectConfigFileV0, string, error) {
	explicitPath = strings.TrimSpace(explicitPath)
	if explicitPath != "" {
		abs, err := filepath.Abs(explicitPath)
		if err != nil {
			return serverProjectConfigFileV0{}, "", fmt.Errorf("%s: path", configFileInvalidPublicCodeV0)
		}
		config, ok, err := loadServerProjectConfigPathV0(abs)
		if err != nil {
			return serverProjectConfigFileV0{}, "", err
		}
		if !ok {
			return serverProjectConfigFileV0{}, "", fmt.Errorf("%s: read", configFileInvalidPublicCodeV0)
		}
		return config, abs, nil
	}
	config, ok, err := loadServerProjectConfigFileV0(projectDir)
	if err != nil {
		return serverProjectConfigFileV0{}, "", err
	}
	if !ok {
		return serverProjectConfigFileV0{}, "", nil
	}
	return config, serverProjectConfigFilePathV0(projectDir), nil
}

func projectConfigFromProjectDirBestEffortV0(projectDir string) serverProjectConfigFileV0 {
	config, _, err := loadServerProjectConfigFileV0(projectDir)
	if err != nil {
		return serverProjectConfigFileV0{}
	}
	return config
}

func projectConfigFromServerConfigBestEffortV0(config orquestaserver.ConfigV0) serverProjectConfigFileV0 {
	config = orquestaserver.NormalizeConfigV0(config)
	if strings.TrimSpace(config.ProjectConfigFilePath) != "" {
		projectConfig, _, err := loadServerProjectConfigPathV0(config.ProjectConfigFilePath)
		if err == nil {
			return projectConfig
		}
		return serverProjectConfigFileV0{}
	}
	return projectConfigFromProjectDirBestEffortV0(config.ProjectWorkDir)
}

func serverAutoprogrammingGoalProgressPolicyConfigFromProjectFileV0(
	fileConfig serverProjectConfigFileV0,
) orquestaserver.AutoprogrammingGoalProgressPolicyConfigV0 {
	return orquestaserver.NormalizeAutoprogrammingGoalProgressPolicyConfigV0(
		orquestaserver.AutoprogrammingGoalProgressPolicyConfigV0{
			CheckpointOnlyHighConsumptionTokens: int64(intProjectConfigOrEnvOrDefaultV0(
				envAutoprogrammingCheckpointOnlyHighConsumptionTokensV0,
				fileConfig.Autoprogramming.CheckpointOnlyHighConsumptionTokens,
				0,
			)),
			CheckpointOnlyMaxWaitSeconds: int64(intProjectConfigOrEnvOrDefaultV0(
				envAutoprogrammingCheckpointOnlyMaxWaitSecondsV0,
				fileConfig.Autoprogramming.CheckpointOnlyMaxWaitSeconds,
				0,
			)),
			NoCheckpointWarningMaxWaitSeconds: int64(intProjectConfigOrEnvOrDefaultV0(
				envAutoprogrammingNoCheckpointWarningMaxWaitSecondsV0,
				fileConfig.Autoprogramming.NoCheckpointWarningMaxWaitSeconds,
				0,
			)),
		},
	)
}

func serverAutoprogrammingGoalProgressPolicyFromConfigV0(config orquestaserver.ConfigV0) orquestamcp.MCPAutoprogrammingGoalProgressPolicyV0 {
	policy := config.AutoprogrammingGoalProgressPolicy
	return orquestamcp.NormalizeMCPAutoprogrammingGoalProgressPolicyV0(
		orquestamcp.MCPAutoprogrammingGoalProgressPolicyV0{
			CheckpointOnlyHighConsumptionTokens: policy.CheckpointOnlyHighConsumptionTokens,
			CheckpointOnlyMaxWaitSeconds:        policy.CheckpointOnlyMaxWaitSeconds,
			NoCheckpointWarningMaxWaitSeconds:   policy.NoCheckpointWarningMaxWaitSeconds,
		},
	)
}

func intProjectConfigOrEnvOrDefaultV0(key string, fileValue *int, fallback int) int {
	if strings.TrimSpace(os.Getenv(key)) != "" {
		return intEnvOrDefaultV0(key, fallback)
	}
	if fileValue != nil && *fileValue > 0 {
		return *fileValue
	}
	return fallback
}

func int64ProjectConfigOrEnvOrDefaultV0(key string, fileValue *int, fallback int64) int64 {
	if strings.TrimSpace(os.Getenv(key)) != "" {
		return int64EnvOrDefaultV0(key, fallback)
	}
	if fileValue != nil && *fileValue > 0 {
		return int64(*fileValue)
	}
	return fallback
}

func stringProjectConfigOrEnvOrDefaultV0(key string, fileValue *string, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	if fileValue != nil {
		if value := strings.TrimSpace(*fileValue); value != "" {
			return value
		}
	}
	return fallback
}

func stringSliceProjectConfigOrEnvOrDefaultV0(key string, fileValue *[]string, fallback []string) []string {
	if strings.TrimSpace(os.Getenv(key)) != "" {
		return csvEnvOrDefaultV0(key, fallback)
	}
	if fileValue == nil {
		return append([]string(nil), fallback...)
	}
	out := make([]string, 0, len(*fileValue))
	for _, item := range *fileValue {
		value := strings.TrimSpace(item)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func boolProjectConfigOrEnvOrDefaultV0(key string, fileValue *bool, fallback bool) bool {
	if strings.TrimSpace(os.Getenv(key)) != "" {
		return boolEnvOrDefaultV0(key, fallback)
	}
	if fileValue != nil {
		return *fileValue
	}
	return fallback
}

func codexGoalBackendFromProjectConfigFileV0(fileConfig serverProjectConfigFileV0) string {
	return stringProjectConfigOrEnvOrDefaultV0(envCodexGoalBackendV0, fileConfig.GoalBackend.Kind, "")
}

func goalBackendPromptLocaleFromProjectConfigFileV0(fileConfig serverProjectConfigFileV0) string {
	if fileConfig.GoalBackend.PromptLocale != nil {
		if value := strings.TrimSpace(*fileConfig.GoalBackend.PromptLocale); value != "" {
			return value
		}
	}
	return "es-ES"
}

func codexGoalTimeoutMSFromProjectConfigFileV0(fileConfig serverProjectConfigFileV0) int {
	return intProjectConfigOrEnvOrDefaultV0(
		envCodexGoalTimeoutMSV0,
		fileConfig.GoalBackend.TimeoutMS,
		defaultCodexGoalTimeoutMSV0,
	)
}

func codexGoalPreflightTimeoutMSFromProjectConfigFileV0(fileConfig serverProjectConfigFileV0) int {
	return intProjectConfigOrEnvOrDefaultV0(
		envCodexGoalPreflightTimeoutMSV0,
		fileConfig.GoalBackend.PreflightTimeoutMS,
		defaultCodexGoalPreflightTimeoutMSV0,
	)
}

func codexGoalBackendProxyDiagnosticAllowedFromProjectConfigFileV0(fileConfig serverProjectConfigFileV0) bool {
	return boolProjectConfigOrEnvOrDefaultV0(
		envAllowAppServerProxyDiagnosticV0,
		fileConfig.GoalBackend.AllowAppServerProxyDiagnostic,
		false,
	)
}

func absDirProjectConfigOrEnvOrDefaultV0(key string, fileValue *string, fallback string) string {
	value := stringProjectConfigOrEnvOrDefaultV0(key, fileValue, fallback)
	abs, err := filepath.Abs(value)
	if err != nil {
		return fallback
	}
	_ = os.MkdirAll(abs, 0o700)
	return abs
}

func configSettingSourceFromEnvOrProjectConfigV0(projectDir string, key string) string {
	if strings.TrimSpace(os.Getenv(key)) != "" {
		return "explicit"
	}
	fileConfig, ok, err := loadServerProjectConfigFileV0(projectDir)
	if err != nil || !ok {
		return "defaulted"
	}
	if serverProjectConfigHasEffectiveValueForEnvKeyV0(fileConfig, key) {
		return configSettingSourceConfigFileV0
	}
	return "defaulted"
}

func configSettingSourceFromConfigOrProjectConfigV0(config orquestaserver.ConfigV0, key string) string {
	config = orquestaserver.NormalizeConfigV0(config)
	fileConfig := serverProjectConfigFileV0{}
	ok := false
	if strings.TrimSpace(config.ProjectConfigFilePath) != "" {
		loaded, loadedOK, err := loadServerProjectConfigPathV0(config.ProjectConfigFilePath)
		if err == nil {
			fileConfig = loaded
			ok = loadedOK
		}
	} else {
		loaded, loadedOK, err := loadServerProjectConfigFileV0(config.ProjectWorkDir)
		if err == nil {
			fileConfig = loaded
			ok = loadedOK
		}
	}
	if !ok {
		if strings.TrimSpace(os.Getenv(key)) != "" {
			return "explicit"
		}
		return "defaulted"
	}
	if strings.TrimSpace(os.Getenv(key)) != "" {
		if serverConfigUsesDaemonSnapshotV0(config) && serverProjectConfigEffectiveValueMatchesEnvV0(fileConfig, key) {
			return configSettingSourceConfigFileV0
		}
		return "explicit"
	}
	if serverProjectConfigHasEffectiveValueForEnvKeyV0(fileConfig, key) {
		return configSettingSourceConfigFileV0
	}
	return "defaulted"
}

func serverConfigUsesDaemonSnapshotV0(config orquestaserver.ConfigV0) bool {
	config = orquestaserver.NormalizeConfigV0(config)
	path := strings.TrimSpace(config.ProjectConfigFilePath)
	stateDir := strings.TrimSpace(config.StateDir)
	if path == "" || stateDir == "" {
		return false
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	absSnapshotPath, err := filepath.Abs(filepath.Join(stateDir, serverDaemonConfigSnapshotDirV0, serverProjectConfigFileNameV0))
	if err != nil {
		return false
	}
	return absPath == absSnapshotPath
}

func serverProjectConfigHasEffectiveValueForEnvKeyV0(config serverProjectConfigFileV0, key string) bool {
	if value := serverProjectConfigAutoprogrammingValueForEnvKeyV0(config.Autoprogramming, key); value != nil {
		return *value > 0
	}
	switch key {
	case envServerAddrV0:
		return configStringPointerHasValueV0(config.Server.Addr)
	case envServerStateDirV0:
		return configStringPointerHasValueV0(config.Server.StateDir)
	case envServerAuditFileV0:
		return configStringPointerHasValueV0(config.Server.AuditFile)
	case envServerAuditDisabledV0:
		return config.Server.AuditDisabled != nil
	case envServerRemoteControlPlaneConfirmV0:
		return config.ControlPlane.RemoteAccessOptIn != nil
	case envServerControlTokenV0:
		return configStringPointerHasValueV0(config.ControlPlane.Token)
	case envServerControlPrincipalV0:
		return configStringPointerHasValueV0(config.ControlPlane.Principal)
	case envServerControlPermissionRefV0:
		return configStringPointerHasValueV0(config.ControlPlane.PermissionRef)
	case envServerControlPublicReasonV0:
		return configStringPointerHasValueV0(config.ControlPlane.PublicReason)
	case envServerReadHeaderTimeoutMSV0:
		return configIntPointerPositiveV0(config.ServerHTTP.ReadHeaderTimeoutMS)
	case envServerReadTimeoutMSV0:
		return configIntPointerPositiveV0(config.ServerHTTP.ReadTimeoutMS)
	case envServerWriteTimeoutMSV0:
		return configIntPointerPositiveV0(config.ServerHTTP.WriteTimeoutMS)
	case envServerIdleTimeoutMSV0:
		return configIntPointerPositiveV0(config.ServerHTTP.IdleTimeoutMS)
	case envServerMaxHeaderBytesV0:
		return configIntPointerPositiveV0(config.ServerHTTP.MaxHeaderBytes)
	case envServerControlBodyMaxBytesV0:
		return configIntPointerPositiveV0(config.ServerHTTP.ControlBodyMaxBytes)
	case envServerShutdownGraceMSV0:
		return configIntPointerPositiveV0(config.ServerLifecycle.ShutdownGraceMS)
	case envServerMaxRunsPerTickV0:
		return configIntPointerPositiveV0(config.ServerSupervisor.MaxRunsPerTick)
	case envServerMaxExecutionsPerTickV0:
		return configIntPointerPositiveV0(config.ServerSupervisor.MaxExecutionsPerTick)
	case envServerQueueLimitV0:
		return configIntPointerPositiveV0(config.ServerSupervisor.QueueLimit)
	case envServerDrainMaxDispatchesV0:
		return configIntPointerPositiveV0(config.ServerSupervisor.DrainMaxDispatches)
	case envServerDrainMaxCommandsV0:
		return configIntPointerPositiveV0(config.ServerSupervisor.DrainMaxCommands)
	case envServerDrainMaxOutboxV0:
		return configIntPointerPositiveV0(config.ServerSupervisor.DrainMaxOutbox)
	case envServerDrainMaxExternalWaitsV0:
		return configIntPointerPositiveV0(config.ServerSupervisor.DrainMaxExternalWaits)
	case envServerIdleSelfImprovementMaxRequestsV0:
		return serverIdleProjectConfigHasValueForEnvKeyV0(config, key)
	case envServerIdleSelfImprovementAfterV0, envServerIdleSelfImprovementDisabledV0,
		envServerIdleSelfImprovementProjectWorkDirV0, envServerIdleSelfImprovementProjectRefV0,
		envServerIdleSelfImprovementWorktreeRefV0, envServerIdleSelfImprovementBranchRefV0,
		envServerIdleSelfImprovementAreaV0, envServerIdleSelfImprovementWriteSetV0,
		envServerIdleSelfImprovementRequiredTestsV0, envServerIdleSelfImprovementContextRefsV0,
		envServerIdleSelfImprovementEvidenceRefsV0, envServerIdleSelfImprovementAcceptanceV0,
		envServerIdleSelfImprovementGoalFirstV0, envServerIdleSelfImprovementFrozenTestsV0,
		envServerIdleSelfImprovementCompactRulesV0, envServerIdleSelfImprovementPriorityScoreV0,
		envServerIdleSelfImprovementTargetQueueV0, envServerIdleSelfImprovementDailyGoalBudgetV0,
		envServerIdleSelfImprovementDailyContextBudgetBytesV0:
		return serverIdleProjectConfigHasValueForEnvKeyV0(config, key)
	case envServerResidentDirectorMaxActionsV0:
		return configIntPointerPositiveV0(config.ServerResident.MaxActions)
	case envWorktreeSnapshotMaxFilesV0:
		return configIntPointerPositiveV0(config.WorktreeSnapshot.MaxFiles)
	case envWorktreeSnapshotMaxFileBytesV0:
		return configIntPointerPositiveV0(config.WorktreeSnapshot.MaxFileBytes)
	case envWorktreeSnapshotMaxTotalBytesV0:
		return configIntPointerPositiveV0(config.WorktreeSnapshot.MaxTotalBytes)
	case envServerDaemonLogMaxBytesV0:
		return configIntPointerPositiveV0(config.DaemonLogs.MaxBytes)
	case envServerDaemonLogMaxRotatedV0:
		return configIntPointerPositiveV0(config.DaemonLogs.MaxRotatedFiles)
	case envServerDaemonLogRetentionDaysV0:
		return configIntPointerPositiveV0(config.DaemonLogs.RetentionDays)
	case envServerDaemonLogRawEnabledV0:
		return config.DaemonLogs.LocalRawEnabled != nil
	case envServerDaemonLogRawReasonV0:
		return configStringPointerHasValueV0(config.DaemonLogs.LocalRawReason)
	case envCodexRuntimeWorkDirV0:
		return configStringPointerHasValueV0(config.CodexRuntime.RuntimeWorkDir)
	case envCodexExecutionModeV0:
		return configStringPointerHasValueV0(config.CodexRuntime.ExecutionMode)
	case envCodexReasoningEffortV0:
		return configStringPointerHasValueV0(config.CodexRuntime.ReasoningEffort)
	case envCodexMaxExpectedSecondsV0:
		return configIntPointerPositiveV0(config.CodexRuntime.MaxExpectedSeconds)
	case envCodexMaxBatchReadyV0:
		return configIntPointerPositiveV0(config.CodexRuntime.MaxBatchReady)
	case envCodexMaxConcurrencyV0:
		return configIntPointerPositiveV0(config.CodexRuntime.MaxConcurrency)
	case envCodexDirectorWaveAgentsV0:
		return configIntPointerPositiveV0(config.CodexDirector.WaveAgents)
	case envCodexDirectorMaxSubagentsPerAgentV0:
		return configIntPointerPositiveV0(config.CodexDirector.MaxSubagentsPerAgent)
	case envCodexDirectorRecursiveAgentBudgetV0:
		return configIntPointerPositiveV0(config.CodexDirector.RecursiveAgentBudget)
	case envCodexGoalBackendV0:
		return configStringPointerHasValueV0(config.GoalBackend.Kind)
	case envCodexGoalTimeoutMSV0:
		return configIntPointerPositiveV0(config.GoalBackend.TimeoutMS)
	case envCodexGoalPreflightTimeoutMSV0:
		return configIntPointerPositiveV0(config.GoalBackend.PreflightTimeoutMS)
	case envAllowAppServerProxyDiagnosticV0:
		return config.GoalBackend.AllowAppServerProxyDiagnostic != nil
	case envCodexWaveAgentsV0, envCodexWaveRefV0, envCodexWaveRuntimeWorkDirV0,
		envCodexWaveSourceCodeHomeV0, envCodexWaveModelV0, envCodexWaveReasoningEffortV0,
		envCodexWaveProfileV0, envCodexWaveSandboxV0, envCodexWaveApprovalPolicyV0,
		envCodexWaveExtraArgsV0, envCodexWaveIsolateHomeV0,
		envCodexWaveStrictCredentialProjectionV0, envCodexWaveProjectMemoriesV0,
		envCodexWaveProjectionMaxFilesV0, envCodexWaveProjectionMaxFileBytesV0,
		envCodexWaveProjectionMaxTotalBytesV0, envCodexWavePurgeRuntimeV0,
		envCodexWavePurgeRuntimeConfirmV0, envCodexWavePurgeRuntimeReportV0,
		envCodexWaveAllowUnmanagedLaunchV0, envCodexWaveUnmanagedLaunchReasonV0,
		envCodexWaveUnmanagedLaunchConfirmV0, envCodexWavePathV0,
		envCodexWaveTailReasonV0, envCodexWaveStopConfirmV0, envCodexWaveStopForceV0:
		return codexWaveProjectConfigHasValueForEnvKeyV0(config.CodexWave, key)
	case envCodexUsageAccountingV0:
		return configStringPointerHasValueV0(config.CodexUsageAccounting.Mode)
	case envCodexUsageLogMaxBytesV0:
		return configIntPointerPositiveV0(config.CodexUsageAccounting.LogMaxBytes)
	case envCodebaseBrokerProviderKindV0:
		return configStringPointerHasValueV0(config.CodebaseBroker.ProviderKind)
	case envCodebaseBrokerExternalIndexerEnabledV0:
		return config.CodebaseBroker.ExternalIndexerEnabled != nil
	case envCodebaseBrokerMaxConcurrentV0:
		return configIntPointerPositiveV0(config.CodebaseBroker.MaxConcurrent)
	case envCodebaseBrokerTimeoutMSV0:
		return configIntPointerPositiveV0(config.CodebaseBroker.TimeoutMS)
	case envCodebaseBrokerStateDirV0:
		return configStringPointerHasValueV0(config.CodebaseBroker.StateDir)
	case envCodebaseBrokerWatchdogEnabledV0:
		return config.CodebaseBroker.WatchdogEnabled != nil
	case envCodebaseBrokerWatchdogStopOrphansV0:
		return config.CodebaseBroker.WatchdogStopOrphans != nil
	case envCodebaseBrokerWatchdogOrphanMinAgeSecondsV0:
		return configIntPointerPositiveV0(config.CodebaseBroker.WatchdogOrphanMinAgeSeconds)
	case envCodebaseBrokerCommandV0:
		return configStringPointerHasValueV0(config.CodebaseBroker.Command)
	case envCodebaseBrokerProjectNameV0:
		return configStringPointerHasValueV0(config.CodebaseBroker.ProjectName)
	case envDomainWorkHTTPBaseURLV0:
		return configStringPointerHasValueV0(config.DomainWork.HTTPBaseURL)
	case envDomainWorkHTTPDomainRefV0:
		return configStringPointerHasValueV0(config.DomainWork.HTTPDomainRef)
	case envDomainWorkFileDirV0:
		return configStringPointerHasValueV0(config.DomainWork.FileDir)
	case envDomainWorkFileEnabledV0:
		return config.DomainWork.FileEnabled != nil
	case envDomainWorkHTTPCreatePathV0:
		return configStringPointerHasValueV0(config.DomainWork.HTTPCreatePath)
	case envDomainWorkHTTPSubmitPathV0:
		return configStringPointerHasValueV0(config.DomainWork.HTTPSubmitPath)
	case envDomainWorkHTTPTimeoutSecondsV0:
		return configIntPointerPositiveV0(config.DomainWork.HTTPTimeoutSeconds)
	case envDomainWorkHTTPEgressModeV0:
		return configStringPointerHasValueV0(config.DomainWork.HTTPEgressMode)
	case envDomainWorkHTTPAllowedHostsV0:
		return configStringSlicePointerHasValueV0(config.DomainWork.HTTPAllowedHosts)
	case envDomainDeliveryLedgerPathV0:
		return configStringPointerHasValueV0(config.DomainWork.DeliveryLedgerPath)
	case envOPESBaseURLV0:
		return configStringPointerHasValueV0(config.OPES.BaseURL)
	case envOPESProjectWorkDirV0:
		return configStringPointerHasValueV0(config.OPES.ProjectWorkDir)
	case envOPESTemporalConfirmV0:
		return config.OPES.TemporalConfirm != nil
	case envOPESBridgeEnabledV0:
		return config.OPESBridge.Enabled != nil
	case envOPESBridgeConfirmV0:
		return config.OPESBridge.Confirm != nil
	case envOPESBridgeDryRunV0:
		return config.OPESBridge.DryRun != nil
	case envOPESBridgeJobTypeV0:
		return configStringPointerHasValueV0(config.OPESBridge.JobType)
	case envOPESBridgeJobRefV0:
		return configStringPointerHasValueV0(config.OPESBridge.JobRef)
	case envOPESBridgeProgramIDV0:
		return configStringPointerHasValueV0(config.OPESBridge.ProgramID)
	case envOPESBridgeTopicIDV0:
		return configStringPointerHasValueV0(config.OPESBridge.TopicID)
	case envOPESBridgeCorrelationIDV0:
		return configStringPointerHasValueV0(config.OPESBridge.CorrelationID)
	case envOPESBridgeJobTypeSequenceV0:
		return configStringSlicePointerHasValueV0(config.OPESBridge.JobTypeSequence)
	case envOPESBridgeLimitV0:
		return configIntPointerPositiveV0(config.OPESBridge.Limit)
	case envOPESBridgeTimeoutSecondsV0:
		return configIntPointerPositiveV0(config.OPESBridge.TimeoutSeconds)
	case envOPESBridgePriorityV0:
		return configIntPointerPositiveV0(config.OPESBridge.Priority)
	case envOPESBridgeIntervalSecondsV0:
		return configIntPointerPositiveV0(config.OPESBridge.IntervalSeconds)
	case envOPESBridgeInitialDelaySecondsV0:
		return configIntPointerPositiveV0(config.OPESBridge.InitialDelaySeconds)
	case envOPESBridgeMaxTicksV0:
		return config.OPESBridge.MaxTicks != nil
	case envOPESBridgeSuperviseSubmittedV0:
		return config.OPESBridge.SuperviseSubmitted != nil
	case envOPESBridgeWaitResidentSecondsV0:
		return config.OPESBridge.WaitResidentSeconds != nil
	case envOPESBridgeWaitResidentIntervalMSV0:
		return configIntPointerPositiveV0(config.OPESBridge.WaitResidentIntervalMS)
	case envOPESBridgeAllowUnfilteredV0:
		return config.OPESBridge.AllowUnfiltered != nil
	case envOPESBridgeInputLedgerDisabledV0:
		return config.OPESBridge.InputLedgerDisabled != nil
	case envOPESBridgeInputLedgerPathV0:
		return configStringPointerHasValueV0(config.OPESBridge.InputLedgerPath)
	case envOPESBridgeProductiveConfirmV0:
		return config.OPESBridge.ProductiveConfirm != nil
	case envOPESBridgeDestinationEvidenceV0:
		return configStringPointerHasValueV0(config.OPESBridge.DestinationEvidenceRef)
	case envOPESBridgeRequireRuntimeCompatibilityV0:
		return config.OPESBridge.RequireRuntimeCompatibility != nil
	case envOPESBridgeRequiredRuntimeBinarySHA256V0:
		return configStringPointerHasValueV0(config.OPESBridge.RequiredRuntimeBinarySHA256)
	case envOPESBridgeRequiredRuntimeBuildRefV0:
		return configStringPointerHasValueV0(config.OPESBridge.RequiredRuntimeBuildRef)
	case envOPESBridgeRequiredRuntimeCommitRefV0:
		return configStringPointerHasValueV0(config.OPESBridge.RequiredRuntimeCommitRef)
	case envOPESBridgeSpeechSynthesisCapabilityV0:
		return configStringPointerHasValueV0(config.OPESBridge.SpeechSynthesis.Capability)
	case envOPESBridgeSpeechSynthesisCapabilityRefV0:
		return configStringPointerHasValueV0(config.OPESBridge.SpeechSynthesis.CapabilityRef)
	case envOPESBridgeSpeechSynthesisEvidenceRefsV0:
		return configStringPointerHasValueV0(config.OPESBridge.SpeechSynthesis.EvidenceRefs)
	case envOPESBridgeSpeechSynthesisReasonV0:
		return configStringPointerHasValueV0(config.OPESBridge.SpeechSynthesis.Reason)
	case envOPESBridgeSpeechSynthesisNetworkReadyV0:
		return configStringPointerHasValueV0(config.OPESBridge.SpeechSynthesis.NetworkReady)
	case envOPESBridgeSpeechSynthesisToolPathReadyV0:
		return configStringPointerHasValueV0(config.OPESBridge.SpeechSynthesis.ToolPathReady)
	case envOPESBridgeSpeechSynthesisQuotaReadyV0:
		return configStringPointerHasValueV0(config.OPESBridge.SpeechSynthesis.ProviderQuotaReady)
	case envOPESBridgeSpeechSynthesisProgressHeartbeatReadyV0:
		return configStringPointerHasValueV0(config.OPESBridge.SpeechSynthesisProgressReady)
	case envOPESBridgeSpeechSynthesisProviderTimeoutReadyV0:
		return configStringPointerHasValueV0(config.OPESBridge.SpeechSynthesisTimeoutReady)
	case envOPESBridgeSpeechSynthesisNoProgressTimeoutSecondsV0:
		return configIntPointerPositiveV0(config.OPESBridge.SpeechSynthesisNoProgress)
	case envOPESBridgeSpeechSynthesisToolWorkDirV0:
		return configStringPointerHasValueV0(config.OPESBridge.SpeechSynthesisToolWorkDir)
	case envOPESBridgeSpeechSynthesisToolCommandV0:
		return configStringPointerHasValueV0(config.OPESBridge.SpeechSynthesisToolCommand)
	case envOPESBridgeSpeechSynthesisToolPreflightTimeoutSecondsV0:
		return configIntPointerPositiveV0(config.OPESBridge.SpeechSynthesisToolPreflight)
	case envOPESBridgeRemoteQACapabilityV0:
		return configStringPointerHasValueV0(config.OPESBridge.RemoteQA.Capability)
	case envOPESBridgeRemoteQACapabilityRefV0:
		return configStringPointerHasValueV0(config.OPESBridge.RemoteQA.CapabilityRef)
	case envOPESBridgeRemoteQAEvidenceRefsV0:
		return configStringPointerHasValueV0(config.OPESBridge.RemoteQA.EvidenceRefs)
	case envOPESBridgeRemoteQAReasonV0:
		return configStringPointerHasValueV0(config.OPESBridge.RemoteQA.Reason)
	case envOPESBridgeRemoteQANetworkReadyV0:
		return configStringPointerHasValueV0(config.OPESBridge.RemoteQA.NetworkReady)
	case envOPESBridgeRemoteQAAuthStateReadyV0:
		return configStringPointerHasValueV0(config.OPESBridge.RemoteQAAuthStateReady)
	case envOPESBridgeRemoteQAQuotaReadyV0:
		return configStringPointerHasValueV0(config.OPESBridge.RemoteQA.ProviderQuotaReady)
	case envSecurityModeV0:
		return configStringPointerHasValueV0(config.RailsSecurity.SecurityMode)
	case envRailsModeV0:
		return configStringPointerHasValueV0(config.RailsSecurity.RailsMode)
	case envDetailProhibitedRailsV0:
		return configStringPointerHasValueV0(config.RailsSecurity.DetailProhibitedRails)
	case envDetailProhibitedRailsScopeV0:
		return configStringPointerHasValueV0(config.RailsSecurity.DetailProhibitedRailsScope)
	case envEgressSanitizerEnabledV0:
		return config.EgressSanitizer.Enabled != nil
	case envEgressSanitizerRefV0:
		return configStringPointerHasValueV0(config.EgressSanitizer.SanitizerRef)
	case envEgressSanitizerLocalModelEnabledV0:
		return config.EgressSanitizer.LocalModel.Enabled != nil
	case envEgressSanitizerLocalModelRefV0:
		return configStringPointerHasValueV0(config.EgressSanitizer.LocalModel.ModelRef)
	case envEgressSanitizerLocalModelPathV0:
		return configStringPointerHasValueV0(config.EgressSanitizer.LocalModel.ModelPath)
	case envEgressSanitizerLocalRuntimeRefV0:
		return configStringPointerHasValueV0(config.EgressSanitizer.LocalModel.RuntimeRef)
	case envEgressSanitizerLocalEvidenceRefV0:
		return configStringPointerHasValueV0(config.EgressSanitizer.LocalModel.EvidenceRef)
	case envEgressSanitizerSidecarEnabledV0:
		return config.EgressSanitizer.Sidecar.Enabled != nil
	case envEgressSanitizerSidecarRefV0:
		return configStringPointerHasValueV0(config.EgressSanitizer.Sidecar.SidecarRef)
	case envEgressSanitizerSidecarAdapterRefV0:
		return configStringPointerHasValueV0(config.EgressSanitizer.Sidecar.AdapterRef)
	case envEgressSanitizerSidecarTransportRefV0:
		return configStringPointerHasValueV0(config.EgressSanitizer.Sidecar.TransportRef)
	case envEgressSanitizerSidecarEvidenceRefV0:
		return configStringPointerHasValueV0(config.EgressSanitizer.Sidecar.EvidenceRef)
	case envEgressSanitizerSidecarCommandV0:
		return configStringPointerHasValueV0(config.EgressSanitizer.Sidecar.Command)
	case envEgressSanitizerSidecarLocalEndpointV0:
		return configStringPointerHasValueV0(config.EgressSanitizer.Sidecar.LocalEndpoint)
	case envOPESRegistryFinalPkgEnabledV0:
		return config.OPESRegistryFinalPkg.Enabled != nil
	case envOPESRegistryFinalPkgConfirmV0:
		return config.OPESRegistryFinalPkg.Confirm != nil
	case envOPESRegistryFinalPkgDryRunV0:
		return config.OPESRegistryFinalPkg.DryRun != nil
	case envOPESRegistryFinalPkgRegistryV0:
		return configStringPointerHasValueV0(config.OPESRegistryFinalPkg.RegistryPath)
	case envOPESRegistryFinalPkgCourseIDV0:
		return configStringPointerHasValueV0(config.OPESRegistryFinalPkg.CourseID)
	case envOPESRegistryFinalPkgCourseRootV0:
		return configStringPointerHasValueV0(config.OPESRegistryFinalPkg.CourseRoot)
	case envOPESRegistryFinalPkgTemplateRunV0:
		return configStringPointerHasValueV0(config.OPESRegistryFinalPkg.TemplateRunRef)
	case envOPESRegistryFinalPkgTemplateTopicV0:
		return configStringPointerHasValueV0(config.OPESRegistryFinalPkg.TemplateTopicID)
	case envOPESRegistryFinalPkgBatchSizeV0:
		return configIntPointerPositiveV0(config.OPESRegistryFinalPkg.BatchSize)
	case envOPESRegistryFinalPkgMaxInFlightV0:
		return configIntPointerPositiveV0(config.OPESRegistryFinalPkg.MaxInFlight)
	case envOPESRegistryFinalPkgIntervalV0:
		return configIntPointerPositiveV0(config.OPESRegistryFinalPkg.IntervalSeconds)
	case envOPESRegistryFinalPkgMaxTicksV0:
		return config.OPESRegistryFinalPkg.MaxTicks != nil
	case envOPESRegistryFinalPkgQueueRefV0:
		return configStringPointerHasValueV0(config.OPESRegistryFinalPkg.QueueRef)
	case envOPESRegistryFinalPkgReconcileV0:
		return config.OPESRegistryFinalPkg.ReconcileEnabled != nil
	case envOPESRegistryFinalPkgReconcileLimitV0:
		return configIntPointerPositiveV0(config.OPESRegistryFinalPkg.ReconcileLimit)
	case envOPESTopicRegistryEnabledV0:
		return config.OPESTopicRegistry.Enabled != nil
	case envOPESTopicRegistryToolPathV0:
		return configStringPointerHasValueV0(config.OPESTopicRegistry.ToolPath)
	case envOPESTopicRegistryAgentIDV0:
		return configStringPointerHasValueV0(config.OPESTopicRegistry.AgentID)
	case envOPESTopicRegistryForceV0:
		return config.OPESTopicRegistry.Force != nil
	default:
		return false
	}
}

func configStringPointerHasValueV0(value *string) bool {
	return value != nil && strings.TrimSpace(*value) != ""
}

func configIntPointerPositiveV0(value *int) bool {
	return value != nil && *value > 0
}

func configStringSlicePointerHasValueV0(value *[]string) bool {
	if value == nil {
		return false
	}
	for _, item := range *value {
		if strings.TrimSpace(item) != "" {
			return true
		}
	}
	return false
}

func serverProjectConfigAutoprogrammingValueForEnvKeyV0(
	config serverProjectConfigAutoprogrammingV0,
	key string,
) *int {
	switch key {
	case envAutoprogrammingCheckpointOnlyHighConsumptionTokensV0:
		return config.CheckpointOnlyHighConsumptionTokens
	case envAutoprogrammingCheckpointOnlyMaxWaitSecondsV0:
		return config.CheckpointOnlyMaxWaitSeconds
	case envAutoprogrammingNoCheckpointWarningMaxWaitSecondsV0:
		return config.NoCheckpointWarningMaxWaitSeconds
	default:
		return nil
	}
}
