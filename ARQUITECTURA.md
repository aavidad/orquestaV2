<!--
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
-->

# Arquitectura objetivo de Orquesta

## Aviso vigente 2026-05-17

La foto operativa actual se ha actualizado despues del giro a nucleo
reutilizable para apps externas. Antes de usar este documento como fuente
canonica, leer:

- `docs/estado_actual_2026-05-17.md`
- `docs/guia_nucleo_orquestacion_2026-05-17.md`
- `AGENTS.md`

Este fichero conserva contexto historico y vision amplia. Cualquier referencia
al antiguo control-plane, `db/`, rutas legacy o programacion como unico dominio
queda subordinada a la frontera vigente del nucleo neutral.

Documento canonico consolidado:

- `docs/BIBLIA_APP_ORQUESTA.md`

Este fichero sigue siendo la referencia especifica de arquitectura, pero la doctrina estatica del proyecto se consolida ya en la biblia para evitar divergencias entre agentes y documentos dispersos.

Este fichero resume la direccion arquitectonica.
La especificacion ampliada de trabajo y la matriz de voto viven en:

- `docs/orquesta_v1_vision.md`
- `docs/op_049_matriz_voto.md`
- `docs/orquesta_v1_roadmap.md`
- `docs/diseno_microprogramacion_dirigida_agentes.md`
- `docs/operacion_agentes_manuales.md`
- `docs/op_096_control_total_estado_proyecto_y_estadisticas.md`

### Manuales Oficiales v1.0
- [Manual de Usuario](docs/manual_usuario.md): Guía funcional y operativa.
- [Manual de Programador](docs/manual_programador.md): Guía de integración y APIs (MCP, A2UI).
- [Manual de Técnico de Sistemas](docs/manual_tecnico_sistemas.md): Guía de despliegue y mantenimiento (Docker, DB).

## Objetivo

Orquesta pasa de ser un coordinador local de tareas a ser el nucleo de orquestacion de todo el workspace `~/Trabajo`.
El nucleo no debe depender de un LLM concreto. Debe depender de un contrato estable de `conector`.

Nota operativa actual:

- el repositorio operativo actual vive en `~/Trabajo/orquesta`
- las referencias antiguas a la ruta legacy previa del repositorio son historicas y no deben usarse como verdad operativa
- cualquier traslado fisico futuro debe tratarse como migracion explicita, no como suposicion documental

## Estado real de la arquitectura en esta sesion

Direccion ya validada por el codigo en curso:

- la autonomia ya no se modela como un simple `start/stop` de agentes; ya opera como politica durable de proyecto con supervisor residente, reviewer reservado y workers acotados
- `finish_app` ya no debe entenderse como una etiqueta decorativa de tarea; el estado real es `finish_app` persistente con bucle `until done or hard blocker`
- el control total por agente y por proyecto ya esta operativo desde Orquesta con cockpit server-first, actividad por agente y correlacion de estado/Git/runtime
- la foto operativa estable de autonomia ya no deja continuidad pendiente abierta: `autonomyPending=0`
- la politica de coste vigente es frugal por defecto: worker barato/local primero, `repair-helper` estrecho antes de relevo caro y `prime` solo por evidencia

Lo que todavia no puede darse por cerrado:

- el cierre pendiente ya no esta en el control total por agente/proyecto, sino en la agregacion global del workspace
- la timeline temporal unificada global y la proyeccion global por ventana siguen evolucionando
- las estadisticas Git ya cubren gobierno por agente/proyecto, pero la capa global agregada aun no esta cerrada como contrato canonico unico
- el coste/tokens global y la homogeneizacion entre proveedores siguen parciales

## Estado actual del daemon y de `status`

Contrato vivo ya validado por el codigo:

