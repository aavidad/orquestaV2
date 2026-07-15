package config

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestRegistryRejectsSemanticEnvironmentValidatorAndCrossValidatorDrift(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*registryFile)
	}{
		{name: "duplicate semantic ref", mutate: func(source *registryFile) {
			source.Keys[1].SemanticRef = source.Keys[0].SemanticRef
		}},
		{name: "duplicate environment", mutate: func(source *registryFile) {
			source.Keys[1].EnvAlias = source.Keys[0].EnvAlias
		}},
		{name: "validator id omitted", mutate: func(source *registryFile) {
			source.Keys[2].ValidatorIDs = nil
		}},
		{name: "validator id invented", mutate: func(source *registryFile) {
			source.Keys[0].ValidatorIDs = []string{"invented"}
		}},
		{name: "semantic validator wrong type", mutate: func(source *registryFile) {
			source.Keys[2].ValidatorIDs = append(source.Keys[2].ValidatorIDs, "opaque_ref")
		}},
		{name: "semantic validator duplicated", mutate: func(source *registryFile) {
			source.Keys[13].ValidatorIDs = []string{"trimmed_non_empty_string", "trimmed_non_empty_string"}
		}},
		{name: "semantic validators conflict", mutate: func(source *registryFile) {
			source.Keys[13].ValidatorIDs = []string{"trimmed_non_empty_string", "trimmed_optional_string"}
		}},
		{name: "cross validator omitted", mutate: func(source *registryFile) {
			source.CrossValidators = source.CrossValidators[1:]
		}},
		{name: "cross validator unknown id", mutate: func(source *registryFile) {
			source.CrossValidators[0].ID = "invented"
		}},
		{name: "cross validator unknown key", mutate: func(source *registryFile) {
			source.CrossValidators[0].Keys[0] = "unknown.key"
		}},
		{name: "cross validator repeated key", mutate: func(source *registryFile) {
			source.CrossValidators[0].Keys[1] = source.CrossValidators[0].Keys[0]
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			source := decodedRegistry(t)
			test.mutate(&source)
			if _, err := parseRegistrySource(marshalRegistry(t, source)); !HasErrorCode(err, ErrorRegistryInvalid) {
				t.Fatalf("registry drift accepted: %v", err)
			}
		})
	}
}

func TestRegistryAliasesAreTypedBoundedAndCanonicalized(t *testing.T) {
	source := decodedRegistry(t)
	source.Aliases = []registryAliasDefinition{
		{Kind: AliasKindTOMLKey, Name: "server.address", Target: KeyServerListen,
			IntroducedRevision: "2026-07-15.7", RemoveAfterRevision: "2026-09-01.9"},
		{Kind: AliasKindEnvironment, Name: "ORQUESTA_SERVER_ADDRESS", Target: KeyServerListen,
			IntroducedRevision: "2026-07-15.7", RemoveAfterRevision: "2026-09-01.9"},
	}
	registry, err := parseRegistrySource(marshalRegistry(t, source))
	if err != nil {
		t.Fatalf("valid aliases rejected: %v", errors.Unwrap(err))
	}
	explicit, err := parseExplicitWithRegistry([]byte("[server]\naddress = \"127.0.0.1:9191\"\n"), registry)
	if err != nil || explicit[KeyServerListen] != "127.0.0.1:9191" {
		t.Fatalf("TOML alias not canonicalized: %#v/%v", explicit, err)
	}
	resolved, _ := defaultResolvedValues(registry)
	if err := applyEnvironmentMap(resolved, registry, map[string]string{"ORQUESTA_SERVER_ADDRESS": "127.0.0.1:9292"}); err != nil ||
		resolved[KeyServerListen].value != "127.0.0.1:9292" {
		t.Fatalf("environment alias not canonicalized: %#v/%v", resolved[KeyServerListen], err)
	}
	resolved, _ = defaultResolvedValues(registry)
	err = applyEnvironmentMap(resolved, registry, map[string]string{
		"ORQUESTA_SERVER_LISTEN": "127.0.0.1:9191", "ORQUESTA_SERVER_ADDRESS": "127.0.0.1:9292",
	})
	assertConfigError(t, err, ErrorValueInvalid, KeyServerListen)

	invalid := []registryAliasDefinition{
		{Kind: "invented", Name: "server.address", Target: KeyServerListen, IntroducedRevision: "a", RemoveAfterRevision: "b"},
		{Kind: AliasKindTOMLKey, Name: "server.listen", Target: KeyServerListen, IntroducedRevision: "a", RemoveAfterRevision: "b"},
		{Kind: AliasKindTOMLKey, Name: "server.address", Target: "missing.key", IntroducedRevision: "a", RemoveAfterRevision: "b"},
		{Kind: AliasKindEnvironment, Name: "bad-env", Target: KeyServerListen, IntroducedRevision: "a", RemoveAfterRevision: "b"},
		{Kind: AliasKindEnvironment, Name: "ORQUESTA_SERVER_LISTEN", Target: KeyServerMCPPath, IntroducedRevision: "a", RemoveAfterRevision: "b"},
		{Kind: AliasKindTOMLKey, Name: "server.address", Target: KeyServerListen, IntroducedRevision: "same", RemoveAfterRevision: "same"},
	}
	for index, alias := range invalid {
		t.Run(alias.Name+string(rune('a'+index)), func(t *testing.T) {
			candidate := decodedRegistry(t)
			candidate.Aliases = []registryAliasDefinition{alias}
			if _, err := parseRegistrySource(marshalRegistry(t, candidate)); !HasErrorCode(err, ErrorRegistryInvalid) {
				t.Fatalf("invalid alias accepted: %+v / %v", alias, err)
			}
		})
	}

	for _, alias := range []registryAliasDefinition{
		{Kind: AliasKindTOMLKey, Name: "server.future", Target: KeyServerListen,
			IntroducedRevision: "2026-07-15.10", RemoveAfterRevision: "2026-09-01.0"},
		{Kind: AliasKindTOMLKey, Name: "server.expired", Target: KeyServerListen,
			IntroducedRevision: "2026-07-15.7", RemoveAfterRevision: "2026-07-15.8"},
	} {
		candidate := decodedRegistry(t)
		candidate.Aliases = []registryAliasDefinition{alias}
		if _, err := parseRegistrySource(marshalRegistry(t, candidate)); !HasErrorCode(err, ErrorRegistryInvalid) {
			t.Fatalf("inactive alias accepted: %+v / %v", alias, err)
		}
	}
}

