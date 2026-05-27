package orquestadirectorscheduler

import (
	"strings"
	"testing"
)

func TestSchedulerTickRailPolicyV0AcceptsOperationalVocabularyInWorkText(t *testing.T) {
	input := validSchedulerTickInputV0()
	input.WorkCandidates[0].CapacityCandidate.Payload.Summary =
		"runtime provider model db sql filesystem docker por refs opacas."
	input.WorkCandidates[0].AgentCandidate.Payload.Summary =
		"Usa adapter_ref y provider_policy_ref sin valores reales."
	input.WorkCandidates[0].CapacityCandidate.Payload.ReasonCode = "runtime_backpressure"

	plan := mustSchedulerTickPlanV0(t, input)

	assertSchedulerPlanV0(t, plan, SchedulerTickStatusCommandsReadyV0, 1)
}

func TestSchedulerTickRailPolicyV0RejectsSensitiveWorkText(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*DirectorSchedulerTickInputV0)
	}{
		{
			name: "summary_secret",
			mutate: func(input *DirectorSchedulerTickInputV0) {
				input.WorkCandidates[0].CapacityCandidate.Payload.Summary = "api_key=valor-real"
			},
		},
		{
			name: "summary_home_path",
			mutate: func(input *DirectorSchedulerTickInputV0) {
				input.WorkCandidates[0].AgentCandidate.Payload.Summary = "/home/operador/proyecto"
			},
		},
		{
			name: "reason_raw_prompt",
			mutate: func(input *DirectorSchedulerTickInputV0) {
				input.WorkCandidates[0].CapacityCandidate.Payload.ReasonCode = "prompt=texto-crudo"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("ORQUESTA_DETAIL_PROHIBITED_RAILS", "on")
			t.Setenv("ORQUESTA_DETAIL_PROHIBITED_RAILS_SCOPE", "*")
			input := validSchedulerTickInputV0()
			tt.mutate(&input)

			_, err := BuildDirectorSchedulerTickV0(input)
			assertSchedulerTickErrorV0(t, err, "payload")
		})
	}
}

func TestSchedulerTickRailPolicyV0ProgrammingPermitePayloadGrande(t *testing.T) {
	t.Setenv("ORQUESTA_SECURITY_MODE", "programming")
	input := validSchedulerTickInputV0()
	input.EvidenceRefs = append(input.EvidenceRefs, strings.Repeat("evidence-ref-large-", 20000))

	plan := mustSchedulerTickPlanV0(t, input)

	assertSchedulerPlanV0(t, plan, SchedulerTickStatusCommandsReadyV0, 1)
}
