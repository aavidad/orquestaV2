package s3

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"math"
	"path"
	"strings"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

const (
	artifactRefPrefix = "artifact:sha256:"
	metadataSchema    = "orquesta.artifact.s3.v1"
)

// Options binds one Store instance to one project namespace. Limits are typed
// inputs from composition; this package reads no environment or global config.
type Options struct {
	Bucket           string
	ProjectRef       goal.ProjectRef
	MaxObjectBytes   int64
	ReconcileTimeout time.Duration
}

// Store adapts an injected S3-compatible object client to ArtifactStore.
// Store is stateless: restart creates another Store over the same client.
type Store struct {
	client           Client
	bucket           string
	projectDigest    string
	maxObjectBytes   int64
	reconcileTimeout time.Duration
}

func New(client Client, options Options) (*Store, error) {
	if client == nil || options.Bucket == "" || strings.TrimSpace(options.Bucket) != options.Bucket ||
		options.ProjectRef.String() == "" || options.MaxObjectBytes <= 0 ||
		options.MaxObjectBytes == math.MaxInt64 || options.ReconcileTimeout <= 0 {
		return nil, artifactError(ports.ArtifactErrorStoreUnavailable, nil)
	}
	projectDigest := sha256.Sum256([]byte(options.ProjectRef.String()))
	return &Store{
		client: client, bucket: options.Bucket,
		projectDigest:  hex.EncodeToString(projectDigest[:]),
		maxObjectBytes: options.MaxObjectBytes, reconcileTimeout: options.ReconcileTimeout,
	}, nil
}

func (store *Store) Put(ctx context.Context, request ports.PutArtifactRequest) (ports.StoredArtifact, error) {
	if err := store.available(ctx); err != nil {
		return ports.StoredArtifact{}, err
	}
	if err := ports.ValidateArtifactMediaType(request.MediaType); err != nil {
		return ports.StoredArtifact{}, err
	}
	if int64(len(request.Content)) > store.maxObjectBytes {
		return ports.StoredArtifact{}, artifactError(ports.ArtifactErrorSizeMismatch, nil)
	}

	digest := sha256.Sum256(request.Content)
	digestText := hex.EncodeToString(digest[:])
	ref, err := goal.NewArtifactRef(artifactRefPrefix + digestText)
	if err != nil {
		return ports.StoredArtifact{}, artifactError(ports.ArtifactErrorIO, err)
	}
	stored := ports.StoredArtifact{
		Ref: ref, Digest: digestText, MediaType: request.MediaType, Size: int64(len(request.Content)),
	}
	locator := store.locator(digestText)
	metadata := ObjectMetadata{
		Schema: metadataSchema, ProjectDigest: store.projectDigest, ArtifactDigest: digestText,
		OriginalMediaType: request.MediaType, Size: stored.Size,
	}
	putErr := store.client.PutIfAbsent(ctx, PutObjectRequest{
		ObjectLocator: locator, Body: bytes.NewReader(request.Content), Size: stored.Size, Metadata: metadata,
	})
	if putErr == nil || errors.Is(putErr, ErrObjectAlreadyExists) {
		if _, err := store.readVerified(ctx, locator, ref, digestText, request.MediaType, stored.Size); err != nil {
			return ports.StoredArtifact{}, err
		}
		return stored, nil
	}

	// A timeout after an object write is an unknown outcome. Reconcile the same
	// immutable key under a short detached context; never invent another key.
	reconcileCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), store.reconcileTimeout)
	defer cancel()
	if _, err := store.readVerified(reconcileCtx, locator, ref, digestText, request.MediaType, stored.Size); err == nil {
		return stored, nil
	} else if code := ports.ArtifactContractErrorCode(err); code != ports.ArtifactErrorNotFound &&
		code != ports.ArtifactErrorIO {
		return ports.StoredArtifact{}, err
	}
	return ports.StoredArtifact{}, artifactError(ports.ArtifactErrorIO, putErr)
}

func (store *Store) Get(
	ctx context.Context,
	ref goal.ArtifactRef,
	expectedSize int64,
) (ports.ArtifactContent, error) {
	if err := store.available(ctx); err != nil {
		return ports.ArtifactContent{}, err
	}
	if expectedSize < 0 {
		return ports.ArtifactContent{}, artifactError(ports.ArtifactErrorExpectedSizeInvalid, nil)
	}
	if expectedSize > store.maxObjectBytes || expectedSize == math.MaxInt64 {
		return ports.ArtifactContent{}, artifactError(ports.ArtifactErrorSizeMismatch, nil)
	}
	digest, err := digestFromRef(ref)
	if err != nil {
		return ports.ArtifactContent{}, err
	}
	return store.readVerified(ctx, store.locator(digest), ref, digest, "", expectedSize)
}

