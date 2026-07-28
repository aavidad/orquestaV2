package acceptance_test

import (
	"bytes"
	"context"
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

const appAnalysisFoundationFixturePath = "acceptance/fixtures/app_analysis_foundation.json"

type appAnalysisFoundationFixture struct {
	SchemaVersion              int                            `json:"schema_version"`
	ContractID                 string                         `json:"contract_id"`
	Status                     string                         `json:"status"`
	CapabilitiesAccredited     []string                       `json:"capabilities_accredited"`
	RelatedRoadmapCapabilities []appAnalysisRoadmapCapability `json:"related_roadmap_capabilities"`
	RelatedAcceptanceContract  appAnalysisAcceptanceContract  `json:"related_acceptance_contract"`
	FoundationFiles            []string                       `json:"foundation_files"`
	Ports                      []string                       `json:"ports"`
	CausalSources              []string                       `json:"causal_sources"`
	ForbiddenImportBoundaries  []string                       `json:"forbidden_import_boundaries"`
	Deferred                   []string                       `json:"deferred"`
	RelevantTestFiles          []string                       `json:"relevant_test_files"`
	RequiredTest               appAnalysisRequiredTest        `json:"required_test"`
}

type appAnalysisRoadmapCapability struct {
	ID           string   `json:"id"`
	Status       string   `json:"status"`
	EvidenceRefs []string `json:"evidence_refs"`
}

type appAnalysisAcceptanceContract struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

type appAnalysisRequiredTest struct {
	Command            string `json:"command"`
	NamePrefix         string `json:"name_prefix"`
	RejectNoTestsToRun bool   `json:"reject_no_tests_to_run"`
}

func TestAcceptanceAppAnalysisFoundation(t *testing.T) {
	root := evidenceRepositoryRoot(t)
	fixture := loadAppAnalysisFoundationFixture(t, root)
	assertAppAnalysisFoundationEnvelope(t, root, fixture)
	assertAppAnalysisFoundationImports(t, root, fixture)
	assertAppAnalysisFoundationIsUnwired(t, root)
	assertAppAnalysisRelevantTests(t, root, fixture)
}

func loadAppAnalysisFoundationFixture(t *testing.T, root string) appAnalysisFoundationFixture {
	t.Helper()
	file, err := os.Open(filepath.Join(root, appAnalysisFoundationFixturePath))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	var fixture appAnalysisFoundationFixture
	if err := decoder.Decode(&fixture); err != nil {
		t.Fatal(err)
	}
	if err := requireAppAnalysisFixtureEOF(decoder); err != nil {
		t.Fatal(err)
	}
	return fixture
}

func requireAppAnalysisFixtureEOF(decoder *json.Decoder) error {
	var extra any
	err := decoder.Decode(&extra)
	if err == io.EOF {
		return nil
	}
	if err == nil {
		return errAppAnalysisFixtureTrailingJSON{}
	}
	return err
}

type errAppAnalysisFixtureTrailingJSON struct{}

func (errAppAnalysisFixtureTrailingJSON) Error() string { return "unexpected trailing JSON" }

func assertAppAnalysisFoundationEnvelope(t *testing.T, root string, fixture appAnalysisFoundationFixture) {
	t.Helper()
	if fixture.SchemaVersion != 1 || fixture.ContractID != "FOUNDATION-APP-ANALYSIS" || fixture.Status != "foundation_only_unwired" {
		t.Fatalf("invalid foundation envelope: %+v", fixture)
	}
	if len(fixture.CapabilitiesAccredited) != 0 {
		t.Fatalf("foundation must not accredit roadmap capabilities: %v", fixture.CapabilitiesAccredited)
	}
	assertAppAnalysisStrings(t, fixture.FoundationFiles, []string{
		"internal/application/app_analysis.go", "internal/application/app_analysis_service.go",
		"internal/application/app_analysis_contract_test.go", "internal/application/app_analysis_service_test.go",
	})
	assertAppAnalysisStrings(t, fixture.Ports, []string{
		"AppAnalysisIntakeSnapshotVerifier", "AppAnalysisArtifactResolver",
		"AppAnalysisReviewVerifier", "AppAnalysisAttachmentStore",
	})
	assertAppAnalysisStrings(t, fixture.CausalSources, []string{
		"authenticated authorization receipt", "project-scoped intake snapshot",
		"review receipt bound to canonical manifest", "artifact provenance receipt",
	})
	assertAppAnalysisStrings(t, fixture.Deferred, []string{
		"bootstrap wiring", "concrete adapters", "public canonical commands",
		"durable persistence integration", "end-to-end acceptance evidence",
	})
	if fixture.RelatedAcceptanceContract != (appAnalysisAcceptanceContract{ID: "AC-V28-DOMAIN-PLUGINS", Status: "planned"}) {
		t.Fatalf("V28 must remain planned: %+v", fixture.RelatedAcceptanceContract)
	}
	if fixture.RequiredTest.Command != "go test -mod=vendor -count=1 -v ./internal/application -run '^TestAppAnalysis'" ||
		fixture.RequiredTest.NamePrefix != "TestAppAnalysis" || !fixture.RequiredTest.RejectNoTestsToRun {
		t.Fatalf("invalid relevant-test inventory: %+v", fixture.RequiredTest)
	}
	for _, file := range fixture.FoundationFiles {
		if _, err := os.Stat(filepath.Join(root, file)); err != nil {
			t.Fatalf("foundation file %q: %v", file, err)
		}
	}
	actualFiles, err := filepath.Glob(filepath.Join(root, "internal/application/app_analysis*.go"))
	if err != nil {
		t.Fatal(err)
	}
	for index := range actualFiles {
		actualFiles[index], err = filepath.Rel(root, actualFiles[index])
		if err != nil {
			t.Fatal(err)
		}
	}
	sort.Strings(actualFiles)
	wantFiles := append([]string(nil), fixture.FoundationFiles...)
	sort.Strings(wantFiles)
	if strings.Join(actualFiles, "\n") != strings.Join(wantFiles, "\n") {
		t.Fatalf("app analysis foundation files=%v want exactly=%v", actualFiles, wantFiles)
	}
	assertAppAnalysisRoadmapState(t, root, fixture)
}

func assertAppAnalysisRoadmapState(t *testing.T, root string, fixture appAnalysisFoundationFixture) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(root, "product/roadmap.json"))
	if err != nil {
		t.Fatal(err)
	}
	var roadmap map[string]any
	if err := json.Unmarshal(raw, &roadmap); err != nil {
		t.Fatal(err)
	}
	entries, ok := roadmap["capability_entries"].([]any)
	if !ok {
		t.Fatal("roadmap capability_entries missing")
	}
	actual := map[string]appAnalysisRoadmapCapability{}
	for _, entry := range entries {
		object, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		id, _ := object["id"].(string)
		if id != "WIZ-10" && id != "EXT-00" {
			continue
		}
		status, _ := object["status"].(string)
		evidence, _ := object["evidence_refs"].([]any)
		actual[id] = appAnalysisRoadmapCapability{ID: id, Status: status, EvidenceRefs: appAnalysisStringValues(evidence)}
	}
	for _, want := range fixture.RelatedRoadmapCapabilities {
		got, ok := actual[want.ID]
		if !ok || got.ID != want.ID || got.Status != "declared" ||
			got.Status != want.Status || len(got.EvidenceRefs) != 0 ||
			len(want.EvidenceRefs) != 0 {
			t.Fatalf("roadmap capability %q must stay declared without evidence: got=%+v want=%+v", want.ID, got, want)
		}
	}
	contracts, ok := roadmap["acceptance_contracts"].([]any)
	if !ok {
		t.Fatal("roadmap acceptance_contracts missing")
	}
	for _, contract := range contracts {
		object, ok := contract.(map[string]any)
		if !ok || object["id"] != fixture.RelatedAcceptanceContract.ID {
			continue
		}
		if object["status"] != fixture.RelatedAcceptanceContract.Status {
			t.Fatalf("AC-V28 status=%v", object["status"])
		}
		return
	}
	t.Fatalf("roadmap contract %q missing", fixture.RelatedAcceptanceContract.ID)
}

