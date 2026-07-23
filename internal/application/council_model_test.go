package application

import (
	"crypto/sha256"
	"testing"

	"orquesta/internal/council"
)

func TestCouncilResolutionIsExactlyOneProof(t *testing.T) {
	digest := CouncilSubjectDigest("sha256:" + string(makeDigest('a')))
	accepted := CouncilResolution{SubjectDigest: digest, DecisionRef: "council-decision:one", DecisionDigest: digest}
	if err := accepted.Validate(); err != nil {
		t.Fatal(err)
	}
	invalid := accepted
	invalid.SkipRef, invalid.SkipDigest = "council-skip:one", digest
	if err := invalid.Validate(); err == nil {
		t.Fatal("mixed decision and skip accepted")
	}
}

func TestCouncilDigestRequiresCanonicalPrefixAndStrictRefs(t *testing.T) {
	if validCouncilDigest(string(makeDigest('a'))) || !validCouncilDigest("sha256:"+string(makeDigest('a'))) {
		t.Fatal("council digest prefix contract changed")
	}
	if validCouncilRef("round\nspoof") || validCouncilRef("round\x00spoof") || validCouncilRef("") {
		t.Fatal("unsafe council ref accepted")
	}
}

func TestCouncilPurposesUseDedicatedSubjectField(t *testing.T) {
	for role, purpose := range map[council.Role]ExecutionPurpose{
		council.RoleProposer: ExecutionPurposeCouncilProposer,
		council.RoleCritic:   ExecutionPurposeCouncilCritic,
		council.RoleArbiter:  ExecutionPurposeCouncilArbiter,
	} {
		got, ok := councilPurpose(role)
		if !ok || got != purpose || got == ExecutionPurposePrimaryReview || got == ExecutionPurposeAdversarialReview {
			t.Fatalf("purpose %s = %s", role, got)
		}
	}
	execution := ExecutionRecord{Purpose: ExecutionPurposeCouncilProposer, CouncilSubjectDigest: CouncilSubjectDigest("sha256:" + string(makeDigest('b')))}
	if execution.ReviewSubjectDigest != "" || execution.CouncilSubjectDigest == "" {
		t.Fatal("council reused review subject")
	}
}

func TestCouncilAuthorityShapesRemainExplicit(t *testing.T) {
	state := OpenCouncilRoundState{RequestRef: "council-open:one", RequestFingerprint: string(makeDigest('c')), LeaseToken: "lease:one", LeaseFence: 7,
		Round: CouncilRoundRecord{Opener: CouncilRoundOpenerDirector, DirectorFence: 7}}
	if state.RequestRef == "" || len(state.RequestFingerprint) != 64 || state.LeaseToken == "" || state.LeaseFence == 0 ||
		state.Round.Opener != CouncilRoundOpenerDirector || state.Round.DirectorFence != state.LeaseFence {
		t.Fatal("director authority shape incomplete")
	}
	auto := OpenCouncilRoundState{Round: CouncilRoundRecord{Opener: CouncilRoundOpenerAuto}}
	if auto.Round.DirectorFence != 0 || auto.LeaseFence != 0 || auto.LeaseToken != "" {
		t.Fatal("auto opener carried director authority")
	}
	skip := CouncilSkipState{RequestRef: "council-skip:one", RequestFingerprint: string(makeDigest('d'))}
	if skip.RequestRef == "" || len(skip.RequestFingerprint) != 64 {
		t.Fatal("skip replay shape incomplete")
	}
}

func TestCouncilRetryStatesKeepNoWorkItemAuthorityPayload(t *testing.T) {
	replaced := CouncilExecutionReplacedState{Claim: ActionClaim{}, FailedExecution: ExecutionRecord{Purpose: ExecutionPurposeCouncilCritic}, ReplacementExecution: ExecutionRecord{Purpose: ExecutionPurposeCouncilCritic}}
	failed := CouncilExecutionFailedState{Claim: ActionClaim{}, Execution: ExecutionRecord{Purpose: ExecutionPurposeCouncilArbiter}}
	if replaced.FailedExecution.Purpose != replaced.ReplacementExecution.Purpose || failed.Execution.Purpose == ExecutionPurposeAuthor {
		t.Fatal("council retry state rebinding contract changed")
	}
}

func TestPlanFingerprintBindsCouncilPolicy(t *testing.T) {
	base := &PlanSpec{WorkItems: []WorkItemSpec{{Key: "writer", CouncilPolicy: council.PolicyAuto}}}
	changed := &PlanSpec{WorkItems: []WorkItemSpec{{Key: "writer", CouncilPolicy: council.PolicyRequired}}}
	if fingerprintPlan(base) == fingerprintPlan(changed) {
		t.Fatal("council policy absent from plan fingerprint")
	}
}

func fingerprintPlan(plan *PlanSpec) string {
	digest := sha256.New()
	writePlanFingerprint(digest, plan)
	return string(digest.Sum(nil))
}

func makeDigest(value byte) []byte {
	result := make([]byte, 64)
	for index := range result {
		result[index] = value
	}
	return result
}