func TestResolveExecutesDeclaredSemanticValidatorsBeforeBootstrapEffects(t *testing.T) {
	tests := []struct {
		name string
		key  Key
		toml string
	}{
		{name: "blank codex command", key: KeyRuntimeCodexCommand, toml: "[runtime.codex]\ncommand = \" \"\n"},
		{name: "untrimmed optional model", key: KeyRuntimeCodexModel, toml: "[runtime.codex]\nmodel = \" xhigh \"\n"},
		{name: "invalid child environment", key: KeyRuntimeCodexEnvAllowlist, toml: "[runtime.codex]\nenv_allowlist = [\"PATH\", \"BAD NAME\"]\n"},
		{name: "blank actor ref", key: KeyIdentityLocalActor, toml: "[identity]\nlocal_actor = \" \"\n"},
		{name: "untrimmed project ref", key: KeyProjectDefault, toml: "[project]\ndefault = \" project:default\"\n"},
		{name: "untrimmed path", key: KeyStateSQLitePath, toml: "[state.sqlite]\npath = \" ./var/state.sqlite\"\n"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := Resolve(ResolveOptions{TOML: []byte(test.toml)})
			assertConfigError(t, err, ErrorValueInvalid, test.key)
		})
	}

	_, err := Resolve(ResolveOptions{Environment: map[string]string{
		"ORQUESTA_RUNTIME_CODEX_ENV_ALLOWLIST": "PATH,bad_name",
	}})
	assertConfigError(t, err, ErrorValueInvalid, KeyRuntimeCodexEnvAllowlist)
}

func TestRegistryRevisionIsCanonicalAndNumericSequenceIsOrdered(t *testing.T) {
	for _, revision := range []string{"", "2026-7-15.8", "2026-07-15.-1", "2026-07-15.08", "2026-07-15"} {
		source := decodedRegistry(t)
		source.Revision = revision
		if _, err := parseRegistrySource(marshalRegistry(t, source)); !HasErrorCode(err, ErrorRegistryInvalid) {
			t.Fatalf("invalid revision %q accepted: %v", revision, err)
		}
	}
	before, _ := parseRegistryRevision("2026-07-15.9")
	after, _ := parseRegistryRevision("2026-07-15.10")
	if before.compare(after) >= 0 {
		t.Fatal("numeric registry sequence ordered lexically")
	}
}

