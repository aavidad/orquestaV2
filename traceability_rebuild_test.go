package orquesta_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

type traceLegacyGoLedger struct {
	DocumentKind  string `json:"document_kind"`
	SchemaVersion int    `json:"schema_version"`
	SourceRoot    string `json:"source_root"`
	SourcePolicy  struct {
		Include           string                `json:"include"`
		ExcludeTestSuffix string                `json:"exclude_test_suffix"`
		ExactExclusions   []traceExactExclusion `json:"exact_exclusions"`
		SymbolKinds       []string              `json:"symbol_kinds"`
	} `json:"source_policy"`
	Baseline    traceGoBaseline   `json:"baseline"`
	ModuleRules []traceModuleRule `json:"module_rules"`
}

type traceExactExclusion struct {
	Path   string `json:"path"`
	Kind   string `json:"kind"`
	Reason string `json:"reason"`
}

type traceGoBaseline struct {
	ModuleCount         int    `json:"module_count"`
	PackageCount        int    `json:"package_count"`
	ProductionFileCount int    `json:"production_file_count"`
	SymbolCount         int    `json:"symbol_count"`
	ModuleSetSHA256     string `json:"module_set_sha256"`
	CensusSHA256        string `json:"census_sha256"`
}

type traceModuleRule struct {
	ID                       string   `json:"id"`
	Paths                    []string `json:"paths"`
	Disposition              string   `json:"disposition"`
	CapabilityIDs            []string `json:"capability_ids"`
	Reason                   string   `json:"reason"`
	CharacterizationRequired bool     `json:"characterization_required"`
}

type tracePendingSourceLedger struct {
	DocumentKind  string               `json:"document_kind"`
	SchemaVersion int                  `json:"schema_version"`
	Sources       []tracePendingSource `json:"sources"`
}

type tracePendingSource struct {
	ID                string   `json:"id"`
	Kind              string   `json:"kind"`
	Status            string   `json:"status"`
	ExactPaths        []string `json:"exact_paths"`
	PathPatterns      []string `json:"path_patterns"`
	ExpectedFileCount int      `json:"expected_file_count"`
	ManifestSHA256    string   `json:"manifest_sha256"`
	NextGate          string   `json:"next_gate"`
}

type traceRoadmap struct {
	CapabilityEntries []struct {
		ID       string `json:"id"`
		Decision string `json:"decision"`
	} `json:"capability_entries"`
}

type traceComputedCensus struct {
	Baseline traceGoBaseline
	Modules  []string
}

