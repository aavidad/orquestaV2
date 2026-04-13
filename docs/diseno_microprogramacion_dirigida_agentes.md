<!--
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
-->

# Diseño — Microprogramación dirigida para agentes

## Motivo

Este documento fija una decision operativa y arquitectonica nueva para Orquesta:

- los agentes dejan de recibir frentes amplios
- pasan a recibir microtareas cerradas por especificacion de funcion

La razon no es teorica. Es una conclusion nacida del propio proyecto:

- la delegacion abierta ha generado deriva respecto al roadmap
- esa deriva ha permitido que siguiera creciendo logica de negocio en `db/`
- tambien ha provocado reescrituras laterales y parches fuera del objetivo original

El problema no es solo de los agentes. Es del contrato de delegacion.
Si el contrato es abierto, el agente pasa a decidir arquitectura.
En Orquesta eso ya no es aceptable.

## Decision

La unidad de trabajo canónica para agentes pasa a ser la `especificacion de funcion`.

El orquestador debe definir:

- archivo exacto
- simbolo o firma a modificar
- descripcion funcional exacta
- precondiciones
- postcondiciones
- dependencias permitidas
- dependencias prohibidas
- tests obligatorios
- `write_set`

El agente solo puede:

- implementar esa unidad
- devolver el parche
- aportar evidencia de validacion

El agente no puede:

- rediseñar arquitectura
- abrir refactors laterales
- tocar archivos fuera del `write_set`
- introducir imports o dependencias no autorizadas

## Que copiamos de `oh-my-codex`

No copiamos su producto completo.
Copiamos el patron que sí resuelve el problema real de gobierno de workers.

### 1. `dispatch` state-first

En `oh-my-codex`, el trabajo no nace de una pulsacion arbitraria en `tmux`.
Nace de estado duradero:

- `src/team/team-ops.ts`
- `src/team/mcp-comm.ts`
- `src/scripts/notify-hook/team-dispatch.ts`

El sistema crea un `dispatch request`, lo deduplica, lo transiciona y solo despues intenta la notificacion.

Estados relevantes:

- `pending`
- `notified`
- `delivered`
- `failed`

Esto evita confundir "he enviado teclas" con "el worker ha recibido y consumido el trabajo".

### 2. Guard de readiness antes de inyectar

`oh-my-codex` no inyecta ciegamente en el pane.
Primero comprueba si el worker está listo:

- `src/scripts/notify-hook/team-tmux-guard.ts`
- `src/team/tmux-session.ts`

Comprueba:

- que el pane exista
- que no esté en shell vacía
- que no esté ocupado con otra tarea
- que el pane parezca listo

Esto es justo lo que Orquesta necesitaba para no tomar `tmux` vivo como sinonimo de trabajo consumido.

### 3. Worker protocol con `ACK` inicial

El worker de `oh-my-codex` no empieza a improvisar trabajo.
Primero hace `ACK`, luego lee `inbox`, reclama tarea y finalmente la completa:

- `skills/worker/SKILL.md`
- `src/scripts/notify-hook/team-worker.ts`

Secuencia:

1. `ACK` inicial al líder
2. lectura de `inbox`
3. claim-safe de tarea
4. ejecucion
5. marcado explicito de entrega y ciclo de vida

Eso reduce ambigüedad operativa y obliga a que el trabajo salga del estado, no de memoria conversacional vieja.

### 4. `tmux` como transporte, no como verdad

El pane es importante, pero no es la fuente de verdad del trabajo.
La verdad vive en:

- `inbox`
- `mailbox`
- `dispatch requests`
- estado del worker
- claim de tarea

Esa separación es la parte buena que Orquesta debe adoptar.

## Adaptacion a Orquesta

En Orquesta, esto se traduce en:

### A. Especificacion de funcion

Entidad canonica para trabajo delegado.

### B. Entrega validada

La entrega del agente solo vale si el orquestador puede comprobar:

- firma
- `write_set`
- imports
- tests obligatorios

### C. Worker protocol acotado

El worker:

- recibe microtarea por inbox/mailbox
- hace `ACK`
- ejecuta solo esa unidad
- devuelve evidencia

### D. `tmux` y runtime debajo de la especificacion

`tmux` sigue siendo util para continuidad y observabilidad.
Pero ya no decide la semantica del trabajo.

### E. `git/worktree` como entrega canónica de codigo

La coordinacion del worker y la entrega del codigo no son la misma cosa.

- `dispatch`, `mailbox` y `ACK` siguen siendo la capa correcta para gobierno operativo
- pero la entrega canónica del codigo debe pasar por `git/worktree`

Decision:

- la fuente principal del codigo entregado deja de ser el transcript del LLM
- el transcript puede seguir aportando evidencia, resumen o fallback
- pero el codigo que Orquesta revisa e integra debe salir del `worktree` aislado del agente

Esto reutiliza lo mejor de `oh-my-codex`:

- estado durable para coordinar workers
- `worktree` como unidad operativa real del codigo
- integracion posterior por `git diff` y `merge`, no por texto libre

En Orquesta eso implica:

- el agente trabaja en una `worktree` activa del proyecto
- la app recoge `git status` y `git diff` de esa `worktree`
- valida que solo toca el `write_set`
- registra una solicitud de merge por la via canónica de `gitaplicacion` / `gitgobernanza`
- la fusion automatica o manual sigue cayendo por la cola de merges ya existente

Regla:

- `// FILE:` o `PATCH_UNIFICADO` quedan como compatibilidad o rescate
- no se promocionan como via principal de entrega para agentes programadores

## Tareas abiertas en Orquesta

Este frente ya existe en Orquesta como un bloque específico de microprogramacion dirigida:

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
- `#521` Entrega canónica de microprogramacion por `git/worktree`
- `#522` Unificar subagentes programadores bajo entrega `git/worktree`
- `#523` Reutilizar OpenClaw/supervisor_subagents con integracion git y sin vias paralelas
- `#533` Adoptar `dispatch` durable canónico (`pending/notified/delivered/failed`) en la app
- `#534` Desacoplar `send_instruction` de `receipt` y cerrar por entrega válida
- `#535` Introducir `ready gate` canónico antes de inyectar a workers interactivos
- `#536` Unificar `receipt` asíncrono desde transcript, git y materialización validada
- `#538` Integrar el patrón operativo de `oh-my-codex` en Ollama, Codex, Claude y OpenClaw sin rutas paralelas
- `#539` Reforzar worktree/git como única entrega canónica para agentes programadores

## Perfiles canonicos de evaluacion de modelos locales

La evaluacion y promocion de modelos locales no se hace con nombres ad hoc de rol.
Se hace con los perfiles canonicos de tarea ya definidos por la app.

Para promocionar un modelo local como worker de microprogramacion dirigida, Orquesta debe probarlo al menos en:

- `implementacion`
- `revision`
- `analisis`

Regla de interpretacion:

- `documentador` es un rol posible de agente, no un `perfil_tarea` canónico
- si se quiere evaluar documentacion, esa prueba cae bajo `analisis` mientras no exista un perfil de tarea mas especifico aprobado por la app

Resultado provisional de evaluacion local (`2026-04-11`):

- `gemma4:26b`: mejor resultado global y unica opcion aprobada por ahora en la smoke canonica por la app
- `qwen3.5:9b`: mejor candidato ligero en prompt directo, pero no aprobado todavia como worker por defecto; en la smoke canonica por la app quedo por detras de Gemma y filtro razonamiento/protocolo
- `qwen3:14b`: utilizable, pero mas ruidoso y menos disciplinado
- `qwen2.5-coder:14b`: no apto como worker local por defecto
- `starcoder2:3b`: descartado para este uso

Regla de promocion:

- un modelo local no se considera aprobado como worker por defecto hasta pasar estas pruebas sobre microtareas cerradas, `write_set` acotado y salida validable por el orquestador
- la smoke canonica vale mas que el prompt directo: si un modelo parece bueno en prueba manual pero falla en el flujo real de Orquesta, no se promociona

Regla operativa para Ollama local:

- varios agentes logicos pueden compartir un mismo pool local
- eso no implica varios workers fisicos concurrentes
- Orquesta debe gobernar slots por pool/modelo para evitar contencion de GPU o RAM entre modelos residentes

## Benchmark externo aplicado

Orquesta no sigue ya un criterio de invención libre en runtime.

Decisión:

- copiar de `oh-my-codex` el contrato `dispatch durable + notified rápido + delivered/failure aparte`
- copiar de `mission-control` la prioridad de observabilidad y deuda operativa visible
- copiar de `automaker`, `Maestro` y `Aperant` la idea de pipeline por fases con workers especializados

Regla:

- se copia el patrón
- no se clona el producto completo
- el encaje se hace sobre la app hexagonal de Orquesta y no creando una segunda arquitectura paralela

## Cambio de estrategia local (`2026-04-12`)

La siguiente evolucion ya no es “hacer que Gemma haga todo”.

Decision:

- Orquesta sigue siendo el orquestador principal
- los LLM dejan de actuar como supervisor libre del sistema
- pasan a ser workers especializados por fase

### Pipeline objetivo

- `especificacion` y replanificacion: `Qwen3.5 27B`
- `implementacion`: `Qwen2.5 Coder 32B`
- `revision`: `DeepSeek Coder V2`
- `solver alternativo` o contraste: `Gemma4 26B MoE`

### Regla fuerte

- no se mantienen varios modelos pesados residentes a la vez salvo evidencia de capacidad real
- la regla por defecto es `un modelo grande activo cada vez`
- Orquesta decide:
  - modelo preferente por fase
  - fallback por fase
  - `pool`
  - `slot`
  - `keep_alive`
  - `timeout`
  - momento de descarga

### Regla adicional para `revision`

La revision puede usar varios modelos, pero no como enjambre libre.

