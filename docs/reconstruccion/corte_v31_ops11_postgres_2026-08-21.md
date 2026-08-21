# Corte V31 OPS-11: revalidación precommit y outcome desconocido

Fecha: 2026-08-21.

Estado: incremento implementado de foundation; `OPS-11` sigue `declared` y
`AC-V31-POSTGRES-S3-MULTIHOST` sigue `planned`. Este corte no es evidence ni
receipt de acreditación.

## Alcance

El foundation PostgreSQL solicita una única transacción serializable para el
CAS de snapshot y la inserción de evento y outbox. Después de emitir esas
escrituras y antes de invocar `Commit`, emite en la misma transacción una
consulta de revalidación que incluye:

- proyecto, Goal y WorkItem exactos;
- revisión nueva y generación del plan;
- token y cerca del lease;
- referencia de mutación aplicada;
- el predicado de vigencia `lease_expires_at > clock_timestamp()`.

Con los dobles locales, si la fila deja de acreditar esa autoridad o el lease
ya ha caducado, el foundation solicita `Rollback` antes de `Commit`. Application
sigue siendo el único writer de lifecycle; el adaptador no decide transiciones.

El preflight obligatorio de `OPS-11` para
`(*Foundation).ApplyAtomicMutation` devolvió
`no_semantic_candidate_for_capability`. Por tanto no se copió ni adaptó código
legacy; se caracterizó el hueco contra el puerto vigente y AC-V31.

## Presupuesto y complejidad

Antes de integrar se fijó para el candidato completo un techo de `P<=50` LOC
productivas netas, `V<=150` LOC netas de tests/fixture, 120 líneas de corte,
200.000 tokens, 90 minutos y menos de 200 MiB persistentes. El consumo medido
contra `HEAD` es `P=+28` (`+34/-6`), `V=+109` (`+154/-45`) y este corte queda
por debajo de 120 líneas. La sesión escritora informó 145.666 tokens, quedó por
debajo de 90 minutos y no dejó temporales ni procesos propios persistentes.

La complejidad productiva añadida es una consulta SQL de revalidación y un
código de error estable. No se añaden paquetes, stores, writers, goroutines,
loops residentes, configuración ni dependencias.

## Claims acotados

Los dobles `database/sql` prueban solo la secuencia solicitada y los negativos
locales de la primitiva. Antes de `Commit`, un fallo de base de datos conserva
`postgres.foundation_unavailable`, una cancelación o deadline conserva
`postgres.foundation_context`, una fila ausente devuelve
`postgres.foundation_stale_fence` y una respuesta incoherente devuelve
`postgres.foundation_invariant`; todos solicitan rollback. Exclusivamente
después de invocar `Commit`, cualquier error, incluida una causa
`context.DeadlineExceeded`, se expone como
`postgres.foundation_commit_unknown` y no provoca una segunda decisión
transaccional.

Ese código permite al caller distinguir el caso que exige reconciliación antes
de decidir replay o retry. No atribuye que PostgreSQL haya confirmado ni
revertido la transacción. La causa permanece disponible mediante `errors.Is`.

Este corte no demuestra la semántica real de PostgreSQL para aislamiento,
`clock_timestamp()`, espera por locks, skew o failover, ni paridad con
`StateRepository`/SQLite.

## Gates abiertos

- driver PostgreSQL concreto, esquema completo y migraciones ejecutables;
- implementación completa del mismo `StateRepository` y suite compartida con
  SQLite;
- replay idempotente durable, conflicto de huella divergente y reconciliación
  de commit desconocido tras restart;
- validación con PostgreSQL real de reloj, espera por locks, serialización y
  carreras;
- observaciones de cuota, colocación y vínculo atómico con la reserva;
- identidad y leases de nodo, affinity, recuperación de host y workspaces
  clonables;
- aislamiento/collaboration multiusuario y multihost, carga y particiones;
- backup, restore, promoción y migración SQLite a PostgreSQL sin dual write;
- composición productiva con una sola fuente activa y receipt del candidato
  sellado.

## Verificación del corte

Pasaron sobre el candidato final del write-set:

```text
go test -mod=vendor -count=1 ./internal/adapters/state/postgres
go test -mod=vendor -count=1 -race ./internal/adapters/state/postgres
go test -mod=vendor -count=1 acceptance/v31_postgres_state_test.go acceptance/evidence_support_test.go -run '^(TestAcceptanceV31PostgresStateFoundation|TestV31PostgresStateFoundationDoesNotClaimOPS11Accreditation)$'
go test -mod=vendor -count=1 -race acceptance/v31_postgres_state_test.go acceptance/evidence_support_test.go -run '^(TestAcceptanceV31PostgresStateFoundation|TestV31PostgresStateFoundationDoesNotClaimOPS11Accreditation)$'
GOFLAGS=-mod=vendor go vet ./internal/adapters/state/postgres
GOFLAGS=-mod=vendor go vet acceptance/v31_postgres_state_test.go acceptance/evidence_support_test.go
git diff --check -- internal/adapters/state/postgres acceptance/v31_postgres_state_test.go acceptance/fixtures/v31_postgres_state.json docs/reconstruccion/corte_v31_ops11_postgres_2026-08-21.md
```

Ningún test con dobles se presenta como integración real ni como acreditación
de `OPS-11`.

Antes de que cambiara el WIP concurrente también pasaron el focal normal/race y
`vet` sobre el paquete `./acceptance`. El rerun final del paquete quedó
bloqueado al compilar tests V25 ajenos: las fixtures de Ollama y
OpenAI-compatible usan campos todavía ausentes de `ProviderCatalogConfig`.
Por la misma razón no se ejecutó `go test -mod=vendor -run '^$' ./...`. Durante
la sesión también cambió WIP fuera de este write-set en SQLite, application,
bootstrap y otras verticales; un resultado global no sería atribuible a este
corte.
