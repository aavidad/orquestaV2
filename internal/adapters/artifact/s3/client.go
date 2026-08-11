package s3

import (
	"context"
	"errors"
	"io"
)

// ErrObjectAlreadyExists reports that a conditional object creation lost a
// race against an existing immutable object.
var ErrObjectAlreadyExists = errors.New("s3.object_already_exists")

// ErrObjectNotFound reports that the requested object does not exist.
var ErrObjectNotFound = errors.New("s3.object_not_found")

// ObjectLocator identifies an object without exposing credentials or client
// configuration to the artifact contract.
type ObjectLocator struct {
	Bucket string
	Key    string
}

// ObjectMetadata binds an immutable blob to its content and project namespace.
// ProjectDigest is deliberately irreversible: project references never become
// backend paths or externally visible object metadata.
type ObjectMetadata struct {
	Schema            string
	ProjectDigest     string
	ArtifactDigest    string
	OriginalMediaType string
	Size              int64
}

// PutObjectRequest asks a backend client to create exactly one immutable key.
// Implementations must apply create-only/If-None-Match semantics atomically.
type PutObjectRequest struct {
	ObjectLocator
	Body     io.Reader
	Size     int64
	Metadata ObjectMetadata
}

// ObjectInfo contains verified metadata returned by the object backend.
type ObjectInfo struct {
	Size     int64
	Metadata ObjectMetadata
}

// Client is the narrow, SDK-neutral object client required by this adapter.
// Credentials, endpoints, retries and provider-specific errors remain owned by
// the concrete client injected by composition.
type Client interface {
	PutIfAbsent(context.Context, PutObjectRequest) error
	Head(context.Context, ObjectLocator) (ObjectInfo, error)
	Open(context.Context, ObjectLocator) (io.ReadCloser, error)
}
