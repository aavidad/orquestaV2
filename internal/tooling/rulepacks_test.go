package tooling

import (
	"bytes"
	"crypto/ed25519"
	"reflect"
	"strings"
	"testing"

	"orquesta/internal/goal"
)

func TestRulePackCatalogResolvesExactScopedPrecedenceAndDefendsAliases(t *testing.T) {
	privateKey, root := rulePackTrust(0x31)
	base := rulePackSpec("policy.base", "1", 10, SkillScope{Kind: SkillScopeGlobal},
		RulePackItem{Kind: RulePackItemWorkflow, ID: "review.flow", ContractRef: "artifact:" + repeatedDigest("b"), ContractDigest: repeatedDigest("b")},
		RulePackItem{Kind: RulePackItemRule, ID: "review.required", ContractRef: "artifact:" + repeatedDigest("a"), ContractDigest: repeatedDigest("a")},
	)
	override := rulePackSpec("policy.project", "2", 20,
		SkillScope{Kind: SkillScopeProject, ProjectRef: "project:alpha"},
		RulePackItem{Kind: RulePackItemRule, ID: "review.required", ContractRef: "artifact:" + repeatedDigest("c"), ContractDigest: repeatedDigest("c")},
	)
	origin := RulePackOrigin{Ref: "catalog:rules:orquesta", RevisionDigest: repeatedDigest("d")}
	baseCandidate := signedRulePack(t, base, origin, root.SignerRef, privateKey)
	overrideCandidate := signedRulePack(t, override, origin, root.SignerRef, privateKey)
	catalog, err := NewRulePackCatalog(origin, []SkillTrustRoot{root}, nil, []RulePackCandidate{overrideCandidate, baseCandidate}, nil)
	if err != nil {
		t.Fatal(err)
	}
	reorderedBase := base
	reorderedBase.Items = []RulePackItem{base.Items[1], base.Items[0]}
	reordered, err := NewRulePackCatalog(origin, []SkillTrustRoot{root}, nil, []RulePackCandidate{
		signedRulePack(t, reorderedBase, origin, root.SignerRef, privateKey), overrideCandidate,
	}, nil)
	if err != nil || reordered.Digest() != catalog.Digest() {
		t.Fatalf("catalog=%q reordered=%q error=%v", catalog.Digest(), reordered.Digest(), err)
	}
	projectAlpha, _ := goal.NewProjectRef("project:alpha")
	alpha, _ := NewSkillScopeContext("", projectAlpha, "", goal.GoalRef{})
	effective, err := catalog.Resolve(alpha)
	if err != nil || len(effective) != 2 || effective[0].PackID != "policy.project" ||
		effective[0].Item.ID != "review.required" || effective[0].Item.ContractDigest != repeatedDigest("c") ||
		effective[1].PackID != "policy.base" || effective[1].Item.ID != "review.flow" {
		t.Fatalf("effective=%+v error=%v", effective, err)
	}
	projectBeta, _ := goal.NewProjectRef("project:beta")
	beta, _ := NewSkillScopeContext("", projectBeta, "", goal.GoalRef{})
	baseOnly, err := catalog.Resolve(beta)
	if err != nil || len(baseOnly) != 2 || baseOnly[0].PackID != "policy.base" || baseOnly[1].PackID != "policy.base" {
		t.Fatalf("base=%+v error=%v", baseOnly, err)
	}

	base.Items[0].ID = "mutated.input"
	baseCandidate.Spec.Items[0].ID = "mutated.candidate"
	listed := catalog.List()
	listed[0].Spec.Scopes[0].Kind = SkillScopeProduct
	listed[0].Spec.Items[0].ID = "mutated.output"
	history := catalog.History()
	history[0].ID = "mutated.history"
	again, _ := catalog.Resolve(alpha)
	if again[0].Item.ID != "review.required" || again[1].Item.ID != "review.flow" ||
		catalog.Origin() != origin || catalog.History()[0].ID == "mutated.history" {
		t.Fatalf("caller mutation changed immutable catalog: %+v", again)
	}
	for _, field := range []string{"Body", "Handler", "Loader", "Executor", "Active", "Installed", "Lifecycle"} {
		if _, exists := reflect.TypeOf(RulePackSpec{}).FieldByName(field); exists {
			t.Fatalf("rule pack exposes runtime field %q", field)
		}
	}
	if !strings.HasPrefix(catalog.Digest(), "sha256:") || (*RulePackCatalog)(nil).Digest() != "" ||
		(*RulePackCatalog)(nil).List() != nil || (*RulePackCatalog)(nil).History() != nil {
		t.Fatal("catalog digest or nil behavior is not stable")
	}
}

