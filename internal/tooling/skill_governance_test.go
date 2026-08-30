package tooling

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"orquesta/internal/goal"
	"orquesta/internal/identity"
)

func TestSkillReleaseCatalogBindsReviewTestsTrustPermissionsAndScope(t *testing.T) {
	registry := skillGovernanceRegistry(t, "1", "2")
	privateKey, root := skillGovernanceTrust(0x31, "signer:release:1")
	releaseOne := signedSkillRelease(t, registry, "review.status", "1", root.SignerRef, privateKey)
	releaseTwo := signedSkillRelease(t, registry, "review.status", "2", root.SignerRef, privateKey)
	catalog, err := NewSkillReleaseCatalog(registry, []SkillTrustRoot{root},
		[]SkillReleaseCandidate{releaseTwo, releaseOne}, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, unrelated := skillGovernanceTrust(0x41, "signer:unused:1")
	reordered, err := NewSkillReleaseCatalog(registry, []SkillTrustRoot{unrelated, root},
		[]SkillReleaseCandidate{releaseOne, releaseTwo}, nil)
	if err != nil || catalog.Digest() != reordered.Digest() || !strings.HasPrefix(catalog.Digest(), "sha256:") {
		t.Fatalf("catalog=%q reordered=%q error=%v", catalog.Digest(), reordered.Digest(), err)
	}
	listed := catalog.List()
	if len(listed) != 2 || listed[0].Registration.Spec.Version != "1" ||
		listed[1].Registration.Spec.Version != "2" || listed[0].Revoked ||
		listed[0].Claim.ReviewVerdict != SkillReviewApproved || listed[0].Claim.TestResult != SkillTestPassed ||
		listed[0].TrustKeyDigest != skillContentDigest(root.PublicKey) || listed[0].ReleaseDigest == "" {
		t.Fatalf("releases=%+v", listed)
	}
	signature, err := base64.RawStdEncoding.DecodeString(listed[0].Signature)
	if err != nil || !ed25519.Verify(root.PublicKey, listed[0].Claim.SignaturePayload(), signature) {
		t.Fatalf("signature=%q error=%v", listed[0].Signature, err)
	}
	project, _ := goal.NewProjectRef("project:alpha")
	context, _ := NewSkillScopeContext("orquesta", project, "", goal.GoalRef{})
	resolved, found, err := catalog.ResolveReviewed("review.status", "2", context)
	if err != nil || !found || resolved.Registration.Spec.Version != "2" ||
		!reflectPermissions(resolved.Registration.Spec.Permissions, []identity.Permission{identity.PermissionArtifactsRead}) {
		t.Fatalf("resolved=%+v found=%v error=%v", resolved, found, err)
	}
	otherProject, _ := goal.NewProjectRef("project:beta")
	other, _ := NewSkillScopeContext("orquesta", otherProject, "", goal.GoalRef{})
	if release, found, err := catalog.ResolveReviewed("review.status", "2", other); err != nil || found || release.Registration.Spec.ID != "" {
		t.Fatalf("cross-project release=%+v found=%v error=%v", release, found, err)
	}
	if _, found, err := catalog.ResolveReviewed("review.status", "2", SkillScopeContext{}); found || SkillErrorCode(err) != ErrorSkillScopeInvalid {
		t.Fatalf("zero context found=%v error=%v", found, err)
	}

	wantDigest := catalog.Digest()
	root.PublicKey[0] ^= 0xff
	releaseOne.Signature[0] ^= 0xff
	listed[0].Registration.Spec.Scopes[0].ProjectRef = "project:mutated"
	if catalog.Digest() != wantDigest || catalog.List()[0].Registration.Spec.Scopes[0].ProjectRef != "project:alpha" {
		t.Fatal("caller mutation changed release catalog")
	}
	if releaseOne.Claim.SignatureSubjectDigest == releaseTwo.Claim.SignatureSubjectDigest {
		t.Fatal("different exact registrations share release signature subject")
	}
	otherPrivate, otherRoot := skillGovernanceTrust(0x42, "signer:release:other")
	otherRelease := signedSkillRelease(t, registry, "review.status", "2", otherRoot.SignerRef, otherPrivate)
	if result, err := NewSkillReleaseCatalog(registry, []SkillTrustRoot{root, otherRoot}, []SkillReleaseCandidate{releaseOne, otherRelease}, nil); result != nil || SkillErrorCode(err) != ErrorSkillReleaseUntrusted {
		t.Fatalf("trust substitution catalog=%v error=%v", result, err)
	}
}

func TestSkillReleaseCatalogRejectsForgedOrIncompleteEvidenceAtomically(t *testing.T) {
	registry := skillGovernanceRegistry(t, "1")
	registration, _ := registry.Lookup("review.status", "1")
	privateKey, root := skillGovernanceTrust(0x32, "signer:release:1")
	valid := signedSkillRelease(t, registry, "review.status", "1", root.SignerRef, privateKey)

	attestation, _ := goal.NewAttestationRef("attestation:skill:1")
	baseEvidence := SkillReleaseEvidence{
		ReviewRef: "review:skill:1", ReviewDigest: repeatedDigest("a"), ReviewVerdict: SkillReviewApproved,
		TestAttestationRef: attestation, TestPolicyDigest: repeatedDigest("b"), TestResult: SkillTestPassed,
	}
	for name, mutate := range map[string]func(*SkillReleaseEvidence){
		"review not approved": func(value *SkillReleaseEvidence) { value.ReviewVerdict = "changes_requested" },
		"test not passed":     func(value *SkillReleaseEvidence) { value.TestResult = "fail" },
		"review ref":          func(value *SkillReleaseEvidence) { value.ReviewRef = "" },
		"review invalid utf8": func(value *SkillReleaseEvidence) { value.ReviewRef = "review:\xff" },
		"review control":      func(value *SkillReleaseEvidence) { value.ReviewRef = "review:a\nb" },
		"review digest":       func(value *SkillReleaseEvidence) { value.ReviewDigest = "sha256:short" },
		"review uppercase":    func(value *SkillReleaseEvidence) { value.ReviewDigest = strings.ToUpper(value.ReviewDigest) },
		"test ref":            func(value *SkillReleaseEvidence) { value.TestAttestationRef = goal.AttestationRef{} },
		"test invalid utf8": func(value *SkillReleaseEvidence) {
			value.TestAttestationRef, _ = goal.NewAttestationRef("attestation:\xff")
		},
		"test policy": func(value *SkillReleaseEvidence) { value.TestPolicyDigest = "sha256:short" },
	} {
		t.Run(name, func(t *testing.T) {
			evidence := baseEvidence
			mutate(&evidence)
			if claim, err := NewSkillReleaseClaim(registration, evidence, root.SignerRef); claim != (SkillReleaseClaim{}) || SkillErrorCode(err) != ErrorSkillReleaseInvalid {
				t.Fatalf("claim=%+v error=%v", claim, err)
			}
		})
	}

	mutations := map[string]func(*SkillReleaseCandidate){
		"registration":   func(value *SkillReleaseCandidate) { value.Claim.RegistrationDigest = repeatedDigest("0") },
		"review subject": func(value *SkillReleaseCandidate) { value.Claim.ReviewSubjectDigest = repeatedDigest("0") },
		"review digest":  func(value *SkillReleaseCandidate) { value.Claim.ReviewDigest = repeatedDigest("0") },
		"test subject":   func(value *SkillReleaseCandidate) { value.Claim.TestSubjectDigest = repeatedDigest("0") },
		"test result":    func(value *SkillReleaseCandidate) { value.Claim.TestResult = "fail" },
		"subject digest": func(value *SkillReleaseCandidate) { value.Claim.SignatureSubjectDigest = repeatedDigest("0") },
		"signature":      func(value *SkillReleaseCandidate) { value.Signature[0] ^= 0xff },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			candidate := cloneReleaseCandidate(valid)
			mutate(&candidate)
			catalog, err := NewSkillReleaseCatalog(registry, []SkillTrustRoot{root}, []SkillReleaseCandidate{candidate}, nil)
			if catalog != nil || (SkillErrorCode(err) != ErrorSkillReleaseInvalid && SkillErrorCode(err) != ErrorSkillReleaseUntrusted) {
				t.Fatalf("catalog=%v error=%v code=%q", catalog, err, SkillErrorCode(err))
			}
		})
	}
	otherPrivate, otherRoot := skillGovernanceTrust(0x42, "signer:other:1")
	untrusted := signedSkillRelease(t, registry, "review.status", "1", otherRoot.SignerRef, otherPrivate)
	if catalog, err := NewSkillReleaseCatalog(registry, []SkillTrustRoot{root}, []SkillReleaseCandidate{untrusted}, nil); catalog != nil || SkillErrorCode(err) != ErrorSkillReleaseUntrusted {
		t.Fatalf("untrusted catalog=%v error=%v", catalog, err)
	}
	if catalog, err := NewSkillReleaseCatalog(registry, []SkillTrustRoot{root}, []SkillReleaseCandidate{valid, valid}, nil); catalog != nil || SkillErrorCode(err) != ErrorSkillReleaseDuplicate {
		t.Fatalf("duplicate catalog=%v error=%v", catalog, err)
	}
	if catalog, err := NewSkillReleaseCatalog(nil, []SkillTrustRoot{root}, nil, nil); catalog != nil || SkillErrorCode(err) != ErrorSkillReleaseInvalid {
		t.Fatalf("nil registry catalog=%v error=%v", catalog, err)
	}
	for _, roots := range [][]SkillTrustRoot{
		{{SignerRef: "", PublicKey: root.PublicKey}},
		{{SignerRef: "signer:\xff", PublicKey: root.PublicKey}},
		{{SignerRef: root.SignerRef, PublicKey: []byte("short")}},
		{root, root},
	} {
		if catalog, err := NewSkillReleaseCatalog(registry, roots, []SkillReleaseCandidate{valid}, nil); catalog != nil ||
			(SkillErrorCode(err) != ErrorSkillReleaseInvalid && SkillErrorCode(err) != ErrorSkillReleaseDuplicate) {
			t.Fatalf("roots=%+v catalog=%v error=%v", roots, catalog, err)
		}
	}
	if catalog, err := NewSkillReleaseCatalog(registry, make([]SkillTrustRoot, MaxSkillTrustRoots+1), nil, nil); catalog != nil || SkillErrorCode(err) != ErrorSkillReleaseInvalid {
		t.Fatalf("unbounded roots catalog=%v error=%v", catalog, err)
	}
	if catalog, err := NewSkillReleaseCatalog(registry, nil, make([]SkillReleaseCandidate, MaxSkillReleaseEntries+1), nil); catalog != nil || SkillErrorCode(err) != ErrorSkillReleaseInvalid {
		t.Fatalf("unbounded releases catalog=%v error=%v", catalog, err)
	}
	invalidTools := skillTestTools(t)
	status, _ := invalidTools.LookupSkillTool("status.read", "1")
	invalidScope := skillCandidate("review.invalid", "1", status)
	invalidScope.Spec.Scopes = []SkillScope{{Kind: SkillScopeProduct, ProductID: "product:\xff"}}
	invalidRegistry, err := NewSkillRegistry(invalidTools, invalidScope)
	if invalidRegistry != nil || SkillErrorCode(err) != ErrorSkillSpecInvalid {
		t.Fatalf("invalid scope registry=%v error=%v", invalidRegistry, err)
	}
	encoded, err := json.Marshal(valid)
	if err != nil || bytes.Contains(encoded, valid.Signature) {
		t.Fatalf("candidate leaked signature: %s error=%v", encoded, err)
	}
}

