package bootstrap

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestPrepareCodexGoEnvironmentPinsConfiguredToolchainAndPrivateCaches(t *testing.T) {
	root := secureCodexGoFixtureBase(t)
	marker := filepath.Join(root, "go-was-executed")
	toolchainRoot := writeFakeCodexGoToolchain(t, filepath.Join(root, "toolchain"), marker)
	cacheRoot := filepath.Join(root, "cache")
	environment, err := prepareCodexGoEnvironmentWithTrust(map[string]string{
		"PATH":        "/usr/bin:/bin",
		"CUSTOM":      "preserved",
		"GOENV":       "poisoned",
		"GOTOOLCHAIN": "auto",
		"GOROOT":      "/poisoned/go",
		"GOCACHE":     "/poisoned/build",
		"GOMODCACHE":  "/poisoned/modules",
		"GOPATH":      "/poisoned/gopath",
		"GOTMPDIR":    "/poisoned/go-tmp",
		"TMPDIR":      "/poisoned/tmp",
	}, cacheRoot, toolchainRoot, trustCodexGoFixtureOwner)
	if err != nil {
		t.Fatalf("prepareCodexGoEnvironment() error = %v", err)
	}
	want := map[string]string{
		"GOENV":       "off",
		"GOTOOLCHAIN": "local",
		"GOROOT":      toolchainRoot,
		"GOCACHE":     filepath.Join(cacheRoot, "build"),
		"GOMODCACHE":  filepath.Join(cacheRoot, "modules"),
		"GOPATH":      filepath.Join(cacheRoot, "gopath"),
		"GOTMPDIR":    filepath.Join(cacheRoot, "go-tmp"),
		"TMPDIR":      filepath.Join(cacheRoot, "tmp"),
		"CUSTOM":      "preserved",
	}
	for name, value := range want {
		if environment[name] != value {
			t.Fatalf("%s = %q, want %q", name, environment[name], value)
		}
	}
	wantPath := filepath.Join(toolchainRoot, "bin") + string(filepath.ListSeparator) + "/usr/bin:/bin"
	if environment["PATH"] != wantPath {
		t.Fatalf("PATH = %q, want %q", environment["PATH"], wantPath)
	}
	for _, path := range append([]string{cacheRoot},
		environment["GOCACHE"], environment["GOMODCACHE"], environment["GOPATH"],
		environment["GOTMPDIR"], environment["TMPDIR"],
	) {
		info, statErr := os.Lstat(path)
		if statErr != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			t.Fatalf("private cache path %q: info=%v error=%v", path, info, statErr)
		}
		if runtime.GOOS != "windows" && info.Mode().Perm() != 0o700 {
			t.Fatalf("private cache path %q mode = %o", path, info.Mode().Perm())
		}
	}
	if _, statErr := os.Stat(marker); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("toolchain validation executed bin/go: %v", statErr)
	}
}

func TestPrepareCodexGoEnvironmentWithoutToolchainDoesNotProbeGo(t *testing.T) {
	root := t.TempDir()
	marker := filepath.Join(root, "go-was-executed")
	fakeBin := filepath.Join(root, "bin")
	if err := os.Mkdir(fakeBin, 0o700); err != nil {
		t.Fatal(err)
	}
	writeExecutableMarker(t, filepath.Join(fakeBin, goExecutableName()), marker)
	cacheRoot := filepath.Join(root, "cache")
	environment, err := prepareCodexGoEnvironment(map[string]string{
		"PATH":   fakeBin,
		"GOROOT": "/must-not-survive",
	}, cacheRoot, "")
	if err != nil {
		t.Fatalf("prepareCodexGoEnvironment() error = %v", err)
	}
	if environment["PATH"] != fakeBin ||
		environment["GOENV"] != "off" ||
		environment["GOTOOLCHAIN"] != "local" {
		t.Fatalf("portable environment = %+v", environment)
	}
	if _, found := environment["GOROOT"]; found {
		t.Fatalf("empty capability retained inherited GOROOT: %+v", environment)
	}
	if _, statErr := os.Stat(marker); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("empty capability probed Go: %v", statErr)
	}
}

func TestPrepareCodexGoEnvironmentRejectsUnsafeCacheBeforeCodex(t *testing.T) {
	root := t.TempDir()
	permissive := filepath.Join(root, "permissive-cache")
	if err := os.Mkdir(permissive, 0o700); err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" {
		if err := os.Chmod(permissive, 0o755); err != nil {
			t.Fatal(err)
		}
		_, err := prepareCodexGoEnvironment(nil, permissive, "")
		if err == nil || err.Error() != codexGoCacheInvalid {
			t.Fatalf("permissive cache error = %v", err)
		}
	}

	target := filepath.Join(root, "target")
	if err := os.Mkdir(target, 0o700); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(root, "cache-alias")
	if err := os.Symlink(target, alias); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}
	_, err := prepareCodexGoEnvironment(nil, alias, "")
	if err == nil || err.Error() != codexGoCacheInvalid {
		t.Fatalf("symlink cache error = %v", err)
	}
}

