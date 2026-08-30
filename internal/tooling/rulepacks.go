package tooling

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

const (
	RulePackItemRule     RulePackItemKind = "rule"
	RulePackItemWorkflow RulePackItemKind = "workflow"

	RulePackRefreshCompleteSnapshot RulePackRefreshCoverage = "complete_snapshot"
	rulePackCatalogContract                                 = "orquesta.tooling.rulepacks.v2"

	ErrorRulePackInvalid             = "tooling.rulepack_invalid"
	ErrorRulePackDuplicate           = "tooling.rulepack_duplicate"
	ErrorRulePackUntrusted           = "tooling.rulepack_untrusted"
	ErrorRulePackRefreshPrecondition = "tooling.rulepack_refresh_precondition"
)

type RulePackItemKind string
type RulePackRefreshCoverage string

// RulePackItem points to a versioned rule or workflow contract. The body is an
// artifact outside this descriptor and this package never loads or executes it.
type RulePackItem struct {
	Kind           RulePackItemKind `json:"kind"`
	ID             string           `json:"id"`
	ContractRef    string           `json:"contract_ref"`
	ContractDigest string           `json:"contract_digest"`
}

type RulePackSpec struct {
	ID             string         `json:"id"`
	Version        string         `json:"version"`
	DescriptionKey string         `json:"description_key"`
	Scopes         []SkillScope   `json:"scopes"`
	Precedence     int64          `json:"precedence"`
	Items          []RulePackItem `json:"items"`
}

type RulePackClaim struct {
	ID                     string         `json:"id"`
	Version                string         `json:"version"`
	PackDigest             string         `json:"pack_digest"`
	Origin                 RulePackOrigin `json:"origin"`
	SignerRef              string         `json:"signer_ref"`
	SignatureSubjectDigest string         `json:"signature_subject_digest"`
}

func (claim RulePackClaim) SignaturePayload() []byte {
	if !validRulePackClaimUTF8(claim) || !validSurfaceDigest(claim.SignatureSubjectDigest) {
		return nil
	}
	return []byte("orquesta.tooling.rulepack-signature.v1\x00" + claim.SignatureSubjectDigest)
}

type RulePackCandidate struct {
	Spec      RulePackSpec  `json:"spec"`
	Claim     RulePackClaim `json:"claim"`
	Signature []byte        `json:"-"`
}

type RulePackRevocationClaim struct {
	ID                     string         `json:"id"`
	Version                string         `json:"version"`
	PackDigest             string         `json:"pack_digest"`
	Origin                 RulePackOrigin `json:"origin"`
	RevocationRef          string         `json:"revocation_ref"`
	ReasonCode             string         `json:"reason_code"`
	SignerRef              string         `json:"signer_ref"`
	SignatureSubjectDigest string         `json:"signature_subject_digest"`
}

func (claim RulePackRevocationClaim) SignaturePayload() []byte {
	if !validRulePackRevocationClaimUTF8(claim) || !validSurfaceDigest(claim.SignatureSubjectDigest) {
		return nil
	}
	return []byte("orquesta.tooling.rulepack-revocation-signature.v1\x00" + claim.SignatureSubjectDigest)
}

type RulePackRevocationCandidate struct {
	Claim     RulePackRevocationClaim `json:"claim"`
	Signature []byte                  `json:"-"`
}

// RulePackRevocationAuthority is separate from publication trust. It grants
// only signature verification for revocations issued by one exact origin ref;
// it is neither application authorization nor durable key custody.
type RulePackRevocationAuthority struct {
	OriginRef string `json:"origin_ref"`
	SignerRef string `json:"signer_ref"`
	PublicKey []byte `json:"-"`
}

type RulePackRegistration struct {
	Spec                     RulePackSpec   `json:"spec"`
	Digest                   string         `json:"digest"`
	Claim                    RulePackClaim  `json:"claim"`
	TrustKeyDigest           string         `json:"trust_key_digest"`
	Signature                string         `json:"signature"`
	Revoked                  bool           `json:"revoked"`
	RevocationOrigin         RulePackOrigin `json:"revocation_origin,omitempty"`
	RevocationRef            string         `json:"revocation_ref,omitempty"`
	RevocationReasonCode     string         `json:"revocation_reason_code,omitempty"`
	RevocationSubjectDigest  string         `json:"revocation_subject_digest,omitempty"`
	RevocationSignerRef      string         `json:"revocation_signer_ref,omitempty"`
	RevocationTrustKeyDigest string         `json:"revocation_trust_key_digest,omitempty"`
	RevocationSignature      string         `json:"revocation_signature,omitempty"`
}

type RulePackOrigin struct {
	Ref            string `json:"ref"`
	RevisionDigest string `json:"revision_digest"`
}

type RulePackRefresh struct {
	Coverage              RulePackRefreshCoverage       `json:"coverage"`
	PreviousCatalogDigest string                        `json:"previous_catalog_digest"`
	PreviousOrigin        RulePackOrigin                `json:"previous_origin"`
	NextOrigin            RulePackOrigin                `json:"next_origin"`
	Candidates            []RulePackCandidate           `json:"candidates"`
	Revocations           []RulePackRevocationCandidate `json:"revocations"`
}

type EffectiveRulePackItem struct {
	PackID      string       `json:"pack_id"`
	PackVersion string       `json:"pack_version"`
	PackDigest  string       `json:"pack_digest"`
	Precedence  int64        `json:"precedence"`
	Item        RulePackItem `json:"item"`
}

