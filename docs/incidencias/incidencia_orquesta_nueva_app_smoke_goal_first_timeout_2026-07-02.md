# Incidencia: smoke real Nueva App no cierra app completa

Fecha: 2026-07-02.

Bug inventario: `BUG-ORQ-20260702-122`.

## Contexto

Se ejecuto `scripts/smoke_goal_first_app_server_real.sh` con el Orquesta local
`fe06470557`, backend `app_server_tmux`, OPES desactivado y workspace temporal
aislado en `/tmp/orquesta-smokes/orquesta-goal-first-app-server.1PvR9N`.

El flujo probado fue el de app autonoma real:

- `POST /api/v0/apps/director` con `director_execution_mode=goal_first`.
- Solicitud `crear_app_completa` para una app web/API minima en Go.
- Observacion por `POST /api/v0/apps/director/goal/observe`.

## Resultado observado

El arranque fue aceptado:

- `run_ref=run-spec-smoke-goal-first-req-smoke-goal-first-b3c054d4d4ed80a32a726f319297cdc1`
- `goal_ref=goal-ref-app-director-run-spec-smoke-goal-first-req-smoke-goal-first-b3c054d4d4ed80a32a726f319297cdc1`
- `external_goal_ref=019f2393-ed26-7061-bd3c-535cdba5be14`

En el poll 22 el goal quedo bloqueado:

- `goal_status=blocked`
- `run_status=bloqueada`
- `current_phase=brainstorming_arquitectura`
- `recommended_action=replan`
- `closure_status=blocked`
- `summary=codex_app_server_goal_active_timeout`
- `artifact_refs=0`

El unico fichero bajo `generated-apps` fue un receipt inicial invalido:

`project/generated-apps/smoke-goal-first/docs/orquesta_goal_result_goal-ref-app-director-run-spec-smoke-goal-first-req-smoke-goal-first-b3c054d4d4e.json`

Ese receipt declara:

- `schema_version=orquesta_goal_result.v0`
- `status=invalid`
- `summary=Resultado durable inicial creado antes de construir y verificar la app.`
- `artifact_refs=[]`
- `evidence_refs=[]`

Al fallar, el smoke intento shutdown normal del backend tmux y recibio HTTP 409
con `status=backend_still_running`. El proceso temporal de app-server quedo
vivo y se limpio manualmente por socket exacto del smoke.

## Lectura arquitectonica

No es un fallo aislado de la UI de Nueva App. Repite el eje de
Goal-first/runtime ya abierto en `BUG-ORQ-20260701-073`,
`BUG-ORQ-20260701-079`, `BUG-ORQ-20260701-085` y `BUG-ORQ-20260701-088`:

- Orquesta puede lanzar el goal y observar estado.
- El agente puede crear un receipt inicial invalido antes de materializar
  artefactos utiles.
- El timeout bloquea la ejecucion en fase temprana sin promover de forma
  automatica a rework/retry acotado.
- El shutdown normal conserva correctamente `backend_still_running`, pero el
  operador/smoke aun necesita limpieza posterior cuando el backend queda vivo.

## Accion esperada

Para poder decir que Orquesta programa una app completa de forma autonoma, el
flujo Nueva App debe demostrar en smoke real que:

1. Materializa codigo de aplicacion bajo `generated-apps`.
2. Mantiene write-set y arquitectura declarada.
3. Ejecuta verificacion local o publica evidencia suficiente.
4. Cierra con `goal_status=complete`, `run_status=cerrada`,
   `closure_status=accepted` y `artifact_refs/evidence_refs` no vacios.
5. Si queda en receipt inicial invalido o brainstorming, replanifica de forma
   acotada sin volver al loop legacy y sin quemar cuota hasta timeout.
