# orquesta-web

Responsabilidad: interfaz primaria para pedir apps, revisar progreso y operar acciones seguras.

Incluye:

- consola inicial `/`;
- pantalla de autoprogramacion `/autoprogramming`;
- nueva app;
- panel de agentes;
- panel de fases;
- panel de control de runs;
- panel de cola multiapp;
- revision de propuestas;
- acciones seguras;
- progreso compacto.
- uso agregado saneado cuando `director/stats` lo publica con opt-in.

La web es adaptador inbound. No decide negocio ni accede a DB directamente.
Para T209, la web no lee logs ni runtime: solo proyecta `usage_summary` y
`agent.usage` ya saneados. Si hay senales vivas sin reporte, muestra
`unknown`/`unavailable` con reason code publico, no `-` ni detalles privados.

Para T210, la web consume `DirectorRunStatsV0`: si stats trae progreso vivo por
entrega, agente o proceso registrado, no debe degradarlo a 0% por fallback
visual. `tasks_closed` sigue siendo cierre aceptado por el nucleo. La politica
de cache/frescura de `/ops` pertenece a T211.

El formulario `/app-change` puede enviar refs opacas de `external_work` para
que Orquesta coordine trabajos de una app externa sin importar su nucleo ni
acoplarse a sus bases de datos, runtime, proveedor o contratos internos.

La consola inicial `/` es un indice operativo fino: no lee estado ni decide
negocio; enlaza a `/ops`, `/nueva-app`, `/autoprogramming`, `/app-change`,
`/director-stats`, `/run-queue` y `/run-control` para que el operador pueda
usar Orquesta desde la web sin recordar rutas API.

La guia de opciones de `/nueva-app` esta servida en `/nueva-app/guia` y vive en
`docs/guia_nueva_app_opciones_2026-06-25.md`. Documenta el asistente guiado, el
modo basico y el modo experto opcional para arquitectura, datos,
almacenamiento, integraciones multiples, accesibilidad, autonomia, calidad y
tooltips; es contrato documental de opciones y no introduce proveedor, DB ni
runtime en la web.

La pantalla `/autoprogramming` consume los endpoints publicos
`/api/v0/autoprogramming/prepare-run`, `/api/v0/autoprogramming/status`,
`/api/v0/autoprogramming/goal/observe` y
`/api/v0/autoprogramming/supervise` desde navegador same-origin. La web prepara
payloads compactos con refs opacas y no interpreta worktrees, ramas, runtime ni
stores. Por defecto anade los marcadores de migracion Goal a la tarea; si
`prepare-run` devuelve un `goal` lanzado o `goals[]` por lote, lo observa por
el `run_ref` seleccionado con polling acotado. En batch muestra un selector de
goals y no asume que el primer `run_ref` cubra todo el trabajo. Si solo devuelve
`goal_specs[]`, lo muestra como handoff goal-first preparado sin empujar al
operador al supervisor legacy.

`/director-stats`, `/run-queue` y `/run-control` conservan respuesta JSON para
clientes finos, pero cuando el navegador pide `text/html` devuelven una shell
HTML operable que usa esos mismos contratos publicos por `fetch`.
`/director-stats` acepta tambien `app_ref` y `external_job_ref` para resolver y
proyectar trabajos externos por refs opacas. La web solo muestra el bloque
`external_job` que llega del contrato publico: estado, razon, issues,
evidencias y diagnosticos; no lee OPES, runtime, logs ni filesystem.
