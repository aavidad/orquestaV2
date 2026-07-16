package orquesta_test

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const moderncSQLitePatchBaseCommit = "df695543b6486dae51f7eab40417543b48d71de7"

func TestModerncSQLiteRuntimeScopePatchIsReproducible(t *testing.T) {
	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for _, relative := range []string{"go.mod", "vendor/modules.txt"} {
		content, readErr := os.ReadFile(filepath.Join(root, relative))
		if readErr != nil {
			t.Fatal(readErr)
		}
		if !strings.Contains(string(content), "modernc.org/sqlite v1.53.0") {
			t.Fatalf("%s does not pin patched modernc.org/sqlite v1.53.0", relative)
		}
	}

	baseFiles := []string{
		"vendor/modernc.org/sqlite/conn.go",
		"vendor/modernc.org/sqlite/fcntl.go",
	}
	patchedFiles := append(append([]string(nil), baseFiles...),
		"vendor/modernc.org/sqlite/connector.go",
		"vendor/modernc.org/sqlite/fcntl_descriptor_unix.go",
		"vendor/modernc.org/sqlite/fcntl_descriptor_other.go",
	)
	temporary := t.TempDir()
	for _, relative := range baseFiles {
		content, showErr := exec.Command(
			"git", "-C", root, "show", moderncSQLitePatchBaseCommit+":"+relative,
		).Output()
		if showErr != nil {
			t.Fatalf("read pristine %s: %v", relative, showErr)
		}
		target := filepath.Join(temporary, filepath.FromSlash(relative))
		if mkdirErr := os.MkdirAll(filepath.Dir(target), 0o755); mkdirErr != nil {
			t.Fatal(mkdirErr)
		}
		if writeErr := os.WriteFile(target, content, 0o644); writeErr != nil {
			t.Fatal(writeErr)
		}
	}

	patchPath := filepath.Join(root, "third_party", "patches", "modernc_sqlite_v1.53.0_runtime_scope.patch")
	command := exec.Command("git", "apply", "--whitespace=nowarn", "-p1", patchPath)
	command.Dir = temporary
	if output, applyErr := command.CombinedOutput(); applyErr != nil {
		t.Fatalf("apply modernc SQLite patch: %v\n%s", applyErr, output)
	}
	for _, relative := range patchedFiles {
		want, readErr := os.ReadFile(filepath.Join(root, filepath.FromSlash(relative)))
		if readErr != nil {
			t.Fatal(readErr)
		}
		got, readErr := os.ReadFile(filepath.Join(temporary, filepath.FromSlash(relative)))
		if readErr != nil {
			t.Fatal(readErr)
		}
		if !bytes.Equal(got, want) {
			t.Errorf("patched %s differs from vendored source", relative)
		}
	}
	if t.Failed() {
		t.Log(fmt.Sprintf("reapply %s after regenerating vendor", filepath.ToSlash(patchPath)))
	}
}
