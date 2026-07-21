package acceptance_test

import (
	"bytes"
	"context"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"

	"orquesta/internal/application"
)

const v16FixturePath = "acceptance/fixtures/v16_workspace_git.json"
const v16ContractBaseGitCommitOID = "3820df2ae89f1a217de1b14d5b88abf8e86c898b"

type v16Fixture struct {
	SchemaVersion            int               `json:"schema_version"`
	ContractID               string            `json:"contract_id"`
	TrustedBaseGitCommitOID  string            `json:"trusted_base_git_commit_oid"`
	Command                  string            `json:"command"`
	ExecutionArgv            []string          `json:"execution_argv"`
	OwnedCapabilityIDs       []string          `json:"owned_capability_ids"`
	DependencyVerticals      []string          `json:"dependency_verticals"`
	RequiredApplicationTypes []v16RequiredType `json:"required_application_types"`
	RequiredOutboundPorts    []v16RequiredPort `json:"required_outbound_ports"`
	RequiredUseCases         []string          `json:"required_use_cases"`
	RequiredActions          []string          `json:"required_actions"`
	RequiredEffectKinds      []string          `json:"required_effect_kinds"`
	RequiredStatuses         []string          `json:"required_statuses"`
	ForbiddenAuthorities     []string          `json:"forbidden_private_authorities"`
	RequiredBehaviorTests    []string          `json:"required_behavior_tests"`
	GitFixture               v16GitFixture     `json:"git_fixture"`
	Actors                   []v16Actor        `json:"actors"`
	Changes                  []v16Change       `json:"changes"`
	CrashFrontiers           []string          `json:"crash_frontiers"`
	SecurityCases            []string          `json:"security_cases"`
	PrivateLeakMarkers       []string          `json:"private_leak_markers"`
	DeferredSurfaces         []string          `json:"deferred_surfaces"`
}

type v16RequiredType struct {
	Name   string   `json:"name"`
	Fields []string `json:"fields"`
}

type v16RequiredPort struct {
	Name    string   `json:"name"`
	Methods []string `json:"methods"`
}

type v16GitFixture struct {
	MinimumVersion string            `json:"minimum_version"`
	ObjectFormat   string            `json:"object_format"`
	TargetRef      string            `json:"target_ref"`
	AuthorName     string            `json:"author_name"`
	AuthorEmail    string            `json:"author_email"`
	BaseOID        string            `json:"base_oid"`
	BaseTreeOID    string            `json:"base_tree_oid"`
	TargetOID      string            `json:"target_oid"`
	TargetTreeOID  string            `json:"target_tree_oid"`
	BaseUnixTime   int64             `json:"base_unix_time"`
	TargetUnixTime int64             `json:"target_unix_time"`
	BaseFiles      []v16FixtureWrite `json:"base_files"`
	TargetChanges  []v16FixtureWrite `json:"target_changes"`
}

type v16Actor struct {
	PrincipalRef string          `json:"principal_ref"`
	ActorRef     string          `json:"actor_ref"`
	Memberships  []v16Membership `json:"memberships"`
}

type v16Membership struct {
	ProjectRef    string `json:"project_ref"`
	RepositoryRef string `json:"repository_ref"`
	Role          string `json:"role"`
}

type v16Change struct {
	Name         string            `json:"name"`
	ExecutionRef string            `json:"execution_ref"`
	WriteSet     []string          `json:"write_set"`
	Writes       []v16FixtureWrite `json:"writes"`
	Expected     string            `json:"expected"`
}

type v16FixtureWrite struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

