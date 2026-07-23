// Package council contains pure, deterministic rules for a three-role council.
package council

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"hash"
	"io"
	"strconv"
	"strings"
	"time"
)

const (
	ContributionSchema    = "orquesta.council.contribution.v1"
	ContributionMediaType = "application/vnd.orquesta.council-contribution+json"
)

var (
	ErrInvalidSubject      = errors.New("council.subject_invalid")
	ErrInvalidPolicy       = errors.New("council.policy_invalid")
	ErrInvalidContribution = errors.New("council.contribution_invalid")
	ErrInvalidFact         = errors.New("council.fact_invalid")
	ErrSubjectMismatch     = errors.New("council.subject_mismatch")
	ErrInvalidSkip         = errors.New("council.skip_invalid")
)

type Policy string

const (
	PolicyAuto           Policy = "auto"
	PolicyRequired       Policy = "required"
	PolicySkipByOperator Policy = "skip_by_operator"
)

func ValidatePolicy(value Policy) error {
	if !validPolicy(value) {
		return ErrInvalidPolicy
	}
	return nil
}

type Role string

const (
	RoleProposer Role = "proposer"
	RoleCritic   Role = "critic"
	RoleArbiter  Role = "arbiter"
)

type Ballot string

const (
	BallotAccept       Ballot = "accept"
	BallotReject       Ballot = "reject"
	BallotAbstain      Ballot = "abstain"
	BallotSecurityVeto Ballot = "security_veto"
)

type Outcome string

const (
	OutcomePending         Outcome = "pending"
	OutcomeAccepted        Outcome = "accepted"
	OutcomeRejected        Outcome = "rejected"
	OutcomeNoConsensus     Outcome = "no_consensus"
	OutcomeBlockedSecurity Outcome = "blocked_security"
)

// Subject binds council to the exact V18-approved review gate and immutable policy.
type Subject struct {
	ProjectRef, ReviewSubjectDigest, ReviewGateDigest string
	Policy                                            Policy
	GoalRef, WorkItemRef, ChangeSetRef, SpecHash      string
	PlanGeneration, WorkItemGeneration                uint64
	AppSpecGeneration                                 uint64
}

func NewSubject(subject Subject) (Subject, error) {
	if !validOpaque(subject.ProjectRef, 512) || !validDigest(subject.ReviewSubjectDigest) ||
		!validDigest(subject.ReviewGateDigest) || ValidatePolicy(subject.Policy) != nil ||
		!validOpaque(subject.GoalRef, 512) || !validOpaque(subject.WorkItemRef, 512) ||
		!validOpaque(subject.ChangeSetRef, 512) || !validHash(subject.SpecHash) ||
		subject.PlanGeneration == 0 || subject.WorkItemGeneration == 0 || subject.AppSpecGeneration == 0 {
		return Subject{}, ErrInvalidSubject
	}
	return subject, nil
}

func (subject Subject) Digest() string {
	return digest("orquesta.council-subject.v1", subject.ProjectRef, subject.ReviewSubjectDigest,
		subject.ReviewGateDigest, string(subject.Policy))
}

type Evidence struct {
	Kind string `json:"kind"`
	Ref  string `json:"ref"`
}

// Contribution is one strict artifact envelope. Its body stays opaque to council.
type Contribution struct {
	Schema        string     `json:"schema"`
	SubjectDigest string     `json:"subject_digest"`
	Role          Role       `json:"role"`
	Body          string     `json:"body"`
	Ballot        Ballot     `json:"ballot"`
	Evidence      []Evidence `json:"evidence"`
}

func DecodeContribution(content []byte) (Contribution, error) {
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	var contribution Contribution
	if err := decoder.Decode(&contribution); err != nil {
		return Contribution{}, ErrInvalidContribution
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return Contribution{}, ErrInvalidContribution
	}
	return NewContribution(contribution)
}

func NewContribution(contribution Contribution) (Contribution, error) {
	if contribution.Schema != ContributionSchema || !validDigest(contribution.SubjectDigest) ||
		!validRole(contribution.Role) || !validText(contribution.Body, 8000) || !validBallot(contribution.Ballot) ||
		len(contribution.Evidence) == 0 || len(contribution.Evidence) > 64 || !validEvidence(contribution.Evidence) ||
		(contribution.Ballot == BallotSecurityVeto && !hasSecurityVetoEvidence(contribution.Evidence)) {
		return Contribution{}, ErrInvalidContribution
	}
	return contribution, nil
}

