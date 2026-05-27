package orquestaruntimecodexdelivery

import (
	"os"
	"path/filepath"
	"testing"

	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
)

func TestCodexProgressReadTailV0RechazaSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target.log")
	path := filepath.Join(dir, orquestaruntimecodex.CodexStdoutFileNameV0)
	if err := os.WriteFile(target, []byte("linea\n"), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	if err := os.Symlink(target, path); err != nil {
		t.Skipf("symlink no disponible: %v", err)
	}

	if _, ok := codexProgressReadTailV0(path, 1024); ok {
		t.Fatalf("tail de symlink no debe leerse")
	}
}
