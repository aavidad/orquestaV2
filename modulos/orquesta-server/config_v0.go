package orquestaserver

import (
	"fmt"
	"net"
	"path/filepath"
	"strings"
	"time"

	orquestarunsupervisor "orquesta/modulos/orquesta-run-supervisor"
)

const (
	DefaultAddrV0                             = "127.0.0.1:8787"
	DefaultStateFileV0                        = "orquesta_server_state_v0.json"
	DefaultAuditFileV0                        = "orquesta_server_audit_v0.jsonl"
	DefaultTickIntervalV0                     = 5 * time.Second
	DefaultGoalObserverIntervalV0             = DefaultTickIntervalV0
	DefaultShutdownGracePeriodV0              = 10 * time.Second
	DefaultSupervisorMaxTicksV0               = 1
	DefaultIdleSelfImprovementAfterV0         = 60 * time.Second
	DefaultIdleSelfImprovementProjectRefV0    = "project-ref-orquesta-server"
	DefaultIdleSelfImprovementWorktreeRefV0   = "worktree-ref-orquesta-server-idle-self-improvement"
	DefaultIdleSelfImprovementBranchRefV0     = "branch-ref-orquesta-server-idle-self-improvement"
	DefaultIdleSelfImprovementRequiredTestV0  = "go test -count=1 ./..."
	DefaultIdleSelfImprovementSuggestedAreaV0 = "automejora"
	DefaultIdleSelfImprovementSkillRefAutoV0  = "skill-ref-orquesta-programacion-autonoma-v0"
	DefaultIdleSelfImprovementSkillRefIntV0   = "skill-ref-orquesta-programacion-integracion-v0"
	DefaultIdleSelfImprovementPriorityScoreV0 = 10
	DefaultIdleSelfImprovementMaxRequestsV0   = 10
	DefaultIdleSelfImprovementTargetQueueV0   = 10
	DefaultGoalObserverMaxItemsV0             = 70
	DefaultResidentDirectorMaxActionsV0       = 20
	DefaultSelfWatchdogCPUHighPercentV0       = 75
	DefaultSelfWatchdogSustainedForV0         = 2 * time.Minute
	DefaultSelfWatchdogNoProgressForV0        = 1 * time.Minute
	SupervisorPublicStatusOKV0                = "ok"
	SupervisorPublicStatusWaitingOutboxV0     = "waiting_outbox"
	SupervisorPublicStatusWaitingExternalV0   = "waiting_external"
	SupervisorPublicStatusRunningLiveV0       = "running_live"
	SupervisorPublicStatusStalledV0           = "stalled"
	SupervisorPublicStatusLaunchFailedV0      = "launch_failed"
	SupervisorPublicStatusExternalEmptyRunV0  = "external_work_empty_run"
	SupervisorPublicStopWaitingOutboxV0       = "waiting_outbox"
	SupervisorPublicStopWaitingExternalV0     = "wait_external"
	SupervisorPublicStopRunningLiveV0         = "running_live"
	SupervisorPublicStopStalledV0             = "external_process_unverified"
	SupervisorPublicStopLaunchFailedV0        = "launch_failed"
	SupervisorPublicStopExternalEmptyRunV0    = "failed_empty_run"
	SupervisorPublicCategoryWaitOutboxV0      = "wait_outbox"
	SupervisorPublicCategoryWaitExternalV0    = "wait_external"
	SupervisorPublicCategoryExternalProcessV0 = "external_process"
)

type ConfigV0 struct {
	Addr                              string
	StateDir                          string
	StateFile                         string
	AuditFile                         string
	AuditDisabled                     bool
	DaemonLogPolicy                   DaemonLogPolicyV0
	HTTPResourceLimits                HTTPResourceLimitsV0
	ControlPlane                      ControlPlaneConfigV0
	EffectiveConfig                   ServerEffectiveConfigV0
	ProjectWorkDir                    string
	RuntimeWorkDir                    string
	IdleSelfImprovementProjectWorkDir string
	ShutdownSignalPolicy              ShutdownSignalPolicyV0
	TickInterval                      time.Duration
	ShutdownGracePeriod               time.Duration
	SupervisorCommand                 orquestarunsupervisor.RunSupervisorCommandV0
	IdleSelfImprovementDisabled       bool
	IdleSelfImprovementAfter          time.Duration
	IdleSelfImprovementProjectRef     string
	IdleSelfImprovementWorktreeRef    string
	IdleSelfImprovementBranchRef      string
	IdleSelfImprovementSuggestedArea  string
	IdleSelfImprovementWriteSet       []string
	IdleSelfImprovementRequiredTests  []string
	IdleSelfImprovementContextRefs    []string
	IdleSelfImprovementEvidenceRefs   []string
	IdleSelfImprovementAcceptance     []string
	IdleSelfImprovementCompactRules   []string
	IdleSelfImprovementSkillRefs      []string
	IdleSelfImprovementGoalFirst      bool
	IdleSelfImprovementPriorityScore  int
	IdleSelfImprovementMaxRequests    int
	IdleSelfImprovementTargetQueue    int
	SelfAuditBacklogEnabled           bool
	GoalObserverEnabled               bool
	GoalObserverEnabledConfigured     bool
	GoalObserverInterval              time.Duration
	GoalObserverMaxItems              int
	ResidentDirectorEnabled           bool
	ResidentDirectorMaxActions        int
	SelfWatchdog                      SelfWatchdogConfigV0
}

