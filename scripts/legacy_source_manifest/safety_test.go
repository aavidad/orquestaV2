// Estas pruebas cubren fronteras de lectura, exclusión y fallo seguro.
package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestManifestDoesNotFollowSymbolicLinks(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "raíz")
	outside := filepath.Join(base, "fuera")
	mustMkdirAll(t, root)
	mustMkdirAll(t, outside)
	gitTest(t, outside, "init", "-q")
	configureGitTestIdentity(t, outside)
	writeTestFile(t, outside, "secret.go", "package secret\n")
	gitTest(t, outside, "add", "secret.go")
	gitTest(t, outside, "commit", "-qm", "fuera")
	link := filepath.Join(root, "enlace")
	if err := os.Symlink(outside, link); err != nil {
		t.Fatal(err)
	}

	output := filepath.Join(base, "manifest.json")
	runManifestTest(t, []string{root}, nil, output)
	result := readManifest(t, output)
	assertSource(t, result.Sources, link, "symbolic_link", "excluded")
	for _, source := range result.Sources {
		if source.Path == outside || strings.HasPrefix(source.Path, outside+string(filepath.Separator)) {
			t.Fatalf("se siguió el enlace fuera de la raíz: %#v", source)
		}
	}
}

func TestManifestRecordsUnreadableRepositoryAsError(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "fuentes")
	repository := filepath.Join(root, "ilegible")
	mustMkdirAll(t, repository)
	gitTest(t, repository, "init", "-q")
	configureGitTestIdentity(t, repository)
	writeTestFile(t, repository, "main.go", "package sample\n")
	gitTest(t, repository, "add", "main.go")
	gitTest(t, repository, "commit", "-qm", "inicial")
	marker := filepath.Join(repository, ".git")
	if err := os.Chmod(marker, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(marker, 0o700)
	})

	output := filepath.Join(base, "manifest.json")
	runManifestTest(t, []string{root}, nil, output)
	result := readManifest(t, output)
	source := assertSource(t, result.Sources, repository, "git_repository", "error")
	if source.ErrorCode != "permission_denied" {
		t.Fatalf("código de error inestable: %#v", source)
	}
	if result.Summary.SourcesWithErrors != 1 {
		t.Fatalf("el resumen no registra el error: %#v", result.Summary)
	}
}

func TestManifestRequiresPhysicalInventoryWhenWorktreeStateCannotBeAssessed(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "fuentes")
	repository := filepath.Join(root, "índice dañado")
	mustMkdirAll(t, repository)
	gitTest(t, repository, "init", "-q")
	configureGitTestIdentity(t, repository)
	writeTestFile(t, repository, "main.go", "package sample\n")
	gitTest(t, repository, "add", "main.go")
	gitTest(t, repository, "commit", "-qm", "inicial")
	writeTestFile(t, repository, ".git/index", "índice deliberadamente inválido")

	output := filepath.Join(base, "manifest.json")
	runManifestTest(t, []string{root}, nil, output)
	result := readManifest(t, output)
	source := assertSource(t, result.Sources, repository, "git_repository", "error")
	if source.ErrorCode != "git_worktree_status_failed" ||
		source.WorktreeState != "dirty_state_unassessed" ||
		!source.RequiresPhysicalInventory {
		t.Fatalf("estado no evaluable sin obligación física: %#v", source)
	}
	if result.Summary.UnassessedWorktrees != 1 || result.Summary.PhysicalInventoriesDue != 1 {
		t.Fatalf("resumen no evaluable inesperado: %#v", result.Summary)
	}
}

func TestManifestRecordsExplicitExclusionsWithoutReadingThem(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "fuentes")
	excluded := filepath.Join(root, "no inspeccionar")
	nestedRepository := filepath.Join(excluded, "repositorio")
	mustMkdirAll(t, nestedRepository)
	gitTest(t, nestedRepository, "init", "-q")
	configureGitTestIdentity(t, nestedRepository)
	writeTestFile(t, nestedRepository, "main.go", "package sample\n")
	gitTest(t, nestedRepository, "add", "main.go")
	gitTest(t, nestedRepository, "commit", "-qm", "inicial")

	output := filepath.Join(base, "manifest.json")
	runManifestTest(t, []string{root}, []string{excluded}, output)
	result := readManifest(t, output)
	source := assertSource(t, result.Sources, excluded, "path", "excluded")
	if source.Reason != "operator_excluded" {
		t.Fatalf("exclusión sin motivo controlado: %#v", source)
	}
	for _, item := range result.Sources {
		if item.Path == nestedRepository {
			t.Fatalf("se inspeccionó contenido excluido: %#v", item)
		}
	}
}

func TestManifestRejectsRootSymbolicLinkAndOutputInsideSource(t *testing.T) {
	base := t.TempDir()
	target := filepath.Join(base, "objetivo")
	mustMkdirAll(t, target)
	link := filepath.Join(base, "raíz-enlace")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(base, "manifest.json")
	runManifestTest(t, []string{link}, nil, output)
	result := readManifest(t, output)
	if len(result.Roots) != 1 || result.Roots[0].Status != "excluded" {
		t.Fatalf("raíz simbólica no rechazada: %#v", result.Roots)
	}
	assertSource(t, result.Sources, link, "symbolic_link", "excluded")

	err := run(options{
		roots:        []string{target},
		manifestPath: filepath.Join(target, "manifest.json"),
		gitTimeout:   defaultGitTimeout,
	})
	if err == nil || !strings.Contains(err.Error(), "dentro de la raíz histórica") {
		t.Fatalf("se esperaba rechazo de salida dentro de fuentes; error=%v", err)
	}
}

func TestManifestRejectsExclusionOutsideExplicitRoots(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "fuentes")
	outside := filepath.Join(base, "fuera")
	mustMkdirAll(t, root)
	err := run(options{
		roots:        []string{root},
		excluded:     []string{outside},
		manifestPath: filepath.Join(base, "manifest.json"),
		gitTimeout:   defaultGitTimeout,
	})
	if err == nil || !strings.Contains(err.Error(), "fuera de las raíces explícitas") {
		t.Fatalf("se esperaba rechazo de exclusión fuera de alcance; error=%v", err)
	}
}

func TestInvalidBundleIsExcludedWithStableReason(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "fuentes")
	mustMkdirAll(t, root)
	invalid := filepath.Join(root, "falso.bundle")
	writeTestFile(t, root, "falso.bundle", "no es un paquete Git\n")
	output := filepath.Join(base, "manifest.json")
	runManifestTest(t, []string{root}, nil, output)
	source := assertSource(t, readManifest(t, output).Sources, invalid, "git_bundle", "excluded")
	if source.Reason != "invalid_git_bundle" {
		t.Fatalf("motivo inesperado: %#v", source)
	}
}