// RulePackHistoryRecord is the monotonic identity and provenance tombstone
// retained by an in-memory refresh chain. It is evidence carried by the
// snapshot, not durable persistence or a compare-and-swap writer.
type RulePackHistoryRecord struct {
	ID                       string         `json:"id"`
	Version                  string         `json:"version"`
	PackDigest               string         `json:"pack_digest"`
	IntroducedOrigin         RulePackOrigin `json:"introduced_origin"`
	SignerRef                string         `json:"signer_ref"`
	TrustKeyDigest           string         `json:"trust_key_digest"`
	SignatureSubjectDigest   string         `json:"signature_subject_digest"`
	Signature                string         `json:"signature"`
	Revoked                  bool           `json:"revoked"`
	RevocationOrigin         RulePackOrigin `json:"revocation_origin,omitempty"`
	RevocationRef            string         `json:"revocation_ref,omitempty"`
	RevocationReasonCode     string         `json:"revocation_reason_code,omitempty"`
	RevocationSignerRef      string         `json:"revocation_signer_ref,omitempty"`
	RevocationTrustKeyDigest string         `json:"revocation_trust_key_digest,omitempty"`
	RevocationSubjectDigest  string         `json:"revocation_subject_digest,omitempty"`
	RevocationSignature      string         `json:"revocation_signature,omitempty"`
}

// RulePackCatalogSnapshot is the complete, inert state required to rebuild one
// catalog projection. It does not persist itself or authorize its contents.
type RulePackCatalogSnapshot struct {
	Contract        string                  `json:"contract"`
	Origin          RulePackOrigin          `json:"origin"`
	ObservedOrigins []RulePackOrigin        `json:"observed_origins"`
	Packs           []RulePackRegistration  `json:"packs"`
	History         []RulePackHistoryRecord `json:"history"`
	CatalogDigest   string                  `json:"catalog_digest"`
}

// RulePackCatalog is one immutable descriptor projection. It owns neither a
// loader, an execution lifecycle, persistence, nor authorization policy.
type RulePackCatalog struct {
	origin  RulePackOrigin
	origins []RulePackOrigin
	entries []RulePackRegistration
	byKey   map[string]int
	history []RulePackHistoryRecord
	digest  string
}

func NewRulePackClaim(source RulePackSpec, origin RulePackOrigin, signerRef string) (RulePackClaim, error) {
	spec, err := canonicalRulePackSpec(source)
	if err != nil || !validRulePackOrigin(origin) || !validSkillSignerRef(signerRef) || !utf8.ValidString(signerRef) {
		return RulePackClaim{}, contractError(ErrorRulePackInvalid, "claim")
	}
	claim := RulePackClaim{
		ID: spec.ID, Version: spec.Version, PackDigest: rulePackSpecDigest(spec), Origin: origin, SignerRef: signerRef,
	}
	claim.SignatureSubjectDigest = skillFactDigest(
		"orquesta.tooling.rulepack-claim.v1", claim.ID, claim.Version, claim.PackDigest,
		claim.Origin.Ref, claim.Origin.RevisionDigest, claim.SignerRef,
	)
	if !validSurfaceDigest(claim.SignatureSubjectDigest) || !validRulePackClaimUTF8(claim) {
		return RulePackClaim{}, contractError(ErrorRulePackInvalid, "claim")
	}
	return claim, nil
}

func NewRulePackRevocationClaim(
	registration RulePackRegistration,
	origin RulePackOrigin, revocationRef, reasonCode, signerRef string,
) (RulePackRevocationClaim, error) {
	spec, err := canonicalRulePackSpec(registration.Spec)
	expectedRegistrationClaim, claimErr := NewRulePackClaim(spec, registration.Claim.Origin, registration.Claim.SignerRef)
	if err != nil || registration.Digest != rulePackSpecDigest(spec) ||
		claimErr != nil || expectedRegistrationClaim != registration.Claim ||
		!validRulePackOrigin(origin) || origin.Ref != registration.Claim.Origin.Ref ||
		!validSkillEvidenceRef(revocationRef) || !validToolID(reasonCode) ||
		!validSkillSignerRef(signerRef) ||
		!validRulePackStrings(revocationRef, reasonCode, signerRef) {
		return RulePackRevocationClaim{}, contractError(ErrorRulePackInvalid, "revocation")
	}
	claim := RulePackRevocationClaim{
		ID: registration.Spec.ID, Version: registration.Spec.Version, PackDigest: registration.Digest,
		Origin: origin, RevocationRef: revocationRef, ReasonCode: reasonCode, SignerRef: signerRef,
	}
	claim.SignatureSubjectDigest = skillFactDigest(
		"orquesta.tooling.rulepack-revocation-claim.v1", claim.ID, claim.Version, claim.PackDigest,
		claim.Origin.Ref, claim.Origin.RevisionDigest, claim.RevocationRef, claim.ReasonCode, claim.SignerRef,
	)
	if !validSurfaceDigest(claim.SignatureSubjectDigest) || !validRulePackRevocationClaimUTF8(claim) {
		return RulePackRevocationClaim{}, contractError(ErrorRulePackInvalid, "revocation")
	}
	return claim, nil
}

