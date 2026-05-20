package orquestadirectorscheduler

import (
	"errors"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirector "orquesta/modulos/orquesta-director"
)

func TestSchedulerTickWrappedExternalErrorV0ConservaCampoPublico(t *testing.T) {
	cases := []struct {
		name   string
		prefix string
		err    error
		want   string
	}{
		{
			name:   "replan",
			prefix: "replan_followup_candidates",
			err: orquestadirector.ReplanFollowupsErrorV0{
				Code:  orquestadirector.ErrDirectorReplanFollowupsInvalidoV0,
				Field: "decision_payload.payload",
			},
			want: "replan_followup_candidates.decision_payload.payload",
		},
		{
			name:   "progress",
			prefix: "progress_supervision_candidates",
			err: orquestadirector.AgentProgressSupervisionErrorV0{
				Code:  orquestadirector.ErrDirectorAgentProgressSupervisionInvalidaV0,
				Field: "report",
			},
			want: "progress_supervision_candidates.report",
		},
		{
			name:   "workflow_command",
			prefix: "replan_followup_candidates",
			err: orquestacoreworkflow.OrchestrationCommandErrorV0{
				Code:  orquestacoreworkflow.ErrPayloadInvalidoV0,
				Field: "payload.followup_refs",
			},
			want: "replan_followup_candidates.payload.followup_refs",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			wrapped := schedulerTickWrappedExternalErrorV0(tc.prefix, tc.err)
			var schedulerErr DirectorSchedulerTickErrorV0
			if !errors.As(wrapped, &schedulerErr) {
				t.Fatalf("expected scheduler error, got %T %v", wrapped, wrapped)
			}
			if schedulerErr.Field != tc.want {
				t.Fatalf("field=%q want %q", schedulerErr.Field, tc.want)
			}
		})
	}
}
