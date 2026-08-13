package orquesta_test

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"
)

const (
	legacyCMDSnapshotPath = "product/knowledge/legacy_cmd_go_function_snapshot_v1.jsonl"
	legacyCMDManifestPath = "product/knowledge/legacy_cmd_go_function_snapshot_v1.manifest.json"
	legacyCMDSourceRoot   = "cmd"
	legacyCMDNewPrefix    = "cmd/orquesta/"
)

type legacyCMDBuildPolicy struct {
	ExcludeTestSuffix              string   `json:"exclude_test_suffix"`
	ExcludedNewSurfacePrefix       string   `json:"excluded_new_surface_prefix"`
	ExcludedNonProductionBuildTags []string `json:"excluded_nonproduction_build_tags"`
	TrackedGoFiles                 int      `json:"tracked_go_file_count"`
	ExcludedNewSurfaceGoFiles      int      `json:"excluded_new_surface_go_file_count"`
	ExcludedTestFiles              int      `json:"excluded_test_file_count"`
	PlatformVariantsIncluded       bool     `json:"platform_variants_included"`
	PlatformConstrainedFiles       int      `json:"platform_constrained_file_count"`
	NonProductionExcludedFiles     int      `json:"nonproduction_build_tag_excluded_file_count"`
}

type legacyCMDPathCount struct {
	Path             string `json:"path"`
	ProductionFiles  int    `json:"production_file_count"`
	FunctionCount    int    `json:"function_count"`
	MethodCount      int    `json:"method_count"`
	DeclarationCount int    `json:"declaration_count"`
}

type legacyCMDSnapshotManifest struct {
	DocumentKind            string               `json:"document_kind"`
	SchemaVersion           int                  `json:"schema_version"`
	Algorithm               string               `json:"algorithm"`
	IdentityContract        string               `json:"identity_contract"`
	SourceRoot              string               `json:"source_root"`
	Scope                   string               `json:"scope"`
	BuildPolicy             legacyCMDBuildPolicy `json:"build_policy"`
	CensusSHA256            string               `json:"census_sha256"`
	GitIndexSnapshotSHA256  string               `json:"git_index_snapshot_sha256"`
	JSONLSHA256             string               `json:"jsonl_sha256"`
	JSONLBytes              int64                `json:"jsonl_bytes"`
	PackageCount            int                  `json:"package_count"`
	ProductionFileCount     int                  `json:"production_file_count"`
	FunctionCount           int                  `json:"function_count"`
	MethodCount             int                  `json:"method_count"`
	DeclarationCount        int                  `json:"declaration_count"`
	UniqueOccurrenceCount   int                  `json:"unique_occurrence_count"`
	UniqueDeclarationCount  int                  `json:"unique_declaration_count"`
	UniqueVariantCount      int                  `json:"unique_variant_count"`
	PathCounts              []legacyCMDPathCount `json:"path_counts"`
	ContainsBodies          bool                 `json:"contains_bodies"`
	SemanticAssessmentState string               `json:"semantic_assessment_state"`
	Authority               string               `json:"authority"`
	CanonicalStateChange    bool                 `json:"canonical_state_change"`
	ClaimsHistoricalGreen   bool                 `json:"claims_historical_green"`
	ClaimsAccreditation     bool                 `json:"claims_accreditation"`
}

type legacyCMDSelection struct {
	trackedGo, excludedNew, excludedTests int
	platform, excludedNonProduction       int
}

