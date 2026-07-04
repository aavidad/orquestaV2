package orquestaruntimeclaude

import (
	"encoding/json"
	pathpkg "path"
	"path/filepath"
	"strings"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

type ClaudeControlFilesV0 struct {
	PacketPath          string
	AckPath             string
	DecisionPath        string
	ShutdownRequestPath string
	ShutdownAckPath     string
}

func BuildClaudeAgentPromptV0(packet orquestaruntime.AgentStartPacketV0, hints []string) string {
	return BuildClaudeAgentPromptWithControlFilesV0(packet, hints, ClaudeControlFilesV0{
		PacketPath: ClaudeAgentPacketFileNameV0,
		AckPath:    ClaudeAgentAckFileNameV0,
	})
}

func BuildClaudeAgentPromptWithControlFilesV0(
	packet orquestaruntime.AgentStartPacketV0,
	hints []string,
	control ClaudeControlFilesV0,
) string {
	packetPath := cleanClaudeControlPathV0(control.PacketPath, ClaudeAgentPacketFileNameV0)
	ackPath := cleanClaudeControlPathV0(control.AckPath, ClaudeAgentAckFileNameV0)
	decisionPath := cleanClaudeControlPathV0(control.DecisionPath, "")
	shutdownRequestPath := cleanClaudeControlPathV0(control.ShutdownRequestPath, "")
	shutdownAckPath := cleanClaudeControlPathV0(control.ShutdownAckPath, "")
	var b strings.Builder
	b.WriteString("Eres un agente externo gobernado por OrquestaV2.\n")
	b.WriteString("Lee ")
	b.WriteString(packetPath)
	b.WriteString(" antes de tocar archivos.\n")
	b.WriteString("Seguridad: no borres, no muevas fuera, no trunques archivos existentes y no salgas del workdir del proyecto.\n")
	b.WriteString("Si una parte requiere un efecto externo no autorizado, conserva el avance posible dentro del workdir y deja la limitacion como nota o tarea derivada en el ACK.\n")
	writeClaudeWriteSetPrecedenceProtocolV0(&b, packet)
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
	b.WriteString("Manten cada fichero Go por debajo de 300 lineas; divide responsabilidades si se acerca a ese limite.\n")
	b.WriteString("Comunicacion compacta: activa $caveman full si existe; si no existe, usa compact equivalente.\n")
	b.WriteString("PASO FINAL OBLIGATORIO: antes de terminar el turno, escribe SIEMPRE el ACK de control ")
	b.WriteString(packet.DeliveryRefs.AckRef)
	b.WriteString(" con el status real (completed/blocked/failed), los ficheros del write-set tocados y la evidencia de pruebas. ")
	b.WriteString("No omitas este paso ni lo dejes para luego: aunque el trabajo ya este en disco, sin ACK Orquesta no recibe el acuse causal. ")
	b.WriteString("Escribe el ACK como ultima accion, no antes de terminar el trabajo.\n")
	b.WriteString("Si falta contexto, no inventes: usa lo disponible, guarda el avance y deja la falta como nota de revision en el ACK.\n")
	if claudePacketHasRequiredTruncatedContextV0(packet) {
		b.WriteString("CONTEXTO TRUNCADO: agent_packet.context contiene entradas required=true y truncated=true. Trabaja con refs/materializacion externa cuando este disponible; si no, guarda avance parcial y anota contexto_truncado_pendiente o contexto_truncado_resuelto en notes.\n")
	}
	if claudePacketHasRequiredRefOnlyContextV0(packet) {
		b.WriteString("CONTEXTO REF_ONLY REQUERIDO: agent_packet.context contiene entradas required=true y mode=ref_only. Sigue required_ref_action si ayuda; si no alcanza, guarda avance parcial y anota contexto_ref_only_pendiente o contexto_ref_only_resuelto en notes.\n")
	}
	b.WriteString("Aplica arquitectura hexagonal e i18n si la tarea genera app o UI.\n")
	b.WriteString("La persistencia concreta solo pertenece a la app generada si la tarea la pide; Orquesta no usa DB por defecto.\n")
	b.WriteString("Para revisiones OPES o trabajos agent_review_report/agent_pair_review_report/director_review_matrix, crea un fichero de producto dentro del write-set con JSON o Markdown estructurado. Debe incluir veredicto, hallazgos, riesgos, rework causal, aprovechamiento de material recuperable y decision.\n")
	b.WriteString("Ejecuta las pruebas obligatorias que aparezcan en el paquete si son razonables para el workdir.\n")
	b.WriteString("No uses git status como criterio obligatorio; si el workdir no es repositorio git, ignora esa comprobacion.\n")
	b.WriteString("Si los tests obligatorios pasan y los ficheros del write-set existen, escribe ACK status completed aunque git no aplique.\n")
	b.WriteString("No imprimas diffs ni pegues artefactos completos; valida en compacto, escribe agent_ack.json y termina con la linea ACK.\n")
	writeClaudeShutdownProtocolV0(&b, shutdownRequestPath, shutdownAckPath)
	b.WriteString("Al terminar, escribe ")
	b.WriteString(ackPath)
	b.WriteString(" con schema orquesta_agent_ack.v0. Alias legacy aceptado por compatibilidad: codex_agent_ack.v0.\n\n")
	writeClaudeAckWriteProtocolV0(&b, ackPath)
	if decisionPath != "" {
		b.WriteString("decision_path: ")
		b.WriteString(decisionPath)
		b.WriteString(" solo para decisiones ejecutables del director; no pertenece al write-set.\n")
		b.WriteString("Es obligatorio solo si objetivo o criterios de cierre lo piden.\n")
		if claudePacketTargetsDirectorV0(packet) {
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
	writeClaudeDurableResultProtocolV0(&b)
	b.WriteString("ACK esperado:\n")
	b.WriteString("{\"schema_version\":\"orquesta_agent_ack.v0\",\"request_id\":\"")
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
	b.WriteString(claudePromptJSONStringArrayV0(claudePromptACKFilesV0(packet)))
	b.WriteString(",\"tests\":")
	b.WriteString(claudePromptJSONStringArrayV0(packet.Task.RequiredTests))
	if len(claudeCompactPromptValuesV0(packet.Task.RequiredTests)) > 0 {
		b.WriteString(",\"test_receipts\":")
		b.WriteString(claudePromptACKTestReceiptsJSONV0(packet.Task.RequiredTests))
	}
	b.WriteString(",\"notes\":")
	b.WriteString(claudePromptJSONStringArrayV0(claudePromptACKNotesV0(packet)))
	b.WriteString("}\n\n")
	b.WriteString("Titulo: ")
	b.WriteString(packet.Task.Title)
	b.WriteString("\nObjetivo: ")
	b.WriteString(packet.Task.Objective)
	b.WriteString("\n")
	writeClaudePromptListSectionV0(&b, "Write-set permitido:", packet.Task.WriteSet)
	writeClaudePromptListSectionV0(&b, "Tests obligatorios:", packet.Task.RequiredTests)
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

func writeClaudeDurableResultProtocolV0(b *strings.Builder) {
	writeClaudeDurableResultProtocolForLocaleV0(b, "")
}

func writeClaudeDurableResultProtocolForLocaleV0(b *strings.Builder, locale string) {
	if claudeGoalPromptEnglishLocaleV0(locale) {
		b.WriteString("NEUTRAL DURABLE RESULT: if the objective, criteria or tests request goal-first, durable result, orquesta_goal_result_v0.json or ORQUESTA_GOAL_RESULT_V0, write that JSON inside the write-set, not in runtime_work_dir unless the write-set allows it.\n")
		b.WriteString("The orquesta_goal_result_v0.json file must contain pure JSON parseable from the first byte: no markdown, no ```json fences, no comments and no surrounding text.\n")
		b.WriteString("The JSON must use schema_version orquesta_goal_result.v0, status complete/blocked/invalid, compact summary, artifact_refs, artifact_paths, materialized_artifacts, checklist, required_test_results, domain_receipt_refs, rework_plan_refs and evidence_refs.\n")
		b.WriteString("All evidence_refs fields, including nested ones in materialized_artifacts and required_test_results, must be arrays of strings: [\"evidence-ref-...\"]; do not use objects {\"ref\":...,\"description\":...} or maps.\n")
		b.WriteString("If you need to describe evidence, put the short description in summary, README, handoff or a documentary artifact; evidence_refs only contains compact string refs.\n")
		b.WriteString("In artifact_paths list real relative paths created, modified or verified; do not hide out-of-scope artifacts: declare them and mark status blocked with rework_plan_refs.\n")
		b.WriteString("In materialized_artifacts separate valid artifacts from recoverable drafts with non-empty artifact_ref, relative path, status valid/partial/invalid/non_publishable and evidence_refs.\n")
		b.WriteString("In required_test_results declare only tests really executed and passed with compact evidence_refs; if evidence or QA is missing, use status blocked/invalid and checklist.missing_refs.\n\n")
		return
	}
	b.WriteString("RESULTADO DURABLE NEUTRAL: si objetivo, criterios o tests piden goal-first, result durable, orquesta_goal_result_v0.json u ORQUESTA_GOAL_RESULT_V0, escribe ese JSON dentro del write-set, no en runtime_work_dir salvo que el write-set lo permita.\n")
	b.WriteString("El fichero orquesta_goal_result_v0.json debe contener JSON puro y parseable desde el primer byte: sin markdown, sin fences ```json, sin comentarios y sin texto alrededor.\n")
	b.WriteString("El JSON debe usar schema_version orquesta_goal_result.v0, status complete/blocked/invalid, summary compacto, artifact_refs, artifact_paths, materialized_artifacts, checklist, required_test_results, domain_receipt_refs, rework_plan_refs y evidence_refs.\n")
	b.WriteString("Todos los campos evidence_refs, incluidos los anidados en materialized_artifacts y required_test_results, deben ser arrays de strings: [\"evidence-ref-...\"]; no uses objetos {\"ref\":...,\"description\":...} ni mapas.\n")
	b.WriteString("Si necesitas describir una evidencia, pon la descripcion breve en summary, README, handoff o artefacto documental; evidence_refs solo contiene refs string compactas.\n")
	b.WriteString("En artifact_paths lista rutas relativas reales creadas, modificadas o verificadas; no ocultes artefactos fuera de scope: declaralos y marca status blocked con rework_plan_refs.\n")
	b.WriteString("En materialized_artifacts separa artefactos validos de borradores recuperables con artifact_ref no vacio, path relativo, status valid/partial/invalid/non_publishable y evidence_refs.\n")
	b.WriteString("En required_test_results declara solo pruebas realmente ejecutadas y pasadas con evidence_refs compactas; si falta evidencia o QA, usa status blocked/invalid y checklist.missing_refs.\n\n")
}

func writeClaudeWriteSetPrecedenceProtocolV0(
	b *strings.Builder,
	packet orquestaruntime.AgentStartPacketV0,
) {
	if orquestaruntime.AgentStartPacketWriteSetClosedV0(packet) {
		b.WriteString("Usa el write-set del paquete como alcance de escritura; si contiene '.', el alcance es todo el repo del proyecto, pero sigue prohibido borrar o salir del workdir.\n")
		b.WriteString("Si falta alcance, no edites fuera: conserva lo util dentro del write-set y registra el faltante como nota de revision o tarea derivada.\n")
		b.WriteString("La unica excepcion son archivos de control indicados por Orquesta.\n")
		return
	}
	b.WriteString("MODO COMPATIBILIDAD LEGACY: usa el write-set como alcance primario y no amplies alcance cuando haya riesgo de causalidad, seguridad o efectos externos.\n")
}

func claudePacketTargetsDirectorV0(packet orquestaruntime.AgentStartPacketV0) bool {
	return strings.Contains(strings.ToLower(strings.TrimSpace(packet.TargetModule)), "director")
}

func writeClaudeAckWriteProtocolV0(b *strings.Builder, ackPath string) {
	if !filepath.IsAbs(ackPath) {
		return
	}
	b.WriteString("PROTOCOLO ACK FUERA DEL PROYECTO: ")
	b.WriteString(ackPath)
	b.WriteString(" es fichero de control, no artefacto del proyecto. Escribelo desde su directorio de control y no crees producto alli.\n\n")
}

func writeClaudeShutdownProtocolV0(b *strings.Builder, requestPath string, ackPath string) {
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

func claudePromptACKNotesV0(packet orquestaruntime.AgentStartPacketV0) []string {
	notes := []string{}
	if claudePacketHasRequiredTruncatedContextV0(packet) {
		notes = append(notes, "contexto_truncado_resuelto: <motivo>")
	}
	if claudePacketRequiresRefOnlyAckEvidenceV0(packet) {
		notes = append(notes, "contexto_ref_only_resuelto: <motivo>")
	}
	return notes
}

func claudePromptJSONStringArrayV0(values []string) string {
	data, err := json.Marshal(claudeCompactPromptValuesV0(values))
	if err != nil {
		return "[]"
	}
	return string(data)
}

func claudePromptACKFilesV0(packet orquestaruntime.AgentStartPacketV0) []string {
	values := claudeCompactPromptValuesV0(packet.Task.WriteSet)
	files := make([]string, 0, len(values))
	for _, value := range values {
		if claudePromptACKFileLooksConcreteV0(value) {
			files = append(files, value)
		}
	}
	if len(files) == 0 {
		return []string{"<path-real-tocado>"}
	}
	return files
}

func claudePromptACKFileLooksConcreteV0(value string) bool {
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

func writeClaudePromptListSectionV0(b *strings.Builder, title string, values []string) {
	b.WriteString("\n")
	b.WriteString(title)
	b.WriteString("\n")
	values = claudeCompactPromptValuesV0(values)
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

func claudeCompactPromptValuesV0(values []string) []string {
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

func cleanClaudeControlPathV0(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return filepath.Clean(value)
}
