package orquestacorereplanner

import (
	"errors"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func TestValidateAgentReworkSignalV0RejectsUnsupportedVerdictAndActions(t *testing.T) {
	cases := map[string]struct {
		verdict          string
		assessmentAction string
		requestedAction  ReplanRecommendedActionV0
		code             string
		field            string
	}{
		"acceptable": {
			verdict:          orquestacoreworkflow.AgentAssessmentVerdictAcceptableV0,
			assessmentAction: orquestacoreworkflow.AgentAssessmentActionContinueV0,
			requestedAction:  ReplanActionRetryTaskV0,
			code:             ErrAgentReworkVerdictNoSoportadoV0,
			field:            "assessment_status",
		},
		"garbage_request_revision": {
			verdict:          orquestacoreworkflow.AgentAssessmentVerdictGarbageV0,
			assessmentAction: orquestacoreworkflow.AgentAssessmentActionRequestRevisionV0,
			requestedAction:  ReplanActionRetryTaskV0,
			code:             ErrAgentReworkActionNoSoportadaV0,
			field:            "assessment_action",
		},
		"needs_revision_stop": {
			verdict:          orquestacoreworkflow.AgentAssessmentVerdictNeedsRevisionV0,
			assessmentAction: orquestacoreworkflow.AgentAssessmentActionStopAgentV0,
			requestedAction:  ReplanActionReplaceAgentV0,
			code:             ErrAgentReworkActionNoSoportadaV0,
			field:            "assessment_action",
		},
		"ask_director_abort": {
			verdict:          orquestacoreworkflow.AgentAssessmentVerdictLoopDetectedV0,
			assessmentAction: orquestacoreworkflow.AgentAssessmentActionAskDirectorV0,
			requestedAction:  ReplanActionAbortTaskV0,
			code:             ErrAgentReworkActionNoSoportadaV0,
			field:            "requested_action",
		},
		"stop_retry": {
			verdict:          orquestacoreworkflow.AgentAssessmentVerdictGarbageV0,
			assessmentAction: orquestacoreworkflow.AgentAssessmentActionStopAgentV0,
			requestedAction:  ReplanActionRetryTaskV0,
			code:             ErrAgentReworkActionNoSoportadaV0,
			field:            "requested_action",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			signal := validAgentReworkSignalV0()
			signal.AssessmentStatus = tc.verdict
			signal.AssessmentAction = tc.assessmentAction
			signal.RequestedAction = tc.requestedAction

			err := ValidateAgentReworkSignalV0(NormalizeAgentReworkSignalV0(signal))
			assertAgentReworkSignalErrorV0(t, err, tc.code, tc.field)
		})
	}
}

func TestValidateAgentReworkSignalV0AllowsOperationalLabels(t *testing.T) {
	signal := validAgentReworkSignalV0()
	signal.Summary = "Incluye transcript policy ref y runtime provider ref."

	err := ValidateAgentReworkSignalV0(NormalizeAgentReworkSignalV0(signal))
	if err != nil {
		t.Fatalf("ValidateAgentReworkSignalV0: %v", err)
	}
}

func TestValidateAgentReworkSignalV0RejectsSensitiveDetails(t *testing.T) {
	signal := validAgentReworkSignalV0()
	signal.Summary = "transcript=raw text"

	err := ValidateAgentReworkSignalV0(NormalizeAgentReworkSignalV0(signal))
	assertAgentReworkSignalErrorV0(t, err, ErrDetalleProhibidoV0, "payload")
}

func assertAgentReworkSignalErrorV0(t *testing.T, err error, code string, field string) {
	t.Helper()
	var publicErr AgentReworkSignalErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected AgentReworkSignalErrorV0, got %T %v", err, err)
	}
	if publicErr.Code != code || publicErr.Field != field {
		t.Fatalf("error=%+v, want code=%s field=%s", publicErr, code, field)
	}
}