func (contribution Contribution) Digest() string {
	values := []string{"orquesta.council-contribution.v1", contribution.Schema, contribution.SubjectDigest,
		string(contribution.Role), contribution.Body, string(contribution.Ballot), strconv.Itoa(len(contribution.Evidence))}
	for _, evidence := range contribution.Evidence {
		values = append(values, evidence.Kind, evidence.Ref)
	}
	return digest(values...)
}

// ContributionFact binds one accredited launch and its sole CAS envelope.
type ContributionFact struct {
	SubjectDigest                 string
	Role                          Role
	Ballot                        Ballot
	ExecutionRef                  string
	ExecutionAttempt              uint64
	LaunchReceiptRef, ExternalRef string
	ArtifactRef, ArtifactDigest   string
	IdempotencyKey                string
	Contribution                  Contribution
}

type BallotFact = ContributionFact

func NewContributionFact(fact ContributionFact) (ContributionFact, error) {
	contribution, err := NewContribution(fact.Contribution)
	if err != nil || !validDigest(fact.SubjectDigest) || !validRole(fact.Role) || !validBallot(fact.Ballot) ||
		fact.SubjectDigest != contribution.SubjectDigest || fact.Role != contribution.Role || fact.Ballot != contribution.Ballot ||
		!validOpaque(fact.ExecutionRef, 512) || fact.ExecutionAttempt == 0 || !validOpaque(fact.LaunchReceiptRef, 1024) ||
		!validOpaque(fact.ExternalRef, 1024) || !validOpaque(fact.ArtifactRef, 1024) || !validHash(fact.ArtifactDigest) ||
		!validOpaque(fact.IdempotencyKey, 512) {
		return ContributionFact{}, ErrInvalidFact
	}
	fact.Contribution = contribution
	return fact, nil
}

func (fact ContributionFact) Digest() string {
	return digest("orquesta.council-fact.v1", fact.SubjectDigest, string(fact.Role), string(fact.Ballot),
		fact.ExecutionRef, strconv.FormatUint(fact.ExecutionAttempt, 10), fact.LaunchReceiptRef, fact.ExternalRef,
		fact.ArtifactRef, fact.ArtifactDigest, fact.IdempotencyKey, fact.Contribution.Digest())
}

type Dissent struct {
	Role       Role
	Ballot     Ballot
	FactDigest string
}

type Decision struct {
	SubjectDigest string
	Outcome       Outcome
	Digest        string
	Dissent       []Dissent
}

// Evaluate waits for exactly all three roles; no normal or veto result is early.
func Evaluate(subject Subject, facts []ContributionFact) (Decision, error) {
	if _, err := NewSubject(subject); err != nil {
		return Decision{}, err
	}
	byRole := make(map[Role]ContributionFact, 3)
	seenExecution, seenLaunch, seenExternal, seenKey := map[string]bool{}, map[string]bool{}, map[string]bool{}, map[string]bool{}
	for _, fact := range facts {
		canonical, err := NewContributionFact(fact)
		if err != nil {
			return Decision{}, err
		}
		if canonical.SubjectDigest != subject.Digest() || seenExecution[canonical.ExecutionRef] || seenLaunch[canonical.LaunchReceiptRef] ||
			seenExternal[canonical.ExternalRef] || seenKey[canonical.IdempotencyKey] || byRole[canonical.Role].Role != "" {
			return Decision{}, ErrSubjectMismatch
		}
		byRole[canonical.Role] = canonical
		seenExecution[canonical.ExecutionRef], seenLaunch[canonical.LaunchReceiptRef], seenExternal[canonical.ExternalRef], seenKey[canonical.IdempotencyKey] = true, true, true, true
	}
	if len(byRole) < 3 {
		return Decision{SubjectDigest: subject.Digest(), Outcome: OutcomePending}, nil
	}
	if len(byRole) != 3 {
		return Decision{}, ErrSubjectMismatch
	}
	accepts, rejects, veto := 0, 0, false
	for _, role := range Roles() {
		switch byRole[role].Ballot {
		case BallotAccept:
			accepts++
		case BallotReject:
			rejects++
		case BallotSecurityVeto:
			veto = true
		}
	}
	outcome := OutcomeNoConsensus
	if veto {
		outcome = OutcomeBlockedSecurity
	} else if accepts >= 2 {
		outcome = OutcomeAccepted
	} else if rejects >= 2 {
		outcome = OutcomeRejected
	}
	decision := Decision{SubjectDigest: subject.Digest(), Outcome: outcome}
	for _, role := range Roles() {
		fact := byRole[role]
		if dissent(outcome, fact.Ballot) {
			decision.Dissent = append(decision.Dissent, Dissent{Role: role, Ballot: fact.Ballot, FactDigest: fact.Digest()})
		}
	}
	decision.Digest = decisionDigest(decision, byRole)
	return decision, nil
}

