package config

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"
)

// LoadOptions selects the optional TOML source. Empty FilePath means
// registry defaults plus declared environment overrides.
type LoadOptions struct {
	FilePath string
}

// Load resolves default < TOML file < declared environment and returns one
// typed snapshot. It never accepts effective-config JSON as input.
func Load(options LoadOptions) (Snapshot, error) {
	return loadWithEnvironment(options, lookupSystemEnvironment)
}

type resolvedValue struct {
	value  any
	source Source
}

func loadWithEnvironment(options LoadOptions, lookup environmentLookup) (Snapshot, error) {
	registry, err := loadRegistry()
	if err != nil {
		return Snapshot{}, err
	}
	resolved := make(map[Key]resolvedValue, len(registry.keys))
	for _, definition := range registry.keys {
		value, err := parseDefaultValue(definition)
		if err != nil {
			return Snapshot{}, &Error{Code: ErrorRegistryInvalid, Key: definition.Key, Cause: err}
		}
		resolved[definition.Key] = resolvedValue{value: value, source: SourceDefault}
	}

	if options.FilePath != "" {
		fileValues, err := loadTOMLFile(options.FilePath, registry)
		if err != nil {
			return Snapshot{}, err
		}
		for key, raw := range fileValues {
			definition, _ := registry.definition(key)
			value, err := parseFileValue(definition, raw)
			if err != nil {
				return Snapshot{}, &Error{Code: ErrorValueInvalid, Key: key, Cause: err}
			}
			resolved[key] = resolvedValue{value: value, source: SourceFile}
		}
	}

	if lookup != nil {
		for _, definition := range registry.keys {
			raw, present := lookup(definition.EnvAlias)
			if !present {
				continue
			}
			value, err := parseEnvironmentValue(definition, raw)
			if err != nil {
				return Snapshot{}, &Error{Code: ErrorValueInvalid, Key: definition.Key, Cause: err}
			}
			resolved[definition.Key] = resolvedValue{value: value, source: SourceEnv}
		}
	}
	return buildSnapshot(registry, resolved)
}

type snapshotHashDocument struct {
	RegistryRevision string              `json:"registry_revision"`
	Entries          []snapshotHashEntry `json:"entries"`
}

type snapshotHashEntry struct {
	Key    Key    `json:"key"`
	Source Source `json:"source"`
	Value  any    `json:"value"`
}

