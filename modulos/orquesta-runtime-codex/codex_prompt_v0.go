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

func BuildCodexAgentPromptWithControlFilesV0(packet orquestaruntime.AgentStartPacketV0, hints []string, control CodexControlFilesV0) string {
	packetPath := cleanControlPathV0(control.PacketPath, CodexAgentPacketFileNameV0)
	ackPath := cleanControlPathV0(control.AckPath, CodexAgentAckFileNameV0)
	decisionPath := cleanControlPathV0(control.DecisionPath, "")
	shutdownRequestPath := cleanControlPathV0(control.ShutdownRequestPath, "")
	shutdownAckPath := cleanControlPathV0(control.ShutdownAckPath, "")
	var b strings.Builder
	writeStringsV0(&b,
		"Eres un agente externo gobernado por OrquestaV2.\n",
		"Lee ", packetPath, " antes de tocar archivos.\n",
		"Contrato activo: aplica las skill_refs materializadas como reglas de ejecucion; no son consejos opcionales. Si una skill, el objetivo, el write-set o el ACK esperado entran en tension, gana el contrato del paquete y deja la tension en notes sin ampliar alcance.\n",
		"Seguridad: no borres, no muevas fuera, no trunques archivos existentes y no salgas del workdir del proyecto.\n",
		"Si una parte requiere un efecto externo no autorizado, conserva el avance posible dentro del workdir y deja la limitacion como nota o tarea derivada en el ACK.\n",
	)
	writeWriteSetPrecedenceProtocolV0(&b, packet)
	writeStringsV0(&b,
		"Las rutas del write-set son relativas al workdir del proyecto, no al directorio de control ni a .orquesta-runtime; crea docs/codigo en el proyecto.\n",
		"Archivos de control permitidos fuera del write-set: ", ackPath,
	)
	if decisionPath != "" {
		writeStringsV0(&b, ", ", decisionPath)
	}
	if shutdownRequestPath != "" && shutdownAckPath != "" {
		writeStringsV0(&b, ", ", shutdownRequestPath, " y ", shutdownAckPath)
	}
	writeStringsV0(&b,
		".\n",
		"Mantén cada fichero Go por debajo de 300 lineas; divide responsabilidades si se acerca a ese limite.\n",
		"Comunicacion compacta: activa $caveman full si existe; si no existe, usa compact equivalente.\n",
		"PASO FINAL OBLIGATORIO: antes de terminar el turno, escribe SIEMPRE el ACK de control ",
		packet.DeliveryRefs.AckRef,
		" con el status real (completed/blocked/failed), los ficheros del write-set tocados y la evidencia de pruebas. ",
		"No omitas este paso ni lo dejes para luego: aunque el trabajo ya este en disco, sin ACK Orquesta no recibe el acuse causal. ",
		"Escribe el ACK como ultima accion, no antes de terminar el trabajo.\n",
		"Si falta contexto, no inventes: usa lo disponible, guarda el avance y deja la falta como nota de revision en el ACK.\n",
	)
	if codexPacketHasRequiredTruncatedContextV0(packet) {
		b.WriteString("CONTEXTO TRUNCADO: agent_packet.context contiene entradas required=true y truncated=true. Trabaja con refs/materializacion externa cuando este disponible; si no, guarda avance parcial y anota contexto_truncado_pendiente o contexto_truncado_resuelto en notes.\n")
	}
	if codexPacketHasRequiredRefOnlyContextV0(packet) {
		b.WriteString("CONTEXTO REF_ONLY REQUERIDO: agent_packet.context contiene entradas required=true y mode=ref_only. Sigue required_ref_action si ayuda; si no alcanza, guarda avance parcial y anota contexto_ref_only_pendiente o contexto_ref_only_resuelto en notes.\n")
	}
	writeStringsV0(&b,
		"Aplica arquitectura hexagonal e i18n si la tarea genera app o UI.\n",
		"La persistencia concreta solo pertenece a la app generada si la tarea la pide; Orquesta no usa DB por defecto.\n",
		"Ejecuta las pruebas obligatorias que aparezcan en el paquete si son razonables para el workdir.\n",
		"No uses git status como criterio obligatorio; si el workdir no es repositorio git, ignora esa comprobacion.\n",
		"Si los tests obligatorios pasan y los ficheros del write-set existen, escribe ACK status completed aunque git no aplique.\n",
		"No imprimas diffs ni pegues artefactos completos; valida en compacto, escribe agent_ack.json y termina con la linea ACK.\n",
	)
	writeShutdownProtocolV0(&b, shutdownRequestPath, shutdownAckPath)
	writeStringsV0(&b, "Al terminar, escribe ", ackPath, " con schema codex_agent_ack.v0.\n\n")
	writeAckWriteProtocolV0(&b, ackPath)
	if decisionPath != "" {
		writeStringsV0(&b,
			"decision_path: ", decisionPath, " solo para decisiones ejecutables del director; no pertenece al write-set.\n",
			"Es obligatorio solo si objetivo o criterios de cierre lo piden.\n",
		)
		if strings.Contains(strings.ToLower(strings.TrimSpace(packet.TargetModule)), "director") {
			b.WriteString("Como target_module de director, escribe decision_path con las decisiones ejecutables que puedas; si no esta completo, conserva el diagnostico y deja follow-up en ACK.\n")
		}
		b.WriteString("Si escribes decision_path, completa antes los ficheros pedidos del write-set, despues escribe ACK y termina; no sigas pensando ni ampliando alcance.\n")
	}
	writeStringsV0(&b,
		"En el ACK, files debe listar rutas reales de archivos de producto tocados, no globs, directorios ni el write-set completo; ejemplo cmd/server/main.go, no cmd/server/**.\n",
		"No incluyas archivos de control en ACK.files: agent_ack.json, director_decisions.json, agent_packet.json, prompts, logs ni checkpoints.\n",
		"Si detectas que algun archivo necesario queda fuera del write-set, no lo edites; continua con lo permitido y registra el faltante como nota o tarea derivada.\n",
		"En el ACK, tests debe listar solo pruebas pasadas; cada test obligatorio pasado debe aparecer exactamente como aparece en el paquete.\n",
		"Si necesitas preparar entorno para ejecutar una prueba obligatoria, hazlo antes o mediante export en la shell; en tests y test_receipts.command conserva el comando obligatorio exacto del paquete, sin prefijos de entorno, wrappers ni sufijos explicativos.\n",
		"Si una prueba obligatoria falla, conserva la evidencia en test_receipts/notes y no la declares como pasada.\n",
		"En modo estricto, anade test_receipts por cada test pasado con comando exacto, status passed, exit_code 0, evidence_refs compactas, occurred_at, sequence y output_redacted=true.\n",
		"No incluyas HOME real, tokens, secretos, prompts, completions ni transcripts completos.\n",
		"files y tests deben ser arrays de strings; no metas stdout/stderr crudo en test_receipts ni notes.\n\n",
		"ACK esperado:\n",
		"{\"schema_version\":\"codex_agent_ack.v0\",\"request_id\":\"", packet.RequestID,
		"\",\"correlation_id\":\"", packet.CorrelationID,
		"\",\"ack_ref\":\"", packet.DeliveryRefs.AckRef,
		"\",\"target_module\":\"", packet.TargetModule,
		"\",\"task_ref\":\"", packet.Task.TaskRef,
		"\",\"status\":\"completed\",\"files\":",
	)
	writeStringsV0(&b, promptJSONStringArrayV0(promptACKFilesV0(packet)), ",\"tests\":", promptJSONStringArrayV0(packet.Task.RequiredTests))
	if len(compactPromptValuesV0(packet.Task.RequiredTests)) > 0 {
		writeStringsV0(&b, ",\"test_receipts\":", promptACKTestReceiptsJSONV0(packet.Task.RequiredTests))
	}
	writeStringsV0(&b, ",\"notes\":", promptJSONStringArrayV0(promptACKNotesV0(packet)), "}\n\nTitulo: ", packet.Task.Title, "\nObjetivo: ", packet.Task.Objective, "\n")
	writePromptListSectionV0(&b, "Write-set permitido:", packet.Task.WriteSet)
	writePromptListSectionV0(&b, "Tests obligatorios:", packet.Task.RequiredTests)
	writePromptListSectionV0(&b, "SkillRefs solicitadas:", packet.Task.SkillRefs)
	writeCodexSkillRefsMaterializedV0(&b, packet.Task.SkillRefs)
	if len(hints) > 0 {
		b.WriteString("\nNotas operativas del conector:\n")
		for _, hint := range hints {
			hint = strings.TrimSpace(hint)
			if hint == "" {
				continue
			}
			writeStringsV0(&b, "- ", hint, "\n")
		}
	}
	return b.String()
}

