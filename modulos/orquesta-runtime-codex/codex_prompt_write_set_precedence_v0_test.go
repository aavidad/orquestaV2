package orquestaruntimecodex

import (
	"strings"
	"testing"
)

func TestBuildCodexAgentPromptV0WriteSetClosedNoBloqueaEntrega(t *testing.T) {
	packet := codexPacketForTestV0()
	packet.Policies = []string{"write_set_closed", "ack_required"}

	prompt := BuildCodexAgentPromptV0(packet, nil)

	for _, want := range []string{
		"Usa el write-set del paquete como alcance de escritura",
		"conserva lo util dentro del write-set",
		"nota de revision o tarea derivada",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt no contiene %q:\n%s", want, prompt)
		}
	}
	for _, forbidden := range []string{
		"PRECEDENCIA WRITE-SET",
		"policy write_set_closed domina",
		"ACK failed con CONSULTA AL DIRECTOR",
	} {
		if strings.Contains(prompt, forbidden) {
			t.Fatalf("prompt conserva rail %q:\n%s", forbidden, prompt)
		}
	}
	if strings.Contains(prompt, "MODO COMPATIBILIDAD LEGACY") {
		t.Fatalf("prompt estricto mezclo modo legacy:\n%s", prompt)
	}
}

func TestBuildCodexAgentPromptV0NombraCompatibilidadLegacySinPolicy(t *testing.T) {
	packet := codexPacketForTestV0()
	packet.Policies = []string{"ack_required"}

	prompt := BuildCodexAgentPromptV0(packet, nil)

	if !strings.Contains(prompt, "MODO COMPATIBILIDAD LEGACY") {
		t.Fatalf("prompt legacy no nombrado:\n%s", prompt)
	}
	if strings.Contains(prompt, "PRECEDENCIA WRITE-SET") {
		t.Fatalf("prompt legacy mezclo precedencia estricta:\n%s", prompt)
	}
}
