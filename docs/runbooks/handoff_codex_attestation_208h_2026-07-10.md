# Handoff: atestacion independiente 208H

Fecha: 2026-07-10. Estado: `candidate_ready_for_independent_review`.

Actualizacion: la rama se rebased sobre `f24926d87` y se detectaron dos
conflictos de integracion que habian ocultado DTOs de resultado JSON y rutas de
`autonomy_program`. Se restauraron desde la base sin alterar la frontera de
208H; los tests focales se repitieron despues. La prueba del adaptador local
ahora ejecuta un `go test -count=1 ./...` real dentro de un modulo Go temporal,
con checkout, cache y entorno aislados. La lista exacta para reejecucion externa
vive en `docs/pruebas_revisor_208h_2026-07-10.md`.

La primera ejecucion de `scripts/orquesta_test_batches.sh` con solo
`ORQUESTA_TEST_BATCH_ROOT` fallo antes de probar codigo: el perfil aislado usa
ademas `ORQUESTA_TEST_CACHE_ROOT` para leases y mantenia el default remoto bajo
`/srv/orquesta-self`. El comando reproducible declara ambas rutas en `/tmp`.
No se modificaron scripts de F3 desde este corte. El receipt posterior ejecuto
dos pases: los paquetes 208H pasaron en ambos; `cmd/orquesta-server` fallo por
la asercion fail-closed desalineada, tmux/hook y ratchet. La asercion 208H se
corrigio y los otros tres frentes quedan fuera de este write-set.

## Alcance y frontera

Trabajo exclusivamente local en
`/home/alberto/Trabajo/orquesta-worktrees/attestation-208h-20260710`. No se
uso SSH, remoto, deploy, runtime productivo ni `codebase-memory-mcp`. La
reparacion directa queda como excepcion acotada porque el operador prohibio
arrancar Orquesta/runtimes para este corte critico.

La rama se rebased localmente sobre `6a8cb3e066`. Se conservaron los documentos
upstream de extraccion documental:

- `docs/diseno_subsistema_extraccion_documental_2026-07-10.md`;
- `docs/inventario_herramientas_extraccion_documental_2026-07-10.md`;
- `docs/runbooks/handoff_codex_extraccion_documental_2026-07-10.md`.

Los commits de codigo existen en esta rama y han sido rebased sobre
`f24926d87`; la documentacion final se commitea con la evidencia de pruebas.

## Findings cerrados en codigo local

1. Las specs goal-first reales de autoprogramacion activan
   `require_independent_required_test_attestation`. El contrato de entrada
   `AutoprogrammingRequiredTestAttestationV0` aporta checkout, revision,
   implementador, credencial, policy y hashes esperados. Sin ese bloque una
   request `goal_ready` queda invalida, no degrada a evidencia autodeclarada.
2. La independencia ya no se decide comparando nombres de agente.
   `GoalRequiredTestIdentityVerifierPortV0` verifica credenciales opacas bajo
   la policy congelada y devuelve principal, independencia y evidencias. Alias
   iguales pueden ser validos solo si el verificador confiable lo confirma;
   nombres distintos no bastan si el verificador rechaza.
3. Spec y receipt quedan ligados a `checkout_ref`, `revision_ref`, revision
   observada del checkout, digest canonico del write-set, hashes esperados,
   command hash y definition hash. Hashes before/after deben coincidir entre si
   y con la foto congelada.
4. `ObserveGoalWorkV0` impone el validator independiente aunque la composicion
   entregue otro closure validator. Si faltan attestor, store, identity
   verifier o closure validator, persiste cierre `blocked` tipado; no existe
   bypass por `required_test_results=passed` del implementador.
5. El replay consulta receipts de la revision actual y solicita al attestor
   solo `RequiredTests` sin receipt. Un receipt presente pero fallido o
   inconsistente bloquea; no se oculta repitiendo todo el lote.
6. `orquesta-state-file` usa lock de fichero Linux por run/recurso y CAS por
   `store_version` para estado de goal. Receipt repetido identico es
   idempotente; mismo ref con payload distinto es conflicto. La spec permanece
   inmutable.
7. La decision del identity verifier se conserva en
   `GoalClosureValidationV0.AttestationVerifications`; estado, principals,
   policy y refs de evidencia sobreviven recrear el store. El servidor cablea
   `GoalRequiredTestAttestationStore` al `stateStore` durable. No se inyecta
   attestor/verifier real por defecto: sin opt-in de operador el servidor falla
   cerrado, como demuestra el test HTTP.
8. El rebase local sobre `6a8cb3e066` termino sin perder documentacion upstream.

El contrato sigue hexagonal: `orquesta-goal` solo conoce DTOs y puertos;
filesystem, locks y ejecucion local viven en adaptadores. No hay proveedor,
modelo, comando, HOME ni credencial concreta hardcodeados.

## Evidencia adversarial

