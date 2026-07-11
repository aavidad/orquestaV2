package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestaserver "orquesta/modulos/orquesta-server"
)

const (
	defaultServerSupervisorMaxExternalWaitsV0 = 1
	maxServerSupervisorMaxExternalWaitsV0     = 70
	defaultServerSupervisorMaxDispatchesV0    = 70
	defaultServerSupervisorMaxOutboxV0        = 70
)

func serverConfigFromEnvV0() (orquestaserver.ConfigV0, error) {
	return serverConfigFromEnvWithProjectConfigPathV0("")
}

func serverConfigFromEnvWithProjectConfigPathV0(projectConfigPath string) (orquestaserver.ConfigV0, error) {
	if err := validateServerSelfProgrammingOnlyPathEnvBeforeMkdirV0(); err != nil {
		return orquestaserver.ConfigV0{}, err
	}
	projectDir, err := projectDirFromEnvOrProjectConfigPathV0(projectConfigPath)
	if err != nil {
		return orquestaserver.ConfigV0{}, err
	}
	projectConfig, loadedProjectConfigPath, err := resolveServerProjectConfigV0(projectDir, projectConfigPath)
	if err != nil {
		return orquestaserver.ConfigV0{}, err
	}
	opesConfig := serverOPESConfigSnapshotFromProjectConfigFileV0(projectConfig)
	opesAutomationContext := serverOPESAutomationContextFromSnapshotV0(opesConfig)
	autoprogrammingGoalProgressPolicy := serverAutoprogrammingGoalProgressPolicyConfigFromProjectFileV0(projectConfig)
	idleSelfImprovementProjectDir := serverIdleSelfImprovementProjectWorkDirFromProjectConfigFileV0(projectConfig, projectDir)
	idleSelfImprovementAfterSeconds := serverIdleSelfImprovementAfterSecondsFromProjectConfigFileV0(projectConfig)
	idleSelfImprovementIdleDisabled := idleSelfImprovementAfterSeconds == 0
	idleSelfImprovementDisabled := serverIdleSelfImprovementDisabledFromProjectConfigFileV0(projectConfig, false) ||
		serverIdleSelfImprovementDisabledForOPESContextV0(
			opesAutomationContext,
			projectDir,
			idleSelfImprovementProjectDir,
			opesConfig,
		)
	stateDir := absDirProjectConfigOrEnvOrDefaultV0(envServerStateDirV0,
		projectConfig.Server.StateDir,
		filepath.Join(defaultControlDirV0(projectDir), "state"))
	runtimeDir := absDirProjectConfigOrEnvOrDefaultV0(envCodexRuntimeWorkDirV0,
		projectConfig.CodexRuntime.RuntimeWorkDir,
		filepath.Join(projectDir, ".orquesta-runtime"))
	supervisorMaxExternalWaits, err := serverSupervisorMaxExternalWaitsFromProjectConfigV0(projectConfig)
	if err != nil {
		return orquestaserver.ConfigV0{}, err
	}
	executionMode := codexExecutionModeFromProjectConfigFileV0(projectConfig)
	goalObserverEnabled, goalObserverConfigured := serverGoalObserverEnabledFromEnvV0()
	config := orquestaserver.ConfigV0{
		Addr:          stringProjectConfigOrEnvOrDefaultV0(envServerAddrV0, projectConfig.Server.Addr, orquestaserver.DefaultAddrV0),
		StateDir:      stateDir,
		AuditFile:     stringProjectConfigOrEnvOrDefaultV0(envServerAuditFileV0, projectConfig.Server.AuditFile, orquestaserver.DefaultAuditFileV0),
		AuditDisabled: boolProjectConfigOrEnvOrDefaultV0(envServerAuditDisabledV0, projectConfig.Server.AuditDisabled, false),
		DaemonLogPolicy: orquestaserver.DaemonLogPolicyV0{
			MaxBytes: int64ProjectConfigOrEnvOrDefaultV0(
				envServerDaemonLogMaxBytesV0,
				projectConfig.DaemonLogs.MaxBytes,
				orquestaserver.DefaultDaemonLogMaxBytesV0,
			),
			MaxRotatedFiles: intProjectConfigOrEnvOrDefaultV0(
				envServerDaemonLogMaxRotatedV0,
				projectConfig.DaemonLogs.MaxRotatedFiles,
				orquestaserver.DefaultDaemonLogMaxRotatedV0,
			),
			RetentionDays: intProjectConfigOrEnvOrDefaultV0(
				envServerDaemonLogRetentionDaysV0,
				projectConfig.DaemonLogs.RetentionDays,
				orquestaserver.DefaultDaemonLogRetentionDaysV0,
			),
			LocalRawEnabled: boolProjectConfigOrEnvOrDefaultV0(envServerDaemonLogRawEnabledV0, projectConfig.DaemonLogs.LocalRawEnabled, false),
			LocalRawReason:  stringProjectConfigOrEnvOrDefaultV0(envServerDaemonLogRawReasonV0, projectConfig.DaemonLogs.LocalRawReason, ""),
		},
		HTTPResourceLimits: orquestaserver.HTTPResourceLimitsV0{
			ReadHeaderTimeout: time.Duration(intProjectConfigOrEnvOrDefaultV0(
				envServerReadHeaderTimeoutMSV0,
				projectConfig.ServerHTTP.ReadHeaderTimeoutMS,
				int(orquestaserver.DefaultHTTPReadHeaderTimeoutV0/time.Millisecond),
			)) * time.Millisecond,
			ReadTimeout: time.Duration(intProjectConfigOrEnvOrDefaultV0(
				envServerReadTimeoutMSV0,
				projectConfig.ServerHTTP.ReadTimeoutMS,
				int(orquestaserver.DefaultHTTPReadTimeoutV0/time.Millisecond),
			)) * time.Millisecond,
			WriteTimeout: time.Duration(intProjectConfigOrEnvOrDefaultV0(
				envServerWriteTimeoutMSV0,
				projectConfig.ServerHTTP.WriteTimeoutMS,
				int(orquestaserver.DefaultHTTPWriteTimeoutV0/time.Millisecond),
			)) * time.Millisecond,
			IdleTimeout: time.Duration(intProjectConfigOrEnvOrDefaultV0(
				envServerIdleTimeoutMSV0,
				projectConfig.ServerHTTP.IdleTimeoutMS,
				int(orquestaserver.DefaultHTTPIdleTimeoutV0/time.Millisecond),
			)) * time.Millisecond,
			MaxHeaderBytes: intProjectConfigOrEnvOrDefaultV0(
				envServerMaxHeaderBytesV0,
				projectConfig.ServerHTTP.MaxHeaderBytes,
				orquestaserver.DefaultHTTPMaxHeaderBytesV0,
			),
			ControlBodyBytes: int64(intProjectConfigOrEnvOrDefaultV0(
				envServerControlBodyMaxBytesV0,
				projectConfig.ServerHTTP.ControlBodyMaxBytes,
				int(orquestaserver.DefaultHTTPControlBodyBytesV0),
			)),
		},
		ControlPlane: orquestaserver.ControlPlaneConfigV0{
			RemoteAccessOptIn: boolProjectConfigOrEnvOrDefaultV0(envServerRemoteControlPlaneConfirmV0, projectConfig.ControlPlane.RemoteAccessOptIn, false),
			Token:             stringProjectConfigOrEnvOrDefaultV0(envServerControlTokenV0, projectConfig.ControlPlane.Token, ""),
			Principal:         stringProjectConfigOrEnvOrDefaultV0(envServerControlPrincipalV0, projectConfig.ControlPlane.Principal, "loopback-local"),
			PermissionRef:     stringProjectConfigOrEnvOrDefaultV0(envServerControlPermissionRefV0, projectConfig.ControlPlane.PermissionRef, "permission-ref-loopback-control-plane"),
			PublicReason:      stringProjectConfigOrEnvOrDefaultV0(envServerControlPublicReasonV0, projectConfig.ControlPlane.PublicReason, "loopback_control_plane"),
		},
		RuntimeIdentity:                   serverRuntimeIdentityFromExecutableV0(),
		ProjectWorkDir:                    projectDir,
		ProjectConfigFilePath:             loadedProjectConfigPath,
		RuntimeWorkDir:                    runtimeDir,
		IdleSelfImprovementProjectWorkDir: idleSelfImprovementProjectDir,
		ShutdownSignalPolicy:              serverShutdownSignalPolicyV0(),
		AutoprogrammingGoalProgressPolicy: autoprogrammingGoalProgressPolicy,
		TickInterval:                      time.Duration(intEnvOrDefaultV0(envServerTickIntervalMSV0, 5000)) * time.Millisecond,
		ShutdownGracePeriod: time.Duration(intProjectConfigOrEnvOrDefaultV0(
			envServerShutdownGraceMSV0,
			projectConfig.ServerLifecycle.ShutdownGraceMS,
			int(orquestaserver.DefaultShutdownGracePeriodV0/time.Millisecond),
		)) * time.Millisecond,
		IdleSelfImprovementAfter:         time.Duration(idleSelfImprovementAfterSeconds) * time.Second,
		IdleSelfImprovementDisabled:      idleSelfImprovementDisabled,
		IdleSelfImprovementIdleDisabled:  idleSelfImprovementIdleDisabled,
		IdleSelfImprovementProjectRef:    serverIdleSelfImprovementStringFromProjectConfigFileV0(projectConfig, envServerIdleSelfImprovementProjectRefV0, orquestaserver.DefaultIdleSelfImprovementProjectRefV0),
		IdleSelfImprovementWorktreeRef:   serverIdleSelfImprovementStringFromProjectConfigFileV0(projectConfig, envServerIdleSelfImprovementWorktreeRefV0, orquestaserver.DefaultIdleSelfImprovementWorktreeRefV0),
		IdleSelfImprovementBranchRef:     serverIdleSelfImprovementStringFromProjectConfigFileV0(projectConfig, envServerIdleSelfImprovementBranchRefV0, orquestaserver.DefaultIdleSelfImprovementBranchRefV0),
		IdleSelfImprovementSuggestedArea: serverIdleSelfImprovementStringFromProjectConfigFileV0(projectConfig, envServerIdleSelfImprovementAreaV0, orquestaserver.DefaultIdleSelfImprovementSuggestedAreaV0),
		IdleSelfImprovementWriteSet:      serverIdleSelfImprovementStringSliceFromProjectConfigFileV0(projectConfig, envServerIdleSelfImprovementWriteSetV0, defaultIdleSelfImprovementWriteSetV0()),
		IdleSelfImprovementRequiredTests: serverIdleSelfImprovementStringSliceFromProjectConfigFileV0(projectConfig, envServerIdleSelfImprovementRequiredTestsV0, []string{orquestaserver.DefaultIdleSelfImprovementRequiredTestV0}),
		IdleSelfImprovementContextRefs:   serverIdleSelfImprovementStringSliceFromProjectConfigFileV0(projectConfig, envServerIdleSelfImprovementContextRefsV0, nil),
		IdleSelfImprovementEvidenceRefs:  serverIdleSelfImprovementStringSliceFromProjectConfigFileV0(projectConfig, envServerIdleSelfImprovementEvidenceRefsV0, nil),
		IdleSelfImprovementAcceptance:    serverIdleSelfImprovementStringSliceFromProjectConfigFileV0(projectConfig, envServerIdleSelfImprovementAcceptanceV0, defaultIdleSelfImprovementAcceptanceV0()),
		IdleSelfImprovementGoalFirst:     serverIdleSelfImprovementGoalFirstFromProjectConfigFileV0(projectConfig),
		IdleSelfImprovementFrozenTests:   serverIdleSelfImprovementFrozenTestsFromProjectConfigFileV0(projectConfig),
		IdleSelfImprovementCompactRules: serverIdleSelfImprovementStringSliceFromProjectConfigFileV0(projectConfig, envServerIdleSelfImprovementCompactRulesV0, []string{
			"comunicacion compacta",
			"trabajo secundario: no bloquear ni mezclar con el trabajo principal",
		}),
		IdleSelfImprovementPriorityScore: serverIdleSelfImprovementIntFromProjectConfigFileV0(projectConfig, envServerIdleSelfImprovementPriorityScoreV0, orquestaserver.DefaultIdleSelfImprovementPriorityScoreV0),
		IdleSelfImprovementMaxRequests:   serverIdleSelfImprovementMaxRequestsFromProjectConfigFileV0(projectConfig),
		IdleSelfImprovementTargetQueue:   serverIdleSelfImprovementIntFromProjectConfigFileV0(projectConfig, envServerIdleSelfImprovementTargetQueueV0, orquestaserver.DefaultIdleSelfImprovementTargetQueueV0),
		IdleSelfImprovementBudget: orquestaautoprogramming.AutoprogrammingIdleSelfImprovementBudgetConfigV0{
			MaxGoalsPerDay:              serverIdleSelfImprovementIntFromProjectConfigFileV0(projectConfig, envServerIdleSelfImprovementDailyGoalBudgetV0, 0),
			MaxContextBudgetBytesPerDay: int64(serverIdleSelfImprovementIntFromProjectConfigFileV0(projectConfig, envServerIdleSelfImprovementDailyContextBudgetBytesV0, 0)),
		},
		SelfAuditBacklogEnabled:       boolEnvOrDefaultV0(envSelfAuditBacklogEnabledV0, false),
		GoalObserverEnabled:           goalObserverEnabled,
		GoalObserverEnabledConfigured: goalObserverConfigured,
		GoalObserverInterval:          time.Duration(intEnvOrZeroV0(envServerGoalObserverIntervalMSV0)) * time.Millisecond,
		GoalObserverMaxItems:          intEnvOrDefaultV0(envServerGoalObserverMaxItemsV0, orquestaserver.DefaultGoalObserverMaxItemsV0),
		ResidentDirectorEnabled:       serverResidentDirectorEnabledFromEnvV0(),
		ResidentDirectorMaxActions:    serverResidentDirectorMaxActionsFromProjectConfigFileV0(projectConfig),
		EscalationDirectorEnabled:     boolEnvOrDefaultV0(envServerEscalationDirectorEnabledV0, false),
		EscalationDirectorCommand:     csvEnvOrDefaultV0(envServerEscalationDirectorCommandV0, nil),
		EscalationDirectorTimeout:     time.Duration(intEnvOrDefaultV0(envServerEscalationDirectorTimeoutSecondsV0, 0)) * time.Second,
		EscalationDirectorMaxPerDay:   intEnvOrDefaultV0(envServerEscalationDirectorMaxPerDayV0, 0),
		SelfWatchdog: orquestaserver.SelfWatchdogConfigV0{
			Disabled:       boolEnvOrDefaultV0(envServerSelfWatchdogDisabledV0, false),
			CPUHighPercent: intEnvOrDefaultV0(envServerSelfWatchdogCPUHighPercentV0, orquestaserver.DefaultSelfWatchdogCPUHighPercentV0),
			SustainedFor: time.Duration(intEnvOrDefaultV0(
				envServerSelfWatchdogSustainedSecondsV0,
				int(orquestaserver.DefaultSelfWatchdogSustainedForV0/time.Second),
			)) * time.Second,
			NoProgressFor: time.Duration(intEnvOrDefaultV0(
				envServerSelfWatchdogNoProgressSecondsV0,
				int(orquestaserver.DefaultSelfWatchdogNoProgressForV0/time.Second),
			)) * time.Second,
		},
		SupervisorCommand: serverSupervisorCommandFromProjectConfigV0(executionMode, projectConfig, supervisorMaxExternalWaits),
	}
	config = orquestaserver.NormalizeConfigV0(config)
	if err := validateServerSelfProgrammingOnlyConfigV0(config); err != nil {
		return orquestaserver.ConfigV0{}, err
	}
	// The effective projection is only published after runtime-model inputs pass
	// the same strict resolver used by the composition root.
	if _, err := ollamaModelManagerConfigFromProjectConfigV0(projectConfig); err != nil {
		return orquestaserver.ConfigV0{}, err
	}
	if _, err := hermesOperatorConfigFromProjectConfigV0(projectConfig); err != nil {
		return orquestaserver.ConfigV0{}, err
	}
	config.EffectiveConfig = serverEffectiveConfigFromEnvAndProjectConfigV0(config, projectConfig)
	return config, orquestaserver.ValidateConfigV0(config)
}

