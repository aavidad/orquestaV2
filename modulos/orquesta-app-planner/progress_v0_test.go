package orquestaappplanner

import "testing"

func TestEvaluateAppPlanProgressV0ExponeListasParaContinuarHastaTerminar(t *testing.T) {
	plan := mustLargeAppPlanForProgressTestV0(t)

	progress, err := EvaluateAppPlanProgressV0(plan, []string{"ack-erp-bootstrap"})
	if err != nil {
		t.Fatalf("EvaluateAppPlanProgressV0: %v", err)
	}
	if progress.Complete || progress.DeliveredUnits != 1 {
		t.Fatalf("progress=%+v", progress)
	}
	if !sameStringSetForTestV0(progress.ReadyTaskRefs, []string{"task-erp-architecture"}) {
		t.Fatalf("ready=%v", progress.ReadyTaskRefs)
	}
	if !appPlannerStringInSetV0(progress.BlockedTaskRefs, "task-erp-domain") {
		t.Fatalf("blocked=%v", progress.BlockedTaskRefs)
	}
}

func TestEvaluateAppPlanProgressV0MarcaCompletoConTodasLasEntregas(t *testing.T) {
	plan := mustLargeAppPlanForProgressTestV0(t)
	deliveries := make([]string, 0, len(plan.Units))
	for _, unit := range plan.Units {
		deliveries = append(deliveries, unit.DeliveryRef)
	}

	progress, err := EvaluateAppPlanProgressV0(plan, deliveries)
	if err != nil {
		t.Fatalf("EvaluateAppPlanProgressV0: %v", err)
	}
	if !progress.Complete || len(progress.PendingTaskRefs) != 0 {
		t.Fatalf("progress=%+v", progress)
	}
}

func mustLargeAppPlanForProgressTestV0(t *testing.T) AppMicrotaskPlanV0 {
	t.Helper()
	plan, err := BuildGoAPIWebMicrotaskPlanV0(AppPlanRequestV0{
		RunRef:  "run-ref-large-progress-001",
		AppRef:  "erp",
		AppName: "ERP",
		API:     true,
		Web:     true,
		Scale:   AppPlanScaleLargeV0,
	})
	if err != nil {
		t.Fatalf("BuildGoAPIWebMicrotaskPlanV0 large: %v", err)
	}
	return plan
}
