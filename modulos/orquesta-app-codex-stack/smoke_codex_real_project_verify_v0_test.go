package orquestaappcodexstack

import (
	"context"
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

const codexStackRealSmokeMaxGoFileLinesV0 = 300

func TestCodexStackRealSmokeVerifyGoFileSizesV0AceptaFicheroManejable(t *testing.T) {
	projectDir := t.TempDir()
	target := filepath.Join(projectDir, "internal", "agenda", "domain")
	if err := os.MkdirAll(target, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(target, "event.go"), []byte("package domain\n"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	codexStackRealSmokeVerifyGoFileSizesV0(t, projectDir)
}

func TestCodexStackRealSmokeProjectTargetExistsV0AceptaWebAdminInternoComoWeb(t *testing.T) {
	projectDir := t.TempDir()
	writeSmokeVerifierFileV0(t, projectDir, "internal/webadmin/handler.go", "package webadmin\n")

	if !codexStackRealSmokeProjectTargetExistsV0(projectDir, "web") {
		t.Fatalf("web interno no reconocido como superficie web")
	}
}

func TestCodexStackRealSmokeVerifyGoAppCompilesV0RechazaImportsRelativos(t *testing.T) {
	projectDir := t.TempDir()
	writeGoModuleForSmokeVerifierV0(t, projectDir)
	writeSmokeVerifierFileV0(t, projectDir, "cmd/server/main.go", "package main\nimport _ \"../internal\"\nfunc main() {}\n")

	if err := codexStackRealSmokeCheckGoAppCompilesV0(projectDir); err == nil {
		t.Fatalf("expected relative import failure")
	}
}

func TestCodexStackRealSmokeVerifyGoAppCompilesV0RechazaSinModulo(t *testing.T) {
	projectDir := t.TempDir()
	writeSmokeVerifierFileV0(t, projectDir, "cmd/server/main.go", "package main\nfunc main() {}\n")

	if err := codexStackRealSmokeCheckGoAppCompilesV0(projectDir); err == nil {
		t.Fatalf("expected missing module failure")
	}
}

func codexStackRealSmokeVerifyGoFileSizesV0(t *testing.T, projectDir string) {
	t.Helper()
	err := filepath.Walk(projectDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil || info == nil || info.IsDir() || filepath.Ext(path) != ".go" {
			return walkErr
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		lines := strings.Count(string(data), "\n")
		if len(data) > 0 && data[len(data)-1] != '\n' {
			lines++
		}
		if lines > codexStackRealSmokeMaxGoFileLinesV0 {
			t.Fatalf("go file demasiado grande %s lines=%d", path, lines)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk go files: %v", err)
	}
}

func codexStackRealSmokeVerifyGoAppCompilesV0(t *testing.T, projectDir string) {
	t.Helper()
	if err := codexStackRealSmokeCheckGoAppCompilesV0(projectDir); err != nil {
		t.Fatal(err)
	}
}

func codexStackRealSmokeCheckGoAppCompilesV0(projectDir string) error {
	hasGoFiles, err := codexStackRealSmokeProjectHasGoFilesV0(projectDir)
	if err != nil {
		return err
	}
	if !hasGoFiles {
		return nil
	}
	if _, err := os.Stat(filepath.Join(projectDir, "go.mod")); err != nil {
		return fmt.Errorf("app Go sin go.mod: %w", err)
	}
	if err := codexStackRealSmokeVerifyGoEntrypointV0(projectDir); err != nil {
		return err
	}
	if err := codexStackRealSmokeVerifyNoGoRelativeImportsV0(projectDir); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "test", "./...")
	cmd.Dir = projectDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("go test ./... app generada: %w\n%s", err, codexStackRealSmokeTruncateV0(string(output)))
	}
	return nil
}

func codexStackRealSmokeProjectHasGoFilesV0(projectDir string) (bool, error) {
	hasGoFiles := false
	err := filepath.WalkDir(projectDir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil || entry == nil || entry.IsDir() {
			return walkErr
		}
		if codexStackRealSmokeInRuntimeDirV0(path) {
			return nil
		}
		if filepath.Ext(path) == ".go" {
			hasGoFiles = true
		}
		return nil
	})
	if err != nil {
		return false, fmt.Errorf("walk go project: %w", err)
	}
	return hasGoFiles, nil
}

func codexStackRealSmokeVerifyGoEntrypointV0(projectDir string) error {
	cmdDir := filepath.Join(projectDir, "cmd")
	found := false
	err := filepath.WalkDir(cmdDir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if entry == nil || entry.IsDir() || filepath.Ext(path) != ".go" {
			return nil
		}
		name, err := codexStackRealSmokeGoPackageNameV0(path)
		if err != nil {
			return err
		}
		if name == "main" {
			found = true
		}
		return nil
	})
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("app Go sin entrypoint package main bajo cmd/")
	}
	return nil
}

func codexStackRealSmokeVerifyNoGoRelativeImportsV0(projectDir string) error {
	return filepath.WalkDir(projectDir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil || entry == nil || entry.IsDir() || filepath.Ext(path) != ".go" {
			return walkErr
		}
		if codexStackRealSmokeInRuntimeDirV0(path) {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, spec := range file.Imports {
			value, err := strconv.Unquote(spec.Path.Value)
			if err != nil {
				return err
			}
			if codexStackRealSmokeGoImportIsRelativeV0(value) {
				return fmt.Errorf("import relativo prohibido %s en %s", value, path)
			}
		}
		return nil
	})
}

func codexStackRealSmokeGoPackageNameV0(path string) (string, error) {
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.PackageClauseOnly)
	if err != nil {
		return "", fmt.Errorf("parse package %s: %w", path, err)
	}
	return file.Name.Name, nil
}

func codexStackRealSmokeGoImportIsRelativeV0(value string) bool {
	return value == "." || value == ".." ||
		strings.HasPrefix(value, "./") ||
		strings.HasPrefix(value, "../") ||
		strings.Contains(value, "/../") ||
		strings.Contains(value, "/./")
}

func codexStackRealSmokeInRuntimeDirV0(path string) bool {
	return strings.Contains(filepath.ToSlash(path), "/.orquesta-codex-runtime/")
}

func writeGoModuleForSmokeVerifierV0(t *testing.T, projectDir string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(projectDir, "go.mod"), []byte("module agenda\n\ngo 1.22\n"), 0o600); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
}

func writeSmokeVerifierFileV0(t *testing.T, projectDir string, relPath string, data string) {
	t.Helper()
	fullPath := filepath.Join(projectDir, relPath)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(fullPath, []byte(data), 0o600); err != nil {
		t.Fatalf("write %s: %v", relPath, err)
	}
}
