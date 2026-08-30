# Corte OPS-02: optimización test-only de SQLite

Fecha de ejecución: 2026-08-22.

Estado: incremento local ejercitado mediante focales, sin promoción de V23 ni
receipt de acreditación.

## Alcance

La optimización amortiza únicamente la construcción de fixtures SQLite de
tests. La plantilla latest vacía general se construye una vez mediante el
runner productivo de migraciones con `fastSQLiteTestDurability`, una política
test-only que usa `synchronous=OFF`; sus copias siguen entrando por el helper
rápido y no demuestran durabilidad `FULL`.

Las fixtures dedicadas de crash/V09 se construyen canónicamente mediante
`Open`, durabilidad `FULL` y la secuencia completa de migraciones. Cada copia
privada y cada reapertura perteneciente a esos recorridos vuelve a entrar por
`Open` con `FULL`; los failpoints, migraciones y recovery que son sujeto de las
pruebas no se sustituyen por la plantilla latest rápida ni por un repositorio
construido a mano.

La matriz multiproceso de crash V09 usa la fixture solo cuando la base aún no
existe. El proceso hijo y las reaperturas posteriores consumen la misma base
durable. Permanecen reales los cinco límites de crash, los failpoints, la
publicación atómica, el reinicio, la verificación de backup, restore y la
comprobación de residuos.

`recovery_v09_test.go` cambió únicamente en dos escenarios restore-only donde
el backup publicado es precondición, no el sujeto probado. La fixture conserva
el backup y su payload real después de cerrar la recuperación que lo creó; los
tests siguen ejecutando el restore adversarial, el intercambio de raíz y la
colisión de publicación. La matriz de failpoints continúa ejecutando
`CreateBackup` en cada subtest.

No se modificó código productivo, configuración de SQLite, timeouts ni
condiciones de skip. `git diff --name-only` y el inventario de no rastreados
mostraron únicamente ficheros `_test.go` y este corte documental.

## Gates focales ejecutados

Normal:

```text
go test -mod=vendor -count=1 ./internal/adapters/state/sqlite \
  -run '^(TestV09RecoveryFailpointsLeaveNoPartialPublication|TestV09RestoreNeverOverwritesTargetCreatedAtPublicationBoundary|TestV09RestoreRetainsInspectedPayloadHandleAcrossRootSwap|TestV09RecoverySurvivesProcessCrashAtEveryBoundary)$'
ok, paquete 4.833 s, pared 5.26 s, RSS máximo 221700 KiB
```

Race proporcional sobre la fixture y el failpoint asociado:

```text
go test -mod=vendor -race -count=1 ./internal/adapters/state/sqlite \
  -run '^(TestV09RecoveryFailpointsLeaveNoPartialPublication|TestV09RestoreNeverOverwritesTargetCreatedAtPublicationBoundary|TestV09RestoreRetainsInspectedPayloadHandleAcrossRootSwap|TestV09RecoverySurvivesProcessCrashAtEveryBoundary)$'
ok, paquete 54.873 s, pared 55.33 s, RSS máximo 267056 KiB
```

La matriz multiproceso conserva su skip preexistente bajo `-race`; el mismo
selector la ejecutó completa en normal y ejercitó bajo race los restores y
failpoints que sí son focales concurrentes.

Estos resultados no equivalen a la suite SQLite completa ni al gate V23-10.
No acreditan OPS-02 por sí solos y no autorizan presentar V23 como candidato
PASS. El siguiente gate de integración debe ejecutarse sobre la composición
que incluya todos los incrementos OPS-02.
