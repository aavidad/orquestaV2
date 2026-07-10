package orquesta_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

const (
	envVarsBudgetMEJ106V0         = 425
	envVarsTestOnlyBudgetMEJ106V0 = 103
)

func TestEnvVarsBudgetMEJ106V0(t *testing.T) {
	root := findRepoRootForEnvVarsBudgetMEJ106V0(t)
	cmd := exec.Command("bash", "scripts/orquesta_metricas_deuda.sh", "--json")
	cmd.Dir = root
	// Bash may emit a locale warning before the metric script can normalize it.
	// The guard consumes JSON, so its child environment must be deterministic.
	cmd.Env = append(os.Environ(), "LC_ALL=C")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("orquesta_metricas_deuda.sh --json fallo: %v\n%s", err, output)
	}
	var metrics struct {
		EnvVarsOrquesta         int `json:"env_vars_orquesta"`
		EnvVarsOrquestaTestOnly int `json:"env_vars_orquesta_test_only"`
	}
	if err := json.Unmarshal(output, &metrics); err != nil {
		t.Fatalf("JSON metricas invalido: %v\n%s", err, output)
	}
	if metrics.EnvVarsOrquesta <= 0 {
		t.Fatalf("env_vars_orquesta invalido: %d", metrics.EnvVarsOrquesta)
	}
	if metrics.EnvVarsOrquesta > envVarsBudgetMEJ106V0 {
		t.Fatalf(
			"env vars productivas ORQUESTA_* suben a %d, maximo MEJ-106=%d; baja o consolida variables antes de anadir nuevas",
			metrics.EnvVarsOrquesta,
			envVarsBudgetMEJ106V0,
		)
	}
	if metrics.EnvVarsOrquestaTestOnly > envVarsTestOnlyBudgetMEJ106V0 {
		t.Fatalf(
			"env vars exclusivas de fixtures ORQUESTA_* suben a %d, maximo MEJ-106=%d; no las conviertas en configuracion global",
			metrics.EnvVarsOrquestaTestOnly,
			envVarsTestOnlyBudgetMEJ106V0,
		)
	}
}

func findRepoRootForEnvVarsBudgetMEJ106V0(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if hasDirEnvVarsBudgetMEJ106V0(dir, "cmd") &&
			hasDirEnvVarsBudgetMEJ106V0(dir, "modulos") &&
			hasDirEnvVarsBudgetMEJ106V0(dir, "scripts") {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("repo root no encontrado desde %s", dir)
		}
		dir = parent
	}
}

func hasDirEnvVarsBudgetMEJ106V0(root string, name string) bool {
	info, err := os.Stat(filepath.Join(root, name))
	return err == nil && info.IsDir()
}
