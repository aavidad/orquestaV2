# Unificacion de ramas - 2026-05-11

## Resultado

`master` se ha avanzado por fast-forward hasta `candidato_refactor` y se ha subido a GitHub. Despues de documentar la decision, `candidato_refactor` tambien se ha eliminado porque ya no aportaba nada distinto de `master`.

Commit final comun:

- `b11cfe7 Document branch unification decisions`

Esto deja en `master` el nucleo nuevo revisado y probado. No se han fusionado ramas legacy completas porque vuelven a mezclar `cmd`, `db` e `internal/controlruntime` como superficies grandes, con conflictos y deuda que ya causaron problemas en V1/V2.

El remoto `origin` queda reducido a `origin/master`. Las ramas antiguas no fusionadas se han archivado como tags antes de borrarlas.

## Criterio usado

Solo se integra codigo que cumpla estas condiciones:

- aporta una capacidad que no exista ya en el nucleo nuevo;
- puede entrar en unidades pequenas y comprobables;
- no reintroduce base de datos concreta como dependencia del nucleo;
- no convierte `cmd` o `db` en centro arquitectonico;
- encaja con hexagonal, conectores, i18n y contexto pequeno por modulo;
- tiene pruebas o se puede probar de forma aislada antes de seguir.

## Ramas revisadas

| Rama | Estado | Decision |
| --- | --- | --- |
| `master` | Estaba detras de `candidato_refactor` | Fast-forward aplicado y subido. |
| `candidato_refactor` | Igual a `master` | Eliminada local y remotamente tras avanzar `master`. |
| `origin/feature/wizard-app-factory` | Sin commits unicos frente a `master` | Eliminada como rama remota obsoleta. |
| `backup/feature-wizard-app-factory-remote-20260411` | 1 commit unico, 99 ficheros, 15k lineas, conflictos en `cmd`, `db`, `internal/controlruntime` | Archivada como tag y eliminada como rama. No fusionar completa. |
| `berserk/salvage-stash-20260330` | 3 commits unicos, scripts con rutas locales y cambio de runtime viejo | Archivada como tag y eliminada como rama. La parte buena de handles fantasma ya existe en `runtimesapp`. |
| `orq-orquesta-codex2` | 1 commit unico sobre merge aislado de worktrees | Archivada como tag y eliminada como rama. La capacidad ya esta en `gitoperaciones`. |
| `orq-orquestador-codex11` | 1 commit unico sobre token_count de Codex | Archivada como tag y eliminada como rama. La capacidad ya esta en `internal/controlruntime`. |
| `orq-orquestador-codex1-t20` | 4 commits unicos, cockpit/API/pipeline sobre `cmd` y `db` | Archivada como tag y eliminada como rama. Sus ideas de estadisticas deben reimplementarse como puertos/modulos. |
| `reinicio-orquesta-v2-2026-05-04` | 145 commits unicos, muchos modulos/documentacion V2 | Mantener como fuente temporal de extraccion controlada. No hacer merge completo por conflictos de ficheros anadidos en ambos lados. |

## Tags de archivo

Tags subidos a `origin` para conservar el historico exacto de ramas eliminadas:

- `archive/branches/2026-05-11/backup-feature-wizard-app-factory`
- `archive/branches/2026-05-11/berserk-salvage-stash`
- `archive/branches/2026-05-11/orq-orquesta-codex2`
- `archive/branches/2026-05-11/orq-orquestador-codex1-t20`
- `archive/branches/2026-05-11/orq-orquestador-codex11`

## Capacidades ya retenidas

Estas capacidades de ramas antiguas ya existen en `master` y no justifican mantener ramas vivas para ellas:

- merge aislado de ramas sin pisar worktrees activas: `gitoperaciones/merge.go`;
- worktree detached para merges temporales: `gitoperaciones/worktree.go`;
- observacion Codex sin reescaneo historico innecesario: `internal/controlruntime/codex_observe.go`;
- arranque que puede superseder handles degradados o fantasmas sin pisar handles sanos: `runtimesapp/service.go`;
- pruebas asociadas en `gitoperaciones`, `internal/controlruntime` y `runtimesapp`.

## Extracciones V2 realizadas

