package tooling

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"orquesta/internal/goal"
)

const (
	ErrorSkillReleaseInvalid                                      = "tooling.skill_release_invalid"
	ErrorSkillReleaseDuplicate                                    = "tooling.skill_release_duplicate"
	ErrorSkillReleaseUntrusted                                    = "tooling.skill_release_untrusted"
	ErrorSkillReleaseStale                                        = "tooling.skill_release_stale"
	MaxSkillTrustRoots, MaxSkillReleaseEntries                    = 64, 1024
	SkillReviewApproved                        SkillReviewVerdict = "approved"
	SkillTestPassed                            SkillTestResult    = "pass"
)

type SkillReviewVerdict string
type SkillTestResult string
type SkillTrustRoot struct {
	SignerRef string `json:"signer_ref"`
	PublicKey []byte `json:"-"`
}
type SkillReleaseEvidence struct {
	ReviewRef          string
	ReviewDigest       string
	ReviewVerdict      SkillReviewVerdict
	TestAttestationRef goal.AttestationRef
	TestPolicyDigest   string
	TestResult         SkillTestResult
}
type SkillReleaseClaim struct {
	ID                     string             `json:"id"`
	Version                string             `json:"version"`
	RegistrationDigest     string             `json:"registration_digest"`
	ReviewRef              string             `json:"review_ref"`
	ReviewSubjectDigest    string             `json:"review_subject_digest"`
	ReviewDigest           string             `json:"review_digest"`
	ReviewVerdict          SkillReviewVerdict `json:"review_verdict"`
	TestAttestationRef     string             `json:"test_attestation_ref"`
	TestSubjectDigest      string             `json:"test_subject_digest"`
	TestPolicyDigest       string             `json:"test_policy_digest"`
	TestResult             SkillTestResult    `json:"test_result"`
	SignerRef              string             `json:"signer_ref"`
	SignatureSubjectDigest string             `json:"signature_subject_digest"`
}
type SkillReleaseCandidate struct {
	Claim     SkillReleaseClaim `json:"claim"`
	Signature []byte            `json:"-"`
}
type SkillRevocationClaim struct {
	ID                         string `json:"id"`
	Version                    string `json:"version"`
	RegistrationDigest         string `json:"registration_digest"`
	SourceReleaseDigest        string `json:"source_release_digest"`
	RevocationRef              string `json:"revocation_ref"`
	ReasonCode                 string `json:"reason_code"`
	RollbackVersion            string `json:"rollback_version"`
	RollbackRegistrationDigest string `json:"rollback_registration_digest"`
	RollbackReleaseDigest      string `json:"rollback_release_digest"`
	SignerRef                  string `json:"signer_ref"`
	SignatureSubjectDigest     string `json:"signature_subject_digest"`
}
type SkillRevocationCandidate struct {
	Claim     SkillRevocationClaim `json:"claim"`
	Signature []byte               `json:"-"`
}
type SkillRelease struct {
	Registration             SkillRegistration `json:"registration"`
	Claim                    SkillReleaseClaim `json:"claim"`
	TrustKeyDigest           string            `json:"trust_key_digest"`
	Signature                string            `json:"signature"`
	Revoked                  bool              `json:"revoked"`
	RevocationRef            string            `json:"revocation_ref,omitempty"`
	RevocationReasonCode     string            `json:"revocation_reason_code,omitempty"`
	RevocationSubjectDigest  string            `json:"revocation_subject_digest,omitempty"`
	RevocationSignerRef      string            `json:"revocation_signer_ref,omitempty"`
	RevocationTrustKeyDigest string            `json:"revocation_trust_key_digest,omitempty"`
	RevocationSignature      string            `json:"revocation_signature,omitempty"`
	RollbackVersion          string            `json:"rollback_version,omitempty"`
	RollbackDigest           string            `json:"rollback_registration_digest,omitempty"`
	ReleaseDigest            string            `json:"release_digest"`
}
type SkillReleaseCatalog struct {
	registry *SkillRegistry
	entries  []SkillRelease
	byKey    map[string]int
	digest   string
}

