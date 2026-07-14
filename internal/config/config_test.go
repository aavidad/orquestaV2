package config

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestLoadReturnsTypedCanonicalDefaults(t *testing.T) {
	snapshot, err := loadWithEnvironment(LoadOptions{}, nil)
	if err != nil {
		t.Fatalf("load defaults: %v", err)
	}
	if snapshot.Server.Listen != "127.0.0.1:8080" || snapshot.Server.MCPPath != "/mcp" {
		t.Fatalf("unexpected server defaults: %+v", snapshot.Server)
	}
	if snapshot.Server.MaxRequestBytes != 1048576 {
		t.Fatalf("server max request bytes = %d, want 1048576", snapshot.Server.MaxRequestBytes)
	}
	if snapshot.State.SQLite.BusyTimeout != 5*time.Second {
		t.Fatalf("unexpected sqlite busy timeout: %s", snapshot.State.SQLite.BusyTimeout)
	}
	if snapshot.State.SQLite.MaxOpenConnections != 8 {
		t.Fatalf("sqlite max open connections = %d, want 8", snapshot.State.SQLite.MaxOpenConnections)
	}
	if snapshot.Identity.LocalActor != "actor:local-owner" || snapshot.Identity.LocalTokenPath != "./var/secrets/local-owner.token" || snapshot.Project.Default != "project:default" {
		t.Fatalf("identity/project refs missing: %+v %+v", snapshot.Identity, snapshot.Project)
	}
	if snapshot.Runtime.MaxOutputBytes != 1048576 || snapshot.Runtime.Codex.MaxDiagnosticBytes != 65536 ||
		snapshot.Runtime.Codex.MaxConcurrentExecutions != 70 || snapshot.Runtime.Codex.ProcessPipeDrainDelay != 250*time.Millisecond {
		t.Fatalf("unexpected runtime limits: %+v", snapshot.Runtime)
	}
	if snapshot.Effective.MaxExistingBytes != 16777216 {
		t.Fatalf("unexpected effective input limit: %+v", snapshot.Effective)
	}
	if snapshot.Scheduler.ObservationInterval != 2*time.Second ||
		snapshot.Scheduler.ClaimLease != 2*time.Minute ||
		snapshot.Scheduler.MaxExecutionAttempts != 3 ||
		snapshot.Scheduler.ExecutionTimeout != 45*time.Minute {
		t.Fatalf("unexpected scheduler defaults: %+v", snapshot.Scheduler)
	}
	if snapshot.API.MaxListLimit != 100 || snapshot.API.Locale != "es" {
		t.Fatalf("unexpected API defaults: %+v", snapshot.API)
	}
	if !strings.HasPrefix(snapshot.Hash, "sha256:") || len(snapshot.Hash) != len("sha256:")+64 {
		t.Fatalf("unexpected snapshot hash: %q", snapshot.Hash)
	}
	if snapshot.RegistryRevision != generatedRegistryRevision {
		t.Fatalf("revision = %q, want %q", snapshot.RegistryRevision, generatedRegistryRevision)
	}
	if snapshot.SchemaVersion != 1 {
		t.Fatalf("schema version = %d, want 1", snapshot.SchemaVersion)
	}
	for _, key := range allGeneratedKeys() {
		metadata, found := snapshot.Metadata(key)
		if !found {
			t.Fatalf("generated key %q missing from snapshot", key)
		}
		if metadata.Source != SourceDefault {
			t.Fatalf("source for %q = %q, want default", key, metadata.Source)
		}
	}
}

func TestLoadRejectsUnknownAndDuplicateTOML(t *testing.T) {
	t.Run("unknown key", func(t *testing.T) {
		path := writeTOML(t, "[server]\nunknown = true\n")
		_, err := loadWithEnvironment(LoadOptions{FilePath: path}, nil)
		assertConfigError(t, err, ErrorUnknownKey, Key("server.unknown"))
	})

	t.Run("unknown empty table", func(t *testing.T) {
		path := writeTOML(t, "[runtime.unknown]\n")
		_, err := loadWithEnvironment(LoadOptions{FilePath: path}, nil)
		assertConfigError(t, err, ErrorUnknownKey, Key("runtime.unknown"))
	})

	t.Run("duplicate key", func(t *testing.T) {
		path := writeTOML(t, "server.listen = \"first\"\nserver.listen = \"second\"\n")
		_, err := loadWithEnvironment(LoadOptions{FilePath: path}, nil)
		assertConfigError(t, err, ErrorFileInvalid, "")
	})
}

