package keysource

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"unsafe"

	"orquesta/internal/credentials"
)

func TestSourceGeneratesCanonicalKeyOnceAndDestroysMaterial(t *testing.T) {
	seed := bytes.Repeat([]byte{0x2a}, ed25519.SeedSize)
	random := &capturingReader{material: seed}
	source := newTestSource(t, random)

	var retained credentials.Secret
	calls := 0
	err := source.WithPrivateKey(context.Background(), func(secret credentials.Secret) error {
		calls++
		retained = secret
		material := secret.Bytes()
		defer clear(material)
		canonical := ed25519.NewKeyFromSeed(seed)
		defer clear(canonical)
		if !bytes.Equal(material, canonical) {
			t.Fatalf("private key is not canonical")
		}
		return nil
	})
	if err != nil || calls != 1 {
		t.Fatalf("result: calls=%d error=%v", calls, err)
	}
	assertSecretDestroyed(t, retained)
	random.assertCapturedBufferCleared(t)
}

func TestSourceDestroysMaterialAndSanitizesCallbackFailure(t *testing.T) {
	privateDetail := bytes.Repeat([]byte("private-callback-detail"), 3)
	random := &capturingReader{material: bytes.Repeat([]byte{0x3b}, ed25519.SeedSize)}
	source := newTestSource(t, random)
	var retained credentials.Secret
	calls := 0
	err := source.WithPrivateKey(context.Background(), func(secret credentials.Secret) error {
		calls++
		retained = secret
		return errors.New(string(privateDetail))
	})
	projection := []byte(fmt.Sprintf("%v %+v %#v", err, err, err))
	if calls != 1 || !credentials.HasErrorCode(err, credentials.ErrorConsumerFailed) ||
		bytes.Contains(projection, privateDetail) {
		t.Fatalf("unsafe callback result: calls=%d error=%s", calls, projection)
	}
	assertSecretDestroyed(t, retained)
	random.assertCapturedBufferCleared(t)
}

func TestSourcePropagatesConsumerPanicAfterDestroyingMaterial(t *testing.T) {
	random := &capturingReader{material: bytes.Repeat([]byte{0x4c}, ed25519.SeedSize)}
	source := newTestSource(t, random)
	var retained credentials.Secret
	panicMarker := &struct{ label string }{"consumer-panic"}

	func() {
		defer func() {
			if recovered := recover(); recovered != panicMarker {
				t.Fatalf("consumer panic changed: %v", recovered)
			}
		}()
		_ = source.WithPrivateKey(context.Background(), func(secret credentials.Secret) error {
			retained = secret
			panic(panicMarker)
		})
	}()

	assertSecretDestroyed(t, retained)
	random.assertCapturedBufferCleared(t)
}

func TestSourceCancellationIsFailClosedAndDestroysMaterial(t *testing.T) {
	t.Run("before generation", func(t *testing.T) {
		random := &capturingReader{material: bytes.Repeat([]byte{0x5d}, ed25519.SeedSize)}
		source := newTestSource(t, random)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		called := false
		err := source.WithPrivateKey(ctx, func(credentials.Secret) error { called = true; return nil })
		if called || !credentials.HasErrorCode(err, credentials.ErrorConsumerFailed) ||
			!errors.Is(err, context.Canceled) || random.calls != 0 {
			t.Fatalf("pre-cancel result: called=%t reads=%d error=%v", called, random.calls, err)
		}
	})

	t.Run("during generation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		random := &capturingReader{
			material: bytes.Repeat([]byte{0x6e}, ed25519.SeedSize),
			after:    cancel,
		}
		source := newTestSource(t, random)
		called := false
		err := source.WithPrivateKey(ctx, func(credentials.Secret) error { called = true; return nil })
		if called || !credentials.HasErrorCode(err, credentials.ErrorConsumerFailed) ||
			!errors.Is(err, context.Canceled) {
			t.Fatalf("generation cancellation: called=%t error=%v", called, err)
		}
		random.assertCapturedBufferCleared(t)
	})

	t.Run("inside callback", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		random := &capturingReader{material: bytes.Repeat([]byte{0x7f}, ed25519.SeedSize)}
		source := newTestSource(t, random)
		var retained credentials.Secret
		calls := 0
		err := source.WithPrivateKey(ctx, func(secret credentials.Secret) error {
			calls++
			retained = secret
			cancel()
			return nil
		})
		if calls != 1 || !credentials.HasErrorCode(err, credentials.ErrorConsumerFailed) ||
			!errors.Is(err, context.Canceled) {
			t.Fatalf("callback cancellation: calls=%d error=%v", calls, err)
		}
		assertSecretDestroyed(t, retained)
		random.assertCapturedBufferCleared(t)
	})
}

