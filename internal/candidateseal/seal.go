// Package candidateseal builds and verifies the non-physical Gate A subject.
// It deliberately knows nothing about KVM, providers, receipts, or Gate B.
package candidateseal

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"path"
	"sort"
	"strings"
)

const (
	Schema = "orquesta.candidate-gate-a.v1"
	Gate   = "A"
)

var (
	ErrInvalidInput    = errors.New("candidate_seal.invalid_input")
	ErrInvalidManifest = errors.New("candidate_seal.invalid_manifest")
	ErrDigestMismatch  = errors.New("candidate_seal.digest_mismatch")
)

// SourceFile is one regular file in the candidate source tree. Mode contains
// only permission bits; timestamps, owners and host paths are not subjects.
type SourceFile struct {
	Path    string
	Mode    uint32
	Content []byte
}

// Input contains the exact bytes that make up one Orquesta candidate.
// Nil binary/config slices mean that the required subject is missing.
type Input struct {
	SourceFiles     []SourceFile
	Binary          []byte
	EffectiveConfig []byte
}

type Subject struct {
	SHA256 string `json:"sha256"`
	Bytes  uint64 `json:"bytes"`
}

type SourceTreeSubject struct {
	SHA256 string `json:"sha256"`
	Files  uint64 `json:"files"`
	Bytes  uint64 `json:"bytes"`
}

// Manifest is the immutable Gate A subject. ManifestSHA256 seals all preceding
// fields and is calculated with that field empty, avoiding self-reference.
type Manifest struct {
	Schema          string            `json:"schema"`
	Gate            string            `json:"gate"`
	SourceTree      SourceTreeSubject `json:"source_tree"`
	Binary          Subject           `json:"binary"`
	EffectiveConfig Subject           `json:"effective_config"`
	CandidateSHA256 string            `json:"candidate_sha256"`
	ManifestSHA256  string            `json:"manifest_sha256"`
}

func Build(input Input) (Manifest, error) {
	files, source, sourceBytes, err := sourceTreeDigest(input.SourceFiles)
	if err != nil {
		return Manifest{}, err
	}
	if input.Binary == nil || input.EffectiveConfig == nil {
		return Manifest{}, fmt.Errorf("%w: required subject missing", ErrInvalidInput)
	}
	if err := validateEffectiveConfig(input.EffectiveConfig); err != nil {
		return Manifest{}, err
	}

	manifest := Manifest{
		Schema:          Schema,
		Gate:            Gate,
		SourceTree:      SourceTreeSubject{SHA256: source, Files: uint64(len(files)), Bytes: sourceBytes},
		Binary:          subject(input.Binary),
		EffectiveConfig: subject(input.EffectiveConfig),
	}
	manifest.CandidateSHA256 = candidateDigest(manifest)
	manifest.ManifestSHA256 = manifestDigest(manifest)
	return manifest, nil
}

func Verify(manifest Manifest, input Input) error {
	if err := validateManifest(manifest); err != nil {
		return err
	}
	expected, err := Build(input)
	if err != nil {
		return err
	}
	if manifest != expected {
		return ErrDigestMismatch
	}
	return nil
}

// Encode returns the stable JSON artifact representation.
func Encode(manifest Manifest) ([]byte, error) {
	if err := validateManifest(manifest); err != nil {
		return nil, err
	}
	encoded, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidManifest, err)
	}
	return append(encoded, '\n'), nil
}

