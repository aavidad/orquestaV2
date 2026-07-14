package config

// Key is a generated canonical registry key.
type Key string

// Source identifies the winning layer for a resolved value.
type Source string

const (
	SourceDefault Source = "default"
	SourceFile    Source = "file"
	SourceEnv     Source = "env"
)

// CredentialRef is an opaque credential identifier, never secret material.
type CredentialRef string

// AliasKind distinguishes human TOML aliases from bootstrap environment
// aliases.
type AliasKind string

const (
	AliasKindTOMLKey     AliasKind = "toml_key"
	AliasKindEnvironment AliasKind = "environment"
)

// AliasDefinition describes a temporary input spelling and its bounded
// migration to one canonical key.
type AliasDefinition struct {
	Kind                AliasKind
	Name                string
	Target              Key
	IntroducedRevision  string
	RemoveAfterRevision string
}

// CrossValidatorDefinition declares one multi-key invariant. Execution is
// selected by ID; the registry remains the only source of the dependency set.
type CrossValidatorDefinition struct {
	ID   string
	Keys []Key
}

// KeyDefinition is a detached description of one canonical key. Slice and
// pointer fields returned by this package are always cloned.
type KeyDefinition struct {
	Key             Key
	GoName          string
	SemanticRef     string
	Type            string
	Default         any
	Sensitive       bool
	Scope           string
	RestartRequired bool
	EnvAlias        string
	ValidatorIDs    []string
	AllowedValues   []string
	Minimum         *int64
	Maximum         *int64
}

// KeyMetadata describes provenance and operational behavior without exposing
// the resolved value.
type KeyMetadata struct {
	Source          Source
	Type            string
	SemanticRef     string
	Sensitive       bool
	Scope           string
	RestartRequired bool
	EnvAlias        string
	ValidatorIDs    []string
	AllowedValues   []string
	Minimum         *int64
	Maximum         *int64
}

// Snapshot is an immutable resolved configuration. Its only state is the
// private canonical entry set; generated getters return values or clones.
type Snapshot struct {
	schemaVersion    int
	registryRevision string
	registryHash     string
	hash             string
	entries          []snapshotEntry
}

type snapshotEntry struct {
	key      Key
	value    any
	metadata KeyMetadata
}

// SchemaVersion returns the canonical registry schema version.
func (s Snapshot) SchemaVersion() int { return s.schemaVersion }

// RegistryRevision returns the human registry revision.
func (s Snapshot) RegistryRevision() string { return s.registryRevision }

// RegistryHash returns the semantic digest of the registry, independent of
// JSON whitespace.
func (s Snapshot) RegistryHash() string { return s.registryHash }

// Hash returns the resolved and redacted snapshot digest.
func (s Snapshot) Hash() string { return s.hash }

// Metadata returns detached provenance and registry metadata.
func (s Snapshot) Metadata(key Key) (KeyMetadata, bool) {
	for _, entry := range s.entries {
		if entry.key == key {
			return cloneKeyMetadata(entry.metadata), true
		}
	}
	return KeyMetadata{}, false
}

func (s Snapshot) value(key Key) (any, bool) {
	for _, entry := range s.entries {
		if entry.key == key {
			return cloneValue(entry.value), true
		}
	}
	return nil, false
}

func cloneKeyMetadata(metadata KeyMetadata) KeyMetadata {
	metadata.ValidatorIDs = append([]string(nil), metadata.ValidatorIDs...)
	metadata.AllowedValues = append([]string(nil), metadata.AllowedValues...)
	metadata.Minimum = cloneInt64Pointer(metadata.Minimum)
	metadata.Maximum = cloneInt64Pointer(metadata.Maximum)
	return metadata
}

func cloneInt64Pointer(value *int64) *int64 {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}
