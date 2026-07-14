package config

import (
	"encoding/json"
)

const redactedValue = "[REDACTED]"
const effectiveDocumentType = "orquesta.effective_config"

type effectiveDocument struct {
	DocumentType     string           `json:"document_type"`
	SchemaVersion    int              `json:"schema_version"`
	RegistryRevision string           `json:"registry_revision"`
	SnapshotHash     string           `json:"snapshot_hash"`
	Entries          []effectiveEntry `json:"entries"`
}

type effectiveEntry struct {
	Key             Key      `json:"key"`
	Value           any      `json:"value"`
	Source          Source   `json:"source"`
	Type            string   `json:"type"`
	SemanticRef     string   `json:"semantic_ref"`
	Sensitive       bool     `json:"sensitive"`
	Scope           string   `json:"scope"`
	RestartRequired bool     `json:"restart_required"`
	EnvAlias        string   `json:"env_alias"`
	ValidatorIDs    []string `json:"validator_ids"`
	Minimum         *int64   `json:"minimum,omitempty"`
	Maximum         *int64   `json:"maximum,omitempty"`
}

// EffectiveJSON returns deterministic, fully resolved and redacted output.
func (s Snapshot) EffectiveJSON() ([]byte, error) {
	document := s.effectiveDocument()
	content, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return nil, &Error{Code: ErrorEffectiveWrite, Cause: err}
	}
	return append(content, '\n'), nil
}

func (s Snapshot) effectiveDocument() effectiveDocument {
	document := effectiveDocument{
		DocumentType:     effectiveDocumentType,
		SchemaVersion:    s.SchemaVersion(),
		RegistryRevision: s.RegistryRevision(),
		SnapshotHash:     s.Hash(),
		Entries:          make([]effectiveEntry, 0, len(s.entries)),
	}
	for _, entry := range s.entries {
		value := canonicalValue(entry.value)
		if entry.metadata.Sensitive {
			value = redactedValue
		}
		document.Entries = append(document.Entries, effectiveEntry{
			Key:             entry.key,
			Value:           value,
			Source:          entry.metadata.Source,
			Type:            entry.metadata.Type,
			SemanticRef:     entry.metadata.SemanticRef,
			Sensitive:       entry.metadata.Sensitive,
			Scope:           entry.metadata.Scope,
			RestartRequired: entry.metadata.RestartRequired,
			EnvAlias:        entry.metadata.EnvAlias,
			ValidatorIDs:    append([]string(nil), entry.metadata.ValidatorIDs...),
			Minimum:         cloneInt64Pointer(entry.metadata.Minimum),
			Maximum:         cloneInt64Pointer(entry.metadata.Maximum),
		})
	}
	return document
}

// MarshalJSON deliberately projects Snapshot through the redacted effective
// representation.
func (s Snapshot) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.effectiveDocument())
}
