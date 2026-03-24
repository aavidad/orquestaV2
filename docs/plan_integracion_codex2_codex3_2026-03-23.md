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

## Estado actual para handoff entre equipos

Fecha: 2026-03-23

Rama de trabajo actual:

- `prueba/merge-total-agentes-20260323`

Bloques ya integrados en esta rama:

- limpieza del bloque de terminales y daemon para no depender de SQL directo desde scripts
- base portable de persistencia y compat SQL de `codex2`
- observabilidad de runtimes y control plane util de `codex3`
- `runtime_mailbox`, checkpoints y cliente fino para operaciones activas
- reintegracion al bootstrap de BD de piezas vivas que se habian quedado fuera:
  - `pools_capacidad`, `pool_modelos`, `politicas_modelo`
  - `decisiones_proyecto`, `documentos_externos`, `git_merges`
  - `memoria_proyectos`, `memoria_fuentes`, `memoria_hallazgos`, `memoria_derivas`
  - `fases_proyecto`, `avance_tareas`, `presupuestos_sesion`
  - `sesiones.pool_id`
- ajuste de `persistencia info` para no requerir apertura de BD
- correccion del flujo MCP de voto de propuesta y refresco del estado devuelto

Validacion rapida hecha antes del handoff:

- `go build ./...`
- `go test ./db -run 'TestGuardarYLeerMemoriaProyecto|TestRegistrarFuentesHallazgosYDerivas|TestRegistrarYLeerPresupuestoSesion|TestEvaluarPresupuestoSesionHandoffPreventivo|TestCalcularResumenProgresoProyecto|TestCalcularResumenProgresoProyectoConFallbackDeEstado|TestGuardarPoolYListarResumen|TestResolverPoliticaModeloEconomicaPorPerfil|TestSchemaNoIncluyeCoordinacionPorFicherosObsoletos|TestSchemaSeparadoEnDDLYSemillas|TestSchemaDDLForDriverPostgresReduceSintaxisSQLite|TestBootstrapPlanForDriver|TestEsErrorMigracionIgnorable|TestListarConectores'`
- `go test ./cmd -run 'TestMCPToolVotarPropuestaActualizaEstado|TestCommandNeedsDBPersistenciaInfo|TestCommandNeedsDBPersistenciaHelp'`

Pendiente explicitamente no cerrado en esta pasada:

- no se ha revalidado aun la bateria completa de `go test ./cmd`
- `architecture_test.go` sigue reflejando deuda real de acceso directo a `db.*` desde `cmd/`
- hay cambios amplios y mezclados en `cmd/`, `db/`, `storage/`, `dashboardapp/` y `proposalapp/`; antes de merge a `master` conviene hacer una pasada final de ordenacion y validacion completa

Siguiente paso recomendado en el otro equipo:

1. partir de `prueba/merge-total-agentes-20260323`
2. ejecutar `go test ./db` completo y despues `go test ./cmd` completo
3. decidir si `architecture_test.go` se deja como gate duro o como deuda explicitamente aceptada para esta fase
4. si la pasada completa cuadra, preparar merge controlado a `master` sin tocar el remoto principal hasta validar

## Actualizacion 2026-03-24

Estado validado hoy:

- `go build ./...` en verde
- `go test ./cmd` completo en verde
- `go test ./db` completo en verde

Lectura practica a fecha 2026-03-24:

- el gate arquitectonico de `cmd` se ha acotado a lo que de verdad esta prohibido en esta fase de migracion: `SQL` crudo y aperturas de BD fuera de puntos controlados
- el uso de funciones de repositorio `db.*` desde `cmd` sigue siendo deuda de migracion, pero ya no bloquea la integracion mientras no se bypasseen los contratos con `SQL` directo
- el paquete `db` ha quedado en verde completo; las regresiones de bootstrap y renderer detectadas el 2026-03-23 y el 2026-03-24 quedaron corregidas
- el `SQL` crudo que quedaba en `cmd` para backlog y reasignacion de tareas ya se ha movido a funciones de `db` en castellano
- antes de mergear a `master`, sigue siendo recomendable decidir si se hace un ultimo merge de prueba desde esta rama total a una rama candidata o si se pasa ya a `master`
