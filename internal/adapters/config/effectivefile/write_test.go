package effectivefile

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"orquesta/internal/config"
)

func TestWriteCreatesAndAtomicallyReplacesOwnerReadOnlyDocument(t *testing.T) {
	root := privateRoot(t)
	path := filepath.Join(root, "created", "nested", "effective.json")
	first := effectiveContent(t, "127.0.0.1:8080")
	second := effectiveContent(t, "127.0.0.1:8181")
	options := Options{Path: path, Content: first, MaxExistingBytes: 1 << 20}
	if err := Write(context.Background(), options); err != nil {
		t.Fatalf("write first: %v", err)
	}
	assertFile(t, path, first, 0o400)
	for _, directory := range []string{filepath.Join(root, "created"), filepath.Join(root, "created", "nested")} {
		info, err := os.Stat(directory)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0o700 {
			t.Fatalf("private directory %s mode = %o", directory, info.Mode().Perm())
		}
	}
	options.Content = second
	if err := Write(context.Background(), options); err != nil {
		t.Fatalf("replace: %v", err)
	}
	assertFile(t, path, second, 0o400)
	assertNoTemporaries(t, filepath.Dir(path))
}

func TestWriteFailpointAndCancellationPreservePreviousDocument(t *testing.T) {
	root := privateRoot(t)
	path := filepath.Join(root, "effective.json")
	previous := effectiveContent(t, "127.0.0.1:8080")
	next := effectiveContent(t, "127.0.0.1:8181")
	base := Options{Path: path, Content: previous, MaxExistingBytes: 1 << 20}
	if err := Write(context.Background(), base); err != nil {
		t.Fatal(err)
	}

	t.Run("failpoint after durable temporary", func(t *testing.T) {
		sentinel := errors.New("simulated-crash")
		called := false
		err := Write(context.Background(), Options{
			Path: path, Content: next, MaxExistingBytes: 1 << 20,
			Failpoint: func(stage string) error {
				called = true
				if stage != afterTempSyncFailpoint {
					t.Fatalf("stage = %q", stage)
				}
				matches, err := filepath.Glob(filepath.Join(root, ".effective-config-*"))
				if err != nil || len(matches) != 1 {
					t.Fatalf("temporary = %#v/%v", matches, err)
				}
				assertFile(t, matches[0], next, 0o600)
				return sentinel
			},
		})
		if !called || !errors.Is(err, sentinel) {
			t.Fatalf("failpoint result = %v", err)
		}
		assertFile(t, path, previous, 0o400)
		assertNoTemporaries(t, root)
	})

	t.Run("cancel after durable temporary", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		err := Write(ctx, Options{
			Path: path, Content: next, MaxExistingBytes: 1 << 20,
			Failpoint: func(stage string) error { cancel(); return nil },
		})
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("cancel result = %v", err)
		}
		assertFile(t, path, previous, 0o400)
		assertNoTemporaries(t, root)
	})
}

func TestWriteSerializesConcurrentWritersAcrossSharedLock(t *testing.T) {
	root := privateRoot(t)
	path := filepath.Join(root, "effective.json")
	first := effectiveContent(t, "127.0.0.1:8081")
	second := effectiveContent(t, "127.0.0.1:8082")
	entered := make(chan struct{})
	release := make(chan struct{})
	firstDone := make(chan error, 1)
	go func() {
		firstDone <- Write(context.Background(), Options{
			Path: path, Content: first, MaxExistingBytes: 1 << 20,
			Failpoint: func(string) error {
				close(entered)
				<-release
				return nil
			},
		})
	}()
	<-entered
	secondDone := make(chan error, 1)
	go func() {
		secondDone <- Write(context.Background(), Options{Path: path, Content: second, MaxExistingBytes: 1 << 20})
	}()
	select {
	case err := <-secondDone:
		t.Fatalf("second writer bypassed lock: %v", err)
	case <-time.After(50 * time.Millisecond):
	}
	close(release)
	if err := <-firstDone; err != nil {
		t.Fatalf("first writer: %v", err)
	}
	if err := <-secondDone; err != nil {
		t.Fatalf("second writer: %v", err)
	}
	assertFile(t, path, second, 0o400)
	assertFile(t, path+".lock", []byte{}, 0o600)
}