func writeStringsV0(b *strings.Builder, values ...string) {
	for _, value := range values {
		b.WriteString(value)
	}
}

func writeWriteSetPrecedenceProtocolV0(b *strings.Builder, packet orquestaruntime.AgentStartPacketV0) {
	if orquestaruntime.AgentStartPacketWriteSetClosedV0(packet) {
		b.WriteString("Usa el write-set del paquete como alcance de escritura; si contiene '.', el alcance es todo el repo del proyecto, pero sigue prohibido borrar o salir del workdir.\n")
		b.WriteString("Si falta alcance, no edites fuera: conserva lo util dentro del write-set y registra el faltante como nota de revision o tarea derivada.\n")
		b.WriteString("La unica excepcion son archivos de control indicados por Orquesta.\n")
		return
	}
	b.WriteString("MODO COMPATIBILIDAD LEGACY: usa el write-set como alcance primario y no amplíes alcance cuando haya riesgo de causalidad, seguridad o efectos externos.\n")
}

func writeAckWriteProtocolV0(b *strings.Builder, ackPath string) {
	if !filepath.IsAbs(ackPath) {
		return
	}
	writeStringsV0(b, "PROTOCOLO ACK FUERA DEL PROYECTO: ", ackPath, " es fichero de control, no artifact del proyecto. No uses apply_patch para escribirlo. Escribelo con shell desde su directorio de control: cd ", filepath.Dir(ackPath), " && crear ", filepath.Base(ackPath), ". No crees docs/codigo en ese directorio de control. La linea visible ACK no sustituye este JSON.\n\n")
}

