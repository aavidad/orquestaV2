# Pruebas declaradas para revision independiente de 208H

Estado: candidato local, no cerrar `BUG-ORQ-20260710-208H` hasta que un revisor
externo ejecute estos comandos en la rama `wip/attestation-208h-20260710`.

Todos los comandos usan ejecucion real `go test -count=1`. No usar
`go test ./...` global: el cierre amplio se hace al final por lotes aislados.

```bash
go test -count=1 ./modulos/orquesta-goal \
  -run 'TestGoalRequiredTestAttestationV0'

go test -count=1 ./modulos/orquesta-runtime-required-test \
  -run 'TestLocalGoalRequiredTestAttestationAdapterV0'

go test -count=1 ./modulos/orquesta-state-file \
  -run 'TestStoreV0GoalRequiredTest(Attestation|FinalSnapshot|Claim)|TestStoreV0GoalStateCAS'

go test -count=1 ./modulos/orquesta-app-codex-stack \
  -run 'TestValidateConfigV0RejectsPartialRequiredTestAttestationPorts|TestLocalGoalRequiredTestAttestorV0EsInyectadoYNoHaceFallback'

go test -count=1 ./modulos/orquesta-app-director-service \
  -run 'TestObserveAppDirectorGoalV0BloqueaRunSiFaltanRequiredTestsV0'

go test -count=1 ./modulos/orquesta-autoprogramming \
  -run 'TestFrozenRequiredTest|TestGoalHasFrozenRequiredTestsPhase'

go test -count=1 ./cmd/orquesta-server \
  -run 'TestGoalRequiredTestAttestationConfigV0|TestBuildStackFromEnvV0WiresCompleteGoalRequiredTestAttestationConfig'
```

Propiedades acreditadas: un receipt del implementador no cierra, identidad no
confiable bloquea, el runner vuelve a observar checkout y write-set antes y
despues del test, un test que muta el write-set falla, receipts contradictorios
bloquean, y el claim durable permite un solo atestador por test/revision incluso
entre procesos.

Verificacion amplia final, siempre despues de los focales y con una ruta de
cache externa al worktree:

```bash
ORQUESTA_TEST_CACHE_ROOT=/tmp/orquesta-test-batches-208h \
ORQUESTA_TEST_BATCH_ROOT=/tmp/orquesta-test-batches-208h \
  scripts/orquesta_test_batches.sh \
  ./modulos/orquesta-goal \
  ./modulos/orquesta-runtime-required-test \
  ./modulos/orquesta-state-file \
  ./modulos/orquesta-app-director-service \
  ./modulos/orquesta-app-codex-stack \
  ./modulos/orquesta-autoprogramming \
  ./cmd/orquesta-server
```

El script exige dos pases y escribe un receipt JSON. Un resultado sin ese
receipt, o con cualquier lote fallido, no acredita 208H.

## Revision ejecutada por el revisor externo (Claude) - 2026-07-10 tarde

Rama revisada: `trabajo/plataforma-agentes` con la integracion ya fusionada
(merge `7444dcf8a`). Ejecucion real, no autodeclarada; los conteos de tests
se verificaron con `-v | grep -c "^=== RUN"` para descartar verdes vacios.

Focales declarados (todos `ok`):

- `./modulos/orquesta-goal -run 'TestGoalRequiredTestAttestationV0'`: ok, 8 tests.
- `./modulos/orquesta-runtime-required-test -run 'TestLocalGoalRequiredTestAttestationAdapterV0'`: ok, 3 tests.
- `./modulos/orquesta-state-file -run 'TestStoreV0GoalRequiredTest(...)|TestStoreV0GoalStateCAS'`: ok, 8 tests.
- `./modulos/orquesta-app-codex-stack -run 'TestValidateConfigV0RejectsPartial...|TestLocalGoalRequiredTestAttestorV0...'`: ok, 2 tests.
- `./modulos/orquesta-app-director-service -run 'TestObserveAppDirectorGoalV0BloqueaRunSiFaltanRequiredTestsV0'`: ok, 1 test.
- `./modulos/orquesta-autoprogramming -run 'TestFrozenRequiredTest|TestGoalHasFrozenRequiredTestsPhase'`: ok, 3 tests.
- `./cmd/orquesta-server -run 'TestGoalRequiredTestAttestationConfigV0|TestBuildStackFromEnvV0Wires...'`: ok, 4 tests.

Ademas, suites COMPLETAS en verde de los modulos tocados:
orquesta-goal, orquesta-state-file, orquesta-runtime-required-test,
orquesta-app-director-service, orquesta-autoprogramming,
orquesta-app-codex-stack y orquesta-mcp (`go test -count=1`, `ok` real).

Verificacion amplia por lotes: NO acreditada aun. Bloqueada por el ratchet
de envs (`TestEnvVarsOrquestaRatchetMEJ106V0`: 536 variables `ORQUESTA_*`
frente al maximo 511) que afecta a `cmd/orquesta-server` dentro del listado;
mismo bloqueador que D3. Se acreditara al aterrizar la consolidacion de envs
del WIP de routing, sin subir el ratchet.

Veredicto del revisor: focales y suites de modulo ACREDITADOS; el cierre de
`BUG-ORQ-20260710-208H` sigue pendiente de (a) lotes dos pases verdes tras
la consolidacion de envs y (b) prueba del atestador real integrado en un
run con servidor configurado (rutas de fallo incluidas).
