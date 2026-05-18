package orquestaappcodexstack

import (
	"context"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
)

func TestAssessmentReplanSourceV0PlanificaReemplazoTrasStopConfirmado(t *testing.T) {
	runRef := "run-ref-assessment-replan-stack-001"
	oldAgentRef := "agent-ref-assessment-old-001"
	taskRef := "task-ref-assessment-stack-001"
	source := AssessmentReplanSourceV0{
		Store: orquestaruntimecodexdelivery.NewInMemoryCodexReceiptDescriptorStoreV0(
			assessmentReplanDescriptorForTestV0(runRef, oldAgentRef, taskRef),
		),
		Capacity: CapacityConfigV0{Tier: orquestacoreworkflow.OrchestrationCapacityXHighV0},
	}

	plans, err := source.BuildAgentAssessmentReplanPlansV0(
		context.Background(),
		assessmentReplanRequestForTestV0(runRef, oldAgentRef, taskRef, false),
	)
	if err != nil {
		t.Fatalf("BuildAgentAssessmentReplanPlansV0: %v", err)
	}
	if len(plans) != 1 {
		t.Fatalf("plans=%d %+v", len(plans), plans)
	}
	plan := plans[0]
	if plan.TaskRef != taskRef ||
		plan.Assessment.AgentRequestID != oldAgentRef ||
		plan.RequestedAction != "replace_agent" ||
		plan.MinimumRecommendedCapacity != orquestacoreworkflow.OrchestrationCapacityXHighV0 ||
		plan.AgentRequestID == "" ||
		plan.CapacityRequestRef == "" {
		t.Fatalf("plan inesperado: %+v", plan)
	}
}

func TestAssessmentReplanSourceV0PlanificaReemplazoTrasAgentePerdido(t *testing.T) {
	runRef := "run-ref-assessment-replan-stack-lost-001"
	oldAgentRef := "agent-ref-assessment-lost-001"
	taskRef := "task-ref-assessment-stack-lost-001"
	source := AssessmentReplanSourceV0{
		Store: orquestaruntimecodexdelivery.NewInMemoryCodexReceiptDescriptorStoreV0(
			assessmentReplanDescriptorForTestV0(runRef, oldAgentRef, taskRef),
		),
	}
	request := assessmentReplanRequestForTestV0(runRef, oldAgentRef, taskRef, false)
	request.Run.ConfirmedStoppedAgents = nil
	request.Run.LostAgents = []string{oldAgentRef}

	plans, err := source.BuildAgentAssessmentReplanPlansV0(context.Background(), request)
	if err != nil {
		t.Fatalf("BuildAgentAssessmentReplanPlansV0 lost: %v", err)
	}
	if len(plans) != 1 || plans[0].Assessment.AgentRequestID != oldAgentRef {
		t.Fatalf("plans lost=%+v", plans)
	}
}

func TestAssessmentReplanSourceV0NoDuplicaSiReplacementYaExiste(t *testing.T) {
	runRef := "run-ref-assessment-replan-stack-002"
	oldAgentRef := "agent-ref-assessment-old-002"
	taskRef := "task-ref-assessment-stack-002"
	source := AssessmentReplanSourceV0{
		Store: orquestaruntimecodexdelivery.NewInMemoryCodexReceiptDescriptorStoreV0(
			assessmentReplanDescriptorForTestV0(runRef, oldAgentRef, taskRef),
		),
	}
	request := assessmentReplanRequestForTestV0(runRef, oldAgentRef, taskRef, false)
	first, err := source.BuildAgentAssessmentReplanPlansV0(context.Background(), request)
	if err != nil || len(first) != 1 {
		t.Fatalf("first plans=%+v err=%v", first, err)
	}
	request = assessmentReplanRequestForTestV0(runRef, oldAgentRef, taskRef, true)
	request.Run.Agents = []string{first[0].AgentRequestID}

	plans, err := source.BuildAgentAssessmentReplanPlansV0(context.Background(), request)
	if err != nil {
		t.Fatalf("BuildAgentAssessmentReplanPlansV0 duplicate: %v", err)
	}
	if len(plans) != 0 {
		t.Fatalf("plan duplicado=%+v", plans)
	}
}

func TestBuildStackV0CableaAssessmentReplanSource(t *testing.T) {
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	if stack.Ports.AssessmentReplanSource == nil {
		t.Fatalf("AssessmentReplanSource no cableado")
	}
}

func assessmentReplanRequestForTestV0(
	runRef string,
	oldAgentRef string,
	taskRef string,
	withoutTaskInProjection bool,
) orquestacionnucleoapp.AgentAssessmentReplanPlanRequestV0 {
	projectionTask := taskRef
	if withoutTaskInProjection {
		projectionTask = ""
	}
	payload := orquestacoreworkflow.AgentWorkAssessedPayloadV0{
		AssessmentRef:  "assessment-ref-" + oldAgentRef,
		PhaseID:        string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
		AgentRequestID: oldAgentRef,
		TaskRef:        projectionTask,
		Verdict:        orquestacoreworkflow.AgentAssessmentVerdictLoopDetectedV0,
		Action:         orquestacoreworkflow.AgentAssessmentActionStopAgentV0,
		Severity:       orquestacoreworkflow.AgentAssessmentSeverityCriticalV0,
	}
	return orquestacionnucleoapp.AgentAssessmentReplanPlanRequestV0{
		Run: orquestacoreworkflow.OrchestrationRunV0{
			RunID:                  runRef,
			CurrentPhase:           orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
			Tasks:                  []string{taskRef},
			AgentAssessments:       []string{orquestacoreworkflow.AgentAssessmentProjectionRefV0(payload)},
			StoppedAgents:          []string{oldAgentRef},
			ConfirmedStoppedAgents: []string{oldAgentRef},
		},
		OccurredAt:   "2026-05-10T13:00:00Z",
		EvidenceRefs: []string{"evidence-ref-assessment-replan-test"},
	}
}

func assessmentReplanDescriptorForTestV0(
	runRef string,
	agentRef string,
	taskRef string,
) orquestaruntimecodexdelivery.CodexReceiptDescriptorV0 {
	return orquestaruntimecodexdelivery.CodexReceiptDescriptorV0{
		DescriptorRef: "receipt-ref-" + agentRef,
		RunID:         runRef,
		AgentRef:      agentRef,
		Spec: orquestaruntime.ExternalAgentLaunchSpecV0{
			RequestID: agentRef,
			AgentPacket: orquestaruntime.AgentStartPacketV0{
				RequestID: agentRef,
				Phase:     string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
				Task: orquestaruntime.AgentStartTaskV0{
					TaskRef: taskRef,
				},
			},
		},
	}
}
