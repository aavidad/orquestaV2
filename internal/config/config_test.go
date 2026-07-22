package config

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestResolveReturnsImmutableTypedCanonicalDefaults(t *testing.T) {
	snapshot := resolveTOML(t, "", nil)
	if snapshot.ServerListen() != "127.0.0.1:8080" || snapshot.ServerMCPPath() != "/mcp" ||
		snapshot.ServerMaxRequestBytes() != 1048576 || snapshot.StateSQLiteBusyTimeout() != 5*time.Second ||
		snapshot.StateSQLiteMaxOpenConnections() != 8 {
		t.Fatal("canonical server/state defaults missing")
	}
	if snapshot.IdentityProvider() != "local_token" || snapshot.IdentityLocalActor() != "actor:local-owner" ||
		snapshot.IdentityLocalTokenPath() != "./var/secrets/local-owner.token" ||
		snapshot.IdentityOIDCIssuer() != "" || snapshot.IdentityOIDCAudience() != "" ||
		len(snapshot.IdentityOIDCRequiredGroups()) != 0 || snapshot.IdentityOIDCClockSkew() != 30*time.Second ||
		snapshot.IdentityOIDCUpstreamTimeout() != 10*time.Second ||
		snapshot.ProjectDefault() != "project:default" || snapshot.DirectorLeaseDuration() != 2*time.Minute {
		t.Fatal("identity/project defaults missing")
	}
	if snapshot.CredentialsLocalPath() != "./var/secrets/credentials.json" ||
		snapshot.CredentialsLocalMaxDocumentBytes() != 1048576 {
		t.Fatal("credential-store defaults missing")
	}
	if snapshot.RuntimeMaxOutputBytes() != 1048576 || snapshot.RuntimeCodexMaxDiagnosticBytes() != 65536 ||
		snapshot.RuntimeCodexMaxConcurrentExecutions() != 70 || snapshot.RuntimeCodexProcessPipeDrainDelay() != 250*time.Millisecond {
		t.Fatal("runtime defaults missing")
	}
	if snapshot.GovernanceBudgetCurrency() != "USD" || snapshot.GovernanceGlobalTokenBudget() != 14000000 ||
		snapshot.GovernanceGlobalMoneyMicrosBudget() != 70000000 ||
		snapshot.GovernanceDefaultExecutionTokenBudget() != 200000 ||
		snapshot.GovernanceDefaultExecutionMoneyMicrosBudget() != 1000000 ||
		snapshot.GovernanceEffectApprovalTTL() != 24*time.Hour {
		t.Fatal("governance defaults missing")
	}
	if snapshot.ConfigEffectiveMaxExistingBytes() != 16777216 || snapshot.SchedulerObservationInterval() != 2*time.Second ||
		snapshot.SchedulerClaimLease() != 2*time.Minute || snapshot.SchedulerMaxExecutionAttempts() != 3 ||
		snapshot.SchedulerMaxChildrenPerParent() != 6 || snapshot.SchedulerExecutionTimeout() != 45*time.Minute ||
		snapshot.MailboxMaxEnvelopeBytes() != 65536 ||
		snapshot.APIMaxListLimit() != 100 || snapshot.APILocale() != "es" {
		t.Fatal("scheduler/API/effective defaults missing")
	}
	if !strings.HasPrefix(snapshot.Hash(), "sha256:") || len(snapshot.Hash()) != len("sha256:")+64 ||
		snapshot.RegistryHash() != generatedRegistrySemanticSHA256 || snapshot.RegistryRevision() != generatedRegistryRevision ||
		snapshot.SchemaVersion() != 2 {
		t.Fatalf("snapshot identity invalid: %s %s %s %d", snapshot.Hash(), snapshot.RegistryHash(), snapshot.RegistryRevision(), snapshot.SchemaVersion())
	}

	beforeHash := snapshot.Hash()
	beforeEffective, _ := snapshot.EffectiveJSON()
	allowlist := snapshot.RuntimeCodexEnvAllowlist()
	allowlist[0] = "MUTATED"
	metadata, _ := snapshot.Metadata(KeyRuntimeCodexEnvAllowlist)
	metadata.ValidatorIDs[0] = "mutated"
	definition, _ := Definition(KeyRuntimeCodexEnvAllowlist)
	definition.Default.([]string)[0] = "MUTATED"
	if got := snapshot.RuntimeCodexEnvAllowlist()[0]; got != "PATH" || snapshot.Hash() != beforeHash {
		t.Fatalf("returned slice mutated snapshot: %q %q", got, snapshot.Hash())
	}
	afterEffective, _ := snapshot.EffectiveJSON()
	if !bytes.Equal(beforeEffective, afterEffective) {
		t.Fatal("detached values mutated effective projection")
	}
	for _, key := range allGeneratedKeys() {
		metadata, found := snapshot.Metadata(key)
		if !found || metadata.Source != SourceDefault {
			t.Fatalf("metadata for %q = %+v/%v", key, metadata, found)
		}
	}
}

