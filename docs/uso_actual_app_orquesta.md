<!--
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
-->

# Uso actual de la app Orquesta

## Objetivo

Explicar como se usa hoy Orquesta sin volver a presentar la operativa V1 como
camino vigente del servidor actual.

Este documento queda clasificado como manual server-first sincronizado. Las
secciones historicas preservadas mas abajo sirven solo como contexto de
compatibilidad y no deben alimentar clientes nuevos, tareas de autoprogramacion
ni pruebas de cierre.
La superficie viva se limita a las rutas versionadas y comandos de esta primera
seccion. Cualquier ruta, comando o politica que aparezca despues de
`Contenido historico V1 preservado` queda en cuarentena historica aunque su texto
original use presente.

## Uso vigente server-first

El servidor residente es la fuente operativa. Web, CLI, API HTTP y MCP son
clientes finos sobre contratos publicos; no leen DB, stores internos,
runtime dirs ni ficheros de control como sustituto del servidor.

Arranque vigente:

```bash
go run ./cmd/orquesta-server run
```

Daemon:

```bash
go run ./cmd/orquesta-server daemon start
```

El puerto por defecto lo fija la composicion `orquesta-server`; en esta rama se
usa `127.0.0.1:8787` salvo configuracion explicita.

Salud y readiness:

- `GET /healthz`: liveness de proceso. No significa que Orquesta este lista
  para lanzar trabajo.
- `GET /api/v0/server/readiness`: readiness operativa. Scripts, smokes y
  operadores deben esperar esta ruta antes de preparar o drenar trabajo.
- `GET /api/v0/server/status`: estado publico compacto y configuracion efectiva
  redactada. `/api/status` queda solo como alias legacy de compatibilidad.

Web vigente:

- `/nueva-app`: pedir app por Director/Goal-first mediante
  `POST /api/v0/apps/director` cuando el puerto esta inyectado. `SolicitarNuevaApp`
  / `POST /api/v0/apps/spec` queda como fallback de spec/backlog preview sin
  agentes.
- `/director-stats`: estadisticas/progreso por `run_ref`.
- `/run-queue`: cola multiapp y prioridad.
- `/run-control`: pausa, reanudacion, parada o cancelacion por run.
- `/ops`: panel operativo de cola, ejecucion, atencion y acciones seguras.

API HTTP versionada:

- `POST /api/v0/apps/spec` (validacion/spec preview; no entrada productiva)
- `POST /api/v0/apps/director`
- `POST /api/v0/apps/{app_ref}/changes`
- `POST /api/v0/director/stats`
- `POST /api/v0/director/human-work/review-plan`
- `POST /api/v0/runs/supervise` (compatibilidad legacy/diagnostico; no usar
  como avance normal de runs `goal_first`; el supervisor global sin `run_ref`
  exige opt-in legacy en la composicion Codex; con `run_ref` legacy exige
  `director_execution_mode=legacy_director_loop` salvo opt-in de composicion)
- `POST /api/v0/runs/control`
- `POST /api/v0/runs/queue/priority`
- `POST /api/v0/autoprogramming/validate-request`
- `POST /api/v0/autoprogramming/self-improvement`
- `POST /api/v0/autoprogramming/prepare-run` (Goal-first por defecto; la
  preparacion de run legacy requiere opt-in de composicion y
  `director_execution_mode=legacy_director_loop`)
- `POST /api/v0/apps/director/goal/observe`
- `POST /api/v0/autoprogramming/goal/observe`
- `POST /api/v0/autoprogramming/status`
- `POST /api/v0/autoprogramming/supervise` (compatibilidad legacy/diagnostico;
  no usar como avance normal de runs `goal_first`; `/autoprogramming/status`
  solo publica acciones hacia este endpoint si la composicion habilita legacy y
  sus payloads seguros incluyen `director_execution_mode=legacy_director_loop`)
