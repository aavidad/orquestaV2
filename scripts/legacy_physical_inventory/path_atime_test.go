// Este fichero acredita que validar rutas hostiles no lee enlaces ni altera su atime.
package main

import (
	"errors"
	"golang.org/x/sys/unix"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestOutputSymlinkInsideRootIsRejectedWithoutReadingLink(t *testing.T) {
	root := t.TempDir()
	link := filepath.Join(root, "salida")
	if err := os.Symlink(t.TempDir(), link); err != nil {
		t.Fatal(err)
	}
	setOldSymlinkAtime(t, link)
	before := atimes(t, link)[0]
	err := executeCommand([]string{
		"--root", "historia=" + root,
		"--jsonl", filepath.Join(link, "censo.jsonl"),
		"--manifest", filepath.Join(link, "censo.manifest.json"),
	}, &discardWriter{})
	if !errors.Is(err, errOutputPath) {
		t.Fatalf("salida dentro de raíz aceptada: %v", err)
	}
	if after := atimes(t, link)[0]; before != after {
		t.Fatalf("la validación leyó el enlace: %v -> %v", before, after)
	}
}
func TestSymbolicRootAncestorIsRejectedWithoutReadingLink(t *testing.T) {
	realParent := t.TempDir()
	if err := os.Mkdir(filepath.Join(realParent, "raiz"), 0o700); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(t.TempDir(), "ancestro")
	if err := os.Symlink(realParent, link); err != nil {
		t.Fatal(err)
	}
	setOldSymlinkAtime(t, link)
	before := atimes(t, link)[0]
	opts := testOptions(t, filepath.Join(link, "raiz"), modeMetadata)
	if err := run(opts); !errors.Is(err, errRootAnchor) {
		t.Fatalf("ancestro simbólico de raíz aceptado: %v", err)
	}
	if after := atimes(t, link)[0]; before != after {
		t.Fatalf("el anclaje leyó el ancestro: %v -> %v", before, after)
	}
}
func TestSymbolicOutputAncestorIsRejectedWithoutReadingLink(t *testing.T) {
	link := filepath.Join(t.TempDir(), "ancestro")
	if err := os.Symlink(t.TempDir(), link); err != nil {
		t.Fatal(err)
	}
	setOldSymlinkAtime(t, link)
	before := atimes(t, link)[0]
	opts := testOptions(t, t.TempDir(), modeMetadata)
	opts.jsonlPath = filepath.Join(link, "censo.jsonl")
	opts.manifestPath = filepath.Join(link, "censo.manifest.json")
	if err := run(opts); !errors.Is(err, errPublication) {
		t.Fatalf("ancestro simbólico de salida aceptado: %v", err)
	}
	if after := atimes(t, link)[0]; before != after {
		t.Fatalf("el anclaje de salida leyó el ancestro: %v -> %v", before, after)
	}
}
func setOldSymlinkAtime(t *testing.T, path string) {
	t.Helper()
	old := unix.NsecToTimespec(time.Unix(1, 0).UnixNano())
	omit := unix.Timespec{Nsec: unix.UTIME_OMIT}
	if err := unix.UtimesNanoAt(
		unix.AT_FDCWD, path, []unix.Timespec{old, omit}, unix.AT_SYMLINK_NOFOLLOW,
	); err != nil {
		t.Fatal(err)
	}
}

type discardWriter struct{}

func (*discardWriter) Write(value []byte) (int, error) { return len(value), nil }