- el plano oficial es server-first: `server start` deja el daemon residente y `server run` conserva el modo foreground
- el daemon no esta realmente listo hasta que convergen `healthz`, `statefile` y `/api/status` decodificable
- el descubrimiento operativo ya admite recuperacion por `healthz` cuando el `statefile` falta o esta stale, pero siempre sobre el mismo `scope` y el mismo backend
- `status` y `/api/status` son la proyeccion canonica de capacidad visible; clientes y web no deben recomponer esa foto desde shell, sesiones o BD directa
- la telemetria compacta visible incluye `workersConectados`, `workersTrabajando` y `supervisoresActivos`
- esos contadores no equivalen a "agentes visibles en bruto": los `workers*` descuentan supervisores reservados y `supervisoresActivos` se publica aparte como capacidad de gobierno

Foto viva consolidada a `2026-04-23`:

- el loop autonomo estable se documenta como `ready`
- la convergencia actual correcta es `autonomyContinuing=0` y `autonomyPending=0`
- el resumen sano esperado publica `autonomyConfirmed>0` solo cuando hay frentes realmente absorbidos
- la telemetria visible debe leerse siempre junto a `workersConectados`, `workersTrabajando` y `supervisoresActivos`, no como simple recuento bruto de filas

Cuello real siguiente:

- el problema principal ya no es `liveness` base del daemon ni del supervisor residente
- el siguiente cuello esta en separar salud operativa real de `work_confirmed`
- un worker con `runtime/handle` degradados y heartbeat/progreso caducados no debe seguir contando como trabajo confirmado solo por evidencia vieja o actividad superficial
- la autocuracion correcta depende de que ese frente decaiga a estado recuperable para disparar `repair-helper`, reinicio coordinado o relevo barato

## Politica consolidada de autonomia operativa

La politica canonica ya no debe leerse como varias heuristicas sueltas. El contrato vigente es este:

- `supervisor residente`: cada proyecto autonomo mantiene un supervisor ligero, estable y reservado para gobernanza
- `control plane event-driven`: el daemon no debe reinyectar churn por pulso ciego; decide sobre eventos, evidencia y estado durable
- `repair-helper local primero`: ante atasco real, la primera respuesta es local, estrecha y barata antes de relevo o escalado
- `prime solo por evidencia`: los modelos caros no son el camino normal; se autorizan solo por bloqueo real, review reincidente, criticidad o riesgo probado

Consecuencias arquitectonicas:

- el supervisor residente no cuenta dentro del cupo ejecutor; `max_workers` mide solo workers reales
- un sistema sano puede tener `supervisoresActivos > 0` y `workersTrabajando = 0`; eso significa gobierno vivo sin ejecucion visible, no fallo
- la ausencia de trabajo nuevo no justifica churn: si el frente esta estable, el control plane debe tender a `standby event-driven`, no a reinstruccion continua
- los contadores visibles son semantica operativa, no decoracion de UI:
  - `workersConectados`: capacidad ejecutora visible
  - `workersTrabajando`: capacidad ejecutora realmente ocupada
  - `supervisoresActivos`: capacidad de gobierno separada del cupo ejecutor

Complemento de corto plazo ya aceptado:

- la inspiracion útil de `Google ADK` para Orquesta no es adoptar otro framework, sino reforzar el núcleo actual con:
  - `AutonomyEvent`
  - `state_delta` por decisión
  - `artifacts` versionados
  - `rewind/replay` parcial de tareas autónomas
- ese trabajo se documenta en [docs/propuesta_adk_eventos_delta_artifacts_rewind_2026-04-23.md](/home/alberto/Trabajo/orquesta/docs/propuesta_adk_eventos_delta_artifacts_rewind_2026-04-23.md)
- criterio estricto:
  - sí a endurecer trazabilidad, reversibilidad y rehidratación
  - no a replatformar Orquesta ni a meter ADK como framework paralelo

Complemento operativo también aceptado:

- la inspiración útil de `oh-my-codex` para Orquesta no es copiar otra CLI ni otro runtime completo, sino reforzar la disciplina de ejecución con:
  - hooks de ciclo de vida
  - equipos por rol
  - `worktree + tmux + resume`
  - perfiles de ejecución
- ese trabajo se documenta en [docs/propuesta_omx_hooks_roles_worktree_resume_2026-04-23.md](/home/alberto/Trabajo/orquesta/docs/propuesta_omx_hooks_roles_worktree_resume_2026-04-23.md)
- regla de convivencia:
  - `ADK` para estado durable, delta y reversibilidad
  - `OMX` para workflow de ejecución y handoff seguro
  - una sola verdad de control plane