func defaultIdleSelfImprovementWriteSetV0() []string {
	return []string{
		"modulos/orquesta-server",
		"modulos/orquesta-server/config_v0.go",
		"modulos/orquesta-server/ports_v0.go",
		"modulos/orquesta-server/supervisor_loop_v0.go",
		"modulos/orquesta-server/status_tracker_v0.go",
		"modulos/orquesta-server/supervisor_loop_v0_test.go",
		"cmd/orquesta-server/config.go",
		"cmd/orquesta-server/config_test.go",
		"cmd/orquesta-server/stack.go",
	}
}
func defaultIdleSelfImprovementAcceptanceV0() []string {
	return []string{
		envServerIdleSelfImprovementAfterV0 + " por defecto dispara tras 60 segundos sin ejecuciones",
		envServerIdleSelfImprovementAfterV0 + "=0 desactiva el disparador por reloj idle",
		"el servidor prepara automejora cuando hay idle o capacidad libre por debajo de " + envServerIdleSelfImprovementTargetQueueV0,
		"el planner salta tareas ya visibles en cola y puede crear una tarea scanner para descubrir nuevos huecos",
		"las secciones narrativas del backlog se filtran y no se convierten en runs de automejora",
		"la proyeccion publica distingue outbox pendiente, wait_external y proceso externo verificado",
		"usar evidencia del fallo y corregir la causa general si es posible",
	}
}

