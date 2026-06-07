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
	b.WriteString("Seguridad: no borres, no muevas fuera, no trunques archivos existentes y no salgas del workdir del proyecto.\n")
	b.WriteString("Si una parte requiere un efecto externo no autorizado, conserva el avance posible dentro del workdir y deja la limitacion como nota o tarea derivada en el ACK.\n")
	writeWriteSetPrecedenceProtocolV0(&b, packet)
	b.WriteString("Las rutas del write-set son relativas al workdir del proyecto, no al directorio de control ni a .orquesta-runtime; crea docs/codigo en el proyecto.\n")
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
	b.WriteString("Comunicacion compacta: activa $caveman full si existe; si no existe, usa compact equivalente.\n")
	b.WriteString("Final visible recomendado: ACK ")
	b.WriteString(packet.DeliveryRefs.AckRef)
	b.WriteString(" <status>.\n")
	b.WriteString("Si falta contexto, no inventes: usa lo disponible, guarda el avance y deja la falta como nota de revision en el ACK.\n")
	if codexPacketHasRequiredTruncatedContextV0(packet) {
		b.WriteString("CONTEXTO TRUNCADO: agent_packet.context contiene entradas required=true y truncated=true. Trabaja con refs/materializacion externa cuando este disponible; si no, guarda avance parcial y anota contexto_truncado_pendiente o contexto_truncado_resuelto en notes.\n")
	}
	if codexPacketHasRequiredRefOnlyContextV0(packet) {
		b.WriteString("CONTEXTO REF_ONLY REQUERIDO: agent_packet.context contiene entradas required=true y mode=ref_only. Sigue required_ref_action si ayuda; si no alcanza, guarda avance parcial y anota contexto_ref_only_pendiente o contexto_ref_only_resuelto en notes.\n")
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
		if codexPacketTargetsDirectorV0(packet) {
			b.WriteString("Como target_module de director, escribe decision_path con las decisiones ejecutables que puedas; si no esta completo, conserva el diagnostico y deja follow-up en ACK.\n")
		}
		b.WriteString("Si escribes decision_path, completa antes los ficheros pedidos del write-set, despues escribe ACK y termina; no sigas pensando ni ampliando alcance.\n")
	}
	b.WriteString("En el ACK, files debe listar rutas reales de archivos de producto tocados, no globs, directorios ni el write-set completo; ejemplo cmd/server/main.go, no cmd/server/**.\n")
	b.WriteString("No incluyas archivos de control en ACK.files: agent_ack.json, director_decisions.json, agent_packet.json, prompts, logs ni checkpoints.\n")
	b.WriteString("Si detectas que algun archivo necesario queda fuera del write-set, no lo edites; continua con lo permitido y registra el faltante como nota o tarea derivada.\n")
	b.WriteString("En el ACK, tests debe listar solo pruebas pasadas; cada test obligatorio pasado debe aparecer exactamente como aparece en el paquete.\n")
	b.WriteString("Si una prueba obligatoria falla, conserva la evidencia en test_receipts/notes y no la declares como pasada.\n")
	b.WriteString("En modo estricto, anade test_receipts por cada test pasado con comando exacto, status passed, exit_code 0, evidence_refs compactas, occurred_at, sequence y output_redacted=true.\n")
	b.WriteString("No incluyas HOME real, tokens, secretos, prompts, completions ni transcripts completos.\n")
	b.WriteString("files y tests deben ser arrays de strings; no metas stdout/stderr crudo en test_receipts ni notes.\n\n")
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
	if len(compactPromptValuesV0(packet.Task.RequiredTests)) > 0 {
		b.WriteString(",\"test_receipts\":")
		b.WriteString(promptACKTestReceiptsJSONV0(packet.Task.RequiredTests))
	}
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

func writeWriteSetPrecedenceProtocolV0(
	b *strings.Builder,
	packet orquestaruntime.AgentStartPacketV0,
) {
	if orquestaruntime.AgentStartPacketWriteSetClosedV0(packet) {
		b.WriteString("Usa el write-set del paquete como alcance de escritura; si contiene '.', el alcance es todo el repo del proyecto, pero sigue prohibido borrar o salir del workdir.\n")
		b.WriteString("Si falta alcance, no edites fuera: conserva lo util dentro del write-set y registra el faltante como nota de revision o tarea derivada.\n")
		b.WriteString("La unica excepcion son archivos de control indicados por Orquesta.\n")
		return
	}
	b.WriteString("MODO COMPATIBILIDAD LEGACY: usa el write-set como alcance primario y no amplíes alcance cuando haya riesgo de causalidad, seguridad o efectos externos.\n")
}

func codexPacketTargetsDirectorV0(packet orquestaruntime.AgentStartPacketV0) bool {
	return strings.Contains(strings.ToLower(strings.TrimSpace(packet.TargetModule)), "director")
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
	b.WriteString(". No crees docs/codigo en ese directorio de control. La linea visible ACK no sustituye este JSON.\n\n")
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
	notes := []string{}
	if codexPacketHasRequiredTruncatedContextV0(packet) {
		notes = append(notes, "contexto_truncado_resuelto: <motivo>")
	}
	if codexPacketRequiresRefOnlyAckEvidenceV0(packet) {
		notes = append(notes, "contexto_ref_only_resuelto: <motivo>")
	}
	return notes
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
