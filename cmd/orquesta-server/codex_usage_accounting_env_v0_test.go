package main

import (
	"os"
	"testing"
)

func TestCodexUsageMetricsFromEnvV0OptIn(t *testing.T) {
	t.Setenv("ORQUESTA_CODEX_USAGE_ACCOUNTING", "")
	if got := codexUsageMetricsFromEnvV0(nil); got != nil {
		t.Fatalf("usage metrics sin opt-in: %#v", got)
	}
	t.Setenv("ORQUESTA_CODEX_USAGE_ACCOUNTING", "runtime_logs")
	if got := codexUsageMetricsFromEnvV0(nil); got != nil {
		t.Fatalf("usage metrics no debe leer logs runtime sin reporte redactado")
	}
	t.Setenv("ORQUESTA_CODEX_USAGE_ACCOUNTING", "redacted_report")
	t.Setenv("ORQUESTA_CODEX_USAGE_LOG_MAX_BYTES", "2048")
	if got := codexUsageMetricsFromEnvV0(nil); got == nil {
		t.Fatalf("usage metrics no habilitado con opt-in")
	}
}

func TestInt64EnvOrDefaultV0(t *testing.T) {
	const key = "ORQUESTA_CODEX_USAGE_ACCOUNTING_TEST_VALUE"
	_ = os.Unsetenv(key)
	if got := int64EnvOrDefaultV0(key, 10); got != 10 {
		t.Fatalf("default=%d", got)
	}
	t.Setenv(key, "42")
	if got := int64EnvOrDefaultV0(key, 10); got != 42 {
		t.Fatalf("value=%d", got)
	}
}
