package acceptance_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"orquesta/internal/goal"
)

const v04FixturePath = "acceptance/fixtures/v04_intent_appspec.json"

type v04Fixture struct {
	SchemaVersion                  int      `json:"schema_version"`
	ReceiptSchemaVersion           int      `json:"receipt_schema_version"`
	ContractID                     string   `json:"contract_id"`
	TrustedBaseGitCommitOID        string   `json:"trusted_base_git_commit_oid"`
	ProductDeltaBaseGitCommitOID   string   `json:"product_delta_base_git_commit_oid"`
	ProductDeltaSealedGitCommitOID string   `json:"product_delta_sealed_git_commit_oid"`
	Command                        string   `json:"command"`
	ExecutionArgv                  []string `json:"execution_argv"`
	OutputPath                     string   `json:"output_path"`
	ReceiptPath                    string   `json:"receipt_path"`
	CandidateSubjects              []string `json:"candidate_subjects"`
	Initial                        struct {
		IntentRef           string    `json:"intent_ref"`
		ActorRef            string    `json:"actor_ref"`
		ProjectRef          string    `json:"project_ref"`
		Statement           string    `json:"statement"`
		SubmittedAt         time.Time `json:"submitted_at"`
		NormalizedObjective string    `json:"normalized_objective"`
		Reason              string    `json:"reason"`
		Confirm             bool      `json:"confirm"`
	} `json:"initial"`
	Amendment struct {
		RequestRef          string `json:"request_ref"`
		IntentRef           string `json:"intent_ref"`
		Statement           string `json:"statement"`
		NormalizedObjective string `json:"normalized_objective"`
		Reason              string `json:"reason"`
		ExpectedGeneration  uint64 `json:"expected_generation"`
		Confirm             bool   `json:"confirm"`
	} `json:"amendment"`
	Assertions []string `json:"assertions"`
}

func TestV04CandidateSubjectsCoverCommittedDelta(t *testing.T) {
	repositoryRoot := evidenceRepositoryRoot(t)
	fixture := evidenceDecodeStrictJSON[v04Fixture](t, filepath.Join(repositoryRoot, filepath.FromSlash(v04FixturePath)))
	if len(fixture.ProductDeltaBaseGitCommitOID) != 40 {
		t.Fatalf("invalid V04 product delta base %q", fixture.ProductDeltaBaseGitCommitOID)
	}
	candidates := make(map[string]struct{}, len(fixture.CandidateSubjects))
	for _, subject := range fixture.CandidateSubjects {
		candidates[subject] = struct{}{}
	}
	changed := v04CandidateChangedPaths(t, repositoryRoot, fixture)
	seen := make(map[string]struct{}, len(changed))
	for _, relative := range changed {
		if relative == "" {
			continue
		}
		if _, duplicate := seen[relative]; duplicate {
			continue
		}
		seen[relative] = struct{}{}
		if v04AllowedOutsideCandidate(relative) {
			continue
		}
		if _, declared := candidates[relative]; !declared {
			t.Errorf("V04 delta path is outside candidate_subjects: %s", relative)
		}
	}
}

func TestV04AcceptanceCommandRunsEachOwnedPackageBehavior(t *testing.T) {
	repositoryRoot := evidenceRepositoryRoot(t)
	fixture := evidenceDecodeStrictJSON[v04Fixture](t, filepath.Join(repositoryRoot, filepath.FromSlash(v04FixturePath)))
	want := "sh -c '" + v04ValidationShellBody() + "'"
	if fixture.Command != want {
		t.Fatalf("V04 command does not run the exact structural and unfiltered behavioral surfaces:\n got: %s\nwant: %s", fixture.Command, want)
	}
	wantArgv := []string{"sh", "-c", v04ValidationShellBody()}
	if !stringSlicesEqual(fixture.ExecutionArgv, wantArgv) {
		t.Fatalf("V04 execution argv=%v, want exact gate %v", fixture.ExecutionArgv, wantArgv)
	}
}

