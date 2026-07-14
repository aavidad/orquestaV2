package config

import "time"

// Key is a generated canonical registry key.
type Key string

// Source identifies the winning layer for a resolved value.
type Source string

const (
	SourceDefault Source = "default"
	SourceFile    Source = "file"
	SourceEnv     Source = "env"
)

// CredentialRef is an opaque credential identifier, never a secret value.
type CredentialRef string

type ServerConfig struct {
	Listen          string
	MCPPath         string
	MaxRequestBytes int64
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
}

type SQLiteConfig struct {
	Path               string
	BusyTimeout        time.Duration
	MaxOpenConnections int64
}

type StateConfig struct {
	SQLite SQLiteConfig
}

type FilesystemArtifactConfig struct {
	Root string
}

type ArtifactConfig struct {
	Filesystem FilesystemArtifactConfig
}

type CodexRuntimeConfig struct {
	Command                 string
	Model                   string
	Reasoning               string
	Timeout                 time.Duration
	ProcessPipeDrainDelay   time.Duration
	MaxDiagnosticBytes      int64
	MaxConcurrentExecutions int64
	WorkRoot                string
	EnvAllowlist            []string
	CredentialRef           CredentialRef `json:"-"`
}

type RuntimeConfig struct {
	Provider       string
	MaxOutputBytes int64
	Codex          CodexRuntimeConfig
}

type IdentityConfig struct {
	LocalActor     string
	LocalTokenPath string
}

type ProjectConfig struct {
	Default string
}

type SchedulerConfig struct {
	PollInterval         time.Duration
	ObservationInterval  time.Duration
	ClaimLease           time.Duration
	MaxExecutionAttempts int64
	ExecutionTimeout     time.Duration
}

type APIConfig struct {
	MaxListLimit int64
	Locale       string
}

type EffectiveConfig struct {
	Path             string
	MaxExistingBytes int64
}

// KeyMetadata describes provenance and operational behavior without exposing
// the resolved value.
type KeyMetadata struct {
	Source          Source
	Type            string
	Sensitive       bool
	Scope           string
	RestartRequired bool
	EnvAlias        string
	Minimum         *int64
	Maximum         *int64
}

// Snapshot is the immutable-by-convention typed result consumed by bootstrap.
// Hash covers registry revision, non-sensitive canonical values and winning
// sources. Sensitive values contribute only the stable redaction marker.
type Snapshot struct {
	SchemaVersion    int
	RegistryRevision string
	Hash             string
	Server           ServerConfig
	State            StateConfig
	Artifact         ArtifactConfig
	Runtime          RuntimeConfig
	Identity         IdentityConfig
	Project          ProjectConfig
	Scheduler        SchedulerConfig
	API              APIConfig
	Effective        EffectiveConfig

	entries []snapshotEntry
}

type snapshotEntry struct {
	key      Key
	value    any
	metadata KeyMetadata
}

// Metadata returns provenance and registry metadata for a generated key.
func (s Snapshot) Metadata(key Key) (KeyMetadata, bool) {
	for _, entry := range s.entries {
		if entry.key == key {
			metadata := entry.metadata
			metadata.Minimum = cloneInt64Pointer(metadata.Minimum)
			metadata.Maximum = cloneInt64Pointer(metadata.Maximum)
			return metadata, true
		}
	}
	return KeyMetadata{}, false
}

func cloneInt64Pointer(value *int64) *int64 {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}
