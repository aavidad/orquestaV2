package orquestarails

import (
	"os"
	"strings"
)

const (
	RailsModeEnvV0               = "ORQUESTA_RAILS_MODE"
	RailsModeOfflineV0           = "offline"
	RailsModeAuditV0             = "audit"
	RailsModeEnforcedV0          = "enforced"
	SecurityModeEnvV0            = "ORQUESTA_SECURITY_MODE"
	SecurityModeProgrammingV0    = "programming"
	SecurityModeProductionLowV0  = "production_low"
	SecurityModeProductionV0     = "production"
	SecurityModeProductionHighV0 = "production_high"
)

func RailsModeV0() string {
	return NormalizeRailsModeV0(os.Getenv(RailsModeEnvV0))
}

func NormalizeRailsModeV0(value string) string {
	return RailsModeOfflineV0
}

func RailsOfflineV0() bool {
	return RailsModeV0() == RailsModeOfflineV0
}

func RailsAuditV0() bool {
	return false
}

func RailsEnforcedV0() bool {
	return false
}

func SecurityModeV0() string {
	return NormalizeSecurityModeV0(os.Getenv(SecurityModeEnvV0))
}

func NormalizeSecurityModeV0(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "legacy":
		return SecurityModeProductionV0
	case "programming", "programacion", "dev", "development", "open", "lax":
		return SecurityModeProgrammingV0
	case "production_low", "prod_low", "low":
		return SecurityModeProductionLowV0
	case "production_high", "prod_high", "high", "strict":
		return SecurityModeProductionHighV0
	case "production", "prod":
		return SecurityModeProductionV0
	default:
		return SecurityModeProductionV0
	}
}

func SecurityModeProgrammingEnabledV0() bool {
	return SecurityModeV0() == SecurityModeProgrammingV0
}

func SecurityModeProductionEnabledV0() bool {
	return !SecurityModeProgrammingEnabledV0()
}
