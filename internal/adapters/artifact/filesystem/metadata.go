package filesystem

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"path"

	"orquesta/internal/ports"
)

const (
	metadataSchema   = "orquesta.artifact.filesystem.v1"
	maxMetadataBytes = 1024
)

type artifactMetadata struct {
	Schema         string `json:"schema"`
	ProjectDigest  string `json:"project_digest"`
	ArtifactDigest string `json:"artifact_digest"`
	MediaType      string `json:"media_type"`
	Size           int64  `json:"size"`
}

func (store *Store) ensureArtifactMetadata(ctx context.Context, digest string, metadata artifactMetadata) error {
	encoded := encodeArtifactMetadata(metadata)
	metadataDigest := sha256.Sum256(encoded)
	err := store.ensureExistingOrWrite(
		ctx, store.metadataPath(digest), encoded, hex.EncodeToString(metadataDigest[:]),
	)
	if err == nil {
		return nil
	}
	switch ports.ArtifactContractErrorCode(err) {
	case ports.ArtifactErrorFileInvalid, ports.ArtifactErrorSizeMismatch, ports.ArtifactErrorDigestMismatch:
		actual, readErr := store.readArtifactMetadata(digest, metadata.Size)
		if readErr != nil {
			return readErr
		}
		if actual == metadata {
			return nil
		}
		if actual.Schema == metadata.Schema && actual.ProjectDigest == metadata.ProjectDigest &&
			actual.ArtifactDigest == metadata.ArtifactDigest && actual.Size == metadata.Size &&
			actual.MediaType != metadata.MediaType {
			return artifactError(ports.ArtifactErrorFileChanged, nil)
		}
		return artifactError(ports.ArtifactErrorDigestMismatch, nil)
	default:
		return err
	}
}

func (store *Store) readArtifactMetadata(digest string, expectedSize int64) (artifactMetadata, error) {
	encoded, err := store.readPrivateFileBounded(store.metadataPath(digest), maxMetadataBytes)
	if err != nil {
		return artifactMetadata{}, err
	}
	metadata, err := decodeArtifactMetadata(encoded)
	if err != nil || metadata.Schema != metadataSchema || metadata.ProjectDigest != store.projectDigest ||
		metadata.ArtifactDigest != digest {
		return artifactMetadata{}, artifactError(ports.ArtifactErrorDigestMismatch, err)
	}
	if metadata.Size != expectedSize {
		return artifactMetadata{}, artifactError(ports.ArtifactErrorSizeMismatch, nil)
	}
	if err := ports.ValidateArtifactMediaType(metadata.MediaType); err != nil {
		return artifactMetadata{}, artifactError(ports.ArtifactErrorDigestMismatch, err)
	}
	return metadata, nil
}

func (store *Store) metadataPath(digest string) string {
	return path.Join(store.namespace, "sha256", digest[:2], digest+".metadata")
}

func encodeArtifactMetadata(metadata artifactMetadata) []byte {
	encoded, _ := json.Marshal(metadata)
	return append(encoded, '\n')
}

func decodeArtifactMetadata(encoded []byte) (artifactMetadata, error) {
	var metadata artifactMetadata
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&metadata); err != nil {
		return artifactMetadata{}, err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return artifactMetadata{}, errors.New("artifact metadata has trailing data")
	}
	if !bytes.Equal(encoded, encodeArtifactMetadata(metadata)) {
		return artifactMetadata{}, errors.New("artifact metadata is not canonical")
	}
	return metadata, nil
}
