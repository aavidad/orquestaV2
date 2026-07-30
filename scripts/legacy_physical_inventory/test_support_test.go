// Este fichero reúne utilidades de prueba para leer artefactos sin ocultar aserciones.
package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func testOptions(t *testing.T, root, mode string) options {
	t.Helper()
	output := t.TempDir()
	if err := os.Chmod(output, 0o700); err != nil {
		t.Fatal(err)
	}
	return options{
		roots:        []rootOption{{alias: "prueba", mode: mode, path: root}},
		jsonlPath:    filepath.Join(output, "censo.jsonl"),
		manifestPath: filepath.Join(output, "censo.manifest.json"),
		locationReader: func(_ int, path string) (physicalLocation, error) {
			if path == root {
				return physicalLocation{device: 1, path: root}, nil
			}
			return physicalLocation{device: 2, path: output}, nil
		},
		budget: budgetOptions{
			maxEntries: 1_000, maxDirectoryEntries: 1_000,
			maxDepth: 64, maxPathBytes: 8 << 10, maxOutputBytes: 8 << 20,
			maxHashBytes: 1 << 20, maxFileBytes: 1 << 20, timeout: time.Minute,
		},
	}
}
func readRecords(t *testing.T, path string) []record {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	var values []record
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 4096), 4<<20)
	for scanner.Scan() {
		var value record
		if err := json.Unmarshal(scanner.Bytes(), &value); err != nil {
			t.Fatal(err)
		}
		values = append(values, value)
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	return values
}
func readManifest(t *testing.T, path string) manifest {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var value manifest
	if err := json.Unmarshal(content, &value); err != nil {
		t.Fatal(err)
	}
	return value
}
func findRecord(values []record, kind string, path ...string) (record, bool) {
	for _, value := range values {
		if value.Kind != kind || len(value.Path) != len(path) {
			continue
		}
		match := true
		for index, segment := range value.Path {
			match = match && segment.Encoding == "utf8" && segment.Value == path[index]
		}
		if match {
			return value, true
		}
	}
	return record{}, false
}
func mustWrite(t *testing.T, path, content string, mode os.FileMode) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		t.Fatal(err)
	}
}
func mustSucceed(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
func existsAt(directoryFD int, name string) bool {
	exists, err := entryExistsAt(directoryFD, name)
	return err == nil && exists
}
func newPublicationForTest(opts options) (*publication, error) {
	roots, err := anchorRootsBefore(opts.roots, time.Now().Add(opts.budget.timeout))
	if err != nil {
		return nil, err
	}
	value, publicationErr := newPublication(opts, roots)
	return value, errors.Join(publicationErr, closeAnchoredRoots(roots))
}
