package main

import (
	"os"
	"strings"
)

func codexSandboxFromEnvV0(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func codexOptionalSandboxFromEnvV0(key string) string {
	return strings.TrimSpace(os.Getenv(key))
}