func TestTraceabilityRebuildLegacyGoCensus(t *testing.T) {
	var ledger traceLegacyGoLedger
	traceDecodeStrict(t, "product/traceability/legacy_go.json", &ledger)
	if ledger.DocumentKind != "legacy_go_census" || ledger.SchemaVersion != 1 || ledger.SourceRoot != "modulos" {
		t.Fatalf("invalid legacy Go ledger header: kind=%q schema=%d root=%q", ledger.DocumentKind, ledger.SchemaVersion, ledger.SourceRoot)
	}
	if ledger.SourcePolicy.Include != "all_descendant_go_files_of_exact_module_paths" ||
		ledger.SourcePolicy.ExcludeTestSuffix != "_test.go" ||
		!reflect.DeepEqual(ledger.SourcePolicy.SymbolKinds, []string{"const", "var", "type", "func", "method"}) {
		t.Fatalf("unexpected census policy: %#v", ledger.SourcePolicy)
	}
	wantExclusions := []traceExactExclusion{
		{
			Path:   "modulos/orquesta-estado-vivo/testdeps/rapid",
			Kind:   "subtree",
			Reason: "Test dependency vendored only through a replace directive; not Orquesta production code.",
		},
		{
			Path:   "modulos/orquesta-server/status_tracker_diagnostics_v0.go",
			Kind:   "file",
			Reason: "Source is explicitly guarded with the build tag ignore and cannot enter a production build.",
		},
	}
	if !reflect.DeepEqual(ledger.SourcePolicy.ExactExclusions, wantExclusions) {
		t.Fatalf("exact exclusions changed without a census policy review: %#v", ledger.SourcePolicy.ExactExclusions)
	}

	accepted := traceAcceptedCapabilities(t)
	mappedModules := make(map[string]string)
	ruleIDs := make(map[string]struct{})
	for _, rule := range ledger.ModuleRules {
		if rule.ID == "" || strings.TrimSpace(rule.Reason) == "" || !rule.CharacterizationRequired {
			t.Fatalf("incomplete module rule: %#v", rule)
		}
		if _, duplicate := ruleIDs[rule.ID]; duplicate {
			t.Fatalf("duplicate module rule id %q", rule.ID)
		}
		ruleIDs[rule.ID] = struct{}{}
		if rule.Disposition != "supersede" && rule.Disposition != "retire" {
			t.Fatalf("rule %q has invalid disposition %q", rule.ID, rule.Disposition)
		}
		if rule.Disposition == "supersede" && len(rule.CapabilityIDs) == 0 {
			t.Fatalf("supersession rule %q has no capability", rule.ID)
		}
		if !sort.StringsAreSorted(rule.Paths) {
			t.Fatalf("rule %q paths are not sorted", rule.ID)
		}
		for _, capabilityID := range rule.CapabilityIDs {
			if _, ok := accepted[capabilityID]; !ok {
				t.Errorf("rule %q references non-accepted capability %q", rule.ID, capabilityID)
			}
		}
		for _, modulePath := range rule.Paths {
			if filepath.Clean(modulePath) != modulePath || !strings.HasPrefix(modulePath, "modulos/orquesta-") || strings.ContainsAny(modulePath, "*?[") {
				t.Fatalf("rule %q uses non-exact module path %q", rule.ID, modulePath)
			}
			if previous, duplicate := mappedModules[modulePath]; duplicate {
				t.Fatalf("module %q mapped by both %q and %q", modulePath, previous, rule.ID)
			}
			mappedModules[modulePath] = rule.ID
		}
	}

	census := traceComputeLegacyGoCensus(t, ledger)
	unmapped := make([]string, 0)
	for _, modulePath := range census.Modules {
		if _, ok := mappedModules[modulePath]; !ok {
			unmapped = append(unmapped, modulePath)
		}
	}
	for modulePath := range mappedModules {
		if !traceSortedContains(census.Modules, modulePath) {
			t.Errorf("mapped legacy module no longer exists: %s", modulePath)
		}
	}
	if len(unmapped) != 0 {
		t.Fatalf("unmapped legacy modules=%d: %v", len(unmapped), unmapped)
	}
	if !reflect.DeepEqual(census.Baseline, ledger.Baseline) {
		t.Fatalf("legacy Go census drift; review new/changed elements and update disposition baseline\n got: %#v\nwant: %#v", census.Baseline, ledger.Baseline)
	}
	if census.Baseline.ModuleCount != 106 || len(mappedModules) != 106 {
		t.Fatalf("module census/mapping = %d/%d, want 106/106", census.Baseline.ModuleCount, len(mappedModules))
	}
	t.Logf("legacy_go: modules=%d packages=%d files=%d symbols=%d unmapped=0 digest=%s",
		census.Baseline.ModuleCount,
		census.Baseline.PackageCount,
		census.Baseline.ProductionFileCount,
		census.Baseline.SymbolCount,
		census.Baseline.CensusSHA256,
	)
}

