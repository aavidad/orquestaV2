// Este fichero prueba el resumen por bloques, la mutación y la reproducibilidad.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSourceContentHashesRegularFileAndDetectsMutation(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "fuente.go")
	mustWrite(t, path, "package ejemplo\n", 0o640)
	stable := testOptions(t, root, modeContent)
	if err := run(stable); err != nil {
		t.Fatal(err)
	}
	value, ok := findRecord(readRecords(t, stable.jsonlPath), "file", "fuente.go")
	sum := sha256.Sum256([]byte("package ejemplo\n"))
	if !ok || value.ContentState != "hashed" ||
		value.ContentSHA256 != hex.EncodeToString(sum[:]) {
		t.Fatalf("resumen inesperado: %#v", value)
	}
	if value.Size == nil || *value.Size != int64(len("package ejemplo\n")) ||
		value.UID == nil || value.GID == nil || value.FileMode == nil {
		t.Fatalf("metadatos cero/numéricos no preservados: %#v", value)
	}
	changing := testOptions(t, root, modeContent)
	changing.afterRead = func(_ string, _ []string) {
		mustWrite(t, path, "package cambiado\n", 0o640)
	}
	if err := run(changing); err == nil {
		t.Fatal("la mutación se presentó como censo completo")
	}
	changed, ok := findRecord(readRecords(t, changing.jsonlPath), "file", "fuente.go")
	if !ok || !changed.ChangedDuringScan || changed.ContentState != "changed" ||
		changed.ContentSHA256 != "" {
		t.Fatalf("mutación no detectada: %#v", changed)
	}
}
func TestHashChecksTimeoutBetweenBoundedBlocks(t *testing.T) {
	path := filepath.Join(t.TempDir(), "grande")
	mustWrite(t, path, string(make([]byte, 256*1024)), 0o600)
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	started := time.Unix(10, 0)
	calls := 0
	state := scanState{
		started: started,
		budget:  budgetOptions{timeout: time.Second},
		now: func() time.Time {
			calls++
			if calls == 1 {
				return started
			}
			return started.Add(2 * time.Second)
		},
	}
	count, err := state.hashBlocks(file, sha256.New(), 256*1024)
	if !errors.Is(err, errBudget) || count != 128*1024 {
		t.Fatalf("cancelación cooperativa inesperada: bytes=%d error=%v", count, err)
	}
}
func TestEntryBudgetPreservesTerminalDiagnostic(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "a"), "a", 0o600)
	mustWrite(t, filepath.Join(root, "b"), "b", 0o600)
	opts := testOptions(t, root, modeContent)
	opts.budget.maxEntries = 3
	reads := 0
	opts.afterRead = func(_ string, _ []string) { reads++ }
	if err := run(opts); err == nil {
		t.Fatal("el presupuesto agotado se presentó como completo")
	}
	records := readRecords(t, opts.jsonlPath)
	if int64(len(records)) != opts.budget.maxEntries {
		t.Fatalf("entradas=%d, se esperaban %d", len(records), opts.budget.maxEntries)
	}
	last := records[len(records)-1]
	if last.Kind != "error" || last.ErrorCode != "budget_exhausted" {
		t.Fatalf("diagnóstico final inesperado: %#v", last)
	}
	if reads != 1 {
		t.Fatalf("el presupuesto abrió o resumió %d entradas; se esperaba una", reads)
	}
}
func TestContentSizeBudgetsSkipWithoutOpeningBeyondAllowance(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "a"), "1234", 0o600)
	mustWrite(t, filepath.Join(root, "b"), "5678", 0o600)
	opts := testOptions(t, root, modeContent)
	opts.budget.maxFileBytes = 4
	opts.budget.maxHashBytes = 5
	if err := run(opts); err == nil {
		t.Fatal("el límite global se presentó como contenido completo")
	}
	records := readRecords(t, opts.jsonlPath)
	first, firstOK := findRecord(records, "file", "a")
	second, secondOK := findRecord(records, "file", "b")
	if !firstOK || first.ContentState != "hashed" ||
		!secondOK || second.ContentState != "skipped_global_limit" {
		t.Fatalf("límite global mal aplicado: %#v / %#v", first, second)
	}
	perFile := testOptions(t, root, modeContent)
	perFile.budget.maxFileBytes = 3
	if err := run(perFile); err == nil {
		t.Fatal("el límite por fichero se presentó como contenido completo")
	}
	value, ok := findRecord(readRecords(t, perFile.jsonlPath), "file", "a")
	if !ok || value.ContentState != "skipped_file_limit" || value.ContentSHA256 != "" {
		t.Fatalf("límite por fichero mal aplicado: %#v", value)
	}
}
func TestOutputBudgetPublishesBoundedPartialGeneration(t *testing.T) {
	root := t.TempDir()
	for index := 0; index < 100; index++ {
		mustWrite(t, filepath.Join(root, fmt.Sprintf("fichero-%03d", index)), "x", 0o600)
	}
	opts := testOptions(t, root, modeMetadata)
	opts.budget.maxOutputBytes = terminalOutputReserve + 900
	if err := run(opts); err == nil {
		t.Fatal("el flujo truncado se presentó como completo")
	}
	info, err := os.Stat(opts.jsonlPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() > opts.budget.maxOutputBytes {
		t.Fatalf("salida=%d, presupuesto=%d", info.Size(), opts.budget.maxOutputBytes)
	}
	records := readRecords(t, opts.jsonlPath)
	last := records[len(records)-1]
	if last.Kind != "error" || last.ErrorCode != "output_budget_exhausted" {
		t.Fatalf("diagnóstico terminal ausente: %#v", last)
	}
	if readManifest(t, opts.manifestPath).Complete {
		t.Fatal("el manifiesto acotado figura completo")
	}
}
func TestStableTreeProducesIdenticalArtifacts(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "directorio"), 0o750); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(root, "zeta"), "z", 0o600)
	mustWrite(t, filepath.Join(root, "directorio", "alfa"), "a", 0o640)
	first := testOptions(t, root, modeContent)
	second := testOptions(t, root, modeContent)
	if err := run(first); err != nil {
		t.Fatal(err)
	}
	if err := run(second); err != nil {
		t.Fatal(err)
	}
	firstJSONL, _ := os.ReadFile(first.jsonlPath)
	secondJSONL, _ := os.ReadFile(second.jsonlPath)
	firstManifest, _ := os.ReadFile(first.manifestPath)
	secondManifest, _ := os.ReadFile(second.manifestPath)
	if !bytes.Equal(firstJSONL, secondJSONL) {
		t.Fatal("el JSONL estable no fue determinista")
	}
	if !bytes.Equal(firstManifest, secondManifest) {
		t.Fatal("el manifiesto estable no fue determinista")
	}
}
