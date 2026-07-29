package main

import (
	"bufio"
	"bytes"
	"crypto/sha1"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"
)

func TestInventoryIsDeterministicAndCoversHistoricalVariantsAndFailures(t *testing.T) {
	repository := t.TempDir()
	gitTest(t, repository, "init", "-q")
	gitTest(t, repository, "config", "user.name", "Inventario")
	gitTest(t, repository, "config", "user.email", "inventario@example.invalid")

	writeTestFile(t, repository, "main.go", "package sample\n\nfunc Stable() int { return 1 }\n")
	gitTest(t, repository, "add", "main.go")
	gitTest(t, repository, "commit", "-qm", "primera")
	first := strings.TrimSpace(gitTest(t, repository, "rev-parse", "HEAD"))
	gitTest(t, repository, "tag", "-a", "v1", "-m", "v1")

	writeTestFile(t, repository, "main.go", "package sample\n\ntype T struct{}\nfunc Stable() int { return 2 }\nfunc (T) Method() {}\n")
	gitTest(t, repository, "add", "main.go")
	gitTest(t, repository, "commit", "-qm", "segunda")
	gitTest(t, repository, "branch", "feature")
	gitTest(t, repository, "checkout", "-q", "--detach", first)
	writeTestFile(t, repository, "broken.go", "package sample\nfunc Broken(\n")
	gitTest(t, repository, "add", "broken.go")
	gitTest(t, repository, "commit", "-qm", "rota")
	gitTest(t, repository, "branch", "broken")
	gitTest(t, repository, "checkout", "-q", "feature")

	firstJSONL := filepath.Join(t.TempDir(), "first.jsonl")
	firstManifest := filepath.Join(t.TempDir(), "first.json")
	secondJSONL := filepath.Join(t.TempDir(), "second.jsonl")
	secondManifest := filepath.Join(t.TempDir(), "second.json")
	if err := run(options{repository: repository, jsonl: firstJSONL, manifest: firstManifest}); err != nil {
		t.Fatal(err)
	}
	if err := run(options{repository: repository, jsonl: secondJSONL, manifest: secondManifest}); err != nil {
		t.Fatal(err)
	}
	assertSameFile(t, firstJSONL, secondJSONL)
	assertSameFile(t, firstManifest, secondManifest)

	records := readTestRecords(t, firstJSONL)
	counts := map[string]int{}
	var methodSeen, failureSeen bool
	stableVariants := map[string]struct{}{}
	for _, item := range records {
		counts[item.RecordKind]++
		if item.RecordKind == "function_occurrence" && item.SymbolKind == "method" && item.Name == "Method" {
			methodSeen = true
		}
		if item.RecordKind == "parse_failure" && item.ErrorCode == "go_parse_failed" {
			failureSeen = true
		}
		if item.RecordKind == "function_variant" && item.Name == "Stable" {
			stableVariants[item.VariantRef] = struct{}{}
		}
	}
	if !methodSeen || !failureSeen || len(stableVariants) != 2 {
		t.Fatalf("cobertura incompleta: método=%v fallo=%v variantes Stable=%d", methodSeen, failureSeen, len(stableVariants))
	}
	if counts["ref"] < 3 || counts["commit_ref_reachability"] == 0 ||
		counts["commit"] != 3 || counts["tree"] == 0 || counts["go_blob"] < 3 {
		t.Fatalf("conteos inesperados: %#v", counts)
	}

	var manifestValue manifest
	content, err := os.ReadFile(firstManifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(content, &manifestValue); err != nil {
		t.Fatal(err)
	}
	if manifestValue.InventorySHA256 == "" || manifestValue.RefSnapshotSHA256 == "" ||
		manifestValue.Counts["function_variant"] < 3 {
		t.Fatalf("manifiesto incompleto: %#v", manifestValue)
	}
}

func TestInventoryRejectsRepositoryWithoutRefs(t *testing.T) {
	repository := t.TempDir()
	gitTest(t, repository, "init", "-q")
	err := run(options{
		repository: repository,
		jsonl:      filepath.Join(t.TempDir(), "out.jsonl"),
		manifest:   filepath.Join(t.TempDir(), "manifest.json"),
	})
	if err == nil {
		t.Fatal("se esperaba rechazo del repositorio sin confirmaciones")
	}
}

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

	history, err := readHistory(repository, nil)
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
	cache := map[string][]treeEntry{}
	for _, commit := range commits {
		expectedHeader := individualCommitHeader(t, repository, commit)
		if actual := history[commit]; !reflect.DeepEqual(actual, expectedHeader) {
			t.Fatalf("cabecera distinta para %s: obtenida=%#v esperada=%#v", commit, actual, expectedHeader)
		}
		actualTree, err := readTree(batch, expectedHeader.tree, sha1.Size, cache)
		if err != nil {
			t.Fatal(err)
		}
		expectedTree := individualTree(t, repository, expectedHeader.tree)
		if !reflect.DeepEqual(actualTree, expectedTree) {
			t.Fatalf("árbol distinto para %s:\nobtenido=%#v\nesperado=%#v", commit, actualTree, expectedTree)
		}
	}
	refs, err := readRefs(repository, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, ref := range refs {
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

func TestInventoryUsesBoundedGitProcesses(t *testing.T) {
	repository := t.TempDir()
	gitTest(t, repository, "init", "-q")
	gitTest(t, repository, "config", "user.name", "Inventario")
	gitTest(t, repository, "config", "user.email", "inventario@example.invalid")
	for index := 0; index < 12; index++ {
		writeTestFile(t, repository, "main.go", "package sample\nfunc Value() int { return "+strconv.Itoa(index)+" }\n")
		gitTest(t, repository, "add", "main.go")
		gitTest(t, repository, "commit", "-qm", "cambio")
		if index%2 == 0 {
			gitTest(t, repository, "branch", "rama-"+strconv.Itoa(index))
		}
	}

	processes := 0
	err := run(options{
		repository: repository,
		jsonl:      filepath.Join(t.TempDir(), "out.jsonl"),
		manifest:   filepath.Join(t.TempDir(), "manifest.json"),
		gitProcessStarted: func() {
			processes++
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if processes > 7 {
		t.Fatalf("el censo abrió %d procesos Git; se esperaban como máximo 7", processes)
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
	content := []byte(gitTest(t, repository, "ls-tree", "-r", "-t", "-z", tree))
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
			Path: string(path),
		})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Path != result[j].Path {
			return result[i].Path < result[j].Path
		}
		return result[i].OID < result[j].OID
	})
	return result
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

func writeTestFile(t *testing.T, repository, relative, content string) {
	t.Helper()
	path := filepath.Join(repository, relative)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
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
	if string(leftContent) != string(rightContent) {
		t.Fatal("el inventario no es determinista")
	}
}

func readTestRecords(t *testing.T, path string) []record {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	var records []record
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var item record
		if err := json.Unmarshal(scanner.Bytes(), &item); err != nil {
			t.Fatal(err)
		}
		records = append(records, item)
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	return records
}
