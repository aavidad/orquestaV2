package application

import (
	"context"
	"testing"
	"time"

	"orquesta/internal/council"
	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

const terminalSecurityTestCode = "provider.security_failure"

func terminalSecurityObservation() ports.AgentObservation {
	return ports.AgentObservation{
		Status:             ports.AgentFailed,
		FailureDisposition: ports.AgentFailureDispositionTerminalSecurity,
		ErrorCode:          terminalSecurityTestCode,
	}
}

func TestTerminalSecurityInterruptsOrdinaryWorkWithoutReplacementOrFalseClosure(t *testing.T) {
	ctx := context.Background()
	clock := &mutableClock{now: time.Date(2026, 7, 25, 9, 0, 0, 0, time.UTC)}
	repository := newMemoryRepository()
	repository.now = clock.Now
	agent := &scriptedAgent{now: clock.Now, observations: []ports.AgentObservation{terminalSecurityObservation()}}
	orchestrator, _ := newTestOrchestrator(t, repository, clock, agent)
	actor, project := testScope(t)
	submitted, err := orchestrator.Submit(ctx, accessForScope(t, actor, project), SubmitRequest{
		RequestRef: "request:terminal-security-work", Statement: "security terminal work", Confirm: true,
	})
	appTestNoError(t, err)
	if _, err := orchestrator.ProcessNext(ctx, "worker:terminal-security"); err != nil {
		t.Fatalf("launch: %v", err)
	}
	if _, err := orchestrator.ProcessNext(ctx, "worker:terminal-security"); err != nil {
		t.Fatalf("observe: %v", err)
	}

	record, err := repository.GetGoal(ctx, submitted.Record.Goal.Ref())
	appTestNoError(t, err)
	item := record.Goal.WorkItems()[0]
	if record.Goal.State() == goal.GoalStateSucceeded || item.State() != goal.WorkItemStateInterrupted ||
		len(record.Executions) != 1 || record.Executions[0].State != ExecutionFailed ||
		record.Executions[0].FailureCode != terminalSecurityTestCode ||
		record.Executions[0].AttemptNo != 1 || record.Executions[0].MaxExecutionAttempts != 3 {
		t.Fatalf("terminal work state=%s item=%s executions=%+v", record.Goal.State(), item.State(), record.Executions)
	}
	if len(record.Artifacts) != 0 || len(record.Attestations) != 0 {
		t.Fatalf("terminal work gained false evidence: artifacts=%d attestations=%d",
			len(record.Artifacts), len(record.Attestations))
	}
	repository.mu.Lock()
	pending := len(repository.actions)
	repository.mu.Unlock()
	if pending != 0 {
		t.Fatalf("terminal work left replacement/action count=%d", pending)
	}
}

func TestTerminalSecurityAbortsReviewRoundAndInterruptsAuthorAfterCleanup(t *testing.T) {
	system := newTestAttestationSystem(t, ports.TestAttestationPassed)
	system.processCommit(t)
	system.process(t, ActionAttestTest, ActionLaunchAgent, ActionLaunchAgent)
	before := system.record(t)
	agent := system.orchestrator.observer.(*scriptedAgent)
	agent.mu.Lock()
	agent.observations = []ports.AgentObservation{terminalSecurityObservation()}
	agent.mu.Unlock()

	system.process(t, ActionObserveAgent)
	mid := system.record(t)
	reviewers, failed, replacements := 0, 0, 0
	for _, execution := range mid.Executions {
		if !isReviewerExecution(execution) {
			continue
		}
		reviewers++
		if execution.State == ExecutionFailed && execution.FailureCode == terminalSecurityTestCode {
			failed++
		}
		if execution.ReplacesExecutionRef.String() != "" {
			replacements++
		}
	}
	if reviewers != 2 || failed != 1 || replacements != 0 || len(mid.Reviews) != 0 ||
		len(mid.Artifacts) != len(before.Artifacts) || system.actionKindCount(ActionStopAgent) != 1 {
		t.Fatalf("review terminal reviewers=%d failed=%d replacements=%d reviews=%d artifacts=%d/%d stops=%d",
			reviewers, failed, replacements, len(mid.Reviews), len(mid.Artifacts), len(before.Artifacts),
			system.actionKindCount(ActionStopAgent))
	}

	assertReviewRoundAborted(t, system, terminalSecurityTestCode)
	closed := system.record(t)
	if closed.Goal.State() == goal.GoalStateSucceeded || len(closed.Reviews) != 0 ||
		len(closed.Artifacts) != len(before.Artifacts) {
		t.Fatalf("review terminal false closure state=%s reviews=%d artifacts=%d/%d",
			closed.Goal.State(), len(closed.Reviews), len(closed.Artifacts), len(before.Artifacts))
	}
}

func TestTerminalSecurityAbortsCouncilCohortWithoutDecisionOrOrphanAction(t *testing.T) {
	system := newCouncilSystem(t, council.PolicyAuto)
	system.processCommit(t)
	system.process(t, ActionAttestTest, ActionLaunchAgent, ActionLaunchAgent, ActionObserveAgent, ActionObserveAgent)
	system.process(t, ActionLaunchAgent, ActionLaunchAgent, ActionLaunchAgent)
	before := system.record(t)
	agent := system.orchestrator.observer.(*scriptedAgent)
	agent.mu.Lock()
	agent.observations = []ports.AgentObservation{terminalSecurityObservation()}
	agent.mu.Unlock()

	system.process(t, ActionObserveAgent)
	mid := system.record(t)
	councilExecutions, failed, replacements := 0, 0, 0
	for _, execution := range mid.Executions {
		if !isCouncilExecution(execution) {
			continue
		}
		councilExecutions++
		if execution.State == ExecutionFailed && execution.FailureCode == terminalSecurityTestCode {
			failed++
		}
		if execution.ReplacesExecutionRef.String() != "" {
			replacements++
		}
	}
	if councilExecutions != 3 || failed != 1 || replacements != 0 ||
		len(mid.CouncilFacts) != len(before.CouncilFacts) ||
		len(mid.CouncilDecisions) != len(before.CouncilDecisions) ||
		len(mid.Artifacts) != len(before.Artifacts) ||
		system.actionKindCount(ActionStopAgent) != 2 ||
		system.actionKindCount(ActionObserveAgent) != 0 {
		t.Fatalf("Council terminal executions=%d failed=%d replacements=%d facts=%d/%d decisions=%d/%d artifacts=%d/%d stops=%d observes=%d",
			councilExecutions, failed, replacements, len(mid.CouncilFacts), len(before.CouncilFacts),
			len(mid.CouncilDecisions), len(before.CouncilDecisions), len(mid.Artifacts), len(before.Artifacts),
			system.actionKindCount(ActionStopAgent), system.actionKindCount(ActionObserveAgent))
	}

	system.process(t, ActionStopAgent, ActionStopAgent)
	closed := system.record(t)
	item := closed.Goal.WorkItems()[0]
	author, found := authorBound(closed, item)
	stopped := 0
	for _, execution := range closed.Executions {
		if isCouncilExecution(execution) && execution.State == ExecutionStopped &&
			execution.FailureCode == "council.round_aborted" {
			stopped++
		}
	}
	if closed.Goal.State() == goal.GoalStateSucceeded || item.State() != goal.WorkItemStateInterrupted ||
		!found || author.State != ExecutionFailed || author.FailureCode != "council.unavailable" ||
		stopped != 2 || len(closed.CouncilFacts) != len(before.CouncilFacts) ||
		len(closed.CouncilDecisions) != len(before.CouncilDecisions) ||
		len(closed.Artifacts) != len(before.Artifacts) ||
		system.actionKindCount(ActionStopAgent) != 0 ||
		system.actionKindCount(ActionObserveAgent) != 0 ||
		system.actionKindCount(ActionIntegrateChange) != 0 {
		t.Fatalf("Council cleanup state=%s item=%s author=%+v stopped=%d facts=%d decisions=%d artifacts=%d actions(stop/observe/integrate)=%d/%d/%d",
			closed.Goal.State(), item.State(), author, stopped, len(closed.CouncilFacts),
			len(closed.CouncilDecisions), len(closed.Artifacts), system.actionKindCount(ActionStopAgent),
			system.actionKindCount(ActionObserveAgent), system.actionKindCount(ActionIntegrateChange))
	}
	result, err := system.orchestrator.ProcessNext(context.Background(), "worker:terminal-security-replay")
	if err != nil || result.Processed {
		t.Fatalf("Council terminal replay result=%+v err=%v", result, err)
	}
}

func TestTerminalSecurityCouncilCleanupFencesDispatchingPeerUntilAcceptedStop(t *testing.T) {
	system := newCouncilSystem(t, council.PolicyAuto)
	system.processCommit(t)
	system.process(t, ActionAttestTest, ActionLaunchAgent, ActionLaunchAgent, ActionObserveAgent, ActionObserveAgent)
	system.process(t, ActionLaunchAgent, ActionLaunchAgent)

	system.repository.mu.Lock()
	record := system.repository.records[system.goalRef]
	var dispatching goal.ExecutionRef
	for index := range record.Executions {
		if !isCouncilExecution(record.Executions[index]) || record.Executions[index].State != ExecutionQueued {
			continue
		}
		record.Executions[index].State = ExecutionDispatching
		dispatching = record.Executions[index].Ref
		break
	}
	for ref, action := range system.repository.actions {
		if action.record.Kind == ActionLaunchAgent && action.record.ExecutionRef == dispatching {
			action.record.AvailableAt = system.orchestrator.clock.Now().Add(time.Minute)
			system.repository.actions[ref] = action
		}
	}
	system.repository.records[system.goalRef] = record
	system.repository.mu.Unlock()
	if dispatching.String() == "" {
		t.Fatal("dispatching Council peer not seeded")
	}
	agent := system.orchestrator.observer.(*scriptedAgent)
	agent.mu.Lock()
	agent.observations = []ports.AgentObservation{terminalSecurityObservation()}
	agent.mu.Unlock()

	system.process(t, ActionObserveAgent)
	mid := system.record(t)
	pendingDispatchCleanup := 0
	for _, control := range mid.Controls {
		if IsCouncilCleanupControl(control) && control.ExecutionRef == dispatching &&
			control.Status == ControlRequested {
			pendingDispatchCleanup++
		}
	}
	if pendingDispatchCleanup != 1 || mid.Goal.WorkItems()[0].State() != goal.WorkItemStateRunning {
		t.Fatalf("dispatch cleanup controls=%d item=%s", pendingDispatchCleanup, mid.Goal.WorkItems()[0].State())
	}
	system.process(t, ActionStopAgent)
	if item := system.record(t).Goal.WorkItems()[0]; item.State() != goal.WorkItemStateRunning {
		t.Fatalf("dispatching peer did not fence interruption: item=%s", item.State())
	}

	system.orchestrator.clock.(*mutableClock).Advance(2 * time.Minute)
	system.process(t, ActionLaunchAgent)
	afterLaunch := system.record(t)
	peer, found := executionByRef(afterLaunch.Executions, dispatching)
	if !found || peer.State != ExecutionRunning || system.actionKindCount(ActionStopAgent) != 1 ||
		system.actionKindCount(ActionObserveAgent) != 0 {
		t.Fatalf("dispatch cleanup launch peer=%+v stops=%d observes=%d",
			peer, system.actionKindCount(ActionStopAgent), system.actionKindCount(ActionObserveAgent))
	}
	system.process(t, ActionStopAgent)
	closed := system.record(t)
	item := closed.Goal.WorkItems()[0]
	author, found := authorBound(closed, item)
	if item.State() != goal.WorkItemStateInterrupted || !found ||
		author.State != ExecutionFailed || author.FailureCode != "council.unavailable" ||
		system.actionKindCount(ActionLaunchAgent) != 0 ||
		system.actionKindCount(ActionObserveAgent) != 0 ||
		system.actionKindCount(ActionStopAgent) != 0 {
		t.Fatalf("dispatch cleanup final item=%s author=%+v actions=%d/%d/%d",
			item.State(), author, system.actionKindCount(ActionLaunchAgent),
			system.actionKindCount(ActionObserveAgent), system.actionKindCount(ActionStopAgent))
	}
}