func TestLoadPrecedenceDefaultFileEnvironment(t *testing.T) {
	path := writeTOML(t, `
[server]
listen = "127.0.0.1:9090"
read_timeout = "21s"

[runtime.codex]
model = "file-model"
env_allowlist = ["FILE_ONLY"]
`)
	environment := map[string]string{
		"ORQUESTA_SERVER_LISTEN":               "127.0.0.1:9191",
		"ORQUESTA_RUNTIME_CODEX_ENV_ALLOWLIST": "PATH, CODEX_HOME, EXTRA_ALLOWED",
	}
	snapshot, err := loadWithEnvironment(LoadOptions{FilePath: path}, mapEnvironment(environment))
	if err != nil {
		t.Fatalf("load layered config: %v", err)
	}
	if snapshot.Server.Listen != "127.0.0.1:9191" {
		t.Fatalf("env did not win: %q", snapshot.Server.Listen)
	}
	if snapshot.Server.ReadTimeout != 21*time.Second || snapshot.Runtime.Codex.Model != "file-model" {
		t.Fatalf("file values missing: %+v %+v", snapshot.Server, snapshot.Runtime.Codex)
	}
	if snapshot.Project.Default != "project:default" {
		t.Fatalf("default missing: %q", snapshot.Project.Default)
	}
	if want := []string{"PATH", "CODEX_HOME", "EXTRA_ALLOWED"}; !reflect.DeepEqual(snapshot.Runtime.Codex.EnvAllowlist, want) {
		t.Fatalf("env allowlist = %#v, want %#v", snapshot.Runtime.Codex.EnvAllowlist, want)
	}
	assertSource(t, snapshot, KeyServerListen, SourceEnv)
	assertSource(t, snapshot, KeyServerReadTimeout, SourceFile)
	assertSource(t, snapshot, KeyProjectDefault, SourceDefault)
}

func TestEffectiveConfigRedactsCredentialAndIsOutputOnly(t *testing.T) {
	effectivePath := filepath.Join(t.TempDir(), "effective_config.json")
	path := writeTOML(t, `
[runtime.codex]
credential_ref = "credential:must-not-leak"

[config]
effective_path = "`+effectivePath+`"
`)
	snapshot, err := loadWithEnvironment(LoadOptions{FilePath: path}, nil)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	effective, err := snapshot.EffectiveJSON()
	if err != nil {
		t.Fatalf("effective json: %v", err)
	}
	assertRedacted(t, effective)
	marshaled, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatalf("marshal snapshot: %v", err)
	}
	assertRedacted(t, marshaled)
	if err := snapshot.WriteEffective(); err != nil {
		t.Fatalf("write effective: %v cause=%v max=%d bytes=%d", err, errors.Unwrap(err), snapshot.Effective.MaxExistingBytes, len(effective))
	}
	if err := snapshot.WriteEffective(); err != nil {
		t.Fatalf("rewrite recognized effective document: %v", err)
	}
	written, err := os.ReadFile(effectivePath)
	if err != nil {
		t.Fatalf("read effective: %v", err)
	}
	assertRedacted(t, written)
	if !strings.Contains(string(written), `"document_type": "orquesta.effective_config"`) {
		t.Fatalf("effective output lacks ownership marker: %s", written)
	}
	info, err := os.Stat(effectivePath)
	if err != nil {
		t.Fatalf("stat effective: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("effective mode = %o, want 600", info.Mode().Perm())
	}

	_, err = loadWithEnvironment(LoadOptions{FilePath: effectivePath}, nil)
	assertConfigError(t, err, ErrorEffectiveInputForbidden, "")
}

