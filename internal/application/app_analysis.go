package application

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"mime"
	"sort"
	"strconv"
	"strings"

	"orquesta/internal/goal"
	"orquesta/internal/identity"
	"orquesta/internal/intake"
)

const (
	AppAnalysisManifestSchema = "orquesta.app-analysis.manifest.v1"
	AppAnalysisEnvelopeSchema = "orquesta.app-analysis.envelope.v1"
	AppAnalysisSnapshotSchema = "orquesta.app-analysis.snapshot.v1"
)

var ErrAppAnalysisInvalid = errors.New("application.app_analysis_invalid")

type AppAnalysisRef string
type AppAnalysisAttachmentKind string

const (
	AppAnalysisAttachmentBehavior AppAnalysisAttachmentKind = "behavior"
	AppAnalysisAttachmentCodebase AppAnalysisAttachmentKind = "codebase"
)

type AppAnalysisPurpose string

const (
	AppAnalysisPurposeCleanRoomReimplementation AppAnalysisPurpose = "clean_room_reimplementation"
	AppAnalysisPurposeRefactorExisting          AppAnalysisPurpose = "refactor_existing"
)

type AppAnalysisClassification string

const AppAnalysisClassificationShareableRedacted AppAnalysisClassification = "shareable_redacted"

type AppAnalysisReviewDecision string

const AppAnalysisReviewAccepted AppAnalysisReviewDecision = "accepted"

// AppAnalysisVerifiedIntake is returned by the project-scoped intake port.
type AppAnalysisVerifiedIntake struct {
	ReceiptRef     string
	ActorRef       goal.ActorRef
	ProjectRef     goal.ProjectRef
	IntakeRef      intake.Ref
	Revision       intake.Revision
	SnapshotDigest string
}

// AppAnalysisVerifiedReview is produced only by AppAnalysisReviewVerifier.
type AppAnalysisVerifiedReview struct {
	ReceiptRef      string
	ActorRef        goal.ActorRef
	ProjectRef      goal.ProjectRef
	SubjectRef      string
	ProviderRef     string
	ProducerRef     string
	ProducerVersion string
	ReviewerRef     identity.PrincipalRef
	Decision        AppAnalysisReviewDecision
	PolicyRef       string
	SubjectDigest   string
}

// AppAnalysisAttachment begins as a neutral descriptor. Resolver-owned
// provenance fields must be empty at ingress and are populated before storage.
type AppAnalysisAttachment struct {
	Kind                    AppAnalysisAttachmentKind `json:"kind"`
	ArtifactRef             goal.ArtifactRef          `json:"-"`
	Digest                  string                    `json:"digest"`
	Size                    int64                     `json:"size"`
	MediaType               string                    `json:"media_type"`
	Classification          AppAnalysisClassification `json:"classification"`
	ProvenanceReceiptRef    string                    `json:"provenance_receipt_ref,omitempty"`
	ProvenanceReceiptDigest string                    `json:"provenance_receipt_digest,omitempty"`
}

type AppAnalysisManifest struct {
	Schema           string                  `json:"schema"`
	SubjectRef       string                  `json:"subject_ref"`
	ProviderRef      string                  `json:"provider_ref"`
	ProducerRef      string                  `json:"producer_ref"`
	ProducerVersion  string                  `json:"producer_version"`
	IntakeRef        intake.Ref              `json:"intake_ref"`
	ExpectedRevision intake.Revision         `json:"expected_revision"`
	Purpose          AppAnalysisPurpose      `json:"purpose"`
	RepositoryRef    identity.RepositoryRef  `json:"-"`
	TreeOID          string                  `json:"tree_oid,omitempty"`
	ReviewReceiptRef string                  `json:"review_receipt_ref"`
	Attachments      []AppAnalysisAttachment `json:"attachments"`
}

type AppAnalysisEnvelope struct {
	Schema               string
	RequestRef           string
	ActorRef             goal.ActorRef
	ProjectRef           goal.ProjectRef
	Manifest             AppAnalysisManifest
	AuthorizationReceipt identity.AuthorizationReceipt
}

