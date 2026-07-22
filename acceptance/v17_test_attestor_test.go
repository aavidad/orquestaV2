package acceptance_test

import (
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

const v17FixturePath = "acceptance/fixtures/v17_test_attestor.json"
const v17ContractBaseGitCommitOID = "eb272b6645928d800619709c9afd272440b0dabf"

const (
	v17RealE2EPackage = "./internal/bootstrap"
	v17RealE2EImport  = "orquesta/internal/bootstrap"
	v17RealE2ETest    = "TestRealGitSQLiteCASBubblewrapAttestationEndToEnd"
	v17RealE2ETag     = "v17_real_e2e"
)

type v17Fixture struct {
	SchemaVersion                  int               `json:"schema_version"`
	ReceiptSchemaVersion           int               `json:"receipt_schema_version"`
	ContractID                     string            `json:"contract_id"`
	TrustedBaseGitCommitOID        string            `json:"trusted_base_git_commit_oid"`
	ProductDeltaBaseGitCommitOID   string            `json:"product_delta_base_git_commit_oid"`
	ProductDeltaSealedGitCommitOID string            `json:"product_delta_sealed_git_commit_oid"`
	SealStatus                     string            `json:"seal_status"`
	SealNote                       string            `json:"seal_note"`
	Command                        string            `json:"command"`
	ExecutionArgv                  []string          `json:"execution_argv"`
	OutputPath                     string            `json:"output_path"`
	ReceiptPath                    string            `json:"receipt_path"`
	CandidateSubjects              []string          `json:"candidate_subjects"`
	OwnedCapabilityIDs             []string          `json:"owned_capability_ids"`
	DependencyVerticals            []string          `json:"dependency_verticals"`
	SimplicityBudget               v17Budget         `json:"simplicity_budget"`
	RequiredTestSpecContract       v17ValueObject    `json:"required_test_spec_contract"`
	RequiredTypes                  []v17RequiredType `json:"required_types"`
	RequiredOutboundPorts          []v16RequiredPort `json:"required_outbound_ports"`
	RequiredObjectStreamPort       v16RequiredPort   `json:"required_object_stream_port"`
	RequiredApplicationMarkers     []string          `json:"required_application_markers"`
	ForbiddenPrivateAuthorities    []string          `json:"forbidden_private_authorities"`
	BubblewrapAdapterDirectory     string            `json:"bubblewrap_adapter_directory"`
	BubblewrapRequiredMarkers      []string          `json:"bubblewrap_required_markers"`
	BubblewrapForbiddenMarkers     []string          `json:"bubblewrap_forbidden_markers"`
	CASAdapterDirectory            string            `json:"cas_adapter_directory"`
	CASRequiredMarkers             []string          `json:"cas_required_markers"`
	RequiredBehaviorTests          []v17BehaviorTest `json:"required_behavior_tests"`
	FocalSuite                     v17Suite          `json:"focal_suite"`
	RaceSuite                      v17Suite          `json:"race_suite"`
	E2ESuite                       v17Suite          `json:"e2e_suite"`
	VetPackages                    []string          `json:"vet_packages"`
	DeferredSurfaces               []string          `json:"deferred_surfaces"`
	RoadmapAssertions              []string          `json:"roadmap_assertions"`
}

type v17Budget struct {
	ProductNetLines       int `json:"product_net_lines"`
	CoreNetLines          int `json:"core_net_lines"`
	AdapterNetLines       int `json:"adapter_net_lines"`
	MigrationNetLines     int `json:"migration_net_lines"`
	TestNetLines          int `json:"test_net_lines"`
	MaxFunctionLines      int `json:"max_function_lines"`
	MaxAddedGoFileLines   int `json:"max_added_go_file_lines"`
	MaxNewProductPackages int `json:"max_new_product_packages"`
}

type v17RequiredType struct {
	Directory string   `json:"directory"`
	Name      string   `json:"name"`
	Fields    []string `json:"fields"`
	Exact     bool     `json:"exact"`
}

type v17ValueObject struct {
	Type        string   `json:"type"`
	InputType   string   `json:"input_type"`
	Constructor string   `json:"constructor"`
	Getters     []string `json:"getters"`
}

type v17BehaviorTest struct {
	Directory string `json:"directory"`
	Name      string `json:"name"`
	Gate      string `json:"gate"`
	GOOS      string `json:"goos,omitempty"`
}

type v17Suite struct {
	Packages []string `json:"packages"`
	Tests    []string `json:"tests"`
}

func TestV17CandidateSubjectsCoverCommittedDelta(t *testing.T) {
	repositoryRoot := evidenceRepositoryRoot(t)
	fixture := evidenceDecodeStrictJSON[v17Fixture](t, filepath.Join(repositoryRoot, v17FixturePath))
	numstat := evidenceAssertCandidateDelta(t, "V17", repositoryRoot, fixture.ProductDeltaBaseGitCommitOID,
		fixture.ProductDeltaSealedGitCommitOID, fixture.CandidateSubjects)
	v17AssertSimplicityBudget(t, repositoryRoot, fixture, numstat)
}

func TestAcceptanceV17TestAttestor(t *testing.T) {
	repositoryRoot := evidenceRepositoryRoot(t)
	fixture := evidenceDecodeStrictJSON[v17Fixture](t, filepath.Join(repositoryRoot, v17FixturePath))
	v17AssertFixture(t, repositoryRoot, fixture)

	t.Run("canonical_subject_types_and_neutral_ports", func(t *testing.T) {
		v17AssertModelAndPorts(t, repositoryRoot, fixture)
	})
	t.Run("one_writer_PASS_does_not_integrate_FAIL_is_durable_and_ACK_is_not_evidence", func(t *testing.T) {
		v17AssertFlowAuthority(t, repositoryRoot, fixture)
	})
	t.Run("filesystem_CAS_Git_snapshot_and_bubblewrap_fail_closed", func(t *testing.T) {
		v17AssertAdapters(t, repositoryRoot, fixture)
	})
	t.Run("focal_race_E2E_and_vet_gates_are_concrete", func(t *testing.T) {
		v17AssertBehaviorTests(t, repositoryRoot, fixture)
	})
}

func TestAcceptanceV17TestAttestorReceipt(t *testing.T) {
	evidenceAssertReceiptV3(t, evidenceRepositoryRoot(t), evidenceReceiptV3Expectation{
		Contract: "AC-V17-TEST-ATTESTOR", FixturePath: v17FixturePath,
		ReceiptPath: "product/evidence/v17_test_attestor.json", ExecutedNotBefore: "2026-07-22T00:00:00Z",
		TrustedBaseGitCommitOID: v17ContractBaseGitCommitOID,
	})
}

func v17AssertFixture(t *testing.T, repositoryRoot string, fixture v17Fixture) {
	t.Helper()
	if fixture.SchemaVersion != 1 || fixture.ReceiptSchemaVersion != 3 ||
		fixture.ContractID != "AC-V17-TEST-ATTESTOR" ||
		fixture.TrustedBaseGitCommitOID != v17ContractBaseGitCommitOID ||
		fixture.ProductDeltaBaseGitCommitOID != v17ContractBaseGitCommitOID ||
		fixture.OutputPath != "product/evidence/v17_test_attestor.output.txt" ||
		fixture.ReceiptPath != "product/evidence/v17_test_attestor.json" {
		t.Fatalf("invalid V17 contract identity: %+v", fixture)
	}
	if fixture.ProductDeltaSealedGitCommitOID == "" {
		if fixture.SealStatus != "pre_p_unsealed_no_evidence" ||
			fixture.SealNote != "Pre-P contract only: empty sealed OID is mandatory; this fixture and any ACK are not product evidence." {
			t.Fatalf("invalid V17 pre-P seal declaration: %+v", fixture)
		}
	} else {
		if fixture.SealStatus != "p_product_delta_sealed_pending_evidence" ||
			fixture.SealNote != "P product commit sealed; S and E remain pending and no PASS is implied." {
			t.Fatalf("invalid V17 post-P pre-E seal declaration: %+v", fixture)
		}
		if err := evidenceValidateSealedCommit(
			repositoryRoot, fixture.ProductDeltaBaseGitCommitOID, fixture.ProductDeltaSealedGitCommitOID,
		); err != nil {
			t.Fatalf("invalid V17 P subject: %v", err)
		}
	}
	if fixture.Command != "sh -c '"+v17ValidationShellBody()+"'" ||
		!reflect.DeepEqual(fixture.ExecutionArgv, []string{"sh", "-c", v17ValidationShellBody()}) {
		t.Fatalf("invalid V17 command/argv: %q %#v", fixture.Command, fixture.ExecutionArgv)
	}
	for _, marker := range []string{
		"CGO_ENABLED=0 go test -mod=vendor -tags=" + v17RealE2ETag + " -json",
		v17RealE2EPackage, "^" + v17RealE2ETest + "$",
		"\\\"Action\\\":\\\"run\\\"", "\\\"Action\\\":\\\"pass\\\"", v17RealE2EImport,
	} {
		if !strings.Contains(v17ValidationShellBody(), marker) {
			t.Fatalf("V17_GATE_REAL_E2E_COMMAND: command lacks %q", marker)
		}
	}
	if err := evidenceValidateCandidateSubjects(fixture.CandidateSubjects, fixture.ReceiptPath, fixture.OutputPath); err != nil {
		t.Fatal(err)
	}
	for _, relative := range fixture.CandidateSubjects {
		if _, err := os.Stat(filepath.Join(repositoryRoot, filepath.FromSlash(relative))); err != nil {
			t.Fatalf("candidate subject %q is not readable before P: %v", relative, err)
		}
	}
	if fixture.ProductDeltaSealedGitCommitOID == "" {
		v17AssertPrePDirtySubjects(t, repositoryRoot, fixture)
	}
	if !reflect.DeepEqual(fixture.OwnedCapabilityIDs, []string{"EVD-01", "EVD-04", "EVD-05", "EVD-13"}) ||
		!reflect.DeepEqual(fixture.DependencyVerticals, []string{
			"atomic_state_outbox", "recovery_backup", "budgets_effects", "workspace_git",
		}) || fixture.SimplicityBudget != (v17Budget{
		ProductNetLines: 6800, CoreNetLines: 1900, AdapterNetLines: 4500, MigrationNetLines: 500,
		TestNetLines: 7500, MaxFunctionLines: 80, MaxAddedGoFileLines: 350, MaxNewProductPackages: 2,
	}) {
		t.Fatalf("invalid V17 ownership, dependencies or simplicity budget: %+v", fixture)
	}
	if len(fixture.RequiredTypes) != 10 || len(fixture.RequiredOutboundPorts) != 1 ||
		len(fixture.RequiredBehaviorTests) < 30 || len(fixture.ForbiddenPrivateAuthorities) != 9 ||
		len(fixture.RoadmapAssertions) != 12 {
		t.Fatalf("invalid V17 coverage counts: %+v", fixture)
	}
	if fixture.RequiredTestSpecContract.Type != "RequiredTestSpec" ||
		fixture.RequiredTestSpecContract.InputType != "RequiredTestSpecInput" ||
		fixture.RequiredTestSpecContract.Constructor != "NewRequiredTestSpec" ||
		!reflect.DeepEqual(fixture.RequiredTestSpecContract.Getters, []string{
			"Ref", "ToolRef", "Arguments", "WorkingDirectory",
		}) {
		t.Fatalf("invalid V17 RequiredTestSpec contract: %+v", fixture.RequiredTestSpecContract)
	}
	if !reflect.DeepEqual(fixture.FocalSuite.Packages, v17FocalPackages()) || len(fixture.FocalSuite.Tests) != 0 ||
		!reflect.DeepEqual(fixture.RaceSuite.Packages, v17RacePackages()) ||
		!reflect.DeepEqual(fixture.RaceSuite.Tests, v17RaceTests()) ||
		!reflect.DeepEqual(fixture.E2ESuite.Packages, v17E2EPackages()) ||
		!reflect.DeepEqual(fixture.E2ESuite.Tests, v17E2ETests()) ||
		!reflect.DeepEqual(fixture.VetPackages, v17FocalPackages()) {
		t.Fatalf("invalid V17 focal/race/E2E/vet suite: %+v", fixture)
	}
	wantDeferred := []string{
		"author_reviewers_refinery_v18", "council_v19", "public_bindings_v20_and_i18n_v21",
		"codex_end_to_end_v22", "general_toolchain_registry_v26",
		"remote_s3_postgres_multihost_v28_v31", "sbom_signing_release_v34",
		"providers_forge_ui_microservices_legacy_compatibility",
	}
	if !reflect.DeepEqual(fixture.DeferredSurfaces, wantDeferred) {
		t.Fatalf("invalid V17 deferred boundary: %v", fixture.DeferredSurfaces)
	}
	if _, err := evidenceGitCanonicalCommit(repositoryRoot, fixture.TrustedBaseGitCommitOID); err != nil {
		t.Fatalf("invalid V17 contract base: %v", err)
	}
}

func v17AssertModelAndPorts(t *testing.T, repositoryRoot string, fixture v17Fixture) {
	t.Helper()
	goalDirectory := filepath.Join(repositoryRoot, "internal", "goal")
	for _, required := range fixture.RequiredTypes {
		v17RequireProductionType(t, filepath.Join(repositoryRoot, required.Directory), required)
	}
	value := fixture.RequiredTestSpecContract
	functions, methods := v17ProductionFunctions(t, goalDirectory, value.Type)
	if !functions[value.Constructor] {
		t.Errorf("V17_GATE_REQUIRED_TEST_SPEC: %s constructor missing", value.Constructor)
	}
	for _, getter := range value.Getters {
		if !methods[getter] {
			t.Errorf("V17_GATE_REQUIRED_TEST_SPEC: %s getter %s missing", value.Type, getter)
		}
	}
	applicationDirectory := filepath.Join(repositoryRoot, "internal", "application")
	for _, required := range fixture.RequiredOutboundPorts {
		v17RequireOutboundPort(t, applicationDirectory, required)
		if count := v16StructFieldTypeCount(t, applicationDirectory, "Dependencies", required.Name); count != 1 {
			t.Errorf("V17_GATE_PORT_WIRING: Dependencies fields of type %s=%d, want 1", required.Name, count)
		}
	}
	streamMethods, found := v16InterfaceMethods(
		t, filepath.Join(repositoryRoot, fixture.BubblewrapAdapterDirectory), fixture.RequiredObjectStreamPort.Name,
	)
	if !found {
		t.Errorf("V17_GATE_OBJECT_STREAM_PORT: adapter port %s missing", fixture.RequiredObjectStreamPort.Name)
	} else {
		for _, name := range fixture.RequiredObjectStreamPort.Methods {
			if _, ok := streamMethods[name]; !ok {
				t.Errorf("V17_GATE_OBJECT_STREAM_PORT: %s lacks %s", fixture.RequiredObjectStreamPort.Name, name)
			}
		}
		if signature := streamMethods["OpenSnapshotStream"]; !strings.Contains(signature, "context.Context") ||
			!strings.Contains(signature, "ports.SnapshotVerificationRequest") ||
			!strings.Contains(signature, "io.ReadCloser") || !strings.Contains(signature, "error") {
			t.Errorf("V17_GATE_OBJECT_STREAM_PORT: unsafe OpenSnapshotStream signature %s", signature)
		}
	}
	for _, typeName := range []string{value.Type, value.InputType} {
		fields, found := v13ProductionTypeFields(t, goalDirectory, typeName)
		if !found {
			continue
		}
		for field := range fields {
			switch strings.ToLower(field) {
			case "command", "shell", "script", "environment", "executablepath":
				t.Errorf("V17 %s leaks executable detail %s", typeName, field)
			}
		}
	}
}

func v17ProductionFunctions(t *testing.T, directory, receiverType string) (map[string]bool, map[string]bool) {
	t.Helper()
	functions, methods := map[string]bool{}, map[string]bool{}
	for _, file := range v16ProductionGoFiles(t, directory, false) {
		parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", file, err)
		}
		for _, declaration := range parsed.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok {
				continue
			}
			if function.Recv == nil {
				functions[function.Name.Name] = true
				continue
			}
			typeName := function.Recv.List[0].Type
			if pointer, ok := typeName.(*ast.StarExpr); ok {
				typeName = pointer.X
			}
			if identifier, ok := typeName.(*ast.Ident); ok && identifier.Name == receiverType {
				methods[function.Name.Name] = true
			}
		}
	}
	return functions, methods
}