func TestRulePackCatalogRejectsImplicitAmbiguousOrUntrustedDescriptors(t *testing.T) {
	privateKey, root := rulePackTrust(0x41)
	valid := rulePackSpec("policy.base", "1", 10, SkillScope{Kind: SkillScopeGlobal},
		RulePackItem{Kind: RulePackItemRule, ID: "review.required", ContractRef: "artifact:" + repeatedDigest("a"), ContractDigest: repeatedDigest("a")},
	)
	origin := RulePackOrigin{Ref: "catalog:rules:orquesta", RevisionDigest: repeatedDigest("d")}
	tests := map[string]func(*RulePackSpec){
		"id":          func(value *RulePackSpec) { value.ID = "policy.*" },
		"version":     func(value *RulePackSpec) { value.Version = "01" },
		"description": func(value *RulePackSpec) { value.DescriptionKey = "rulepack.other.description" },
		"scope":       func(value *RulePackSpec) { value.Scopes = nil },
		"precedence":  func(value *RulePackSpec) { value.Precedence = 0 },
		"items":       func(value *RulePackSpec) { value.Items = nil },
		"kind":        func(value *RulePackSpec) { value.Items[0].Kind = "prompt" },
		"item id":     func(value *RulePackSpec) { value.Items[0].ID = "review.*" },
		"ref":         func(value *RulePackSpec) { value.Items[0].ContractRef = "latest" },
		"digest":      func(value *RulePackSpec) { value.Items[0].ContractDigest = "sha256:short" },
		"duplicate":   func(value *RulePackSpec) { value.Items = append(value.Items, value.Items[0]) },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			spec := cloneRulePackSpec(valid)
			mutate(&spec)
			if claim, err := NewRulePackClaim(spec, origin, root.SignerRef); claim != (RulePackClaim{}) || ErrorCode(err) != ErrorRulePackInvalid {
				t.Fatalf("claim=%+v error=%v code=%q", claim, err, ErrorCode(err))
			}
		})
	}
	candidate := signedRulePack(t, valid, origin, root.SignerRef, privateKey)
	forged := candidate
	forged.Signature = append([]byte(nil), candidate.Signature...)
	forged.Signature[0] ^= 0xff
	if catalog, err := NewRulePackCatalog(origin, []SkillTrustRoot{root}, nil, []RulePackCandidate{forged}, nil); catalog != nil || ErrorCode(err) != ErrorRulePackUntrusted {
		t.Fatalf("forged catalog=%v error=%v", catalog, err)
	}
	if catalog, err := NewRulePackCatalog(origin, []SkillTrustRoot{root}, nil, []RulePackCandidate{candidate, candidate}, nil); catalog != nil || ErrorCode(err) != ErrorRulePackDuplicate {
		t.Fatalf("duplicate catalog=%v error=%v", catalog, err)
	}
	competing := rulePackSpec("policy.competing", "1", 10, SkillScope{Kind: SkillScopeGlobal}, valid.Items[0])
	if catalog, err := NewRulePackCatalog(origin, []SkillTrustRoot{root}, nil, []RulePackCandidate{
		candidate, signedRulePack(t, competing, origin, root.SignerRef, privateKey),
	}, nil); catalog != nil || ErrorCode(err) != ErrorRulePackDuplicate {
		t.Fatalf("ambiguous catalog=%v error=%v", catalog, err)
	}
	badOrigin := origin
	badOrigin.Ref = "catalog:rules:*"
	if catalog, err := NewRulePackCatalog(badOrigin, []SkillTrustRoot{root}, nil, []RulePackCandidate{candidate}, nil); catalog != nil || ErrorCode(err) != ErrorRulePackInvalid {
		t.Fatalf("wildcard origin catalog=%v error=%v", catalog, err)
	}
	otherOrigin := RulePackOrigin{Ref: origin.Ref, RevisionDigest: repeatedDigest("e")}
	if catalog, err := NewRulePackCatalog(otherOrigin, []SkillTrustRoot{root}, nil, []RulePackCandidate{candidate}, nil); catalog != nil || ErrorCode(err) != ErrorRulePackUntrusted {
		t.Fatalf("transplanted origin catalog=%v error=%v", catalog, err)
	}
	catalog, err := NewRulePackCatalog(origin, []SkillTrustRoot{root}, nil, []RulePackCandidate{candidate}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result, err := catalog.Resolve(SkillScopeContext{}); result != nil || ErrorCode(err) != ErrorSkillScopeInvalid {
		t.Fatalf("invalid context result=%v error=%v", result, err)
	}
}