func defaultControlDirV0(projectDir string) string {
	cleanProject := filepath.Clean(projectDir)
	base := filepath.Base(cleanProject)
	if base == "" || base == "." || base == string(os.PathSeparator) {
		base = "project"
	}
	return filepath.Join(filepath.Dir(cleanProject), ".orquesta-control", base)
}

func projectDirFromEnvV0() (string, error) {
	return projectDirFromEnvOrFallbackV0("")
}

func projectDirFromEnvOrProjectConfigPathV0(projectConfigPath string) (string, error) {
	projectConfigPath = strings.TrimSpace(projectConfigPath)
	if projectConfigPath == "" {
		return projectDirFromEnvV0()
	}
	abs, err := filepath.Abs(projectConfigPath)
	if err != nil {
		return "", fmt.Errorf("project_config_path_invalid")
	}
	return projectDirFromEnvOrFallbackV0(filepath.Dir(abs))
}

func projectDirFromEnvOrFallbackV0(fallback string) (string, error) {
	value := strings.TrimSpace(os.Getenv(envCodexProjectWorkDirV0))
	if value == "" {
		value = strings.TrimSpace(fallback)
	}
	if value == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("project_work_dir_unavailable")
		}
		value = cwd
	}
	abs, err := filepath.Abs(value)
	if err != nil {
		return "", fmt.Errorf("project_work_dir_invalid")
	}
	return abs, nil
}