func TestLegacyCMDFunctionSnapshotIndexMatchesGitIndex(t *testing.T) {
	var manifest legacyCMDSnapshotManifest
	traceDecodeStrict(t, legacyCMDManifestPath, &manifest)
	if manifest.DocumentKind != "legacy_cmd_go_function_snapshot_index" ||
		manifest.SchemaVersion != 1 || manifest.Algorithm != "legacy-go-function-snapshot-index.v1" ||
		manifest.IdentityContract != "legacy-go-function-identity.v1-compatible" ||
		manifest.SourceRoot != legacyCMDSourceRoot ||
		manifest.Scope != "cmd/** excluding cmd/orquesta/**, _test.go and non-production-only build tags" ||
		manifest.BuildPolicy.ExcludeTestSuffix != "_test.go" ||
		manifest.BuildPolicy.ExcludedNewSurfacePrefix != legacyCMDNewPrefix ||
		!reflect.DeepEqual(manifest.BuildPolicy.ExcludedNonProductionBuildTags, []string{"ignore", "tools", "test", "integration"}) ||
		!manifest.BuildPolicy.PlatformVariantsIncluded || manifest.ContainsBodies ||
		manifest.SemanticAssessmentState != "structural_only_not_semantically_assessed" ||
		manifest.Authority != "derived_advisory_read_model" || manifest.CanonicalStateChange ||
		manifest.ClaimsHistoricalGreen || manifest.ClaimsAccreditation {
		t.Fatalf("manifiesto cmd inválido: %#v", manifest)
	}

	snapshot := traceLoadGitIndexSnapshot(t, ".", legacyCMDSourceRoot, func(path string) bool {
		return strings.HasSuffix(path, ".go") &&
			!strings.HasSuffix(path, manifest.BuildPolicy.ExcludeTestSuffix) &&
			!strings.HasPrefix(path, manifest.BuildPolicy.ExcludedNewSurfacePrefix)
	})
	entryByPath := make(map[string]traceGitIndexEntry, len(snapshot.Entries))
	selection := legacyCMDSelection{}
	for _, entry := range snapshot.Entries {
		entryByPath[entry.Path] = entry
		if !strings.HasSuffix(entry.Path, ".go") {
			continue
		}
		selection.trackedGo++
		if strings.HasPrefix(entry.Path, manifest.BuildPolicy.ExcludedNewSurfacePrefix) {
			selection.excludedNew++
			continue
		}
		if strings.HasSuffix(entry.Path, manifest.BuildPolicy.ExcludeTestSuffix) {
			selection.excludedTests++
		}
	}

	expected := make(map[string]legacyFunctionSnapshotRecord)
	acceptedPaths := make([]string, 0, len(snapshot.Contents))
	constraintsByPath := make(map[string][]string)
	pathCounts := make(map[string]*legacyCMDPathCount)
	packages := make(map[string]struct{})
	functions, methods := 0, 0
	for path, content := range snapshot.Contents {
		constraints := legacyCMDSourceConstraints(content)
		if legacyCMDNonProductionOnly(constraints, manifest.BuildPolicy.ExcludedNonProductionBuildTags) {
			selection.excludedNonProduction++
			continue
		}
		if len(constraints) != 0 {
			selection.platform++
		}
		constraintsByPath[path] = constraints
		acceptedPaths = append(acceptedPaths, path)
		entry, found := entryByPath[path]
		if !found {
			t.Fatalf("blob sin entrada Git: %s", path)
		}
		rows, packageName := legacyCMDExpectedRows(t, path, entry.OID, content)
		packages[filepath.ToSlash(filepath.Dir(path))] = struct{}{}
		prefix := strings.Join(strings.Split(path, "/")[:2], "/")
		count := pathCounts[prefix]
		if count == nil {
			count = &legacyCMDPathCount{Path: prefix}
			pathCounts[prefix] = count
		}
		count.ProductionFiles++
		for _, row := range rows {
			if row.PackageName != packageName {
				t.Fatalf("package inconsistente: %s", path)
			}
			if _, duplicate := expected[row.OccurrenceRef]; duplicate {
				t.Fatalf("occurrence AST duplicada: %s", row.OccurrenceRef)
			}
			expected[row.OccurrenceRef] = row
			switch row.SymbolKind {
			case "func":
				functions++
				count.FunctionCount++
			case "method":
				methods++
				count.MethodCount++
			default:
				t.Fatalf("symbol_kind inesperado: %s", row.SymbolKind)
			}
			count.DeclarationCount++
		}
	}
	sort.Strings(acceptedPaths)
	listing := legacyCMDAcceptedListing(t, acceptedPaths, entryByPath, constraintsByPath)
	policyJSON, err := json.Marshal(manifest.BuildPolicy)
	if err != nil {
		t.Fatal(err)
	}
	if manifest.GitIndexSnapshotSHA256 != legacyCMDDigest("orquesta.legacy-go-cmd-function-snapshot-index.v1", listing) ||
		manifest.CensusSHA256 != legacyCMDDigest("orquesta.legacy-go-cmd-function-census.v1", append(policyJSON, listing...)) {
		t.Fatal("digest de selección del Git index o censo cmd deriva")
	}

	raw := traceReadGitIndexOverlayFile(t, legacyCMDSnapshotPath, traceGitIndexSnapshot{})
	contentSHA := sha256.Sum256(raw)
	if len(raw) == 0 || raw[len(raw)-1] != '\n' || bytes.Contains(raw, []byte{'\r'}) ||
		bytes.Contains(raw, []byte("canonical_source")) || int64(len(raw)) != manifest.JSONLBytes ||
		"sha256:"+hex.EncodeToString(contentSHA[:]) != manifest.JSONLSHA256 {
		t.Fatal("JSONL cmd viola LF/body-free/digest/bytes")
	}
	actual := legacyCMDReadRows(t, raw)
	for occurrence, want := range expected {
		got, found := actual[occurrence]
		if !found {
			t.Fatalf("declaración ausente: %s %s#%s", occurrence, want.SourcePath, want.Name)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("drift blob/AST en %s#%s: got=%#v want=%#v", want.SourcePath, want.Name, got, want)
		}
	}
	uniqueDeclarations, uniqueVariants := map[string]struct{}{}, map[string]struct{}{}
	for _, row := range actual {
		uniqueDeclarations[row.DeclarationRef] = struct{}{}
		uniqueVariants[row.VariantRef] = struct{}{}
	}
	wantPathCounts := make([]legacyCMDPathCount, 0, len(pathCounts))
	for _, count := range pathCounts {
		wantPathCounts = append(wantPathCounts, *count)
	}
	sort.Slice(wantPathCounts, func(i, j int) bool { return wantPathCounts[i].Path < wantPathCounts[j].Path })
	if len(actual) != len(expected) || len(actual) != manifest.DeclarationCount ||
		len(acceptedPaths) != manifest.ProductionFileCount || functions != manifest.FunctionCount ||
		methods != manifest.MethodCount || len(packages) != manifest.PackageCount ||
		len(actual) != manifest.UniqueOccurrenceCount || len(uniqueDeclarations) != manifest.UniqueDeclarationCount ||
		len(uniqueVariants) != manifest.UniqueVariantCount || !reflect.DeepEqual(wantPathCounts, manifest.PathCounts) ||
		selection.trackedGo != manifest.BuildPolicy.TrackedGoFiles ||
		selection.excludedNew != manifest.BuildPolicy.ExcludedNewSurfaceGoFiles ||
		selection.excludedTests != manifest.BuildPolicy.ExcludedTestFiles ||
		selection.platform != manifest.BuildPolicy.PlatformConstrainedFiles ||
		selection.excludedNonProduction != manifest.BuildPolicy.NonProductionExcludedFiles {
		t.Fatalf("conteos cmd derivan: files=%d funcs=%d methods=%d declarations=%d selection=%#v manifest=%#v",
			len(acceptedPaths), functions, methods, len(actual), selection, manifest)
	}
	t.Logf("cmd legacy snapshot: files=%d funcs=%d methods=%d declarations=%d", len(acceptedPaths), functions, methods, len(actual))
}