func v17RequireProductionType(t *testing.T, directory string, required v17RequiredType) {
	t.Helper()
	fields, found := v13ProductionTypeFields(t, directory, required.Name)
	if !found {
		t.Errorf("V17_GATE_SUBJECT_MODEL: production type %s missing", required.Name)
		return
	}
	for _, field := range required.Fields {
		if !fields[field] {
			t.Errorf("V17_GATE_SUBJECT_MODEL: %s lacks %s", required.Name, field)
		}
	}
	if required.Exact && len(fields) != len(required.Fields) {
		t.Errorf("V17_GATE_SUBJECT_MODEL: %s fields=%v, want exact %v", required.Name, fields, required.Fields)
	}
}

func v17RequireOutboundPort(t *testing.T, directory string, required v16RequiredPort) {
	t.Helper()
	methods, found := v16InterfaceMethods(t, directory, required.Name)
	if !found {
		t.Errorf("V17_GATE_PORT_WIRING: application outbound port %s missing", required.Name)
		return
	}
	for _, name := range required.Methods {
		signature, ok := methods[name]
		if !ok {
			t.Errorf("V17_GATE_PORT_WIRING: %s lacks %s", required.Name, name)
			continue
		}
		if !strings.Contains(signature, "context.Context") || strings.Count(signature, "ports.") < 2 ||
			!strings.Contains(signature, "error") {
			t.Errorf("V17_GATE_PORT_WIRING: %s.%s must use context and neutral ports request/result: %s", required.Name, name, signature)
		}
	}
}

