package orquesta_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
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

func traceComputeLegacyGoCensus(t *testing.T, ledger traceLegacyGoLedger) traceComputedCensus {
	t.Helper()
	snapshot := traceLoadGitIndexSnapshot(t, ".", ledger.SourceRoot, func(path string) bool {
		return filepath.Ext(path) == ".go" &&
			!strings.HasSuffix(path, ledger.SourcePolicy.ExcludeTestSuffix) &&
			!traceExcluded(path, false, ledger.SourcePolicy.ExactExclusions)
	})
	moduleSet := make(map[string]struct{})
	for _, entry := range snapshot.Entries {
		relative := strings.TrimPrefix(entry.Path, ledger.SourceRoot+"/")
		moduleName, _, found := strings.Cut(relative, "/")
		if found && strings.HasPrefix(moduleName, "orquesta-") {
			moduleSet[ledger.SourceRoot+"/"+moduleName] = struct{}{}
		}
	}
	modules := make([]string, 0, len(moduleSet))
	for modulePath := range moduleSet {
		modules = append(modules, modulePath)
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
	contentPaths := make([]string, 0, len(snapshot.Contents))
	for path := range snapshot.Contents {
		contentPaths = append(contentPaths, path)
	}
	sort.Strings(contentPaths)
	for _, path := range contentPaths {
		content := snapshot.Contents[path]
		parsed, err := parser.ParseFile(fset, path, content, parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		fileCount++
		fileSum := sha256.Sum256(content)
		lines = append(lines, "file|"+path+"|"+hex.EncodeToString(fileSum[:]))
		packageKey := filepath.ToSlash(filepath.Dir(path)) + "|" + parsed.Name.Name
		packages[packageKey] = struct{}{}
		for _, declaration := range parsed.Decls {
			symbolLines, err := traceDeclarationSymbols(fset, path, declaration)
			if err != nil {
				t.Fatal(err)
			}
			symbolCount += len(symbolLines)
			lines = append(lines, symbolLines...)
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
