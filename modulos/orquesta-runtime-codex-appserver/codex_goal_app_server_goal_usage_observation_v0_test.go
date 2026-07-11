package orquestaruntimecodexappserver

import (
	"context"
	"testing"
	"time"

	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestaruntimecodexgoal "orquesta/modulos/orquesta-runtime-codex-goal"
)

func TestServerCodexAppServerGoalBackendV0PublicaUsoBajoConTimestampProveedorV0(t *testing.T) {
	updatedAt := time.Date(2026, 7, 11, 10, 34, 56, 0, time.UTC)
	protocol := &fakeCodexAppServerProtocolV0{observedGoal: &serverCodexAppServerThreadGoalV0{
		ThreadID:        "thread-ref-goal-usage-low-001",
		Status:          "active",
		TokensUsed:      42,
		TimeUsedSeconds: 7,
		UpdatedAt:       serverCodexAppServerTimestampV0{set: true, value: updatedAt.Unix()},
	}}
	backend := serverCodexAppServerGoalBackendV0{
		Protocol:                protocol,
		HighTokenUsageThreshold: 100,
		Now:                     func() time.Time { return updatedAt.Add(time.Minute) },
	}

	receipt, err := backend.ObserveCodexGoalV0(context.Background(), orquestaruntimecodexgoal.CodexGoalObservationRequestV0{
		GoalRef:         "goal-ref-usage-low-001",
		ExternalGoalRef: "thread-ref-goal-usage-low-001",
	})
	if err != nil {
		t.Fatalf("ObserveCodexGoalV0: %v", err)
	}
	usage := receipt.UsageObservation
	if usage.TokensAccumulated != 42 || usage.RuntimeSeconds != 7 ||
		usage.ObservedAt != "2026-07-11T10:34:56Z" ||
		usage.SourceRef != codexAppServerGoalUsageSourceRefV0("thread-ref-goal-usage-low-001") ||
		len(usage.EvidenceRefs) != 1 || usage.EvidenceRefs[0] != "evidence-ref-codex-app-server-goal-usage-observed" ||
		receipt.Summary != "codex_app_server_goal_status_active" {
		t.Fatalf("receipt=%+v", receipt)
	}
}

func TestServerCodexAppServerGoalBackendV0UsoUsaNowSinUpdatedAtV0(t *testing.T) {
	now := time.Date(2026, 7, 11, 10, 35, 0, 123000000, time.UTC)
	protocol := &fakeCodexAppServerProtocolV0{observedGoal: &serverCodexAppServerThreadGoalV0{
		ThreadID: "thread-ref-goal-usage-fallback-001",
		Status:   "complete",
	}}
	backend := serverCodexAppServerGoalBackendV0{
		Protocol: protocol,
		Now:      func() time.Time { return now },
	}

	receipt, err := backend.ObserveCodexGoalV0(context.Background(), orquestaruntimecodexgoal.CodexGoalObservationRequestV0{
		GoalRef:         "goal-ref-usage-fallback-001",
		ExternalGoalRef: "thread-ref-goal-usage-fallback-001",
	})
	if err != nil {
		t.Fatalf("ObserveCodexGoalV0: %v", err)
	}
	usage := receipt.UsageObservation
	if receipt.Status != orquestagoal.GoalStatusCompleteV0 ||
		usage.ObservedAt != "2026-07-11T10:35:00.123Z" ||
		usage.SourceRef != codexAppServerGoalUsageSourceRefV0("thread-ref-goal-usage-fallback-001") ||
		len(usage.EvidenceRefs) != 1 || usage.EvidenceRefs[0] != "evidence-ref-codex-app-server-goal-usage-observed" {
		t.Fatalf("receipt=%+v", receipt)
	}
}
