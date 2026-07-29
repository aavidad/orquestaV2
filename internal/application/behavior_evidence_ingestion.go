package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"mime"
	"strconv"
	"time"
	"unicode/utf8"
)

const (
	BehaviorEvidenceManifestSchema         = "application.behavior-evidence.manifest.v1"
	BehaviorEvidenceContractVersion        = "1.0.0"
	BehaviorEvidenceReceiptSchema          = "orquesta.application-behavior-evidence.receipt.v1"
	BehaviorEvidenceMaxArtifacts           = 64
	BehaviorEvidenceMaxArtifactBytes int64 = 16 * 1024 * 1024
	BehaviorEvidenceMaxTotalBytes    int64 = 64 * 1024 * 1024
)

var (
	ErrBehaviorEvidenceInvalid  = errors.New("application.behavior_evidence_invalid")
	ErrBehaviorEvidenceConflict = errors.New("application.behavior_evidence_conflict")
)

type BehaviorEvidenceProducer struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type BehaviorEvidenceReview struct {
	Status               string `json:"status"`
	ReviewerRef          string `json:"reviewer_ref"`
	ReviewedAt           string `json:"reviewed_at"`
	ReviewedDigestSHA256 string `json:"reviewed_digest_sha256"`
	Scope                string `json:"scope"`
}

type BehaviorEvidenceArtifact struct {
	ArtifactRef    string `json:"artifact_ref"`
	Kind           string `json:"kind"`
	Schema         string `json:"schema"`
	MediaType      string `json:"media_type"`
	SizeBytes      int64  `json:"size_bytes"`
	SHA256         string `json:"sha256"`
	Classification string `json:"classification"`
	ReviewStatus   string `json:"review_status"`
}

type BehaviorEvidenceManifest struct {
	Schema          string                     `json:"schema"`
	ContractVersion string                     `json:"contract_version"`
	AnalysisRef     string                     `json:"analysis_ref"`
	RequestRef      string                     `json:"request_ref"`
	SubjectRef      string                     `json:"subject_ref"`
	Purpose         string                     `json:"purpose"`
	Producer        BehaviorEvidenceProducer   `json:"producer"`
	CreatedAt       string                     `json:"created_at"`
	Classification  string                     `json:"classification"`
	Review          BehaviorEvidenceReview     `json:"review"`
	Artifacts       []BehaviorEvidenceArtifact `json:"artifacts"`
	ManifestDigest  string                     `json:"manifest_digest"`
}

type BehaviorEvidenceReceipt struct {
	Schema         string `json:"schema"`
	ReceiptRef     string `json:"receipt_ref"`
	ManifestDigest string `json:"manifest_digest"`
	AcceptedAt     string `json:"accepted_at"`
}

// BehaviorEvidenceReceiptStore is the replaceable persistence boundary. The
// adapter must atomically return the original receipt when the same manifest
// digest is accepted again.
type BehaviorEvidenceReceiptStore interface {
	AcceptBehaviorEvidenceManifest(
		context.Context,
		BehaviorEvidenceManifest,
		time.Time,
	) (BehaviorEvidenceReceipt, error)
}

type BehaviorEvidenceIngestionService struct {
	receipts BehaviorEvidenceReceiptStore
	clock    Clock
}

func NewBehaviorEvidenceIngestionService(
	receipts BehaviorEvidenceReceiptStore,
	clock Clock,
) (*BehaviorEvidenceIngestionService, error) {
	if receipts == nil || clock == nil {
		return nil, errors.New("application.behavior_evidence_config_invalid")
	}
	return &BehaviorEvidenceIngestionService{receipts: receipts, clock: clock}, nil
}

func (service *BehaviorEvidenceIngestionService) IngestBehaviorEvidenceManifest(
	ctx context.Context,
	manifest BehaviorEvidenceManifest,
) (BehaviorEvidenceReceipt, error) {
	if service == nil || service.receipts == nil || service.clock == nil {
		return BehaviorEvidenceReceipt{}, errors.New("application.unavailable")
	}
	if err := ValidateBehaviorEvidenceManifest(manifest); err != nil {
		return BehaviorEvidenceReceipt{}, err
	}
	receipt, err := service.receipts.AcceptBehaviorEvidenceManifest(
		ctx,
		cloneBehaviorEvidenceManifest(manifest),
		service.clock.Now().UTC(),
	)
	if err != nil {
		return BehaviorEvidenceReceipt{}, err
	}
	if err := validateBehaviorEvidenceReceipt(receipt, manifest.ManifestDigest); err != nil {
		return BehaviorEvidenceReceipt{}, err
	}
	return receipt, nil
}

func ValidateBehaviorEvidenceManifest(manifest BehaviorEvidenceManifest) error {
	if err := validateBehaviorEvidenceShape(manifest); err != nil {
		return err
	}
	digest, err := BehaviorEvidenceManifestDigest(manifest)
	if err != nil {
		return err
	}
	if manifest.ManifestDigest != digest {
		return ErrBehaviorEvidenceInvalid
	}
	return nil
}