func appAnalysisStringValues(values []any) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if text, ok := value.(string); ok {
			result = append(result, text)
		}
	}
	return result
}

func assertAppAnalysisFoundationImports(t *testing.T, root string, fixture appAnalysisFoundationFixture) {
	t.Helper()
	for _, source := range fixture.FoundationFiles[:2] {
		parsed, err := parser.ParseFile(token.NewFileSet(), filepath.Join(root, source), nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s: %v", source, err)
		}
		for _, imported := range parsed.Imports {
			path := strings.Trim(imported.Path.Value, "\"")
			for _, forbidden := range fixture.ForbiddenImportBoundaries {
				if strings.HasPrefix(path, forbidden) {
					t.Fatalf("%s imports forbidden boundary %q", source, path)
				}
			}
		}
	}
}

func assertAppAnalysisFoundationIsUnwired(t *testing.T, root string) {
	t.Helper()
	for _, directory := range []string{"cmd/orquesta", "internal/commands", "internal/bootstrap", "product/evidence"} {
		var matches []string
		err := filepath.WalkDir(filepath.Join(root, directory), func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				return nil
			}
			content, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			if bytes.Contains(content, []byte("app-analysis")) || bytes.Contains(content, []byte("app_analysis")) {
				matches = append(matches, strings.TrimPrefix(path, root+string(filepath.Separator)))
			}
			return nil
		})
		if err != nil && !os.IsNotExist(err) {
			t.Fatal(err)
		}
		if len(matches) != 0 {
			t.Fatalf("foundation unexpectedly wired in %s: %v", directory, matches)
		}
	}
}