func serverIdleSelfImprovementDisabledForOPESContextV0(
	opesAutomationContext bool,
	projectDir string,
	idleSelfImprovementProjectDir string,
	opesConfig serverOPESConfigSnapshotV0,
) bool {
	if !opesAutomationContext {
		return false
	}
	if strings.TrimSpace(os.Getenv(envServerIdleSelfImprovementProjectWorkDirV0)) == "" {
		return true
	}
	if sameAbsDirForConfigV0(idleSelfImprovementProjectDir, projectDir) {
		return true
	}
	return opesConfig.HasProjectWorkDir() &&
		sameAbsDirForConfigV0(idleSelfImprovementProjectDir, opesConfig.ProjectWorkDir)
}

func sameAbsDirForConfigV0(left string, right string) bool {
	leftAbs, err := filepath.Abs(strings.TrimSpace(left))
	if err != nil {
		return false
	}
	rightAbs, err := filepath.Abs(strings.TrimSpace(right))
	if err != nil {
		return false
	}
	return filepath.Clean(leftAbs) == filepath.Clean(rightAbs)
}

func absDirEnvOrDefaultV0(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		value = fallback
	}
	abs, err := filepath.Abs(value)
	if err != nil {
		return fallback
	}
	_ = os.MkdirAll(abs, 0o700)
	return abs
}

