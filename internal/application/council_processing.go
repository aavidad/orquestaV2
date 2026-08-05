package application

import (
	"context"
	"errors"
	"time"

	"orquesta/internal/council"
	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

func (orchestrator *Orchestrator) processCouncilObservation(ctx context.Context, claim ActionClaim,
	record GoalRecord, item goal.WorkItem, itemState ExecutionRecord,
) error {
	execution := itemState
	observation, observeErr := orchestrator.observer.ObserveAgent(ctx, agentObserveRequest(execution))
	if observeErr != nil {
		if orchestrator.executionExpired(execution, claim) {
			return orchestrator.replaceCouncilExecution(ctx, claim, record, execution, "application.execution_expired",
				failedExecutionMayRetry, orchestrator.clock.Now(), unknownUsage(), 0, false)
		}
		return orchestrator.requeue(ctx, claim, execution, "agent.observe_failed")
	}
	if observation.ExecutionRef != execution.Ref {
		return orchestrator.replaceCouncilExecution(ctx, claim, record, execution, "agent.observation_execution_mismatch",
			failedExecutionMayRetry, orchestrator.clock.Now(), observation.Usage, int64(len(observation.Content)), false)
	}
	if err := ports.ValidateAgentObservation(observation, execution.MaxOutputBytes); err != nil {
		code := ports.AgentContractErrorCode(err)
		if isSpecHashFenceCode(code) {
			return orchestrator.quarantine(ctx, claim, code)
		}
		return orchestrator.replaceCouncilExecution(ctx, claim, record, execution, code,
			failedExecutionMayRetry, orchestrator.clock.Now(), observation.Usage, int64(len(observation.Content)), false)
	}
	if observation.SpecHash != record.Goal.SpecHash() {
		return orchestrator.quarantine(ctx, claim, "agent.observation_spec_hash_mismatch")
	}
	at := lifecycleTime(orchestrator.clock.Now(), record.Goal, item)
	execution.LastObservedAt, execution.ProviderObservedAt = at, observation.ObservedAt.UTC()
	switch observation.Status {
	case ports.AgentPending, ports.AgentRunning:
		if orchestrator.executionExpired(execution, claim) {
			return orchestrator.replaceCouncilExecution(ctx, claim, record, execution, "application.execution_expired",
				failedExecutionMayRetry, at, observation.Usage, int64(len(observation.Content)), false)
		}
		return orchestrator.requeue(ctx, claim, execution, "")
	case ports.AgentFailed:
		return orchestrator.replaceCouncilExecution(ctx, claim, record, execution, observation.ErrorCode,
			failedExecutionRetryPolicyFor(observation), at, observation.Usage, int64(len(observation.Content)), false)
	case ports.AgentCompleted:
		if !compatibleMediaType(execution.ArtifactMediaType, observation.MediaType) {
			return orchestrator.replaceCouncilExecution(ctx, claim, record, execution, "agent.observation_media_type_mismatch",
				failedExecutionMayRetry, at, observation.Usage, int64(len(observation.Content)), false)
		}
		return orchestrator.recordCouncilObservation(ctx, claim, record, item, execution, observation, at)
	default:
		return orchestrator.replaceCouncilExecution(ctx, claim, record, execution, "agent.observation_status_invalid",
			failedExecutionMayRetry, at, observation.Usage, int64(len(observation.Content)), false)
	}
}

func (orchestrator *Orchestrator) recordCouncilObservation(ctx context.Context, claim ActionClaim, record GoalRecord,
	item goal.WorkItem, execution ExecutionRecord, observation ports.AgentObservation, at time.Time,
) error {
	role, ok := councilRole(execution)
	if !ok {
		return &StateError{Code: StateConflict}
	}
	round, err := councilAttachment(record, item, execution)
	if err != nil {
		return orchestrator.quarantine(ctx, claim, "council.attachment_invalid")
	}
	payload, err := council.DecodeContribution(observation.Content)
	if err != nil || payload.Role != role || payload.SubjectDigest != string(round.SubjectDigest) {
		return orchestrator.replaceCouncilExecution(ctx, claim, record, execution, "council.contribution_invalid",
			failedExecutionMayRetry, at, observation.Usage, int64(len(observation.Content)), false)
	}
	stored, err := orchestrator.publishTestArtifact(ctx, ports.PutArtifactRequest{MediaType: council.ContributionMediaType, Content: observation.Content})
	if err != nil {
		return orchestrator.replaceCouncilExecution(ctx, claim, record, execution, "artifact.store_failed",
			failedExecutionMayRetry, at, observation.Usage, int64(len(observation.Content)), false)
	}
	fact, err := council.NewContributionFact(council.ContributionFact{SubjectDigest: string(round.SubjectDigest), Role: role, Ballot: payload.Ballot,
		ExecutionRef: execution.Ref.String(), ExecutionAttempt: execution.AttemptNo, LaunchReceiptRef: execution.LaunchReceiptRef,
		ExternalRef: execution.ExternalRef, ArtifactRef: stored.Ref.String(), ArtifactDigest: stored.Digest,
		IdempotencyKey: execution.IdempotencyKey, Contribution: payload})
	if err != nil {
		return orchestrator.quarantine(ctx, claim, "council.contribution_invalid")
	}
	artifact := ArtifactRecord{OccurrenceRef: "artifact-occurrence:council:" + execution.Ref.String(), Kind: ArtifactKindCouncilContribution,
		Stored: stored, GoalRef: record.Goal.Ref(), WorkItemRef: execution.WorkItemRef, ExecutionRef: execution.Ref,
		ExecutionAttempt: execution.AttemptNo, PlanGeneration: execution.PlanGeneration, WorkItemGeneration: item.Revision(),
		AppSpecGeneration: execution.AppSpecGeneration, SpecHash: execution.SpecHash, CreatedAt: at.UTC()}
	facts := append(councilFactsForDecision(record.CouncilFacts, round.SubjectDigest), fact)
	decision, err := council.Evaluate(round.Subject, facts)
	if err != nil {
		return orchestrator.quarantine(ctx, claim, "council.contribution_invalid")
	}
	var persisted *CouncilDecisionRecord
	if decision.Outcome != council.OutcomePending {
		digest, ok := councilDigest(decision.Digest)
		if !ok {
			return errors.New("council.decision_invalid")
		}
		persisted = &CouncilDecisionRecord{Ref: "council-decision:" + string(round.SubjectDigest), RoundRef: round.Ref,
			SubjectDigest: round.SubjectDigest, Decision: decision, DecisionDigest: digest, RecordedAt: at.UTC()}
	}
	execution.State, execution.FinishedAt = ExecutionSucceeded, at.UTC()
	settlement, err := settlementFor(record, execution, observation.Usage, int64(len(observation.Content)), at)
	if err != nil {
		return err
	}
	events := []EventRecord{{Ref: "event:council-contribution:" + execution.Ref.String(), Kind: "council.contribution_recorded",
		GoalRef: record.Goal.Ref(), WorkItemRef: execution.WorkItemRef, ExecutionRef: execution.Ref, OccurredAt: at.UTC()}}
	if persisted != nil {
		events = append(events, EventRecord{Ref: "event:council-decision:" + string(round.SubjectDigest), Kind: "council.decision_recorded",
			GoalRef: record.Goal.Ref(), WorkItemRef: execution.WorkItemRef, ExecutionRef: execution.Ref, OccurredAt: at.UTC()})
	}
	return orchestrator.state.RecordCouncilContribution(ctx, CouncilContributionState{Claim: claim,
		ExpectedGoalRevision: record.Goal.Revision(), ExpectedItemRevision: artifact.WorkItemGeneration,
		Execution: execution, Artifact: artifact, Fact: fact, Decision: persisted, BudgetSettlement: settlement,
		Events: events, OperationAt: at.UTC()})
}

func councilFactsForDecision(all []council.ContributionFact, digest CouncilSubjectDigest) []council.ContributionFact {
	result := make([]council.ContributionFact, 0, 3)
	for _, fact := range all {
		if fact.SubjectDigest == string(digest) {
			result = append(result, fact)
		}
	}
	return result
}