func NewSkillReleaseClaim(
	registration SkillRegistration,
	evidence SkillReleaseEvidence,
	signerRef string,
) (SkillReleaseClaim, error) {
	if !validSkillID(registration.Spec.ID) || !validSkillReleaseRegistration(registration) ||
		registration.Digest != skillSpecDigest(registration.Spec) {
		return SkillReleaseClaim{}, skillContractError(ErrorSkillReleaseInvalid, "registration")
	}
	if _, err := parseSkillVersion(registration.Spec.Version); err != nil {
		return SkillReleaseClaim{}, skillContractError(ErrorSkillReleaseInvalid, "version")
	}
	if !validSkillEvidenceRef(evidence.ReviewRef) || !validCanonicalSkillDigest(evidence.ReviewDigest) ||
		evidence.ReviewVerdict != SkillReviewApproved || !validSkillEvidenceRef(evidence.TestAttestationRef.String()) ||
		!validCanonicalSkillDigest(evidence.TestPolicyDigest) || evidence.TestResult != SkillTestPassed ||
		!validSkillSignerRef(signerRef) {
		return SkillReleaseClaim{}, skillContractError(ErrorSkillReleaseInvalid, "evidence")
	}
	reviewSubject := skillFactDigest("orquesta.tooling.skill-review-subject.v1", registration.Digest)
	testSubject := skillFactDigest(
		"orquesta.tooling.skill-test-subject.v1", registration.Digest, evidence.TestPolicyDigest,
	)
	claim := SkillReleaseClaim{
		ID: registration.Spec.ID, Version: registration.Spec.Version,
		RegistrationDigest: registration.Digest,
		ReviewRef:          evidence.ReviewRef, ReviewSubjectDigest: reviewSubject,
		ReviewDigest: evidence.ReviewDigest, ReviewVerdict: evidence.ReviewVerdict,
		TestAttestationRef: evidence.TestAttestationRef.String(), TestSubjectDigest: testSubject,
		TestPolicyDigest: evidence.TestPolicyDigest, TestResult: evidence.TestResult, SignerRef: signerRef,
	}
	claim.SignatureSubjectDigest = skillFactDigest(
		"orquesta.tooling.skill-release-claim.v1", claim.ID, claim.Version,
		claim.RegistrationDigest, claim.ReviewRef, claim.ReviewSubjectDigest, claim.ReviewDigest,
		string(claim.ReviewVerdict), claim.TestAttestationRef, claim.TestSubjectDigest,
		claim.TestPolicyDigest, string(claim.TestResult), claim.SignerRef,
	)
	return claim, nil
}
func (claim SkillReleaseClaim) SignaturePayload() []byte {
	return []byte("orquesta.tooling.skill-release-signature.v1\x00" + claim.SignatureSubjectDigest)
}
func NewSkillRevocationClaim(
	source SkillRelease,
	rollback SkillRelease,
	revocationRef string,
	reasonCode string,
	signerRef string,
) (SkillRevocationClaim, error) {
	sourceVersion, sourceErr := parseSkillVersion(source.Registration.Spec.Version)
	rollbackVersion, rollbackErr := parseSkillVersion(rollback.Registration.Spec.Version)
	if sourceErr != nil || rollbackErr != nil || source.Registration.Spec.ID != rollback.Registration.Spec.ID ||
		rollbackVersion >= sourceVersion || source.Registration.Digest != skillSpecDigest(source.Registration.Spec) ||
		rollback.Registration.Digest != skillSpecDigest(rollback.Registration.Spec) || source.Revoked || rollback.Revoked ||
		source.ReleaseDigest != skillReleaseDigest(source) || rollback.ReleaseDigest != skillReleaseDigest(rollback) ||
		source.Claim.SignerRef != signerRef || rollback.Claim.SignerRef != signerRef ||
		source.TrustKeyDigest != rollback.TrustKeyDigest || !validSkillEvidenceRef(revocationRef) ||
		!validSkillID(reasonCode) || !validSkillSignerRef(signerRef) {
		return SkillRevocationClaim{}, skillContractError(ErrorSkillReleaseInvalid, "revocation")
	}
	claim := SkillRevocationClaim{
		ID: source.Registration.Spec.ID, Version: source.Registration.Spec.Version,
		RegistrationDigest: source.Registration.Digest, SourceReleaseDigest: source.ReleaseDigest,
		RevocationRef: revocationRef, ReasonCode: reasonCode,
		RollbackVersion:            rollback.Registration.Spec.Version,
		RollbackRegistrationDigest: rollback.Registration.Digest, RollbackReleaseDigest: rollback.ReleaseDigest,
		SignerRef: signerRef,
	}
	claim.SignatureSubjectDigest = skillFactDigest(
		"orquesta.tooling.skill-revocation-claim.v1", claim.ID, claim.Version,
		claim.RegistrationDigest, claim.SourceReleaseDigest, claim.RevocationRef, claim.ReasonCode,
		claim.RollbackVersion, claim.RollbackRegistrationDigest, claim.RollbackReleaseDigest, claim.SignerRef,
	)
	return claim, nil
}
func (claim SkillRevocationClaim) SignaturePayload() []byte {
	return []byte("orquesta.tooling.skill-revocation-signature.v1\x00" + claim.SignatureSubjectDigest)
}
func NewSkillReleaseCatalog(
	registry *SkillRegistry,
	trustRoots []SkillTrustRoot,
	releases []SkillReleaseCandidate,
	revocations []SkillRevocationCandidate,
) (*SkillReleaseCatalog, error) {
	if registry == nil || len(trustRoots) > MaxSkillTrustRoots || len(releases) > MaxSkillReleaseEntries ||
		len(revocations) > len(releases) {
		return nil, skillContractError(ErrorSkillReleaseInvalid, "registry")
	}
	trusted, err := canonicalSkillTrustRoots(trustRoots)
	if err != nil {
		return nil, err
	}
	entries := make([]SkillRelease, 0, len(releases))
	byKey := make(map[string]int, len(releases))
	bySkillTrust := make(map[string]string)
	for _, candidate := range releases {
		registration, found := registry.Lookup(candidate.Claim.ID, candidate.Claim.Version)
		if !found || registration.Digest != candidate.Claim.RegistrationDigest {
			return nil, skillContractError(ErrorSkillReleaseInvalid, "registration")
		}
		expected, err := releaseClaimFromStored(registration, candidate.Claim)
		if err != nil || expected != candidate.Claim {
			return nil, skillContractError(ErrorSkillReleaseInvalid, "claim")
		}
		publicKey, trustedSigner := trusted[candidate.Claim.SignerRef]
		if !trustedSigner || len(candidate.Signature) != ed25519.SignatureSize ||
			!ed25519.Verify(publicKey, candidate.Claim.SignaturePayload(), candidate.Signature) {
			return nil, skillContractError(ErrorSkillReleaseUntrusted, "signature")
		}
		key := skillRegistryKey(candidate.Claim.ID, candidate.Claim.Version)
		if _, duplicate := byKey[key]; duplicate {
			return nil, skillContractError(ErrorSkillReleaseDuplicate, "release")
		}
		trustIdentity := candidate.Claim.SignerRef + "\x00" + skillContentDigest(publicKey)
		if previous, found := bySkillTrust[candidate.Claim.ID]; found && previous != trustIdentity {
			return nil, skillContractError(ErrorSkillReleaseUntrusted, "trust_substitution")
		}
		bySkillTrust[candidate.Claim.ID], byKey[key] = trustIdentity, len(entries)
		entries = append(entries, SkillRelease{
			Registration: registration, Claim: candidate.Claim,
			TrustKeyDigest: skillContentDigest(publicKey),
			Signature:      base64.RawStdEncoding.EncodeToString(candidate.Signature),
		})
	}
	for index := range entries {
		entries[index].ReleaseDigest = skillReleaseDigest(entries[index])
	}
	seenRevocations := make(map[string]struct{}, len(revocations))
	for _, candidate := range revocations {
		if _, duplicate := seenRevocations[candidate.Claim.RevocationRef]; duplicate {
			return nil, skillContractError(ErrorSkillReleaseDuplicate, "revocation_ref")
		}
		seenRevocations[candidate.Claim.RevocationRef] = struct{}{}
		sourceIndex, sourceFound := byKey[skillRegistryKey(candidate.Claim.ID, candidate.Claim.Version)]
		rollbackIndex, rollbackFound := byKey[skillRegistryKey(candidate.Claim.ID, candidate.Claim.RollbackVersion)]
		if !sourceFound || !rollbackFound || entries[sourceIndex].Revoked {
			return nil, skillContractError(ErrorSkillReleaseInvalid, "revocation")
		}
		expected, err := NewSkillRevocationClaim(
			entries[sourceIndex], entries[rollbackIndex],
			candidate.Claim.RevocationRef, candidate.Claim.ReasonCode, candidate.Claim.SignerRef,
		)
		if err != nil || expected != candidate.Claim {
			return nil, skillContractError(ErrorSkillReleaseInvalid, "revocation")
		}
		publicKey, trustedSigner := trusted[candidate.Claim.SignerRef]
		if !trustedSigner || len(candidate.Signature) != ed25519.SignatureSize ||
			!ed25519.Verify(publicKey, candidate.Claim.SignaturePayload(), candidate.Signature) {
			return nil, skillContractError(ErrorSkillReleaseUntrusted, "revocation_signature")
		}
		entries[sourceIndex].Revoked = true
		entries[sourceIndex].RevocationRef = candidate.Claim.RevocationRef
		entries[sourceIndex].RevocationReasonCode = candidate.Claim.ReasonCode
		entries[sourceIndex].RevocationSubjectDigest = candidate.Claim.SignatureSubjectDigest
		entries[sourceIndex].RevocationSignerRef = candidate.Claim.SignerRef
		entries[sourceIndex].RevocationTrustKeyDigest = skillContentDigest(publicKey)
		entries[sourceIndex].RevocationSignature = base64.RawStdEncoding.EncodeToString(candidate.Signature)
		entries[sourceIndex].RollbackVersion = candidate.Claim.RollbackVersion
		entries[sourceIndex].RollbackDigest = candidate.Claim.RollbackRegistrationDigest
	}
	for _, entry := range entries {
		if !entry.Revoked {
			continue
		}
		rollback := entries[byKey[skillRegistryKey(entry.Registration.Spec.ID, entry.RollbackVersion)]]
		if rollback.Revoked || rollback.Registration.Digest != entry.RollbackDigest {
			return nil, skillContractError(ErrorSkillReleaseInvalid, "rollback")
		}
	}
	sort.Slice(entries, func(left, right int) bool {
		return skillIdentityLess(
			entries[left].Registration.Spec.ID, entries[left].Registration.Spec.Version,
			entries[right].Registration.Spec.ID, entries[right].Registration.Spec.Version,
		)
	})
	byKey = make(map[string]int, len(entries))
	for index := range entries {
		entries[index].ReleaseDigest = skillReleaseDigest(entries[index])
		byKey[skillRegistryKey(entries[index].Registration.Spec.ID, entries[index].Registration.Spec.Version)] = index
	}
	return &SkillReleaseCatalog{
		registry: registry, entries: entries, byKey: byKey, digest: skillReleaseCatalogDigest(entries),
	}, nil
}
func (catalog *SkillReleaseCatalog) List() []SkillRelease {
	if catalog == nil {
		return nil
	}
	result := make([]SkillRelease, len(catalog.entries))
	for index, entry := range catalog.entries {
		result[index] = cloneSkillRelease(entry)
	}
	return result
}

