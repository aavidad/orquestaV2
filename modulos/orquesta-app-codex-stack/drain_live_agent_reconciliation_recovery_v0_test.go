package orquestaappcodexstack

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirector "orquesta/modulos/orquesta-director"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func TestApplyStoppedAgentReconciliationV0ReproyectaAssessmentDurableStale(t *testing.T) {
	ctx := context.Background()
	runRef := "run-ref-live-reconciliation-durable-assessment-001"
	agentRef := "agent-ref-launch-outbox-recovery-001"
	reportRef := "agent-progress-report-ref-live-agent-reconciliation-partial-001"
	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	eventSink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	run := codexStackLaunchOutboxRecoveryRunV0(t, ctx, runStore, eventSink, runRef)
	request := DrainRunRequestV0{
		RunRef:        runRef,
		CorrelationID: "corr-live-reconciliation-durable-assessment-001",
		OccurredAt:    "2026-06-28T21:00:00Z",
	}
	originalObservation := stoppedAgentReconciliationObservationForTestV0(
		run,
		agentRef,
		reportRef,
		[]string{"evidence-ref-provider-quota-exhausted"},
	)
	originalInput := stoppedAgentSupervisionInputV0(request, run, originalObservation)
	original, err := orquestadirector.BuildAgentProgressSupervisionV0(originalInput)
	if err != nil {
		t.Fatalf("BuildAgentProgressSupervisionV0 original: %v", err)
	}
	result, err := orquestacoreworkflow.HandleCommandV0(run, original.AssessCommand)
	if err != nil {
		t.Fatalf("HandleCommandV0 original assess: %v", err)
	}
	if len(result.Events) == 0 || result.Events[0].EventType != orquestacoreworkflow.OrchestrationEventAgentWorkAssessedV0 {
		t.Fatalf("events originales=%+v", result.Events)
	}
	if err := eventSink.AppendRunEventsV0(ctx, runRef, result.Events[:1]); err != nil {
		t.Fatalf("AppendRunEventsV0 partial assessment: %v", err)
	}
	stale, err := runStore.LoadRunV0(ctx, runRef)
	if err != nil {
		t.Fatalf("LoadRunV0 stale: %v", err)
	}
	assessmentBaseRef := liveAgentReconciliationAssessmentBaseRefV0(originalInput.AssessmentRef)
	if codexStackRefsContainPartV0(stale.AgentAssessments, assessmentBaseRef) {
		t.Fatalf("la preparacion del test no debe proyectar assessment: %v", stale.AgentAssessments)
	}

	stack := StackV0{
		Stores: StoresV0{
			RunStore:  runStore,
			EventSink: eventSink,
		},
		Ports: orquestaappdirectorservice.StartAppDirectorPortsV0{
			RunStore:    runStore,
			EventSink:   eventSink,
			EventReader: eventSink,
		},
	}
	nextObservation := stoppedAgentReconciliationObservationForTestV0(
		run,
		agentRef,
		reportRef,
		[]string{
			"evidence-ref-provider-quota-exhausted",
			"evidence-ref-artifact-without-ack",
		},
	)
	if err := stack.applyStoppedAgentReconciliationV0(ctx, request, run, nextObservation); err != nil {
		t.Fatalf("applyStoppedAgentReconciliationV0 no debe propagar idempotency conflict: %v", err)
	}
	loaded, err := runStore.LoadRunV0(ctx, runRef)
	if err != nil {
		t.Fatalf("LoadRunV0 final: %v", err)
	}
	if !codexStackRefsContainPartV0(loaded.AgentAssessments, assessmentBaseRef) {
		t.Fatalf("assessment durable no reproyectada: %v", loaded.AgentAssessments)
	}
	assessments := agentWorkAssessedEventsForTestV0(eventSink.EventsV0(), assessmentBaseRef)
	if len(assessments) != 1 {
		t.Fatalf("assessment durable duplicada: %d events=%+v", len(assessments), eventSink.EventsV0())
	}
	if !agentWorkAssessedEventHasEvidenceForTestV0(assessments[0], "evidence-ref-provider-quota-exhausted") ||
		agentWorkAssessedEventHasEvidenceForTestV0(assessments[0], "evidence-ref-artifact-without-ack") {
		t.Fatalf("la reproyeccion debe conservar payload durable original: %s", string(assessments[0].Payload))
	}
}

