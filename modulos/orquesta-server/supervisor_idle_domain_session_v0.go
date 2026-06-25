package orquestaserver

import (
	"context"
	"path/filepath"
	"strings"
	"time"
)

const (
	idleSelfImprovementDomainSessionSuppressedReasonV0 = "idle_self_improvement_suppressed_by_domain_session"
	idleSelfImprovementDomainSessionEvidenceV0         = "evidence-ref-idle-self-improvement-domain-session"
)

func (runtime *RuntimeV0) idleSelfImprovementDomainSessionSuppressionV0() IdleSelfImprovementBlockerResultV0 {
	if idleSelfImprovementDomainSessionOptInV0(runtime.config) {
		return IdleSelfImprovementBlockerResultV0{}
	}
	state := runtime.tracker.SnapshotV0()
	if !idleSelfImprovementDomainSessionConfiguredV0(runtime.config, state) {
		return IdleSelfImprovementBlockerResultV0{}
	}
	return IdleSelfImprovementBlockerResultV0{
		Blocked:        true,
		Reason:         idleSelfImprovementDomainSessionSuppressedReasonV0,
		EvidenceRefs:   []string{idleSelfImprovementDomainSessionEvidenceV0},
		Message:        "automejora idle suprimida por sesion de dominio activa o reciente",
		RecoveryAction: "configure_idle_self_improvement_project_workdir_separado",
		NextActions: []string{
			"mantener_automejora_en_pausa_hasta_cerrar_sesion_dominio",
			"configurar_workdir_separado_para_opt_in_explicito",
		},
	}
}

func (runtime *RuntimeV0) blockIdleSelfImprovementDomainSessionV0(
	ctx context.Context,
	now time.Time,
) bool {
	blocked := runtime.idleSelfImprovementDomainSessionSuppressionV0()
	if !blocked.Blocked {
		return false
	}
	runtime.auditEventV0(ctx, "idle_self_improvement_check", "blocked", "", map[string]interface{}{"reason": blocked.Reason})
	runtime.persistStateTransitionV0(ctx, runtime.tracker.MarkIdleSelfImprovementBlockedV0(blocked, now), "idle_self_improvement_blocked")
	return true
}

func idleSelfImprovementDomainSessionOptInV0(config ConfigV0) bool {
	projectDir := cleanComparableWorkDirV0(config.ProjectWorkDir)
	idleDir := cleanComparableWorkDirV0(config.IdleSelfImprovementProjectWorkDir)
	return projectDir != "" && idleDir != "" && projectDir != idleDir
}

func idleSelfImprovementDomainSessionConfiguredV0(config ConfigV0, state StateV0) bool {
	if strings.TrimSpace(state.ExternalBridgeComponent) != "" ||
		strings.TrimSpace(state.ExternalBridgeStatus) != "" {
		return true
	}
	for _, setting := range config.EffectiveConfig.Settings {
		key := strings.ToUpper(strings.TrimSpace(setting.Key))
		scope := strings.ToLower(strings.TrimSpace(setting.Scope))
		if !idleSelfImprovementDomainSessionSettingActiveV0(setting) {
			continue
		}
		if strings.HasPrefix(key, "ORQUESTA_OPES_") ||
			strings.Contains(scope, "external_bridge") ||
			strings.Contains(scope, "domain_work") ||
			strings.Contains(scope, "opes") {
			return true
		}
	}
	return false
}

func idleSelfImprovementDomainSessionSettingActiveV0(setting ServerConfigSettingV0) bool {
	value := strings.ToLower(strings.TrimSpace(setting.Value))
	source := strings.ToLower(strings.TrimSpace(setting.Source))
	if value == "" || value == "false" || value == "0" || value == "absent" {
		return false
	}
	if source == "defaulted" {
		return false
	}
	return source == "explicit" ||
		source == "derived" ||
		strings.Contains(value, "configured") ||
		value == "true"
}

func cleanComparableWorkDirV0(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if abs, err := filepath.Abs(value); err == nil {
		value = abs
	}
	return filepath.Clean(value)
}
