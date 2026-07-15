package jsonimport

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"

	"orquesta/internal/config"
)

const transformIdentityString = "identity_string"

var canonicalMappings = [...]Mapping{
	{
		LegacyPath:  "autoprogramming.checkpoint_only_high_consumption_tokens",
		SourceKind:  SourceKindInteger,
		Disposition: DispositionDeferred,
		Reason:      "legacy_autoprogramming_policy_has_no_canonical_key",
	},
	{
		LegacyPath:  "control_plane.remote_access_opt_in",
		SourceKind:  SourceKindBool,
		Disposition: DispositionDeferred,
		Reason:      "legacy_remote_access_policy_has_no_canonical_key",
	},
	{
		LegacyPath:  "control_plane.token",
		SourceKind:  SourceKindString,
		Disposition: DispositionSecretRequired,
		Reason:      "secret_requires_credential_store_migration",
	},
	{
		LegacyPath:  "operator_director_mailbox.enabled",
		SourceKind:  SourceKindBool,
		Disposition: DispositionDeferred,
		Reason:      "legacy_mailbox_switch_has_no_canonical_key",
	},
	{
		LegacyPath:  "runtime_models.allowed_models",
		SourceKind:  SourceKindStringArray,
		Disposition: DispositionDeferred,
		Reason:      "legacy_runtime_allowed_models_has_no_canonical_key",
	},
	{
		LegacyPath:  "runtime_models.enabled",
		SourceKind:  SourceKindBool,
		Disposition: DispositionDeferred,
		Reason:      "legacy_runtime_model_switch_has_no_canonical_key",
	},
	{
		LegacyPath:  "schema_version",
		SourceKind:  SourceKindString,
		Disposition: DispositionSchema,
		Reason:      "legacy_document_schema_discriminator",
	},
	{
		LegacyPath:  "server.addr",
		SourceKind:  SourceKindString,
		TargetKey:   config.KeyServerListen,
		Transform:   transformIdentityString,
		Disposition: DispositionMapped,
	},
	{
		LegacyPath:  "server.state_dir",
		SourceKind:  SourceKindString,
		Disposition: DispositionDeferred,
		Reason:      "legacy_state_directory_is_not_sqlite_database_path",
	},
}

// CanonicalMappings returns a detached copy of the sole product-owned legacy
// mapping table. Callers cannot inject or mutate migration authority.
func CanonicalMappings() []Mapping {
	result := make([]Mapping, len(canonicalMappings))
	copy(result, canonicalMappings[:])
	return result
}

func canonicalMappingIndex() (map[string]Mapping, map[string]struct{}) {
	index := make(map[string]Mapping, len(canonicalMappings))
	prefixes := make(map[string]struct{})
	for _, mapping := range canonicalMappings {
		index[mapping.LegacyPath] = mapping
		parts := strings.Split(mapping.LegacyPath, ".")
		for end := 1; end < len(parts); end++ {
			prefixes[strings.Join(parts[:end], ".")] = struct{}{}
		}
	}
	return index, prefixes
}

func canonicalMappingSHA256() string {
	mappings := CanonicalMappings()
	payload, _ := json.Marshal(struct {
		SchemaVersion int       `json:"schema_version"`
		DocumentType  string    `json:"document_type"`
		Mappings      []Mapping `json:"mappings"`
	}{
		SchemaVersion: 1,
		DocumentType:  "orquesta.config.legacy-json-mappings.v1",
		Mappings:      mappings,
	})
	return bytesSHA256(payload)
}

func bytesSHA256(content []byte) string {
	digest := sha256.Sum256(content)
	return "sha256:" + hex.EncodeToString(digest[:])
}
