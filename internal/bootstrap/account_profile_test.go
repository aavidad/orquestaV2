//go:build linux

package bootstrap

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"sync/atomic"
	"testing"

	"orquesta/internal/adapters/agent/codex"
	localruntime "orquesta/internal/adapters/system/local"
	"orquesta/internal/application"
	"orquesta/internal/config"
)

func TestProductionAgentFactoryWiresExclusiveAccountProfile(t *testing.T) {
	base := secureBootstrapAccountBase(t)
	configPath := writeTestConfig(t, base)
	accountRoot := filepath.Join(base, "accounts")
	profilePath := filepath.Join(accountRoot, "Codex1")
	if err := os.MkdirAll(profilePath, 0o700); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{accountRoot, profilePath} {
		if err := os.Chmod(path, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(profilePath, "auth.json"), []byte(`{"tokens":"bootstrap"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(base)
	replaceTestConfigValue(t, configPath,
		"[runtime.codex]\n",
		"[runtime.codex]\naccount_home_root = \"./accounts\"\naccount_profile = \"Codex1\"\n",
	)
	replaceTestConfigValue(t, configPath, "max_concurrent_executions = 4", "max_concurrent_executions = 1")
	snapshot, err := loadConfigSnapshot(context.Background(), configPath)
	if err != nil {
		t.Fatalf("loadConfigSnapshot() error = %v", err)
	}
	first, err := productionAgentFactory(snapshot, localruntime.Clock{})
	if err != nil {
		t.Fatalf("productionAgentFactory(first) error = %v", err)
	}
	if _, err := productionAgentFactory(snapshot, localruntime.Clock{}); codex.ErrorCode(err) != codex.CodeAccountProfileUnavailable {
		t.Fatalf("productionAgentFactory(second) error=%v code=%q", err, codex.ErrorCode(err))
	}
	if err := first.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown(first) error = %v", err)
	}
	third, err := productionAgentFactory(snapshot, localruntime.Clock{})
	if err != nil {
		t.Fatalf("productionAgentFactory(after release) error = %v", err)
	}
	if err := third.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown(third) error = %v", err)
	}
}

func TestProductionAgentFactoryWiresOnePoolForMultipleExclusiveProfiles(t *testing.T) {
	base := secureBootstrapAccountBase(t)
	configPath := writeTestConfig(t, base)
	accountRoot := filepath.Join(base, "accounts")
	if err := os.Mkdir(accountRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	for _, profile := range []string{"Codex1", "Codex2"} {
		profilePath := filepath.Join(accountRoot, profile)
		if err := os.Mkdir(profilePath, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(
			filepath.Join(profilePath, "auth.json"),
			[]byte(`{"tokens":"bootstrap-`+profile+`"}`),
			0o600,
		); err != nil {
			t.Fatal(err)
		}
	}
	replaceTestConfigValue(t, configPath,
		"[runtime.codex]\n",
		"[runtime.codex]\naccount_home_root = "+strconv.Quote(accountRoot)+"\n"+
			"account_profiles = [\"Codex1\", \"Codex2\"]\n",
	)
	replaceTestConfigValue(t, configPath, "max_concurrent_executions = 4", "max_concurrent_executions = 2")
	snapshot, err := loadConfigSnapshot(context.Background(), configPath)
	if err != nil {
		t.Fatalf("loadConfigSnapshot() error = %v", err)
	}

	first, err := productionAgentFactory(snapshot, localruntime.Clock{})
	if err != nil {
		t.Fatalf("productionAgentFactory(first) error = %v", err)
	}
	if _, ok := first.(*codex.Pool); !ok {
		t.Fatalf("productionAgentFactory(first) type = %T, want *codex.Pool", first)
	}
	if _, ok := first.(application.AgentController); !ok {
		t.Fatalf("pool does not expose exact execution controls: %T", first)
	}
	if _, ok := first.(workspaceResolverBinder); !ok {
		t.Fatalf("pool does not expose workspace binding: %T", first)
	}
	if _, ok := first.(runtimeScopeBinder); !ok {
		t.Fatalf("pool does not expose runtime-scope binding: %T", first)
	}
	if _, err := productionAgentFactory(snapshot, localruntime.Clock{}); codex.ErrorCode(err) != codex.CodeAccountProfileUnavailable {
		t.Fatalf("productionAgentFactory(second) error=%v code=%q", err, codex.ErrorCode(err))
	}
	profileWorkRoots, err := os.ReadDir(filepath.Join(base, "work", "profiles"))
	if err != nil {
		t.Fatalf("read pool work roots: %v", err)
	}
	if len(profileWorkRoots) != 2 {
		t.Fatalf("pool work roots = %d, want 2", len(profileWorkRoots))
	}
	for _, entry := range profileWorkRoots {
		if !entry.IsDir() || entry.Name() == "Codex1" || entry.Name() == "Codex2" {
			t.Fatalf("pool exposed a raw profile identity in work root: %q", entry.Name())
		}
	}
	if err := first.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown(first) error = %v", err)
	}

	reopened, err := productionAgentFactory(snapshot, localruntime.Clock{})
	if err != nil {
		t.Fatalf("productionAgentFactory(after release) error = %v", err)
	}
	if err := reopened.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown(reopened) error = %v", err)
	}
}

func TestBuildRejectsCanonicalAccountHomeRuntimeOverlapBeforeFactory(t *testing.T) {
	root := t.TempDir()
	configPath := writeTestConfig(t, root)
	workRoot := filepath.Join(root, "work")
	if err := os.Mkdir(workRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	accountAlias := filepath.Join(root, "account-alias")
	if err := os.Symlink(workRoot, accountAlias); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}
	replaceTestConfigValue(t, configPath,
		"[runtime.codex]\n",
		"[runtime.codex]\naccount_home_root = "+strconv.Quote(accountAlias)+"\naccount_profile = \"Codex1\"\n",
	)
	replaceTestConfigValue(t, configPath, "max_concurrent_executions = 4", "max_concurrent_executions = 1")
	var calls atomic.Int64
	runtime, err := Build(context.Background(), Options{
		ConfigPath: configPath,
		AgentFactory: func(config.Snapshot, application.Clock) (AgentAdapter, error) {
			calls.Add(1)
			return nil, errors.New("unexpected factory call")
		},
	})
	if runtime != nil || err == nil || err.Error() != "bootstrap.runtime_paths_overlap" {
		t.Fatalf("Build() runtime=%v error=%v", runtime, err)
	}
	if calls.Load() != 0 {
		t.Fatalf("agent factory calls = %d", calls.Load())
	}
}

func secureBootstrapAccountBase(t *testing.T) string {
	t.Helper()
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	base, err := os.MkdirTemp(home, ".orquesta-bootstrap-account-test-")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(base, 0o700); err != nil {
		_ = os.RemoveAll(base)
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(base); err != nil {
			t.Errorf("RemoveAll(%s) error = %v", base, err)
		}
	})
	return base
}
