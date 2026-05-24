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

## Guardas

- `worktree_ref` y `branch_ref` se conservan como refs opacas.
- La promocion no corre sin cierre de run, review aceptada y evidencias `passed`
  para todos los tests requeridos.
- Si hay otra run ejecutable con `write_set` solapado, la cola conserva la run
  como pendiente de promocion.
- El archivo de staging es idempotente: si el manifest ya existe y coincide, se
  acepta; si difiere, bloquea con evidencia compacta.

Validacion focal:

```bash
go test -count=1 ./modulos/orquesta-autoprogramming ./modulos/orquesta-runtime-worktree ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server
```