Estado operativo verificable hoy:

- `POST /api/repos/mejorar` ya puede abrir una mejora con `finish_app`, `autonomia_persistente`, supervisor residente, reviewer reservado y `max_workers`
- `supervisionapp.BuildResidentSupervisorEventDrivenPolicyInput(...)` ya existe y fija la normalización base del perfil residente; no todos los entrypoints pasan todavía por ese helper, pero el contrato ya no vive solo en documentos
- `GET /api/proyectos/{slug}/cockpit` y `GET /api/proyectos/{slug}/control` ya dan control total utilizable a nivel proyecto
- `GET /api/agentes/{agente}/actividad` y `orquesta agente actividad <agente>` ya dan control temporal y Git a nivel agente
- `gitestadisticasapp` ya aporta `branch`, `touched_files`, `pending/committed added/deleted lines` y commits recientes por repo/cwd
- el cierre pendiente ya no es "hacer visible agente/proyecto", sino consolidar la proyeccion global de workspace, la timeline global y la capa canonica de coste/tokens

## Principios

1. La identidad del agente es estable.
   `codex1`, `codex2`, `claude1`, `gemini1` son identidades de trabajo. No dependen del proyecto.

2. El proyecto es una entidad explicita.
   Un agente se asigna a un proyecto. La sesion vive dentro de esa asignacion.

3. El conector encapsula el runtime.
   El nucleo sabe que existe un conector con transporte, comando y metadata.
   El nucleo no necesita saber si por debajo corre Codex CLI, Claude Code, Gemini CLI o un runtime futuro.

4. La sesion es reanudable.
   Cada sesion puede guardar `external_session_id`, `resume_payload_json`, `cwd`, `branch` y `resumen_continuidad`.

5. Las reglas, skills y workflows salen de la BD.
   El agente recibe su briefing desde Orquesta, no desde ficheros dispersos.

6. La base de datos es un adaptador, no el núcleo.
   La lógica de negocio no debe depender de SQLite ni de SQL directo.
   El núcleo habla con puertos; la persistencia entra por adaptadores.

7. La web y la app desktop son clientes finos.
   No deben hablar directamente con SQLite, Git, runtimes, proveedores LLM ni conectores externos.
   Solo hablan con Orquesta y Orquesta actúa como único plano de control.

8. La delegacion a agentes es cerrada y verificable.
   El orquestador define el contrato exacto de trabajo: archivo, firma, write_set, tests obligatorios y restricciones.
   Los agentes ejecutan microtareas de implementacion. No rediseñan arquitectura ni abren frentes amplios por su cuenta.

## Desarrollo hexagonal

- `core` o `application`
  Casos de uso, reglas y servicios.
- `ports`
  Interfaces de persistencia, runtime, VCS, memoria, investigación y notificación.
- `adapters/inbound`
  CLI, HTTP, MCP, web, app desktop.
- `adapters/outbound`
  SQLite, Git, conectores LLM, web search, filesystem.

Regla práctica:

- lo nuevo debe entrar por servicio + puertos
- `db/` no debe volver a actuar como núcleo de aplicación
- la migración del legado se hará por fases
- la unidad canónica de delegación a agentes es la especificación de función, no el frente amplio
- una base nueva debe nacer neutra en agentes: la persistencia puede sembrar catálogo y conectores base, pero no debe registrar flotas legacy por defecto
- el alta de agentes y la composición de flota se hacen de forma explícita desde la app/API; no desde seeds ocultas del esquema
- `server_autobootstrap` es opt-in explícito y no puede activarse por defecto en una instalación nueva
- `finish_app` no es solo una prioridad de tarea: cuando se pide en modo persistente, activa una `policy` durable del proyecto con `Codex supervisor`, `Codex reviewer` y cuota de workers explícita
- en esa `policy`, `supervisor` y `reviewer` son reservas de gobernanza; `max_workers` cuenta solo workers reales/ejecutores
- ese modo persistente debe operar como bucle `until done or hard blocker`, sin supervisión humana continua pero con guardas duras de progreso neto, CPU y cruce de proyecto
- la orquestación autónoma no debe quemar `prime` por defecto: la política correcta es `supervisor ligero siempre online + worker rápido por defecto + escalado puntual a prime por evidencia`
- el escalado a `prime` solo se autoriza cuando un worker rápido no progresa, falla review repetidamente, entra en restart loop o la tarea es crítica en arquitectura/seguridad
- los workers deben ejecutarse con prompt compacto y disciplina de salida mínima: nada de prosa ornamental ni eco de consola inútil; solo código, diff, tests y evidencia corta cuando haga falta
- ante atasco, Orquesta debe intentar primero un `repair-helper` barato y de alcance estrecho antes de subir a `prime` o de cargar trabajo pesado en supervisor
- la escalera de recuperación correcta es: `worker actual -> repair-helper local rápido -> relevo worker -> prime`, salvo riesgo alto o fallo reincidente probado
- en pool local compartido, regla canónica: un solo modelo Ollama cargado a la vez; si hacen falta varios agentes, primero se multiplican sobre ese modelo antes de cambiar de modelo y forzar thrash de VRAM
- el proyecto debe tener `control total` visible desde Orquesta:
  - fecha de inicio real
  - porcentaje calculado de fin
  - flota por proyecto
  - timeline por ventana
  - estado Git, worktrees y merges
  - ficheros tocados y lineas modificadas por agente/proyecto
- la API Git canónica es parte del producto, no una utilidad lateral:
  - Orquesta debe exponer ramas, worktrees, diffs, commits, merges y hotspots por ventana
  - las lineas añadidas/borradas y los ficheros tocados deben poder agregarse por agente, proyecto y global
- la observabilidad temporal es requisito de autonomía:
  - preguntas como `que ha programado Codex1 en la ultima hora` o `que ha cambiado este proyecto hoy` deben resolverse por API/CLI/web/MCP, no por shell manual
- **Módulos e Idioma:** Todos los adaptadores y aplicaciones de dominio deben utilizar nomenclatura exclusiva en **castellano**, habitualmente sufijados con `app` (P.ej. usar `tareasapp`, `sesionesapp` en vez de `taskapp` o `sessionapp`). El objetivo es erradicar completamente el inglés de los nombres de carpeta para evitar confusiones de los modelos LLM (agentes).
- **i18n por defecto:** toda superficie nueva visible, clave pública o documento nuevo debe nacer preparada para i18n; no se admite crear funcionalidad nueva que obligue a rehacer la internacionalización después.

## Wizard de nueva app

El wizard `/nueva-app` no es un formulario decorativo. Es una fase guiada de descubrimiento que debe traducir decisiones de producto y arquitectura a backlog base.

Reglas:

- el wizard pregunta por fases, no como bloque amorfo
- toda decisión capturada debe viajar al `AppSpec` y reflejarse en las tareas generadas
- la arquitectura no se deja al criterio libre del LLM; Orquesta fuerza baseline profesional y usa las respuestas del wizard para afinarla

Fases mínimas del wizard:

- `briefing/brainstorming`: nombre, descripción, alcance, proyecto destino, integraciones y restricciones
- `arquitectura del núcleo`: estilo de arquitectura, núcleo residente, jobs, colas, scheduler, multi-tenant, feature flags, offline/sync
- `interfaces y acceso`: frontend, API, estilo de API, auth, RBAC, proveedor de identidad, notificaciones, webhooks, ficheros
  - si hay frontend web, el wizard debe preguntar también por `frontend_stack` y por capacidad de temas/branding desacoplado
  - theming no es un detalle visual suelto: debe quedar soportado por tokens/temas y no por estilos hardcodeados
  - si hay UI, la app debe nacer con una superficie visual explícita: normalmente `web/`; en modo `server-rendered`, al menos `adaptadores/web/`
