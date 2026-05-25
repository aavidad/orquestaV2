package stopreason

import "testing"

func TestProjectV0NormalizaTaxonomiaPublicaV0(t *testing.T) {
	cases := []struct {
		name     string
		stop     string
		category string
		public   string
	}{
		{"idle", "no_execution", CategoryIdleV0, PublicReasonIdleNoExecutionV0},
		{"run_budget", "max_ticks", CategoryBudgetV0, PublicReasonBudgetMaxTicksV0},
		{"director_budget", "stop_max_steps", CategoryBudgetV0, PublicReasonBudgetMaxStepsV0},
		{"outbox", "wait_outbox", CategoryWaitOutboxV0, PublicReasonWaitOutboxV0},
		{"external", "wait_external", CategoryWaitExternalV0, PublicReasonWaitExternalV0},
		{"step_error", "stop_error", CategoryErrorV0, PublicReasonErrorStepV0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			projection := ProjectV0(ProjectionInputV0{Source: "test", StopReason: tc.stop})
			if projection.Category != tc.category || projection.PublicReason != tc.public {
				t.Fatalf("projection=%+v", projection)
			}
		})
	}
}

func TestProjectV0CompactaRefsYContadoresV0(t *testing.T) {
	projection := ProjectV0(ProjectionInputV0{
		Source:            " source ",
		StopReason:        " wait_outbox ",
		Executions:        2,
		Ticks:             3,
		PendingOutboxRefs: []string{" outbox-ref-001 ", "", "outbox-ref-001", "outbox-ref-002"},
	})
	if projection.Source != "source" ||
		projection.StopReason != "wait_outbox" ||
		projection.Executions != 2 ||
		projection.Ticks != 3 ||
		len(projection.PendingOutboxRefs) != 2 {
		t.Fatalf("projection=%+v", projection)
	}
}