func TestParseExplicitRejectsUnknownDuplicateAndOversizeTOML(t *testing.T) {
	_, err := ParseExplicit([]byte("[server]\nunknown = true\n"))
	assertConfigError(t, err, ErrorUnknownKey, Key("server.unknown"))
	_, err = ParseExplicit([]byte("[runtime.unknown]\n"))
	assertConfigError(t, err, ErrorUnknownKey, Key("runtime.unknown"))
	_, err = ParseExplicit([]byte("server.listen = \"first\"\nserver.listen = \"second\"\n"))
	assertConfigError(t, err, ErrorFileInvalid, "")
	_, err = ParseExplicit(bytes.Repeat([]byte{'#'}, int(SourceMaxBytes())+1))
	assertConfigError(t, err, ErrorFileInvalid, "")
}

func TestResolvePrecedenceAndCapturedEnvironment(t *testing.T) {
	snapshot := resolveTOML(t, `
[server]
listen = "127.0.0.1:9090"
read_timeout = "21s"

[runtime.codex]
model = "file-model"
env_allowlist = ["FILE_ONLY"]
`, map[string]string{
		"ORQUESTA_SERVER_LISTEN":               "127.0.0.1:9191",
		"ORQUESTA_RUNTIME_CODEX_ENV_ALLOWLIST": "PATH, CODEX_HOME, EXTRA_ALLOWED",
	})
	if snapshot.ServerListen() != "127.0.0.1:9191" || snapshot.ServerReadTimeout() != 21*time.Second ||
		snapshot.RuntimeCodexModel() != "file-model" || snapshot.ProjectDefault() != "project:default" {
		t.Fatal("default < file < env precedence failed")
	}
	want := []string{"PATH", "CODEX_HOME", "EXTRA_ALLOWED"}
	if !reflect.DeepEqual(snapshot.RuntimeCodexEnvAllowlist(), want) {
		t.Fatalf("env allowlist = %#v, want %#v", snapshot.RuntimeCodexEnvAllowlist(), want)
	}
	assertSource(t, snapshot, KeyServerListen, SourceEnv)
	assertSource(t, snapshot, KeyServerReadTimeout, SourceFile)
	assertSource(t, snapshot, KeyProjectDefault, SourceDefault)
	_, err := Resolve(ResolveOptions{Environment: map[string]string{"UNDECLARED": "value"}})
	assertConfigError(t, err, ErrorUnknownKey, Key("UNDECLARED"))
}

func TestOIDCAndDirectorSettingsResolveOnlyThroughCanonicalRegistry(t *testing.T) {
	snapshot := resolveTOML(t, `
[identity]
provider = "oidc"

[identity.oidc]
issuer = "https://idp.example.test/tenant/v2.0"
audience = "orquesta"
required_groups = ["orquesta-users", "operators"]
clock_skew = "45s"
upstream_timeout = "12s"

[director]
lease_duration = "90s"
`, nil)
	if snapshot.IdentityProvider() != "oidc" ||
		snapshot.IdentityOIDCIssuer() != "https://idp.example.test/tenant/v2.0" ||
		snapshot.IdentityOIDCAudience() != "orquesta" || snapshot.IdentityOIDCClockSkew() != 45*time.Second ||
		snapshot.IdentityOIDCUpstreamTimeout() != 12*time.Second ||
		snapshot.DirectorLeaseDuration() != 90*time.Second ||
		!reflect.DeepEqual(snapshot.IdentityOIDCRequiredGroups(), []string{"orquesta-users", "operators"}) {
		t.Fatalf("OIDC/director canonical settings lost: %+v", snapshot)
	}
	groups := snapshot.IdentityOIDCRequiredGroups()
	groups[0] = "mutated"
	if snapshot.IdentityOIDCRequiredGroups()[0] != "orquesta-users" {
		t.Fatal("OIDC group getter leaks mutable registry state")
	}
}