func v17AssertFlowAuthority(t *testing.T, repositoryRoot string, fixture v17Fixture) {
	t.Helper()
	applicationSource := v16ReadProductionGo(t, filepath.Join(repositoryRoot, "internal", "application"))
	for _, marker := range fixture.RequiredApplicationMarkers {
		if !strings.Contains(applicationSource, marker) {
			t.Errorf("V17_GATE_APPLICATION_FLOW: application flow lacks %q", marker)
		}
	}
	authoritySource := applicationSource + v16ReadProductionGo(t, filepath.Join(repositoryRoot, "internal", "ports"))
	for _, forbidden := range fixture.ForbiddenPrivateAuthorities {
		if strings.Contains(authoritySource, "type "+forbidden+" ") {
			t.Errorf("V17 duplicate lifecycle/store/queue/scheduler %s is forbidden", forbidden)
		}
	}
}

func v17AssertAdapters(t *testing.T, repositoryRoot string, fixture v17Fixture) {
	t.Helper()
	bubblewrap := v16ReadProductionGoOptional(t, filepath.Join(repositoryRoot, fixture.BubblewrapAdapterDirectory))
	if bubblewrap == "" {
		t.Error("V17_GATE_BUBBLEWRAP: TestAttestor adapter missing")
	}
	for _, marker := range fixture.BubblewrapRequiredMarkers {
		if !strings.Contains(bubblewrap, marker) {
			t.Errorf("V17_GATE_BUBBLEWRAP: adapter lacks %q", marker)
		}
	}
	for _, forbidden := range fixture.BubblewrapForbiddenMarkers {
		if strings.Contains(bubblewrap, forbidden) {
			t.Errorf("V17 bubblewrap adapter contains forbidden ambient mechanism %q", forbidden)
		}
	}
	cas := v16ReadProductionGoOptional(t, filepath.Join(repositoryRoot, fixture.CASAdapterDirectory))
	for _, marker := range fixture.CASRequiredMarkers {
		if !strings.Contains(cas, marker) {
			t.Errorf("V17_GATE_CAS: filesystem CAS hardening lacks %q", marker)
		}
	}
}

