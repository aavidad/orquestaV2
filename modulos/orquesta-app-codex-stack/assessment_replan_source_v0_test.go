package orquestaappcodexstack

import (
	"context"
	"strings"
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

func TestAssessmentReplanSourceV0NoReplanificaAssessmentDeTareaCerrada(t *testing.T) {
	runRef := "run-ref-assessment-replan-stack-covered-closed-001"
	oldAgentRef := "agent-ref-assessment-covered-closed-001"
	taskRef := "task-ref-assessment-covered-closed-001"
	source := AssessmentReplanSourceV0{
		Store: orquestaruntimecodexdelivery.NewInMemoryCodexReceiptDescriptorStoreV0(
			assessmentReplanDescriptorForTestV0(runRef, oldAgentRef, taskRef),
		),
	}
	request := assessmentReplanRequestForTestV0(runRef, oldAgentRef, taskRef, false)
	request.Run.ClosedTasks = []string{taskRef}

	plans, err := source.BuildAgentAssessmentReplanPlansV0(context.Background(), request)
	if err != nil {
		t.Fatalf("BuildAgentAssessmentReplanPlansV0 covered closed: %v", err)
	}
	if len(plans) != 0 {
		t.Fatalf("assessment de tarea cerrada no debe abrir replan: %+v", plans)
	}
}

func TestAssessmentReplanSourceV0NoReplanificaAssessmentConReviewAceptada(t *testing.T) {
	runRef := "run-ref-assessment-replan-stack-covered-review-001"
	oldAgentRef := "agent-ref-assessment-covered-review-001"
	taskRef := "task-ref-assessment-covered-review-001"
	deliveryRef := "delivery-ref-assessment-covered-review-001"
	descriptor := assessmentReplanDescriptorForTestV0(runRef, oldAgentRef, taskRef)
	descriptor.Spec.AgentPacket.DeliveryRefs.AckRef = deliveryRef
	source := AssessmentReplanSourceV0{
		Store: orquestaruntimecodexdelivery.NewInMemoryCodexReceiptDescriptorStoreV0(descriptor),
	}
	request := assessmentReplanRequestForTestV0(runRef, oldAgentRef, taskRef, false)
	request.Run.Deliveries = []string{deliveryRef}
	request.Run.AcceptedReviews = []string{"accepted-review-ref-" + deliveryRef}

	plans, err := source.BuildAgentAssessmentReplanPlansV0(context.Background(), request)
	if err != nil {
		t.Fatalf("BuildAgentAssessmentReplanPlansV0 covered review: %v", err)
	}
	if len(plans) != 0 {
		t.Fatalf("assessment con review aceptada no debe abrir replan: %+v", plans)
	}
}

func TestAssessmentReplanSourceV0NoReplanificaAssessmentConDeliveryRegistrada(t *testing.T) {
	runRef := "run-ref-assessment-replan-stack-covered-delivery-001"
	oldAgentRef := "agent-ref-assessment-covered-delivery-001"
	taskRef := "task-ref-assessment-covered-delivery-001"
	deliveryRef := "delivery-ref-assessment-covered-delivery-001"
	descriptor := assessmentReplanDescriptorForTestV0(runRef, oldAgentRef, taskRef)
	descriptor.Spec.AgentPacket.DeliveryRefs.AckRef = deliveryRef
	source := AssessmentReplanSourceV0{
		Store: orquestaruntimecodexdelivery.NewInMemoryCodexReceiptDescriptorStoreV0(descriptor),
	}
	request := assessmentReplanRequestForTestV0(runRef, oldAgentRef, taskRef, false)
	request.Run.Deliveries = []string{deliveryRef}
	request.Run.DeliveredAgents = []string{oldAgentRef}

	plans, err := source.BuildAgentAssessmentReplanPlansV0(context.Background(), request)
	if err != nil {
		t.Fatalf("BuildAgentAssessmentReplanPlansV0 covered delivery: %v", err)
	}
	if len(plans) != 0 {
		t.Fatalf("assessment con delivery registrada debe esperar review/rework, no replacement: %+v", plans)
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

func TestStackRunHasRecoverableTerminalAssessmentV0IgnoraAssessmentCubiertoPorReviewAceptada(t *testing.T) {
	agentRef := "agent-ref-stack-assessment-covered-review-001"
	taskRef := "task-ref-stack-assessment-covered-review-001"
	deliveryRef := "delivery-ref-stack-assessment-covered-review-001"
	run := orquestacoreworkflow.OrchestrationRunV0{
		RunID:        "run-ref-stack-assessment-covered-review-001",
		CurrentPhase: orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		Tasks:        []string{taskRef},
		AgentAssessments: []string{
			orquestacoreworkflow.AgentAssessmentProjectionRefV0(orquestacoreworkflow.AgentWorkAssessedPayloadV0{
				AssessmentRef:  "assessment-ref-" + agentRef,
				PhaseID:        string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
				AgentRequestID: agentRef,
				TaskRef:        taskRef,
				DeliveryRef:    deliveryRef,
				Verdict:        orquestacoreworkflow.AgentAssessmentVerdictLoopDetectedV0,
				Action:         orquestacoreworkflow.AgentAssessmentActionStopAgentV0,
				Severity:       orquestacoreworkflow.AgentAssessmentSeverityCriticalV0,
			}),
		},
		DeliveredAgents: []string{agentRef},
		Deliveries:      []string{deliveryRef},
		AcceptedReviews: []string{"accepted-review-ref-" + deliveryRef},
	}
	if stackRunHasRecoverableTerminalAssessmentV0(run) {
		t.Fatalf("assessment cubierto por review aceptada no debe disparar recovery")
	}
	run.AcceptedReviews = nil
	if stackRunHasRecoverableTerminalAssessmentV0(run) {
		t.Fatalf("assessment con delivery registrada debe esperar review/rework aunque falte review aceptada")
	}
}

func TestStackRunHasRecoverableTerminalAssessmentV0IgnoraAssessmentDeTareaCerrada(t *testing.T) {
	agentRef := "agent-ref-stack-assessment-covered-task-001"
	closedTaskRef := "task-ref-stack-assessment-covered-task-001"
	openTaskRef := "task-ref-stack-assessment-open-task-001"
	run := orquestacoreworkflow.OrchestrationRunV0{
		RunID:        "run-ref-stack-assessment-covered-task-001",
		CurrentPhase: orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		Tasks:        []string{closedTaskRef, openTaskRef},
		ClosedTasks:  []string{closedTaskRef},
		AgentAssessments: []string{
			orquestacoreworkflow.AgentAssessmentProjectionRefV0(orquestacoreworkflow.AgentWorkAssessedPayloadV0{
				AssessmentRef:  "assessment-ref-" + agentRef,
				PhaseID:        string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
				AgentRequestID: agentRef,
				TaskRef:        closedTaskRef,
				Verdict:        orquestacoreworkflow.AgentAssessmentVerdictLoopDetectedV0,
				Action:         orquestacoreworkflow.AgentAssessmentActionStopAgentV0,
				Severity:       orquestacoreworkflow.AgentAssessmentSeverityCriticalV0,
			}),
		},
		StoppedAgents:          []string{agentRef},
		ConfirmedStoppedAgents: []string{agentRef},
	}
	if stackRunHasRecoverableTerminalAssessmentV0(run) {
		t.Fatalf("assessment de tarea cerrada no debe disparar recovery aunque haya otras tareas abiertas")
	}
}

func TestAssessmentReplanSourceV0NoReplanificaScannerSinAssessmentTerminal(t *testing.T) {
	runRef := "request-ref-autoprogramming-backlog-scanner-test-001"
	oldAgentRef := "agent-ref-assessment-backlog-scanner-001"
	taskRef := "task-autoprogramming-backlog-scanner-001"
	source := AssessmentReplanSourceV0{
		Store: orquestaruntimecodexdelivery.NewInMemoryCodexReceiptDescriptorStoreV0(
			assessmentReplanDescriptorForTestV0(runRef, oldAgentRef, taskRef),
		),
	}
	request := assessmentReplanRequestForTestV0(runRef, oldAgentRef, taskRef, false)
	request.Run.AgentAssessments = nil

	plans, err := source.BuildAgentAssessmentReplanPlansV0(context.Background(), request)
	if err != nil {
		t.Fatalf("BuildAgentAssessmentReplanPlansV0 scanner: %v", err)
	}
	if len(plans) != 0 {
		t.Fatalf("scanner backlog sin assessment terminal no debe abrir replan: %+v", plans)
	}
}

func TestAssessmentReplanSourceV0ReemplazaScannerPerdidoConAskDirector(t *testing.T) {
	runRef := "request-ref-autoprogramming-backlog-scanner-test-ask-director-001"
	oldAgentRef := "agent-ref-assessment-backlog-scanner-ask-director-001"
	taskRef := "task-autoprogramming-backlog-scanner-ask-director-001"
	source := AssessmentReplanSourceV0{
		Store: orquestaruntimecodexdelivery.NewInMemoryCodexReceiptDescriptorStoreV0(
			assessmentReplanDescriptorForTestV0(runRef, oldAgentRef, taskRef),
		),
	}
	request := assessmentReplanRequestForTestV0(runRef, oldAgentRef, taskRef, false)
	request.Run.StoppedAgents = nil
	request.Run.ConfirmedStoppedAgents = nil
	request.Run.LostAgents = []string{oldAgentRef}
	request.Run.AgentAssessments = []string{
		orquestacoreworkflow.AgentAssessmentProjectionRefV0(orquestacoreworkflow.AgentWorkAssessedPayloadV0{
			AssessmentRef:  "assessment-ref-" + oldAgentRef,
			PhaseID:        string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			AgentRequestID: oldAgentRef,
			TaskRef:        taskRef,
			Verdict:        orquestacoreworkflow.AgentAssessmentVerdictNeedsRevisionV0,
			Action:         orquestacoreworkflow.AgentAssessmentActionAskDirectorV0,
			Severity:       orquestacoreworkflow.AgentAssessmentSeverityHighV0,
		}),
	}

	plans, err := source.BuildAgentAssessmentReplanPlansV0(context.Background(), request)
	if err != nil {
		t.Fatalf("BuildAgentAssessmentReplanPlansV0 scanner ask_director: %v", err)
	}
	if len(plans) != 1 ||
		plans[0].TaskRef != taskRef ||
		plans[0].Assessment.AgentRequestID != oldAgentRef ||
		plans[0].RequestedAction != "replace_agent" {
		t.Fatalf("scanner backlog perdido debe abrir replacement: %+v", plans)
	}
}

func TestAssessmentReplanSourceV0ConvierteAskDirectorPerdidoEnReemplazoDelDirector(t *testing.T) {
	runRef := "run-ref-assessment-replan-stack-lost-ask-director-001"
	oldAgentRef := "agent-ref-assessment-lost-ask-director-001"
	taskRef := "task-ref-assessment-stack-lost-ask-director-001"
	source := AssessmentReplanSourceV0{
		Store: orquestaruntimecodexdelivery.NewInMemoryCodexReceiptDescriptorStoreV0(
			assessmentReplanDescriptorForTestV0(runRef, oldAgentRef, taskRef),
		),
	}
	request := assessmentReplanRequestForTestV0(runRef, oldAgentRef, taskRef, false)
	request.Run.StoppedAgents = nil
	request.Run.ConfirmedStoppedAgents = nil
	request.Run.LostAgents = []string{oldAgentRef}
	request.Run.AgentAssessments = []string{
		orquestacoreworkflow.AgentAssessmentProjectionRefV0(orquestacoreworkflow.AgentWorkAssessedPayloadV0{
			AssessmentRef:  "assessment-ref-" + oldAgentRef,
			PhaseID:        string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			AgentRequestID: oldAgentRef,
			TaskRef:        taskRef,
			Verdict:        orquestacoreworkflow.AgentAssessmentVerdictNeedsRevisionV0,
			Action:         orquestacoreworkflow.AgentAssessmentActionAskDirectorV0,
			Severity:       orquestacoreworkflow.AgentAssessmentSeverityHighV0,
		}),
	}

	plans, err := source.BuildAgentAssessmentReplanPlansV0(context.Background(), request)
	if err != nil {
		t.Fatalf("BuildAgentAssessmentReplanPlansV0 lost ask director: %v", err)
	}
	if len(plans) != 1 ||
		plans[0].Assessment.AgentRequestID != oldAgentRef ||
		plans[0].RequestedAction != "replace_agent" {
		t.Fatalf("ask_director perdido debe generar replacement del Director: %+v", plans)
	}
}

func TestAssessmentReplanSourceV0ConvierteAskDirectorConfirmadoParadoEnReemplazo(t *testing.T) {
	runRef := "run-ref-assessment-replan-stack-confirmed-ask-director-001"
	oldAgentRef := "agent-ref-assessment-confirmed-ask-director-001"
	taskRef := "task-ref-assessment-stack-confirmed-ask-director-001"
	source := AssessmentReplanSourceV0{
		Store: orquestaruntimecodexdelivery.NewInMemoryCodexReceiptDescriptorStoreV0(
			assessmentReplanDescriptorForTestV0(runRef, oldAgentRef, taskRef),
		),
	}
	request := assessmentReplanRequestForTestV0(runRef, oldAgentRef, taskRef, false)
	request.Run.LostAgents = nil
	request.Run.StoppedAgents = []string{oldAgentRef}
	request.Run.ConfirmedStoppedAgents = []string{oldAgentRef}
	request.Run.AgentAssessments = []string{
		orquestacoreworkflow.AgentAssessmentProjectionRefV0(orquestacoreworkflow.AgentWorkAssessedPayloadV0{
			AssessmentRef:  "assessment-ref-" + oldAgentRef,
			PhaseID:        string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			AgentRequestID: oldAgentRef,
			TaskRef:        taskRef,
			Verdict:        orquestacoreworkflow.AgentAssessmentVerdictNeedsRevisionV0,
			Action:         orquestacoreworkflow.AgentAssessmentActionAskDirectorV0,
			Severity:       orquestacoreworkflow.AgentAssessmentSeverityHighV0,
		}),
	}

	plans, err := source.BuildAgentAssessmentReplanPlansV0(context.Background(), request)
	if err != nil {
		t.Fatalf("BuildAgentAssessmentReplanPlansV0 confirmed ask director: %v", err)
	}
	if len(plans) != 1 ||
		plans[0].Assessment.AgentRequestID != oldAgentRef ||
		plans[0].RequestedAction != "replace_agent" {
		t.Fatalf("ask_director confirmado parado debe generar replacement: %+v", plans)
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

func TestAssessmentReplanSourceV0NoEncadenaReemplazosParaMismaTarea(t *testing.T) {
	runRef := "run-ref-assessment-replan-stack-chain-001"
	oldAgentRef := "agent-ref-assessment-chain-old-001"
	replacementAgentRef := "agent-ref-assessment-chain-replacement-001"
	taskRef := "task-ref-assessment-stack-chain-001"
	source := AssessmentReplanSourceV0{
		Store: orquestaruntimecodexdelivery.NewInMemoryCodexReceiptDescriptorStoreV0(
			assessmentReplanDescriptorForTestV0(runRef, oldAgentRef, taskRef),
			assessmentReplanDescriptorForTestV0(runRef, replacementAgentRef, taskRef),
		),
	}
	request := assessmentReplanRequestForTestV0(runRef, replacementAgentRef, taskRef, false)
	request.Run.ReplanDecisions = []string{
		"replan-ref-chain-001#source:assessment-ref-chain-old#task:" + taskRef +
			"#action:" + string(orquestacoreworkflow.ReplanDecisionActionReplaceAgentV0) +
			"#followups:" + replacementAgentRef,
	}
	request.Run.Agents = []string{replacementAgentRef}
	request.Run.StartedAgents = []string{replacementAgentRef}
	request.Run.StoppedAgents = nil
	request.Run.ConfirmedStoppedAgents = nil

	plans, err := source.BuildAgentAssessmentReplanPlansV0(context.Background(), request)
	if err != nil {
		t.Fatalf("BuildAgentAssessmentReplanPlansV0 chain: %v", err)
	}
	if len(plans) != 0 {
		t.Fatalf("replan encadenado para misma tarea=%+v", plans)
	}
}

func TestAssessmentReplanSourceV0ReintentaSiFollowupFalloTerminal(t *testing.T) {
	runRef := "run-ref-assessment-replan-stack-chain-failed-001"
	oldAgentRef := "agent-ref-assessment-chain-failed-old-001"
	replacementAgentRef := "agent-ref-assessment-chain-failed-replacement-001"
	taskRef := "task-ref-assessment-stack-chain-failed-001"
	source := AssessmentReplanSourceV0{
		Store: orquestaruntimecodexdelivery.NewInMemoryCodexReceiptDescriptorStoreV0(
			assessmentReplanDescriptorForTestV0(runRef, oldAgentRef, taskRef),
			assessmentReplanDescriptorForTestV0(runRef, replacementAgentRef, taskRef),
		),
	}
	request := assessmentReplanRequestForTestV0(runRef, replacementAgentRef, taskRef, false)
	request.Run.ReplanDecisions = []string{
		"replan-ref-chain-failed-001#source:assessment-ref-chain-old#task:" + taskRef +
			"#action:" + string(orquestacoreworkflow.ReplanDecisionActionReplaceAgentV0) +
			"#followups:" + replacementAgentRef,
	}
	request.Run.Agents = []string{replacementAgentRef}
	request.Run.StoppedAgents = []string{replacementAgentRef}
	request.Run.ConfirmedStoppedAgents = []string{replacementAgentRef}

	plans, err := source.BuildAgentAssessmentReplanPlansV0(context.Background(), request)
	if err != nil {
		t.Fatalf("BuildAgentAssessmentReplanPlansV0 chain failed: %v", err)
	}
	if len(plans) != 1 ||
		plans[0].TaskRef != taskRef ||
		plans[0].AgentRequestID == replacementAgentRef {
		t.Fatalf("debe reintentar followup terminal con nuevo replacement, plans=%+v", plans)
	}
}

func TestAssessmentReplanSourceV0CortaBucleTrasTresFollowupsFallidos(t *testing.T) {
	runRef := "run-ref-assessment-replan-stack-chain-failed-max-001"
	taskRef := "task-ref-assessment-stack-chain-failed-max-001"
	failedRefs := []string{
		"agent-ref-assessment-chain-failed-max-replacement-001",
		"agent-ref-assessment-chain-failed-max-replacement-002",
		"agent-ref-assessment-chain-failed-max-replacement-003",
	}
	source := AssessmentReplanSourceV0{
		Store: orquestaruntimecodexdelivery.NewInMemoryCodexReceiptDescriptorStoreV0(
			assessmentReplanDescriptorForTestV0(runRef, failedRefs[2], taskRef),
		),
	}
	request := assessmentReplanRequestForTestV0(runRef, failedRefs[2], taskRef, false)
	request.Run.ReplanDecisions = []string{
		"replan-ref-chain-failed-max-001#source:assessment-ref-chain-old#task:" + taskRef +
			"#action:" + string(orquestacoreworkflow.ReplanDecisionActionReplaceAgentV0) +
			"#followups:" + failedRefs[0],
		"replan-ref-chain-failed-max-002#source:assessment-ref-chain-old#task:" + taskRef +
			"#action:" + string(orquestacoreworkflow.ReplanDecisionActionReplaceAgentV0) +
			"#followups:" + failedRefs[1],
		"replan-ref-chain-failed-max-003#source:assessment-ref-chain-old#task:" + taskRef +
			"#action:" + string(orquestacoreworkflow.ReplanDecisionActionReplaceAgentV0) +
			"#followups:" + failedRefs[2],
	}
	request.Run.Agents = append([]string(nil), failedRefs...)
	request.Run.StoppedAgents = append([]string(nil), failedRefs...)
	request.Run.ConfirmedStoppedAgents = append([]string(nil), failedRefs...)

	plans, err := source.BuildAgentAssessmentReplanPlansV0(context.Background(), request)
	if err != nil {
		t.Fatalf("BuildAgentAssessmentReplanPlansV0 chain failed max: %v", err)
	}
	if len(plans) != 0 {
		t.Fatalf("debe cortar bucle tras tres followups terminales, plans=%+v", plans)
	}
}

func TestAssessmentReplanSourceV0RecuperaReplanConFollowupNoMaterializado(t *testing.T) {
	runRef := "run-ref-assessment-replan-stack-recover-001"
	oldAgentRef := "agent-ref-assessment-recover-old-001"
	replacementAgentRef := "agent-ref-assessment-recover-replacement-001"
	taskRef := "task-ref-assessment-stack-recover-001"
	source := AssessmentReplanSourceV0{
		Store: orquestaruntimecodexdelivery.NewInMemoryCodexReceiptDescriptorStoreV0(
			assessmentReplanDescriptorForTestV0(runRef, oldAgentRef, taskRef),
		),
	}
	request := assessmentReplanRequestForTestV0(runRef, oldAgentRef, taskRef, false)
	request.Run.ReplanDecisions = []string{
		"replan-ref-recover-001#source:assessment-ref-recover-old#task:" + taskRef +
			"#action:" + string(orquestacoreworkflow.ReplanDecisionActionReplaceAgentV0) +
			"#followups:" + replacementAgentRef + "+capacity-ref-recover-001",
	}

	plans, err := source.BuildAgentAssessmentReplanPlansV0(context.Background(), request)
	if err != nil {
		t.Fatalf("BuildAgentAssessmentReplanPlansV0 recover: %v", err)
	}
	if len(plans) != 1 {
		t.Fatalf("debe recuperar replacement no materializado, plans=%+v", plans)
	}
}

func TestAssessmentReplanSourceV0GeneraRefsCompactasParaAssessmentAnidado(t *testing.T) {
	runRef := "run-ref-assessment-replan-stack-compact-001"
	oldAgentRef := "agent-ref-assessment-assessment-ref-agent-progress-report-ref-agent-ref-assessment-assessment-ref-agent-progress-report-ref-agent-ref-task-autoprogramming-d6f0b05f2e4d-g01-000088-000016"
	taskRef := "task-autoprogramming-d6f0b05f2e4d-g01"
	source := AssessmentReplanSourceV0{
		Store: orquestaruntimecodexdelivery.NewInMemoryCodexReceiptDescriptorStoreV0(
			assessmentReplanDescriptorForTestV0(runRef, oldAgentRef, taskRef),
		),
	}
	request := assessmentReplanRequestForTestV0(runRef, oldAgentRef, taskRef, false)
	request.Run.AgentAssessments = []string{
		orquestacoreworkflow.AgentAssessmentProjectionRefV0(orquestacoreworkflow.AgentWorkAssessedPayloadV0{
			AssessmentRef:  "assessment-ref-agent-progress-report-ref-" + oldAgentRef,
			PhaseID:        string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			AgentRequestID: oldAgentRef,
			TaskRef:        taskRef,
			Verdict:        orquestacoreworkflow.AgentAssessmentVerdictLoopDetectedV0,
			Action:         orquestacoreworkflow.AgentAssessmentActionStopAgentV0,
			Severity:       orquestacoreworkflow.AgentAssessmentSeverityCriticalV0,
		}),
	}

	plans, err := source.BuildAgentAssessmentReplanPlansV0(context.Background(), request)
	if err != nil {
		t.Fatalf("BuildAgentAssessmentReplanPlansV0 compact: %v", err)
	}
	if len(plans) != 1 {
		t.Fatalf("plans=%+v", plans)
	}
	if got := plans[0].AgentRequestID; len(got) > 120 ||
		got == "" ||
		strings.Contains(got, oldAgentRef) {
		t.Fatalf("agent_request_id no compacto: %q len=%d", got, len(got))
	}
	again, err := source.BuildAgentAssessmentReplanPlansV0(context.Background(), request)
	if err != nil || len(again) != 1 || again[0].AgentRequestID != plans[0].AgentRequestID {
		t.Fatalf("agent_request_id no determinista: first=%+v again=%+v err=%v", plans, again, err)
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
		AckPath:       "ack-path-" + agentRef,
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
