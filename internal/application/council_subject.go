package application

import (
	"errors"

	"orquesta/internal/council"
	"orquesta/internal/goal"
)

// councilSubject rebuilds the only admissible Council input: the exact,
// approved V18 subject and gate for the immutable author change.
func councilSubject(record GoalRecord, item goal.WorkItem, author ExecutionRecord,
	change ChangeSet, policy council.Policy, testPolicy TestAttestationPolicy,
) (council.Subject, error) {
	if council.ValidatePolicy(policy) != nil || policy == council.PolicySkipByOperator ||
		author.State != ExecutionAwaitingIntegration {
		return council.Subject{}, errors.New("council.subject_invalid")
	}
	reviewSubject, gate, err := reviewGateForChange(record, author, change, testPolicy)
	if err != nil || gate.Status != "approved" {
		return council.Subject{}, errors.New("council.review_gate_required")
	}
	return council.NewSubject(council.Subject{
		ProjectRef: record.Goal.Project().String(), ReviewSubjectDigest: reviewSubject.Digest(),
		ReviewGateDigest: gate.Digest, Policy: policy, GoalRef: record.Goal.Ref().String(),
		WorkItemRef: item.Ref().String(), ChangeSetRef: change.Ref.String(), SpecHash: author.SpecHash,
		PlanGeneration: uint64(author.PlanGeneration), WorkItemGeneration: uint64(item.Revision()),
		AppSpecGeneration: uint64(author.AppSpecGeneration),
	})
}

func councilRoundFor(record GoalRecord, digest CouncilSubjectDigest) (CouncilRoundRecord, bool) {
	for _, round := range record.CouncilRounds {
		if round.SubjectDigest == digest {
			return round, true
		}
	}
	return CouncilRoundRecord{}, false
}