func buildSnapshot(registry registry, resolved map[Key]resolvedValue) (Snapshot, error) {
	snapshot := Snapshot{
		SchemaVersion:    registry.schemaVersion,
		RegistryRevision: registry.revision,
		entries:          make([]snapshotEntry, 0, len(registry.keys)),
	}
	hashDocument := snapshotHashDocument{
		RegistryRevision: registry.revision,
		Entries:          make([]snapshotHashEntry, 0, len(registry.keys)),
	}
	for _, definition := range registry.keys {
		value, found := resolved[definition.Key]
		if !found {
			return Snapshot{}, &Error{Code: ErrorRegistryInvalid, Key: definition.Key}
		}
		canonical := canonicalValue(value.value)
		hashValue := canonical
		if definition.Sensitive {
			hashValue = redactedValue
		}
		snapshot.entries = append(snapshot.entries, snapshotEntry{
			key:   definition.Key,
			value: canonical,
			metadata: KeyMetadata{
				Source:          value.source,
				Type:            string(definition.Type),
				Sensitive:       definition.Sensitive,
				Scope:           definition.Scope,
				RestartRequired: definition.RestartRequired,
				EnvAlias:        definition.EnvAlias,
				Minimum:         cloneInt64Pointer(definition.Minimum),
				Maximum:         cloneInt64Pointer(definition.Maximum),
			},
		})
		hashDocument.Entries = append(hashDocument.Entries, snapshotHashEntry{
			Key:    definition.Key,
			Source: value.source,
			Value:  hashValue,
		})
	}

	snapshot.Server = ServerConfig{
		Listen:          stringValue(resolved, KeyServerListen),
		MCPPath:         stringValue(resolved, KeyServerMCPPath),
		MaxRequestBytes: int64Value(resolved, KeyServerMaxRequestBytes),
		ReadTimeout:     durationValue(resolved, KeyServerReadTimeout),
		WriteTimeout:    durationValue(resolved, KeyServerWriteTimeout),
		IdleTimeout:     durationValue(resolved, KeyServerIdleTimeout),
		ShutdownTimeout: durationValue(resolved, KeyServerShutdownTimeout),
	}
	snapshot.State.SQLite = SQLiteConfig{
		Path:               stringValue(resolved, KeyStateSQLitePath),
		BusyTimeout:        durationValue(resolved, KeyStateSQLiteBusyTimeout),
		MaxOpenConnections: int64Value(resolved, KeyStateSQLiteMaxOpenConnections),
	}
	snapshot.Artifact.Filesystem.Root = stringValue(resolved, KeyArtifactFilesystemRoot)
	snapshot.Runtime = RuntimeConfig{
		Provider:       stringValue(resolved, KeyRuntimeProvider),
		MaxOutputBytes: int64Value(resolved, KeyRuntimeMaxOutputBytes),
		Codex: CodexRuntimeConfig{
			Command:                 stringValue(resolved, KeyRuntimeCodexCommand),
			Model:                   stringValue(resolved, KeyRuntimeCodexModel),
			Reasoning:               stringValue(resolved, KeyRuntimeCodexReasoning),
			Timeout:                 durationValue(resolved, KeyRuntimeCodexTimeout),
			ProcessPipeDrainDelay:   durationValue(resolved, KeyRuntimeCodexProcessPipeDrainDelay),
			MaxDiagnosticBytes:      int64Value(resolved, KeyRuntimeCodexMaxDiagnosticBytes),
			MaxConcurrentExecutions: int64Value(resolved, KeyRuntimeCodexMaxConcurrentExecutions),
			WorkRoot:                stringValue(resolved, KeyRuntimeCodexWorkRoot),
			EnvAllowlist:            stringListValue(resolved, KeyRuntimeCodexEnvAllowlist),
			CredentialRef:           credentialRefValue(resolved, KeyRuntimeCodexCredentialRef),
		},
	}
	snapshot.Identity = IdentityConfig{
		LocalActor:     stringValue(resolved, KeyIdentityLocalActor),
		LocalTokenPath: stringValue(resolved, KeyIdentityLocalTokenPath),
	}
	snapshot.Project.Default = stringValue(resolved, KeyProjectDefault)
	snapshot.Scheduler = SchedulerConfig{
		PollInterval:        durationValue(resolved, KeySchedulerPollInterval),
		ObservationInterval: durationValue(resolved, KeySchedulerObservationInterval),
		ClaimLease:          durationValue(resolved, KeySchedulerClaimLease),
		MaxActionAttempts:   int64Value(resolved, KeySchedulerMaxActionAttempts),
		ExecutionTimeout:    durationValue(resolved, KeySchedulerExecutionTimeout),
	}
	snapshot.API = APIConfig{
		MaxListLimit: int64Value(resolved, KeyAPIMaxListLimit),
		Locale:       stringValue(resolved, KeyAPILocale),
	}
	snapshot.Effective = EffectiveConfig{
		Path:             stringValue(resolved, KeyConfigEffectivePath),
		MaxExistingBytes: int64Value(resolved, KeyConfigEffectiveMaxExistingBytes),
	}

	hashPayload, err := json.Marshal(hashDocument)
	if err != nil {
		return Snapshot{}, &Error{Code: ErrorRegistryInvalid, Cause: err}
	}
	digest := sha256.Sum256(hashPayload)
	snapshot.Hash = "sha256:" + hex.EncodeToString(digest[:])
	return snapshot, nil
}

func stringValue(values map[Key]resolvedValue, key Key) string {
	switch value := values[key].value.(type) {
	case string:
		return value
	case CredentialRef:
		return string(value)
	default:
		return ""
	}
}

func durationValue(values map[Key]resolvedValue, key Key) time.Duration {
	value, _ := values[key].value.(time.Duration)
	return value
}

func int64Value(values map[Key]resolvedValue, key Key) int64 {
	value, _ := values[key].value.(int64)
	return value
}

func stringListValue(values map[Key]resolvedValue, key Key) []string {
	value, _ := values[key].value.([]string)
	return append([]string(nil), value...)
}

func credentialRefValue(values map[Key]resolvedValue, key Key) CredentialRef {
	value, _ := values[key].value.(CredentialRef)
	return value
}