func NewRulePackCatalog(
	origin RulePackOrigin,
	trustRoots []SkillTrustRoot,
	revocationAuthorities []RulePackRevocationAuthority,
	candidates []RulePackCandidate,
	revocations []RulePackRevocationCandidate,
) (*RulePackCatalog, error) {
	if !validRulePackOrigin(origin) || len(candidates) == 0 || !validRulePackTrustRootsUTF8(trustRoots) {
		return nil, contractError(ErrorRulePackInvalid, "catalog")
	}
	trusted, err := canonicalSkillTrustRoots(trustRoots)
	if err != nil {
		return nil, contractError(ErrorRulePackUntrusted, "trust_roots")
	}
	revokers, err := canonicalRulePackRevocationAuthorities(revocationAuthorities)
	if err != nil {
		return nil, err
	}
	entries := make([]RulePackRegistration, 0, len(candidates))
	byKey := make(map[string]int, len(candidates))
	for _, candidate := range candidates {
		spec, err := canonicalRulePackSpec(candidate.Spec)
		if err != nil {
			return nil, err
		}
		expected, claimErr := NewRulePackClaim(spec, origin, candidate.Claim.SignerRef)
		publicKey, signerFound := trusted[candidate.Claim.SignerRef]
		payload := candidate.Claim.SignaturePayload()
		if claimErr != nil || expected != candidate.Claim || !signerFound ||
			len(payload) == 0 || len(candidate.Signature) != ed25519.SignatureSize ||
			!ed25519.Verify(publicKey, payload, candidate.Signature) {
			return nil, contractError(ErrorRulePackUntrusted, "candidate")
		}
		key := registryKey(spec.ID, spec.Version)
		if _, duplicate := byKey[key]; duplicate {
			return nil, contractError(ErrorRulePackDuplicate, "identity")
		}
		byKey[key] = len(entries)
		entries = append(entries, RulePackRegistration{
			Spec: spec, Digest: candidate.Claim.PackDigest, Claim: candidate.Claim,
			TrustKeyDigest: skillContentDigest(publicKey),
			Signature:      base64.RawStdEncoding.EncodeToString(candidate.Signature),
		})
	}
	for _, candidate := range revocations {
		index, found := byKey[registryKey(candidate.Claim.ID, candidate.Claim.Version)]
		if !found || entries[index].Revoked || entries[index].Digest != candidate.Claim.PackDigest {
			return nil, contractError(ErrorRulePackInvalid, "revocation_target")
		}
		expected, claimErr := NewRulePackRevocationClaim(
			entries[index], origin, candidate.Claim.RevocationRef, candidate.Claim.ReasonCode, candidate.Claim.SignerRef,
		)
		publicKey, signerFound := revokers[rulePackRevocationAuthorityKey(candidate.Claim.Origin.Ref, candidate.Claim.SignerRef)]
		payload := candidate.Claim.SignaturePayload()
		if claimErr != nil || expected != candidate.Claim || !signerFound ||
			len(payload) == 0 || len(candidate.Signature) != ed25519.SignatureSize ||
			!ed25519.Verify(publicKey, payload, candidate.Signature) {
			return nil, contractError(ErrorRulePackUntrusted, "revocation")
		}
		entry := &entries[index]
		entry.Revoked, entry.RevocationRef, entry.RevocationReasonCode = true, candidate.Claim.RevocationRef, candidate.Claim.ReasonCode
		entry.RevocationOrigin = candidate.Claim.Origin
		entry.RevocationSubjectDigest, entry.RevocationSignerRef = candidate.Claim.SignatureSubjectDigest, candidate.Claim.SignerRef
		entry.RevocationTrustKeyDigest = skillContentDigest(publicKey)
		entry.RevocationSignature = base64.RawStdEncoding.EncodeToString(candidate.Signature)
	}
	if hasRulePackTrustSubstitution(entries) {
		return nil, contractError(ErrorRulePackUntrusted, "trust_substitution")
	}
	if hasAmbiguousRulePackPrecedence(entries) {
		return nil, contractError(ErrorRulePackDuplicate, "precedence")
	}
	sort.Slice(entries, func(left, right int) bool {
		return surfaceIdentityLess(entries[left].Spec.ID, entries[left].Spec.Version, entries[right].Spec.ID, entries[right].Spec.Version)
	})
	byKey = make(map[string]int, len(entries))
	for index := range entries {
		byKey[registryKey(entries[index].Spec.ID, entries[index].Spec.Version)] = index
	}
	history := make([]RulePackHistoryRecord, len(entries))
	for index := range entries {
		history[index] = rulePackHistoryFromRegistration(entries[index])
	}
	catalog := &RulePackCatalog{
		origin: origin, origins: []RulePackOrigin{origin}, entries: entries, byKey: byKey, history: history,
	}
	catalog.digest = rulePackCatalogDigest(origin, catalog.origins, entries, history)
	if catalog.digest == "" {
		return nil, contractError(ErrorRulePackInvalid, "catalog_digest")
	}
	return catalog, nil
}

// Refresh validates a pure transition from this exact snapshot. It does not
// publish the returned catalog and is not a compare-and-swap operation; a
// caller that stores catalogs must serialize and persist the transition.
func (catalog *RulePackCatalog) Refresh(
	trustRoots []SkillTrustRoot,
	revocationAuthorities []RulePackRevocationAuthority,
	request RulePackRefresh,
) (*RulePackCatalog, error) {
	if catalog == nil || request.PreviousCatalogDigest != catalog.digest || request.PreviousOrigin != catalog.origin {
		return nil, contractError(ErrorRulePackRefreshPrecondition, "previous")
	}
	if request.Coverage != RulePackRefreshCompleteSnapshot ||
		!validSurfaceDigest(request.PreviousCatalogDigest) || !validRulePackOrigin(request.NextOrigin) ||
		request.NextOrigin.Ref != catalog.origin.Ref || rulePackOriginObserved(catalog.origins, request.NextOrigin) {
		return nil, contractError(ErrorRulePackInvalid, "next_origin")
	}
	next, err := NewRulePackCatalog(
		request.NextOrigin, trustRoots, revocationAuthorities, request.Candidates, request.Revocations,
	)
	if err != nil {
		return nil, err
	}
	history, historyErr := mergeRulePackHistory(catalog.history, catalog.byKey, next.entries, next.byKey)
	if historyErr != nil {
		return nil, historyErr
	}
	next.history = history
	next.origins = append(append([]RulePackOrigin(nil), catalog.origins...), request.NextOrigin)
	next.digest = rulePackCatalogDigest(next.origin, next.origins, next.entries, next.history)
	if next.digest == "" {
		return nil, contractError(ErrorRulePackInvalid, "catalog_digest")
	}
	return next, nil
}