func TestSkillRevocationRequiresSignedExplicitLowerReviewedRollback(t *testing.T) {
	registry := skillGovernanceRegistry(t, "1", "2", "3")
	privateKey, root := skillGovernanceTrust(0x33, "signer:release:1")
	releases := []SkillReleaseCandidate{
		signedSkillRelease(t, registry, "review.status", "3", root.SignerRef, privateKey),
		signedSkillRelease(t, registry, "review.status", "1", root.SignerRef, privateKey),
		signedSkillRelease(t, registry, "review.status", "2", root.SignerRef, privateKey),
	}
	revocation := signedSkillRevocation(t, registry, "3", "1", root.SignerRef, privateKey)
	initial, err := NewSkillReleaseCatalog(registry, []SkillTrustRoot{root}, releases, nil)
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := initial.Refresh(initial.Digest(), []SkillTrustRoot{root}, releases,
		[]SkillRevocationCandidate{revocation})
	if err != nil {
		t.Fatal(err)
	}
	listed := catalog.List()
	if len(listed) != 3 || !listed[2].Revoked || listed[2].RevocationRef != "revocation:skill:3" ||
		listed[2].RevocationReasonCode != "security.withdrawn" ||
		listed[2].RevocationSignerRef != root.SignerRef || listed[2].RevocationTrustKeyDigest != skillContentDigest(root.PublicKey) ||
		listed[2].RevocationSubjectDigest != revocation.Claim.SignatureSubjectDigest || listed[2].RevocationSignature == "" {
		t.Fatalf("revocation evidence=%+v", listed)
	}
	revocationSignature, err := base64.RawStdEncoding.DecodeString(listed[2].RevocationSignature)
	if err != nil || !ed25519.Verify(root.PublicKey, revocation.Claim.SignaturePayload(), revocationSignature) {
		t.Fatalf("revocation signature=%q error=%v", listed[2].RevocationSignature, err)
	}
	project, _ := goal.NewProjectRef("project:alpha")
	context, _ := NewSkillScopeContext("orquesta", project, "", goal.GoalRef{})
	if release, found, err := catalog.ResolveReviewed("review.status", "3", context); err != nil || found || release.Registration.Spec.ID != "" {
		t.Fatalf("revoked release=%+v found=%v error=%v", release, found, err)
	}
	rollback, found, err := catalog.ResolveRollback("review.status", "3", context)
	if err != nil || !found || rollback.Registration.Spec.Version != "1" || rollback.Revoked {
		t.Fatalf("rollback=%+v found=%v error=%v", rollback, found, err)
	}
	if release, found, err := catalog.ResolveRollback("review.status", "2", context); err != nil || found || release.Registration.Spec.ID != "" {
		t.Fatalf("unrevoked rollback=%+v found=%v error=%v", release, found, err)
	}
	sourceOne, sourceThree := initial.List()[0], initial.List()[2]
	if claim, err := NewSkillRevocationClaim(sourceOne, sourceThree, "revocation:bad", "security.withdrawn", root.SignerRef); claim != (SkillRevocationClaim{}) || SkillErrorCode(err) != ErrorSkillReleaseInvalid {
		t.Fatalf("forward rollback claim=%+v error=%v", claim, err)
	}
	forged := revocation
	forged.Signature = append([]byte(nil), revocation.Signature...)
	forged.Signature[0] ^= 0xff
	if result, err := NewSkillReleaseCatalog(registry, []SkillTrustRoot{root}, releases, []SkillRevocationCandidate{forged}); result != nil || SkillErrorCode(err) != ErrorSkillReleaseUntrusted {
		t.Fatalf("forged revocation catalog=%v error=%v", result, err)
	}
	revokeTwo := signedSkillRevocation(t, registry, "2", "1", root.SignerRef, privateKey)
	revokeThreeToTwo := signedSkillRevocation(t, registry, "3", "2", root.SignerRef, privateKey)
	if result, err := NewSkillReleaseCatalog(registry, []SkillTrustRoot{root}, releases, []SkillRevocationCandidate{revokeThreeToTwo, revokeTwo}); result != nil || SkillErrorCode(err) != ErrorSkillReleaseInvalid {
		t.Fatalf("revoked rollback target catalog=%v error=%v", result, err)
	}
	if result, err := NewSkillReleaseCatalog(registry, []SkillTrustRoot{root}, releases[0:1], []SkillRevocationCandidate{revocation}); result != nil || SkillErrorCode(err) != ErrorSkillReleaseInvalid {
		t.Fatalf("missing rollback release catalog=%v error=%v", result, err)
	}
	if result, err := catalog.Refresh(initial.Digest(), []SkillTrustRoot{root}, releases, []SkillRevocationCandidate{revocation}); result != nil || SkillErrorCode(err) != ErrorSkillReleaseStale {
		t.Fatalf("stale refresh catalog=%v error=%v", result, err)
	}
	if replay, err := catalog.Refresh(catalog.Digest(), []SkillTrustRoot{root}, releases, []SkillRevocationCandidate{revocation}); err != nil || replay.Digest() != catalog.Digest() {
		t.Fatalf("exact replay catalog=%v error=%v", replay, err)
	}
	if result, err := catalog.Refresh(catalog.Digest(), []SkillTrustRoot{root}, releases, nil); result != nil || SkillErrorCode(err) != ErrorSkillReleaseInvalid {
		t.Fatalf("revocation resurrection catalog=%v error=%v", result, err)
	}
	all := initial.List()
	duplicateClaim, err := NewSkillRevocationClaim(all[1], all[0], revocation.Claim.RevocationRef, "security.withdrawn", root.SignerRef)
	if err != nil {
		t.Fatal(err)
	}
	duplicate := SkillRevocationCandidate{Claim: duplicateClaim, Signature: ed25519.Sign(privateKey, duplicateClaim.SignaturePayload())}
	if result, err := NewSkillReleaseCatalog(registry, []SkillTrustRoot{root}, releases, []SkillRevocationCandidate{revocation, duplicate}); result != nil || SkillErrorCode(err) != ErrorSkillReleaseDuplicate {
		t.Fatalf("duplicate revocation ref catalog=%v error=%v", result, err)
	}
	three, _ := registry.Lookup("review.status", "3")
	attestation, _ := goal.NewAttestationRef("attestation:skill:3")
	alternateClaim, err := NewSkillReleaseClaim(three, SkillReleaseEvidence{
		ReviewRef: "review:skill:3", ReviewDigest: repeatedDigest("c"), ReviewVerdict: SkillReviewApproved,
		TestAttestationRef: attestation, TestPolicyDigest: repeatedDigest("b"), TestResult: SkillTestPassed,
	}, root.SignerRef)
	if err != nil {
		t.Fatal(err)
	}
	alternate := SkillReleaseCandidate{Claim: alternateClaim, Signature: ed25519.Sign(privateKey, alternateClaim.SignaturePayload())}
	if result, err := NewSkillReleaseCatalog(registry, []SkillTrustRoot{root}, []SkillReleaseCandidate{alternate, releases[1], releases[2]}, []SkillRevocationCandidate{revocation}); result != nil || SkillErrorCode(err) != ErrorSkillReleaseInvalid {
		t.Fatalf("revocation transplanted across evidence catalog=%v error=%v", result, err)
	}
	_, otherRoot := skillGovernanceTrust(0x43, "signer:release:other")
	if claim, err := NewSkillRevocationClaim(sourceThree, sourceOne, "revocation:other", "security.withdrawn", otherRoot.SignerRef); claim != (SkillRevocationClaim{}) || SkillErrorCode(err) != ErrorSkillReleaseInvalid {
		t.Fatalf("unauthorized revoker claim=%+v error=%v", claim, err)
	}
}

