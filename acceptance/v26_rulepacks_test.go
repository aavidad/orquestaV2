package acceptance_test

import (
	"bytes"
	"crypto/ed25519"
	"reflect"
	"strings"
	"testing"

	"orquesta/internal/goal"
	"orquesta/internal/tooling"
)

func TestV26TLS13RulePacksRefreshExactScopePrecedenceAndRevocation(t *testing.T) {
	privateKey := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x73}, ed25519.SeedSize))
	root := tooling.SkillTrustRoot{
		SignerRef: "signer:v26:rulepacks:1",
		PublicKey: append([]byte(nil), privateKey[ed25519.SeedSize:]...),
	}
	revocationKey := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x74}, ed25519.SeedSize))
	revoker := tooling.RulePackRevocationAuthority{
		OriginRef: "catalog:v26:rulepacks", SignerRef: "signer:v26:rulepacks:revoker",
		PublicKey: append([]byte(nil), revocationKey[ed25519.SeedSize:]...),
	}
	base := v26RulePack("policy.review.base", 10, tooling.SkillScope{Kind: tooling.SkillScopeGlobal}, "a")
	project := v26RulePack("policy.review.project", 20,
		tooling.SkillScope{Kind: tooling.SkillScopeProject, ProjectRef: "project:alpha"}, "b")
	origin := tooling.RulePackOrigin{Ref: "catalog:v26:rulepacks", RevisionDigest: v26RulePackDigest("c")}
	baseCandidate := v26SignedRulePack(t, base, origin, root.SignerRef, privateKey)
	projectCandidate := v26SignedRulePack(t, project, origin, root.SignerRef, privateKey)
	current, err := tooling.NewRulePackCatalog(origin, []tooling.SkillTrustRoot{root}, nil,
		[]tooling.RulePackCandidate{projectCandidate, baseCandidate}, nil)
	if err != nil {
		t.Fatal(err)
	}
	projectRef, _ := goal.NewProjectRef("project:alpha")
	context, _ := tooling.NewSkillScopeContext("", projectRef, "", goal.GoalRef{})
	effective, err := current.Resolve(context)
	if err != nil || len(effective) != 1 || effective[0].PackID != project.ID ||
		effective[0].Item.ContractDigest != v26RulePackDigest("b") {
		t.Fatalf("effective=%+v error=%v", effective, err)
	}

	var projectRegistration tooling.RulePackRegistration
	for _, registration := range current.List() {
		if registration.Spec.ID == project.ID {
			projectRegistration = registration
		}
	}
	nextOrigin := tooling.RulePackOrigin{Ref: origin.Ref, RevisionDigest: v26RulePackDigest("d")}
	revocationClaim, err := tooling.NewRulePackRevocationClaim(
		projectRegistration, nextOrigin, "revocation:v26:rulepacks:1", "security.withdrawn", revoker.SignerRef,
	)
	if err != nil {
		t.Fatal(err)
	}
	revocation := tooling.RulePackRevocationCandidate{
		Claim: revocationClaim, Signature: ed25519.Sign(revocationKey, revocationClaim.SignaturePayload()),
	}
	next, err := current.Refresh([]tooling.SkillTrustRoot{root}, []tooling.RulePackRevocationAuthority{revoker}, tooling.RulePackRefresh{
		Coverage:              tooling.RulePackRefreshCompleteSnapshot,
		PreviousCatalogDigest: current.Digest(), PreviousOrigin: origin, NextOrigin: nextOrigin,
		Candidates: []tooling.RulePackCandidate{
			v26SignedRulePack(t, base, nextOrigin, root.SignerRef, privateKey),
			v26SignedRulePack(t, project, nextOrigin, root.SignerRef, privateKey),
		},
		Revocations: []tooling.RulePackRevocationCandidate{revocation},
	})
	if err != nil {
		t.Fatal(err)
	}
	revoked := false
	for _, registration := range next.List() {
		if registration.Spec.ID == project.ID {
			revoked = registration.Revoked
		}
	}
	effective, err = next.Resolve(context)
	if err != nil || len(effective) != 1 || effective[0].PackID != base.ID || !revoked || len(next.History()) != 2 {
		t.Fatalf("fallback=%+v packs=%+v error=%v", effective, next.List(), err)
	}
	restored, err := tooling.RestoreRulePackCatalog(
		next.Snapshot(), next.Digest(), []tooling.SkillTrustRoot{root}, []tooling.RulePackRevocationAuthority{revoker},
	)
	if err != nil {
		t.Fatal(err)
	}
	effective, err = restored.Resolve(context)
	if err != nil || len(effective) != 1 || effective[0].PackID != base.ID {
		t.Fatalf("restored fallback=%+v error=%v", effective, err)
	}
	copy := next.Snapshot()
	copy.Packs[0].Spec.Items[0].ID = "mutated.snapshot"
	copy.History[0].ID = "mutated.history"
	if next.Snapshot().Packs[0].Spec.Items[0].ID == "mutated.snapshot" || next.History()[0].ID == "mutated.history" {
		t.Fatal("snapshot or history aliases catalog state")
	}

	omissionOrigin := tooling.RulePackOrigin{Ref: origin.Ref, RevisionDigest: v26RulePackDigest("f")}
	omitted, err := current.Refresh([]tooling.SkillTrustRoot{root}, nil, tooling.RulePackRefresh{
		Coverage:              tooling.RulePackRefreshCompleteSnapshot,
		PreviousCatalogDigest: current.Digest(), PreviousOrigin: origin, NextOrigin: omissionOrigin,
		Candidates: []tooling.RulePackCandidate{
			v26SignedRulePack(t, base, omissionOrigin, root.SignerRef, privateKey),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	reintroductionOrigin := tooling.RulePackOrigin{Ref: origin.Ref, RevisionDigest: v26RulePackDigest("9")}
	if result, err := omitted.Refresh([]tooling.SkillTrustRoot{root}, nil, tooling.RulePackRefresh{
		Coverage:              tooling.RulePackRefreshCompleteSnapshot,
		PreviousCatalogDigest: omitted.Digest(), PreviousOrigin: omissionOrigin, NextOrigin: reintroductionOrigin,
		Candidates: []tooling.RulePackCandidate{
			v26SignedRulePack(t, base, reintroductionOrigin, root.SignerRef, privateKey),
			v26SignedRulePack(t, project, reintroductionOrigin, root.SignerRef, privateKey),
		},
	}); result != nil || tooling.ErrorCode(err) != tooling.ErrorRulePackInvalid {
		t.Fatalf("reintroduced identity result=%v error=%v", result, err)
	}
	otherKey := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x75}, ed25519.SeedSize))
	otherRoot := tooling.SkillTrustRoot{
		SignerRef: "signer:v26:rulepacks:other",
		PublicKey: append([]byte(nil), otherKey[ed25519.SeedSize:]...),
	}
	projectV2 := project
	projectV2.Version = "2"
	if result, err := current.Refresh([]tooling.SkillTrustRoot{root, otherRoot}, nil, tooling.RulePackRefresh{
		Coverage:              tooling.RulePackRefreshCompleteSnapshot,
		PreviousCatalogDigest: current.Digest(), PreviousOrigin: origin, NextOrigin: reintroductionOrigin,
		Candidates: []tooling.RulePackCandidate{
			v26SignedRulePack(t, base, reintroductionOrigin, root.SignerRef, privateKey),
			v26SignedRulePack(t, projectV2, reintroductionOrigin, otherRoot.SignerRef, otherKey),
		},
	}); result != nil || tooling.ErrorCode(err) != tooling.ErrorRulePackUntrusted {
		t.Fatalf("substituted trust result=%v error=%v", result, err)
	}
	droppedOrigin := tooling.RulePackOrigin{Ref: origin.Ref, RevisionDigest: v26RulePackDigest("e")}
	if result, err := next.Refresh([]tooling.SkillTrustRoot{root}, []tooling.RulePackRevocationAuthority{revoker}, tooling.RulePackRefresh{
		Coverage:              tooling.RulePackRefreshCompleteSnapshot,
		PreviousCatalogDigest: next.Digest(), PreviousOrigin: nextOrigin, NextOrigin: droppedOrigin,
		Candidates: []tooling.RulePackCandidate{
			v26SignedRulePack(t, base, droppedOrigin, root.SignerRef, privateKey),
			v26SignedRulePack(t, project, droppedOrigin, root.SignerRef, privateKey),
		},
	}); result != nil || tooling.ErrorCode(err) != tooling.ErrorRulePackInvalid {
		t.Fatalf("dropped revocation result=%v error=%v", result, err)
	}
	if result, err := next.Refresh([]tooling.SkillTrustRoot{root}, []tooling.RulePackRevocationAuthority{revoker}, tooling.RulePackRefresh{
		Coverage:              tooling.RulePackRefreshCompleteSnapshot,
		PreviousCatalogDigest: next.Digest(), PreviousOrigin: nextOrigin, NextOrigin: origin,
		Candidates: []tooling.RulePackCandidate{
			v26SignedRulePack(t, base, origin, root.SignerRef, privateKey),
			v26SignedRulePack(t, project, origin, root.SignerRef, privateKey),
		},
	}); result != nil || tooling.ErrorCode(err) != tooling.ErrorRulePackInvalid {
		t.Fatalf("A-B-A replay result=%v error=%v", result, err)
	}
	for _, field := range []string{"Body", "Handler", "Loader", "Execute", "Active", "Installed", "Persisted"} {
		if _, exists := reflect.TypeOf(tooling.RulePackSpec{}).FieldByName(field); exists {
			t.Fatalf("rule pack attributes unimplemented runtime field %q", field)
		}
	}
	for _, field := range []string{"Authorized", "Loaded", "Executed", "Installed", "Handler", "Body"} {
		if _, exists := reflect.TypeOf(tooling.EffectiveRulePackItem{}).FieldByName(field); exists {
			t.Fatalf("resolve projection attributes runtime field %q", field)
		}
	}
}

func v26RulePack(id string, precedence int64, scope tooling.SkillScope, digestByte string) tooling.RulePackSpec {
	digest := v26RulePackDigest(digestByte)
	return tooling.RulePackSpec{
		ID: id, Version: "1", DescriptionKey: "rulepack." + id + ".description",
		Scopes: []tooling.SkillScope{scope}, Precedence: precedence,
		Items: []tooling.RulePackItem{{
			Kind: tooling.RulePackItemRule, ID: "review.required",
			ContractRef: "artifact:" + digest, ContractDigest: digest,
		}},
	}
}

func v26SignedRulePack(
	t *testing.T,
	spec tooling.RulePackSpec,
	origin tooling.RulePackOrigin,
	signerRef string,
	privateKey ed25519.PrivateKey,
) tooling.RulePackCandidate {
	t.Helper()
	claim, err := tooling.NewRulePackClaim(spec, origin, signerRef)
	if err != nil {
		t.Fatal(err)
	}
	return tooling.RulePackCandidate{
		Spec: spec, Claim: claim, Signature: ed25519.Sign(privateKey, claim.SignaturePayload()),
	}
}

func v26RulePackDigest(character string) string {
	return "sha256:" + strings.Repeat(character, 64)
}
