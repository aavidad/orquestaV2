//go:build linux

package filesource

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"orquesta/internal/credentials"
)

var validAuthJSON = []byte(`{"auth_mode":"chatgpt","tokens":{"access_token":"private"}}`)

func TestSourceReadsPrivateStableJSONObjectAndDestroysSecret(t *testing.T) {
	path := writeFixture(t, validAuthJSON, 0o600)
	source := newFixtureSource(t, path, uint64(len(validAuthJSON)))
	var retained credentials.Secret
	calls := 0
	if err := source.WithSecret(context.Background(), func(secret credentials.Secret) error {
		calls++
		retained = secret
		material := secret.Bytes()
		defer clear(material)
		if !bytes.Equal(material, validAuthJSON) {
			t.Fatalf("callback material mismatch: %q", material)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("callback calls = %d", calls)
	}
	if material := retained.Bytes(); len(material) != len(validAuthJSON) || !bytes.Equal(material, make([]byte, len(material))) {
		t.Fatalf("retained callback secret was not cleared: %q", material)
	}
	if got := fmt.Sprintf("%v %#v", source, source); bytes.Contains([]byte(got), []byte(path)) {
		t.Fatalf("source projection leaked path: %s", got)
	}
}

func TestSourceAcceptsOnlyExactPrivateFileAuthority(t *testing.T) {
	t.Run("mode_0400", func(t *testing.T) {
		path := writeFixture(t, validAuthJSON, 0o400)
		source := newFixtureSource(t, path, uint64(len(validAuthJSON)))
		if err := source.WithSecret(context.Background(), func(credentials.Secret) error { return nil }); err != nil {
			t.Fatal(err)
		}
	})

	tests := []struct {
		name  string
		setup func(*testing.T) (string, uint32, uint64)
	}{
		{name: "group readable mode", setup: func(t *testing.T) (string, uint32, uint64) {
			return writeFixture(t, validAuthJSON, 0o640), uint32(os.Geteuid()), uint64(len(validAuthJSON))
		}},
		{name: "owner mismatch", setup: func(t *testing.T) (string, uint32, uint64) {
			return writeFixture(t, validAuthJSON, 0o600), uint32(os.Geteuid()) + 1, uint64(len(validAuthJSON))
		}},
		{name: "hardlink", setup: func(t *testing.T) (string, uint32, uint64) {
			path := writeFixture(t, validAuthJSON, 0o600)
			if err := os.Link(path, path+".alias"); err != nil {
				t.Fatal(err)
			}
			return path, uint32(os.Geteuid()), uint64(len(validAuthJSON))
		}},
		{name: "final symlink", setup: func(t *testing.T) (string, uint32, uint64) {
			path := writeFixture(t, validAuthJSON, 0o600)
			link := path + ".link"
			if err := os.Symlink(path, link); err != nil {
				t.Fatal(err)
			}
			return link, uint32(os.Geteuid()), uint64(len(validAuthJSON))
		}},
		{name: "ancestor symlink", setup: func(t *testing.T) (string, uint32, uint64) {
			root := t.TempDir()
			realDirectory := filepath.Join(root, "real")
			if err := os.Mkdir(realDirectory, 0o700); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(realDirectory, "auth.json")
			if err := os.WriteFile(path, validAuthJSON, 0o600); err != nil {
				t.Fatal(err)
			}
			linkDirectory := filepath.Join(root, "linked")
			if err := os.Symlink(realDirectory, linkDirectory); err != nil {
				t.Fatal(err)
			}
			return filepath.Join(linkDirectory, "auth.json"), uint32(os.Geteuid()), uint64(len(validAuthJSON))
		}},
		{name: "oversize", setup: func(t *testing.T) (string, uint32, uint64) {
			return writeFixture(t, validAuthJSON, 0o600), uint32(os.Geteuid()), uint64(len(validAuthJSON) - 1)
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path, owner, maximum := test.setup(t)
			source, err := New(path, owner, maximum)
			if err != nil {
				t.Fatal(err)
			}
			called := false
			err = source.WithSecret(context.Background(), func(credentials.Secret) error {
				called = true
				return nil
			})
			if !credentials.HasErrorCode(err, credentials.ErrorUnsafeFile) || called {
				t.Fatalf("unsafe file accepted: called=%v err=%v", called, err)
			}
		})
	}
}

func TestSourceRejectsReplacementAndMutationAcrossPinnedReads(t *testing.T) {
	t.Run("replacement even with equal content", func(t *testing.T) {
		path := writeFixture(t, validAuthJSON, 0o600)
		source := newFixtureSource(t, path, uint64(len(validAuthJSON)))
		source.hooks.beforeReopen = func() {
			if err := os.Rename(path, path+".old"); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, validAuthJSON, 0o600); err != nil {
				t.Fatal(err)
			}
		}
		assertUnsafeWithoutCallback(t, source)
	})

	t.Run("truncated after metadata", func(t *testing.T) {
		path := writeFixture(t, validAuthJSON, 0o600)
		source := newFixtureSource(t, path, uint64(len(validAuthJSON)))
		source.hooks.afterFirstMetadata = func() {
			if err := os.Truncate(path, 2); err != nil {
				t.Fatal(err)
			}
		}
		assertUnsafeWithoutCallback(t, source)
	})
}

func TestSourceRequiresStrictNonemptyJSONObjectWithoutSchema(t *testing.T) {
	tests := []struct {
		name    string
		content []byte
		valid   bool
	}{
		{name: "unknown exact names", content: []byte(`{"Unknown":{"nested":1},"A":2,"a":3}`), valid: true},
		{name: "empty object", content: []byte(`{}`)},
		{name: "array root", content: []byte(`[1]`)},
		{name: "scalar root", content: []byte(`true`)},
		{name: "top duplicate", content: []byte(`{"a":1,"a":2}`)},
		{name: "escaped exact duplicate", content: []byte(`{"a":1,"\u0061":2}`)},
		{name: "nested duplicate", content: []byte(`{"a":{"b":1,"b":2}}`)},
		{name: "trailing object", content: []byte(`{"a":1}{"b":2}`)},
		{name: "trailing junk", content: []byte(`{"a":1}x`)},
		{name: "invalid utf8", content: []byte{'{', '"', 'a', '"', ':', '"', 0xff, '"', '}'}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := writeFixture(t, test.content, 0o600)
			source := newFixtureSource(t, path, uint64(len(test.content)))
			calls := 0
			err := source.WithSecret(context.Background(), func(credentials.Secret) error {
				calls++
				return nil
			})
			if test.valid {
				if err != nil || calls != 1 {
					t.Fatalf("valid schema-neutral object rejected: calls=%d err=%v", calls, err)
				}
				return
			}
			if !credentials.HasErrorCode(err, credentials.ErrorUnsafeFile) || calls != 0 {
				t.Fatalf("invalid JSON accepted: calls=%d err=%v", calls, err)
			}
		})
	}
}