func TestSourceRejectsInvalidDependencies(t *testing.T) {
	var nilReader *capturingReader
	if source, err := New(Options{Random: nilReader}); source != nil ||
		!credentials.HasErrorCode(err, credentials.ErrorInvalidRequest) {
		t.Fatalf("typed nil reader accepted: source=%v error=%v", source, err)
	}
	var nilUnsafePointer unsafePointerValue
	if !nilInterface(nilUnsafePointer) {
		t.Fatal("typed nil unsafe pointer accepted")
	}
	var marker byte
	if nilInterface(unsafePointerValue(unsafe.Pointer(&marker))) {
		t.Fatal("non-nil unsafe pointer rejected")
	}

	source := newTestSource(t, bytes.NewReader(make([]byte, ed25519.SeedSize)))
	var nilSource *Source
	var nilContext *testContext
	for name, call := range map[string]func() error{
		"typed nil source": func() error {
			return nilSource.WithPrivateKey(context.Background(), func(credentials.Secret) error { return nil })
		},
		"nil context": func() error {
			return source.WithPrivateKey(nil, func(credentials.Secret) error { return nil })
		},
		"typed nil context": func() error {
			return source.WithPrivateKey(nilContext, func(credentials.Secret) error { return nil })
		},
		"nil callback": func() error {
			return source.WithPrivateKey(context.Background(), nil)
		},
	} {
		t.Run(name, func(t *testing.T) {
			if err := call(); !credentials.HasErrorCode(err, credentials.ErrorInvalidRequest) {
				t.Fatalf("invalid dependency accepted: %v", err)
			}
		})
	}
}

func TestSourceRejectsShortAndFailedEntropyWithoutLeaking(t *testing.T) {
	privateDetail := []byte("private-random-failure")
	for _, test := range []struct {
		name   string
		reader io.Reader
	}{
		{
			name: "short",
			reader: &capturingReader{
				material: bytes.Repeat([]byte{0x81}, ed25519.SeedSize-1),
				err:      io.ErrUnexpectedEOF,
			},
		},
		{
			name: "failed",
			reader: &capturingReader{
				material: bytes.Repeat([]byte{0x92}, ed25519.SeedSize/2),
				err:      errors.New(string(privateDetail)),
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			source := newTestSource(t, test.reader)
			called := false
			err := source.WithPrivateKey(context.Background(), func(credentials.Secret) error {
				called = true
				return nil
			})
			projection := []byte(fmt.Sprintf("%v %+v %#v", err, err, err))
			if called || !credentials.HasErrorCode(err, credentials.ErrorConsumerFailed) ||
				bytes.Contains(projection, privateDetail) {
				t.Fatalf("unsafe entropy failure: called=%t error=%s", called, projection)
			}
			if captured, ok := test.reader.(*capturingReader); ok {
				captured.assertCapturedBufferCleared(t)
			}
		})
	}
}

func TestSourceClearsSeedAndUnlocksWhenEntropyPanics(t *testing.T) {
	random := &capturingReader{
		material:   bytes.Repeat([]byte{0xa3}, ed25519.SeedSize),
		panicValue: "entropy-panic",
	}
	source := newTestSource(t, random)
	func() {
		defer func() {
			if recovered := recover(); recovered != random.panicValue {
				t.Fatalf("entropy panic changed: %v", recovered)
			}
		}()
		_ = source.WithPrivateKey(context.Background(), func(credentials.Secret) error { return nil })
	}()
	random.assertCapturedBufferCleared(t)

	random.panicValue = nil
	if err := source.WithPrivateKey(context.Background(), func(credentials.Secret) error { return nil }); err != nil {
		t.Fatalf("source remained locked after panic: %v", err)
	}
}

