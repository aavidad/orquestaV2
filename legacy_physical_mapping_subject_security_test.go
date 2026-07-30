// Este contrato calcula el sujeto físico exacto del validador y rechaza
// cualquier entrada directa adicional, no regular o de tipo incierto.
package orquesta_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"syscall"
	"testing"
)

const (
	physicalMappingApp     = "scripts/legacy_physical_mapping"
	physicalMappingSubject = "7e85ac617f396023d76c1751f4122151bf45926d059972b4d67fe2bd00bb2a71"
)

var physicalMappingFiles = []string{
	"README.md", "base.go", "candidate.go", "candidate_mutation_test.go",
	"command_test.go", "json.go", "json_security_test.go", "main.go",
	"manifest_contract_test.go", "model.go", "ownership.go",
	"ownership_mutation_test.go", "test_support_test.go", "validation_test.go",
}

func TestPhysicalMappingAggregateRejectsNonRegularDirectEntries(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*testing.T, string)
	}{
		{"subdirectorio con código", func(t *testing.T, root string) {
			path := filepath.Join(root, "extra")
			if err := os.Mkdir(path, 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(path, "main.go"), []byte("package extra\n"), 0o600); err != nil {
				t.Fatal(err)
			}
		}},
		{"enlace simbólico", func(t *testing.T, root string) {
			path := filepath.Join(root, "base.go")
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink("candidate.go", path); err != nil {
				t.Fatal(err)
			}
		}},
		{"FIFO", func(t *testing.T, root string) {
			path := filepath.Join(root, "base.go")
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			if err := syscall.Mkfifo(path, 0o600); err != nil {
				t.Fatal(err)
			}
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := physicalMappingFixture(t)
			test.mutate(t, root)
			if _, err := physicalMappingAggregate(root); err == nil {
				t.Fatal("el agregado aceptó una entrada directa no regular o adicional")
			}
		})
	}
}

func physicalMappingAggregate(root string) (string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return "", err
	}
	byName := make(map[string]os.DirEntry, len(entries))
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
		byName[entry.Name()] = entry
	}
	sort.Strings(names)
	if !reflect.DeepEqual(names, physicalMappingFiles) {
		return "", fmt.Errorf("universo directo inesperado: %q", names)
	}
	var lines bytes.Buffer
	for _, name := range physicalMappingFiles {
		path := filepath.Join(root, name)
		lstat, err := os.Lstat(path)
		if err != nil {
			return "", err
		}
		info, err := byName[name].Info()
		if err != nil {
			return "", err
		}
		if !lstat.Mode().IsRegular() || !info.Mode().IsRegular() || !os.SameFile(lstat, info) {
			return "", fmt.Errorf("entrada directa no regular o incierta: %s", name)
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		sum := sha256.Sum256(content)
		logicalPath := filepath.ToSlash(filepath.Join(physicalMappingApp, name))
		_, _ = fmt.Fprintf(&lines, "%s  %s\n", hex.EncodeToString(sum[:]), logicalPath)
	}
	aggregate := sha256.Sum256(lines.Bytes())
	return hex.EncodeToString(aggregate[:]), nil
}

func physicalMappingFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, name := range physicalMappingFiles {
		content, err := os.ReadFile(filepath.Join(physicalMappingApp, name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, name), content, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return root
}
