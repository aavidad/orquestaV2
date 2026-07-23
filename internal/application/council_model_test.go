package application

import (
	"crypto/sha256"
	"testing"

	"orquesta/internal/council"
)

func TestCouncilResolutionIsExactlyOneProof(t *testing.T) {
	digest := string(makeDigest('a'))
	accepted := CouncilResolution{SubjectDigest: CouncilSubjectDigest(digest), DecisionRef: "council-decision:one", DecisionDigest: digest}
	if err := accepted.Validate(); err != nil {
		t.Fatal(err)
	}
	invalid := accepted
	invalid.SkipRef, invalid.SkipDigest = "council-skip:one", digest
	if err := invalid.Validate(); err == nil {
		t.Fatal("mixed decision and skip accepted")
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
