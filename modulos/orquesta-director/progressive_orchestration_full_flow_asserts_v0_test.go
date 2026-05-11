package orquestadirector

import (
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func assertProgressiveRuntimeLaunchV0(
	t *testing.T,
	launched orquestaruntime.RuntimeFakeLifecycleSnapshotV0,
	agentRef string,
) {
	t.Helper()
	if launched.Status != orquestaruntime.RuntimeFakeLifecycleLaunchedV0 {
		t.Fatalf("runtime launch status=%s, want launched", launched.Status)
	}
	if launched.AgentRequestID != agentRef {
		t.Fatalf("runtime launch agent=%q, want %q", launched.AgentRequestID, agentRef)
	}
}

func assertProgressiveFullFlowClosedV0(
	t *testing.T,
	run orquestacoreworkflow.OrchestrationRunV0,
	taskRef string,
	deliveryRef string,
	reviewRef string,
	acceptedReviewRef string,
	validationRef string,
	closureRef string,
) {
	t.Helper()
	if run.Status != orquestacoreworkflow.OrchestrationRunStatusClosedV0 ||
		run.CurrentPhase != orquestacoreworkflow.OrchestrationPhaseCierreV0 {
		t.Fatalf("run final inesperado: status=%s phase=%s", run.Status, run.CurrentPhase)
	}
	for ref, values := range map[string][]string{
		taskRef:           run.ClosedTasks,
		deliveryRef:       run.Deliveries,
		reviewRef:         run.Reviews,
		acceptedReviewRef: run.AcceptedReviews,
		validationRef:     run.Validations,
		closureRef:        run.Closures,
	} {
		if !containsProgressiveRefV0(values, ref) {
			t.Fatalf("ref %q no proyectada en %+v", ref, values)
		}
	}
}
