package orquestaappcodexstack

import (
	"testing"
	"time"

	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
)

func TestBuildStackV0CableaProgressLeaseBridgeConBudgetV0(t *testing.T) {
	config := ConfigV0{
		Codex: CodexRuntimeConfigV0{
			ProgressBudget: codexProgressBudgetForLeaseBridgeTestV0(),
		},
	}

	progress, lease := progressAndLeaseSourcesV0(config)
	if progress == nil || lease == nil {
		t.Fatalf("progress=%T lease=%T", progress, lease)
	}
	progressBridge, ok := progress.(*orquestacionnucleoapp.AgentProgressLeaseBridgeV0)
	if !ok {
		t.Fatalf("progress source=%T", progress)
	}
	leaseBridge, ok := lease.(*orquestacionnucleoapp.AgentProgressLeaseBridgeV0)
	if !ok || leaseBridge != progressBridge {
		t.Fatalf("lease source=%T progress=%T", lease, progress)
	}
	policy, ok := codexStackLeasePolicyV0(config)
	if !ok || policy.HeartbeatTimeoutSeconds != 90 || policy.TotalTimeoutSeconds != 240 {
		t.Fatalf("policy=%+v ok=%v", policy, ok)
	}
}

func TestProgressAndLeaseSourcesV0SinBudgetNoActivaLeaseV0(t *testing.T) {
	progress, lease := progressAndLeaseSourcesV0(ConfigV0{})
	if progress == nil {
		t.Fatalf("progress source nil")
	}
	if lease != nil {
		t.Fatalf("lease source=%T", lease)
	}
}

func codexProgressBudgetForLeaseBridgeTestV0() orquestaruntimecodexdelivery.CodexBudgetActivityPolicyV0 {
	return orquestaruntimecodexdelivery.CodexBudgetActivityPolicyV0{
		MaxExpected:     240 * time.Second,
		NoActivityLimit: 90 * time.Second,
	}
}