func skillGovernanceRegistry(t *testing.T, versions ...string) *SkillRegistry {
	t.Helper()
	tools := skillTestTools(t)
	status, _ := tools.LookupSkillTool("status.read", "1")
	candidates := make([]SkillCandidate, len(versions))
	for index, version := range versions {
		candidates[index] = skillCandidate("review.status", version, status)
		candidates[index].Spec.Scopes = []SkillScope{{Kind: SkillScopeProject, ProjectRef: "project:alpha"}}
	}
	registry, err := NewSkillRegistry(tools, candidates...)
	if err != nil {
		t.Fatal(err)
	}
	return registry
}

func skillGovernanceTrust(fill byte, signerRef string) (ed25519.PrivateKey, SkillTrustRoot) {
	privateKey := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{fill}, ed25519.SeedSize))
	return privateKey, SkillTrustRoot{
		SignerRef: signerRef, PublicKey: append([]byte(nil), privateKey[ed25519.SeedSize:]...),
	}
}

func signedSkillRelease(t *testing.T, registry *SkillRegistry, id, version, signerRef string, privateKey ed25519.PrivateKey) SkillReleaseCandidate {
	t.Helper()
	registration, found := registry.Lookup(id, version)
	if !found {
		t.Fatalf("missing registration %s@%s", id, version)
	}
	attestation, _ := goal.NewAttestationRef("attestation:skill:" + version)
	claim, err := NewSkillReleaseClaim(registration, SkillReleaseEvidence{
		ReviewRef: "review:skill:" + version, ReviewDigest: repeatedDigest("a"), ReviewVerdict: SkillReviewApproved,
		TestAttestationRef: attestation, TestPolicyDigest: repeatedDigest("b"), TestResult: SkillTestPassed,
	}, signerRef)
	if err != nil {
		t.Fatal(err)
	}
	return SkillReleaseCandidate{Claim: claim, Signature: ed25519.Sign(privateKey, claim.SignaturePayload())}
}