func TestTraceabilityRebuildPendingSourceInventory(t *testing.T) {
	var ledger tracePendingSourceLedger
	traceDecodeStrict(t, "product/traceability/pending_sources.json", &ledger)
	if ledger.DocumentKind != "pending_source_inventory" || ledger.SchemaVersion != 1 || len(ledger.Sources) != 4 {
		t.Fatalf("invalid pending source ledger header: %#v", ledger)
	}
	wantKinds := map[string]string{
		"historical_task_sources":     "task",
		"historical_bug_sources":      "bug",
		"historical_skill_sources":    "skill",
		"historical_rulepack_sources": "rulepack",
	}
	seen := make(map[string]struct{})
	for _, source := range ledger.Sources {
		wantKind, ok := wantKinds[source.ID]
		if !ok || source.Kind != wantKind || strings.TrimSpace(source.NextGate) == "" {
			t.Fatalf("invalid pending source entry: %#v", source)
		}
		if _, duplicate := seen[source.ID]; duplicate {
			t.Fatalf("duplicate pending source id %q", source.ID)
		}
		seen[source.ID] = struct{}{}
		paths := traceExpandSourcePaths(t, source)
		gotDigest := traceFileManifestDigest(t, paths)
		if len(paths) != source.ExpectedFileCount || gotDigest != source.ManifestSHA256 {
			t.Errorf("pending source %q drift: files=%d digest=%s, want files=%d digest=%s",
				source.ID, len(paths), gotDigest, source.ExpectedFileCount, source.ManifestSHA256)
		}
		if source.Kind == "rulepack" {
			if source.Status != "absent_pending_contract" || len(paths) != 0 {
				t.Errorf("rulepack source must remain explicitly absent until TLS-13 contract exists")
			}
		} else if source.Status != "inventory_only_pending_disposition" || len(paths) == 0 {
			t.Errorf("source %q must be non-empty inventory-only pending disposition", source.ID)
		}
	}
	t.Log("pending source ranges frozen; entries remain intentionally undisposed")
}

