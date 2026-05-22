package orquestaruntimecodex

import (
	"encoding/json"
	pathpkg "path"
	"path/filepath"
	"strings"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

type CodexControlFilesV0 struct {
	PacketPath          string
	AckPath             string
	DecisionPath        string
	ShutdownRequestPath string
	ShutdownAckPath     string
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
	shutdownRequestPath := cleanControlPathV0(control.ShutdownRequestPath, "")
	shutdownAckPath := cleanControlPathV0(control.ShutdownAckPath, "")
	var b strings.Builder
	b.WriteString("Eres un agente externo gobernado por OrquestaV2.\n")
	b.WriteString("Lee ")
	b.WriteString(packetPath)
	b.WriteString(" antes de tocar archivos.\n")
	b.WriteString("Usa el write-set del paquete como alcance primario; si contiene '.', tienes permiso sobre todo el repo del proyecto.\n")
	b.WriteString("Si el write-set no contiene '.', no edites fuera de ese alcance salvo archivos de control o una ampliacion imprescindible para cumplir el objetivo, que debes justificar en notes.\n")
	b.WriteString("Archivos de control permitidos fuera del write-set: ")
	b.WriteString(ackPath)
	if decisionPath != "" {
		b.WriteString(", ")
		b.WriteString(decisionPath)
	}
	if shutdownRequestPath != "" && shutdownAckPath != "" {
		b.WriteString(", ")
		b.WriteString(shutdownRequestPath)
		b.WriteString(" y ")
		b.WriteString(shutdownAckPath)
	}
	b.WriteString(".\n")
	b.WriteString("Mantén cada fichero Go por debajo de 300 lineas; divide responsabilidades si se acerca a ese limite.\n")
	b.WriteString("PROTOCOLO COMPACTO OBLIGATORIO: activa $caveman full si existe; si no existe, usa compact equivalente.\n")
	b.WriteString("Sin narrativa visible. Final visible maximo una linea: ACK ")
	b.WriteString(packet.DeliveryRefs.AckRef)
	b.WriteString(" <status>. Incumplir este protocolo invalida la entrega.\n")
	b.WriteString("Si falta contexto, no inventes: escribe una nota CONSULTA AL DIRECTOR en el ACK.\n")
	if codexPacketHasRequiredTruncatedContextV0(packet) {
		b.WriteString("CONTEXTO TRUNCADO REQUERIDO: agent_packet.context contiene entradas required=true y truncated=true. No completes salvo que puedas resolverlo con refs/materializacion externa; si completas, notes debe incluir contexto_truncado_resuelto: <motivo>. Si no, usa status failed y CONSULTA AL DIRECTOR.\n")
	}
	b.WriteString("Aplica arquitectura hexagonal e i18n si la tarea genera app o UI.\n")
	b.WriteString("La persistencia concreta solo pertenece a la app generada si la tarea la pide; Orquesta no usa DB por defecto.\n")
	b.WriteString("Ejecuta las pruebas obligatorias que aparezcan en el paquete si son razonables para el workdir.\n")
	b.WriteString("No uses git status como criterio obligatorio; si el workdir no es repositorio git, ignora esa comprobacion.\n")
	b.WriteString("Si los tests obligatorios pasan y los ficheros del write-set existen, escribe ACK status completed aunque git no aplique.\n")
	b.WriteString("No imprimas diffs ni pegues artefactos completos; valida en compacto, escribe agent_ack.json y termina con la linea ACK.\n")
	writeShutdownProtocolV0(&b, shutdownRequestPath, shutdownAckPath)
	b.WriteString("Al terminar, escribe ")
	b.WriteString(ackPath)
	b.WriteString(" con schema codex_agent_ack.v0.\n\n")
	writeAckWriteProtocolV0(&b, ackPath)
	if decisionPath != "" {
		b.WriteString("decision_path: ")
		b.WriteString(decisionPath)
		b.WriteString(" solo para decisiones ejecutables del director; no pertenece al write-set.\n")
		b.WriteString("Es obligatorio solo si objetivo o criterios de cierre lo piden.\n")
		b.WriteString("Si escribes decision_path, completa antes los ficheros pedidos del write-set, despues escribe ACK y termina; no sigas pensando ni ampliando alcance.\n")
	}
	b.WriteString("En el ACK, files debe listar rutas reales de archivos de producto tocados, no globs, directorios ni el write-set completo; ejemplo cmd/server/main.go, no cmd/server/**.\n")
	b.WriteString("Si algun archivo queda fuera del write-set estrecho, notes debe explicar por que era necesario.\n")
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
	b.WriteString(promptJSONStringArrayV0(promptACKFilesV0(packet)))
	b.WriteString(",\"tests\":")
	b.WriteString(promptJSONStringArrayV0(packet.Task.RequiredTests))
	b.WriteString(",\"notes\":")
	b.WriteString(promptJSONStringArrayV0(promptACKNotesV0(packet)))
	b.WriteString("}\n\n")
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

func writeAckWriteProtocolV0(b *strings.Builder, ackPath string) {
	if !filepath.IsAbs(ackPath) {
		return
	}
	b.WriteString("PROTOCOLO ACK FUERA DEL PROYECTO: ")
	b.WriteString(ackPath)
	b.WriteString(" es fichero de control, no artifact del proyecto. No uses apply_patch para escribirlo. Escribelo con shell desde su directorio de control: cd ")
	b.WriteString(filepath.Dir(ackPath))
	b.WriteString(" && crear ")
	b.WriteString(filepath.Base(ackPath))
	b.WriteString(". La linea visible ACK no sustituye este JSON.\n\n")
}

func writeShutdownProtocolV0(b *strings.Builder, requestPath string, ackPath string) {
	if requestPath == "" || ackPath == "" {
		return
	}
	b.WriteString("CHECKPOINT DE APAGADO: antes de cada bloque de edicion o prueba comprueba si existe ")
	b.WriteString(requestPath)
	b.WriteString(".\n")
	b.WriteString("Si existe, lee ese JSON, para en un punto consistente, no amplias alcance y escribe ")
	b.WriteString(ackPath)
	b.WriteString(" con schema codex_shutdown_checkpoint_ack.v0, run_ref, agent_ref, checkpoint_ref y status checkpoint_ready.\n")
	b.WriteString("Despues del ACK de checkpoint no sigas ejecutando trabajo largo; termina con salida compacta.\n")
}

func promptJSONStringArrayV0(values []string) string {
	data, err := json.Marshal(compactPromptValuesV0(values))
	if err != nil {
		return "[]"
	}
	return string(data)
}

func promptACKNotesV0(packet orquestaruntime.AgentStartPacketV0) []string {
	if !codexPacketHasRequiredTruncatedContextV0(packet) {
		return nil
	}
	return []string{"contexto_truncado_resuelto: <motivo>"}
}

func promptACKFilesV0(packet orquestaruntime.AgentStartPacketV0) []string {
	values := compactPromptValuesV0(packet.Task.WriteSet)
	files := make([]string, 0, len(values))
	for _, value := range values {
		if promptACKFileLooksConcreteV0(value) {
			files = append(files, value)
		}
	}
	if len(files) == 0 {
		return []string{"<path-real-tocado>"}
	}
	return files
}

func promptACKFileLooksConcreteV0(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" || value == "." || strings.ContainsAny(value, "*?[") ||
		strings.HasSuffix(value, "/") {
		return false
	}
	base := pathpkg.Base(value)
	if strings.Contains(base, ".") {
		return true
	}
	switch strings.ToLower(base) {
	case "makefile", "readme", "license":
		return true
	default:
		return false
	}
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

func codexPacketHasRequiredTruncatedContextV0(
	packet orquestaruntime.AgentStartPacketV0,
) bool {
	for _, entry := range packet.Context.Entries {
		if entry.Required && entry.Truncated {
			return true
		}
	}
	return false
}