// AppAnalysisSnapshot is a public durable DTO. Snapshot returns detached data;
// RestoreAppAnalysis validates every digest and causal binding.
type AppAnalysisSnapshot struct {
	Schema               string
	Ref                  AppAnalysisRef
	Digest               string
	Manifest             AppAnalysisManifest
	Intake               AppAnalysisVerifiedIntake
	Provenance           []AppAnalysisArtifactProvenanceReceipt
	Review               AppAnalysisVerifiedReview
	VerifiedIntakeDigest string
	VerifiedReviewDigest string
}

type AppAnalysis struct {
	ref                  AppAnalysisRef
	digest               string
	manifest             AppAnalysisManifest
	intake               AppAnalysisVerifiedIntake
	provenance           []AppAnalysisArtifactProvenanceReceipt
	review               AppAnalysisVerifiedReview
	verifiedIntakeDigest string
	verifiedReviewDigest string
}

func (analysis AppAnalysis) Ref() AppAnalysisRef { return analysis.ref }
func (analysis AppAnalysis) Digest() string      { return analysis.digest }
func (analysis AppAnalysis) Manifest() AppAnalysisManifest {
	return cloneAppAnalysisManifest(analysis.manifest)
}
func (analysis AppAnalysis) Intake() AppAnalysisVerifiedIntake { return analysis.intake }
func (analysis AppAnalysis) Provenance() []AppAnalysisArtifactProvenanceReceipt {
	return append([]AppAnalysisArtifactProvenanceReceipt(nil), analysis.provenance...)
}
func (analysis AppAnalysis) Review() AppAnalysisVerifiedReview { return analysis.review }
func (analysis AppAnalysis) VerifiedIntakeDigest() string      { return analysis.verifiedIntakeDigest }
func (analysis AppAnalysis) VerifiedReviewDigest() string      { return analysis.verifiedReviewDigest }
func (analysis AppAnalysis) Snapshot() AppAnalysisSnapshot {
	return AppAnalysisSnapshot{
		Schema: AppAnalysisSnapshotSchema, Ref: analysis.ref, Digest: analysis.digest,
		Manifest: analysis.Manifest(), Intake: analysis.intake,
		Provenance: analysis.Provenance(), Review: analysis.review,
		VerifiedIntakeDigest: analysis.verifiedIntakeDigest,
		VerifiedReviewDigest: analysis.verifiedReviewDigest,
	}
}

func RestoreAppAnalysis(snapshot AppAnalysisSnapshot) (AppAnalysis, error) {
	if snapshot.Schema != AppAnalysisSnapshotSchema {
		return AppAnalysis{}, invalidAppAnalysis("snapshot.schema", nil)
	}
	analysis, err := buildAppAnalysis(
		snapshot.Manifest,
		snapshot.Intake,
		append([]AppAnalysisArtifactProvenanceReceipt(nil), snapshot.Provenance...),
		snapshot.Review,
	)
	if err != nil {
		return AppAnalysis{}, err
	}
	if snapshot.Ref != analysis.Ref() ||
		snapshot.Digest != analysis.Digest() ||
		snapshot.VerifiedIntakeDigest != analysis.VerifiedIntakeDigest() ||
		snapshot.VerifiedReviewDigest != analysis.VerifiedReviewDigest() {
		return AppAnalysis{}, invalidAppAnalysis("snapshot.integrity", nil)
	}
	return analysis, nil
}

