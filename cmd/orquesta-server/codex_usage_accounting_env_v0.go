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
	mode := strings.TrimSpace(os.Getenv(envCodexUsageAccountingV0))
	if mode != "redacted_report" && mode != "runtime_usage_report" {
		return nil
	}
	return orquestaappcodexstack.CodexStackRuntimeUsageMetricsSourceV0{
		Store:    receiptStore,
		MaxBytes: int64EnvOrDefaultV0(envCodexUsageLogMaxBytesV0, 65536),
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