func v17AssertBehaviorTests(t *testing.T, repositoryRoot string, fixture v17Fixture) {
	t.Helper()
	v17AssertRealE2ESelection(t, repositoryRoot)
	v17AssertRaceSelection(t, repositoryRoot, fixture)
	selected := make(map[string][]string)
	for _, required := range fixture.RequiredBehaviorTests {
		if required.Directory == "" || required.Name == "" || required.Gate == "" {
			t.Fatalf("incomplete V17 behavior gate: %+v", required)
		}
		goos := required.GOOS
		if goos == "" {
			goos = "linux"
		}
		tags := []string(nil)
		key := required.Directory + ":" + goos
		if required.Name == v17RealE2ETest {
			tags, key = []string{v17RealE2ETag}, key+":"+v17RealE2ETag
		}
		if _, found := selected[key]; !found {
			var err error
			selected[key], err = v17SelectedTestNames(filepath.Join(repositoryRoot, required.Directory), goos, tags)
			if err != nil {
				t.Fatalf("%s: select behavior package: %v", required.Gate, err)
			}
		}
		if count := v17TestNameCount(selected[key], required.Name); count != 1 {
			t.Errorf("%s: executable behavior test %s/%s definitions=%d, want 1",
				required.Gate, required.Directory, required.Name, count)
		}
	}
}

