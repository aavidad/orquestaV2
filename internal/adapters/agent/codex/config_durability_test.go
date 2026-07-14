package codex

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestEnsureWorkRootPathCreatesEachComponentAndSyncsItsParent(t *testing.T) {
	base := t.TempDir()
	one := filepath.Join(base, "one")
	two := filepath.Join(one, "two")
	workRoot := filepath.Join(two, "work")
	var synced []string
	if err := ensureWorkRootPath(workRoot, func(directory string) error {
		synced = append(synced, directory)
		return syncFilesystemDirectory(directory)
	}); err != nil {
		t.Fatalf("ensureWorkRootPath() error = %v", err)
	}
	if want := []string{base, one, two}; !reflect.DeepEqual(synced, want) {
		t.Fatalf("synced parents = %#v, want %#v", synced, want)
	}
	for _, directory := range []string{one, two, workRoot} {
		info, err := os.Lstat(directory)
		if err != nil || !info.IsDir() || info.Mode().Perm() != 0o700 {
			t.Fatalf("directory %s info=%v error=%v", directory, info, err)
		}
	}
}