func legacyCMDExpectedRows(t *testing.T, path, oid string, content []byte) ([]legacyFunctionSnapshotRecord, string) {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, content, parser.AllErrors|parser.ParseComments|parser.SkipObjectResolution)
	if err != nil || file == nil || file.Name == nil {
		t.Fatalf("parse fail-closed %s: %v", path, err)
	}
	var rows []legacyFunctionSnapshotRecord
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok {
			continue
		}
		bad := false
		ast.Inspect(function, func(node ast.Node) bool {
			switch node.(type) {
			case *ast.BadDecl, *ast.BadExpr, *ast.BadStmt:
				bad = true
				return false
			default:
				return !bad
			}
		})
		if function.Name == nil || bad {
			t.Fatalf("declaración AST inválida: %s", path)
		}
		start := fset.PositionFor(function.Pos(), false)
		end := fset.PositionFor(function.End(), false)
		if start.Offset < 0 || end.Offset < start.Offset || end.Offset > len(content) {
			t.Fatalf("rango AST inválido: %s", path)
		}
		canonicalFunction := *function
		canonicalFunction.Doc = nil
		canonical := legacyCMDFormat(t, fset, &canonicalFunction)
		signatureFunction := *function
		signatureFunction.Doc = nil
		signatureFunction.Body = nil
		signature := legacyCMDFormat(t, fset, &signatureFunction)
		kind, receiver := "func", ""
		if function.Recv != nil && len(function.Recv.List) > 0 {
			kind = "method"
			receiver = legacyCMDFormat(t, fset, function.Recv.List[0].Type)
		}
		bodySHA := ""
		if function.Body != nil {
			bodySHA = legacyCMDDigestStrings("orquesta.legacy-go-function-body.v1", legacyCMDFormat(t, fset, function.Body))
		}
		astSHA := legacyCMDDigestStrings("orquesta.legacy-go-function-ast.v1", canonical)
		variant := "go-function-variant:" + astSHA
		declarationRef := legacyCMDDigestStrings("orquesta.legacy-go-declaration.v2", oid,
			strconv.Itoa(start.Offset), strconv.Itoa(end.Offset), variant)
		rows = append(rows, legacyFunctionSnapshotRecord{
			SchemaVersion: 1, OccurrenceRef: legacyCMDDigestStrings("orquesta.legacy-go-function-occurrence.v1", legacyCMDSourceRoot, path, declarationRef),
			DeclarationRef: declarationRef, VariantRef: variant, SourceRoot: legacyCMDSourceRoot,
			SourcePath: path, GitBlobOID: oid, BlobDigest: legacyCMDDigest("orquesta.legacy-go-blob.v1", content),
			PackagePath: filepath.ToSlash(filepath.Dir(path)), PackageName: file.Name.Name,
			SymbolKind: kind, Name: function.Name.Name, Receiver: receiver, Signature: signature,
			SourceSHA: legacyCMDDigest("orquesta.legacy-go-function-source.v1", content[start.Offset:end.Offset]),
			ASTSHA:    astSHA, BodySHA: bodySHA, StartLine: start.Line, EndLine: end.Line,
		})
	}
	return rows, file.Name.Name
}

