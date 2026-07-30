// Este fichero acredita que el recorrido físico no abre ni atraviesa objetos hostiles.
package main

import (
	"bytes"
	"golang.org/x/sys/unix"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMetadataOnlyDoesNotFollowOrOpenFilesystemObjects(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	mustWrite(t, filepath.Join(outside, "no-leer.txt"), "SECRETO-INCONFUNDIBLE", 0o600)
	closed := filepath.Join(root, "cerrado.txt")
	mustWrite(t, closed, "contenido cerrado", 0o000)
	defer os.Chmod(closed, 0o600)
	if err := os.Symlink(filepath.Join(outside, "no-leer.txt"), filepath.Join(root, "hostil")); err != nil {
		t.Fatal(err)
	}
	if err := unix.Mkfifo(filepath.Join(root, "canal"), 0o600); err != nil {
		t.Fatal(err)
	}
	socketPath := filepath.Join(root, "socket")
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	nonUTF8 := string([]byte{'n', 0xff})
	mustWrite(t, filepath.Join(root, nonUTF8), "x", 0o600)
	if err := os.Mkdir(filepath.Join(root, "sensible"), 0o700); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(root, "sensible", "clave"), "OTRO-SECRETO", 0o600)
	opts := testOptions(t, root, modeMetadata)
	opts.roots[0].denied = [][]string{{"sensible"}}
	if err := run(opts); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(opts.jsonlPath)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(raw, []byte("SECRETO-INCONFUNDIBLE")) ||
		bytes.Contains(raw, []byte("OTRO-SECRETO")) ||
		bytes.Contains(raw, []byte(outside)) {
		t.Fatal("la salida reveló contenido o destino sensible")
	}
	records := readRecords(t, opts.jsonlPath)
	regular, ok := findRecord(records, "file", "cerrado.txt")
	if !ok || regular.ContentState != "not_requested" || regular.ContentSHA256 != "" {
		t.Fatalf("el regular fue abierto o quedó mal clasificado: %#v", regular)
	}
	link, ok := findRecord(records, "symlink", "hostil")
	if !ok || link.ContentSHA256 != "" {
		t.Fatalf("el enlace no quedó registrado sin lectura: %#v", link)
	}
	fifo, ok := findRecord(records, "special", "canal")
	if !ok || fifo.SpecialType != "fifo" {
		t.Fatalf("FIFO ausente o incorrecto: %#v", fifo)
	}
	socket, ok := findRecord(records, "special", "socket")
	if !ok || socket.SpecialType != "socket" {
		t.Fatalf("socket ausente o incorrecto: %#v", socket)
	}
	if _, ok := findRecord(records, "exclusion", "sensible"); !ok {
		t.Fatal("no se registró el subárbol sensible")
	}
	for _, value := range records {
		if len(value.Path) == 1 && value.Path[0].Encoding == "base64" {
			return
		}
	}
	t.Fatal("el nombre no UTF-8 no se codificó en base64")
}
func TestDirectoryMutationAfterListingDowngradesCompleteness(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "inicial"), "inicial", 0o600)
	opts := testOptions(t, root, modeMetadata)
	changed := false
	opts.afterReadDirectory = func(_ string, parts []string) {
		if changed || len(parts) != 0 {
			return
		}
		changed = true
		mustWrite(t, filepath.Join(root, "tardio"), "tardio", 0o600)
	}
	if err := run(opts); err == nil {
		t.Fatal("la mutación del directorio se presentó como completa")
	}
	value := readManifest(t, opts.manifestPath)
	if value.Complete {
		t.Fatal("el manifiesto no rebajó completitud")
	}
	errorRecord, ok := findRecord(readRecords(t, opts.jsonlPath), "error")
	if !ok || errorRecord.ErrorCode != "changed_during_scan" {
		t.Fatalf("diagnóstico de mutación ausente: %#v", errorRecord)
	}
}
func TestMetadataEntryMutationIsDetectedWithoutDirectoryChange(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "dato")
	mustWrite(t, path, "a", 0o600)
	opts := testOptions(t, root, modeMetadata)
	changed := false
	opts.beforeVerifyEntry = func(_ string, parts []string) {
		if changed || len(parts) != 1 || parts[0] != "dato" {
			return
		}
		changed = true
		mustWrite(t, path, "contenido distinto", 0o600)
	}
	if err := run(opts); err == nil {
		t.Fatal("la mutación del fichero se presentó como completa")
	}
	value, ok := findRecord(readRecords(t, opts.jsonlPath), "file", "dato")
	if !ok || !value.ChangedDuringScan {
		t.Fatalf("mutación de metadatos no detectada: %#v", value)
	}
}
func TestMountIDMismatchIsExcludedBeforeMetadataClassification(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "montado"), "dato", 0o600)
	opts := testOptions(t, root, modeMetadata)
	opts.mountIDReader = func(_ int, name string, _ int) (uint64, error) {
		if name == "montado" {
			return ^uint64(0), nil
		}
		return mountIDAt(-1, name, 0)
	}
	if err := run(opts); err != nil {
		t.Fatal(err)
	}
	records := readRecords(t, opts.jsonlPath)
	exclusion, ok := findRecord(records, "exclusion", "montado")
	if !ok || exclusion.Reason != "mount_boundary" {
		t.Fatalf("montaje no excluido: %#v", exclusion)
	}
	if _, ok := findRecord(records, "file", "montado"); ok {
		t.Fatal("el montaje se acreditó como fichero ordinario")
	}
}
func TestDepthAndPathBudgetsBoundTraversal(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "uno", "dos"), 0o700); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(root, "uno", "dos", "dato"), "dato", 0o600)
	depth := testOptions(t, root, modeMetadata)
	depth.budget.maxDepth = 1
	if err := run(depth); err == nil {
		t.Fatal("el límite de profundidad se presentó como completo")
	}
	records := readRecords(t, depth.jsonlPath)
	if value, ok := findRecord(records, "exclusion", "uno"); !ok ||
		value.Reason != "depth_limit" || value.RejectedSegment == nil {
		t.Fatalf("límite de profundidad ausente: %#v", value)
	}
	pathBudget := testOptions(t, root, modeMetadata)
	pathBudget.budget.maxPathBytes = 2
	if err := run(pathBudget); err == nil {
		t.Fatal("el límite de ruta se presentó como completo")
	}
	value, ok := findRecord(readRecords(t, pathBudget.jsonlPath), "exclusion")
	if !ok || value.Reason != "path_bytes_limit" {
		t.Fatalf("límite de bytes de ruta ausente: %#v", value)
	}
}
func TestNoAtimeReadingKeepsDirectoryFileAndSymlinkTimes(t *testing.T) {
	root := t.TempDir()
	filePath := filepath.Join(root, "dato")
	linkPath := filepath.Join(root, "enlace")
	mustWrite(t, filePath, "dato", 0o600)
	if err := os.Symlink("dato", linkPath); err != nil {
		t.Fatal(err)
	}
	old := unix.NsecToTimespec(time.Unix(1, 0).UnixNano())
	omit := unix.Timespec{Nsec: unix.UTIME_OMIT}
	for _, item := range []struct {
		path  string
		flags int
	}{{root, 0}, {filePath, 0}, {linkPath, unix.AT_SYMLINK_NOFOLLOW}} {
		if err := unix.UtimesNanoAt(unix.AT_FDCWD, item.path, []unix.Timespec{old, omit}, item.flags); err != nil {
			t.Fatal(err)
		}
	}
	before := atimes(t, root, filePath, linkPath)
	opts := testOptions(t, root, modeContent)
	if err := run(opts); err != nil {
		t.Fatal(err)
	}
	after := atimes(t, root, filePath, linkPath)
	for index := range before {
		if before[index] != after[index] {
			t.Fatalf("atime mutado en índice %d: %v -> %v", index, before[index], after[index])
		}
	}
}
func atimes(t *testing.T, paths ...string) []unix.Timespec {
	t.Helper()
	result := make([]unix.Timespec, len(paths))
	for index, path := range paths {
		var stat unix.Stat_t
		if err := unix.Lstat(path, &stat); err != nil {
			t.Fatal(err)
		}
		result[index] = stat.Atim
	}
	return result
}
func TestDirectoryBudgetPublishesRecoverableIncompleteResult(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"a", "b", "c"} {
		mustWrite(t, filepath.Join(root, name), name, 0o600)
	}
	exact := testOptions(t, root, modeMetadata)
	exact.budget.maxDirectoryEntries = 3
	mustSucceed(t, run(exact))
	exactManifest := readManifest(t, exact.manifestPath)
	if !exactManifest.Complete || exactManifest.Roots[0].Counts["file"] != 3 {
		t.Fatalf("el límite exacto N no recorrió las tres entradas: %#v", exactManifest)
	}
	opts := testOptions(t, root, modeMetadata)
	opts.budget.maxDirectoryEntries = 2
	if err := run(opts); err == nil {
		t.Fatal("el exceso del directorio se presentó como completo")
	}
	value := readManifest(t, opts.manifestPath)
	if value.Complete {
		t.Fatal("el manifiesto parcial figura completo")
	}
	if value.Roots[0].Counts["file"] != 2 {
		t.Fatalf("se procesaron %d entradas; se esperaban N=2 y no N+1", value.Roots[0].Counts["file"])
	}
	records := readRecords(t, opts.jsonlPath)
	terminal, ok := findRecord(records, "error")
	if !ok || terminal.ErrorCode != "directory_budget_exhausted" {
		t.Fatalf("diagnóstico terminal ausente: %#v", terminal)
	}
}
func TestDirectoryReplacementBetweenObservationAndOpenIsRejected(t *testing.T) {
	root := t.TempDir()
	original := filepath.Join(root, "directorio")
	if err := os.Mkdir(original, 0o700); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(original, "original"), "original", 0o600)
	opts := testOptions(t, root, modeMetadata)
	replaced := false
	opts.beforeOpenDirectory = func(_ string, parts []string) {
		if replaced || len(parts) != 1 || parts[0] != "directorio" {
			return
		}
		replaced = true
		if err := os.Rename(original, filepath.Join(root, "retirado")); err != nil {
			t.Fatal(err)
		}
		if err := os.Mkdir(original, 0o700); err != nil {
			t.Fatal(err)
		}
		mustWrite(t, filepath.Join(original, "sustituto"), "sustituto", 0o600)
	}
	if err := run(opts); err == nil {
		t.Fatal("la sustitución de directorio se presentó como completa")
	}
	records := readRecords(t, opts.jsonlPath)
	value, ok := findRecord(records, "error", "directorio")
	if !ok || value.ErrorCode != "changed_during_scan" {
		t.Fatalf("sustitución no detectada: %#v", value)
	}
	if _, ok := findRecord(records, "file", "directorio", "sustituto"); ok {
		t.Fatal("se atravesó el directorio sustituto")
	}
}