func TestSensitiveCredentialReferenceCannotBeGuessedThroughSnapshotHash(t *testing.T) {
	firstPath := writeTOML(t, "[runtime.codex]\ncredential_ref = \"credential:short-one\"\n")
	secondPath := writeTOML(t, "[runtime.codex]\ncredential_ref = \"credential:short-two\"\n")
	first, err := loadWithEnvironment(LoadOptions{FilePath: firstPath}, nil)
	if err != nil {
		t.Fatalf("load first: %v", err)
	}
	second, err := loadWithEnvironment(LoadOptions{FilePath: secondPath}, nil)
	if err != nil {
		t.Fatalf("load second: %v", err)
	}
	if first.Runtime.Codex.CredentialRef == second.Runtime.Codex.CredentialRef {
		t.Fatal("credential fixture did not differ")
	}
	if first.Hash != second.Hash {
		t.Fatalf("public snapshot hash acts as sensitive-value oracle: %q != %q", first.Hash, second.Hash)
	}
	firstEffective, _ := first.EffectiveJSON()
	secondEffective, _ := second.EffectiveJSON()
	if !bytes.Equal(firstEffective, secondEffective) {
		t.Fatal("redacted effective output differs by sensitive value")
	}
}

func TestWriteEffectiveNeverReplacesUnknownOrSymlinkDestination(t *testing.T) {
	t.Run("unknown private file", func(t *testing.T) {
		root := t.TempDir()
		destination := filepath.Join(root, "owned-by-someone-else.json")
		want := []byte(`{"owner":"other"}`)
		if err := os.WriteFile(destination, want, 0o600); err != nil {
			t.Fatalf("write sentinel: %v", err)
		}
		path := writeTOML(t, "[config]\neffective_path = "+strconv.Quote(destination)+"\n")
		snapshot, err := loadWithEnvironment(LoadOptions{FilePath: path}, nil)
		if err != nil {
			t.Fatalf("load: %v", err)
		}
		if err := snapshot.WriteEffective(); err == nil {
			t.Fatal("unknown destination was replaced")
		}
		got, err := os.ReadFile(destination)
		if err != nil || !bytes.Equal(got, want) {
			t.Fatalf("sentinel changed: %q err=%v", got, err)
		}
	})

	t.Run("symlink", func(t *testing.T) {
		root := t.TempDir()
		target := filepath.Join(root, "target.json")
		want := []byte(`{"owner":"target"}`)
		if err := os.WriteFile(target, want, 0o600); err != nil {
			t.Fatalf("write target: %v", err)
		}
		destination := filepath.Join(root, "effective.json")
		if err := os.Symlink(target, destination); err != nil {
			t.Skipf("symlink unsupported: %v", err)
		}
		path := writeTOML(t, "[config]\neffective_path = "+strconv.Quote(destination)+"\n")
		snapshot, err := loadWithEnvironment(LoadOptions{FilePath: path}, nil)
		if err != nil {
			t.Fatalf("load: %v", err)
		}
		if err := snapshot.WriteEffective(); err == nil {
			t.Fatal("symlink destination was replaced")
		}
		got, err := os.ReadFile(target)
		if err != nil || !bytes.Equal(got, want) {
			t.Fatalf("symlink target changed: %q err=%v", got, err)
		}
	})
}

func TestWriteEffectiveRejectsCanonicalLimitBelowGeneratedProjection(t *testing.T) {
	root := t.TempDir()
	destination := filepath.Join(root, "effective.json")
	path := writeTOML(t, "[config]\neffective_path = "+strconv.Quote(destination)+"\neffective_max_existing_bytes = 1024\n")
	snapshot, err := loadWithEnvironment(LoadOptions{FilePath: path}, nil)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if err := snapshot.WriteEffective(); err == nil {
		t.Fatal("oversize generated effective projection was written")
	}
	if _, err := os.Lstat(destination); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("oversize destination exists: %v", err)
	}
}

func TestSnapshotHashAndEffectiveJSONAreStable(t *testing.T) {
	first := writeTOML(t, `
[server]
listen = "127.0.0.1:7777"
read_timeout = "22s"

[runtime.codex]
model = "stable-model"
`)
	second := writeTOML(t, `
[runtime.codex]
model = "stable-model"

[server]
read_timeout = "22s"
listen = "127.0.0.1:7777"
`)
	firstSnapshot, err := loadWithEnvironment(LoadOptions{FilePath: first}, nil)
	if err != nil {
		t.Fatalf("load first: %v", err)
	}
	secondSnapshot, err := loadWithEnvironment(LoadOptions{FilePath: second}, nil)
	if err != nil {
		t.Fatalf("load second: %v", err)
	}
	if firstSnapshot.Hash != secondSnapshot.Hash {
		t.Fatalf("hash changed with TOML order: %q != %q", firstSnapshot.Hash, secondSnapshot.Hash)
	}
	firstEffective, _ := firstSnapshot.EffectiveJSON()
	secondEffective, _ := secondSnapshot.EffectiveJSON()
	if !reflect.DeepEqual(firstEffective, secondEffective) {
		t.Fatalf("effective config changed with TOML order")
	}
}

