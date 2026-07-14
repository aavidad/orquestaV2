package ports

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"orquesta/internal/goal"
)

func TestArtifactContractsBindReferenceDigestSizeAndMediaTypeToContent(t *testing.T) {
	content := []byte("contract artifact")
	digest := sha256.Sum256(content)
	digestText := hex.EncodeToString(digest[:])
	ref, _ := goal.NewArtifactRef("artifact:sha256:" + digestText)
	request := PutArtifactRequest{MediaType: "text/plain", Content: content}
	stored := StoredArtifact{Ref: ref, Digest: digestText, MediaType: request.MediaType, Size: int64(len(content))}
	if err := ValidateStoredArtifact(request, stored); err != nil {
		t.Fatalf("valid stored artifact: %v", err)
	}
	artifact := ArtifactContent{Ref: ref, Digest: digestText, Size: int64(len(content)), Content: content}
	if err := ValidateArtifactContent(artifact); err != nil {
		t.Fatalf("valid artifact content: %v", err)
	}

	wrong := stored
	wrong.Size++
	if code := ArtifactContractErrorCode(ValidateStoredArtifact(request, wrong)); code != "artifact.stored_size_mismatch" {
		t.Fatalf("stored mismatch code = %q", code)
	}
	corrupt := artifact
	corrupt.Content = []byte("different")
	if code := ArtifactContractErrorCode(ValidateArtifactContent(corrupt)); code != "artifact.content_ref_mismatch" {
		t.Fatalf("content mismatch code = %q", code)
	}
}