func envOrDefaultV0(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func intEnvOrDefaultV0(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func idleSelfImprovementAfterSecondsFromEnvV0() int {
	return intEnvOrDefaultAllowZeroFromKeysV0(
		int(orquestaserver.DefaultIdleSelfImprovementAfterV0/time.Second),
		envServerIdleSelfImprovementAfterV0,
		envServerIdleSelfImprovementAfterLegacyV0,
	)
}

func intEnvOrDefaultAllowZeroFromKeysV0(fallback int, keys ...string) int {
	for _, key := range keys {
		value := strings.TrimSpace(os.Getenv(key))
		if value == "" {
			continue
		}
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 0 {
			return fallback
		}
		return parsed
	}
	return fallback
}

func csvEnvOrDefaultV0(key string, fallback []string) []string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return append([]string(nil), fallback...)
	}
	var out []string
	for _, item := range strings.Split(value, ",") {
		item = strings.TrimSpace(item)
		if item != "" {
			out = append(out, item)
		}
	}
	return out
}

func serverStackIdleSelfImprovementBacklogRequestAllowedV0(
	request orquestaserver.IdleSelfImprovementRequestV0,
) bool {
	if !serverStackIdleSelfImprovementBacklogRefAllowedV0(request.RequestRef) {
		return false
	}
	for _, contextRef := range request.ContextRefs {
		sectionRef := serverStackIdleSelfImprovementBacklogSectionContextV0(contextRef)
		if sectionRef != "" && !serverStackIdleSelfImprovementBacklogSectionAllowedV0(sectionRef) {
			return false
		}
	}
	if strings.EqualFold(strings.TrimSpace(request.SuggestedArea), "backlog-scan") {
		return true
	}
	return true
}

