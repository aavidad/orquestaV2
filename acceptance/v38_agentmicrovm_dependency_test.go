package acceptance

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

const (
	v38B10Module  = "github.com/aavidad/agente_microvm/conectores/orquesta"
	v38B10Version = "v0.0.0-20260814005716-2768389c82c0"
)

func TestV38B10PinsPublishedAgentMicroVMConnectorWithoutLocalReplace(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Dir(filepath.Dir(file))
	goMod := v38B10Read(t, filepath.Join(root, "go.mod"))
	modules := v38B10Read(t, filepath.Join(root, "vendor", "modules.txt"))

	exact := v38B10Module + " " + v38B10Version
	if strings.Count(goMod, exact) != 1 || strings.Count(modules, "# "+exact) != 1 {
		t.Fatalf("B10 debe fijar exactamente el módulo publicado %q", exact)
	}
	for _, line := range strings.Split(goMod, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "replace ") && strings.Contains(line, v38B10Module) {
			t.Fatal("B10 no admite replace local del conector publicado")
		}
	}
	vendorRoot := filepath.Join(root, "vendor", filepath.FromSlash(v38B10Module))
	for _, name := range []string{"cliente.go", "concesion.go", "contrato.go", "perfil.go"} {
		if info, err := os.Stat(filepath.Join(vendorRoot, name)); err != nil || !info.Mode().IsRegular() {
			t.Fatalf("módulo vendorizado incompleto en %s: %v", name, err)
		}
	}
}

func v38B10Read(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(content)
}