func AppAnalysisManifestDigest(manifest AppAnalysisManifest) (string, error) {
	canonical, err := canonicalAppAnalysisManifest(manifest)
	if err != nil {
		return "", err
	}
	digest := fingerprintDigest("orquesta.app-analysis.manifest-digest.v1")
	writeFingerprintField(digest, canonical.Schema)
	writeFingerprintField(digest, canonical.SubjectRef)
	writeFingerprintField(digest, canonical.ProviderRef)
	writeFingerprintField(digest, canonical.ProducerRef)
	writeFingerprintField(digest, canonical.ProducerVersion)
	writeFingerprintField(digest, string(canonical.IntakeRef))
	writeFingerprintField(digest, strconv.FormatUint(uint64(canonical.ExpectedRevision), 10))
	writeFingerprintField(digest, string(canonical.Purpose))
	writeFingerprintField(digest, canonical.RepositoryRef.String())
	writeFingerprintField(digest, canonical.TreeOID)
	writeFingerprintField(digest, canonical.ReviewReceiptRef)
	writeFingerprintField(digest, strconv.Itoa(len(canonical.Attachments)))
	for _, attachment := range canonical.Attachments {
		writeAppAnalysisAttachmentFingerprint(digest, attachment, true)
	}
	return fingerprintHex(digest), nil
}

func AppAnalysisAttachmentDescriptorDigest(
	attachment AppAnalysisAttachment,
) (string, error) {
	if attachment.ProvenanceReceiptRef != "" ||
		attachment.ProvenanceReceiptDigest != "" {
		return "", invalidAppAnalysis("descriptor.provenance", nil)
	}
	canonical, err := canonicalAppAnalysisAttachment(attachment)
	if err != nil {
		return "", err
	}
	digest := fingerprintDigest("orquesta.app-analysis.attachment-descriptor.v1")
	writeAppAnalysisAttachmentFingerprint(digest, canonical, false)
	return fingerprintHex(digest), nil
}

func AppAnalysisArtifactProvenanceReceiptDigest(
	receipt AppAnalysisArtifactProvenanceReceipt,
) (string, error) {
	if !validAppAnalysisOpaque(receipt.ReceiptRef, 512) ||
		receipt.ActorRef.String() == "" ||
		receipt.ProjectRef.String() == "" ||
		!validAppAnalysisOpaque(receipt.SubjectRef, 512) ||
		!validAppAnalysisOpaque(receipt.ProviderRef, 512) ||
		!validAppAnalysisOpaque(receipt.ProducerRef, 512) ||
		!validAppAnalysisOpaque(receipt.ProducerVersion, 128) ||
		!validAppAnalysisDigest(receipt.ManifestDigest) ||
		receipt.ArtifactRef.String() == "" ||
		!validAppAnalysisDigest(receipt.DescriptorDigest) {
		return "", invalidAppAnalysis("provenance", nil)
	}
	return fingerprintFields(
		"orquesta.app-analysis.artifact-provenance.v1",
		receipt.ReceiptRef,
		receipt.ActorRef.String(),
		receipt.ProjectRef.String(),
		receipt.SubjectRef,
		receipt.ProviderRef,
		receipt.ProducerRef,
		receipt.ProducerVersion,
		receipt.ManifestDigest,
		receipt.ArtifactRef.String(),
		receipt.DescriptorDigest,
	), nil
}

func AppAnalysisVerifiedReviewDigest(
	review AppAnalysisVerifiedReview,
) (string, error) {
	if !validAppAnalysisOpaque(review.ReceiptRef, 512) ||
		review.ActorRef.String() == "" ||
		review.ProjectRef.String() == "" ||
		!validAppAnalysisOpaque(review.SubjectRef, 512) ||
		!validAppAnalysisOpaque(review.ProviderRef, 512) ||
		!validAppAnalysisOpaque(review.ProducerRef, 512) ||
		!validAppAnalysisOpaque(review.ProducerVersion, 128) ||
		review.ReviewerRef.String() == "" ||
		review.Decision != AppAnalysisReviewAccepted ||
		!validAppAnalysisOpaque(review.PolicyRef, 512) ||
		!validAppAnalysisDigest(review.SubjectDigest) {
		return "", invalidAppAnalysis("review", nil)
	}
	return fingerprintFields(
		"orquesta.app-analysis.verified-review.v1",
		review.ReceiptRef,
		review.ActorRef.String(),
		review.ProjectRef.String(),
		review.SubjectRef,
		review.ProviderRef,
		review.ProducerRef,
		review.ProducerVersion,
		review.ReviewerRef.String(),
		string(review.Decision),
		review.PolicyRef,
		review.SubjectDigest,
	), nil
}