func TestAcceptanceV16WorkspaceGit(t *testing.T) {
	repositoryRoot := evidenceRepositoryRoot(t)
	fixture := evidenceDecodeStrictJSON[v16Fixture](t, filepath.Join(repositoryRoot, filepath.FromSlash(v16FixturePath)))
	v16AssertFixture(t, repositoryRoot, fixture)

	applicationDirectory := filepath.Join(repositoryRoot, "internal", "application")
	portsDirectory := filepath.Join(repositoryRoot, "internal", "ports")

	t.Run("one_writer_and_two_outbound_ports", func(t *testing.T) {
		statePort := reflect.TypeOf((*application.StateRepository)(nil)).Elem()
		dependencies := reflect.TypeOf(application.Dependencies{})
		stateFields := 0
		for index := 0; index < dependencies.NumField(); index++ {
			if dependencies.Field(index).Type == statePort {
				stateFields++
			}
		}
		if stateFields != 1 {
			t.Errorf("V16_RED Dependencies StateRepository fields=%d, want 1", stateFields)
		}
		for _, port := range fixture.RequiredOutboundPorts {
			v16RequireOutboundInterface(t, applicationDirectory, port)
			if count := v16StructFieldTypeCount(t, applicationDirectory, "Dependencies", port.Name); count != 1 {
				t.Errorf("V16_RED Dependencies fields of type %s=%d, want 1", port.Name, count)
			}
		}
		production := v16ReadProductionGo(t, applicationDirectory)
		for _, forbidden := range fixture.ForbiddenAuthorities {
			if strings.Contains(production, "type "+forbidden+" ") {
				t.Errorf("V16 private authority %s is forbidden", forbidden)
			}
		}
	})

	t.Run("immutable_facts_are_causal_and_path_free", func(t *testing.T) {
		for _, required := range fixture.RequiredApplicationTypes {
			v14RequireProductionFields(t, applicationDirectory, required.Name, required.Fields)
			fields, found := v13ProductionTypeFields(t, applicationDirectory, required.Name)
			if found {
				for _, forbidden := range []string{"Path", "WorkspacePath", "RepositoryPath", "URL", "Argv", "Environment", "Secret"} {
					if fields[forbidden] {
						t.Errorf("V16 %s leaks adapter-private field %s", required.Name, forbidden)
					}
				}
			}
		}
		if !v16ProductionTypeExists(t, portsDirectory, "ExecutionWorkspaceRef") {
			t.Error("V16_RED ports.ExecutionWorkspaceRef missing")
		}
		v14RequireProductionFields(t, portsDirectory, "AgentLaunchRequest", []string{"ExecutionWorkspaceRef"})
	})

	t.Run("public_use_cases_admit_and_query_but_do_not_create_lifecycle", func(t *testing.T) {
		orchestrator := reflect.TypeOf((*application.Orchestrator)(nil))
		for _, useCase := range fixture.RequiredUseCases {
			v16RequireUseCase(t, orchestrator, useCase)
		}
		production := v16ReadProductionGo(t, applicationDirectory)
		for _, value := range append(append([]string{}, fixture.RequiredActions...), fixture.RequiredEffectKinds...) {
			if !strings.Contains(production, `"`+value+`"`) {
				t.Errorf("V16_RED application contract lacks %q", value)
			}
		}
		for _, status := range fixture.RequiredStatuses {
			if !strings.Contains(production, `"`+status+`"`) {
				t.Errorf("V16_RED application contract lacks status %q", status)
			}
		}
	})

	t.Run("sqlite_config_git_adapter_and_bootstrap_are_real", func(t *testing.T) {
		migration := filepath.Join(repositoryRoot, "internal", "adapters", "state", "sqlite", "migrations", "011_workspace_git.sql")
		content, err := os.ReadFile(migration)
		if err != nil {
			t.Errorf("V16_RED SQLite migration 011 missing: %v", err)
		} else {
			lower := strings.ToLower(string(content))
			for _, token := range []string{"workspace", "change", "merge", "integration", "outbox", "effect"} {
				if !strings.Contains(lower, token) {
					t.Errorf("V16_RED migration lacks %s", token)
				}
			}
		}
		registry, err := os.ReadFile(filepath.Join(repositoryRoot, "config", "registry.json"))
		if err != nil {
			t.Fatal(err)
		}
		if count := strings.Count(string(registry), `"workspace.local.root"`); count != 1 {
			t.Errorf("V16_RED canonical workspace.local.root definitions=%d, want 1", count)
		}
		adapterDirectory := filepath.Join(repositoryRoot, "internal", "adapters", "workspace", "gitlocal")
		adapterSource := v16ReadProductionGoOptional(t, adapterDirectory)
		for _, required := range []string{"exec.CommandContext", "merge-tree", "update-ref", "--porcelain", "--lock"} {
			if !strings.Contains(adapterSource, required) {
				t.Errorf("V16_RED Git CLI adapter lacks %q", required)
			}
		}
		for _, forbidden := range []string{"go-git", "libgit2", "sh -c", `"merge"`} {
			if strings.Contains(adapterSource, forbidden) {
				t.Errorf("V16 Git adapter contains forbidden mechanism %q", forbidden)
			}
		}
	})

	t.Run("all_behavior_security_restart_e2e_and_race_gates_exist", func(t *testing.T) {
		testSource := v16ReadGoTests(t,
			applicationDirectory,
			portsDirectory,
			filepath.Join(repositoryRoot, "internal", "adapters", "state", "sqlite"),
			filepath.Join(repositoryRoot, "internal", "adapters", "workspace", "gitlocal"),
			filepath.Join(repositoryRoot, "internal", "adapters", "agent", "codex"),
			filepath.Join(repositoryRoot, "internal", "bootstrap"),
		)
		for _, name := range fixture.RequiredBehaviorTests {
			if !strings.Contains(testSource, "func "+name+"(") {
				t.Errorf("V16_RED executable behavior test missing: %s", name)
			}
		}
	})
}