func TestWriteRejectsParentDirectoryReplacementBeforeRename(t *testing.T) {
	root := privateRoot(t)
	live := filepath.Join(root, "live")
	path := filepath.Join(live, "effective.json")
	previous := effectiveContent(t, "127.0.0.1:8080")
	next := effectiveContent(t, "127.0.0.1:8181")
	if err := Write(context.Background(), Options{Path: path, Content: previous, MaxExistingBytes: 1 << 20}); err != nil {
		t.Fatal(err)
	}
	entered := make(chan struct{})
	release := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- Write(context.Background(), Options{
			Path: path, Content: next, MaxExistingBytes: 1 << 20,
			Failpoint: func(string) error {
				close(entered)
				<-release
				return nil
			},
		})
	}()
	<-entered
	moved := filepath.Join(root, "moved")
	if err := os.Rename(live, moved); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(live, 0o700); err != nil {
		t.Fatal(err)
	}
	attacker := effectiveContent(t, "127.0.0.1:9999")
	writeMode(t, path, attacker, 0o400)
	close(release)
	err := <-done
	if !errors.Is(err, ErrUnsafeFilesystem) {
		t.Fatalf("replaced directory accepted: %v", err)
	}
	assertFile(t, filepath.Join(moved, "effective.json"), previous, 0o400)
	assertFile(t, path, attacker, 0o400)
	assertNoTemporaries(t, moved)
}

func TestReservedPathsAreExplicitAndDetached(t *testing.T) {
	path := "/private/effective.json"
	first := ReservedPaths(path)
	if !reflect.DeepEqual(first, []string{path + ".lock"}) {
		t.Fatalf("reserved paths = %#v", first)
	}
	first[0] = "mutated"
	if ReservedPaths(path)[0] == "mutated" || len(ReservedPaths("")) != 0 {
		t.Fatal("reserved path output shares state or accepts empty path")
	}
}

func TestWriteRejectsUnsafeDestinationWithoutChangingIt(t *testing.T) {
	valid := effectiveContent(t, "127.0.0.1:8080")
	tests := []struct {
		name string
		make func(*testing.T, string) (string, []byte)
		want error
	}{
		{
			name: "unknown ownership marker",
			make: func(t *testing.T, root string) (string, []byte) {
				path := filepath.Join(root, "effective.json")
				content := []byte(`{"document_type":"someone.else"}`)
				writeMode(t, path, content, 0o400)
				return path, content
			},
			want: ErrDestinationUnrecognized,
		},
		{
			name: "writable mode",
			make: func(t *testing.T, root string) (string, []byte) {
				path := filepath.Join(root, "effective.json")
				writeMode(t, path, valid, 0o600)
				return path, valid
			},
			want: ErrDestinationUnrecognized,
		},
		{
			name: "symlink",
			make: func(t *testing.T, root string) (string, []byte) {
				target := filepath.Join(root, "target.json")
				writeMode(t, target, valid, 0o400)
				path := filepath.Join(root, "effective.json")
				if err := os.Symlink(target, path); err != nil {
					t.Skip(err)
				}
				return path, valid
			},
			want: ErrUnsafeFilesystem,
		},
		{
			name: "hardlink",
			make: func(t *testing.T, root string) (string, []byte) {
				target := filepath.Join(root, "target.json")
				writeMode(t, target, valid, 0o400)
				path := filepath.Join(root, "effective.json")
				if err := os.Link(target, path); err != nil {
					t.Skip(err)
				}
				return path, valid
			},
			want: ErrDestinationUnrecognized,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := privateRoot(t)
			path, previous := test.make(t, root)
			err := Write(context.Background(), Options{Path: path, Content: valid, MaxExistingBytes: 1 << 20})
			if !errors.Is(err, test.want) {
				t.Fatalf("error = %v, want %v", err, test.want)
			}
			if test.name == "symlink" {
				content, readErr := os.ReadFile(filepath.Join(root, "target.json"))
				if readErr != nil || !bytes.Equal(content, previous) {
					t.Fatalf("target changed: %v", readErr)
				}
			} else {
				content, readErr := os.ReadFile(path)
				if readErr != nil || !bytes.Equal(content, previous) {
					t.Fatalf("destination changed: %v", readErr)
				}
			}
			assertNoTemporaries(t, root)
		})
	}
}

