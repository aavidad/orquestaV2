package bootstrap

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/goal"
)

func TestCodexProductionProcessReceivesPinnedGoEnvironment(t *testing.T) {
	root := secureCodexGoFixtureBase(t)
	environmentPath := filepath.Join(root, "codex-environment")
	argumentsPath := filepath.Join(root, "codex-arguments")
	helperPath := filepath.Join(root, "codex-helper.sh")
	writeCodexGoEnvironmentHelper(t, helperPath, environmentPath, argumentsPath)
	toolchainRoot := writeFakeCodexGoToolchain(
		t, filepath.Join(root, "toolchain"), filepath.Join(root, "go-was-executed"),
	)
	configPath := writeTestConfig(t, root)
	replaceTestConfigValue(t, configPath, "[runtime.codex]\ntimeout = \"1s\"",
		"[runtime.codex]\n"+
			"command = "+strconv.Quote(helperPath)+"\n"+
			"go_toolchain_root = "+strconv.Quote(toolchainRoot)+"\n"+
			"env_allowlist = [\"PATH\", \"HOME\", \"CODEX_HOME\", \"GOENV\", \"GOTOOLCHAIN\", \"GOROOT\", \"GOCACHE\", \"GOMODCACHE\", \"GOPATH\", \"GOTMPDIR\", \"TMPDIR\", \"GOPROXY\"]\n"+
			"timeout = \"5s\"",
	)
	configureTestCodexRuntimeCgroup(t, configPath)

	t.Setenv("PATH", "/poisoned/bin")
	t.Setenv("GOENV", "/poisoned/go/env")
	t.Setenv("GOTOOLCHAIN", "auto")
	t.Setenv("GOROOT", "/poisoned/go")
	t.Setenv("GOCACHE", "/poisoned/build")
	t.Setenv("GOMODCACHE", "/poisoned/modules")
	t.Setenv("GOPATH", "/poisoned/gopath")
	t.Setenv("GOTMPDIR", "/poisoned/go-tmp")
	t.Setenv("TMPDIR", "/poisoned/tmp")
	t.Setenv("GOPROXY", "https://proxy.example.test")

	runtime, err := Build(context.Background(), Options{
		ConfigPath:                    configPath,
		Version:                       "bug460-codex-go-environment",
		codexGoToolchainTrustForTests: trustCodexGoFixtureOwner,
	})
	if err != nil {
		t.Fatalf("Build(production Codex) error = %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := runtime.Shutdown(ctx); err != nil {
			t.Errorf("Shutdown() error = %v", err)
		}
	})
	if err := runtime.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	submitted, err := runtime.Orchestrator().Submit(
		context.Background(),
		testRuntimeAccess(t, runtime),
		applicationSubmitForCodexGoEnvironment(),
	)
	if err != nil {
		t.Fatalf("Submit() error = %v", err)
	}
	terminal := waitTerminalGoal(t, runtime, submitted.Record.Goal.Ref())
	if terminal.Goal.State() != goal.GoalStateSucceeded {
		t.Fatalf("Goal state = %s", terminal.Goal.State())
	}

	got := readCodexGoEnvironmentCapture(t, environmentPath)
	cacheRoot := filepath.Join(root, "cache", "codex-go")
	want := map[string]string{
		"GOENV":       "off",
		"GOTOOLCHAIN": "local",
		"GOROOT":      toolchainRoot,
		"GOCACHE":     filepath.Join(cacheRoot, "build"),
		"GOMODCACHE":  filepath.Join(cacheRoot, "modules"),
		"GOPATH":      filepath.Join(cacheRoot, "gopath"),
		"GOTMPDIR":    filepath.Join(cacheRoot, "go-tmp"),
		"TMPDIR":      filepath.Join(cacheRoot, "tmp"),
		"GOPROXY":     "https://proxy.example.test",
		"PATH":        filepath.Join(toolchainRoot, "bin") + string(filepath.ListSeparator) + "/poisoned/bin",
	}
	for name, value := range want {
		if got[name] != value {
			t.Fatalf("physical %s = %q, want %q; environment=%+v", name, got[name], value, got)
		}
	}
	requireCodexGoShellPolicy(t, argumentsPath)
}

func TestCodexGoToolchainOwnerPreflightFailsBeforeCodexInvocation(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows has no Unix UID/GID ownership")
	}
	root := secureCodexGoFixtureBase(t)
	invocationPath := filepath.Join(root, "codex-invoked")
	helperPath := filepath.Join(root, "codex-helper.sh")
	if err := os.WriteFile(helperPath, []byte(
		"#!/bin/sh\nprintf invoked > "+strconv.Quote(invocationPath)+"\n",
	), 0o700); err != nil {
		t.Fatal(err)
	}
	toolchainRoot := writeFakeCodexGoToolchain(
		t,
		filepath.Join(root, "user-owned-toolchain"),
		filepath.Join(root, "go-was-executed"),
	)
	info, statErr := os.Lstat(toolchainRoot)
	if statErr != nil {
		t.Fatal(statErr)
	}
	if codexGoToolchainOwnerTrusted(info) {
		t.Skip("test process created a root-owned fixture")
	}
	configPath := writeTestConfig(t, root)
	replaceTestConfigValue(t, configPath, "[runtime.codex]\ntimeout = \"1s\"",
		"[runtime.codex]\n"+
			"command = "+strconv.Quote(helperPath)+"\n"+
			"go_toolchain_root = "+strconv.Quote(toolchainRoot)+"\n"+
			"timeout = \"5s\"",
	)
	orquestaRuntime, err := Build(context.Background(), Options{ConfigPath: configPath})
	if orquestaRuntime != nil || err == nil || err.Error() != codexGoToolchainInvalid {
		t.Fatalf("user-owned toolchain Build() = runtime:%v error:%v", orquestaRuntime, err)
	}
	if _, invocationErr := os.Stat(invocationPath); !errors.Is(invocationErr, os.ErrNotExist) {
		t.Fatalf("untrusted owner reached Codex process: %v", invocationErr)
	}
	if _, cacheErr := os.Stat(filepath.Join(root, "cache", "codex-go")); !errors.Is(cacheErr, os.ErrNotExist) {
		t.Fatalf("untrusted owner created Go cache before rejection: %v", cacheErr)
	}
}

