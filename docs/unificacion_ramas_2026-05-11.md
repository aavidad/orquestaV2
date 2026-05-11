# Unificacion de ramas - 2026-05-11

## Resultado

`master` se ha avanzado por fast-forward hasta `candidato_refactor` y se ha subido a GitHub.

Commit final comun:

- `a21892f Ignore generated Orquesta run artifacts`

Esto deja en `master` el nucleo nuevo revisado y probado. No se han fusionado ramas legacy completas porque vuelven a mezclar `cmd`, `db` e `internal/controlruntime` como superficies grandes, con conflictos y deuda que ya causaron problemas en V1/V2.

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
| `candidato_refactor` | Rama buena de trabajo | Queda como igual a `master` tras el avance. |
| `origin/feature/wizard-app-factory` | Sin commits unicos frente a `master` | Ya esta absorbida; se puede borrar como rama remota obsoleta. |
| `backup/feature-wizard-app-factory-remote-20260411` | 1 commit unico, 99 ficheros, 15k lineas, conflictos en `cmd`, `db`, `internal/controlruntime` | No fusionar completa. Solo sirve como referencia forense; muchas ideas ya estan supersedidas por modulos nuevos. |
| `berserk/salvage-stash-20260330` | 3 commits unicos, scripts con rutas locales y cambio de runtime viejo | No fusionar completa. La parte buena de handles fantasma ya existe en `runtimesapp` con una solucion mas completa. |
| `orq-orquesta-codex2` | 1 commit unico sobre merge aislado de worktrees | No fusionar completa. La capacidad ya esta en `gitoperaciones` con promocion aislada, worktree detached y proteccion de rama activa. |
| `orq-orquestador-codex11` | 1 commit unico sobre token_count de Codex | No fusionar completa. La capacidad ya esta en `internal/controlruntime` con `searchAnchor` y `accountAnchor`. |
| `orq-orquestador-codex1-t20` | 4 commits unicos, cockpit/API/pipeline sobre `cmd` y `db` | No fusionar por bloque. Puede aportar ideas de estadisticas, pero deben reimplementarse como puertos/modulos, no como ampliacion legacy. |
| `reinicio-orquesta-v2-2026-05-04` | 145 commits unicos, muchos modulos/documentacion V2 | Mantener como fuente temporal de extraccion controlada. No hacer merge completo por conflictos de ficheros anadidos en ambos lados. |

## Capacidades ya retenidas

Estas capacidades de ramas antiguas ya existen en `master` y no justifican mantener ramas vivas para ellas:

- merge aislado de ramas sin pisar worktrees activas: `gitoperaciones/merge.go`;
- worktree detached para merges temporales: `gitoperaciones/worktree.go`;
- observacion Codex sin reescaneo historico innecesario: `internal/controlruntime/codex_observe.go`;
- arranque que puede superseder handles degradados o fantasmas sin pisar handles sanos: `runtimesapp/service.go`;
- pruebas asociadas en `gitoperaciones`, `internal/controlruntime` y `runtimesapp`.

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

1. Borrar ramas remotas ya absorbidas, empezando por `origin/feature/wizard-app-factory`.
2. Para ramas no absorbidas, crear una etiqueta de archivo antes de borrar si se quiere conservar el commit exacto.
3. Mantener `reinicio-orquesta-v2-2026-05-04` hasta terminar la extraccion de modulos V2 que realmente aporten capacidades nuevas.
4. No fusionar `backup/*`, `berserk/*` ni ramas `orq-*` por merge completo; si se reutiliza algo, se porta por microtarea con test.