func traceComputeLegacyGoCensus(t *testing.T, ledger traceLegacyGoLedger) traceComputedCensus {
	t.Helper()
	entries, err := os.ReadDir(ledger.SourceRoot)
	if err != nil {
		t.Fatal(err)
	}
	modules := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			modules = append(modules, filepath.ToSlash(filepath.Join(ledger.SourceRoot, entry.Name())))
		}
	}
	sort.Strings(modules)

	packages := make(map[string]struct{})
	lines := make([]string, 0, 20000)
	for _, modulePath := range modules {
		lines = append(lines, "module|"+modulePath)
	}
	fileCount := 0
	symbolCount := 0
	fset := token.NewFileSet()
	for _, modulePath := range modules {
		err := filepath.WalkDir(modulePath, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			slashPath := filepath.ToSlash(path)
			if entry.IsDir() {
				if slashPath != modulePath && traceExcluded(slashPath, true, ledger.SourcePolicy.ExactExclusions) {
					return filepath.SkipDir
				}
				return nil
			}
			if filepath.Ext(path) != ".go" || strings.HasSuffix(path, ledger.SourcePolicy.ExcludeTestSuffix) ||
				traceExcluded(slashPath, false, ledger.SourcePolicy.ExactExclusions) {
				return nil
			}
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			parsed, err := parser.ParseFile(fset, path, content, parser.SkipObjectResolution)
			if err != nil {
				return fmt.Errorf("parse %s: %w", slashPath, err)
			}
			fileCount++
			fileSum := sha256.Sum256(content)
			lines = append(lines, "file|"+slashPath+"|"+hex.EncodeToString(fileSum[:]))
			packageKey := filepath.ToSlash(filepath.Dir(path)) + "|" + parsed.Name.Name
			packages[packageKey] = struct{}{}
			for _, declaration := range parsed.Decls {
				symbolLines, err := traceDeclarationSymbols(fset, slashPath, declaration)
				if err != nil {
					return err
				}
				symbolCount += len(symbolLines)
				lines = append(lines, symbolLines...)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	for packageKey := range packages {
		lines = append(lines, "package|"+packageKey)
	}
	sort.Strings(lines)
	return traceComputedCensus{
		Modules: modules,
		Baseline: traceGoBaseline{
			ModuleCount:         len(modules),
			PackageCount:        len(packages),
			ProductionFileCount: fileCount,
			SymbolCount:         symbolCount,
			ModuleSetSHA256:     traceStringsDigest(modules),
			CensusSHA256:        traceStringsDigest(lines),
		},
	}
}

func traceDeclarationSymbols(fset *token.FileSet, filePath string, declaration ast.Decl) ([]string, error) {
	var lines []string
	switch typed := declaration.(type) {
	case *ast.GenDecl:
		kind := strings.ToLower(typed.Tok.String())
		for _, spec := range typed.Specs {
			switch named := spec.(type) {
			case *ast.TypeSpec:
				lines = append(lines, "symbol|"+filePath+"|type||"+named.Name.Name)
			case *ast.ValueSpec:
				for _, name := range named.Names {
					if name.Name != "_" {
						lines = append(lines, "symbol|"+filePath+"|"+kind+"||"+name.Name)
					}
				}
			}
		}
	case *ast.FuncDecl:
		kind := "func"
		receiver := ""
		if typed.Recv != nil && len(typed.Recv.List) > 0 {
			kind = "method"
			var rendered bytes.Buffer
			if err := format.Node(&rendered, fset, typed.Recv.List[0].Type); err != nil {
				return nil, fmt.Errorf("render receiver in %s: %w", filePath, err)
			}
			receiver = rendered.String()
		}
		lines = append(lines, "symbol|"+filePath+"|"+kind+"|"+receiver+"|"+typed.Name.Name)
	}
	return lines, nil
}

func traceExcluded(path string, directory bool, exclusions []traceExactExclusion) bool {
	for _, exclusion := range exclusions {
		switch exclusion.Kind {
		case "file":
			if !directory && path == exclusion.Path {
				return true
			}
		case "subtree":
			if path == exclusion.Path || strings.HasPrefix(path, exclusion.Path+"/") {
				return true
			}
		}
	}
	return false
}

func traceAcceptedCapabilities(t *testing.T) map[string]struct{} {
	t.Helper()
	content, err := os.ReadFile("product/roadmap.json")
	if err != nil {
		t.Fatal(err)
	}
	var roadmap traceRoadmap
	if err := json.Unmarshal(content, &roadmap); err != nil {
		t.Fatal(err)
	}
	accepted := make(map[string]struct{})
	for _, entry := range roadmap.CapabilityEntries {
		if entry.Decision == "accept" {
			accepted[entry.ID] = struct{}{}
		}
	}
	return accepted
}

func traceExpandSourcePaths(t *testing.T, source tracePendingSource) []string {
	t.Helper()
	unique := make(map[string]struct{})
	for _, exactPath := range source.ExactPaths {
		if _, err := os.Stat(exactPath); err != nil {
			t.Fatalf("pending source %q exact path %q: %v", source.ID, exactPath, err)
		}
		unique[filepath.ToSlash(exactPath)] = struct{}{}
	}
	for _, pattern := range source.PathPatterns {
		matches, err := filepath.Glob(filepath.FromSlash(pattern))
		if err != nil {
			t.Fatalf("pending source %q invalid pattern %q: %v", source.ID, pattern, err)
		}
		if len(matches) == 0 {
			t.Fatalf("pending source %q pattern %q matches no files", source.ID, pattern)
		}
		for _, match := range matches {
			info, err := os.Stat(match)
			if err != nil || !info.Mode().IsRegular() {
				t.Fatalf("pending source %q path %q is not a regular file", source.ID, match)
			}
			unique[filepath.ToSlash(match)] = struct{}{}
		}
	}
	paths := make([]string, 0, len(unique))
	for path := range unique {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths
}

func traceFileManifestDigest(t *testing.T, paths []string) string {
	t.Helper()
	lines := make([]string, 0, len(paths))
	for _, path := range paths {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(content)
		lines = append(lines, path+"|sha256:"+hex.EncodeToString(sum[:]))
	}
	return traceStringsDigest(lines)
}

func traceStringsDigest(lines []string) string {
	hash := sha256.New()
	for _, line := range lines {
		_, _ = io.WriteString(hash, line)
		_, _ = io.WriteString(hash, "\n")
	}
	return "sha256:" + hex.EncodeToString(hash.Sum(nil))
}

func traceSortedContains(values []string, target string) bool {
	index := sort.SearchStrings(values, target)
	return index < len(values) && values[index] == target
}

func traceDecodeStrict(t *testing.T, path string, target any) {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		t.Fatalf("decode %s trailing content: %v", path, err)
	}
}
