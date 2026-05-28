package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	orquestaserver "orquesta/modulos/orquesta-server"
)

const (
	defaultServerSupervisorMaxExternalWaitsV0 = 1
	maxServerSupervisorMaxExternalWaitsV0     = 70
	defaultServerSupervisorMaxDispatchesV0    = 10
	defaultServerSupervisorMaxOutboxV0        = 10
)

func serverConfigFromEnvV0() (orquestaserver.ConfigV0, error) {
	projectDir, err := projectDirFromEnvV0()
	if err != nil {
		return orquestaserver.ConfigV0{}, err
	}
	stateDir := absDirEnvOrDefaultV0(envServerStateDirV0,
		filepath.Join(defaultControlDirV0(projectDir), "state"))
	runtimeDir := absDirEnvOrDefaultV0(envCodexRuntimeWorkDirV0,
		filepath.Join(projectDir, ".orquesta-runtime"))
	supervisorMaxExternalWaits, err := serverSupervisorMaxExternalWaitsV0()
	if err != nil {
		return orquestaserver.ConfigV0{}, err
	}
	executionMode := codexExecutionModeFromEnvV0()
	config := orquestaserver.ConfigV0{
		Addr:          envOrDefaultV0(envServerAddrV0, orquestaserver.DefaultAddrV0),
		StateDir:      stateDir,
		AuditFile:     envOrDefaultV0(envServerAuditFileV0, orquestaserver.DefaultAuditFileV0),
		AuditDisabled: boolEnvOrDefaultV0(envServerAuditDisabledV0, false),
		DaemonLogPolicy: orquestaserver.DaemonLogPolicyV0{
			MaxBytes:        int64(intEnvOrDefaultV0(envServerDaemonLogMaxBytesV0, int(orquestaserver.DefaultDaemonLogMaxBytesV0))),
			MaxRotatedFiles: intEnvOrDefaultV0(envServerDaemonLogMaxRotatedV0, orquestaserver.DefaultDaemonLogMaxRotatedV0),
			RetentionDays:   intEnvOrDefaultV0(envServerDaemonLogRetentionDaysV0, orquestaserver.DefaultDaemonLogRetentionDaysV0),
			LocalRawEnabled: boolEnvOrDefaultV0(envServerDaemonLogRawEnabledV0, false),
			LocalRawReason:  envOrDefaultV0(envServerDaemonLogRawReasonV0, ""),
		},
		HTTPResourceLimits: orquestaserver.HTTPResourceLimitsV0{
			ReadHeaderTimeout: time.Duration(intEnvOrDefaultV0(envServerReadHeaderTimeoutMSV0, int(orquestaserver.DefaultHTTPReadHeaderTimeoutV0/time.Millisecond))) * time.Millisecond,
			ReadTimeout:       time.Duration(intEnvOrDefaultV0(envServerReadTimeoutMSV0, int(orquestaserver.DefaultHTTPReadTimeoutV0/time.Millisecond))) * time.Millisecond,
			WriteTimeout:      time.Duration(intEnvOrDefaultV0(envServerWriteTimeoutMSV0, int(orquestaserver.DefaultHTTPWriteTimeoutV0/time.Millisecond))) * time.Millisecond,
			IdleTimeout:       time.Duration(intEnvOrDefaultV0(envServerIdleTimeoutMSV0, int(orquestaserver.DefaultHTTPIdleTimeoutV0/time.Millisecond))) * time.Millisecond,
			MaxHeaderBytes:    intEnvOrDefaultV0(envServerMaxHeaderBytesV0, orquestaserver.DefaultHTTPMaxHeaderBytesV0),
			ControlBodyBytes:  int64(intEnvOrDefaultV0(envServerControlBodyMaxBytesV0, int(orquestaserver.DefaultHTTPControlBodyBytesV0))),
		},
		ControlPlane: orquestaserver.ControlPlaneConfigV0{
			RemoteAccessOptIn: boolEnvOrDefaultV0(envServerRemoteControlPlaneConfirmV0, false),
			Token:             os.Getenv(envServerControlTokenV0),
			Principal:         envOrDefaultV0(envServerControlPrincipalV0, "loopback-local"),
			PermissionRef:     envOrDefaultV0(envServerControlPermissionRefV0, "permission-ref-loopback-control-plane"),
			PublicReason:      envOrDefaultV0(envServerControlPublicReasonV0, "loopback_control_plane"),
		},
		ProjectWorkDir:       projectDir,
		RuntimeWorkDir:       runtimeDir,
		ShutdownSignalPolicy: serverShutdownSignalPolicyV0(),
		TickInterval:         time.Duration(intEnvOrDefaultV0(envServerTickIntervalMSV0, 5000)) * time.Millisecond,
		ShutdownGracePeriod: time.Duration(intEnvOrDefaultV0(
			envServerShutdownGraceMSV0,
			int(orquestaserver.DefaultShutdownGracePeriodV0/time.Millisecond),
		)) * time.Millisecond,
		IdleSelfImprovementAfter: time.Duration(intEnvOrDefaultAllowZeroV0(
			envServerIdleSelfImprovementAfterV0,
			int(orquestaserver.DefaultIdleSelfImprovementAfterV0/time.Second),
		)) * time.Second,
		IdleSelfImprovementDisabled:      strings.TrimSpace(os.Getenv(envServerIdleSelfImprovementAfterV0)) == "0",
		IdleSelfImprovementProjectRef:    envOrDefaultV0(envServerIdleSelfImprovementProjectRefV0, orquestaserver.DefaultIdleSelfImprovementProjectRefV0),
		IdleSelfImprovementWorktreeRef:   envOrDefaultV0(envServerIdleSelfImprovementWorktreeRefV0, orquestaserver.DefaultIdleSelfImprovementWorktreeRefV0),
		IdleSelfImprovementBranchRef:     envOrDefaultV0(envServerIdleSelfImprovementBranchRefV0, orquestaserver.DefaultIdleSelfImprovementBranchRefV0),
		IdleSelfImprovementSuggestedArea: envOrDefaultV0(envServerIdleSelfImprovementAreaV0, orquestaserver.DefaultIdleSelfImprovementSuggestedAreaV0),
		IdleSelfImprovementWriteSet:      csvEnvOrDefaultV0(envServerIdleSelfImprovementWriteSetV0, defaultIdleSelfImprovementWriteSetV0()),
		IdleSelfImprovementRequiredTests: csvEnvOrDefaultV0(envServerIdleSelfImprovementRequiredTestsV0, []string{orquestaserver.DefaultIdleSelfImprovementRequiredTestV0}),
		IdleSelfImprovementContextRefs:   csvEnvOrDefaultV0(envServerIdleSelfImprovementContextRefsV0, nil),
		IdleSelfImprovementEvidenceRefs:  csvEnvOrDefaultV0(envServerIdleSelfImprovementEvidenceRefsV0, nil),
		IdleSelfImprovementAcceptance:    csvEnvOrDefaultV0(envServerIdleSelfImprovementAcceptanceV0, defaultIdleSelfImprovementAcceptanceV0()),
		IdleSelfImprovementCompactRules:  csvEnvOrDefaultV0(envServerIdleSelfImprovementCompactRulesV0, defaultIdleSelfImprovementCompactRulesV0()),
		IdleSelfImprovementPriorityScore: intEnvOrDefaultV0(envServerIdleSelfImprovementPriorityScoreV0, orquestaserver.DefaultIdleSelfImprovementPriorityScoreV0),
		IdleSelfImprovementMaxRequests:   intEnvOrDefaultV0(envServerIdleSelfImprovementMaxRequestsV0, orquestaserver.DefaultIdleSelfImprovementMaxRequestsV0),
		IdleSelfImprovementTargetQueue:   intEnvOrDefaultV0(envServerIdleSelfImprovementTargetQueueV0, orquestaserver.DefaultIdleSelfImprovementTargetQueueV0),
		SupervisorCommand:                serverSupervisorCommandFromEnvV0(executionMode, supervisorMaxExternalWaits),
	}
	config.EffectiveConfig = serverEffectiveConfigFromEnvV0(config)
	return orquestaserver.NormalizeConfigV0(config), orquestaserver.ValidateConfigV0(config)
}

