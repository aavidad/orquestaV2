package main

import (
	"strings"

	orquestaserver "orquesta/modulos/orquesta-server"
)

func codexServerPromptHintsV0(config orquestaserver.ConfigV0) []string {
	baseURL := publicServerBaseURLV0(config.Addr)
	return []string{
		"Puedes usar las APIs operativas de Orquesta expuestas por este servidor si estan vivas: " + baseURL + ".",
		codexToolbeltHTTPHintV0(),
		codexToolbeltMCPHintV0(),
		"Para contexto de codigo, simbolos o arquitectura usa Orquesta como broker central por POST /api/v0/codebase/query u orquesta.codebase.query.v0; no arranques codebase-memory-mcp propio ni indexadores por agente salvo opt-in central explicito.",
		"En trabajos DomainWork/OPES ya asignados, no crees ni subas jobs manualmente: genera el artefacto en el write-set y el ACK; el bridge de Orquesta ejecuta submit_artifact. Si una tarea pide usar /api/v0/domain-work, el envelope correcto usa job_request para create_job y artifact_submission para submit_artifact.",
		"Antes de programar codigo nuevo, busca codigo reutilizable compatible con rg en modulos, cmd, docs, scripts y variantes v0/v1/v2/v3/legacy.",
		"Si una API o entrega falla por forma reparable, normaliza alias/campos/rutas o pide correccion; no descartes trabajo completo salvo seguridad, causalidad, refs imposibles, datos sensibles o efectos externos no autorizados.",
		"No conviertas palabras genericas como clave, capacity, provider, model, runtime, token, prompt o transcript en bloqueo automatico; solo los secretos con valor, causalidad rota, refs imposibles o efectos externos no autorizados son veto fuerte.",
		"Trabaja compacto: usa $caveman full o equivalente si esta disponible, lee contexto por refs con rg/sed/find, no pegues documentos largos y entrega ACK/final breve con cambios, pruebas, bloqueos y siguiente paso.",
		"Usa razonamiento medium por defecto; reserva xhigh para riesgo tecnico real o instruccion explicita y deja el motivo en evidencia breve.",
		"Si hay corte, presupuesto corto o relevo, escribe checkpoint/handoff durable con estado, archivos tocados, pruebas, bloqueos y proxima accion; conserva trabajo parcial aprovechable.",
		"Estas APIs son adaptadores de composicion: no metas HTTP, MCP, Codex, OPES, DB, directorios de usuario, credenciales, proveedor ni rutas locales dentro del nucleo.",
		"En cualquier app que programes o mejores, centraliza configuracion/env en una unica superficie canonica con nombres, defaults y documentacion; no dupliques variables con nombres distintos repartidas por el codigo.",
		"Para trabajos amplios, divide por write-set sin pisar a otros agentes; si hay solape, agenda o delega revision/correccion en vez de sobrescribir.",
		"Si detectas una mejora general repetible, usa self-improvement con evidencia y write-set propio; auto_prepare_run solo si quieres que el puerto inyectado prepare una run en cola.",
		"Si la revision humana necesita operador, usa review-plan con raise_operator_question u orquesta.operator.directed_query.v0; si falta conector, conserva plan y trata la reparacion como accion publica.",
	}
}

func publicServerBaseURLV0(addr string) string {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		addr = orquestaserver.DefaultAddrV0
	}
	if strings.HasPrefix(addr, "http://") || strings.HasPrefix(addr, "https://") {
		return addr
	}
	if strings.HasPrefix(addr, ":") {
		addr = "127.0.0.1" + addr
	}
	return "http://" + addr
}
