// Estas pruebas fijan el snapshot completo de referencias y demuestran que la
// historia se deriva de sus OID inmutables, sin volver a consultar nombres.
package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReferenceSnapshotIncludesStashCustomAndNonCommitRefs(t *testing.T) {
	repository := newCommittedRepository(t)
	first := strings.TrimSpace(gitTest(t, repository, "rev-parse", "HEAD"))
	writeTestFile(t, repository, "second.go", "package sample\nfunc Second() {}\n")
	gitTest(t, repository, "add", "second.go")
	gitTest(t, repository, "commit", "-qm", "segunda")
	second := strings.TrimSpace(gitTest(t, repository, "rev-parse", "HEAD"))
	gitTest(t, repository, "reset", "--hard", "-q", first)
	gitTest(t, repository, "update-ref", "refs/stash", second)
	gitTest(t, repository, "update-ref", "refs/orquesta/archivo", second)
	writeTestFile(t, repository, "objeto.txt", "contenido sin confirmación")
	blob := strings.TrimSpace(gitTest(t, repository, "hash-object", "-w", "objeto.txt"))
	gitTest(t, repository, "update-ref", "refs/orquesta/blob", blob)
	branch := strings.TrimSpace(gitTest(t, repository, "symbolic-ref", "HEAD"))

	refs, err := readRefs(repository, nil)
	if err != nil {
		t.Fatal(err)
	}
	byName := make(map[string]refInfo, len(refs))
	for _, ref := range refs {
		byName[string(ref.Name)] = ref
	}
	for _, name := range []string{branch, "refs/stash", "refs/orquesta/archivo", "refs/orquesta/blob"} {
		if _, found := byName[name]; !found {
			t.Fatalf("el snapshot omitió %s: %#v", name, refs)
		}
	}
	if byName["refs/stash"].Commit != second ||
		byName["refs/orquesta/archivo"].Commit != second {
		t.Fatalf("referencias históricas mal peladas: %#v", byName)
	}
	if byName["refs/orquesta/blob"].Object != blob ||
		byName["refs/orquesta/blob"].Commit != "" {
		t.Fatalf("referencia no commit perdida o falseada: %#v", byName["refs/orquesta/blob"])
	}
	history, err := readHistory(repository, refs, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 2 {
		t.Fatalf("la historia del snapshot contiene %d confirmaciones: %#v", len(history), history)
	}
	if _, found := history[first]; !found {
		t.Fatalf("falta la confirmación inicial %s", first)
	}
	if _, found := history[second]; !found {
		t.Fatalf("falta la confirmación de refs/stash %s", second)
	}
}

func TestHistoryUsesInitialObjectIDsDespiteReferenceRoundTrip(t *testing.T) {
	repository := newCommittedRepository(t)
	initial := strings.TrimSpace(gitTest(t, repository, "rev-parse", "HEAD"))
	snapshot, err := readRefs(repository, nil)
	if err != nil {
		t.Fatal(err)
	}
	tree := strings.TrimSpace(gitTest(t, repository, "rev-parse", "HEAD^{tree}"))
	intruder := strings.TrimSpace(gitTest(t, repository, "commit-tree", tree, "-m", "intrusa"))
	branch := strings.TrimSpace(gitTest(t, repository, "symbolic-ref", "HEAD"))
	gitTest(t, repository, "update-ref", branch, intruder)

	history, err := readHistory(repository, snapshot, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 1 {
		t.Fatalf("la historia incorporó estado ajeno al snapshot: %#v", history)
	}
	if _, found := history[initial]; !found {
		t.Fatalf("falta el OID inicial %s", initial)
	}
	if _, found := history[intruder]; found {
		t.Fatalf("se inyectó la confirmación posterior %s", intruder)
	}
	gitTest(t, repository, "update-ref", branch, initial)
	after, err := readRefs(repository, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !equalRefs(snapshot, after) {
		t.Fatalf("la ida y vuelta no restauró el snapshot: antes=%#v después=%#v", snapshot, after)
	}
}

func TestDetachedHeadIsPartOfTheHistorySnapshot(t *testing.T) {
	repository := newCommittedRepository(t)
	branch := strings.TrimSpace(gitTest(t, repository, "symbolic-ref", "HEAD"))
	writeTestFile(t, repository, "detached.go", "package sample\nfunc Detached() {}\n")
	gitTest(t, repository, "add", "detached.go")
	gitTest(t, repository, "commit", "-qm", "separada")
	detached := strings.TrimSpace(gitTest(t, repository, "rev-parse", "HEAD"))
	gitTest(t, repository, "checkout", "--detach", "-q", detached)
	gitTest(t, repository, "update-ref", "-d", branch)

	refs, err := readRefs(repository, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(refs) != 1 || !bytes.Equal(refs[0].Name, []byte("HEAD")) ||
		refs[0].Commit != detached {
		t.Fatalf("HEAD separado no quedó censado: %#v", refs)
	}
	history, err := readHistory(repository, refs, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, found := history[detached]; !found {
		t.Fatalf("la historia omitió HEAD separado %s", detached)
	}
}

func TestReferenceToMissingObjectFailsWithoutFetching(t *testing.T) {
	repository := newCommittedRepository(t)
	missing := strings.Repeat("f", 40)
	path := filepath.Join(repository, ".git", "refs", "orquesta", "ausente")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(missing+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := readRefs(repository, nil)
	if err == nil || !strings.Contains(err.Error(), "objeto ausente") {
		t.Fatalf("se esperaba fallo explícito por objeto ausente; obtenido: %v", err)
	}
}

func TestAnnotatedTagToMissingCommitFailsWithoutFetching(t *testing.T) {
	repository := newCommittedRepository(t)
	missing := strings.Repeat("e", 40)
	tagContent := "object " + missing + "\n" +
		"type commit\n" +
		"tag ausente\n" +
		"tagger Inventario <inventario@example.invalid> 0 +0000\n\n" +
		"destino ausente\n"
	writeTestFile(t, repository, "tag-invalida", tagContent)
	tagObject := strings.TrimSpace(gitTest(
		t, repository, "hash-object", "--literally", "-t", "tag", "-w", "tag-invalida",
	))
	gitTest(t, repository, "update-ref", "refs/tags/ausente", tagObject)
	_, err := readRefs(repository, nil)
	if err == nil || !strings.Contains(err.Error(), "alcanza el objeto ausente") {
		t.Fatalf("se esperaba fallo por destino ausente de etiqueta; obtenido: %v", err)
	}
}

func newCommittedRepository(t *testing.T) string {
	t.Helper()
	repository := t.TempDir()
	gitTest(t, repository, "init", "-q")
	gitTest(t, repository, "config", "user.name", "Inventario")
	gitTest(t, repository, "config", "user.email", "inventario@example.invalid")
	writeTestFile(t, repository, "main.go", "package sample\nfunc First() {}\n")
	gitTest(t, repository, "add", "main.go")
	gitTest(t, repository, "commit", "-qm", "primera")
	return repository
}
