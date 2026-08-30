package acceptance_test

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"

	"orquesta/internal/goal"
	"orquesta/internal/identity"
	"orquesta/internal/tooling"
)

func TestV26TLS09ReviewedSignedTestedSkillRevocationUsesExactRollback(t *testing.T) {
	tools := v26SkillToolCatalog{}
	registry, err := tooling.NewSkillRegistry(tools,
		v26GovernedSkill("review.governed", "2"),
		v26GovernedSkill("review.governed", "1"),
	)
	if err != nil {
		t.Fatal(err)
	}
	privateKey := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x59}, ed25519.SeedSize))
	root := tooling.SkillTrustRoot{
		SignerRef: "signer:v26:release:1",
		PublicKey: append([]byte(nil), privateKey[ed25519.SeedSize:]...),
	}
	releaseOne := v26SignedRelease(t, registry, "1", root.SignerRef, privateKey)
	releaseTwo := v26SignedRelease(t, registry, "2", root.SignerRef, privateKey)
	releases := []tooling.SkillReleaseCandidate{releaseTwo, releaseOne}
	initial, err := tooling.NewSkillReleaseCatalog(registry, []tooling.SkillTrustRoot{root}, releases, nil)
	if err != nil {
		t.Fatal(err)
	}
	one, two := initial.List()[0], initial.List()[1]
	revocationClaim, err := tooling.NewSkillRevocationClaim(
		two, one, "revocation:v26:2", "security.withdrawn", root.SignerRef,
	)
	if err != nil {
		t.Fatal(err)
	}
	revocation := tooling.SkillRevocationCandidate{
		Claim: revocationClaim, Signature: ed25519.Sign(privateKey, revocationClaim.SignaturePayload()),
	}
	catalog, err := initial.Refresh(initial.Digest(), []tooling.SkillTrustRoot{root}, releases,
		[]tooling.SkillRevocationCandidate{revocation})
	if err != nil {
		t.Fatal(err)
	}
	projectAlpha, _ := goal.NewProjectRef("project:alpha")
	context, _ := tooling.NewSkillScopeContext("orquesta", projectAlpha, identity.RoleReviewer, goal.GoalRef{})
	if release, found, err := catalog.ResolveReviewed("review.governed", "2", context); err != nil || found || release.Registration.Spec.ID != "" {
		t.Fatalf("revoked release=%+v found=%v error=%v", release, found, err)
	}
	rollback, found, err := catalog.ResolveRollback("review.governed", "2", context)
	if err != nil || !found || rollback.Registration.Spec.Version != "1" || rollback.Revoked ||
		rollback.Claim.ReviewVerdict != tooling.SkillReviewApproved || rollback.Claim.TestResult != tooling.SkillTestPassed ||
		len(rollback.Registration.Spec.Permissions) != 0 {
		t.Fatalf("rollback=%+v found=%v error=%v", rollback, found, err)
	}
	projectBeta, _ := goal.NewProjectRef("project:beta")
	other, _ := tooling.NewSkillScopeContext("orquesta", projectBeta, identity.RoleReviewer, goal.GoalRef{})
	if release, found, err := catalog.ResolveRollback("review.governed", "2", other); err != nil || found || release.Registration.Spec.ID != "" {
		t.Fatalf("cross-project rollback=%+v found=%v error=%v", release, found, err)
	}
	forged := releaseOne
	forged.Signature = append([]byte(nil), releaseOne.Signature...)
	forged.Signature[0] ^= 0xff
	if result, err := tooling.NewSkillReleaseCatalog(registry, []tooling.SkillTrustRoot{root}, []tooling.SkillReleaseCandidate{forged}, nil); result != nil || tooling.SkillErrorCode(err) != tooling.ErrorSkillReleaseUntrusted {
		t.Fatalf("forged catalog=%v error=%v", result, err)
	}
}

func v26GovernedSkill(id, version string) tooling.SkillCandidate {
	instructions := []byte("---\nname: " + id + "\ndescription: Governed reviewed skill\n---\n\n# Governed instructions\n")
	digestBytes := sha256.Sum256(instructions)
	digest := "sha256:" + hex.EncodeToString(digestBytes[:])
	return tooling.SkillCandidate{
		Spec: tooling.SkillSpec{
			ID: id, Version: version, Format: tooling.SkillFormatMarkdownV1,
			DescriptionKey: "skill." + id + ".description", InstructionsRef: "artifact:" + digest,
			InstructionsDigest: digest,
			Scopes:             []tooling.SkillScope{{Kind: tooling.SkillScopeProject, ProjectRef: "project:alpha"}},
			RequiredTools:      []tooling.SkillToolRequirement{}, Permissions: []identity.Permission{},
		},
		Instructions: instructions,
	}
}

func v26SignedRelease(t *testing.T, registry *tooling.SkillRegistry, version, signerRef string, privateKey ed25519.PrivateKey) tooling.SkillReleaseCandidate {
	t.Helper()
	registration, _ := registry.Lookup("review.governed", version)
	attestation, _ := goal.NewAttestationRef("attestation:v26:" + version)
	claim, err := tooling.NewSkillReleaseClaim(registration, tooling.SkillReleaseEvidence{
		ReviewRef:    "review:v26:" + version,
		ReviewDigest: "sha256:" + strings.Repeat("a", 64), ReviewVerdict: tooling.SkillReviewApproved,
		TestAttestationRef: attestation,
		TestPolicyDigest:   "sha256:" + strings.Repeat("b", 64), TestResult: tooling.SkillTestPassed,
	}, signerRef)
	if err != nil {
		t.Fatal(err)
	}
	return tooling.SkillReleaseCandidate{
		Claim: claim, Signature: ed25519.Sign(privateKey, claim.SignaturePayload()),
	}
}
