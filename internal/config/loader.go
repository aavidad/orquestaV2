package config

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"path"
	"path/filepath"
	"strings"
	"time"
)

// ResolveOptions contains captured inputs. Environment must contain only
// environment names declared by the canonical registry.
type ResolveOptions struct {
	TOML        []byte
	Environment map[string]string
	SourcePath  string
}

// Resolve resolves captured inputs without reading process-global state.
func Resolve(options ResolveOptions) (Snapshot, error) {
	registry, err := loadRegistry()
	if err != nil {
		return Snapshot{}, err
	}
	values, err := defaultResolvedValues(registry)
	if err != nil {
		return Snapshot{}, err
	}
	if options.TOML != nil {
		explicit, err := parseExplicitWithRegistry(options.TOML, registry)
		if err != nil {
			return Snapshot{}, err
		}
		applyExplicit(values, explicit)
	}
	for name := range options.Environment {
		_, canonical := registry.environmentTargets[name]
		_, alias := registry.environmentAliases[name]
		if !canonical && !alias {
			return Snapshot{}, &Error{Code: ErrorUnknownKey, Key: Key(name)}
		}
	}
	if err := applyEnvironmentMap(values, registry, options.Environment); err != nil {
		return Snapshot{}, err
	}
	if err := validateCrossRegistryValues(registry, values, options.SourcePath); err != nil {
		return Snapshot{}, err
	}
	return buildSnapshot(registry, values)
}

type resolvedValue struct {
	value  any
	source Source
}

func defaultResolvedValues(registry registry) (map[Key]resolvedValue, error) {
	resolved := make(map[Key]resolvedValue, len(registry.keys))
	for _, definition := range registry.keys {
		value, err := parseDefaultValue(definition)
		if err != nil {
			return nil, &Error{Code: ErrorRegistryInvalid, Key: definition.Key, Cause: err}
		}
		resolved[definition.Key] = resolvedValue{value: cloneValue(value), source: SourceDefault}
	}
	return resolved, nil
}

func applyExplicit(resolved map[Key]resolvedValue, explicit map[Key]any) {
	for key, value := range explicit {
		resolved[key] = resolvedValue{value: cloneValue(value), source: SourceFile}
	}
}

func applyEnvironmentMap(resolved map[Key]resolvedValue, registry registry, environment map[string]string) error {
	normalized := make(map[Key]string, len(environment))
	for name, raw := range environment {
		key, canonical := registry.environmentTargets[name]
		if !canonical {
			key = registry.environmentAliases[name]
		}
		if _, duplicate := normalized[key]; duplicate {
			return &Error{Code: ErrorValueInvalid, Key: key, Cause: fmt.Errorf("config_environment_alias_ambiguous")}
		}
		normalized[key] = raw
	}
	for key, raw := range normalized {
		definition, _ := registry.definition(key)
		value, err := parseEnvironmentValue(definition, raw)
		if err != nil {
			return &Error{Code: ErrorValueInvalid, Key: definition.Key, Cause: err}
		}
		resolved[definition.Key] = resolvedValue{value: cloneValue(value), source: SourceEnv}
	}
	return nil
}

func validateCrossRegistryValues(registry registry, values map[Key]resolvedValue, sourcePath string) error {
	fail := func(id string) error {
		return &Error{Code: ErrorCrossValidation, Cause: fmt.Errorf("%s", id)}
	}
	for _, validator := range registry.crossValidators {
		switch validator.ID {
		case "runtime_codex_timeout_before_scheduler_execution_timeout":
			runtimeTimeout, runtimeOK := values[KeyRuntimeCodexTimeout].value.(time.Duration)
			executionTimeout, executionOK := values[KeySchedulerExecutionTimeout].value.(time.Duration)
			if !runtimeOK || !executionOK || runtimeTimeout >= executionTimeout {
				return fail(validator.ID)
			}
		case "server_listen_loopback":
			listen, ok := values[KeyServerListen].value.(string)
			host, _, err := net.SplitHostPort(listen)
			ip := net.ParseIP(host)
			if !ok || err != nil || ip == nil || !ip.IsLoopback() {
				return fail(validator.ID)
			}
		case "server_mcp_path_literal":
			value, ok := values[KeyServerMCPPath].value.(string)
			if !ok || !literalMCPPath(value) {
				return fail(validator.ID)
			}
		case "runtime_paths_disjoint":
			if !runtimePathsDisjoint(values, sourcePath) {
				return fail(validator.ID)
			}
		default:
			return fail(validator.ID)
		}
	}
	return nil
}

func literalMCPPath(value string) bool {
	if !strings.HasPrefix(value, "/") || value == "/" || path.Clean(value) != value {
		return false
	}
	for _, character := range value {
		switch {
		case character >= 'a' && character <= 'z':
		case character >= 'A' && character <= 'Z':
		case character >= '0' && character <= '9':
		case strings.ContainsRune("/-._~", character):
		default:
			return false
		}
	}
	return true
}