func assertAppAnalysisRelevantTests(t *testing.T, root string, fixture appAnalysisFoundationFixture) {
	t.Helper()
	var names []string
	for _, source := range fixture.RelevantTestFiles {
		parsed, err := parser.ParseFile(token.NewFileSet(), filepath.Join(root, source), nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", source, err)
		}
		for _, declaration := range parsed.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if ok && strings.HasPrefix(function.Name.Name, fixture.RequiredTest.NamePrefix) {
				names = append(names, function.Name.Name)
			}
		}
	}
	sort.Strings(names)
	if len(names) < 10 {
		t.Fatalf("relevant app analysis test inventory is too small: %v", names)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	command := exec.CommandContext(
		ctx,
		"go", "test", "-mod=vendor", "-count=1", "-v",
		"./internal/application", "-run", "^TestAppAnalysis",
	)
	command.Dir = root
	output, err := command.CombinedOutput()
	if ctx.Err() != nil {
		t.Fatalf("%s timed out: %v\n%s", fixture.RequiredTest.Command, ctx.Err(), output)
	}
	if err != nil {
		t.Fatalf("%s failed: %v\n%s", fixture.RequiredTest.Command, err, output)
	}
	if fixture.RequiredTest.RejectNoTestsToRun && (bytes.Contains(output, []byte("no tests to run")) || !bytes.Contains(output, []byte("PASS"))) {
		t.Fatalf("relevant test command produced no valid pass:\n%s", output)
	}
	for _, name := range names {
		if !bytes.Contains(output, []byte("--- PASS: "+name)) {
			t.Fatalf("inventoried test %q did not run:\n%s", name, output)
		}
	}
}

func assertAppAnalysisStrings(t *testing.T, got, want []string) {
	t.Helper()
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("got=%v want=%v", got, want)
	}
}
