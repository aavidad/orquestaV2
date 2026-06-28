# Tarea Orquesta: servidor `startup_ready` sin backend Goal para external-work OPES

Fecha: 2026-06-27.

## Contexto

Trabajo OPES: Integrador Social Grupo B, temas 019-024.

Servidor usado por OPES:

- `127.0.0.1:19025`
- estado publicado: `startup_ready`
- mensaje: `director: orquesta preparada; autodiagnostico sin runs transitorios ni cola sucia`

## Fallo observado

Al enviar `POST /api/v0/external-work/run` con peticiones OPES validas, Orquesta
devolvio `400`:

```json
{
  "estado": "error",
  "route_policy": "goal_first",
  "director_execution_mode": "goal_first",
  "next_actions": [
    "configure_codex_goal_backend",
    "do_not_fallback_to_legacy_director_loop"
  ],
  "errores_publicos": [
    {
      "code": "external_work_goal_backend_required",
      "field": "codex_goal_backend"
    }
  ]
}
```

Peticiones afectadas:

- `tema_019_post_response.json`
- `tema_020_post_response.json`
- `tema_021_post_response.json`

Ruta OPES:

`/home/alberto/Trabajo/OPES/opes-salidas/diputacion_granada/cursos_opes_2026/administracion_especial/auxiliar-tecnico-superior-de-integracion-social/00_control/director_20260627/orquesta_responses_19025/`

## Diagnostico

El servidor puede arrancar como `startup_ready` aunque no tenga
`ORQUESTA_CODEX_GOAL_BACKEND` configurado. En modo vigente `goal_first`,
`external-work/run` no puede crear trabajos sin backend Goal y no debe caer al
loop legacy.

El estado visible tambien mostraba:

- `ORQUESTA_SERVER_RESIDENT_DIRECTOR_ENABLED=false`
- ausencia de `ORQUESTA_CODEX_GOAL_BACKEND` en el entorno del proceso

## Mejora solicitada

La comprobacion de arranque/autodiagnostico debe distinguir:

- servidor HTTP vivo;
- cola limpia;
- backend Goal configurado y operativo;
- external-work OPES ejecutable en modo `goal_first`.

Si el backend Goal falta y `external-work/run` es una ruta esperada, el estado
no deberia presentarse como listo sin advertencia operativa. Como minimo debe
mostrar un diagnostico visible con accion concreta:

`export ORQUESTA_CODEX_GOAL_BACKEND=app_server_stdio` y reiniciar servidor.

## Workaround aplicado por OPES

No se toca codigo de Orquesta porque hay otro agente trabajando en el nucleo.
Se reinicia solo el servidor OPES `19025` con:

```bash
ORQUESTA_CODEX_GOAL_BACKEND=app_server_stdio
ORQUESTA_CODEX_COMMAND=/home/alberto/.nvm/versions/node/v20.19.2/bin/codex
```

Despues se reintentaran las peticiones de temas 019-024.

## Segundo fallo observado tras configurar backend

Reinicio limpio con state/runtime nuevo:

- `ORQUESTA_CODEX_GOAL_BACKEND=app_server_stdio`
- `ORQUESTA_SERVER_RESIDENT_DIRECTOR_ENABLED=true`
- `ORQUESTA_SERVER_STATE_DIR=.../19025_goal_state`
- `ORQUESTA_CODEX_RUNTIME_WORKDIR=.../19025_goal_runtime`

Estado efectivo correcto:

- `status=running`
- `startup_status=startup_ready`
- `resident_director_status=running/ok`
- `state_persist_status=ok`
- `ORQUESTA_CODEX_GOAL_BACKEND=app_server_stdio`

Resultado al relanzar temas 019-024:

- 019: `POST /api/v0/external-work/run` devuelve `200`, con `goal_ref` y
  `external_goal_ref`.
- 020: `200`, con `goal_ref` y `external_goal_ref`.
- 021: `200`, con `goal_ref` y `external_goal_ref`.
- 022: `200`, con `goal_ref` y `external_goal_ref`.
- 023: `400`, `external_work_goal_launch_failed`, `field=goal_launcher`.
- 024: `400`, `external_work_goal_launch_failed`, `field=goal_launcher`.

Problema: para 023 y 024 quedan runs persistidas en
`orchestration-state/runs/`, pero no hay `GoalWorkStateV0` correspondiente en
`app_director_goal_states/`. Eso deja una run parcial que no es observable como
goal-first y obliga al consumidor OPES a cambiar de `run_ref` en el reintento.