func v04ValidationShellBody() string {
	structural := "go test -mod=vendor -count=1 ./acceptance -run \"^(TestAcceptanceV04IntentAppSpec|TestV04.*)$\""
	behaviorPackages := []string{
		".",
		"./internal/goal",
		"./internal/application",
		"./internal/ports",
		"./internal/adapters/agent/fake",
		"./internal/adapters/agent/codex",
		"./internal/adapters/state/sqlite",
		"./internal/interfaces/mcp",
		"./internal/i18n",
		"./internal/bootstrap",
		"./cmd/orquesta",
	}
	return structural + " && go test -mod=vendor -count=1 " + strings.Join(behaviorPackages, " ")
}

func TestV04CandidateDeltaFreezesAtSealedHead(t *testing.T) {
	repositoryRoot := evidenceRepositoryRoot(t)
	fixture := evidenceDecodeStrictJSON[v04Fixture](t, filepath.Join(repositoryRoot, filepath.FromSlash(v04FixturePath)))
	if len(fixture.ProductDeltaSealedGitCommitOID) != 40 || fixture.ProductDeltaSealedGitCommitOID == fixture.ProductDeltaBaseGitCommitOID {
		t.Fatalf("invalid sealed V04 product delta %q from base %q", fixture.ProductDeltaSealedGitCommitOID, fixture.ProductDeltaBaseGitCommitOID)
	}
	command := exec.Command("git", "-C", repositoryRoot, "merge-base", "--is-ancestor", fixture.ProductDeltaBaseGitCommitOID, fixture.ProductDeltaSealedGitCommitOID)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("V04 delta_git_head is not an existing descendant commit: %v: %s", err, output)
	}
	sealed := strings.Join(v04CandidateDiffArguments(fixture), "\x00")
	wantSealed := strings.Join([]string{"diff", "--name-only", fixture.ProductDeltaBaseGitCommitOID, fixture.ProductDeltaSealedGitCommitOID, "--"}, "\x00")
	if sealed != wantSealed {
		t.Fatalf("V04 historical product delta arguments are not frozen: got=%q want=%q", sealed, wantSealed)
	}
}

type v04PackageShape struct {
	Types      map[string]struct{}
	Fields     map[string]map[string]struct{}
	FieldTypes map[string][]string
	Methods    map[string]map[string]struct{}
	Constants  map[string]string
}