func TestRulePackRefreshHistoryRejectsAnyReintroductionAfterOmission(t *testing.T) {
	privateKey, root := rulePackTrust(0x61)
	origin := RulePackOrigin{Ref: "catalog:rules:orquesta", RevisionDigest: repeatedDigest("1")}
	removed := rulePackSpec("policy.removed", "1", 10, SkillScope{Kind: SkillScopeGlobal},
		RulePackItem{Kind: RulePackItemRule, ID: "review.required", ContractRef: "artifact:" + repeatedDigest("a"), ContractDigest: repeatedDigest("a")},
	)
	survivor := rulePackSpec("policy.survivor", "1", 10, SkillScope{Kind: SkillScopeGlobal},
		RulePackItem{Kind: RulePackItemRule, ID: "audit.required", ContractRef: "artifact:" + repeatedDigest("b"), ContractDigest: repeatedDigest("b")},
	)
	initial, err := NewRulePackCatalog(origin, []SkillTrustRoot{root}, nil, []RulePackCandidate{
		signedRulePack(t, removed, origin, root.SignerRef, privateKey),
		signedRulePack(t, survivor, origin, root.SignerRef, privateKey),
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	omissionOrigin := RulePackOrigin{Ref: origin.Ref, RevisionDigest: repeatedDigest("2")}
	if result, err := initial.Refresh([]SkillTrustRoot{root}, nil, RulePackRefresh{
		PreviousCatalogDigest: initial.Digest(), PreviousOrigin: origin, NextOrigin: omissionOrigin,
		Candidates: []RulePackCandidate{signedRulePack(t, survivor, omissionOrigin, root.SignerRef, privateKey)},
	}); result != nil || ErrorCode(err) != ErrorRulePackInvalid {
		t.Fatalf("silent omission result=%v error=%v", result, err)
	}
	omitted, err := initial.Refresh([]SkillTrustRoot{root}, nil, RulePackRefresh{
		Coverage:              RulePackRefreshCompleteSnapshot,
		PreviousCatalogDigest: initial.Digest(), PreviousOrigin: origin, NextOrigin: omissionOrigin,
		Candidates: []RulePackCandidate{signedRulePack(t, survivor, omissionOrigin, root.SignerRef, privateKey)},
	})
	if err != nil || len(omitted.List()) != 1 || len(omitted.History()) != 2 {
		t.Fatalf("omitted catalog=%v entries=%d history=%+v error=%v", omitted, len(omitted.List()), omitted.History(), err)
	}

	reintroductionOrigin := RulePackOrigin{Ref: origin.Ref, RevisionDigest: repeatedDigest("3")}
	rewritten := cloneRulePackSpec(removed)
	rewritten.Items[0].ContractDigest = repeatedDigest("c")
	rewritten.Items[0].ContractRef = "artifact:" + repeatedDigest("c")
	if result, err := omitted.Refresh([]SkillTrustRoot{root}, nil, RulePackRefresh{
		Coverage:              RulePackRefreshCompleteSnapshot,
		PreviousCatalogDigest: omitted.Digest(), PreviousOrigin: omissionOrigin, NextOrigin: reintroductionOrigin,
		Candidates: []RulePackCandidate{
			signedRulePack(t, rewritten, reintroductionOrigin, root.SignerRef, privateKey),
			signedRulePack(t, survivor, reintroductionOrigin, root.SignerRef, privateKey),
		},
	}); result != nil || ErrorCode(err) != ErrorRulePackInvalid {
		t.Fatalf("rewritten tombstone result=%v error=%v", result, err)
	}

	if result, err := omitted.Refresh([]SkillTrustRoot{root}, nil, RulePackRefresh{
		Coverage:              RulePackRefreshCompleteSnapshot,
		PreviousCatalogDigest: omitted.Digest(), PreviousOrigin: omissionOrigin, NextOrigin: reintroductionOrigin,
		Candidates: []RulePackCandidate{
			signedRulePack(t, removed, reintroductionOrigin, root.SignerRef, privateKey),
			signedRulePack(t, survivor, reintroductionOrigin, root.SignerRef, privateKey),
		},
	}); result != nil || ErrorCode(err) != ErrorRulePackInvalid {
		t.Fatalf("exact reintroduction result=%v error=%v", result, err)
	}
}

func TestRulePackRefreshRejectsTrustSubstitutionAcrossVersions(t *testing.T) {
	privateKey, root := rulePackTrust(0x62)
	origin := RulePackOrigin{Ref: "catalog:rules:trust", RevisionDigest: repeatedDigest("1")}
	versionOne := rulePackSpec("policy.versioned", "1", 10, SkillScope{Kind: SkillScopeGlobal},
		RulePackItem{Kind: RulePackItemRule, ID: "review.required", ContractRef: "artifact:" + repeatedDigest("a"), ContractDigest: repeatedDigest("a")},
	)
	current, err := NewRulePackCatalog(origin, []SkillTrustRoot{root}, nil, []RulePackCandidate{
		signedRulePack(t, versionOne, origin, root.SignerRef, privateKey),
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	rotatedPrivateKey := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x63}, ed25519.SeedSize))
	rotatedRoot := SkillTrustRoot{
		SignerRef: root.SignerRef, PublicKey: append([]byte(nil), rotatedPrivateKey[ed25519.SeedSize:]...),
	}
	versionTwo := rulePackSpec("policy.versioned", "2", 10, SkillScope{Kind: SkillScopeGlobal}, versionOne.Items[0])
	nextOrigin := RulePackOrigin{Ref: origin.Ref, RevisionDigest: repeatedDigest("2")}
	if result, err := current.Refresh([]SkillTrustRoot{rotatedRoot}, nil, RulePackRefresh{
		Coverage: RulePackRefreshCompleteSnapshot, PreviousCatalogDigest: current.Digest(),
		PreviousOrigin: origin, NextOrigin: nextOrigin,
		Candidates: []RulePackCandidate{signedRulePack(t, versionTwo, nextOrigin, rotatedRoot.SignerRef, rotatedPrivateKey)},
	}); result != nil || ErrorCode(err) != ErrorRulePackUntrusted {
		t.Fatalf("rotated trust result=%v error=%v", result, err)
	}
	otherPrivateKey := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x64}, ed25519.SeedSize))
	otherRoot := SkillTrustRoot{
		SignerRef: "signer:rulepacks:other", PublicKey: append([]byte(nil), otherPrivateKey[ed25519.SeedSize:]...),
	}
	if result, err := NewRulePackCatalog(origin, []SkillTrustRoot{root, otherRoot}, nil, []RulePackCandidate{
		signedRulePack(t, versionOne, origin, root.SignerRef, privateKey),
		signedRulePack(t, versionTwo, origin, otherRoot.SignerRef, otherPrivateKey),
	}, nil); result != nil || ErrorCode(err) != ErrorRulePackUntrusted {
		t.Fatalf("mixed catalog trust result=%v error=%v", result, err)
	}
	if result, err := current.Refresh([]SkillTrustRoot{otherRoot}, nil, RulePackRefresh{
		Coverage: RulePackRefreshCompleteSnapshot, PreviousCatalogDigest: current.Digest(),
		PreviousOrigin: origin, NextOrigin: nextOrigin,
		Candidates: []RulePackCandidate{signedRulePack(t, versionTwo, nextOrigin, otherRoot.SignerRef, otherPrivateKey)},
	}); result != nil || ErrorCode(err) != ErrorRulePackUntrusted {
		t.Fatalf("substituted signer result=%v error=%v", result, err)
	}
}