func AppAnalysisVerifiedIntakeDigest(
	evidence AppAnalysisVerifiedIntake,
) (string, error) {
	if !validAppAnalysisOpaque(evidence.ReceiptRef, 512) ||
		evidence.ActorRef.String() == "" ||
		evidence.ProjectRef.String() == "" ||
		!validIntakeStateRef(evidence.IntakeRef) ||
		evidence.Revision == 0 ||
		!validAppAnalysisDigest(evidence.SnapshotDigest) {
		return "", invalidAppAnalysis("intake", nil)
	}
	return fingerprintFields(
		"orquesta.app-analysis.verified-intake.v1",
		evidence.ReceiptRef,
		evidence.ActorRef.String(),
		evidence.ProjectRef.String(),
		string(evidence.IntakeRef),
		strconv.FormatUint(uint64(evidence.Revision), 10),
		evidence.SnapshotDigest,
	), nil
}

func buildAppAnalysis(
	manifest AppAnalysisManifest,
	intakeEvidence AppAnalysisVerifiedIntake,
	provenance []AppAnalysisArtifactProvenanceReceipt,
	review AppAnalysisVerifiedReview,
) (AppAnalysis, error) {
	canonical, err := canonicalAppAnalysisPersistedManifest(manifest)
	if err != nil {
		return AppAnalysis{}, err
	}
	subjectManifest := appAnalysisSubjectManifest(canonical)
	subjectDigest, err := AppAnalysisManifestDigest(subjectManifest)
	if err != nil {
		return AppAnalysis{}, err
	}
	if err := validateAppAnalysisVerifiedIntake(subjectManifest, intakeEvidence); err != nil {
		return AppAnalysis{}, err
	}
	verifiedIntakeDigest, err := AppAnalysisVerifiedIntakeDigest(intakeEvidence)
	if err != nil {
		return AppAnalysis{}, err
	}
	if err := validateAppAnalysisVerifiedReview(subjectManifest, subjectDigest, review); err != nil {
		return AppAnalysis{}, err
	}
	if review.ActorRef != intakeEvidence.ActorRef ||
		review.ProjectRef != intakeEvidence.ProjectRef {
		return AppAnalysis{}, invalidAppAnalysis("scope", nil)
	}
	verifiedReviewDigest, err := AppAnalysisVerifiedReviewDigest(review)
	if err != nil {
		return AppAnalysis{}, err
	}
	canonicalProvenance, err := validateAppAnalysisProvenance(
		canonical,
		subjectDigest,
		intakeEvidence,
		provenance,
	)
	if err != nil {
		return AppAnalysis{}, err
	}
	digest, err := AppAnalysisManifestDigest(canonical)
	if err != nil {
		return AppAnalysis{}, err
	}
	return AppAnalysis{
		ref: AppAnalysisRef("app-analysis:" + digest), digest: digest,
		manifest: canonical, intake: intakeEvidence,
		provenance: canonicalProvenance, review: review,
		verifiedIntakeDigest: verifiedIntakeDigest,
		verifiedReviewDigest: verifiedReviewDigest,
	}, nil
}

func canonicalAppAnalysisIngressManifest(
	manifest AppAnalysisManifest,
) (AppAnalysisManifest, error) {
	canonical, err := canonicalAppAnalysisManifest(manifest)
	if err != nil {
		return AppAnalysisManifest{}, err
	}
	for _, attachment := range canonical.Attachments {
		if attachment.ProvenanceReceiptRef != "" ||
			attachment.ProvenanceReceiptDigest != "" {
			return AppAnalysisManifest{}, invalidAppAnalysis("manifest.provenance", nil)
		}
	}
	return canonical, nil
}