func v17AssertRaceSelection(t *testing.T, repositoryRoot string, fixture v17Fixture) {
	t.Helper()
	selected := make(map[string]int)
	for _, packagePath := range fixture.RaceSuite.Packages {
		directory := filepath.Join(repositoryRoot, filepath.FromSlash(strings.TrimPrefix(packagePath, "./")))
		names, err := v17SelectedTestNames(directory, "linux", nil)
		if err != nil {
			t.Fatalf("V17_GATE_RACE_SELECTION: select %s: %v", packagePath, err)
		}
		for _, name := range names {
			selected[name]++
		}
	}
	for _, name := range fixture.RaceSuite.Tests {
		if selected[name] != 1 {
			t.Errorf("V17_GATE_RACE_SELECTION: Linux race test %s definitions=%d, want 1", name, selected[name])
		}
	}
}

func v17AssertPrePDirtySubjects(t *testing.T, repositoryRoot string, fixture v17Fixture) {
	t.Helper()
	tracked, err := evidenceGit(repositoryRoot, "diff", "--name-only", fixture.ProductDeltaBaseGitCommitOID, "--")
	if err != nil {
		t.Fatal(err)
	}
	untracked, err := evidenceGit(repositoryRoot, "ls-files", "--others", "--exclude-standard")
	if err != nil {
		t.Fatal(err)
	}
	unique := make(map[string]struct{})
	for _, output := range [][]byte{tracked, untracked} {
		for _, relative := range strings.Split(strings.TrimSpace(string(output)), "\n") {
			if relative != "" {
				unique[filepath.ToSlash(relative)] = struct{}{}
			}
		}
	}
	dirty := make([]string, 0, len(unique))
	for relative := range unique {
		dirty = append(dirty, relative)
	}
	sort.Strings(dirty)
	if !reflect.DeepEqual(dirty, fixture.CandidateSubjects) {
		t.Fatalf("V17_GATE_PRE_P_SUBJECTS: dirty delta differs from fixture:\ndirty=%v\nfixture=%v",
			dirty, fixture.CandidateSubjects)
	}
}

