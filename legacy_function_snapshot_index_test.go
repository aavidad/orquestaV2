package orquesta_test

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"strconv"
	"strings"
	"testing"
)

const (
	legacyFunctionSnapshotPath    = "product/knowledge/legacy_go_function_snapshot_v1.jsonl"
	legacyFunctionManifestPath    = "product/knowledge/legacy_go_function_snapshot_v1.manifest.json"
	legacyFunctionAssessmentsPath = "product/knowledge/legacy_reuse_assessments_v1.jsonl"
)

type legacyFunctionSnapshotManifest struct {
	DocumentKind            string `json:"document_kind"`
	SchemaVersion           int    `json:"schema_version"`
	Algorithm               string `json:"algorithm"`
	SourceRoot              string `json:"source_root"`
	CensusSHA256            string `json:"census_sha256"`
	GitIndexSnapshotSHA256  string `json:"git_index_snapshot_sha256"`
	JSONLSHA256             string `json:"jsonl_sha256"`
	JSONLBytes              int64  `json:"jsonl_bytes"`
	ProductionFileCount     int    `json:"production_file_count"`
	FunctionCount           int    `json:"function_count"`
	MethodCount             int    `json:"method_count"`
	DeclarationCount        int    `json:"declaration_count"`
	UniqueOccurrenceCount   int    `json:"unique_occurrence_count"`
	UniqueDeclarationCount  int    `json:"unique_declaration_count"`
	UniqueVariantCount      int    `json:"unique_variant_count"`
	ContainsBodies          bool   `json:"contains_bodies"`
	SemanticAssessmentState string `json:"semantic_assessment_state"`
	Authority               string `json:"authority"`
	CanonicalStateChange    bool   `json:"canonical_state_change"`
	ClaimsAccreditation     bool   `json:"claims_accreditation"`
}

type legacyFunctionSnapshotRecord struct {
	SchemaVersion  int    `json:"schema_version"`
	OccurrenceRef  string `json:"occurrence_ref"`
	DeclarationRef string `json:"declaration_ref"`
	VariantRef     string `json:"variant_ref"`
	SourceRoot     string `json:"source_root"`
	SourcePath     string `json:"source_path"`
	GitBlobOID     string `json:"git_blob_oid"`
	BlobDigest     string `json:"blob_digest"`
	PackagePath    string `json:"package_path"`
	PackageName    string `json:"package_name"`
	SymbolKind     string `json:"symbol_kind"`
	Name           string `json:"name"`
	Receiver       string `json:"receiver,omitempty"`
	Signature      string `json:"signature"`
	SourceSHA      string `json:"source_sha256"`
	ASTSHA         string `json:"ast_sha256"`
	BodySHA        string `json:"body_sha256,omitempty"`
	StartLine      int    `json:"start_line"`
	EndLine        int    `json:"end_line"`
}

func TestLegacyFunctionSnapshotIndexMatchesGitIndex(t *testing.T) {
	var ledger traceLegacyGoLedger
	traceDecodeStrict(t, "product/traceability/legacy_go.json", &ledger)
	var manifest legacyFunctionSnapshotManifest
	traceDecodeStrict(t, legacyFunctionManifestPath, &manifest)
	if manifest.DocumentKind != "legacy_go_function_snapshot_index" ||
		manifest.SchemaVersion != 1 || manifest.SourceRoot != ledger.SourceRoot ||
		manifest.CensusSHA256 != ledger.Baseline.CensusSHA256 ||
		manifest.ContainsBodies || manifest.ClaimsAccreditation ||
		manifest.CanonicalStateChange || manifest.Authority != "derived_advisory_read_model" {
		t.Fatalf("manifiesto estructural inválido: %#v", manifest)
	}
	rawJSONL, err := os.ReadFile(legacyFunctionSnapshotPath)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(rawJSONL)
	if int64(len(rawJSONL)) != manifest.JSONLBytes ||
		"sha256:"+hex.EncodeToString(digest[:]) != manifest.JSONLSHA256 ||
		len(rawJSONL) == 0 || rawJSONL[len(rawJSONL)-1] != '\n' ||
		bytes.Contains(rawJSONL, []byte("canonical_source")) {
		t.Fatal("bytes, digest, LF o política body-free del índice no coinciden")
	}

	snapshot := traceLoadGitIndexSnapshot(t, ".", ledger.SourceRoot, func(path string) bool {
		return strings.HasSuffix(path, ".go") &&
			!strings.HasSuffix(path, ledger.SourcePolicy.ExcludeTestSuffix) &&
			!traceExcluded(path, false, ledger.SourcePolicy.ExactExclusions)
	})
	expected := legacySnapshotExpectedDeclarations(t, snapshot)
	actual, functions, methods := legacyReadSnapshotRecords(t, rawJSONL)
	if len(snapshot.Contents) != ledger.Baseline.ProductionFileCount ||
		len(snapshot.Contents) != manifest.ProductionFileCount ||
		len(actual) != len(expected) || len(actual) != manifest.DeclarationCount ||
		functions != manifest.FunctionCount || methods != manifest.MethodCount ||
		manifest.UniqueOccurrenceCount != len(actual) {
		t.Fatalf("conteos inconsistentes: files=%d declarations=%d/%d funcs=%d methods=%d manifest=%#v",
			len(snapshot.Contents), len(actual), len(expected), functions, methods, manifest)
	}
	for key := range expected {
		if _, found := actual[key]; !found {
			t.Fatalf("declaración ausente del índice: %s", key)
		}
	}
	legacyRequireAssessmentOccurrencesInSnapshot(t, actual)
	t.Logf("legacy functions: files=%d funcs=%d methods=%d declarations=%d",
		len(snapshot.Contents), functions, methods, len(actual))
}

