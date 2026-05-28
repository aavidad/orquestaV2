package main

import (
	"os"
	"strings"
)

const (
	codexExecutionModeParallelV0 = "parallel"
	codexExecutionModeSerialV0   = "serial"
)

func codexExecutionModeFromEnvV0() string {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(envCodexExecutionModeV0))) {
	case codexExecutionModeSerialV0:
		return codexExecutionModeSerialV0
	case codexExecutionModeParallelV0, "":
		return codexExecutionModeParallelV0
	default:
		return defaultCodexExecutionModeV0
	}
}

func codexExecutionModeCapPositiveV0(mode string, value int) int {
	if value <= 0 {
		value = 1
	}
	if mode == codexExecutionModeSerialV0 && value > 1 {
		return 1
	}
	return value
}

func codexExecutionModeCapIntEnvOrDefaultV0(mode string, key string, fallback int) int {
	return codexExecutionModeCapPositiveV0(mode, intEnvOrDefaultV0(key, fallback))
}