func TestEffectiveConfigProjectionIsPureRedactedAndOutputOnly(t *testing.T) {
	snapshot := resolveTOML(t, `
[runtime.codex]
credential_ref = "credential:must-not-leak"
`, nil)
	effective, err := snapshot.EffectiveJSON()
	if err != nil {
		t.Fatalf("effective JSON: %v", err)
	}
	assertRedacted(t, effective)
	marshaled, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatalf("marshal snapshot: %v", err)
	}
	assertRedacted(t, marshaled)
	_, err = ParseExplicit(effective)
	assertConfigError(t, err, ErrorEffectiveInputForbidden, "")
}

func TestCredentialReferenceIsCanonicalAndCannotBecomeHashOracle(t *testing.T) {
	first := resolveTOML(t, "[runtime.codex]\ncredential_ref = \"credential:short-one\"", nil)
	second := resolveTOML(t, "[runtime.codex]\ncredential_ref = \"credential:short-two\"", nil)
	if first.RuntimeCodexCredentialRef() == second.RuntimeCodexCredentialRef() || first.Hash() != second.Hash() {
		t.Fatal("credential refs differ but redacted snapshot hash must not")
	}
	firstEffective, _ := first.EffectiveJSON()
	secondEffective, _ := second.EffectiveJSON()
	if !bytes.Equal(firstEffective, secondEffective) {
		t.Fatal("effective output acts as credential-reference oracle")
	}
	for _, invalid := range []string{"secret", "sk-live-value", "credential:", "credential:UPPER", "credential:.leading"} {
		_, err := Resolve(ResolveOptions{TOML: []byte("[runtime.codex]\ncredential_ref = " + strconv.Quote(invalid))})
		assertConfigError(t, err, ErrorValueInvalid, KeyRuntimeCodexCredentialRef)
	}
}

func TestRenderExplicitIsCanonicalAndSnapshotStable(t *testing.T) {
	first := []byte("[server]\nlisten = \"127.0.0.1:7777\"\nread_timeout = \"22s\"\n[runtime.codex]\nmodel = \"stable-model\"\n")
	second := []byte("[runtime.codex]\nmodel = \"stable-model\"\n[server]\nread_timeout = \"22s\"\nlisten = \"127.0.0.1:7777\"\n")
	firstSnapshot, err := Resolve(ResolveOptions{TOML: first})
	if err != nil {
		t.Fatal(err)
	}
	secondSnapshot, err := Resolve(ResolveOptions{TOML: second})
	if err != nil {
		t.Fatal(err)
	}
	if firstSnapshot.Hash() != secondSnapshot.Hash() {
		t.Fatal("TOML order changed hash")
	}
	firstEffective, _ := firstSnapshot.EffectiveJSON()
	secondEffective, _ := secondSnapshot.EffectiveJSON()
	if !bytes.Equal(firstEffective, secondEffective) {
		t.Fatal("TOML order changed effective output")
	}
	values, err := ParseExplicit(second)
	if err != nil {
		t.Fatal(err)
	}
	rendered, err := RenderExplicit(values)
	if err != nil {
		t.Fatal(err)
	}
	valuesAgain, err := ParseExplicit(rendered)
	if err != nil || !reflect.DeepEqual(values, valuesAgain) {
		t.Fatalf("roundtrip = %#v/%v", valuesAgain, err)
	}
	renderedAgain, _ := RenderExplicit(valuesAgain)
	if !bytes.Equal(rendered, renderedAgain) {
		t.Fatal("render is not deterministic")
	}
}

