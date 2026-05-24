package orquestaappcodexstack

import (
	"context"
	"testing"

	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
)

func TestReviewReworkReplanSourceV0FallbackNoCreaSegundoPadreSinReceipt(t *testing.T) {
	source := ReviewReworkReplanSourceV0{
		Store: orquestaruntimecodexdelivery.NewInMemoryCodexReceiptDescriptorStoreV0(),
	}
	request := reviewReworkPlanRequestForTestV0(false)
	request.Run.Agents = []string{"agent-ref-task-ref-first"}
	request.Run.StartedAgents = []string{"agent-ref-task-ref-first"}

	plans, err := source.BuildReviewReworkReplanPlansV0(context.Background(), request)
	if err != nil {
		t.Fatalf("BuildReviewReworkReplanPlansV0: %v", err)
	}
	if len(plans) != 0 {
		t.Fatalf("fallback sin receipt no debe relanzar otro padre: %+v", plans)
	}
}