func writeShutdownProtocolV0(b *strings.Builder, requestPath string, ackPath string) {
	if requestPath == "" || ackPath == "" {
		return
	}
	writeStringsV0(b,
		"CHECKPOINT DE APAGADO: antes de cada bloque de edicion o prueba comprueba si existe ", requestPath, ".\n",
		"Si existe, lee ese JSON, para en un punto consistente, no amplias alcance y escribe ", ackPath, " con schema codex_shutdown_checkpoint_ack.v0, run_ref, agent_ref, checkpoint_ref y status checkpoint_ready.\n",
		"Despues del ACK de checkpoint no sigas ejecutando trabajo largo; termina con salida compacta.\n",
	)
}

func promptJSONStringArrayV0(values []string) string {
	data, _ := json.Marshal(compactPromptValuesV0(values))
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
		value = strings.TrimSpace(value)
		if value == "" || value == "." || strings.ContainsAny(value, "*?[") ||
			strings.HasSuffix(value, "/") {
			continue
		}
		base := pathpkg.Base(value)
		if strings.Contains(base, ".") {
			files = append(files, value)
			continue
		}
		switch strings.ToLower(base) {
		case "makefile", "readme", "license":
			files = append(files, value)
		}
	}
	if len(files) == 0 {
		return []string{"<path-real-tocado>"}
	}
	return files
}