// Decode rejects unknown fields and trailing data before validating the seal.
func Decode(encoded []byte) (Manifest, error) {
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	var manifest Manifest
	if err := decoder.Decode(&manifest); err != nil {
		return Manifest{}, fmt.Errorf("%w: %v", ErrInvalidManifest, err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return Manifest{}, fmt.Errorf("%w: trailing data", ErrInvalidManifest)
	}
	if err := validateManifest(manifest); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

func validateManifest(manifest Manifest) error {
	if manifest.Schema != Schema || manifest.Gate != Gate ||
		manifest.SourceTree.Files == 0 || !validDigest(manifest.SourceTree.SHA256) ||
		!validDigest(manifest.Binary.SHA256) || !validDigest(manifest.EffectiveConfig.SHA256) ||
		!validDigest(manifest.CandidateSHA256) || !validDigest(manifest.ManifestSHA256) {
		return ErrInvalidManifest
	}
	if manifest.CandidateSHA256 != candidateDigest(manifest) || manifest.ManifestSHA256 != manifestDigest(manifest) {
		return ErrDigestMismatch
	}
	return nil
}

func sourceTreeDigest(input []SourceFile) ([]SourceFile, string, uint64, error) {
	if len(input) == 0 {
		return nil, "", 0, fmt.Errorf("%w: source tree missing", ErrInvalidInput)
	}
	files := append([]SourceFile(nil), input...)
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	h := sha256.New()
	writeField(h, []byte("orquesta.source-tree.v1"))
	var total uint64
	for index, file := range files {
		if file.Content == nil || !validSourcePath(file.Path) || file.Mode > 0o7777 ||
			(index > 0 && files[index-1].Path == file.Path) {
			return nil, "", 0, fmt.Errorf("%w: invalid source file", ErrInvalidInput)
		}
		writeField(h, []byte(file.Path))
		var mode [4]byte
		binary.BigEndian.PutUint32(mode[:], file.Mode)
		writeField(h, mode[:])
		writeField(h, file.Content)
		if uint64(len(file.Content)) > math.MaxUint64-total {
			return nil, "", 0, fmt.Errorf("%w: source tree size overflow", ErrInvalidInput)
		}
		total += uint64(len(file.Content))
	}
	return files, "sha256:" + hex.EncodeToString(h.Sum(nil)), total, nil
}

func validSourcePath(value string) bool {
	return value != "" && value != "." && !strings.HasPrefix(value, "/") &&
		!strings.Contains(value, "\\") && path.Clean(value) == value &&
		!strings.HasPrefix(value, "../")
}

func subject(content []byte) Subject {
	digest := sha256.Sum256(content)
	return Subject{SHA256: "sha256:" + hex.EncodeToString(digest[:]), Bytes: uint64(len(content))}
}

func candidateDigest(manifest Manifest) string {
	h := sha256.New()
	writeField(h, []byte("orquesta.candidate-gate-a.subject.v1"))
	writeField(h, []byte(manifest.SourceTree.SHA256))
	writeField(h, []byte(manifest.Binary.SHA256))
	writeField(h, []byte(manifest.EffectiveConfig.SHA256))
	return "sha256:" + hex.EncodeToString(h.Sum(nil))
}

func manifestDigest(manifest Manifest) string {
	h := sha256.New()
	writeField(h, []byte("orquesta.candidate-gate-a.manifest.v1"))
	for _, value := range []string{
		manifest.Schema, manifest.Gate, manifest.SourceTree.SHA256,
		fmt.Sprintf("%d", manifest.SourceTree.Files), fmt.Sprintf("%d", manifest.SourceTree.Bytes),
		manifest.Binary.SHA256, fmt.Sprintf("%d", manifest.Binary.Bytes),
		manifest.EffectiveConfig.SHA256, fmt.Sprintf("%d", manifest.EffectiveConfig.Bytes),
		manifest.CandidateSHA256,
	} {
		writeField(h, []byte(value))
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil))
}

type writer interface{ Write([]byte) (int, error) }

func writeField(w writer, value []byte) {
	var size [8]byte
	binary.BigEndian.PutUint64(size[:], uint64(len(value)))
	_, _ = w.Write(size[:])
	_, _ = w.Write(value)
}

func validDigest(value string) bool {
	if len(value) != len("sha256:")+sha256.Size*2 || !strings.HasPrefix(value, "sha256:") || strings.ToLower(value) != value {
		return false
	}
	decoded, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	return err == nil && len(decoded) == sha256.Size
}

type effectiveConfigDocument struct {
	DocumentType     string                 `json:"document_type"`
	SchemaVersion    int                    `json:"schema_version"`
	RegistryRevision string                 `json:"registry_revision"`
	SnapshotHash     string                 `json:"snapshot_hash"`
	Entries          []effectiveConfigEntry `json:"entries"`
}

type effectiveConfigEntry struct {
	Key             string          `json:"key"`
	Value           json.RawMessage `json:"value"`
	Source          string          `json:"source"`
	Type            string          `json:"type"`
	SemanticRef     string          `json:"semantic_ref"`
	Sensitive       bool            `json:"sensitive"`
	Scope           string          `json:"scope"`
	RestartRequired bool            `json:"restart_required"`
	EnvAlias        string          `json:"env_alias"`
	ValidatorIDs    []string        `json:"validator_ids"`
	Minimum         *int64          `json:"minimum,omitempty"`
	Maximum         *int64          `json:"maximum,omitempty"`
}

func validateEffectiveConfig(content []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	var document effectiveConfigDocument
	if err := decoder.Decode(&document); err != nil {
		return fmt.Errorf("%w: effective config", ErrInvalidInput)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF || document.DocumentType != "orquesta.effective_config" ||
		document.SchemaVersion <= 0 || strings.TrimSpace(document.RegistryRevision) == "" ||
		!validDigest(document.SnapshotHash) || len(document.Entries) == 0 {
		return fmt.Errorf("%w: effective config", ErrInvalidInput)
	}
	keys := make(map[string]struct{}, len(document.Entries))
	for _, entry := range document.Entries {
		if strings.TrimSpace(entry.Key) == "" || strings.TrimSpace(entry.Source) == "" ||
			strings.TrimSpace(entry.Type) == "" || strings.TrimSpace(entry.SemanticRef) == "" ||
			strings.TrimSpace(entry.Scope) == "" || entry.Value == nil {
			return fmt.Errorf("%w: effective config entry", ErrInvalidInput)
		}
		if _, exists := keys[entry.Key]; exists {
			return fmt.Errorf("%w: duplicate effective config entry", ErrInvalidInput)
		}
		keys[entry.Key] = struct{}{}
		if entry.Sensitive {
			var value string
			if json.Unmarshal(entry.Value, &value) != nil || value != "[REDACTED]" {
				return fmt.Errorf("%w: effective config not redacted", ErrInvalidInput)
			}
		}
	}
	return nil
}