func (catalog *RulePackCatalog) Origin() RulePackOrigin {
	if catalog == nil {
		return RulePackOrigin{}
	}
	return catalog.origin
}

func (catalog *RulePackCatalog) Digest() string {
	if catalog == nil {
		return ""
	}
	return catalog.digest
}

func (catalog *RulePackCatalog) List() []RulePackRegistration {
	if catalog == nil {
		return nil
	}
	result := make([]RulePackRegistration, len(catalog.entries))
	for index := range catalog.entries {
		result[index] = cloneRulePackRegistration(catalog.entries[index])
	}
	return result
}

func (catalog *RulePackCatalog) History() []RulePackHistoryRecord {
	if catalog == nil {
		return nil
	}
	return append([]RulePackHistoryRecord(nil), catalog.history...)
}

func (catalog *RulePackCatalog) Snapshot() RulePackCatalogSnapshot {
	if catalog == nil {
		return RulePackCatalogSnapshot{}
	}
	return RulePackCatalogSnapshot{
		Contract: rulePackCatalogContract, Origin: catalog.origin,
		ObservedOrigins: append([]RulePackOrigin(nil), catalog.origins...),
		Packs:           cloneRulePackRegistrations(catalog.entries),
		History:         append([]RulePackHistoryRecord(nil), catalog.history...),
		CatalogDigest:   catalog.digest,
	}
}

// RestoreRulePackCatalog validates and rebuilds one exact snapshot pinned by
// expectedCatalogDigest. It performs no I/O, publication, authorization or CAS.
func RestoreRulePackCatalog(
	source RulePackCatalogSnapshot,
	expectedCatalogDigest string,
	trustRoots []SkillTrustRoot,
	revocationAuthorities []RulePackRevocationAuthority,
) (*RulePackCatalog, error) {
	source = cloneRulePackCatalogSnapshot(source)
	if source.Contract != rulePackCatalogContract || !validSurfaceDigest(expectedCatalogDigest) ||
		source.CatalogDigest != expectedCatalogDigest || !validRulePackOriginChain(source.Origin, source.ObservedOrigins) {
		return nil, contractError(ErrorRulePackInvalid, "snapshot")
	}
	computedDigest := rulePackCatalogDigest(source.Origin, source.ObservedOrigins, source.Packs, source.History)
	if computedDigest == "" || computedDigest != expectedCatalogDigest {
		return nil, contractError(ErrorRulePackInvalid, "snapshot_digest")
	}

	candidates := make([]RulePackCandidate, 0, len(source.Packs))
	revocations := make([]RulePackRevocationCandidate, 0, len(source.Packs))
	for _, registration := range source.Packs {
		signature, err := base64.RawStdEncoding.DecodeString(registration.Signature)
		if err != nil {
			return nil, contractError(ErrorRulePackInvalid, "snapshot_signature")
		}
		candidates = append(candidates, RulePackCandidate{
			Spec: registration.Spec, Claim: registration.Claim, Signature: signature,
		})
		if registration.Revoked {
			revocationSignature, err := base64.RawStdEncoding.DecodeString(registration.RevocationSignature)
			if err != nil {
				return nil, contractError(ErrorRulePackInvalid, "snapshot_revocation_signature")
			}
			revocations = append(revocations, RulePackRevocationCandidate{
				Claim: RulePackRevocationClaim{
					ID: registration.Spec.ID, Version: registration.Spec.Version, PackDigest: registration.Digest,
					Origin: registration.RevocationOrigin, RevocationRef: registration.RevocationRef,
					ReasonCode: registration.RevocationReasonCode, SignerRef: registration.RevocationSignerRef,
					SignatureSubjectDigest: registration.RevocationSubjectDigest,
				},
				Signature: revocationSignature,
			})
		}
	}
	rebuilt, err := NewRulePackCatalog(source.Origin, trustRoots, revocationAuthorities, candidates, revocations)
	if err != nil || !equalRulePackRegistrations(rebuilt.entries, source.Packs) {
		if err != nil {
			return nil, err
		}
		return nil, contractError(ErrorRulePackInvalid, "snapshot_packs")
	}
	trusted, err := canonicalSkillTrustRoots(trustRoots)
	if err != nil {
		return nil, contractError(ErrorRulePackUntrusted, "snapshot_trust_roots")
	}
	revokers, err := canonicalRulePackRevocationAuthorities(revocationAuthorities)
	if err != nil {
		return nil, err
	}
	if err := validateRulePackHistorySignatures(source.History, source.ObservedOrigins, trusted, revokers); err != nil {
		return nil, err
	}
	merged, err := mergeRulePackHistory(source.History, rebuilt.byKey, rebuilt.entries, rebuilt.byKey)
	if err != nil || !equalRulePackHistory(merged, source.History) {
		if err != nil {
			return nil, err
		}
		return nil, contractError(ErrorRulePackInvalid, "snapshot_history")
	}
	rebuilt.origins = append([]RulePackOrigin(nil), source.ObservedOrigins...)
	rebuilt.history = append([]RulePackHistoryRecord(nil), source.History...)
	rebuilt.digest = computedDigest
	return rebuilt, nil
}