func writePromptListSectionV0(b *strings.Builder, title string, values []string) {
	compact := compactPromptValuesV0(values)
	if len(compact) == 0 {
		return
	}
	writeStringsV0(b, "\n", title, "\n")
	for _, value := range compact {
		writeStringsV0(b, "- ", value, "\n")
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

func writeCodexSkillRefsMaterializedV0(b *strings.Builder, refs []string) {
	materialized := codexMaterializedSkillRefsV0(refs)
	if len(materialized) == 0 {
		return
	}
	b.WriteString("\nSkills materializadas:\n")
	for _, skill := range materialized {
		writeStringsV0(b, "- ", skill.Ref, ": ", skill.Text, "\n")
	}
}

type codexMaterializedSkillV0 struct{ Ref, Text string }

func codexMaterializedSkillRefsV0(refs []string) []codexMaterializedSkillV0 {
	seen := map[string]bool{}
	result := []codexMaterializedSkillV0{}
	for _, ref := range compactPromptValuesV0(refs) {
		key := strings.ToLower(strings.TrimSpace(ref))
		text, ok := codexSkillRefTextV0(key)
		if !ok || seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, codexMaterializedSkillV0{Ref: ref, Text: text})
	}
	return result
}

func codexSkillRefTextV0(ref string) (string, bool) {
	switch ref {
	case "skill-ref-orquesta-programacion-v0", "skill-ref-orquesta-programacion-autonoma-v0":
		return "Lee AGENTS local, declara write-set estrecho, divide si procede, integra sin pisar cambios ajenos, prueba focal y conserva avances recuperables como evidencia o rework.", true
	case "skill-ref-orquesta-programacion-integracion-v0":
		return "Mantén hexagonal: nucleo neutral, puerto pequeno, adaptador opt-in, configuracion canonica en composicion y tests de frontera/fake.", true
	case "skill-ref-orquesta-programacion-web-app-v0", "skill-ref-orquesta-programacion-web-local-v0", "skill-ref-orquesta-programacion-ui-v0", "skill-ref-orquesta-web-app-v0", "skill-ref-orquesta-ui-v0":
		return "Web app/UI: cliente fino sobre API/MCP; respeta estilo existente, i18n, accesibilidad, responsive, assets cargados y validacion local.", true
	case "skill-ref-orquesta-api-rest-v0", "skill-ref-orquesta-programacion-api-rest-v0", "skill-ref-orquesta-programacion-rest-api-v0":
		return "API REST: contratos versionados, handlers finos, errores publicos estables, refs opacas, idempotencia y tests de handler sin filtrar tokens ni rutas locales.", true
	case "skill-ref-orquesta-mcp-v0", "skill-ref-orquesta-programacion-mcp-v0":
		return "MCP: tools/resources versionados como adaptador fino sobre puertos, schemas publicos compactos, errores JSON-RPC saneados y tests sin leer stores privados.", true
	case "skill-ref-orquesta-api-rest-mcp-v0", "skill-ref-orquesta-programacion-api-rest-mcp-v0":
		return "API REST/MCP: contratos versionados, errores publicos estables, refs opacas, wiring opt-in y pruebas de handler/tool sin filtrar tokens ni rutas locales.", true
	case "skill-ref-orquesta-datos-persistencia-v0", "skill-ref-orquesta-programacion-datos-persistencia-v0", "skill-ref-orquesta-programacion-data-persistence-v0", "skill-ref-orquesta-persistencia-v0":
		return "Datos/persistencia: dominio por puertos; DB/files concretos solo en composicion/app opt-in, configuracion canonica, migraciones/tests propios, sin DB por defecto en Orquesta.", true
	case "skill-ref-orquesta-opes-dominio-v0", "skill-ref-orquesta-opes-consumidor-v0", "skill-ref-orquesta-domain-work-opes-v0", "skill-ref-opes-dominio-v0":
		return "OPES queda como dominio/adaptador consumidor: reglas, validadores y ensamblado fuera del nucleo; efectos reales solo con conector opt-in, scope seguro y sin filtros por palabras.", true
	case "skill-ref-opes-protocolo-agentes-compactos-v0":
		return "OPES compacto: caveman/compact si existe, contexto por refs, lotes para Gemini/Claude, write-set declarado, ACK breve y normalizacion recuperable sin vetos por palabras.", true
	case "skill-ref-opes-tests-4-respuestas-v0":
		return "OPES tests: reutiliza primero, respeta pregunta humana, exige 4 opciones con una correcta y 3 distractores utiles sin duplicidades, explicacion y revision pregunta/opcion.", true
	case "skill-ref-opes-html-web-uso-v0":
		return "OPES HTML USO/TCAE: curso local con canon visual existente, resumen/ampliado, bases, tests, aula, tutor/RAG, audio, imagenes, responsive y capturas antes de publicar.", true
	case "skill-ref-opes-paquete-api-produccion-v0":
		return "OPES produccion: primero paquete local validado, SHA/backup, subida por API con slug confirmado, no pisar cursos ajenos y rollback documentado.", true
	case "skill-ref-opes-qa-final-curso-v0":
		return "OPES QA final: validar programa, reutilizacion, texto, tests, audio, visuales, HTML, RAG, juegos, paquete, capturas y revisiones Codex/Gemini/Claude antes de apto.", true
	case "skill-ref-orquesta-runtime-v0", "skill-ref-orquesta-runtime-modelos-v0":
		return "Runtime/modelos quedan en adaptadores opt-in: proveedor, HOME, tokens, base_url y modelo no entran en core ni payload publico; exponer acciones por puertos.", true
	case "skill-ref-orquesta-programacion-release-v0":
		return "Antes de release revisa worktree, separa cambios ajenos, ejecuta diff check y pruebas, no borres sin evidencia, commit/push solo si el operador lo pidio.", true
	case "skill-ref-orquesta-programacion-tests-v0":
		return "Valida en orden: prueba focal, frontera si cambia contrato, integracion si cambia wiring, git diff --check y suite amplia si el cambio es transversal.", true
	case "skill-ref-orquesta-revision-v0", "skill-ref-orquesta-programacion-revision-v0":
		return "Revisa primero bugs/regresiones, arquitectura, tests faltantes, seguridad y mantenibilidad; findings con ruta/linea, severidad, razon y prueba esperada.", true
	case "skill-ref-orquesta-director-agentes-v0":
		return "Como Director reparte tareas, limita contexto por agente, conserva entregas recuperables, pide rework causal y cierra solo con evidencias y pruebas.", true
	case "skill-ref-orquesta-ordenacion-trabajo-v0":
		return "Convierte objetivo en backlog, plan, olas, tasks, agentes, artefactos, revision, tests y cierre; espera por refs concretas cuando existan.", true
	case "skill-ref-orquesta-artefacto-modular-v0":
		return "Crea artefactos enchufables con manifest, refs opacas, assets locales, i18n/accesibilidad si hay UI y validacion local antes de entrega.", true
	case "skill-ref-orquesta-revision-consejo-votacion-v0":
		return "Organiza candidatos, critica y votos como insumo; el Director decide, fusiona o pide rework sin descartar trabajo recuperable por formato.", true
	default:
		return "", false
	}
}
