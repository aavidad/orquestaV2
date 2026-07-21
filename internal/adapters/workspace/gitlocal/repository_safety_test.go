package gitlocal

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGitWorkspaceRejectsUnsafeFilesystemAndGitControls(t *testing.T) {
	root := t.TempDir()
	if err := os.Chmod(root, 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := New(Config{Root: root, GitCommand: "/usr/bin/git", Locator: testLocator{}, Now: time.Now})
	if ErrorCodeOf(err) != CodeWorkspaceUnsafe {
		t.Fatalf("err=%v", err)
	}
}

func TestGitWorkspaceValidatesPrivateAncestorsAndAllowsStickyTempAncestor(t *testing.T) {
	unsafeParent, err := os.MkdirTemp("/tmp", "orquesta-gitlocal-unsafe-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(unsafeParent) })
	if err := os.Chmod(unsafeParent, 0o777); err != nil {
		t.Fatal(err)
	}
	if _, err := New(Config{Root: filepath.Join(unsafeParent, "private"), GitCommand: "/usr/bin/git", Locator: testLocator{}, Now: time.Now}); ErrorCodeOf(err) != CodeWorkspaceUnsafe {
		t.Fatalf("unsafe ancestor err=%v", err)
	}

	stickyParent, err := os.MkdirTemp("/tmp", "orquesta-gitlocal-sticky-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(stickyParent) })
	if err := os.Chmod(stickyParent, 0o777|os.ModeSticky); err != nil {
		t.Fatal(err)
	}
	if _, err := New(Config{Root: filepath.Join(stickyParent, "private"), GitCommand: "/usr/bin/git", Locator: testLocator{}, Now: time.Now}); err != nil {
		t.Fatalf("sticky ancestor rejected: %v", err)
	}

	base := t.TempDir()
	realParent := filepath.Join(base, "real-parent")
	if err := os.Mkdir(realParent, 0o700); err != nil {
		t.Fatal(err)
	}
	linkedParent := filepath.Join(base, "linked-parent")
	if err := os.Symlink(realParent, linkedParent); err != nil {
		t.Fatal(err)
	}
	if _, err := New(Config{Root: filepath.Join(linkedParent, "private"), GitCommand: "/usr/bin/git", Locator: testLocator{}, Now: time.Now}); ErrorCodeOf(err) != CodeWorkspaceUnsafe {
		t.Fatalf("symlink ancestor err=%v", err)
	}
}

func TestGitWorkspaceIsolatesHostConfigAndRejectsExecutableRepositoryControls(t *testing.T) {
	adapter, request := testAdapterAndPrepare(t)
	hostile := []byte("[filter \"evil\"]\n\tclean = /bin/false\n[core]\n\thooksPath = /host/hooks\n")
	if err := os.WriteFile(filepath.Join(adapter.root, ".gitconfig"), hostile, 0o600); err != nil {
		t.Fatal(err)
	}
	systemConfig := filepath.Join(t.TempDir(), "system.gitconfig")
	if err := os.WriteFile(systemConfig, hostile, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_CONFIG_SYSTEM", systemConfig)
	if _, err := adapter.Prepare(context.Background(), request); err != nil {
		t.Fatalf("isolated host config affected Git: %v", err)
	}

	hostileAdapter, hostileRequest := testAdapterAndPrepare(t)
	repository := hostileAdapter.loc.(testLocator).binding.Path
	gitTest(t, repository, "config", "filter.evil.clean", "/bin/false")
	if _, err := hostileAdapter.Prepare(context.Background(), hostileRequest); ErrorCodeOf(err) != CodeWorkspaceUnsafe {
		t.Fatalf("executable repo config err=%v", err)
	}
}

func TestGitWorkspaceRejectsRepositoryOverlappingPrivateRoot(t *testing.T) {
	repository := t.TempDir()
	gitTest(t, repository, "init", "-b", "main")
	gitTest(t, repository, "config", "user.name", "Test")
	gitTest(t, repository, "config", "user.email", "test@example.invalid")
	if err := os.WriteFile(filepath.Join(repository, "allowed.txt"), []byte("base"), 0o600); err != nil {
		t.Fatal(err)
	}
	gitTest(t, repository, "add", "allowed.txt")
	gitTest(t, repository, "commit", "-m", "base")
	if err := os.Chmod(repository, 0o700); err != nil {
		t.Fatal(err)
	}
	adapter, err := New(Config{
		Root: filepath.Join(repository, "var", "workspaces"), GitCommand: "/usr/bin/git",
		Locator: testLocator{binding: LocalRepositoryBinding{Path: repository, TargetRef: "refs/heads/main"}},
		Now:     time.Now,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, request := testAdapterAndPrepare(t)
	if _, err := adapter.Prepare(context.Background(), request); ErrorCodeOf(err) != CodeRepositoryInvalid {
		t.Fatalf("overlapping repository/workspace root err=%v", err)
	}
}