- `POST /api/v0/governance/catalog/query`
- `POST /api/v0/core/function-contracts/list`
- `POST /api/v0/core/function-contracts/view`
- `POST /api/v0/operational-status/query`
- `POST /api/v0/domain-work`
- `POST /api/v0/external-work/run` (Goal-first para trabajo externo nuevo; el
  fallback legacy exige composicion opt-in y
  `director_execution_mode=legacy_director_loop`)
- `POST /api/v0/ops/agent-runtime-detail`
- `POST /api/v0/server/shutdown`
- `GET /api/v0/server/resources`
- `GET /api/v0/workspace/timeline`

CLI vigente:

- `app spec solicitar`: cliente fino de `POST /api/v0/apps/spec`.
- `app spec bootstrap`: cuarentena legacy; informa `route_policy` y remite a
  `/api/v0/apps/director`.
- `doctor contratos`: `POST /api/v0/operational-status/query`.
- `contratos funcion listar|ver`: rutas read-only
  `/api/v0/core/function-contracts/list|view`.
- `gobernanza catalogo listar|ver`: `POST /api/v0/governance/catalog/query`.
- Autoprogramacion: `prepare-run` y `status`; si la respuesta trae
  `goal_ref`/`run_ref` goal-first, continuar por `goal/observe`. Crear un run
  legacy desde `prepare-run` es compatibilidad historica: requiere opt-in de
  composicion y `director_execution_mode=legacy_director_loop` en el payload.
  `supervise` queda para runs legacy/resident sin `GoalWorkStateV0` o para
  diagnostico controlado y requiere la misma marca cuando se empuja un
  `run_ref` legacy. En `/ops` y MCP, las acciones seguras publicadas por
  `autoprogramming/status` son la fuente de verdad: `observe_goal` gana a
  `supervise` cuando un run tiene estado Goal.

MCP/toolbelt vigente para IA cuando el transporte esta disponible:

- `orquesta.apps.arrancar_director.v0`
- `orquesta.autoprogramming.prepare_run.v0`
- `orquesta.autoprogramming.status.v0`
- `orquesta.autoprogramming.observe_goal.v0`
- `orquesta.autoprogramming.supervise.v0`
- `orquesta.autoprogramming.self_improvement.propose.v0`
- `orquesta.director.stats.v0`
- `orquesta.runs.supervisor.v0`
- `orquesta.runs.control.v0`
- `orquesta.run_queue.priority.v0`
- `orquesta.domain_work.v0`
- `orquesta.external_work.run.v0` (trabajo externo Goal-first; compatibilidad
  legacy solo con opt-in y `director_execution_mode=legacy_director_loop`)

No vigente como requisito operativo:

- `./orquesta serve`: forma V1; usar `cmd/orquesta-server`.
- rutas `/api/*` sin `/api/v0`: historicas o aliases legacy, no contrato nuevo.
- OpenClaw como requisito de notificaciones: historico/externo, no composicion
  vigente del nucleo.
- AP-077 como politica vigente de esta composicion: contexto historico de DBV1,
  no permiso para leer o mutar persistencia local.
- wrappers `scripts/inicio_agente.sh` y Terminator: rescate manual, no runtime
  paralelo ni fuente de verdad.
- SQLite/DBV1, `cmd/db/internal` y `ensureLocalDB`: no se reintroducen.

Contrato T121 para wrappers manuales:

- `scripts/inicio_agente.sh`, `scripts/cargar_agentes.sh`,
  `scripts/terminator_agentes.sh` y `scripts/agente_console.sh` son recuperacion
  u operacion asistida.
- Exigen servidor residente listo por `/api/v0/server/readiness` antes de
  actuar, salvo `--dry-run`.
- Delegan en CLI/API publica; no mutan DB, stores, worktrees ni runtime por una
  ruta lateral.
- `cargar_agentes.sh` y `terminator_agentes.sh` son dry-run por defecto y no
  arrancan flotas legacy sin `--confirm`.
- `agente_console.sh` rechaza comandos de runtime directo; cualquier runtime
  vivo debe estar gobernado por OrquestaV2.