// BehaviorEvidenceManifestDigest implements the connector's normative
// canonical JSON: UTF-8, lexicographically sorted object keys, compact output,
// arrays in their original order, and manifest_digest omitted.
func BehaviorEvidenceManifestDigest(manifest BehaviorEvidenceManifest) (string, error) {
	if err := validateBehaviorEvidenceShapeWithoutDigest(manifest); err != nil {
		return "", err
	}
	canonical := appendBehaviorEvidenceManifest(nil, manifest)
	sum := sha256.Sum256(canonical)
	return hex.EncodeToString(sum[:]), nil
}

func validateBehaviorEvidenceShape(manifest BehaviorEvidenceManifest) error {
	if !validBehaviorEvidenceDigest(manifest.ManifestDigest) {
		return ErrBehaviorEvidenceInvalid
	}
	return validateBehaviorEvidenceShapeWithoutDigest(manifest)
}

func validateBehaviorEvidenceShapeWithoutDigest(manifest BehaviorEvidenceManifest) error {
	if manifest.Schema != BehaviorEvidenceManifestSchema ||
		manifest.ContractVersion != BehaviorEvidenceContractVersion ||
		!validBehaviorEvidenceOpaqueRef(manifest.AnalysisRef) ||
		!validBehaviorEvidenceOpaqueRef(manifest.RequestRef) ||
		!validBehaviorEvidenceOpaqueRef(manifest.SubjectRef) ||
		(manifest.Purpose != "clean_room_reimplementation" &&
			manifest.Purpose != "refactor_existing") ||
		!validBehaviorEvidenceText(manifest.Producer.Name, 160, true) ||
		!validBehaviorEvidenceText(manifest.Producer.Version, 80, true) ||
		!validBehaviorEvidenceTimestamp(manifest.CreatedAt) ||
		manifest.Classification != "shareable_redacted" ||
		manifest.Review.Status != "approved" ||
		!validBehaviorEvidenceOpaqueRef(manifest.Review.ReviewerRef) ||
		!validBehaviorEvidenceTimestamp(manifest.Review.ReviewedAt) ||
		!validBehaviorEvidenceDigest(manifest.Review.ReviewedDigestSHA256) ||
		manifest.Review.Scope != "shareable_redacted_evidence" ||
		len(manifest.Artifacts) > BehaviorEvidenceMaxArtifacts {
		return ErrBehaviorEvidenceInvalid
	}

	var total int64
	seenRefs := make(map[string]struct{}, len(manifest.Artifacts))
	for _, artifact := range manifest.Artifacts {
		if !validBehaviorEvidenceOpaqueRef(artifact.ArtifactRef) ||
			!validBehaviorEvidenceText(artifact.Kind, 80, true) ||
			!validBehaviorEvidenceText(artifact.Schema, 160, true) ||
			!validBehaviorEvidenceText(artifact.MediaType, 100, true) ||
			artifact.SizeBytes < 0 ||
			artifact.SizeBytes > BehaviorEvidenceMaxArtifactBytes ||
			!validBehaviorEvidenceDigest(artifact.SHA256) ||
			artifact.Classification != "shareable_redacted" ||
			artifact.ReviewStatus != "approved" {
			return ErrBehaviorEvidenceInvalid
		}
		if _, _, err := mime.ParseMediaType(artifact.MediaType); err != nil {
			return ErrBehaviorEvidenceInvalid
		}
		if _, duplicate := seenRefs[artifact.ArtifactRef]; duplicate {
			return ErrBehaviorEvidenceInvalid
		}
		seenRefs[artifact.ArtifactRef] = struct{}{}
		if total > BehaviorEvidenceMaxTotalBytes-artifact.SizeBytes {
			return ErrBehaviorEvidenceInvalid
		}
		total += artifact.SizeBytes
	}
	return nil
}

func validateBehaviorEvidenceReceipt(
	receipt BehaviorEvidenceReceipt,
	manifestDigest string,
) error {
	if receipt.Schema != BehaviorEvidenceReceiptSchema ||
		!validBehaviorEvidenceOpaqueRef(receipt.ReceiptRef) ||
		receipt.ManifestDigest != manifestDigest ||
		!validBehaviorEvidenceTimestamp(receipt.AcceptedAt) {
		return ErrBehaviorEvidenceConflict
	}
	return nil
}

func validBehaviorEvidenceOpaqueRef(value string) bool {
	if len(value) == 0 || len(value) > 160 {
		return false
	}
	for _, current := range value {
		if !((current >= 'a' && current <= 'z') ||
			(current >= 'A' && current <= 'Z') ||
			(current >= '0' && current <= '9') ||
			current == '.' || current == '_' || current == '-') {
			return false
		}
	}
	return true
}

