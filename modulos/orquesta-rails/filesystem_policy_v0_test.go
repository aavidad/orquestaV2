package orquestarails

import (
	"path/filepath"
	"testing"
)

func TestWorkspaceRelativePathAllowedV0AceptaRutasRelativasSeguras(t *testing.T) {
	for _, path := range []string{
		"README.md",
		"modulos/orquesta-rails/filesystem_policy_v0.go",
		"cmd/orquesta-server",
	} {
		if !WorkspaceRelativePathAllowedV0(path, false) {
			t.Fatalf("ruta relativa segura rechazada: %q", path)
		}
	}
}

func TestWorkspaceRelativePathAllowedV0RaizSoloSiSePermite(t *testing.T) {
	if !WorkspaceRelativePathAllowedV0(".", true) {
		t.Fatal("root write-set deberia permitirse con allowRoot")
	}
	if WorkspaceRelativePathAllowedV0(".", false) {
		t.Fatal("root write-set no debe permitirse sin allowRoot")
	}
}

func TestWorkspaceRelativePathAllowedV0RechazaSalidaDelWorkspace(t *testing.T) {
	for _, path := range []string{
		"",
		"/tmp/fuera",
		"../fuera",
		"modulos/../fuera",
		`C:\Users\operador\repo`,
		`modulos\orquesta`,
		"~/repo",
		"$HOME/repo",
		"https://example.test/repo",
		"ok\nmal",
	} {
		if WorkspaceRelativePathAllowedV0(path, true) {
			t.Fatalf("ruta insegura aceptada: %q", path)
		}
	}
}

func TestPathInsideWorkspaceV0(t *testing.T) {
	root := t.TempDir()
	inside := filepath.Join(root, "modulos", "orquesta")
	outside := filepath.Join(root, "..", "fuera")
	if !PathInsideWorkspaceV0(root, inside) {
		t.Fatalf("inside rechazado: %s", inside)
	}
	if PathInsideWorkspaceV0(root, outside) {
		t.Fatalf("outside aceptado: %s", outside)
	}
}

func TestCommandTextContainsDestructiveFilesystemOperationV0(t *testing.T) {
	for _, command := range []string{"rm -rf docs", "rmdir build", "os.RemoveAll(path)"} {
		if !CommandTextContainsDestructiveFilesystemOperationV0(command) {
			t.Fatalf("operacion destructiva no detectada: %q", command)
		}
	}
	for _, command := range []string{"go test ./...", "git status --short", "mkdir -p docs"} {
		if CommandTextContainsDestructiveFilesystemOperationV0(command) {
			t.Fatalf("comando no destructivo marcado: %q", command)
		}
	}
}