func v16AssertFixture(t *testing.T, repositoryRoot string, fixture v16Fixture) {
	t.Helper()
	if fixture.SchemaVersion != 1 || fixture.ContractID != "AC-V16-WORKSPACE-GIT" ||
		fixture.TrustedBaseGitCommitOID != v16ContractBaseGitCommitOID ||
		fixture.Command != "go test -mod=vendor -count=1 ./acceptance -run '^TestAcceptanceV16WorkspaceGit$'" ||
		!reflect.DeepEqual(fixture.ExecutionArgv, []string{
			"go", "test", "-mod=vendor", "-count=1", "./acceptance", "-run", "^TestAcceptanceV16WorkspaceGit$",
		}) ||
		!reflect.DeepEqual(fixture.OwnedCapabilityIDs, []string{"STG-02", "STG-10", "EXT-10"}) ||
		!reflect.DeepEqual(fixture.DependencyVerticals, []string{
			"goal_dag_phases", "atomic_state_outbox", "identity_projects_rbac", "controls", "budgets_effects",
		}) ||
		!reflect.DeepEqual(fixture.RequiredUseCases, []string{"ListPendingChanges", "IntegrateChange"}) ||
		!reflect.DeepEqual(fixture.RequiredActions, []string{"prepare_workspace", "commit_change", "integrate_change"}) ||
		!reflect.DeepEqual(fixture.RequiredEffectKinds, fixture.RequiredActions) ||
		!reflect.DeepEqual(fixture.RequiredStatuses, []string{"clean", "conflicted", "stale", "integrated"}) {
		t.Fatalf("invalid V16 fixture identity: %+v", fixture)
	}
	if len(fixture.RequiredApplicationTypes) != 4 || len(fixture.RequiredOutboundPorts) != 2 ||
		len(fixture.RequiredBehaviorTests) != 15 || len(fixture.CrashFrontiers) != 7 ||
		len(fixture.SecurityCases) != 20 || len(fixture.PrivateLeakMarkers) != 5 {
		t.Fatalf("invalid V16 fixture coverage counts: %+v", fixture)
	}
	if fixture.GitFixture.MinimumVersion != "2.38.0" || fixture.GitFixture.ObjectFormat != "sha1" ||
		fixture.GitFixture.TargetRef != "refs/heads/main" || fixture.GitFixture.AuthorName == "" ||
		fixture.GitFixture.AuthorEmail == "" || fixture.GitFixture.BaseUnixTime <= 0 ||
		fixture.GitFixture.TargetUnixTime <= fixture.GitFixture.BaseUnixTime ||
		len(fixture.GitFixture.BaseFiles) != 2 || len(fixture.GitFixture.TargetChanges) != 2 {
		t.Fatalf("invalid V16 real Git fixture: %+v", fixture.GitFixture)
	}
	oid := regexp.MustCompile(`^[0-9a-f]{40}$`)
	for _, value := range []string{
		fixture.GitFixture.BaseOID, fixture.GitFixture.BaseTreeOID,
		fixture.GitFixture.TargetOID, fixture.GitFixture.TargetTreeOID,
	} {
		if !oid.MatchString(value) {
			t.Fatalf("invalid V16 sha1 object id %q", value)
		}
	}
	if len(fixture.Actors) != 2 || len(fixture.Actors[0].Memberships) != 1 || len(fixture.Actors[1].Memberships) != 2 ||
		len(fixture.Changes) != 3 || fixture.Changes[0].Expected != "clean" ||
		fixture.Changes[1].Expected != "conflicted" || fixture.Changes[2].Expected != "rejected_without_git_mutation" {
		t.Fatalf("invalid V16 actor/change scenarios: actors=%+v changes=%+v", fixture.Actors, fixture.Changes)
	}
	for _, change := range fixture.Changes {
		if change.Name == "" || change.ExecutionRef == "" || len(change.WriteSet) == 0 || len(change.Writes) == 0 {
			t.Fatalf("incomplete V16 change scenario: %+v", change)
		}
	}
	if !reflect.DeepEqual(fixture.DeferredSurfaces, []string{
		"remote_push", "github_gitlab_gitea_forge_adapters", "hostile_process_sandbox_and_attestation",
		"automatic_retention_cleanup", "postgres_s3_multihost",
	}) {
		t.Fatalf("invalid V16 deferred boundary: %v", fixture.DeferredSurfaces)
	}
	if _, err := evidenceGitCanonicalCommit(repositoryRoot, fixture.TrustedBaseGitCommitOID); err != nil {
		t.Fatalf("invalid V16 contract base: %v", err)
	}
}

