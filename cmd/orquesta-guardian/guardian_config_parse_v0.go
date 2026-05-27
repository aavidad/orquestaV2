package main

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	guardianConfigInvalidBoolV0      = "guardian_config_invalid_bool"
	guardianConfigInvalidDurationV0  = "guardian_config_invalid_duration"
	guardianConfigInvalidIntV0       = "guardian_config_invalid_int"
	guardianConfigInvalidBudgetV0    = "guardian_config_invalid_budget"
	guardianConfigInvalidAllowlistV0 = "guardian_config_invalid_allowlist"
	guardianConfigInvalidFlagV0      = "guardian_config_invalid_flag"
)

type guardianConfigParseErrorV0 struct {
	Code  string
	Field string
}

func (err guardianConfigParseErrorV0) Error() string {
	if err.Field == "" {
		return err.Code
	}
	return err.Code + ":" + err.Field
}

func strictEnvBoolOrDefaultV0(key string, fallback bool) (bool, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}
	switch strings.ToLower(value) {
	case "1", "true", "yes", "on":
		return true, nil
	case "0", "false", "no", "off":
		return false, nil
	default:
		return false, guardianConfigParseErrorV0{Code: guardianConfigInvalidBoolV0, Field: key}
	}
}

func strictEnvDurationOrDefaultV0(key string, fallback time.Duration) (time.Duration, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}
	parsed, err := time.ParseDuration(value)
	if err != nil || parsed <= 0 {
		return 0, guardianConfigParseErrorV0{Code: guardianConfigInvalidDurationV0, Field: key}
	}
	return parsed, nil
}

func strictEnvIntOrDefaultV0(key string, fallback int) (int, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, guardianConfigParseErrorV0{Code: guardianConfigInvalidIntV0, Field: key}
	}
	return parsed, nil
}

func strictEnvPositiveIntOrDefaultV0(key string, fallback int) (int, error) {
	parsed, err := strictEnvIntOrDefaultV0(key, fallback)
	if err != nil {
		return 0, err
	}
	if parsed <= 0 {
		return 0, guardianConfigParseErrorV0{Code: guardianConfigInvalidBudgetV0, Field: key}
	}
	return parsed, nil
}

func strictEnvPositiveInt64OrDefaultV0(key string, fallback int64) (int64, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed <= 0 {
		return 0, guardianConfigParseErrorV0{Code: guardianConfigInvalidBudgetV0, Field: key}
	}
	return parsed, nil
}

func strictGuardianEnvAllowlistFromEnvV0() ([]string, error) {
	value := strings.TrimSpace(os.Getenv(envGuardianCommandEnvAllowlistV0))
	if value == "" {
		return defaultGuardianEnvAllowlistV0(), nil
	}
	items := compactStringsV0(strings.Split(value, ","))
	if len(items) == 0 {
		return nil, guardianConfigParseErrorV0{
			Code:  guardianConfigInvalidAllowlistV0,
			Field: envGuardianCommandEnvAllowlistV0,
		}
	}
	for _, item := range items {
		if !guardianValidEnvNameV0(item) {
			return nil, guardianConfigParseErrorV0{
				Code:  guardianConfigInvalidAllowlistV0,
				Field: envGuardianCommandEnvAllowlistV0,
			}
		}
	}
	return items, nil
}

func guardianValidEnvNameV0(value string) bool {
	if value == "" {
		return false
	}
	for i, r := range value {
		if r == '_' || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9' && i > 0) {
			continue
		}
		return false
	}
	return true
}

func guardianConfigReasonCodeV0(err error) string {
	var parseErr guardianConfigParseErrorV0
	if errors.As(err, &parseErr) && parseErr.Code != "" {
		return parseErr.Code
	}
	return ""
}

func validateParsedGuardianConfigV0(config guardianConfigV0) error {
	switch {
	case config.HealthTimeout <= 0:
		return guardianConfigParseErrorV0{Code: guardianConfigInvalidDurationV0, Field: "health-timeout"}
	case config.CommandTimeout <= 0:
		return guardianConfigParseErrorV0{Code: guardianConfigInvalidDurationV0, Field: "command-timeout"}
	case config.ShutdownTimeout <= 0:
		return guardianConfigParseErrorV0{Code: guardianConfigInvalidDurationV0, Field: "shutdown-timeout"}
	case config.LeaseTTL <= 0:
		return guardianConfigParseErrorV0{Code: guardianConfigInvalidDurationV0, Field: "lease-ttl"}
	case config.CommandOutputMaxBytes <= 0:
		return guardianConfigParseErrorV0{Code: guardianConfigInvalidBudgetV0, Field: "command-output-max-bytes"}
	case config.ArtifactMaxBytes <= 0:
		return guardianConfigParseErrorV0{Code: guardianConfigInvalidBudgetV0, Field: "artifact-max-bytes"}
	case config.ShutdownQueueLimit < 0:
		return guardianConfigParseErrorV0{Code: guardianConfigInvalidIntV0, Field: "shutdown-queue-limit"}
	case config.RepairMaxAttempts <= 0:
		return guardianConfigParseErrorV0{Code: guardianConfigInvalidBudgetV0, Field: "repair-max-attempts"}
	}
	for _, item := range config.EnvAllowlist {
		if !guardianValidEnvNameV0(item) {
			return guardianConfigParseErrorV0{Code: guardianConfigInvalidAllowlistV0, Field: "env-allowlist"}
		}
	}
	return nil
}
