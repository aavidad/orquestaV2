package orquestarunqueue

import (
	"sort"
	"strings"
	"time"
)

func RankRunCandidatesV0(candidates []RunSchedulingCandidateV0, policy RunQueueRankingPolicyV0) []RankedRunCandidateV0 {
	ranked := make([]RankedRunCandidateV0, 0, len(candidates))
	for _, candidate := range candidates {
		if !IsExecutableRunStatusV0(candidate.Status) {
			continue
		}
		copied := cloneRunSchedulingCandidateV0(candidate)
		fairness := evaluateCandidateFairnessV0(copied, policy)
		copied.FairnessGroupRef = fairness.groupRef
		ranked = append(ranked, RankedRunCandidateV0{
			RunSchedulingCandidateV0: copied,
			AgingBoost:               agingBoostV0(copied, policy),
			FairnessBoost:            fairness.boost,
			FairnessPaused:           fairness.paused,
			FairnessReasonCodes:      fairness.reasonCodes,
		})
	}

	sort.SliceStable(ranked, func(i, j int) bool {
		left := ranked[i]
		right := ranked[j]
		if left.PriorityScore != right.PriorityScore {
			return left.PriorityScore > right.PriorityScore
		}
		if left.FairnessPaused != right.FairnessPaused {
			return !left.FairnessPaused
		}
		if left.FairnessBoost != right.FairnessBoost {
			return left.FairnessBoost > right.FairnessBoost
		}
		if left.AgingBoost != right.AgingBoost {
			return left.AgingBoost > right.AgingBoost
		}
		return updatedAtBeforeV0(left.UpdatedAt, right.UpdatedAt, policy.MissingUpdatedAtLast)
	})

	ranked = filterRankedRunCandidatesByWorksetV0(ranked, RunQueueWorksetPolicyV0{
		RequireClaims: policy.RequireWorksetClaims,
	})
	for index := range ranked {
		ranked[index].Rank = index + 1
	}
	return ranked
}

func IsExecutableRunStatusV0(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case RunStatusPausedV0, RunStatusDeliveredV0, RunStatusCanceledV0, RunStatusStoppedV0, RunStatusClosedV0:
		return false
	default:
		return true
	}
}

func agingBoostV0(candidate RunSchedulingCandidateV0, policy RunQueueRankingPolicyV0) int {
	if policy.Now.IsZero() || candidate.UpdatedAt.IsZero() {
		return 0
	}
	if policy.AgingAfterSeconds <= 0 || policy.AgingBoostPerStep <= 0 {
		return 0
	}
	age := policy.Now.Sub(candidate.UpdatedAt)
	after := time.Duration(policy.AgingAfterSeconds) * time.Second
	if age < after {
		return 0
	}
	stepSeconds := policy.AgingStepSeconds
	if stepSeconds <= 0 {
		stepSeconds = policy.AgingAfterSeconds
	}
	step := time.Duration(stepSeconds) * time.Second
	boost := int(age/step) * policy.AgingBoostPerStep
	if boost <= 0 {
		boost = policy.AgingBoostPerStep
	}
	if policy.MaxAgingBoost > 0 && boost > policy.MaxAgingBoost {
		return policy.MaxAgingBoost
	}
	return boost
}

func updatedAtBeforeV0(left time.Time, right time.Time, missingLast bool) bool {
	if left.Equal(right) {
		return false
	}
	if left.IsZero() || right.IsZero() {
		if !missingLast {
			return left.IsZero() && !right.IsZero()
		}
		return !left.IsZero() && right.IsZero()
	}
	return left.Before(right)
}

func cloneRunSchedulingCandidateV0(candidate RunSchedulingCandidateV0) RunSchedulingCandidateV0 {
	if candidate.EvidenceRefs != nil {
		candidate.EvidenceRefs = append([]string(nil), candidate.EvidenceRefs...)
	}
	candidate.WorksetClaims = cloneRunQueueWorksetClaimsV0(candidate.WorksetClaims)
	return candidate
}