func v16RequireOutboundInterface(t *testing.T, directory string, required v16RequiredPort) {
	t.Helper()
	methods, found := v16InterfaceMethods(t, directory, required.Name)
	if !found {
		t.Errorf("V16_RED application outbound port %s missing", required.Name)
		return
	}
	for _, name := range required.Methods {
		signature, ok := methods[name]
		if !ok {
			t.Errorf("V16_RED %s lacks %s", required.Name, name)
			continue
		}
		if !strings.Contains(signature, "context.Context") || strings.Count(signature, "ports.") < 2 ||
			!strings.Contains(signature, "error") {
			t.Errorf("V16_RED %s.%s must use context plus neutral ports request/result: %s", required.Name, name, signature)
		}
	}
}

func v16InterfaceMethods(t *testing.T, directory, typeName string) (map[string]string, bool) {
	t.Helper()
	for _, file := range v16ProductionGoFiles(t, directory, false) {
		parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", file, err)
		}
		for _, declaration := range parsed.Decls {
			general, ok := declaration.(*ast.GenDecl)
			if !ok || general.Tok != token.TYPE {
				continue
			}
			for _, specification := range general.Specs {
				typeSpec, ok := specification.(*ast.TypeSpec)
				if !ok || typeSpec.Name.Name != typeName {
					continue
				}
				iface, ok := typeSpec.Type.(*ast.InterfaceType)
				if !ok {
					return nil, true
				}
				methods := make(map[string]string)
				for _, field := range iface.Methods.List {
					if len(field.Names) != 1 {
						continue
					}
					var rendered bytes.Buffer
					if err := format.Node(&rendered, token.NewFileSet(), field.Type); err != nil {
						t.Fatalf("render %s.%s: %v", typeName, field.Names[0].Name, err)
					}
					methods[field.Names[0].Name] = rendered.String()
				}
				return methods, true
			}
		}
	}
	return nil, false
}