- La correlacion durable usa `agent_ref`, `task_ref`, `run_ref`,
  `external_session_id` y `worktree_ref` como refs opacas, sin publicar rutas
  privadas.

## Contenido historico V1 preservado

El contenido siguiente se conserva para trazabilidad. No es manual vigente, no
declara endpoints publicos nuevos y no debe generar tareas de codigo salvo
compatibilidad o rescate explicitamente autorizados. Si contradice la seccion
server-first anterior, gana la seccion server-first y las fuentes vigentes:
`AGENTS.md`, `docs/estado_actual_2026-05-17.md` y
`docs/guia_nucleo_orquestacion_2026-05-17.md`.

## Historico V1: capas actuales

Orquesta se usa hoy por tres vías complementarias, pero con un único plano operativo válido:

1. Servicio/daemon
   Es la fuente de verdad operativa para sesiones, tareas, propuestas, votos y control de agentes.

2. Web
   Es un cliente HTTP sobre el servicio, con dashboard y acciones de gestión.

3. CLI y API HTTP/JSON
   Actúan como clientes del servicio. El modo local queda solo para recuperación explícita.

## Historico V1: arranque del panel web

```bash
cd ~/Trabajo/orquesta
./orquesta serve
```

Por defecto queda en:

```text
http://127.0.0.1:16543
```

## Historico V1: que ofrecia la web

Rutas HTML disponibles:

- `/`
  Dashboard.
- `/tareas`
  Lista, alta, detalle y acciones de tareas.
- `/propuestas`
  Lista, alta, detalle y acciones de propuestas.

La web actual sirve para:

- ver progreso general
- ver tareas en progreso
- crear tareas
- reasignar o cambiar estado de tareas desde sus vistas
- crear propuestas
- votar o cerrar propuestas desde sus vistas

## Historico V1: que ofrecia la API

Rutas JSON principales:

- `/api/status`
- `/api/agentes`
- `/api/agentes/{agente}/overview`
- `/api/reglas`
- `/api/skills`
- `/api/workflows`
- `/api/proyectos`
- `/api/proyectos/{slug}/cockpit`
- `/api/conectores`
- `/api/asignaciones`
- `/api/asignaciones/activar`
- `/api/locks`
- `/api/tareas`
- `/api/propuestas`
- `/api/worktrees`
- `/api/repos/materializar`
- `/api/repos/revisar`
- `/api/repos/mejorar`
- `/api/audit`
- `/api/runtime-transcript`
- `/api/sesiones/inicio`
- `/api/sesiones/guardar`
- `/api/sesiones/fin`
- `/api/sesiones/continuar`
- `/api/agente/preparar`
- `/api/agente/tick`

La API ya cubre más superficie que la web.
Por eso, mientras la interfaz gráfica no llegue a todo, la combinación correcta es:

- servicio/daemon como fuente de verdad
- web y CLI como clientes del servicio
- API para integración con escritorio, automatizaciones y control plane
- OpenClaw Gateway como adaptador saliente opcional de notificaciones, configurado por `openclaw_gateway_url`, `openclaw_gateway_token` y `openclaw_gateway_operator`

Capacidades nuevas ya utilizables en el estado actual:

- `repo add`, `repo revisar` y `repo mejorar` por carril server-first
- `repo mejorar --finish-app --autonomia-persistente` para sembrar un frente de cierre de app con policy durable
- `orquesta agente actividad <agente> --desde <ventana>` para actividad temporal por agente
- cockpit de control total por proyecto para ver tareas, agentes activos, mailbox pendiente, drift y gobernanza sin consultar tablas manualmente
- autonomia persistente con supervisor residente, reviewer reservado, workers reales acotados y `autonomyPending=0` como foto operativa estable

Consultas de briefing de agentes ya cubiertas por API:

- `GET /api/reglas?tipo_agente=programador`
- `GET /api/skills?agente=Codex1`
- `GET /api/workflows?tipo_agente=programador`
- `GET /api/workflows?tipo_agente=programador&nombre=inicio-sesion`