func TestStoppedAgentSupervisionInputV0ClaveCambiaConPayloadYRepiteIgual(t *testing.T) {
	run := orquestacoreworkflow.OrchestrationRunV0{
		RunID:        "run-ref-live-reconciliation-key-001",
		CurrentPhase: orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
	}
	request := DrainRunRequestV0{
		RunRef:        run.RunID,
		CorrelationID: "corr-live-reconciliation-key-001",
		OccurredAt:    "2026-06-28T22:00:00Z",
	}
	const agentRef = "agent-ref-live-reconciliation-key-001"
	const reportRef = "agent-progress-report-ref-live-reconciliation-key-000052"
	first := stoppedAgentSupervisionInputV0(request, run, stoppedAgentReconciliationObservationForTestV0(
		run,
		agentRef,
		reportRef,
		[]string{"evidence-ref-provider-quota-exhausted"},
	))
	repeated := stoppedAgentSupervisionInputV0(request, run, stoppedAgentReconciliationObservationForTestV0(
		run,
		agentRef,
		reportRef,
		[]string{"evidence-ref-provider-quota-exhausted"},
	))
	repeatedNewReport := stoppedAgentSupervisionInputV0(request, run, stoppedAgentReconciliationObservationForTestV0(
		run,
		agentRef,
		"agent-progress-report-ref-live-reconciliation-key-000053",
		[]string{"evidence-ref-provider-quota-exhausted"},
	))
	changed := stoppedAgentSupervisionInputV0(request, run, stoppedAgentReconciliationObservationForTestV0(
		run,
		agentRef,
		reportRef,
		[]string{"evidence-ref-provider-quota-exhausted", "evidence-ref-artifact-without-ack"},
	))
	if first.CommandMeta.IdempotencyKey == "" ||
		first.CommandMeta.IdempotencyKey != repeated.CommandMeta.IdempotencyKey ||
		first.AssessmentRef != repeated.AssessmentRef {
		t.Fatalf("payload identico debe conservar ids: first=%+v repeated=%+v", first, repeated)
	}
	if first.CommandMeta.IdempotencyKey != repeatedNewReport.CommandMeta.IdempotencyKey ||
		first.AssessmentRef != repeatedNewReport.AssessmentRef ||
		first.QuestionID != repeatedNewReport.QuestionID {
		t.Fatalf("mismo estado con report_id nuevo debe conservar ids: first=%+v repeated_new_report=%+v", first, repeatedNewReport)
	}
	if first.CommandMeta.IdempotencyKey == changed.CommandMeta.IdempotencyKey ||
		first.AssessmentRef == changed.AssessmentRef {
		t.Fatalf("payload distinto debe cambiar ids: first=%+v changed=%+v", first, changed)
	}
	if liveAgentReconciliationAssessmentBaseRefV0(first.AssessmentRef) !=
		liveAgentReconciliationAssessmentBaseRefV0(changed.AssessmentRef) {
		t.Fatalf("payload distinto debe conservar base semantica: first=%s changed=%s", first.AssessmentRef, changed.AssessmentRef)
	}
}

func stoppedAgentReconciliationObservationForTestV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	agentRef string,
	reportRef string,
	evidenceRefs []string,
) orquestacionnucleoapp.AgentProgressObservationV0 {
	report := orquestaruntime.AgentProgressReportV0{
		ReportID:         strings.TrimSpace(reportRef),
		RunID:            strings.TrimSpace(run.RunID),
		AgentRequestID:   strings.TrimSpace(agentRef),
		Status:           orquestaruntime.AgentStoppedV0,
		BudgetStatus:     orquestaruntime.AgentProgressBudgetCapacityLimitedV0,
		DecisionRequired: true,
		Summary:          "Proceso parado por capacidad externa durante reconciliacion.",
		EvidenceRefs:     compactStringsV0(evidenceRefs),
	}
	return orquestacionnucleoapp.AgentProgressObservationV0{
		CandidateRef:     "progress-candidate-ref-" + strings.TrimSpace(reportRef),
		Report:           report,
		PhaseID:          string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
		TaskRef:          "task-ref-launch-outbox-recovery-001",
		DecisionRequired: true,
		EvidenceRefs:     append([]string(nil), report.EvidenceRefs...),
	}
}

func agentWorkAssessedEventsForTestV0(
	events []orquestacoreworkflow.OrchestrationEventV0,
	assessmentBaseRef string,
) []orquestacoreworkflow.OrchestrationEventV0 {
	result := []orquestacoreworkflow.OrchestrationEventV0{}
	for _, event := range events {
		if strings.TrimSpace(event.EventType) != orquestacoreworkflow.OrchestrationEventAgentWorkAssessedV0 {
			continue
		}
		var payload orquestacoreworkflow.AgentWorkAssessedPayloadV0
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			continue
		}
		if liveAgentReconciliationAssessmentBaseRefV0(payload.AssessmentRef) == strings.TrimSpace(assessmentBaseRef) {
			result = append(result, event)
		}
	}
	return result
}

func agentWorkAssessedEventHasEvidenceForTestV0(
	event orquestacoreworkflow.OrchestrationEventV0,
	evidenceRef string,
) bool {
	var payload orquestacoreworkflow.AgentWorkAssessedPayloadV0
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return false
	}
	for _, ref := range payload.EvidenceRefs {
		if strings.TrimSpace(ref) == strings.TrimSpace(evidenceRef) {
			return true
		}
	}
	return false
}