```text
Fecha: 2026-05-11
Origen: reinicio-orquesta-v2-2026-05-04
Destino: master
Modulo: modulos/orquesta-capacity
Decision: Extraer el modulo completo como capacidad pura e independiente.
Motivo: Cubre seleccion dinamica de capacidad, low/medium/high/xhigh,
multi-HOME, cuota, handoff preventivo y politica de escalado sin DB, runtime,
proveedor ni modelo hardcodeado.
Validacion:
  - go test -count=1 ./modulos/orquesta-capacity
  - jq empty modulos/orquesta-capacity/docs/schemas/*.json modulos/orquesta-capacity/docs/fixtures/*/*.json
  - git diff --check
Notas: No se ha fusionado la rama V2. Solo se ha portado esta unidad.
```

```text
Fecha: 2026-05-11
Origen: reinicio-orquesta-v2-2026-05-04
Destino: master
Modulo: modulos/orquesta-persistence
Decision: Extraer el contrato de persistencia y ledger outbox en memoria.
Motivo: Aporta persistencia hexagonal validable, idempotencia y ACK de outbox
sin elegir SQLite, Postgres, MySQL ni ningun motor concreto. Los nombres de
motores aparecen solo en listas de rechazo y fixtures invalidos.
Adaptacion aplicada: el test de conflicto de outbox se actualizo al contrato
actual de `orquesta-core-workflow`, que exige `task_ref` y
`capacity_request_ref` en `LaunchRuntimeAgent`.
Validacion:
  - go test -count=1 ./modulos/orquesta-persistence ./modulos/orquesta-core-workflow
  - jq empty modulos/orquesta-persistence/docs/schemas/*.json modulos/orquesta-persistence/docs/fixtures/*/*.json
  - git diff --check
Notas: No se ha fusionado la rama V2. Solo se ha portado esta unidad.
```

```text
Fecha: 2026-05-11
Origen: reinicio-orquesta-v2-2026-05-04
Destino: master
Modulo: modulos/orquesta-i18n-docs
Decision: Extraer el builder y contratos i18n/docs iniciales.
Motivo: i18n y documentacion son reglas transversales por defecto para apps
generadas. El modulo produce planes serializables sin LLM, DB, runtime ni
filesystem productivo.
Validacion:
  - go test -count=1 ./modulos/orquesta-i18n-docs
  - jq empty modulos/orquesta-i18n-docs/docs/schemas/*.json modulos/orquesta-i18n-docs/docs/fixtures/*/*.json
  - git diff --check
Notas: No se ha fusionado la rama V2. Solo se ha portado esta unidad.
```

```text
Fecha: 2026-05-11
Origen: reinicio-orquesta-v2-2026-05-04
Destino: master
Modulo: modulos/orquesta-deploy
Decision: Extraer DeploymentPlan v0 y adaptadores dry-run.
Motivo: Cubre preparacion de entorno/deploy, matriz multi-OS, healthcheck,
rollback y targets local/contenedor/kubernetes/paas/desktop/mobile_store sin
crear artefactos reales ni elegir proveedor.
Validacion:
  - go test -count=1 ./modulos/orquesta-deploy
  - jq empty modulos/orquesta-deploy/docs/schemas/*.json modulos/orquesta-deploy/docs/fixtures/*/*.json
  - git diff --check
Notas: No se ha fusionado la rama V2. Solo se ha portado esta unidad. Los
adaptadores son dry-run puros; no generan Dockerfile, compose, manifests,
scripts, IaC ni conectan a servicios externos.
```

## Validacion

Comando ejecutado:

```bash
go test -count=1 ./gitoperaciones ./internal/controlruntime ./runtimesapp
```

Resultado:

- `orquesta/gitoperaciones`: OK
- `orquesta/internal/controlruntime`: OK
- `orquesta/runtimesapp`: OK

## Siguiente limpieza segura

Acciones recomendadas:

1. Revisar `reinicio-orquesta-v2-2026-05-04` por modulos, no por merge completo.
2. Extraer solo capacidades V2 que no existan ya en `master`.
3. Cuando termine esa extraccion, archivar tambien `reinicio-orquesta-v2-2026-05-04` como tag y eliminar la rama/worktree.
4. No restaurar `backup/*`, `berserk/*` ni ramas `orq-*` por merge completo; si se reutiliza algo, se porta por microtarea con test.
