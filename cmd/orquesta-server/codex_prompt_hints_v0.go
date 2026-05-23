package main

import (
	"strings"

	orquestaserver "orquesta/modulos/orquesta-server"
)

func codexServerPromptHintsV0(config orquestaserver.ConfigV0) []string {
	baseURL := publicServerBaseURLV0(config.Addr)
	return []string{
		"Puedes usar las APIs operativas de Orquesta expuestas por este servidor si estan vivas: " + baseURL + ".",
		"Toolbelt HTTP: POST /api/v0/autoprogramming/status, /api/v0/autoprogramming/supervise, /api/v0/autoprogramming/prepare-run, /api/v0/autoprogramming/self-improvement, /api/v0/director/human-work/review-plan, /api/v0/runs/supervise, /api/v0/runs/control, /api/v0/runs/queue/priority, /api/v0/director/stats, /api/v0/domain-work, /api/v0/external-work/run.",
		"Toolbelt MCP equivalente: orquesta.autoprogramming.status.v0, orquesta.autoprogramming.supervise.v0, orquesta.autoprogramming.prepare_run.v0, orquesta.autoprogramming.self_improvement.propose.v0, orquesta.director.human_work.review_plan.v0, orquesta.runs.control.v0, orquesta.run_queue.priority.v0, orquesta.director.stats.v0, orquesta.domain_work.v0 y orquesta.operator.operations.v0.",
		"Antes de programar codigo nuevo, busca codigo reutilizable compatible con rg en modulos, cmd, docs, scripts y variantes v0/v1/v2/v3/legacy.",
		"Si una API o entrega falla por forma reparable, normaliza alias/campos/rutas o pide correccion; no descartes trabajo completo salvo seguridad, causalidad, refs imposibles, datos sensibles o efectos externos no autorizados.",
		"Estas APIs son adaptadores de composicion: no metas HTTP, MCP, Codex, OPES, DB, HOME, tokens, proveedor ni rutas locales dentro del nucleo.",
		"Para trabajos amplios, divide por write-set sin pisar a otros agentes; si hay solape, agenda o delega revision/correccion en vez de sobrescribir.",
		"Si detectas una mejora general repetible mientras haces otro trabajo, usa /api/v0/autoprogramming/self-improvement u orquesta.autoprogramming.self_improvement.propose.v0 para proponer automejora separada de baja prioridad con evidencia y write-set propio; no bloquees ni ensucies el trabajo principal.",
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
