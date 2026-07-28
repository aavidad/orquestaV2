package application

import (
	"testing"
	"time"

	"orquesta/internal/council"
	"orquesta/internal/ports"
)

func TestValidatePersistedCouncilSubjectResolvesRetriedAuthorFromChangeExecutionRef(t *testing.T) {
	system := newCouncilSystem(t, council.PolicyAuto)
	agent := system.orchestrator.observer.(*scriptedAgent)
	agent.mu.Lock()
	agent.observations = []ports.AgentObservation{
		{Status: ports.AgentFailed, ErrorCode: "provider.retryable_failure"},
		{Status: ports.AgentCompleted, MediaType: "text/plain", Content: []byte("retried candidate")},
	}
	agent.mu.Unlock()

	system.process(t, ActionPrepareWorkspace, ActionLaunchAgent, ActionObserveAgent)
	system.orchestrator.clock.(*mutableClock).Advance(time.Second)
	system.process(t, ActionPrepareWorkspace, ActionLaunchAgent, ActionObserveAgent, ActionCommitChange)
	system.process(t, ActionAttestTest, ActionLaunchAgent, ActionLaunchAgent, ActionObserveAgent, ActionObserveAgent)

	record := system.record(t)
	if len(record.Executions) < 4 || len(record.ChangeSets) != 1 ||
		record.Executions[0].State != ExecutionFailed ||
		record.Executions[1].Purpose != ExecutionPurposeAuthor ||
		record.Executions[1].AttemptNo != 2 ||
		record.ChangeSets[0].ExecutionRef != record.Executions[1].Ref {
		t.Fatalf("author retry causality incomplete: executions=%+v changes=%+v", record.Executions, record.ChangeSets)
	}
	if len(record.Reviews) != 2 || len(record.CouncilRounds) != 1 {
		t.Fatalf("approved retry did not reach Council: reviews=%d rounds=%d", len(record.Reviews), len(record.CouncilRounds))
	}
	if err := ValidatePersistedCouncilSubject(record, record.CouncilRounds[0].Subject); err != nil {
		t.Fatalf("retried author Council subject rejected: %v", err)
	}
}