func TestWriteRejectsUnsafeDirectoryOwnerSizeAndDocument(t *testing.T) {
	valid := effectiveContent(t, "127.0.0.1:8080")
	t.Run("symlink directory", func(t *testing.T) {
		root := privateRoot(t)
		realDirectory := filepath.Join(root, "real")
		if err := os.Mkdir(realDirectory, 0o700); err != nil {
			t.Fatal(err)
		}
		linkedDirectory := filepath.Join(root, "linked")
		if err := os.Symlink(realDirectory, linkedDirectory); err != nil {
			t.Skip(err)
		}
		err := Write(context.Background(), Options{Path: filepath.Join(linkedDirectory, "effective.json"), Content: valid, MaxExistingBytes: 1 << 20})
		if !errors.Is(err, ErrUnsafeFilesystem) {
			t.Fatalf("error = %v", err)
		}
	})
	t.Run("public directory", func(t *testing.T) {
		root := privateRoot(t)
		directory := filepath.Join(root, "public")
		if err := os.Mkdir(directory, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(directory, 0o755); err != nil {
			t.Fatal(err)
		}
		err := Write(context.Background(), Options{Path: filepath.Join(directory, "effective.json"), Content: valid, MaxExistingBytes: 1 << 20})
		if !errors.Is(err, ErrUnsafeFilesystem) {
			t.Fatalf("error = %v", err)
		}
	})
	t.Run("oversize previous", func(t *testing.T) {
		root := privateRoot(t)
		path := filepath.Join(root, "effective.json")
		previous := append(append([]byte(nil), valid...), bytes.Repeat([]byte{' '}, 512)...)
		writeMode(t, path, previous, 0o400)
		err := Write(context.Background(), Options{Path: path, Content: valid, MaxExistingBytes: int64(len(valid) + 16)})
		if !errors.Is(err, ErrDestinationUnrecognized) {
			t.Fatalf("error = %v", err)
		}
		assertFile(t, path, previous, 0o400)
	})
	t.Run("foreign owner identity", func(t *testing.T) {
		identity := fileIdentity{uid: uint32(os.Geteuid()) + 1, links: 1}
		if err := validateDestinationProperties(0o400, int64(len(valid)), identity, 1<<20); !errors.Is(err, ErrDestinationUnrecognized) {
			t.Fatalf("foreign identity accepted: %v", err)
		}
	})
	t.Run("invalid new document", func(t *testing.T) {
		err := Write(context.Background(), Options{Path: filepath.Join(t.TempDir(), "effective.json"), Content: []byte(`{"secret":"value"}`), MaxExistingBytes: 1 << 20})
		if !errors.Is(err, ErrDocumentInvalid) {
			t.Fatalf("error = %v", err)
		}
	})
	t.Run("invalid options and canceled context", func(t *testing.T) {
		if err := Write(context.Background(), Options{}); !errors.Is(err, ErrInvalidOptions) {
			t.Fatalf("error = %v", err)
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		err := Write(ctx, Options{Path: filepath.Join(t.TempDir(), "effective.json"), Content: valid, MaxExistingBytes: 1 << 20})
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("error = %v", err)
		}
	})
}

func effectiveContent(t *testing.T, listen string) []byte {
	t.Helper()
	snapshot, err := config.Resolve(config.ResolveOptions{TOML: []byte("[server]\nlisten = \"" + listen + "\"\n")})
	if err != nil {
		t.Fatal(err)
	}
	content, err := snapshot.EffectiveJSON()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(content), "credential:") {
		t.Fatal("fixture contains credential material")
	}
	return content
}

func writeMode(t *testing.T, path string, content []byte, mode os.FileMode) {
	t.Helper()
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, mode); err != nil {
		t.Fatal(err)
	}
}

func privateRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.Chmod(root, 0o700); err != nil {
		t.Fatal(err)
	}
	return root
}

func assertFile(t *testing.T, path string, want []byte, mode os.FileMode) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(content, want) {
		t.Fatalf("content changed at %s", path)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != mode {
		t.Fatalf("mode = %o, want %o", info.Mode().Perm(), mode)
	}
}

func assertNoTemporaries(t *testing.T, root string) {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(root, ".effective-config-*"))
	if err != nil || len(matches) != 0 {
		t.Fatalf("temporary residue = %#v/%v", matches, err)
	}
}
