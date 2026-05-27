# Autoprogramacion: staging y promocion

Rail opt-in para automejora residente:

```text
run autoprogramming cerrada causalmente
  -> review aceptada + RequiredTestEvidenceV0 passed
  -> sin runs vivos con write-set solapado
  -> promocion por puerto de staging
  -> archivo idempotente de manifest sin borrar la worktree
```

## Contrato

- `orquesta-autoprogramming` decide si una run esta lista para promocion con
  refs opacas, tests requeridos y trabajos vivos.
- `orquesta-app-codex-stack` ejecuta el rail solo cuando la composicion inyecta
  `AutoprogrammingPromotion`.
- `orquesta-runtime-worktree` aporta el adaptador Git/worktree externo:
  valida cambios contra `write_set`, bloquea borrados y escribe archivo de
  staging sin borrar artefactos.
- `cmd/orquesta-server` lo cablea solo con
  `ORQUESTA_SERVER_AUTOPROGRAMMING_PROMOTION_ENABLED=1`.

## Variables

- `ORQUESTA_SERVER_AUTOPROGRAMMING_PROMOTION_ENABLED=1`: activa el rail.
- `ORQUESTA_SERVER_AUTOPROGRAMMING_PROMOTION_REPO_REF`: ref opaca del repo.
- `ORQUESTA_SERVER_AUTOPROGRAMMING_PROMOTION_APP_REF`: ref opaca de la app.
- `ORQUESTA_SERVER_AUTOPROGRAMMING_PROMOTION_COMMIT_MESSAGE`: mensaje de
  promocion local.
- `ORQUESTA_SERVER_AUTOPROGRAMMING_PROMOTION_ARCHIVE_DIR`: directorio operativo
  del adaptador para manifests; no cruza al nucleo.
- `ORQUESTA_SERVER_AUTOPROGRAMMING_PROMOTION_GUARDIAN_ENABLED=1`: exige guardian
  externo antes de cerrar el efecto de promocion como completo.
- `ORQUESTA_SERVER_AUTOPROGRAMMING_PROMOTION_GUARDIAN_COMMAND`: comando opt-in
  del guardian. Default: `go run ./cmd/orquesta-guardian check-promote`.
- `ORQUESTA_SERVER_AUTOPROGRAMMING_PROMOTION_GUARDIAN_CURRENT_BIN`,
  `CANDIDATE_BIN`, `LAST_GOOD_BIN`, `STATE_DIR`, `BUILD_COMMAND`,
  `TEST_COMMANDS`, `HEALTH_TIMEOUT`, `COMMAND_TIMEOUT`, `ARTIFACT_ROOT`,
  `ARTIFACT_MAX_BYTES`, `REPAIR_COMMAND` y `REPAIR_CODEX_*`: superficie
  canonica del servidor para pasar datos al guardian sin meter runtime, HOME,
  proveedor ni rutas de binarios en el nucleo. El reparador Codex es opt-in; si
  se activa, el servidor proyecta write-set/tests declarados y conserva las refs
  `run_ref`, `promotion_ref`, `worktree_ref` y `branch_ref` como opacas.

## Guardas

- `worktree_ref` y `branch_ref` se conservan como refs opacas.
- La promocion no corre sin cierre de run, review aceptada y evidencias `passed`
  para todos los tests requeridos.
- Si hay otra run ejecutable con `write_set` solapado, la cola conserva la run
  como pendiente de promocion.
- El archivo de staging es idempotente: si el manifest ya existe y coincide, se
  acepta; si difiere, bloquea con evidencia compacta.
- Si promocion, push o archivo devuelve `pending`, `pending_push` o `blocked`,
  la cola no se marca como `closed`; el retry debe conservar evidence refs
  compactas.
- Si el guardian esta activo y el candidato no compila, no pasa tests o no
  responde al healthcheck temporal, el puerto devuelve `blocked` retryable con
  evidencia `guardian_failed`; el binario vivo no se sustituye y la cola no se
  cierra como promocion completa.
- Si el guardian detecta artefacto mayor al presupuesto, symlink/hardlink
  inseguro, destino no regular, ruta fuera de la raiz declarada o cambio entre
  `Lstat`/open/post-copy, publica un reason code de artefacto y estado
  recuperable (`promotion_incomplete` o `last_good_unverified`) sin declarar
  `candidate_promoted`.

## Nota operativa

No se promueven cambios de autoprogramacion mientras exista cualquier run viva,
ejecutable, en revision o pendiente de agente externo con `write_set` solapado.
La promocion queda pendiente y debe reevaluar el estado vivo antes de cada
intento; no basta con que el candidato haya pasado tests si otra entrega puede
modificar el mismo alcance. `worktree_ref` y `branch_ref` siguen siendo refs
opacas y no se convierten en rutas ni nombres Git para resolver el solape.

## Evidencia T40

El e2e acotado cerrado el 2026-05-24 vive en
`TestCodexStackAutoprogrammingPromotionV0E2ERepoTemporalReplayV0`: usa repo Git
temporal, run de autoprogramacion cerrada causalmente, review aceptada,
`RequiredTestEvidenceV0` `passed`, promocion opt-in por el puerto del stack,
commit solo dentro del `write_set` y archivo idempotente. El replay repite el
tick y verifica que no aparecen commits ni manifests duplicados.

Push remoto/productivo no queda habilitado por este runbook. Esa superficie
requiere guardas propias de remoto, rama, ventana operativa y confirmacion de
operador.

Validacion focal:

```bash
go test -count=1 ./modulos/orquesta-autoprogramming ./modulos/orquesta-runtime-worktree ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server
```
