<!--
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
-->

# Orquesta v1 — Roadmap de implementacion

## Objetivo del roadmap

Convertir la vision de `OP-049` en un orden de trabajo que reduzca riesgo, preserve datos y permita avanzar con varios agentes en paralelo.

## Principios de ejecucion

1. Ninguna migracion destructiva.
2. Cada bloque debe dejar el sistema mejor que antes.
3. La API y la CLI deben crecer sobre los mismos casos de uso.
4. La concurrencia contra SQLite debe reducirse, no aumentar.
5. Las capacidades de alto riesgo se activan despues de tener trazabilidad y control.
6. La hexagonalizacion es gate: no se debe seguir metiendo logica de negocio nueva en `db/` para acelerar la web, la API o la futura app de escritorio.
7. La delegacion a agentes no puede ser abierta: el trabajo remoto debe salir de especificaciones de funcion cerradas y verificables.
8. `tmux` y el runtime son transporte y observabilidad; la verdad del trabajo debe vivir en estado durable, dispatch y validacion de entrega.

## Bloque 1 — Consolidacion del modelo ya abierto

### Objetivo

Cerrar correctamente la base ya implementada de:

- proyectos
- asignaciones
- sesiones reanudables
- conectores
- locks
- worktrees
- API JSON base

### Entregables

- limpieza de puertos y adaptadores
- repositorios SQLite coherentes
- comandos CLI consistentes
- tests de humo de sesiones, locks y worktrees
- cierre de voto de `OP-050`, `OP-051` y `OP-052` sin duplicidades activas
- congelacion del traslado fisico del repo mientras existan agentes activos en la ruta actual

## Bloque 2 — Hexagonalizacion progresiva

### Objetivo

Sacar del paquete `db` la logica que aun actua como nucleo.

### Prioridad real

Este bloque actua como prerequisito de arquitectura.
Se puede seguir corrigiendo o completando superficie visible, pero no debe crecer la deuda estructural.
Antes de ampliar en serio:

- panel web completo
- control activo de agentes vivos
- app de escritorio

hay que mover casos de uso y reglas de negocio fuera de `db/`.

### Decision operativa añadida durante el bloque

El proyecto ha llegado a una conclusion adicional durante la propia ejecucion del roadmap:

- la delegacion amplia a agentes hace divergir la solucion del roadmap
- esa deriva termina manifestandose como deuda de hexagonalidad y cambios laterales fuera de control

Por eso, este bloque ya no solo exige sacar logica de `db/`.
Tambien exige cambiar el modelo de delegacion: Orquesta debe mandar microtareas cerradas por especificacion de funcion.

### Entregables

- servicios de aplicacion para propuestas, tareas, sesiones y proyectos
- puertos de persistencia claros
- adaptadores SQLite reducidos a persistencia
- BD base neutra: sin siembra de agentes legacy; la flota se registra explícitamente por la app
- `server_autobootstrap` desactivado por defecto y activable solo de forma explícita para flotas oficiales
- modelo canónico de `especificacion de funcion`
- validacion de entregas por `write_set`, firma, imports y tests
- protocolo de worker por `inbox + ACK + claim-safe`
- planificador de microprogramacion dirigida sobre funciones y no sobre frentes amplios
- matriz de evaluacion de modelos locales por perfil canónico (`implementacion`, `revision`, `analisis`)
- regla de promocion: ningun modelo local se aprueba como worker por defecto sin pasar esas pruebas dentro del flujo canonico de microprogramacion
- control de capacidad por `pool` y `slots` para workers locales de Ollama, separando agentes logicos de workers fisicos concurrentes

Frente abierto en Orquesta para esta decision:

