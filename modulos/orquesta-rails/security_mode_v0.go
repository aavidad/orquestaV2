package orquestarails

import (
	"os"
	"strings"
)

const (
	SecurityModeEnvV0            = "ORQUESTA_SECURITY_MODE"
	SecurityModeProgrammingV0    = "programming"
	SecurityModeProductionLowV0  = "production_low"
	SecurityModeProductionV0     = "production"
	SecurityModeProductionHighV0 = "production_high"
)

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