func TestSourceCancellationTypedNilAndCallbackFailureAreFailClosed(t *testing.T) {
	path := writeFixture(t, validAuthJSON, 0o600)
	source := newFixtureSource(t, path, uint64(len(validAuthJSON)))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	called := false
	err := source.WithSecret(ctx, func(credentials.Secret) error { called = true; return nil })
	if !credentials.HasErrorCode(err, credentials.ErrorConsumerFailed) || !errors.Is(err, context.Canceled) || called {
		t.Fatalf("pre-cancel was not fail closed: called=%v err=%v", called, err)
	}

	var nilSource *Source
	if err := nilSource.WithSecret(context.Background(), func(credentials.Secret) error { return nil }); !credentials.HasErrorCode(err, credentials.ErrorInvalidRequest) {
		t.Fatalf("typed nil source accepted: %v", err)
	}
	if err := source.WithSecret(nil, func(credentials.Secret) error { return nil }); !credentials.HasErrorCode(err, credentials.ErrorInvalidRequest) {
		t.Fatalf("nil context accepted: %v", err)
	}
	var typedNilContext *testContext
	if err := source.WithSecret(typedNilContext, func(credentials.Secret) error { return nil }); !credentials.HasErrorCode(err, credentials.ErrorInvalidRequest) {
		t.Fatalf("typed nil context accepted: %v", err)
	}
	if err := source.WithSecret(context.Background(), nil); !credentials.HasErrorCode(err, credentials.ErrorInvalidRequest) {
		t.Fatalf("nil callback accepted: %v", err)
	}

	var retained credentials.Secret
	calls := 0
	err = source.WithSecret(context.Background(), func(secret credentials.Secret) error {
		calls++
		retained = secret
		return errors.New("private callback detail " + string(validAuthJSON))
	})
	projection := []byte(fmt.Sprintf("%v %+v %#v", err, err, err))
	if !credentials.HasErrorCode(err, credentials.ErrorConsumerFailed) || calls != 1 ||
		bytes.Contains(projection, validAuthJSON) || bytes.Contains(projection, []byte(path)) {
		t.Fatalf("unsafe callback failure: calls=%d err=%s", calls, projection)
	}
	if material := retained.Bytes(); len(material) != len(validAuthJSON) || !bytes.Equal(material, make([]byte, len(material))) {
		t.Fatalf("failed callback retained material: %q", material)
	}

	ctx, cancel = context.WithCancel(context.Background())
	err = source.WithSecret(ctx, func(secret credentials.Secret) error {
		retained = secret
		cancel()
		return nil
	})
	if !credentials.HasErrorCode(err, credentials.ErrorConsumerFailed) || !errors.Is(err, context.Canceled) {
		t.Fatalf("callback cancellation was not propagated safely: %v", err)
	}
	if material := retained.Bytes(); len(material) != len(validAuthJSON) || !bytes.Equal(material, make([]byte, len(material))) {
		t.Fatalf("cancelled callback retained material: %q", material)
	}
}