// Resolve returns only visible, non-revoked descriptors. Higher precedence
// wins per exact item kind/ID; matching scope does not grant authorization.
func (catalog *RulePackCatalog) Resolve(context SkillScopeContext) ([]EffectiveRulePackItem, error) {
	if catalog == nil || !context.valid {
		return nil, contractError(ErrorSkillScopeInvalid, "context")
	}
	visible := make([]RulePackRegistration, 0, len(catalog.entries))
	for _, entry := range catalog.entries {
		if !entry.Revoked && matchesCuratedScope(entry.Spec.Scopes, context) {
			visible = append(visible, entry)
		}
	}
	sort.Slice(visible, func(left, right int) bool {
		if visible[left].Spec.Precedence != visible[right].Spec.Precedence {
			return visible[left].Spec.Precedence > visible[right].Spec.Precedence
		}
		return surfaceIdentityLess(visible[left].Spec.ID, visible[left].Spec.Version, visible[right].Spec.ID, visible[right].Spec.Version)
	})
	seen := make(map[string]struct{})
	result := make([]EffectiveRulePackItem, 0)
	for _, entry := range visible {
		for _, item := range entry.Spec.Items {
			key := string(item.Kind) + "\x00" + item.ID
			if _, shadowed := seen[key]; shadowed {
				continue
			}
			seen[key] = struct{}{}
			result = append(result, EffectiveRulePackItem{
				PackID: entry.Spec.ID, PackVersion: entry.Spec.Version, PackDigest: entry.Digest,
				Precedence: entry.Spec.Precedence, Item: item,
			})
		}
	}
	return result, nil
}

func canonicalRulePackSpec(source RulePackSpec) (RulePackSpec, error) {
	if !validRulePackSpecUTF8(source) {
		return RulePackSpec{}, contractError(ErrorRulePackInvalid, "utf8")
	}
	if !validToolID(source.ID) || strings.ContainsAny(source.ID, "*?[]") {
		return RulePackSpec{}, contractError(ErrorRulePackInvalid, "id")
	}
	if _, err := parseVersion(source.Version); err != nil {
		return RulePackSpec{}, contractError(ErrorRulePackInvalid, "version")
	}
	if source.DescriptionKey != "rulepack."+source.ID+".description" || !validToolID(source.DescriptionKey) ||
		source.Precedence <= 0 || source.Precedence > 1_000_000 {
		return RulePackSpec{}, contractError(ErrorRulePackInvalid, "metadata")
	}
	scopes, err := canonicalSkillScopes(source.Scopes)
	if err != nil {
		return RulePackSpec{}, contractError(ErrorRulePackInvalid, "scopes")
	}
	items := append([]RulePackItem(nil), source.Items...)
	if len(items) == 0 {
		return RulePackSpec{}, contractError(ErrorRulePackInvalid, "items")
	}
	sort.Slice(items, func(left, right int) bool { return rulePackItemKey(items[left]) < rulePackItemKey(items[right]) })
	for index, item := range items {
		if (item.Kind != RulePackItemRule && item.Kind != RulePackItemWorkflow) || !validToolID(item.ID) ||
			!validSurfaceDigest(item.ContractDigest) || item.ContractRef != "artifact:"+item.ContractDigest ||
			index > 0 && rulePackItemKey(item) == rulePackItemKey(items[index-1]) {
			return RulePackSpec{}, contractError(ErrorRulePackInvalid, "items")
		}
	}
	source.Scopes, source.Items = scopes, items
	return source, nil
}

func hasAmbiguousRulePackPrecedence(entries []RulePackRegistration) bool {
	seen := make(map[string]struct{})
	for _, entry := range entries {
		if entry.Revoked {
			continue
		}
		for _, item := range entry.Spec.Items {
			key := string(item.Kind) + "\x00" + item.ID + "\x00" + strconv.FormatInt(entry.Spec.Precedence, 10)
			if _, duplicate := seen[key]; duplicate {
				return true
			}
			seen[key] = struct{}{}
		}
	}
	return false
}

func hasRulePackTrustSubstitution(entries []RulePackRegistration) bool {
	type publicationTrust struct {
		signerRef string
		keyDigest string
	}
	observed := make(map[string]publicationTrust, len(entries))
	for _, entry := range entries {
		candidate := publicationTrust{entry.Claim.SignerRef, entry.TrustKeyDigest}
		if known, found := observed[entry.Spec.ID]; found && known != candidate {
			return true
		}
		observed[entry.Spec.ID] = candidate
	}
	return false
}

func validRulePackOrigin(origin RulePackOrigin) bool {
	return validRulePackStrings(origin.Ref, origin.RevisionDigest) && validRequiredResourceToken(origin.Ref) &&
		!strings.ContainsAny(origin.Ref, "*?[]") && validSurfaceDigest(origin.RevisionDigest)
}

func validRulePackOriginChain(origin RulePackOrigin, observed []RulePackOrigin) bool {
	if !validRulePackOrigin(origin) || len(observed) == 0 || observed[len(observed)-1] != origin {
		return false
	}
	seen := make(map[RulePackOrigin]struct{}, len(observed))
	for _, candidate := range observed {
		if !validRulePackOrigin(candidate) || candidate.Ref != origin.Ref {
			return false
		}
		if _, duplicate := seen[candidate]; duplicate {
			return false
		}
		seen[candidate] = struct{}{}
	}
	return true
}