func TestCanonicalRegistryAndEveryGeneratedArtifactStaySynchronized(t *testing.T) {
	registryPath := filepath.Join("..", "..", "config", "registry.json")
	source, err := os.ReadFile(registryPath)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(source)
	var compactSource bytes.Buffer
	if err := json.Compact(&compactSource, source); err != nil {
		t.Fatal(err)
	}
	if got := hex.EncodeToString(digest[:]); got != generatedRegistrySourceSHA256 || compactSource.String() != generatedRegistryJSON {
		t.Fatalf("embedded registry drift: %s", got)
	}
	registry, err := loadRegistry()
	if err != nil {
		t.Fatal(err)
	}
	if len(registry.keys) != len(allGeneratedKeys()) {
		t.Fatal("generated key count drift")
	}
	generatedGo, err := os.ReadFile("keys_generated.go")
	if err != nil {
		t.Fatal(err)
	}
	if lines := bytes.Count(generatedGo, []byte{'\n'}); lines > 450 {
		t.Fatalf("generated config Go exceeds 450 lines: %d", lines)
	}

	temporary := t.TempDir()
	outputs := map[string]string{
		"keys_generated.go":     filepath.Join("keys_generated.go"),
		"orquesta.schema.json":  filepath.Join("..", "..", "config", "generated", "orquesta.schema.json"),
		"ui.json":               filepath.Join("..", "..", "config", "generated", "ui.json"),
		"orquesta.toml.example": filepath.Join("..", "..", "config", "orquesta.toml.example"),
		"reference.md":          filepath.Join("..", "..", "config", "generated", "reference.md"),
	}
	command := exec.Command("go", "run", "-mod=vendor", "./cmd/configgen", "-registry", registryPath,
		"-go-output", filepath.Join(temporary, "keys_generated.go"),
		"-schema-output", filepath.Join(temporary, "orquesta.schema.json"),
		"-ui-output", filepath.Join(temporary, "ui.json"),
		"-example-output", filepath.Join(temporary, "orquesta.toml.example"),
		"-doc-output", filepath.Join(temporary, "reference.md"))
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("configgen: %v: %s", err, output)
	}
	for generatedName, canonicalPath := range outputs {
		want, err := os.ReadFile(canonicalPath)
		if err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(filepath.Join(temporary, generatedName))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, want) {
			t.Fatalf("generated artifact drift: %s", canonicalPath)
		}
	}
}

func TestExampleIsCompleteAndContainsNoSecretMaterial(t *testing.T) {
	registry, err := loadRegistry()
	if err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join("..", "..", "config", "orquesta.toml.example"))
	if err != nil {
		t.Fatal(err)
	}
	values, err := ParseExplicit(content)
	if err != nil {
		t.Fatal(err)
	}
	if len(values) != len(registry.keys) {
		t.Fatalf("example keys = %d, want %d", len(values), len(registry.keys))
	}
	snapshot, err := Resolve(ResolveOptions{TOML: content})
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.RuntimeCodexModel() != "" || snapshot.RuntimeCodexCredentialRef() != "" {
		t.Fatal("example invents provider identity")
	}
}

func TestProcessEnvironmentReadLivesOnlyInDedicatedLoader(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	needle := "os." + "LookupEnv"
	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(content), needle) && filepath.Base(path) != "env_loader.go" {
			t.Fatalf("process environment read outside env_loader.go: %s", path)
		}
	}
}

func TestEffectiveProjectionHasNoFilesystemDependency(t *testing.T) {
	parsed, err := parser.ParseFile(token.NewFileSet(), "effective.go", nil, parser.ImportsOnly)
	if err != nil {
		t.Fatal(err)
	}
	for _, imported := range parsed.Imports {
		path, err := strconv.Unquote(imported.Path.Value)
		if err != nil {
			t.Fatal(err)
		}
		if path != "encoding/json" {
			t.Fatalf("effective projection imports concrete dependency %q", path)
		}
	}
}

func resolveTOML(t *testing.T, content string, environment map[string]string) Snapshot {
	t.Helper()
	var source []byte
	if strings.TrimSpace(content) != "" {
		source = []byte(strings.TrimSpace(content) + "\n")
	}
	snapshot, err := Resolve(ResolveOptions{TOML: source, Environment: environment, SourcePath: "test://inline"})
	if err != nil {
		t.Fatalf("resolve: %v (%v)", err, errors.Unwrap(err))
	}
	return snapshot
}

func mapEnvironment(values map[string]string) environmentLookup {
	return func(name string) (string, bool) { value, found := values[name]; return value, found }
}

func assertConfigError(t *testing.T, err error, code ErrorCode, key Key) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected %s", code)
	}
	var configError *Error
	if !errors.As(err, &configError) {
		t.Fatalf("error type = %T: %v", err, err)
	}
	if configError.Code != code || configError.Key != key {
		t.Fatalf("error = (%s, %s), want (%s, %s)", configError.Code, configError.Key, code, key)
	}
}

func assertSource(t *testing.T, snapshot Snapshot, key Key, want Source) {
	t.Helper()
	metadata, found := snapshot.Metadata(key)
	if !found || metadata.Source != want {
		t.Fatalf("source for %q = %+v/%v, want %q", key, metadata, found, want)
	}
}

func assertRedacted(t *testing.T, content []byte) {
	t.Helper()
	if strings.Contains(string(content), "credential:must-not-leak") || !strings.Contains(string(content), redactedValue) {
		t.Fatalf("effective redaction failed: %s", content)
	}
}
