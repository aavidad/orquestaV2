# Autoprogramacion por MCP - 2026-05-23

## Alcance

Este runbook valida la frontera MCP para autoprogramacion y trabajo externo.
MCP queda como adaptador inbound opt-in sobre puertos publicos: no abre runtime,
no lee persistencia productiva y no conoce Codex, OPES, DB, HOME, OAuth,
credenciales, proveedor ni modelo.

Write-set del corte:

- `modulos/orquesta-mcp`
- `modulos/orquesta-operator-mcp`
- `docs/runbooks/autoprogramacion_mcp_2026-05-23.md`

## Contrato publico

Tools MCP relevantes ya publicados por `RegisterMCPTransportV0`:

- `orquesta.autoprogramming.prepare_run.v0`: prepara un run continuable por
  executor inyectado y devuelve `run_ref`, `workflow_task_refs`,
  `wait_agent_refs` y request `continue`.
- `orquesta.autoprogramming.self_improvement.propose.v0`: transforma un fallo
  observado por director/agente en `AutoprogrammingRequestV0` de segundo plano,
  con `priority_score` bajo y `prepare_run` listo para el paso siguiente.
- `orquesta.director.human_work.review_plan.v0`: convierte una orden humana
  amplia en plan revisable y, si procede, en request de prepare-run sin saltarse
  el review.
- `orquesta.runs.supervisor.v0`: supervisa una run concreta o una cola
  inyectada con limites acotados.
- `orquesta.director.stats.v0`: consulta stats compactas y contexto de decision
  por refs opacas.
- `orquesta.server.shutdown.v0`: coordina apagado por caso de uso inyectado,
  RunControl, supervisor y stats.
- `orquesta.domain_work.v0`: crea jobs de dominio o entrega artefactos por
  puertos `DomainWork` inyectados, sin nombrar OPES ni otro producto.
- `orquesta.external_work.run.v0`: crea una run neutral para un trabajo externo
  ya especificado.

Resources MCP relevantes:

- `orquesta.contracts.shared.v0`
- `orquesta.operational.status.v0`
- `orquesta.core_workflow.contracts.v0`
- `orquesta.operator.operations.v0`

## Fronteras

- Los tools MCP delegan en ejecutores o puertos publicos; si falta binding,
  devuelven error publico `mcp_transport_tool_unbound`.
- `orquesta-operator-mcp` define capacidades operativas por refs opacas y
  conectores; no expone tablas, event-store, procesos, PID, prompts ni
  transcripts.
- `domain_work` y `external_work/run` transportan refs opacas y contratos de
  dominio. El dominio externo conserva datos, validadores, persistencia y
  ensamblado.
- OPES, Codex, servidor MCP real, MCPO, red, filesystem productivo y runtime real
  son adaptadores de composicion opt-in fuera de estos modulos.
- `operational_director.task_source: director_decision` pertenece a la cadena
  causal del Director Operativo y no se materializa desde MCP como contrato de
  producto.

## Validacion del corte

Ejecutar desde la raiz del repo:

```bash
go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-operator-mcp
```

Criterios de aceptacion manual:

- `RegisterMCPTransportV0` registra resources compactos y los tools listados.
- `prepare_run`, supervisor, stats, shutdown, `domain_work` y
  `external_work/run` quedan opt-in por puerto o executor inyectado.
- `self_improvement` no encola por si mismo: propone trabajo secundario con
  evidencia y deja la ejecucion a `prepare_run` + cola, manteniendo prioridad
  baja para no bloquear el trabajo principal.
- Las respuestas usan refs opacas, errores publicos y payloads compactos.
- No hay imports de OPES, Codex runtime concreto, DB, HOME, OAuth, credenciales,
  proveedor ni modelo dentro de `orquesta-mcp` ni `orquesta-operator-mcp`.
- La ruta de dominio externo conserva arquitectura hexagonal: Orquesta coordina;
  la app propietaria valida y ensambla.

## Resultado 2026-05-23

Validado con la bateria focal obligatoria del paquete:

- `go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-operator-mcp`
- `validar criterios de aceptacion del cambio`
- `validar contrato externo de dominio`
