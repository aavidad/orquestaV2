package orquestaruntime

import (
	"context"
	"testing"
	"time"
)

func TestProcessRuntimeEffectContextV0AplicaDeadlineLegacy(t *testing.T) {
	ctx, cancel := processRuntimeEffectContextV0(context.Background(), time.Second)
	defer cancel()

	if _, ok := ctx.Deadline(); !ok {
		t.Fatalf("contexto sin deadline")
	}
}

func TestProcessRuntimeEffectPolicyV0NormalizaTimeouts(t *testing.T) {
	connector := NewProcessRuntimeConnectorWithPolicyV0(ProcessRuntimeEffectPolicyV0{
		LaunchTimeout: 5 * time.Millisecond,
		StopTimeout:   7 * time.Millisecond,
	})

	if connector.effectPolicy.LaunchTimeout != 5*time.Millisecond ||
		connector.effectPolicy.StopTimeout != 7*time.Millisecond {
		t.Fatalf("policy=%+v", connector.effectPolicy)
	}
}