func TestRulePackRefreshUsesLocalPreconditionAndKeepsRevocationFailClosed(t *testing.T) {
	privateKey, root := rulePackTrust(0x51)
	base := rulePackSpec("policy.base", "1", 10, SkillScope{Kind: SkillScopeGlobal},
		RulePackItem{Kind: RulePackItemRule, ID: "review.required", ContractRef: "artifact:" + repeatedDigest("a"), ContractDigest: repeatedDigest("a")},
	)
	override := rulePackSpec("policy.override", "1", 20, SkillScope{Kind: SkillScopeGlobal},
		RulePackItem{Kind: RulePackItemRule, ID: "review.required", ContractRef: "artifact:" + repeatedDigest("b"), ContractDigest: repeatedDigest("b")},
	)
	origin := RulePackOrigin{Ref: "catalog:rules:orquesta", RevisionDigest: repeatedDigest("c")}
	baseCandidate := signedRulePack(t, base, origin, root.SignerRef, privateKey)
	overrideCandidate := signedRulePack(t, override, origin, root.SignerRef, privateKey)
	unrevoked, err := NewRulePackCatalog(origin, []SkillTrustRoot{root}, nil, []RulePackCandidate{baseCandidate, overrideCandidate}, nil)
	if err != nil {
		t.Fatal(err)
	}
	overrideRegistration := findRulePack(t, unrevoked, "policy.override", "1")
	otherPrivateKey := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x52}, ed25519.SeedSize))
	otherRoot := SkillTrustRoot{
		SignerRef: "signer:rulepacks:other", PublicKey: append([]byte(nil), otherPrivateKey[ed25519.SeedSize:]...),
	}
	revocationClaim, err := NewRulePackRevocationClaim(
		overrideRegistration, origin, "revocation:rules:unauthorized", "security.withdrawn", otherRoot.SignerRef,
	)
	if err != nil {
		t.Fatal(err)
	}
	revocation := RulePackRevocationCandidate{
		Claim: revocationClaim, Signature: ed25519.Sign(otherPrivateKey, revocationClaim.SignaturePayload()),
	}
	if result, err := NewRulePackCatalog(origin, []SkillTrustRoot{root, otherRoot}, nil,
		[]RulePackCandidate{baseCandidate, overrideCandidate}, []RulePackRevocationCandidate{revocation}); result != nil || ErrorCode(err) != ErrorRulePackUntrusted {
		t.Fatalf("general root revoked without authority result=%v error=%v", result, err)
	}
	revocationAuthorities := []RulePackRevocationAuthority{{
		OriginRef: origin.Ref, SignerRef: otherRoot.SignerRef, PublicKey: otherRoot.PublicKey,
	}}
	wrongOriginAuthority := []RulePackRevocationAuthority{{
		OriginRef: "catalog:rules:other", SignerRef: otherRoot.SignerRef, PublicKey: otherRoot.PublicKey,
	}}
	if result, err := NewRulePackCatalog(origin, []SkillTrustRoot{root}, wrongOriginAuthority,
		[]RulePackCandidate{baseCandidate, overrideCandidate}, []RulePackRevocationCandidate{revocation}); result != nil || ErrorCode(err) != ErrorRulePackUntrusted {
		t.Fatalf("cross-origin revoker result=%v error=%v", result, err)
	}
	current, err := NewRulePackCatalog(origin, []SkillTrustRoot{root}, revocationAuthorities,
		[]RulePackCandidate{baseCandidate, overrideCandidate}, []RulePackRevocationCandidate{revocation})
	if err != nil {
		t.Fatal(err)
	}
	context, _ := NewSkillScopeContext("", goal.ProjectRef{}, "", goal.GoalRef{})
	effective, err := current.Resolve(context)
	if err != nil || len(effective) != 1 || effective[0].PackID != "policy.base" {
		t.Fatalf("revoked pack remained effective: %+v error=%v", effective, err)
	}
	nextOrigin := RulePackOrigin{Ref: origin.Ref, RevisionDigest: repeatedDigest("d")}
	nextBaseCandidate := signedRulePack(t, base, nextOrigin, root.SignerRef, privateKey)
	nextOverrideCandidate := signedRulePack(t, override, nextOrigin, root.SignerRef, privateKey)
	nextRevocation := signedRulePackRevocation(
		t, overrideRegistration, nextOrigin, revocationClaim.RevocationRef, revocationClaim.ReasonCode,
		otherRoot.SignerRef, otherPrivateKey,
	)
	request := RulePackRefresh{
		Coverage:              RulePackRefreshCompleteSnapshot,
		PreviousCatalogDigest: current.Digest(), PreviousOrigin: origin, NextOrigin: nextOrigin,
		Candidates:  []RulePackCandidate{nextOverrideCandidate, nextBaseCandidate},
		Revocations: []RulePackRevocationCandidate{nextRevocation},
	}
	next, err := current.Refresh([]SkillTrustRoot{root}, revocationAuthorities, request)
	if err != nil || next.Origin() != nextOrigin || next.Digest() == current.Digest() {
		t.Fatalf("next=%v error=%v", next, err)
	}
	currentHistory, nextHistory := current.History(), next.History()
	if len(currentHistory) != 2 || len(nextHistory) != 2 ||
		nextHistory[1].RevocationOrigin != currentHistory[1].RevocationOrigin ||
		nextHistory[1].RevocationSubjectDigest != currentHistory[1].RevocationSubjectDigest ||
		nextHistory[1].RevocationSignature != currentHistory[1].RevocationSignature {
		t.Fatalf("revocation provenance changed across refresh: current=%+v next=%+v", currentHistory, nextHistory)
	}

	stale := request
	stale.PreviousCatalogDigest = repeatedDigest("0")
	if result, err := current.Refresh([]SkillTrustRoot{root}, nil, stale); result != nil || ErrorCode(err) != ErrorRulePackRefreshPrecondition {
		t.Fatalf("stale result=%v error=%v", result, err)
	}
	wrongPreviousOrigin := request
	wrongPreviousOrigin.PreviousOrigin.RevisionDigest = repeatedDigest("0")
	if result, err := current.Refresh([]SkillTrustRoot{root}, nil, wrongPreviousOrigin); result != nil || ErrorCode(err) != ErrorRulePackRefreshPrecondition {
		t.Fatalf("wrong previous origin result=%v error=%v", result, err)
	}
	dropped := request
	dropped.Revocations = nil
	if result, err := current.Refresh([]SkillTrustRoot{root}, nil, dropped); result != nil || ErrorCode(err) != ErrorRulePackInvalid {
		t.Fatalf("dropped revocation result=%v error=%v", result, err)
	}
	rewrittenRevocation := request
	rewrittenRevocation.Revocations = []RulePackRevocationCandidate{signedRulePackRevocation(
		t, overrideRegistration, nextOrigin, revocationClaim.RevocationRef, "security.superseded",
		otherRoot.SignerRef, otherPrivateKey,
	)}
	if result, err := current.Refresh([]SkillTrustRoot{root}, revocationAuthorities, rewrittenRevocation); result != nil || ErrorCode(err) != ErrorRulePackInvalid {
		t.Fatalf("rewritten revocation result=%v error=%v", result, err)
	}
	rewrittenSpec := cloneRulePackSpec(base)
	rewrittenSpec.Items[0].ContractDigest = repeatedDigest("e")
	rewrittenSpec.Items[0].ContractRef = "artifact:" + repeatedDigest("e")
	rewritten := request
	rewritten.Candidates = []RulePackCandidate{
		signedRulePack(t, rewrittenSpec, nextOrigin, root.SignerRef, privateKey), nextOverrideCandidate,
	}
	if result, err := current.Refresh([]SkillTrustRoot{root}, revocationAuthorities, rewritten); result != nil || ErrorCode(err) != ErrorRulePackInvalid {
		t.Fatalf("rewritten version result=%v error=%v", result, err)
	}
	wrongSource := request
	wrongSource.NextOrigin.Ref = "catalog:rules:other"
	if result, err := current.Refresh([]SkillTrustRoot{root}, nil, wrongSource); result != nil || ErrorCode(err) != ErrorRulePackInvalid {
		t.Fatalf("switched origin result=%v error=%v", result, err)
	}
	forgedRevocation := revocation
	forgedRevocation.Signature = append([]byte(nil), revocation.Signature...)
	forgedRevocation.Signature[0] ^= 0xff
	if result, err := NewRulePackCatalog(origin, []SkillTrustRoot{root}, revocationAuthorities,
		[]RulePackCandidate{baseCandidate, overrideCandidate}, []RulePackRevocationCandidate{forgedRevocation}); result != nil || ErrorCode(err) != ErrorRulePackUntrusted {
		t.Fatalf("forged revocation result=%v error=%v", result, err)
	}

	branchOrigin := RulePackOrigin{Ref: origin.Ref, RevisionDigest: repeatedDigest("f")}
	branch, err := current.Refresh([]SkillTrustRoot{root}, revocationAuthorities, RulePackRefresh{
		Coverage:              RulePackRefreshCompleteSnapshot,
		PreviousCatalogDigest: current.Digest(), PreviousOrigin: origin, NextOrigin: branchOrigin,
		Candidates: []RulePackCandidate{
			signedRulePack(t, base, branchOrigin, root.SignerRef, privateKey),
			signedRulePack(t, override, branchOrigin, root.SignerRef, privateKey),
		},
		Revocations: []RulePackRevocationCandidate{signedRulePackRevocation(
			t, overrideRegistration, branchOrigin, revocationClaim.RevocationRef, revocationClaim.ReasonCode,
			otherRoot.SignerRef, otherPrivateKey,
		)},
	})
	if err != nil || branch == nil || branch.Digest() == next.Digest() {
		t.Fatalf("refresh precondition unexpectedly serialized branches: branch=%v error=%v", branch, err)
	}
}

