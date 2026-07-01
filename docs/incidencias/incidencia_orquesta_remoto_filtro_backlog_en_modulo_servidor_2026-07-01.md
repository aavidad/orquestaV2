# Incidencia: filtro de backlog ejecutable dentro del modulo servidor

Fecha: 2026-07-01

## Resumen

El agente remoto dejo un diff que anade
`idleSelfImprovementExecutableBacklogRequestRefV0` en
`modulos/orquesta-server/config_v0.go` para descartar refs de backlog narrativas
antes de lanzar automejora.

La intencion operativa es correcta: no lanzar secciones narrativas como si
fueran tareas ejecutables. Pero el lugar es dudoso: la composicion
`cmd/orquesta-server` ya tiene el puerto
`FilterIdleSelfImprovementRequestsV0` y la politica
`serverStackIdleSelfImprovementBacklogRequestAllowedV0`, con tests para filtrar
refs no ejecutables y permitir aliases federados (`T*`, `APG-*`,
`SRV-TASK-*`, `MCP-*`, `scanner-*`).

## Clasificacion

- Area: autoprogramacion / planner de backlog.
- Estado: no integrado como cambio de nucleo/modulo.
- Severidad: media, porque mover politica de composicion al modulo puede crear
  rails de nombres en una capa que debe aceptar puertos y decisiones externas.
- Hipotesis arquitectonica: falta dejar claro que el filtro de backlog
  ejecutable pertenece al adaptador/composicion que conoce el formato de las
  secciones, no al normalizador causal generico.

## Accion

No integrar el helper en `modulos/orquesta-server/config_v0.go` sin redisenar la
frontera. Si se necesita cerrar la regresion, reforzar tests en
`cmd/orquesta-server` y en el puerto de filtrado, no con un filtro duro adicional
en la normalizacion causal.

