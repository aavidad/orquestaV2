package main

import (
	"os"
	"path/filepath"
	"testing"

	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestaruntimecodexgoal "orquesta/modulos/orquesta-runtime-codex-goal"
)

func TestPrepareCodexGoalWriteSetV0NoFallaConFicheroExistenteV0(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "modulos", "orquesta-server")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	existing := filepath.Join(dir, "config_v0.go")
	if err := os.WriteFile(existing, []byte("package orquestaserver\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	backend := serverCodexAppServerGoalBackendV0{CWD: root}
	packet := orquestaruntimecodexgoal.CodexGoalStartPacketV0{
		WriteSet: []orquestagoal.GoalWriteScopeV0{
			{Path: "modulos/orquesta-server"},
			{Path: "modulos/orquesta-server/config_v0.go"},
		},
	}
	if err := backend.prepareCodexGoalWriteSetV0(packet); err != nil {
		t.Fatalf("prepare fallo con fichero existente en write-set: %v", err)
	}
	info, err := os.Stat(existing)
	if err != nil {
		t.Fatal(err)
	}
	if info.IsDir() {
		t.Fatal("el fichero existente del write-set se convirtio en directorio")
	}
}

func TestPrepareCodexGoalWriteSetV0NoCreaDirectorioParaRutaConExtensionV0(t *testing.T) {
	root := t.TempDir()
	backend := serverCodexAppServerGoalBackendV0{CWD: root}
	packet := orquestaruntimecodexgoal.CodexGoalStartPacketV0{
		WriteSet: []orquestagoal.GoalWriteScopeV0{
			{Path: "scripts/orquesta_metricas_deuda.sh"},
			{Path: "scripts"},
		},
	}
	if err := backend.prepareCodexGoalWriteSetV0(packet); err != nil {
		t.Fatalf("prepare fallo: %v", err)
	}
	if info, err := os.Stat(filepath.Join(root, "scripts", "orquesta_metricas_deuda.sh")); err == nil && info.IsDir() {
		t.Fatal("una ruta de fichero nuevo del write-set se creo como directorio")
	}
	info, err := os.Stat(filepath.Join(root, "scripts"))
	if err != nil || !info.IsDir() {
		t.Fatalf("la ruta de directorio del write-set no se creo: %v", err)
	}
}

func TestCodexAppServerWriteSetLooksLikeFileV0TrataExtensionesComoFicheroV0(t *testing.T) {
	cases := map[string]bool{
		"docs/informe.md":            true,
		"scripts/foo.sh":             true,
		"modulos/x/config_v0.go":     true,
		"docs/resultado.json":        true,
		"modulos/orquesta-server":    false,
		"generated-apps/demo":        false,
		"docs/paquete/":              false,
		"modulos/orquesta-estado-vivo": false,
	}
	for path, want := range cases {
		if got := codexAppServerWriteSetLooksLikeFileV0(path); got != want {
			t.Fatalf("codexAppServerWriteSetLooksLikeFileV0(%q) = %v, esperado %v", path, got, want)
		}
	}
}