func canonicalAppAnalysisPersistedManifest(
	manifest AppAnalysisManifest,
) (AppAnalysisManifest, error) {
	canonical, err := canonicalAppAnalysisManifest(manifest)
	if err != nil {
		return AppAnalysisManifest{}, err
	}
	for _, attachment := range canonical.Attachments {
		if attachment.ProvenanceReceiptRef == "" ||
			!validAppAnalysisDigest(attachment.ProvenanceReceiptDigest) {
			return AppAnalysisManifest{}, invalidAppAnalysis("manifest.provenance", nil)
		}
	}
	return canonical, nil
}

func canonicalAppAnalysisManifest(manifest AppAnalysisManifest) (AppAnalysisManifest, error) {
	if manifest.Schema != AppAnalysisManifestSchema {
		return AppAnalysisManifest{}, invalidAppAnalysis("manifest.schema", nil)
	}
	if !validAppAnalysisOpaque(manifest.SubjectRef, 512) ||
		!validAppAnalysisOpaque(manifest.ProviderRef, 512) ||
		!validAppAnalysisOpaque(manifest.ProducerRef, 512) ||
		!validAppAnalysisOpaque(manifest.ProducerVersion, 128) ||
		!validIntakeStateRef(manifest.IntakeRef) ||
		manifest.ExpectedRevision == 0 ||
		!validAppAnalysisOpaque(manifest.ReviewReceiptRef, 512) {
		return AppAnalysisManifest{}, invalidAppAnalysis("manifest.identity", nil)
	}
	switch manifest.Purpose {
	case AppAnalysisPurposeRefactorExisting:
		if manifest.RepositoryRef.String() == "" || !validAppAnalysisTreeOID(manifest.TreeOID) {
			return AppAnalysisManifest{}, invalidAppAnalysis("manifest.refactor_source", nil)
		}
	case AppAnalysisPurposeCleanRoomReimplementation:
		if manifest.RepositoryRef.String() != "" || manifest.TreeOID != "" {
			return AppAnalysisManifest{}, invalidAppAnalysis("manifest.clean_room_source", nil)
		}
	default:
		return AppAnalysisManifest{}, invalidAppAnalysis("manifest.purpose", nil)
	}
	if len(manifest.Attachments) == 0 || len(manifest.Attachments) > 32 {
		return AppAnalysisManifest{}, invalidAppAnalysis("manifest.attachments", nil)
	}
	canonical := cloneAppAnalysisManifest(manifest)
	seen := make(map[string]struct{}, len(canonical.Attachments))
	for index := range canonical.Attachments {
		attachment, err := canonicalAppAnalysisAttachment(canonical.Attachments[index])
		if err != nil {
			return AppAnalysisManifest{}, invalidAppAnalysis(
				"manifest.attachments["+strconv.Itoa(index)+"]",
				err,
			)
		}
		canonical.Attachments[index] = attachment
		key := attachment.ArtifactRef.String()
		if _, exists := seen[key]; exists {
			return AppAnalysisManifest{}, invalidAppAnalysis(
				"manifest.attachments["+strconv.Itoa(index)+"].artifact_ref",
				nil,
			)
		}
		seen[key] = struct{}{}
	}
	sort.Slice(canonical.Attachments, func(left, right int) bool {
		return appAnalysisAttachmentSortKey(canonical.Attachments[left]) <
			appAnalysisAttachmentSortKey(canonical.Attachments[right])
	})
	return canonical, nil
}

