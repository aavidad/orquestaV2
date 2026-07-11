<!--
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
-->

# OP 096 — Control total de estado de proyecto y estadisticas

> **OP historica.** Conserva decisiones y evidencia del control-plane V1, pero
> sus afirmaciones de estado vivo no describen por si solas la composicion
> actual. La foto vigente y la generacion productiva se fijan en `AGENTS.md`,
> `docs/estado_actual_2026-05-17.md` y
> `docs/mapa_generaciones_director_2026-07-03.md`.

## Objetivo

Fijar el contrato operativo para que Orquesta pueda responder, sin shell ni inspeccion manual ad hoc, preguntas como:

- que ha hecho `Codex1` en la ultima hora
- que archivos ha tocado cada agente
- cuantas lineas ha modificado cada agente, proyecto o frente
- cuanto queda para terminar un proyecto
- que ramas, worktrees, sesiones y revisiones estan vivas
- que coste, cuota, tokens y presupuesto se estan consumiendo
- que bloqueos existen y que esta haciendo la autonomia para resolverlos

Esto no es telemetria decorativa. Es un requisito de control total del proyecto.

## Estado real en esta sesion

Avance real ya visible en el repo:

- existe control total server-first por proyecto en `/api/proyectos/{slug}/cockpit`
- existe control total por agente con `orquesta agente actividad <agente>` y ventana temporal, resumen operativo, audit, transcript y estadistica Git por repo/cwd
- existe un servicio de estadisticas Git con `branch`, `touched_files`, `pending/committed added/deleted lines` y commits recientes
- `repo mejorar` ya puede sembrar autonomia persistente de tipo `finish_app` con supervisor residente, reviewer reservado y `max_workers`
- `max_workers` ya cuenta solo workers reales/ejecutores; supervisor y reviewer reservados quedan fuera del cupo
- el resumen operativo ya expone `autonomyPending` y la foto estable que debe documentarse hoy es `autonomyPending=0`
- el control plane ya esta metiendo disciplina de coste: compactacion de frentes, drenaje de `prime` y `repair-helper` antes de escalado caro

Huecos que siguen abiertos:

- no hay todavia una proyeccion global unica que agregue agente + proyecto + workspace con el mismo contrato temporal
- no hay endpoints dedicados de estadisticas globales con contrato estable
- la deuda principal ya no esta en agente/proyecto sino en la agregacion global y en la capa temporal global
- la telemetria de coste/tokens global sigue siendo parcial y heterogenea segun proveedor
- la clasificacion de transcript sigue siendo parcial y todavía ensucia la observabilidad pasiva
- la lectura global de retirados/fuera de orquestacion y `workers_stuck` sigue menos consolidada que la lectura agente/proyecto

Regla de complementariedad aceptada:

- la línea `ADK contracts` y la línea `oh-my-codex style runtime` son complementarias
- `ADK` refuerza evento, delta, artifacts y rewind
- `OMX` refuerza hooks, roles, `worktree + tmux + resume` y perfiles de ejecución
- ambas pueden convivir si la observabilidad canónica y el control plane siguen siendo únicos

## Delta de estado a 2026-04-28

Frentes que ya no deben documentarse con ambigüedad:

- retiro/fuera de orquestación:
  - el mecanismo base ya existe
  - el control plane ya cubre no reactivar retirados, consumir mailbox residual y bloquear tarea huérfana preservando trazabilidad
  - la deuda residual está en la proyección global y en la lectura uniforme de estas incidencias

- degradación `workers_stuck`:
  - ya existe flujo canónico con restart coordinado, continuidad local por `session_resume`, `repair-helper` y cierre del helper
  - el frente abierto aquí es de tuning y observabilidad, no de ausencia de mecanismo

- clasificación de transcript:
  - sigue siendo una deuda abierta
  - el problema material es `classification=''` en `runtime_transcript`, no una clase de dominio persistida `sin_clasificar`
  - la capa de actividad por agente acaba proyectando parte de esa bolsa como `sin_clasificar`

- control total/estadísticas:
  - por agente y por proyecto ya es estado operativo real
  - el plano aún abierto sigue siendo la agregación global temporal y de coste del workspace

Fotografía histórica útil para priorización del transcript:

- `backups/legacy-sqlite-20260422/orquesta.db` contiene `203500` filas en `runtime_transcript`
- `203474` conservan `classification` vacío
- el grueso cae en `pty_out` y parece dominado por ruido TUI/bootstrap no filtrado

## Estado operativo verificable hoy

Superficies ya implementadas y utilizables:

- `GET /api/proyectos/{slug}/cockpit`
  - backlog visible, tareas activas/reservadas, agentes activos, mailbox pendiente, drift de worktrees, propuestas/review gates y contadores de runtime mailbox/orders