## Tercer fallo observado: observe_goal opaco

Para 019-022 se intento:

`POST /api/v0/autoprogramming/goal/observe`

con los `run_ref` devueltos por `external-work/run`.

Resultado para los cuatro:

```json
{
  "estado": "error",
  "errores_publicos": [
    {
      "code": "autoprogramming_observe_goal_http_error",
      "field": "executor",
      "message": "autoprogramming_observe_goal_error"
    }
  ]
}
```

El audit HTTP solo muestra `500` y no aporta causa operativa suficiente para
saber si el fallo viene de app-server, del thread Codex, del `GoalWorkStateV0`,
del cierre, del parseo de resultado o de permisos.

## Mejora solicitada adicional

1. Si `GoalLauncher` falla tras persistir la run, Orquesta debe dejar estado
   reparable claro o rollback del contenedor, no una run goal-first sin
   `GoalWorkStateV0`.
2. `external_work_goal_launch_failed` debe incluir diagnostico publico
   suficiente: backend, timeout, app-server, cuota, thread, validacion, etc.
3. `observe_goal` no debe devolver solo `autoprogramming_observe_goal_error`;
   debe exponer un reason code publico accionable.
4. El residente deberia observar goals activos o al menos publicarlos como
   `running_goal_first_observe_pending`; ahora queda `idle` aunque existen
   goals lanzados.

## Cuarto hallazgo: perdida de valores de input en GoalWorkSpecV0

Al inspeccionar un `GoalWorkStateV0` de tema 019, el `spec.context_refs` conserva
entradas como:

```json
{
  "kind": "input_field",
  "ref": "input-field-required_read_refs",
  "purpose": "Nombre de campo disponible; el payload queda fuera del spec Goal."
}
```

El spec no incluye los valores completos de campos criticos del payload OPES:

- `course_root_abs`
- `topic_dir_abs`
- `topic_manifest_abs`
- `program_json_abs`
- `required_read_refs`
- `required_outputs`
- `output_contract`

Para OPES esto rompe o degrada la autonomia: el agente Goal recibe una tarea de
dominio, pero pierde las rutas y contratos concretos que necesita para trabajar
con precision y sin rehacer. La construccion de `GoalWorkSpecV0` debe incluir
valores seguros de campos operativos no sensibles, o al menos refs resolubles
durables a un payload completo local.

Accion esperada: ajustar external-work goal-first para que el goal tenga acceso
al contrato OPES completo de forma compacta y segura, no solo a nombres de
campos.

## Evidencia adicional de subagente OPES

Subagente de apoyo `Newton` reviso en solo lectura el estado local de `19025` y
confirmo:

- `019-022` tienen `external_goal_ref`:
  - 019: `019f095a-e7f6-78c2-9888-9f4407ea9d0a`
  - 020: `019f095b-4c4a-7060-8fba-825a9e0a3440`
  - 021: `019f095b-e47d-7571-9ad5-d6e4213c4fdd`
  - 022: `019f095c-71c5-71b1-b74e-13f738c49466`
- `app_director_goal_states/*.json` deja los cuatro goals en `status=running`,
  sin `observations` ni `result_refs`.
- `19025_goal_runtime` no contiene ficheros recuperables.
- El log del servidor solo aporta `shutdown_timeout`, sin causa de
  `observe_goal`.
- Hay escritura real parcial en OPES:
  - tema 019: `matriz_reutilizacion_tema.md`, `plan_tests_tutor_tema.md` e
    informes de extension/checkpoint;
  - tema 020: `04_markdown/tema_020_ampliado.md` e informes/checkpoint.
- temas 021-022: no hay artefactos nuevos atribuibles a la ola cuando se reviso.
- temas 023-024: no tienen thread recuperable por fallo de `goal_launcher`.

Actualizacion posterior: no se detuvo `19025` porque, minutos despues de la
revision inicial, aparecieron artefactos utiles de los goals 021 y 022:

- tema 022: `04_markdown/tema_022_resumen.md` y
  `04_markdown/tema_022_ampliado.md`;
- tema 021: informes de extension, calidad, partes a rehacer, matriz,
  checkpoint, validacion de texto publico y checkpoint de consolidacion.

Conclusion operativa: aunque `observe_goal` falla, los goals pueden seguir
trabajando y escribiendo en disco. No se deben matar automaticamente ante el
primer `observe_goal=500`; primero hay que comprobar actividad real en el
write-set del curso.

## Incidencia adicional 2026-06-27: rutas canonicas en reintentos OPES

