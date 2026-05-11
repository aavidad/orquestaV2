package orquestaappcodexstack

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestCodexStackRealSmokeVerifyProjectFileV0AceptaDirectorioConContenido(t *testing.T) {
	projectDir := t.TempDir()
	writeSetDir := filepath.Join(projectDir, "internal", "agenda", "domain")
	if err := os.MkdirAll(writeSetDir, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	data := strings.Repeat("contenido de dominio hexagonal con reglas pequenas\n", 5)
	if err := os.WriteFile(filepath.Join(writeSetDir, "event.go"), []byte(data), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	codexStackRealSmokeVerifyProjectFileV0(t, projectDir, "internal/agenda/domain")
}

func TestCodexStackRealSmokeVerifyProjectFileV0AceptaGoModPequeno(t *testing.T) {
	projectDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectDir, "go.mod"), []byte("module app\n\ngo 1.22\n"), 0o600); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	codexStackRealSmokeVerifyProjectFileV0(t, projectDir, "go.mod")
}

func TestCodexStackRealSmokeVerifyProjectFileV0AceptaGlobStar(t *testing.T) {
	projectDir := t.TempDir()
	writeSmokeVerifierFileV0(t, projectDir, "internal/domain/event_test.go", "package domain\n")

	codexStackRealSmokeVerifyProjectFileV0(t, projectDir, "internal/**/*_test.go")
}

func codexStackRealSmokeVerifyProjectFileV0(t *testing.T, projectDir string, relPath string) {
	t.Helper()
	clean := filepath.Clean(strings.TrimSpace(relPath))
	if clean == "." || filepath.IsAbs(clean) || strings.HasPrefix(clean, "..") {
		t.Fatalf("path invalido en write-set: %q", relPath)
	}
	if codexStackRealSmokeHasGlobV0(clean) {
		codexStackRealSmokeVerifyProjectGlobV0(t, projectDir, filepath.ToSlash(clean))
		return
	}
	fullPath := filepath.Join(projectDir, clean)
	codexStackRealSmokeVerifyConcreteProjectPathV0(t, projectDir, fullPath)
}

func codexStackRealSmokeVerifyConcreteProjectPathV0(
	t *testing.T,
	projectDir string,
	fullPath string,
) {
	t.Helper()
	rel, err := filepath.Rel(projectDir, fullPath)
	if err != nil {
		t.Fatalf("path fuera de proyecto %s: %v", fullPath, err)
	}
	clean := filepath.ToSlash(filepath.Clean(rel))
	info, err := os.Stat(fullPath)
	if err != nil {
		t.Fatalf("doc missing %s: %v", clean, err)
	}
	if info.IsDir() {
		codexStackRealSmokeVerifyProjectDirV0(t, fullPath, clean)
		return
	}
	data, err := os.ReadFile(fullPath)
	if err != nil {
		t.Fatalf("doc missing %s: %v", clean, err)
	}
	if codexStackRealSmokeNeedsTextMinimumV0(clean) && len(strings.TrimSpace(string(data))) < 160 {
		t.Fatalf("doc demasiado pequeno %s", clean)
	}
	if !codexStackRealSmokeNeedsTextMinimumV0(clean) && len(strings.TrimSpace(string(data))) == 0 {
		t.Fatalf("artifact vacio %s", clean)
	}
}

func codexStackRealSmokeVerifyProjectGlobV0(t *testing.T, projectDir string, pattern string) {
	t.Helper()
	matches := codexStackRealSmokeGlobMatchesV0(t, projectDir, pattern)
	if len(matches) == 0 {
		t.Fatalf("glob sin artifacts %s", pattern)
	}
	for _, match := range matches {
		codexStackRealSmokeVerifyConcreteProjectPathV0(t, projectDir, match)
	}
}

func codexStackRealSmokeGlobMatchesV0(
	t *testing.T,
	projectDir string,
	pattern string,
) []string {
	t.Helper()
	re := codexStackRealSmokeGlobRegexpV0(t, pattern)
	matches := []string{}
	err := filepath.Walk(projectDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil || info == nil {
			return walkErr
		}
		if info.IsDir() && info.Name() == ".orquesta-codex-runtime" {
			return filepath.SkipDir
		}
		rel, err := filepath.Rel(projectDir, path)
		if err != nil || rel == "." {
			return err
		}
		if re.MatchString(filepath.ToSlash(rel)) {
			matches = append(matches, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk glob %s: %v", pattern, err)
	}
	return matches
}

func codexStackRealSmokeGlobRegexpV0(t *testing.T, pattern string) *regexp.Regexp {
	t.Helper()
	var b strings.Builder
	b.WriteString("^")
	for i := 0; i < len(pattern); i++ {
		if strings.HasPrefix(pattern[i:], "**") {
			b.WriteString(".*")
			i++
			continue
		}
		switch pattern[i] {
		case '*':
			b.WriteString("[^/]*")
		case '?':
			b.WriteString("[^/]")
		default:
			b.WriteString(regexp.QuoteMeta(string(pattern[i])))
		}
	}
	b.WriteString("$")
	re, err := regexp.Compile(b.String())
	if err != nil {
		t.Fatalf("glob invalido %s: %v", pattern, err)
	}
	return re
}

func codexStackRealSmokeHasGlobV0(path string) bool {
	return strings.ContainsAny(path, "*?[")
}

func codexStackRealSmokeNeedsTextMinimumV0(path string) bool {
	return strings.HasPrefix(path, "docs/") ||
		strings.HasSuffix(path, ".md") ||
		strings.HasSuffix(path, ".html")
}

func codexStackRealSmokeVerifyProjectDirV0(t *testing.T, fullPath string, relPath string) {
	t.Helper()
	files := 0
	totalBytes := 0
	err := filepath.Walk(fullPath, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil || info == nil || info.IsDir() {
			return walkErr
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files++
		totalBytes += len(strings.TrimSpace(string(data)))
		return nil
	})
	if err != nil {
		t.Fatalf("dir invalido %s: %v", relPath, err)
	}
	if files == 0 || totalBytes < 160 {
		t.Fatalf("dir demasiado pequeno %s files=%d bytes=%d", relPath, files, totalBytes)
	}
}
