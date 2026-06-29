# Mapa de autoprogramacion complementaria 10x6 - 2026-05-26

Este documento es una guia de estructura para una ola complementaria de
autoprogramacion. No declara cerrada ninguna capacidad por si solo: solo ordena
la lectura, los frentes y las secciones que deben usar los agentes antes de
editar codigo.

## Autoridad

La lectura canonica sigue este orden:

1. `AGENTS.md` y `docs/estado_actual_2026-05-17.md`.
2. `docs/guia_nucleo_orquestacion_2026-05-17.md`,
   `docs/corte_cierre_generico_director_operativo_2026-05-17.md` y
   `docs/matriz_pruebas_reales_y_smoke_2026-05-17.md`.
3. `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md` y shards
   declarados ahi.
4. `AGENTS.md` y docs locales del modulo tocado.

## Capacidad objetivo

La ola complementaria debe preservar esta forma:

| Nivel | Objetivo operativo | Guarda |
| --- | --- | --- |
| Padres | Hasta 10 tareas padres visibles en cola/residente. | El servidor usa cola objetivo de automejora; no debe inventar tareas si el backlog no aporta huecos concretos. |
| Hijos | Hasta 6 subagentes por padre cuando aporten paralelismo real. | La delegacion recursiva es opt-in, con parent/child refs, profundidad, fanout, presupuesto y review causal. |
| Cierre | Entregas pequenas con tests focales y ACK compacto. | El Director revisa evidencias; el subagente no declara cierre del arbol. |

Refs de codigo vigentes para contrastar la capacidad:

- `modulos/orquesta-server/config_v0.go`: `DefaultIdleSelfImprovementTargetQueueV0 = 10`.
- `modulos/orquesta-director-operativo/types_v0.go`:
  `MaxOperationalDirectorMaxSubagentsPerAgentV0 = 6`.
- `cmd/orquesta-server/effective_config_v0.go`:
  `ORQUESTA_CODEX_DIRECTOR_MAX_SUBAGENTS_PER_AGENT`.

## Mapa conceptual

```text
backlog vivo
  -> planner residente / capacidad libre
  -> request padre con write-set y tests
  -> Director Operativo: plan, olas, wait, review, tests, replan/cierre
  -> runtime/composicion por puertos
  -> ACK con cambios, pruebas, bloqueos y evidencia durable
```

Separacion de responsabilidades:

| Pieza | Responsabilidad | No debe hacer |
| --- | --- | --- |
| `orquesta-director-operativo` | Contrato puro, guardas, presupuesto y plan de olas. | Ejecutar runtime, leer filesystem productivo o elegir proveedor. |
| `orquesta-app-director-service` | Entrar/reentrar al loop, derivar waits, plan-state y cierre por puertos. | Deducir dominio desde internals externos. |
| `orquesta-orchestration-core` | Workflow tasks, outbox, reviews, tests durables y cierre causal. | Saltarse comandos/eventos idempotentes. |
| `orquesta-server` | Supervisor residente, readiness, statefile y disparo acotado de automejora. | Conocer Codex/OPES como dominio interno. |
| `cmd/orquesta-server` | Composition root y wiring opt-in. | Mover reglas de composicion al nucleo. |
| `orquesta-web` y MCP | Superficies operativas sobre puertos. | Leer stores privados o reconstruir logica del Director. |

## Indice de secciones para una ola 10x6

Cada padre debe elegir una seccion primaria y conservar cambios pequenos.

| Seccion | Area | Entrega esperada |
| --- | --- | --- |
| S1 | Estructura, indice y mapa conceptual. | Documento o ajuste de refs que reduzca ambiguedad sin tocar runtime. |
| S2 | Director Operativo puro. | Guardas o tests focales de plan, alias, fanout, presupuesto y repair policy. |
| S3 | Materializacion y workflow. | Fix pequeno en tasks/outbox/plan-state con test causal. |
| S4 | `app-director-service`. | Reentrada, wait acotado, review/replan o cierre por puerto con test offline. |
| S5 | Autoprogramacion residente. | Cola, scanner, deduplicacion o blockers con test de servidor/cmd. |
| S6 | Servidor residente. | Readiness, status, auditoria publica o supervisor acotado. |
| S7 | Stack Codex/composicion. | Wiring opt-in, ACK/evidencia, tests requeridos o smoke fake acotado. |
| S8 | Web/MCP. | Proyeccion visible, acciones seguras y estados compactos sin acceso directo a stores. |
| S9 | OPES/domain-work. | Solo conectores opt-in y refs opacas; derivados/cierre OPES temporal cerrado funcionalmente, con residuales de calidad editorial, coste y automatizacion larga. |
| S10 | Pruebas, rails y runbooks. | Foco de regresion, rail observado, o runbook de smoke reproducible. |

## Plan de secciones por padre

1. Leer autoridad y docs locales.
2. Declarar write-set estrecho.
3. Buscar evidencia vigente con `rg` antes de editar.
4. Si el hueco ya esta cerrado por matriz/corte, no reabrirlo; registrar solo
   regresion demostrada.
5. Dividir en hijos solo si hay trabajo paralelizable real; maximo 6 por padre.
6. Conservar refs de linaje, presupuesto, write-set y tests en cada entrega.
7. Ejecutar test focal y, si el Director lo exige, la bateria agregada.
8. Cerrar ACK con rutas tocadas, pruebas, resultado y bloqueos.

## Riesgos que deben bloquear o pedir replan

- Conflicto de write-set con cambios vivos de otro agente.
- Necesidad de editar fuera del write-set global autorizado.
- Falta de contexto de dominio en OPES u otra app externa.
- Efecto externo no autorizado, secreto, path sensible o token real.
- Causalidad rota: refs imposibles, task sin parent requerido, wait que se
  amplia a todos los agentes vivos o cierre sin evidencia durable.

## Verificacion recomendada

Para una entrega documental:

```bash
git diff --check
```

Para cambios del objetivo completo de la ola:

```bash
go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-core-workflow ./modulos/orquesta-orchestration-core ./modulos/orquesta-director-operativo ./modulos/orquesta-app-director-service ./modulos/orquesta-app-codex-stack ./modulos/orquesta-autoprogramming ./modulos/orquesta-server ./modulos/orquesta-web
```
