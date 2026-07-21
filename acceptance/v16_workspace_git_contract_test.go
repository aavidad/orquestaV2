package acceptance_test

import (
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"orquesta/internal/application"
)

func v16AssertOneWriterAndOutboundPorts(t *testing.T, applicationDirectory string, fixture v16Fixture) {
	t.Helper()
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
}

func v16AssertImmutableFacts(t *testing.T, applicationDirectory, portsDirectory string, fixture v16Fixture) {
	t.Helper()
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
}

func v16AssertPublicUseCases(t *testing.T, applicationDirectory string, fixture v16Fixture) {
	t.Helper()
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
}

func v16AssertConcreteAdapters(t *testing.T, repositoryRoot string) {
	t.Helper()
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
	if count := len(regexp.MustCompile(`"key"\s*:\s*"workspace\.local\.root"`).FindAll(registry, -1)); count != 1 {
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
}

func v16AssertBehaviorTests(t *testing.T, repositoryRoot, applicationDirectory, portsDirectory string,
	fixture v16Fixture,
) {
	t.Helper()
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
}
