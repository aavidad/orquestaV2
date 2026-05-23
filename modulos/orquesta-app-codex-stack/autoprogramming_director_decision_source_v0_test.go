package orquestaappcodexstack

import (
	"context"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoragent "orquesta/modulos/orquesta-director-agent"
	orquestadirectoragentworkflow "orquesta/modulos/orquesta-director-agent-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func TestAutoprogrammingDirectorDecisionSourceV0AbreRevisionTrasEntregaV0(t *testing.T) {
	runRef := "run-autoprogramming-open-review-001"
	taskRef := "task-autoprogramming-open-review-001"
	agentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(taskRef)
	run := stackDeliveredRunForQueueTestV0(runRef, taskRef, agentRef)
	run.AppSpecRef = "app-spec-ref-autoprogramming-open-review-001"
	run.CurrentPhase = orquestacoreworkflow.OrchestrationPhaseProgramacionV0
	source := AutoprogrammingDirectorDecisionSourceV0{
		TaskStore: orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(
			stackDeliveredAutoprogrammingTaskForTestV0(runRef, taskRef),
		),
	}

	decisions, err := source.ListDirectorAgentDecisionsV0(
		context.Background(),
		orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0{Run: run},
	)
	if err != nil {
		t.Fatalf("ListDirectorAgentDecisionsV0: %v", err)
	}
	if len(decisions) != 1 {
		t.Fatalf("decisions=%+v", decisions)
	}
	decision := decisions[0]
	if decision.CommandType != orquestadirectoragent.DirectorAgentCommandOpenPhaseV0 ||
		decision.OpenPhase == nil ||
		decision.OpenPhase.PhaseID != string(orquestacoreworkflow.OrchestrationPhaseRevisionV0) {
		t.Fatalf("decision=%+v", decision)
	}
	if !codexStackStringInSetForTestV0(decision.EvidenceRefs, autoprogrammingDirectorOpenReviewEvidenceV0) {
		t.Fatalf("evidence_refs=%v", decision.EvidenceRefs)
	}
}

func TestAutoprogrammingDirectorDecisionSourceV0NoAbreRevisionConAgentesPendientesV0(t *testing.T) {
	runRef := "run-autoprogramming-open-review-pending-001"
	taskRef := "task-autoprogramming-open-review-pending-001"
	agentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(taskRef)
	run := stackDeliveredRunForQueueTestV0(runRef, taskRef, agentRef)
	run.AppSpecRef = "app-spec-ref-autoprogramming-open-review-pending-001"
	run.CurrentPhase = orquestacoreworkflow.OrchestrationPhaseProgramacionV0
	run.DeliveredAgents = nil
	run.DeliveredTasks = nil
	run.Deliveries = nil
	source := AutoprogrammingDirectorDecisionSourceV0{
		TaskStore: orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(
			stackDeliveredAutoprogrammingTaskForTestV0(runRef, taskRef),
		),
	}

	decisions, err := source.ListDirectorAgentDecisionsV0(
		context.Background(),
		orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0{Run: run},
	)
	if err != nil {
		t.Fatalf("ListDirectorAgentDecisionsV0: %v", err)
	}
	if len(decisions) != 0 {
		t.Fatalf("decisions=%+v", decisions)
	}
}
