# Incidencia: bucle de rework por railes estrechos en T260/T261

Fecha: 2026-06-11

## Resumen

Durante el cierre de las tareas T260 y T261, Orquesta siguió lanzando agentes de
`Correccion de entrega tras revision` aunque las entregas ya declaraban el mismo
estado: cambio documental aplicado, prueba global `go test -count=1 ./...`
fallida por entorno/sandbox y trabajo técnico pendiente fuera del `write_set`.

No era trabajo útil nuevo. Era un bucle de rework causado por la combinacion de:

- `write_set` limitado a `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
  y `docs/runbooks`.
- `required_tests` con `go test -count=1 ./...`.
- contexto `ref_only` con `materialization_missing`.
- fallos repetidos de entorno: cache Go en solo lectura, sockets `httptest`
  denegados y wrappers Codex con `exit status 75`.
- revisor que interpreta cada prueba global fallida como nueva entrega
  insuficiente, aunque el propio ACK declare que no puede corregirse dentro del
  `write_set`.

## Evidencia observada

Agentes activos repetidos bajo:

- `request-ref-autoprogramming-backlog-t260-corregir-estados-falsos-running-en-agentes-externos-*`
- `request-ref-autoprogramming-backlog-t261-evitar-carrera-concurrente-al-instalar-skills-de-codex-*`

Paquetes activos con título:

```text
Correccion de entrega tras revision
```

Contrato repetido:

```text
write_set:
  - docs/autoprogramacion_orquesta_pendientes_2026-05-23.md
  - docs/runbooks
required_tests:
  - go test -count=1 ./...
```

ACKs repetidos de T261 indican:

```text
status: completed
files: docs/autoprogramacion_orquesta_pendientes_2026-05-23.md
required_test_failed: go test -count=1 ./... fallo por cache Go read-only,
socket loopback/httptest denegado y wrapper Codex exit 75.
```

El diff documental acumulado llego a miles de lineas en
`docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`, con multiples
parrafos de cierre equivalentes.

## Cambio requerido en Orquesta

Orquesta debe cortar el rework automatico cuando una entrega cumple todas estas
condiciones:

1. El ACK declara `status=completed`.
2. El unico fallo es una prueba requerida no ejecutable por entorno/sandbox o
   fuera del `write_set`.
3. El `write_set` no permite modificar el codigo que podria corregir la prueba.
4. Ya existe un ACK previo con la misma causa normalizada para la misma tarea,
   `parent_run_ref`, `supersedes_run_ref` o dedupe key.

En ese caso el estado correcto no es lanzar otro padre de rework documental. El
estado correcto es una tarea derivada explicita:

```text
blocked_external_or_scope_change_required
```

con una de estas acciones:

- abrir rework causal con `write_set` de codigo;
- ejecutar smoke opt-in fuera del sandbox estrecho;
- marcar deuda externa documentada y detener reintentos.

## Criterios de aceptacion

- No se relanza indefinidamente `Correccion de entrega tras revision` si el
  nuevo agente solo puede editar documentos y la causa del fallo ya esta
  deduplicada.
- El supervisor agrupa fallos equivalentes de `go test -count=1 ./...` por causa
  normalizada: `readonly_gocache`, `socket_denied`, `codex_wrapper_exit_75`.
- El ACK puede cerrar como `completed_with_external_validation_pending` o
  `blocked_external_or_scope_change_required` sin provocar rework documental.
- El director ve una sola deuda accionable, no decenas de parrafos repetidos en
  el backlog.
- Las pruebas focales de T260/T261 no dependen de una prueba global que el
  sandbox del agente no puede ejecutar.

## Arreglo programado

Implementado en `modulos/orquesta-app-codex-stack` el 2026-06-11:

- `ReviewReworkReplanSourceV0` consulta una guarda antes de crear otro plan de
  rework.
- La guarda corta solo el caso estrecho: ACK `completed`, `write_set`
  exclusivamente documental, `required_tests` con `go test -count=1 ./...` y
  evidencia de fallo externo como `GOCACHE` en solo lectura, socket/httptest
  denegado, sandbox o `exit 75`.
- Si el `write_set` toca codigo, se mantiene el flujo normal de correccion y
  puede crearse una tarea de rework causal.
- Prueba de regresion:
  `TestReviewReworkReplanSourceV0NoRelanzaBucleDocumentalPorGoTestGlobalNoEjecutable`.

Validacion local:

```bash
GOCACHE=/tmp/orquesta-go-cache-fix GOTMPDIR=/tmp/orquesta-go-tmp-fix \
  go test -count=1 ./modulos/orquesta-app-codex-stack
```

## Registro de cierre 2026-06-11

La correccion
`agent-ref-task-ref-review-rework-task-ref-review-rework-task-ref-revie-3564e080ccd41d45c75d4a0a35bff9b4`
se debe tratar como cierre documental de rework, no como evidencia de un fallo
nuevo. Mantiene la entrega valida acumulada, resuelve el contexto `ref_only` por
lectura local y deja la accion tecnica fuera de alcance hasta que exista
write-set de codigo, smoke opt-in o evidencia causal posterior al corte local.

Nuevos paquetes equivalentes con el mismo write-set documental, la misma prueba
global no ejecutable en sandbox y la misma causa normalizada deben deduplicarse
como `blocked_external_or_scope_change_required` o
`sincronizado/no-op documental`, sin anadir otra sincronizacion acumulativa al
backlog.