func (store *Store) readVerified(
	ctx context.Context,
	locator ObjectLocator,
	ref goal.ArtifactRef,
	digest string,
	expectedMediaType string,
	expectedSize int64,
) (ports.ArtifactContent, error) {
	if err := ctx.Err(); err != nil {
		return ports.ArtifactContent{}, artifactError(ports.ArtifactErrorIO, err)
	}
	info, err := store.client.Head(ctx, locator)
	if err != nil {
		return ports.ArtifactContent{}, mapClientError(err)
	}
	if info.Size != expectedSize || info.Metadata.Size != expectedSize || info.Size > store.maxObjectBytes {
		return ports.ArtifactContent{}, artifactError(ports.ArtifactErrorSizeMismatch, nil)
	}
	if info.Metadata.Schema != metadataSchema || info.Metadata.ProjectDigest != store.projectDigest ||
		info.Metadata.ArtifactDigest != digest {
		return ports.ArtifactContent{}, artifactError(ports.ArtifactErrorDigestMismatch, nil)
	}
	if err := ports.ValidateArtifactMediaType(info.Metadata.OriginalMediaType); err != nil {
		return ports.ArtifactContent{}, artifactError(ports.ArtifactErrorDigestMismatch, err)
	}
	if expectedMediaType != "" && info.Metadata.OriginalMediaType != expectedMediaType {
		return ports.ArtifactContent{}, artifactError(ports.ArtifactErrorFileChanged, nil)
	}

	stream, err := store.client.Open(ctx, locator)
	if err != nil {
		return ports.ArtifactContent{}, mapClientError(err)
	}
	if stream == nil {
		return ports.ArtifactContent{}, artifactError(ports.ArtifactErrorIO, nil)
	}
	content, readErr := io.ReadAll(io.LimitReader(stream, expectedSize+1))
	closeErr := stream.Close()
	if readErr != nil {
		return ports.ArtifactContent{}, artifactError(ports.ArtifactErrorIO, readErr)
	}
	if closeErr != nil {
		return ports.ArtifactContent{}, artifactError(ports.ArtifactErrorIO, closeErr)
	}
	if int64(len(content)) != expectedSize {
		return ports.ArtifactContent{}, artifactError(ports.ArtifactErrorSizeMismatch, nil)
	}
	contentDigest := sha256.Sum256(content)
	if hex.EncodeToString(contentDigest[:]) != digest {
		return ports.ArtifactContent{}, artifactError(ports.ArtifactErrorDigestMismatch, nil)
	}
	return ports.ArtifactContent{
		Ref: ref, Digest: digest, MediaType: info.Metadata.OriginalMediaType,
		Size: expectedSize, Content: content,
	}, nil
}

func (store *Store) available(ctx context.Context) error {
	if store == nil || store.client == nil || store.bucket == "" || store.projectDigest == "" {
		return artifactError(ports.ArtifactErrorStoreUnavailable, nil)
	}
	if err := ctx.Err(); err != nil {
		return artifactError(ports.ArtifactErrorIO, err)
	}
	return nil
}

func (store *Store) locator(digest string) ObjectLocator {
	return ObjectLocator{
		Bucket: store.bucket,
		Key:    path.Join("projects", store.projectDigest, "sha256", digest[:2], digest+".blob"),
	}
}

func digestFromRef(ref goal.ArtifactRef) (string, error) {
	digest, found := strings.CutPrefix(ref.String(), artifactRefPrefix)
	if !found || len(digest) != sha256.Size*2 {
		return "", artifactError(ports.ArtifactErrorRefInvalid, nil)
	}
	decoded, err := hex.DecodeString(digest)
	if err != nil || len(decoded) != sha256.Size {
		return "", artifactError(ports.ArtifactErrorRefInvalid, nil)
	}
	return digest, nil
}

func mapClientError(err error) error {
	if errors.Is(err, ErrObjectNotFound) {
		return artifactError(ports.ArtifactErrorNotFound, nil)
	}
	return artifactError(ports.ArtifactErrorIO, err)
}

func artifactError(code string, cause error) error {
	return ports.NewArtifactContractError(code, cause)
}

var _ application.ArtifactStore = (*Store)(nil)
