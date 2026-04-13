<!--
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
-->

# Benchmark externo — Patrones a copiar

## Objetivo

Este documento fija qué copiamos de orquestadores y runtimes externos que ya funcionan mejor que algunas partes de Orquesta.

La regla ya no es "inventar una solución propia si se nos ocurre una".
La regla pasa a ser:

- copiar primero el patrón operativo ya validado
- adaptarlo a la app hexagonal de Orquesta
- añadir solo lo que Orquesta necesita y el producto externo no cubre

## Fuentes revisadas

- `oh-my-codex`
- `OpenClaw` del propio árbol de Orquesta
- `mission-control`
- `automaker`
- `Maestro`
- `Aperant`
- `ruflo`

## Decisión de arquitectura

Orquesta no se sustituye por otro producto.

Se mantiene Orquesta porque ya tiene:

- modelo de proyectos, tareas, review, merges, sesiones y runtimes
- API server-first
- servicios de app
- validación por `write_set`
- integración `git/worktree`

Lo que sí cambia es el criterio de implementación:

- la parte operativa de workers y dispatch debe copiar patrones probados
- no se aceptan nuevas invenciones en runtime sin una necesidad concreta y tests

## Qué copiamos de cada referencia

## `oh-my-codex`

Copiar:

- `dispatch` durable con estados `pending/notified/delivered/failed`
- separación estricta entre:
  - enqueue
  - notify
  - receipt
  - resultado
- readiness gate antes de inyectar en `tmux`
- `tmux` como transporte, no como verdad
- mailbox/inbox/ack como capa operativa

No copiar:

- su producto completo como sustituto de Orquesta
- su estructura de ficheros como fuente canónica de negocio

Adaptación a Orquesta:

- el estado durable vive en la app y en el plano de control de Orquesta
- la entrega de código se cierra por `git/worktree` o materialización validada
- el cierre de órdenes no puede depender de que el batch de envío siga vivo

## OpenClaw

Copiar:

- su uso como gateway y supervisor observable
- su capacidad de disparar acciones sin secuestrar el hilo principal
- su modelo de supervisor como consumidor de estado, no como fuente de verdad

No copiar:

- convertir OpenClaw en otro plano de control paralelo

Adaptación a Orquesta:

- OpenClaw queda como conector/supervisor
- Orquesta sigue siendo la autoridad de tareas, runtime y merges

## Mission Control

Copiar:

- dashboard de gobierno y operaciones
- observabilidad de agentes, gasto, estado y errores
- separación entre gateway y control operativo

No copiar:

- una segunda app con otra verdad de estado

Adaptación a Orquesta:

- seguir reforzando `status`, cockpit y observabilidad de pools, dispatch y revisión

## Automaker / Maestro / Aperant

Copiar:

- pipelines explícitos por fase
- workers especializados por rol
- worktrees/ramas aisladas
- gates de revisión antes de integrar

No copiar:

- autonomía abierta sin gates fuertes
- flujos donde el LLM planifica y ejecuta sin control determinista de la app

Adaptación a Orquesta:

- Orquesta describe y decide el pipeline
- los modelos ejecutan fases concretas
- la revisión puede ser escalonada y multi-modelo, pero siempre secuencial y estructurada

## Ruflo

Evidencia revisada insuficiente para copiar patrones finos de implementación.

Decisión:

- se mantiene como referencia secundaria
- no se abre código siguiendo ese repositorio hasta estudiar mejor su runtime y su contrato de workers

## Patrón canónico que adopta Orquesta

1. Orquesta decide el siguiente paso del pipeline.
2. Orquesta crea `dispatch` durable.
3. El runtime solo intenta `notify`.
4. La orden pasa a `notified` rápido o a `failed`.
5. El cierre real solo llega por:
   - entrega Git válida
   - materialización válida dentro del `write_set`
   - bloqueo explícito estructurado
6. `tmux`, `ollama`, `codex`, `claude` o `openclaw` son solo conectores de transporte/ejecución.

## Tareas ejecutivas derivadas

- `#541` Introducir contrato canónico de `dispatch_state` en `runtimesapp`
- `#542` Marcar `notified` y `failed` desde la app antes del cierre por entrega
- `#543` Reescribir el runner de `send_instruction` para que no bloquee esperando el resultado del modelo
- `#544` Adoptar `ready gate` canónico inspirado en `oh-my-codex` para conectores interactivos
- `#545` Proyectar deuda de dispatch (`pending/notified/failed`) en `status`, cockpit y web
- `#546` Alinear Ollama, Codex, Claude y OpenClaw con el mismo contrato de dispatch
- `#547` Mantener benchmark vivo de referencias externas y retirar soluciones propias superadas

## Antipatrones ya prohibidos

- batch síncrono que hace a la vez envío, espera, receipt y cierre
- dar por buena una entrega por texto en transcript sin validación
- usar `cmd` como capa de negocio para aplicar código
- crear nuevas rutas paralelas de estado fuera de la app
- usar `db/` como atajo para decisiones de negocio nuevas

## Resultado esperado

Cuando este benchmark esté incorporado, Orquesta debe quedar así:

- orquestador determinista en la app
- workers intercambiables por conector
- dispatch durable
- entrega canónica por `git/worktree`
- revisión estructurada y escalonada
- integración final gobernada por la app y no por el modelo
