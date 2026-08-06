// Package keysource generates callback-scoped Ed25519 private keys in memory.
// It does not read configuration or persist credential material.
package keysource

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"io"
	"reflect"
	"sync"

	"orquesta/internal/credentials"
)

// Options contains the source's process-local dependencies. Random defaults to
// crypto/rand.Reader. Injection exists only to make entropy failures and
// cancellation deterministic in tests.
type Options struct {
	Random io.Reader
}

// Source generates a fresh canonical Ed25519 private key for each callback.
type Source struct {
	random   io.Reader
	randomMu *sync.Mutex
}

// New constructs an in-memory Ed25519 private-key source.
func New(options Options) (*Source, error) {
	random := options.Random
	if random == nil {
		random = rand.Reader
	}
	if nilInterface(random) {
		return nil, credentials.NewError(credentials.ErrorInvalidRequest, "ed25519_key_source")
	}
	return &Source{random: random, randomMu: &sync.Mutex{}}, nil
}

// WithPrivateKey invokes consume exactly once with a fresh canonical private
// key. Every adapter-owned seed, private key, and Secret copy is cleared before
// return, including error, cancellation, and panic paths.
//
// Consumer panics deliberately propagate: the consuming credentials use case
// owns panic sanitization. Deferred destruction still runs before propagation.
func (source *Source) WithPrivateKey(
	ctx context.Context,
	consume func(credentials.Secret) error,
) error {
	if source == nil || nilInterface(ctx) || consume == nil ||
		nilInterface(source.random) || source.randomMu == nil {
		return credentials.NewError(credentials.ErrorInvalidRequest, "ed25519_key_source")
	}
	if err := ctx.Err(); err != nil {
		return contextError(err)
	}

	seed := make([]byte, ed25519.SeedSize)
	defer clear(seed)
	if err := source.readSeed(seed); err != nil {
		return credentials.NewError(credentials.ErrorConsumerFailed, "ed25519_random")
	}
	if err := ctx.Err(); err != nil {
		return contextError(err)
	}

	privateKey := ed25519.NewKeyFromSeed(seed)
	defer clear(privateKey)
	secret, err := credentials.NewSecret(privateKey)
	if err != nil {
		return credentials.NewError(credentials.ErrorConsumerFailed, "ed25519_private_key")
	}
	defer secret.Destroy()

	if err := ctx.Err(); err != nil {
		return contextError(err)
	}
	if err := consume(secret); err != nil {
		if contextErr := ctx.Err(); contextErr != nil {
			return contextError(contextErr)
		}
		return credentials.NewError(credentials.ErrorConsumerFailed, "ed25519_private_key_callback")
	}
	if err := ctx.Err(); err != nil {
		return contextError(err)
	}
	return nil
}

func (source *Source) readSeed(seed []byte) error {
	source.randomMu.Lock()
	defer source.randomMu.Unlock()
	_, err := io.ReadFull(source.random, seed)
	return err
}

func contextError(err error) error {
	return credentials.WrapError(credentials.ErrorConsumerFailed, "context", err)
}

func nilInterface(value any) bool {
	if value == nil {
		return true
	}
	reflected := reflect.ValueOf(value)
	switch reflected.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice,
		reflect.UnsafePointer:
		return reflected.IsNil()
	default:
		return false
	}
}

func (*Source) String() string   { return "[REDACTED]" }
func (*Source) GoString() string { return "keysource.Source{[REDACTED]}" }

var _ credentials.Ed25519PrivateKeySource = (*Source)(nil)