type ControlPlaneConfigV0 struct {
	RemoteAccessOptIn bool
	Token             string
	Principal         string
	PermissionRef     string
	PublicReason      string
}

func NormalizeConfigV0(config ConfigV0) ConfigV0 {
	config.Addr = strings.TrimSpace(config.Addr)
	if config.Addr == "" {
		config.Addr = DefaultAddrV0
	}
	config.StateDir = strings.TrimSpace(config.StateDir)
	config.StateFile = strings.TrimSpace(config.StateFile)
	if config.StateFile == "" {
		config.StateFile = DefaultStateFileV0
	}
	config.AuditFile = strings.TrimSpace(config.AuditFile)
	if config.AuditFile == "" {
		config.AuditFile = DefaultAuditFileV0
	}
	config.DaemonLogPolicy = NormalizeDaemonLogPolicyV0(config.DaemonLogPolicy)
	config.HTTPResourceLimits = NormalizeHTTPResourceLimitsV0(config.HTTPResourceLimits)
	config.ControlPlane.Token = strings.TrimSpace(config.ControlPlane.Token)
	config.ControlPlane.Principal = strings.TrimSpace(config.ControlPlane.Principal)
	config.ControlPlane.PermissionRef = strings.TrimSpace(config.ControlPlane.PermissionRef)
	config.ControlPlane.PublicReason = strings.TrimSpace(config.ControlPlane.PublicReason)
	if config.ControlPlane.Principal == "" {
		config.ControlPlane.Principal = "loopback-local"
	}
	if config.ControlPlane.PermissionRef == "" {
		config.ControlPlane.PermissionRef = "permission-ref-loopback-control-plane"
	}
	if config.ControlPlane.PublicReason == "" {
		config.ControlPlane.PublicReason = "loopback_control_plane"
	}
	config.EffectiveConfig = NormalizeServerEffectiveConfigV0(config.EffectiveConfig)
	config.ProjectWorkDir = strings.TrimSpace(config.ProjectWorkDir)
	config.RuntimeWorkDir = strings.TrimSpace(config.RuntimeWorkDir)
	config.IdleSelfImprovementProjectWorkDir = strings.TrimSpace(config.IdleSelfImprovementProjectWorkDir)
	if config.IdleSelfImprovementProjectWorkDir == "" {
		config.IdleSelfImprovementProjectWorkDir = config.ProjectWorkDir
	}
	config.ShutdownSignalPolicy = NormalizeShutdownSignalPolicyV0(config.ShutdownSignalPolicy)
	if config.TickInterval <= 0 {
		config.TickInterval = DefaultTickIntervalV0
	}
	if config.GoalObserverInterval <= 0 {
		config.GoalObserverInterval = config.TickInterval
	}
	if config.GoalObserverInterval <= 0 {
		config.GoalObserverInterval = DefaultGoalObserverIntervalV0
	}
	if config.ShutdownGracePeriod <= 0 {
		config.ShutdownGracePeriod = DefaultShutdownGracePeriodV0
	}
	if config.SupervisorCommand.MaxTicks <= 0 {
		config.SupervisorCommand.MaxTicks = DefaultSupervisorMaxTicksV0
	}
	if config.IdleSelfImprovementDisabled {
		config.IdleSelfImprovementAfter = 0
	}
	if !config.IdleSelfImprovementDisabled && config.IdleSelfImprovementAfter <= 0 {
		config.IdleSelfImprovementAfter = DefaultIdleSelfImprovementAfterV0
	}
	config.IdleSelfImprovementProjectRef = strings.TrimSpace(config.IdleSelfImprovementProjectRef)
	if config.IdleSelfImprovementProjectRef == "" {
		config.IdleSelfImprovementProjectRef = DefaultIdleSelfImprovementProjectRefV0
	}
	config.IdleSelfImprovementWorktreeRef = strings.TrimSpace(config.IdleSelfImprovementWorktreeRef)
	if config.IdleSelfImprovementWorktreeRef == "" {
		config.IdleSelfImprovementWorktreeRef = DefaultIdleSelfImprovementWorktreeRefV0
	}
	config.IdleSelfImprovementBranchRef = strings.TrimSpace(config.IdleSelfImprovementBranchRef)
	if config.IdleSelfImprovementBranchRef == "" {
		config.IdleSelfImprovementBranchRef = DefaultIdleSelfImprovementBranchRefV0
	}
	config.IdleSelfImprovementSuggestedArea = strings.TrimSpace(config.IdleSelfImprovementSuggestedArea)
	if config.IdleSelfImprovementSuggestedArea == "" {
		config.IdleSelfImprovementSuggestedArea = DefaultIdleSelfImprovementSuggestedAreaV0
	}
	config.IdleSelfImprovementWriteSet = compactConfigStringsV0(config.IdleSelfImprovementWriteSet)
	config.IdleSelfImprovementRequiredTests = compactConfigStringsV0(config.IdleSelfImprovementRequiredTests)
	if len(config.IdleSelfImprovementRequiredTests) == 0 {
		config.IdleSelfImprovementRequiredTests = []string{DefaultIdleSelfImprovementRequiredTestV0}
	}
	config.IdleSelfImprovementContextRefs = compactConfigStringsV0(config.IdleSelfImprovementContextRefs)
	config.IdleSelfImprovementEvidenceRefs = compactConfigStringsV0(config.IdleSelfImprovementEvidenceRefs)
	config.IdleSelfImprovementAcceptance = compactConfigStringsV0(config.IdleSelfImprovementAcceptance)
	config.IdleSelfImprovementCompactRules = compactConfigStringsV0(config.IdleSelfImprovementCompactRules)
	config.IdleSelfImprovementSkillRefs = compactConfigStringsV0(config.IdleSelfImprovementSkillRefs)
	if len(config.IdleSelfImprovementSkillRefs) == 0 {
		config.IdleSelfImprovementSkillRefs = []string{
			DefaultIdleSelfImprovementSkillRefAutoV0,
			DefaultIdleSelfImprovementSkillRefIntV0,
		}
	}
	if config.IdleSelfImprovementPriorityScore <= 0 {
		config.IdleSelfImprovementPriorityScore = DefaultIdleSelfImprovementPriorityScoreV0
	}
	if config.IdleSelfImprovementMaxRequests <= 0 {
		config.IdleSelfImprovementMaxRequests = DefaultIdleSelfImprovementMaxRequestsV0
	}
	if config.IdleSelfImprovementTargetQueue <= 0 {
		config.IdleSelfImprovementTargetQueue = DefaultIdleSelfImprovementTargetQueueV0
	}
	if config.IdleSelfImprovementTargetQueue < config.IdleSelfImprovementMaxRequests {
		config.IdleSelfImprovementTargetQueue = config.IdleSelfImprovementMaxRequests
	}
	if !config.GoalObserverEnabledConfigured {
		config.GoalObserverEnabled = true
	}
	if config.GoalObserverMaxItems <= 0 {
		config.GoalObserverMaxItems = DefaultGoalObserverMaxItemsV0
	}
	if config.ResidentDirectorMaxActions <= 0 {
		config.ResidentDirectorMaxActions = DefaultResidentDirectorMaxActionsV0
	}
	config.SelfWatchdog = NormalizeSelfWatchdogConfigV0(config.SelfWatchdog)
	return config
}