- `#505` Definir especificacion de funcion y write-set obligatorio
- `#506` Emitir microtareas desde Orquesta sobre especificaciones de funcion
- `#507` Validar entregas de agentes por firma, write-set y tests obligatorios
- `#508` Endurecer dispatch state-first con ack, receipt y ready gate
- `#509` Restringir a los agentes al modo microprogramacion dirigida
- `#510` Integrar review y merge solo desde entregas validadas
- `#516` Definir pool local Gemma4 con agentes logicos y slots canonicos
- `#517` Implementar conector canonico `ollama_pool_local` compartido
- `#518` Arbitrar slots de pools locales en el scheduler
- `#519` Mantener contexto resumido por agente logico en pools locales
- `#520` Conservar `ollama-cli/tmux` como via experimental compatible
- `#521` Entrega canonica de microprogramacion por `git/worktree`
- `#522` Unificar subagentes programadores bajo entrega `git/worktree`
- `#523` Reutilizar OpenClaw y `supervisor_subagents` con integracion git sin rutas paralelas
- `#524` Definir pipeline canonico por fases (`especificacion`, `implementacion`, `revision`, `correccion`, `integracion`)
- `#525` Introducir scheduler multi-modelo local por `perfil_tarea`, `pool`, `slot`, `keep_alive` y `timeout`
- `#526` Promover Orquesta como orquestador determinista y reducir el LLM de supervisor a excepcion
- `#527` Integrar `Qwen3.5 27B` como worker de especificacion y replanificacion
- `#528` Integrar `Qwen2.5 Coder 32B` como worker canonico de implementacion
- `#529` Integrar `DeepSeek Coder V2` como worker canonico de revision y deteccion de regresiones
- `#530` Mantener `Gemma4 26B MoE` como solver alternativo y fallback controlado
- `#531` Orquestar carga y descarga de modelos pesados de Ollama por fase sin residencia concurrente innecesaria
- `#532` Introducir revision escalonada multi-modelo con segunda opinion opcional (`premium` o alternativa local)
- `#533` Adoptar `dispatch` durable canonico (`pending/notified/delivered/failed`) en la app
- `#534` Desacoplar `send_instruction` de `receipt` y cerrar por entrega valida
- `#535` Introducir `ready gate` canonico antes de inyectar a workers interactivos
- `#536` Unificar `receipt` asíncrono desde transcript, git y materialización validada
- `#537` Persistir y observar la deuda de dispatch/notify/delivery en `status`, web y API
- `#538` Integrar el patrón operativo de `oh-my-codex` en Ollama, Codex, Claude y OpenClaw sin rutas paralelas
- `#539` Reforzar worktree/git como única entrega canónica para agentes programadores
- `#540` Incorporar benchmark continuo de patrones externos (`oh-my-codex`, `mission-control`, `automaker`, `Maestro`, `Aperant`)
- `#541` Introducir contrato canónico de `dispatch_state` en `runtimesapp`
- `#542` Marcar `notified` y `failed` desde la app antes del cierre por entrega
- `#543` Reescribir el runner de `send_instruction` para que no bloquee esperando el resultado del modelo
- `#544` Adoptar `ready gate` canónico inspirado en `oh-my-codex` para conectores interactivos
- `#545` Proyectar deuda de dispatch (`pending/notified/failed`) en `status`, cockpit y web
- `#546` Alinear Ollama, Codex, Claude y OpenClaw con el mismo contrato de dispatch
- `#547` Mantener benchmark vivo de referencias externas y retirar soluciones propias superadas
- `#548` Introducir puerto hexagonal de gestion de modelos runtime (`listar`, `activar`, `detener`, `descargar`)
- `#549` Implementar adaptador Ollama para gestion de modelos runtime
- `#550` Preparar adaptador futuro `vllm` sin tocar el nucleo

Decision adicional de este bloque:

- `dispatch/mailbox/ACK` siguen siendo la capa de control del worker
- `git/worktree` pasa a ser la capa canónica de entrega de codigo
- los formatos `// FILE:` y `PATCH_UNIFICADO` quedan como compatibilidad o rescate, no como objetivo final
- el orquestador principal no pasa a ser otro LLM: el orquestador principal sigue siendo Orquesta
- los LLM pasan a ser workers especializados por fase y no deciden la arquitectura por su cuenta
- el pipeline canónico debe declarar explícitamente su `carril` por fase:
  - `microprogramacion_local` para workers locales/mini con contrato rígido
  - `premium_worktree` para implementación/especificación/corrección premium con `write_set` y worktree aislada
  - `revision_diff` para revisores que trabajan sobre diff, tests y hallazgos estructurados
  - `determinista_app` para pasos resueltos por la propia app sin delegar en un modelo
