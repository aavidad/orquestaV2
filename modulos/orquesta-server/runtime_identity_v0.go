package orquestaserver

import (
	"path/filepath"
	"strings"
)

const (
	ServerRuntimeIdentitySchemaVersionV0 = "orquesta_server_runtime_identity.v0"
	ServerRuntimeBinaryPathRefV0         = "server-runtime-binary-path"
)

type ServerRuntimeIdentityV0 struct {
	SchemaVersion string   `json:"schema_version,omitempty"`
	BinaryPath    string   `json:"binary_path,omitempty"`
	BinaryPathRef string   `json:"binary_path_ref,omitempty"`
	BinaryName    string   `json:"binary_name,omitempty"`
	BinarySHA256  string   `json:"binary_sha256,omitempty"`
	BuildRef      string   `json:"build_ref,omitempty"`
	CommitRef     string   `json:"commit_ref,omitempty"`
	StartedAt     string   `json:"started_at,omitempty"`
	EvidenceRefs  []string `json:"evidence_refs,omitempty"`
}

type ServerPublicRuntimeIdentityV0 struct {
	SchemaVersion string   `json:"schema_version,omitempty"`
	BinaryPathRef string   `json:"binary_path_ref,omitempty"`
	BinaryName    string   `json:"binary_name,omitempty"`
	BinarySHA256  string   `json:"binary_sha256,omitempty"`
	BuildRef      string   `json:"build_ref,omitempty"`
	CommitRef     string   `json:"commit_ref,omitempty"`
	StartedAt     string   `json:"started_at,omitempty"`
	EvidenceRefs  []string `json:"evidence_refs,omitempty"`
}

func NormalizeServerRuntimeIdentityV0(identity ServerRuntimeIdentityV0) ServerRuntimeIdentityV0 {
	identity.SchemaVersion = strings.TrimSpace(identity.SchemaVersion)
	if identity.SchemaVersion == "" {
		identity.SchemaVersion = ServerRuntimeIdentitySchemaVersionV0
	}
	identity.BinaryPath = strings.TrimSpace(identity.BinaryPath)
	identity.BinaryPathRef = strings.TrimSpace(identity.BinaryPathRef)
	if identity.BinaryPathRef == "" && identity.BinaryPath != "" {
		identity.BinaryPathRef = ServerRuntimeBinaryPathRefV0
	}
	identity.BinaryName = strings.TrimSpace(identity.BinaryName)
	if identity.BinaryName == "" && identity.BinaryPath != "" {
		identity.BinaryName = filepath.Base(identity.BinaryPath)
	}
	identity.BinarySHA256 = strings.TrimSpace(identity.BinarySHA256)
	identity.BuildRef = strings.TrimSpace(identity.BuildRef)
	identity.CommitRef = strings.TrimSpace(identity.CommitRef)
	identity.StartedAt = strings.TrimSpace(identity.StartedAt)
	identity.EvidenceRefs = compactServerStringsV0(identity.EvidenceRefs)
	if identityEmptyV0(identity) {
		return ServerRuntimeIdentityV0{}
	}
	return identity
}

func NewServerPublicRuntimeIdentityV0(state StateV0) ServerPublicRuntimeIdentityV0 {
	identity := NormalizeServerRuntimeIdentityV0(state.RuntimeIdentity)
	if identityEmptyV0(identity) {
		return ServerPublicRuntimeIdentityV0{}
	}
	if identity.StartedAt == "" {
		identity.StartedAt = strings.TrimSpace(state.StartedAt)
	}
	return ServerPublicRuntimeIdentityV0{
		SchemaVersion: identity.SchemaVersion,
		BinaryPathRef: identity.BinaryPathRef,
		BinaryName:    identity.BinaryName,
		BinarySHA256:  identity.BinarySHA256,
		BuildRef:      identity.BuildRef,
		CommitRef:     identity.CommitRef,
		StartedAt:     identity.StartedAt,
		EvidenceRefs:  append([]string(nil), identity.EvidenceRefs...),
	}
}

func identityEmptyV0(identity ServerRuntimeIdentityV0) bool {
	return strings.TrimSpace(identity.BinaryPath) == "" &&
		strings.TrimSpace(identity.BinaryPathRef) == "" &&
		strings.TrimSpace(identity.BinaryName) == "" &&
		strings.TrimSpace(identity.BinarySHA256) == "" &&
		strings.TrimSpace(identity.BuildRef) == "" &&
		strings.TrimSpace(identity.CommitRef) == "" &&
		strings.TrimSpace(identity.StartedAt) == "" &&
		len(identity.EvidenceRefs) == 0
}
