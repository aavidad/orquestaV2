# Contratos: orquesta-estado-vivo

## Frontera

`orquesta-estado-vivo` es nucleo neutral. No observa procesos, no lee stores,
no consulta HTTP, no ejecuta comandos y no importa otros modulos de Orquesta.
Las composiciones y adaptadores convierten sus fuentes a `EvidenciaEstadoV0`.

## `EvidenciaEstadoV0`

Una evidencia representa una observacion de una sola fuente. La fuente conserva
su estado bruto en `Estado` y aporta flags estructurales cuando los conoce:
`ProcesoVivo`, `Terminal` y `Aceptado`.

La evidencia no decide fase. La frase operativa local es:

Las fuentes aportan evidencia; SOLO ConstruirProyeccionCicloVidaV0 decide fase

## `ConstruirProyeccionCicloVidaV0`

La funcion es pura y determinista:

- recibe todas las evidencias como datos;
- recibe `ahora` y `umbralHuerfano`;
- agrupa por `RunRef`; si falta, por `GoalRef`;
- ordena nodos y evidencias antes de devolver;
- devuelve `SchemaVersion` igual a `orquesta_proyeccion_ciclo_vida.v0`;
- no inventa verde cuando no hay evidencia: devuelve fase `desconocido`.

## Precedencia

1. Proceso vivo sin terminal: `proceso_vivo`.
2. Terminal aceptado sin proceso vivo: `terminal_aceptado`.
3. Terminal no aceptado sin proceso vivo: `terminal_rework`.
4. Proceso vivo y terminal simultaneos: `conflicto` con
   `proceso_vivo_tras_terminal`.
5. Estado persistido `blocked` o `bloqueado`: `bloqueado`.
6. Solo `run_marker` sin estado ni proceso: `huerfano` si supera
   `umbralHuerfano`, si no `lanzado`.
7. `receipt` no terminal con indicio de artefacto: `entregado_parcial`.
8. Sin evidencias: `desconocido`.
9. Un nodo por trabajo, agrupado por `RunRef` o `GoalRef`.

Como normalizacion neutral adicional, `outbox`/`queue` pendientes proyectan
`solicitado`, `wait_external` proyecta `lanzado`, y una fuente de proceso con
`ProcesoVivo=true` proyecta `proceso_vivo`.