func (catalog *SkillReleaseCatalog) Digest() string {
	if catalog == nil {
		return ""
	}
	return catalog.digest
}

func (catalog *SkillReleaseCatalog) ResolveReviewed(
	id string,
	version string,
	context SkillScopeContext,
) (SkillRelease, bool, error) {
	if catalog == nil {
		return SkillRelease{}, false, nil
	}
	if _, visible, err := catalog.registry.ResolveScoped(id, version, context); err != nil || !visible {
		return SkillRelease{}, false, err
	}
	index, found := catalog.byKey[skillRegistryKey(id, version)]
	if !found || catalog.entries[index].Revoked {
		return SkillRelease{}, false, nil
	}
	return cloneSkillRelease(catalog.entries[index]), true, nil
}

func (catalog *SkillReleaseCatalog) ResolveRollback(
	id string,
	revokedVersion string,
	context SkillScopeContext,
) (SkillRelease, bool, error) {
	if catalog == nil {
		return SkillRelease{}, false, nil
	}
	if _, visible, err := catalog.registry.ResolveScoped(id, revokedVersion, context); err != nil || !visible {
		return SkillRelease{}, false, err
	}
	sourceIndex, found := catalog.byKey[skillRegistryKey(id, revokedVersion)]
	if !found || !catalog.entries[sourceIndex].Revoked {
		return SkillRelease{}, false, nil
	}
	rollbackVersion := catalog.entries[sourceIndex].RollbackVersion
	if _, visible, err := catalog.registry.ResolveScoped(id, rollbackVersion, context); err != nil || !visible {
		return SkillRelease{}, false, err
	}
	rollback := catalog.entries[catalog.byKey[skillRegistryKey(id, rollbackVersion)]]
	return cloneSkillRelease(rollback), true, nil
}