- `datos y persistencia`: base de datos, motor, caché, búsqueda, object storage, import/export, auditoría
- `idiomas y documentación`: idiomas iniciales y, en proyectos ya arrancados, idiomas adicionales a incorporar de forma incremental
- `plataformas y compliance`: plataformas objetivo, SO y normativas aplicables
- `delivery y operación`: target de despliegue, artefacto principal (ejecutable, Docker, librería, instalador, firmware, sitio estático), observabilidad, backups y disaster recovery
- `calidad`: disciplina de testing y baseline profesional obligatoria

Baseline fija, no opcional:

- arquitectura con límites claros
- modularidad
- testing base
- tooling de calidad
- configuración reproducible por entorno
- política de errores y logging estructurado
- seguridad mínima y revisión de dependencias
- theming y branding desacoplados para cualquier app con UI web, desktop o móvil

Documentación fija, no opcional:

- guía para usuarios
- guía técnica para desarrollo
- guía de sistemas y operación
- si el proyecto activa i18n, esos manuales deben existir al menos en los idiomas elegidos en el wizard
- si más adelante se añaden idiomas nuevos, el backlog incremental debe incluir también i18n, documentación y QA para esos idiomas

## Modelo actual

### Proyectos

- Tabla `proyectos`
- Tipos: `raiz`, `grupo`, `repo`
- Soporta arbol de proyectos con `parent_id`
- Registra `ruta_abs` y `slug`
- La ingestión de repositorios tiene dos entradas canónicas: origen local (`--path`) y origen remoto (`--git`)
- Ambos orígenes deben materializarse primero como copia local canónica gobernada por Orquesta antes de asignar agentes o abrir pipeline
- La URL remota es metadata de origen; la revisión, implementación, handoff y merge ocurren siempre sobre la copia local materializada o una `worktree` derivada de ella
- Una vez materializado, el flujo operativo es único: `review`, `improve`, `handoff`, `merge` y `worktree` no distinguen entre repo local y repo remoto

### Asignaciones

- Tabla `asignaciones`
- Une `agente -> proyecto`
- Solo una asignacion activa por agente
- Sirve para repartir capacidad: por ejemplo `AutofirmaV2 = 2 agentes`, `ProyectoX = 4 agentes`

### Sesiones

- Tabla `sesiones`
- Guarda:
  - `proyecto_id`
  - `conector_id`
  - `cwd`
  - `herramienta`
  - `external_session_id`
  - `resume_payload_json`
  - `resumen_continuidad`
  - `branch`
  - `heartbeat_at`
  - `host`
  - `pid`

### Conectores

- Tabla `conectores`
- Contrato mínimo:
  - `slug`
  - `nombre`
  - `transporte`
  - `comando`
  - `args_json`
  - `env_json`
  - `metadata_json`

Ejemplos iniciales:

- `codex-cli`
- `claude-code`
- `gemini-cli`

## Comandos ya soportados

- `orquesta proyecto descubrir <ruta>`
- `orquesta proyecto listar`
- `orquesta repo add --path <ruta>`
- `orquesta repo add --git <url> [--branch <rama>] [--destino <ruta>]`
- `orquesta repo revisar --proyecto <slug|id> [--plan]`
- `orquesta repo mejorar --proyecto <slug|id> --titulo <...> [--descripcion <...>]`
- `orquesta asignacion activar --agente <agente> --proyecto <slug|id>`
- `orquesta conector listar`
- `orquesta conector ver <slug|id>`
- `orquesta conector registrar ...`
- `orquesta sesion inicio <agente> --proyecto <slug> --conector <slug>`
- `orquesta sesion guardar <agente> --proyecto <slug> ...`
- `orquesta sesion continuar <agente> --proyecto <slug>`

## Superficie API canónica para repos

La web no debe reconstruir estos flujos en cliente. El daemon expone ya el carril server-first para ingestión y trabajo sobre repos:

- `POST /api/repos/materializar`
  - entrada: `path` local o `git` remoto, opcionalmente `branch` y `destino`
  - salida: proyecto repo materializado y metadata de origen
- `POST /api/repos/revisar`
  - entrada: `proyecto` ya existente o `path/git` para materializar al vuelo
  - `plan=true` devuelve el siguiente paso determinista del pipeline
  - sin `plan`, ejecuta y despacha el siguiente paso del pipeline local