func v17AssertRealE2ESelection(t *testing.T, repositoryRoot string) {
	t.Helper()
	if err := v17RealE2ESelectionError(filepath.Join(repositoryRoot, "internal", "bootstrap")); err != nil {
		t.Fatal(err)
	}
}

func v17RealE2ESelectionError(directory string) error {
	tagged, err := v17SelectedTestNames(directory, "linux", []string{v17RealE2ETag})
	if err != nil {
		return fmt.Errorf("V17_GATE_REAL_E2E_BUILD: select tagged Linux tests: %w", err)
	}
	if count := v17TestNameCount(tagged, v17RealE2ETest); count != 1 {
		return fmt.Errorf("V17_GATE_REAL_E2E_BUILD: tagged Linux selected %s=%d, want 1; stale name/tag", v17RealE2ETest, count)
	}
	untagged, err := v17SelectedTestNames(directory, "linux", nil)
	if err != nil {
		return fmt.Errorf("V17_GATE_REAL_E2E_BUILD: select untagged Linux tests: %w", err)
	}
	if count := v17TestNameCount(untagged, v17RealE2ETest); count != 0 {
		return fmt.Errorf("V17_GATE_REAL_E2E_BUILD: %s is not gated exclusively by %s", v17RealE2ETest, v17RealE2ETag)
	}
	return nil
}

func v17SelectedTestNames(directory, goos string, tags []string) ([]string, error) {
	context := build.Default
	context.GOOS = goos
	context.CgoEnabled = false
	context.BuildTags = append([]string(nil), tags...)
	packageInfo, err := context.ImportDir(directory, 0)
	if err != nil {
		return nil, err
	}
	files := append(append([]string{}, packageInfo.TestGoFiles...), packageInfo.XTestGoFiles...)
	names := make([]string, 0, len(files))
	for _, name := range files {
		parsed, err := parser.ParseFile(token.NewFileSet(), filepath.Join(directory, name), nil, 0)
		if err != nil {
			return nil, err
		}
		for _, declaration := range parsed.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if ok && function.Recv == nil {
				names = append(names, function.Name.Name)
			}
		}
	}
	return names, nil
}

func v17TestNameCount(names []string, want string) int {
	count := 0
	for _, name := range names {
		if name == want {
			count++
		}
	}
	return count
}

