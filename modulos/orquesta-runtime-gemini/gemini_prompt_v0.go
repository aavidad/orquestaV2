package orquestaruntimegemini

import (
	"encoding/json"
	"path/filepath"
	"strings"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

type GeminiControlFilesV0 struct {
	PacketPath string
	AckPath    string
}

func BuildGeminiAgentPromptV0(packet orquestaruntime.AgentStartPacketV0, hints []string) string {
	return BuildGeminiAgentPromptWithControlFilesV0(packet, hints, GeminiControlFilesV0{
		PacketPath: GeminiAgentPacketFileNameV0,
		AckPath:    GeminiAgentAckFileNameV0,
	})
}

func BuildGeminiAgentPromptWithControlFilesV0(
	packet orquestaruntime.AgentStartPacketV0,
	hints []string,
	control GeminiControlFilesV0,
) string {
	packetPath := cleanGeminiControlPathV0(control.PacketPath, GeminiAgentPacketFileNameV0)
	ackPath := cleanGeminiControlPathV0(control.AckPath, GeminiAgentAckFileNameV0)
	var b strings.Builder
	b.WriteString("Eres un agente externo gobernado por OrquestaV2.\n")
	b.WriteString("Lee ")
	b.WriteString(packetPath)
	b.WriteString(" antes de tocar archivos.\n")
	b.WriteString("Usa el write-set del paquete como alcance cerrado para producto. No borres, no muevas fuera del proyecto y no salgas del workdir.\n")
	b.WriteString("Si falta contexto o alcance, no tires el trabajo por una palabra exacta: completa lo recuperable y deja CONSULTA AL DIRECTOR en notes cuando haga falta.\n")
	b.WriteString("Corta fuerte solo por seguridad, refs imposibles, datos sensibles o efectos externos no autorizados.\n")
	b.WriteString("Archivos de control permitidos fuera del write-set: ")
	b.WriteString(ackPath)
	b.WriteString(". No incluyas control files en ACK.files.\n")
	b.WriteString("PROTOCOLO COMPACTO OBLIGATORIO: usa salida compacta; no pegues diffs ni artefactos completos en stdout.\n")
	b.WriteString("Final visible maximo una linea: ACK ")
	b.WriteString(packet.DeliveryRefs.AckRef)
	b.WriteString(" <status>. La linea visible no sustituye el JSON durable.\n")
	b.WriteString("Para tareas generate_visual_asset, visual_asset o infografia, crea un fichero de producto dentro del write-set con JSON valido: {\"artifact_type\":\"visual_asset\",\"payload_json\":{\"format\":\"svg\",\"title\":\"...\",\"caption\":\"...\",\"alt_text\":\"...\",\"svg\":\"...\"}}. Tambien son validos mermaid, html_panel o markdown si el contrato lo pide. No uses binarios, data_uri ni URLs remotas.\n")
	b.WriteString("En visuales educativos, incluye alt_text accesible, caption breve, formato claro y contenido autocontenido.\n")
	b.WriteString("Ejecuta las pruebas obligatorias si son razonables para el workdir. Si no aplican por ser trabajo externo, explica la razon en notes sin bloquear por una frase exacta.\n")
	b.WriteString("Al terminar, escribe ")
	b.WriteString(ackPath)
	b.WriteString(" con schema codex_agent_ack.v0.\n\n")
	writeGeminiAckWriteProtocolV0(&b, ackPath)
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
	b.WriteString(geminiPromptJSONStringArrayV0(geminiPromptACKFilesV0(packet)))
	b.WriteString(",\"tests\":")
	b.WriteString(geminiPromptJSONStringArrayV0(packet.Task.RequiredTests))
	b.WriteString(",\"notes\":[]}\n\n")
	b.WriteString("Titulo: ")
	b.WriteString(packet.Task.Title)
	b.WriteString("\nObjetivo: ")
	b.WriteString(packet.Task.Objective)
	b.WriteString("\n")
	writeGeminiPromptListSectionV0(&b, "Write-set permitido:", packet.Task.WriteSet)
	writeGeminiPromptListSectionV0(&b, "Tests obligatorios:", packet.Task.RequiredTests)
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

func writeGeminiAckWriteProtocolV0(b *strings.Builder, ackPath string) {
	if !filepath.IsAbs(ackPath) {
		return
	}
	b.WriteString("PROTOCOLO ACK FUERA DEL PROYECTO: ")
	b.WriteString(ackPath)
	b.WriteString(" es fichero de control, no artefacto del proyecto. Escribelo desde su directorio de control y no crees producto alli.\n\n")
}

func geminiPromptJSONStringArrayV0(values []string) string {
	data, err := json.Marshal(geminiCompactPromptValuesV0(values))
	if err != nil {
		return "[]"
	}
	return string(data)
}

func geminiPromptACKFilesV0(packet orquestaruntime.AgentStartPacketV0) []string {
	values := geminiCompactPromptValuesV0(packet.Task.WriteSet)
	files := make([]string, 0, len(values))
	for _, value := range values {
		if geminiPromptACKFileLooksConcreteV0(value) {
			files = append(files, value)
		}
	}
	if len(files) == 0 {
		return []string{"<path-real-tocado>"}
	}
	return files
}

func geminiPromptACKFileLooksConcreteV0(value string) bool {
	value = strings.TrimSpace(value)
	return value != "" &&
		value != "." &&
		!strings.HasSuffix(value, "/") &&
		!strings.Contains(value, "*") &&
		filepath.Base(value) != "."
}

func writeGeminiPromptListSectionV0(b *strings.Builder, title string, values []string) {
	b.WriteString("\n")
	b.WriteString(title)
	b.WriteString("\n")
	values = geminiCompactPromptValuesV0(values)
	if len(values) == 0 {
		b.WriteString("- <ninguno>\n")
		return
	}
	for _, value := range values {
		b.WriteString("- ")
		b.WriteString(value)
		b.WriteString("\n")
	}
}

func geminiCompactPromptValuesV0(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func cleanGeminiControlPathV0(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}
