package main

import (
	"os"
	"strconv"
	"strings"

	orquestaappcodexstack "orquesta/modulos/orquesta-app-codex-stack"
)

func codexUsageMetricsFromEnvV0(
	receiptStore orquestaappcodexstack.CodexReceiptStorePortV0,
) orquestaappcodexstack.CodexStackAgentUsageMetricsProviderPortV0 {
	return codexUsageMetricsFromModeV0(
		strings.TrimSpace(os.Getenv(envCodexUsageAccountingV0)),
		int64EnvOrDefaultV0(envCodexUsageLogMaxBytesV0, 65536),
		receiptStore,
	)
}

func codexUsageMetricsFromProjectConfigV0(
	projectDir string,
	receiptStore orquestaappcodexstack.CodexReceiptStorePortV0,
) orquestaappcodexstack.CodexStackAgentUsageMetricsProviderPortV0 {
	config, _, err := loadServerProjectConfigFileV0(projectDir)
	if err != nil {
		return codexUsageMetricsFromEnvV0(receiptStore)
	}
	return codexUsageMetricsFromModeV0(
		codexUsageAccountingModeFromProjectConfigFileV0(config),
		codexUsageLogMaxBytesFromProjectConfigFileV0(config),
		receiptStore,
	)
}

func codexUsageAccountingModeFromProjectConfigFileV0(config serverProjectConfigFileV0) string {
	return stringProjectConfigOrEnvOrDefaultV0(
		envCodexUsageAccountingV0,
		config.CodexUsageAccounting.Mode,
		"",
	)
}

func codexUsageLogMaxBytesFromProjectConfigFileV0(config serverProjectConfigFileV0) int64 {
	return int64ProjectConfigOrEnvOrDefaultV0(
		envCodexUsageLogMaxBytesV0,
		config.CodexUsageAccounting.LogMaxBytes,
		65536,
	)
}

func codexUsageMetricsFromModeV0(
	mode string,
	maxBytes int64,
	receiptStore orquestaappcodexstack.CodexReceiptStorePortV0,
) orquestaappcodexstack.CodexStackAgentUsageMetricsProviderPortV0 {
	mode = strings.TrimSpace(mode)
	if mode != "redacted_report" && mode != "runtime_usage_report" {
		return nil
	}
	return orquestaappcodexstack.CodexStackRuntimeUsageMetricsSourceV0{
		Store:    receiptStore,
		MaxBytes: maxBytes,
	}
}

func int64EnvOrDefaultV0(key string, fallback int64) int64 {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}