func legacySnapshotExpectedDeclarations(
	t *testing.T,
	snapshot traceGitIndexSnapshot,
) map[string]struct{} {
	t.Helper()
	expected := make(map[string]struct{})
	for path, content := range snapshot.Contents {
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, content,
			parser.AllErrors|parser.ParseComments|parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		for _, declaration := range file.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || function.Name == nil {
				continue
			}
			kind := "func"
			receiver := ""
			if function.Recv != nil && len(function.Recv.List) > 0 {
				kind = "method"
				var rendered bytes.Buffer
				if err := format.Node(&rendered, fset, function.Recv.List[0].Type); err != nil {
					t.Fatal(err)
				}
				receiver = rendered.String()
			}
			start := fset.PositionFor(function.Pos(), false).Line
			end := fset.PositionFor(function.End(), false).Line
			key := legacyFunctionLocationKey(path, kind, receiver, function.Name.Name, start, end)
			if _, duplicate := expected[key]; duplicate {
				t.Fatalf("declaración AST duplicada: %s", key)
			}
			expected[key] = struct{}{}
		}
	}
	return expected
}

func legacyReadSnapshotRecords(
	t *testing.T,
	raw []byte,
) (map[string]legacyFunctionSnapshotRecord, int, int) {
	t.Helper()
	actual := make(map[string]legacyFunctionSnapshotRecord)
	occurrences := make(map[string]struct{})
	declarations := make(map[string]struct{})
	functions := 0
	methods := 0
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	scanner.Buffer(make([]byte, 64*1024), 4<<20)
	for line := 1; scanner.Scan(); line++ {
		content := append([]byte(nil), scanner.Bytes()...)
		traceRequireJSONWithoutDuplicateKeys(t, fmt.Sprintf("snapshot line %d", line), content)
		decoder := json.NewDecoder(bytes.NewReader(content))
		decoder.DisallowUnknownFields()
		var record legacyFunctionSnapshotRecord
		if err := decoder.Decode(&record); err != nil {
			t.Fatalf("decode snapshot line %d: %v", line, err)
		}
		if record.SchemaVersion != 1 || record.SourceRoot != "modulos" ||
			record.OccurrenceRef == "" || record.DeclarationRef == "" ||
			record.VariantRef == "" || record.SourcePath == "" ||
			record.Signature == "" || record.BodySHA == "" {
			t.Fatalf("registro incompleto en línea %d: %#v", line, record)
		}
		switch record.SymbolKind {
		case "func":
			functions++
		case "method":
			methods++
		default:
			t.Fatalf("symbol_kind inválido: %s", record.SymbolKind)
		}
		if _, duplicate := occurrences[record.OccurrenceRef]; duplicate {
			t.Fatalf("occurrence_ref duplicada: %s", record.OccurrenceRef)
		}
		if _, duplicate := declarations[record.DeclarationRef]; duplicate {
			t.Fatalf("declaration_ref duplicada: %s", record.DeclarationRef)
		}
		occurrences[record.OccurrenceRef] = struct{}{}
		declarations[record.DeclarationRef] = struct{}{}
		key := legacyFunctionLocationKey(record.SourcePath, record.SymbolKind,
			record.Receiver, record.Name, record.StartLine, record.EndLine)
		if _, duplicate := actual[key]; duplicate {
			t.Fatalf("slot de declaración duplicado: %s", key)
		}
		actual[key] = record
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	return actual, functions, methods
}

func legacyRequireAssessmentOccurrencesInSnapshot(
	t *testing.T,
	records map[string]legacyFunctionSnapshotRecord,
) {
	t.Helper()
	occurrences := make(map[string]struct{}, len(records))
	for _, record := range records {
		occurrences[record.OccurrenceRef] = struct{}{}
	}
	file, err := os.Open(legacyFunctionAssessmentsPath)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 4<<20)
	for scanner.Scan() {
		var row struct {
			CharacterizationRef string `json:"characterization_ref"`
			FunctionMapping     struct {
				Links []struct {
					OccurrenceRef string `json:"occurrence_ref"`
				} `json:"links"`
			} `json:"function_mapping"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &row); err != nil {
			t.Fatal(err)
		}
		for _, link := range row.FunctionMapping.Links {
			if _, found := occurrences[link.OccurrenceRef]; !found {
				t.Fatalf("assessment %s enlaza una aparición fuera del snapshot: %s",
					row.CharacterizationRef, link.OccurrenceRef)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
}

func legacyFunctionLocationKey(path, kind, receiver, name string, start, end int) string {
	return strings.Join([]string{path, kind, receiver, name, strconv.Itoa(start), strconv.Itoa(end)}, "\x00")
}
