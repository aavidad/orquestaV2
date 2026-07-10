# Clasificacion de retencion S13 - nota de la tarea T9103

Fecha: 2026-07-10. Ejecutada por el revisor (Claude) fuera del piloto D2,
segun el formato de la tarea T9103 de
`docs/backlog_piloto_autonomia_2026-07-10.md`.

Artefacto durable: `docs/clasificacion_retencion_s13_2026-07-10.json`
(validado con `jq empty`; 103 artefactos: 3 retener, 75 archivar,
25 candidato_borrar).

Criterio aplicado (conservador):

- `retener`: documentacion (incidencias) y lo ya archivado en
  `docs/historico/`.
- `archivar`: receipts `orquesta_goal_result_*` (evidencia de cierres, no
  fuente) y cualquier marcador citado por documentacion; se mueven a
  `docs/historico/` en una ola gobernada, nunca se borran directamente.
- `candidato_borrar`: marcadores `checkpoint_started*` sin ninguna
  referencia documental (grep por basename en *.md/*.go/*.sh versionados);
  riesgo bajo porque el estado durable vive en los stores, no en el arbol.

En esta tarea NO se borro, movio ni modifico ningun artefacto historico.
La poda posterior debe ser una ola gobernada con git como garantia y este
JSON como entrada (autorizacion de poda del operador 2026-07-05).
