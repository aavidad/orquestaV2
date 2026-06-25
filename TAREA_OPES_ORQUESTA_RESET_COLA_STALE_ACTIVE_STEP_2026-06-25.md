# Tarea Orquesta: reset conservador por cola OPES stale y `active_step` inválido

Fecha: 2026-06-25

## Contexto

Durante el cierre local del temario OPES `oficial-de-servicios-multiples` se encontró una cola de Orquesta aparentemente activa, pero sin agentes reales ejecutándose.

Servidor afectado:

- workspace OPES: `/home/alberto/Trabajo/OPES`
- curso: `opes-salidas/diputacion_granada/cursos_opes_2026/administracion_especial/oficial-de-servicios-multiples`
- puerto local: `127.0.0.1:8799`

## Síntomas observados

- `/api/v0/autoprogramming/status` informaba 119 runs en estado `running`.
- `pgrep -af 'codex|orquesta|claude|gemini'` no mostraba agentes Codex/Gemini/Claude reales asociados a esas runs; solo el servidor de Orquesta y la sesión interactiva.
- `/api/v0/runs/supervise` falló con:

```text
estado: error
stop_reason: runtime_error
diagnostics: app_director_service_invalido: operational_director_plan_state.active_step
```

- `/api/v0/server/shutdown` quedó en `waiting_checkpoint`:
  - `runs_stopped`: 14
  - `agents_in_flight`: 65
  - `checkpoints_pending`: 34
  - `checkpoint_agents_pending`: 57

## Mitigación aplicada

Como no había procesos de agentes vivos, se hizo reset conservador local:

- backup completo de estado y runtime en:
  `00_control/orquesta_backups/20260625_090517_reset_stale_active_step`
- renombrado, no borrado, del estado anterior:
  - `00_control/orquesta_state_stale_20260625_090517`
  - `00_control/orquesta_runtime/runtime_stale_20260625_090517`
- arranque de Orquesta con estado limpio y artefactos OPES intactos.

## Cambio necesario en Orquesta

Orquesta debe distinguir estado lógico antiguo de trabajo vivo real.

Requisitos propuestos:

1. Si una run aparece como `running` pero no existe proceso/agente asociado, debe reclasificarse como `stale`, `lost` o `needs_replan`, no seguir bloqueando supervisión ni apagado.
2. El fallo `operational_director_plan_state.active_step` inválido debe degradar solo la run afectada, no romper la supervisión global de la cola.
3. `/api/v0/server/shutdown` debe permitir cierre limpio cuando los `agents_in_flight` sean solo registros obsoletos sin PID vivo.
4. El estado público debe exponer conteos separados: `queued`, `running_live`, `running_stale`, `blocked`, `lost`, `completed` y `failed`.
5. El director residente OPES debe poder convertir runs `stale/lost` en tareas de replanificación sin intervención manual.

## Avance 2026-06-25

Cerrado en el stack Codex/Orquesta el aislamiento del fallo
`operational_director_plan_state.active_step`:

- el core y el store siguen validando estricto;
- `orquesta-app-codex-stack` clasifica ese fallo interno como recuperable;
- `/api/v0/runs/supervise` con `run_ref` devuelve `ok` con
  `last.status=needs_replan`, diagnóstico público
  `operational_plan_state_active_step_needs_replan` y acción
  `replan_operational_director_active_step`;
- la run afectada se sincroniza a cola no ejecutable `stopped` con evidencia,
  en vez de quedar `running` o tumbar el endpoint;
- la supervisión global convierte el mismo caso en resultado de drain
  `needs_replan`, rota la run a `stopped` y no propaga error fatal al tick;
- las recuperaciones previas al coordinador también apartan esa run concreta
  cuando reciben el mismo fallo, sin abortar toda la cola.

Evidencia local:

```bash
go test -count=1 ./modulos/orquesta-app-codex-stack
go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-run-supervisor ./modulos/orquesta-run-coordinator
```

Pendiente de esta tarea: taxonomía completa de estado vivo/stale/lost por PID y
apagado cooperativo cuando solo queden registros obsoletos sin proceso vivo.

## Criterio de aceptación

- Una cola con runs antiguas sin procesos vivos no impide `supervise`.
- `shutdown` no queda indefinidamente en `waiting_checkpoint` si no hay agentes reales.
- El operador puede saber por API si algo está vivo, colgado, perdido o simplemente en cola.
- La recuperación no requiere borrar artefactos OPES ni perder entregas previas.