func TestAcceptanceV04IntentAppSpec(t *testing.T) {
	repositoryRoot := evidenceRepositoryRoot(t)
	fixture := evidenceDecodeStrictJSON[v04Fixture](t, filepath.Join(repositoryRoot, filepath.FromSlash(v04FixturePath)))
	if fixture.SchemaVersion != 1 || fixture.ReceiptSchemaVersion != 3 || fixture.ContractID != "AC-V04-INTENT-APPSPEC" ||
		fixture.TrustedBaseGitCommitOID != "a301a3bbacd80c1ea2d47422a2964339dcd70980" || len(fixture.ExecutionArgv) != 3 ||
		fixture.OutputPath != "product/evidence/v04_intent_appspec.output.txt" || fixture.ReceiptPath != "product/evidence/v04_intent_appspec.json" || len(fixture.Assertions) != 8 {
		t.Fatalf("invalid V04 fixture header: %+v", fixture)
	}
	if !sort.StringsAreSorted(fixture.CandidateSubjects) || v04HasDuplicate(fixture.CandidateSubjects) {
		t.Fatalf("candidate subjects must be unique and sorted: %v", fixture.CandidateSubjects)
	}

	t.Run("intent_manifest_preserves_exact_input", func(t *testing.T) {
		v04AssertExactIntent(t, fixture)
	})

	t.Run("single_domain_chain", func(t *testing.T) {
		shape := v04LoadPackageShape(t, repositoryRoot, "internal/goal")
		v04RequireType(t, shape, "AppSpec")
		v04RequireFieldType(t, shape, "Goal", "AppSpec")
		v04ForbidFieldType(t, shape, "Goal", "IntentManifest")
		v04RequireMethods(t, shape, "Goal", "AppSpec", "Intent", "SpecHash")
	})

	t.Run("application_owns_confirmed_create_and_amend", func(t *testing.T) {
		shape := v04LoadPackageShape(t, repositoryRoot, "internal/application")
		v04RequireFields(t, shape, "SubmitRequest", "Confirm", "NormalizedObjective")
		v04RequireFields(t, shape, "AmendRequest", "Confirm", "ExpectedSourceRevision", "ExpectedSourceSpecHash", "NormalizedObjective", "Reason", "RequestRef", "SourceGoalRef", "Statement")
		v04RequireMethods(t, shape, "Orchestrator", "Amend", "Submit")
		v04RequireConstant(t, shape, "initialAppSpecReason", fixture.Initial.Reason)
	})

	t.Run("provider_boundary_echoes_spec_hash", func(t *testing.T) {
		shape := v04LoadPackageShape(t, repositoryRoot, "internal/ports")
		for _, typeName := range []string{"AgentLaunchRequest", "AgentLaunchReceipt", "AgentObservation"} {
			v04RequireFields(t, shape, typeName, "SpecHash")
		}
	})

	t.Run("sqlite_and_mcp_are_real_adapters", func(t *testing.T) {
		v04AssertSQLiteMigration(t, repositoryRoot)
		shape := v04LoadPackageShape(t, repositoryRoot, "internal/interfaces/mcp")
		v04RequireConstant(t, shape, "ToolGoalsAmend", "orquesta.goals.amend")
		v04RequireFields(t, shape, "CreateGoalInput", "Confirm", "NormalizedObjective")
		v04RequireFields(t, shape, "AmendGoalInput", "Confirm", "ExpectedSourceRevision", "ExpectedSourceSpecHash", "NormalizedObjective", "Reason", "RequestRef", "SourceGoalRef", "Statement")
	})
}

func TestAcceptanceV04IntentAppSpecReceipt(t *testing.T) {
	repositoryRoot := evidenceRepositoryRoot(t)
	evidenceAssertReceiptV3(t, repositoryRoot, evidenceReceiptV3Expectation{
		Contract:                "AC-V04-INTENT-APPSPEC",
		FixturePath:             v04FixturePath,
		ReceiptPath:             "product/evidence/v04_intent_appspec.json",
		ExecutedNotBefore:       "2026-07-14T00:00:00+02:00",
		TrustedBaseGitCommitOID: "a301a3bbacd80c1ea2d47422a2964339dcd70980",
	})
}

func v04AssertExactIntent(t *testing.T, fixture v04Fixture) {
	t.Helper()
	intentRef, err := goal.NewIntentRef(fixture.Initial.IntentRef)
	if err != nil {
		t.Fatal(err)
	}
	actorRef, err := goal.NewActorRef(fixture.Initial.ActorRef)
	if err != nil {
		t.Fatal(err)
	}
	projectRef, err := goal.NewProjectRef(fixture.Initial.ProjectRef)
	if err != nil {
		t.Fatal(err)
	}
	input := goal.IntentManifestInput{
		Ref: intentRef, Actor: actorRef, Project: projectRef,
		Statement: fixture.Initial.Statement, SubmittedAt: fixture.Initial.SubmittedAt,
	}
	manifest, err := goal.NewIntentManifest(input)
	if err != nil {
		t.Fatalf("NewIntentManifest() error = %v", err)
	}
	if manifest.Statement() != fixture.Initial.Statement || manifest.Hash() == "" {
		t.Fatalf("IntentManifest normalized or omitted exact input: statement=%q hash=%q", manifest.Statement(), manifest.Hash())
	}
	replayed, err := goal.NewIntentManifest(input)
	if err != nil || replayed.Hash() != manifest.Hash() {
		t.Fatalf("IntentManifest hash is not deterministic: replay=%q original=%q err=%v", replayed.Hash(), manifest.Hash(), err)
	}
	input.Statement += " "
	changed, err := goal.NewIntentManifest(input)
	if err != nil || changed.Hash() == manifest.Hash() {
		t.Fatalf("IntentManifest hash did not bind exact statement bytes: changed=%q original=%q err=%v", changed.Hash(), manifest.Hash(), err)
	}
}