func TestSourceDestroysSecretWhenCallbackPanics(t *testing.T) {
	path := writeFixture(t, validAuthJSON, 0o600)
	source := newFixtureSource(t, path, uint64(len(validAuthJSON)))
	var retained credentials.Secret
	const panicMarker = "callback panic marker"

	func() {
		defer func() {
			if recovered := recover(); recovered != panicMarker {
				t.Fatalf("callback panic changed: %v", recovered)
			}
		}()
		_ = source.WithSecret(context.Background(), func(secret credentials.Secret) error {
			retained = secret
			panic(panicMarker)
		})
	}()

	material := retained.Bytes()
	defer clear(material)
	if len(material) != len(validAuthJSON) || !bytes.Equal(material, make([]byte, len(material))) {
		t.Fatalf("panicking callback retained material: %q", material)
	}
}

func TestSourceRejectsJSONObjectBeyondDepthLimit(t *testing.T) {
	for _, test := range []struct {
		name  string
		depth int
		valid bool
	}{
		{name: "boundary", depth: 126, valid: true},
		{name: "beyond boundary", depth: 127},
	} {
		t.Run(test.name, func(t *testing.T) {
			content := []byte(`{"root":` + strings.Repeat(`[`, test.depth) + `0` + strings.Repeat(`]`, test.depth) + `}`)
			path := writeFixture(t, content, 0o600)
			source := newFixtureSource(t, path, uint64(len(content)))
			called := false
			err := source.WithSecret(context.Background(), func(credentials.Secret) error {
				called = true
				return nil
			})
			if test.valid {
				if err != nil || !called {
					t.Fatalf("depth boundary rejected: called=%v err=%v", called, err)
				}
				return
			}
			if !credentials.HasErrorCode(err, credentials.ErrorUnsafeFile) || called {
				t.Fatalf("excessive depth accepted: called=%v err=%v", called, err)
			}
		})
	}
}

func TestSourceSupportsConcurrentIndependentCallbacks(t *testing.T) {
	path := writeFixture(t, validAuthJSON, 0o600)
	source := newFixtureSource(t, path, uint64(len(validAuthJSON)))
	const workers = 32
	var calls atomic.Int64
	var wait sync.WaitGroup
	errorsByWorker := make(chan error, workers)
	wait.Add(workers)
	for range workers {
		go func() {
			defer wait.Done()
			err := source.WithSecret(context.Background(), func(secret credentials.Secret) error {
				material := secret.Bytes()
				defer clear(material)
				if !bytes.Equal(material, validAuthJSON) {
					return errors.New("material mismatch")
				}
				calls.Add(1)
				return nil
			})
			errorsByWorker <- err
		}()
	}
	wait.Wait()
	close(errorsByWorker)
	for err := range errorsByWorker {
		if err != nil {
			t.Fatal(err)
		}
	}
	if calls.Load() != workers {
		t.Fatalf("callback calls = %d", calls.Load())
	}
}

func TestNewRejectsNonCanonicalOrUnboundedSource(t *testing.T) {
	for name, input := range map[string]struct {
		path string
		max  uint64
	}{
		"empty":          {path: "", max: 1},
		"relative":       {path: "auth.json", max: 1},
		"unclean":        {path: "/tmp/../tmp/auth.json", max: 1},
		"surrounding ws": {path: " /tmp/auth.json", max: 1},
		"nul":            {path: "/tmp/auth\x00.json", max: 1},
		"root":           {path: "/", max: 1},
		"zero max":       {path: "/tmp/auth.json", max: 0},
		"over hard max":  {path: "/tmp/auth.json", max: MaximumMaterialBytes + 1},
	} {
		t.Run(name, func(t *testing.T) {
			if source, err := New(input.path, uint32(os.Geteuid()), input.max); source != nil ||
				!credentials.HasErrorCode(err, credentials.ErrorInvalidRequest) {
				t.Fatalf("invalid constructor accepted: source=%v err=%v", source, err)
			}
		})
	}
}

type testContext struct{ context.Context }

func assertUnsafeWithoutCallback(t *testing.T, source *Source) {
	t.Helper()
	called := false
	err := source.WithSecret(context.Background(), func(credentials.Secret) error {
		called = true
		return nil
	})
	if !credentials.HasErrorCode(err, credentials.ErrorUnsafeFile) || called {
		t.Fatalf("unstable source accepted: called=%v err=%v", called, err)
	}
}

func writeFixture(t *testing.T, content []byte, mode os.FileMode) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "auth.json")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, mode); err != nil {
		t.Fatal(err)
	}
	return path
}

func newFixtureSource(t *testing.T, path string, maximum uint64) *Source {
	t.Helper()
	source, err := New(path, uint32(os.Geteuid()), maximum)
	if err != nil {
		t.Fatal(err)
	}
	return source
}
