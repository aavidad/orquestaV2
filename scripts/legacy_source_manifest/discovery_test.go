// Estas pruebas cubren el descubrimiento y la identidad física de las fuentes.
package main

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestManifestDiscoversRepositoriesWorktreesAndBundlesDeterministically(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "fuentes históricas")
	repository := filepath.Join(root, "repositorio con espacios")
	worktree := filepath.Join(root, "árbol enlazado")
	bare := filepath.Join(root, "respaldo.git")
	bundle := filepath.Join(root, "copias", "historia.bundle")
	mustMkdirAll(t, repository)
	gitTest(t, repository, "init", "-q")
	configureGitTestIdentity(t, repository)
	writeTestFile(t, repository, "main.go", "package sample\nfunc First() {}\n")
	gitTest(t, repository, "add", "main.go")
	gitTest(t, repository, "commit", "-qm", "primera")
	gitTest(t, repository, "tag", "-a", "v1", "-m", "v1")
	mustMkdirAll(t, filepath.Dir(worktree))
	gitTest(t, repository, "worktree", "add", "-q", "-b", "trabajo", worktree)
	gitTest(t, repository, "clone", "-q", "--bare", repository, bare)
	mustMkdirAll(t, filepath.Dir(bundle))
	gitTest(t, repository, "bundle", "create", bundle, "--all")

	before := snapshotGitMetadata(t, repository)
	first := filepath.Join(base, "primero.json")
	second := filepath.Join(base, "segundo.json")
	runManifestTest(t, []string{root}, nil, first)
	runManifestTest(t, []string{root}, nil, second)
	assertSameFile(t, first, second)
	after := snapshotGitMetadata(t, repository)
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("la inspección modificó metadatos Git:\nantes=%#v\ndespués=%#v", before, after)
	}

	result := readManifest(t, first)
	if result.ManifestSHA256 == "" || result.Summary.IncludedSources != 4 {
		t.Fatalf("manifiesto incompleto: %#v", result.Summary)
	}
	assertSource(t, result.Sources, repository, "git_repository", "included")
	assertSource(t, result.Sources, worktree, "git_worktree", "included")
	assertSource(t, result.Sources, bare, "git_bare_repository", "included")
	bundleSource := assertSource(t, result.Sources, bundle, "git_bundle", "included")
	if bundleSource.ContentSHA256 == "" || bundleSource.SizeBytes == 0 || len(bundleSource.References) == 0 {
		t.Fatalf("paquete Git sin sello suficiente: %#v", bundleSource)
	}
	repositorySource := sourceAt(t, result.Sources, repository)
	if repositorySource.HeadObject == "" || repositorySource.ReferenceSHA256 == "" ||
		repositorySource.ReachableCommitCount == nil || *repositorySource.ReachableCommitCount != 1 ||
		repositorySource.WorktreeState != "clean" {
		t.Fatalf("repositorio sin revisión reproducible y limpia: %#v", repositorySource)
	}
}

func TestManifestKeepsDirtyWorktreeDistinctFromCleanWorktreeAtSameRevision(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "fuentes")
	repository := filepath.Join(root, "principal")
	dirtyWorktree := filepath.Join(root, "trabajo sucio")
	mustMkdirAll(t, repository)
	gitTest(t, repository, "init", "-q")
	configureGitTestIdentity(t, repository)
	writeTestFile(t, repository, "main.go", "package sample\n")
	gitTest(t, repository, "add", "main.go")
	gitTest(t, repository, "commit", "-qm", "inicial")
	gitTest(t, repository, "worktree", "add", "-q", "-b", "sucia", dirtyWorktree)
	writeTestFile(t, dirtyWorktree, "sin-confirmar.txt", "contenido que el manifiesto no debe copiar\n")

	output := filepath.Join(base, "manifest.json")
	runManifestTest(t, []string{root}, nil, output)
	result := readManifest(t, output)
	clean := assertSource(t, result.Sources, repository, "git_repository", "included")
	dirty := assertSource(t, result.Sources, dirtyWorktree, "git_worktree", "included")
	if clean.HeadObject != dirty.HeadObject {
		t.Fatalf("la prueba exige la misma revisión: limpia=%s sucia=%s", clean.HeadObject, dirty.HeadObject)
	}
	if clean.WorktreeState != "clean" || clean.RequiresPhysicalInventory {
		t.Fatalf("árbol limpio mal clasificado: %#v", clean)
	}
	if dirty.WorktreeState != "dirty" || dirty.WorktreeChangeCount != 1 ||
		dirty.WorktreeStatusSHA256 == "" || !dirty.RequiresPhysicalInventory {
		t.Fatalf("árbol sucio sin obligación física: %#v", dirty)
	}
	if clean.SourceSHA256 == dirty.SourceSHA256 {
		t.Fatal("dos árboles físicos distintos quedaron deduplicados por la revisión")
	}
	if result.Summary.DirtyWorktrees != 1 || result.Summary.PhysicalInventoriesDue != 1 {
		t.Fatalf("resumen físico inesperado: %#v", result.Summary)
	}
}
