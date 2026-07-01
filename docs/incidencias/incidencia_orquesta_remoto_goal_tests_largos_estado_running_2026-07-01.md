# Incidencia: goals remotos running con tests largos y reconciliacion incompleta

Fecha: 2026-07-01

## Resumen

En el Orquesta aislado remoto `/srv/orquesta-self/runtime/audit-a920a47b`, el
servidor `a920a47b` queda saludable pero `autoprogramming/status` mantiene goals
goal-first como `running_live` y recomienda `observe_active_goals` de forma
repetida.

Al mismo tiempo hay comandos `go test` lanzados por Codex/app-server con edades
superiores a una hora y algunos por encima de dos horas. El endpoint
`/api/v0/autoprogramming/goal/observe` responde `goal_status=running` y
`summary=codex_app_server_thread_status_active`, sin convertir el caso en
timeout operativo ni en bloqueo accionable.

## Evidencia observada

- `POST /api/v0/autoprogramming/status` publica `running_live=4`,
  `observed_runs=4` y `recommended_action=observe_active_goals:goals`.
- `POST /api/v0/autoprogramming/goals/observe-active` acepta la operacion en
  segundo plano, pero tras esperar vuelve a aparecer la misma recomendacion.
- `POST /api/v0/autoprogramming/goal/observe` para `srv-task-011`, `t208` y
  `t260` devuelve `goal_status=running` y `recommended_action=observe_later`.
- En procesos remotos aparecen `go test -count=1 ./...`,
  `go test -count=1 ./cmd/orquesta-server` y
  `go test -count=1 ./modulos/orquesta-server` con duraciones muy altas.
- Existen `orquesta_goal_result_v0.json` en disco para trabajos distintos, por
  ejemplo `srv-task-024`, lo que indica que hay resultados locales que no
  equivalen necesariamente a los run_refs vivos proyectados por el servidor.

## Clasificacion

- Area: servidor / app-server goal-first / required-test runner.
- Estado: abierta.
- Severidad: alta operativa, porque impide saber si Orquesta avanza, esta
  bloqueada o consume recursos sin progreso.
- Hipotesis arquitectonica: el estado vivo de app-server se toma como verdad
  suficiente mientras el thread siga activo, pero falta una capa de lease/TTL y
  reconciliacion por comando ejecutado. `observe_active_goals` no debe crear un
  bucle indefinido de "observe_later" si los procesos hijos superan un umbral
  operacional o si existe evidencia durable contradictoria.

## Acciones

1. Mitigacion local: reutilizar `GOCACHE` padre explicito y seguro en el
   `required_test_runner` para reducir recompilaciones y duracion de pruebas.
2. Pendiente estructural: registrar procesos hijo por goal/turno con inicio,
   comando normalizado, TTL, ultima salida observada y decision publica
   (`running`, `timeout`, `blocked`, `needs_rework`, `terminal`).
3. Pendiente de reconciliacion: si un result file aparece en disco, validarlo
   contra `run_ref/goal_ref` activo antes de usarlo; si no coincide, publicarlo
   como diagnostico `result_file_unmatched`, no como cierre silencioso.
4. Pendiente de supervision: `observe_active_goals` debe ser idempotente por
   `operation_ref` y no relanzarse indefinidamente sin cambiar estado o
   diagnostico.

