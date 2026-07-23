package council

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

func subject(t *testing.T, policy Policy) Subject {
	t.Helper()
	hash := strings.Repeat("a", 64)
	value, err := NewSubject(Subject{
		ProjectRef: "project:one", ReviewSubjectDigest: "sha256:" + hash, ReviewGateDigest: "sha256:" + strings.Repeat("b", 64), Policy: policy,
		GoalRef: "goal:one", WorkItemRef: "work-item:one", ChangeSetRef: "change:one", SpecHash: hash,
		PlanGeneration: 1, WorkItemGeneration: 2, AppSpecGeneration: 3,
	})
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func fact(t *testing.T, subject Subject, role Role, ballot Ballot) ContributionFact {
	t.Helper()
	evidence := []Evidence{{Kind: "proof", Ref: "evidence:" + string(role)}}
	if ballot == BallotSecurityVeto {
		evidence = []Evidence{{Kind: "security_veto", Ref: "evidence:veto"}}
	}
	artifactByte := map[Role]string{RoleProposer: "1", RoleCritic: "2", RoleArbiter: "3"}[role]
	value, err := NewContributionFact(ContributionFact{
		SubjectDigest: subject.Digest(), Role: role, Ballot: ballot, ExecutionRef: "execution:" + string(role), ExecutionAttempt: 1,
		LaunchReceiptRef: "receipt:" + string(role), ExternalRef: "external:" + string(role), ArtifactRef: "artifact:" + string(role),
		ArtifactDigest: strings.Repeat(artifactByte, 64), IdempotencyKey: "key:" + string(role),
		Contribution: Contribution{Schema: ContributionSchema, SubjectDigest: subject.Digest(), Role: role, Body: "body:" + string(role), Ballot: ballot, Evidence: evidence},
	})
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func TestSubjectDigestBindsOnlyContractFieldsWithLengthFraming(t *testing.T) {
	base := subject(t, PolicyAuto)
	for name, mutate := range map[string]func(*Subject){
		"project":        func(v *Subject) { v.ProjectRef = "project:two" },
		"review subject": func(v *Subject) { v.ReviewSubjectDigest = "sha256:" + strings.Repeat("c", 64) },
		"review gate":    func(v *Subject) { v.ReviewGateDigest = "sha256:" + strings.Repeat("d", 64) },
		"policy":         func(v *Subject) { v.Policy = PolicyRequired },
	} {
		t.Run(name, func(t *testing.T) {
			changed := base
			mutate(&changed)
			if changed.Digest() == base.Digest() {
				t.Fatal("digest missed contract field")
			}
		})
	}
	left, right := base, base
	left.ProjectRef, left.ReviewSubjectDigest = "a", "sha256:"+strings.Repeat("b", 64)
	right.ProjectRef, right.ReviewSubjectDigest = "ab", "sha256:"+strings.Repeat("b", 63)+"a"
	if left.Digest() == right.Digest() {
		t.Fatal("ambiguous concatenation accepted")
	}
}

func TestContributionEnvelopeIsStrictAndVetoIsTyped(t *testing.T) {
	s := subject(t, PolicyAuto)
	valid := fmt.Sprintf(`{"schema":%q,"subject_digest":%q,"role":"proposer","body":"proposal","ballot":"accept","evidence":[{"kind":"proof","ref":"evidence:one"}]}`, ContributionSchema, s.Digest())
	if _, err := DecodeContribution([]byte(valid)); err != nil {
		t.Fatal(err)
	}
	for _, invalid := range []string{
		valid[:len(valid)-1] + `,"extra":true}`, `{"schema":"orquesta.council.contribution.v1","subject_digest":"bad","role":"proposer","body":"x","ballot":"accept","evidence":[]}`,
		fmt.Sprintf(`{"schema":%q,"subject_digest":%q,"role":"arbiter","body":"veto","ballot":"security_veto","evidence":[{"kind":"high","ref":"log"}]}`, ContributionSchema, s.Digest()),
	} {
		if _, err := DecodeContribution([]byte(invalid)); !errors.Is(err, ErrInvalidContribution) {
			t.Fatalf("invalid envelope: %v", err)
		}
	}
	multiline := Contribution{Schema: ContributionSchema, SubjectDigest: s.Digest(), Role: RoleProposer, Body: "proposal\nrationale", Ballot: BallotAccept, Evidence: []Evidence{{Kind: "proof", Ref: "evidence:one"}}}
	if _, err := NewContribution(multiline); err != nil {
		t.Fatalf("multiline body rejected: %v", err)
	}
	multiline.Body = "proposal\x00rationale"
	if _, err := NewContribution(multiline); !errors.Is(err, ErrInvalidContribution) {
		t.Fatalf("NUL body accepted: %v", err)
	}
}

func TestPolicyValidationAndRolesAreSafeForCallers(t *testing.T) {
	for _, policy := range []Policy{PolicyAuto, PolicyRequired, PolicySkipByOperator} {
		if err := ValidatePolicy(policy); err != nil {
			t.Fatalf("policy %q: %v", policy, err)
		}
	}
	if !errors.Is(ValidatePolicy("later"), ErrInvalidPolicy) {
		t.Fatal("unknown policy accepted")
	}
	first, second := Roles(), Roles()
	first[0] = Role("changed")
	if len(second) != 3 || second[0] != RoleProposer {
		t.Fatalf("roles not copied: %v", second)
	}
}

func TestEvaluateWaitsForThreeAndDerivesExactOutcomesAndDissent(t *testing.T) {
	s := subject(t, PolicyAuto)
	cases := []struct {
		name    string
		ballots []Ballot
		outcome Outcome
		dissent int
	}{
		{"accepted", []Ballot{BallotAccept, BallotAccept, BallotReject}, OutcomeAccepted, 1},
		{"rejected", []Ballot{BallotReject, BallotReject, BallotAbstain}, OutcomeRejected, 1},
		{"no consensus", []Ballot{BallotAccept, BallotReject, BallotAbstain}, OutcomeNoConsensus, 3},
		{"veto", []Ballot{BallotSecurityVeto, BallotAccept, BallotReject}, OutcomeBlockedSecurity, 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			facts := []ContributionFact{fact(t, s, RoleProposer, tc.ballots[0]), fact(t, s, RoleCritic, tc.ballots[1]), fact(t, s, RoleArbiter, tc.ballots[2])}
			decision, err := Evaluate(s, facts)
			if err != nil || decision.Outcome != tc.outcome || len(decision.Dissent) != tc.dissent || decision.Digest == "" {
				t.Fatalf("decision=%+v err=%v", decision, err)
			}
			reversed, err := Evaluate(s, []ContributionFact{facts[2], facts[0], facts[1]})
			if err != nil || reversed.Digest != decision.Digest {
				t.Fatalf("replay changed result: %+v %v", reversed, err)
			}
		})
	}
	pending, err := Evaluate(s, []ContributionFact{fact(t, s, RoleProposer, BallotSecurityVeto), fact(t, s, RoleCritic, BallotAccept)})
	if err != nil || pending.Outcome != OutcomePending || pending.Digest != "" {
		t.Fatalf("early veto decided: %+v %v", pending, err)
	}
}

func TestEvaluateRejectsSubstitutionAndReplayDigestBindsPayload(t *testing.T) {
	s := subject(t, PolicyAuto)
	proposer, critic, arbiter := fact(t, s, RoleProposer, BallotAccept), fact(t, s, RoleCritic, BallotAccept), fact(t, s, RoleArbiter, BallotReject)
	if _, err := Evaluate(s, []ContributionFact{proposer, critic, arbiter}); err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func(){
		"receipt":   func() { critic.LaunchReceiptRef = proposer.LaunchReceiptRef },
		"execution": func() { critic.ExecutionRef = proposer.ExecutionRef },
		"subject":   func() { critic.SubjectDigest = "sha256:" + strings.Repeat("c", 64) },
	} {
		t.Run(name, func(t *testing.T) {
			proposer, critic, arbiter = fact(t, s, RoleProposer, BallotAccept), fact(t, s, RoleCritic, BallotAccept), fact(t, s, RoleArbiter, BallotReject)
			mutate()
			if _, err := Evaluate(s, []ContributionFact{proposer, critic, arbiter}); err == nil {
				t.Fatal("substitution accepted")
			}
		})
	}
	first, second := fact(t, s, RoleProposer, BallotAccept), fact(t, s, RoleProposer, BallotAccept)
	second.IdempotencyKey = "key:changed"
	if first.Digest() == second.Digest() {
		t.Fatal("changed replay payload retained digest")
	}
}

func TestSkipBindsPolicySubjectAndSpec(t *testing.T) {
	s := subject(t, PolicySkipByOperator)
	skip := Skip{PrincipalRef: "principal:operator", Reason: "authorized\nwith record", SpecHash: s.SpecHash, IdempotencyKey: "skip:one", CouncilSubjectDigest: s.Digest(), RecordedAtUTC: time.Unix(1, 0).UTC()}
	first, err := NewSkip(s, skip)
	if err != nil || first.Digest() != skip.Digest() {
		t.Fatalf("skip=%+v err=%v", first, err)
	}
	for name, mutate := range map[string]func(*Skip){
		"subject": func(v *Skip) { v.CouncilSubjectDigest = "sha256:" + strings.Repeat("c", 64) },
		"spec":    func(v *Skip) { v.SpecHash = strings.Repeat("c", 64) },
		"key":     func(v *Skip) { v.IdempotencyKey = "" },
		"utc":     func(v *Skip) { v.RecordedAtUTC = v.RecordedAtUTC.In(time.FixedZone("offset", 3600)) },
		"reason":  func(v *Skip) { v.Reason = "authorized\x00record" },
	} {
		t.Run(name, func(t *testing.T) {
			changed := skip
			mutate(&changed)
			if _, err := NewSkip(s, changed); !errors.Is(err, ErrInvalidSkip) {
				t.Fatalf("invalid skip=%v", err)
			}
		})
	}
	if _, err := NewSkip(subject(t, PolicyAuto), skip); !errors.Is(err, ErrInvalidSkip) {
		t.Fatal("auto policy accepted skip")
	}
}
