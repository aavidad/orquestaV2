# Incidencia: autor del Council resuelto por intento no causal tras retry

Fecha: 2026-07-29.
ID: `BUG-ORQ-20260729-590`.
Estado: corregida, revisada e integrada como desbloqueo operativo; pendiente de
promoción del servidor residente.

Referencias del hallazgo:

- Goal: `goal:348e28a2d2e1fd2bc9e57ede8252a62f`.
- WorkItem: `work-item:6912246e2165ea5475f45bbcc8c8626f`.
- ejecución de reparación: `execution:22b8dc42fff7f76ff1fb9182ae2c1688`.
- capacidades: `GOV-12`, `GOV-13`, `GOV-14`, `EVD-06` y `EVD-07`.
- contratos: `AC-V18-INDEPENDENT-REVIEWS` y `AC-V19-COUNCIL`.

## Estado observado

Cuando un autor fallaba y el mismo WorkItem producía un segundo intento
correcto, el `ChangeSet` quedaba ligado al segundo `ExecutionRef`. Sin embargo,
`ValidatePersistedCouncilSubject` recorría las ejecuciones y devolvía el primer
autor que coincidía solo en Goal, WorkItem, generaciones y `spec_hash`.

El primer intento fallido también satisface esos campos. Al compararlo después
con `change.ExecutionRef`, la validación rechazaba como
`council.persisted_subject_invalid` un subject construido desde el PASS y las
dos reviews válidas del segundo intento. En SQLite el rechazo impedía abrir o
recuperar la ronda durable. No se observó aceptación falsa: el fallo era un
bloqueo de progreso causal.

## Causa arquitectónica

La resolución introducía una autoridad implícita por orden de lectura sobre
`record.Executions`. Esa heurística competía con la relación durable exacta que
ya posee el `ChangeSet`.

Invariante restaurado:

```text
Council subject
  -> ChangeSetRef exacto
  -> ChangeSet.ExecutionRef exacto
  -> Execution autora exacta
  -> PASS y dos reviews del mismo subject
```

Un intento anterior, aunque comparta Goal, WorkItem, generaciones y
`spec_hash`, no puede sustituir al autor nombrado por el cambio.

## Corrección

La validación resuelve primero el `ChangeSet` por el ref del subject y después
busca exclusivamente la ejecución indicada por `change.ExecutionRef`. Sobre
esa ejecución exacta conserva todas las comprobaciones de propósito, scope,
generaciones y `spec_hash`.

No se añadieron puertos, stores, writers ni lifecycle alternativos. SQLite
continúa siendo adaptador y `internal/application` conserva la decisión causal.

## Prueba de lección

- `TestValidatePersistedCouncilSubjectResolvesRetriedAuthorFromChangeExecutionRef`
  crea un intento autor fallido y un retry correcto; prueba que el cambio
  referencia el intento 2 y que el subject persistido sigue siendo válido.
- `TestSQLiteV19CouncilAuthorRetryPassReviewsAndThreeMemberRoundSurvivesRestart`
  persiste el retry, un PASS, dos reviews y una ronda de tres miembros; reabre
  SQLite, revalida el subject y obtiene una decisión aceptada con tres facts.
- La mutación equivalente —volver a elegir el primer autor por campos
  generales— hace fallar ambos regresores antes de abrir el Council.

Evidencia focal verde:

```text
go test -mod=vendor -count=1 ./internal/application \
  -run '^TestValidatePersistedCouncilSubjectResolvesRetriedAuthorFromChangeExecutionRef$'
go test -mod=vendor -count=1 ./internal/adapters/state/sqlite \
  -run '^TestSQLiteV19CouncilAuthorRetryPassReviewsAndThreeMemberRoundSurvivesRestart$'
```

Evidencia adicional verde, con `umask 0022` para que los fixtures de modos
inseguros creen exactamente los permisos solicitados:

```text
go test -mod=vendor -count=1 \
  ./internal/application ./internal/adapters/state/sqlite
go test -mod=vendor -race -count=1 \
  ./internal/application ./internal/adapters/state/sqlite \
  -run '^(TestValidatePersistedCouncilSubjectResolvesRetriedAuthorFromChangeExecutionRef|TestSQLiteV19CouncilAuthorRetryPassReviewsAndThreeMemberRoundSurvivesRestart)$'
go test -mod=vendor -run '^$' ./...
GOFLAGS=-mod=vendor go vet ./internal/... ./cmd/orquesta
```

La atestación física original terminó en cuarentena por
`test_attestor.output_limit_exceeded` después de ejecutar la microVM. No se
repitió el efecto ambiguo. El cambio se revisó e integró mediante la excepción
acotada de desbloqueo operativo, con los dos regresores focales y su variante
`-race` verdes. La incidencia del límite de salida queda separada en
`BUG-ORQ-20260729-591`; no se presenta la ejecución cuarentenada como
atestación válida.

## Criterio de reapertura

Reabrir si un subject del Council depende del orden de `Executions`, si un
retry autor válido no supera recovery, o si PASS, reviews, ronda o decisión
pueden ligarse a un `ExecutionRef` distinto del fijado por el `ChangeSet`.
