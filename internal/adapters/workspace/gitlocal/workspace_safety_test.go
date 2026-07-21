package gitlocal

import (
	"context"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestReadGitControlRejectsSymlinkFIFODeviceAndHardlink(t *testing.T) {
	t.Run("symlink", func(t *testing.T) {
		assertUnsafeControlMutation(t, func(control string) error {
			source := control + ".source"
			if err := os.Rename(control, source); err != nil {
				return err
			}
			return os.Symlink(source, control)
		})
	})
	t.Run("fifo", func(t *testing.T) {
		assertUnsafeControlMutation(t, func(control string) error {
			if err := os.Rename(control, control+".source"); err != nil {
				return err
			}
			return syscall.Mkfifo(control, 0o600)
		})
	})
	t.Run("hardlink", func(t *testing.T) {
		assertUnsafeControlMutation(t, func(control string) error {
			source := control + ".source"
			if err := os.Rename(control, source); err != nil {
				return err
			}
			return os.Link(source, control)
		})
	})
	t.Run("device", func(t *testing.T) {
		if _, err := readSafeGitControlFile("/dev/null"); ErrorCodeOf(err) != CodeWorkspaceUnsafe {
			t.Fatalf("device control err=%v", err)
		}
	})
}

func assertUnsafeControlMutation(t *testing.T, mutate func(string) error) {
	t.Helper()
	adapter, request := testAdapterAndPrepare(t)
	if _, err := adapter.Prepare(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	control := filepath.Join(adapter.workspacePath(request.WorkspaceRef), ".git")
	if err := mutate(control); err != nil {
		t.Fatal(err)
	}
	if _, err := adapter.ResolveExecutionWorkspace(context.Background(), request.WorkspaceRef); ErrorCodeOf(err) != CodeWorkspaceUnsafe {
		t.Fatalf("unsafe control err=%v", err)
	}
}

func TestWorkspaceRejectsGitControlBoundToDifferentRepository(t *testing.T) {
	first, firstRequest := testAdapterAndPrepare(t)
	second, secondRequest := testAdapterAndPrepare(t)
	if _, err := first.Prepare(context.Background(), firstRequest); err != nil {
		t.Fatal(err)
	}
	if _, err := second.Prepare(context.Background(), secondRequest); err != nil {
		t.Fatal(err)
	}
	otherControl, err := os.ReadFile(filepath.Join(second.workspacePath(secondRequest.WorkspaceRef), ".git"))
	if err != nil {
		t.Fatal(err)
	}
	control := filepath.Join(first.workspacePath(firstRequest.WorkspaceRef), ".git")
	if err := os.WriteFile(control, otherControl, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := first.ResolveExecutionWorkspace(context.Background(), firstRequest.WorkspaceRef); ErrorCodeOf(err) != CodeWorkspaceUnsafe {
		t.Fatalf("foreign common-dir err=%v", err)
	}
}

func TestSafeGitDirectoryRejectsWritableMetadataAndAncestry(t *testing.T) {
	tests := []struct {
		name   string
		mode   os.FileMode
		mutate func(string, string) string
	}{
		{name: "group_writable_metadata", mode: 0o770, mutate: chmodMetadata},
		{name: "world_writable_metadata", mode: 0o702, mutate: chmodMetadata},
		{name: "sticky_world_writable_metadata", mode: 0o777 | os.ModeSticky, mutate: chmodMetadata},
		{name: "group_writable_ancestor", mode: 0o770, mutate: chmodMetadataAncestor},
		{name: "world_writable_ancestor", mode: 0o702, mutate: chmodMetadataAncestor},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			parent := filepath.Join(t.TempDir(), "metadata-parent")
			metadata := filepath.Join(parent, "worktree-metadata")
			if err := os.MkdirAll(metadata, 0o700); err != nil {
				t.Fatal(err)
			}
			path := test.mutate(parent, metadata)
			if err := os.Chmod(path, test.mode); err != nil {
				t.Fatal(err)
			}
			if _, err := safeGitDirectory(metadata); ErrorCodeOf(err) != CodeWorkspaceUnsafe {
				t.Fatalf("safeGitDirectory(%s) err=%v", metadata, err)
			}
		})
	}
}

func chmodMetadata(_ string, metadata string) string { return metadata }

func chmodMetadataAncestor(parent string, _ string) string { return parent }