- `GET /api/proyectos/{slug}/control`
  - agrega `overview`, `cockpit`, `status`, tareas, porcentaje calculado, fechas `started_at/last_activity_at`, filas operativas por agente y agregado Git del proyecto
- `GET /api/agentes/{agente}/actividad`
  - devuelve detalle de agente, auditoria, transcript, resumen operativo y estadistica Git por ventana; la CLI `orquesta agente actividad <agente>` ya consume ese mismo contrato
- `GET /api/agentes/{agente}/overview`, `GET /api/audit` y `GET /api/runtime-transcript`
  - completan la base observable que usa hoy el control total por agente/proyecto
- `gitestadisticasapp`
  - ya calcula `branch`, `pending_files`, `recent_commit_files`, `touched_files`, `pending/committed added/deleted lines`, `pending_shortstat` y `recent_commit_count`

Lectura correcta del estado actual:

- `control total por proyecto` ya esta operativo
- `control temporal por agente` ya esta operativo
- la capa que sigue pendiente no es local ni por proyecto; es la agregacion global canonica del workspace
- transcript sigue siendo la principal deuda de calidad de señal dentro del control total
- retiro/fuera de orquestación y `workers_stuck` ya pertenecen al estado operativo vigente, no a la lista de ideas futuras

## Principios

1. La observabilidad canonica vive en Orquesta.
   No se reconstruye ad hoc desde shell, `git status`, `tmux` ni consultas manuales a BD.

2. Toda estadistica debe poder proyectarse por:
   - agente
   - proyecto
   - tarea/frente
   - ventana temporal
   - global workspace

3. Git es fuente primaria de evidencia de codigo.
   El diff, los ficheros tocados, las lineas añadidas/eliminadas, la rama y la worktree deben entrar en la telemetria canonica.

4. El progreso no se mide por LOC.
   LOC y diff son evidencia de actividad, no porcentaje de fin.

5. El cockpit de proyecto debe ser accionable.
   Debe servir para gobernar, no solo para mirar.

6. Toda superficie nueva debe nacer hexagonal y preparada para i18n.
   La API de control Git, el cockpit y las estadisticas no pueden acoplarse a un backend concreto ni a textos hardcodeados monolingues.

## Preguntas que Orquesta debe poder responder

### Por agente

- que tarea tiene ahora
- desde cuando trabaja en ella
- tiempo efectivo en ejecucion, tiempo atascado y tiempo en cooldown
- sesiones abiertas, reanudables y cerradas
- runtimes y handles activos, pausados, fallidos y reanimados
- guidance pendiente, guidance confirmada y guidance fallida
- handoffs recibidos, emitidos, completados y fallidos
- repairs lanzados para ese agente y su resultado
- ultimo progreso observable y ultima evidencia de trabajo real
- presupuesto/cuota/tokens por ventana y fuente
- archivos tocados en la ventana consultada
- lineas añadidas, borradas y netas en la ventana consultada
- commits asociados, branch y worktree activa
- reviews pasadas, fallidas y pendientes

### Por proyecto

- cuando se descubrio/registro
- cuando empezo trabajo real
- ultima actividad util
- fase actual y porcentaje de avance calculado
- tareas por estado
- tareas bloqueadas y motivo dominante
- agentes asignados, conectados, trabajando, pausados, retirados y supervisor/reviewer reservados
- worktrees activas, sucias, desfasadas y listas para merge
- ramas activas por agente
- diffs abiertos, gates de review, merges pendientes y merges hechos
- archivos calientes del proyecto en la ventana consultada
- hotspots por modulo/carpeta/fichero
- actividad por hora/dia
- consumo de cuota y presupuesto por pool/modelo/agente
- incidents operativos: reinicios, restart loops, repair-helpers, rehidrataciones y degradaciones

### Global

- estado total del workspace
- proyectos activos, bloqueados, degradados y terminados
- capacidad real disponible por pool
- carga por agente, por proyecto y por frente
- consumo total de presupuesto
- tendencia de throughput, handoffs, repairs y revisiones
- ranking de hotspots, bloqueos recurrentes y agentes mas fiables

## Cockpit canónico de proyecto

Cada proyecto debe exponer un cockpit server-first unico.

### Minimo obligatorio

- identidad: `id`, `slug`, nombre, repo canonico, fecha de alta
- gobernanza: supervisor residente, reviewer reservado, workers reales, policy activa, modo autonomia y estado `autonomyPending`
- progreso: fase actual, porcentaje calculado, hitos y backlog
- actividad: agentes activos, tareas activas, timeline reciente, ultimos eventos
- git: branch base, ahead/behind, dirty, worktrees, diffs abiertos, merges
- calidad: review gates, tests relevantes, ultima integracion valida
- coste: tokens/cuota/presupuesto por agente y agregado
- riesgo: bloqueos, drift, worktrees sucias, handoffs pendientes, repair-helpers vivos

