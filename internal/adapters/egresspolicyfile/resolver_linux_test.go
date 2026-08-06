//go:build linux

package egresspolicyfile

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	microvm "github.com/aavidad/agente_microvm/conectores/orquesta"

	"orquesta/internal/application"
)

func TestResolverLoadsExactPolicyOnce(t *testing.T) {
	ref, path, raw, digest := writeValidPolicy(t, "exact")
	resolver, err := New(ref, path, digest, uint32(os.Geteuid()), int64(len(raw)))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Filesystem changes after construction cannot alter the loaded authority.
	if err := os.WriteFile(path, []byte("replaced after construction"), 0o600); err != nil {
		t.Fatal(err)
	}
	want := application.EgressPolicyAuthority{
		PolicyRef: ref, PayloadSHA256: digest, CanonicalPayload: string(raw),
	}
	got, err := resolver.ResolveEgressPolicy(context.Background(), ref)
	if err != nil || got != want || application.ValidateEgressPolicyAuthority(got) != nil {
		t.Fatalf("ResolveEgressPolicy() = %+v, %v; want %+v", got, err, want)
	}
	got.CanonicalPayload = "caller-local-change"
	again, err := resolver.ResolveEgressPolicy(context.Background(), ref)
	if err != nil || again != want {
		t.Fatalf("second ResolveEgressPolicy() = %+v, %v; want immutable %+v", again, err, want)
	}

	other, _ := application.NewEgressPolicyRef("egreso:other")
	if got, err := resolver.ResolveEgressPolicy(context.Background(), other); got != (application.EgressPolicyAuthority{}) ||
		!IsError(err, CodePolicyNotFound) {
		t.Fatalf("different ref = %+v, %v", got, err)
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := resolver.ResolveEgressPolicy(cancelled, ref); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled context error = %v", err)
	}
	if _, err := resolver.ResolveEgressPolicy(nil, ref); !IsError(err, CodeOptionsInvalid) {
		t.Fatalf("nil context error = %v", err)
	}
	var typedNil *typedNilContext
	if _, err := resolver.ResolveEgressPolicy(typedNil, ref); !IsError(err, CodeOptionsInvalid) {
		t.Fatalf("typed nil context error = %v", err)
	}
	var absent *Resolver
	if _, err := absent.ResolveEgressPolicy(context.Background(), ref); !IsError(err, CodeResolverUnavailable) {
		t.Fatalf("nil resolver error = %v", err)
	}
	var typedNilError *Error
	if IsError(typedNilError, CodeResolverUnavailable) {
		t.Fatal("typed nil error matched a concrete adapter error")
	}
}

func TestResolverConcurrentReadsRemainImmutable(t *testing.T) {
	ref, path, raw, digest := writeValidPolicy(t, "concurrent")
	resolver, err := New(ref, path, digest, uint32(os.Geteuid()), maximumCanonicalPayloadBytes)
	if err != nil {
		t.Fatal(err)
	}
	want := string(raw)
	var wait sync.WaitGroup
	for index := 0; index < 64; index++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			for attempt := 0; attempt < 100; attempt++ {
				authority, resolveErr := resolver.ResolveEgressPolicy(context.Background(), ref)
				if resolveErr != nil || authority.CanonicalPayload != want || authority.PayloadSHA256 != digest {
					t.Errorf("ResolveEgressPolicy() = %+v, %v", authority, resolveErr)
					return
				}
			}
		}()
	}
	wait.Wait()
}