func v16RequireUseCase(t *testing.T, orchestrator reflect.Type, name string) {
	t.Helper()
	method, ok := orchestrator.MethodByName(name)
	if !ok {
		t.Errorf("V16_RED Orchestrator lacks %s", name)
		return
	}
	contextType := reflect.TypeOf((*context.Context)(nil)).Elem()
	errorType := reflect.TypeOf((*error)(nil)).Elem()
	if method.Type.NumIn() != 4 || method.Type.In(1) != contextType ||
		method.Type.In(2) != reflect.TypeOf(application.Access{}) ||
		method.Type.In(3).Name() != name+"Request" || method.Type.NumOut() != 2 ||
		method.Type.Out(0).Name() != name+"Result" || method.Type.Out(1) != errorType {
		t.Errorf("V16_RED %s signature=%s", name, method.Type)
	}
}

func v16StructFieldTypeCount(t *testing.T, directory, typeName, fieldType string) int {
	t.Helper()
	for _, file := range v16ProductionGoFiles(t, directory, false) {
		parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", file, err)
		}
		for _, declaration := range parsed.Decls {
			general, ok := declaration.(*ast.GenDecl)
			if !ok || general.Tok != token.TYPE {
				continue
			}
			for _, specification := range general.Specs {
				typeSpec, ok := specification.(*ast.TypeSpec)
				if !ok || typeSpec.Name.Name != typeName {
					continue
				}
				structure, ok := typeSpec.Type.(*ast.StructType)
				if !ok {
					return 0
				}
				count := 0
				for _, field := range structure.Fields.List {
					identifier, ok := field.Type.(*ast.Ident)
					if ok && identifier.Name == fieldType {
						count += len(field.Names)
					}
				}
				return count
			}
		}
	}
	return 0
}

func v16ProductionTypeExists(t *testing.T, directory, typeName string) bool {
	t.Helper()
	for _, file := range v16ProductionGoFiles(t, directory, false) {
		parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", file, err)
		}
		for _, declaration := range parsed.Decls {
			general, ok := declaration.(*ast.GenDecl)
			if !ok || general.Tok != token.TYPE {
				continue
			}
			for _, specification := range general.Specs {
				typeSpec, ok := specification.(*ast.TypeSpec)
				if ok && typeSpec.Name.Name == typeName {
					return true
				}
			}
		}
	}
	return false
}

func v16ReadProductionGo(t *testing.T, directory string) string {
	t.Helper()
	var source strings.Builder
	for _, file := range v16ProductionGoFiles(t, directory, false) {
		content, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		source.Write(content)
		source.WriteByte('\n')
	}
	return source.String()
}

func v16ReadProductionGoOptional(t *testing.T, directory string) string {
	t.Helper()
	if _, err := os.Stat(directory); err != nil {
		if os.IsNotExist(err) {
			return ""
		}
		t.Fatalf("stat %s: %v", directory, err)
	}
	return v16ReadProductionGo(t, directory)
}

func v16ReadGoTests(t *testing.T, directories ...string) string {
	t.Helper()
	var source strings.Builder
	for _, directory := range directories {
		if _, err := os.Stat(directory); err != nil {
			if os.IsNotExist(err) {
				t.Errorf("V16_RED test package missing: %s", directory)
				continue
			}
			t.Fatalf("stat test package %s: %v", directory, err)
		}
		for _, file := range v16ProductionGoFiles(t, directory, true) {
			content, err := os.ReadFile(file)
			if err != nil {
				t.Fatalf("read %s: %v", file, err)
			}
			source.Write(content)
			source.WriteByte('\n')
		}
	}
	return source.String()
}

func v16ProductionGoFiles(t *testing.T, directory string, tests bool) []string {
	t.Helper()
	entries, err := os.ReadDir(directory)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		t.Fatalf("read %s: %v", directory, err)
	}
	var files []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") ||
			(strings.HasSuffix(entry.Name(), "_test.go") != tests) {
			continue
		}
		files = append(files, filepath.Join(directory, entry.Name()))
	}
	sort.Strings(files)
	return files
}