func TestRulePackSnapshotRestoreRejectsRevocationResurrectionAndDefendsCopies(t *testing.T) {
	privateKey, root := rulePackTrust(0x65)
	revocationKey := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x66}, ed25519.SeedSize))
	revoker := RulePackRevocationAuthority{
		OriginRef: "catalog:rules:restore", SignerRef: "signer:rulepacks:revoker",
		PublicKey: append([]byte(nil), revocationKey[ed25519.SeedSize:]...),
	}
	origin := RulePackOrigin{Ref: revoker.OriginRef, RevisionDigest: repeatedDigest("6")}
	spec := rulePackSpec("policy.restore", "1", 10, SkillScope{Kind: SkillScopeGlobal},
		RulePackItem{Kind: RulePackItemRule, ID: "restore.required", ContractRef: "artifact:" + repeatedDigest("a"), ContractDigest: repeatedDigest("a")},
	)
	candidate := signedRulePack(t, spec, origin, root.SignerRef, privateKey)
	unrevoked, err := NewRulePackCatalog(origin, []SkillTrustRoot{root}, nil, []RulePackCandidate{candidate}, nil)
	if err != nil {
		t.Fatal(err)
	}
	revocation := signedRulePackRevocation(
		t, unrevoked.List()[0], origin, "revocation:rules:restore", "security.withdrawn",
		revoker.SignerRef, revocationKey,
	)
	current, err := NewRulePackCatalog(origin, []SkillTrustRoot{root}, []RulePackRevocationAuthority{revoker},
		[]RulePackCandidate{candidate}, []RulePackRevocationCandidate{revocation})
	if err != nil {
		t.Fatal(err)
	}
	snapshot := current.Snapshot()
	restored, err := RestoreRulePackCatalog(
		snapshot, current.Digest(), []SkillTrustRoot{root}, []RulePackRevocationAuthority{revoker},
	)
	context, _ := NewSkillScopeContext("", goal.ProjectRef{}, "", goal.GoalRef{})
	if err != nil || restored.Digest() != current.Digest() || len(restored.History()) != 1 {
		t.Fatalf("restored=%v history=%+v error=%v", restored, restored.History(), err)
	}
	if effective, err := restored.Resolve(context); err != nil || len(effective) != 0 {
		t.Fatalf("revoked snapshot became effective: %+v error=%v", effective, err)
	}
	if result, err := RestoreRulePackCatalog(
		current.Snapshot(), repeatedDigest("0"), []SkillTrustRoot{root}, []RulePackRevocationAuthority{revoker},
	); result != nil || ErrorCode(err) != ErrorRulePackInvalid {
		t.Fatalf("wrong expected digest result=%v error=%v", result, err)
	}

	snapshot.Packs[0].Spec.Items[0].ID = "mutated.snapshot"
	snapshot.History[0].Revoked = false
	if current.Snapshot().Packs[0].Spec.Items[0].ID != "restore.required" || !current.History()[0].Revoked {
		t.Fatal("snapshot output aliases catalog state")
	}

	revived := current.Snapshot()
	revived.Packs[0].Revoked = false
	revived.Packs[0].RevocationOrigin = RulePackOrigin{}
	revived.Packs[0].RevocationRef = ""
	revived.Packs[0].RevocationReasonCode = ""
	revived.Packs[0].RevocationSubjectDigest = ""
	revived.Packs[0].RevocationSignerRef = ""
	revived.Packs[0].RevocationTrustKeyDigest = ""
	revived.Packs[0].RevocationSignature = ""
	revived.CatalogDigest = rulePackCatalogDigest(
		revived.Origin, revived.ObservedOrigins, revived.Packs, revived.History,
	)
	if result, err := RestoreRulePackCatalog(
		revived, revived.CatalogDigest, []SkillTrustRoot{root}, []RulePackRevocationAuthority{revoker},
	); result != nil || ErrorCode(err) != ErrorRulePackInvalid {
		t.Fatalf("revived snapshot result=%v error=%v", result, err)
	}

	rotatedPrivateKey := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x67}, ed25519.SeedSize))
	rotatedRoot := SkillTrustRoot{
		SignerRef: root.SignerRef, PublicKey: append([]byte(nil), rotatedPrivateKey[ed25519.SeedSize:]...),
	}
	if result, err := RestoreRulePackCatalog(
		current.Snapshot(), current.Digest(), []SkillTrustRoot{rotatedRoot}, []RulePackRevocationAuthority{revoker},
	); result != nil || ErrorCode(err) != ErrorRulePackUntrusted {
		t.Fatalf("substituted restore root result=%v error=%v", result, err)
	}
}