func runtimePathsDisjoint(values map[Key]resolvedValue, sourcePath string) bool {
	canonical := func(key Key) (string, bool) {
		raw, ok := values[key].value.(string)
		if !ok || strings.TrimSpace(raw) == "" {
			return "", false
		}
		absolute, err := filepath.Abs(filepath.Clean(raw))
		return absolute, err == nil
	}
	statePath, stateOK := canonical(KeyStateSQLitePath)
	artifactRoot, artifactOK := canonical(KeyArtifactFilesystemRoot)
	credentialPath, credentialOK := canonical(KeyCredentialsLocalPath)
	workRoot, workOK := canonical(KeyRuntimeCodexWorkRoot)
	effectivePath, effectiveOK := canonical(KeyConfigEffectivePath)
	tokenPath, tokenOK := canonical(KeyIdentityLocalTokenPath)
	if !stateOK || !artifactOK || !credentialOK || !workOK || !effectiveOK || !tokenOK {
		return false
	}
	stateDirectory, tokenDirectory := filepath.Dir(statePath), filepath.Dir(tokenPath)
	pairs := [][2]string{
		{stateDirectory, artifactRoot}, {stateDirectory, workRoot}, {artifactRoot, workRoot},
		{credentialPath, stateDirectory}, {credentialPath, artifactRoot}, {credentialPath, workRoot},
		{credentialPath, effectivePath}, {credentialPath, tokenPath},
		{effectivePath, statePath}, {effectivePath, artifactRoot}, {effectivePath, workRoot},
		{tokenDirectory, stateDirectory}, {tokenDirectory, artifactRoot}, {tokenDirectory, workRoot},
		{tokenDirectory, effectivePath},
	}
	for _, pair := range pairs {
		if pathsOverlap(pair[0], pair[1]) {
			return false
		}
	}
	if strings.TrimSpace(sourcePath) != "" {
		configPath, err := filepath.Abs(filepath.Clean(sourcePath))
		if err != nil {
			return false
		}
		for _, other := range []string{statePath, artifactRoot, credentialPath, workRoot, effectivePath, tokenDirectory} {
			if pathsOverlap(configPath, other) {
				return false
			}
		}
	}
	return true
}

func pathsOverlap(left, right string) bool {
	relative, err := filepath.Rel(left, right)
	if err == nil && (relative == "." || (relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)))) {
		return true
	}
	relative, err = filepath.Rel(right, left)
	return err == nil && (relative == "." || (relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))))
}

type snapshotHashDocument struct {
	RegistryRevision string              `json:"registry_revision"`
	RegistryHash     string              `json:"registry_hash"`
	Entries          []snapshotHashEntry `json:"entries"`
}

type snapshotHashEntry struct {
	Key    Key    `json:"key"`
	Source Source `json:"source"`
	Value  any    `json:"value"`
}

func buildSnapshot(registry registry, resolved map[Key]resolvedValue) (Snapshot, error) {
	snapshot := Snapshot{
		schemaVersion: registry.schemaVersion, registryRevision: registry.revision,
		registryHash: registry.semanticHash, entries: make([]snapshotEntry, 0, len(registry.keys)),
	}
	hashDocument := snapshotHashDocument{
		RegistryRevision: registry.revision, RegistryHash: registry.semanticHash,
		Entries: make([]snapshotHashEntry, 0, len(registry.keys)),
	}
	for _, definition := range registry.keys {
		value, found := resolved[definition.Key]
		if !found {
			return Snapshot{}, &Error{Code: ErrorRegistryInvalid, Key: definition.Key}
		}
		stored := cloneValue(value.value)
		hashValue := cloneValue(canonicalValue(stored))
		if definition.Sensitive {
			hashValue = redactedValue
		}
		snapshot.entries = append(snapshot.entries, snapshotEntry{
			key: definition.Key, value: stored,
			metadata: KeyMetadata{
				Source: value.source, Type: string(definition.Type), SemanticRef: definition.SemanticRef,
				Sensitive: definition.Sensitive, Scope: definition.Scope, RestartRequired: definition.RestartRequired,
				EnvAlias: definition.EnvAlias, ValidatorIDs: append([]string(nil), definition.ValidatorIDs...),
				AllowedValues: append([]string(nil), definition.AllowedValues...),
				Minimum:       cloneInt64Pointer(definition.Minimum), Maximum: cloneInt64Pointer(definition.Maximum),
			},
		})
		hashDocument.Entries = append(hashDocument.Entries, snapshotHashEntry{
			Key: definition.Key, Source: value.source, Value: cloneValue(hashValue),
		})
	}

	hashPayload, err := json.Marshal(hashDocument)
	if err != nil {
		return Snapshot{}, &Error{Code: ErrorRegistryInvalid, Cause: err}
	}
	digest := sha256.Sum256(hashPayload)
	snapshot.hash = "sha256:" + hex.EncodeToString(digest[:])
	return snapshot, nil
}