En los temas 023 y 024 de Integrador Social se detecto que el fallo de
`external_work_goal_launch_failed` no implicaba ausencia de material util:

- tema 023 ya tenia `04_markdown/tema_023_ampliado.md` y
  `04_markdown/tema_023_resumen.md` con extension suficiente para Grupo B;
- tema 024 tenia material valido en `02_markdown/tema_024_ampliado.md` y
  `02_markdown/tema_024_resumen.md`, pero el contrato del reintento esperaba
  salidas en `04_markdown/`.

El Director OPES preparo requests nuevos `19025b` con IDs nuevos y objetivo
explicito de consolidacion, no creacion desde cero. La necesidad tecnica para
Orquesta es doble:

1. si un tema tiene material valido en una ruta no canonica, Orquesta debe
   normalizarlo como tarea de consolidacion (`02_markdown -> 04_markdown`) y
   conservar el contenido util;
2. el diagnostico de external-work debe distinguir fallo de lanzamiento, fallo
   de contrato de rutas y falta real de material, para que no se reescriban
   temas aprovechables.

Accion esperada: anadir validacion causal de rutas canonicas en el contrato
OPES y sugerir `consolidate_checkpoint_topic_from_existing_material` cuando
exista texto suficiente en rutas recuperables.

## Evidencia 19025c: request valido, fallo en goal launcher

Tras corregir el body de los reintentos `19025b` a `19025c` usando el schema
correcto de `DomainWorkFieldV0` (`value`, `values` y `value_json`), el endpoint
dejo de devolver `request_body_invalido`, pero siguio fallando antes de crear
goal:

- `tema_023_post_response_goal_backend_19025c.json`:
  `external_work_goal_launch_failed`, `field=goal_launcher`.
- `tema_024_post_response_goal_backend_19025c.json`:
  `external_work_goal_launch_failed`, `field=goal_launcher`.

Esto descarta que el problema principal sea el contrato OPES del reintento. El
lanzador goal-first debe exponer causa accionable: timeout, backend no vivo,
spawn rechazado, cuota, app-server saturado, conflicto de run/ref, permisos,
working dir o error interno. Sin esa causa, el Director OPES solo puede aplicar
workaround legacy documentado y seguir la produccion sin tocar el nucleo.

## Evidencia 19026 legacy: aceptado pero no observable/materializado

Se arranco un servidor aislado `127.0.0.1:19026` con:

- `ORQUESTA_EXTERNAL_WORK_LEGACY_DIRECTOR_LOOP=1`;
- `ORQUESTA_AUTOPROGRAMMING_LEGACY_DIRECTOR_LOOP=1`;
- `director_execution_mode=legacy_director_loop`.

Los reintentos de temas 023 y 024 fueron aceptados por
`/api/v0/external-work/run` con `HTTP 200` y
`route_policy=legacy_director_loop`. Sin embargo:

- el director residente siguio registrando `goal_first_state_missing` para el
  run 023, aunque el alta habia sido legacy;
- la primera llamada a `/api/v0/autoprogramming/supervise` sin
  `director_execution_mode` fallo correctamente con
  `legacy_supervise_requires_director_execution_mode`;
- la llamada corregida con `director_execution_mode=legacy_director_loop`
  devolvio `HTTP 202 accepted_background`;
- no aparecieron artefactos en los write-sets de los temas 023/024;
- `/api/v0/autoprogramming/status` por run y por cola devolvio `HTTP 504`
  `autoprogramming_status_timeout`.

Ademas se observo un proceso interno `orquesta-app-codex-stack.test` durante la
supervision, sin que aparecieran procesos Codex de dominio ni entregas.

Accion esperada: el modo legacy aceptado no debe ser evaluado por checks
goal-first, y `autoprogramming/status` debe devolver estado parcial accionable
aunque el supervisor este ocupado. Si una supervision queda en background sin
artefactos, Orquesta debe exponer si esta ejecutando proveedor, tests internos,
esperando cola, bloqueada por lock o fallando en silencio.

## Workaround operativo propuesto para OPES

Mientras el agente de Orquesta arregla goal-first, OPES puede usar Orquesta con
compatibilidad legacy explicita para trabajos productivos acotados que no
conviene dejar parados:

```bash
ORQUESTA_EXTERNAL_WORK_LEGACY_DIRECTOR_LOOP=1
ORQUESTA_AUTOPROGRAMMING_LEGACY_DIRECTOR_LOOP=1
director_execution_mode=legacy_director_loop
```

