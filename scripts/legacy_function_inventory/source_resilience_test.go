// Estas pruebas fijan la recuperación del AST parcial y excluyen de la
// superficie Go los enlaces simbólicos y gitlinks con nombre engañoso.
package main

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestPartialASTEmitsValidDeclarationAndParseFailure(t *testing.T) {
	repository := newCommittedRepository(t)
	writeTestFile(t, repository, "partial.go", "package sample\nfunc Kept() {}\nfunc Broken(\n")
	gitTest(t, repository, "add", "partial.go")
	gitTest(t, repository, "commit", "-qm", "fuente parcial")

	jsonl := filepath.Join(t.TempDir(), "inventory.jsonl")
	if err := run(options{
		repository: repository,
		jsonl:      jsonl,
		manifest:   filepath.Join(t.TempDir(), "manifest.json"),
	}); err != nil {
		t.Fatal(err)
	}
	declarationBlob := ""
	failures := make(map[string]bool)
	for _, item := range readTestRecords(t, jsonl) {
		if item.RecordKind == "go_declaration" && item.Name == "Kept" {
			declarationBlob = item.BlobID
		}
		if item.RecordKind == "go_declaration" && item.Name == "Broken" {
			t.Fatalf("se inventarió la declaración rota: %#v", item)
		}
		if item.RecordKind == "parse_failure" {
			failures[item.BlobID] = true
		}
	}
	if declarationBlob == "" || !failures[declarationBlob] {
		t.Fatalf("no coexistieron declaración válida y fallo: blob=%q fallos=%v", declarationBlob, failures)
	}
}

func TestSymlinkAndSubmoduleNamedGoAreNotParsed(t *testing.T) {
	repository := newCommittedRepository(t)
	writeTestFile(t, repository, "contenido-enlace", "package falsepositive\nfunc Wrong() {}\n")
	symlinkBlob := strings.TrimSpace(gitTest(t, repository, "hash-object", "-w", "contenido-enlace"))
	submoduleCommit := strings.TrimSpace(gitTest(t, repository, "rev-parse", "HEAD"))
	gitTest(t, repository, "update-index", "--add", "--cacheinfo",
		"120000", symlinkBlob, "enlace.go")
	gitTest(t, repository, "update-index", "--add", "--cacheinfo",
		"160000", submoduleCommit, "submodulo.go")
	gitTest(t, repository, "commit", "-qm", "entradas no regulares")

	jsonl := filepath.Join(t.TempDir(), "inventory.jsonl")
	if err := run(options{
		repository: repository,
		jsonl:      jsonl,
		manifest:   filepath.Join(t.TempDir(), "manifest.json"),
	}); err != nil {
		t.Fatal(err)
	}
	entries := make(map[string]record)
	for _, item := range readTestRecords(t, jsonl) {
		if item.RecordKind == "tree_entry" {
			entries[item.ChildObjectID] = item
		}
		if (item.RecordKind == "go_blob" || item.RecordKind == "go_declaration" ||
			item.RecordKind == "parse_failure") &&
			(item.BlobID == symlinkBlob || item.BlobID == submoduleCommit) {
			t.Fatalf("se analizó una entrada Git no regular como Go: %#v", item)
		}
	}
	if entries[symlinkBlob].FileMode != "120000" ||
		entries[symlinkBlob].ChildType != "blob" {
		t.Fatalf("enlace simbólico no conservado: %#v", entries[symlinkBlob])
	}
	if entries[submoduleCommit].FileMode != "160000" ||
		entries[submoduleCommit].ChildType != "commit" {
		t.Fatalf("submódulo no conservado: %#v", entries[submoduleCommit])
	}
}
