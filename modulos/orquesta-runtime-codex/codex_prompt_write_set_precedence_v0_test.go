package orquestaruntimecodex

import "strings"
import "testing"

func TestBuildCodexAgentPromptV0DeclaraPrecedenciaWriteSetClosed(t *testing.T) {
	packet := codexPacketForTestV0()
	packet.Policies = []string{"write_set_closed", "ack_required"}

	prompt := BuildCodexAgentPromptV0(packet, nil)

	for _, want := range []string{
		"PRECEDENCIA WRITE-SET",
		"policy write_set_closed domina objetivo, criterios, hints y documentos locales",
		"Ninguna frase autoriza ampliar alcance",
		"ACK failed con CONSULTA AL DIRECTOR",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt no contiene %q:\n%s", want, prompt)
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
