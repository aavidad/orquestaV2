package ports

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"mime"
	"strings"

	"orquesta/internal/goal"
)

const artifactSHA256RefPrefix = "artifact:sha256:"

const (
	ArtifactErrorStoreUnavailable      = "artifact.store_unavailable"
	ArtifactErrorRootRequired          = "artifact.root_required"
	ArtifactErrorRootInvalid           = "artifact.root_invalid"
	ArtifactErrorRootPermissions       = "artifact.root_permissions"
	ArtifactErrorDirectoryInvalid      = "artifact.directory_invalid"
	ArtifactErrorFileInvalid           = "artifact.file_invalid"
	ArtifactErrorFileChanged           = "artifact.file_changed"
	ArtifactErrorSizeMismatch          = "artifact.size_mismatch"
	ArtifactErrorDigestMismatch        = "artifact.digest_mismatch"
	ArtifactErrorNotFound              = "artifact.not_found"
	ArtifactErrorRefInvalid            = "artifact.ref_invalid"
	ArtifactErrorExpectedSizeInvalid   = "artifact.expected_size_invalid"
	ArtifactErrorMediaTypeInvalid      = "artifact.media_type_invalid"
	ArtifactErrorFilesystemUnsupported = "artifact.filesystem_unsupported"
	ArtifactErrorIO                    = "artifact.io"
)

type PutArtifactRequest struct {
	MediaType string
	Content   []byte
}

type StoredArtifact struct {
	Ref       goal.ArtifactRef
	Digest    string
	MediaType string
	Size      int64
}

type ArtifactContent struct {
	Ref       goal.ArtifactRef
	Digest    string
	MediaType string
	Size      int64
	Content   []byte
}

type ArtifactContractError struct {
	Code  string
	cause error
}

func (err *ArtifactContractError) Error() string {
	if err == nil {
		return ""
	}
	return err.Code
}

func (err *ArtifactContractError) Unwrap() error {
	if err == nil {
		return nil
	}
	return err.cause
}

func NewArtifactContractError(code string, cause error) *ArtifactContractError {
	return &ArtifactContractError{Code: code, cause: cause}
}

func ArtifactContractErrorCode(err error) string {
	var contractErr *ArtifactContractError
	if errors.As(err, &contractErr) {
		return contractErr.Code
	}
	return ""
}

func ValidateArtifactMediaType(value string) error {
	if value == "" || len(value) > 255 || strings.TrimSpace(value) != value {
		return &ArtifactContractError{Code: ArtifactErrorMediaTypeInvalid}
	}
	parsed, _, err := mime.ParseMediaType(value)
	if err != nil || !strings.Contains(parsed, "/") {
		return &ArtifactContractError{Code: ArtifactErrorMediaTypeInvalid}
	}
	return nil
}

func ValidateStoredArtifact(request PutArtifactRequest, stored StoredArtifact) error {
	if err := ValidateArtifactMediaType(request.MediaType); err != nil {
		return err
	}
	digest := sha256.Sum256(request.Content)
	wantDigest := hex.EncodeToString(digest[:])
	switch {
	case stored.Ref.String() != artifactSHA256RefPrefix+wantDigest:
		return &ArtifactContractError{Code: "artifact.stored_ref_mismatch"}
	case stored.Digest != wantDigest:
		return &ArtifactContractError{Code: "artifact.stored_digest_mismatch"}
	case stored.MediaType != request.MediaType:
		return &ArtifactContractError{Code: "artifact.stored_media_type_mismatch"}
	case stored.Size != int64(len(request.Content)):
		return &ArtifactContractError{Code: "artifact.stored_size_mismatch"}
	default:
		return nil
	}
}

func ValidateArtifactContent(content ArtifactContent) error {
	digest := sha256.Sum256(content.Content)
	wantDigest := hex.EncodeToString(digest[:])
	switch {
	case content.Ref.String() != artifactSHA256RefPrefix+wantDigest:
		return &ArtifactContractError{Code: "artifact.content_ref_mismatch"}
	case content.Digest != wantDigest:
		return &ArtifactContractError{Code: "artifact.content_digest_mismatch"}
	case content.Size != int64(len(content.Content)):
		return &ArtifactContractError{Code: "artifact.content_size_mismatch"}
	default:
		return nil
	}
}
