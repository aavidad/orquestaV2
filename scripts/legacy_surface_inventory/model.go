// Este fichero define el grafo V4 y los tipos internos del inventario.
package main

import (
	"bufio"
	"bytes"
	"hash"
	"io"
	"os"
	"os/exec"
)

const (
	schemaVersion           = 4
	blobPrefixLimit         = 128
	inventoryAlgorithm      = "orquesta.legacy-surface-object-graph.v4"
	inventoryHashDomain     = "orquesta.legacy-surface-inventory.jsonl.v4"
	referenceHashDomain     = "orquesta.legacy-surface-inventory.refs.v4"
	recordSchemaDomain      = "orquesta.legacy-surface-inventory.record-schema.v4"
	classificationAlgorithm = "orquesta.legacy-surface-classification.v4"
	vendoredReviewDuty      = "revisar_parches_conexiones_y_licencias_propias"
)

var recordKinds = []string{
	"cabecera_inventario",
	"referencia",
	"confirmacion",
	"objeto_arbol",
	"entrada_arbol",
	"hechos_blob",
	"fallo",
}

type options struct {
	repository string
	jsonl      string
	manifest   string

	gitProcessStarted      func()
	renameFile             func(string, string) error
	interruptAfterManifest func() error
}

type record struct {
	RecordKind    string `json:"record_kind"`
	SchemaVersion int    `json:"schema_version,omitempty"`
	Algorithm     string `json:"algorithm,omitempty"`

	RefName    string   `json:"ref_name,omitempty"`
	ObjectID   string   `json:"object_id,omitempty"`
	ObjectType string   `json:"object_type,omitempty"`
	TargetID   string   `json:"target_object_id,omitempty"`
	TargetType string   `json:"target_object_type,omitempty"`
	CommitID   string   `json:"commit_id,omitempty"`
	TreeID     string   `json:"tree_id,omitempty"`
	Parents    []string `json:"parents,omitempty"`

	PathEncoding      string `json:"path_encoding,omitempty"`
	PathSegment       string `json:"path_segment,omitempty"`
	PathSegmentBase64 string `json:"path_segment_base64,omitempty"`
	FileMode          string `json:"file_mode,omitempty"`
	ChildType         string `json:"child_type,omitempty"`
	ChildObjectID     string `json:"child_object_id,omitempty"`
	ExclusionCause    string `json:"exclusion_cause,omitempty"`
	ReviewObligation  string `json:"review_obligation,omitempty"`

	BlobID              string          `json:"blob_id,omitempty"`
	BlobSize            int64           `json:"blob_size,omitempty"`
	BlobSHA             string          `json:"blob_sha256,omitempty"`
	ClassificationRef   string          `json:"classification_ref,omitempty"`
	Families            []surfaceFamily `json:"families,omitempty"`
	DetectedTypes       []string        `json:"detected_types,omitempty"`
	FileModes           []string        `json:"file_modes,omitempty"`
	Encoding            string          `json:"encoding,omitempty"`
	LineCount           int64           `json:"line_count,omitempty"`
	Summaries           []string        `json:"summaries,omitempty"`
	ClassificationCount int             `json:"classification_count,omitempty"`

	ErrorCode   string `json:"error_code,omitempty"`
	ErrorDetail string `json:"error_detail,omitempty"`
}

type manifest struct {
	SchemaVersion           int            `json:"schema_version"`
	Algorithm               string         `json:"algorithm"`
	InventoryHashDomain     string         `json:"inventory_hash_domain"`
	RecordSchemaSHA256      string         `json:"record_schema_sha256"`
	ClassificationAlgorithm string         `json:"classification_algorithm"`
	Repository              string         `json:"repository"`
	GitObjectFormat         string         `json:"git_object_format"`
	InventorySHA256         string         `json:"inventory_sha256"`
	InventoryBytes          int64          `json:"inventory_bytes"`
	RefSnapshotSHA256       string         `json:"ref_snapshot_sha256"`
	Counts                  map[string]int `json:"counts"`
}

type refInfo struct {
	Name       string `json:"name"`
	Object     string `json:"object"`
	ObjectType string `json:"object_type"`
	Target     string `json:"target"`
	TargetType string `json:"target_type"`
}

type historyEntry struct {
	tree    string
	parents []string
}

type treeEntry struct {
	mode string
	oid  string
	name []byte
}

type gitBatch struct {
	command *exec.Cmd
	input   io.WriteCloser
	output  *bufio.Reader
	stderr  bytes.Buffer
	closed  bool
}

type gitObjectHeader struct {
	oid     string
	kind    string
	size    int64
	missing bool
}

type blobFacts struct {
	size         int64
	sha256       string
	prefix       []byte
	encoding     string
	lineCount    int64
	trailingByte byte
}

type blobAggregate struct {
	facts           blobFacts
	families        map[surfaceFamily]struct{}
	detectedTypes   map[string]struct{}
	fileModes       map[string]struct{}
	summaries       map[string]struct{}
	classifications map[string]struct{}
}

type utf8Validator struct {
	valid bool
	carry []byte
}

type inventoryWriter struct {
	file    *os.File
	name    string
	hasher  hash.Hash
	encoder interface{ Encode(any) error }
	counts  map[string]int
	sealed  bool
}

type classification struct {
	families []surfaceFamily
}

type surfaceFamily struct {
	Family  string `json:"family"`
	Subtype string `json:"subtype"`
}
