package acceptance

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestV38B01ProviderIsolationQuotaAndPlacementStayOrthogonal(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Dir(filepath.Dir(file))
	selection := v38B01Read(t, root, "internal/bootstrap/runtime.go")
	provider := strings.Index(selection, `snapshot.RuntimeProvider() != "codex"`)
	isolation := strings.Index(selection, `snapshot.RuntimeIsolation() != "process"`)
	if provider < 0 || isolation < provider || !strings.Contains(selection, "bootstrap.runtime_isolation_not_composed") {
		t.Fatal("la selección parcial no falla cerrada antes del adaptador de proceso")
	}

	quota := v38B01Read(t, root, "internal/adapters/agent/codex/controlador_cuota.go")
	for _, authority := range []string{"AgentLauncher", "StateRepository", "WorkItem", "GoalRecord"} {
		if strings.Contains(quota, authority) {
			t.Fatalf("el controlador de cuota ganó autoridad %s", authority)
		}
	}

	capacity := v38B01Read(t, root, "internal/application/candidatos_capacidad_agente.go")
	processing := v38B01Read(t, root, "internal/application/processing.go")
	governance := v38B01Read(t, root, "internal/bootstrap/governance.go")
	if !strings.Contains(capacity, "PlacementRef") || !strings.Contains(processing, "CapacityCandidates:") ||
		strings.Contains(processing+governance, "RuntimeCodexMaxConcurrentExecutions") {
		t.Fatal("colocación, capacidad o límite global volvieron a depender de Codex")
	}

	b04 := v38B01Read(t, root, "product/evidence/candidates/v38/b04_compatibility_v1.json")
	if !strings.Contains(b04, `"protocol":"agentmicrovm.local.v1"`) ||
		!strings.Contains(b04, `"status":"exercised_without_kvm"`) {
		t.Fatal("B01 no está ligado al contrato cruzado B04")
	}
}

func v38B01Read(t *testing.T, root, relative string) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(root, relative))
	if err != nil {
		t.Fatal(err)
	}
	return string(content)
}