Esto debe quedar documentado como workaround de produccion OPES, no como
reversion del diseño goal-first. No debe tocar codigo ni modificar el nucleo de
Orquesta desde OPES.

## Incidencia 19026 legacy: avisos de stream fd y cierre lento de ACK

En la ola `19026_legacy` de Integrador Social, los subagentes y assessments de
los temas 023 y 024 produjeron artefactos utiles y ACKs validos. No obstante,
varios logs de Codex mostraron repetidamente:

```text
Failed to create stream fd: Operation not permitted
```

El aviso no bloqueo los comandos: los logs muestran `succeeded`, ficheros
escritos y ACKs finales. En el assessment final de reutilizacion del tema 024
se observo ademas cierre lento: el agente escribio en stderr "Ultima accion:
escribir agent_ack.json" y el proceso siguio vivo unos minutos antes de que
apareciera `agent_ack.json`.

Evidencia:

- run:
  `run-opes-integracion-social-b-t024-consolidacion-checkpoint-20260627-19025c`;
- agente:
  `agent-ref-assessment-task-ref-app-change-appchange-b717892086bea75f051174c00e49e3eb-s-1b788db515866a072ea570e957e565b3`;
- ACK final: `status=completed`;
- ficheros utiles:
  `temas/tema_024/subroles/reutilizacion/matriz_reutilizacion_tema_024_subrol.md`,
  `00_control/orquesta_runs/.../decision_reutilizacion_tema_024_19025c.md` y
  `00_control/director_20260627/subroles/reutilizacion/decision_reutilizacion_tema_024_19025c.md`.

Accion esperada: revisar si el wrapper de ejecucion Codex/PTY o permisos del
sandbox genera esos avisos y si el bridge puede publicar un estado intermedio
"ACK en escritura/cierre" para no confundir una salida lenta con bloqueo real.

## Incidencia 19026b: subroles utiles sin promocion canonica

En la ola `19026b` de Integrador Social se lanzo consolidacion/integracion de
los temas 021 y 025-031. El tema 021 tenia margen textual estrecho frente al
minimo B. El subrol `redaccion` produjo un bloque publicable de refuerzo:

- `temas/tema_021/subroles/redaccion/bloques_publicables_refuerzo_tema_021_19026b.md`
- informe:
  `temas/tema_021/subroles/redaccion/informe_redaccion_s3_tema_021_19026b.md`

El informe declara que el bloque es integrable, pero tambien indica que no
edito `borrador_ampliado.md` porque el subrol quedo limitado a
`temas/tema_021/subroles/redaccion`. Aunque el request externo permitia
`temas/tema_021`, Orquesta subdividio el write-set de forma que la pieza util
no pudo promocionarse a `04_markdown/tema_021_ampliado.md`.

Impacto: Orquesta produce trabajo valido, pero no cierra el ciclo de
consolidacion canonica. El Director OPES sigue necesitando una fase posterior
para integrar bloques publicables, recalcular extension y dejar el tema en
`pendiente_integracion_qa`.

Accion esperada: el Director/Orquesta debe crear automaticamente una fase
`promocion_canonica` o `integracion_editorial` cuando un subrol entrega piezas
`integrable` fuera del Markdown canonico. Esa fase debe tener write-set sobre
`04_markdown`, validaciones de extension/texto publico y estado honesto sin
marcar `ready` si faltan HTML, visuales, tests, RAG, audio o QA.

## Incidencia 19026b: esfuerzo de razonamiento alto por defecto en subagentes OPES

En la ola `19026b` de Integrador Social, los procesos lanzados por Orquesta para
subroles y assessments aparecen con:

```text
-c model_reasoning_effort="high"
```

La regla operativa OPES de agentes compactos pide razonamiento `medium` por
defecto y `xhigh/high` solo cuando este justificado. En esta ola no consta una
justificacion por subrol para usar `high` en todos los procesos. El resultado es
trabajo valido, pero con consumo y latencia superiores a lo previsto.

Evidencia: procesos activos de los runs
`run-opes-integracion-social-b-t026-consolidacion-integracion-20260627-19026b`
a
`run-opes-integracion-social-b-t031-consolidacion-integracion-20260627-19026b`
mostraban `model_reasoning_effort="high"` en todos los subroles observados.

Accion esperada: Orquesta debe aceptar un perfil OPES compacto por defecto
(`medium`, salida breve, ACK/evidencia corta) y elevar esfuerzo solo por tarea o
subrol cuando el Director OPES lo pida expresamente. Tambien conviene exponer en
estado/API el perfil efectivo usado por cada agente para auditar consumo.
