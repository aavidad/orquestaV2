// Estas pruebas fijan la normalización y deduplicación de rutas.
package main

import (
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

func TestNormalizedPathsAreSortedAndUnique(t *testing.T) {
	base := t.TempDir()
	values, err := normalizeUniquePaths([]string{
		filepath.Join(base, "b"),
		filepath.Join(base, "a", "..", "b"),
		filepath.Join(base, "a"),
	})
	if err != nil {
		t.Fatal(err)
	}
	expected := []string{filepath.Join(base, "a"), filepath.Join(base, "b")}
	sort.Strings(expected)
	if !reflect.DeepEqual(values, expected) {
		t.Fatalf("rutas inesperadas: obtenidas=%v esperadas=%v", values, expected)
	}
}
