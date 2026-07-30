// Este fichero define el grafo JSONL V2 y sus identidades criptográficas.
// No recorre Git ni escribe ficheros.
package main

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"hash"
	"reflect"
	"unicode/utf8"
)

const (
	schemaVersion       = 2
	inventoryAlgorithm  = "git-object-graph-go-declarations.v2"
	inventoryHashDomain = "orquesta.legacy-function-inventory.jsonl.v2"
	referenceHashDomain = "orquesta.legacy-function-inventory.refs.v2"
	recordSchemaDomain  = "orquesta.legacy-function-inventory.record-schema.v2"
)

var recordKinds = []string{
	"inventory_header",
	"reference",
	"commit",
	"tree_object",
	"tree_entry",
	"go_blob",
	"go_declaration",
	"function_variant",
	"parse_failure",
}

type record struct {
	RecordKind    string `json:"record_kind"`
	SchemaVersion int    `json:"schema_version,omitempty"`
	Algorithm     string `json:"algorithm,omitempty"`

	RefName  string   `json:"ref_name,omitempty"`
	ObjectID string   `json:"object_id,omitempty"`
	CommitID string   `json:"commit_id,omitempty"`
	Parents  []string `json:"parents,omitempty"`
	TreeID   string   `json:"tree_id,omitempty"`

	PathEncoding      string `json:"path_encoding,omitempty"`
	PathSegment       string `json:"path_segment,omitempty"`
	PathSegmentBase64 string `json:"path_segment_base64,omitempty"`
	FileMode          string `json:"file_mode,omitempty"`
	ChildType         string `json:"child_type,omitempty"`
	ChildObjectID     string `json:"child_object_id,omitempty"`

	BlobID   string `json:"blob_id,omitempty"`
	BlobSize int    `json:"blob_size,omitempty"`
	BlobSHA  string `json:"blob_sha256,omitempty"`

	Package          string   `json:"package,omitempty"`
	BuildConstraints []string `json:"build_constraints,omitempty"`
	Generated        bool     `json:"generated,omitempty"`

	DeclarationRef string `json:"declaration_ref,omitempty"`
	SymbolKind     string `json:"symbol_kind,omitempty"`
	Name           string `json:"name,omitempty"`
	Exported       bool   `json:"exported,omitempty"`
	Receiver       string `json:"receiver,omitempty"`
	Signature      string `json:"signature,omitempty"`
	StartOffset    int    `json:"start_offset,omitempty"`
	EndOffset      int    `json:"end_offset,omitempty"`
	StartLine      int    `json:"start_line,omitempty"`
	EndLine        int    `json:"end_line,omitempty"`
	SourceSHA      string `json:"source_sha256,omitempty"`
	DocSHA         string `json:"doc_sha256,omitempty"`
	ASTCanonical   string `json:"ast_canonical_version,omitempty"`
	ASTSHA         string `json:"ast_sha256,omitempty"`
	BodySHA        string `json:"body_sha256,omitempty"`
	VariantRef     string `json:"variant_ref,omitempty"`

	CanonicalSource string `json:"canonical_source,omitempty"`
	ErrorCode       string `json:"error_code,omitempty"`
	ErrorDetail     string `json:"error_detail,omitempty"`
}

type manifest struct {
	SchemaVersion       int            `json:"schema_version"`
	Algorithm           string         `json:"algorithm"`
	InventoryHashDomain string         `json:"inventory_hash_domain"`
	RecordSchemaSHA256  string         `json:"record_schema_sha256"`
	Repository          string         `json:"repository"`
	GitObjectFormat     string         `json:"git_object_format"`
	GoVersion           string         `json:"go_version"`
	InventorySHA256     string         `json:"inventory_sha256"`
	InventoryBytes      int64          `json:"inventory_bytes"`
	Counts              map[string]int `json:"counts"`
	RefSnapshotSHA256   string         `json:"ref_snapshot_sha256"`
}

type recordSchemaDescription struct {
	Kinds  []string `json:"kinds"`
	Fields []string `json:"fields"`
}

func encodePathSegment(path []byte) (encoding, text, encoded string) {
	if utf8.Valid(path) {
		return "utf8", string(path), ""
	}
	return "base64", "", base64.StdEncoding.EncodeToString(path)
}

func digest(domain string, content []byte) string {
	hash := sha256.New()
	writeDigestField(hash, []byte(domain))
	writeDigestField(hash, content)
	return "sha256:" + hex.EncodeToString(hash.Sum(nil))
}

func inventoryDigest(content []byte) string {
	hashValue := newInventoryHash()
	_, _ = hashValue.Write(content)
	return "sha256:" + hex.EncodeToString(hashValue.Sum(nil))
}

func newInventoryHash() hash.Hash {
	hashValue := sha256.New()
	writeDigestField(hashValue, []byte(inventoryHashDomain))
	return hashValue
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

func recordSchemaDigest() string {
	recordType := reflect.TypeOf(record{})
	fields := make([]string, 0, recordType.NumField())
	for index := 0; index < recordType.NumField(); index++ {
		field := recordType.Field(index)
		fields = append(fields, field.Name+"\x00"+string(field.Tag)+"\x00"+field.Type.String())
	}
	return digestJSON(recordSchemaDomain, recordSchemaDescription{
		Kinds: recordKinds, Fields: fields,
	})
}

func writeDigestField(hash interface{ Write([]byte) (int, error) }, content []byte) {
	var length [8]byte
	binary.BigEndian.PutUint64(length[:], uint64(len(content)))
	_, _ = hash.Write(length[:])
	_, _ = hash.Write(content)
}
