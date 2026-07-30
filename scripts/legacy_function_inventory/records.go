// Este fichero define el contrato JSONL y su identidad criptográfica estable.
// No recorre Git ni escribe ficheros: normaliza, ordena y resume los hechos
// producidos por el coordinador.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strconv"
	"strings"
)

type record struct {
	RecordKind string `json:"record_kind"`

	RefName  string   `json:"ref_name,omitempty"`
	ObjectID string   `json:"object_id,omitempty"`
	CommitID string   `json:"commit_id,omitempty"`
	TreeID   string   `json:"tree_id,omitempty"`
	Parents  []string `json:"parents,omitempty"`

	Path     string `json:"path,omitempty"`
	FileMode string `json:"file_mode,omitempty"`
	BlobID   string `json:"blob_id,omitempty"`
	BlobSize int    `json:"blob_size,omitempty"`
	BlobSHA  string `json:"blob_sha256,omitempty"`

	Package          string   `json:"package,omitempty"`
	BuildConstraints []string `json:"build_constraints,omitempty"`
	TestFile         bool     `json:"test_file,omitempty"`
	Generated        bool     `json:"generated,omitempty"`

	SymbolKind      string `json:"symbol_kind,omitempty"`
	Name            string `json:"name,omitempty"`
	Exported        bool   `json:"exported,omitempty"`
	Receiver        string `json:"receiver,omitempty"`
	Signature       string `json:"signature,omitempty"`
	StartOffset     int    `json:"start_offset,omitempty"`
	EndOffset       int    `json:"end_offset,omitempty"`
	StartLine       int    `json:"start_line,omitempty"`
	EndLine         int    `json:"end_line,omitempty"`
	SourceSHA       string `json:"source_sha256,omitempty"`
	DocSHA          string `json:"doc_sha256,omitempty"`
	ASTCanonical    string `json:"ast_canonical_version,omitempty"`
	ASTSHA          string `json:"ast_sha256,omitempty"`
	BodySHA         string `json:"body_sha256,omitempty"`
	VariantRef      string `json:"variant_ref,omitempty"`
	OccurrenceRef   string `json:"occurrence_ref,omitempty"`
	CanonicalSource string `json:"canonical_source,omitempty"`

	ErrorCode   string `json:"error_code,omitempty"`
	ErrorDetail string `json:"error_detail,omitempty"`
}

type manifest struct {
	SchemaVersion     int            `json:"schema_version"`
	Algorithm         string         `json:"algorithm"`
	Repository        string         `json:"repository"`
	GitObjectFormat   string         `json:"git_object_format"`
	GoVersion         string         `json:"go_version"`
	InventorySHA256   string         `json:"inventory_sha256"`
	Counts            map[string]int `json:"counts"`
	RefSnapshotSHA256 string         `json:"ref_snapshot_sha256"`
}

func sortRecords(records []record) {
	sort.Slice(records, func(i, j int) bool {
		left, right := records[i], records[j]
		leftKey := strings.Join([]string{
			left.RecordKind, left.RefName, left.CommitID, left.TreeID, left.Path,
			left.BlobID, left.VariantRef, strconv.Itoa(left.StartOffset),
		}, "\x00")
		rightKey := strings.Join([]string{
			right.RecordKind, right.RefName, right.CommitID, right.TreeID, right.Path,
			right.BlobID, right.VariantRef, strconv.Itoa(right.StartOffset),
		}, "\x00")
		return leftKey < rightKey
	})
}

func digest(domain string, content []byte) string {
	hash := sha256.New()
	writeDigestField(hash, []byte(domain))
	writeDigestField(hash, content)
	return "sha256:" + hex.EncodeToString(hash.Sum(nil))
}

func digestStrings(domain string, values ...string) string {
	hash := sha256.New()
	writeDigestField(hash, []byte(domain))
	for _, value := range values {
		writeDigestField(hash, []byte(value))
	}
	return "sha256:" + hex.EncodeToString(hash.Sum(nil))
}

func digestJSON(domain string, value any) string {
	content, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return digest(domain, content)
}

func writeDigestField(hash interface{ Write([]byte) (int, error) }, content []byte) {
	var length [8]byte
	for index := 7; index >= 0; index-- {
		length[index] = byte(len(content))
		contentLength := len(content) >> (8 * (7 - index))
		length[index] = byte(contentLength)
	}
	_, _ = hash.Write(length[:])
	_, _ = hash.Write(content)
}
