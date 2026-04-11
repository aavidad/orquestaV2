# Handoff OpenClaw Berserk 2026-04-02

## Objetivo inmediato

Se ha copiado el proyecto completo a `berserk@192.168.122.152:/home/berserk/Trabajo/orquesta/` para seguir la orquestación desde ese equipo con OpenClaw.

Tambien se ha copiado el arbol de perfiles de agentes a:

- `/home/berserk/Trabajo/codex-perfiles`

Eso incluye:

- `bin/codex-perfil`
- `homes/Codex1..Codex12/auth.json`
- configuracion por home de agente

Dentro del propio repo remoto ya van incluidos tambien:

- `.orquesta-runtime`
- `.orquesta-worktrees`
- `orquesta.db`

## Punto exacto donde queda Orquesta

- Estado global visto con `./orquesta status`:
  - `410/476` tareas completadas (`86%`)
  - `4` tareas en progreso
  - `1` tarea reservada
  - `7` tareas retenidas por cuota
- Agentes conectados y con trabajo activo:
  - `Codex6`
  - `Codex8`
  - `Codex11`
  - `Codex12`

## Tareas activas ahora mismo

- `#484` `Autobloquear agentes degradados por estado operativo` -> `Codex6`
- `#485` `Sustituir control por PTY con status estructurado por perfil` -> `Codex8`
- `#487` `Autocerrar tareas con evidencia de entrega valida` -> `Codex11`
- `#488` `Auto-reasignar tareas desde agentes bloqueados a workers sanos` -> `Codex12`

## Tarea reservada

- `#486` `Calcular progreso automatico de tareas en web y API` -> `Codex8`

## Tareas retenidas por cuota

- `#410` -> `Codex4`
- `#412` -> `Codex3`
- `#413` -> `Codex4`
- `#414` -> `Codex4`
- `#416` -> `Codex3`
- `#421` -> `Codex3`
- `#477` -> `Codex2`

## Estado de cuentas útil

- `Codex6` -> `carlos@avidad.com` -> efectivo `70%`
- `Codex8` -> `alberto@avidad.com` -> efectivo `19%`
- `Codex11` -> `codex3@avidad.com` -> efectivo `74%`
- `Codex12` -> `carlos@avidad.com` -> efectivo `70%`

## Lo ultimo que se corrigio

- Se corrigio un punto muerto real en `runtime_orders/send_instruction`:
  - cuando el handle iba por `session_resume` pero no admitia input interactivo, Orquesta intentaba entregar guidance como si fuese un `resume` interactivo
  - eso dejaba ordenes en `ejecutando/pendiente` y atascaba la cola
- Ahora, en ese caso, la guidance se deja durable en mailbox y la orden se completa sin bloquear la lane caliente.
- Archivo clave tocado:
  - `db/controlplane_entities.go`
- Test añadido:
  - `TestRuntimeOrderSendInstructionMailboxSessionResumeQuedaDurable`

## Señales reales vistas antes del traspaso

- `Codex6` y `Codex12` son los agentes con evidencia mas clara de trabajo real en sus worktrees.
- `Codex8` tiene diff abierto, pero su progreso reciente era menos claro.
- `Codex11` consumio `resume`, pero todavia no habia dejado entrega util clara.

## Archivos donde parecia haber trabajo real

- `Codex6`
  - `cmd/cliente_servidor_recursos.go`
  - `cmd/controlplane_autonomia_nivel2.go`
  - `cmd/controlplane_support.go`
  - `db/agentes_merge.go`
  - `db/agentes_merge_test.go`
  - `db/schema.go`
  - `db/sesiones.go`
  - `planocontrol/runner.go`
- `Codex12`
  - `db/controlplane_entities.go`
  - `db/controlplane_entities_test.go`
  - `db/db.go`
  - `db/handoff_manager.go`
  - `db/handoff_manager_test.go`
  - `db/planificador.go`
  - `db/sqlwrap.go`
  - `db/runtime_orders_hot_index.go`
  - `db/runtime_orders_hot_index_test.go`

## Siguiente paso recomendado para OpenClaw

1. Arrancar Orquesta en Berserk y comprobar `server doctor`.
2. Revisar primero los diffs/worktrees de `Codex6` y `Codex12`.
3. Vigilar `Codex8` y `Codex11`; si siguen sin entrega util, marcarlos bloqueados y reasignar.
4. Mantener `#486` como siguiente bloque despues de cerrar `#485`.
5. No reactivar las tareas retenidas por cuota hasta que sus cuentas vuelvan a estar operativas.

## Comandos utiles

```bash
./orquesta status
./orquesta tarea listar --estado en_progreso
./orquesta runtime ordenes --estado pendiente
./orquesta agente presupuesto --refresh --json
./orquesta openclaw
```
