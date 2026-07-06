package orquesta_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

const envVarsBudgetMEJ106V0 = 513

func TestEnvVarsBudgetMEJ106V0(t *testing.T) {
	root := findRepoRootForEnvVarsBudgetMEJ106V0(t)
	cmd := exec.Command("bash", "scripts/orquesta_metricas_deuda.sh", "--json")
	cmd.Dir = root
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("orquesta_metricas_deuda.sh --json fallo: %v\n%s", err, output)
	}
	var metrics struct {
		EnvVarsOrquesta int `json:"env_vars_orquesta"`
	}
	if err := json.Unmarshal(output, &metrics); err != nil {
		t.Fatalf("JSON metricas invalido: %v\n%s", err, output)
	}
	if metrics.EnvVarsOrquesta <= 0 {
		t.Fatalf("env_vars_orquesta invalido: %d", metrics.EnvVarsOrquesta)
	}
	if metrics.EnvVarsOrquesta > envVarsBudgetMEJ106V0 {
		if envVarsBudgetIncreaseJustifiedMEJ106V0(root, metrics.EnvVarsOrquesta) {
			return
		}
		t.Fatalf(
			"env vars ORQUESTA_* suben a %d, maximo MEJ-106=%d; baja o consolida variables antes de anadir nuevas",
			metrics.EnvVarsOrquesta,
			envVarsBudgetMEJ106V0,
		)
	}
}

func envVarsBudgetIncreaseJustifiedMEJ106V0(root string, value int) bool {
	needle := "env_vars_orquesta_allow_increase_to=" + strconv.Itoa(value)
	for _, rel := range []string{
		"docs/bitacora_correccion_pericial_2026-07-03.md",
		"docs/inventario_bugs_orquesta_2026-06-30.md",
	} {
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err == nil && strings.Contains(string(data), needle) {
			return true
		}
	}
	return false
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
