package orquestacoreleases

import "testing"

func TestEvaluateAgentLeaseV0HeartbeatVigenteContinua(t *testing.T) {
	input := validAgentLeaseEvaluationInputV0()
	input.NowObservedAt = "2026-05-06T10:15:30Z"
	input.LastHeartbeat.ObservedAt = "2026-05-06T10:15:00Z"
	input.LastHeartbeat.Status = AgentHeartbeatAliveV0
	input.Policy.TimeoutAction = AgentLeaseTimeoutStopAgentV0

	assessment, err := EvaluateAgentLeaseV0(input)
	if err != nil {
		t.Fatalf("EvaluateAgentLeaseV0: %v", err)
	}

	requireAgentTimeoutAssessmentV0(t, assessment, AgentTimeoutDecisionContinueV0, AgentTimeoutReasonHeartbeatCurrentV0)
}

func TestEvaluateAgentLeaseV0HeartbeatExpiradoUsaTimeoutActionDePolicy(t *testing.T) {
	input := validAgentLeaseEvaluationInputV0()
	input.LaunchObservedAt = "2026-05-06T10:05:00Z"
	input.NowObservedAt = "2026-05-06T10:17:01Z"
	input.LastHeartbeat.ObservedAt = "2026-05-06T10:15:00Z"
	input.Policy.TimeoutAction = AgentLeaseTimeoutReplanTaskV0

	assessment, err := EvaluateAgentLeaseV0(input)
	if err != nil {
		t.Fatalf("EvaluateAgentLeaseV0: %v", err)
	}

	requireAgentTimeoutAssessmentV0(t, assessment, AgentTimeoutDecisionReplanTaskV0, AgentTimeoutReasonHeartbeatTimeoutV0)
}

func TestEvaluateAgentLeaseV0SinHeartbeatYLaunchTimeoutVencidoProduceDecisionDurable(t *testing.T) {
	input := validAgentLeaseEvaluationInputV0()
	input.LastHeartbeat = nil
	input.LaunchObservedAt = "2026-05-06T10:00:00Z"
	input.NowObservedAt = "2026-05-06T10:03:00Z"
	input.Policy.TimeoutAction = AgentLeaseTimeoutAskDirectorV0

	assessment, err := EvaluateAgentLeaseV0(input)
	if err != nil {
		t.Fatalf("EvaluateAgentLeaseV0: %v", err)
	}

	requireAgentTimeoutAssessmentV0(t, assessment, AgentTimeoutDecisionAskDirectorV0, AgentTimeoutReasonLaunchTimeoutV0)
	if assessment.RunRef != input.RunRef || assessment.AgentRequestID != input.AgentRequestID || assessment.LeaseRef != input.LeaseRef {
		t.Fatalf("assessment no preserva refs compactas: %#v", assessment)
	}
}

func TestEvaluateAgentLeaseV0HeartbeatStoppedFailedProduceTerminalSinStopRuntime(t *testing.T) {
	tests := []struct {
		name     string
		status   AgentHeartbeatStatusV0
		decision AgentTimeoutDecisionV0
		reason   string
	}{
		{
			name:     "stopped",
			status:   AgentHeartbeatStoppedV0,
			decision: AgentTimeoutDecisionMarkStoppedV0,
			reason:   AgentTimeoutReasonHeartbeatStoppedV0,
		},
		{
			name:     "failed",
			status:   AgentHeartbeatFailedV0,
			decision: AgentTimeoutDecisionMarkFailedV0,
			reason:   AgentTimeoutReasonHeartbeatFailedV0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := validAgentLeaseEvaluationInputV0()
			input.LastHeartbeat.Status = test.status
			input.Policy.TimeoutAction = AgentLeaseTimeoutStopAgentV0

			assessment, err := EvaluateAgentLeaseV0(input)
			if err != nil {
				t.Fatalf("EvaluateAgentLeaseV0: %v", err)
			}

			requireAgentTimeoutAssessmentV0(t, assessment, test.decision, test.reason)
		})
	}
}

