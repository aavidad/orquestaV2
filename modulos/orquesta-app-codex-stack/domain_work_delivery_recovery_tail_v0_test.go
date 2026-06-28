package orquestaappcodexstack

import (
	"os"
	"path/filepath"
	"testing"

	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
)

func TestDomainWorkRecoveryV0NoDependeDeLogTailSiArtefactoEsValido(t *testing.T) {
	fixture := newDomainWorkRecoveryAckFixtureForTestV0(t, true)
	if err := os.WriteFile(
		filepath.Join(filepath.Dir(fixture.descriptor.AckPath), orquestaruntimecodex.CodexStderrFileNameV0),
		[]byte("salida compacta sin diagnostico de ACK"),
		0o600,
	); err != nil {
		t.Fatalf("write stderr: %v", err)
	}

	ack, ok := recoverDomainWorkAckV0(fixture.descriptor, fixture.task, fixture.record)

	if !ok || !stringInSetV0([]string(ack.Files), fixture.fileRef) {
		t.Fatalf("ack=%+v ok=%v", ack, ok)
	}
}