func v04LoadPackageShape(t *testing.T, repositoryRoot, relative string) v04PackageShape {
	t.Helper()
	directory := filepath.Join(repositoryRoot, filepath.FromSlash(relative))
	packages, err := parser.ParseDir(token.NewFileSet(), directory, func(info os.FileInfo) bool {
		return !strings.HasSuffix(info.Name(), "_test.go")
	}, 0)
	if err != nil {
		t.Errorf("parse package %s: %v", relative, err)
		return v04PackageShape{}
	}
	shape := v04PackageShape{
		Types: make(map[string]struct{}), Fields: make(map[string]map[string]struct{}),
		FieldTypes: make(map[string][]string), Methods: make(map[string]map[string]struct{}),
		Constants: make(map[string]string),
	}
	for _, parsedPackage := range packages {
		if strings.HasSuffix(parsedPackage.Name, "_test") {
			continue
		}
		for _, file := range parsedPackage.Files {
			v04CollectDeclarations(shape, file)
		}
	}
	return shape
}

func v04CollectDeclarations(shape v04PackageShape, file *ast.File) {
	for _, declaration := range file.Decls {
		switch typed := declaration.(type) {
		case *ast.GenDecl:
			for _, rawSpec := range typed.Specs {
				switch spec := rawSpec.(type) {
				case *ast.TypeSpec:
					shape.Types[spec.Name.Name] = struct{}{}
					structure, ok := spec.Type.(*ast.StructType)
					if !ok {
						continue
					}
					if shape.Fields[spec.Name.Name] == nil {
						shape.Fields[spec.Name.Name] = make(map[string]struct{})
					}
					for _, field := range structure.Fields.List {
						fieldType := v04ExpressionName(field.Type)
						shape.FieldTypes[spec.Name.Name] = append(shape.FieldTypes[spec.Name.Name], fieldType)
						for _, name := range field.Names {
							shape.Fields[spec.Name.Name][name.Name] = struct{}{}
						}
					}
				case *ast.ValueSpec:
					if typed.Tok != token.CONST {
						continue
					}
					for index, name := range spec.Names {
						value := ""
						if index < len(spec.Values) {
							if literal, ok := spec.Values[index].(*ast.BasicLit); ok && literal.Kind == token.STRING {
								value, _ = strconv.Unquote(literal.Value)
							}
						}
						shape.Constants[name.Name] = value
					}
				}
			}
		case *ast.FuncDecl:
			if typed.Recv == nil || len(typed.Recv.List) != 1 {
				continue
			}
			receiver := v04ExpressionName(typed.Recv.List[0].Type)
			if shape.Methods[receiver] == nil {
				shape.Methods[receiver] = make(map[string]struct{})
			}
			shape.Methods[receiver][typed.Name.Name] = struct{}{}
		}
	}
}

func v04ExpressionName(expression ast.Expr) string {
	switch typed := expression.(type) {
	case *ast.Ident:
		return typed.Name
	case *ast.StarExpr:
		return v04ExpressionName(typed.X)
	case *ast.SelectorExpr:
		return v04ExpressionName(typed.X) + "." + typed.Sel.Name
	case *ast.ArrayType:
		return "[]" + v04ExpressionName(typed.Elt)
	default:
		return ""
	}
}

