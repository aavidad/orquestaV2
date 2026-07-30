// Estas pruebas contrastan el grafo y los árboles leídos por lotes con las
// consultas individuales de Git, sin probar la serialización del inventario.
package main

import (
	"bytes"
	"crypto/sha1"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func TestBatchHistoryAndTreesMatchIndividualGitQueries(t *testing.T) {
	repository := t.TempDir()
	gitTest(t, repository, "init", "-q")
	gitTest(t, repository, "config", "user.name", "Inventario")
	gitTest(t, repository, "config", "user.email", "inventario@example.invalid")

	writeTestFile(t, repository, "main.go", "package sample\nfunc First() {}\n")
	writeTestFile(t, repository, "nested/extra.go", "package nested\nfunc Extra() {}\n")
	gitTest(t, repository, "add", ".")
	gitTest(t, repository, "commit", "-qm", "primera")
	gitTest(t, repository, "tag", "-a", "v1", "-m", "v1")
	writeTestFile(t, repository, "main.go", "package sample\nfunc Second() {}\n")
	gitTest(t, repository, "add", "main.go")
	gitTest(t, repository, "commit", "-qm", "segunda")
	gitTest(t, repository, "branch", "copia")

	refs, err := readRefs(repository, nil)
	if err != nil {
		t.Fatal(err)
	}
	history, err := readHistory(repository, refs, nil)
	if err != nil {
		t.Fatal(err)
	}
	commits := strings.Fields(gitTest(t, repository, "rev-list", "--all"))
	if len(history) != len(commits) {
		t.Fatalf("el grafo por lotes contiene %d confirmaciones; Git enumera %d", len(history), len(commits))
	}
	batch, err := newGitBatch(repository, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer batch.close()
	for _, commit := range commits {
		expectedHeader := individualCommitHeader(t, repository, commit)
		if actual := history[commit]; !reflect.DeepEqual(actual, expectedHeader) {
			t.Fatalf("cabecera distinta para %s: obtenida=%#v esperada=%#v", commit, actual, expectedHeader)
		}
		actualTree, err := readTreeObject(batch, expectedHeader.tree, sha1.Size)
		if err != nil {
			t.Fatal(err)
		}
		expectedTree := individualTree(t, repository, expectedHeader.tree)
		if !reflect.DeepEqual(actualTree, expectedTree) {
			t.Fatalf("árbol distinto para %s:\nobtenido=%#v\nesperado=%#v", commit, actualTree, expectedTree)
		}
	}
	for _, ref := range refs {
		if ref.Commit == "" {
			continue
		}
		actual, err := reachableCommits(ref.Commit, history)
		if err != nil {
			t.Fatal(err)
		}
		expected := strings.Fields(gitTest(t, repository, "rev-list", ref.Commit))
		sort.Strings(expected)
		if !reflect.DeepEqual(actual, expected) {
			t.Fatalf("alcanzabilidad distinta para %s: obtenida=%v esperada=%v", ref.Name, actual, expected)
		}
	}
}

func individualCommitHeader(t *testing.T, repository, commit string) historyEntry {
	t.Helper()
	content := gitTest(t, repository, "cat-file", "commit", commit)
	var result historyEntry
	for _, line := range strings.Split(content, "\n") {
		if line == "" {
			break
		}
		if strings.HasPrefix(line, "tree ") {
			result.tree = strings.TrimPrefix(line, "tree ")
		}
		if strings.HasPrefix(line, "parent ") {
			result.parents = append(result.parents, strings.TrimPrefix(line, "parent "))
		}
	}
	return result
}

func individualTree(t *testing.T, repository, tree string) []treeEntry {
	t.Helper()
	content := []byte(gitTest(t, repository, "ls-tree", "-z", tree))
	var result []treeEntry
	for _, raw := range bytes.Split(content, []byte{0}) {
		if len(raw) == 0 {
			continue
		}
		header, path, found := bytes.Cut(raw, []byte{'\t'})
		fields := strings.Fields(string(header))
		if !found || len(fields) != 3 {
			t.Fatalf("entrada de árbol inválida: %q", raw)
		}
		result = append(result, treeEntry{
			Mode: fields[0],
			Type: fields[1],
			OID:  fields[2],
			Path: bytes.Clone(path),
		})
	}
	sort.Slice(result, func(i, j int) bool {
		if comparison := bytes.Compare(result[i].Path, result[j].Path); comparison != 0 {
			return comparison < 0
		}
		return result[i].OID < result[j].OID
	})
	return result
}
