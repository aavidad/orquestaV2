// Estas ayudas preparan repositorios y leen salidas para las pruebas del censo.
// No contienen criterios de clasificación, seguridad ni publicación.
package main

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func newTestRepository(t *testing.T) string {
	t.Helper()
	repository := t.TempDir()
	gitTest(t, repository, "init", "-q")
	gitTest(t, repository, "config", "user.name", "Inventario")
	gitTest(t, repository, "config", "user.email", "inventario@example.invalid")
	return repository
}

func writeTestBytes(t *testing.T, repository, relative string, content []byte) {
	t.Helper()
	filePath := filepath.Join(repository, relative)
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filePath, content, 0o600); err != nil {
		t.Fatal(err)
	}
}

func gitTest(t *testing.T, repository string, arguments ...string) string {
	t.Helper()
	return gitTestInput(t, repository, nil, arguments...)
}

func gitTestInput(t *testing.T, repository string, input []byte, arguments ...string) string {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", repository}, arguments...)...)
	command.Stdin = bytes.NewReader(input)
	content, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v: %s", strings.Join(arguments, " "), err, content)
	}
	return string(content)
}

func readRecords(t *testing.T, filePath string) []record {
	t.Helper()
	file, err := os.Open(filePath)
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

type reconstructedSurface struct {
	path  []byte
	entry record
	facts record
}

func reconstructSurfaces(t *testing.T, records []record) []reconstructedSurface {
	t.Helper()
	trees := map[string][]record{}
	blobs := map[string]record{}
	rootSet := map[string]struct{}{}
	for _, item := range records {
		switch item.RecordKind {
		case "confirmacion":
			rootSet[item.TreeID] = struct{}{}
		case "entrada_arbol":
			trees[item.TreeID] = append(trees[item.TreeID], item)
		case "hechos_blob":
			blobs[item.BlobID] = item
		}
	}
	roots := make([]string, 0, len(rootSet))
	for root := range rootSet {
		roots = append(roots, root)
	}
	var result []reconstructedSurface
	var walk func(string, []byte)
	walk = func(treeID string, prefix []byte) {
		for _, entry := range trees[treeID] {
			segment, err := decodeSegment(entry)
			if err != nil {
				t.Fatal(err)
			}
			filePath := joinGitPath(prefix, segment)
			if entry.ExclusionCause != "" {
				result = append(result, reconstructedSurface{path: filePath, entry: entry})
				continue
			}
			if entry.ChildType == "tree" {
				walk(entry.ChildObjectID, filePath)
				continue
			}
			result = append(result, reconstructedSurface{
				path: filePath, entry: entry, facts: blobs[entry.ChildObjectID],
			})
		}
	}
	for _, root := range roots {
		walk(root, nil)
	}
	return result
}

func decodeSegment(item record) ([]byte, error) {
	if item.PathEncoding == "utf8" {
		return []byte(item.PathSegment), nil
	}
	return base64.StdEncoding.DecodeString(item.PathSegmentBase64)
}

func surfacesByUTF8Path(t *testing.T, records []record) map[string][]reconstructedSurface {
	t.Helper()
	result := map[string][]reconstructedSurface{}
	for _, surface := range reconstructSurfaces(t, records) {
		if strings.ToValidUTF8(string(surface.path), "") != string(surface.path) {
			continue
		}
		result[string(surface.path)] = append(result[string(surface.path)], surface)
	}
	return result
}

func readJSONFile(t *testing.T, filePath string, destination any) {
	t.Helper()
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(content, destination); err != nil {
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
	if !reflect.DeepEqual(leftContent, rightContent) {
		t.Fatalf("salidas distintas:\n%s\n%s", leftContent, rightContent)
	}
}

func familyNames(item record) []string {
	result := make([]string, 0, len(item.Families))
	for _, family := range item.Families {
		result = append(result, family.Family)
	}
	return result
}

func equalStringSlices(left, right []string) bool {
	return reflect.DeepEqual(left, right)
}

func containsString(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}
