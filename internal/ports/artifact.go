package ports

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"

	"orquesta/internal/goal"
)

const artifactSHA256RefPrefix = "artifact:sha256:"

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
	Code string
}

func (err *ArtifactContractError) Error() string {
	if err == nil {
		return ""
	}
	return err.Code
}

func ArtifactContractErrorCode(err error) string {
	var contractErr *ArtifactContractError
	if errors.As(err, &contractErr) {
		return contractErr.Code
	}
	return ""
}

func ValidateStoredArtifact(request PutArtifactRequest, stored StoredArtifact) error {
	digest := sha256.Sum256(request.Content)
	wantDigest := hex.EncodeToString(digest[:])
	switch {
	case stored.Ref.String() != artifactSHA256RefPrefix+wantDigest:
		return &ArtifactContractError{Code: "artifact.stored_ref_mismatch"}
	case stored.Digest != wantDigest:
		return &ArtifactContractError{Code: "artifact.stored_digest_mismatch"}
	case stored.MediaType != request.MediaType || strings.TrimSpace(stored.MediaType) == "":
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