func decisionDigest(decision Decision, facts map[Role]ContributionFact) string {
	values := []string{"orquesta.council-decision.v1", decision.SubjectDigest, string(decision.Outcome)}
	for _, role := range Roles() {
		values = append(values, string(role), facts[role].Digest())
	}
	for _, dissent := range decision.Dissent {
		values = append(values, string(dissent.Role), string(dissent.Ballot), dissent.FactDigest)
	}
	return digest(values...)
}

// Skip is an authorization fact payload. Principal permission is application-owned.
type Skip struct {
	PrincipalRef, Reason, SpecHash, IdempotencyKey, CouncilSubjectDigest string
	RecordedAtUTC                                                        time.Time
}

func NewSkip(subject Subject, skip Skip) (Skip, error) {
	if _, err := NewSubject(subject); err != nil || subject.Policy != PolicySkipByOperator ||
		!validOpaque(skip.PrincipalRef, 512) || !validText(skip.Reason, 4000) || skip.RecordedAtUTC.IsZero() ||
		skip.RecordedAtUTC.Location() != time.UTC || !validHash(skip.SpecHash) ||
		!validOpaque(skip.IdempotencyKey, 512) || skip.SpecHash != subject.SpecHash || skip.CouncilSubjectDigest != subject.Digest() {
		return Skip{}, ErrInvalidSkip
	}
	return skip, nil
}

func (skip Skip) Digest() string {
	return digest("orquesta.council-skip.v1", skip.PrincipalRef, skip.Reason, skip.RecordedAtUTC.UTC().Format(time.RFC3339Nano),
		skip.SpecHash, skip.IdempotencyKey, skip.CouncilSubjectDigest)
}

func Roles() []Role { return []Role{RoleProposer, RoleCritic, RoleArbiter} }
func dissent(outcome Outcome, ballot Ballot) bool {
	switch outcome {
	case OutcomeAccepted:
		return ballot == BallotReject || ballot == BallotAbstain
	case OutcomeRejected:
		return ballot == BallotAccept || ballot == BallotAbstain
	case OutcomeNoConsensus:
		return true
	case OutcomeBlockedSecurity:
		return ballot != BallotSecurityVeto
	}
	return false
}
func validPolicy(value Policy) bool {
	return value == PolicyAuto || value == PolicyRequired || value == PolicySkipByOperator
}
func validRole(value Role) bool {
	return value == RoleProposer || value == RoleCritic || value == RoleArbiter
}
func validBallot(value Ballot) bool {
	return value == BallotAccept || value == BallotReject || value == BallotAbstain || value == BallotSecurityVeto
}
func validEvidence(values []Evidence) bool {
	for _, value := range values {
		if !validOpaque(value.Kind, 128) || !validOpaque(value.Ref, 1024) {
			return false
		}
	}
	return true
}
func hasSecurityVetoEvidence(values []Evidence) bool {
	for _, value := range values {
		if value.Kind == "security_veto" {
			return true
		}
	}
	return false
}
func validOpaque(value string, maximum int) bool {
	return value != "" && len(value) <= maximum && strings.TrimSpace(value) == value && !strings.ContainsAny(value, "\r\n\x00")
}
func validText(value string, maximum int) bool {
	return value != "" && len(value) <= maximum && strings.TrimSpace(value) != "" && !strings.ContainsRune(value, '\x00')
}
func validDigest(value string) bool {
	return strings.HasPrefix(value, "sha256:") && len(value) == len("sha256:")+sha256.Size*2 && validHex(strings.TrimPrefix(value, "sha256:"))
}
func validHash(value string) bool { return len(value) == sha256.Size*2 && validHex(value) }
func validHex(value string) bool {
	decoded, err := hex.DecodeString(value)
	return err == nil && hex.EncodeToString(decoded) == value
}
func digest(values ...string) string {
	h := sha256.New()
	for _, value := range values {
		writeField(h, value)
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil))
}
func writeField(h hash.Hash, value string) {
	var size [8]byte
	binary.BigEndian.PutUint64(size[:], uint64(len(value)))
	_, _ = h.Write(size[:])
	_, _ = h.Write([]byte(value))
}
