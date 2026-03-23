<!--
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
-->

# Plan de integracion de ramas `codex2` y `codex3`

Fecha: 2026-03-23

## Objetivo

Integrar el trabajo util de `master`, `codex2` y `codex3` sin perder:

- la integracion de terminales Terminator con sesiones y worktrees de Orquesta
- el arranque de Orquesta como servicio persistente
- la evolucion hacia cliente fino, escritor unico y arquitectura hexagonal

## Decision principal

No procede hacer `merge` bruto de `review/codex2` ni de `review/codex3`.

Las dos ramas reescriben superficies calientes de `cmd/`, `db/`, `serve`, `sesion`
y cliente-servidor. Un merge directo mezclaría modelos incompatibles y pondría en
riesgo justo el trabajo que interesa conservar de `master`.

La integracion correcta es **selectiva**:

1. conservar comportamiento valido de `master`
2. sanearlo para que cumpla las reglas de arquitectura
3. extraer de `codex2` y `codex3` solo las piezas que encajan con la direccion vigente

## Lo que hay que preservar de `master`

### Terminales y continuidad

- `scripts/terminator_agentes.sh`
- `scripts/agente_console.sh`
- contratos de `sesion inicio`, `sesion guardar`, `sesion continuar`
- resolucion y reutilizacion de worktrees por agente
- continuidad via `external_session_id`, `resume_payload_json`, `cwd`, `branch`,
  `resumen_continuidad`, `host` y `pid`

### Servicio persistente

- `orquesta serve` como base del daemon local
- vigilancia de reanimacion, salud y planificacion
- despliegue por `systemd`

## Refactor ya aplicado en `refactor/op-086-terminator-daemon`

- los scripts de Terminator ya no consultan directamente la BD de Orquesta
- `scripts/agente_console.sh` obtiene continuidad por `orquesta sesion continuar`
- `scripts/terminator_agentes.sh` resuelve worktrees por `orquesta worktree resolver`
- `orquesta.service` y `orquesta-vigilante.service` quedaron saneados como plantillas
- se extrajo un `controlplane.Runner` para sacar automatismos de `cmd/serve.go`
- `go build ./...` y `go test ./cmd` estan en verde tras este corte

Nota:

- sigue existiendo lectura del estado local de Codex (`~/.codex/state_5.sqlite`) para
  detectar el hilo externo. Eso no toca la persistencia de Orquesta, pero debe
  tratarse como adaptador del runtime y no como logica de aplicacion.

## Que rescatar de `codex2`

### Integrar primero

- `storage/storage.go`
- `storage/dialect.go`
- `db/sqlwrap.go`
- `db/sqlcompat.go`
- `db/schema_backend.go`
- `db/schema_render.go`
- `db/schema_spec.go`
- tests de storage, dialecto, compat SQL y bootstrap por backend

### Integrar despues

- contratos de servicios de aplicacion:
  - `sessionapp`
  - `taskapp`
  - `dashboardapp`
  - `governance`
  - `runtimectl`

### No integrar tal cual

- `cmd/sesion.go`
- `cmd/server_client.go`
- `db/sesiones.go`

Motivo:

- en `master` esas zonas ya contienen semantica especifica de continuidad,
  worktrees, daemon, heartbeats y cliente fino que `codex2` no cubre por completo.

## Que rescatar de `codex3`

### Integrar primero

- `internal/runtimeobs/*`
- contratos de persistencia y observabilidad de runtimes
- `runtime_handles`
- `runtime_orders`
- handoff vivo asociado a runtimes

### No integrar tal cual

- `cmd/server.go`
- `internal/localrpc/*` como sustitucion del cliente-servidor actual
- delegacion global de comandos por `exec` RPC

Motivo:

- `master` ya sigue una direccion de cliente fino basada en API tipada HTTP.
- mezclar RPC generico de `codex3` con el cliente actual duplicaria modelos y
  complicaria la fuente de verdad del control plane.

## Orden recomendado de integracion

1. Terminar OP-086:
   sanear terminales y daemon en `master` sin perder funcionalidad.
2. Absorber de `codex2` la infraestructura portable de persistencia.
3. Mover la logica viva de `cmd/agente.go` a servicios/puertos.
4. Integrar de `codex3` la observabilidad normalizada y los contratos de
   `runtime_handles` y `runtime_orders`.
5. Reconciliar sesiones, worktrees y control de runtime sobre un unico plano
   de control (`serve` + API).
6. Solo entonces plantear merges funcionales o cherry-picks mas amplios.

## Riesgos a evitar

- volver a introducir acceso SQL directo a la BD de Orquesta desde scripts
- perder campos de continuidad de sesion
- mezclar dos clientes finos distintos en paralelo
- reescribir `cmd/sesion.go`, `cmd/agente.go` o `cmd/server_client.go`
  sin preservar los contratos de runtime ya usados por Terminator
- integrar ramas que borran documentacion o scripts utiles del repositorio

## Criterio de aceptacion

La integracion se considerara correcta cuando se cumpla todo esto:

- terminales y reanudacion siguen funcionando
- `orquesta serve` es el punto de control estable
- la CLI y la web no necesitan tocar SQLite directamente
- la capa de persistencia puede apuntar a mas de un backend
- la observabilidad de runtimes entra por adaptadores y contratos claros
- la logica de aplicacion sale progresivamente de `db/` y de scripts