func legacyCMDReadRows(t *testing.T, raw []byte) map[string]legacyFunctionSnapshotRecord {
	t.Helper()
	rows := make(map[string]legacyFunctionSnapshotRecord)
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	scanner.Buffer(make([]byte, 64<<10), 4<<20)
	for line := 1; scanner.Scan(); line++ {
		content := append([]byte(nil), scanner.Bytes()...)
		traceRequireJSONWithoutDuplicateKeys(t, fmt.Sprintf("cmd snapshot line %d", line), content)
		decoder := json.NewDecoder(bytes.NewReader(content))
		decoder.DisallowUnknownFields()
		var row legacyFunctionSnapshotRecord
		if err := decoder.Decode(&row); err != nil {
			t.Fatalf("decode line %d: %v", line, err)
		}
		if err := decoder.Decode(&struct{}{}); err != io.EOF {
			t.Fatalf("contenido posterior line %d", line)
		}
		var canonical bytes.Buffer
		encoder := json.NewEncoder(&canonical)
		encoder.SetEscapeHTML(false)
		if err := encoder.Encode(row); err != nil || !bytes.Equal(bytes.TrimSuffix(canonical.Bytes(), []byte{'\n'}), content) {
			t.Fatalf("JSON no canónico line %d", line)
		}
		if row.SchemaVersion != 1 || row.SourceRoot != legacyCMDSourceRoot || row.OccurrenceRef == "" ||
			row.DeclarationRef == "" || row.VariantRef == "" || row.Signature == "" || row.BodySHA == "" ||
			strings.HasPrefix(row.SourcePath, legacyCMDNewPrefix) || strings.HasSuffix(row.SourcePath, "_test.go") {
			t.Fatalf("fila incompleta/fuera de scope line %d: %#v", line, row)
		}
		if _, duplicate := rows[row.OccurrenceRef]; duplicate {
			t.Fatalf("occurrence duplicada: %s", row.OccurrenceRef)
		}
		rows[row.OccurrenceRef] = row
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	return rows
}

func legacyCMDSourceConstraints(content []byte) []string {
	var constraints []string
	for index, raw := range strings.Split(string(content), "\n") {
		if index > 40 {
			break
		}
		line := strings.TrimSpace(raw)
		if strings.HasPrefix(line, "//go:build ") || strings.HasPrefix(line, "// +build ") {
			constraints = append(constraints, line)
		}
	}
	sort.Strings(constraints)
	return constraints
}

func legacyCMDNonProductionOnly(constraints, excludedTags []string) bool {
	for _, line := range constraints {
		expression := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(line, "//go:build"), "// +build"))
		for _, tag := range excludedTags {
			if expression == tag {
				return true
			}
		}
	}
	return false
}