func ValidateConfigV0(config ConfigV0) error {
	config = NormalizeConfigV0(config)
	if strings.TrimSpace(config.StateDir) == "" || !filepath.IsAbs(config.StateDir) {
		return fmt.Errorf("orquesta_server: state_dir invalido")
	}
	if filepath.Base(config.StateFile) != config.StateFile {
		return fmt.Errorf("orquesta_server: state_file invalido")
	}
	if filepath.Base(config.AuditFile) != config.AuditFile || filepath.Ext(config.AuditFile) != ".jsonl" {
		return fmt.Errorf("orquesta_server: audit_file invalido")
	}
	if !controlPlaneAddrIsLoopbackV0(config.Addr) {
		if !config.ControlPlane.RemoteAccessOptIn {
			return fmt.Errorf("orquesta_server: control_plane_remote_opt_in_required")
		}
		if !controlPlaneRemoteTokenStrongEnoughV0(config.ControlPlane.Token) {
			return fmt.Errorf("orquesta_server: control_plane_token_required")
		}
	}
	if strings.TrimSpace(config.ProjectWorkDir) != "" && !filepath.IsAbs(config.ProjectWorkDir) {
		return fmt.Errorf("orquesta_server: project_work_dir invalido")
	}
	if strings.TrimSpace(config.RuntimeWorkDir) != "" && !filepath.IsAbs(config.RuntimeWorkDir) {
		return fmt.Errorf("orquesta_server: runtime_work_dir invalido")
	}
	if strings.TrimSpace(config.IdleSelfImprovementProjectWorkDir) != "" &&
		!filepath.IsAbs(config.IdleSelfImprovementProjectWorkDir) {
		return fmt.Errorf("orquesta_server: idle_self_improvement_project_work_dir invalido")
	}
	return nil
}

