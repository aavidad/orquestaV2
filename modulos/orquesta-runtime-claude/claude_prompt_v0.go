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
	return BuildClaudeAgentPromptWithLocaleV0(packet, hints, "")
}

func BuildClaudeAgentPromptWithLocaleV0(
	packet orquestaruntime.AgentStartPacketV0,
	hints []string,
	locale string,
) string {
	return BuildClaudeAgentPromptWithLocaleAndControlFilesV0(packet, hints, locale, ClaudeControlFilesV0{
		PacketPath: ClaudeAgentPacketFileNameV0,
		AckPath:    ClaudeAgentAckFileNameV0,
	})
}

func BuildClaudeAgentPromptWithControlFilesV0(
	packet orquestaruntime.AgentStartPacketV0,
	hints []string,
	control ClaudeControlFilesV0,
) string {
	return BuildClaudeAgentPromptWithLocaleAndControlFilesV0(packet, hints, "", control)
}

func BuildClaudeAgentPromptWithLocaleAndControlFilesV0(
	packet orquestaruntime.AgentStartPacketV0,
	hints []string,
	locale string,
	control ClaudeControlFilesV0,
) string {
	packetPath := cleanClaudeControlPathV0(control.PacketPath, ClaudeAgentPacketFileNameV0)
	ackPath := cleanClaudeControlPathV0(control.AckPath, ClaudeAgentAckFileNameV0)
	decisionPath := cleanClaudeControlPathV0(control.DecisionPath, "")
	shutdownRequestPath := cleanClaudeControlPathV0(control.ShutdownRequestPath, "")
	shutdownAckPath := cleanClaudeControlPathV0(control.ShutdownAckPath, "")
	if claudeGoalPromptEnglishLocaleV0(locale) {
		return buildClaudeAgentPromptEnglishV0(packet, hints, locale, ClaudeControlFilesV0{
			PacketPath:          packetPath,
			AckPath:             ackPath,
			DecisionPath:        decisionPath,
			ShutdownRequestPath: shutdownRequestPath,
			ShutdownAckPath:     shutdownAckPath,
		})
	}
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
	writeClaudeDurableResultProtocolForLocaleV0(&b, locale)
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

func buildClaudeAgentPromptEnglishV0(
	packet orquestaruntime.AgentStartPacketV0,
	hints []string,
	locale string,
	control ClaudeControlFilesV0,
) string {
	packetPath := control.PacketPath
	ackPath := control.AckPath
	decisionPath := control.DecisionPath
	shutdownRequestPath := control.ShutdownRequestPath
	shutdownAckPath := control.ShutdownAckPath
	var b strings.Builder
	b.WriteString("You are an external agent governed by OrquestaV2.\n")
	b.WriteString("Read ")
	b.WriteString(packetPath)
	b.WriteString(" before touching files.\n")
	b.WriteString("Safety: do not delete, move files outside scope, truncate existing files, or leave the project workdir.\n")
	b.WriteString("If part of the task needs an unauthorized external effect, keep all possible progress inside the workdir and record the limitation as a note or derived task in the ACK.\n")
	writeClaudeWriteSetPrecedenceProtocolForLocaleV0(&b, packet, locale)
	b.WriteString("Write-set paths are relative to the project workdir, not to the control directory or .orquesta-runtime; create docs/code in the project.\n")
	b.WriteString("Allowed control files outside the write-set: ")
	b.WriteString(ackPath)
	if decisionPath != "" {
		b.WriteString(", ")
		b.WriteString(decisionPath)
	}
	if shutdownRequestPath != "" && shutdownAckPath != "" {
		b.WriteString(", ")
		b.WriteString(shutdownRequestPath)
		b.WriteString(" and ")
		b.WriteString(shutdownAckPath)
	}
	b.WriteString(".\n")
	b.WriteString("Keep every Go file below 300 lines; split responsibilities if a file approaches that limit.\n")
	b.WriteString("Compact communication: enable $caveman full if available; otherwise use an equivalent compact style.\n")
	b.WriteString("MANDATORY FINAL STEP: before ending the turn, ALWAYS write the control ACK ")
	b.WriteString(packet.DeliveryRefs.AckRef)
	b.WriteString(" with the real status (completed/blocked/failed), touched write-set files and test evidence. ")
	b.WriteString("Do not omit this step or leave it for later: even if the work is already on disk, without ACK Orquesta does not receive causal acknowledgement. ")
	b.WriteString("Write the ACK as the last action, not before finishing the work.\n")
	b.WriteString("If context is missing, do not invent: use what is available, save progress and record the gap as a review note in the ACK.\n")
	if claudePacketHasRequiredTruncatedContextV0(packet) {
		b.WriteString("TRUNCATED CONTEXT: agent_packet.context contains required=true and truncated=true entries. Work with refs/external materialization when available; otherwise save partial progress and add contexto_truncado_pendiente or contexto_truncado_resuelto in notes.\n")
	}
	if claudePacketHasRequiredRefOnlyContextV0(packet) {
		b.WriteString("REQUIRED REF_ONLY CONTEXT: agent_packet.context contains required=true and mode=ref_only entries. Follow required_ref_action when useful; if that is not enough, save partial progress and add contexto_ref_only_pendiente or contexto_ref_only_resuelto in notes.\n")
	}
	b.WriteString("Apply hexagonal architecture and i18n if the task generates an app or UI.\n")
	b.WriteString("Concrete persistence belongs only to the generated app if the task asks for it; Orquesta does not use DB by default.\n")
	b.WriteString("For OPES reviews or agent_review_report/agent_pair_review_report/director_review_matrix work, create a product file inside the write-set with structured JSON or Markdown. It must include verdict, findings, risks, causal rework, reuse of recoverable material and decision.\n")
	b.WriteString("Run required tests listed in the packet when reasonable for the workdir.\n")
	b.WriteString("Do not use git status as a mandatory criterion; if the workdir is not a git repository, ignore that check.\n")
	b.WriteString("If required tests pass and write-set files exist, write ACK status completed even if git does not apply.\n")
	b.WriteString("Do not print diffs or paste complete artifacts; validate compactly, write agent_ack.json and finish with the ACK line.\n")
	writeClaudeShutdownProtocolForLocaleV0(&b, shutdownRequestPath, shutdownAckPath, locale)
	b.WriteString("When finished, write ")
	b.WriteString(ackPath)
	b.WriteString(" with schema orquesta_agent_ack.v0. Legacy alias accepted for compatibility: codex_agent_ack.v0.\n\n")
	writeClaudeAckWriteProtocolForLocaleV0(&b, ackPath, locale)
	if decisionPath != "" {
		b.WriteString("decision_path: ")
		b.WriteString(decisionPath)
		b.WriteString(" only for executable director decisions; it does not belong to the write-set.\n")
		b.WriteString("It is mandatory only if the objective or closure criteria ask for it.\n")
		if claudePacketTargetsDirectorV0(packet) {
			b.WriteString("As a director target_module, write decision_path with the executable decisions you can; if incomplete, keep the diagnostic and leave follow-up in ACK.\n")
		}
		b.WriteString("If you write decision_path, first complete requested write-set files, then write ACK and stop; do not keep thinking or expanding scope.\n")
	}
	b.WriteString("In ACK, files must list real product file paths touched, not globs, directories or the full write-set; example cmd/server/main.go, not cmd/server/**.\n")
	b.WriteString("Do not include control files in ACK.files: agent_ack.json, director_decisions.json, agent_packet.json, prompts, logs or checkpoints.\n")
	b.WriteString("If you detect that a needed file is outside the write-set, do not edit it; continue with allowed scope and record the missing file as a note or derived task.\n")
	b.WriteString("In ACK, tests must list only passed tests; each passed required test must appear exactly as it appears in the packet.\n")
	b.WriteString("If a required test fails, keep evidence in test_receipts/notes and do not declare it passed.\n")
	b.WriteString("In strict mode, add test_receipts for each passed test with exact command, status passed, exit_code 0, compact evidence_refs, occurred_at, sequence and output_redacted=true.\n")
	b.WriteString("Do not include real HOME, tokens, secrets, prompts, completions or full transcripts.\n")
	b.WriteString("files and tests must be string arrays; do not put raw stdout/stderr in test_receipts or notes.\n\n")
	writeClaudeDurableResultProtocolForLocaleV0(&b, locale)
	b.WriteString("Expected ACK:\n")
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
	b.WriteString("Title: ")
	b.WriteString(packet.Task.Title)
	b.WriteString("\nObjective: ")
	b.WriteString(packet.Task.Objective)
	b.WriteString("\n")
	writeClaudePromptListSectionForLocaleV0(&b, "Allowed write-set:", packet.Task.WriteSet, locale)
	writeClaudePromptListSectionForLocaleV0(&b, "Required tests:", packet.Task.RequiredTests, locale)
	if len(hints) > 0 {
		b.WriteString("\nConnector operational notes:\n")
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

func writeClaudeDurableResultProtocolForLocaleV0(b *strings.Builder, locale string) {
	if claudeGoalPromptEnglishLocaleV0(locale) {
		b.WriteString("NEUTRAL DURABLE RESULT: the final goal receipt belongs in the designated runtime receipt path, never in the product write-set.\n")
		b.WriteString("The orquesta_goal_result_v0.json file must contain pure JSON parseable from the first byte: no markdown, no ```json fences, no comments and no surrounding text.\n")
		b.WriteString("The JSON must use schema_version orquesta_goal_result.v0, status complete/blocked/invalid, compact summary, artifact_refs, artifact_paths, materialized_artifacts, checklist, required_test_results, domain_receipt_refs, rework_plan_refs and evidence_refs.\n")
		b.WriteString("All evidence_refs fields, including nested ones in materialized_artifacts and required_test_results, must be arrays of strings: [\"evidence-ref-...\"]; do not use objects {\"ref\":...,\"description\":...} or maps.\n")
		b.WriteString("If you need to describe evidence, put the short description in summary, README, handoff or a documentary artifact; evidence_refs only contains compact string refs.\n")
		b.WriteString("In artifact_paths list real relative paths created, modified or verified; do not hide out-of-scope artifacts: declare them and mark status blocked with rework_plan_refs.\n")
		b.WriteString("In materialized_artifacts separate valid artifacts from recoverable drafts with non-empty artifact_ref, relative path, status valid/partial/invalid/non_publishable and evidence_refs.\n")
		b.WriteString("In required_test_results declare only tests really executed and passed with compact evidence_refs; if evidence or QA is missing, use status blocked/invalid and checklist.missing_refs.\n\n")
		return
	}
	b.WriteString("RESULTADO DURABLE NEUTRAL: el recibo final del goal pertenece a la ruta runtime designada, nunca dentro del write-set de producto.\n")
	b.WriteString("El fichero orquesta_goal_result_v0.json debe contener JSON puro y parseable desde el primer byte: sin markdown, sin fences ```json, sin comentarios y sin texto alrededor.\n")
	b.WriteString("El JSON debe usar schema_version orquesta_goal_result.v0, status complete/blocked/invalid, summary compacto, artifact_refs, artifact_paths, materialized_artifacts, checklist, required_test_results, domain_receipt_refs, rework_plan_refs y evidence_refs.\n")
	b.WriteString("Todos los campos evidence_refs, incluidos los anidados en materialized_artifacts y required_test_results, deben ser arrays de strings: [\"evidence-ref-...\"]; no uses objetos {\"ref\":...,\"description\":...} ni mapas.\n")
	b.WriteString("Si necesitas describir una evidencia, pon la descripcion breve en summary, README, handoff o artefacto documental; evidence_refs solo contiene refs string compactas.\n")
	b.WriteString("En artifact_paths lista rutas relativas reales creadas, modificadas o verificadas; no ocultes artefactos fuera de scope: declaralos y marca status blocked con rework_plan_refs.\n")
	b.WriteString("En materialized_artifacts separa artefactos validos de borradores recuperables con artifact_ref no vacio, path relativo, status valid/partial/invalid/non_publishable y evidence_refs.\n")
	b.WriteString("En required_test_results declara solo pruebas realmente ejecutadas y pasadas con evidence_refs compactas; si falta evidencia o QA, usa status blocked/invalid y checklist.missing_refs.\n\n")
}

func claudeGoalPromptWithRuntimeReceiptV0(prompt string, runtimeDir string, goalRef string, locale string) string {
	path := claudeGoalRuntimeResultPathV0(runtimeDir, goalRef)
	if claudeGoalPromptEnglishLocaleV0(locale) {
		return prompt + "Write the final pure JSON receipt to runtime path: " + path + ". Do not list that receipt in artifact_paths or materialized_artifacts.\n"
	}
	return prompt + "Escribe el recibo JSON final en la ruta runtime: " + path + ". No declares ese recibo en artifact_paths ni materialized_artifacts.\n"
}

func writeClaudeWriteSetPrecedenceProtocolV0(
	b *strings.Builder,
	packet orquestaruntime.AgentStartPacketV0,
) {
	writeClaudeWriteSetPrecedenceProtocolForLocaleV0(b, packet, "")
}

func writeClaudeWriteSetPrecedenceProtocolForLocaleV0(
	b *strings.Builder,
	packet orquestaruntime.AgentStartPacketV0,
	locale string,
) {
	if claudeGoalPromptEnglishLocaleV0(locale) {
		if orquestaruntime.AgentStartPacketWriteSetClosedV0(packet) {
			b.WriteString("Use the packet write-set as write scope; if it contains '.', scope is the whole project repo, but deleting files or leaving the workdir is still forbidden.\n")
			b.WriteString("If scope is missing, do not edit outside it: keep useful progress inside the write-set and record the missing scope as a review note or derived task.\n")
			b.WriteString("The only exception is control files declared by Orquesta.\n")
			return
		}
		b.WriteString("LEGACY COMPATIBILITY MODE: use the write-set as primary scope and do not expand scope when there is causal, safety or external-effect risk.\n")
		return
	}
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
	writeClaudeAckWriteProtocolForLocaleV0(b, ackPath, "")
}

func writeClaudeAckWriteProtocolForLocaleV0(b *strings.Builder, ackPath string, locale string) {
	if !filepath.IsAbs(ackPath) {
		return
	}
	if claudeGoalPromptEnglishLocaleV0(locale) {
		b.WriteString("ACK PROTOCOL OUTSIDE PROJECT: ")
		b.WriteString(ackPath)
		b.WriteString(" is a control file, not a project artifact. Write it from its control directory and do not create product there.\n\n")
		return
	}
	b.WriteString("PROTOCOLO ACK FUERA DEL PROYECTO: ")
	b.WriteString(ackPath)
	b.WriteString(" es fichero de control, no artefacto del proyecto. Escribelo desde su directorio de control y no crees producto alli.\n\n")
}

func writeClaudeShutdownProtocolV0(b *strings.Builder, requestPath string, ackPath string) {
	writeClaudeShutdownProtocolForLocaleV0(b, requestPath, ackPath, "")
}

func writeClaudeShutdownProtocolForLocaleV0(b *strings.Builder, requestPath string, ackPath string, locale string) {
	if requestPath == "" || ackPath == "" {
		return
	}
	if claudeGoalPromptEnglishLocaleV0(locale) {
		b.WriteString("SHUTDOWN CHECKPOINT: before each edit or test block, check whether ")
		b.WriteString(requestPath)
		b.WriteString(" exists.\n")
		b.WriteString("If it exists, read that JSON, stop at a consistent point, do not expand scope and write ")
		b.WriteString(ackPath)
		b.WriteString(" with schema codex_shutdown_checkpoint_ack.v0, run_ref, agent_ref, checkpoint_ref and status checkpoint_ready.\n")
		b.WriteString("After the checkpoint ACK, do not continue long-running work; finish with compact output.\n")
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
	writeClaudePromptListSectionForLocaleV0(b, title, values, "")
}

func writeClaudePromptListSectionForLocaleV0(b *strings.Builder, title string, values []string, locale string) {
	b.WriteString("\n")
	b.WriteString(title)
	b.WriteString("\n")
	values = claudeCompactPromptValuesV0(values)
	if len(values) == 0 {
		if claudeGoalPromptEnglishLocaleV0(locale) {
			b.WriteString("- <none>\n")
			return
		}
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