- implementador declara `passed` y attestor independiente falla;
- verifier no confiable bloquea aunque los nombres de agente sean distintos;
- alias de agente iguales no sustituyen la decision verificable del puerto;
- mutacion de checkout, revision, write-set o hashes bloquea;
- lifecycle con validator permisivo y puertos ausentes sigue bloqueando;
- replay con dos tests y un receipt previo invoca solo el test ausente;
- ocho subprocesos reales escribiendo el mismo receipt dejan un unico documento;
- dos subprocesos haciendo CAS desde `store_version=1` dejan un unico ganador y
  estado version 2;
- evidencia de identidad sobrevive recrear `StoreV0`.

## Pruebas verdes

```bash
go test -count=1 \
  ./modulos/orquesta-goal \
  ./modulos/orquesta-autoprogramming \
  ./modulos/orquesta-state-file \
  ./modulos/orquesta-app-director-service \
  ./modulos/orquesta-app-codex-stack \
  ./modulos/orquesta-mcp \
  ./modulos/orquesta-web

go test -race -count=1 \
  ./modulos/orquesta-goal \
  ./modulos/orquesta-autoprogramming \
  ./modulos/orquesta-state-file \
  ./modulos/orquesta-app-director-service \
  ./modulos/orquesta-mcp \
  ./modulos/orquesta-web

go test -race -count=1 ./modulos/orquesta-app-codex-stack \
  -run 'Test(LocalGoalRequiredTestAttestorV0|BuildDirectorPortsV0CableaAttestor|PrepareAutoprogrammingRunV0GoalReady|CodexStackAutoprogrammingPrepareRunAPIV0GoalReady|CodexStackAutoprogrammingPromotionV0GoalFirst)'

go test -count=1 . \
  -run 'Test(EnvVarsBudgetMEJ106V0|NeutralOrchestrationPackagesDoNotImportProductAdapters)$'
go test -count=1 ./cmd/orquesta-server \
  -run 'TestEnvVarsOrquestaRatchetMEJ106V0$'
git diff --check
```

La primera pasada `go test -race` del stack completo no detecto carreras, pero
fallo el umbral temporal de `TestSimulacionDeterministaFallosGoalFirstV0`:
74.97 s con instrumentacion race. La pasada focal anterior excluye solo esa
prueba de rendimiento y queda verde.

## Bateria transversal

`go test -count=1 ./...` dejo verdes todos los modulos y comandos salvo siete
tests preexistentes de cleanup/tmux en `cmd/orquesta-server`:

- `TestWaitForStateHealthyV0LimpiaGoalBackendConfiguradoSiMuereTrasReadinessV0`;
- `TestCleanupCodexGoalBackendAfterStartupFailureIfDaemonGoneV0SoloMataSiProcesoCayo`;
- `TestCleanupCodexGoalBackendAfterStartupFailureIfDaemonGoneV0MataSesionConfiguradaSinOwnerMarker`;
- `TestCleanupCodexGoalBackendAfterStartupFailureIfDaemonGoneV0MataProcesoPropioSinSocketV0`;
- `TestStopServerCommandV0ForceConDaemonMuertoLimpiaBackendGoalConfigurado`;
- `TestStopServerCommandV0SinForceConDaemonMuertoNoLimpiaBackendGoal`;
- `TestStopServerCommandV0ForceConDaemonIdentityMismatchNoLimpiaBackendGoal`.

Los fallos observados son `codex_app_server_tmux_generation_conflict`, ausencia
de `kill-session` en el fake y proceso fake que queda vivo. Ya aparecian en la
linea base de esta sesion y no pertenecen al write-set 208H. Los ratchets
MEJ-106 quedan verdes en 521 tras retirar prefijos `ORQUESTA_` de los helpers
de test multiproceso.

## Siguiente accion

Terra debe revisar el diff local, especialmente el contrato del identity
verifier y la semantica CAS. Para un smoke real posterior, el operador debe
inyectar un attestor aislado, un verifier de credenciales y policy confiable;
este corte no autoriza remoto, deploy ni proveedor real.

## Receipt final local

El receipt `orquesta_test_batches_receipt.v1` de
`/tmp/orquesta-test-batches-208h-focal/receipt.json` registra dos pases
consecutivos verdes: 6 paquetes, 4 lotes y 12 ejecuciones. Incluye
`orquesta-goal`, `orquesta-runtime-required-test`, `orquesta-state-file`,
`orquesta-app-director-service`, `orquesta-app-codex-stack` y
`orquesta-autoprogramming`.

El lote ampliado que tambien incluyo `cmd/orquesta-server` fallo en dos pases.
El test 208H desalineado se corrigio y su focal queda verde. Siguen abiertos y
fuera de 208H: deduplicacion de shutdown hooks, cleanup tmux/app-server y el
ratchet de variables (`537 > 511`). No se usan como evidencia de cierre.
