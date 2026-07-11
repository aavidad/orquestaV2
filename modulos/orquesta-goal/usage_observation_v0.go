package orquestagoal

import (
	"strings"
	"time"
)

// IsZero lets encoding/json omit an absent observation with the omitzero tag.
func (observation GoalUsageObservationV0) IsZero() bool {
	return observation.TokensAccumulated == 0 &&
		observation.RuntimeSeconds == 0 &&
		observation.ObservedAt == "" &&
		len(observation.EvidenceRefs) == 0 &&
		observation.SourceRef == ""
}

// NormalizeGoalUsageObservationV0 returns a canonical, independently owned
// observation. Invalid timestamps and negative metrics remain visible for
// structural validation instead of being silently repaired.
func NormalizeGoalUsageObservationV0(observation GoalUsageObservationV0) GoalUsageObservationV0 {
	observation.ObservedAt = strings.TrimSpace(observation.ObservedAt)
	if observedAt, err := time.Parse(time.RFC3339, observation.ObservedAt); err == nil {
		observation.ObservedAt = observedAt.UTC().Format(time.RFC3339Nano)
	}
	observation.EvidenceRefs = compactGoalStringsV0(observation.EvidenceRefs)
	observation.SourceRef = strings.TrimSpace(observation.SourceRef)
	return observation
}

// GoalUsageObservationEmptyV0 reports whether an observation carries no usage
// or provenance. It preserves compatibility for result receipts that predate
// usage observation.
func GoalUsageObservationEmptyV0(observation GoalUsageObservationV0) bool {
	observation = NormalizeGoalUsageObservationV0(observation)
	return observation.TokensAccumulated == 0 &&
		observation.RuntimeSeconds == 0 &&
		observation.ObservedAt == "" &&
		len(observation.EvidenceRefs) == 0 &&
		observation.SourceRef == ""
}

func ValidateGoalUsageObservationV0(observation GoalUsageObservationV0) []GoalWorkIssueV0 {
	observation = NormalizeGoalUsageObservationV0(observation)
	if GoalUsageObservationEmptyV0(observation) {
		return nil
	}

	var issues []GoalWorkIssueV0
	if observation.TokensAccumulated < 0 {
		issues = append(issues, GoalWorkIssueV0{Code: ErrGoalUsageObservationInvalidV0, Field: "usage_observation.tokens_accumulated"})
	}
	if observation.RuntimeSeconds < 0 {
		issues = append(issues, GoalWorkIssueV0{Code: ErrGoalUsageObservationInvalidV0, Field: "usage_observation.runtime_seconds"})
	}
	if observation.ObservedAt != "" {
		if _, err := time.Parse(time.RFC3339, observation.ObservedAt); err != nil {
			issues = append(issues, GoalWorkIssueV0{Code: ErrGoalUsageObservationTimestampInvalidV0, Field: "usage_observation.observed_at"})
		}
	} else {
		issues = append(issues, GoalWorkIssueV0{Code: ErrGoalUsageObservationInvalidV0, Field: "usage_observation.observed_at"})
	}
	for _, evidenceRef := range observation.EvidenceRefs {
		validateRequiredGoalRefV0(&issues, "usage_observation.evidence_refs", evidenceRef)
	}
	if len(observation.EvidenceRefs) == 0 {
		issues = append(issues, GoalWorkIssueV0{Code: ErrGoalUsageObservationEvidenceRequiredV0, Field: "usage_observation.evidence_refs"})
	}
	if observation.SourceRef == "" {
		issues = append(issues, GoalWorkIssueV0{Code: ErrGoalUsageObservationSourceRequiredV0, Field: "usage_observation.source_ref"})
	} else {
		validateGoalRefsV0(&issues, "usage_observation.source_ref", observation.SourceRef)
	}
	return issues
}
