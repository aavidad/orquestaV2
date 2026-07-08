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

func TestBuildCodexAgentPromptV0MuestraSkillRefsSolicitadas(t *testing.T) {
	packet := codexPacketForTestV0()
	packet.Task.SkillRefs = []string{
		"skill-ref-orquesta-programacion-v0",
		"skill-ref-orquesta-programacion-v0",
		"skill-ref-orquesta-revision-v0",
	}

	prompt := BuildCodexAgentPromptV0(packet, nil)

	for _, want := range []string{
		"SkillRefs solicitadas:",
		"- skill-ref-orquesta-programacion-v0",
		"- skill-ref-orquesta-revision-v0",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt no contiene %q:\n%s", want, prompt)
		}
	}
	section := promptSectionForTestV0(prompt, "SkillRefs solicitadas:", "Skills materializadas:")
	if strings.Count(section, "- skill-ref-orquesta-programacion-v0") != 1 {
		t.Fatalf("prompt no compacta skill_refs duplicadas:\n%s", section)
	}
	if strings.Contains(prompt, "SKILL.md") || strings.Contains(prompt, "skills/") {
		t.Fatalf("prompt no debe resolver skill_refs a rutas:\n%s", prompt)
	}
}

func TestBuildCodexAgentPromptV0MaterializaSkillsProgramacionCompactas(t *testing.T) {
	packet := codexPacketForTestV0()
	packet.Task.SkillRefs = []string{
		"skill-ref-orquesta-programacion-autonoma-v0",
		"skill-ref-orquesta-programacion-minima-v0",
		"skill-ref-orquesta-programacion-integracion-v0",
	}

	prompt := BuildCodexAgentPromptV0(packet, nil)

	for _, want := range []string{
		"Skills materializadas:",
		"aplica las skill_refs materializadas como reglas de ejecucion",
		"gana el contrato del paquete",
		"Lee AGENTS local",
		"Diff minimo",
		"sin necesidad demostrada",
		"Mantén hexagonal",
		"adaptador opt-in",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt no materializa skill compacta %q:\n%s", want, prompt)
		}
	}
}

func TestBuildCodexAgentPromptV0SkillProgramacionMinimaOverheadAcotado(t *testing.T) {
	base := codexPacketForTestV0()
	base.Task.SkillRefs = []string{
		"skill-ref-orquesta-programacion-autonoma-v0",
		"skill-ref-orquesta-programacion-integracion-v0",
	}
	withMinimal := codexPacketForTestV0()
	withMinimal.Task.SkillRefs = []string{
		"skill-ref-orquesta-programacion-autonoma-v0",
		"skill-ref-orquesta-programacion-minima-v0",
		"skill-ref-orquesta-programacion-integracion-v0",
	}

	basePrompt := BuildCodexAgentPromptV0(base, nil)
	minimalPrompt := BuildCodexAgentPromptV0(withMinimal, nil)
	overheadBytes := len(minimalPrompt) - len(basePrompt)
	t.Logf("programacion_minima_prompt_overhead_bytes=%d", overheadBytes)
	if overheadBytes <= 0 || overheadBytes > 512 {
		t.Fatalf("overhead skill programacion minima fuera de rango: %d bytes", overheadBytes)
	}
	if !strings.Contains(minimalPrompt, "skill-ref-orquesta-programacion-minima-v0") {
		t.Fatalf("prompt no incluye skill minima:\n%s", minimalPrompt)
	}
	if strings.Count(minimalPrompt, "skill-ref-orquesta-programacion-minima-v0") != 2 {
		t.Fatalf("prompt duplica o pierde skill minima:\n%s", minimalPrompt)
	}
}

func TestBuildCodexAgentPromptV0MaterializaSkillRefsRolesSeparadas(t *testing.T) {
	packet := codexPacketForTestV0()
	packet.Task.SkillRefs = []string{
		"skill-ref-orquesta-programacion-web-app-v0",
		"skill-ref-orquesta-programacion-api-rest-v0",
		"skill-ref-orquesta-programacion-mcp-v0",
		"skill-ref-orquesta-programacion-datos-persistencia-v0",
		"skill-ref-orquesta-opes-dominio-v0",
		"skill-ref-opes-protocolo-agentes-compactos-v0",
		"skill-ref-opes-tests-4-respuestas-v0",
		"skill-ref-opes-html-web-uso-v0",
		"skill-ref-opes-paquete-api-produccion-v0",
		"skill-ref-opes-qa-final-curso-v0",
	}

	prompt := BuildCodexAgentPromptV0(packet, nil)

	for _, want := range []string{
		"- skill-ref-orquesta-programacion-web-app-v0: Web app/UI",
		"- skill-ref-orquesta-programacion-api-rest-v0: API REST",
		"- skill-ref-orquesta-programacion-mcp-v0: MCP",
		"- skill-ref-orquesta-programacion-datos-persistencia-v0: Datos/persistencia",
		"- skill-ref-orquesta-opes-dominio-v0: OPES queda como dominio/adaptador consumidor",
		"cliente fino sobre API/MCP",
		"adaptador fino sobre puertos",
		"sin DB por defecto en Orquesta",
		"sin filtros por palabras",
		"- skill-ref-opes-protocolo-agentes-compactos-v0: OPES compacto",
		"- skill-ref-opes-tests-4-respuestas-v0: OPES tests",
		"respeta pregunta humana",
		"3 distractores utiles sin duplicidades",
		"- skill-ref-opes-html-web-uso-v0: OPES HTML USO/TCAE",
		"- skill-ref-opes-paquete-api-produccion-v0: OPES produccion",
		"subida por API con slug confirmado",
		"- skill-ref-opes-qa-final-curso-v0: OPES QA final",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt no materializa rol %q:\n%s", want, prompt)
		}
	}
	if strings.Contains(prompt, "SKILL.md") || strings.Contains(prompt, "skills/") {
		t.Fatalf("prompt no debe resolver skill_refs a rutas:\n%s", prompt)
	}
}

func TestBuildCodexAgentPromptV0MaterializaSkillRefsInyectadasPorComposicion(t *testing.T) {
	packet := codexPacketForTestV0()
	packet.Task.SkillRefs = []string{"skill-ref-catalogo-declarado-v0"}

	prompt := BuildCodexAgentPromptWithControlFilesV0(packet, nil, CodexControlFilesV0{
		PacketPath: CodexAgentPacketFileNameV0,
		AckPath:    CodexAgentAckFileNameV0,
		SkillInstructions: []CodexSkillInstructionV0{{
			SkillRef: "skill-ref-catalogo-declarado-v0",
			Text:     "Catalogo declarado: aplica workspace administrativo denso con filtros, tablas y estados semanticos.",
		}},
	})

	for _, want := range []string{
		"- skill-ref-catalogo-declarado-v0: Catalogo declarado",
		"workspace administrativo denso",
		"SkillRefs solicitadas:",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt no materializa skill inyectada %q:\n%s", want, prompt)
		}
	}
	if strings.Contains(prompt, "SKILL.md") || strings.Contains(prompt, "skills/") {
		t.Fatalf("prompt no debe resolver skill_refs a rutas:\n%s", prompt)
	}
}

func promptSectionForTestV0(prompt string, start string, end string) string {
	_, after, ok := strings.Cut(prompt, start)
	if !ok {
		return prompt
	}
	beforeEnd, _, ok := strings.Cut(after, end)
	if !ok {
		return after
	}
	return beforeEnd
}
