package orquestacoreworkflow

import "testing"

func TestRecordReplanDecisionCommandV0AcceptsBlockedQualityGateSourceInProgramming(t *testing.T) {
	run := mustReplanDecisionQualityGateProgrammingRunV0(t)
	gate := mustRecordQualityGateCommandV0(
		t,
		"cmd-quality-gate-blocked-replan",
		"idem-quality-gate-blocked-replan",
		validProgrammingQualityGatePayloadV0("quality-gate-blocked-replan", QualityGateDecisionBlockedV0),
	)
	run = mustApplySingleCommandEventV0(t, run, gate)
	command := mustRecordReplanDecisionFromQualityGateCommandV0(
		t,
		"cmd-replan-quality-gate-blocked",
		"idem-replan-quality-gate-blocked",
		"replan-decision-quality-gate-blocked",
		"quality-gate-blocked-replan",
	)

	result, err := HandleCommandV0(run, command)
	if err != nil {
		t.Fatalf("handle RecordReplanDecision from blocked QualityGate: %v", err)
	}
	assertSingleEventTypeV0(t, result, OrchestrationEventReplanDecisionRecordedV0)
	if len(result.Outbox) != 0 {
		t.Fatalf("outbox=%d, want empty", len(result.Outbox))
	}
	next := mustApplyReducerEventV0(t, run, result.Events[0])
	if issues := ValidateOrchestrationRunV0(next); len(issues) != 0 {
		t.Fatalf("run con replan desde quality gate bloqueante invalido: %+v", issues)
	}
}

func TestRecordReplanDecisionCommandV0RejectsMissingOrNonBlockedQualityGateSource(t *testing.T) {
	t.Run("missing_gate", func(t *testing.T) {
		run := mustReplanDecisionQualityGateProgrammingRunV0(t)
		command := mustRecordReplanDecisionFromQualityGateCommandV0(
			t,
			"cmd-replan-quality-gate-missing",
			"idem-replan-quality-gate-missing",
			"replan-decision-quality-gate-missing",
			"quality-gate-missing",
		)

		_, err := HandleCommandV0(run, command)
		assertCommandErrorV0(t, err, ErrTransicionInvalidaV0, "payload.source_ref")
	})

	t.Run("accepted_gate", func(t *testing.T) {
		run := mustReplanDecisionQualityGateProgrammingRunV0(t)
		gate := mustRecordQualityGateCommandV0(
			t,
			"cmd-quality-gate-accepted-replan",
			"idem-quality-gate-accepted-replan",
			validProgrammingQualityGatePayloadV0("quality-gate-accepted-replan", QualityGateDecisionAcceptedV0),
		)
		run = mustApplySingleCommandEventV0(t, run, gate)
		command := mustRecordReplanDecisionFromQualityGateCommandV0(
			t,
			"cmd-replan-quality-gate-accepted",
			"idem-replan-quality-gate-accepted",
			"replan-decision-quality-gate-accepted",
			"quality-gate-accepted-replan",
		)

		_, err := HandleCommandV0(run, command)
		assertCommandErrorV0(t, err, ErrTransicionInvalidaV0, "payload.source_ref")
	})
}

func mustReplanDecisionQualityGateProgrammingRunV0(t *testing.T) OrchestrationRunV0 {
	t.Helper()
	run := mustFunctionContractReadyRunV0(t)
	task := mustCreateMicrotaskCommandV0(t, "cmd-create-task-for-quality-gate-replan", "idem-create-task-for-quality-gate-replan", "task-ncw-009")
	run = mustApplySingleCommandEventV0(t, run, task)
	open := mustOpenPhaseCommandV0(t, "cmd-open-programacion-quality-gate-replan", "idem-open-programacion-quality-gate-replan", OrchestrationPhaseProgramacionV0)
	return mustApplySingleCommandEventV0(t, run, open)
}

func mustRecordReplanDecisionFromQualityGateCommandV0(
	t *testing.T,
	commandID string,
	idempotencyKey string,
	replanRef string,
	gateRef string,
) OrchestrationCommandV0 {
	t.Helper()
	command, err := NewRecordReplanDecisionCommandV0(
		validCommandMetaV0(commandID, idempotencyKey),
		validReplanDecisionFromQualityGatePayloadV0(replanRef, gateRef),
	)
	return mustCommandV0(t, command, err)
}

func validReplanDecisionFromQualityGatePayloadV0(replanRef string, gateRef string) ReplanDecisionRecordedPayloadV0 {
	payload := validReplanDecisionPayloadV0(replanRef)
	payload.SourceRef = gateRef
	payload.AcceptedAction = ReplanDecisionActionAskDirectorV0
	payload.FollowupRefs = []string{"followup-quality-gate-director-001"}
	payload.Summary = "Decision compacta para tratar un quality gate bloqueante en programacion."
	return payload
}

func validProgrammingQualityGatePayloadV0(
	gateRef string,
	decision QualityGateDecisionV0,
) RecordQualityGateCommandPayloadV0 {
	payload := validQualityGatePayloadV0(gateRef, decision)
	payload.PhaseID = string(OrchestrationPhaseProgramacionV0)
	payload.SubjectRef = "task-ncw-009"
	return payload
}