func serverStackIdleSelfImprovementBacklogRefAllowedV0(requestRef string) bool {
	const prefix = "request-ref-autoprogramming-backlog-"
	requestRef = strings.TrimSpace(requestRef)
	if !strings.HasPrefix(requestRef, prefix) {
		return true
	}
	suffix := strings.TrimPrefix(requestRef, prefix)
	if strings.HasPrefix(suffix, "scanner-") {
		return true
	}
	return serverStackIdleSelfImprovementBacklogSectionAllowedV0(suffix)
}

func serverStackIdleSelfImprovementBacklogSectionContextV0(contextRef string) string {
	for _, prefix := range []string{"backlog_section:", "backlog_local_alias:"} {
		if value := strings.TrimSpace(strings.TrimPrefix(contextRef, prefix)); value != strings.TrimSpace(contextRef) {
			return value
		}
	}
	return ""
}

func serverStackIdleSelfImprovementBacklogSectionAllowedV0(suffix string) bool {
	suffix = strings.ToLower(strings.TrimSpace(suffix))
	for _, prefix := range []string{"t", "apg-", "srv-task-", "mcp-"} {
		rest := strings.TrimPrefix(suffix, prefix)
		if rest == suffix || rest == "" || rest[0] < '0' || rest[0] > '9' {
			continue
		}
		for index := 1; index < len(rest); index++ {
			if rest[index] < '0' || rest[index] > '9' {
				return rest[index] == '-'
			}
		}
		return true
	}
	return false
}
