// Estas ayudas construyen repositorios temporales y leen resultados de prueba.
// No contienen criterios de aceptación: sirven a las pruebas de cada frontera.
package main

import (
	"bufio"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

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