// Refresh uses a local digest fence; it is not persistence or durable CAS.
func (catalog *SkillReleaseCatalog) Refresh(
	previousCatalogDigest string,
	trustRoots []SkillTrustRoot,
	releases []SkillReleaseCandidate,
	revocations []SkillRevocationCandidate,
) (*SkillReleaseCatalog, error) {
	if catalog == nil || previousCatalogDigest != catalog.digest || !validCanonicalSkillDigest(previousCatalogDigest) {
		return nil, skillContractError(ErrorSkillReleaseStale, "previous_catalog")
	}
	next, err := NewSkillReleaseCatalog(catalog.registry, trustRoots, releases, revocations)
	if err != nil {
		return nil, err
	}
	if len(next.entries) != len(catalog.entries) {
		return nil, skillContractError(ErrorSkillReleaseInvalid, "history")
	}
	for _, current := range catalog.entries {
		index, found := next.byKey[skillRegistryKey(current.Registration.Spec.ID, current.Registration.Spec.Version)]
		if !found || !sameSkillReleasePublication(current, next.entries[index]) ||
			current.Revoked && current.ReleaseDigest != next.entries[index].ReleaseDigest {
			return nil, skillContractError(ErrorSkillReleaseInvalid, "history")
		}
	}
	return next, nil
}