func rulePackOriginObserved(origins []RulePackOrigin, candidate RulePackOrigin) bool {
	for _, observed := range origins {
		if observed == candidate {
			return true
		}
	}
	return false
}

func rulePackItemKey(item RulePackItem) string { return string(item.Kind) + "\x00" + item.ID }

func cloneRulePackRegistration(source RulePackRegistration) RulePackRegistration {
	source.Spec.Scopes = append([]SkillScope(nil), source.Spec.Scopes...)
	source.Spec.Items = append([]RulePackItem(nil), source.Spec.Items...)
	return source
}

func cloneRulePackRegistrations(source []RulePackRegistration) []RulePackRegistration {
	result := make([]RulePackRegistration, len(source))
	for index := range source {
		result[index] = cloneRulePackRegistration(source[index])
	}
	return result
}

func cloneRulePackCatalogSnapshot(source RulePackCatalogSnapshot) RulePackCatalogSnapshot {
	source.ObservedOrigins = append([]RulePackOrigin(nil), source.ObservedOrigins...)
	source.Packs = cloneRulePackRegistrations(source.Packs)
	source.History = append([]RulePackHistoryRecord(nil), source.History...)
	return source
}

func equalRulePackRegistrations(left, right []RulePackRegistration) bool {
	if len(left) != len(right) {
		return false
	}
	leftJSON, leftErr := json.Marshal(left)
	rightJSON, rightErr := json.Marshal(right)
	return leftErr == nil && rightErr == nil && string(leftJSON) == string(rightJSON)
}

func equalRulePackHistory(left, right []RulePackHistoryRecord) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func rulePackHistoryFromRegistration(source RulePackRegistration) RulePackHistoryRecord {
	return RulePackHistoryRecord{
		ID: source.Spec.ID, Version: source.Spec.Version, PackDigest: source.Digest,
		IntroducedOrigin: source.Claim.Origin, SignerRef: source.Claim.SignerRef, TrustKeyDigest: source.TrustKeyDigest,
		SignatureSubjectDigest: source.Claim.SignatureSubjectDigest, Signature: source.Signature,
		Revoked: source.Revoked, RevocationOrigin: source.RevocationOrigin,
		RevocationRef: source.RevocationRef, RevocationReasonCode: source.RevocationReasonCode,
		RevocationSignerRef: source.RevocationSignerRef, RevocationTrustKeyDigest: source.RevocationTrustKeyDigest,
		RevocationSubjectDigest: source.RevocationSubjectDigest, RevocationSignature: source.RevocationSignature,
	}
}

func mergeRulePackHistory(
	previous []RulePackHistoryRecord,
	previousByKey map[string]int,
	entries []RulePackRegistration,
	byKey map[string]int,
) ([]RulePackHistoryRecord, error) {
	history := append([]RulePackHistoryRecord(nil), previous...)
	positions := make(map[string]int, len(history))
	maximumVersion := make(map[string]uint64, len(history))
	type publicationTrust struct {
		signerRef string
		keyDigest string
	}
	historicalTrust := make(map[string]publicationTrust, len(history))
	for index, record := range history {
		key := registryKey(record.ID, record.Version)
		if _, duplicate := positions[key]; duplicate {
			return nil, contractError(ErrorRulePackInvalid, "history_duplicate")
		}
		positions[key] = index
		version, err := parseVersion(record.Version)
		if err != nil {
			return nil, contractError(ErrorRulePackInvalid, "history_version")
		}
		if known, found := maximumVersion[record.ID]; !found || version > known {
			maximumVersion[record.ID] = version
		}
		trust := publicationTrust{record.SignerRef, record.TrustKeyDigest}
		if known, found := historicalTrust[record.ID]; found && known != trust {
			return nil, contractError(ErrorRulePackUntrusted, "history_trust_substitution")
		}
		historicalTrust[record.ID] = trust
		if record.Revoked {
			entryIndex, found := byKey[registryKey(record.ID, record.Version)]
			if !found || !entries[entryIndex].Revoked {
				return nil, contractError(ErrorRulePackInvalid, "revocation_dropped")
			}
		}
	}
	for _, entry := range entries {
		candidate := rulePackHistoryFromRegistration(entry)
		key := registryKey(candidate.ID, candidate.Version)
		position, found := positions[key]
		if !found {
			version, err := parseVersion(candidate.Version)
			if err != nil || version < maximumVersion[candidate.ID] {
				return nil, contractError(ErrorRulePackInvalid, "history_downgrade")
			}
			candidateTrust := publicationTrust{candidate.SignerRef, candidate.TrustKeyDigest}
			if known, observed := historicalTrust[candidate.ID]; observed && known != candidateTrust {
				return nil, contractError(ErrorRulePackUntrusted, "history_trust_substitution")
			}
			if version > maximumVersion[candidate.ID] {
				maximumVersion[candidate.ID] = version
			}
			historicalTrust[candidate.ID] = candidateTrust
			positions[key] = len(history)
			history = append(history, candidate)
			continue
		}
		known := history[position]
		if _, previouslyPresent := previousByKey[key]; !previouslyPresent {
			return nil, contractError(ErrorRulePackInvalid, "history_reintroduced")
		}
		if candidate.PackDigest != known.PackDigest || candidate.SignerRef != known.SignerRef ||
			candidate.TrustKeyDigest != known.TrustKeyDigest || candidate.IntroducedOrigin.Ref != known.IntroducedOrigin.Ref {
			return nil, contractError(ErrorRulePackInvalid, "history_rewritten")
		}
		if known.Revoked {
			if !candidate.Revoked || candidate.RevocationRef != known.RevocationRef ||
				candidate.RevocationReasonCode != known.RevocationReasonCode ||
				candidate.RevocationSignerRef != known.RevocationSignerRef ||
				candidate.RevocationTrustKeyDigest != known.RevocationTrustKeyDigest ||
				candidate.RevocationOrigin.Ref != known.RevocationOrigin.Ref {
				return nil, contractError(ErrorRulePackInvalid, "revocation_rewritten")
			}
			continue
		}
		if candidate.Revoked {
			candidate.IntroducedOrigin = known.IntroducedOrigin
			candidate.SignatureSubjectDigest = known.SignatureSubjectDigest
			candidate.Signature = known.Signature
			history[position] = candidate
		}
	}
	sort.Slice(history, func(left, right int) bool {
		return surfaceIdentityLess(history[left].ID, history[left].Version, history[right].ID, history[right].Version)
	})
	return history, nil
}