Estado actual del catalogo de skills:

- indice ordenado por `prioridad`, `escenario` y `nombre`
- metadata minima ya expuesta para skills: `escenario`, `prioridad`, `aliases_json`, `herramientas_json`
- mutaciones versionadas y auditadas
- anti-duplicado funcional por equivalencia canonica, no solo por nombre exacto

## Historico V1: flujo para un agente manual

Entrada manual de compatibilidad o recuperación:

```bash
scripts/inicio_agente.sh <agente>
```

Ejemplos:

```bash
scripts/inicio_agente.sh Codex2
scripts/inicio_agente.sh Codex3 --tarea 155
scripts/inicio_agente.sh antigravity --proyecto orquestador --no-auto
```

Ese wrapper hace:

1. `orquesta sesion inicio`
2. muestra tareas activas del agente
3. inicia la tarea si se le pasa o si hay una única candidata clara

## Historico V1: regla practica para documentadores

Los agentes documentadores como `antigravity` deben usar:

- la web para seguir estado general
- la CLI para iniciar sesión, tomar tarea y votar
- la API solo cuando se documente o se pruebe integración

## Historico V1: limitaciones registradas

- la web todavía no cubre todo el modelo de proyectos, conectores y control activo de agentes
- el arranque autónomo persistente ya opera con supervisor residente; los scripts manuales siguen existiendo como compatibilidad y rescate
- una instalación nueva ya no debe levantar flota legacy por seed implícito: para pruebas locales de Ollama o flotas específicas, los agentes se registran explícitamente desde la app/API
- la app de escritorio aún no existe como producto terminado
- parte del gobierno operativo sigue pasando por CLI y scripts
- el control total ya cubre agente/proyecto; lo pendiente queda en la agregación global del workspace y en la homogeneización global de coste/tokens

## Historico V1: estado de Ollama local

La política vigente para agentes locales de Ollama es dual:

1. Vía experimental y de compatibilidad
   - `agente` -> `ollama-cli` -> `tmux`
   - útil para smokes, depuración, comparación de candidatos y rescate

2. Vía canónica objetivo
   - `pool local`
   - `slots`
   - agentes lógicos por `perfil_tarea`
   - microprogramación dirigida servida por la app

Regla operativa:

- la vía experimental no desaparece de golpe
- pero la promoción de modelos locales y el camino preferente de producción deben moverse al pool compartido gobernado por la app
- con recursos actuales, `gemma4:26b` es el worker local preferente y Qwen queda en estado experimental mientras no supere la smoke canónica de Orquesta

## Historico V1: Politica de Acceso a Persistencia (AP-077)

AP-077 queda preservada como politica historica de DBV1 y no como requisito
vigente de la composicion server-first. No autoriza ni exige acceso directo a
persistencia local para clientes nuevos.

No se permite el acceso directo a la base de datos (p. ej. mediante `sqlite3`) para realizar mutaciones o escrituras en el flujo normal de trabajo. 

La lectura e inspección directa es excepcional y solo se tolera mientras la CLI/API no proporcione la observabilidad y administración necesarias. Cuando la cobertura sea total, el acceso externo quedará bloqueado. Para más detalles, ver [Política de Acceso a Persistencia (ES)](politica_acceso_persistencia_es.md) y [Persistence Access Policy (EN)](politica_acceso_persistencia_en.md).

## Historico V1: estado objetivo

La dirección de producto es:

- misma lógica de negocio para CLI, web y API
- control centralizado de agentes vivos e integridad de los datos
- política de acceso a la persistencia (AP-077) integrada en la operativa
- menos pasos manuales
- documentación completa ES/EN en ficheros separados para todos los proyectos gobernados por Orquesta
Este script ya no es la vía operativa principal de Orquesta. Se conserva para recuperación, compatibilidad y operación manual controlada mientras el servicio completa el gobierno extremo a extremo de runtimes vivos.