func signedSkillRevocation(t *testing.T, registry *SkillRegistry, sourceVersion, rollbackVersion, signerRef string, privateKey ed25519.PrivateKey) SkillRevocationCandidate {
	t.Helper()
	root := SkillTrustRoot{SignerRef: signerRef, PublicKey: append([]byte(nil), privateKey[ed25519.SeedSize:]...)}
	base, err := NewSkillReleaseCatalog(registry, []SkillTrustRoot{root}, []SkillReleaseCandidate{
		signedSkillRelease(t, registry, "review.status", sourceVersion, signerRef, privateKey),
		signedSkillRelease(t, registry, "review.status", rollbackVersion, signerRef, privateKey),
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	var source, rollback SkillRelease
	for _, release := range base.List() {
		if release.Registration.Spec.Version == sourceVersion {
			source = release
		}
		if release.Registration.Spec.Version == rollbackVersion {
			rollback = release
		}
	}
	claim, err := NewSkillRevocationClaim(source, rollback, "revocation:skill:"+sourceVersion, "security.withdrawn", signerRef)
	if err != nil {
		t.Fatal(err)
	}
	return SkillRevocationCandidate{Claim: claim, Signature: ed25519.Sign(privateKey, claim.SignaturePayload())}
}

func cloneReleaseCandidate(source SkillReleaseCandidate) SkillReleaseCandidate {
	source.Signature = append([]byte(nil), source.Signature...)
	return source
}

func repeatedDigest(character string) string {
	return "sha256:" + strings.Repeat(character, 64)
}

func reflectPermissions(got, want []identity.Permission) bool {
	if len(got) != len(want) {
		return false
	}
	for index := range got {
		if got[index] != want[index] {
			return false
		}
	}
	return true
}