func validateRulePackHistorySignatures(
	history []RulePackHistoryRecord,
	origins []RulePackOrigin,
	trusted map[string]ed25519.PublicKey,
	revokers map[string]ed25519.PublicKey,
) error {
	identities := make(map[string]struct{}, len(history))
	type publicationTrust struct {
		signerRef string
		keyDigest string
	}
	historicalTrust := make(map[string]publicationTrust, len(history))
	for _, record := range history {
		key := registryKey(record.ID, record.Version)
		if _, duplicate := identities[key]; duplicate {
			return contractError(ErrorRulePackInvalid, "history_duplicate")
		}
		identities[key] = struct{}{}
		if !validToolID(record.ID) || strings.ContainsAny(record.ID, "*?[]") ||
			!validSurfaceDigest(record.PackDigest) || !validRulePackOrigin(record.IntroducedOrigin) ||
			!rulePackOriginObserved(origins, record.IntroducedOrigin) || !validSkillSignerRef(record.SignerRef) ||
			!validSurfaceDigest(record.TrustKeyDigest) {
			return contractError(ErrorRulePackInvalid, "history_publication")
		}
		if _, err := parseVersion(record.Version); err != nil {
			return contractError(ErrorRulePackInvalid, "history_version")
		}
		trust := publicationTrust{record.SignerRef, record.TrustKeyDigest}
		if known, found := historicalTrust[record.ID]; found && known != trust {
			return contractError(ErrorRulePackUntrusted, "history_trust_substitution")
		}
		historicalTrust[record.ID] = trust
		claim := RulePackClaim{
			ID: record.ID, Version: record.Version, PackDigest: record.PackDigest,
			Origin: record.IntroducedOrigin, SignerRef: record.SignerRef,
		}
		claim.SignatureSubjectDigest = skillFactDigest(
			"orquesta.tooling.rulepack-claim.v1", claim.ID, claim.Version, claim.PackDigest,
			claim.Origin.Ref, claim.Origin.RevisionDigest, claim.SignerRef,
		)
		publicKey, signerFound := trusted[record.SignerRef]
		signature, signatureErr := base64.RawStdEncoding.DecodeString(record.Signature)
		if record.SignatureSubjectDigest != claim.SignatureSubjectDigest || !signerFound ||
			record.TrustKeyDigest != skillContentDigest(publicKey) || signatureErr != nil ||
			len(signature) != ed25519.SignatureSize ||
			base64.RawStdEncoding.EncodeToString(signature) != record.Signature ||
			!ed25519.Verify(publicKey, claim.SignaturePayload(), signature) {
			return contractError(ErrorRulePackUntrusted, "history_publication")
		}
		if !record.Revoked {
			if record.RevocationOrigin != (RulePackOrigin{}) || record.RevocationRef != "" ||
				record.RevocationReasonCode != "" || record.RevocationSignerRef != "" ||
				record.RevocationTrustKeyDigest != "" || record.RevocationSubjectDigest != "" ||
				record.RevocationSignature != "" {
				return contractError(ErrorRulePackInvalid, "history_revocation")
			}
			continue
		}
		revocation := RulePackRevocationClaim{
			ID: record.ID, Version: record.Version, PackDigest: record.PackDigest,
			Origin: record.RevocationOrigin, RevocationRef: record.RevocationRef,
			ReasonCode: record.RevocationReasonCode, SignerRef: record.RevocationSignerRef,
		}
		revocation.SignatureSubjectDigest = skillFactDigest(
			"orquesta.tooling.rulepack-revocation-claim.v1", revocation.ID, revocation.Version,
			revocation.PackDigest, revocation.Origin.Ref, revocation.Origin.RevisionDigest,
			revocation.RevocationRef, revocation.ReasonCode, revocation.SignerRef,
		)
		revocationKey, revokerFound := revokers[rulePackRevocationAuthorityKey(
			record.RevocationOrigin.Ref, record.RevocationSignerRef,
		)]
		revocationSignature, signatureErr := base64.RawStdEncoding.DecodeString(record.RevocationSignature)
		if !validRulePackOrigin(record.RevocationOrigin) || !rulePackOriginObserved(origins, record.RevocationOrigin) ||
			!validSkillEvidenceRef(record.RevocationRef) || !validToolID(record.RevocationReasonCode) ||
			!validSkillSignerRef(record.RevocationSignerRef) || !validSurfaceDigest(record.RevocationTrustKeyDigest) ||
			record.RevocationSubjectDigest != revocation.SignatureSubjectDigest || !revokerFound ||
			record.RevocationTrustKeyDigest != skillContentDigest(revocationKey) || signatureErr != nil ||
			len(revocationSignature) != ed25519.SignatureSize ||
			base64.RawStdEncoding.EncodeToString(revocationSignature) != record.RevocationSignature ||
			!ed25519.Verify(revocationKey, revocation.SignaturePayload(), revocationSignature) {
			return contractError(ErrorRulePackUntrusted, "history_revocation")
		}
	}
	return nil
}