func TestEvaluateAgentLeaseV0RechazaNowYHeartbeatNoDeterministas(t *testing.T) {
	input := validAgentLeaseEvaluationInputV0()
	input.NowObservedAt = "2026-05-06T10:14:59Z"
	input.LastHeartbeat.ObservedAt = "2026-05-06T10:15:00Z"

	_, err := EvaluateAgentLeaseV0(input)
	requireAgentLeaseErrorCodeV0(t, err, ErrAgentLeaseTiempoInvalidoV0)

	input = validAgentLeaseEvaluationInputV0()
	input.NowObservedAt = "2026-05-06T10:15:00+02:00"
	_, err = EvaluateAgentLeaseV0(input)
	requireAgentLeaseErrorCodeV0(t, err, ErrAgentLeaseObservedAtV0)
}

func TestEvaluateAgentLeaseV0EntradaInvalidaNoHacePanic(t *testing.T) {
	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("EvaluateAgentLeaseV0 hizo panic: %v", recovered)
		}
	}()
	input := validAgentLeaseEvaluationInputV0()
	input.NowObservedAt = "2026-02-31T10:15:00Z"

	_, err := EvaluateAgentLeaseV0(input)
	requireAgentLeaseErrorCodeV0(t, err, ErrAgentLeaseObservedAtV0)
}

func TestAgentLeaseEvaluationInputV0ValidateMatchesEvaluationBoundary(t *testing.T) {
	input := validAgentLeaseEvaluationInputV0()
	input.NowObservedAt = "2026-02-31T10:15:00Z"

	issues := input.Validate()
	if len(issues) == 0 || issues[0].Code != ErrAgentLeaseObservedAtV0 {
		t.Fatalf("input.Validate()=%#v, want observed_at issue", issues)
	}
	_, err := EvaluateAgentLeaseV0(input)
	requireAgentLeaseErrorCodeV0(t, err, issues[0].Code)
}

func TestEvaluateAgentLeaseV0PropagaEvidenciaCompactaSinDuplicados(t *testing.T) {
	input := validAgentLeaseEvaluationInputV0()
	input.EvidenceRefs = []string{"evidence-input-lse-002", "evidence-lse-001"}
	input.Policy.EvidenceRefs = []string{"evidence-lse-001"}
	input.LastHeartbeat.EvidenceRefs = []string{"evidence-heartbeat-lse-002"}

	assessment, err := EvaluateAgentLeaseV0(input)
	if err != nil {
		t.Fatalf("EvaluateAgentLeaseV0: %v", err)
	}

	want := []string{
		"evidence-input-lse-002",
		"evidence-lse-001",
		"heartbeat-lse-001",
		"progress-report-lse-001",
		"evidence-heartbeat-lse-002",
	}
	if len(assessment.EvidenceRefs) != len(want) {
		t.Fatalf("evidence_refs=%#v, want %#v", assessment.EvidenceRefs, want)
	}
	for i := range want {
		if assessment.EvidenceRefs[i] != want[i] {
			t.Fatalf("evidence_refs=%#v, want %#v", assessment.EvidenceRefs, want)
		}
	}
}

func validAgentLeaseEvaluationInputV0() AgentLeaseEvaluationInputV0 {
	heartbeat := validAgentHeartbeatReportV0()
	return AgentLeaseEvaluationInputV0{
		AssessmentRef:    "assessment-lse-002",
		RunRef:           heartbeat.RunRef,
		AgentRequestID:   heartbeat.AgentRequestID,
		LeaseRef:         heartbeat.LeaseRef,
		LaunchObservedAt: "2026-05-06T10:00:00Z",
		NowObservedAt:    "2026-05-06T10:15:30Z",
		Policy:           validAgentLeasePolicyV0(),
		LastHeartbeat:    &heartbeat,
		EvidenceRefs:     []string{"evidence-input-lse-002"},
	}
}

func requireAgentTimeoutAssessmentV0(t *testing.T, assessment AgentTimeoutAssessmentV0, decision AgentTimeoutDecisionV0, reason string) {
	t.Helper()
	if assessment.Decision != decision {
		t.Fatalf("decision=%q, want %q: %#v", assessment.Decision, decision, assessment)
	}
	if assessment.ReasonCode != reason {
		t.Fatalf("reason_code=%q, want %q: %#v", assessment.ReasonCode, reason, assessment)
	}
	if issues := assessment.Validate(); len(issues) > 0 {
		t.Fatalf("assessment invalido: %#v", issues)
	}
}