func TestNewRejectsInvalidOptionsBeforeReading(t *testing.T) {
	ref, path, raw, digest := writeValidPolicy(t, "options")
	badRef := application.EgressPolicyRef(" invalid")
	upperDigest := strings.ToUpper(digest)
	tests := []struct {
		name     string
		ref      application.EgressPolicyRef
		path     string
		digest   string
		maximum  int64
		wantCode string
	}{
		{"invalid ref", badRef, path, digest, int64(len(raw)), CodeOptionsInvalid},
		{"relative path", ref, filepath.Base(path), digest, int64(len(raw)), CodeOptionsInvalid},
		{"unclean path", ref, filepath.Dir(path) + "/./" + filepath.Base(path), digest, int64(len(raw)), CodeOptionsInvalid},
		{"empty digest", ref, path, "", int64(len(raw)), CodeOptionsInvalid},
		{"uppercase digest", ref, path, upperDigest, int64(len(raw)), CodeOptionsInvalid},
		{"zero maximum", ref, path, digest, 0, CodeOptionsInvalid},
		{"excessive maximum", ref, path, digest, maximumCanonicalPayloadBytes + 1, CodeOptionsInvalid},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resolver, err := New(test.ref, test.path, test.digest, uint32(os.Geteuid()), test.maximum)
			if resolver != nil || !IsError(err, test.wantCode) {
				t.Fatalf("New() = %#v, %v; want %s", resolver, err, test.wantCode)
			}
		})
	}
}

