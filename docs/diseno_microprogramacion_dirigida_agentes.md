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