Se hace en cadena:

- revisor base
- segunda opinion si el gate lo pide
- desempate o auditoria final solo si sigue habiendo duda

Orden recomendado:

- `DeepSeek Coder V2` como revisor local duro
- `Qwen3.5 27B` o `Gemma4 26B MoE` como segunda opinion local
- `gpt-5.4` o `Claude` como segunda opinion premium opcional

Cada revisor debe devolver:

- hallazgos estructurados
- severidad
- evidencia
- recomendacion `aceptar`, `corregir` o `escalar`

Reglas:

- siempre sobre la misma entrega Git
- siempre sobre el mismo `write_set`
- siempre sobre los mismos tests/gates
- nunca por mayoria simple ni por “debate” entre modelos
- la decision final la toma Orquesta

### Motivo

- con GPU local pesada, la autonomia buena no es una granja libre de workers pesados
- la autonomia buena es un pipeline determinista con workers especializados y gates fuertes
- eso reduce deriva, reduce contencion y deja el control real en Orquesta

### Implicacion arquitectonica

- `ollama_pool_local` no puede quedarse como un conector especial para Gemma
- debe evolucionar a base de scheduler local multi-modelo por fase
- el conector cambia; el nucleo de microprogramacion, validacion y entrega Git se mantiene

## Tareas abiertas del siguiente bloque

- `#524` Definir pipeline canónico por fases (`especificacion`, `implementacion`, `revision`, `correccion`, `integracion`)
- `#525` Introducir scheduler multi-modelo local por `perfil_tarea`, `pool`, `slot`, `keep_alive` y `timeout`
- `#526` Promover Orquesta como orquestador determinista y reducir el LLM de supervisor a excepcion
- `#527` Integrar `Qwen3.5 27B` como worker de especificacion y replanificacion
- `#528` Integrar `Qwen2.5 Coder 32B` como worker canonico de implementacion
- `#529` Integrar `DeepSeek Coder V2` como worker canonico de revision y deteccion de regresiones
- `#530` Mantener `Gemma4 26B MoE` como solver alternativo y fallback controlado
- `#531` Orquestar carga y descarga de modelos pesados de Ollama por fase sin residencia concurrente innecesaria
- `#532` Introducir revision escalonada multi-modelo con segunda opinion opcional (`premium` o alternativa local)

## Estrategia local inmediata

La estrategia local actual no es eliminar de golpe la via existente de Ollama.
Es convivir con dos caminos claramente diferenciados.

### 1. Via de compatibilidad y experimentacion

Se conserva la via actual:

- `agente` -> `ollama-cli` -> `tmux`

Motivo:

- ya existe y sirve para smokes, depuracion y comparativas de modelos
- sigue siendo util para probar candidatos como Qwen sin bloquear la evolucion del diseño principal

Regla:

- esta via no desaparece por ahora
- queda como compatibilidad, pruebas y rescate
- no se promociona como arquitectura local preferente para producción del orquestador

### 2. Via canónica objetivo para local

La via objetivo pasa a ser:

- `pool local`
- `slots`
- `agentes logicos`
- `perfiles_tarea` canonicos
- un unico modelo fisico cargado cuando sea posible

Primer caso priorizado:

- `pool`: `ollama-gemma4`
- `modelo`: `gemma4:26b`
- `slots_maximos`: `1` al inicio

Motivo:

- con los recursos actuales, Gemma ha sido el unico modelo local aprobado en la smoke canonica por la app
- la carga real del host no permite tratar varios modelos pesados como si cada uno fuera un worker barato
- el coste real esta en concurrencia, KV cache, GPU/RAM y latencia, no solo en el parametro activo del MoE

Estado transitorio actual:

- `gemma4:26b` sigue siendo el baseline operativo mientras se descargan y validan `Qwen3.5 27B`, `Qwen2.5 Coder 32B` y `DeepSeek`
- no se congela la arquitectura en Gemma; se usa Gemma como worker transitorio y fallback

### Compatibilidad entre ambas vias

La nueva via debe ser compatible con la actual.

Regla de diseño:

- `ollama-cli/tmux` se mantiene para modelos experimentales, pruebas comparativas y recuperación
- `ollama_pool_local` o equivalente se introduce como conector canónico nuevo
- la selección entre una u otra vía se decide por política de modelo y pool, no por scripts laterales

### Restriccion arquitectonica

La implementación de esta nueva via debe seguir Bloque 2:

- app-first
- puertos claros
- adaptadores de persistencia reducidos
- sin volver a meter lógica nueva de negocio en `db/`
- i18n y castellano en toda superficie nueva de la app

## Regla de transicion

Hasta que este frente cierre:

- no se deben delegar a agentes frentes amplios del núcleo
- las nuevas unidades deben bajar ya como microtareas acotadas
- lo nuevo debe seguir castellano e i18n desde origen
- y toda mutacion operativa debe entrar por servicios `*app`, no por acceso directo a `db`
