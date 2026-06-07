package orquestadirectorsupervisor

import (
	"errors"
	"testing"

	orquestadirectorrunner "orquesta/modulos/orquesta-director-runner"
	orquestadirectorscheduler "orquesta/modulos/orquesta-director-scheduler"
)

func TestBuildDirectorSupervisorBriefingV0ExponeNextActionContinue(t *testing.T) {
	decision := assertSupervisorActionV0(
		t,
		supervisorTestInputV0(orquestadirectorrunner.DirectorCycleStatusCommandsAppliedV0),
		DirectorSupervisorActionContinueV0,
		true,
	)
	decision.EvidenceRefs = []string{"evidence-ref-continue-001", "evidence-ref-continue-001"}

	briefing := mustBuildSupervisorBriefingV0(t, decision)

	if briefing.SchemaVersion != DirectorSupervisorBriefingSchemaV0 ||
		briefing.NextAction == nil ||
		briefing.NextAction.Kind != DirectorSupervisorActionKindRunStepV0 ||
		!briefing.NextAction.SafeToApply ||
		briefing.NextAction.RequiresDirector {
		t.Fatalf("briefing=%+v", briefing)
	}
	if len(briefing.ActionQueue) != 1 || len(briefing.Timeline) != 1 {
		t.Fatalf("queue/timeline=%+v/%+v", briefing.ActionQueue, briefing.Timeline)
	}
	if len(briefing.EvidenceRefs) != 1 || briefing.EvidenceRefs[0] != "evidence-ref-continue-001" {
		t.Fatalf("evidence refs=%v", briefing.EvidenceRefs)
	}
}

func TestBuildDirectorSupervisorBriefingV0OutboxPendingPriorizaDispatch(t *testing.T) {
	input := supervisorTestInputV0(orquestadirectorrunner.DirectorCycleStatusOutboxPendingV0)
	input.LastStepResult.PendingOutboxAfterRefs = []string{"outbox-ref-002", "outbox-ref-001", "outbox-ref-002"}
	decision := assertSupervisorActionV0(t, input, DirectorSupervisorActionWaitOutboxV0, false)

	briefing := mustBuildSupervisorBriefingV0(t, decision)

	if briefing.NextAction.Kind != DirectorSupervisorActionKindDispatchOutboxV0 ||
		!briefing.NextAction.SafeToApply ||
		briefing.NextAction.RequiresDirector {
		t.Fatalf("next=%+v", briefing.NextAction)
	}
	if len(briefing.NextAction.TargetRefs) != 2 ||
		briefing.NextAction.TargetRefs[0] != "outbox-ref-002" ||
		briefing.NextAction.TargetRefs[1] != "outbox-ref-001" {
		t.Fatalf("target refs=%v", briefing.NextAction.TargetRefs)
	}
}

func TestBuildDirectorSupervisorBriefingV0NeedsDirectorNoSeAutoaplica(t *testing.T) {
	decision := assertSupervisorActionV0(
		t,
		supervisorTestInputV0(orquestadirectorrunner.DirectorCycleStatusNeedsDirectorV0),
		DirectorSupervisorActionNeedsDirectorV0,
		false,
	)
	decision.WaitingReasons = []orquestadirectorscheduler.SchedulerWaitingReasonV0{
		orquestadirectorscheduler.SchedulerWaitingDirectorQuestionPendingV0,
	}

	briefing := mustBuildSupervisorBriefingV0(t, decision)

	if briefing.NextAction.Kind != DirectorSupervisorActionKindAskDirectorV0 ||
		briefing.NextAction.SafeToApply ||
		!briefing.NextAction.RequiresDirector {
		t.Fatalf("next=%+v", briefing.NextAction)
	}
	if len(briefing.NextAction.TargetRefs) != 1 ||
		briefing.NextAction.TargetRefs[0] != "waiting_reason:director_question_pending" {
		t.Fatalf("target refs=%v", briefing.NextAction.TargetRefs)
	}
}

func TestBuildDirectorSupervisorBriefingV0RechazaDecisionIncompleta(t *testing.T) {
	decision := DirectorSupervisorDecisionV0{
		RunRef:                   supervisorTestRunRefV0,
		Action:                   DirectorSupervisorActionContinueV0,
		AutonomousRecommendation: DirectorSupervisorAutonomousContinueV0,
		StepNumber:               1,
		MaxSteps:                 2,
	}

	_, err := BuildDirectorSupervisorBriefingV0(DirectorSupervisorBriefingInputV0{Decision: decision})
	if err == nil {
		t.Fatal("expected error")
	}
	var publicErr DirectorSupervisorErrorV0
	if !errors.As(err, &publicErr) ||
		publicErr.Code != ErrDirectorSupervisorDecisionInvalidaV0 ||
		publicErr.Field != "decision.reason_code" {
		t.Fatalf("err=%+v", err)
	}
}

func mustBuildSupervisorBriefingV0(
	t *testing.T,
	decision DirectorSupervisorDecisionV0,
) DirectorSupervisorBriefingV0 {
	t.Helper()
	briefing, err := BuildDirectorSupervisorBriefingV0(DirectorSupervisorBriefingInputV0{
		Decision:     decision,
		ObjectiveRef: "objective-ref-supervisor-briefing",
		ContextRefs:  []string{"context-ref-001", "context-ref-001"},
	})
	if err != nil {
		t.Fatalf("build briefing: %v", err)
	}
	if briefing.RunRef != decision.RunRef ||
		briefing.ObjectiveRef != "objective-ref-supervisor-briefing" ||
		len(briefing.ContextRefs) != 1 {
		t.Fatalf("briefing=%+v", briefing)
	}
	return briefing
}