func startupCleanupModeEffectiveValueV0() string {
	return envOrDefaultV0(envStartupCleanupModeV0, startupCleanupModeDiagnoseV0)
}
func defaultIdleSelfImprovementWriteSetV0() []string {
	return []string{
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
		envServerIdleSelfImprovementAfterV0 + "=0 desactiva automejora idle",
		"el servidor prepara automejora cuando hay idle o capacidad libre por debajo de " + envServerIdleSelfImprovementTargetQueueV0,
		"el planner salta tareas ya visibles en cola y puede crear una tarea scanner para descubrir nuevos huecos",
		"las secciones narrativas del backlog se filtran y no se convierten en runs de automejora",
		"usar evidencia del fallo y corregir la causa general si es posible",
	}
}
func defaultIdleSelfImprovementCompactRulesV0() []string {
	return []string{
		"comunicacion compacta",
		"trabajo secundario: no bloquear ni mezclar con el trabajo principal",
	}
}
func serverSupervisorMaxExternalWaitsV0() (int, error) {
	raw := strings.TrimSpace(os.Getenv(envServerDrainMaxExternalWaitsV0))
	if raw == "" {
		return defaultServerSupervisorMaxExternalWaitsV0, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("%s incompatible: usar entero positivo hasta %d", envServerDrainMaxExternalWaitsV0, maxServerSupervisorMaxExternalWaitsV0)
	}
	if value > maxServerSupervisorMaxExternalWaitsV0 {
		return 0, fmt.Errorf("%s incompatible con supervisor residente: max=%d", envServerDrainMaxExternalWaitsV0, maxServerSupervisorMaxExternalWaitsV0)
	}
	return value, nil
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
	value := strings.TrimSpace(os.Getenv(envCodexProjectWorkDirV0))
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
	if err := os.MkdirAll(abs, 0o700); err != nil {
		return "", fmt.Errorf("project_work_dir_unavailable")
	}
	return abs, nil
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

func intEnvOrDefaultAllowZeroV0(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 0 {
		return fallback
	}
	return parsed
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
	if strings.EqualFold(strings.TrimSpace(request.SuggestedArea), "backlog-scan") {
		return true
	}
	for _, contextRef := range request.ContextRefs {
		sectionRef := serverStackIdleSelfImprovementBacklogSectionContextV0(contextRef)
		if sectionRef != "" && !serverStackIdleSelfImprovementBacklogSectionAllowedV0(sectionRef) {
			return false
		}
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