func TestSourceSerializesInjectedReaderAndSupportsConcurrentCallbacks(t *testing.T) {
	random := &concurrencyRejectingReader{}
	source := newTestSource(t, random)
	const workers = 32
	var calls atomic.Int64
	var wait sync.WaitGroup
	errorsByWorker := make(chan error, workers)
	wait.Add(workers)
	for range workers {
		go func() {
			defer wait.Done()
			errorsByWorker <- source.WithPrivateKey(context.Background(), func(secret credentials.Secret) error {
				material := secret.Bytes()
				defer clear(material)
				if len(material) != ed25519.PrivateKeySize {
					return errors.New("private key size")
				}
				calls.Add(1)
				return nil
			})
		}()
	}
	wait.Wait()
	close(errorsByWorker)
	for err := range errorsByWorker {
		if err != nil {
			t.Fatal(err)
		}
	}
	if calls.Load() != workers || random.overlap.Load() {
		t.Fatalf("concurrent result: calls=%d reader_overlap=%t", calls.Load(), random.overlap.Load())
	}
}

func TestNewUsesCryptoRandAndSourceProjectionIsRedacted(t *testing.T) {
	source, err := New(Options{})
	if err != nil {
		t.Fatal(err)
	}
	if source.random != rand.Reader {
		t.Fatal("default random source is not crypto/rand.Reader")
	}
	calls := 0
	if err := source.WithPrivateKey(context.Background(), func(secret credentials.Secret) error {
		calls++
		material := secret.Bytes()
		defer clear(material)
		if len(material) != ed25519.PrivateKeySize {
			t.Fatalf("private key size = %d", len(material))
		}
		return nil
	}); err != nil || calls != 1 {
		t.Fatalf("production random result: calls=%d error=%v", calls, err)
	}
	if got := fmt.Sprintf("%v %#v", source, source); got != "[REDACTED] keysource.Source{[REDACTED]}" {
		t.Fatalf("unsafe source projection: %s", got)
	}
}

type capturingReader struct {
	material   []byte
	err        error
	after      func()
	panicValue any
	captured   []byte
	calls      int
}

func (reader *capturingReader) Read(destination []byte) (int, error) {
	reader.calls++
	reader.captured = destination
	written := copy(destination, reader.material)
	if reader.after != nil {
		reader.after()
	}
	if reader.panicValue != nil {
		panic(reader.panicValue)
	}
	if reader.err != nil {
		return written, reader.err
	}
	return written, nil
}

func (reader *capturingReader) assertCapturedBufferCleared(t *testing.T) {
	t.Helper()
	if len(reader.captured) != ed25519.SeedSize ||
		!bytes.Equal(reader.captured, make([]byte, len(reader.captured))) {
		t.Fatalf("seed buffer not cleared: %x", reader.captured)
	}
}

type concurrencyRejectingReader struct {
	inRead  atomic.Bool
	overlap atomic.Bool
	next    atomic.Uint32
}

func (reader *concurrencyRejectingReader) Read(destination []byte) (int, error) {
	if !reader.inRead.CompareAndSwap(false, true) {
		reader.overlap.Store(true)
		return 0, errors.New("concurrent read")
	}
	defer reader.inRead.Store(false)
	value := byte(reader.next.Add(1))
	for index := range destination {
		destination[index] = value
	}
	time.Sleep(time.Millisecond)
	return len(destination), nil
}

type testContext struct{ context.Context }

type unsafePointerValue unsafe.Pointer

func newTestSource(t *testing.T, random io.Reader) *Source {
	t.Helper()
	source, err := New(Options{Random: random})
	if err != nil {
		t.Fatal(err)
	}
	return source
}

func assertSecretDestroyed(t *testing.T, secret credentials.Secret) {
	t.Helper()
	material := secret.Bytes()
	defer clear(material)
	if len(material) != ed25519.PrivateKeySize ||
		!bytes.Equal(material, make([]byte, len(material))) {
		t.Fatalf("callback secret not destroyed: %x", material)
	}
}
