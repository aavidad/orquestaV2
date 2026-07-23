package application

import (
	"errors"
	"strings"
	"time"

	"orquesta/internal/council"
	"orquesta/internal/goal"
	"orquesta/internal/identity"
)

// CouncilSubjectDigest names immutable Council subject without creating a
// second authority beside GoalRecord.
type CouncilSubjectDigest string

type CouncilRoundRecord struct {
	Ref            string
	GoalRef        goal.GoalRef
	WorkItemRef    goal.WorkItemRef
	ChangeSetRef   string
	Subject        council.Subject
	SubjectDigest  CouncilSubjectDigest
	OpenedBy       identity.PrincipalRef
	OpenedAt       time.Time
	IdempotencyKey string
}

type CouncilDecisionRecord struct {
	Ref           string
	RoundRef      string
	SubjectDigest CouncilSubjectDigest
	Decision      council.Decision
	RecordedAt    time.Time
}

type CouncilSkipRecord struct {
	Ref           string
	Subject       council.Subject
	SubjectDigest CouncilSubjectDigest
	Skip          council.Skip
	RecordedAt    time.Time
}

// CouncilResolution is integration's one-of proof. A skip is never encoded as
// a synthetic decision.
type CouncilResolution struct {
	SubjectDigest  CouncilSubjectDigest
	DecisionRef    string
	DecisionDigest string
	SkipRef        string
	SkipDigest     string
}

func (resolution CouncilResolution) Validate() error {
	if !validCouncilDigest(string(resolution.SubjectDigest)) {
		return errors.New("council.resolution_invalid")
	}
	decision := strings.TrimSpace(resolution.DecisionRef) != "" || strings.TrimSpace(resolution.DecisionDigest) != ""
	skip := strings.TrimSpace(resolution.SkipRef) != "" || strings.TrimSpace(resolution.SkipDigest) != ""
	if decision == skip || (decision && (!validCouncilRef(resolution.DecisionRef) || !validCouncilDigest(resolution.DecisionDigest))) ||
		(skip && (!validCouncilRef(resolution.SkipRef) || !validCouncilDigest(resolution.SkipDigest))) {
		return errors.New("council.resolution_invalid")
	}
	return nil
}

func councilResolutionEqual(left, right *CouncilResolution) bool {
	if left == nil || right == nil {
		return left == right
	}
	return *left == *right
}

func validCouncilDigest(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, char := range value {
		if !(char >= '0' && char <= '9' || char >= 'a' && char <= 'f') {
			return false
		}
	}
	return true
}

func validCouncilRef(value string) bool {
	return strings.TrimSpace(value) != "" && strings.TrimSpace(value) == value
}

func councilPurpose(role council.Role) (ExecutionPurpose, bool) {
	switch role {
	case council.RoleProposer:
		return ExecutionPurposeCouncilProposer, true
	case council.RoleCritic:
		return ExecutionPurposeCouncilCritic, true
	case council.RoleArbiter:
		return ExecutionPurposeCouncilArbiter, true
	default:
		return "", false
	}
}

func councilRole(execution ExecutionRecord) (council.Role, bool) {
	switch execution.Purpose {
	case ExecutionPurposeCouncilProposer:
		return council.RoleProposer, true
	case ExecutionPurposeCouncilCritic:
		return council.RoleCritic, true
	case ExecutionPurposeCouncilArbiter:
		return council.RoleArbiter, true
	default:
		return "", false
	}
}

func validateCouncilIntegrationResolution(record GoalRecord, policy council.Policy, resolution *CouncilResolution) error {
	if resolution == nil || resolution.Validate() != nil {
		return &StateError{Code: StateConflict}
	}
	if resolution.DecisionRef != "" {
		if policy == council.PolicySkipByOperator {
			return &StateError{Code: StateConflict}
		}
		for _, decision := range record.CouncilDecisions {
			if decision.Ref == resolution.DecisionRef && decision.SubjectDigest == resolution.SubjectDigest &&
				decision.Decision.Digest == resolution.DecisionDigest && decision.Decision.Outcome == council.OutcomeAccepted {
				return nil
			}
		}
		return &StateError{Code: StateConflict}
	}
	if policy != council.PolicySkipByOperator {
		return &StateError{Code: StateConflict}
	}
	for _, skip := range record.CouncilSkips {
		if skip.Ref == resolution.SkipRef && skip.SubjectDigest == resolution.SubjectDigest && skip.Skip.Digest() == resolution.SkipDigest {
			return nil
		}
	}
	return &StateError{Code: StateConflict}
}
