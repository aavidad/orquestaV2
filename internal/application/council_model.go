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

type CouncilRoundOpener string

const (
	CouncilRoundOpenerAuto     CouncilRoundOpener = "auto"
	CouncilRoundOpenerDirector CouncilRoundOpener = "director"
)

type CouncilRoundRecord struct {
	Ref                     string
	GoalRef                 goal.GoalRef
	WorkItemRef             goal.WorkItemRef
	ChangeSetRef            string
	Subject                 council.Subject
	SubjectDigest           CouncilSubjectDigest
	OpenedBy                identity.PrincipalRef
	OpenedAt                time.Time
	IdempotencyKey          string
	Opener                  CouncilRoundOpener
	DirectorFence           uint64
	RequestRef              string
	RequestFingerprint      string
	AuthorizationReceiptRef string
}

type CouncilDecisionRecord struct {
	Ref            string
	RoundRef       string
	SubjectDigest  CouncilSubjectDigest
	Decision       council.Decision
	DecisionDigest CouncilSubjectDigest
	RecordedAt     time.Time
}

type CouncilSkipRecord struct {
	Ref                     string
	Subject                 council.Subject
	SubjectDigest           CouncilSubjectDigest
	Skip                    council.Skip
	SkipDigest              CouncilSubjectDigest
	RecordedAt              time.Time
	RequestRef              string
	RequestFingerprint      string
	AuthorizationReceiptRef string
}

// CouncilResolution is integration's one-of proof. A skip is never encoded as
// a synthetic decision.
type CouncilResolution struct {
	SubjectDigest  CouncilSubjectDigest
	DecisionRef    string
	DecisionDigest CouncilSubjectDigest
	SkipRef        string
	SkipDigest     CouncilSubjectDigest
}

func (resolution CouncilResolution) Validate() error {
	if !validCouncilDigest(string(resolution.SubjectDigest)) {
		return errors.New("council.resolution_invalid")
	}
	decision := strings.TrimSpace(resolution.DecisionRef) != "" || strings.TrimSpace(string(resolution.DecisionDigest)) != ""
	skip := strings.TrimSpace(resolution.SkipRef) != "" || strings.TrimSpace(string(resolution.SkipDigest)) != ""
	if decision == skip || (decision && (!validCouncilRef(resolution.DecisionRef) || !validCouncilDigest(string(resolution.DecisionDigest)))) ||
		(skip && (!validCouncilRef(resolution.SkipRef) || !validCouncilDigest(string(resolution.SkipDigest)))) {
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
	if len(value) != len("sha256:")+64 || !strings.HasPrefix(value, "sha256:") {
		return false
	}
	for _, char := range value[len("sha256:"):] {
		if !(char >= '0' && char <= '9' || char >= 'a' && char <= 'f') {
			return false
		}
	}
	return true
}

func validCouncilRef(value string) bool {
	return len(value) > 0 && len(value) <= 512 && strings.TrimSpace(value) == value &&
		!strings.ContainsAny(value, "\r\n\x00")
}

func councilDigest(value string) (CouncilSubjectDigest, bool) {
	if validCouncilDigest(value) {
		return CouncilSubjectDigest(value), true
	}
	return "", false
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
				decision.DecisionDigest == resolution.DecisionDigest && decision.Decision.Outcome == council.OutcomeAccepted {
				return nil
			}
		}
		return &StateError{Code: StateConflict}
	}
	if policy != council.PolicySkipByOperator {
		return &StateError{Code: StateConflict}
	}
	for _, skip := range record.CouncilSkips {
		if skip.Ref == resolution.SkipRef && skip.SubjectDigest == resolution.SubjectDigest && skip.SkipDigest == resolution.SkipDigest {
			return nil
		}
	}
	return &StateError{Code: StateConflict}
}
