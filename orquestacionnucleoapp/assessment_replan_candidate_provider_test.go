package orquestacionnucleoapp

import (
	"context"
	"errors"
	"testing"

	orquestacorereplanner "orquesta/modulos/orquesta-core-replanner"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func TestAgentAssessmentReplanCandidateProviderV0RejectsUnsupportedMaterializationAction(t *testing.T) {
	runRef := "run-assessment-replan-plan-001"
	run := mustActiveProgrammingRunV0(t, runRef)
	run.Tasks = append(run.Tasks, "task-ref-001")
	provider := AgentAssessmentReplanCandidateProviderV0{
		PlanSource: staticAssessmentReplanPlanSourceV0{Plans: []AgentAssessmentReplanPlanV0{
			assessmentReplanAskDirectorPlanV0(runRef),
		}},
	}

	_, err := provider.BuildSchedulerCandidatesV0(context.Background(), SchedulerCandidateRequestV0{
		Run:           run,
		OccurredAt:    "2026-05-09T16:10:00Z",
		CorrelationID: "corr-assessment-replan-plan-001",
	})

	var publicErr ErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected ErrorV0, got %T %v", err, err)
	}
	if publicErr.Code != ErrNucleoOrquestacionInvalidoV0 || publicErr.Field != "requested_action" {
		t.Fatalf("error=%+v", publicErr)
	}
}

type staticAssessmentReplanPlanSourceV0 struct {
	Plans []AgentAssessmentReplanPlanV0
}

func (source staticAssessmentReplanPlanSourceV0) BuildAgentAssessmentReplanPlansV0(
	context.Context,
	AgentAssessmentReplanPlanRequestV0,
) ([]AgentAssessmentReplanPlanV0, error) {
	return append([]AgentAssessmentReplanPlanV0(nil), source.Plans...), nil
}

func assessmentReplanAskDirectorPlanV0(runRef string) AgentAssessmentReplanPlanV0 {
	return AgentAssessmentReplanPlanV0{
		CandidateRef:    "assessment-replan-candidate-ask-001",
		ReplanRef:       "replan-ref-assessment-ask-001",
		SignalRef:       "signal-ref-assessment-ask-001",
		TaskRef:         "task-ref-001",
		RequestedAction: orquestacorereplanner.ReplanActionAskDirectorV0,
		Summary:         "Consultar al director por evaluacion ambigua.",
		EvidenceRefs:    []string{"evidence-ref-assessment-ask-001"},
		Assessment: orquestacoreworkflow.AgentWorkAssessedPayloadV0{
			AssessmentRef:  "assessment-ref-ask-001",
			PhaseID:        string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			AgentRequestID: "agent-ref-ask-001",
			TaskRef:        "task-ref-001",
			Verdict:        orquestacoreworkflow.AgentAssessmentVerdictLoopDetectedV0,
			Action:         orquestacoreworkflow.AgentAssessmentActionAskDirectorV0,
			Severity:       orquestacoreworkflow.AgentAssessmentSeverityHighV0,
			Summary:        "Evaluacion requiere consulta externa.",
			EvidenceRefs:   []string{"evidence-ref-assessment-ask-payload-001"},
		},
	}
}
