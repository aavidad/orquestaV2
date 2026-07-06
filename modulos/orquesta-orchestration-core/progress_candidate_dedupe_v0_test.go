package orquestacionnucleoapp

import (
	"context"
	"testing"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

// Contrato TAREA-D4/P3: dos ticks sin cambio causal no emiten una segunda
// pareja assessment+question; el candidato se salta mientras la accion
// anterior siga pendiente en la proyeccion del run.
func TestProgressSupervisionCandidateProviderV0NoRepiteParejaPendienteSinCambioCausalV0(t *testing.T) {
	runRef := "run-nucleo-progress-dedupe-001"
	agentRef := "agent-ref-progress-dedupe-001"
	observation := AgentProgressObservationV0{
		Report:  progressObservationDecisionReportV0(runRef, agentRef, orquestaruntime.AgentStalledV0),
		TaskRef: "task-ref-progress-dedupe-001",
	}
	provider := ProgressSupervisionCandidateProviderV0{
		ProgressSource: staticAgentProgressObservationSourceV0{Observations: []AgentProgressObservationV0{observation}},
		RequestedBy:    "orquestacion-nucleo",
	}
	run := mustActiveProgrammingRunV0(t, runRef)

	first, err := provider.BuildSchedulerCandidatesV0(context.Background(), SchedulerCandidateRequestV0{
		Run:        run,
		OccurredAt: "2026-05-09T12:20:00Z",
	})
	if err != nil {
		t.Fatalf("primer tick: %v", err)
	}
	if len(first.ProgressSupervisionCandidates) != 1 {
		t.Fatalf("primer tick debe producir candidato: %+v", first.ProgressSupervisionCandidates)
	}
	supervision := first.ProgressSupervisionCandidates[0].SupervisionInput
	if supervision.AssessmentRef == "" || supervision.QuestionID == "" {
		t.Fatalf("refs estables vacias: %+v", supervision)
	}

	// Segundo tick: el run ya refleja la evaluacion y la pregunta pendiente.
	run.AgentAssessments = append(
		run.AgentAssessments,
		supervision.AssessmentRef+"#phase:programacion#agent:"+agentRef+"#verdict:needs_revision#action:ask_director#severity:high",
	)
	run.DirectorQuestions = append(run.DirectorQuestions, supervision.QuestionID)

	second, err := provider.BuildSchedulerCandidatesV0(context.Background(), SchedulerCandidateRequestV0{
		Run:        run,
		OccurredAt: "2026-05-09T12:20:05Z",
	})
	if err != nil {
		t.Fatalf("segundo tick: %v", err)
	}
	if len(second.ProgressSupervisionCandidates) != 0 {
		t.Fatalf("segundo tick sin cambio causal no debe re-emitir la pareja: %+v", second.ProgressSupervisionCandidates)
	}

	// Con la pregunta respondida, la supervision vuelve a ser elegible.
	run.DirectorAnsweredQuestions = append(run.DirectorAnsweredQuestions, supervision.QuestionID)
	third, err := provider.BuildSchedulerCandidatesV0(context.Background(), SchedulerCandidateRequestV0{
		Run:        run,
		OccurredAt: "2026-05-09T12:20:10Z",
	})
	if err != nil {
		t.Fatalf("tercer tick: %v", err)
	}
	if len(third.ProgressSupervisionCandidates) != 1 {
		t.Fatalf("con pregunta respondida debe volver a supervisar: %+v", third.ProgressSupervisionCandidates)
	}
}

func TestProgressSupervisionCandidateProviderV0NoRepiteLoopConStopYaPedidoV0(t *testing.T) {
	runRef := "run-nucleo-progress-dedupe-loop-001"
	agentRef := "agent-ref-progress-dedupe-loop-001"
	observation := AgentProgressObservationV0{
		Report:  progressObservationDecisionReportV0(runRef, agentRef, orquestaruntime.AgentLoopDetectedV0),
		TaskRef: "task-ref-progress-dedupe-loop-001",
	}
	provider := ProgressSupervisionCandidateProviderV0{
		ProgressSource: staticAgentProgressObservationSourceV0{Observations: []AgentProgressObservationV0{observation}},
		RequestedBy:    "orquestacion-nucleo",
	}
	run := mustActiveProgrammingRunV0(t, runRef)

	first, err := provider.BuildSchedulerCandidatesV0(context.Background(), SchedulerCandidateRequestV0{
		Run:        run,
		OccurredAt: "2026-05-09T12:21:00Z",
	})
	if err != nil {
		t.Fatalf("primer tick: %v", err)
	}
	if len(first.ProgressSupervisionCandidates) != 1 {
		t.Fatalf("primer tick debe producir candidato: %+v", first.ProgressSupervisionCandidates)
	}
	supervision := first.ProgressSupervisionCandidates[0].SupervisionInput

	// El run ya refleja la evaluacion y el stop pedido para ese agente: cada
	// tick posterior con el mismo loop no debe generar otra pareja.
	run.AgentAssessments = append(
		run.AgentAssessments,
		supervision.AssessmentRef+"#phase:programacion#agent:"+agentRef+"#verdict:loop_detected#action:stop_agent#severity:critical",
	)
	run.AgentStopRequests = append(run.AgentStopRequests, agentRef+"#reason:loop_detected")

	second, err := provider.BuildSchedulerCandidatesV0(context.Background(), SchedulerCandidateRequestV0{
		Run:        run,
		OccurredAt: "2026-05-09T12:21:05Z",
	})
	if err != nil {
		t.Fatalf("segundo tick: %v", err)
	}
	if len(second.ProgressSupervisionCandidates) != 0 {
		t.Fatalf("loop con stop ya pedido no debe re-emitir pareja por tick: %+v", second.ProgressSupervisionCandidates)
	}
}
