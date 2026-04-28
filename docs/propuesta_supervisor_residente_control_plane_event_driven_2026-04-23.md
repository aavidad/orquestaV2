<!--
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
-->

# Estado consolidado: supervisor residente + control plane event-driven

Fecha: `2026-04-23`
Estado: contrato operativo ya alcanzado; quedan remates de consolidacion explicita

## Objetivo

Dejar documentado como contrato canónico de autonomía persistente el perfil que hoy ya opera en el repo:

- el `supervisor` queda reservado y residente por proyecto
- el trabajo operativo normal se mueve por `runtime_orders`, `runtime_mailbox`, tareas y checkpoints
- `max_workers` cuenta solo workers no reservados
- el supervisor no se recicla como worker por defecto salvo recuperación explícita

## Qué ya existe y conviene preservar

- `supervisionapp.Policy` ya persiste `SupervisorAgente`, `ReserveSupervisor`, `MaxWorkers`, `EstadoAutonomia` y marcas temporales.
- `db/proyectoAdmiteWorkerAutonomiaParaAgente(...)` ya excluye supervisor/reviewer reservados del cupo de workers.
- `db.ReconciliarEstadoAutonomia()` ya limpia residuos donde un supervisor reservado acaba con payloads u órdenes de worker.
- `cmd/server_autobootstrap.go` y `cmd/repo_api.go` ya crean políticas muy cercanas a este perfil, pero cada entrada recompone el contrato a mano.

## Punto de consolidacion pendiente

La semántica ya está viva en el comportamiento del sistema, pero no está declarada en un punto único. Hoy el perfil "supervisor residente + control plane event-driven" todavía depende de repetir a mano varios flags:

- `enabled=true`
- `reserve_supervisor=true`
- `auto_create_tasks=true`
- `estado_autonomia=activo`
- `supervisor_agente` no vacío
- `reviewer_agente` consistente si `review_required=true`

Eso ya no niega el estado alcanzado, pero sigue haciendo fácil que una entrada nueva active autonomía persistente con una combinación parcialmente válida.

## Consolidacion tecnica pendiente

### Fase 1. Canonicalizar el contrato en capa de política

Crear un helper puro en `supervisionapp` como punto único de normalización:

- `BuildResidentSupervisorEventDrivenPolicyInput(...)`

Invariantes del helper:

- fuerza `enabled=true`
- fuerza `reserve_supervisor=true`
- fuerza `auto_create_tasks=true`
- normaliza trims y `definition_of_done_json`
- valida `supervisor_agente`
- valida `reviewer_agente` cuando `review_required=true`
- deja `max_workers` como cuota exclusiva de workers
- por defecto deja `estado_autonomia=activo`

Esto ya refleja la semantica alcanzada hoy; lo pendiente es consolidarlo como punto unico explicito.

### Fase 2. Sustituir ensamblado manual en entrypoints de alta

Cuando el frente de `cmd/` esté estable, migrar estos puntos a usar el helper:

- `cmd/server_autobootstrap.go`
- `cmd/repo_api.go`
- `cmd/api.go`
- `cmd/proyectos_web.go`
- `cmd/autonomia_bootstrap.go`

Regla de migración:

- toda política persistente de tipo "finish_app" o "autonomía viva por proyecto" debe pasar por el helper antes de `UpsertProjectPolicy`

### Fase 3. Endurecer semántica en tests de integración

Añadir o reforzar tests que verifiquen:

- el supervisor reservado no consume `max_workers`
- el supervisor reservado no acumula mailbox/órdenes de worker tras reconciliación
- el bootstrap crea supervisor residente primero y workers después
- reactivar una policy viva no degrada a modo "todos son workers"

## Alcance explícitamente fuera de esta entrega

- no tocar `cmd/controlplane_support.go`
- no tocar `agentesapp/service.go`
- no cambiar el loop del runner ni el dispatcher productivo
- no reescribir bootstrap ni API en un worktree con cambios concurrentes

## Resultado visible hoy

Hoy debe documentarse asi:

- supervisor residente y estable
- workers reales/ejecutores guiados por eventos
- cuota separada entre supervisor/reviewer reservados y workers reales
- `max_workers` cuenta solo ejecutores
- `autonomyPending=0` refleja continuidad drenada en la foto estable