### Timeline

El cockpit debe ofrecer timeline consultable por ventana:

- eventos de tarea
- eventos de runtime
- eventos de git
- eventos de revision/integracion
- eventos de presupuesto
- eventos de autonomia

La pregunta “que ha hecho `Codex1` en la ultima hora en `orquestador`” debe resolverse leyendo esta timeline, no recomponiendo shell manualmente.

Matiz de estado real:

- hoy ya puede resolverse razonablemente por agente/proyecto usando `agente actividad`, `audit`, `runtime-transcript` y estadística Git
- desde el 2026-05-25 existe el contrato inicial
  `WorkspaceTimelineQueryV0` para agregacion global de workspace por puerto
  read-only; las fuentes reales aun pueden responder `not_available` si la
  composicion no inyecta adaptador
- lo que sigue sin cerrarse es la cobertura real completa de fuentes historicas
  y la calidad del transcript pasivo

## Estadisticas Git canónicas

Orquesta debe tener una API de control de versiones Git propia.

### Debe exponer como minimo

- repositorios por proyecto
- ramas y upstream
- worktrees y su relacion con agente/tarea
- snapshots de diff
- estado `dirty`, `ahead`, `behind`
- commits asociados a tarea, agente y proyecto
- ficheros tocados por ventana
- lineas añadidas, borradas y netas por ventana
- hotspots por fichero, carpeta y modulo
- merges propuestos, aprobados, rechazados y ejecutados

### Estadisticas mínimas por diff/commit

- `files_changed`
- `insertions`
- `deletions`
- `lines_net`
- `diff_size`
- `touched_paths`
- `dominant_modules`
- `commit_count`
- `worktree_id`
- `branch`
- `base_ref`

### Regla de integracion

Git no vive como calculo lateral.
Debe entrar por un puerto hexagonal de observabilidad/control Git y proyectarse por CLI, API, MCP y web.

## API objetivo de control total

### Consultas

- `GET /api/proyectos/{slug}/cockpit`
- `GET /api/proyectos/{slug}/timeline`
- `POST /api/v0/workspace/timeline`
- `GET /api/proyectos/{slug}/estadisticas`
- `GET /api/agentes/{agente}/estadisticas`
- `GET /api/git/repos`
- `GET /api/git/repos/{repo}/estado`
- `GET /api/git/repos/{repo}/worktrees`
- `GET /api/git/repos/{repo}/branches`
- `GET /api/git/repos/{repo}/diffs`
- `GET /api/git/repos/{repo}/commits`
- `GET /api/git/repos/{repo}/hotspots`

### Filtros mínimos

- `desde`
- `hasta`
- `agente`
- `proyecto`
- `tarea`
- `branch`
- `worktree`
- `estado`
- `agrupar_por`
- `limit`

### Acciones mínimas

- solicitar refresh Git
- registrar baseline Git
- abrir review
- aprobar/rechazar integracion
- ejecutar merge canónico
- solicitar checkpoint de worktree
- reinyectar tarea/handoff/repair desde cockpit

Estado de implementacion actual frente a esta API:

- implementado ya: `GET /api/proyectos/{slug}/cockpit`
- implementado ya: `GET /api/proyectos/{slug}/control`
- implementado ya para control por agente/proyecto: `GET /api/agentes/{agente}/actividad`, `orquesta agente actividad <agente>`, `GET /api/audit`, `GET /api/runtime-transcript`, `GET /api/agentes/{agente}/overview`
- implementado ya para ingestion/trabajo de repos: `POST /api/repos/materializar`, `POST /api/repos/revisar`, `POST /api/repos/mejorar`
- pendiente: la capa global de `timeline`, `estadisticas` y agregacion canonica unica para workspace
- pendiente: reducir la dependencia de transcript poco clasificado para explicar trabajo real por agente

## Modelo de datos observable

No hace falta una tabla monstruo unica.
Si hace falta un modelo canónico agregable.

### Entidades mínimas a proyectar

- proyecto
- agente
- asignacion
- tarea
- sesion
- runtime
- runtime_handle
- runtime_order
- runtime_mailbox
- worktree
- repo_git
- branch_git
- snapshot_git
- diff_git
- merge_git
- review_gate
- presupuesto
- evento_normalizado

### Regla

Las estadisticas agregadas pueden materializarse en snapshots o vistas de lectura, pero la verdad operacional sigue saliendo de Orquesta y sus puertos.

