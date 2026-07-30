// Este fichero caracteriza aliases físicos con un lector falso, sin fingir un bind real.
package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestPhysicalOutputAliasIsRejectedBeforeAnyEffect(t *testing.T) {
	root := t.TempDir()
	opts := testOptions(t, root, modeMetadata)
	output := filepath.Dir(opts.jsonlPath)
	opts.locationReader = fakeLocations(map[string]physicalLocation{
		root:   {device: 7, path: "/datos/legacy"},
		output: {device: 7, path: "/datos/legacy/privado"},
	})
	if _, err := newPublicationForTest(opts); !errors.Is(err, errPhysicalOverlap) {
		t.Fatalf("alias físico de salida aceptado: %v", err)
	}
	entries, err := os.ReadDir(output)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("se produjeron efectos antes de separar: %v", entries)
	}
	opts = testOptions(t, t.TempDir(), modeMetadata)
	opts.locationReader = nil
	if _, err := newPublicationForTest(opts); !errors.Is(err, errPhysicalOverlap) {
		t.Fatalf("mismo dispositivo disjunto aceptado: %v", err)
	}
	entries, err = os.ReadDir(filepath.Dir(opts.jsonlPath))
	if err != nil || len(entries) != 0 {
		t.Fatalf("el rechazo por dispositivo produjo efectos: %v / %v", entries, err)
	}
}
func TestPhysicalRootDuplicatesAndOverlapsAreRejected(t *testing.T) {
	for name, secondPath := range map[string]string{
		"duplicada": "/datos/legacy",
		"contenida": "/datos/legacy/subarbol",
	} {
		t.Run(name, func(t *testing.T) {
			first, second := t.TempDir(), t.TempDir()
			opts := testOptions(t, first, modeMetadata)
			opts.roots = append(opts.roots, rootOption{alias: "segunda", mode: modeMetadata, path: second})
			output := filepath.Dir(opts.jsonlPath)
			opts.locationReader = fakeLocations(map[string]physicalLocation{
				first:  {device: 7, path: "/datos/legacy"},
				second: {device: 7, path: secondPath},
				output: {device: 8, path: "/evidencia"},
			})
			if _, err := newPublicationForTest(opts); !errors.Is(err, errPhysicalOverlap) {
				t.Fatalf("raíces físicas solapadas aceptadas: %v", err)
			}
		})
	}
}
func TestMovedRootCannotHidePhysicalOutputBeforeFirstEffect(t *testing.T) {
	base := t.TempDir()
	source := filepath.Join(base, "source")
	evidence := filepath.Join(source, "evidence")
	if err := os.MkdirAll(evidence, 0o700); err != nil {
		t.Fatal(err)
	}
	moved := filepath.Join(base, "moved")
	opts := testOptions(t, source, modeMetadata)
	opts.locationReader = nil
	opts.jsonlPath = filepath.Join(moved, "evidence", "censo.jsonl")
	opts.manifestPath = filepath.Join(moved, "evidence", "censo.manifest.json")
	root, err := anchorRoot(opts.roots[0])
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(source, moved); err != nil {
		t.Fatal(err)
	}
	_, publicationErr := newPublication(opts, []anchoredRoot{root})
	closeErr := root.file.Close()
	if !errors.Is(publicationErr, errPhysicalOverlap) || closeErr != nil {
		t.Fatalf("el movimiento ocultó el solapamiento: %v / %v", publicationErr, closeErr)
	}
	entries, err := os.ReadDir(filepath.Join(moved, "evidence"))
	if err != nil || len(entries) != 0 {
		t.Fatalf("el solapamiento produjo efectos: %v / %v", entries, err)
	}
}
func fakeLocations(values map[string]physicalLocation) physicalLocationReader {
	return func(_ int, lexicalPath string) (physicalLocation, error) {
		value, ok := values[lexicalPath]
		if !ok {
			return physicalLocation{}, errors.New("ubicacion_falsa_ausente")
		}
		return value, nil
	}
}
