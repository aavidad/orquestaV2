// Este fichero reúne utilidades de prueba sin añadir conducta al manifiesto.
package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func runManifestTest(t *testing.T, roots, excluded []string, output string) {
	t.Helper()
	if err := run(options{
		roots:        roots,
		excluded:     excluded,
		manifestPath: output,
		gitTimeout:   10 * time.Second,
	}); err != nil {
		t.Fatal(err)
	}
}

func readManifest(t *testing.T, path string) manifest {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var result manifest
	if err := json.Unmarshal(content, &result); err != nil {
		t.Fatal(err)
	}
	expectedDigest := result.ManifestSHA256
	result.ManifestSHA256 = ""
	if actual := digestJSON(manifestAlgorithm, result); actual != expectedDigest {
		t.Fatalf("sello inválido: obtenido=%s esperado=%s", expectedDigest, actual)
	}
	result.ManifestSHA256 = expectedDigest
	return result
}

func assertSource(t *testing.T, sources []sourceRecord, path, kind, status string) sourceRecord {
	t.Helper()
	source := sourceAt(t, sources, path)
	if source.Kind != kind || source.Status != status {
		t.Fatalf("fuente %q inesperada: %#v", path, source)
	}
	if source.SourceSHA256 == "" {
		t.Fatalf("fuente %q sin sello", path)
	}
	expected := source.SourceSHA256
	source.SourceSHA256 = ""
	if actual := digestJSON(sourceSealAlgorithm, source); actual != expected {
		t.Fatalf("sello de fuente %q inválido: obtenido=%s esperado=%s", path, expected, actual)
	}
	source.SourceSHA256 = expected
	return source
}

func sourceAt(t *testing.T, sources []sourceRecord, path string) sourceRecord {
	t.Helper()
	normalized, err := filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}
	normalized = filepath.Clean(normalized)
	for _, source := range sources {
		if source.Path == normalized {
			return source
		}
	}
	t.Fatalf("fuente %q ausente en %#v", normalized, sources)
	return sourceRecord{}
}

func snapshotGitMetadata(t *testing.T, repository string) map[string]string {
	t.Helper()
	gitDirectory := filepath.Join(repository, ".git")
	result := map[string]string{}
	err := filepath.WalkDir(gitDirectory, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(gitDirectory, path)
		if err != nil {
			return err
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		result[filepath.ToSlash(relative)] = strings.Join([]string{
			info.Mode().String(),
			string(content),
		}, "\x00")
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func configureGitTestIdentity(t *testing.T, repository string) {
	t.Helper()
	gitTest(t, repository, "config", "user.name", "Inventario")
	gitTest(t, repository, "config", "user.email", "inventario@example.invalid")
}

func gitTest(t *testing.T, repository string, arguments ...string) string {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", repository}, arguments...)...)
	content, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v: %s", strings.Join(arguments, " "), err, content)
	}
	return string(content)
}

func mustMkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
}

func writeTestFile(t *testing.T, root, relative, content string) {
	t.Helper()
	path := filepath.Join(root, relative)
	mustMkdirAll(t, filepath.Dir(path))
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func assertSameFile(t *testing.T, left, right string) {
	t.Helper()
	leftContent, err := os.ReadFile(left)
	if err != nil {
		t.Fatal(err)
	}
	rightContent, err := os.ReadFile(right)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(leftContent, rightContent) {
		t.Fatal("el manifiesto no es determinista")
	}
}