## Coste mínimo y politica de tokens

La telemetria debe hacer visibles:

- tokens estimados/observados por agente y ventana
- cuota y presupuesto restante por fuente
- coste por tarea/frente cuando exista
- razon de escalado a `prime`
- ahorro aplicado por compresion/briefing compacto

### Politica canónica

- supervisor ligero siempre online
- worker rapido/barato por defecto
- `repair-helper` local y compacto antes que `prime`
- `prime` solo por evidencia operativa o criticidad alta
- workers con salida minima y prompt austero
- preferir `caveman`, `compact` o compresion equivalente cuando el runtime lo permita
- evitar eco de consola, narracion larga y texto no util para validacion

Estado real de politica en el repo:

- esta politica ya no es solo doctrina: hay señales reales de compactacion premium, drenaje de `prime`, workers no-prime preferidos y `repair-helper`
- lo pendiente es cerrar la medicion canonica del ahorro y del coste por agente/proyecto/global

## Autonomia total supervisada

La app debe poder autogestionarse, pero no como bucle ciego.

### Contrato

- supervisor residente y reservado
- workers reales/ejecutores
- reviewer reservado
- repair-helper local rapido para atascos
- escalado a `prime` solo por evidencia
- guardas duras de progreso, CPU, cruce de proyecto y cuota

Regla explicita de cupo:

- `max_workers` mide solo workers reales
- `supervisor` y `reviewer` no consumen ese cupo
- `autonomyPending=0` documenta continuidad drenada, no perdida de autonomia

### La observabilidad es requisito de autonomia

Sin control total del estado de proyecto no hay autonomia fiable.
El supervisor necesita ver:

- quien toca que
- que progreso real existe
- que diff y que archivos se han movido
- si el frente esta atascado o solo pendiente de confirmacion
- cuanto presupuesto queda
- si el handoff o el relevo ha mejorado o empeorado el frente

## Requisitos de UX

- una unica verdad de proyecto en CLI, API, MCP y web
- mismo vocabulario en todas las superficies
- vistas compactas por defecto y detalle bajo demanda
- posibilidad de preguntar por ventana temporal: ultima hora, hoy, ultimas 24h, ultimo ciclo
- poder responder en una sola vista:
  - que se ha hecho
  - que queda
  - quien lo hace
  - donde esta el codigo
  - cuanto se ha tocado
  - que bloquea

## Restricciones de arquitectura

- hexagonal obligatoria
- i18n obligatoria para toda clave y superficie publica nueva
- nada de acoplar la API Git a un proveedor concreto ni a shell como contrato principal
- nada de telemetria solo en logs crudos
- nada de dashboards que inventen otra verdad distinta de la API canonica

## Roadmap mínimo derivado

1. Cockpit canónico de proyecto con timeline y porcentaje real.
2. API de estadisticas por agente/proyecto/global.
3. API Git canónica con diff, ficheros y LOC por ventana.
4. Normalización de eventos para responder preguntas temporales.
5. Hotspots por fichero/modulo y actividad por ventana.
6. Coste/tokens/presupuesto agregados por agente y proyecto.
7. Acciones de control desde cockpit: review, merge, checkpoint, repair, relevo.
8. Convergencia total entre CLI, web, API y MCP sobre la misma proyección.

Priorizacion real a 2026-04-23:

1. extender el control total ya operativo por agente/proyecto al plano global
2. elevar `agente actividad` y `gitestadisticasapp` a API canónica global de estadisticas
3. cerrar agregacion temporal/global y correlacion unica de workspace con task/runtime/worktree
4. rematar coste/tokens/presupuesto global con fuentes normalizadas

## Inventario exacto de proximos pasos

1. Definir `GET /api/estadisticas` o equivalente como agregador global estable sobre proyecto + agente + ventana, sin rehacer la semantica ya viva de `cockpit`, `control` y `actividad`.
2. Extraer una timeline global unica que mezcle tarea, runtime, mailbox, handoff, Git y review sin tener que consultar varias rutas manualmente.
3. Subir `gitestadisticasapp` de servicio por `cwd` a puerto canonico de observabilidad Git reutilizable por proyecto, agente y workspace.
4. Normalizar coste/tokens/cuota/presupuesto por proveedor para que la capa global no dependa de heuristicas distintas segun runtime.
5. Reforzar `work_confirmed` contra salud operativa real para que la vista global no cuente trabajo muerto como continuidad viva.
6. Mantener la frontera doctrinal: `ADK contracts` solo como endurecimiento posterior de eventos/deltas/artifacts, no como sustitucion de esta API ni del control plane actual.
