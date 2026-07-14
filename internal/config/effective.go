package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
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

// WriteEffective atomically writes redacted effective config to the canonical
// configured path using owner-only permissions.
func (s Snapshot) WriteEffective() error {
	content, err := s.EffectiveJSON()
	if err != nil {
		return err
	}
	maxExistingBytes := s.ConfigEffectiveMaxExistingBytes()
	if maxExistingBytes <= 0 || int64(len(content)) > maxExistingBytes {
		return &Error{Code: ErrorEffectiveWrite, Cause: errors.New("config.effective_output_exceeds_canonical_limit")}
	}
	path := s.ConfigEffectivePath()
	if path == "" {
		return &Error{Code: ErrorEffectiveWrite}
	}
	directory := filepath.Dir(path)
	if err := ensureEffectiveDirectory(directory); err != nil {
		return &Error{Code: ErrorEffectiveWrite, Cause: err}
	}
	if err := s.validateEffectiveDestination(path, maxExistingBytes); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(directory, ".effective-config-*")
	if err != nil {
		return &Error{Code: ErrorEffectiveWrite, Cause: err}
	}
	temporaryPath := temporary.Name()
	keepTemporary := true
	defer func() {
		_ = temporary.Close()
		if keepTemporary {
			_ = os.Remove(temporaryPath)
		}
	}()
	if _, err := temporary.Write(content); err != nil {
		return &Error{Code: ErrorEffectiveWrite, Cause: err}
	}
	if err := temporary.Sync(); err != nil {
		return &Error{Code: ErrorEffectiveWrite, Cause: err}
	}
	if err := temporary.Chmod(0o400); err != nil {
		return &Error{Code: ErrorEffectiveWrite, Cause: err}
	}
	if err := temporary.Sync(); err != nil {
		return &Error{Code: ErrorEffectiveWrite, Cause: err}
	}
	if err := temporary.Close(); err != nil {
		return &Error{Code: ErrorEffectiveWrite, Cause: err}
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return &Error{Code: ErrorEffectiveWrite, Cause: err}
	}
	keepTemporary = false
	if err := syncEffectiveDirectory(directory); err != nil {
		return &Error{Code: ErrorEffectiveWrite, Cause: err}
	}
	return nil
}

func ensureEffectiveDirectory(directory string) error {
	directory, err := filepath.Abs(filepath.Clean(directory))
	if err != nil {
		return err
	}
	missing := make([]string, 0)
	for current := directory; ; current = filepath.Dir(current) {
		info, statErr := os.Lstat(current)
		if statErr == nil {
			if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
				return errors.New("config.effective_directory_invalid")
			}
			break
		}
		if !errors.Is(statErr, os.ErrNotExist) {
			return statErr
		}
		missing = append(missing, current)
		parent := filepath.Dir(current)
		if parent == current {
			return errors.New("config.effective_directory_invalid")
		}
	}
	for index := len(missing) - 1; index >= 0; index-- {
		current := missing[index]
		if err := os.Mkdir(current, 0o700); err != nil {
			return err
		}
		if err := syncEffectiveDirectory(filepath.Dir(current)); err != nil {
			return err
		}
	}
	info, err := os.Lstat(directory)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return errors.New("config.effective_directory_invalid")
	}
	return nil
}

func syncEffectiveDirectory(directory string) error {
	opened, err := os.Open(directory)
	if err != nil {
		return err
	}
	defer opened.Close()
	return opened.Sync()
}

func (s Snapshot) validateEffectiveDestination(path string, maxExistingBytes int64) error {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return &Error{Code: ErrorEffectiveWrite, Cause: err}
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || info.Mode().Perm()&0o077 != 0 ||
		info.Size() <= 0 || info.Size() > maxExistingBytes {
		return &Error{Code: ErrorEffectiveWrite, Cause: errors.New("config.effective_destination_unrecognized")}
	}
	opened, err := os.Open(path)
	if err != nil {
		return &Error{Code: ErrorEffectiveWrite, Cause: err}
	}
	payload, err := io.ReadAll(io.LimitReader(opened, maxExistingBytes+1))
	closeErr := opened.Close()
	if err != nil || closeErr != nil || int64(len(payload)) > maxExistingBytes {
		return &Error{Code: ErrorEffectiveWrite, Cause: errors.Join(err, closeErr, errors.New("config.effective_destination_unrecognized"))}
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	var previous effectiveDocument
	if err := decoder.Decode(&previous); err != nil {
		return &Error{Code: ErrorEffectiveWrite, Cause: errors.New("config.effective_destination_unrecognized")}
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return &Error{Code: ErrorEffectiveWrite, Cause: errors.New("config.effective_destination_unrecognized")}
	}
	if previous.DocumentType != effectiveDocumentType || previous.SchemaVersion != s.SchemaVersion() ||
		strings.TrimSpace(previous.RegistryRevision) == "" || !strings.HasPrefix(previous.SnapshotHash, "sha256:") ||
		len(previous.Entries) == 0 {
		return &Error{Code: ErrorEffectiveWrite, Cause: errors.New("config.effective_destination_unrecognized")}
	}
	return nil
}
