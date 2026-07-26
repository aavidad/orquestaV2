# V23: corte durable del intake único

Fecha: 2026-07-26.

Estado: **partial_green_unsealed**.

> Actualización posterior: dossier durable, comandos públicos y la transición
> atómica confirmación/freeze/Goal ya están integrados y acreditados de forma
> parcial por `3c24f113`. El resto de este documento conserva el alcance y los
> límites del corte durable inicial.

Este corte cierra la primera integración vertical del intake de Wizard:
dominio puro, writer de aplicación, autorización causal, CAS y replay
durables en SQLite, comandos públicos y reinicio real de la composición. No
cierra V23 completa, no crea el dossier, no confirma ni congela el intake, no
crea el plan causal, no sella receipt y no promociona el roadmap.

## Alcance integrado

- `internal/intake` sigue siendo dominio puro e inmutable. Chat y formulario
  aplican cambios sobre la misma referencia y revisión.
- `internal/application` es el único writer. Cada create/apply deriva
  fingerprint, digest y receipt estables, exige `expected_revision` y liga la
  autorización a la operación y al `request_ref` exactos.
- `internal/adapters/state/sqlite` implementa `IntakeStore` con migración 017.
  Conserva snapshot actual, cadena histórica, recibos inmutables, aislamiento
  `actor + project + intake_ref`, replay exacto y CAS transaccional.
- Recovery V23 revalida cardinalidad, snapshots, cadena causal, fingerprints,
  receipts, autorización histórica y estado actual antes de admitir la base.
- Los comandos `orquesta.intakes.create|get|apply` salen del registro canónico.
  Actor, proyecto, request y fingerprint no entran por payload; los enlaza la
  autoridad autenticada de la composición.
- Bootstrap inyecta el store SQLite y prueba create/apply/get, replay antes y
  después de reinicio, conflicto stale y rechazo de autoridad falsificada.

Commits del corte:

- `402d8bc4`: servicio application durable e idempotente;
- `b96a7a95`: validación por replay de la cadena causal;
- `01852c11`: autorización ligada a cada mutación;
- `8beef330`: frontera canónica de `request_ref`;
- `c0236024`: migración 017, CAS, replay y recovery SQLite;
- `7de8afeb`: wiring del Orchestrator y bootstrap;
- `cd0d4a19`: comandos, catálogo y compatibilidad de replay V20;
- `5f34b347`: allowlists arquitectónicas exactas y E2E de upgrade en bootstrap;
- `5ee54a9d`: fixture, bindings y gate vertical de aceptación.

## Compatibilidad de auditoría V20

El `registry_digest` histórico de V20 era el hash global
`sha256:56e48ffa327e9b593628ca07cc025ed073ad28cb529d25d11dde4a7440b94033`.
Añadir comandos cambiaba ese hash y podía convertir un replay legítimo de un
comando V20 inalterado en `ErrAuditConflict`.

La transición conserva un ledger inmutable de las 25 definiciones V20. Una
definición histórica reutiliza el digest global solo cuando el hash canónico de
la `Definition` completa coincide con el ledger; cualquier cambio semántico cae
a un digest nuevo por definición y SQLite mantiene la comparación exacta. Los
comandos nuevos usan directamente identidad por definición. El ratchet fija 25
entradas: retirarlas o cambiar su contenido exige una modificación explícita y
revisable del ledger.

La prueba unitaria acredita canonicalización, ledger y conflicto semántico. El
E2E vive en bootstrap —no dentro de `commands`— y conecta fila histórica,
Dispatcher actual y SQLite real; el replay conserva un solo efecto de Goal y
una definición mutada sigue en conflicto.

## Evidencia ejecutada

- `go test -race -count=1 ./internal/application -run Intake`: verde.
- Focal application + SQLite: verde; SQLite `5.697s`.
- Focal SQLite V23 con `-race`: verde; `144.7s`, proceso cerrado.
- `go test -count=1 ./internal/adapters/state/sqlite`: verde; `134.922s`.
- `commandgen -check`, commands completo e i18n: verdes.
- Focal commands intake/ledger con `-race`: verde; test `12.081s`.
- E2E bootstrap `TestV23IntakeDispatcherPersistsCASAndReplayAcrossRestart`:
  verde.
- E2E bootstrap
  `TestHistoricalRegistryAdmissionReplaysThroughCurrentDispatcherAndSQLite`:
  verde.
- Aceptación V20/V21 afectada y aceptación pura V23: verdes.
- Gates raíz `TestRebuildArchitecture` y
  `TestNeutralOrchestrationPackagesDoNotImportProductAdapters`: verdes.

`acceptance/fixtures/v23_wizard.json` conserva el RequiredTest puro del dominio
y añade un `integration_gate` separado con los tests que respaldan los tres
scopes declarados como integrados. `candidate_files` conserva el write-set del
contrato puro e `integration_files` enumera la implementación vertical. Así el
contrato puro no se confunde con la evidencia
application/SQLite/commands/bootstrap.

## Límites y siguiente dependencia

Al cerrar este corte quedaban pendientes:

- default canónico de rondas bajo `L-CONFIG`;
- dossier, confirmación explícita y freeze;
- creación causal del plan;
- catálogo Wizard completo, templates/domain packs y superficie web;
- sello, receipt y promoción del roadmap.

La dependencia indicada al cerrar este corte ya fue resuelta. La siguiente
dependencia vigente es `canonical_round_default_and_wizard_catalog`.

Deuda no bloqueante: cada apply relee y revalida la cadena histórica completa;
el coste acumulado es O(N²) si un intake alcanza muchas revisiones. Antes de
escalar ese volumen debe introducirse checkpoint o prueba incremental sin
rebajar recovery ni la verificación causal completa.

El rojo V02 por ocho binarios Firecracker fuera de `cmd/orquesta/**` es
`BUG-ORQ-20260726-513`, anterior e independiente de este corte. No se ratchea
aquí: la rama correctiva ya está preparada, pero debe integrarse después del
E2E root congelado para no invalidar sus hashes.
