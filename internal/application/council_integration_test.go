package application

import "testing"

func TestIntegrationTargetDigestKeepsLegacyV2AndBindsCouncilResolution(t *testing.T) {
	change := ChangeSet{}
	legacy := integrationTargetDigest(change, "target", "gate")
	if legacy != integrationTargetDigest(change, "target", "gate", nil) {
		t.Fatal("legacy target digest changed")
	}
	digest := CouncilSubjectDigest("sha256:" + string(makeDigest('a')))
	decision := &CouncilResolution{SubjectDigest: digest, DecisionRef: "decision:one", DecisionDigest: digest}
	skip := &CouncilResolution{SubjectDigest: digest, SkipRef: "skip:one", SkipDigest: digest}
	if legacy == integrationTargetDigest(change, "target", "gate", decision) ||
		integrationTargetDigest(change, "target", "gate", decision) == integrationTargetDigest(change, "target", "gate", skip) {
		t.Fatal("council resolution absent from target digest")
	}
}

func TestCloneActionRecordDoesNotAliasCouncilResolution(t *testing.T) {
	digest := CouncilSubjectDigest("sha256:" + string(makeDigest('b')))
	action := ActionRecord{CouncilResolution: &CouncilResolution{SubjectDigest: digest, SkipRef: "skip:one", SkipDigest: digest},
		EffectIntent: EffectIntent{CouncilResolution: &CouncilResolution{SubjectDigest: digest, DecisionRef: "decision:one", DecisionDigest: digest}}}
	cloned := cloneActionRecord(action)
	cloned.CouncilResolution.SkipRef = "skip:mutated"
	cloned.EffectIntent.CouncilResolution.DecisionRef = "decision:mutated"
	if action.CouncilResolution.SkipRef != "skip:one" || action.EffectIntent.CouncilResolution.DecisionRef != "decision:one" {
		t.Fatal("persisted action resolution aliased response")
	}
}