func TestNewRejectsUnsafeFileMetadata(t *testing.T) {
	owner := uint32(os.Geteuid())
	t.Run("digest mismatch", func(t *testing.T) {
		ref, path, raw, _ := writeValidPolicy(t, "digest")
		wrong := sha256.Sum256([]byte("different"))
		_, err := New(ref, path, hex.EncodeToString(wrong[:]), owner, int64(len(raw)))
		assertAdapterError(t, err, CodeDigestMismatch, path)
	})
	t.Run("bounded read", func(t *testing.T) {
		ref, path, raw, digest := writeValidPolicy(t, "bounded")
		_, err := New(ref, path, digest, owner, int64(len(raw)-1))
		assertAdapterError(t, err, CodeFileUnsafe, path)
	})
	t.Run("empty", func(t *testing.T) {
		root := t.TempDir()
		path := filepath.Join(root, "empty.json")
		if err := os.WriteFile(path, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		ref, _ := application.NewEgressPolicyRef("egreso:empty")
		digest := sha256.Sum256(nil)
		_, err := New(ref, path, hex.EncodeToString(digest[:]), owner, 1)
		assertAdapterError(t, err, CodeFileUnsafe, path)
	})
	t.Run("group writable", func(t *testing.T) {
		ref, path, raw, digest := writeValidPolicy(t, "group-write")
		if err := os.Chmod(path, 0o660); err != nil {
			t.Fatal(err)
		}
		_, err := New(ref, path, digest, owner, int64(len(raw)))
		assertAdapterError(t, err, CodeFileUnsafe, path)
	})
	t.Run("other writable", func(t *testing.T) {
		ref, path, raw, digest := writeValidPolicy(t, "other-write")
		if err := os.Chmod(path, 0o602); err != nil {
			t.Fatal(err)
		}
		_, err := New(ref, path, digest, owner, int64(len(raw)))
		assertAdapterError(t, err, CodeFileUnsafe, path)
	})
	t.Run("wrong owner", func(t *testing.T) {
		ref, path, raw, digest := writeValidPolicy(t, "owner")
		_, err := New(ref, path, digest, owner+1, int64(len(raw)))
		assertAdapterError(t, err, CodeFileUnsafe, path)
	})
	t.Run("hardlink", func(t *testing.T) {
		ref, path, raw, digest := writeValidPolicy(t, "hardlink")
		if err := os.Link(path, path+".link"); err != nil {
			t.Fatal(err)
		}
		_, err := New(ref, path, digest, owner, int64(len(raw)))
		assertAdapterError(t, err, CodeFileUnsafe, path)
	})
	t.Run("file symlink", func(t *testing.T) {
		ref, path, raw, digest := writeValidPolicy(t, "symlink")
		link := path + ".symlink"
		if err := os.Symlink(path, link); err != nil {
			t.Fatal(err)
		}
		_, err := New(ref, link, digest, owner, int64(len(raw)))
		assertAdapterError(t, err, CodeFileUnsafe, link)
	})
	t.Run("parent symlink", func(t *testing.T) {
		ref, path, raw, digest := writeValidPolicy(t, "parent-symlink")
		link := filepath.Join(t.TempDir(), "linked-parent")
		if err := os.Symlink(filepath.Dir(path), link); err != nil {
			t.Fatal(err)
		}
		linkedPath := filepath.Join(link, filepath.Base(path))
		_, err := New(ref, linkedPath, digest, owner, int64(len(raw)))
		assertAdapterError(t, err, CodeFileUnsafe, linkedPath)
	})
	t.Run("directory", func(t *testing.T) {
		ref, _, raw, digest := writeValidPolicy(t, "directory")
		path := t.TempDir()
		_, err := New(ref, path, digest, owner, int64(len(raw)))
		assertAdapterError(t, err, CodeFileUnsafe, path)
	})
}

func TestNewUsesStrictPublicEgressDecoder(t *testing.T) {
	owner := uint32(os.Geteuid())
	tests := []struct {
		name     string
		ref      string
		payload  []byte
		wantCode string
	}{
		{"invalid UTF-8", "egreso:utf8", []byte{'{', '"', 0xff, '"', '}'}, CodePayloadInvalid},
		{"field casing", "egreso:casing", []byte(`{"Esquema":"agentmicrovm.concesion-egreso.v1","referencia":"egreso:casing","destinos":[],"maximo_conexiones":1,"limite_tiempo_ms":1,"limite_subida_bytes":1,"limite_bajada_bytes":1}`), CodePayloadInvalid},
		{"unknown field", "egreso:unknown", []byte(`{"esquema":"agentmicrovm.concesion-egreso.v1","referencia":"egreso:unknown","destinos":[],"maximo_conexiones":1,"limite_tiempo_ms":1,"limite_subida_bytes":1,"limite_bajada_bytes":1,"extra":true}`), CodePayloadInvalid},
		{"duplicate field", "egreso:duplicate", []byte(`{"esquema":"agentmicrovm.concesion-egreso.v1","referencia":"egreso:duplicate","referencia":"egreso:duplicate","destinos":[],"maximo_conexiones":1,"limite_tiempo_ms":1,"limite_subida_bytes":1,"limite_bajada_bytes":1}`), CodePayloadInvalid},
		{"trailing root", "egreso:trailing", []byte(`{"esquema":"agentmicrovm.concesion-egreso.v1","referencia":"egreso:trailing","destinos":[],"maximo_conexiones":1,"limite_tiempo_ms":1,"limite_subida_bytes":1,"limite_bajada_bytes":1} true`), CodePayloadInvalid},
		{"wrong schema", "egreso:schema", []byte(`{"esquema":"agentmicrovm.concesion-egreso.v0","referencia":"egreso:schema","destinos":[],"maximo_conexiones":1,"limite_tiempo_ms":1,"limite_subida_bytes":1,"limite_bajada_bytes":1}`), CodePayloadInvalid},
		{"mismatched internal ref", "egreso:outer", []byte(`{"esquema":"agentmicrovm.concesion-egreso.v1","referencia":"egreso:inner","destinos":[],"maximo_conexiones":1,"limite_tiempo_ms":1,"limite_subida_bytes":1,"limite_bajada_bytes":1}`), CodePolicyRefMismatch},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, "policy.json")
			if err := os.WriteFile(path, test.payload, 0o600); err != nil {
				t.Fatal(err)
			}
			ref, err := application.NewEgressPolicyRef(test.ref)
			if err != nil {
				t.Fatal(err)
			}
			digest := sha256.Sum256(test.payload)
			_, err = New(ref, path, hex.EncodeToString(digest[:]), owner, maximumCanonicalPayloadBytes)
			assertAdapterError(t, err, test.wantCode, path)
		})
	}
}

func TestNewPreservesDecoderAcceptedTrailingWhitespace(t *testing.T) {
	ref, path, raw, _ := writeValidPolicy(t, "trailing-whitespace")
	raw = append(raw, '\n', '\t')
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(raw)
	wantDigest := hex.EncodeToString(digest[:])
	resolver, err := New(ref, path, wantDigest, uint32(os.Geteuid()), int64(len(raw)))
	if err != nil {
		t.Fatalf("New() rejected public decoder whitespace: %v", err)
	}
	authority, err := resolver.ResolveEgressPolicy(context.Background(), ref)
	if err != nil || authority.CanonicalPayload != string(raw) || authority.PayloadSHA256 != wantDigest {
		t.Fatalf("authority = %+v, %v; exact trailing bytes were not preserved", authority, err)
	}
}

