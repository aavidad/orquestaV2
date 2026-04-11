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

## Regla de transicion

Hasta que este frente cierre:

- no se deben delegar a agentes frentes amplios del núcleo
- las nuevas unidades deben bajar ya como microtareas acotadas
- lo nuevo debe seguir castellano e i18n desde origen
- y toda mutacion operativa debe entrar por servicios `*app`, no por acceso directo a `db`
