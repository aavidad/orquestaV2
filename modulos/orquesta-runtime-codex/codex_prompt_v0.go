package orquestaruntimecodex

import (
	"encoding/json"
	"path/filepath"
	"strings"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

type CodexControlFilesV0 struct {
	PacketPath   string
	AckPath      string
	DecisionPath string
}

func BuildCodexAgentPromptV0(packet orquestaruntime.AgentStartPacketV0, hints []string) string {
	return BuildCodexAgentPromptWithControlFilesV0(packet, hints, CodexControlFilesV0{
		PacketPath: CodexAgentPacketFileNameV0,
		AckPath:    CodexAgentAckFileNameV0,
	})
}

func BuildCodexAgentPromptWithControlFilesV0(
	packet orquestaruntime.AgentStartPacketV0,
	hints []string,
	control CodexControlFilesV0,
) string {
	packetPath := cleanControlPathV0(control.PacketPath, CodexAgentPacketFileNameV0)
	ackPath := cleanControlPathV0(control.AckPath, CodexAgentAckFileNameV0)
	decisionPath := cleanControlPathV0(control.DecisionPath, "")
	var b strings.Builder
	b.WriteString("Eres un agente externo gobernado por OrquestaV2.\n")
	b.WriteString("Lee ")
	b.WriteString(packetPath)
	b.WriteString(" antes de tocar archivos.\n")
	b.WriteString("Respeta estrictamente el write-set del paquete.\n")
	b.WriteString("No edites archivos fuera del write-set salvo ")
	b.WriteString(ackPath)
	if decisionPath != "" {
		b.WriteString(" y ")
		b.WriteString(decisionPath)
		b.WriteString(" como archivos de control")
	}
	b.WriteString(".\n")
	b.WriteString("Mantén cada fichero Go por debajo de 300 lineas; divide responsabilidades si se acerca a ese limite.\n")
	b.WriteString("Activa $caveman o compact si esta disponible; usa salida minima y evidencia corta.\n")
	b.WriteString("Si falta contexto, no inventes: escribe una nota CONSULTA AL DIRECTOR en el ACK.\n")
	b.WriteString("Aplica arquitectura hexagonal e i18n si la tarea genera app o UI.\n")
	b.WriteString("La persistencia concreta solo pertenece a la app generada si la tarea la pide; Orquesta no usa DB por defecto.\n")
	b.WriteString("Ejecuta las pruebas obligatorias que aparezcan en el paquete si son razonables para el workdir.\n")
	b.WriteString("Al terminar, escribe ")
	b.WriteString(ackPath)
	b.WriteString(" con schema codex_agent_ack.v0.\n\n")
	if decisionPath != "" {
		b.WriteString("decision_path: ")
		b.WriteString(decisionPath)
		b.WriteString(" solo para decisiones ejecutables del director; no pertenece al write-set.\n")
		b.WriteString("Es obligatorio solo si objetivo o criterios de cierre lo piden.\n")
	}
	b.WriteString("En el ACK, files debe listar todos los paths tocados del write-set y nada fuera de el.\n")
	b.WriteString("En el ACK, tests debe listar solo pruebas pasadas; cada test obligatorio pasado debe aparecer exactamente como aparece en el paquete.\n")
	b.WriteString("Si una prueba obligatoria falla, el ACK debe usar status failed y no declarar esa prueba en tests como pasada.\n")
	b.WriteString("No incluyas HOME real, tokens, secretos, prompts, completions ni transcripts completos.\n")
	b.WriteString("files y tests deben ser arrays de strings; los detalles estructurados van en notes.\n\n")
	b.WriteString("ACK esperado:\n")
	b.WriteString("{\"schema_version\":\"codex_agent_ack.v0\",\"request_id\":\"")
	b.WriteString(packet.RequestID)
	b.WriteString("\",\"correlation_id\":\"")
	b.WriteString(packet.CorrelationID)
	b.WriteString("\",\"ack_ref\":\"")
	b.WriteString(packet.DeliveryRefs.AckRef)
	b.WriteString("\",\"target_module\":\"")
	b.WriteString(packet.TargetModule)
	b.WriteString("\",\"task_ref\":\"")
	b.WriteString(packet.Task.TaskRef)
	b.WriteString("\",\"status\":\"completed\",\"files\":")
	b.WriteString(promptJSONStringArrayV0(packet.Task.WriteSet))
	b.WriteString(",\"tests\":")
	b.WriteString(promptJSONStringArrayV0(packet.Task.RequiredTests))
	b.WriteString(",\"notes\":[]}\n\n")
	b.WriteString("Titulo: ")
	b.WriteString(packet.Task.Title)
	b.WriteString("\nObjetivo: ")
	b.WriteString(packet.Task.Objective)
	b.WriteString("\n")
	writePromptListSectionV0(&b, "Write-set permitido:", packet.Task.WriteSet)
	writePromptListSectionV0(&b, "Tests obligatorios:", packet.Task.RequiredTests)
	if len(hints) > 0 {
		b.WriteString("\nNotas operativas del conector:\n")
		for _, hint := range hints {
			if strings.TrimSpace(hint) == "" {
				continue
			}
			b.WriteString("- ")
			b.WriteString(strings.TrimSpace(hint))
			b.WriteString("\n")
		}
	}
	return b.String()
}

func promptJSONStringArrayV0(values []string) string {
	data, err := json.Marshal(compactPromptValuesV0(values))
	if err != nil {
		return "[]"
	}
	return string(data)
}

func writePromptListSectionV0(b *strings.Builder, title string, values []string) {
	compact := compactPromptValuesV0(values)
	if len(compact) == 0 {
		return
	}
	b.WriteString("\n")
	b.WriteString(title)
	b.WriteString("\n")
	for _, value := range compact {
		b.WriteString("- ")
		b.WriteString(value)
		b.WriteString("\n")
	}
}

func compactPromptValuesV0(values []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	return result
}

func cleanControlPathV0(value string, fallback string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}
	return filepath.Clean(trimmed)
}