- `POST /api/repos/mejorar`
  - entrada: `proyecto` ya existente o `path/git` para materializar al vuelo
  - crea la tarea de mejora y opcionalmente despacha el siguiente paso del pipeline

Regla:

- CLI y web deben converger en esta misma superficie HTTP; no se admiten dos secuencias de negocio distintas para repos.

## Patrones tomados de los repos revisados

### `Z-M-Huang/claude-codex-gemini`

- Gestion explicita de sesiones por runtime
- Persistencia de contexto reanudable
- Registro estructurado del estado de cada agente

Aplicacion en Orquesta:

- `conectores`
- `external_session_id`
- `resume_payload_json`
- `resumen_continuidad`

### `nxtg-ai/forge-orchestrator`

- Nucleo desacoplado del runtime
- Capa de adaptadores/proveedores
- Coordinacion centrada en tareas y ejecucion

Aplicacion en Orquesta:

- el nucleo depende del contrato `conector`
- el runtime se mueve a datos configurables y no a condicionales por modelo

### `smtg-ai/claude-squad`

- Identidad estable de agentes
- Trabajo paralelo con contexto separado
- Afinidad entre agente y contexto de trabajo

Aplicacion en Orquesta:

- identidad estable de agentes
- asignaciones activas por proyecto
- sesiones reanudables por proyecto

### `dnnyngyen/gemini-cli-orchestrator`

- Operativa CLI clara para lanzar y seguir agentes
- Flujo sencillo de sesion y continuidad

Aplicacion en Orquesta:

- comandos `sesion inicio`, `sesion guardar`, `sesion continuar`
- modelo de conector CLI como primer transporte

### `oh-my-codex`

- estado canónico de equipo y workers fuera del pane
- `dispatch request` durable con transiciones `pending -> notified -> delivered/failed`
- `inbox` y `mailbox` como verdad de trabajo, no `send-keys` ad hoc
- guard de readiness antes de inyectar en `tmux`
- worker protocol con `ACK` inicial, claim-safe de tarea y marcado explícito de entrega

Aplicacion en Orquesta:

- `tmux` se usa como transporte persistente, no como fuente de verdad
- la entrega a agentes debe pasar por estado duradero y confirmación de consumo
- la unidad de trabajo delegada debe ser microprogramación dirigida por especificación de función
- el agente debe recibir una instrucción acotada y verificable, no un frente arquitectónico abierto

Referencia documental asociada:

- `docs/diseno_microprogramacion_dirigida_agentes.md`

## Siguiente fase recomendada

1. Crear un paquete de dominio para conectores.
   Por ejemplo `internal/runtime` o `runtime/` con interfaz:

   ```go
   type Driver interface {
       Lanzar(ctx context.Context, req Lanzamiento) (SesionExterna, error)
       Reanudar(ctx context.Context, req Reanudacion) (SesionExterna, error)
       Heartbeat(ctx context.Context, sesion SesionExterna) error
       Detener(ctx context.Context, sesion SesionExterna) error
   }
   ```

2. Resolver el driver por `conector.transporte`.
   El nucleo decide por `cli`, `mcp_stdio`, `mcp_http`, `api`, sin codificar proveedores concretos.

3. Hacer de `orquesta serve` el escritor principal.
   La CLI y la web deberían hablar con el servidor local para centralizar la trazabilidad y la gestión del acceso concurrente, aprovechando que el almacenamiento principal ya está sobre PostgreSQL.

4. Añadir politicas de reparto de agentes.
   Ejemplo: `slots_por_proyecto`, prioridad, exclusiones y colas.

5. Inyectar reglas, skills y workflows por sesion/proyecto.
   No solo por rol del agente, tambien por proyecto y por tipo de conector.

## Fase de refactor competitivo

La refactorizacion competitiva solo debe existir si mantiene la filosofia del proyecto y mejora o preserva la seguridad.

La unidad recomendada para esta fase no es el repositorio entero ni el modulo amplio.
La unidad recomendada es el `fork de funcion`:

- una misma especificacion de funcion o simbolo
- varias variantes aisladas
- varios modelos si hace falta
- mismo contexto de app
- misma arquitectura objetivo
- mismo contrato de validacion

El objetivo no es abrir debate libre entre agentes.
El objetivo es explorar variantes pequeñas y comparables sin perder la forma del proyecto.

### Reglas duras

1. Backup previo obligatorio.
   Antes de modificar nada, Orquesta debe generar una copia de seguridad o snapshot verificable del estado de trabajo.

2. Contrato congelado.
   Antes del experimento se fijan tests, benchmark, firmas publicas y restricciones funcionales.

3. Benchmark bootstrap, no veredicto fijo.
   La primera puntuacion de un agente local solo establece un punto de partida neutral o calibrado.
   El orquestador debe poder subir o bajar esa puntuacion con trabajo real posterior.

4. Evaluacion por materias.
   Los agentes locales no se comparan con una unica nota agregada.
   La arquitectura debe guardar scores separados para materias como codigo, documentacion, orquestacion, brainstorming, analisis, revision, frontend, testing, arquitectura e infraestructura.

5. Seleccion ponderada por tarea.
   El scheduler no elige por “mejor media”.
   Debe calcular fitness ponderado segun la naturaleza de la tarea.
   Ejemplo: para infraestructura, `infraestructura` pesa mas que `brainstorming`; para ideacion ocurre lo contrario.

6. Aislamiento por candidato.
   Cada candidato corre en su propia rama o worktree. Nunca sobre la misma copia de trabajo.

7. Seguridad no regresiva.
   Ningun candidato puede ganar si reduce controles de seguridad, endurecimiento, auditoria o validaciones.

8. Filosofia del proyecto no regresiva.
   Ningun candidato puede ganar si rompe convenciones arquitectonicas, reglas de dominio o estilo de mantenimiento acordado.

9. Revision minima por pares no autores.
   El ganador solo puede seleccionarse si al menos dos agentes revisores independientes lo aprueban.
   Ningun revisor puede ser autor del candidato revisado.

10. Correctitud antes que rendimiento.
   El orden de evaluacion es: correccion, seguridad, filosofia del proyecto, rendimiento, memoria, mantenibilidad.

11. Forks solo sobre contexto canonico.
    Todo `fork de funcion` debe heredar:
    - arquitectura declarada del proyecto
    - convenciones de capas
    - limites del modulo
    - dependencias permitidas
    - tests obligatorios
    - restricciones de seguridad y observabilidad

12. Hexagonalidad no regresiva.
    Si el proyecto es hexagonal, ningun fork puede “ganar” si mueve logica de dominio a adaptadores, rompe puertos, acopla la app a infraestructura o introduce atajos fuera de capa.

13. Contexto compartido pequeno y durable.
    La comparacion entre forks debe apoyarse en contexto compartido selectivo, no en historiales gigantes entre agentes.
    El orquestador debe poder inyectar antes del prompt:
    - decisiones
    - restricciones
    - hallazgos previos
    - follow-ups de sidecars
    - filosofia de proyecto

14. Variacion multi-modelo, no barra libre.
    Usar varios modelos esta permitido y es deseable cuando aumenta cobertura de ideas, pero siempre:
    - sobre la misma especificacion de funcion
    - en worktrees aisladas
    - con `write_set` cerrado
    - y con comparacion determinista posterior

### Modelo recomendado

- `experimento_refactor`
- `candidato_refactor`
- `revision_candidato`
- `metrica_candidato`
- `backup_experimento`
- `fork_funcion`
- `shared_context_item`

### Politica inicial en configuracion

- `refactor_backup_required=true`
- `refactor_min_reviewers=2`
- `refactor_reviewer_must_be_non_author=true`
- `refactor_preserve_security=true`
- `refactor_preserve_project_philosophy=true`

## Restricciones

- No perder datos actuales.
- Todas las migraciones deben ser aditivas e idempotentes.
- La persistencia principal corre sobre PostgreSQL, resolviendo los antiguos bloqueos por concurrencia que existían con SQLite. El acceso debe seguir centralizándose por arquitectura, no por limitación del motor.