func canonicalAppAnalysisAttachment(
	attachment AppAnalysisAttachment,
) (AppAnalysisAttachment, error) {
	if attachment.Kind != AppAnalysisAttachmentBehavior &&
		attachment.Kind != AppAnalysisAttachmentCodebase {
		return AppAnalysisAttachment{}, invalidAppAnalysis("kind", nil)
	}
	if attachment.ArtifactRef.String() == "" ||
		!validAppAnalysisDigest(attachment.Digest) ||
		attachment.ArtifactRef.String() != "artifact:sha256:"+attachment.Digest ||
		attachment.Size <= 0 {
		return AppAnalysisAttachment{}, invalidAppAnalysis("artifact", nil)
	}
	mediaType, err := normalizeAppAnalysisMediaType(attachment.MediaType)
	if err != nil {
		return AppAnalysisAttachment{}, invalidAppAnalysis("media_type", err)
	}
	attachment.MediaType = mediaType
	if attachment.Classification != AppAnalysisClassificationShareableRedacted {
		return AppAnalysisAttachment{}, invalidAppAnalysis("classification", nil)
	}
	if (attachment.ProvenanceReceiptRef == "") !=
		(attachment.ProvenanceReceiptDigest == "") ||
		(attachment.ProvenanceReceiptRef != "" &&
			(!validAppAnalysisOpaque(attachment.ProvenanceReceiptRef, 512) ||
				!validAppAnalysisDigest(attachment.ProvenanceReceiptDigest))) {
		return AppAnalysisAttachment{}, invalidAppAnalysis("provenance", nil)
	}
	return attachment, nil
}

func validateAppAnalysisVerifiedIntake(
	manifest AppAnalysisManifest,
	evidence AppAnalysisVerifiedIntake,
) error {
	if !validAppAnalysisOpaque(evidence.ReceiptRef, 512) ||
		evidence.ActorRef.String() == "" ||
		evidence.ProjectRef.String() == "" ||
		evidence.IntakeRef != manifest.IntakeRef ||
		evidence.Revision != manifest.ExpectedRevision ||
		!validAppAnalysisDigest(evidence.SnapshotDigest) {
		return invalidAppAnalysis("intake", nil)
	}
	return nil
}

func validateAppAnalysisVerifiedReview(
	manifest AppAnalysisManifest,
	subjectDigest string,
	review AppAnalysisVerifiedReview,
) error {
	if review.ReceiptRef != manifest.ReviewReceiptRef ||
		review.SubjectRef != manifest.SubjectRef ||
		review.ProviderRef != manifest.ProviderRef ||
		review.ProducerRef != manifest.ProducerRef ||
		review.ProducerVersion != manifest.ProducerVersion ||
		review.SubjectDigest != subjectDigest {
		return invalidAppAnalysis("review", nil)
	}
	_, err := AppAnalysisVerifiedReviewDigest(review)
	return err
}

func validateAppAnalysisProvenance(
	manifest AppAnalysisManifest,
	subjectDigest string,
	intakeEvidence AppAnalysisVerifiedIntake,
	provenance []AppAnalysisArtifactProvenanceReceipt,
) ([]AppAnalysisArtifactProvenanceReceipt, error) {
	if len(provenance) != len(manifest.Attachments) {
		return nil, invalidAppAnalysis("provenance.count", nil)
	}
	result := append([]AppAnalysisArtifactProvenanceReceipt(nil), provenance...)
	for index, attachment := range manifest.Attachments {
		receipt := result[index]
		digest, err := AppAnalysisArtifactProvenanceReceiptDigest(receipt)
		if err != nil ||
			receipt.ActorRef != intakeEvidence.ActorRef ||
			receipt.ProjectRef != intakeEvidence.ProjectRef ||
			receipt.SubjectRef != manifest.SubjectRef ||
			receipt.ProviderRef != manifest.ProviderRef ||
			receipt.ProducerRef != manifest.ProducerRef ||
			receipt.ProducerVersion != manifest.ProducerVersion ||
			receipt.ManifestDigest != subjectDigest ||
			receipt.ArtifactRef != attachment.ArtifactRef ||
			attachment.ProvenanceReceiptRef != receipt.ReceiptRef ||
			attachment.ProvenanceReceiptDigest != digest {
			return nil, invalidAppAnalysis(
				"provenance["+strconv.Itoa(index)+"]",
				err,
			)
		}
		descriptor := attachment
		descriptor.ProvenanceReceiptRef = ""
		descriptor.ProvenanceReceiptDigest = ""
		descriptorDigest, descriptorErr := AppAnalysisAttachmentDescriptorDigest(descriptor)
		if descriptorErr != nil || receipt.DescriptorDigest != descriptorDigest {
			return nil, invalidAppAnalysis(
				"provenance["+strconv.Itoa(index)+"].descriptor",
				descriptorErr,
			)
		}
	}
	return result, nil
}

