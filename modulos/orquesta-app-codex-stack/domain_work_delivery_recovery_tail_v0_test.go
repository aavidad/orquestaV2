package orquestaappcodexstack

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
)

func TestDomainWorkRecoveryFileContainsAnyV0LeeTailAcotado(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, orquestaruntimecodex.CodexStderrFileNameV0)
	if err := os.WriteFile(
		path,
		[]byte("prefijo sin senal\n"+strings.Repeat("padding\n", 10*1024)+"turn interrupted\n"),
		0o600,
	); err != nil {
		t.Fatalf("write stderr: %v", err)
	}

	if !domainWorkRecoveryFileContainsAnyV0(path, []string{"turn interrupted"}) {
		t.Fatalf("tail no detecto interrupcion")
	}
	if domainWorkRecoveryFileContainsAnyV0(path, []string{"prefijo sin senal"}) {
		t.Fatalf("lector no debe clasificar por prefijo fuera del tail")
	}
}
