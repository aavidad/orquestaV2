# Resultado sync remoto Orquesta a920a47b

Fecha: 2026-07-01

## Alcance

Servidor aislado: `berserk@uso.dipgra.cloud`

Worktrees:

- Viejo preservado: `/srv/orquesta-self/runtime/audit-225a6752-next`
- Nuevo activo: `/srv/orquesta-self/runtime/audit-a920a47b`

No se tocaron servicios productivos, OPES productivo, nginx, bases de datos ni
puertos externos. El servidor sigue escuchando solo en `127.0.0.1:18787`.

## Commits cargados

- `e997fb86` - publica snapshot parcial cuando la observacion goal-first es
  rechazada por el backend.
- `393e7bfd` - documenta la reapertura remota incorrecta de startup-lock.
- `a920a47b` - integra pruebas utiles producidas por el agente remoto y rechaza
  el cambio incorrecto de `codex_wrapper`.

El bundle remoto validado quedo en:

- `/home/berserk/orquesta-inbox/orquesta-a920a47b.bundle`

El diff sucio del agente remoto quedo preservado en:

- `/home/berserk/orquesta-inbox/remote-dirty-diff-20260701T183353Z.patch`
- `/home/berserk/orquesta-inbox/remote-dirty-status-20260701T183353Z.txt`

## Pruebas

En local:

```bash
go test -count=1 ./modulos/orquesta-autoprogramming ./modulos/orquesta-server ./cmd/orquesta-server
```

En remoto, worktree `audit-a920a47b`:

```bash
GOCACHE=/srv/orquesta-self/runtime/a920a47b/go-cache \
GOTMPDIR=/srv/orquesta-self/runtime/a920a47b/go-tmp \
go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-app-codex-stack ./modulos/orquesta-autoprogramming ./modulos/orquesta-server ./cmd/orquesta-server
```

Resultado remoto: todos los paquetes pasaron.

## Estado runtime tras reinicio

`/api/v0/server/readiness` devuelve `ready=true`, `startup_ready=true` y
`status=running`.

Identidad observada:

- `HEAD`: `a920a47bd0`
- `binary_sha256`: `933736e420d56ac38f20f15b4ead6d78a1def9b7562b5bc5a3745341d699d9e5`

Sesiones aisladas vivas tras reinicio:

- `orquesta-server-latest`
- `orquesta-goal-30b1df755281b891`

## Resultado de la prueba Orquesta

El fallo opaco previo en:

- `request-ref-autoprogramming-backlog-t260-goal-first-codex-loop-delgado-c13b7862`

ya no devuelve solo `autoprogramming_observe_goal_http_error`.

Tras `a920a47b`, `POST /api/v0/autoprogramming/goal/observe` devuelve:

- `estado=error`
- `partial=true`
- `goal_status=blocked`
- `closure_status=blocked`
- `closure_needs_rework=true`
- `recommended_action=replan`
- `result_ref=evidence-ref-codex-app-server-goal-result-missing-after-timeout`
- `errores_publicos[0].code=codex_goal_observation_rejected`

Esto confirma que la API ya publica snapshot parcial accionable. El siguiente
paso no es investigar a ciegas, sino replanificar ese goal T260.

## Cambios remotos clasificados

Integrado:

- Cobertura APG-003/APG-004/APG-005 de automejora.
- Diagnostico ampliado en `required_test_runner_integration_test.go`.
- Ajuste de fixture SRV-TASK-024 en `supervisor_resident_pending_policy_v0_test.go`.

Rechazado:

- Cambios de `modulos/orquesta-runtime-codex/codex_wrapper_v0.go`.
- Cambios de `modulos/orquesta-runtime-codex/codex_wrapper_v0_test.go`.

Motivo: reabrian la decision ya cerrada de startup-lock. Cualquier variable
`ORQUESTA_CODEX_STARTUP_LOCK_*` declarada por el operador debe contar como
configuracion explicita.

Pendiente:

- Replan de T260 desde la nueva salida accionable.
- Enlazar esta incidencia/resultado en `docs/inventario_bugs_orquesta_2026-06-30.md`
  cuando se pueda tocar ese fichero sin mezclar cambios de OPES/otras sesiones.