- la promocion de agentes premium a trabajo real del repo no se hace por disponibilidad del CLI ni por una sola smoke:
  - `Gemini` solo pasa a especificacion real cuando deje diff util y repetible por `premium_worktree`
  - `Codex` solo pasa a implementacion/correccion real cuando mantenga cierre estable del carril premium
  - `Claude` solo pasa a revision real cuando entregue hallazgos estructurados repetibles por `revision_diff`
- con hardware local pesado, la regla por defecto es `un modelo grande activo cada vez`; la concurrencia lógica no autoriza residencia física simultánea de varios modelos pesados
- la fase de `revision` puede usar varios revisores, pero en cadena y con hallazgos estructurados; nunca como debate libre ni como mayoria informal
- se copian sin complejos los patrones operativos mejores de productos externos cuando resuelven mejor el mismo problema; la originalidad no es objetivo de arquitectura

## Bloque 3 — Memoria y conocimiento

### Objetivo

Igualar la parte fuerte de knowledge capture y prepararse para la fase de investigacion.

### Entregables

- `memoria_proyecto`
- `memoria_tarea`
- `fuentes`
- `hallazgos`
- `decisiones`
- API para registrar y consultar conocimiento

## Bloque 4 — Deteccion de deriva

### Objetivo

Detectar cuando una sesion o una rama se desvian de:

- la tarea
- la propuesta
- la fase
- la rama esperada

### Entregables

- modelo de deriva
- verificacion en `tick`
- alertas o bloqueos de integracion

## Bloque 5 — Fases y progreso

### Objetivo

Hacer que proyecto y tarea tengan fase explicita y progreso calculable.

### Entregables

- catalogo de fases
- estado de fase por proyecto
- estado de fase por tarea
- formula de progreso basada en tareas, gates y ramas

## Bloque 6 — Gobierno Git

### Objetivo

Dar visibilidad y control real sobre ramas, commits, push y merge.

### Entregables

- inventario de repositorios por proyecto
- ramas por agente
- snapshots Git
- estado `dirty`, `ahead`, `behind`
- merges orquestados y gates previos

## Bloque 7 — MCP y conectores avanzados

### Objetivo

Servir contexto, reglas y operaciones de Orquesta mediante MCP y reforzar la abstraccion de runtimes.

### Entregables

- servidor MCP local
- `resources`
- `prompts`
- `tools`
- conectores `mcp_stdio` y `mcp_http`

## Bloque 8 — Investigacion como producto

### Objetivo

Convertir la fase de investigacion en una capacidad de primer nivel.

### Entregables

- workflows de investigacion
- registro de fuentes
- sintesis estructurada
- integracion con memoria y planificacion

## Bloque 9 — Operacion manual y runtime

### Objetivo

Cerrar bien el ciclo practico de agentes manuales y semiautomaticos.

### Entregables

- wrappers de consola estables
- persistencia determinista de `external_session_id`
- soporte Terminator
- arranque, pausa y reanudacion desde Orquesta
- control de presupuesto restante por sesion
- checkpoint y relevo preventivo segun politica aprobada

Nota de diseno:

- este bloque ya no puede interpretarse como "dar autonomia abierta a la consola del agente"
- el runtime solo resuelve transporte, continuidad y observabilidad
- la unidad de trabajo que baja al agente debe venir cerrada desde el orquestador

## Bloque 10 — Refactorizacion competitiva

### Objetivo

Permitir experimentos controlados con varias variantes y seleccion rigurosa.

### Entregables

- experimentos de refactor
- candidatos aislados
- backups previos
- scorecards
- revision por al menos dos no autores

## Bloque 11 — Web y app de escritorio

### Objetivo

Hacer usable toda la orquestacion sin depender de la terminal.

### Entregables

- panel de agentes
- panel de proyectos
- panel de propuestas y votos
- panel de Git y merges
- panel de fases y progreso

## Reparto recomendado entre agentes

### Codex1

- documentacion
- arquitectura
- orden de implementacion
- integracion transversal

### Codex2

- MCP
- conectores de runtime
- capa de prompts/resources/tools

### Codex3

- memoria
- fuentes
- hallazgos
- deriva

## Gates por bloque

Cada bloque debe cerrar con:

- build correcto
- tests correctos
- migraciones aditivas verificadas
- documentacion actualizada
- propuesta o voto si hay decision estructural