func TestCanonicalRegistryAndGeneratedKeysStaySynchronized(t *testing.T) {
	registryPath := filepath.Join("..", "..", "config", "registry.json")
	source, err := os.ReadFile(registryPath)
	if err != nil {
		t.Fatalf("read registry: %v", err)
	}
	digest := sha256.Sum256(source)
	if got := hex.EncodeToString(digest[:]); got != generatedRegistrySourceSHA256 {
		t.Fatalf("registry changed without go generate: got %s, generated %s", got, generatedRegistrySourceSHA256)
	}
	if string(source) != generatedRegistryJSON {
		t.Fatalf("embedded generated registry differs from canonical registry")
	}

	registry, err := loadRegistry()
	if err != nil {
		t.Fatalf("load generated registry: %v", err)
	}
	generatedKeys := allGeneratedKeys()
	if len(registry.keys) != len(generatedKeys) {
		t.Fatalf("registry keys = %d, generated = %d", len(registry.keys), len(generatedKeys))
	}
	for index, definition := range registry.keys {
		if definition.Key != generatedKeys[index] {
			t.Fatalf("generated key %d = %q, want %q", index, generatedKeys[index], definition.Key)
		}
	}
}

func TestExampleIsCompleteAndContainsNoSecretValues(t *testing.T) {
	registry, err := loadRegistry()
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}
	examplePath := filepath.Join("..", "..", "config", "orquesta.toml.example")
	values, err := loadTOMLFile(examplePath, registry)
	if err != nil {
		t.Fatalf("load example: %v", err)
	}
	if len(values) != len(registry.keys) {
		t.Fatalf("example has %d explicit keys, want %d", len(values), len(registry.keys))
	}
	for _, definition := range registry.keys {
		if _, found := values[definition.Key]; !found {
			t.Fatalf("example omits %q", definition.Key)
		}
		if definition.Sensitive && definition.Type != valueTypeCredentialRef {
			t.Fatalf("sensitive key %q accepts secret material", definition.Key)
		}
	}
	snapshot, err := loadWithEnvironment(LoadOptions{FilePath: examplePath}, nil)
	if err != nil {
		t.Fatalf("resolve example: %v", err)
	}
	if snapshot.Runtime.Codex.Model != "" || snapshot.Runtime.Codex.CredentialRef != "" {
		t.Fatalf("example must not invent model or credential selection")
	}
}

func TestProcessEnvironmentReadLivesOnlyInDedicatedLoader(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	needle := "os." + "LookupEnv"
	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		if strings.Contains(string(content), needle) && filepath.Base(path) != "env_loader.go" {
			t.Fatalf("process environment read outside env_loader.go: %s", path)
		}
	}
}

func writeTOML(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "orquesta.toml")
	if err := os.WriteFile(path, []byte(strings.TrimSpace(content)+"\n"), 0o600); err != nil {
		t.Fatalf("write TOML: %v", err)
	}
	return path
}

func mapEnvironment(values map[string]string) environmentLookup {
	return func(name string) (string, bool) {
		value, found := values[name]
		return value, found
	}
}

func assertConfigError(t *testing.T, err error, code ErrorCode, key Key) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected %s", code)
	}
	var configError *Error
	if !errors.As(err, &configError) {
		t.Fatalf("error type = %T, want *config.Error: %v", err, err)
	}
	if configError.Code != code || configError.Key != key {
		t.Fatalf("error = (%s, %s), want (%s, %s)", configError.Code, configError.Key, code, key)
	}
}

func assertSource(t *testing.T, snapshot Snapshot, key Key, want Source) {
	t.Helper()
	metadata, found := snapshot.Metadata(key)
	if !found {
		t.Fatalf("missing metadata for %q", key)
	}
	if metadata.Source != want {
		t.Fatalf("source for %q = %q, want %q", key, metadata.Source, want)
	}
}

func assertRedacted(t *testing.T, content []byte) {
	t.Helper()
	if strings.Contains(string(content), "credential:must-not-leak") {
		t.Fatalf("effective config leaked credential ref: %s", content)
	}
	if !strings.Contains(string(content), redactedValue) {
		t.Fatalf("effective config lacks redaction marker: %s", content)
	}
}