func canonicalSkillTrustRoots(source []SkillTrustRoot) (map[string]ed25519.PublicKey, error) {
	trusted := make(map[string]ed25519.PublicKey, len(source))
	for _, root := range source {
		if !validSkillSignerRef(root.SignerRef) || len(root.PublicKey) != ed25519.PublicKeySize {
			return nil, skillContractError(ErrorSkillReleaseInvalid, "trust_root")
		}
		if _, duplicate := trusted[root.SignerRef]; duplicate {
			return nil, skillContractError(ErrorSkillReleaseDuplicate, "trust_root")
		}
		trusted[root.SignerRef] = append(ed25519.PublicKey(nil), root.PublicKey...)
	}
	return trusted, nil
}

func releaseClaimFromStored(
	registration SkillRegistration,
	claim SkillReleaseClaim,
) (SkillReleaseClaim, error) {
	attestationRef, err := goal.NewAttestationRef(claim.TestAttestationRef)
	if err != nil {
		return SkillReleaseClaim{}, err
	}
	return NewSkillReleaseClaim(registration, SkillReleaseEvidence{
		ReviewRef: claim.ReviewRef, ReviewDigest: claim.ReviewDigest, ReviewVerdict: claim.ReviewVerdict,
		TestAttestationRef: attestationRef, TestPolicyDigest: claim.TestPolicyDigest,
		TestResult: claim.TestResult,
	}, claim.SignerRef)
}

func validSkillEvidenceRef(value string) bool { return validSkillGovernanceToken(value) }

func validSkillSignerRef(value string) bool {
	return len(value) <= 200 && validSkillGovernanceToken(value)
}

func validCanonicalSkillDigest(value string) bool {
	return validSkillDigest(value) && value == strings.ToLower(value)
}

func validSkillReleaseRegistration(registration SkillRegistration) bool {
	for _, scope := range registration.Spec.Scopes {
		for _, value := range []string{string(scope.Kind), scope.ProductID, scope.ProjectRef, string(scope.Role), scope.GoalRef} {
			if value != "" && (len(value) > 200 || !validSkillGovernanceToken(value)) {
				return false
			}
		}
	}
	return true
}

func validSkillGovernanceToken(value string) bool {
	if value == "" || len(value) > 1024 || !utf8.ValidString(value) || strings.TrimSpace(value) != value {
		return false
	}
	for _, character := range value {
		if unicode.IsControl(character) {
			return false
		}
	}
	return true
}

func sameSkillReleasePublication(left, right SkillRelease) bool {
	return left.Registration.Digest == right.Registration.Digest && left.Claim == right.Claim && left.TrustKeyDigest == right.TrustKeyDigest && left.Signature == right.Signature
}

func skillFactDigest(contract string, fields ...string) string {
	return skillGovernanceJSONDigest(struct {
		Contract string   `json:"contract"`
		Fields   []string `json:"fields"`
	}{contract, fields})
}

func cloneSkillRelease(source SkillRelease) SkillRelease {
	source.Registration = cloneSkillRegistration(source.Registration)
	return source
}

func skillReleaseDigest(release SkillRelease) string {
	release.ReleaseDigest = ""
	return skillGovernanceJSONDigest(release)
}

func skillReleaseCatalogDigest(entries []SkillRelease) string {
	return skillGovernanceJSONDigest(struct {
		Contract string         `json:"contract"`
		Releases []SkillRelease `json:"releases"`
	}{"orquesta.tooling.skill-releases.v1", entries})
}

func skillGovernanceJSONDigest(subject any) string {
	encoded, err := json.Marshal(subject)
	if err != nil {
		panic("canonical skill governance cannot fail JSON encoding: " + err.Error())
	}
	return skillContentDigest(encoded)
}