func legacyCMDAcceptedListing(t *testing.T, paths []string, entries map[string]traceGitIndexEntry, constraints map[string][]string) []byte {
	t.Helper()
	var listing bytes.Buffer
	for _, path := range paths {
		entry, found := entries[path]
		if !found {
			t.Fatalf("entrada ausente: %s", path)
		}
		listing.WriteString(entry.Mode)
		listing.WriteByte('\t')
		listing.WriteString(entry.OID)
		listing.WriteByte('\t')
		listing.WriteString(path)
		listing.WriteByte(0)
		for _, constraint := range constraints[path] {
			listing.WriteString(constraint)
			listing.WriteByte(0)
		}
	}
	return listing.Bytes()
}

func legacyCMDFormat(t *testing.T, fset *token.FileSet, node ast.Node) string {
	t.Helper()
	var output bytes.Buffer
	if err := format.Node(&output, fset, node); err != nil {
		t.Fatal(err)
	}
	return output.String()
}

func legacyCMDDigest(domain string, content []byte) string {
	hash := sha256.New()
	legacyCMDWriteDigestField(hash, []byte(domain))
	legacyCMDWriteDigestField(hash, content)
	return "sha256:" + hex.EncodeToString(hash.Sum(nil))
}

func legacyCMDDigestStrings(domain string, values ...string) string {
	hash := sha256.New()
	legacyCMDWriteDigestField(hash, []byte(domain))
	for _, value := range values {
		legacyCMDWriteDigestField(hash, []byte(value))
	}
	return "sha256:" + hex.EncodeToString(hash.Sum(nil))
}

func legacyCMDWriteDigestField(writer io.Writer, content []byte) {
	var length [8]byte
	binary.BigEndian.PutUint64(length[:], uint64(len(content)))
	_, _ = writer.Write(length[:])
	_, _ = writer.Write(content)
}