func TestCodexGoToolchainPreflightFailsBeforeCodexInvocation(t *testing.T) {
	root := t.TempDir()
	invocationPath := filepath.Join(root, "codex-invoked")
	helperPath := filepath.Join(root, "codex-helper.sh")
	if err := os.WriteFile(helperPath, []byte(
		"#!/bin/sh\nprintf invoked > "+strconv.Quote(invocationPath)+"\n",
	), 0o700); err != nil {
		t.Fatal(err)
	}
	configPath := writeTestConfig(t, root)
	replaceTestConfigValue(t, configPath, "[runtime.codex]\ntimeout = \"1s\"",
		"[runtime.codex]\n"+
			"command = "+strconv.Quote(helperPath)+"\n"+
			"go_toolchain_root = "+strconv.Quote(filepath.Join(root, "missing-toolchain"))+"\n"+
			"timeout = \"5s\"",
	)
	runtime, err := Build(context.Background(), Options{ConfigPath: configPath})
	if runtime != nil || err == nil || err.Error() != codexGoToolchainInvalid {
		t.Fatalf("invalid toolchain Build() = runtime:%v error:%v", runtime, err)
	}
	if _, statErr := os.Stat(invocationPath); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("invalid toolchain reached Codex process: %v", statErr)
	}
	if _, statErr := os.Stat(filepath.Join(root, "cache", "codex-go")); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("invalid toolchain created Go cache before rejection: %v", statErr)
	}
}

func applicationSubmitForCodexGoEnvironment() application.SubmitRequest {
	return application.SubmitRequest{
		RequestRef: "request:bug460-codex-go-environment",
		Statement:  "capture exact physical Go environment",
		Confirm:    true,
	}
}

func writeCodexGoEnvironmentHelper(t *testing.T, helperPath, environmentPath, argumentsPath string) {
	t.Helper()
	helper := "#!/bin/sh\n" +
		"set -eu\n" +
		": > " + strconv.Quote(argumentsPath) + "\n" +
		"for argument in \"$@\"; do printf '%s\\n' \"$argument\" >> " + strconv.Quote(argumentsPath) + "; done\n" +
		"{\n" +
		"  printf 'PATH=%s\\n' \"$PATH\"\n" +
		"  printf 'GOENV=%s\\n' \"$GOENV\"\n" +
		"  printf 'GOTOOLCHAIN=%s\\n' \"$GOTOOLCHAIN\"\n" +
		"  printf 'GOROOT=%s\\n' \"$GOROOT\"\n" +
		"  printf 'GOCACHE=%s\\n' \"$GOCACHE\"\n" +
		"  printf 'GOMODCACHE=%s\\n' \"$GOMODCACHE\"\n" +
		"  printf 'GOPATH=%s\\n' \"$GOPATH\"\n" +
		"  printf 'GOTMPDIR=%s\\n' \"$GOTMPDIR\"\n" +
		"  printf 'TMPDIR=%s\\n' \"$TMPDIR\"\n" +
		"  printf 'GOPROXY=%s\\n' \"$GOPROXY\"\n" +
		"} > " + strconv.Quote(environmentPath) + "\n" +
		"output=\n" +
		"while [ \"$#\" -gt 0 ]; do\n" +
		"  if [ \"$1\" = \"--output-last-message\" ]; then shift; output=$1; fi\n" +
		"  shift\n" +
		"done\n" +
		"prompt=$(/bin/cat)\n" +
		"test -n \"$prompt\" && test -n \"$output\"\n" +
		"printf '%s\\n' '{\"artifact\":\"artifact:bug460-go-environment\"}' > \"$output\"\n"
	if err := os.WriteFile(helperPath, []byte(helper), 0o700); err != nil {
		t.Fatal(err)
	}
}

func readCodexGoEnvironmentCapture(t *testing.T, path string) map[string]string {
	t.Helper()
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	values := make(map[string]string)
	for _, line := range strings.Split(strings.TrimSpace(string(payload)), "\n") {
		name, value, found := strings.Cut(line, "=")
		if !found || name == "" {
			t.Fatalf("invalid environment capture line %q", line)
		}
		values[name] = value
	}
	return values
}

func requireCodexGoShellPolicy(t *testing.T, path string) {
	t.Helper()
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{
		"GOENV", "GOTOOLCHAIN", "GOROOT", "GOCACHE", "GOMODCACHE", "GOPATH", "GOTMPDIR", "TMPDIR",
	} {
		if !strings.Contains(string(payload), strconv.Quote(name)) {
			t.Fatalf("Codex shell include_only omitted %s: %s", name, payload)
		}
	}
}