func TestResolveExecutesEveryDeclaredCrossValidator(t *testing.T) {
	tests := []struct {
		name string
		toml string
	}{
		{name: "runtime timeout", toml: "[runtime.codex]\ntimeout = \"45m\""},
		{name: "non loopback", toml: "[server]\nlisten = \"0.0.0.0:8080\""},
		{name: "non literal MCP path", toml: "[server]\nmcp_path = \"/mcp/../other\""},
		{name: "overlapping paths", toml: "[artifact.filesystem]\nroot = \"./var/state\""},
		{name: "credential store equals local token", toml: "[credentials.local]\npath = \"./var/secrets/local-owner.token\""},
		{name: "credential store inside artifacts", toml: "[credentials.local]\npath = \"./var/artifacts/credentials.json\""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := Resolve(ResolveOptions{TOML: []byte(test.toml)})
			assertConfigError(t, err, ErrorCrossValidation, "")
		})
	}
	if got := CrossValidators(); len(got) != 4 {
		t.Fatalf("cross validator catalog = %+v", got)
	} else {
		got[0].Keys[0] = "mutated"
		if CrossValidators()[0].Keys[0] == "mutated" {
			t.Fatal("cross validator keys are not detached")
		}
	}
}

func TestCaptureEnvironmentReadsOnlyRegistryNames(t *testing.T) {
	t.Setenv("ORQUESTA_SERVER_LISTEN", "127.0.0.1:9393")
	t.Setenv("ORQUESTA_V07_UNDECLARED_SENTINEL", "must-not-be-captured")
	captured, err := CaptureEnvironment()
	if err != nil {
		t.Fatal(err)
	}
	if captured["ORQUESTA_SERVER_LISTEN"] != "127.0.0.1:9393" {
		t.Fatalf("declared environment omitted: %#v", captured)
	}
	if _, leaked := captured["ORQUESTA_V07_UNDECLARED_SENTINEL"]; leaked {
		t.Fatalf("undeclared environment captured: %#v", captured)
	}
	registry, _ := loadRegistry()
	for name := range captured {
		if _, declared := registry.environmentTargets[name]; !declared {
			if _, alias := registry.environmentAliases[name]; !alias {
				t.Fatalf("captured name is outside registry: %s", name)
			}
		}
	}
}

func TestDefinitionAndAliasOutputsAreDetached(t *testing.T) {
	definitions := Definitions()
	if len(definitions) != len(allGeneratedKeys()) {
		t.Fatalf("definitions = %d", len(definitions))
	}
	listIndex := -1
	for index := range definitions {
		if definitions[index].Key == KeyRuntimeCodexEnvAllowlist {
			listIndex = index
			break
		}
	}
	if listIndex < 0 {
		t.Fatal("list definition missing")
	}
	definitions[listIndex].ValidatorIDs[0] = "mutated"
	definitions[listIndex].Default.([]string)[0] = "mutated"
	again, _ := Definition(KeyRuntimeCodexEnvAllowlist)
	if again.ValidatorIDs[0] == "mutated" || again.Default.([]string)[0] == "mutated" {
		t.Fatal("definition output shares registry state")
	}
	if aliases := Aliases(); aliases == nil || len(aliases) != 0 {
		t.Fatalf("canonical empty alias catalog = %#v", aliases)
	}
}

func decodedRegistry(t *testing.T) registryFile {
	t.Helper()
	var source registryFile
	decoder := json.NewDecoder(strings.NewReader(generatedRegistryJSON))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&source); err != nil {
		t.Fatal(err)
	}
	return source
}

func marshalRegistry(t *testing.T, source registryFile) []byte {
	t.Helper()
	content, err := json.Marshal(source)
	if err != nil {
		t.Fatal(err)
	}
	return content
}

func TestPrecedenceAndCatalogSlicesAreDetached(t *testing.T) {
	registry, err := loadRegistry()
	if err != nil {
		t.Fatal(err)
	}
	want := []Source{SourceDefault, SourceFile, SourceEnv}
	if !reflect.DeepEqual(registry.precedence, want) {
		t.Fatalf("precedence = %#v", registry.precedence)
	}
	if SourceMaxBytes() != 1048576 || AuditEntryMaxBytes() != 65536 {
		t.Fatalf("document limits = %d/%d", SourceMaxBytes(), AuditEntryMaxBytes())
	}
}