func rulePackSpecDigest(spec RulePackSpec) string {
	if !validRulePackSpecUTF8(spec) {
		return ""
	}
	encoded, err := json.Marshal(spec)
	if err != nil {
		return ""
	}
	digest := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(digest[:])
}

func rulePackCatalogDigest(
	origin RulePackOrigin,
	origins []RulePackOrigin,
	entries []RulePackRegistration,
	history []RulePackHistoryRecord,
) string {
	if !validRulePackOrigin(origin) || !validRulePackCatalogUTF8(origins, entries, history) {
		return ""
	}
	encoded, err := json.Marshal(struct {
		Contract string                  `json:"contract"`
		Origin   RulePackOrigin          `json:"origin"`
		Origins  []RulePackOrigin        `json:"observed_origins"`
		Packs    []RulePackRegistration  `json:"packs"`
		History  []RulePackHistoryRecord `json:"history"`
	}{rulePackCatalogContract, origin, origins, entries, history})
	if err != nil {
		return ""
	}
	digest := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(digest[:])
}

func canonicalRulePackRevocationAuthorities(
	source []RulePackRevocationAuthority,
) (map[string]ed25519.PublicKey, error) {
	result := make(map[string]ed25519.PublicKey, len(source))
	for _, authority := range source {
		if !validRequiredResourceToken(authority.OriginRef) || strings.ContainsAny(authority.OriginRef, "*?[]") ||
			!validSkillSignerRef(authority.SignerRef) ||
			!validRulePackStrings(authority.OriginRef, authority.SignerRef) ||
			len(authority.PublicKey) != ed25519.PublicKeySize {
			return nil, contractError(ErrorRulePackInvalid, "revocation_authority")
		}
		key := rulePackRevocationAuthorityKey(authority.OriginRef, authority.SignerRef)
		if _, duplicate := result[key]; duplicate {
			return nil, contractError(ErrorRulePackDuplicate, "revocation_authority")
		}
		result[key] = append(ed25519.PublicKey(nil), authority.PublicKey...)
	}
	return result, nil
}

func rulePackRevocationAuthorityKey(originRef, signerRef string) string {
	return originRef + "\x00" + signerRef
}

func validRulePackTrustRootsUTF8(roots []SkillTrustRoot) bool {
	for _, root := range roots {
		if !utf8.ValidString(root.SignerRef) {
			return false
		}
	}
	return true
}

func validRulePackSpecUTF8(spec RulePackSpec) bool {
	if !validRulePackStrings(spec.ID, spec.Version, spec.DescriptionKey) {
		return false
	}
	for _, scope := range spec.Scopes {
		if !validRulePackStrings(
			string(scope.Kind), scope.ProductID, scope.ProjectRef, string(scope.Role), scope.GoalRef,
		) {
			return false
		}
	}
	for _, item := range spec.Items {
		if !validRulePackStrings(string(item.Kind), item.ID, item.ContractRef, item.ContractDigest) {
			return false
		}
	}
	return true
}

func validRulePackClaimUTF8(claim RulePackClaim) bool {
	return validRulePackOrigin(claim.Origin) && validRulePackStrings(
		claim.ID, claim.Version, claim.PackDigest, claim.SignerRef, claim.SignatureSubjectDigest,
	)
}

func validRulePackRevocationClaimUTF8(claim RulePackRevocationClaim) bool {
	return validRulePackOrigin(claim.Origin) && validRulePackStrings(
		claim.ID, claim.Version, claim.PackDigest, claim.RevocationRef, claim.ReasonCode,
		claim.SignerRef, claim.SignatureSubjectDigest,
	)
}

func validRulePackCatalogUTF8(
	origins []RulePackOrigin,
	entries []RulePackRegistration,
	history []RulePackHistoryRecord,
) bool {
	for _, origin := range origins {
		if !validRulePackOrigin(origin) {
			return false
		}
	}
	for _, entry := range entries {
		if !validRulePackSpecUTF8(entry.Spec) || !validRulePackClaimUTF8(entry.Claim) ||
			!validRulePackStrings(
				entry.Digest, entry.TrustKeyDigest, entry.Signature, string(entry.RevocationOrigin.Ref),
				entry.RevocationOrigin.RevisionDigest, entry.RevocationRef, entry.RevocationReasonCode,
				entry.RevocationSubjectDigest, entry.RevocationSignerRef, entry.RevocationTrustKeyDigest,
				entry.RevocationSignature,
			) {
			return false
		}
	}
	for _, record := range history {
		if !validRulePackStrings(
			record.ID, record.Version, record.PackDigest, record.IntroducedOrigin.Ref,
			record.IntroducedOrigin.RevisionDigest, record.SignerRef, record.TrustKeyDigest,
			record.SignatureSubjectDigest, record.Signature, record.RevocationOrigin.Ref,
			record.RevocationOrigin.RevisionDigest, record.RevocationRef, record.RevocationReasonCode,
			record.RevocationSignerRef, record.RevocationTrustKeyDigest, record.RevocationSubjectDigest,
			record.RevocationSignature,
		) {
			return false
		}
	}
	return true
}

func validRulePackStrings(values ...string) bool {
	for _, value := range values {
		if !utf8.ValidString(value) {
			return false
		}
	}
	return true
}