func TestRulePackRefreshRejectsObservedOriginReplayAndHistoricalDowngrade(t *testing.T) {
	privateKey, root := rulePackTrust(0x71)
	originA := RulePackOrigin{Ref: "catalog:rules:replay", RevisionDigest: repeatedDigest("a")}
	originB := RulePackOrigin{Ref: originA.Ref, RevisionDigest: repeatedDigest("b")}
	originC := RulePackOrigin{Ref: originA.Ref, RevisionDigest: repeatedDigest("c")}
	versionTwo := rulePackSpec("policy.versioned", "2", 20, SkillScope{Kind: SkillScopeGlobal},
		RulePackItem{Kind: RulePackItemRule, ID: "versioned.rule", ContractRef: "artifact:" + repeatedDigest("2"), ContractDigest: repeatedDigest("2")},
	)
	anchor := rulePackSpec("policy.anchor", "1", 10, SkillScope{Kind: SkillScopeGlobal},
		RulePackItem{Kind: RulePackItemRule, ID: "anchor.rule", ContractRef: "artifact:" + repeatedDigest("1"), ContractDigest: repeatedDigest("1")},
	)
	initial, err := NewRulePackCatalog(originA, []SkillTrustRoot{root}, nil, []RulePackCandidate{
		signedRulePack(t, versionTwo, originA, root.SignerRef, privateKey),
		signedRulePack(t, anchor, originA, root.SignerRef, privateKey),
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	omitted, err := initial.Refresh([]SkillTrustRoot{root}, nil, RulePackRefresh{
		Coverage:              RulePackRefreshCompleteSnapshot,
		PreviousCatalogDigest: initial.Digest(), PreviousOrigin: originA, NextOrigin: originB,
		Candidates: []RulePackCandidate{signedRulePack(t, anchor, originB, root.SignerRef, privateKey)},
	})
	if err != nil {
		t.Fatal(err)
	}
	if replay, err := omitted.Refresh([]SkillTrustRoot{root}, nil, RulePackRefresh{
		Coverage:              RulePackRefreshCompleteSnapshot,
		PreviousCatalogDigest: omitted.Digest(), PreviousOrigin: originB, NextOrigin: originA,
		Candidates: []RulePackCandidate{signedRulePack(t, anchor, originA, root.SignerRef, privateKey)},
	}); replay != nil || ErrorCode(err) != ErrorRulePackInvalid {
		t.Fatalf("A-B-A replay=%v error=%v", replay, err)
	}

	lower := rulePackSpec("policy.versioned", "1", 20, SkillScope{Kind: SkillScopeGlobal},
		RulePackItem{Kind: RulePackItemRule, ID: "versioned.rule", ContractRef: "artifact:" + repeatedDigest("1"), ContractDigest: repeatedDigest("1")},
	)
	if downgraded, err := omitted.Refresh([]SkillTrustRoot{root}, nil, RulePackRefresh{
		Coverage:              RulePackRefreshCompleteSnapshot,
		PreviousCatalogDigest: omitted.Digest(), PreviousOrigin: originB, NextOrigin: originC,
		Candidates: []RulePackCandidate{
			signedRulePack(t, lower, originC, root.SignerRef, privateKey),
			signedRulePack(t, anchor, originC, root.SignerRef, privateKey),
		},
	}); downgraded != nil || ErrorCode(err) != ErrorRulePackInvalid {
		t.Fatalf("historical downgrade=%v error=%v", downgraded, err)
	}

	advanced, err := omitted.Refresh([]SkillTrustRoot{root}, nil, RulePackRefresh{
		Coverage:              RulePackRefreshCompleteSnapshot,
		PreviousCatalogDigest: omitted.Digest(), PreviousOrigin: originB, NextOrigin: originC,
		Candidates: []RulePackCandidate{signedRulePack(t, anchor, originC, root.SignerRef, privateKey)},
	})
	if err != nil {
		t.Fatal(err)
	}
	if replay, err := advanced.Refresh([]SkillTrustRoot{root}, nil, RulePackRefresh{
		Coverage:              RulePackRefreshCompleteSnapshot,
		PreviousCatalogDigest: advanced.Digest(), PreviousOrigin: originC, NextOrigin: originB,
		Candidates: []RulePackCandidate{signedRulePack(t, anchor, originB, root.SignerRef, privateKey)},
	}); replay != nil || ErrorCode(err) != ErrorRulePackInvalid {
		t.Fatalf("A-B-C-B replay=%v error=%v", replay, err)
	}
}

func TestRulePacksRejectInvalidUTF8BeforeSignaturesAndDigests(t *testing.T) {
	privateKey, root := rulePackTrust(0x72)
	origin := RulePackOrigin{Ref: "catalog:rules:utf8", RevisionDigest: repeatedDigest("a")}
	valid := rulePackSpec("policy.utf8", "1", 10, SkillScope{Kind: SkillScopeGlobal},
		RulePackItem{Kind: RulePackItemRule, ID: "utf8.rule", ContractRef: "artifact:" + repeatedDigest("b"), ContractDigest: repeatedDigest("b")},
	)
	invalid := string([]byte{0xff})
	tests := map[string]func(*RulePackSpec){
		"id":          func(value *RulePackSpec) { value.ID = "policy." + invalid },
		"version":     func(value *RulePackSpec) { value.Version = "1" + invalid },
		"description": func(value *RulePackSpec) { value.DescriptionKey = "rulepack.policy.utf8." + invalid },
		"scope": func(value *RulePackSpec) {
			value.Scopes = []SkillScope{{Kind: SkillScopeProduct, ProductID: invalid}}
		},
		"item kind":   func(value *RulePackSpec) { value.Items[0].Kind = RulePackItemKind(invalid) },
		"item id":     func(value *RulePackSpec) { value.Items[0].ID = "utf8." + invalid },
		"item ref":    func(value *RulePackSpec) { value.Items[0].ContractRef = "artifact:" + invalid },
		"item digest": func(value *RulePackSpec) { value.Items[0].ContractDigest = repeatedDigest("b") + invalid },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			spec := cloneRulePackSpec(valid)
			mutate(&spec)
			if claim, err := NewRulePackClaim(spec, origin, root.SignerRef); claim != (RulePackClaim{}) || ErrorCode(err) != ErrorRulePackInvalid {
				t.Fatalf("claim=%+v error=%v", claim, err)
			}
		})
	}
	if claim, err := NewRulePackClaim(valid, RulePackOrigin{Ref: origin.Ref + invalid, RevisionDigest: origin.RevisionDigest}, root.SignerRef); claim != (RulePackClaim{}) || ErrorCode(err) != ErrorRulePackInvalid {
		t.Fatalf("origin claim=%+v error=%v", claim, err)
	}
	if claim, err := NewRulePackClaim(valid, origin, root.SignerRef+invalid); claim != (RulePackClaim{}) || ErrorCode(err) != ErrorRulePackInvalid {
		t.Fatalf("signer claim=%+v error=%v", claim, err)
	}
	candidate := signedRulePack(t, valid, origin, root.SignerRef, privateKey)
	if catalog, err := NewRulePackCatalog(origin, []SkillTrustRoot{{SignerRef: root.SignerRef + invalid, PublicKey: root.PublicKey}}, nil,
		[]RulePackCandidate{candidate}, nil); catalog != nil || ErrorCode(err) != ErrorRulePackInvalid {
		t.Fatalf("invalid root catalog=%v error=%v", catalog, err)
	}
	if catalog, err := NewRulePackCatalog(origin, []SkillTrustRoot{root}, []RulePackRevocationAuthority{{
		OriginRef: origin.Ref + invalid, SignerRef: "signer:revoker:utf8", PublicKey: root.PublicKey,
	}}, []RulePackCandidate{candidate}, nil); catalog != nil || ErrorCode(err) != ErrorRulePackInvalid {
		t.Fatalf("invalid authority catalog=%v error=%v", catalog, err)
	}
	catalog, err := NewRulePackCatalog(origin, []SkillTrustRoot{root}, nil, []RulePackCandidate{candidate}, nil)
	if err != nil {
		t.Fatal(err)
	}
	registration := catalog.List()[0]
	if claim, err := NewRulePackRevocationClaim(
		registration, origin, "revocation:utf8:"+invalid, "security.withdrawn", "signer:revoker:utf8",
	); claim != (RulePackRevocationClaim{}) || ErrorCode(err) != ErrorRulePackInvalid {
		t.Fatalf("revocation claim=%+v error=%v", claim, err)
	}
	invalidClaim := candidate.Claim
	invalidClaim.SignatureSubjectDigest = invalid
	if payload := invalidClaim.SignaturePayload(); payload != nil {
		t.Fatalf("invalid claim produced signature payload %x", payload)
	}
	history := catalog.History()
	history[0].RevocationRef = invalid
	if digest := rulePackCatalogDigest(catalog.Origin(), []RulePackOrigin{catalog.Origin()}, catalog.List(), history); digest != "" {
		t.Fatalf("invalid history was digested: %q", digest)
	}
}