func controlPlaneRemoteTokenStrongEnoughV0(token string) bool {
	token = strings.TrimSpace(token)
	if len(token) < 32 {
		return false
	}
	normalized := strings.ToLower(token)
	for _, prefix := range []string{"change-me", "changeme", "replace-me", "replaceme", "example", "placeholder"} {
		if strings.HasPrefix(normalized, prefix) {
			return false
		}
	}
	return true
}

func controlPlaneAddrIsLoopbackV0(addr string) bool {
	host, _, err := net.SplitHostPort(strings.TrimSpace(addr))
	if err != nil {
		host = strings.TrimSpace(addr)
	}
	host = strings.Trim(host, "[]")
	if host == "" {
		return false
	}
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func StatePathV0(config ConfigV0) string {
	config = NormalizeConfigV0(config)
	return filepath.Join(config.StateDir, config.StateFile)
}

func AuditPathV0(config ConfigV0) string {
	config = NormalizeConfigV0(config)
	return filepath.Join(config.StateDir, "audit", config.AuditFile)
}

func compactConfigStringsV0(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	if out == nil {
		return []string{}
	}
	return out
}

func normalizeIdleSelfImprovementCausalRequestsV0(requests []IdleSelfImprovementRequestV0) []IdleSelfImprovementRequestV0 {
	out := make([]IdleSelfImprovementRequestV0, 0, len(requests))
	byTarget := map[string]int{}
	for _, request := range requests {
		request.RequestRef = strings.TrimSpace(request.RequestRef)
		if request.RequestRef == "" {
			continue
		}
		baseRef := idleSelfImprovementCausalBaseRefV0(request.RequestRef)
		target := idleSelfImprovementDedupeTargetV0(request)
		request.ActiveAttemptRef = request.RequestRef
		if baseRef != request.RequestRef {
			request.ParentRunRef = firstNonEmptyIdleSelfImprovementV0(request.ParentRunRef, baseRef)
			request.SupersedesRunRef = firstNonEmptyIdleSelfImprovementV0(request.SupersedesRunRef, baseRef)
			request.RescueReason = firstNonEmptyIdleSelfImprovementV0(request.RescueReason, "retry")
			request.ContextRefs = compactConfigStringsV0(append(request.ContextRefs,
				"parent_run_ref:"+request.ParentRunRef,
				"rescue_reason:"+request.RescueReason,
				"supersedes_run_ref:"+request.SupersedesRunRef,
				"active_attempt_ref:"+request.ActiveAttemptRef,
			))
			request.EvidenceRefs = compactConfigStringsV0(append(request.EvidenceRefs, "evidence-ref-idle-self-improvement-rescue-causal-link"))
		}
		if index, ok := byTarget[target]; ok {
			out[index].ContextRefs = compactConfigStringsV0(append(out[index].ContextRefs,
				"deduped_rescue_ref:"+request.RequestRef,
			))
			out[index].EvidenceRefs = compactConfigStringsV0(append(out[index].EvidenceRefs, "evidence-ref-idle-self-improvement-rescue-deduped"))
			continue
		}
		byTarget[target] = len(out)
		out = append(out, request)
	}
	return out
}

func idleSelfImprovementCausalBaseRefV0(value string) string {
	value = strings.TrimSpace(value)
	for _, marker := range []string{"-retry-", "-reconcile-"} {
		if index := strings.Index(value, marker); index > 0 {
			return value[:index]
		}
	}
	return value
}