func TestV17RealE2ESelectionGuard(t *testing.T) {
	directory := t.TempDir()
	write := func(name, source string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(directory, name), []byte(source), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("subject.go", "package bootstrap\n")
	write("real_linux_test.go", "//go:build linux && v17_real_e2e\n\npackage bootstrap\n\nfunc "+v17RealE2ETest+"() {}\n")
	if err := v17RealE2ESelectionError(directory); err != nil {
		t.Fatalf("tagged test selection: %v", err)
	}
	if err := os.WriteFile(filepath.Join(directory, "real_linux_test.go"), []byte("//go:build linux\n\npackage bootstrap\n\nfunc "+v17RealE2ETest+"() {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := v17RealE2ESelectionError(directory); err == nil {
		t.Fatal("untagged stale-tag selection passed")
	}
}

func TestV17AcceptanceCommandUsesNoDestructiveTemporaryCleanup(t *testing.T) {
	body := v17ValidationShellBody()
	for _, forbidden := range []string{"rm -rf", "trap \"rm"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("V17_GATE_EXECUTABLE_COMMAND: destructive cleanup %q remains", forbidden)
		}
	}
	if !strings.Contains(body, "e2e_events=$(CGO_ENABLED=0 go test") ||
		!strings.Contains(body, "e2e_status=$?") {
		t.Fatal("V17_GATE_EXECUTABLE_COMMAND: bounded in-memory E2E selection is missing")
	}
}

func v17ValidationShellBody() string {
	acceptance := "git diff --check " + v17ContractBaseGitCommitOID + " HEAD --" +
		" && go test -mod=vendor -count=1 . ./acceptance -run \"^(TestProductRoadmapIsExhaustiveAndCausal|TestProductRoadmapV17ScopeAndExecutableContract|TestV17EvidenceBelongsOnlyToTestAttestorCapabilities|TestV17AcceptanceCommandRunsTestAttestorConsumers|TestV17AcceptanceCommandUsesNoDestructiveTemporaryCleanup|TestRebuildArchitecture|TestTraceabilityRebuildBugLessons|TestTraceabilityRebuildHistoricalBugIDs|TestTraceabilityRebuildHistoricalBugReviewBindings|TestTraceabilityRebuildSchemaValidatesCanonicalLedgers|TestHistoricalBugCapabilityCoverageNeverInfersLegacyClosure|TestAcceptanceV02AuthorityRules|TestAcceptanceV03CanonicalLedgers|TestAcceptanceV06AtomicStateOutbox|TestAcceptanceV09RecoveryBackup|TestAcceptanceV15BudgetsEffects|TestAcceptanceV16WorkspaceGit|TestAcceptanceV17TestAttestor|TestV17CandidateSubjectsCoverCommittedDelta)$\""
	focal := "go test -mod=vendor -count=1 " + strings.Join(v17FocalPackages(), " ")
	race := "timeout --kill-after=10s 240s go test -mod=vendor -race -count=1 -timeout=210s " +
		strings.Join(v17RacePackages(), " ") + " -run \"^(" + strings.Join(v17RaceTests(), "|") + ")$\""
	e2e := "e2e_events=$(CGO_ENABLED=0 go test -mod=vendor -tags=" + v17RealE2ETag +
		" -json -count=1 -timeout=180s " + v17RealE2EPackage +
		" -run \"^" + v17RealE2ETest + "$\" 2>&1); e2e_status=$?; " +
		"printf '%s\\n' \"$e2e_events\"; [ \"$e2e_status\" -eq 0 ] && " +
		"printf '%s\\n' \"$e2e_events\" | grep -F \"\\\"Action\\\":\\\"run\\\"\" | " +
		"grep -F \"\\\"Package\\\":\\\"" + v17RealE2EImport + "\\\"\" | " +
		"grep -F \"\\\"Test\\\":\\\"" + v17RealE2ETest + "\\\"\" >/dev/null && " +
		"printf '%s\\n' \"$e2e_events\" | grep -F \"\\\"Action\\\":\\\"pass\\\"\" | " +
		"grep -F \"\\\"Package\\\":\\\"" + v17RealE2EImport + "\\\"\" | " +
		"grep -F \"\\\"Test\\\":\\\"" + v17RealE2ETest + "\\\"\" >/dev/null"
	vet := "GOFLAGS=-mod=vendor go vet " + strings.Join(v17FocalPackages(), " ")
	return strings.Join([]string{acceptance, focal, race, e2e, vet}, " && ")
}

func v17FocalPackages() []string {
	return []string{
		"./internal/goal", "./internal/ports", "./internal/application", "./internal/config",
		"./internal/adapters/agent/codex", "./internal/interfaces/mcp",
		"./internal/adapters/artifact/filesystem", "./internal/adapters/workspace/gitlocal",
		"./internal/adapters/attestor/bubblewrap", "./internal/adapters/state/sqlite",
		"./internal/bootstrap", "./cmd/orquesta",
	}
}

func v17RacePackages() []string {
	return []string{
		"./internal/application", "./internal/adapters/artifact/filesystem",
		"./internal/adapters/workspace/gitlocal", "./internal/adapters/attestor/bubblewrap",
		"./internal/adapters/state/sqlite", "./internal/bootstrap",
	}
}

func v17RaceTests() []string {
	return []string{
		"TestStoreConcurrentPutPreservesSinglePrivateBlob", "TestConcurrentStoreHandlesPublishOnePrivateBlob",
		"TestCloseIsConcurrentAndIdempotent", "TestPinnedGitConcurrentCommandsSurviveCloseRace",
		"TestConcurrentCommitReplayRestoresSnapshotChangeMarkerAcrossRestart",
		"TestConcurrentAttestationFenceAllowsOneWriter", "TestExpiredAttestClaimFenceCannotWriteAfterSpecialLeaseRecovery",
		"TestTestAttestationCrashFrontiersReplayExactlyOnce",
		"TestTerminalAttestationReceiptPreventsRestartRerun",
	}
}

func v17E2EPackages() []string {
	return []string{v17RealE2EPackage}
}

func v17E2ETests() []string {
	return []string{v17RealE2ETest}
}

func v17AssertSimplicityBudget(t *testing.T, repositoryRoot string, fixture v17Fixture, numstat []byte) {
	t.Helper()
	budget := fixture.SimplicityBudget
	evidenceAssertSimplicityBudget(t, "V17", numstat, v17SimplicityClass, map[string]int{
		"core": budget.CoreNetLines, "adapters": budget.AdapterNetLines,
		"migration": budget.MigrationNetLines, "tests": budget.TestNetLines,
	}, budget.ProductNetLines)
	changed, err := evidenceGit(repositoryRoot, "diff", "--no-renames", "--name-status",
		fixture.ProductDeltaBaseGitCommitOID, fixture.ProductDeltaSealedGitCommitOID, "--")
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range strings.Split(strings.TrimSpace(string(changed)), "\n") {
		fields := strings.SplitN(row, "\t", 2)
		if len(fields) != 2 || fields[0] != "A" {
			continue
		}
		class := v17SimplicityClass(fields[1])
		if (class != "core" && class != "adapters") || !strings.HasSuffix(fields[1], ".go") ||
			strings.HasSuffix(fields[1], "_test.go") {
			continue
		}
		if lines := v15LineCount(v15GitBlob(t, repositoryRoot, fixture.ProductDeltaSealedGitCommitOID, fields[1])); lines > budget.MaxAddedGoFileLines {
			t.Errorf("V17 added productive Go file %s has %d lines, max %d",
				fields[1], lines, budget.MaxAddedGoFileLines)
		}
	}
	v15AssertStructuralSimplicityWithClassifier(
		t, repositoryRoot, fixture.ProductDeltaBaseGitCommitOID, fixture.ProductDeltaSealedGitCommitOID, v17SimplicityClass,
	)
}

func v17SimplicityClass(relative string) string {
	switch {
	case strings.HasSuffix(relative, "_test.go"), strings.HasPrefix(relative, "acceptance/"):
		return "tests"
	case relative == "internal/adapters/state/sqlite/migrations/012_test_attestor.sql":
		return "migration"
	case strings.HasPrefix(relative, "internal/goal/"), strings.HasPrefix(relative, "internal/application/"),
		strings.HasPrefix(relative, "internal/ports/"), strings.HasPrefix(relative, "internal/governance/"):
		if strings.HasSuffix(relative, ".go") {
			return "core"
		}
	case strings.HasPrefix(relative, "internal/adapters/"), strings.HasPrefix(relative, "internal/bootstrap/"),
		strings.HasPrefix(relative, "internal/interfaces/"),
		strings.HasPrefix(relative, "internal/config/"), strings.HasPrefix(relative, "config/"),
		strings.HasPrefix(relative, "cmd/orquesta/"):
		return "adapters"
	case strings.HasPrefix(relative, "docs/reconstruccion/"), relative == "product/roadmap.json",
		relative == "product_roadmap_test.go", strings.HasPrefix(relative, "product/traceability/"):
		return "metadata"
	}
	return ""
}

func TestV17SimplicityClassCoversEveryCandidate(t *testing.T) {
	t.Parallel()
	for relative, want := range map[string]string{
		"internal/goal/required_tests.go":                                 "core",
		"internal/application/test_attestation_processing.go":             "core",
		"internal/ports/test_attestor.go":                                 "core",
		"internal/governance/budget.go":                                   "core",
		"internal/adapters/attestor/bubblewrap/config.go":                 "adapters",
		"internal/interfaces/mcp/tools.go":                                "adapters",
		"internal/bootstrap/runtime.go":                                   "adapters",
		"internal/config/registry.go":                                     "adapters",
		"config/registry.json":                                            "adapters",
		"cmd/orquesta/main.go":                                            "adapters",
		"internal/adapters/state/sqlite/migrations/012_test_attestor.sql": "migration",
		"internal/application/test_attestation_flow_test.go":              "tests",
		"acceptance/fixtures/v17_test_attestor.json":                      "tests",
		"product/traceability/rebuild_bugs.jsonl":                         "metadata",
	} {
		if got := v17SimplicityClass(relative); got != want {
			t.Errorf("v17SimplicityClass(%q)=%q, want %q", relative, got, want)
		}
	}
	repositoryRoot := evidenceRepositoryRoot(t)
	fixture := evidenceDecodeStrictJSON[v17Fixture](t, filepath.Join(repositoryRoot, v17FixturePath))
	for _, relative := range fixture.CandidateSubjects {
		if got := v17SimplicityClass(relative); got == "" {
			t.Errorf("v17SimplicityClass(%q) is empty", relative)
		}
	}
}

func TestV17CanonicalBudgetCountsTopLevelConfigAndTestsBeforeProduct(t *testing.T) {
	t.Parallel()
	for path, want := range map[string]string{
		"config/registry.json": "adapters", "product_roadmap_test.go": "tests",
		"internal/config/config_test.go": "tests", "internal/config/registry.go": "adapters",
	} {
		if got := v17SimplicityClass(path); got != want {
			t.Errorf("v17SimplicityClass(%q)=%q, want %q", path, got, want)
		}
	}
}