func rulePackTrust(seed byte) (ed25519.PrivateKey, SkillTrustRoot) {
	privateKey := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{seed}, ed25519.SeedSize))
	return privateKey, SkillTrustRoot{
		SignerRef: "signer:rulepacks:1", PublicKey: append([]byte(nil), privateKey[ed25519.SeedSize:]...),
	}
}

func rulePackSpec(id, version string, precedence int64, scope SkillScope, items ...RulePackItem) RulePackSpec {
	return RulePackSpec{
		ID: id, Version: version, DescriptionKey: "rulepack." + id + ".description",
		Scopes: []SkillScope{scope}, Precedence: precedence, Items: append([]RulePackItem(nil), items...),
	}
}

func signedRulePack(
	t *testing.T,
	spec RulePackSpec,
	origin RulePackOrigin,
	signerRef string,
	privateKey ed25519.PrivateKey,
) RulePackCandidate {
	t.Helper()
	claim, err := NewRulePackClaim(spec, origin, signerRef)
	if err != nil {
		t.Fatal(err)
	}
	return RulePackCandidate{Spec: cloneRulePackSpec(spec), Claim: claim, Signature: ed25519.Sign(privateKey, claim.SignaturePayload())}
}

func signedRulePackRevocation(
	t *testing.T,
	registration RulePackRegistration,
	origin RulePackOrigin,
	revocationRef, reasonCode, signerRef string,
	privateKey ed25519.PrivateKey,
) RulePackRevocationCandidate {
	t.Helper()
	claim, err := NewRulePackRevocationClaim(registration, origin, revocationRef, reasonCode, signerRef)
	if err != nil {
		t.Fatal(err)
	}
	return RulePackRevocationCandidate{Claim: claim, Signature: ed25519.Sign(privateKey, claim.SignaturePayload())}
}

func cloneRulePackSpec(source RulePackSpec) RulePackSpec {
	source.Scopes = append([]SkillScope(nil), source.Scopes...)
	source.Items = append([]RulePackItem(nil), source.Items...)
	return source
}

func findRulePack(t *testing.T, catalog *RulePackCatalog, id, version string) RulePackRegistration {
	t.Helper()
	for _, registration := range catalog.List() {
		if registration.Spec.ID == id && registration.Spec.Version == version {
			return registration
		}
	}
	t.Fatalf("rule pack %s@%s not found", id, version)
	return RulePackRegistration{}
}