func TestPrepareCodexGoEnvironmentRejectsInvalidConfiguredToolchainBeforeCacheEffect(t *testing.T) {
	root := secureCodexGoFixtureBase(t)
	missingGoRoot := filepath.Join(root, "missing-go")
	if err := os.MkdirAll(filepath.Join(missingGoRoot, "bin"), 0o700); err != nil {
		t.Fatal(err)
	}
	nonExecutableRoot := filepath.Join(root, "non-executable")
	if err := os.MkdirAll(filepath.Join(nonExecutableRoot, "bin"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nonExecutableRoot, "bin", goExecutableName()), []byte("not executable"), 0o600); err != nil {
		t.Fatal(err)
	}
	targetRoot := writeFakeCodexGoToolchain(t, filepath.Join(root, "target-toolchain"), filepath.Join(root, "unused"))
	aliasRoot := filepath.Join(root, "toolchain-alias")
	if err := os.Symlink(targetRoot, aliasRoot); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}
	symlinkedTreeRoot := writeFakeCodexGoToolchain(
		t, filepath.Join(root, "symlinked-tree"), filepath.Join(root, "unused-symlinked"),
	)
	if err := os.Symlink(
		filepath.Join(symlinkedTreeRoot, "bin", goExecutableName()),
		filepath.Join(symlinkedTreeRoot, "linked-tool"),
	); err != nil {
		t.Skipf("tree symlink unsupported: %v", err)
	}
	writableTreeRoot := writeFakeCodexGoToolchain(
		t, filepath.Join(root, "writable-tree"), filepath.Join(root, "unused-writable"),
	)
	writableTool := filepath.Join(writableTreeRoot, "pkg", "tool", "compile")
	if err := os.MkdirAll(filepath.Dir(writableTool), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(writableTool, []byte("tool"), 0o700); err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" {
		if err := os.Chmod(writableTool, 0o720); err != nil {
			t.Fatal(err)
		}
	}

	tests := []struct {
		name string
		root string
	}{
		{name: "relative", root: "relative/toolchain"},
		{name: "missing", root: filepath.Join(root, "absent")},
		{name: "missing bin go", root: missingGoRoot},
		{name: "non executable bin go", root: nonExecutableRoot},
		{name: "symlink", root: aliasRoot},
		{name: "symlinked tree", root: symlinkedTreeRoot},
	}
	if runtime.GOOS != "windows" {
		tests = append(tests, struct {
			name string
			root string
		}{name: "group writable descendant", root: writableTreeRoot})
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cacheRoot := filepath.Join(root, "cache-"+test.name)
			_, err := prepareCodexGoEnvironmentWithTrust(
				map[string]string{"PATH": "/usr/bin"},
				cacheRoot,
				test.root,
				trustCodexGoFixtureOwner,
			)
			if err == nil || err.Error() != codexGoToolchainInvalid {
				t.Fatalf("invalid toolchain error = %v", err)
			}
			if _, statErr := os.Stat(cacheRoot); !errors.Is(statErr, os.ErrNotExist) {
				t.Fatalf("invalid toolchain created cache before rejection: %v", statErr)
			}
		})
	}
}

func TestPrepareCodexGoEnvironmentRejectsNonRootOwnedToolchain(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows has no Unix UID/GID ownership")
	}
	root := secureCodexGoFixtureBase(t)
	toolchainRoot := writeFakeCodexGoToolchain(
		t,
		filepath.Join(root, "user-owned-toolchain"),
		filepath.Join(root, "must-not-execute"),
	)
	info, statErr := os.Lstat(toolchainRoot)
	if statErr != nil {
		t.Fatal(statErr)
	}
	if codexGoToolchainOwnerTrusted(info) {
		t.Skip("test process created a root-owned fixture")
	}
	cacheRoot := filepath.Join(root, "cache")
	_, err := prepareCodexGoEnvironment(
		map[string]string{"PATH": "/usr/bin"},
		cacheRoot,
		toolchainRoot,
	)
	if err == nil || err.Error() != codexGoToolchainInvalid {
		t.Fatalf("user-owned toolchain error = %v", err)
	}
	if _, statErr := os.Stat(cacheRoot); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("untrusted owner created cache before rejection: %v", statErr)
	}
}

func TestPrepareCodexGoEnvironmentEnforcesInjectedOwnershipPolicy(t *testing.T) {
	root := secureCodexGoFixtureBase(t)
	toolchainRoot := writeFakeCodexGoToolchain(
		t,
		filepath.Join(root, "synthetically-untrusted-toolchain"),
		filepath.Join(root, "must-not-execute"),
	)
	cacheRoot := filepath.Join(root, "cache")
	_, err := prepareCodexGoEnvironmentWithTrust(
		map[string]string{"PATH": "/usr/bin"},
		cacheRoot,
		toolchainRoot,
		func(os.FileInfo) bool { return false },
	)
	if err == nil || err.Error() != codexGoToolchainInvalid {
		t.Fatalf("synthetically untrusted toolchain error = %v", err)
	}
	if _, statErr := os.Stat(cacheRoot); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("synthetically untrusted owner created cache before rejection: %v", statErr)
	}
}

func writeFakeCodexGoToolchain(t *testing.T, root, marker string) string {
	t.Helper()
	bin := filepath.Join(root, "bin")
	if err := os.MkdirAll(bin, 0o700); err != nil {
		t.Fatal(err)
	}
	writeExecutableMarker(t, filepath.Join(bin, goExecutableName()), marker)
	return root
}

func writeExecutableMarker(t *testing.T, path, marker string) {
	t.Helper()
	content := []byte("#!/bin/sh\nprintf executed > " + marker + "\n")
	if runtime.GOOS == "windows" {
		content = []byte("@echo executed > " + marker + "\r\n")
	}
	if err := os.WriteFile(path, content, 0o700); err != nil {
		t.Fatal(err)
	}
}

func trustCodexGoFixtureOwner(os.FileInfo) bool {
	return true
}

func secureCodexGoFixtureBase(t *testing.T) string {
	t.Helper()
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	root, err := os.MkdirTemp(home, ".orquesta-codex-go-test-")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(root, 0o700); err != nil {
		_ = os.RemoveAll(root)
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(root); err != nil {
			t.Errorf("RemoveAll(%s) error = %v", root, err)
		}
	})
	return root
}
