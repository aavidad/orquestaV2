package orquestarunqueue

import (
	"strings"
	"time"
)

const missingFairnessGroupPauseV0 = "pause"

type runQueueFairnessEvaluationV0 struct {
	groupRef    string
	paused      bool
	boost       int
	reasonCodes []string
}

func evaluateCandidateFairnessV0(
	candidate RunSchedulingCandidateV0,
	policy RunQueueRankingPolicyV0,
) runQueueFairnessEvaluationV0 {
	groupRef, missing := candidateFairnessGroupRefV0(candidate, policy)
	reasons := []string{}
	if missing {
		reasons = append(reasons, RunQueueFairnessGroupMissingV0)
	}
	paused := fairnessGroupPausedV0(groupRef, policy)
	if missing && strings.EqualFold(policy.MissingFairnessGroupPolicy, missingFairnessGroupPauseV0) {
		paused = true
	}
	if paused {
		reasons = append(reasons, RunQueueFairnessGroupPausedV0)
	}
	boost := fairnessGroupBoostV0(groupRef, candidate, policy)
	if boost > 0 {
		reasons = append(reasons, RunQueueFairnessGroupBoostedV0)
	}
	return runQueueFairnessEvaluationV0{
		groupRef:    groupRef,
		paused:      paused,
		boost:       boost,
		reasonCodes: compactRunQueueStringsV0(reasons),
	}
}

func candidateFairnessGroupRefV0(
	candidate RunSchedulingCandidateV0,
	policy RunQueueRankingPolicyV0,
) (string, bool) {
	groupRef := strings.TrimSpace(candidate.FairnessGroupRef)
	if groupRef != "" {
		return groupRef, false
	}
	if appRef := strings.TrimSpace(candidate.AppRef); appRef != "" {
		return "app:" + appRef, true
	}
	if defaultRef := strings.TrimSpace(policy.DefaultFairnessGroupRef); defaultRef != "" {
		return defaultRef, true
	}
	return "run:" + strings.TrimSpace(candidate.RunRef), true
}

func fairnessGroupPausedV0(groupRef string, policy RunQueueRankingPolicyV0) bool {
	if policy.MaxRunsPerFairnessGroup <= 0 || policy.FairnessWindowSeconds <= 0 {
		return false
	}
	if !fairnessGroupWithinWindowV0(groupRef, policy) {
		return false
	}
	return policy.FairnessGroupRunCounts[groupRef] >= policy.MaxRunsPerFairnessGroup
}

func fairnessGroupBoostV0(
	groupRef string,
	candidate RunSchedulingCandidateV0,
	policy RunQueueRankingPolicyV0,
) int {
	if policy.FairnessBoostPerWindow <= 0 || policy.FairnessBoostAfterSeconds <= 0 ||
		policy.Now.IsZero() {
		return 0
	}
	last := policy.FairnessGroupLastSelectedAt[groupRef]
	if last.IsZero() {
		last = candidate.UpdatedAt
	}
	if last.IsZero() {
		return 0
	}
	if policy.Now.Sub(last) < time.Duration(policy.FairnessBoostAfterSeconds)*time.Second {
		return 0
	}
	return policy.FairnessBoostPerWindow
}

func fairnessGroupWithinWindowV0(groupRef string, policy RunQueueRankingPolicyV0) bool {
	if policy.Now.IsZero() {
		return false
	}
	last := policy.FairnessGroupLastSelectedAt[groupRef]
	if last.IsZero() {
		return false
	}
	window := time.Duration(policy.FairnessWindowSeconds) * time.Second
	return policy.Now.Sub(last) < window
}