func TestNewDetectsFileMutationAndReplacementDuringLoad(t *testing.T) {
	owner := uint32(os.Geteuid())
	t.Run("metadata changes after identity", func(t *testing.T) {
		ref, path, raw, digest := writeValidPolicy(t, "race-mode")
		_, err := newWithHooks(ref, path, digest, owner, int64(len(raw)), loaderHooks{
			afterFirstIdentity: func() {
				if chmodErr := os.Chmod(path, 0o660); chmodErr != nil {
					t.Fatal(chmodErr)
				}
			},
		})
		assertAdapterError(t, err, CodeFileUnsafe, path)
	})
	t.Run("path replacement after first identity", func(t *testing.T) {
		ref, path, raw, digest := writeValidPolicy(t, "race-replace-before-read")
		replacement := path + ".replacement"
		if err := os.WriteFile(replacement, raw, 0o600); err != nil {
			t.Fatal(err)
		}
		_, err := newWithHooks(ref, path, digest, owner, int64(len(raw)), loaderHooks{
			afterFirstIdentity: func() {
				if renameErr := os.Rename(replacement, path); renameErr != nil {
					t.Fatal(renameErr)
				}
			},
		})
		assertAdapterError(t, err, CodeFileUnsafe, path)
	})
	t.Run("hardlink appears after identity", func(t *testing.T) {
		ref, path, raw, digest := writeValidPolicy(t, "race-link")
		_, err := newWithHooks(ref, path, digest, owner, int64(len(raw)), loaderHooks{
			afterFirstIdentity: func() {
				if linkErr := os.Link(path, path+".link"); linkErr != nil {
					t.Fatal(linkErr)
				}
			},
		})
		assertAdapterError(t, err, CodeFileUnsafe, path)
	})
	t.Run("path replacement after first read", func(t *testing.T) {
		ref, path, raw, digest := writeValidPolicy(t, "race-replace")
		replacement := path + ".replacement"
		if err := os.WriteFile(replacement, raw, 0o600); err != nil {
			t.Fatal(err)
		}
		_, err := newWithHooks(ref, path, digest, owner, int64(len(raw)), loaderHooks{
			afterFirstRead: func() {
				if renameErr := os.Rename(replacement, path); renameErr != nil {
					t.Fatal(renameErr)
				}
			},
		})
		assertAdapterError(t, err, CodeFileUnsafe, path)
	})
}

type typedNilContext struct{}

func (*typedNilContext) Deadline() (time.Time, bool) { panic("typed nil context used") }
func (*typedNilContext) Done() <-chan struct{}       { panic("typed nil context used") }
func (*typedNilContext) Err() error                  { panic("typed nil context used") }
func (*typedNilContext) Value(any) any               { panic("typed nil context used") }

func writeValidPolicy(t *testing.T, suffix string) (application.EgressPolicyRef, string, []byte, string) {
	t.Helper()
	ref, err := application.NewEgressPolicyRef("egreso:" + suffix)
	if err != nil {
		t.Fatal(err)
	}
	grant := microvm.ConcesionEgreso{
		Esquema: microvm.EsquemaConcesionEgreso, Referencia: ref.String(),
		Destinos:         []microvm.DestinoEgreso{{Host: "api.openai.com", Puertos: []uint16{443}}},
		MaximoConexiones: 4, LimiteTiempoMS: 60_000,
		LimiteSubidaBytes: 1 << 20, LimiteBajadaBytes: 8 << 20,
	}
	raw, err := json.Marshal(grant)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	path := filepath.Join(root, "policy.json")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(raw)
	return ref, path, raw, hex.EncodeToString(digest[:])
}

func assertAdapterError(t *testing.T, err error, code, sensitivePath string) {
	t.Helper()
	if !IsError(err, code) {
		t.Fatalf("error = %v; want %s", err, code)
	}
	if strings.Contains(err.Error(), sensitivePath) {
		t.Fatalf("error leaks local path: %q", err)
	}
}
