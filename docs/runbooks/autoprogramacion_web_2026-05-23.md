# Autoprogramacion web - 2026-05-23

## Alcance

Este runbook valida la superficie web de autoprogramacion que recibe trabajo
humano o externo y lo transporta a contratos publicos de Orquesta. La web sigue
siendo adaptador inbound: no planifica, no decide runtime, no lee DB ni conoce
internals de apps propietarias.

Write-set del corte:

- `modulos/orquesta-web`
- `docs/runbooks/autoprogramacion_web_2026-05-23.md`

Fuente operativa del trabajo:

- `operational_director.task_source: director_decision`

## Contrato publico

Rutas web cubiertas:

- `GET|POST /nueva-app`
- `GET|POST /app-change`
- `GET /director-stats`
- `GET|POST /run-control`
- `GET /run-queue`

Contratos externos consumidos por la web:

- `SolicitarNuevaApp v0` para convertir el formulario de nueva app en
  `AppSpecRequestV0` y recibir `AppSpecV0` + backlog propuesto.
- `orquesta.apps.arrancar_director.v0` como destino opt-in del submit de nueva
  app cuando la composicion inyecta el puerto del director.
- `orquesta.apps.request_change.v0` para cambios sobre apps existentes.
- `orquesta.run_queue.priority.v0` para ranking y prioridad de runs.
- `orquesta.runs.control.v0` para `pause`, `resume`, `stop` y `cancel`.
- `director-stats` por bridge REST/MCP para progreso compacto.

## Fronteras

- La primera pantalla operativa es `/nueva-app`; no es landing ni marketing.
- Los textos visibles viven en catalogos i18n locales, con espanol por defecto
  y fallback controlado.
- `/app-change` puede transportar `external_work` con refs opacas:
  `external_project_ref`, `external_interface_refs`, `external_work_kind` y
  `external_work_refs`.
- La web no interpreta el dominio externo ni mueve validadores de producto al
  modulo web; los errores publicos pertenecen al contrato consumido.
- La web no materializa backlog, no arranca agentes, no abre runtime y no toca
  persistencia. Todas las acciones salen por clientes/puertos inyectados.
- Los paneles son sobrios y compactos: muestran estado, refs y errores
  publicos, no dumps internos, prompts, HOME, OAuth, proveedor, modelo, DSN ni
  detalles de almacenamiento.
- Los ficheros Go del modulo quedan por debajo de 300 lineas.

## Validacion del corte

Ejecutar desde la raiz del repo:

```bash
go test -count=1 ./modulos/orquesta-web
```

Criterios de aceptacion manual:

- La UI mantiene arquitectura hexagonal: handlers y render consumen puertos
  locales, no DB/runtime/stores concretos.
- La pantalla web es operativa desde el primer viewport y conserva i18n.
- El trabajo externo usa refs opacas y no acopla Orquesta a OPES ni a otra app
  propietaria.
- `run-control` y `run-queue` delegan mutaciones al contrato REST/MCP y no leen
  estado interno desde web.
- `/director-stats` expone acciones seguras de operador con POST explicito:
  para runs goal-first observa por `/api/v0/autoprogramming/goal/observe` o
  `/api/v0/apps/director/goal/observe`; `/api/v0/autoprogramming/supervise` se
  muestra solo como accion legacy cuando no hay `GoalWorkStateV0`. Pausa,
  reanudacion y parada siguen por `/run-control`.
- El contrato externo de dominio queda en adaptadores/composicion; la app
  propietaria conserva datos, reglas, validadores y ensamblado.

## Resultado 2026-05-23

Validado con la bateria focal obligatoria del paquete:

- `go test -count=1 ./modulos/orquesta-web`
- `validar criterios de aceptacion del cambio`
- `validar contrato externo de dominio`