func v04RequireType(t *testing.T, shape v04PackageShape, name string) {
	t.Helper()
	if _, ok := shape.Types[name]; !ok {
		t.Errorf("required type %s is missing", name)
	}
}

func v04RequireFields(t *testing.T, shape v04PackageShape, typeName string, fields ...string) {
	t.Helper()
	for _, field := range fields {
		if _, ok := shape.Fields[typeName][field]; !ok {
			t.Errorf("%s.%s is missing", typeName, field)
		}
	}
}

func v04RequireFieldType(t *testing.T, shape v04PackageShape, typeName, fieldType string) {
	t.Helper()
	for _, candidate := range shape.FieldTypes[typeName] {
		if candidate == fieldType {
			return
		}
	}
	t.Errorf("%s lacks field of type %s", typeName, fieldType)
}

func v04ForbidFieldType(t *testing.T, shape v04PackageShape, typeName, fieldType string) {
	t.Helper()
	for _, candidate := range shape.FieldTypes[typeName] {
		if candidate == fieldType {
			t.Errorf("%s retains duplicate authority field of type %s", typeName, fieldType)
		}
	}
}

func v04RequireMethods(t *testing.T, shape v04PackageShape, receiver string, methods ...string) {
	t.Helper()
	for _, method := range methods {
		if _, ok := shape.Methods[receiver][method]; !ok {
			t.Errorf("%s.%s is missing", receiver, method)
		}
	}
}

func v04RequireConstant(t *testing.T, shape v04PackageShape, name, value string) {
	t.Helper()
	if got, ok := shape.Constants[name]; !ok || got != value {
		t.Errorf("constant %s=%q, want %q", name, got, value)
	}
}

func v04AssertSQLiteMigration(t *testing.T, repositoryRoot string) {
	t.Helper()
	path := filepath.Join(repositoryRoot, "internal/adapters/state/sqlite/migrations/003_app_specs.sql")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Errorf("read SQLite V04 migration: %v", err)
		return
	}
	normalized := strings.ToLower(string(content))
	for _, fragment := range []string{
		"create table app_specs", "orquesta:go-backfill app_specs",
		"before update on intents", "before delete on intents",
		"before update on app_specs", "before delete on app_specs",
	} {
		if !strings.Contains(normalized, fragment) {
			t.Errorf("SQLite V04 migration lacks %q", fragment)
		}
	}
}

func v04HasDuplicate(values []string) bool {
	for index := 1; index < len(values); index++ {
		if values[index] == values[index-1] {
			return true
		}
	}
	return false
}

func v04GitPaths(t *testing.T, repositoryRoot string, arguments ...string) []string {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", repositoryRoot}, arguments...)...)
	output, err := command.Output()
	if err != nil {
		t.Fatalf("git %s: %v", strings.Join(arguments, " "), err)
	}
	text := strings.TrimSpace(string(output))
	if text == "" {
		return nil
	}
	return strings.Split(text, "\n")
}

func v04CandidateChangedPaths(t *testing.T, repositoryRoot string, fixture v04Fixture) []string {
	t.Helper()
	return v04GitPaths(t, repositoryRoot, v04CandidateDiffArguments(fixture)...)
}

func v04CandidateDiffArguments(fixture v04Fixture) []string {
	return []string{"diff", "--name-only", fixture.ProductDeltaBaseGitCommitOID, fixture.ProductDeltaSealedGitCommitOID, "--"}
}

func v04AllowedOutsideCandidate(relative string) bool {
	if relative == "docs/reconstruccion/estado_y_handoff_rebuild.md" {
		return true
	}
	for _, prefix := range []string{
		"product/evidence/v01_source_integration.",
		"product/evidence/v03_canonical_ledgers.",
		"product/evidence/v04_intent_appspec.",
	} {
		if strings.HasPrefix(relative, prefix) {
			return true
		}
	}
	return false
}
