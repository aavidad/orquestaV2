package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestServerEnvRegistryV0EvitaLecturasDirectasORQUESTAEnProduccion(t *testing.T) {
	root := "."
	forbidden := regexp.MustCompile(`(os\.Getenv|os\.Setenv|envOrDefaultV0|intEnvOrDefaultV0|boolEnvOrDefaultV0|firstNonEmptyEnvV0|csvEnvOrDefaultV0|durationSecondsEnvOrDefaultV0|intEnvOrZeroV0)\("ORQUESTA_`)
	var hits []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() ||
			!strings.HasSuffix(path, ".go") ||
			strings.HasSuffix(path, "_test.go") ||
			filepath.Base(path) == "server_env_registry_v0.go" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if forbidden.Match(data) {
			hits = append(hits, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk env registry: %v", err)
	}
	if len(hits) > 0 {
		t.Fatalf("variables ORQUESTA_* leidas fuera del registro canonico: %s", strings.Join(hits, ", "))
	}
}

func TestServerEnvRegistryV0TieneMetadataParaConfiguracionEfectiva(t *testing.T) {
	for key, metadata := range serverEffectiveEnvRegistryV0 {
		if strings.TrimSpace(key) == "" ||
			strings.TrimSpace(metadata.Scope) == "" ||
			strings.TrimSpace(metadata.Label) == "" ||
			strings.TrimSpace(metadata.Description) == "" {
			t.Fatalf("metadata incompleta para %q: %+v", key, metadata)
		}
	}
}

func TestServerEnvRegistryV0TimeoutsDeclaranUnidadEnNombre(t *testing.T) {
	var invalid []string
	for key, metadata := range serverEffectiveEnvRegistryV0 {
		if metadata.Deprecated {
			continue
		}
		if !strings.Contains(key, "TIMEOUT") {
			continue
		}
		if strings.HasSuffix(key, "_TIMEOUT_MS") ||
			strings.HasSuffix(key, "_TIMEOUT_SECONDS") ||
			strings.Contains(key, "_TIMEOUT_MS_") ||
			strings.Contains(key, "_TIMEOUT_SECONDS_") ||
			strings.HasSuffix(key, "_TIMEOUT_READY") {
			continue
		}
		invalid = append(invalid, key)
	}
	if len(invalid) > 0 {
		t.Fatalf("timeouts sin unidad explicita en serverEffectiveEnvRegistryV0: %s", strings.Join(invalid, ", "))
	}
}
