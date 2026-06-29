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
- `#548` Activar `finish_app` persistente con `Codex supervisor` y `Codex reviewer` desde `repo mejorar`
- `#549` Forzar bucle autonomo `until done or hard blocker` con guardas de progreso, CPU y cruce de proyecto
- `#550` Exponer supervisor/reviewer/workers persistentes en CLI, API y notas de tarea sin rutas paralelas
- `#548` Introducir puerto hexagonal de gestion de modelos runtime (`listar`, `activar`, `detener`, `descargar`)
- `#549` Implementar adaptador Ollama para gestion de modelos runtime
- `#550` Preparar adaptador futuro `vllm` sin tocar el nucleo
- `#551` Introducir alta canónica de repositorios (`repo add`) con origen local (`--path`) u origen remoto (`--git`)
- `#552` Materializar repos remotos en copia local canónica antes de abrir pipeline, sesiones o worktrees
- `#553` Unificar `repo revisar` y `repo mejorar` para que operen sobre el mismo proyecto materializado sin distinguir el origen del repo
- `#554` Persistir scorecards mutables de agentes locales por materias canónicas
- `#555` Inicializar bootstrap neutro de score para cada agente local al darse de alta
- `#556` Ponderar la elección del worker local por materia según fase, carril y perfil de tarea
- `#557` Registrar observaciones automáticas tras entregas reales para subir o bajar el fitness local
- `#558` Exponer matriz de scores locales por CLI, API y web operativa
- `#559` Añadir benchmark manual y benchmark continuo para recalibrar agentes locales sin reiniciar su historial
- `#560` Separar score, confianza y número de muestras para evitar promocionar agentes con histórico débil
- `#561` Introducir `shared_context_items` durables y tipados por proyecto/agente para recall selectivo antes del prompt
- `#562` Inyectar contexto compartido selectivo en `prepare/bootstrap` sin compartir transcripts completos entre agentes
- `#563` Exponer contexto compartido por CLI, API y web operativa con filtros por proyecto, agente, tipo y peso
- `#564` Añadir `memory flush` previo a compactacion para promover decisiones, restricciones y hallazgos a memoria durable
- `#565` Consolidar en background señales repetidas de contexto corto a memoria durable de proyecto
- `#566` Definir `fork de funcion` como unidad canonica de refactor competitivo dentro de `repo mejorar`
- `#567` Lanzar variantes aisladas de una misma funcion con varios modelos y `write_set` comun preservando arquitectura de proyecto
- `#568` Comparar forks de funcion por tests, seguridad, invariantes arquitectonicas y benchmark antes de integracion
- `#569` Rechazar automaticamente forks ganadores que degraden hexagonalidad, observabilidad o contratos de capa
- `#570` Exponer en web/API el laboratorio de forks de funcion con candidatos, metricas y revisiones no autor
- `#571` Crear cockpit canónico de proyecto con control total del estado operativo, Git y autonomia
- `#572` Exponer timeline por ventana (`ultima hora`, `24h`, `ciclo actual`) por agente, proyecto y global
- `#573` Añadir API de estadisticas por agente con tareas, runtimes, handoffs, repairs y presupuesto
- `#574` Añadir API de estadisticas por proyecto con fases, porcentaje, flota, hotspots y bloqueos
- `#575` Añadir API global de capacidad, throughput, presupuesto y bloqueos recurrentes
- `#576` Introducir puerto hexagonal de observabilidad/control Git para repos, ramas, worktrees, diffs y merges
- `#577` Exponer ficheros tocados y `insertions/deletions/net` por agente, proyecto, tarea y ventana temporal
- `#578` Calcular hotspots por fichero, carpeta y modulo desde evidencia Git canonica
- `#579` Unificar cockpit, status, OpenClaw y MCP sobre la misma proyección temporal y estadistica
- `#580` Registrar eventos normalizados de codigo, runtime, review, merge y presupuesto para consultas temporales
- `#581` Exponer acciones Git canonicas por API/web/MCP: refresh, checkpoint, review, merge y reconciliacion
- `#582` Hacer visible el coste/tokens/presupuesto por agente, proyecto y global con fuentes y ventanas
- `#583` Forzar que supervisor/reviewer/workers usen politica `token-frugal` con `caveman/compact` cuando el runtime lo permita
- `#584` Añadir vista de trabajo por agente capaz de responder `que ha hecho X en la ultima hora`
- `#585` Añadir vista de estado total de proyecto con `inicio real`, `% fin`, `riesgos`, `bloqueos` y `ultima actividad util`

### Estado real de ejecucion a 2026-04-23

Este bloque ya no esta en fase puramente teórica.
En el repo actual ya hay avance visible en:

- `repo mejorar` con `finish_app` persistente y policy durable
- reserva de supervisor/reviewer y cuota explicita de workers reales
- auto-creacion de trabajo para supervision autonoma
- compactacion de frentes premium y drenaje de `prime`
- `repair-helper` barato como paso previo de recuperacion
- score local y contexto compartido como base de seleccion/mejora de workers
- control total por agente/proyecto ya operativo
- `autonomyPending=0` como foto estable de continuidad drenada

Sigue pendiente dentro del mismo bloque:

- cerrar la proyeccion total global de observabilidad/estadisticas para que la autonomia no dependa de shell ni de lectura parcial de audit/transcript a escala workspace
- rematar la separacion hexagonal de Git/estadisticas/control temporal para que no queden como utilidades de borde

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
- API Git canónica hexagonal
- ficheros tocados y LOC por agente/proyecto/ventana
- hotspots por fichero/modulo
- timeline Git integrada con timeline operativa

## Bloque 6.b — Control total y estadisticas

### Objetivo

Dar control total del estado de proyecto y permitir autonomia supervisada con evidencia completa.

### Entregables

- cockpit canónico de proyecto
- timeline temporal por agente/proyecto/global
- estadisticas completas por agente
- estadisticas completas por proyecto
- estadisticas globales del workspace
- porcentaje real de avance por proyecto
- coste/tokens/presupuesto por ventana y fuente
- correlacion entre tarea, agente, runtime, worktree, diff y merge
- API y MCP capaces de responder preguntas temporales sin shell manual

### Estado real de ejecucion a 2026-04-23

En curso claro:

- cockpit server-first de proyecto ya operativo, aunque todavia ligero
- CLI `agente actividad` ya operativa para ventana temporal por agente
- proyeccion base de audit + transcript + Git stats por repo
- web de proyectos ya consumiendo cockpit por API

Pendiente real antes de declarar este bloque cerrado:

- timeline canonica por ventana en API para proyecto/agente/global
- agregacion Git canonica por proyecto y global, no solo por `cwd`
- porcentaje real de avance con señal fiable
- coste/tokens/presupuesto con fuentes canonicas y agregacion por ventana
- endpoints dedicados de estadisticas equivalentes a la ambicion de `OP 096`

## Bloque 7 — MCP y conectores avanzados

Nota de vigencia 2026-06-29: este bloque es historico. `mcp_stdio` aqui no
autoriza backend Goal stdio ni ruta normal de operacion; la ruta Goal-first
vigente usa `app_server_tmux` y MCP/HTTP solo como adaptadores opt-in.

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
- forks de funcion sobre especificaciones cerradas
- comparacion multi-modelo sobre la misma funcion y el mismo contrato
- preservacion automatica de arquitectura de proyecto durante la mejora

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