func appAnalysisSubjectManifest(manifest AppAnalysisManifest) AppAnalysisManifest {
	result := cloneAppAnalysisManifest(manifest)
	for index := range result.Attachments {
		result.Attachments[index].ProvenanceReceiptRef = ""
		result.Attachments[index].ProvenanceReceiptDigest = ""
	}
	return result
}

func normalizeAppAnalysisMediaType(value string) (string, error) {
	if value == "" || len(value) > 255 || strings.ContainsAny(value, "\x00\r\n") {
		return "", invalidAppAnalysis("media_type", nil)
	}
	parsed, parameters, err := mime.ParseMediaType(strings.TrimSpace(value))
	if err != nil || !strings.Contains(parsed, "/") {
		return "", invalidAppAnalysis("media_type", err)
	}
	normalized := mime.FormatMediaType(strings.ToLower(parsed), parameters)
	if normalized == "" || len(normalized) > 255 {
		return "", invalidAppAnalysis("media_type", nil)
	}
	return normalized, nil
}

func writeAppAnalysisAttachmentFingerprint(
	digest hash.Hash,
	attachment AppAnalysisAttachment,
	includeProvenance bool,
) {
	writeFingerprintField(digest, string(attachment.Kind))
	writeFingerprintField(digest, attachment.ArtifactRef.String())
	writeFingerprintField(digest, attachment.Digest)
	writeFingerprintField(digest, strconv.FormatInt(attachment.Size, 10))
	writeFingerprintField(digest, attachment.MediaType)
	writeFingerprintField(digest, string(attachment.Classification))
	if includeProvenance {
		writeFingerprintField(digest, attachment.ProvenanceReceiptRef)
		writeFingerprintField(digest, attachment.ProvenanceReceiptDigest)
	}
}

func validAppAnalysisDigest(value string) bool {
	if len(value) != sha256.Size*2 || strings.ToLower(value) != value {
		return false
	}
	decoded, err := hex.DecodeString(value)
	return err == nil && hex.EncodeToString(decoded) == value
}

func validAppAnalysisTreeOID(value string) bool {
	if (len(value) != 40 && len(value) != 64) || strings.ToLower(value) != value {
		return false
	}
	decoded, err := hex.DecodeString(value)
	return err == nil && hex.EncodeToString(decoded) == value
}

func validAppAnalysisOpaque(value string, maximum int) bool {
	return value != "" && len(value) <= maximum &&
		strings.TrimSpace(value) == value &&
		!strings.ContainsAny(value, "\x00\r\n")
}

func validAppAnalysisRef(ref AppAnalysisRef) bool {
	const prefix = "app-analysis:"
	return strings.HasPrefix(string(ref), prefix) &&
		validAppAnalysisDigest(strings.TrimPrefix(string(ref), prefix))
}

func appAnalysisAttachmentSortKey(attachment AppAnalysisAttachment) string {
	return string(attachment.Kind) + "\x00" + attachment.ArtifactRef.String()
}

func cloneAppAnalysisManifest(manifest AppAnalysisManifest) AppAnalysisManifest {
	manifest.Attachments = append([]AppAnalysisAttachment(nil), manifest.Attachments...)
	return manifest
}

func invalidAppAnalysis(field string, cause error) error {
	if cause == nil {
		return fmt.Errorf("%w: %s", ErrAppAnalysisInvalid, field)
	}
	return fmt.Errorf("%w: %s: %v", ErrAppAnalysisInvalid, field, cause)
}
