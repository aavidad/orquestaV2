// Estas pruebas demuestran que el grafo reconstruye confirmación, ruta y
// declaración sin materializar una aparición por cada confirmación.
package main

import (
	"bytes"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGraphReconstructsExactFunctionProvenance(t *testing.T) {
	repository := t.TempDir()
	gitTest(t, repository, "init", "-q")
	gitTest(t, repository, "config", "user.name", "Inventario")
	gitTest(t, repository, "config", "user.email", "inventario@example.invalid")
	writeTestFile(t, repository, "main.go", "package sample\nfunc First() {}\n")
	writeTestFile(t, repository, "nested/extra.go", "package nested\nfunc Extra() {}\n")
	gitTest(t, repository, "add", ".")
	gitTest(t, repository, "commit", "-qm", "primera")
	first := strings.TrimSpace(gitTest(t, repository, "rev-parse", "HEAD"))
	writeTestFile(t, repository, "main.go", "package sample\nfunc Second() {}\n")
	gitTest(t, repository, "add", "main.go")
	gitTest(t, repository, "commit", "-qm", "segunda")
	second := strings.TrimSpace(gitTest(t, repository, "rev-parse", "HEAD"))

	jsonl := filepath.Join(t.TempDir(), "inventory.jsonl")
	manifestPath := filepath.Join(t.TempDir(), "manifest.json")
	if err := run(options{repository: repository, jsonl: jsonl, manifest: manifestPath}); err != nil {
		t.Fatal(err)
	}
	records := readTestRecords(t, jsonl)
	provenance, branch := reconstructProvenance(t, records)
	expected := map[string]struct{}{
		provenanceKey(branch, first, "main.go", "First"):          {},
		provenanceKey(branch, first, "nested/extra.go", "Extra"):  {},
		provenanceKey(branch, second, "main.go", "Second"):        {},
		provenanceKey(branch, second, "nested/extra.go", "Extra"): {},
	}
	if len(provenance) != len(expected) {
		t.Fatalf("procedencia inesperada: obtenida=%v esperada=%v", provenance, expected)
	}
	for key := range expected {
		if _, exists := provenance[key]; !exists {
			t.Fatalf("falta procedencia %q en %v", key, provenance)
		}
	}
}

func TestGraphPreservesNonUTF8PathSegmentAsBase64(t *testing.T) {
	repository := t.TempDir()
	gitTest(t, repository, "init", "-q")
	gitTest(t, repository, "config", "user.name", "Inventario")
	gitTest(t, repository, "config", "user.email", "inventario@example.invalid")
	rawName := []byte{'n', 'o', 'm', 'b', 'r', 'e', '-', 0xff, '.', 'g', 'o'}
	path := filepath.Join(repository, string(rawName))
	if err := os.WriteFile(path, []byte("package sample\nfunc Preserved() {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	gitTest(t, repository, "add", "--", string(rawName))
	gitTest(t, repository, "commit", "-qm", "ruta binaria")

	jsonl := filepath.Join(t.TempDir(), "inventory.jsonl")
	if err := run(options{
		repository: repository,
		jsonl:      jsonl,
		manifest:   filepath.Join(t.TempDir(), "manifest.json"),
	}); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(jsonl)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(content, []byte("\xef\xbf\xbd")) {
		t.Fatal("el JSONL contiene un carácter de sustitución y perdió bytes de ruta")
	}
	records := readTestRecords(t, jsonl)
	var encodedEntry record
	declarations := map[string]bool{}
	for _, item := range records {
		if item.RecordKind == "tree_entry" && item.PathEncoding == "base64" {
			encodedEntry = item
		}
		if item.RecordKind == "go_declaration" && item.Name == "Preserved" {
			declarations[item.BlobID] = true
		}
	}
	if encodedEntry.PathSegmentBase64 != base64.StdEncoding.EncodeToString(rawName) ||
		encodedEntry.PathSegment != "" {
		t.Fatalf("segmento binario alterado: %#v", encodedEntry)
	}
	decoded, err := decodeRecordSegment(encodedEntry)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(decoded, rawName) || !declarations[encodedEntry.ChildObjectID] {
		t.Fatalf("ruta o enlace al blob no reconstruible: ruta=%x entrada=%#v", decoded, encodedEntry)
	}
}

func reconstructProvenance(t *testing.T, records []record) (map[string]struct{}, string) {
	t.Helper()
	commits := map[string]record{}
	trees := map[string][]record{}
	declarations := map[string][]record{}
	var branch record
	for _, item := range records {
		switch item.RecordKind {
		case "reference":
			if strings.HasPrefix(item.RefName, "refs/heads/") {
				branch = item
			}
		case "commit":
			commits[item.CommitID] = item
		case "tree_entry":
			trees[item.TreeID] = append(trees[item.TreeID], item)
		case "go_declaration":
			declarations[item.BlobID] = append(declarations[item.BlobID], item)
		}
	}
	if branch.RefName == "" {
		t.Fatal("no se encontró referencia de rama")
	}
	result := map[string]struct{}{}
	seenCommits := map[string]struct{}{}
	var walkCommit func(string)
	walkCommit = func(commitID string) {
		if _, exists := seenCommits[commitID]; exists {
			return
		}
		seenCommits[commitID] = struct{}{}
		commit, exists := commits[commitID]
		if !exists {
			t.Fatalf("confirmación %s ausente", commitID)
		}
		walkTreeProvenance(t, commit.TreeID, nil, trees, declarations, func(path, name string) {
			result[provenanceKey(branch.RefName, commitID, path, name)] = struct{}{}
		})
		for _, parent := range commit.Parents {
			walkCommit(parent)
		}
	}
	walkCommit(branch.CommitID)
	return result, branch.RefName
}

func walkTreeProvenance(
	t *testing.T,
	treeID string,
	prefix []byte,
	trees map[string][]record,
	declarations map[string][]record,
	visit func(path, name string),
) {
	t.Helper()
	for _, entry := range trees[treeID] {
		segment, err := decodeRecordSegment(entry)
		if err != nil {
			t.Fatal(err)
		}
		path := append(append([]byte{}, prefix...), segment...)
		if entry.ChildType == "tree" {
			path = append(path, '/')
			walkTreeProvenance(t, entry.ChildObjectID, path, trees, declarations, visit)
			continue
		}
		for _, declaration := range declarations[entry.ChildObjectID] {
			visit(string(path), declaration.Name)
		}
	}
}

func decodeRecordSegment(item record) ([]byte, error) {
	if item.PathEncoding == "utf8" {
		return []byte(item.PathSegment), nil
	}
	return base64.StdEncoding.DecodeString(item.PathSegmentBase64)
}

func provenanceKey(refName, commitID, path, function string) string {
	return strings.Join([]string{refName, commitID, path, function}, "\x00")
}