func validBehaviorEvidenceDigest(value string) bool {
	if len(value) != sha256.Size*2 {
		return false
	}
	for _, current := range value {
		if !((current >= '0' && current <= '9') ||
			(current >= 'a' && current <= 'f')) {
			return false
		}
	}
	return true
}

func validBehaviorEvidenceText(value string, maximum int, required bool) bool {
	count := utf8.RuneCountInString(value)
	return utf8.ValidString(value) && count <= maximum && (!required || count > 0)
}

func validBehaviorEvidenceTimestamp(value string) bool {
	if value == "" {
		return false
	}
	_, err := time.Parse(time.RFC3339, value)
	return err == nil
}

func cloneBehaviorEvidenceManifest(manifest BehaviorEvidenceManifest) BehaviorEvidenceManifest {
	clone := manifest
	clone.Artifacts = append([]BehaviorEvidenceArtifact(nil), manifest.Artifacts...)
	return clone
}

func appendBehaviorEvidenceManifest(target []byte, manifest BehaviorEvidenceManifest) []byte {
	target = append(target, '{')
	target = appendJSONField(target, "analysis_ref", manifest.AnalysisRef, true)
	target = append(target, ',', '"')
	target = append(target, "artifacts"...)
	target = append(target, '"', ':', '[')
	for index, artifact := range manifest.Artifacts {
		if index > 0 {
			target = append(target, ',')
		}
		target = appendBehaviorEvidenceArtifact(target, artifact)
	}
	target = append(target, ']')
	target = appendJSONField(target, "classification", manifest.Classification, false)
	target = appendJSONField(target, "contract_version", manifest.ContractVersion, false)
	target = appendJSONField(target, "created_at", manifest.CreatedAt, false)
	target = append(target, ',', '"')
	target = append(target, "producer"...)
	target = append(target, '"', ':', '{')
	target = appendJSONField(target, "name", manifest.Producer.Name, true)
	target = appendJSONField(target, "version", manifest.Producer.Version, false)
	target = append(target, '}')
	target = appendJSONField(target, "purpose", manifest.Purpose, false)
	target = appendJSONField(target, "request_ref", manifest.RequestRef, false)
	target = append(target, ',', '"')
	target = append(target, "review"...)
	target = append(target, '"', ':', '{')
	target = appendJSONField(target, "reviewed_at", manifest.Review.ReviewedAt, true)
	target = appendJSONField(target, "reviewed_digest_sha256", manifest.Review.ReviewedDigestSHA256, false)
	target = appendJSONField(target, "reviewer_ref", manifest.Review.ReviewerRef, false)
	target = appendJSONField(target, "scope", manifest.Review.Scope, false)
	target = appendJSONField(target, "status", manifest.Review.Status, false)
	target = append(target, '}')
	target = appendJSONField(target, "schema", manifest.Schema, false)
	target = appendJSONField(target, "subject_ref", manifest.SubjectRef, false)
	return append(target, '}')
}

func appendBehaviorEvidenceArtifact(
	target []byte,
	artifact BehaviorEvidenceArtifact,
) []byte {
	target = append(target, '{')
	target = appendJSONField(target, "artifact_ref", artifact.ArtifactRef, true)
	target = appendJSONField(target, "classification", artifact.Classification, false)
	target = appendJSONField(target, "kind", artifact.Kind, false)
	target = appendJSONField(target, "media_type", artifact.MediaType, false)
	target = appendJSONField(target, "review_status", artifact.ReviewStatus, false)
	target = appendJSONField(target, "schema", artifact.Schema, false)
	target = appendJSONField(target, "sha256", artifact.SHA256, false)
	target = append(target, ',')
	target = appendJSONString(target, "size_bytes")
	target = append(target, ':')
	target = strconv.AppendInt(target, artifact.SizeBytes, 10)
	return append(target, '}')
}

func appendJSONField(target []byte, key string, value string, first bool) []byte {
	if !first {
		target = append(target, ',')
	}
	target = appendJSONString(target, key)
	target = append(target, ':')
	return appendJSONString(target, value)
}

func appendJSONString(target []byte, value string) []byte {
	const hexadecimal = "0123456789abcdef"
	target = append(target, '"')
	for _, current := range value {
		switch current {
		case '"', '\\':
			target = append(target, '\\', byte(current))
		case '\b':
			target = append(target, '\\', 'b')
		case '\f':
			target = append(target, '\\', 'f')
		case '\n':
			target = append(target, '\\', 'n')
		case '\r':
			target = append(target, '\\', 'r')
		case '\t':
			target = append(target, '\\', 't')
		default:
			if current < 0x20 {
				target = append(
					target,
					'\\', 'u', '0', '0',
					hexadecimal[byte(current)>>4],
					hexadecimal[byte(current)&0x0f],
				)
			} else {
				target = utf8.AppendRune(target, current)
			}
		}
	}
	return append(target, '"')
}
