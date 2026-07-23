package sqlite

import (
	"errors"
	"reflect"

	"orquesta/internal/application"
	"orquesta/internal/council"
	"orquesta/internal/goal"
)

func validateGoalRecordCouncil(
	record application.GoalRecord,
	items map[goal.WorkItemRef]goal.WorkItem,
	executions map[goal.ExecutionRef]application.ExecutionRecord,
) error {
	rounds := make(map[application.CouncilSubjectDigest]application.CouncilRoundRecord, len(record.CouncilRounds))
	for _, round := range record.CouncilRounds {
		item, found := items[round.WorkItemRef]
		policy, policyFound := item.CouncilPolicy()
		if !found || !policyFound || policy != round.Subject.Policy ||
			round.Subject.GoalRef != record.Goal.Ref().String() ||
			round.Subject.WorkItemRef != round.WorkItemRef.String() ||
			round.SubjectDigest != application.CouncilSubjectDigest(round.Subject.Digest()) ||
			round.Subject.Policy == council.PolicySkipByOperator {
			return errors.New("sqlite.goal_record_council_round_invalid")
		}
		if application.ValidatePersistedCouncilSubject(record, round.Subject) != nil {
			return errors.New("sqlite.goal_record_council_subject_invalid")
		}
		if _, duplicate := rounds[round.SubjectDigest]; duplicate {
			return errors.New("sqlite.goal_record_council_round_duplicate")
		}
		rounds[round.SubjectDigest] = round
	}
	roles := make(map[application.CouncilSubjectDigest]map[council.Role]bool, len(rounds))
	for _, fact := range record.CouncilFacts {
		digest := application.CouncilSubjectDigest(fact.SubjectDigest)
		round, found := rounds[digest]
		executionRef, refErr := goal.NewExecutionRef(fact.ExecutionRef)
		execution, executionFound := executions[executionRef]
		role, roleFound := councilExecutionRole(execution)
		if !found || refErr != nil || !executionFound || !roleFound || role != fact.Role ||
			execution.GoalRef != round.GoalRef || execution.WorkItemRef != round.WorkItemRef ||
			execution.CouncilSubjectDigest != digest || execution.ReviewSubjectDigest != "" ||
			execution.AttemptNo != fact.ExecutionAttempt || execution.LaunchReceiptRef != fact.LaunchReceiptRef ||
			execution.ExternalRef != fact.ExternalRef || execution.State != application.ExecutionSucceeded {
			return errors.New("sqlite.goal_record_council_fact_invalid")
		}
		matches := 0
		for _, artifact := range record.Artifacts {
			if artifact.Kind == application.ArtifactKindCouncilContribution &&
				artifact.ExecutionRef == execution.Ref && artifact.Stored.Ref.String() == fact.ArtifactRef &&
				artifact.Stored.Digest == fact.ArtifactDigest && artifact.Stored.MediaType == council.ContributionMediaType {
				matches++
			}
		}
		if matches != 1 {
			return errors.New("sqlite.goal_record_council_artifact_invalid")
		}
		if roles[digest] == nil {
			roles[digest] = make(map[council.Role]bool, 3)
		}
		if roles[digest][fact.Role] {
			return errors.New("sqlite.goal_record_council_role_duplicate")
		}
		roles[digest][fact.Role] = true
	}
	decisions := make(map[application.CouncilSubjectDigest]bool, len(record.CouncilDecisions))
	for _, decision := range record.CouncilDecisions {
		round, found := rounds[decision.SubjectDigest]
		evaluated, err := council.Evaluate(round.Subject, councilFactsFor(record.CouncilFacts, decision.SubjectDigest))
		if !found || err != nil || evaluated.Outcome == council.OutcomePending ||
			decision.RoundRef != round.Ref || !reflect.DeepEqual(decision.Decision, evaluated) ||
			decision.DecisionDigest != application.CouncilSubjectDigest(evaluated.Digest) ||
			decisions[decision.SubjectDigest] {
			return errors.New("sqlite.goal_record_council_decision_invalid")
		}
		decisions[decision.SubjectDigest] = true
	}
	for digest, byRole := range roles {
		if len(byRole) == 3 && !decisions[digest] {
			return errors.New("sqlite.goal_record_council_decision_missing")
		}
	}
	skips := make(map[application.CouncilSubjectDigest]bool, len(record.CouncilSkips))
	for _, skip := range record.CouncilSkips {
		itemRef, err := goal.NewWorkItemRef(skip.Subject.WorkItemRef)
		item, found := items[itemRef]
		policy, hasPolicy := item.CouncilPolicy()
		if err != nil || !found || !hasPolicy || policy != council.PolicySkipByOperator ||
			skip.SubjectDigest != application.CouncilSubjectDigest(skip.Subject.Digest()) ||
			skip.SkipDigest != application.CouncilSubjectDigest(skip.Skip.Digest()) ||
			skips[skip.SubjectDigest] {
			return errors.New("sqlite.goal_record_council_skip_invalid")
		}
		if application.ValidatePersistedCouncilSubject(record, skip.Subject) != nil {
			return errors.New("sqlite.goal_record_council_subject_invalid")
		}
		if _, exists := rounds[skip.SubjectDigest]; exists {
			return errors.New("sqlite.goal_record_council_resolution_ambiguous")
		}
		skips[skip.SubjectDigest] = true
	}
	return nil
}
