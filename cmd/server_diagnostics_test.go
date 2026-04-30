package cmd

import (
	"testing"
	"time"

	"orquesta/agentesapp"
)

func TestBuildServerRuntimeHealthIncluyeHotPaths(t *testing.T) {
	prevPanel := serverRuntimeHealthPanelDiagnosticsFn
	defer func() { serverRuntimeHealthPanelDiagnosticsFn = prevPanel }()

	recordServerSelfHealDiagnostics(2*time.Second, map[string]time.Duration{
		"degradados": 1200 * time.Millisecond,
		"autonomia":  300 * time.Millisecond,
	})
	serverRuntimeHealthPanelDiagnosticsFn = func() agentesapp.PanelBuildDiagnostics {
		return agentesapp.PanelBuildDiagnostics{
			Generated: "2026-04-30T18:00:00Z",
			TotalMS:   750,
			Slow:      true,
			PhasesMS:  map[string]int64{"checkpoints": 500},
		}
	}

	health := buildServerRuntimeHealth()
	if health.Goroutines <= 0 {
		t.Fatalf("goroutines inesperadas: %+v", health)
	}
	if health.HotPaths.LastSelfHeal == nil || health.HotPaths.LastSelfHeal.TotalMS != 2000 {
		t.Fatalf("last self_heal ausente: %+v", health.HotPaths)
	}
	if health.HotPaths.LastPanelBuild == nil || health.HotPaths.LastPanelBuild.TotalMS != 750 {
		t.Fatalf("last panel build ausente: %+v", health.HotPaths)
	}
}

func TestMCPServerRuntimeHealthAndHotPathsTools(t *testing.T) {
	prevPanel := serverRuntimeHealthPanelDiagnosticsFn
	defer func() { serverRuntimeHealthPanelDiagnosticsFn = prevPanel }()

	recordServerSelfHealDiagnostics(1500*time.Millisecond, map[string]time.Duration{
		"wake": 10 * time.Millisecond,
	})
	serverRuntimeHealthPanelDiagnosticsFn = func() agentesapp.PanelBuildDiagnostics {
		return agentesapp.PanelBuildDiagnostics{
			Generated: "2026-04-30T18:00:00Z",
			TotalMS:   640,
			Slow:      true,
		}
	}

	result, err := callMCPTool("orquesta.server.runtime_health", nil)
	if err != nil {
		t.Fatalf("runtime_health tool: %v", err)
	}
	health, _ := result["structuredContent"].(serverRuntimeHealth)
	if health.HotPaths.LastSelfHeal == nil || health.HotPaths.LastPanelBuild == nil {
		t.Fatalf("runtime_health sin hot paths: %+v", health)
	}

	result, err = callMCPTool("orquesta.server.hot_paths", nil)
	if err != nil {
		t.Fatalf("hot_paths tool: %v", err)
	}
	hotPaths, _ := result["structuredContent"].(serverHotPathDiagnostics)
	if hotPaths.LastSelfHeal == nil || hotPaths.LastPanelBuild == nil {
		t.Fatalf("hot_paths incompleto: %+v", hotPaths)
	}
}
